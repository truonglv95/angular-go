# angular-go

Experimental Angular toolchain powered by a Go port of `ngtsc`.

`angular-go` packages the Go compiler daemon, Vite compile/linker plugins, Angular builders, and a fast Vitest component testbed behind one npm package with multiple entry points.

## Status

This package is usable for the current SPA/Vite demo path, but it is not a full replacement for Angular CLI or Angular's official `ngtsc` yet.

Use it when you want to evaluate:

- Go-based Angular AOT compilation.
- Vite dev/build integration.
- Partial declaration linking for Angular libraries.
- Fast component tests through Vitest and a lightweight Angular runtime harness.
- Angular CLI-style `ng build`, `ng serve`, and `ng test` targets backed by `angular-go`.

Keep Angular's official builder/compiler available for production applications that need SSR, i18n, service worker, full TestBed compatibility, or strict Angular CLI parity.

## Install

```bash
npm install angular-go
```

The package uses optional platform packages to install the native `go-ngc` binary:

- `angular-go-darwin-arm64`
- `angular-go-darwin-x64`
- `angular-go-linux-x64`
- `angular-go-win32-x64`

If the platform binary is not installed, `angular-go` falls back to a `go-ngc` executable on `PATH`.

## Entry Points

```ts
import angularGo from 'angular-go/vite';
import { angularGoCompile } from 'angular-go/compile';
import { angularGoLinker } from 'angular-go/linker';
import { renderComponent, GoTestBed } from 'angular-go/testbed';
import { setupAngularGoTestbed } from 'angular-go/testbed/setup';
import { createAngularGoVitestConfig } from 'angular-go/vitest';
```

The root entry point is intentionally small:

```ts
import { angularGo, angularGoCompile, angularGoLinker } from 'angular-go';
```

Use `angular-go/testbed` explicitly for component testing APIs so importing the root package does not pull Angular runtime testing code.

## Vite Usage

```ts
// vite.config.ts
import { defineConfig } from 'vite';
import angularGo from 'angular-go/vite';

export default defineConfig({
  root: 'src',
  plugins: [
    angularGo({
      mode: 'server',
      project: 'tsconfig.app.json',
      outDir: 'out-tsc/app',
      compile: true,
      link: true,
      hmr: process.env.NG_GO_HMR === '1',
      recompileOnChange: true
    })
  ],
  build: {
    outDir: '../dist',
    emptyOutDir: true
  }
});
```

### Split Compile And Linker Plugins

```ts
import { angularGoCompile } from 'angular-go/compile';
import { angularGoLinker } from 'angular-go/linker';

export default {
  plugins: [
    angularGoCompile({
      mode: 'server',
      project: 'tsconfig.app.json',
      outDir: 'out-tsc/app'
    }),
    angularGoLinker()
  ]
};
```

Use split plugins when you only need linking for dependencies, or when you want to control compile/link phases separately.

## Angular Builder Usage

Configure `angular.json`:

```json
{
  "projects": {
    "app": {
      "architect": {
        "build": {
          "builder": "angular-go:application",
          "options": {
            "browser": "src/main.ts",
            "tsConfig": "tsconfig.app.json",
            "polyfills": ["src/polyfills.ts"],
            "assets": [{ "glob": "**/*", "input": "public" }],
            "styles": ["src/styles.css"]
          },
          "configurations": {
            "production": {
              "outputHashing": "all",
              "subresourceIntegrity": true,
              "statsJson": true,
              "extractLicenses": true,
              "budgets": [
                {
                  "type": "initial",
                  "maximumWarning": "2MB",
                  "maximumError": "4MB"
                }
              ]
            },
            "development": {
              "optimization": false,
              "sourceMap": true
            }
          }
        },
        "serve": {
          "builder": "angular-go:dev-server",
          "configurations": {
            "development": {
              "buildTarget": "app:build:development"
            }
          }
        },
        "test": {
          "builder": "angular-go:unit-test",
          "options": {
            "tsConfig": "tsconfig.spec.json",
            "outDir": "out-tsc/spec",
            "watch": false,
            "testEnvironment": "happy-dom"
          }
        }
      }
    }
  }
}
```

Then run:

```bash
ng build
ng serve
ng test
```

## Vitest Testbed

`angular-go/testbed` provides a fast component test harness for common Angular component tests.

```ts
// vitest.config.ts
import { createAngularGoVitestConfig } from 'angular-go/vitest';

export default createAngularGoVitestConfig({
  project: 'tsconfig.spec.json',
  outDir: 'out-tsc/spec',
  setupFiles: ['test-setup.ts']
});
```

```ts
// src/test-setup.ts
import { setupAngularGoTestbed } from 'angular-go/testbed/setup';

setupAngularGoTestbed();
```

```ts
import { renderComponent, GoTestBed } from 'angular-go/testbed';
import { provideRouter } from '@angular/router';
import { AppComponent } from './app.component';

it('renders a component', async () => {
  const fixture = await renderComponent(AppComponent, {
    providers: [provideRouter([])],
    inputs: { title: 'Hello' }
  });

  expect(fixture.nativeElement.textContent).toContain('Hello');
});

it('supports a small GoTestBed compatibility subset', async () => {
  GoTestBed.resetTestingModule();
  await GoTestBed.configureTestingModule({
    imports: [AppComponent],
    providers: [provideRouter([])]
  }).compileComponents();

  const fixture = GoTestBed.createComponent(AppComponent);
  fixture.detectChanges();
  expect(fixture.componentInstance).toBeTruthy();
});
```

