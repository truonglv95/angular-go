# Angular JSON Builder Option Support Roadmap

Date: 2026-06-06

Baseline schema inspected from:

- `angular-packages/new-demo-app/node_modules/@angular/build/src/builders/application/schema.json`
- `angular-packages/new-demo-app/node_modules/@angular/build/src/builders/dev-server/schema.json`
- `angular-packages/new-demo-app/node_modules/@angular/build/src/builders/unit-test/schema.json`
- `angular-packages/new-demo-app/node_modules/@angular/build/src/builders/extract-i18n/schema.json`
- `angular-packages/new-demo-app/node_modules/@angular/build/src/builders/ng-packagr/schema.json`
- `angular-packages/new-demo-app/node_modules/@angular/cli/lib/config/schema.json`

Angular baseline: `@angular/build` / `@angular/compiler-cli` `21.2.15`.

## Executive Summary

The Go compiler should not directly support every `angular.json` option. Angular's `angular.json` options are builder-level configuration, while `go-ngc` is the compiler/linker.

Production shape should be three layers:

1. `go-ngc`: compiler/linker only.
2. `vite-plugin-angular-go`: Vite compile/link/HMR/in-memory integration.
3. `@angular-go/build` or equivalent builder adapter: reads `angular.json`, applies Angular builder defaults, validates schema, merges configurations, and maps options to Vite/Rollup/go-ngc.

Without the builder adapter, the current plugin can work for demos, but it cannot honestly claim full Angular builder compatibility.

## Current State

Current `go-ngc` and Vite plugin support:

- compile from `tsConfig` / `project`;
- global/local compilation mode;
- linker for partial declarations;
- daemon/server mode;
- opt-in HMR;
- Vite production build and dev integration;
- build-time chunk size summary.

Current gaps:

- no `angular.json` target parser;
- no Architect-compatible builder entrypoint;
- no schema-compatible option normalization;
- no complete mapping for assets, styles, scripts, index, budgets, output hashing, service worker, i18n, SSR, or test/extract targets;
- Vite users still configure most app-level behavior manually.

## Latest Builder Audit

Date: 2026-06-06

New package found:

- `angular-packages/angular-go-build`
- `angular-packages/angular-go-build/builders.json`
- `angular-packages/angular-go-build/src/builders/application/index.ts`
- `angular-packages/angular-go-build/src/builders/dev-server/index.ts`

Verified commands:

```bash
cd angular-packages/angular-go-build && npm run build
cd angular-packages/vite-plugin-angular-go && npm run build
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH npm run build -- --configuration development
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH npm run build -- --configuration production
go test ./angular-packages/compiler/...
go test ./angular-packages/compiler_cli/...
```

Results:

- `angular-go-build` TypeScript build passes.
- `vite-plugin-angular-go` TypeScript build passes.
- `new-demo-app` development build passes through `angular-go-build:application`.
- `new-demo-app` production build passes through `angular-go-build:application`.
- Top-level `application` and `dev-server` schema option names match local `@angular/build` schemas.
- `go test ./angular-packages/compiler_cli/...` passes.
- `go test ./angular-packages/compiler/...` currently fails in low-level expression parser and render3 AST/template transform tests.

Important findings:

- The builder implementation is currently a working skeleton, not Angular CLI-compatible semantics.
- `builders.json` points to `dist/builders/...`, but `dist` is ignored by the repo. The package must either ship compiled output or use an install/build lifecycle that guarantees `dist` exists before Angular Architect loads it.
- The compiled builder currently emits a Node warning because CommonJS output loads Vite's ESM package. The builder package should be converted to ESM or use a clean dynamic import boundary.
- Global styles are modeled as Rollup JS entries, producing empty `style-0.js` chunks. Angular CLI does not expose these empty JS entries.
- `anyComponentStyle` budget is currently applied to all CSS assets, including global styles. Angular's `anyComponentStyle` should apply to component styles only.
- Asset copying uses broad directory copy and does not honor Angular's full asset pattern semantics: `glob`, `input`, `output`, `ignore`, and `followSymlinks`.
- `polyfills` is in the schema but not wired into Vite entry ordering.
- Object-form `styles` and `scripts` are in the schema but the implementation treats entries as strings.
- `outputPath` object form only uses `base`; `browser`, `server`, and `media` are not implemented.
- `sourceMap` object form is only partially mapped.
- `optimization` object form is only partially mapped and does not handle style/font sub-options.
- `index: false`, `index.output`, preload, base href, deploy URL, SRI, and cross-origin semantics are incomplete.
- `dev-server` resolves `buildTarget` with a naive `split(':')`; comma-separated configurations and Angular target parsing semantics are not covered.
- `dev-server` does not map `define`, `liveReload`, `servePath`, `poll`, `prebundle`, `proxyConfig` edge cases, or target option merge details completely.

