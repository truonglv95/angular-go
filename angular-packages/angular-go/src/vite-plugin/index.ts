import { version as viteVersion } from 'vite';
import type { ResolvedConfig, ViteDevServer } from 'vite';
import { spawn } from 'node:child_process';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import zlib from 'node:zlib';
import { createGoNgcClient, GoNgcClient } from './client.js';
import { resolveGoNgcPath } from '../compiler-path.js';

export interface OutputFile {
  path: string;
  text: string;
  hash?: string;
  kind?: 'js' | 'map' | 'dts' | 'other';
}

export interface DiagnosticMessage {
  category: 'error' | 'warning' | 'message' | 'suggestion';
  code: number;
  message: string;
  file?: string;
}

export interface BuildResult {
  outputs: OutputFile[];
  diagnostics?: DiagnosticMessage[];
  status?: number;
}

export interface AngularGoBaseOptions {
  /**
   * Path to the go-ngc binary. Defaults to 'go-ngc' in PATH.
   */
  compilerPath?: string;
  /**
   * Project root used for path resolution. Defaults to Vite's process cwd.
   */
  projectRoot?: string;
  /**
   * Print compiler/linker timing information.
   */
  verbose?: boolean;
}

export interface AngularGoCompileOptions extends AngularGoBaseOptions {
  /**
   * Angular tsconfig passed to `go-ngc -p`.
   */
  project?: string;
  /**
   * Additional or replacement arguments for go-ngc. When set, this overrides
   * the default `['-p', project]` compile command.
   */
  args?: string[];
  /**
   * Directory where go-ngc writes JavaScript. Defaults to Angular CLI's app outDir.
   */
  outDir?: string;
  /**
   * Re-run go-ngc when app source/template/style files change in dev. Defaults to true.
   */
  recompileOnChange?: boolean;
  /**
   * Enable Angular component Hot Module Replacement. Defaults to false.
   * HMR is opt-in and is always disabled during production builds.
   */
  hmr?: boolean;
  /**
   * Compilation mode. 'global' resolves templates across the entire program.
   * 'local' compiles each file in isolation. Defaults to 'global'.
   */
  compilationMode?: 'global' | 'local';
  /**
   * B#2 FIX: Plugin operating mode.
   * - 'server' (default): go-ngc runs as a long-lived daemon; outputs are
   *   served from memory via JSON-RPC.
   * - 'default': go-ngc is invoked once and outputs are written to disk.
   */
  mode?: 'server' | 'default';
}

export interface AngularGoLinkerOptions extends AngularGoBaseOptions {
  /**
   * Escape non-ASCII identifiers in Vite/esbuild output. Defaults to true.
   */
  ascii?: boolean;
  /**
   * Re-run Vite dependency optimization so dependencies are linked with the
   * current go-ngc binary. Defaults to true while the linker output is still
   * evolving.
   */
  forceOptimizeDeps?: boolean;
}

export interface AngularGoOptions extends AngularGoCompileOptions, AngularGoLinkerOptions {
  /**
   * Enable app source compilation. Defaults to true.
   */
  compile?: boolean;
  /**
   * Enable partial declaration linking. Defaults to true.
   */
  link?: boolean;
}

interface LinkCacheEntry {
  key: string;
  code: string;
}

class BoundedMap<K, V> {
  private map = new Map<K, V>();
  constructor(private maxSize: number) {}
  get(key: K): V | undefined {
    return this.map.get(key);
  }
  set(key: K, value: V): this {
    if (this.map.size >= this.maxSize && !this.map.has(key)) {
      const oldestKey = this.map.keys().next().value;
      if (oldestKey !== undefined) {
        this.map.delete(oldestKey);
      }
    }
    this.map.set(key, value);
    return this;
  }
  delete(key: K): boolean {
    return this.map.delete(key);
  }
  clear(): void {
    this.map.clear();
  }
  get size(): number {
    return this.map.size;
  }
}

class GoNgcError extends Error {
  readonly stdout: string;
  readonly stderr: string;
  readonly args: string[];
  readonly code: number | null;
  readonly id?: string;
  readonly loc?: { file?: string; line: number; column: number };
  readonly frame?: string;

  constructor(args: string[], code: number | null, stdout: string, stderr: string) {
    const rawOutput = [stderr.trim(), stdout.trim()].filter(Boolean).join('\n');
    super(formatGoNgcError(args, code, stdout, stderr));
    this.name = 'GoNgcError';
    this.args = args;
    this.code = code;
    this.stdout = stdout;
    this.stderr = stderr;

    const match = rawOutput.match(/^([^\n]+?):(\d+):(\d+) - error ([A-Z0-9]+):\s+([^\n]+(?:\n\n[\s\S]+?(?=\n\/?.*?:\d+:\d+ - error|\n*$))?)/m);
    if (match) {
      const file = match[1];
      const line = parseInt(match[2], 10);
      const column = parseInt(match[3], 10);
      const rawMessage = match[5].trim();
      
      const frameMatch = rawMessage.match(/(^\d+\s+.*[\s\S]*)/m);
      let frame = undefined;
      let message = match[4] + ': ' + rawMessage;
      
      if (frameMatch) {
        frame = frameMatch[1];
        message = match[4] + ': ' + rawMessage.substring(0, frameMatch.index).trim();
      }

      this.message = message;
      this.id = file;
      this.loc = { file, line, column };
      this.frame = frame;
    }
  }
}

const angularPartialDeclarationMarker = 'ɵɵngDeclare';
let linkCache = new BoundedMap<string, { key: string; code: string }>(500);

let sharedDaemonClient: GoNgcClient | null = null;
let sharedDaemonContextId = '';
// B#2 FIX: pluginMode is now set per-plugin-instance from options (see angularGoCompile).
// The module-level variable is kept only as a shared reference for angularGoLinker.
let pluginMode: 'server' | 'memory' | 'default' = 'server';
let memoryOutputs = new Map<string, OutputFile>();

const sourceFilePattern = /\.(ts|html|css|scss|sass|less)$/;
const projectInputPattern = /\.(ts|html|css|scss|sass|less|json)$/;

function cleanId(id: string): string {
  return id.split('?', 1)[0];
}

function hashText(text: string): string {
  return crypto.createHash('sha1').update(text).digest('hex');
}

function statKey(filePath: string): string {
  const stat = fs.statSync(filePath);
  return `${stat.mtimeMs}:${stat.size}`;
}

function optionalStatKey(filePath: string): string {
  try {
    return statKey(filePath);
  } catch {
    return 'missing';
  }
}

function findPackageJson(startDir: string, packageName: string): string | null {
  let dir = startDir;
  while (true) {
    const candidate = path.join(dir, 'node_modules', packageName, 'package.json');
    if (fs.existsSync(candidate)) {
      return candidate;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      return null;
    }
    dir = parent;
  }
}

function packageVersion(startDir: string, packageName: string): string {
  const packageJson = findPackageJson(startDir, packageName);
  if (packageJson == null) {
    return 'unknown';
  }
  try {
    return JSON.parse(fs.readFileSync(packageJson, 'utf8')).version || 'unknown';
  } catch {
    return 'unknown';
  }
}

