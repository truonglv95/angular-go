import { createBuilder, BuilderContext, BuilderOutput } from '@angular-devkit/architect';
import { createServer, build, createLogger } from 'vite';
import path from 'node:path';
import fs from 'node:fs';
import * as angularGoModule from '../../vite-plugin/index.js';
import { createRequire } from 'node:module';
import { resolveGoNgcPath } from '../../compiler-path.js';
import { formatAngularDevBundleOutput, formatAngularDevServerReadyOutput } from '../shared/angular-cli-dev-output.js';
import { bundleCompilerOutputsWithRolldown } from '../shared/app-bundler.js';
import { collectBundleSizeSummary, CompilerOutputFile } from '../shared/bundle-stats.js';
import { normalizeStringList, normalizeGlobalEntries } from '../application/index.js';
import { startSpinner, stopSpinner } from '../shared/spinner.js';

const angularGo = ((angularGoModule as any).default?.default || (angularGoModule as any).default || angularGoModule) as any;
const require = createRequire(import.meta.url);

export default createBuilder<any, BuilderOutput>(async (options, context): Promise<BuilderOutput> => {
  const cliOutput = options.cliOutput ?? 'angular';
  const useAngularOutput = cliOutput === 'angular';

  if (!useAngularOutput && cliOutput !== 'silent') {
    context.logger.info(`Starting development server with go-ngc and Vite...`);
  }

  try {
    const workspaceRoot = context.workspaceRoot;
    
    let buildOptions: any = {};
    if (options.buildTarget) {
      const parsed = parseBuildTarget(options.buildTarget);
      buildOptions = await context.getTargetOptions({ project: parsed.project, target: parsed.target }) as any;
      if (parsed.configuration) {
        const configs = parsed.configuration.split(',');
        for (const config of configs) {
          const configOptions = await context.getTargetOptions({
            project: parsed.project,
            target: parsed.target,
            configuration: config.trim()
          }) as any;
          buildOptions = {
            ...buildOptions,
            ...configOptions,
            define: {
              ...(buildOptions.define || {}),
              ...(configOptions.define || {})
            }
          };
        }
      }
    }

    const tsConfigPath = path.resolve(workspaceRoot, buildOptions.tsConfig || 'tsconfig.app.json');
    const browserEntry = path.resolve(workspaceRoot, buildOptions.browser || 'src/main.ts');
    
    const compilerPath = resolveGoNgcPath(workspaceRoot, buildOptions.compilerPath || options.compilerPath || 'go-ngc', import.meta.url);

    const host = options.host || '127.0.0.1';
    const port = options.port ?? 4200;

    if (options.inspect) {
      context.logger.warn(`angular-go-build:dev-server does not support 'inspect' option in non-SSR mode.`);
    }

    const baseHref = normalizeServeBase(options.servePath || buildOptions.baseHref || '/');

    // Resolve proxyConfig
    let proxy: any = undefined;
    if (options.proxyConfig) {
      const proxyPath = path.resolve(workspaceRoot, options.proxyConfig);
      if (fs.existsSync(proxyPath)) {
        if (proxyPath.endsWith('.json')) {
          try {
            proxy = JSON.parse(fs.readFileSync(proxyPath, 'utf8'));
          } catch (e: any) {
            context.logger.error(`Failed to parse proxy config JSON: ${e.message}`);
          }
        } else {
          try {
            const imported = require(proxyPath);
            proxy = imported.default || imported;
          } catch (requireErr: any) {
            try {
              const imported = await import(proxyPath);
              proxy = imported.default || imported;
            } catch (importErr: any) {
              context.logger.error(`Failed to load proxy config: ${importErr.message || importErr}`);
            }
          }
        }
      } else {
        context.logger.warn(`Proxy config file not found: ${proxyPath}`);
      }
    }

    // Resolve allowedHosts
    let allowedHosts: any = undefined;
    if (options.allowedHosts !== undefined) {
      allowedHosts = options.allowedHosts;
    }

    const external = buildOptions.externalDependencies || [];

    const hmrEnabled = options.hmr !== undefined ? options.hmr : (options.liveReload !== false);
    const liveReloadEnabled = options.liveReload !== false;
    const polyfills = normalizeStringList(buildOptions.polyfills || []);
    const styles = normalizeGlobalEntries(buildOptions.styles || [], 'style');
    const scripts = normalizeGlobalEntries(buildOptions.scripts || [], 'script');

    // Style preprocessor options
    const cssConfig: any = {};
    let styleIncludePaths: string[] | undefined;
    if (buildOptions.stylePreprocessorOptions && typeof buildOptions.stylePreprocessorOptions === 'object') {
      const includePaths = (buildOptions.stylePreprocessorOptions.includePaths || []).map((p: string) =>
        path.resolve(workspaceRoot, p)
      );
      if (includePaths.length > 0) {
        styleIncludePaths = includePaths;
        cssConfig.preprocessorOptions = {
          sass: {
            includePaths
          },
          scss: {
            includePaths
          }
        };
      }
    }

    // Define the dry-run configuration for calculating stats
    const dryRunConfig: any = {
      configFile: false,
      root: path.dirname(browserEntry),
      base: baseHref,
      css: cssConfig,
      mode: 'development',
      define: {
        ...(buildOptions.define || {}),
        ...(options.define || {}),
      },
      resolve: {
        mainFields: ['module'],
        alias: [],
        preserveSymlinks: buildOptions.preserveSymlinks === true || options.preserveSymlinks === true
      },
      cacheDir: path.resolve(workspaceRoot, '.angular/cache/angular-go-dry-run'),
      optimizeDeps: {
        disabled: true,
        noDiscovery: true,
      },
      plugins: [], // will be populated from viteConfig
      logLevel: 'silent' as const,
      build: {
        write: false,
        minify: false,
        sourcemap: false,
        rollupOptions: {
          input: {},
          output: {
            entryFileNames: 'assets/[name].js',
            chunkFileNames: 'assets/[name].js',
            assetFileNames: 'assets/[name].[ext]',
          }
        }
      }
    };

    const setDryRunBuildInProgress = (angularGoModule as any).setDryRunBuildInProgress;
    
    async function printRebuildSummary(reason: string, compilerOutputs: CompilerOutputFile[]) {
      const started = Date.now();
      try {
        setDryRunBuildInProgress(true);
        
        const input: Record<string, string> = {
          main: browserEntry
        };



        // Polyfills entry
        if (polyfills.length > 0) {
          const firstPolyfill = polyfills[0];
          const polyfillPath = path.resolve(workspaceRoot, firstPolyfill);
          if (fs.existsSync(polyfillPath)) {
            input['polyfills'] = polyfillPath;
          }
        }

        // Style entry points
        for (const entry of styles) {
          const stylePath = path.resolve(workspaceRoot, entry.input);
          if (fs.existsSync(stylePath)) {
            input[entry.bundleName] = stylePath;
          }
        }

        // Script entry points
        for (const entry of scripts) {
          const scriptPath = path.resolve(workspaceRoot, entry.input);
          if (fs.existsSync(scriptPath)) {
            input[entry.bundleName] = scriptPath;
          }
        }

        dryRunConfig.build.rollupOptions.input = input;

        dryRunConfig.build.rollupOptions.external = (id: string) => {
          if (id.startsWith('.') || id.startsWith('/') || id.startsWith('\\') || path.isAbsolute(id)) {
            return false;
          }
          if (id.endsWith('.css') || id.endsWith('.scss') || id.endsWith('.sass') || id.includes('?')) {
            return false;
          }
          return true;
        };

        const rollupOutput = await build(dryRunConfig);
        const outputs = Array.isArray(rollupOutput) ? rollupOutput : [rollupOutput as any];
        const bundleRecord: Record<string, any> = {};
        for (const out of outputs) {
          if (out && (out as any).output) {
            for (const chunkOrAsset of (out as any).output) {
              bundleRecord[chunkOrAsset.fileName] = chunkOrAsset;
            }
          }
        }
        
        const appBundleRecord = await bundleCompilerOutputsWithRolldown(compilerOutputs, [
          { name: 'main', fileName: path.basename(browserEntry, path.extname(browserEntry)) + '.js' },
          ...polyfills.map(value => ({
            name: path.basename(value, path.extname(value)),
            fileName: path.basename(value, path.extname(value)) + '.js'
          }))
        ]);
        const drySummary = collectBundleSizeSummary(bundleRecord);
        const appBundleSummary = collectBundleSizeSummary(appBundleRecord);
        const cssRows = drySummary.initial.filter(row => row.kind === 'css');
        const lazyCssRows = drySummary.lazy.filter(row => row.kind === 'css');
        const initial = [...appBundleSummary.initial, ...cssRows];
        const lazy = [...appBundleSummary.lazy, ...lazyCssRows];
        const summary = {
          initial,
          lazy,
          initialTotalRawSize: initial.reduce((total, row) => total + row.rawSize, 0)
        };
        const durationMs = Date.now() - started;
        
        context.logger.info(formatAngularDevBundleOutput(summary, { durationMs }));
      } catch (err: any) {
        context.logger.error(`[angular-go] Failed to generate bundle summary: ${err.stack || err.message || err}`);
      } finally {
        setDryRunBuildInProgress(false);
      }
    }

    let localUrl = '';
    let hasPrintedWatchMessage = false;
    let hasPrintedUrlBlock = false;
    let compileSucceeded = false;

    const viteConfig: any = {
      root: path.dirname(browserEntry),
      configFile: false,
      base: baseHref,
      css: cssConfig,
      cacheDir: path.resolve(workspaceRoot, '.angular/cache/angular-go-vite'),
      logLevel: cliOutput === 'silent' ? 'silent' : (useAngularOutput ? 'info' : undefined),
      customLogger: useAngularOutput ? createAngularDevServerLogger(context) : undefined,
      define: {
        ...(buildOptions.define || {}),
        ...(options.define || {}),
      },
      server: {
        host,
        port,
        https: options.ssl ? (
          options.sslKey || options.sslCert ? {
            key: options.sslKey ? fs.readFileSync(path.resolve(workspaceRoot, options.sslKey)) : undefined,
            cert: options.sslCert ? fs.readFileSync(path.resolve(workspaceRoot, options.sslCert)) : undefined,
          } : true
        ) : undefined,
        headers: options.headers,
        open: options.open,
        proxy,
        allowedHosts,
        hmr: hmrEnabled || liveReloadEnabled ? undefined : false,
        watch: options.watch === false ? null : (options.poll ? { usePolling: true, interval: options.poll } : undefined),
      },
      plugins: [
        angularGo({
          mode: 'server',
          compilerPath: compilerPath,
          project: path.relative(workspaceRoot, tsConfigPath),
          outDir: 'out-tsc/app',
          compile: true,
          link: true,
          hmr: hmrEnabled,
          recompileOnChange: options.watch !== false,
          styleIncludePaths,
          appBundle: true,
          appBundleEntryFileNames: [
            path.basename(browserEntry, path.extname(browserEntry)) + '.js',
            ...polyfills.map(value => path.basename(value, path.extname(value)) + '.js')
          ],
          devBundleSummary: false, // Turn off plugin's native console.log to avoid duplicates
          onCompileStart: () => {
            if (useAngularOutput) {
              startSpinner('Building...');
            }
          },
          onCompileComplete: async ({ reason, outputs, durationMs, error }: { reason: string; durationMs: number; outputs: CompilerOutputFile[]; error?: any }) => {
            if (useAngularOutput) {
              stopSpinner();
              if (error) {
                const durationSec = (durationMs / 1000).toFixed(3);
                const ts = new Date().toISOString();
                context.logger.error(`Application bundle generation failed. [${durationSec} seconds] - ${ts}\n`);
                if (!hasPrintedWatchMessage) {
                  hasPrintedWatchMessage = true;
                  context.logger.info(`Watch mode enabled. Watching for file changes...`);
                }
              } else {
                compileSucceeded = true;
                await printRebuildSummary(reason, outputs);
                if (!hasPrintedWatchMessage) {
                  hasPrintedWatchMessage = true;
                  context.logger.info(`Watch mode enabled. Watching for file changes...`);
                }
                if (!hasPrintedUrlBlock && localUrl) {
                  hasPrintedUrlBlock = true;
                  context.logger.info(
                    `NOTE: Raw file sizes do not reflect development server per-request transformations.\n` +
                    `  ➜  Local:   ${localUrl}\n` +
                    `  ➜  press h + enter to show help`
                  );
                }
              }
            }
          }
        }),
        createHtmlAssetsPlugin(
          path.dirname(browserEntry),
          polyfills,
          styles,
          scripts,
          workspaceRoot
        ),
        createLoaderPlugin(buildOptions.loader)
      ],
      resolve: {
        mainFields: ['module'],
        alias: [],
        preserveSymlinks: buildOptions.preserveSymlinks === true || options.preserveSymlinks === true
      }
    };

    // Filter plugins for dryRunConfig
    dryRunConfig.plugins = viteConfig.plugins.filter(
      (p: any) => p && p.name && !p.name.startsWith('vite:')
    );

    // External dependencies handling
    if (external.length > 0) {
      viteConfig.optimizeDeps = viteConfig.optimizeDeps || {};
      viteConfig.optimizeDeps.exclude = [
        ...(viteConfig.optimizeDeps.exclude || []),
        ...external
      ];
      viteConfig.build = viteConfig.build || {};
      viteConfig.build.rollupOptions = viteConfig.build.rollupOptions || {};
      viteConfig.build.rollupOptions.external = external;
      
      dryRunConfig.optimizeDeps.exclude = [
        ...(dryRunConfig.optimizeDeps.exclude || []),
        ...external
      ];
      dryRunConfig.build.rollupOptions.external = external;
    }

    if (options.prebundle === false) {
      viteConfig.optimizeDeps = {
        ...(viteConfig.optimizeDeps || {}),
        disabled: true
      };
    } else if (options.prebundle && typeof options.prebundle === 'object' && Array.isArray(options.prebundle.exclude)) {
      viteConfig.optimizeDeps = {
        ...(viteConfig.optimizeDeps || {}),
        exclude: [
          ...(viteConfig.optimizeDeps?.exclude || []),
          ...options.prebundle.exclude
        ]
      };
    }

    // Loaders mapping
    if (buildOptions.loader && typeof buildOptions.loader === 'object') {
      viteConfig.esbuild = viteConfig.esbuild || {};
      viteConfig.esbuild.loader = {
        ...(viteConfig.esbuild.loader || {}),
        ...buildOptions.loader
      };
      dryRunConfig.esbuild = dryRunConfig.esbuild || {};
      dryRunConfig.esbuild.loader = {
        ...(dryRunConfig.esbuild.loader || {}),
        ...buildOptions.loader
      };
    }

    // Resolve fileReplacements
    if (buildOptions.fileReplacements && Array.isArray(buildOptions.fileReplacements)) {
      for (const replacement of buildOptions.fileReplacements) {
        const replacePath = path.resolve(workspaceRoot, replacement.replace);
        const withPath = path.resolve(workspaceRoot, replacement.with);
        viteConfig.resolve.alias.push({
          find: replacePath,
          replacement: withPath
        });
        dryRunConfig.resolve.alias.push({
          find: replacePath,
          replacement: withPath
        });
      }
    }

    const server = await createServer(viteConfig);
    await server.listen();
    
    const address = server.httpServer?.address();
    const resolvedHost = typeof address === 'object' && address ? address.address : host;
    const resolvedPort = typeof address === 'object' && address ? address.port : port;
    localUrl = formatLocalUrl(options.ssl === true, host, resolvedHost, resolvedPort);

    if (!useAngularOutput && cliOutput !== 'silent') {
      context.logger.info(`Vite Dev Server is running at ${localUrl}`);
    } else if (useAngularOutput && !hasPrintedUrlBlock && compileSucceeded) {
      hasPrintedUrlBlock = true;
      context.logger.info(
        `NOTE: Raw file sizes do not reflect development server per-request transformations.\n` +
        `  ➜  Local:   ${localUrl}\n` +
        `  ➜  press h + enter to show help`
      );
    }

    return new Promise<BuilderOutput>((resolve) => {
      const shutdown = async () => {
        await server.close();
        resolve({ success: true });
      };
      process.once('SIGINT', shutdown);
      process.once('SIGTERM', shutdown);
    });
  } catch (error: any) {
    context.logger.error(`Dev Server failed: ${error.message || error}`);
    return { success: false, error: error.message };
  }
});

