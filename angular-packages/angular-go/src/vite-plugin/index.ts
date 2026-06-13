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
import { bundleCompilerOutputsWithRolldown } from '../builders/shared/app-bundler.js';

const isDebug = process.env.NG_GO_DEBUG === '1' || process.env.NG_GO_DEBUG === 'true';

export interface OutputFile {
  path: string;
  text?: string;
  hash?: string;
  kind?: 'js' | 'map' | 'dts' | 'other';
}

export interface DiagnosticMessage {
  category: 'error' | 'warning' | 'message' | 'suggestion';
  code: number;
  message: string;
  file?: string;
  formatted?: string;
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
  /**
   * Preserve ES imports during TypeScript emit. Defaults to true because Vite/Rollup
   * handle tree-shaking and import elision more cheaply than the TypeScript emit
   * resolver for large Angular projects.
   */
  preserveImports?: boolean;
  /**
   * If false, does not print the dev bundle size summary on serve mode.
   * Defaults to true.
   */
  devBundleSummary?: boolean;
  /**
   * Callback invoked when a project compilation completes.
   */
  onCompileComplete?: (event: { reason: string; durationMs: number; outputs: OutputFile[]; error?: any }) => void | Promise<void>;
  /**
   * Callback invoked when a project compilation starts.
   */
  onCompileStart?: (event: { reason: string }) => void | Promise<void>;
  /**
   * Bundle local application entry outputs in memory before serving them.
   * This mirrors Angular's dev-server model where main.js contains the static app graph
   * while bare package imports stay external/prebundled by Vite.
   */
  appBundle?: boolean;
  /**
   * JavaScript output file names to bundle as application entries.
   */
  appBundleEntryFileNames?: string[];
  /**
   * Paths to include for Sass/Less resolution.
   */
  styleIncludePaths?: string[];
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
let isDryRunBuildInProgress = false;
export function setDryRunBuildInProgress(value: boolean) {
  isDryRunBuildInProgress = value;
}
// B#2 FIX: pluginMode is now set per-plugin-instance from options (see angularGoCompile).
// The module-level variable is kept only as a shared reference for angularGoLinker.
let pluginMode: 'server' | 'memory' | 'default' = 'server';
let memoryOutputs = new Map<string, OutputFile>();
let appBundledEntryOutputs = new Map<string, OutputFile>();
let appBundledPublicOutputs = new Map<string, OutputFile>();

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

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
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
  const preserveImports = options.preserveImports !== false;
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
  const templateUpdates = new Map<string, string>();

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

      try {
        const result = await sharedDaemonClient.getHmrUpdate(sharedDaemonContextId, componentId);
        if (result && result.code) {
          templateUpdates.set(componentId, result.code);
        }
      } catch (e: any) {
        devServer.config.logger.warn(`[angular-go] Failed to pre-fetch HMR update for ${componentId}: ${e.message}`);
      }

