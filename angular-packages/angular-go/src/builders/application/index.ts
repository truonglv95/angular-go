import { createBuilder, BuilderContext, BuilderOutput } from '@angular-devkit/architect';
import { build } from 'vite';
import path from 'node:path';
import fs from 'node:fs';
import crypto from 'node:crypto';
import * as angularGoModule from '../../vite-plugin/index.js';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import { resolveGoNgcPath } from '../../compiler-path.js';

const angularGo = ((angularGoModule as any).default?.default || (angularGoModule as any).default || angularGoModule) as any;
const require = createRequire(import.meta.url);
const __dirname = path.dirname(fileURLToPath(import.meta.url));

export interface GlobalEntry {
  input: string;
  bundleName: string;
  inject: boolean;
}

export default createBuilder<any, BuilderOutput>(async (options, context): Promise<BuilderOutput> => {
  context.logger.info(`Building application with go-ngc and Vite...`);

  try {
    const workspaceRoot = context.workspaceRoot;
    if (options.aot === false) {
      return { success: false, error: 'angular-go-build requires AOT compilation. The `aot: false` option is not supported.' };
    }
    const unsupportedError = getUnsupportedFeatureError(options);
    if (unsupportedError) {
      return { success: false, error: unsupportedError };
    }
    warnUnsupportedOptions(options, context, [
      'inlineStyleLanguage',
      'i18nMissingTranslation',
      'i18nDuplicateTranslation',
      'security',
      'webWorkerTsConfig',
      'watch',
      'poll'
    ]);
    const tsConfigPath = path.resolve(workspaceRoot, options.tsConfig || 'tsconfig.app.json');
    const browserEntry = path.resolve(workspaceRoot, options.browser || 'src/main.ts');
    
    const compilerPath = resolveGoNgcPath(workspaceRoot, options.compilerPath || 'go-ngc', import.meta.url);

    // Normalize outputPath
    let baseOutDir = 'dist';
    let browserSubdir = '';
    let mediaSubdir = 'media';
    if (typeof options.outputPath === 'string') {
      baseOutDir = options.outputPath;
    } else if (options.outputPath && typeof options.outputPath === 'object') {
      baseOutDir = options.outputPath.base || 'dist';
      browserSubdir = options.outputPath.browser !== undefined ? options.outputPath.browser : 'browser';
      mediaSubdir = options.outputPath.media !== undefined ? options.outputPath.media : 'media';
    }
    const outDir = browserSubdir ? path.join(baseOutDir, browserSubdir) : baseOutDir;
    const absoluteOutDir = path.resolve(workspaceRoot, outDir);

    const parseSize = (sizeStr: string | undefined): number | null => {
      if (!sizeStr) return null;
      const match = sizeStr.match(/^(\d+(?:\.\d+)?)\s*(B|kB|MB|GB|kb|mb|gb)?$/i);
      if (!match) return null;
      const val = parseFloat(match[1]);
      const unit = (match[2] || 'B').toLowerCase();
      switch (unit) {
        case 'kb':
        case 'k':
          return val * 1024;
        case 'mb':
        case 'm':
          return val * 1024 * 1024;
        case 'gb':
        case 'g':
          return val * 1024 * 1024 * 1024;
        default:
          return val;
      }
    };

    const formatBytes = (bytes: number): string => {
      if (bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'kB', 'MB', 'GB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    // Output hashing configuration
    const hashing = options.outputHashing || 'none';
    const useJsHash = hashing === 'all' || hashing === 'bundles';
    const useAssetHash = hashing === 'all' || hashing === 'media';
    const namedChunks = options.namedChunks !== false;

    // Base Href / Deploy URL configuration
    const baseHref = options.baseHref || '/';
    const assetUrlBase = normalizeUrlBase(options.deployUrl || baseHref);
    const preloadInitial = !(options.index && typeof options.index === 'object' && options.index.preloadInitial === false);

    // Optimization mapping
    let minify: 'esbuild' | false = 'esbuild';
    let cssMinify: boolean = true;
    if (options.optimization === false) {
      minify = false;
      cssMinify = false;
    } else if (options.optimization && typeof options.optimization === 'object') {
      minify = options.optimization.scripts !== false ? 'esbuild' : false;
      if (options.optimization.styles === false) {
        cssMinify = false;
      } else if (options.optimization.styles && typeof options.optimization.styles === 'object') {
        cssMinify = options.optimization.styles.minify !== false;
      }
    }

    // Sourcemap mapping
    let useJsSourcemap: boolean | 'hidden' = false;
    let useCssSourcemap: boolean = false;
    if (options.sourceMap === true) {
      useJsSourcemap = true;
      useCssSourcemap = true;
    } else if (options.sourceMap && typeof options.sourceMap === 'object') {
      if (options.sourceMap.scripts) {
        useJsSourcemap = options.sourceMap.hidden ? 'hidden' : true;
      }
      if (options.sourceMap.styles) {
        useCssSourcemap = true;
      }
    }

    // Style preprocessor options
    const cssConfig: any = {
      devSourcemap: useCssSourcemap
    };
    let styleIncludePaths: string[] | undefined;
    if (options.stylePreprocessorOptions && typeof options.stylePreprocessorOptions === 'object') {
      const includePaths = (options.stylePreprocessorOptions.includePaths || []).map((p: string) =>
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

    const polyfills = normalizeStringList(options.polyfills || []);
    const styles = normalizeGlobalEntries(options.styles || [], 'style');
    const scripts = normalizeGlobalEntries(options.scripts || [], 'script');
    const globalStyleNames = new Set(styles.map((entry) => entry.bundleName));

    const viteConfig: any = {
      root: path.dirname(browserEntry),
      configFile: false,
      base: assetUrlBase,
      define: options.define || undefined,
      css: cssConfig,
      build: {
        outDir: path.relative(path.dirname(browserEntry), absoluteOutDir),
        emptyOutDir: options.deleteOutputPath !== false,
        sourcemap: useJsSourcemap,
        minify: minify,
        cssMinify: cssMinify,
      },
      plugins: [
        createPolyfillMainPlugin(browserEntry, polyfills, workspaceRoot, tsConfigPath),
        removeGlobalStyleJsChunksPlugin(globalStyleNames),
        angularGo({
          mode: 'server',
          compilerPath: compilerPath,
          project: path.relative(workspaceRoot, tsConfigPath),
          outDir: 'out-tsc/app',
          compile: true,
          link: true,
          hmr: false,
          recompileOnChange: false,
          styleIncludePaths
        }),
        createLoaderPlugin(options.loader)
      ],
      resolve: {
        tsconfigPaths: true,
        mainFields: ['module'],
        conditions: Array.isArray(options.conditions) ? options.conditions : undefined,
        alias: [],
        preserveSymlinks: options.preserveSymlinks === true
      }
    };

    // Loaders mapping
    if (options.loader && typeof options.loader === 'object') {
      viteConfig.esbuild = viteConfig.esbuild || {};
      viteConfig.esbuild.loader = {
        ...(viteConfig.esbuild.loader || {}),
        ...options.loader
      };
    }

    // File replacements
    if (options.fileReplacements && Array.isArray(options.fileReplacements)) {
      for (const replacement of options.fileReplacements) {
        const replacePath = path.resolve(workspaceRoot, replacement.replace);
        const withPath = path.resolve(workspaceRoot, replacement.with);
        viteConfig.resolve.alias.push({
          find: replacePath,
          replacement: withPath
        });
      }
    }

    // Index file configuration
    let indexInput = '';
    let indexOutput = 'index.html';
    if (options.index !== false) {
      if (typeof options.index === 'string') {
        indexInput = options.index;
      } else if (options.index && typeof options.index === 'object') {
        indexInput = options.index.input || 'src/index.html';
        indexOutput = options.index.output || 'index.html';
      } else {
        indexInput = 'src/index.html';
      }
    }

    const entryPoints: Record<string, string> = {
      main: browserEntry
    };
    if (indexInput) {
      const indexHtmlPath = path.resolve(workspaceRoot, indexInput);
      if (fs.existsSync(indexHtmlPath)) {
        entryPoints['index'] = indexHtmlPath;
      }
    }

    // Style inputs
    for (let i = 0; i < styles.length; i++) {
      const stylePath = path.resolve(workspaceRoot, styles[i].input);
      if (fs.existsSync(stylePath)) {
        entryPoints[styles[i].bundleName] = stylePath;
      }
    }

    // Script inputs
    for (let i = 0; i < scripts.length; i++) {
      const scriptPath = path.resolve(workspaceRoot, scripts[i].input);
      if (fs.existsSync(scriptPath)) {
        entryPoints[scripts[i].bundleName] = scriptPath;
      }
    }

    viteConfig.build.rollupOptions = {
      input: entryPoints,
      external: options.externalDependencies || undefined,
      output: {
        entryFileNames: useJsHash ? 'assets/[name]-[hash].js' : 'assets/[name].js',
        chunkFileNames: useJsHash 
          ? (namedChunks ? 'assets/[name]-[hash].js' : 'assets/[hash].js') 
          : 'assets/[name].js',
        assetFileNames: useAssetHash
          ? `${mediaSubdir}/[name]-[hash].[ext]`
          : `${mediaSubdir}/[name].[ext]`,
      }
    };

    const buildResult = await build(viteConfig);
    const outputs = (Array.isArray(buildResult) ? buildResult : [buildResult]) as any[];

    // Post-process index.html to inject global styles and scripts
    const finalIndexHtmlPath = path.resolve(absoluteOutDir, indexOutput);
    if (indexInput && fs.existsSync(finalIndexHtmlPath) && options.index !== false) {
      let htmlContent = fs.readFileSync(finalIndexHtmlPath, 'utf8');

      // Find compiled CSS files for style entry points
      const injectedStyles: string[] = [];
      for (let i = 0; i < styles.length; i++) {
        if (!styles[i].inject) continue;
        const bundleName = styles[i].bundleName;
        for (const out of outputs) {
          for (const item of out.output) {
            if (item.type === 'asset' && item.fileName.endsWith('.css') && 
                (item.name === bundleName || item.fileName.includes(bundleName))) {
              injectedStyles.push(`${assetUrlBase}${item.fileName}`);
            }
          }
        }
      }

      // Find compiled JS files for script entry points
      const injectedScripts: string[] = [];
      for (let i = 0; i < scripts.length; i++) {
        if (!scripts[i].inject) continue;
        const bundleName = scripts[i].bundleName;
        for (const out of outputs) {
          for (const item of out.output) {
            if (item.type === 'chunk' && 
                (item.name === bundleName || item.fileName.includes(bundleName))) {
              injectedScripts.push(`${assetUrlBase}${item.fileName}`);
            }
          }
        }
      }

      // Inject style tags before </head>
      if (injectedStyles.length > 0) {
        const styleTags = injectedStyles.map(href => `<link rel="stylesheet" href="${href}">`).join('\n  ');
        htmlContent = htmlContent.replace('</head>', `  ${styleTags}\n</head>`);
      }

      // Inject script tags before </body>
      if (injectedScripts.length > 0) {
        const scriptTags = injectedScripts.map(src => `<script src="${src}"></script>`).join('\n  ');
        htmlContent = htmlContent.replace('</body>', `  ${scriptTags}\n</body>`);
      }

      // Subresource Integrity (SRI) and Cross-Origin settings
      const crossOrigin = options.crossOrigin || 'none';
      const useCrossOrigin = crossOrigin !== 'none' || options.subresourceIntegrity === true;
      const crossOriginVal = crossOrigin === 'none' ? 'anonymous' : crossOrigin;
      const useSri = options.subresourceIntegrity === true;

      if (useSri || useCrossOrigin) {
        // Parse and enrich all script/link tags
        htmlContent = htmlContent.replace(/<(script|link)\b([^>]*?)(src|href)="([^"]+)"([^>]*?)>/g, (tag, name, before, attr, url, after) => {
          if (tag.includes('integrity=')) return tag;

          let relUrl = url;
          if (url.startsWith(assetUrlBase)) {
            relUrl = url.substring(assetUrlBase.length);
          } else if (url.startsWith(baseHref)) {
            relUrl = url.substring(baseHref.length);
          } else if (url.startsWith('/')) {
            relUrl = url.substring(1);
          }
          relUrl = relUrl.split('?')[0];

          const filePath = path.resolve(absoluteOutDir, relUrl);
          if (fs.existsSync(filePath)) {
            let extra = '';
            if (useCrossOrigin && !tag.includes('crossorigin=')) {
              extra += ` crossorigin="${crossOriginVal}"`;
            }
            if (useSri) {
              try {
                const content = fs.readFileSync(filePath);
                const hash = crypto.createHash('sha384').update(content).digest('base64');
                extra += ` integrity="sha384-${hash}"`;
              } catch (e) {
                // ignore
              }
            }
            return `<${name}${before}${attr}="${url}"${extra}${after}>`;
          }
          return tag;
        });
      }

      if (preloadInitial) {
        const preloadTags = collectInitialChunkFiles(outputs)
          .filter(fileName => fileName.endsWith('.js') && !fileName.includes('index'))
          .map(fileName => `${assetUrlBase}${fileName}`)
          .filter(url => !htmlContent.includes(`href="${url}"`))
          .map(url => `<link rel="modulepreload" href="${url}">`);
        if (preloadTags.length > 0) {
          htmlContent = htmlContent.replace('</head>', `  ${preloadTags.join('\n  ')}\n</head>`);
        }
      }

      fs.writeFileSync(finalIndexHtmlPath, htmlContent, 'utf8');
      context.logger.info(`Injected ${injectedStyles.length} styles and ${injectedScripts.length} scripts into ${indexOutput}`);
      if (useSri) {
        context.logger.info(`Applied Subresource Integrity (SRI) hashes to script and style tags`);
      }
    }

    // Assets Copying
    if (options.assets && Array.isArray(options.assets)) {
      for (const asset of options.assets) {
        const copied = copyAssetPattern(workspaceRoot, absoluteOutDir, asset);
        if (copied > 0) {
          context.logger.info(`Copied ${copied} asset file(s) from ${typeof asset === 'string' ? asset : asset.input}`);
        }
      }
    }

    const commonJsWarnings = collectCommonJsWarnings(outputs, workspaceRoot, path.dirname(tsConfigPath), options.allowedCommonJsDependencies || []);
    for (const warning of commonJsWarnings) {
      context.logger.warn(warning);
    }

    // Stats JSON generation
    if (options.statsJson) {
      try {
        const stats = {
          builtAt: new Date().toISOString(),
          outputPath: absoluteOutDir,
          outputs: outputs.flatMap(out => 
            out.output.map((item: any) => ({
              fileName: item.fileName,
              type: item.type,
              size: item.type === 'chunk' ? (item.code?.length || 0) : (item.source?.length || 0)
            }))
          )
        };
        fs.writeFileSync(
          path.join(absoluteOutDir, 'stats.json'),
          JSON.stringify(stats, null, 2),
          'utf8'
        );
        context.logger.info(`Generated build stats at stats.json`);
      } catch (e: any) {
        context.logger.warn(`Failed to generate stats.json: ${e.message}`);
      }
    }

    // License extraction
    if (options.extractLicenses !== false) {
      try {
        const projectRoot = path.dirname(tsConfigPath);
        extractThirdPartyLicenses(workspaceRoot, projectRoot, absoluteOutDir);
        context.logger.info(`Extracted third-party licenses to 3rdpartylicenses.txt`);
      } catch (e: any) {
        context.logger.warn(`Failed to extract licenses: ${e.message}`);
      }
    }

    const chunksMap = new Map<string, any>();
    for (const out of outputs) {
      for (const item of out.output) {
        if (item.type === 'chunk') {
          chunksMap.set(item.fileName, item);
        }
      }
    }

    const initialFiles = new Set<string>();
    for (const out of outputs) {
      for (const item of out.output) {
        if (item.type === 'chunk' && item.isEntry) {
          initialFiles.add(item.fileName);
          const addImports = (imports: string[]) => {
            for (const imp of imports) {
              if (!initialFiles.has(imp)) {
                initialFiles.add(imp);
                const child = chunksMap.get(imp);
                if (child) {
                  addImports(child.imports);
                }
              }
            }
          };
          addImports(item.imports);
        }
      }
    }

    const initialCss = new Set<string>();
    for (const fileName of initialFiles) {
      const chunk = chunksMap.get(fileName);
      if (chunk && chunk.viteMetadata?.importedCss) {
        for (const cssFile of chunk.viteMetadata.importedCss) {
          initialCss.add(cssFile);
        }
      }
    }

    const flatOutputs = outputs.flatMap(out => out.output);
    const globalCssFiles = new Set<string>();
    for (const item of flatOutputs) {
      if (item.type !== 'asset' || !item.fileName.endsWith('.css')) continue;
      for (const styleName of globalStyleNames) {
        if (item.name === styleName || item.fileName.includes(styleName)) {
          globalCssFiles.add(item.fileName);
        }
      }
    }

    const outputSize = (item: any): number => {
      if (item.type === 'chunk') return Buffer.byteLength(item.code || '', 'utf8');
      const source = item.source || '';
      return typeof source === 'string' ? Buffer.byteLength(source, 'utf8') : source.byteLength;
    };
    const jsOutputs = flatOutputs.filter((item: any) => item.type === 'chunk');
    const allSizedOutputs = flatOutputs.filter((item: any) => item.type === 'chunk' || item.type === 'asset');
    const componentStyleOutputs = flatOutputs.filter((item: any) =>
      item.type === 'asset' && item.fileName.endsWith('.css') && !globalCssFiles.has(item.fileName)
    );
    const initialOutputNames = new Set([...initialFiles, ...initialCss]);
    const initialTotalSize = allSizedOutputs
      .filter((item: any) => initialOutputNames.has(item.fileName))
      .reduce((sum: number, item: any) => sum + outputSize(item), 0);

    const matchesBudgetName = (item: any, name: string | undefined): boolean => {
      if (!name) return false;
      const baseName = path.basename(item.fileName, path.extname(item.fileName));
      return item.name === name || baseName === name || item.fileName.includes(name);
    };

    const checkBudget = (budget: any, label: string, size: number): boolean => {
      let failed = false;
      const maxWarning = parseSize(budget.maximumWarning || budget.warning);
      const maxError = parseSize(budget.maximumError || budget.error);
      const minWarning = parseSize(budget.minimumWarning);
      const minError = parseSize(budget.minimumError);

      if (maxError !== null && size > maxError) {
        context.logger.error(`Error: ${label} size of ${formatBytes(size)} exceeded budget limit of ${budget.maximumError || budget.error}.`);
        failed = true;
      } else if (maxWarning !== null && size > maxWarning) {
        context.logger.warn(`Warning: ${label} size of ${formatBytes(size)} exceeded budget limit of ${budget.maximumWarning || budget.warning}.`);
      }

      if (minError !== null && size < minError) {
        context.logger.error(`Error: ${label} size of ${formatBytes(size)} is below budget minimum of ${budget.minimumError}.`);
        failed = true;
      } else if (minWarning !== null && size < minWarning) {
        context.logger.warn(`Warning: ${label} size of ${formatBytes(size)} is below budget minimum of ${budget.minimumWarning}.`);
      }
      return failed;
    };

    let hasBudgetError = false;
    const budgets = options.budgets || [];
    for (const budget of budgets) {
      if (budget.type === 'initial') {
        hasBudgetError = checkBudget(budget, 'Initial bundle', initialTotalSize) || hasBudgetError;
      }

      if (budget.type === 'bundle') {
        const bundleOutputs = allSizedOutputs.filter((item: any) => matchesBudgetName(item, budget.name));
        const bundleSize = bundleOutputs.reduce((sum: number, item: any) => sum + outputSize(item), 0);
        hasBudgetError = checkBudget(budget, `Bundle ${budget.name || '(unnamed)'}`, bundleSize) || hasBudgetError;
      }

      if (budget.type === 'all') {
        const allSize = allSizedOutputs.reduce((sum: number, item: any) => sum + outputSize(item), 0);
        hasBudgetError = checkBudget(budget, 'All bundles', allSize) || hasBudgetError;
      }

      if (budget.type === 'any') {
        for (const item of allSizedOutputs) {
          hasBudgetError = checkBudget(budget, item.fileName, outputSize(item)) || hasBudgetError;
        }
      }

      if (budget.type === 'allScript') {
        const scriptSize = jsOutputs.reduce((sum: number, item: any) => sum + outputSize(item), 0);
        hasBudgetError = checkBudget(budget, 'All scripts', scriptSize) || hasBudgetError;
      }

      if (budget.type === 'anyScript') {
        for (const item of jsOutputs) {
          hasBudgetError = checkBudget(budget, item.fileName, outputSize(item)) || hasBudgetError;
        }
      }

      if (budget.type === 'anyComponentStyle') {
        for (const item of componentStyleOutputs) {
          hasBudgetError = checkBudget(budget, `Component style ${item.fileName}`, outputSize(item)) || hasBudgetError;
        }
      }
    }

    if (hasBudgetError) {
      return { success: false, error: 'Budgets exceeded.' };
    }

    context.logger.info(`Build completed successfully.`);
    return { success: true };
  } catch (error: any) {
    context.logger.error(`Build failed: ${error.message || error}`);
    return { success: false, error: error.message };
  }
});

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

export function normalizeStringList(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((entry): entry is string => typeof entry === 'string') : [];
}

export function normalizeGlobalEntries(value: unknown, fallbackPrefix: string): GlobalEntry[] {
  if (!Array.isArray(value)) return [];
  
  const getDefaultBundleName = (input: string, fallback: string): string => {
    try {
      const filename = path.basename(input);
      const ext = path.extname(filename);
      return filename.substring(0, filename.length - ext.length);
    } catch {
      return fallback;
    }
  };

  return value.flatMap((entry, index): GlobalEntry[] => {
    const fallback = `${fallbackPrefix}-${index}`;
    if (typeof entry === 'string') {
      return [{ input: entry, bundleName: getDefaultBundleName(entry, fallback), inject: true }];
    }
    if (entry && typeof entry === 'object' && typeof (entry as any).input === 'string') {
      return [{
        input: (entry as any).input,
        bundleName: typeof (entry as any).bundleName === 'string' && (entry as any).bundleName.length > 0
          ? (entry as any).bundleName
          : getDefaultBundleName((entry as any).input, fallback),
        inject: (entry as any).inject !== false
      }];
    }
    return [];
  });
}

function createPolyfillMainPlugin(browserEntry: string, polyfills: string[], workspaceRoot: string, tsConfigPath: string) {
  const normalizedBrowserEntry = path.resolve(browserEntry);
  return {
    name: 'angular-go-polyfill-main',
    transform(code: string, id: string) {
      if (polyfills.length === 0 || path.resolve(id.split('?', 1)[0]) !== normalizedBrowserEntry) {
        return null;
      }
      const imports = polyfills.map((polyfill) => {
        let resolved = polyfill;
        if (polyfill.startsWith('.') || polyfill.startsWith('/') || polyfill.startsWith('\\') || fs.existsSync(path.resolve(workspaceRoot, polyfill))) {
          resolved = path.resolve(workspaceRoot, polyfill);
        } else {
          try {
            resolved = require.resolve(polyfill, {
              paths: [path.dirname(tsConfigPath), workspaceRoot, __dirname]
            });
          } catch {
            resolved = polyfill;
          }
        }
        return `import ${JSON.stringify(toViteImportPath(resolved))};`;
      });
      return {
        code: `${imports.join('\n')}\n${code}`,
        map: null
      };
    }
  };
}

function toViteImportPath(filePath: string): string {
  return path.isAbsolute(filePath) ? filePath.split(path.sep).join('/') : filePath;
}

function normalizeUrlBase(value: string): string {
  if (!value) return '/';
  return value.endsWith('/') ? value : `${value}/`;
}

function removeGlobalStyleJsChunksPlugin(globalStyleNames: Set<string>) {
  return {
    name: 'angular-go-remove-global-style-js-chunks',
    generateBundle(_options: unknown, bundle: Record<string, any>) {
      for (const [fileName, item] of Object.entries(bundle)) {
        if (item.type !== 'chunk') continue;
        if (globalStyleNames.has(item.name)) {
          delete bundle[fileName];
        }
      }
    }
  };
}

export function copyAssetPattern(workspaceRoot: string, absoluteOutDir: string, asset: any): number {
  if (typeof asset === 'string') {
    const srcPath = path.resolve(workspaceRoot, asset);
    if (!fs.existsSync(srcPath)) return 0;
    const destPath = safeJoin(absoluteOutDir, path.basename(asset));
    copyPath(srcPath, destPath, true);
    return countFiles(srcPath);
  }

  if (!asset || typeof asset !== 'object') return 0;
  const inputPath = path.resolve(workspaceRoot, asset.input || '');
  if (!fs.existsSync(inputPath)) return 0;

  const output = typeof asset.output === 'string' ? asset.output.replace(/^\/+/, '') : '';
  const glob = typeof asset.glob === 'string' ? asset.glob : '**/*';
  const ignore = Array.isArray(asset.ignore) ? asset.ignore.filter((entry: unknown): entry is string => typeof entry === 'string') : [];
  const followSymlinks = asset.followSymlinks === true;
  let copied = 0;

  for (const filePath of listFiles(inputPath, followSymlinks)) {
    const rel = path.relative(inputPath, filePath).split(path.sep).join('/');
    if (!matchesGlob(rel, glob)) continue;
    if (ignore.some((pattern: string) => matchesGlob(rel, pattern))) continue;
    const destPath = safeJoin(absoluteOutDir, path.posix.join(output, rel));
    fs.mkdirSync(path.dirname(destPath), { recursive: true });
    fs.copyFileSync(filePath, destPath);
    copied++;
  }

  return copied;
}

function listFiles(root: string, followSymlinks: boolean): string[] {
  const files: string[] = [];
  const visit = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const filePath = path.join(dir, entry.name);
      if (entry.isSymbolicLink() && !followSymlinks) continue;
      const stat = followSymlinks ? fs.statSync(filePath) : null;
      if (entry.isDirectory() || stat?.isDirectory()) {
        visit(filePath);
      } else if (entry.isFile() || stat?.isFile()) {
        files.push(filePath);
      }
    }
  };
  visit(root);
  return files;
}