function createAngularDevServerLogger(context: BuilderContext) {
  const baseLogger = createLogger();
  const shouldIgnore = (message: string): boolean => {
    const msg = String(message);
    return msg.includes('The file does not exist at') || msg.includes('optimizeDeps.exclude');
  };

  return {
    ...baseLogger,
    info(message: string, options?: any) {
      if (shouldIgnore(message)) {
        return;
      }
      if (
        message.includes('ready in') ||
        message.includes('Local:') ||
        message.includes('Network:') ||
        message.includes('press h + enter') ||
        message.includes('Forced re-optimization of dependencies') ||
        message.includes('[optimizer] bundling dependencies') ||
        message.includes('[vite] (client)') ||
        message.includes('page reload') ||
        message.includes('hmr update') ||
        message.includes('hot update')
      ) {
        return;
      }
      baseLogger.info(message, options);
    },
    warn(message: string, options?: any) {
      if (shouldIgnore(message)) {
        return;
      }
      baseLogger.warn(message, options);
    },
    warnOnce(message: string, options?: any) {
      if (shouldIgnore(message)) {
        return;
      }
      baseLogger.warnOnce(message, options);
    },
    error(message: string, options?: any) {
      if (shouldIgnore(message)) {
        return;
      }
      baseLogger.error(message, options);
    },
    clearScreen() {
      // no-op
    }
  };
}

