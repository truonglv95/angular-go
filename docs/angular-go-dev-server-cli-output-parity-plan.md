# Angular Go Dev Server CLI Output Parity Plan

This plan describes how to make `angular-go:dev-server` print the same console output shape as Angular's official `@angular/build:dev-server` during `ng serve`.

The target output is:

```text
Initial chunk files | Names         | Raw size
main.js             | main          | 72.36 kB |
styles.css          | styles        | 16.11 kB |
polyfills.js        | polyfills     | 91 bytes |

                    | Initial total | 88.56 kB

Application bundle generation complete. [1.832 seconds] - 2026-06-07T16:38:33.595Z

Watch mode enabled. Watching for file changes...
NOTE: Raw file sizes do not reflect development server per-request transformations.
  ➜  Local:   http://localhost:4200/
  ➜  press h + enter to show help
```

## Current State

`angular-go:dev-server` currently starts a Vite dev server directly:

- Source: `angular-packages/angular-go/src/builders/dev-server/index.ts`
- It prints:
  - `Starting development server with go-ngc and Vite...`
  - `Vite Dev Server is running at http://...`
- It does not own an Angular CLI-style bundle table.
- Vite's normal dev output can still leak through unless silenced.

`angular-go` Vite plugin already has a production/dry-run bundle summary:

- Source: `angular-packages/angular-go/src/vite-plugin/index.ts`
- It currently prints `Angular Initial Bundle Size Summary`.
- That output is useful for prod build, but it does not match Angular dev-server output.
- The dev-server builder should own Angular CLI output formatting instead of pushing this responsibility into the compiler or linker.

## Design Principle

Keep responsibilities clean:

- `go-ngc`: compile/link Angular code.
- Vite plugin: integrate compile/link/HMR with Vite.
- Dev-server builder: own CLI output shape, terminal UX, watch messages, server URL messages.

This avoids changing compiler behavior just to match console output.

## Target Architecture

Add a shared Angular CLI-style reporting layer:

```text
angular-packages/angular-go/src/builders/shared/
  angular-cli-dev-output.ts
  bundle-stats.ts
```

Then wire it into:

```text
angular-packages/angular-go/src/builders/dev-server/index.ts
```

Optional later reuse:

```text
angular-packages/angular-go/src/builders/application/index.ts
```

## Phase 1: Add Dev Output Schema Option

Update:

```text
angular-packages/angular-go/src/builders/dev-server/schema.json
```

Add:

```json
{
  "cliOutput": {
    "type": "string",
    "enum": ["angular", "vite", "silent"],
    "default": "angular",
    "description": "Controls dev-server console output style."
  }
}
```

Behavior:

- `angular`: print Angular CLI-compatible output.
- `vite`: keep Vite's native logger behavior.
- `silent`: suppress non-error dev-server logs.

Default should be `angular` for `ng serve` parity.

Acceptance:

- `ng serve` uses Angular-style output by default.
- Existing direct `npx vite` output remains unchanged.

## Phase 2: Silence Vite Logs When Angular Output Is Enabled

In `dev-server/index.ts`, when `cliOutput === 'angular'`:

```ts
const cliOutput = options.cliOutput ?? 'angular';
const useAngularOutput = cliOutput === 'angular';

const viteConfig = {
  logLevel: useAngularOutput ? 'silent' : undefined,
  customLogger: useAngularOutput ? createAngularDevServerLogger(context) : undefined,
  ...
};
```

Logger rules:

- Suppress Vite ready banner.
- Suppress duplicate `Local:` line.
- Allow errors and warnings through.
- Preserve useful dependency optimization warnings if they affect app behavior.

Suggested helper:

```ts
function createAngularDevServerLogger(context: BuilderContext) {
  return {
    hasErrorLogged(error?: Error) {
      return false;
    },
    info(message: string) {
      // intentionally suppressed
    },
    warn(message: string) {
      context.logger.warn(message);
    },
    warnOnce(message: string) {
      context.logger.warn(message);
    },
    error(message: string) {
      context.logger.error(message);
    },
    clearScreen() {
      // no-op; Angular CLI does not clear this output block.
    }
  };
}
```

Acceptance:

- No `VITE v... ready in ...` line when `cliOutput: "angular"`.
- No duplicate Vite local URL line.
- Build errors still appear.

## Phase 3: Extract Bundle Stats Model

Create:

```text
angular-packages/angular-go/src/builders/shared/bundle-stats.ts
```

Types:

```ts
export interface BundleSizeRow {
  file: string;
  name: string;
  rawSize: number;
  kind: 'js' | 'css';
  initial: boolean;
}

export interface BundleSizeSummary {
  initial: BundleSizeRow[];
  lazy: BundleSizeRow[];
  initialTotalRawSize: number;
}
```

Add utilities:

```ts
export function formatBytes(bytes: number): string;
export function collectBundleSizeSummary(bundle: Record<string, any>): BundleSizeSummary;
```

Important details:

- Dev output only displays raw size, not gzip.
- Use `bytes` label for values under 1024 bytes if Angular's output does.
- Use `kB`, `MB`, `GB` with two decimals.
- Preserve output ordering:
  1. `main.js`
  2. `styles.css`
  3. `polyfills.js`
  4. remaining initial chunks, stable sorted
- Ignore empty style JS proxy chunks.
- Classify CSS imported by initial chunks as initial.

Implementation notes:

- Reuse logic currently embedded in `angularGoCompile.generateBundle`.
- Remove ANSI color from this dev reporter. Angular dev output is plain table output.
- Keep production summary color/gzip logic separate.

Acceptance:

- Given a synthetic Rollup bundle object, stats match expected initial rows and total.
- `polyfills.js` appears when configured.
- `styles.css` appears when global styles are configured.

## Phase 4: Add Angular CLI Table Formatter

Create:

```text
angular-packages/angular-go/src/builders/shared/angular-cli-dev-output.ts
```

Formatter API:

```ts
export interface DevBundleOutputOptions {
  durationMs: number;
  timestamp?: Date;
}

export function formatAngularDevBundleOutput(
  summary: BundleSizeSummary,
  options: DevBundleOutputOptions
): string;

export function formatAngularDevServerReadyOutput(url: string): string;
```

Expected format:

```text
Initial chunk files | Names         | Raw size
main.js             | main          | 72.36 kB |
styles.css          | styles        | 16.11 kB |
polyfills.js        | polyfills     | 91 bytes |

                    | Initial total | 88.56 kB

Application bundle generation complete. [1.832 seconds] - 2026-06-07T16:38:33.595Z
```

Formatting rules:

- Column 1 header: `Initial chunk files`
- Column 2 header: `Names`
- Column 3 header: `Raw size`
- Add trailing `|` after each asset row to match Angular's output shape.
- Blank line before completion message.
- Duration:
  - seconds with three decimals
  - `durationMs / 1000`
- Timestamp:
  - `new Date().toISOString()`
- Total row:
  - empty first column
  - `Initial total` second column
  - raw total third column

Server ready formatter:

```text
Watch mode enabled. Watching for file changes...
NOTE: Raw file sizes do not reflect development server per-request transformations.
  ➜  Local:   http://localhost:4200/
  ➜  press h + enter to show help
```

Acceptance:

- Snapshot tests can compare exact string output.
- Output width adapts when file names are longer than `main.js`.
- Output remains stable across platforms.

## Phase 5: Implement Dry Dev Bundle Build

In `dev-server/index.ts`, add a helper:

```ts
async function runDevBundleSummaryBuild(params: {
  viteConfig: any;
  browserEntry: string;
  workspaceRoot: string;
  buildOptions: any;
  context: BuilderContext;
}): Promise<BundleSizeSummary>
```

It should call Vite `build()` with:

```ts
const result = await build({
  ...sharedViteConfig,
  configFile: false,
  logLevel: 'silent',
  build: {
    write: false,
    minify: false,
    sourcemap: false,
    rollupOptions: {
      input: createDevRollupInputs(...),
      output: {
        entryFileNames: '[name].js',
        chunkFileNames: '[name].js',
        assetFileNames: '[name].[ext]'
      }
    }
  }
});
```

Important:

- Do not use hashed file names in dev output.
- Do not write to disk.
- Use the same Angular Go plugin setup as the dev server.
- Avoid recursively triggering another dry build from inside the plugin.
- Avoid double-starting `go-ngc --server` if the main dev server already has one.

Recommended approach:

1. Create the main Vite server config.
2. Build a second "summary config" with a plugin option:

```ts
angularGo({
  ...
  devSummaryBuild: true
})
```

3. In the Vite plugin, make `devSummaryBuild` disable the current internal dry-run summary.

Alternative, cleaner:

- Do not reuse the compile plugin dry-run path.
- Instead, use the same compiled output already produced by the dev server's startup compile.
- Run Vite build only for bundling and size collection.