Compiler/linker status relevant to builder production:

- The in-memory daemon protocol exists and supports build, get output, HMR update listing, linker file/code, invalidation, and dispose.
- Builds are serialized globally because the TypeScript checker/link store is not goroutine-safe.
- The current compiler path is good enough for the demo app and selected linker/runtime cases.
- The compiler is not globally green: expression parser recovery and render3 AST/template transform tests currently fail. These may be unrelated to the new builder, but they must be resolved or explicitly classified before production builder claims.
- There are still many `panic("unimplemented")` stubs in ported ngtsc areas such as `ngtsc/imports`, `ngtsc/partial_evaluator`, and `ngtsc/shims`. Even if those are not on the current demo path, production builder use must not allow unsupported paths to crash the process.

## Application Builder Options

Source builder: `@angular/build:application`.

### Compiler-Owned Options

These options should be normalized by the builder and passed to `go-ngc`.

| Option | Required support | Current state | Target behavior |
| --- | --- | --- | --- |
| `tsConfig` | Must | Supported via Vite plugin `project` | Resolve from workspace root and pass to `go-ngc -p`. |
| `aot` | Must | Effectively always AOT | Accept `true`; reject or explicitly unsupported for `false` unless JIT support is added. |
| `browser` | Must | Vite entry is configured manually | Use as Vite/Rollup browser entry. |
| `fileReplacements` | Must | Not builder-owned yet | Apply before TS/Angular resolution via Vite alias/plugin and compiler host mapping. |
| `inlineStyleLanguage` | Must | Component CSS path exists; preprocessors incomplete | Normalize CSS/SCSS/SASS/LESS handling and pass resource info to compiler. |
| `stylePreprocessorOptions` | Must | Not fully builder-owned | Map include paths and Sass options into Vite CSS/preprocessor config. |
| `sourceMap` | Must | Partial | Normalize boolean/object into scripts/styles/hidden/vendor/sourcesContent and map to Vite plus `go-ngc`. |
| `preserveSymlinks` | Must | Not builder-owned | Map to resolver and compiler host behavior. |

### Vite/Rollup-Owned Options

These should be handled by the builder adapter and Vite config.

| Option | Required support | Current state | Target behavior |
| --- | --- | --- | --- |
| `outputPath` | Must | Manual Vite config | Support string and object form: `base`, `browser`, `server`, `media`. |
| `index` | Must | Manual Vite config | Support string/object/false, output path, and `preloadInitial`. |
| `assets` | Must | Manual Vite config | Copy string and object asset patterns with Angular-compatible security boundaries. |
| `styles` | Must | Manual Vite config | Treat as global style entrypoints; support string and object form if schema uses complex entries in future. |
| `scripts` | Must | Not implemented | Support global script bundles and injection order. |
| `polyfills` | Must | Manual Vite config | Support array entries and inject before app entry. |
| `optimization` | Must | Vite default/manual | Normalize boolean/object: scripts, styles minify, inline critical, special comments, font inline. |
| `outputHashing` | Must | Vite default/manual | Support `none`, `all`, `media`, `bundles`. |
| `deleteOutputPath` | Must | Vite default/manual | Clean output only when enabled. |
| `extractLicenses` | Must | Vite/Rollup plugin needed | Extract third-party licenses consistently with Angular CLI. |
| `namedChunks` | Must | Vite default/manual | Map to Rollup output chunk naming. |
| `subresourceIntegrity` | Must for prod parity | Not implemented | Add SRI hashes to generated index assets. |
| `crossOrigin` | Must | Not implemented | Apply to generated script/link tags. |
| `baseHref` | Must | Manual Vite config | Apply to index generation and asset URLs. |
| `deployUrl` | Must | Manual Vite config | Apply to emitted asset URL prefix where Angular supports it. |
| `externalDependencies` | Must | Not builder-owned | Map to Vite/Rollup externals. |
| `allowedCommonJsDependencies` | Must | Not implemented | Warn on CommonJS usage except allowlisted packages. |
| `loader` | Must | Not builder-owned | Map extension loaders to Vite/esbuild/Rollup handling. |
| `define` | Must | Not builder-owned | Map to Vite `define`, while preserving Angular metadata semantics. |
| `conditions` | Must | Not builder-owned | Map to package resolution conditions. |
| `clearScreen` | Should | Not implemented | Map to dev/build logging behavior. |
| `verbose` | Should | Partially supported | Forward to compiler and builder logging. |
| `progress` | Should | Not implemented | Add CLI progress or accept with no-op in non-CLI mode. |
| `statsJson` | Should | Not implemented | Emit build stats JSON compatible enough for analysis. |
| `budgets` | Must for prod builder | Partial summary only | Enforce `initial`, `bundle`, `all`, `any`, `anyScript`, `allScript`, `anyComponentStyle`. |