function parseBuildTarget(buildTarget: string) {
  const firstColon = buildTarget.indexOf(':');
  const secondColon = firstColon === -1 ? -1 : buildTarget.indexOf(':', firstColon + 1);
  if (firstColon === -1) {
    throw new Error(`Invalid buildTarget '${buildTarget}'. Expected project:target[:configuration].`);
  }
  const project = buildTarget.slice(0, firstColon);
  const target = secondColon === -1 ? buildTarget.slice(firstColon + 1) : buildTarget.slice(firstColon + 1, secondColon);
  if (secondColon === -1) {
    return { project, target };
  }
  return { project, target, configuration: buildTarget.slice(secondColon + 1) };
}

function normalizeServeBase(value: string): string {
  if (!value || value === '/') return '/';
  const withLeadingSlash = value.startsWith('/') ? value : `/${value}`;
  return withLeadingSlash.endsWith('/') ? withLeadingSlash : `${withLeadingSlash}/`;
}

function formatLocalUrl(https: boolean, requestedHost: string, resolvedHost: string, port: number): string {
  const protocol = https ? 'https' : 'http';
  const host = requestedHost && requestedHost !== '0.0.0.0' && requestedHost !== '::'
    ? requestedHost
    : resolvedHost;
  const formattedHost = host.includes(':') && !host.startsWith('[') ? `[${host}]` : host;
  return `${protocol}://${formattedHost}:${port}/`;
}