function shouldSkipProjectInput(filePath: string, projectRoot: string, outDir: string): boolean {
  const rel = path.relative(projectRoot, filePath);
  if (rel.startsWith('..') || path.isAbsolute(rel)) {
    return true;
  }
  const parts = rel.split(path.sep);
  if (parts.includes('node_modules') || parts.includes('.angular') || parts.includes('.vite') || parts.includes('dist')) {
    return true;
  }
  const outRel = path.relative(outDir, filePath);
  return !outRel.startsWith('..') && !path.isAbsolute(outRel);
}



function compilerSalt(compilerPath: string): string {
  return `${compilerPath}:${optionalStatKey(compilerPath)}`;
}

function formatGoNgcError(args: string[], code: number | null, stdout: string, stderr: string): string {
  const output = [stderr.trim(), stdout.trim()].filter(Boolean).join('\n');
  const command = `go-ngc ${args.join(' ')}`;
  if (output) {
    return `${command} failed with code ${code ?? 'unknown'}\n${output}`;
  }
  return `${command} failed with code ${code ?? 'unknown'}`;
}

function runGoNgc(projectRoot: string, compilerPath: string, args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    const proc = spawn(compilerPath, args, {
      cwd: projectRoot,
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    const stdoutChunks: Buffer[] = [];
    const stderrChunks: Buffer[] = [];

    proc.stdout!.on('data', (data) => {
      stdoutChunks.push(data);
    });
    proc.stderr!.on('data', (data) => {
      stderrChunks.push(data);
    });
    proc.on('error', reject);
    proc.on('close', (code) => {
      const stdout = Buffer.concat(stdoutChunks).toString('utf8');
      const stderr = Buffer.concat(stderrChunks).toString('utf8');
      if (code === 0) {
        resolve(stdout);
      } else {
        reject(new GoNgcError(args, code, stdout, stderr));
      }
    });
  });
}

