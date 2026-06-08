import path from 'node:path';

export interface BundleSizeRow {
  file: string;
  name: string;
  rawSize: number;
  kind: 'js' | 'css';
  initial: boolean;
}

export interface BundleSizeSummary {
  initial: BundleSizeRow[];
  lazy: BundleSizeRow[];
  initialTotalRawSize: number;
}

export interface CompilerOutputFile {
  path: string;
  text?: string;
  kind?: 'js' | 'map' | 'dts' | 'other';
}

export interface CompilerOutputSummaryOptions {
  mainFileName?: string;
  polyfillFileNames?: string[];
}

export function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 bytes';
  if (bytes === 1) return '1 byte';
  const k = 1000;
  const sizes = ['bytes', 'kB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const fractionDigits = i === 0 ? 0 : 2;
  return `${(bytes / Math.pow(k, i)).toFixed(fractionDigits)} ${sizes[i]}`;
}

export function collectBundleSizeSummary(bundle: Record<string, any>): BundleSizeSummary {
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

  const initialRows: BundleSizeRow[] = [];
  const lazyRows: BundleSizeRow[] = [];

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
      initialRows.push({
        file: fileName.replace(/^assets\//, ''),
        name: item.name || 'main',
        rawSize: Buffer.byteLength(item.code || '', 'utf8'),
        kind: 'js',
        initial: true
      });
    }
  }

  // CSS initial
  for (const fileName of initialCss) {
    const item = bundle[fileName] as any;
    if (item && item.type === 'asset') {
      const source = item.source;
      const rawSize = typeof source === 'string' ? Buffer.byteLength(source, 'utf8') : source.byteLength;
      initialRows.push({
        file: fileName.replace(/^assets\//, ''),
        name: path.basename(fileName, '.css'),
        rawSize,
        kind: 'css',
        initial: true
      });
    }
  }

  // JS lazy
  for (const fileName of lazyChunks) {
    const item = bundle[fileName] as any;
    if (item && item.type === 'chunk') {
      lazyRows.push({
        file: fileName.replace(/^assets\//, ''),
        name: item.name || 'lazy',
        rawSize: Buffer.byteLength(item.code || '', 'utf8'),
        kind: 'js',
        initial: false
      });
    }
  }

  // CSS lazy
  for (const fileName of lazyCss) {
    const item = bundle[fileName] as any;
    if (item && item.type === 'asset') {
      const source = item.source;
      const rawSize = typeof source === 'string' ? Buffer.byteLength(source, 'utf8') : source.byteLength;
      lazyRows.push({
        file: fileName.replace(/^assets\//, ''),
        name: path.basename(fileName, '.css'),
        rawSize,
        kind: 'css',
        initial: false
      });
    }
  }

  let initialTotalRawSize = 0;
  for (const row of initialRows) {
    initialTotalRawSize += row.rawSize;
  }

  return {
    initial: initialRows,
    lazy: lazyRows,
    initialTotalRawSize
  };
}

export function collectCompilerOutputSizeSummary(
  outputs: CompilerOutputFile[],
  options: CompilerOutputSummaryOptions = {}
): BundleSizeSummary {
  const jsOutputs = new Map<string, CompilerOutputFile>();
  for (const output of outputs) {
    if (output.kind !== 'js' || output.text === undefined) {
      continue;
    }
    jsOutputs.set(normalizePath(output.path), output);
  }

  const mainFileName = options.mainFileName || 'main.js';
  const mainOutputPath = findOutputByBaseName(jsOutputs, mainFileName);
  const polyfillFileNames = new Set(options.polyfillFileNames || ['polyfills.js']);
  const polyfillOutputPaths = new Set<string>();
  for (const fileName of polyfillFileNames) {
    const outputPath = findOutputByBaseName(jsOutputs, fileName);
    if (outputPath) {
      polyfillOutputPaths.add(outputPath);
    }
  }

  const initialRows: BundleSizeRow[] = [];
  const lazyRows: BundleSizeRow[] = [];

  if (mainOutputPath) {
    const reachable = collectStaticReachableOutputs(mainOutputPath, jsOutputs);
    let rawSize = 0;
    for (const outputPath of reachable) {
      if (polyfillOutputPaths.has(outputPath) || outputPath.endsWith('/test-setup.js')) {
        continue;
      }
      rawSize += Buffer.byteLength(jsOutputs.get(outputPath)?.text || '', 'utf8');
    }
    initialRows.push({
      file: 'main.js',
      name: 'main',
      rawSize,
      kind: 'js',
      initial: true,
    });
  }

  for (const outputPath of polyfillOutputPaths) {
    initialRows.push({
      file: path.basename(outputPath),
      name: path.basename(outputPath, '.js'),
      rawSize: Buffer.byteLength(jsOutputs.get(outputPath)?.text || '', 'utf8'),
      kind: 'js',
      initial: true,
    });
  }

  const initialTotalRawSize = initialRows.reduce((total, row) => total + row.rawSize, 0);
  return { initial: initialRows, lazy: lazyRows, initialTotalRawSize };
}

function normalizePath(filePath: string): string {
  return filePath.replace(/\\/g, '/');
}

function findOutputByBaseName(outputs: Map<string, CompilerOutputFile>, baseName: string): string | undefined {
  for (const outputPath of outputs.keys()) {
    if (path.basename(outputPath) === baseName) {
      return outputPath;
    }
  }
  return undefined;
}

function collectStaticReachableOutputs(
  entryPath: string,
  outputs: Map<string, CompilerOutputFile>
): Set<string> {
  const reachable = new Set<string>();
  const pending = [entryPath];

  while (pending.length > 0) {
    const currentPath = pending.pop();
    if (!currentPath || reachable.has(currentPath)) {
      continue;
    }
    reachable.add(currentPath);

    const current = outputs.get(currentPath);
    if (!current?.text) {
      continue;
    }

    for (const specifier of staticImportSpecifiers(current.text)) {
      const resolved = resolveRelativeOutputImport(currentPath, specifier, outputs);
      if (resolved && !reachable.has(resolved)) {
        pending.push(resolved);
      }
    }
  }

  return reachable;
}

function staticImportSpecifiers(code: string): string[] {
  const specifiers: string[] = [];
  const patterns = [
    /\bimport\s+(?:[^'"]*?\s+from\s+)?["']([^"']+)["']/g,
    /\bexport\s+[^'"]*?\s+from\s+["']([^"']+)["']/g,
  ];

  for (const pattern of patterns) {
    for (const match of code.matchAll(pattern)) {
      specifiers.push(match[1]);
    }
  }

  return specifiers;
}

function resolveRelativeOutputImport(
  importerPath: string,
  specifier: string,
  outputs: Map<string, CompilerOutputFile>
): string | undefined {
  if (!specifier.startsWith('.')) {
    return undefined;
  }

  const importerDir = path.dirname(importerPath);
  const candidate = normalizePath(path.resolve(importerDir, specifier));
  const candidates = [
    candidate,
    `${candidate}.js`,
    normalizePath(path.join(candidate, 'index.js')),
  ];

  return candidates.find(value => outputs.has(value));
}