function createLoaderPlugin(loaderOptions: Record<string, string> | undefined) {
  return {
    name: 'angular-go-loader-plugin',
    enforce: 'pre' as const,
    resolveId(source: string, importer: string | undefined) {
      if (!loaderOptions || !importer) return null;
      const cleanSource = source.split('?')[0];
      const ext = path.extname(cleanSource);
      const loaderType = loaderOptions[ext];
      if (loaderType) {
        let resolvedPath = '';
        if (source.startsWith('.') || source.startsWith('/') || source.startsWith('\\')) {
          resolvedPath = path.resolve(path.dirname(importer), cleanSource);
        } else {
          resolvedPath = cleanSource;
        }
        return `${resolvedPath}?angular-go-loader=${loaderType}`;
      }
      return null;
    },
    load(id: string) {
      if (!loaderOptions) return null;
      const match = id.match(/\?angular-go-loader=(.+)$/);
      if (match) {
        const loaderType = match[1];
        const cleanId = id.split('?')[0];
        if (fs.existsSync(cleanId)) {
          const content = fs.readFileSync(cleanId);
          if (loaderType === 'text') {
            return `export default ${JSON.stringify(content.toString('utf8'))};`;
          } else if (loaderType === 'dataurl') {
            const mimeType = 'application/octet-stream';
            return `export default "data:${mimeType};base64,${content.toString('base64')}";`;
          } else if (loaderType === 'binary') {
            return `const base64 = "${content.toString('base64')}";
const binary = atob(base64);
const bytes = new Uint8Array(binary.length);
for (let i = 0; i < binary.length; i++) {
  bytes[i] = binary.charCodeAt(i);
}
export default bytes;`;
          }
        }
      }
      return null;
    }
  };
}

