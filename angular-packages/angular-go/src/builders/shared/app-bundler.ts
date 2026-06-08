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
      jsOutputs.set(normalizePath(output.path), output);
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
    external: (id: string) => isBareImport(id),
    treeshake: false,
    plugins: [
      {
        name: 'angular-go-memory-outputs',
        resolveId(source: string, importer: string | undefined) {
          if (isBareImport(source)) {
            return { id: source, external: true };
          }

          if (path.isAbsolute(source)) {
            const normalized = normalizePath(source);
            return jsOutputs.has(normalized) ? normalized : null;
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
