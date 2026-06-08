import { describe, it, expect } from 'vitest';
import { formatAngularDevBundleOutput, formatAngularDevServerReadyOutput } from './angular-cli-dev-output.js';
import { BundleSizeSummary } from './bundle-stats.js';

describe('angular-cli-dev-output', () => {
  describe('formatAngularDevBundleOutput', () => {
    it('should format bundle stats table exactly like Angular CLI', () => {
      const summary: BundleSizeSummary = {
        initial: [
          { file: 'main.js', name: 'main', rawSize: 74096, kind: 'js', initial: true },
          { file: 'styles.css', name: 'styles', rawSize: 16496, kind: 'css', initial: true },
          { file: 'polyfills.js', name: 'polyfills', rawSize: 91, kind: 'js', initial: true }
        ],
        lazy: [],
        initialTotalRawSize: 90683
      };

      const options = {
        durationMs: 1832,
        timestamp: new Date('2026-06-07T16:38:33.595Z')
      };

      const output = formatAngularDevBundleOutput(summary, options);

      // Strip ANSI escape codes to check pure text formatting and alignment
      const cleanOutput = output.replace(/\x1b\[[0-9;]*m/g, '');

      // Check header
      expect(cleanOutput).toContain('Initial chunk files | Names         | Raw size');

      // Check rows
      expect(cleanOutput).toContain('main.js             | main          | 74.10 kB |');
      expect(cleanOutput).toContain('styles.css          | styles        | 16.50 kB |');
      expect(cleanOutput).toContain('polyfills.js        | polyfills     | 91 bytes |');

      // Check footer
      expect(cleanOutput).toContain('                    | Initial total | 90.68 kB');

      // Check completion status
      expect(cleanOutput).toContain('Application bundle generation complete. [1.832 seconds] - 2026-06-07T16:38:33.595Z');
    });

    it('should align columns correctly even with longer file names', () => {
      const summary: BundleSizeSummary = {
        initial: [
          { file: 'main-with-extremely-long-name-for-testing.js', name: 'main', rawSize: 1024, kind: 'js', initial: true },
          { file: 'styles.css', name: 'styles', rawSize: 512, kind: 'css', initial: true }
        ],
        lazy: [],
        initialTotalRawSize: 1536
      };

      const options = {
        durationMs: 1000,
        timestamp: new Date('2026-06-07T16:38:33.595Z')
      };

      const output = formatAngularDevBundleOutput(summary, options);
      const cleanOutput = output.replace(/\x1b\[[0-9;]*m/g, '');

      // The header column 1 should be padded to match the longest filename
      expect(cleanOutput).toContain('Initial chunk files                          | Names         |  Raw size');
      expect(cleanOutput).toContain('main-with-extremely-long-name-for-testing.js | main          |   1.02 kB | ');
      expect(cleanOutput).toContain('styles.css                                   | styles        | 512 bytes | ');
    });
  });

  describe('formatAngularDevServerReadyOutput', () => {
    it('should format ready messages correctly', () => {
      const url = 'http://localhost:4200/';
      const output = formatAngularDevServerReadyOutput(url);

      expect(output).toContain('Watch mode enabled. Watching for file changes...');
      expect(output).toContain('NOTE: Raw file sizes do not reflect development server per-request transformations.');
      expect(output).toContain('➜  Local:   http://localhost:4200/');
      expect(output).toContain('➜  press h + enter to show help');
    });
  });
});
