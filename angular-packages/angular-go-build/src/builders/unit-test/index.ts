import { createBuilder, BuilderContext, BuilderOutput } from '@angular-devkit/architect';
import { createRequire } from 'node:module';
import path from 'node:path';
import fs from 'node:fs';

const require = createRequire(import.meta.url);

export default createBuilder<any, BuilderOutput>(async (options, context): Promise<BuilderOutput> => {
  context.logger.info(`Running unit tests with Vitest and go-ngc...`);

  try {
    const workspaceRoot = context.workspaceRoot;
    const tsConfigPath = path.resolve(workspaceRoot, options.tsConfig || 'tsconfig.spec.json');

    // Resolve go-ngc compiler path
    let compilerPath = '';
    let currentDir = workspaceRoot;
    while (currentDir !== path.dirname(currentDir)) {
      const candidate = path.join(currentDir, 'go-ngc');
      if (fs.existsSync(candidate)) {
        compilerPath = candidate;
        break;
      }
      currentDir = path.dirname(currentDir);
    }
    if (!compilerPath) {
      compilerPath = 'go-ngc'; // Fallback to PATH resolution
    }

    const { startVitest } = await import('vitest/node');
    const { createAngularGoVitestConfig } = await import('angular-go-testbed/vitest');

    const setupFiles: string[] = [];
    if (options.polyfills) {
      if (Array.isArray(options.polyfills)) {
        setupFiles.push(...options.polyfills);
      } else if (typeof options.polyfills === 'string') {
        setupFiles.push(options.polyfills);
      }
    }
    // Also add test-setup.ts if it exists
    const localSetupFile = path.join(workspaceRoot, 'src/test-setup.ts');
    if (fs.existsSync(localSetupFile)) {
      setupFiles.push('test-setup.ts');
    }

    // Resolve Vitest config
    const vitestConfig = createAngularGoVitestConfig({
      project: options.tsConfig || 'tsconfig.spec.json',
      compilerPath: compilerPath,
      outDir: options.outDir || 'out-tsc/spec',
      setupFiles: setupFiles.length > 0 ? setupFiles : undefined,
      testEnvironment: options.testEnvironment,
      compilationMode: options.compilationMode,
    });

    const watch = options.watch === true;

    // Explicitly configure maxWorkers: 1 and fileParallelism: false in the cliOptions
    // to force sequential execution of test files, preventing concurrent compiler daemon panics.
    const vitest = await startVitest('test', [], {
      watch,
      run: !watch,
      maxWorkers: 1,
      fileParallelism: false
    } as any, vitestConfig as any);

    if (!vitest) {
      return { success: false, error: 'Failed to start Vitest.' };
    }

    if (!watch) {
      const testModules = vitest.state.getTestModules();
      const allPassed = testModules.every((mod: any) => mod.ok());
      await vitest.close();
      if (!allPassed) {
        return { success: false, error: 'Some unit tests failed.' };
      }
    }

    return { success: true };
  } catch (err: any) {
    context.logger.error(`Error running unit tests: ${err.stack || err.message}`);
    return { success: false, error: err.message };
  }
});
