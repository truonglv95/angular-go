import { spawn, ChildProcess } from 'node:child_process';
import { BuildResult, OutputFile } from './index.js';
import { StringDecoder } from 'node:string_decoder';

export interface CreateContextOptions {
  project: string;
  compilationMode?: 'global' | 'local';
  hmr?: boolean;
  preserveImports?: boolean;
  styleIncludePaths?: string[];
}

export interface GoNgcClient {
  createContext(options: CreateContextOptions): Promise<string>;
  build(contextId: string): Promise<BuildResult>;
  getOutput(contextId: string, path: string): Promise<OutputFile | null>;
  getHmrUpdate(contextId: string, componentId: string): Promise<{ code: string | null }>;
  listHmrUpdates(contextId: string): Promise<{ componentIds: string[] }>;
  linkFile(path: string): Promise<{ code: string; changed: boolean }>;
  linkCode(code: string, id: string): Promise<{ code: string; changed: boolean }>;
  invalidate(contextId: string, file: string): Promise<void>;
  dispose(contextId: string): Promise<void>;
  close(): Promise<void>;
}

interface HelloResult {
  protocolVersion: number;
  serverName: string;
  capabilities: string[];
}

const SUPPORTED_PROTOCOL_VERSION = 1;

// B#10: Timeout constants
const BUILD_TIMEOUT_MS = 120_000;   // 2 minutes — compile can be slow on first run
const HMR_TIMEOUT_MS   = 15_000;   // 15 seconds
const LINK_TIMEOUT_MS  = 30_000;   // 30 seconds
const DEFAULT_TIMEOUT_MS = 30_000;

const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_BASE_DELAY_MS = 300;

// Method → timeout mapping
const METHOD_TIMEOUTS: Record<string, number> = {
  build:          BUILD_TIMEOUT_MS,
  get_hmr_update: HMR_TIMEOUT_MS,
  link_file:      LINK_TIMEOUT_MS,
  link_code:      LINK_TIMEOUT_MS,
};