export function angularGoCompile(options: AngularGoCompileOptions = {}): any {
  let config: ResolvedConfig;
  let server: ViteDevServer | undefined;
  let projectRoot = '';
  let outDir = '';
  let compilerPath = options.compilerPath || 'go-ngc';
  let compileArgs: string[] = [];
  let compilePromise: Promise<void> | null = null;
  let lastCompileKey = '';
  let projectInputsCache: Set<string> | null = null;
  const recompileOnChange = options.recompileOnChange !== false;
  let enableHmr = false;

  // B#2 FIX: Read mode from options (defaults to 'server').
  // This also updates the module-level pluginMode used by the linker.
  pluginMode = (options.mode as 'server' | 'memory' | 'default') ?? 'server';

  function getProjectInputs(projectRoot: string, outDir: string): string[] {
    if (projectInputsCache != null) {
      const files = Array.from(projectInputsCache);
      files.sort();
      return files;
    }
    
    const files: string[] = [];
    const walk = (dir: string) => {
      let entries: fs.Dirent[];
      try {
        entries = fs.readdirSync(dir, { withFileTypes: true });
      } catch {
        return;
      }
      for (const entry of entries) {
        const filePath = path.join(dir, entry.name);
        if (shouldSkipProjectInput(filePath, projectRoot, outDir)) {
          continue;
        }
        if (entry.isDirectory()) {
          walk(filePath);
        } else if (entry.isFile() && projectInputPattern.test(filePath)) {
          files.push(filePath);
        }
      }
    };
    walk(projectRoot);
    projectInputsCache = new Set(files);
    files.sort();
    return files;
  }

  const htmlToTs = new Map<string, Set<string>>();
  const cssToTs = new Map<string, Set<string>>();
  const servedHmrRequests = new Set<string>();

  // B#11 FIX: Eagerly parse all .ts files to populate htmlToTs / cssToTs at
  // startup. Without this, style/template HMR misses the first change event
  // because the maps are only filled lazily during the load() hook.
  function parseStyleMappingsForFile(tsFilePath: string, sourceCode: string): void {
    const dir = path.dirname(tsFilePath);

    // templateUrl: '...'
    const templateMatch = sourceCode.match(/templateUrl\s*:\s*['"]([^'"]+)['"]/);
    if (templateMatch) {
      const htmlPath = path.resolve(dir, templateMatch[1]);
      if (!htmlToTs.has(htmlPath)) htmlToTs.set(htmlPath, new Set());
      htmlToTs.get(htmlPath)!.add(tsFilePath);
    }

    // styleUrl: '...' (Angular 17+)
    const styleMatch = sourceCode.match(/styleUrl\s*:\s*['"]([^'"]+)['"]/);
    if (styleMatch) {
      const cssPath = path.resolve(dir, styleMatch[1]);
      if (!cssToTs.has(cssPath)) cssToTs.set(cssPath, new Set());
      cssToTs.get(cssPath)!.add(tsFilePath);
    }

    // styleUrls: ['...', '...']  (multi-line safe)
    const styleUrlsMatch = sourceCode.match(/styleUrls\s*:\s*\[([^\]]+)\]/s);
    if (styleUrlsMatch) {
      const urls = styleUrlsMatch[1].matchAll(/['"]([^'"]+)['"]/g);
      for (const [, url] of urls) {
        const cssPath = path.resolve(dir, url);
        if (!cssToTs.has(cssPath)) cssToTs.set(cssPath, new Set());
        cssToTs.get(cssPath)!.add(tsFilePath);
      }
    }
  }

  async function scanProjectForStyleMappings(): Promise<void> {
    const walk = async (dir: string) => {
      let entries: fs.Dirent[];
      try { entries = await fs.promises.readdir(dir, { withFileTypes: true }); }
      catch { return; }

      await Promise.all(
        entries.map(async (entry) => {
          const filePath = path.join(dir, entry.name);
          if (shouldSkipProjectInput(filePath, projectRoot, outDir)) return;
          if (entry.isDirectory()) {
            await walk(filePath);
            return;
          }
          if (!entry.isFile() || !filePath.endsWith('.ts') || filePath.endsWith('.d.ts')) return;
          try {
            const source = await fs.promises.readFile(filePath, 'utf8');
            parseStyleMappingsForFile(filePath, source);
          } catch { /* ignore unreadable files */ }
        })
      );
    };
    await walk(projectRoot);
  }

  async function sendHmrUpdatesForFiles(devServer: ViteDevServer, tsFiles: Iterable<string>): Promise<void> {
    if (!enableHmr || !sharedDaemonClient) {
      return;
    }

    const relFiles = new Set<string>();
    for (const tsFile of tsFiles) {
      relFiles.add(path.relative(projectRoot, cleanId(tsFile)).replace(/\\/g, '/'));
    }

    const { componentIds } = await sharedDaemonClient.listHmrUpdates(sharedDaemonContextId);
    const timestamp = Date.now();

    for (const componentId of componentIds) {
      const atIdx = componentId.lastIndexOf('@');
      const componentFile = atIdx >= 0 ? componentId.slice(0, atIdx) : componentId;
      if (!relFiles.has(componentFile)) {
        continue;
      }

      for (const mod of devServer.moduleGraph.idToModuleMap.values()) {
        if (mod.id && mod.id.includes('/@ng/component') && mod.id.includes(`c=${encodeURIComponent(componentId)}`)) {
          devServer.moduleGraph.invalidateModule(mod);
        }
      }

      devServer.ws.send({
        type: 'custom',
        event: 'angular:component-update',
        data: {
          id: encodeURIComponent(componentId),
          timestamp,
        },
      });
    }
  }

  function invalidateTsModulesForFiles(tsFiles: Iterable<string>): void {
    if (!server) {
      return;
    }
    for (const tsFile of tsFiles) {
      const tsMods = server.moduleGraph.getModulesByFile(cleanId(tsFile));
      if (!tsMods) {
        continue;
      }
      for (const mod of tsMods) {
        server.moduleGraph.invalidateModule(mod);
      }
    }
  }

  function invalidateTsModulesForResource(filePath: string): void {
    if (filePath.endsWith('.html') && htmlToTs.has(filePath)) {
      invalidateTsModulesForFiles(htmlToTs.get(filePath)!);
      return;
    }
    if (
      (filePath.endsWith('.css') || filePath.endsWith('.scss') ||
       filePath.endsWith('.less') || filePath.endsWith('.sass')) &&
      cssToTs.has(filePath)
    ) {
      invalidateTsModulesForFiles(cssToTs.get(filePath)!);
    }
  }


  function log(message: string) {
    if (options.verbose) {
      config.logger.info(`[angular-go:compile] ${message}`);
    }
  }

  function resolveFromProjectRoot(value: string): string {
    if (path.isAbsolute(value)) {
      return value;
    }
    return path.resolve(projectRoot, value);
  }

  function normalizeCompileArgs(args: string[]): string[] {
    const pathValueFlags = new Set(['-p', '--project']);
    const normalized: string[] = [];

    for (let i = 0; i < args.length; i++) {
      const arg = args[i];
      normalized.push(arg);

      if (pathValueFlags.has(arg) && i + 1 < args.length) {
        normalized.push(resolveFromProjectRoot(args[i + 1]));
        i++;
      }
    }

    return normalized;
  }

  let compileError: Error | null = null;
  // H8 FIX: Cache the projectCompileKey to avoid re-running statSync() for every
  // file on every load() call. Invalidated by watchChange events.
  let cachedCompileKey = '';
  let cachedInputStats = new Map<string, string>();
  let lastInputStats = new Map<string, string>();
  let compileKeyDirty = true;

  function computeProjectState(): { key: string; stats: Map<string, string> } {
    const hash = crypto.createHash('sha1');
    const stats = new Map<string, string>();
    hash.update(compilerSalt(compilerPath));
    hash.update(JSON.stringify(compileArgs));
    hash.update(packageVersion(projectRoot, '@angular/core'));
    hash.update(packageVersion(projectRoot, '@angular/compiler-cli'));
    for (const filePath of getProjectInputs(projectRoot, outDir)) {
      const key = optionalStatKey(filePath);
      stats.set(filePath, key);
      hash.update(path.relative(projectRoot, filePath));
      hash.update(key);
    }
    return { key: hash.digest('hex'), stats };
  }

  function changedInputs(nextStats: Map<string, string>): string[] {
    const changed: string[] = [];
    for (const [filePath, key] of nextStats) {
      if (lastInputStats.get(filePath) !== key) {
        changed.push(filePath);
      }
    }
    for (const filePath of lastInputStats.keys()) {
      if (!nextStats.has(filePath)) {
        changed.push(filePath);
      }
    }
    return changed;
  }

  async function invalidateChangedInputs(files: string[]): Promise<void> {
    if (!sharedDaemonClient) {
      return;
    }
    for (const filePath of files) {
      if (projectInputPattern.test(filePath) && !shouldSkipProjectInput(filePath, projectRoot, outDir)) {
        await sharedDaemonClient.invalidate(sharedDaemonContextId, filePath);
      }
    }
  }

  async function ensureFreshDevCompile(reason: string): Promise<void> {
    if (config.command !== 'serve' || !recompileOnChange) {
      return;
    }
    const nextState = computeProjectState();
    if (nextState.key === lastCompileKey) {
      return;
    }
    await invalidateChangedInputs(changedInputs(nextState.stats));
    cachedCompileKey = nextState.key;
    cachedInputStats = nextState.stats;
    compileKeyDirty = false;
    await compileProject(reason, true);
  }

  async function compileProject(reason: string, force = false): Promise<void> {
    // C5 FIX: When force=true and a non-forced compile is already running,
    // wait for it to finish then start a fresh one instead of returning early.
    if (compilePromise) {
      if (!force) {
        return compilePromise;
      }
      // Wait for current compile to finish, then fall through to re-compile.
      await compilePromise.catch(() => {});
    }

    // H8 FIX: Only recompute compile key when dirty.
    if (compileKeyDirty) {
      const state = computeProjectState();
      cachedCompileKey = state.key;
      cachedInputStats = state.stats;
      compileKeyDirty = false;
    }
    const compileKey = cachedCompileKey;
    if (!force && compileKey === lastCompileKey && fs.existsSync(outDir)) {
      log(`skipped compile for unchanged inputs (${reason})`);
      return;
    }

    const started = Date.now();
    compileError = null;
    let buildOperation: Promise<any>;
    if (pluginMode === 'server' && sharedDaemonClient && sharedDaemonContextId) {
      buildOperation = sharedDaemonClient.build(sharedDaemonContextId);
    } else {
      buildOperation = runGoNgc(projectRoot, compilerPath, compileArgs).then((stdout) => {
        if (compileArgs.includes('--format=json')) {
          return JSON.parse(stdout);
        }
        return { outputs: [] };
      });
    }

    compilePromise = buildOperation
      .then((result: BuildResult) => {
        lastCompileKey = compileKey;
        lastInputStats = new Map(cachedInputStats);
        if (pluginMode === 'memory' || pluginMode === 'server') {
          try {
            memoryOutputs.clear();
            if (result.outputs) {
              for (const output of result.outputs) {
                const absPath = path.resolve(projectRoot, output.path);
                memoryOutputs.set(absPath, output);
              }
            }
          } catch (e) {
            console.error('Failed to parse in-memory outputs', e);
          }
        }
        log(`compile completed in ${Date.now() - started}ms`);
      })
      .catch((e) => {
        compileError = e;
        throw e;
      })
      .finally(() => {
        compilePromise = null;
      });

    return compilePromise;
  }

  function projectCompileKey(): string {
    return computeProjectState().key;
  }

  function outputPathForSource(id: string): string | null {
    const filePath = cleanId(id);
    if (!filePath.endsWith('.ts') || filePath.endsWith('.d.ts') || filePath.includes(`${path.sep}node_modules${path.sep}`)) {
      return null;
    }

    const relativePath = path.relative(projectRoot, filePath);
    if (relativePath.startsWith('..') || path.isAbsolute(relativePath)) {
      return null;
    }

    return path.join(outDir, relativePath).replace(/\.ts$/, '.js');
  }

  return {
    name: 'vite-plugin-angular-go-compile',
    enforce: 'pre',

    configResolved(resolvedConfig: ResolvedConfig) {
      config = resolvedConfig;
      projectRoot = options.projectRoot ? path.resolve(options.projectRoot) : process.cwd();
      outDir = resolveFromProjectRoot(options.outDir || 'out-tsc/app');
      compilerPath = resolveGoNgcPath(projectRoot, compilerPath, import.meta.url);
      compileArgs = normalizeCompileArgs(options.args || ['-p', options.project || 'tsconfig.app.json']);
      // Keep server mode for production build to avoid spawning subprocesses
      // for linking and compilation.
      if (config.command === 'build' && pluginMode === 'server') {
        // pluginMode = 'default';
      }
      
      enableHmr = config.command === 'serve' && options.hmr === true;
      const compilationMode = options.compilationMode || 'global';

      if (!compileArgs.find(arg => arg.startsWith('--compilationMode'))) {
        compileArgs.push(`--compilationMode=${compilationMode}`);
      }

      if (enableHmr) {
        if (!compileArgs.includes('--hmr')) {
          compileArgs.push('--hmr');
        }
      } else {
        compileArgs = compileArgs.filter(arg => arg !== '--hmr');
      }

      if (pluginMode === 'memory') {
        if (!compileArgs.some(arg => arg === '--write=false' || arg === '--write' || arg.startsWith('--write='))) {
          compileArgs.push('--write=false');
        }
        if (!compileArgs.some(arg => arg === '--format=json' || arg === '--format' || arg.startsWith('--format='))) {
          compileArgs.push('--format=json');
        }
      }
    },

    configureServer(devServer: ViteDevServer) {
      server = devServer;

      // ─── HMR component middleware ───────────────────────────────────────
      // When Angular does `import(/* @vite-ignore */ ɵɵgetReplaceMetadataURL(...))`,
      // Vite bypasses its resolveId/load hooks and the browser sends a raw HTTP
      // GET request. We intercept it here and serve the HMR module directly.
      devServer.middlewares.use(async (req: any, res: any, next: any) => {
        if (!enableHmr) {
          return next();
        }
        const url: string = req.url || '';
        const qIdx = url.indexOf('?');
        const pathname = qIdx >= 0 ? url.slice(0, qIdx) : url;

        if (!pathname.endsWith('/@ng/component')) {
          return next();
        }

        const qs = qIdx >= 0 ? url.slice(qIdx + 1) : '';
        const params = new URLSearchParams(qs);
        const c = params.get('c');

        if (!c || !sharedDaemonClient) {
          res.writeHead(404, { 'Content-Type': 'text/plain' });
          res.end('HMR component not found');
          return;
        }

        try {
          const result = await devServer.transformRequest(url);
          if (result && result.code) {
            res.writeHead(200, {
              'Content-Type': 'application/javascript',
              'Cache-Control': 'no-cache, no-store, must-revalidate',
            });
            res.end(result.code);
            return;
          }
          res.writeHead(200, { 'Content-Type': 'application/javascript' });
          res.end('export default null;');
        } catch (e: any) {
          devServer.config.logger.warn(
            `[angular-go] Failed to serve transformed HMR update for ${url}: ${e.message}`
          );
          res.writeHead(200, { 'Content-Type': 'application/javascript' });
          res.end('export default null;');
        }
      });
      // ────────────────────────────────────────────────────────────────────

      server.watcher.on('add', (filePath) => {
        if (projectInputsCache && projectInputPattern.test(filePath) && !shouldSkipProjectInput(filePath, projectRoot, outDir)) {
          projectInputsCache.add(filePath);
        }
      });
      server.watcher.on('unlink', (filePath) => {
        if (projectInputsCache) {
          projectInputsCache.delete(filePath);
        }
      });
    },

    watchChange(id: string, change: { event: 'create' | 'update' | 'delete' }) {
      if (!projectInputsCache || !projectInputPattern.test(id) || shouldSkipProjectInput(id, projectRoot, outDir)) {
        return;
      }
      // H8 FIX: Mark compile key dirty when a project file changes.
      compileKeyDirty = true;
      if (change.event === 'delete') {
        projectInputsCache.delete(id);
      } else {
        projectInputsCache.add(id);
      }
      invalidateTsModulesForResource(cleanId(id));
    },

    async buildStart() {
      const isServe = config.command === 'serve';

      // C3 FIX: Only start the daemon when running the dev server.
      // Production builds (vite build) must NOT start the daemon:
      //   1. The daemon is designed for long-lived watch sessions, not one-shot builds.
      //   2. Sending hmr:true to the daemon causes HMR scaffolding to be emitted
      //      into the production bundle (~5-10 KB per component).
      // Start the daemon when pluginMode is 'server' (both for serve and build)
      if (pluginMode === 'server' && (!sharedDaemonClient || !sharedDaemonContextId)) {
        if (!sharedDaemonClient) {
          sharedDaemonClient = createGoNgcClient(compilerPath, projectRoot);
        }
        sharedDaemonContextId = await sharedDaemonClient.createContext({
          project: options.project || 'tsconfig.app.json',
          compilationMode: options.compilationMode || 'global',
          // C3 FIX: Only enable HMR when explicitly requested in serve mode.
          hmr: enableHmr,
        });

        // M1 FIX: Register SIGINT/SIGTERM handlers so the daemon is cleanly shut down
        // even when the user presses Ctrl+C during a build/serve session.
        const shutdown = async () => {
          if (sharedDaemonClient) {
            await sharedDaemonClient.close().catch(() => {});
            sharedDaemonClient = null;
            sharedDaemonContextId = '';
          }
          process.exit(0);
        };
        process.once('SIGINT', shutdown);
        process.once('SIGTERM', shutdown);
      }

      // Eagerly scan all project .ts files so htmlToTs / cssToTs are populated
      // before the first file-change event arrives. This is required for both
      // Angular HMR and the non-HMR full-reload path for external templates/styles.
      if (isServe && recompileOnChange) {
        await scanProjectForStyleMappings();
      }

      for (const filePath of getProjectInputs(projectRoot, outDir)) {
        this.addWatchFile(filePath);
      }
      try {
        await compileProject('startup');
      } catch (e: any) {
        if (config.command === 'build') {
          this.error(e);
        } else {
          config.logger.error(e.message);
        }
      }
    },

    resolveId(id: string) {
      if (!enableHmr) {
        return;
      }
      // Split on '?' so that '.ts' inside query params like
      // c=src%2Fapp%2Fapp.ts%40App does NOT trigger the old
      // `!id.includes('.ts')` guard and cause a 404.
      // Example: id = '/app/@ng/component?c=src%2Fapp%2Fapp.ts%40App&t=123'
      //          pathPart = '/app/@ng/component'
      const pathPart = id.split('?')[0];
      if (pathPart.endsWith('/@ng/component') || pathPart.endsWith('/@ng/component.ts')) {
        // Return the full id (including query string) as a virtual module.
        // The \0 prefix tells Vite/Rollup this is a virtual module and to skip
        // file-system resolution — it will call our load() hook instead.
        return '\0' + id;
      }
    },

    async load(id: string) {
      if (id.startsWith('\0') && id.includes('/@ng/component')) {
        if (!enableHmr) {
          return null;
        }
        const idWithoutNull = id.slice(1); // remove leading \0
        const urlObj = new URL(idWithoutNull, 'http://localhost');
        // URLSearchParams.get() already URL-decodes %2F→/ and %40→@
        // so c is the raw "relPath@ClassName" string, e.g. "src/app/app.ts@App"
        let c = urlObj.searchParams.get('c');
        if (c) {
          c = decodeURIComponent(c);
          const atIdx = c.lastIndexOf('@');
          const filePath = atIdx !== -1 ? c.slice(0, atIdx) : c;
          const compName = atIdx !== -1 ? c.slice(atIdx + 1) : '';
          if (filePath && compName) {
            if (pluginMode === 'server' && sharedDaemonClient) {
              try {
                const result = await (sharedDaemonClient as GoNgcClient).getHmrUpdate(sharedDaemonContextId, c);
                if (result && result.code) {
                  return { code: result.code, map: null };
                }
                // Daemon responded but no update available yet — return a no-op
                // update so Angular doesn't crash with a failed import.
                log(`HMR update not yet available for ${c}; returning no-op`);
              } catch (e: any) {
                config.logger.warn(`[angular-go] Failed to get HMR update for ${c}: ${e.message}`);
              }
            }
          }
        }
        // No HMR code available: return a no-op module so the dynamic import
        // succeeds (browser doesn't get ERR_ABORTED) and Angular gracefully
        // keeps the existing component state.
        return { code: 'export default null;', map: null };
      }

      const jsPath = outputPathForSource(id);
      if (jsPath == null) {
        return null;
      }

      await ensureFreshDevCompile(`freshness check for ${path.relative(projectRoot, cleanId(id))}`);

      if (pluginMode !== 'server' && pluginMode !== 'memory') {
        if (!fs.existsSync(jsPath)) {
          try {
            await compileProject(`missing output for ${path.relative(projectRoot, cleanId(id))}`, true);
          } catch (e) {
            // ignore here, we will throw compileError below
          }
        }
      } else {
        if (!memoryOutputs.has(jsPath)) {
          try {
            await compileProject(`missing memory output for ${path.relative(projectRoot, cleanId(id))}`, true);
          } catch (e) {
            // ignore here
          }
        }
      }

      if (compileError) {
        this.error(compileError);
      }

      if (pluginMode !== 'server' && pluginMode !== 'memory') {
        if (!fs.existsSync(jsPath)) {
          this.error(`go-ngc did not emit ${jsPath} for ${cleanId(id)}`);
        }
      }

      const sourcePath = cleanId(id);
      if (sourcePath.endsWith('.ts')) {
        const sourceCode = fs.readFileSync(sourcePath, 'utf8');
        const templateMatch = sourceCode.match(/templateUrl\s*:\s*['"]([^'"]+)['"]/);
        if (templateMatch) {
          const htmlPath = path.resolve(path.dirname(sourcePath), templateMatch[1]);
          if (!htmlToTs.has(htmlPath)) htmlToTs.set(htmlPath, new Set());
          htmlToTs.get(htmlPath)!.add(id);
        }
        const styleMatch = sourceCode.match(/styleUrl\s*:\s*['"]([^'"]+)['"]/);
        if (styleMatch) {
          const cssPath = path.resolve(path.dirname(sourcePath), styleMatch[1]);
          if (!cssToTs.has(cssPath)) cssToTs.set(cssPath, new Set());
          cssToTs.get(cssPath)!.add(id);
        }
        const styleUrlsMatch = sourceCode.match(/styleUrls\s*:\s*\[([^\]]+)\]/);
        if (styleUrlsMatch) {
          const urls = styleUrlsMatch[1].match(/['"]([^'"]+)['"]/g);
          if (urls) {
            for (const url of urls) {
              const cleanUrl = url.replace(/['"]/g, '');
              const cssPath = path.resolve(path.dirname(sourcePath), cleanUrl);
              if (!cssToTs.has(cssPath)) cssToTs.set(cssPath, new Set());
              cssToTs.get(cssPath)!.add(id);
            }
          }
        }
      }

      let map = null;
      let code: string | null = null;
      const mapPath = jsPath + '.map';
      if (pluginMode === 'server' || pluginMode === 'memory') {
        const memOut = memoryOutputs.get(jsPath);
        if (memOut) {
          code = memOut.text;
        }
        const memMap = memoryOutputs.get(mapPath);
        if (memMap) {
          map = JSON.parse(memMap.text);
        }
      }

      if (code == null) {
        if (fs.existsSync(mapPath)) {
          try {
            map = JSON.parse(fs.readFileSync(mapPath, 'utf8'));
          } catch {
            // ignore
          }
        }
        code = fs.readFileSync(jsPath, 'utf8');
      }
      if (enableHmr) {
        code = code.replace(/import\s*\(\s*i0\.ɵɵgetReplaceMetadataURL/g, 'import(/* @vite-ignore */ i0.ɵɵgetReplaceMetadataURL');
      }
      if (map && sourcePath.endsWith('.ts') && fs.existsSync(sourcePath)) {
        try {
          const sourceCodeForMap = fs.readFileSync(sourcePath, 'utf8');
          map.sources = [sourcePath];
          map.sourcesContent = [sourceCodeForMap];
          map.sourceRoot = undefined;
        } catch {
          // Keep the compiler-provided map when the source cannot be read.
        }
      }

      return { code, map };
    },

    async handleHotUpdate({ file: filePath, server }: any) {
      if (!recompileOnChange) {
        return;
      }

      filePath = cleanId(filePath);
      if (!sourceFilePattern.test(filePath) || filePath.includes(`${path.sep}node_modules${path.sep}`)) {
        return;
      }

      const relativePath = path.relative(projectRoot, filePath);
      if (relativePath.startsWith('..') || path.isAbsolute(relativePath)) {
        return;
      }

      // Check 1: Component Template (.html)
      if (filePath.endsWith('.html') && htmlToTs.has(filePath)) {
        log(`handleHotUpdate template changed: ${relativePath}`);
        if (sharedDaemonClient) {
          // Invalidate the HTML file on the daemon
          await sharedDaemonClient.invalidate(sharedDaemonContextId, filePath);
          // Invalidate all associated TS files on the daemon so it re-reads them
          for (const tsFile of htmlToTs.get(filePath)!) {
            await sharedDaemonClient.invalidate(sharedDaemonContextId, tsFile);
          }
        }

        // Force compile the project immediately to build the new template and HMR code
        await compileProject(`change in template ${relativePath}`, true);

        if (!enableHmr) {
          server?.ws.send({ type: 'full-reload', path: '*' });
          return [];
        }

        // Send HMR update event to browser and invalidate module graph cache for virtual modules
        if (server) {
          const affectedTsFiles = htmlToTs.get(filePath)!;
          invalidateTsModulesForFiles(affectedTsFiles);
          await sendHmrUpdatesForFiles(server, affectedTsFiles);
        }
        return [];
      }

      // Check 2: Component Styles (.css, .scss, etc.)
      if (
        (filePath.endsWith('.css') || filePath.endsWith('.scss') ||
         filePath.endsWith('.less') || filePath.endsWith('.sass')) &&
        cssToTs.has(filePath)
      ) {
        log(`handleHotUpdate style changed: ${relativePath}`);
        if (sharedDaemonClient) {
          // Invalidate style file on the daemon
          await sharedDaemonClient.invalidate(sharedDaemonContextId, filePath);
          // Invalidate TS files on the daemon so it re-reads them
          for (const tsFile of cssToTs.get(filePath)!) {
            await sharedDaemonClient.invalidate(sharedDaemonContextId, tsFile);
          }
        }

        // Force compile the project immediately to build the new styles and HMR code
        await compileProject(`change in style ${relativePath}`, true);

        if (!enableHmr) {
          server?.ws.send({ type: 'full-reload', path: '*' });
          return [];
        }

        // Send HMR update event to browser and invalidate module graph cache for virtual modules
        if (server) {
          const affectedTsFiles = cssToTs.get(filePath)!;
          invalidateTsModulesForFiles(affectedTsFiles);
          await sendHmrUpdatesForFiles(server, affectedTsFiles);
        }
        return [];
      }

      // Default: Other files (e.g. .ts files or global styles) require recompilation
      try {
        if (sharedDaemonClient) {
          await sharedDaemonClient.invalidate(sharedDaemonContextId, filePath);
        }
        await compileProject(`change in ${relativePath}`);
      } catch (e: any) {
        config.logger.error(e.message);
        server?.ws.send({
          type: 'error',
          err: {
            message: e.message,
            stack: '',
            plugin: 'vite-plugin-angular-go',
            id: e.id,
            loc: e.loc,
            frame: e.frame
          }
        });
        return;
      }

      if (filePath.endsWith('.ts')) {
        if (!enableHmr) {
          server?.ws.send({ type: 'full-reload', path: '*' });
          return [];
        }
        try {
          const sourceCode = fs.readFileSync(filePath, 'utf8');
          parseStyleMappingsForFile(filePath, sourceCode);
          await sendHmrUpdatesForFiles(server, [filePath]);
        } catch {
          // ignore read errors
        }
        return [];
      } else {
        const mod = server.moduleGraph.getModuleById(filePath);
        if (mod) {
          return [mod];
        }
      }
    },

    async generateBundle(options: any, bundle: any) {
      if (config.command !== 'build') {
        return;
      }

      const initialChunks = new Set<string>();
      const lazyChunks = new Set<string>();
      const initialCss = new Set<string>();
      const lazyCss = new Set<string>();

      // 1. Identify entry chunks and recursively trace their static imports
      for (const [fileName, itemVal] of Object.entries(bundle)) {
        const item = itemVal as any;
        if (item.type === 'chunk' && item.isEntry) {
          initialChunks.add(fileName);
          const addImports = (imports: string[]) => {
            for (const imp of imports) {
              if (!initialChunks.has(imp)) {
                initialChunks.add(imp);
                const child = bundle[imp] as any;
                if (child && child.type === 'chunk') {
                  addImports(child.imports);
                }
              }
            }
          };
          addImports(item.imports);
        }
      }

      // 2. Classify other chunks as lazy
      for (const [fileName, itemVal] of Object.entries(bundle)) {
        const item = itemVal as any;
        if (item.type === 'chunk' && !initialChunks.has(fileName)) {
          lazyChunks.add(fileName);
        }
      }

      // 3. Classify CSS assets
      for (const [fileName, itemVal] of Object.entries(bundle)) {
        const item = itemVal as any;
        if (item.type === 'asset' && fileName.endsWith('.css')) {
          let importedByInitial = false;
          let importedByLazy = false;

          for (const chunkName of initialChunks) {
            const chunk = bundle[chunkName] as any;
            if (chunk && chunk.viteMetadata?.importedCss?.has(fileName)) {
              importedByInitial = true;
            }
          }

          for (const chunkName of lazyChunks) {
            const chunk = bundle[chunkName] as any;
            if (chunk && chunk.viteMetadata?.importedCss?.has(fileName)) {
              importedByLazy = true;
            }
          }

          if (importedByInitial || !importedByLazy) {
            initialCss.add(fileName);
          } else {
            lazyCss.add(fileName);
          }
        }
      }

      // 4. Compute sizes
      interface SizeRow {
        file: string;
        name: string;
        rawSize: number;
        gzipSize: number;
        type: 'js' | 'css';
      }

      const initialRows: SizeRow[] = [];
      const lazyRows: SizeRow[] = [];

      const formatBytes = (bytes: number): string => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'kB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
      };

      const colorSize = (sizeStr: string, bytes: number): string => {
        if (bytes < 100 * 1024) {
          return `\x1b[32m${sizeStr}\x1b[39m`; // Green
        } else if (bytes < 500 * 1024) {
          return `\x1b[33m${sizeStr}\x1b[39m`; // Yellow
        } else {
          return `\x1b[31m\x1b[1m${sizeStr}\x1b[22m\x1b[39m`; // Bold Red
        }
      };

      const isEmptyCssEntryChunk = (item: any): boolean => {
        return item?.type === 'chunk'
          && item.isEntry
          && item.name?.startsWith('style-')
          && Buffer.byteLength(item.code || '', 'utf8') === 0
          && item.viteMetadata?.importedCss?.size > 0;
      };

      // JS initial
      for (const fileName of initialChunks) {
        const item = bundle[fileName] as any;
        if (item && item.type === 'chunk') {
          if (isEmptyCssEntryChunk(item)) {
            continue;
          }
          const code = item.code;
          const rawSize = Buffer.byteLength(code, 'utf8');
          const gzipSize = zlib.gzipSync(code).byteLength;
          initialRows.push({
            file: fileName,
            name: item.name || 'main',
            rawSize,
            gzipSize,
            type: 'js'
          });
        }
      }

      // CSS initial
      for (const fileName of initialCss) {
        const item = bundle[fileName] as any;
        if (item && item.type === 'asset') {
          const source = item.source;
          const rawSize = typeof source === 'string' ? Buffer.byteLength(source, 'utf8') : source.byteLength;
          const gzipSize = zlib.gzipSync(source).byteLength;
          initialRows.push({
            file: fileName,
            name: path.basename(fileName, '.css'),
            rawSize,
            gzipSize,
            type: 'css'
          });
        }
      }

      // JS lazy
      for (const fileName of lazyChunks) {
        const item = bundle[fileName] as any;
        if (item && item.type === 'chunk') {
          const code = item.code;
          const rawSize = Buffer.byteLength(code, 'utf8');
          const gzipSize = zlib.gzipSync(code).byteLength;
          lazyRows.push({
            file: fileName,
            name: item.name || 'lazy',
            rawSize,
            gzipSize,
            type: 'js'
          });
        }
      }

      // CSS lazy
      for (const fileName of lazyCss) {
        const item = bundle[fileName] as any;
        if (item && item.type === 'asset') {
          const source = item.source;
          const rawSize = typeof source === 'string' ? Buffer.byteLength(source, 'utf8') : source.byteLength;
          const gzipSize = zlib.gzipSync(source).byteLength;
          lazyRows.push({
            file: fileName,
            name: path.basename(fileName, '.css'),
            rawSize,
            gzipSize,
            type: 'css'
          });
        }
      }

      // 5. Generate beautiful console output
      let initialRawTotal = 0;
      let initialGzipTotal = 0;
      for (const row of initialRows) {
        initialRawTotal += row.rawSize;
        initialGzipTotal += row.gzipSize;
      }

      const maxFileLen = Math.max(
        'Initial Chunk Files'.length,
        'Lazy Chunk Files'.length,
        ...initialRows.map(r => r.file.length),
        ...lazyRows.map(r => r.file.length)
      );
      const maxNameLen = Math.max(
        'Names'.length,
        ...initialRows.map(r => r.name.length),
        ...lazyRows.map(r => r.name.length)
      );

      const pad = (str: string, len: number) => str.padEnd(len);
      const padStart = (str: string, len: number) => str.padStart(len);

      const outputLines: string[] = [];
      outputLines.push(`\n\x1b[1m\x1b[36mAngular Initial Bundle Size Summary:\x1b[0m\n`);
      outputLines.push(
        `  ` +
        `\x1b[1m${pad('Initial Chunk Files', maxFileLen)}\x1b[22m | ` +
        `\x1b[1m${pad('Names', maxNameLen)}\x1b[22m | ` +
        `\x1b[1m${padStart('Raw Size', 12)}\x1b[22m | ` +
        `\x1b[1m${padStart('Estimated Transfer Size', 24)}\x1b[22m`
      );
      outputLines.push(
        `  ` +
        `\x1b[90m${'-'.repeat(maxFileLen)}\x1b[39m-+-` +
        `\x1b[90m${'-'.repeat(maxNameLen)}\x1b[39m-+-` +
        `\x1b[90m${'-'.repeat(12)}\x1b[39m-+-` +
        `\x1b[90m${'-'.repeat(24)}\x1b[39m`
      );

      for (const row of initialRows) {
        const fileStr = row.type === 'js' ? `\x1b[36m${row.file}\x1b[39m` : `\x1b[35m${row.file}\x1b[39m`;
        const nameStr = row.name;
        const rawSizeStr = formatBytes(row.rawSize);
        const gzipSizeStr = colorSize(formatBytes(row.gzipSize), row.gzipSize);
        
        outputLines.push(
          `  ` +
          `${pad(fileStr, maxFileLen + (fileStr.length - row.file.length))} | ` +
          `${pad(nameStr, maxNameLen)} | ` +
          `${padStart(rawSizeStr, 12)} | ` +
          `${padStart(gzipSizeStr, 24 + (gzipSizeStr.length - formatBytes(row.gzipSize).length))}`
        );
      }

      outputLines.push(
        `  ` +
        `\x1b[90m${'-'.repeat(maxFileLen)}\x1b[39m-+-` +
        `\x1b[90m${'-'.repeat(maxNameLen)}\x1b[39m-+-` +
        `\x1b[90m${'-'.repeat(12)}\x1b[39m-+-` +
        `\x1b[90m${'-'.repeat(24)}\x1b[39m`
      );

      const totalLabel = `\x1b[1mInitial Total\x1b[22m`;
      const totalRawStr = `\x1b[1m${formatBytes(initialRawTotal)}\x1b[22m`;
      const totalGzipStr = colorSize(formatBytes(initialGzipTotal), initialGzipTotal);

      outputLines.push(
        `  ` +
        `${pad('', maxFileLen)} | ` +
        `${pad(totalLabel, maxNameLen + (totalLabel.length - 'Initial Total'.length))} | ` +
        `${padStart(totalRawStr, 12 + (totalRawStr.length - formatBytes(initialRawTotal).length))} | ` +
        `${padStart(totalGzipStr, 24 + (totalGzipStr.length - formatBytes(initialGzipTotal).length))}`
      );

      if (lazyRows.length > 0) {
        outputLines.push(`\n  ` +
          `\x1b[1m${pad('Lazy Chunk Files', maxFileLen)}\x1b[22m | ` +
          `\x1b[1m${pad('Names', maxNameLen)}\x1b[22m | ` +
          `\x1b[1m${padStart('Raw Size', 12)}\x1b[22m | ` +
          `\x1b[1m${padStart('Estimated Transfer Size', 24)}\x1b[22m`
        );
        outputLines.push(
          `  ` +
          `\x1b[90m${'-'.repeat(maxFileLen)}\x1b[39m-+-` +
          `\x1b[90m${'-'.repeat(maxNameLen)}\x1b[39m-+-` +
          `\x1b[90m${'-'.repeat(12)}\x1b[39m-+-` +
          `\x1b[90m${'-'.repeat(24)}\x1b[39m`
        );

        for (const row of lazyRows) {
          const fileStr = `\x1b[90m${row.file}\x1b[39m`;
          const nameStr = `\x1b[90m${row.name}\x1b[39m`;
          const rawSizeStr = formatBytes(row.rawSize);
          const gzipSizeStr = colorSize(formatBytes(row.gzipSize), row.gzipSize);
          
          outputLines.push(
            `  ` +
            `${pad(fileStr, maxFileLen + (fileStr.length - row.file.length))} | ` +
            `${pad(nameStr, maxNameLen + (nameStr.length - row.name.length))} | ` +
            `${padStart(rawSizeStr, 12)} | ` +
            `${padStart(gzipSizeStr, 24 + (gzipSizeStr.length - formatBytes(row.gzipSize).length))}`
          );
        }
      }

      console.log(outputLines.join('\n') + '\n');
    },

    async closeBundle() {
      if (sharedDaemonClient) {
        await sharedDaemonClient.close();
        sharedDaemonClient = null;
        sharedDaemonContextId = '';
      }
    }
  };
}