function toViteUrl(filePath: string, viteRoot: string): string {
  const absPath = path.resolve(filePath);
  if (absPath.startsWith(viteRoot)) {
    return '/' + path.relative(viteRoot, absPath).replace(/\\/g, '/');
  }
  // Outside root, use /@fs/ prefix
  return '/@fs' + (absPath.startsWith('/') ? '' : '/') + absPath.replace(/\\/g, '/');
}

function createHtmlAssetsPlugin(
  viteRoot: string,
  polyfills: string[],
  styles: { input: string; inject: boolean }[],
  scripts: { input: string; inject: boolean }[],
  workspaceRoot: string
) {
  return {
    name: 'angular-go-html-assets',
    transformIndexHtml(html: string) {
      const tags: string[] = [];

      // 1. Inject Polyfills (as module scripts)
      for (const polyfill of polyfills) {
        let absPath = '';
        if (polyfill.startsWith('.') || polyfill.startsWith('/') || polyfill.startsWith('\\') || fs.existsSync(path.resolve(workspaceRoot, polyfill))) {
          absPath = path.resolve(workspaceRoot, polyfill);
        } else {
          try {
            absPath = require.resolve(polyfill, {
              paths: [workspaceRoot]
            });
          } catch {
            absPath = polyfill;
          }
        }
        const url = toViteUrl(absPath, viteRoot);
        tags.push(`<script type="module" src="${url}"></script>`);
      }

      // 2. Inject Styles (as link tags)
      for (const style of styles) {
        if (!style.inject) continue;
        const absPath = path.resolve(workspaceRoot, style.input);
        const url = toViteUrl(absPath, viteRoot);
        tags.push(`<link rel="stylesheet" href="${url}">`);
      }

      // 3. Inject Scripts (as normal scripts)
      for (const script of scripts) {
        if (!script.inject) continue;
        const absPath = path.resolve(workspaceRoot, script.input);
        const url = toViteUrl(absPath, viteRoot);
        tags.push(`<script src="${url}"></script>`);
      }

      if (tags.length === 0) {
        return html;
      }

      // Inject tags before the closing </head> tag
      const joinedTags = tags.join('\n  ');
      if (html.includes('</head>')) {
        return html.replace('</head>', `  ${joinedTags}\n</head>`);
      }
      return html + '\n' + joinedTags;
    }
  };
}
