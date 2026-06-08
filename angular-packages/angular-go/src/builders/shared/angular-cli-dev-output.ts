import { stripVTControlCharacters } from 'node:util';
import { BundleSizeRow, BundleSizeSummary, formatBytes } from './bundle-stats.js';

export interface DevBundleOutputOptions {
  durationMs: number;
  timestamp?: Date;
}

type TableCell = string;

const ANSI = {
  bold: (value: string) => `\x1b[1m${value}\x1b[22m`,
  cyan: (value: string) => `\x1b[36m${value}\x1b[39m`,
  green: (value: string) => `\x1b[32m${value}\x1b[39m`,
  dim: (value: string) => `\x1b[2m${value}\x1b[22m`,
};

export function formatAngularDevBundleOutput(
  summary: BundleSizeSummary,
  options: DevBundleOutputOptions
): string {
  const initialRows = [...summary.initial].sort(sortRowsByRawSizeDesc);
  const lazyRows = [...summary.lazy].sort(sortRowsByRawSizeDesc);
  const tableRows: TableCell[][] = [];

  if (initialRows.length > 0) {
    tableRows.push(
      ['Initial chunk files', 'Names', 'Raw size'].map(ANSI.bold),
      ...initialRows.map(formatStatsRow)
    );

    tableRows.push([], [
      '',
      ANSI.bold('Initial total'),
      ANSI.bold(formatBytes(summary.initialTotalRawSize)),
      '',
    ]);
  }

  if (initialRows.length > 0 && lazyRows.length > 0) {
    tableRows.push([]);
  }

  if (lazyRows.length > 0) {
    tableRows.push(
      ['Lazy chunk files', 'Names', 'Raw size'].map(ANSI.bold),
      ...lazyRows.map(formatStatsRow)
    );
  }

  const durationSec = (options.durationMs / 1000).toFixed(3);
  const ts = options.timestamp || new Date();
  return `\n${generateTableText(tableRows)}\n\n` +
    `Application bundle generation complete. [${durationSec} seconds] - ${ts.toISOString()}\n`;
}

export function formatAngularDevServerReadyOutput(url: string): string {
  const outputLines = [
    `Watch mode enabled. Watching for file changes...`,
    `NOTE: Raw file sizes do not reflect development server per-request transformations.`,
    `  ➜  Local:   ${url}`,
    `  ➜  press h + enter to show help`
  ];
  return outputLines.join('\n');
}

function sortRowsByRawSizeDesc(a: BundleSizeRow, b: BundleSizeRow): number {
  return b.rawSize - a.rawSize;
}

function formatStatsRow(row: BundleSizeRow): TableCell[] {
  return [
    ANSI.green(row.file),
    ANSI.dim(row.name || '-'),
    ANSI.cyan(formatBytes(row.rawSize)),
    '',
  ];
}

function generateTableText(rows: TableCell[][]): string {
  const longest: number[] = [];

  for (const row of rows) {
    for (let i = 0; i < row.length; i++) {
      const current = row[i];
      if (current === undefined) continue;
      const length = stripVTControlCharacters(current).length;
      longest[i] = Math.max(longest[i] ?? 0, length);
    }
  }

  const output: string[] = [];
  for (const row of rows) {
    const padded = row.map((cell, index) => {
      const length = stripVTControlCharacters(cell).length;
      const padding = ' '.repeat((longest[index] ?? 0) - length);
      return index >= 2 ? padding + cell : cell + padding;
    });
    output.push(padded.join('\x1b[2m | \x1b[22m'));
  }

  return output.join('\n');
}
