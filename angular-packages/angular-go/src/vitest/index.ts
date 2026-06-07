import { UserConfig as ViteUserConfig } from 'vite';
import angularGo from '../vite-plugin/index.js';
import path from 'node:path';
import fs from 'node:fs';


export interface AngularGoVitestConfigOptions {
  project?: string;
  compilerPath?: string;
  compilationMode?: 'global' | 'local';
  testEnvironment?: 'jsdom' | 'happy-dom' | 'browser';
  setupFiles?: string[];
  root?: string;
  outDir?: string;
}

function findUp(startDir: string, predicate: (dir: string) => boolean): string {
  let dir = path.resolve(startDir);
  while (true) {
    if (predicate(dir)) {
      return dir;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      return path.resolve(startDir);
    }
    dir = parent;
  }
}

function findWorkspaceRoot(): string {
  return findUp(process.cwd(), (dir) =>
    fs.existsSync(path.join(dir, 'package.json')) &&
    fs.existsSync(path.join(dir, 'tsconfig.spec.json'))
  );
}

function findGoNgc(workspaceRoot: string): string {
  const fromWorkspace = findUp(workspaceRoot, (dir) => fs.existsSync(path.join(dir, 'go-ngc')));
  const candidate = path.join(fromWorkspace, 'go-ngc');
  return fs.existsSync(candidate) ? candidate : 'go-ngc';
}

export function createAngularGoVitestConfig(options: AngularGoVitestConfigOptions = {}): ViteUserConfig & { test?: any } {
  const workspaceRoot = findWorkspaceRoot();
  const root = options.root ? path.resolve(workspaceRoot, options.root) : path.join(workspaceRoot, 'src');
  const testEnvironment = options.testEnvironment ?? 'happy-dom';
  const compilerPath = options.compilerPath ?? findGoNgc(workspaceRoot);
  const project = options.project
    ? path.resolve(workspaceRoot, options.project)
    : path.join(workspaceRoot, 'tsconfig.spec.json');

  return {
    root,
    define: {
      'process.env.NG_GO_VERSION': JSON.stringify('1.0.0')
    },
    plugins: [
      {
        name: 'angular-go-testbed-resolver',
        enforce: 'pre',
        async resolveId(source: string, importer: string | undefined) {
          if (
            source.startsWith('@angular/') ||
            source === '@angular' ||
            source.startsWith('rxjs') ||
            source === 'happy-dom' ||
            source === 'jsdom'
          ) {
            const target = path.resolve(workspaceRoot, 'node_modules', source);
            const resolved = await this.resolve(target, importer, { skipSelf: true });
            return resolved;
          }
          return null;
        }
      } as any,
      angularGo({
        mode: 'server',
        compilerPath,
        project,
        projectRoot: workspaceRoot,
        outDir: options.outDir ?? 'out-tsc/spec',
        compile: true,
        link: true,
        hmr: false,
        recompileOnChange: true,
        compilationMode: options.compilationMode
      }) as any
    ],
    ssr: {
      noExternal: true
    },
    test: {
      globals: true,
      environment: testEnvironment,
      setupFiles: options.setupFiles,
      maxWorkers: 1, // Limit workers to 1 to run tests sequentially
      fileParallelism: false, // Disable parallel file execution to prevent concurrent compiler map writes
      server: {
        deps: {
          inline: true
        }
      },
      deps: {
        inline: true
      }
    },
    resolve: {
      mainFields: ['module'],
      preserveSymlinks: true,
      dedupe: [
        '@angular/core',
        '@angular/common',
        '@angular/platform-browser',
        '@angular/router',
        '@angular/forms',
        'rxjs'
      ]
    }
  } as any;
}
