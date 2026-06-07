import { createAngularGoVitestConfig } from 'angular-go/vitest';

export default createAngularGoVitestConfig({
  project: 'tsconfig.spec.json',
  outDir: 'out-tsc/spec',
  setupFiles: ['test-setup.ts']
});