### Angular Platform Features

These are production-builder features, but they can be phased after the SPA builder is stable.

| Option | Required support | Current state | Target behavior |
| --- | --- | --- | --- |
| `i18nMissingTranslation` | Must eventually | Not implemented | Match Angular diagnostics/warnings. |
| `i18nDuplicateTranslation` | Must eventually | Not implemented | Match Angular diagnostics/warnings. |
| `localize` | Must eventually | Not implemented | Generate localized bundles. |
| `webWorkerTsConfig` | Should | Not implemented | Support worker builds and module graph entries. |
| `serviceWorker` | Should | Not implemented | Generate service worker config from string/false. |
| `security.autoCsp` | Should | Not implemented | Generate CSP hashes for index scripts. |
| `security.allowedHosts` | SSR-only | Not implemented | Apply to SSR server builder when added. |
| `server` | SSR phase | Not implemented | Support server entry or explicit false. |
| `ssr` | SSR phase | Not implemented | Support boolean/object with `entry` and `experimentalPlatform`. |
| `prerender` | SSG phase | Not implemented | Support boolean/object with route discovery and routes file. |
| `appShell` | SSG phase | Not implemented | Support after SSR/prerender foundation. |
| `outputMode` | SSR/SSG phase | Not implemented | Support once multi-output application builder exists. |

## Dev Server Builder Options

Source builder: `@angular/build:dev-server`.

| Option | Required support | Current state | Target behavior |
| --- | --- | --- | --- |
| `buildTarget` | Must | Not parsed from angular.json | Resolve project:target:configuration and merge target options. |
| `port` | Must | Manual Vite config | Map to Vite server port. |
| `host` | Must | Manual Vite config | Map to Vite server host. |
| `proxyConfig` | Must | Manual Vite config | Load proxy config file and map to Vite proxy. |
| `ssl` | Must | Manual Vite config | Map to Vite HTTPS config. |
| `sslKey` | Must | Manual Vite config | Load key file. |
| `sslCert` | Must | Manual Vite config | Load cert file. |
| `allowedHosts` | Must | Manual Vite config | Support boolean or string array. |
| `define` | Must | Not builder-owned | Merge with build target define. |
| `headers` | Must | Manual Vite config | Map to Vite server headers. |
| `open` | Must | Manual Vite config | Map to Vite server open. |
| `verbose` | Should | Partial | Forward to plugin/compiler logging. |
| `liveReload` | Must | Vite default/manual | Disable reload when false. |
| `servePath` | Should | Not implemented | Map to Vite base/server path behavior. |
| `hmr` | Must | Supported as plugin opt-in | Keep opt-in and configuration-driven. |
| `watch` | Must | Manual Vite config | Disable watcher when false. |
| `poll` | Should | Manual Vite config | Map to Vite watcher polling. |
| `inspect` | SSR-only | Not implemented | Support after SSR dev server exists. |
| `prebundle` | Must | Partial linker optimize behavior | Map to Vite optimizeDeps and support `exclude`. |

## Other Angular Builders

### `@angular/build:unit-test`

Not required for the first production application builder, but required for full Angular CLI replacement.

Options to support later:

- `buildTarget`
- `tsConfig`
- `runner`
- `runnerConfig`
- `browsers`
- `browserViewport`
- `include`
- `exclude`
- `filter`
- `watch`
- `headless`
- `debug`
- `ui`
- `coverage`
- `coverageInclude`
- `coverageExclude`
- `coverageReporters`
- `coverageThresholds`
- `coverageWatermarks`
- `reporters`
- `outputFile`
- `providersFile`
- `setupFiles`
- `progress`
- `listTests`
- `dumpVirtualFiles`

