import { createBuilder, BuilderContext, BuilderOutput } from '@angular-devkit/architect';
import { createServer } from 'vite';
import path from 'node:path';
import fs from 'node:fs';
import * as angularGoModule from 'vite-plugin-angular-go';
import { createRequire } from 'node:module';

const angularGo = ((angularGoModule as any).default?.default || (angularGoModule as any).default || angularGoModule) as any;
const require = createRequire(import.meta.url);

export default createBuilder<any, BuilderOutput>(async (options, context): Promise<BuilderOutput> => {
  context.logger.info(`Starting development server with go-ngc and Vite...`);

  try {
    const workspaceRoot = context.workspaceRoot;
    
    let buildOptions: any = {};
    if (options.buildTarget) {
      buildOptions = await context.getTargetOptions(parseBuildTarget(options.buildTarget) as any);
    }

    const tsConfigPath = path.resolve(workspaceRoot, buildOptions.tsConfig || 'tsconfig.app.json');
    const browserEntry = path.resolve(workspaceRoot, buildOptions.browser || 'src/main.ts');
    
    // Resolve go-ngc compiler path
    let compilerPath = '';
    let currentDir = workspaceRoot;
    while (currentDir !== path.dirname(currentDir)) {
      const candidate = path.join(currentDir, 'go-ngc');
      if (fs.existsSync(candidate)) {
        compilerPath = candidate;
        break;
      }
      currentDir = path.dirname(currentDir);
    }
    if (!compilerPath) {
      compilerPath = 'go-ngc'; // Fallback to PATH resolution
    }

    const host = options.host || '127.0.0.1';
    const port = options.port ?? 4200;

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

    const hmrEnabled = options.hmr === true;
    const liveReloadEnabled = options.liveReload !== false;
    const viteConfig: any = {
      root: path.dirname(browserEntry),
      configFile: false,
      base: baseHref,
      define: {
        ...(buildOptions.define || {}),
        ...(options.define || {}),
      },
      server: {
        host,
        port,
        https: options.ssl ? {
          key: options.sslKey ? fs.readFileSync(path.resolve(workspaceRoot, options.sslKey)) : undefined,
          cert: options.sslCert ? fs.readFileSync(path.resolve(workspaceRoot, options.sslCert)) : undefined,
        } : undefined,
        headers: options.headers,
        open: options.open,
        proxy,
        allowedHosts,
        hmr: hmrEnabled || liveReloadEnabled ? undefined : false,
        watch: options.poll ? { usePolling: true, interval: options.poll } : undefined,
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
          recompileOnChange: options.watch !== false
        }),
        createLoaderPlugin(buildOptions.loader)
      ],
      resolve: {
        mainFields: ['module'],
        alias: [],
        preserveSymlinks: buildOptions.preserveSymlinks === true || options.preserveSymlinks === true
      }
    };

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
      }
    }

    const server = await createServer(viteConfig);
    await server.listen();
    
    const address = server.httpServer?.address();
    const resolvedHost = typeof address === 'object' && address ? address.address : host;
    const resolvedPort = typeof address === 'object' && address ? address.port : port;
    context.logger.info(`Vite Dev Server is running at http://${resolvedHost}:${resolvedPort}/`);

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