function copyPath(srcPath: string, destPath: string, followSymlinks: boolean): void {
  const stat = fs.statSync(srcPath);
  if (stat.isDirectory()) {
    for (const filePath of listFiles(srcPath, followSymlinks)) {
      const rel = path.relative(srcPath, filePath);
      const target = safeJoin(destPath, rel);
      fs.mkdirSync(path.dirname(target), { recursive: true });
      fs.copyFileSync(filePath, target);
    }
    return;
  }
  fs.mkdirSync(path.dirname(destPath), { recursive: true });
  fs.copyFileSync(srcPath, destPath);
}

function countFiles(srcPath: string): number {
  if (!fs.existsSync(srcPath)) return 0;
  const stat = fs.statSync(srcPath);
  return stat.isDirectory() ? listFiles(srcPath, true).length : 1;
}

function safeJoin(root: string, relPath: string): string {
  const normalizedRoot = path.resolve(root);
  const target = path.resolve(normalizedRoot, relPath);
  if (target !== normalizedRoot && !target.startsWith(`${normalizedRoot}${path.sep}`)) {
    throw new Error(`Asset output path escapes output directory: ${relPath}`);
  }
  return target;
}

export function matchesGlob(rel: string, glob: string): boolean {
  const normalized = rel.split(path.sep).join('/');
  if (glob === '**/*' || glob === '**') return true;
  if (glob.startsWith('**/*.')) return normalized.endsWith(glob.slice(4));
  if (glob.startsWith('**/')) return normalized.endsWith(glob.slice(3));
  if (!glob.includes('*')) return normalized === glob;
  const escaped = glob
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')
    .replace(/\*\*/g, '.*')
    .replace(/\*/g, '[^/]*');
  return new RegExp(`^${escaped}$`).test(normalized);
}