### `@angular/build:extract-i18n`

Required once i18n support is targeted:

- `buildTarget`
- `format`
- `progress`
- `outputPath`
- `outFile`
- `i18nDuplicateTranslation`

### `@angular/build:ng-packagr`

Library packaging should stay delegated to `ng-packagr` until the application builder is production-grade:

- `project`
- `tsConfig`
- `watch`
- `poll`

## Implementation Plan

## Immediate Priority Queue

These tasks should be done before calling the builder production-ready.

### P0: Packaging And Runtime Loading

Tasks:

- Decide package output policy:
  - commit `dist` intentionally; or
  - add `prepare`/`prepack` and ensure file-based local installs build `dist`; or
  - change builder resolution strategy so Angular Architect can load source through a supported runtime.
- Convert `angular-go-build` to clean ESM loading for Vite:
  - set package `"type": "module"`; or
  - use dynamic `await import('vite')` from a CJS-compatible boundary.
- Add a CI gate that deletes `angular-packages/angular-go-build/dist`, reinstalls, and verifies `ng build` still works.

Acceptance:

- A clean checkout can run `ng build` without manually running `npm run build` inside `angular-go-build`.
- No CommonJS-loading-ESM warning is emitted by the builder.

### P1: Normalize Angular Options Before Generating Vite Config

Tasks:

- Move option parsing into separate tested modules:
  - `normalizeOutputPath`
  - `normalizeIndex`
  - `normalizeOptimization`
  - `normalizeSourceMap`
  - `normalizeAssets`
  - `normalizeGlobalStyles`
  - `normalizeGlobalScripts`
  - `normalizePolyfills`
  - `normalizeBudgets`
- Add golden tests for every schema shape, including boolean/object union options.
- Make unsupported options fail explicitly with a diagnostic until implemented.

Acceptance:

- Builder implementation is mostly orchestration; semantics live in tested helpers.
- Unknown unsupported combinations do not silently fall through to wrong output.

### P2: Fix Application Output Semantics

Tasks:

- Replace global style-as-JS-entry behavior with CSS-only handling so empty `style-0.js` chunks disappear.
- Implement global script injection order and object-form `{ input, bundleName, inject }`.
- Implement polyfills as ordered pre-entry imports.
- Implement Angular asset patterns with glob, input, output, ignore, followSymlinks, and output-path containment checks.
- Implement `outputPath` object form including `browser` and `media`.
- Implement `index: false`, object output path, preload initial assets, `baseHref`, `deployUrl`, `crossOrigin`, and SRI.

Acceptance:

- Production output layout matches Angular CLI for the supported SPA cases.
- Global styles/scripts/polyfills do not create extra bogus chunks.

### P3: Budget And Bundle Reporting Correctness

Tasks:

- Share one bundle graph classifier between `vite-plugin-angular-go` and `angular-go-build`.
- Implement all budget types:
  - `initial`
  - `bundle`
  - `all`
  - `any`
  - `allScript`
  - `anyScript`
  - `anyComponentStyle`
- Distinguish component CSS from global CSS for `anyComponentStyle`.
- Use byte size consistently instead of string length.
- Add tests with warning and error thresholds.

Acceptance:

- Budget behavior does not warn/error on the wrong asset class.
- Build fails on budget error and passes with warning-only budgets.

### P4: Dev Server Compatibility

Tasks:

- Replace manual `buildTarget.split(':')` with Angular target parsing semantics.
- Support comma-separated configurations and correct option merging.
- Map `define`, `liveReload`, `servePath`, `watch`, `poll`, `prebundle`, `headers`, `allowedHosts`, SSL, and proxy config.
- Keep component HMR opt-in, but make the default match the documented builder behavior.
- Add E2E smoke for `ng serve`, not only direct `vite`.

Acceptance:

- `ng serve` is config-driven from `angular.json`.
- Dev server behavior matches the same application options used by production build.

### P5: Compiler Crash-Hardening For Builder Use

Tasks:

- Audit every reachable `panic("unimplemented")` from builder compile/link paths.
- Convert unsupported compiler features into structured diagnostics where possible.
- Keep true internal invariant panics, but recover at CLI/server boundaries and return a failed build result.
- Add regression tests that unsupported cases fail gracefully without killing `go-ngc --server`.

