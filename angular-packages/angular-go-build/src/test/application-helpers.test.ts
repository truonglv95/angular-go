import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import {
  collectCommonJsWarnings,
  collectInitialChunkFiles,
  copyAssetPattern,
  matchesGlob,
  normalizeGlobalEntries,
} from '../builders/application/index.js';

test('normalizes string and object global entries', () => {
  assert.deepEqual(normalizeGlobalEntries(['src/styles.css'], 'style'), [
    { input: 'src/styles.css', bundleName: 'style-0', inject: true },
  ]);

  assert.deepEqual(
    normalizeGlobalEntries([{ input: 'src/admin.css', bundleName: 'admin', inject: false }], 'style'),
    [{ input: 'src/admin.css', bundleName: 'admin', inject: false }],
  );
});

test('matches common Angular asset glob patterns', () => {
  assert.equal(matchesGlob('icons/logo.svg', '**/*'), true);
  assert.equal(matchesGlob('icons/logo.svg', '**/*.svg'), true);
  assert.equal(matchesGlob('icons/logo.svg', '**/*.png'), false);
  assert.equal(matchesGlob('favicon.ico', 'favicon.ico'), true);
});

test('copies object asset patterns with ignore and safe output', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'angular-go-build-'));
  const input = path.join(root, 'public');
  const out = path.join(root, 'dist');
  fs.mkdirSync(path.join(input, 'icons'), { recursive: true });
  fs.writeFileSync(path.join(input, 'icons', 'logo.svg'), '<svg/>');
  fs.writeFileSync(path.join(input, 'icons', 'debug.txt'), 'debug');

  const copied = copyAssetPattern(root, out, {
    glob: '**/*.svg',
    input: 'public',
    output: 'assets',
    ignore: ['**/debug.txt'],
  });

  assert.equal(copied, 1);
  assert.equal(fs.existsSync(path.join(out, 'assets', 'icons', 'logo.svg')), true);
  assert.equal(fs.existsSync(path.join(out, 'assets', 'icons', 'debug.txt')), false);
});

test('collects entry chunks and static imports', () => {
  const files = collectInitialChunkFiles([
    {
      output: [
        { type: 'chunk', fileName: 'assets/main.js', isEntry: true, imports: ['assets/vendor.js'] },
        { type: 'chunk', fileName: 'assets/vendor.js', imports: [] },
        { type: 'chunk', fileName: 'assets/lazy.js', imports: [] },
      ],
    },
  ]);

  assert.deepEqual(files.sort(), ['assets/main.js', 'assets/vendor.js']);
});

test('warns for likely CommonJS packages unless allowlisted', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'angular-go-cjs-'));
  const pkgDir = path.join(root, 'node_modules', 'legacy-lib');
  fs.mkdirSync(pkgDir, { recursive: true });
  fs.writeFileSync(path.join(pkgDir, 'package.json'), JSON.stringify({ name: 'legacy-lib', main: 'index.js' }));

  const outputs = [{
    output: [{
      type: 'chunk',
      moduleIds: [path.join(pkgDir, 'index.js')],
    }],
  }];

  assert.equal(collectCommonJsWarnings(outputs, root, root, []).length, 1);
  assert.equal(collectCommonJsWarnings(outputs, root, root, ['legacy-lib']).length, 0);
});
