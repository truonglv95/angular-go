import { describe, it, expect } from 'vitest';
import { formatBytes, collectBundleSizeSummary, collectCompilerOutputSizeSummary } from './bundle-stats.js';

describe('bundle-stats', () => {
  describe('formatBytes', () => {
    it('should format 0 bytes correctly', () => {
      expect(formatBytes(0)).toBe('0 bytes');
    });

    it('should format 1 byte correctly', () => {
      expect(formatBytes(1)).toBe('1 byte');
    });

    it('should format bytes under 1000 correctly', () => {
      expect(formatBytes(91)).toBe('91 bytes');
      expect(formatBytes(512)).toBe('512 bytes');
    });

    it('should format kilobytes correctly', () => {
      expect(formatBytes(1024)).toBe('1.02 kB');
      expect(formatBytes(74096)).toBe('74.10 kB');
    });

    it('should format megabytes correctly', () => {
      expect(formatBytes(4_210_000)).toBe('4.21 MB');
    });
  });

  describe('collectBundleSizeSummary', () => {
    it('should classify entry chunks as initial and other chunks as lazy', () => {
      const mockBundle: Record<string, any> = {
        'main.js': {
          type: 'chunk',
          isEntry: true,
          code: 'console.log("hello");',
          name: 'main',
          imports: ['shared.js']
        },
        'shared.js': {
          type: 'chunk',
          isEntry: false,
          code: 'export const x = 123;',
          name: 'shared',
          imports: []
        },
        'lazy-chunk.js': {
          type: 'chunk',
          isEntry: false,
          code: 'console.log("lazy");',
          name: 'lazy-chunk',
          imports: []
        },
        'styles.css': {
          type: 'asset',
          source: 'body { color: red; }',
          fileName: 'styles.css'
        }
      };

      // Styles imported by initial chunks should be initial
      mockBundle['main.js'].viteMetadata = {
        importedCss: new Set(['styles.css'])
      };

      const summary = collectBundleSizeSummary(mockBundle);

      expect(summary.initial).toHaveLength(3); // main.js, shared.js, styles.css
      expect(summary.lazy).toHaveLength(1); // lazy-chunk.js
      expect(summary.initialTotalRawSize).toBeGreaterThan(0);

      const mainRow = summary.initial.find(r => r.file === 'main.js');
      expect(mainRow).toBeDefined();
      expect(mainRow?.initial).toBe(true);
      expect(mainRow?.kind).toBe('js');

      const cssRow = summary.initial.find(r => r.file === 'styles.css');
      expect(cssRow).toBeDefined();
      expect(cssRow?.kind).toBe('css');

      const lazyRow = summary.lazy.find(r => r.file === 'lazy-chunk.js');
      expect(lazyRow).toBeDefined();
      expect(lazyRow?.initial).toBe(false);
    });
  });

  describe('collectCompilerOutputSizeSummary', () => {
    it('should summarize statically reachable compiler outputs as main.js', () => {
      const summary = collectCompilerOutputSizeSummary([
        {
          path: '/project/out/src/main.js',
          kind: 'js',
          text: 'import { App } from "./app/app";\nconsole.log(App);'
        },
        {
          path: '/project/out/src/app/app.js',
          kind: 'js',
          text: 'export class App {}'
        },
        {
          path: '/project/out/src/polyfills.js',
          kind: 'js',
          text: 'console.log("polyfills");'
        },
        {
          path: '/project/out/src/lazy.js',
          kind: 'js',
          text: 'console.log("lazy");'
        }
      ]);

      expect(summary.initial).toHaveLength(2);
      expect(summary.initial.find(row => row.file === 'main.js')?.rawSize).toBe(
        Buffer.byteLength('import { App } from "./app/app";\nconsole.log(App);', 'utf8') +
        Buffer.byteLength('export class App {}', 'utf8')
      );
      expect(summary.initial.find(row => row.file === 'polyfills.js')?.rawSize).toBe(
        Buffer.byteLength('console.log("polyfills");', 'utf8')
      );
      expect(summary.initial.some(row => row.file === 'lazy.js')).toBe(false);
    });
  });
});
