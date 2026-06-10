import { createBuilder, BuilderContext, BuilderOutput, targetFromTargetString } from '@angular-devkit/architect';
import path from 'node:path';
import fs from 'node:fs';
import { execSync } from 'node:child_process';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);

interface ExtractI18nOptions {
  browserTarget: string;
  outFile?: string;
}

export default createBuilder<ExtractI18nOptions, BuilderOutput>(async (options, context): Promise<BuilderOutput> => {
  context.logger.info('Running extract-i18n builder (angular-go)...');

  try {
    const browserTarget = targetFromTargetString(options.browserTarget);
    const targetOptions = await context.getTargetOptions(browserTarget);
    const workspaceRoot = context.workspaceRoot;

    // 1. Run the build to generate the JS chunks
    context.logger.info(`Building target ${options.browserTarget}...`);
    const run = await context.scheduleTarget(browserTarget);
    const result = await run.result;
    if (!result.success) {
      context.logger.error('Build failed, cannot extract i18n.');
      return { success: false };
    }

    // 2. Resolve output path
    const outputPath = targetOptions.outputPath as string;
    if (!outputPath) {
      return { success: false, error: 'Cannot find outputPath in browser target options.' };
    }
    const absoluteOutDir = path.resolve(workspaceRoot, outputPath);

    // 3. Resolve go-localize binary
    const binName = process.platform === 'darwin' ? 'go-localize-darwin-arm64' : 'go-localize';
    let goLocalizePath = '';
    try {
      goLocalizePath = require.resolve('@angular-go/darwin-arm64/bin/' + binName);
    } catch (e) {
      goLocalizePath = path.resolve(workspaceRoot, 'node_modules', '@angular-go', 'darwin-arm64', 'bin', binName);
    }

    if (!fs.existsSync(goLocalizePath)) {
      return { success: false, error: `Cannot find go-localize binary at ${goLocalizePath}` };
    }

    // 4. Read i18n configuration from project metadata
    const projectMetadata = await context.getProjectMetadata(browserTarget.project);
    const i18nConfig = projectMetadata?.i18n as any || {};
    
    let outFile = options.outFile;
    let localesArg = '';

    if (!outFile) {
      if (i18nConfig.locales) {
        // Find the directory of the first configured locale to place messages.xlf
        const localeKeys = Object.keys(i18nConfig.locales);
        if (localeKeys.length > 0) {
          const firstLocalePath = i18nConfig.locales[localeKeys[0]];
          outFile = path.join(path.dirname(firstLocalePath), 'messages.xlf');
        }
      }
      if (!outFile) {
        outFile = `apps/${browserTarget.project}/src/locales/messages.xlf`;
      }
    }
    const absoluteOutFile = path.resolve(workspaceRoot, outFile);

    if (i18nConfig.locales) {
      const syncPaths = Object.values(i18nConfig.locales).map(p => path.resolve(workspaceRoot, p as string));
      if (syncPaths.length > 0) {
        localesArg = `--locales=${syncPaths.join(',')}`;
      }
    }

    // 5. Execute go-localize extract
    context.logger.info(`Extracting to ${outFile}...`);
    const cmd = `${goLocalizePath} --mode=extract --dir=${path.join(absoluteOutDir, 'assets')} --outFile=${absoluteOutFile} ${localesArg}`;
    
    execSync(cmd, { stdio: 'inherit' });

    context.logger.info(`Extract-i18n completed successfully.`);
    return { success: true };
  } catch (err: any) {
    context.logger.error(`Extract-i18n failed: ${err.message}`);
    return { success: false, error: err.message };
  }
});