function warnUnsupportedOptions(options: any, context: BuilderContext, names: string[]): void {
  for (const name of names) {
    if (isMeaningfulUnsupportedValue(name, options[name])) {
      context.logger.warn(`angular-go-build currently ignores unsupported application option '${name}'.`);
    }
  }
}

function isMeaningfulUnsupportedValue(name: string, value: unknown): boolean {
  if (value === undefined || value === false || value === null) return false;
  if (Array.isArray(value)) return value.length > 0;
  if (name === 'inlineStyleLanguage' && value === 'css') return false;
  if ((name === 'i18nMissingTranslation' || name === 'i18nDuplicateTranslation') && value === 'warning') return false;
  if (name === 'security' && typeof value === 'object') {
    const security = value as any;
    return security.autoCsp !== undefined && security.autoCsp !== false;
  }
  return true;
}

function getUnsupportedFeatureError(options: any): string | null {
  const failIfEnabled = [
    'localize',
    'serviceWorker',
    'server',
    'ssr',
    'prerender',
    'appShell',
    'outputMode'
  ];
  for (const name of failIfEnabled) {
    if (isMeaningfulUnsupportedValue(name, options[name])) {
      return `angular-go-build does not yet support application option '${name}'. Disable it or use @angular/build for this target.`;
    }
  }
  return null;
}

