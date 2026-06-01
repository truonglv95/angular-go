import type { ResolvedConfig, ViteDevServer } from 'vite';
import { spawn } from 'node:child_process';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

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

class GoNgcError extends Error {
  readonly stdout: string;
  readonly stderr: string;
  readonly args: string[];
  readonly code: number | null;

  constructor(args: string[], code: number | null, stdout: string, stderr: string) {
    super(formatGoNgcError(args, code, stdout, stderr));
    this.name = 'GoNgcError';
    this.args = args;
    this.code = code;
    this.stdout = stdout;
    this.stderr = stderr;
  }
}

const angularPartialDeclarationMarker = 'ɵɵngDeclare';
const sourceFilePattern = /\.(ts|html|css|scss|sass|less)$/;
const projectInputPattern = /\.(ts|html|css|scss|sass|less|json)$/;

function cleanId(id: string): string {
  return id.split('?', 1)[0];
}

function resolveCompilerPath(projectRoot: string, compilerPath: string): string {
  if (compilerPath.includes('/') || compilerPath.includes(path.sep)) {
    return path.isAbsolute(compilerPath) ? compilerPath : path.resolve(projectRoot, compilerPath);
  }
  return compilerPath;
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

function collectProjectInputs(projectRoot: string, outDir: string): string[] {
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
  files.sort();
  return files;
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
  const recompileOnChange = options.recompileOnChange !== false;

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
  async function compileProject(reason: string, force = false): Promise<void> {
    if (compilePromise) {
      return compilePromise;
    }

    const compileKey = projectCompileKey();
    if (!force && compileKey === lastCompileKey && fs.existsSync(outDir)) {
      log(`skipped compile for unchanged inputs (${reason})`);
      return;
    }

    const started = Date.now();
    compileError = null;
    compilePromise = runGoNgc(projectRoot, compilerPath, compileArgs)
      .then(() => {
        lastCompileKey = compileKey;
        log(`compiled ${path.relative(config.root, projectRoot) || '.'} (${reason}) in ${Date.now() - started}ms`);
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
    const hash = crypto.createHash('sha1');
    hash.update(compilerSalt(compilerPath));
    hash.update(JSON.stringify(compileArgs));
    hash.update(packageVersion(projectRoot, '@angular/core'));
    hash.update(packageVersion(projectRoot, '@angular/compiler-cli'));
    for (const filePath of collectProjectInputs(projectRoot, outDir)) {
      hash.update(path.relative(projectRoot, filePath));
      hash.update(optionalStatKey(filePath));
    }
    return hash.digest('hex');
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
      compilerPath = resolveCompilerPath(projectRoot, compilerPath);
      compileArgs = normalizeCompileArgs(options.args || ['-p', options.project || 'tsconfig.app.json']);
    },

    configureServer(devServer: ViteDevServer) {
      server = devServer;
    },

    async buildStart() {
      for (const filePath of collectProjectInputs(projectRoot, outDir)) {
        this.addWatchFile(filePath);
      }
      try {
        await compileProject('startup');
      } catch (e: any) {
        if (config.command === 'build') {
          this.error(e.message);
        } else {
          config.logger.error(e.message);
        }
      }
    },

    async load(id: string) {
      const jsPath = outputPathForSource(id);
      if (jsPath == null) {
        return null;
      }

      if (!fs.existsSync(jsPath)) {
        try {
          await compileProject(`missing output for ${path.relative(projectRoot, cleanId(id))}`, true);
        } catch (e) {
          // ignore here, we will throw compileError below
        }
      }

      if (compileError) {
        this.error(compileError.message);
      }

      if (!fs.existsSync(jsPath)) {
        this.error(`go-ngc did not emit ${jsPath} for ${cleanId(id)}`);
      }

      return { code: fs.readFileSync(jsPath, 'utf8'), map: null };
    },

    async handleHotUpdate(ctx: any) {
      if (!recompileOnChange) {
        return;
      }

      const filePath = cleanId(ctx.file);
      if (!sourceFilePattern.test(filePath) || filePath.includes(`${path.sep}node_modules${path.sep}`)) {
        return;
      }

      const relativePath = path.relative(projectRoot, filePath);
      if (relativePath.startsWith('..') || path.isAbsolute(relativePath)) {
        return;
      }

      try {
        await compileProject(`change in ${relativePath}`);
      } catch (e: any) {
        config.logger.error(e.message);
        server?.ws.send({
          type: 'error',
          err: {
            message: e.message,
            stack: '',
            plugin: 'vite-plugin-angular-go',
          }
        });
        return [];
      }
      server?.moduleGraph.invalidateAll();
      server?.ws.send({ type: 'full-reload' });
      return [];
    },
  };
}

export function angularGoLinker(options: AngularGoLinkerOptions = {}): any {
  let projectRoot = '';
  let compilerPath = options.compilerPath || 'go-ngc';
  const ascii = options.ascii !== false;
  const forceOptimizeDeps = options.forceOptimizeDeps !== false;
  const linkCache = new Map<string, LinkCacheEntry>();

  function configurePaths() {
    projectRoot = options.projectRoot ? path.resolve(options.projectRoot) : process.cwd();
    compilerPath = resolveCompilerPath(projectRoot, compilerPath);
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

    const linked = await runGoNgc(projectRoot, compilerPath, ['--link', filePath]);
    linkCache.set(cacheId, { key, code: linked });
    return linked;
  }

  async function linkCode(code: string, id: string): Promise<string> {
    const key = codeCacheKey(code);
    const cacheId = `code:${id}:${key}`;
    const cached = linkCache.get(cacheId);
    if (cached) {
      return cached.code;
    }

    const tempPath = path.join(
      os.tmpdir(),
      `angular-go-link-${process.pid}-${Date.now()}-${Math.random().toString(36).slice(2)}.mjs`,
    );

    fs.writeFileSync(tempPath, code);
    try {
      const linked = await runGoNgc(projectRoot, compilerPath, ['--link', tempPath]);
      linkCache.set(cacheId, { key, code: linked });
      return linked;
    } finally {
      fs.rmSync(tempPath, { force: true });
    }
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