### Testbed Support

Supported today:

- Standalone component rendering.
- Providers and environment providers.
- NgModule provider imports through `importProvidersFrom`.
- Classic inputs through `setInput`.
- Output tracking for `EventEmitter`-style outputs.
- Manual change detection.
- DOM query helpers.
- Basic event helpers: `type`, `clear`, `check`, `select`, `dispatch`.
- Basic `GoTestBed` subset:
  - `configureTestingModule`
  - `compileComponents`
  - `createComponent`
  - `inject`
  - `resetTestingModule`
  - `overrideProvider`

Not fully supported yet:

- Dynamic `declarations` in test modules.
- `overrideTemplate`, `overrideComponent`, `overrideDirective`.
- Full Angular `fakeAsync`, `tick`, and zone testing semantics.
- Full Angular `DebugElement` behavior.
- Browser-layout-sensitive tests in `happy-dom`/`jsdom`.

Use Vitest browser mode or Playwright for overlay, layout, focus, animation, and complex UI library tests.

## Performance

The main performance goal is to reduce repeated Angular compiler startup work:

- `go-ngc --server` runs as a long-lived daemon.
- Compilation outputs can be served from memory.
- Vite provides module graph caching and fast transform invalidation.
- The linker caches processed dependency output.
- Vitest tests run through the same Vite transform pipeline.

Observed in the demo app on the current workspace:

- `ng test --watch=false`: 2 spec files, 12 tests, roughly 3-4 seconds.
- Production build: Vite build succeeds with Go compile/linker and bundle summary output.
- Development build: source maps and unminified output work through the same builder.

Performance depends heavily on project size, dependency graph, cold/warm daemon state, and whether tests use `happy-dom`, `jsdom`, browser mode, or Playwright.

The current implementation optimizes the common SPA/component-test path first. Incremental semantic graph reuse and broader cache invalidation are still areas for future work.

## Support Matrix Versus Angular

| Area | angular-go status | Angular official status |
| --- | --- | --- |
| AOT component compilation | Supported for covered SPA/demo cases | Fully supported |
| Standalone components | Supported for covered cases | Fully supported |
| NgModule metadata | Partially supported | Fully supported |
| Template control flow | Covered for common cases | Fully supported |
| Forms | Basic runtime/test coverage | Fully supported |
| Router | Basic providers/helper support | Fully supported |
| Partial declaration linker | Supported for covered libraries | Fully supported |
| Vite dev/build plugin | Supported | Angular builder owns Vite integration internally |
| HMR | Opt-in, component HMR path under active development | Official Angular HMR behavior |
| `angular.json` application builder | Partial SPA support | Fully supported |
| `ng test` | Vitest builder supported | Official builder support |
| Angular `TestBed` | Fast subset only | Fully supported |
| SSR/prerender/app shell | Not supported; fails fast | Supported |
| i18n/localize | Not supported as builder feature | Supported |
| Service worker | Not supported as builder feature | Supported |
| Web workers | Not supported as builder feature | Supported |
| Library packaging/ng-packagr | Not supported | Supported |

## Builder Option Coverage

Supported or partially supported by `angular-go:application`:

- `browser`
- `tsConfig`
- `polyfills`
- `assets`
- `styles`
- `scripts`
- `index`
- `baseHref`
- `deployUrl`
- `outputPath`
- `optimization`
- `sourceMap`
- `outputHashing`
- `deleteOutputPath`
- `extractLicenses`
- `subresourceIntegrity`
- `crossOrigin`
- `statsJson`
- `budgets`
- `allowedCommonJsDependencies`
- `externalDependencies`
- `define`
- `loader`
- `conditions`
- `namedChunks`

### Dev Server Option Coverage (angular-go:dev-server)

Supported or partially supported options by `angular-go:dev-server`:

- `buildTarget`
- `port`
- `host`
- `proxyConfig`
- `ssl`
- `sslKey`
- `sslCert`
- `headers`
- `open`
- `liveReload`
- `hmr`
- `watch`
- `allowedHosts`
- `define`
- `cliOutput` — Controls dev-server console output style:
  - `angular` (default): print Angular CLI-compatible output.
  - `vite`: keep Vite's native logger behavior.
  - `silent`: suppress non-error dev-server logs.

Unsupported application features fail fast or warn depending on risk:

- `aot: false`
- `localize`
- `serviceWorker`
- `server`
- `ssr`
- `prerender`
- `appShell`
- `outputMode`

## Known Limitations

- The Go compiler is still being brought to parity with Angular's current `ngtsc`.
- Some Angular compiler internals remain incomplete and may fail on cases outside the current test matrix.
- The builder is optimized for browser SPA workflows first.
- The testbed is a fast component harness, not a full Angular `TestBed` clone.
- The package currently publishes native binaries for darwin arm64/x64, linux x64, and win32 x64.

## Publishing

Publish platform packages first, then the main package:

```bash
npm publish angular-packages/angular-go-darwin-arm64
npm publish angular-packages/angular-go-darwin-x64
npm publish angular-packages/angular-go-linux-x64
npm publish angular-packages/angular-go-win32-x64
npm publish angular-packages/angular-go
```

In this repository, run from each package directory:

```bash
npm publish
```

Use `npm publish --dry-run` before publishing a new version.