export function collectInitialChunkFiles(outputs: any[]): string[] {
  const bundle = new Map<string, any>();
  for (const out of outputs) {
    for (const item of out.output || []) {
      if (item.type === 'chunk') {
        bundle.set(item.fileName, item);
      }
    }
  }

  const initial = new Set<string>();
  const addChunk = (fileName: string) => {
    if (initial.has(fileName)) return;
    initial.add(fileName);
    const chunk = bundle.get(fileName);
    for (const imported of chunk?.imports || []) {
      addChunk(imported);
    }
  };

  for (const [fileName, item] of bundle) {
    if (item.isEntry) addChunk(fileName);
  }
  return Array.from(initial);
}

export function collectCommonJsWarnings(outputs: any[], workspaceRoot: string, projectRoot: string, allowed: string[]): string[] {
  if (allowed.includes('*')) return [];
  const allowedSet = new Set(allowed);
  const packages = new Set<string>();
  for (const out of outputs) {
    for (const item of out.output || []) {
      if (item.type !== 'chunk') continue;
      for (const moduleId of item.moduleIds || []) {
        const packageName = packageNameFromNodeModule(moduleId);
        if (!packageName || allowedSet.has(packageName)) continue;
        if (isLikelyCommonJsPackage(packageName, workspaceRoot, projectRoot)) {
          packages.add(packageName);
        }
      }
    }
  }
  return Array.from(packages).sort().map(pkg =>
    `CommonJS dependency '${pkg}' may cause optimization bailouts. Add it to allowedCommonJsDependencies to silence this warning.`
  );
}

