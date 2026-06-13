import path from 'node:path';
import { rolldown } from 'rolldown';
import type { OutputBundle } from 'rolldown';
import type { CompilerOutputFile } from './bundle-stats.js';

export interface AppBundleEntry {
  name: string;
  fileName: string;
}

export async function bundleCompilerOutputsWithRolldown(
  outputs: CompilerOutputFile[],
  entries: AppBundleEntry[]
): Promise<Record<string, OutputBundle[string]>> {
  const jsOutputs = new Map<string, CompilerOutputFile>();
  for (const output of outputs) {
    if (output.kind === 'js' && output.text !== undefined) {
      const outputPath = path.isAbsolute(output.path) ? output.path : path.resolve(output.path);
      jsOutputs.set(normalizePath(outputPath), {
        ...output,
        path: outputPath,
      });
    }
  }

  const input: Record<string, string> = {};
  for (const entry of entries) {
    const outputPath = findOutputByBaseName(jsOutputs, entry.fileName);
    if (outputPath) {
      input[entry.name] = outputPath;
    }
  }

  if (Object.keys(input).length === 0) {
    return {};
  }

  const bundle = await rolldown({
    input,
    external: (id: string) => isBareImport(id) && !resolveBareOutputImport(id, jsOutputs),
    treeshake: false,
    plugins: [
      {
        name: 'angular-go-memory-outputs',
        resolveId(source: string, importer: string | undefined) {
          if (isBareImport(source)) {
            const outputPath = resolveBareOutputImport(source, jsOutputs);
            if (outputPath) {
              return outputPath;
            }
            return { id: source, external: true };
          }

          if (path.isAbsolute(source)) {
            const normalized = normalizeImportToOutputPath(source);
            if (jsOutputs.has(normalized)) {
              return normalized;
            }
            // Fallback: search by suffix since output paths may include out-tsc/app
            for (const outPath of jsOutputs.keys()) {
              if (outPath.endsWith(normalized) || normalized.endsWith(outPath)) {
                return outPath;
              }
              // Sometimes normalized is /Users/.../apps/demo/src/app/file.js
              // and outPath is /Users/.../out-tsc/app/apps/demo/src/app/file.js
              // Let's strip the workspace root from normalized and check if outPath ends with it
              const basename = path.basename(normalized);
              if (path.basename(outPath) === basename) {
                // simple heuristic if basenames match
                if (outPath.replace('/out-tsc/app', '') === normalized) {
                   return outPath;
                }
              }
            }
            console.log('[DEBUG] resolveId failed for absolute source:', source);
            return null;
          }

          if (source.startsWith('.') && importer) {
            return resolveRelativeOutputImport(importer, source, jsOutputs) || null;
          }

          return null;
        },
        load(id: string) {
          return jsOutputs.get(normalizePath(id))?.text ?? null;
        },
      },
    ],
  });

  try {
    const result = await bundle.generate({
      format: 'esm',
      sourcemap: false,
      entryFileNames: '[name].js',
      chunkFileNames: '[name].js',
      assetFileNames: '[name].[ext]',
    });

    const record: Record<string, OutputBundle[string]> = {};
    for (const item of result.output) {
      record[item.fileName] = item;
    }
    return record;
  } finally {
    await bundle.close();
  }
}

function normalizePath(filePath: string): string {
  return filePath.replace(/\\/g, '/');
}

function normalizeImportToOutputPath(specifier: string): string {
  const normalized = normalizePath(specifier);
  if (/\.[cm]?ts$/.test(normalized)) {
    return normalized.replace(/\.[cm]?ts$/, '.js');
  }
  if (normalized.endsWith('.mjs') || normalized.endsWith('.js')) {
    return normalized;
  }
  return `${normalized}.js`;
}

function isBareImport(id: string): boolean {
  return !id.startsWith('.') && !id.startsWith('/') && !path.isAbsolute(id);
}

function findOutputByBaseName(outputs: Map<string, CompilerOutputFile>, baseName: string): string | undefined {
  for (const outputPath of outputs.keys()) {
    if (path.basename(outputPath) === baseName) {
      return outputPath;
    }
  }
  return undefined;
}

function resolveRelativeOutputImport(
  importerPath: string,
  specifier: string,
  outputs: Map<string, CompilerOutputFile>
): string | undefined {
  const importerDir = path.dirname(normalizePath(importerPath));
  const candidate = normalizePath(path.resolve(importerDir, specifier));
  const candidates = [
    candidate,
    `${candidate}.js`,
    normalizePath(path.join(candidate, 'index.js')),
  ];

  return candidates.find(value => outputs.has(value));
}

function resolveBareOutputImport(
  specifier: string,
  outputs: Map<string, CompilerOutputFile>
): string | undefined {
  const normalized = normalizeImportToOutputPath(specifier);
  const suffix = `/${normalized}`;
  const matches: string[] = [];

  for (const outputPath of outputs.keys()) {
    if (outputPath.endsWith(suffix) || outputPath.endsWith(`/${path.basename(normalized)}`)) {
      matches.push(outputPath);
    }
  }

  return matches.length === 1 ? matches[0] : undefined;
}
