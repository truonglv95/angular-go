import { UserConfig as ViteUserConfig } from 'vite';
import angularGo from '../vite-plugin/index.js';
import path from 'node:path';


export interface AngularGoVitestConfigOptions {
  project?: string;
  compilerPath?: string;
  compilationMode?: 'global' | 'local';
  testEnvironment?: 'jsdom' | 'happy-dom' | 'browser';
  setupFiles?: string[];
  root?: string;
  outDir?: string;
}

export function createAngularGoVitestConfig(options: AngularGoVitestConfigOptions = {}): ViteUserConfig & { test?: any } {
  const root = options.root ?? 'src';
  const testEnvironment = options.testEnvironment ?? 'happy-dom';

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
            const target = path.resolve(process.cwd(), 'node_modules', source);
            const resolved = await this.resolve(target, importer, { skipSelf: true });
            return resolved;
          }
          return null;
        }
      } as any,
      angularGo({
        mode: 'server',
        compilerPath: options.compilerPath ?? '../../go-ngc',
        project: options.project ?? 'tsconfig.spec.json',
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