Acceptance:

- Startup prints the bundle table once.
- No recursive dry builds.
- No output files are created by the summary build.
- Dev server still starts normally.

## Phase 6: Wire Startup Output Flow

Current flow:

```ts
const server = await createServer(viteConfig);
await server.listen();
context.logger.info(`Vite Dev Server is running at ...`);
```

New Angular-output flow:

```ts
const startTime = performance.now();
const server = await createServer(viteConfig);

const summary = await runDevBundleSummaryBuild(...);
const durationMs = performance.now() - startTime;
context.logger.info(formatAngularDevBundleOutput(summary, { durationMs }));

await server.listen();
context.logger.info(formatAngularDevServerReadyOutput(localUrl));
```

Potential order issue:

Angular official output prints bundle generation before URL output. To match that:

1. Create server.
2. Run summary build.
3. Print bundle table.
4. Listen.
5. Print watch/local URL block.

If Vite needs `listen()` before some plugin behavior works, invert steps 2 and 4 but still suppress Vite's own ready banner.

Acceptance:

- First `ng serve` output order matches Angular official.
- Local URL is printed only once.
- `host: "0.0.0.0"` should still show a useful local URL.

## Phase 7: Wire Rebuild Output Flow

Angular official prints the bundle table again after rebuilds:

```text
Initial chunk files | Names         | Raw size
...
Application bundle generation complete. [2.788 seconds] - timestamp
```

Implementation options:

### Option A: Hook Vite `handleHotUpdate`

Add a dev-server-only plugin after `angularGo(...)`:

```ts
function angularCliRebuildReporterPlugin(...) {
  return {
    name: 'angular-go-cli-rebuild-reporter',
    async handleHotUpdate(ctx) {
      scheduleReport();
    }
  };
}
```

Use debounce:

```ts
const rebuildDebounceMs = 30;
```

Flow:

1. File change arrives.
2. Vite plugin `angularGo` recompiles.
3. Reporter waits for current transform/recompile to settle.
4. Reporter runs dry bundle build.
5. Reporter prints bundle table.

Risk:

- Plugin ordering matters.
- `handleHotUpdate` hooks can run before Angular compile promise has finished.

### Option B: Emit Event From Angular Go Compile Plugin

Add optional callback to `angularGo` compile options:

```ts
onCompileComplete?: (event: {
  reason: string;
  durationMs: number;
  command: 'serve' | 'build';
}) => void | Promise<void>;
```

Dev-server builder passes:

```ts
onCompileComplete: async () => {
  await printRebuildSummary();
}
```

This is cleaner because the compile plugin knows exactly when `go-ngc` finished.

Recommended: Option B.

Acceptance:

- Editing `app.html` prints a new Angular-style bundle block.
- Editing `app.ts` prints a new Angular-style bundle block.
- Editing global `styles.css` prints a new Angular-style bundle block.
- Rapid multiple changes coalesce into one output block.

## Phase 8: Avoid Duplicate Summary From Vite Plugin

Current code in `vite-plugin/index.ts` can call `build()` internally and print:

```text
Application bundle generation complete. [...]
```

For Angular CLI parity:

- Dev-server builder should be the only owner of this output.
- Vite plugin should not print dev bundle summary by default.

Add option:

```ts
devBundleSummary?: false | 'plugin' | 'builder';
```

Default:

- Direct Vite plugin usage: `false` or existing behavior, depending on current user expectation.
- Angular builder usage: `builder`.

Simpler first step:

- In builder-created `angularGo(...)`, pass:

```ts
devBundleSummary: false
```

Acceptance:

- No duplicate `Application bundle generation complete` lines.
- Direct `npx vite` behavior is intentionally documented.

## Phase 9: Match Dev Entry Names

Angular official output uses:

```text
main.js
styles.css
polyfills.js
```

The builder must create stable entry names:

- Browser entry: `main`
- Polyfills entry: `polyfills`
- Global styles entry: `styles`

Current production builder has helper logic for polyfills/styles:

- `createPolyfillMainPlugin(...)`
- global styles/scripts normalization

Refactor shared helpers:

```text
angular-packages/angular-go/src/builders/shared/entries.ts
```

Move or duplicate carefully:

```ts
export function createDevEntryPoints(params): Record<string, string>;
export function normalizeGlobalEntries(...);
```

Acceptance:

- `styles.css` appears when `angular.json` has global styles.
- `polyfills.js` appears when `polyfills` is configured.
- Names match Angular official exactly for supported cases.