      // Comment out invalidation to prevent Vite from sending a redundant reload request for the old virtual module.
      // The Angular HMR runtime already requests the new virtual module using the updated timestamp.
      // for (const mod of devServer.moduleGraph.idToModuleMap.values()) {
      //   if (mod.id && mod.id.includes('/@ng/component') && mod.id.includes(`c=${encodeURIComponent(componentId)}`)) {
      //     devServer.moduleGraph.invalidateModule(mod);
      //   }
      // }
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
      if (tsMods) {
        for (const mod of tsMods) {
          server.moduleGraph.invalidateModule(mod);
        }
      }
    }
    const virtualMods = getVirtualComponentModules(tsFiles);
    for (const mod of virtualMods) {
      server.moduleGraph.invalidateModule(mod);
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
    if (isDryRunBuildInProgress) {
      return;
    }
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
    if (options.onCompileStart) {
      try {
        await options.onCompileStart({ reason });
      } catch (err: any) {
        console.error('Failed to run onCompileStart callback', err);
      }
    }
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
      .then(async (result: BuildResult) => {
        lastCompileKey = compileKey;
        lastInputStats = new Map(cachedInputStats);

        if (result.diagnostics && result.diagnostics.length > 0) {
          const errors = result.diagnostics.filter(d => d.category === 'error');
          const warnings = result.diagnostics.filter(d => d.category === 'warning');

          if (warnings.length > 0) {
            for (const w of warnings) {
              if (w.formatted) {
                config.logger.warn(w.formatted);
              } else {
                const msg = w.message || '';
                const filePrefix = w.file ? `${w.file}: ` : '';
                config.logger.warn(`[angular-go] ${filePrefix}warning TS${w.code}: ${msg}`);
              }
            }
          }

          if (errors.length > 0) {
            const errorLines = errors.map(e => {
              if (e.formatted) {
                return e.formatted;
              }
              const msg = e.message || '';
              const filePrefix = e.file ? `${e.file}: ` : '';
              return `${filePrefix}error TS${e.code}: ${msg}`;
            });
            throw new Error(errorLines.join('\n'));
          }
        }

        if (pluginMode === 'memory' || pluginMode === 'server') {
          try {
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
        
        const durationMs = Date.now() - started;
        const callbackOutputs = await hydrateOutputsForCallback(result.outputs || []);
        if (options.appBundle) {
          await updateAppBundledEntryOutputs(callbackOutputs);
          invalidateAppBundles();
        }
        if (options.onCompileComplete) {
          try {
            await options.onCompileComplete({ reason, durationMs, outputs: callbackOutputs });
          } catch (callbackErr: any) {
            console.error('Failed to run onCompileComplete callback', callbackErr);
          }
        }

        if (config.command === 'serve' && options.devBundleSummary !== false) {
          isDryRunBuildInProgress = true;
          try {
            const { build } = await import('vite');
            const configInput = config.build?.rollupOptions?.input;
            let input: any = { main: 'src/main.ts' };
            if (configInput) {
              input = configInput;
            } else {
              const rootDir = config.root || process.cwd();
              const mainCandidates = [
                path.resolve(rootDir, 'main.ts'),
                path.resolve(rootDir, 'src/main.ts'),
                path.resolve(rootDir, 'src/main.js'),
              ];
              for (const cand of mainCandidates) {
                if (fs.existsSync(cand)) {
                  input = { main: cand };
                  break;
                }
              }
            }

            const userPlugins = (config.plugins || []).filter(
              (p: any) => p && p.name && !p.name.startsWith('vite:')
            );

            const buildConfig = {
              configFile: false,
              root: config.root,
              base: config.base,
              mode: config.mode,
              define: config.define,
              resolve: config.resolve,
              publicDir: config.publicDir,
              cacheDir: path.resolve(config.root, '.angular/cache/angular-go-dry-run'),
              optimizeDeps: {
                disabled: true,
                noDiscovery: true,
              },
              plugins: userPlugins,
              logLevel: 'silent' as const,
              build: {
                write: false,
                minify: false,
                sourcemap: false,
                rollupOptions: {
                  input,
                  output: {
                    entryFileNames: 'assets/[name].js',
                    chunkFileNames: 'assets/[name].js',
                    assetFileNames: 'assets/[name].[ext]',
                  }
                }
              }
            };

            await build(buildConfig as any);
            console.log(`Application bundle generation complete. [${((Date.now() - started) / 1000).toFixed(3)} seconds] - ${new Date().toISOString()}\n`);
          } catch (buildErr: any) {
            console.warn(`[angular-go] Could not calculate initial bundle size: ${buildErr.stack || buildErr.message || buildErr}`);
          } finally {
            isDryRunBuildInProgress = false;
          }
        } else {
          log(`compile completed in ${durationMs}ms`);
        }
      })
      .catch((e) => {
        compileError = e;
        if (config && config.command === 'serve') {
          config.logger.error(`\x1b[31m[angular-go] Compilation failed:\n${e.message}\x1b[0m`);
          if (server) {
            server.ws.send({
              type: 'error',
              err: {
                message: e.message,
                stack: e.stack || '',
                plugin: 'vite-plugin-angular-go',
                id: e.id,
                loc: e.loc,
                frame: e.frame,
              }
            });
          }
          if (options.onCompileComplete) {
            const durationMs = Date.now() - started;
            Promise.resolve(options.onCompileComplete({ reason, durationMs, error: e, outputs: [] })).catch(err => {
              console.error('Failed to run onCompileComplete callback for error', err);
            });
          }
          // Do not re-throw in serve mode to prevent load/transform/HMR hook failures
          // which cause Vite's module graph to break and trigger forced re-optimization.
          return;
        }
        throw e;
      })
      .finally(() => {
        compilePromise = null;
      });

    return compilePromise;
  }

  async function hydrateOutputsForCallback(outputs: OutputFile[]): Promise<OutputFile[]> {
    if (pluginMode !== 'server' || !sharedDaemonClient || !sharedDaemonContextId) {
      return outputs;
    }

    const hydrated: OutputFile[] = [];
    for (const output of outputs) {
      if (output.kind !== 'js' || output.text !== undefined) {
        hydrated.push(output);
        continue;
      }
      try {
        hydrated.push(await sharedDaemonClient.getOutput(sharedDaemonContextId, output.path) || output);
      } catch {
        hydrated.push(output);
      }
    }
    return hydrated;
  }

  async function updateAppBundledEntryOutputs(outputs: OutputFile[]): Promise<void> {
    const entryFileNames = options.appBundleEntryFileNames || ['main.js'];
    const entries = entryFileNames.map(fileName => ({
      name: path.basename(fileName, path.extname(fileName)),
      fileName,
    }));
    const bundled = await bundleCompilerOutputsWithRolldown(outputs, entries);

    appBundledEntryOutputs.clear();
    appBundledPublicOutputs.clear();

    // --- Entry chunks (e.g. main.js) ---
    const entryChunkNames = new Set<string>();
    for (const entry of entries) {
      const sourceOutputPath = findOutputPathByBaseName(outputs, entry.fileName);
      const bundledItem = bundled[`${entry.name}.js`];
      if (!sourceOutputPath || !bundledItem || bundledItem.type !== 'chunk') {
        continue;
      }
      entryChunkNames.add(`${entry.name}.js`);
      const absPath = path.resolve(projectRoot, sourceOutputPath);
      appBundledEntryOutputs.set(absPath, {
        path: absPath,
        text: bundledItem.code,
        kind: 'js',
      });
      // Register under the .js URL (e.g. /main.js)
      appBundledPublicOutputs.set(`/${entry.fileName}`, {
        path: `/${entry.fileName}`,
        text: bundledItem.code,
        kind: 'js',
      });
    }

    // --- Lazy chunks (dynamic-import splits from Rolldown) ---
    // These are NOT entry chunks but the browser requests them when the bundled
    // main.js executes a dynamic import().  Without registering them here the
    // middleware passes the request to Vite which can't resolve them → 404.
    for (const [fileName, item] of Object.entries(bundled)) {
      if ((item as any).type !== 'chunk') continue;
      if (entryChunkNames.has(fileName)) continue; // already handled above
      appBundledPublicOutputs.set(`/${fileName}`, {
        path: `/${fileName}`,
        text: (item as any).code,
        kind: 'js',
      });
    }
  }

  function invalidateAppBundles(): void {
    if (!server) {
      return;
    }
    for (const publicPath of appBundledPublicOutputs.keys()) {
      // /@ng/bundle/* is the ID Vite's module graph tracks after transformRequest
      const bundleId = `/@ng/bundle${publicPath}`;
      const mod = server.moduleGraph.getModuleById(bundleId);
      if (mod) {
        server.moduleGraph.invalidateModule(mod);
      }
    }
  }

  function getAppBundleModules(): any[] {
    if (!server) {
      return [];
    }
    const modules: any[] = [];
    for (const publicPath of appBundledPublicOutputs.keys()) {
      const bundleId = `/@ng/bundle${publicPath}`;
      const mod = server.moduleGraph.getModuleById(bundleId);
      if (mod) {
        modules.push(mod);
      }
    }
    return modules;
  }

  function getVirtualComponentModules(affectedTsFiles: Iterable<string>): any[] {
    const modules: any[] = [];
    if (!server) {
      return [];
    }
    const relFiles = new Set<string>();
    for (const tsFile of affectedTsFiles) {
      relFiles.add(path.relative(projectRoot, cleanId(tsFile)).replace(/\\/g, '/'));
    }
    for (const mod of server.moduleGraph.idToModuleMap.values()) {
      if (mod.id && mod.id.includes('/@ng/component')) {
        const qIdx = mod.id.indexOf('?');
        if (qIdx >= 0) {
          const params = new URLSearchParams(mod.id.slice(qIdx + 1));
          const c = params.get('c');
          if (c) {
            const atIdx = c.lastIndexOf('@');
            const componentFile = atIdx >= 0 ? c.slice(0, atIdx) : c;
            if (relFiles.has(componentFile)) {
              modules.push(mod);
            }
          }
        }
      }
    }
    return modules;
  }

  function findOutputPathByBaseName(outputs: OutputFile[], baseName: string): string | undefined {
    return outputs.find(output => output.kind === 'js' && path.basename(output.path) === baseName)?.path;
  }

  function projectCompileKey(): string {
    return computeProjectState().key;
  }

  function outputPathForSource(id: string): string | null {
    const filePath = cleanId(id);
    if (!filePath.endsWith('.ts') || filePath.endsWith('.d.ts') || filePath.includes(`${path.sep}node_modules${path.sep}`)) {
      return null;
    }

    const expectedSuffix = filePath.replace(/\.ts$/, '.js');
    const expectedName = path.basename(expectedSuffix);

    // If we are in memory mode, we can search the exact key
    if (memoryOutputs.size > 0) {
      let bestMatchKey: string | null = null;
      let maxMatchCount = 0;
      const expectedNameLower = expectedName.toLowerCase();

      for (const key of memoryOutputs.keys()) {
        if (key.toLowerCase().endsWith(expectedNameLower)) {
          // Find the longest matching directory suffix
          const keyParts = key.toLowerCase().split(path.sep);
          const expectedParts = expectedSuffix.toLowerCase().split(path.sep);
          
          let matchCount = 0;
          for (let i = 1; i <= Math.min(keyParts.length, expectedParts.length); i++) {
            if (keyParts[keyParts.length - i] === expectedParts[expectedParts.length - i]) {
              matchCount++;
            } else {
              break;
            }
          }
          if (matchCount > maxMatchCount) {
            maxMatchCount = matchCount;
            bestMatchKey = key; // Return the ORIGINAL case-preserving key
          }
        }
      }
      
      if (bestMatchKey && maxMatchCount > 0) {
        return bestMatchKey;
      }
    }

    // Fallback logic if memoryOutputs is empty (e.g. before first compile or in default mode)
    let relativePath = path.relative(projectRoot, filePath);
    
    // If it's an Nx/Angular workspace, use the workspace root for the relative path
    const workspaceRoot = findUp(projectRoot, (dir) => fs.existsSync(path.join(dir, 'nx.json')) || fs.existsSync(path.join(dir, 'angular.json')));
    if (workspaceRoot) {
      relativePath = path.relative(workspaceRoot, filePath);
    } else if (relativePath.startsWith('..') || path.isAbsolute(relativePath)) {
      return null;
    }

    return path.join(outDir, relativePath).replace(/\.ts$/, '.js');
  }

  // Find a file by walking up directories
  function findUp(dir: string, predicate: (dir: string) => boolean): string | null {
    let current = path.resolve(dir);
    while (current) {
      if (predicate(current)) return current;
      const parent = path.dirname(current);
      if (parent === current) break;
      current = parent;
    }
    return null;
  }

  return {
    name: 'vite-plugin-angular-go-compile',
    enforce: 'pre',

    configResolved(resolvedConfig: ResolvedConfig) {
      if (isDryRunBuildInProgress) {
        return;
      }
      config = resolvedConfig;
      projectRoot = options.projectRoot ? path.resolve(options.projectRoot) : process.cwd();
      outDir = resolveFromProjectRoot(options.outDir || 'out-tsc/app');
      compilerPath = resolveGoNgcPath(projectRoot, compilerPath, import.meta.url);

      const angularVersion = packageVersion(projectRoot, '@angular/core');
      if (angularVersion !== 'unknown') {
        const majorVersion = parseInt(angularVersion.split('.')[0], 10);
        if (majorVersion !== 21) {
          resolvedConfig.logger.warn(
            `\x1b[33m[angular-go] Warning: @angular/core version ${angularVersion} is installed. ` +
            `angular-go is designed and tested for Angular version 21.x. Incompatibilities may occur.\x1b[0m`
          );
        }
      }
      const cliVersion = packageVersion(projectRoot, '@angular/compiler-cli');
      if (cliVersion !== 'unknown') {
        const majorVersion = parseInt(cliVersion.split('.')[0], 10);
        if (majorVersion !== 21) {
          resolvedConfig.logger.warn(
            `\x1b[33m[angular-go] Warning: @angular/compiler-cli version ${cliVersion} is installed. ` +
            `angular-go is designed and tested for Angular version 21.x. Incompatibilities may occur.\x1b[0m`
          );
        }
      }

      compileArgs = normalizeCompileArgs(options.args || ['-p', options.project || 'tsconfig.app.json']);
      // Keep server mode for production build to avoid spawning subprocesses
      // for linking and compilation.
      if (config.command === 'build' && pluginMode === 'server') {
        // pluginMode = 'default';
      }
      
      enableHmr = config.command === 'serve' && options.hmr === true;
      isDebug && console.log('[angular-go:compile] configResolved enableHmr:', enableHmr, 'options.hmr:', options.hmr, 'config.command:', config.command);
      const compilationMode = options.compilationMode || 'local';

      if (!compileArgs.find(arg => arg.startsWith('--compilationMode'))) {
        compileArgs.push(`--compilationMode=${compilationMode}`);
      }
      if (!options.styleIncludePaths) {
        options.styleIncludePaths = [];
      }
      // Add node_modules by default so SASS can resolve external package imports
      const nodeModulesPath = path.join(process.cwd(), 'node_modules');
      const rootNodeModulesPath = path.join(process.cwd(), '../../node_modules'); // For monorepos
      if (!options.styleIncludePaths.includes(nodeModulesPath)) {
        options.styleIncludePaths.push(nodeModulesPath);
      }
      if (!options.styleIncludePaths.includes(rootNodeModulesPath)) {
        options.styleIncludePaths.push(rootNodeModulesPath);
      }
      
      if (options.styleIncludePaths && options.styleIncludePaths.length > 0) {
        compileArgs.push(`--style-include-paths=${options.styleIncludePaths.join(',')}`);
      }
      if (preserveImports && !compileArgs.some(arg => arg === '--preserve-imports')) {
        compileArgs.push('--preserve-imports');
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

      // Serve app entry bundles from the in-memory Rolldown output. The bundle
      // keeps dynamic imports as lazy boundaries while presenting one eager
      // main.js to the browser and to Vite's import analysis pipeline.
      devServer.middlewares.use(async (req: any, res: any, next: any) => {
        if (!options.appBundle) {
          return next();
        }

        const url: string = req.url || '';
        const qIdx = url.indexOf('?');
        const pathname = qIdx >= 0 ? url.slice(0, qIdx) : url;
        const publicOutput = appBundledPublicOutputs.get(pathname);
        if (!publicOutput?.text) {
          return next();
        }

        try {
          await ensureFreshDevCompile(`freshness check for app bundle ${pathname}`);
          // Use transformRequest with a /@ng/bundle/ virtual path so that
          // Vite's import-analysis plugin runs and rewrites bare specifiers
          // (@angular/localize/init, @angular/core, …) to their pre-bundled
          // /.vite/deps/ paths.  A \0-prefixed virtual ID would skip import
          // analysis; the /@ng/bundle/ prefix keeps it in Vite's normal
          // transform pipeline without conflicting with real files.
          const result = await devServer.transformRequest(`/@ng/bundle${pathname}`);
          if (!result?.code) {
            return next();
          }
          res.writeHead(200, {
            'Content-Type': 'application/javascript',
            'Cache-Control': 'no-cache, no-store, must-revalidate',
          });
          res.end(result.code);
        } catch (e: any) {
          devServer.config.logger.warn(
            `[angular-go] Failed to serve app bundle for ${pathname}: ${e.message}`
          );
          return next();
        }
      });

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
          const hmrCode = templateUpdates.get(c);
          if (hmrCode) {
            const result = await devServer.transformRequest(url);
            if (result && result.code) {
              res.writeHead(200, {
                'Content-Type': 'application/javascript',
                'Cache-Control': 'no-cache, no-store, must-revalidate',
              });
              res.end(result.code);
              return;
            }
          }
          res.writeHead(200, { 'Content-Type': 'application/javascript' });
          res.end('');
        } catch (e: any) {
          devServer.config.logger.warn(
            `[angular-go] Failed to serve transformed HMR update for ${url}: ${e.message}`
          );
          res.writeHead(200, { 'Content-Type': 'application/javascript' });
          res.end('');
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

    transformIndexHtml(html: string) {
      if (!options.appBundle) {
        return html;
      }

      let nextHtml = html;
      const tagsToInject: string[] = [];

      for (const fileName of options.appBundleEntryFileNames || ['main.js']) {
        // Remove any stale script tags that reference this entry (both .ts and .js
        // variants), so we never end up with duplicates after injection.
        const tsName = fileName.replace(/\.js$/, '.ts');
        const removePattern = (name: string) =>
          new RegExp(
            `[ \t]*<script[^>]+type=["']module["'][^>]+src=["']/${escapeRegExp(name)}["'][^>]*>\\s*</script>[ \t]*(\r?\n)?`,
            'g'
          );
        nextHtml = nextHtml.replace(removePattern(tsName), '');
        nextHtml = nextHtml.replace(removePattern(fileName), '');

        // Queue the canonical .js tag for injection.
        tagsToInject.push(`  <script type="module" src="/${fileName}"></script>`);
      }

      // Inject all entry script tags just before </body>.
      if (tagsToInject.length > 0) {
        const block = tagsToInject.join('\n');
        nextHtml = nextHtml.includes('</body>')
          ? nextHtml.replace('</body>', `${block}\n</body>`)
          : nextHtml + '\n' + block;
      }

      return nextHtml;
    },

    async buildStart() {
      if (isDryRunBuildInProgress) {
        return;
      }
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
          compilationMode: options.compilationMode || 'local',
          // C3 FIX: Only enable HMR when explicitly requested in serve mode.
          hmr: enableHmr,
          preserveImports,
          styleIncludePaths: options.styleIncludePaths,
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

    resolveId(id: string, importer?: string) {
      if (options.appBundle) {
        // /@ng/bundle/* is the virtual URL used by transformRequest for the
        // pre-bundled app. Keep this in Vite's normal transform pipeline so
        // import-analysis can rewrite bare package specifiers.
        if (id.startsWith('/@ng/bundle/')) {
          const publicPath = id.slice('/@ng/bundle'.length); // e.g. /main.js
          if (appBundledPublicOutputs.has(publicPath)) {
            return id;
          }
        }

        // Lazy route imports inside the bundled main chunk are emitted as
        // relative imports (e.g. ./module-dashboard.module.js). Resolve them
        // against the public bundle path so Vite can load the generated chunk
        // from memory instead of trying to find a sibling of the virtual id.
        if (id.startsWith('.') && importer) {
          const cleanImporter = cleanId(importer);
          let importerPublicPath: string | undefined;
          let useBundleUrl = false;

          if (cleanImporter.startsWith('/@ng/bundle/')) {
            importerPublicPath = cleanImporter.slice('/@ng/bundle'.length);
            useBundleUrl = true;
          } else if (cleanImporter.startsWith('\0angular-go-app-bundle:')) {
            importerPublicPath = cleanImporter.slice('\0angular-go-app-bundle:'.length);
          } else if (cleanImporter.startsWith('angular-go-app-bundle:')) {
            importerPublicPath = cleanImporter.slice('angular-go-app-bundle:'.length);
          }

          if (importerPublicPath) {
            const publicPath = path.posix.normalize(
              path.posix.join(path.posix.dirname(importerPublicPath), id)
            );
            const normalizedPublicPath = publicPath.startsWith('/') ? publicPath : `/${publicPath}`;
            if (appBundledPublicOutputs.has(normalizedPublicPath)) {
              return useBundleUrl
                ? `/@ng/bundle${normalizedPublicPath}`
                : `\0angular-go-app-bundle:${normalizedPublicPath}`;
            }
          }
        }

        const clean = cleanId(id);
        if (!clean.startsWith('/@ng/bundle/') && appBundledPublicOutputs.has(clean)) {
          return `\0angular-go-app-bundle:${clean}`;
        }
      }

      isDebug && console.log('[angular-go:compile] resolveId:', id, 'enableHmr:', enableHmr);
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
      // /@ng/bundle/* — non-\0 virtual modules for pre-bundled app chunks.
      // These go through Vite's full transform pipeline (including import
      // analysis) so bare specifiers are rewritten to pre-bundled paths.
      if (options.appBundle && id.startsWith('/@ng/bundle/')) {
        const publicPath = id.slice('/@ng/bundle'.length); // e.g. /main.js
        const output = appBundledPublicOutputs.get(publicPath);
        if (output?.text !== undefined) {
          let code = output.text;
          if (enableHmr) {
            code = code.replace(
              /import\s*\(\s*([a-zA-Z0-9_$]*\.)?ɵɵgetReplaceMetadataURL/g,
              'import(/* @vite-ignore */ $1ɵɵgetReplaceMetadataURL'
            );
          }
          return { code, map: null };
        }
        return null;
      }

      if (id.startsWith('\0angular-go-app-bundle:')) {
        const publicPath = id.slice('\0angular-go-app-bundle:'.length);
        const output = appBundledPublicOutputs.get(publicPath);
        if (output?.text !== undefined) {
          let code = output.text;
          if (enableHmr) {
            code = code.replace(/import\s*\(\s*([a-zA-Z0-9_$]*\.)?ɵɵgetReplaceMetadataURL/g, 'import(/* @vite-ignore */ $1ɵɵgetReplaceMetadataURL');
          }
          return { code, map: null };
        }
        return null;
      }

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
            const hmrCode = templateUpdates.get(c);
            if (hmrCode) {
              let code = `import { ɵɵreplaceMetadata as ɵɵreplaceMetadata_hmr } from '@angular/core';\n` + hmrCode;
              
              // Rewrite relative imports to absolute paths relative to Vite's config.root
              // because virtual module ID doesn't have a correct directory for Vite resolution.
              const fileDir = path.dirname(filePath); // e.g. src/app
              const absFileDir = path.resolve(projectRoot, fileDir);
              let viteBasePath = path.relative(config.root, absFileDir);
              if (viteBasePath) viteBasePath = '/' + viteBasePath.replace(/\\/g, '/');
              else viteBasePath = '';

              code = code.replace(/from\s+['"](\.[^'"]+)['"]/g, (match, relPath) => {
                const resolved = path.posix.join(viteBasePath || '/', relPath);
                return `from '${resolved}'`;
              });
              code = code.replace(/import\s*\(\s*['"](\.[^'"]+)['"]\s*\)/g, (match, relPath) => {
                const resolved = path.posix.join(viteBasePath || '/', relPath);
                return `import('${resolved}')`;
              });

              return { code, map: null };
            }
          }
        }
        // No HMR code available: return a no-op module so the dynamic import
        // succeeds (browser doesn't get ERR_ABORTED) and Angular gracefully
        // keeps the existing component state.
        return { code: '', map: null };
      }

      const jsPath = outputPathForSource(id);
      isDebug && console.log('[angular-go:compile] load id:', id, 'jsPath:', jsPath, 'inMemory:', jsPath ? memoryOutputs.has(jsPath) : false);
      if (jsPath == null) {
        return null;
      }

      await ensureFreshDevCompile(`freshness check for ${path.relative(projectRoot, cleanId(id))}`);

      if (!isDryRunBuildInProgress) {
        if (pluginMode !== 'server' && pluginMode !== 'memory') {
          if (!fs.existsSync(jsPath)) {
            try {
              await compileProject(`missing output for ${path.relative(projectRoot, cleanId(id))}`, false);
            } catch (e) {
              // ignore here, we will throw compileError below
            }
          }
        } else {
          if (!memoryOutputs.has(jsPath)) {
            try {
              await compileProject(`missing memory output for ${path.relative(projectRoot, cleanId(id))}`, false);
            } catch (e) {
              // ignore here
            }
          }
        }
      }

      if (compileError) {
        this.error(compileError);
      }

      if (pluginMode !== 'server' && pluginMode !== 'memory') {
        if (!fs.existsSync(jsPath)) {
          // If go-ngc didn't compile it (e.g. polyfills.ts), let Vite handle it natively
          return null;
        }
      }
      let sourcePath = cleanId(id);
      if (!fs.existsSync(sourcePath)) {
        const resolved = path.join(projectRoot, sourcePath);
        if (fs.existsSync(resolved)) {
          sourcePath = resolved;
        }
      }
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
      const appBundledOutput = appBundledEntryOutputs.get(jsPath);
      if (appBundledOutput && typeof appBundledOutput.text === 'string') {
        let code = appBundledOutput.text;
        if (enableHmr) {
          code = code.replace(/import\s*\(\s*([a-zA-Z0-9_$]*\.)?ɵɵgetReplaceMetadataURL/g, 'import(/* @vite-ignore */ $1ɵɵgetReplaceMetadataURL');
        }
        isDebug && console.log('[angular-go:compile] load returning code for:', id, 'snippet:', code.slice(0, 300));
        return { code, map: null };
      }

      if (pluginMode === 'server' || pluginMode === 'memory') {
        const memOut = memoryOutputs.get(jsPath);
        if (memOut && typeof memOut.text === 'string') {
          code = memOut.text;
        } else if (pluginMode === 'server' && sharedDaemonClient) {
          const output = await sharedDaemonClient.getOutput(sharedDaemonContextId, jsPath);
          if (output && typeof output.text === 'string') {
            memoryOutputs.set(jsPath, output);
            code = output.text;
          }
        }
        const memMap = memoryOutputs.get(mapPath);
        if (memMap && typeof memMap.text === 'string') {
          map = JSON.parse(memMap.text);
        } else if (pluginMode === 'server' && sharedDaemonClient) {
          const output = await sharedDaemonClient.getOutput(sharedDaemonContextId, mapPath);
          if (output && typeof output.text === 'string') {
            memoryOutputs.set(mapPath, output);
            map = JSON.parse(output.text);
          }
        }
      }

      if (code == null || code === '') {
        if (!memoryOutputs.has(jsPath)) {
          console.log('[DEBUG] Failed to find in memoryOutputs:', jsPath);
          console.log('[DEBUG] Available keys:', Array.from(memoryOutputs.keys()).slice(0, 5));
        }
        if (id.includes('non-standalone.component')) {
          console.log('[DEBUG-NON-STANDALONE] code is null! jsPath:', jsPath);
        }
        if (compileError && config && config.command === 'serve') {
          return { code: 'export default null;', map: null };
        }
        if (fs.existsSync(mapPath)) {
          try {
            map = JSON.parse(fs.readFileSync(mapPath, 'utf8'));
          } catch {
            // ignore
          }
        }
        if (fs.existsSync(jsPath)) {
          code = fs.readFileSync(jsPath, 'utf8');
        } else {
          // If go-ngc didn't compile it (e.g. polyfills.ts) and it's not on disk, let Vite handle it
          return null;
        }
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

      if (code) {
        isDebug && console.log('[angular-go:compile] load returning code for:', id, 'snippet:', code.slice(0, 300));
        if (id.includes('app.ts')) {
          isDebug && console.log('[angular-go:compile] FULL app.ts code:', code);
        }
        if (id.includes('non-standalone.component.ts')) {
          console.log('[DEBUG-CODE] returning code for non-standalone.component.ts:', code.includes('ɵcmp') ? 'HAS_CMP' : 'NO_CMP', 'length:', code.length);
        }
      }
      return { code, map };
    },

    async transform(code: string, id: string) {
      if (!enableHmr) {
        return null;
      }
      
      // We only transform modules inside our src folder to avoid transforming node_modules unnecessarily
      if (id.includes('node_modules')) {
        return null;
      }

      let transformed = code;
      let hasChange = false;

      if (transformed.includes('ɵɵgetReplaceMetadataURL')) {
        transformed = transformed.replace(/import\s*\(\s*([a-zA-Z0-9_$]*\.)?ɵɵgetReplaceMetadataURL/g, 'import(/* @vite-ignore */ $1ɵɵgetReplaceMetadataURL');
        hasChange = true;
      }



      if (hasChange) {
        isDebug && console.log('[angular-go:compile] transform applied HMR cache setup to:', id);
        return { code: transformed, map: null };
      }

      return null;
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

        // Send HMR update event to browser and invalidate module graph cache for virtual modules
        if (server) {
          const affectedTsFiles = htmlToTs.get(filePath)!;
          invalidateTsModulesForFiles(affectedTsFiles);
          if (enableHmr) {
            await sendHmrUpdatesForFiles(server, affectedTsFiles);
          }

          if (enableHmr) {
            return getVirtualComponentModules(affectedTsFiles);
          }
          if (options.appBundle) {
            return getAppBundleModules();
          } else {
            const modules: any[] = [];
            for (const tsFile of affectedTsFiles) {
              const mods = server.moduleGraph.getModulesByFile(tsFile);
              if (mods) {
                modules.push(...mods);
              }
            }
            return modules;
          }
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

        // Send HMR update event to browser and invalidate module graph cache for virtual modules
        if (server) {
          const affectedTsFiles = cssToTs.get(filePath)!;
          invalidateTsModulesForFiles(affectedTsFiles);
          if (enableHmr) {
            await sendHmrUpdatesForFiles(server, affectedTsFiles);
          }

          if (enableHmr) {
            return getVirtualComponentModules(affectedTsFiles);
          }
          if (options.appBundle) {
            return getAppBundleModules();
          } else {
            const modules: any[] = [];
            for (const tsFile of affectedTsFiles) {
              const mods = server.moduleGraph.getModulesByFile(tsFile);
              if (mods) {
                modules.push(...mods);
              }
            }
            return modules;
          }
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
        try {
          const sourceCode = fs.readFileSync(filePath, 'utf8');
          parseStyleMappingsForFile(filePath, sourceCode);
          if (enableHmr) {
            await sendHmrUpdatesForFiles(server, [filePath]);
          }

          if (enableHmr) {
            return getVirtualComponentModules([filePath]);
          }
          if (options.appBundle) {
            return getAppBundleModules();
          } else {
            const mods = server.moduleGraph.getModulesByFile(filePath);
            if (mods) {
              return Array.from(mods);
            }
          }
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
      if (config.command !== 'build' && !isDryRunBuildInProgress) {
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
        if (bytes === 1) return '1 byte';
        if (bytes < 1024) return `${bytes} bytes`;
        const k = 1024;
        const sizes = ['bytes', 'kB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
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
      for (const row of initialRows) {
        initialRawTotal += row.rawSize;
      }

      const displayInitialRows = initialRows.map(row => ({
        ...row,
        file: row.file.replace(/^assets\//, ''),
        sizeStr: formatBytes(row.rawSize)
      }));

      const displayLazyRows = lazyRows.map(row => ({
        ...row,
        file: row.file.replace(/^assets\//, ''),
        sizeStr: formatBytes(row.rawSize)
      }));

      const maxFileLen = Math.max(
        'Initial chunk files'.length,
        'Lazy chunk files'.length,
        ...displayInitialRows.map(r => r.file.length),
        ...displayLazyRows.map(r => r.file.length)
      );
      const maxNameLen = Math.max(
        13, // 'Initial total'.length
        ...displayInitialRows.map(r => r.name.length),
        ...displayLazyRows.map(r => r.name.length)
      );
      const maxRawSizeLen = Math.max(
        8, // 'Raw size'.length
        ...displayInitialRows.map(r => r.sizeStr.length),
        ...displayLazyRows.map(r => r.sizeStr.length),
        formatBytes(initialRawTotal).length
      );

      const col2Width = maxNameLen + 1;

      const pad = (str: string, len: number) => str.padEnd(len);
      const padStart = (str: string, len: number) => str.padStart(len);

      const outputLines: string[] = [];
      outputLines.push(
        `  ` +
        `\x1b[1m${pad('Initial chunk files', maxFileLen)}\x1b[22m | ` +
        `\x1b[1m${pad('Names', col2Width)}\x1b[22m | ` +
        `\x1b[1m${pad('Raw size', maxRawSizeLen)}\x1b[22m`
      );

      for (const row of displayInitialRows) {
        const fileStr = row.type === 'js' ? `\x1b[36m${row.file}\x1b[39m` : `\x1b[35m${row.file}\x1b[39m`;
        const nameStr = `\x1b[90m${row.name}\x1b[39m`;
        
        let sizeColor = '';
        if (row.rawSize < 100 * 1024) {
          sizeColor = '\x1b[32m';
        } else if (row.rawSize < 500 * 1024) {
          sizeColor = '\x1b[33m';
        } else {
          sizeColor = '\x1b[31m\x1b[1m';
        }
        const sizeStr = `${sizeColor}${row.sizeStr}\x1b[39m\x1b[22m`;

        const filePadded = pad(fileStr, maxFileLen + (fileStr.length - row.file.length));
        const namePadded = pad(nameStr, col2Width + (nameStr.length - row.name.length));
        const sizePadded = padStart(sizeStr, maxRawSizeLen + (sizeStr.length - row.sizeStr.length));
        
        outputLines.push(`  ${filePadded} | ${namePadded} | ${sizePadded} | `);
      }

      outputLines.push('');

      const totalLabel = `\x1b[1mInitial total\x1b[22m`;
      const totalRawVal = formatBytes(initialRawTotal);
      const totalRawStr = `\x1b[1m${totalRawVal}\x1b[22m`;

      const spaces1 = ' '.repeat(maxFileLen);
      const labelPadded = pad(totalLabel, col2Width + (totalLabel.length - 'Initial total'.length));
      const sizePadded = padStart(totalRawStr, maxRawSizeLen + (totalRawStr.length - totalRawVal.length));

      outputLines.push(`  ${spaces1} | ${labelPadded} | ${sizePadded}`);

      if (displayLazyRows.length > 0) {
        outputLines.push('');
        outputLines.push(
          `  ` +
          `\x1b[1m${pad('Lazy chunk files', maxFileLen)}\x1b[22m | ` +
          `\x1b[1m${pad('Names', col2Width)}\x1b[22m | ` +
          `\x1b[1m${pad('Raw size', maxRawSizeLen)}\x1b[22m`
        );

        for (const row of displayLazyRows) {
          const fileStr = `\x1b[90m${row.file}\x1b[39m`;
          const nameStr = `\x1b[90m${row.name}\x1b[39m`;
          
          let sizeColor = '';
          if (row.rawSize < 100 * 1024) {
            sizeColor = '\x1b[32m';
          } else if (row.rawSize < 500 * 1024) {
            sizeColor = '\x1b[33m';
          } else {
            sizeColor = '\x1b[31m\x1b[1m';
          }
          const sizeStr = `${sizeColor}${row.sizeStr}\x1b[39m\x1b[22m`;

          const filePadded = pad(fileStr, maxFileLen + (fileStr.length - row.file.length));
          const namePadded = pad(nameStr, col2Width + (nameStr.length - row.name.length));
          const sizePadded = padStart(sizeStr, maxRawSizeLen + (sizeStr.length - row.sizeStr.length));
          
          outputLines.push(`  ${filePadded} | ${namePadded} | ${sizePadded} | `);
        }
      }

      console.log('\n' + outputLines.join('\n') + '\n');
    },

    async closeBundle() {
      if (isDryRunBuildInProgress) {
        return;
      }
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
  const forceOptimizeDeps = options.forceOptimizeDeps === true;
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
      try {
        return await linkFile(filePath);
      } catch (err) {
        // Best effort: if linkFile fails (e.g. because the file was deleted/renamed by Vite's pre-bundling optimizer),
        // fallback to linking the in-memory code.
      }
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
      if (isDryRunBuildInProgress) {
        return;
      }
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