export function createGoNgcClient(compilerPath: string, projectRoot: string): GoNgcClient {
  let proc: ChildProcess | null = null;
  let nextId = 1;
  const pendingRequests = new Map<number, {
    resolve: (res: any) => void;
    reject: (err: any) => void;
    timer: ReturnType<typeof setTimeout> | null;
  }>();
  let buffer = '';
  const decoder = new StringDecoder('utf8');

  // B#3 FIX: contexts map now stores options for re-registration after reconnect
  const contexts = new Map<string, CreateContextOptions>();
  let helloPromise: Promise<HelloResult> | null = null;

  // B#10 FIX: Reconnect state
  let reconnectAttempts = 0;
  let reconnecting = false;
  let shuttingDown = false;

  // ---------------------------------------------------------------------------
  // Daemon lifecycle
  // ---------------------------------------------------------------------------

  function startDaemon() {
    if (proc || shuttingDown) return;

    proc = spawn(compilerPath, ['--server'], {
      cwd: projectRoot,
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    const p = proc;

    p.stdout!.on('data', (chunk: Buffer) => {
      buffer += decoder.write(chunk);
      let newlineIdx: number;
      while ((newlineIdx = buffer.indexOf('\n')) >= 0) {
        const line = buffer.slice(0, newlineIdx);
        buffer = buffer.slice(newlineIdx + 1);
        if (!line.trim()) continue;
        try {
          const res = JSON.parse(line);
          const pending = pendingRequests.get(res.id);
          if (pending) {
            pendingRequests.delete(res.id);
            if (pending.timer) clearTimeout(pending.timer);
            if (res.error) {
              pending.reject(new Error(`Daemon error [${res.error.code}]: ${res.error.message}`));
            } else {
              pending.resolve(res.result);
            }
          }
        } catch {
          console.error('[go-ngc] Failed to parse daemon response:', line);
        }
      }
    });

    p.stderr!.on('data', (chunk: Buffer) => {
      const text = chunk.toString('utf8').trim();
      if (text) console.error(`[go-ngc] ${text}`);
    });

    p.stdin!.on('error', (err) => {
      console.error(`[go-ngc] stdin error: ${err.message}`);
    });

    p.on('error', (err) => {
      console.error(`[go-ngc] Process error: ${err.message}`);
    });

    // B#10 FIX: On unexpected exit, reject in-flight requests and auto-reconnect.
    p.on('exit', (code) => {
      if (proc === p) proc = null;

      // Cancel all pending requests
      const exitError = new Error(`Daemon exited unexpectedly (code ${code})`);
      for (const req of pendingRequests.values()) {
        if (req.timer) clearTimeout(req.timer);
        req.reject(exitError);
      }
      pendingRequests.clear();
      buffer = '';

      if (shuttingDown) return;

      // Auto-reconnect with exponential backoff
      if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS && !reconnecting) {
        reconnecting = true;
        const delay = RECONNECT_BASE_DELAY_MS * Math.pow(2, reconnectAttempts);
        reconnectAttempts++;
        console.warn(`[go-ngc] Daemon exited, reconnecting in ${delay}ms (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})…`);

        setTimeout(async () => {
          reconnecting = false;
          startDaemon();

          // B#3: Re-register all known contexts with the new daemon instance
          for (const [id, opts] of contexts) {
            try {
              helloPromise = null;
              await ensureHello();
              await sendRequest('create_context', { ...opts, contextId: id }, DEFAULT_TIMEOUT_MS);
            } catch { /* best-effort */ }
          }
        }, delay);
      } else if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
        console.error('[go-ngc] Daemon failed to restart after maximum reconnect attempts. Please restart Vite.');
      }
    });

    // Reset reconnect counter on successful startup
    reconnectAttempts = 0;
  }

  // ---------------------------------------------------------------------------
  // RPC transport
  // ---------------------------------------------------------------------------

  function sendRequest(method: string, params: any, timeoutMs?: number): Promise<any> {
    return new Promise((resolve, reject) => {
      startDaemon();

      if (!proc) {
        reject(new Error('Daemon is not running'));
        return;
      }

      const id = nextId++;
      const timeout = timeoutMs ?? METHOD_TIMEOUTS[method] ?? DEFAULT_TIMEOUT_MS;

      // B#10 FIX: Per-request timeout so a hung daemon never blocks indefinitely.
      let timer: ReturnType<typeof setTimeout> | null = null;
      if (timeout > 0) {
        timer = setTimeout(() => {
          pendingRequests.delete(id);
          reject(new Error(`[go-ngc] Request '${method}' timed out after ${timeout}ms`));
        }, timeout);
      }

      pendingRequests.set(id, { resolve, reject, timer });
      const msg = JSON.stringify({ id, method, params }) + '\n';
      proc!.stdin!.write(msg, (err) => {
        if (err) {
          pendingRequests.delete(id);
          if (timer) clearTimeout(timer);
          reject(err);
        }
      });
    });
  }

  async function ensureHello(): Promise<HelloResult> {
    if (helloPromise) {
      return helloPromise;
    }

    helloPromise = sendRequest('hello', {}, DEFAULT_TIMEOUT_MS).then((res: HelloResult) => {
      if (!res || res.protocolVersion !== SUPPORTED_PROTOCOL_VERSION) {
        throw new Error(
          `Unsupported go-ngc protocol version ${res?.protocolVersion ?? 'unknown'}; expected ${SUPPORTED_PROTOCOL_VERSION}`
        );
      }
      if (res.serverName !== 'go-ngc') {
        throw new Error(`Unexpected daemon server name ${res.serverName}`);
      }
      return res;
    });

    return helloPromise;
  }

  // ---------------------------------------------------------------------------
  // Public client API
  // ---------------------------------------------------------------------------

  return {
    // B#3 FIX: Send create_context RPC so the server pre-warms the context
    // (loads tsconfig, creates compiler host) before the first build arrives.
    async createContext(options: CreateContextOptions): Promise<string> {
      const id = `ctx_${Date.now()}_${Math.random().toString(36).substring(2)}`;
      contexts.set(id, options);
      await ensureHello();
      // Best-effort: if the daemon isn't ready yet the build handler will
      // lazily create the context, matching the existing server behaviour.
      try {
        await sendRequest('create_context', { ...options, contextId: id }, DEFAULT_TIMEOUT_MS);
      } catch (e) {
        console.warn(`[go-ngc] Could not pre-warm context (will be created on first build): ${(e as Error).message}`);
      }
      return id;
    },

    async build(contextId: string): Promise<BuildResult> {
      await ensureHello();
      const options = contexts.get(contextId);
      if (!options) throw new Error(`Unknown context ${contextId}`);
      const res = await sendRequest('build', { ...options, contextId });
      return res as BuildResult;
    },

    // B#3 FIX: Implement getOutput via RPC instead of returning null stub.
    async getOutput(contextId: string, filePath: string): Promise<OutputFile | null> {
      try {
        return await sendRequest('get_output', { contextId, path: filePath }, DEFAULT_TIMEOUT_MS);
      } catch {
        return null;
      }
    },

    async getHmrUpdate(contextId: string, componentId: string): Promise<{ code: string | null }> {
      return sendRequest('get_hmr_update', { contextId, componentId });
    },

    async listHmrUpdates(contextId: string): Promise<{ componentIds: string[] }> {
      return sendRequest('list_hmr_updates', { contextId });
    },

    async linkFile(path: string): Promise<{ code: string; changed: boolean }> {
      return sendRequest('link_file', { path });
    },

    async linkCode(code: string, id: string): Promise<{ code: string; changed: boolean }> {
      return sendRequest('link_code', { code, id });
    },

    async invalidate(contextId: string, file: string): Promise<void> {
      await sendRequest('invalidate', { contextId, file }, DEFAULT_TIMEOUT_MS);
    },

    async dispose(contextId: string): Promise<void> {
      contexts.delete(contextId);
      try {
        await sendRequest('dispose_context', { contextId }, DEFAULT_TIMEOUT_MS);
      } catch { /* best-effort */ }
    },

    async close(): Promise<void> {
      shuttingDown = true;
      if (proc) {
        try {
          await sendRequest('shutdown', {}, 5_000);
        } catch { /* process may already be gone */ }
        proc = null;
      }
    },
  };
}