## Phase 10: Tests

Add unit tests for formatters:

```text
angular-packages/angular-go/src/builders/shared/angular-cli-dev-output.spec.ts
angular-packages/angular-go/src/builders/shared/bundle-stats.spec.ts
```

If current package test setup does not include these, add Vitest test config for `angular-go`.

Formatter tests:

- exact output with `main.js`, `styles.css`, `polyfills.js`
- long filenames align columns
- bytes/kB/MB formatting
- empty lazy rows ignored

Builder E2E test:

```text
angular-packages/new-demo-app/tests/dev-server-output.spec.ts
```

or add to existing Playwright/builder smoke tests.

Test command:

```bash
cd angular-packages/new-demo-app
npm run e2e:prepare
npm run dev:go
```

Automated assertion:

- Spawn `npm run dev:go`.
- Wait for:
  - `Initial chunk files | Names`
  - `Application bundle generation complete.`
  - `Watch mode enabled. Watching for file changes...`
  - `Local:   http://localhost:4200/`
- Edit `src/app/app.html`.
- Assert another `Application bundle generation complete.` appears.

Acceptance:

- Output is stable enough for snapshot/regression tests.
- Dev server still serves app on port 4200.
- Rebuild output appears after file edits.

## Phase 11: Developer UX Options

Add docs to:

```text
angular-packages/angular-go/README.md
```

Document:

```json
{
  "serve": {
    "builder": "angular-go:dev-server",
    "options": {
      "cliOutput": "angular"
    }
  }
}
```

Modes:

```text
angular: Angular CLI-compatible output.
vite:    Native Vite output.
silent:  Only errors/warnings.
```

Also document:

- `NG_GO_HMR=1` controls component HMR if config wires it.
- Output parity does not imply exact Angular internal dev-server implementation parity.

## Risks And Mitigations

### Risk: Dry Bundle Build Slows Startup

The Angular-style table requires bundling metadata. A dry build adds overhead.

Mitigation:

- Make it configurable via `cliOutput`.
- Cache summary when inputs have not changed.
- Debounce rebuild summary generation.

### Risk: Recursive Build From Plugin

The Vite plugin already has dry-run logic. If the builder also calls `build()`, recursion can happen.

Mitigation:

- Add an explicit plugin option to disable plugin-owned summary output.
- Use a guard flag in the plugin for summary builds.

### Risk: Output Size Differs From Angular Official

Vite dev per-request transforms differ from Angular's internal builder output.

Mitigation:

- Match output shape first.
- Accept small size differences unless the bundle construction is identical.
- Keep Angular's note:

```text
NOTE: Raw file sizes do not reflect development server per-request transformations.
```

### Risk: Duplicate Logs

Vite and Angular builder logs can both print URL and ready messages.

Mitigation:

- Use `logLevel: 'silent'`.
- Use `customLogger`.
- Print URL only from dev-server builder.

## Implementation Order

1. Add `cliOutput` schema option.
2. Add shared formatter module and unit tests.
3. Add bundle stats collector and unit tests.
4. Silence Vite logs in Angular output mode.
5. Add dry dev bundle summary build in dev-server builder.
6. Print startup Angular-style table and ready block.
7. Add compile-complete callback or rebuild reporter.
8. Print rebuild Angular-style table after TS/HTML/CSS edits.
9. Disable duplicate plugin summary for builder path.
10. Add E2E output assertions.
11. Update README.

## Minimum Viable Version

For a first implementation that already feels close:

1. Add formatter.
2. Run dry build once on startup.
3. Print Angular-style table.
4. Print watch/local URL block.
5. Suppress Vite ready banner.

This gets startup output close to Angular official.

Then add rebuild output parity in the next step.

## Production-Ready Acceptance Criteria

`npm run dev:go` should print:

```text
Initial chunk files | Names         | Raw size
...
Application bundle generation complete. [x.xxx seconds] - <ISO timestamp>

Watch mode enabled. Watching for file changes...
NOTE: Raw file sizes do not reflect development server per-request transformations.
  ➜  Local:   http://localhost:4200/
  ➜  press h + enter to show help
```

After editing `src/app/app.html`, it should print again:

```text
Initial chunk files | Names         | Raw size
...
Application bundle generation complete. [x.xxx seconds] - <ISO timestamp>
```

There should be no:

```text
Starting development server with go-ngc and Vite...
Vite Dev Server is running at ...
VITE v... ready in ...
```

when `cliOutput` is `angular`.