function packageNameFromNodeModule(moduleId: string): string | null {
  const marker = `${path.sep}node_modules${path.sep}`;
  const index = moduleId.lastIndexOf(marker);
  if (index < 0) return null;
  const rest = moduleId.slice(index + marker.length).split(path.sep);
  if (rest[0]?.startsWith('@') && rest[1]) return `${rest[0]}/${rest[1]}`;
  return rest[0] || null;
}

function isLikelyCommonJsPackage(packageName: string, workspaceRoot: string, projectRoot: string): boolean {
  const candidates = [
    path.resolve(projectRoot, 'node_modules', packageName, 'package.json'),
    path.resolve(workspaceRoot, 'node_modules', packageName, 'package.json')
  ];
  for (const pkgPath of candidates) {
    if (!fs.existsSync(pkgPath)) continue;
    try {
      const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
      if (pkg.type === 'module') return false;
      if (pkg.module || pkg.es2015 || pkg.fesm2022 || pkg.exports) return false;
      return Boolean(pkg.main);
    } catch {
      return false;
    }
  }
  return false;
}

function extractThirdPartyLicenses(workspaceRoot: string, projectRoot: string, absoluteOutDir: string) {
  const visited = new Set<string>();
  let licensesText = '3RD PARTY LICENSES\n\n';

  function crawl(pkgJsonPath: string) {
    if (!fs.existsSync(pkgJsonPath)) return;
    try {
      const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, 'utf8'));
      const deps = { ...(pkg.dependencies || {}), ...(pkg.peerDependencies || {}) };

      for (const dep of Object.keys(deps)) {
        if (visited.has(dep)) continue;
        visited.add(dep);

        let depDir = '';
        const candidatePaths = [
          path.resolve(projectRoot, 'node_modules', dep),
          path.resolve(workspaceRoot, 'node_modules', dep)
        ];
        for (const cand of candidatePaths) {
          if (fs.existsSync(cand)) {
            depDir = cand;
            break;
          }
        }

        if (!depDir) continue;

        const depPkgJsonPath = path.join(depDir, 'package.json');
        if (fs.existsSync(depPkgJsonPath)) {
          const depPkg = JSON.parse(fs.readFileSync(depPkgJsonPath, 'utf8'));
          licensesText += `========================================================================\n`;
          licensesText += `Package: ${depPkg.name}\n`;
          licensesText += `Version: ${depPkg.version}\n`;
          licensesText += `License: ${depPkg.license || 'Unknown'}\n`;
          if (depPkg.author) {
            licensesText += `Author: ${typeof depPkg.author === 'string' ? depPkg.author : depPkg.author.name || ''}\n`;
          }
          if (depPkg.description) {
            licensesText += `Description: ${depPkg.description}\n`;
          }

          const files = fs.readdirSync(depDir);
          const licenseFile = files.find(f => f.match(/^licen[sc]e(?:\.txt|\.md)?$/i));
          if (licenseFile) {
            const licenseText = fs.readFileSync(path.join(depDir, licenseFile), 'utf8');
            licensesText += `\nLicense Text:\n${licenseText}\n`;
          }
          licensesText += `========================================================================\n\n`;

          crawl(depPkgJsonPath);
        }
      }
    } catch (e) {
      // Ignore errors
    }
  }

  const projectPkgJson = path.resolve(projectRoot, 'package.json');
  crawl(projectPkgJson);

  fs.writeFileSync(path.join(absoluteOutDir, '3rdpartylicenses.txt'), licensesText, 'utf8');
}
