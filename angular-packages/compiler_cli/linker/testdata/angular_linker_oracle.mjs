#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';

const requireFromCwd = createRequire(path.join(process.cwd(), 'package.json'));

const [inputPath, outputPath] = process.argv.slice(2);
if (!inputPath || !outputPath) {
  console.error('usage: node angular_linker_oracle.mjs <input.mjs> <output.js>');
  process.exit(2);
}

const babel = requireFromCwd('@babel/core');
const linkerBabelPath = requireFromCwd.resolve('@angular/compiler-cli/linker/babel');
const { createEs2015LinkerPlugin } = await import(pathToFileURL(linkerBabelPath));

const absoluteInputPath = path.resolve(inputPath);
const code = fs.readFileSync(absoluteInputPath, 'utf8');

const plugin = createEs2015LinkerPlugin({
  fileSystem: {
    resolve: (filePath) => filePath,
    exists: (filePath) => fs.existsSync(filePath),
    dirname: (filePath) => path.dirname(filePath),
    relative: (from, to) => path.relative(from, to),
    readFile: (filePath) => fs.readFileSync(filePath, 'utf8'),
  },
  logger: {
    level: 0,
    debug() {},
    info() {},
    warn() {},
    error() {},
  },
  options: {
    linkerJitMode: false,
  },
});

const result = babel.transformSync(code, {
  filename: absoluteInputPath,
  plugins: [plugin],
  compact: false,
  retainLines: false,
});

fs.mkdirSync(path.dirname(path.resolve(outputPath)), { recursive: true });
fs.writeFileSync(outputPath, result.code);