Acceptance:

- A bad app cannot permanently kill the builder daemon through an unsupported Angular feature.
- Vite/Angular Architect receive structured diagnostics.

### Phase 1: Schema Compatibility Package

Goal: provide an Angular-compatible builder surface without changing compiler internals.

Tasks:

- Add a package such as `angular-packages/angular-go-build`.
- Add `builders.json` with `application` and `dev-server` entries.
- Copy or generate schemas from the inspected Angular schema version.
- Add option validation tests using real `angular.json` samples.
- Add target/configuration merge logic matching Angular workspace semantics.

Acceptance:

- A workspace can use `builder: "@angular-go/build:application"` and validate the same core options as `@angular/build:application`.
- Unsupported options are rejected with explicit messages, not ignored silently.

### Phase 2: Normalize Application Options

Goal: turn raw `angular.json` options into a stable internal config.

Tasks:

- Normalize boolean/object `optimization`.
- Normalize boolean/object `sourceMap`.
- Normalize string/object `outputPath`.
- Normalize string/object/false `index`.
- Normalize asset patterns and enforce output path safety.
- Normalize file replacements and path resolution.
- Normalize scripts/styles/polyfills into ordered entry lists.
- Normalize budgets into strongly typed threshold checks.

Acceptance:

- Golden tests cover every option shape listed in the application schema.
- Defaults match Angular for omitted options.

### Phase 3: Generate Vite Config From Angular Options

Goal: let `angular.json` drive Vite instead of requiring manual Vite config parity.

Tasks:

- Generate Vite root/base/build/server config.
- Wire `angularGoCompile` and `angularGoLinker` from normalized options.
- Map `define`, `conditions`, `loader`, `externalDependencies`, `prebundle`.
- Implement assets/index/styles/scripts/polyfills handling.
- Preserve the ability to use compile-only and linker-only plugins separately.

Acceptance:

- `new-demo-app` can run using the builder-derived Vite config.
- Manual Vite config no longer needs to duplicate Angular build options.

### Phase 4: Production Output Features

Goal: match visible Angular CLI build behavior.

Tasks:

- Enforce budgets using raw and gzip sizes where applicable.
- Implement output hashing modes.
- Implement index generation including preload, base href, deploy URL, cross origin, and SRI.
- Implement license extraction.
- Emit optional stats JSON.
- Add Angular-style build summary for initial and lazy chunks.

Acceptance:

- `npm run vite:build` and builder build produce equivalent output layout for the supported SPA feature set.
- Budget failures fail the build with useful diagnostics.

### Phase 5: Dev Server Builder

Goal: support `ng serve`-style configuration through `angular.json`.

Tasks:

- Resolve `buildTarget`.
- Merge target configuration with dev-server options.
- Map host, port, open, SSL, headers, allowed hosts, proxy, watch, poll, live reload, HMR, prebundle.
- Keep HMR opt-in and explicit.
- Add dev E2E tests for config-driven serve.

Acceptance:

- `@angular-go/build:dev-server` starts Vite with config resolved from `angular.json`.
- Dev server behavior is reproducible from workspace config alone.

### Phase 6: Deferred Full Angular Features

Goal: move from SPA production builder to broader Angular CLI parity.

Tasks:

- i18n extraction and localization.
- SSR build outputs.
- SSG/prerender route discovery.
- service worker config generation.
- web worker builds.
- unit-test builder integration.
- library packaging delegation or replacement strategy.

Acceptance:

- Each deferred feature has a compatibility fixture against `@angular/build`.
- Unsupported feature use fails with an explicit roadmap-linked diagnostic.

## Recommended Boundary

Do not make `go-ngc` parse `angular.json` directly.

`go-ngc` should receive normalized compiler options:

- project / tsconfig path;
- compilation mode;
- HMR mode;
- source map mode;
- file replacement map;
- preserve symlinks;
- resource preprocessing inputs;
- output mode: disk or memory.

The builder adapter should own:

- workspace target resolution;
- CLI/default option merging;
- Vite config generation;
- asset/index/style/script/polyfill orchestration;
- budget and bundle reporting;
- dev-server behavior;
- SSR/SSG/service-worker orchestration.

This keeps the compiler reusable and prevents `go-ngc` from becoming a second Angular CLI.