export function angularGoLinker(options: AngularGoLinkerOptions = {}): any {
  let projectRoot = '';
  let compilerPath = options.compilerPath || 'go-ngc';
  const ascii = options.ascii !== false;
  const forceOptimizeDeps = options.forceOptimizeDeps !== false;
  const linkCache = new BoundedMap<string, LinkCacheEntry>(500);
  let cacheDir = '';

  function configurePaths() {
    projectRoot = options.projectRoot ? path.resolve(options.projectRoot) : process.cwd();
    compilerPath = resolveGoNgcPath(projectRoot, compilerPath, import.meta.url);
  }

  function ensureCacheDir(): string {
    if (cacheDir) return cacheDir;
    configurePaths();
    cacheDir = path.join(projectRoot, '.angular', 'cache', 'vite-plugin-angular-go');
    fs.mkdirSync(cacheDir, { recursive: true });
    return cacheDir;
  }

  function getCacheFilePath(key: string): string {
    return path.join(ensureCacheDir(), `${hashText(key)}.js`);
  }

  function readFromDiskCache(key: string): string | null {
    const cacheFile = getCacheFilePath(key);
    if (fs.existsSync(cacheFile)) {
      try {
        return fs.readFileSync(cacheFile, 'utf8');
      } catch {
        return null;
      }
    }
    return null;
  }

  function writeToDiskCache(key: string, code: string): void {
    const dir = ensureCacheDir();
    const cacheFile = getCacheFilePath(key);
    const hashedKey = hashText(key);
    try {
      const tempFile = path.join(dir, `${hashedKey}.${process.pid}.${Date.now()}-${Math.random().toString(36).slice(2)}.tmp`);
      fs.writeFileSync(tempFile, code, 'utf8');
      fs.renameSync(tempFile, cacheFile);
    } catch {
      // ignore
    }
  }

  function compilerCacheSalt(): string {
    configurePaths();
    return `${compilerSalt(compilerPath)}:${packageVersion(projectRoot, '@angular/core')}:${packageVersion(projectRoot, '@angular/compiler-cli')}`;
  }

  function fileCacheKey(filePath: string): string {
    return `${compilerCacheSalt()}:${statKey(filePath)}`;
  }

  function codeCacheKey(code: string): string {
    return hashText(`${compilerCacheSalt()}:${code}`);
  }

  async function linkFile(filePath: string): Promise<string> {
    const key = fileCacheKey(filePath);
    const cacheId = `file:${filePath}`;
    const cached = linkCache.get(cacheId);
    if (cached?.key === key) {
      return cached.code;
    }

    const diskCached = readFromDiskCache(key);
    if (diskCached != null) {
      linkCache.set(cacheId, { key, code: diskCached });
      return diskCached;
    }

    let linked: string;
    if (pluginMode === 'server') {
      if (!sharedDaemonClient) {
        sharedDaemonClient = createGoNgcClient(compilerPath, projectRoot);
      }
      const result = await sharedDaemonClient.linkFile(filePath);
      linked = result.code;
    } else {
      linked = await runGoNgc(projectRoot, compilerPath, ['--link', filePath]);
    }
    linkCache.set(cacheId, { key, code: linked });
    writeToDiskCache(key, linked);
    return linked;
  }

  async function linkCode(code: string, id: string): Promise<string> {
    const key = codeCacheKey(code);
    const cacheId = `code:${id}:${key}`;
    const cached = linkCache.get(cacheId);
    if (cached) {
      return cached.code;
    }

    const diskCached = readFromDiskCache(key);
    if (diskCached != null) {
      linkCache.set(cacheId, { key, code: diskCached });
      return diskCached;
    }

    let linked: string;
    if (pluginMode === 'server') {
      if (!sharedDaemonClient) {
        sharedDaemonClient = createGoNgcClient(compilerPath, projectRoot);
      }
      const result = await sharedDaemonClient.linkCode(code, id);
      linked = result.code;
    } else {
      const tempPath = path.join(
        os.tmpdir(),
        `angular-go-link-${process.pid}-${Date.now()}-${Math.random().toString(36).slice(2)}.mjs`,
      );

      fs.writeFileSync(tempPath, code);
      try {
        linked = await runGoNgc(projectRoot, compilerPath, ['--link', tempPath]);
      } finally {
        fs.rmSync(tempPath, { force: true });
      }
    }

    linkCache.set(cacheId, { key, code: linked });
    writeToDiskCache(key, linked);
    return linked;
  }



  async function linkIfNeeded(code: string, id: string): Promise<string | null> {
    if (!code.includes(angularPartialDeclarationMarker)) {
      return null;
    }

    const filePath = cleanId(id);
    if (fs.existsSync(filePath)) {
      return linkFile(filePath);
    }

    return linkCode(code, id);
  }

  return {
    name: 'vite-plugin-angular-go-linker',
    enforce: 'pre',

    config() {
      const linkerSalt = compilerCacheSalt();
      const isVite8 = viteVersion && viteVersion.startsWith('8');

      if (isVite8) {
        return {
          optimizeDeps: {
            force: forceOptimizeDeps,
            rolldownOptions: {
              transform: {
                define: {
                  __ANGULAR_GO_LINKER_CACHE_SALT__: JSON.stringify(linkerSalt),
                },
              },
              plugins: [
                {
                  name: 'angular-go-linker',
                  transform: async (code: string, id: string) => {
                    if (id.endsWith('.js') || id.endsWith('.mjs')) {
                      const linked = await linkIfNeeded(code, id);
                      if (linked == null) {
                        return null;
                      }
                      return { code: linked };
                    }
                    return null;
                  },
                },
              ],
            },
          },
        } as any;
      }

      return {
        esbuild: ascii ? { charset: 'ascii' } : undefined,
        optimizeDeps: {
          force: forceOptimizeDeps,
          esbuildOptions: {
            charset: ascii ? 'ascii' : undefined,
            define: {
              __ANGULAR_GO_LINKER_CACHE_SALT__: JSON.stringify(linkerSalt),
            },
            plugins: [
              {
                name: 'angular-go-linker',
                setup(build: any) {
                  build.onLoad({ filter: /\.m?js$/ }, async (args: any) => {
                    const code = fs.readFileSync(args.path, 'utf8');
                    const linked = await linkIfNeeded(code, args.path);
                    if (linked == null) {
                      return null;
                    }
                    return { contents: linked, loader: 'js' };
                  });
                },
              },
            ],
          },
        },
      };
    },

    configResolved() {
      configurePaths();
    },

    async transform(code: string, id: string) {
      const linked = await linkIfNeeded(code, id);
      if (linked == null) {
        return null;
      }
      return { code: linked, map: null };
    },

    handleHotUpdate(ctx: any) {
      if (!ctx.file.includes(`${path.sep}node_modules${path.sep}`)) {
        return;
      }
      linkCache.delete(`file:${ctx.file}`);
    },

    async closeBundle() {
      if (sharedDaemonClient) {
        await sharedDaemonClient.close().catch(() => {});
        sharedDaemonClient = null;
        sharedDaemonContextId = '';
      }
    },
  };
}

export default function angularGo(options: AngularGoOptions = {}): any[] {
  const plugins: any[] = [];

  if (options.compile !== false) {
    plugins.push(angularGoCompile(options));
  }

  if (options.link !== false) {
    plugins.push(angularGoLinker(options));
  }

  return plugins;
}
