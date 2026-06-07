# Angular Go Testbed Roadmap

Date: 2026-06-07

## Goal

Build a fast Angular component testing system powered by:

- Vitest as the test runner.
- Vite as the transform/runtime module graph.
- `vite-plugin-angular-go` for compile and linker integration.
- `go-ngc --server` for daemon-based, in-memory Angular compilation.
- Angular runtime APIs for real component creation, dependency injection, change detection, and teardown.

This testbed should optimize the common component/unit-test loop without pretending to immediately replace every API and semantic edge case in Angular's official `TestBed`.

Target positioning:

1. `angular-go-testbed` is the fast default for application component tests.
2. Angular `TestBed` remains the compatibility oracle for advanced testing semantics.
3. Playwright or Vitest browser mode remains required for layout, overlay, animation, and browser-only behavior.

## Roadmap Status

Latest audit: 2026-06-07.

Verified commands:

```bash
cd angular-packages/angular-go-testbed && npm run build && npm test
cd angular-packages/vite-plugin-angular-go && npm run build && npm test
cd angular-packages/angular-go-build && npm run build && npm test
cd angular-packages/new-demo-app && npm test -- --watch=false
```

Latest result:

- `angular-go-testbed`: 1 internal test file, 3 tests passed.
- `new-demo-app`: 2 spec files, 12 tests passed through `angular-go-build:unit-test`.
- `ng test` no longer prints the previous daemon `Unknown context` message.
- Vite sourcemap missing-source warnings are removed by rewriting transform sourcemaps to the original TS source plus `sourcesContent`.

Phase status:

- **Phase 0 - Baseline Audit And Guardrails**: Completed.
- **Phase 1 - Package Skeleton**: Completed.
- **Phase 2 - Vitest Preset & Vite Plugin Integration**: MVP completed.
- **Phase 3 - Runtime Environment**: MVP completed; cleanup now tracks both fast fixtures and `GoTestBed` fixtures.
- **Phase 4 - `renderComponent` MVP**: Completed.
- **Phase 5 - Inputs, Outputs, Signals, & CD**: Partial; classic inputs/outputs and manual change detection are covered, signal-specific tests still need to be added.
- **Phase 6 - Dependency Injection & Providers**: MVP completed; plain providers and NgModule provider imports are covered.
- **Phase 7 - Router Support**: Partial; helper APIs exist, but route navigation/guards/resolvers need dedicated specs.
- **Phase 8 - Forms Support**: Partial; reactive text input is covered, template-driven forms, checkbox/select forms and validation need dedicated specs.
- **Phase 9 - Content Projection**: Partial; precompiled host component projection is covered, explicit host wrapper API and `NgTemplateOutlet` cases still need specs.
- **Phase 10 - `GoTestBed` Compatibility**: MVP completed; subset includes `configureTestingModule`, `compileComponents`, `createComponent`, `inject`, `resetTestingModule`, and `overrideProvider`.
- **Phase 11 - Async, Zone, & Stability**: Partial; `whenStable()` uses `ApplicationRef.isStable`, but fake timers/zone semantics are not production-class yet.
- **Phase 12 - Browser Mode & UI Libraries**: Not completed; config option exists, but browser-mode PrimeNG/CDK overlay specs are still missing.
- **Phase 13 - Parity Test Matrix**: Partial; current demo matrix covers inputs, outputs, DI, provider imports, structural directives, projection, forms MVP and unsupported declarations.
- **Phase 14 - Performance Benchmarking**: Partial; benchmark helper exists, but no benchmark suite/JSON baseline exists yet.
- **Phase 15 - Developer Experience**: Partial; daemon context and sourcemap noise are fixed, but compiler/linker diagnostic formatting still needs dedicated tests.
- **Phase 16 - Builder Integration**: MVP completed; `angular-go-build:unit-test` runs `ng test` through Vitest and go-ngc.
- **Phase 17 - Production Readiness Gate**: Not completed; current state is working MVP, not a full Angular `TestBed` replacement.

## Architecture

```text
Vitest
  -> Vite plugin pipeline
    -> angularGoCompile()
      -> go-ngc --server
      -> in-memory JS/d.ts/source-map outputs
    -> angularGoLinker()
      -> partial declaration linker for dependencies
  -> jsdom or browser runtime
    -> angular-go-testbed runtime harness
      -> Angular createComponent / ApplicationRef / EnvironmentInjector
```

## Package Layout

Create a new package:

```text
angular-packages/angular-go-testbed/
  package.json
  tsconfig.json
  src/
    index.ts
    render.ts
    fixture.ts
    environment.ts
    testbed.ts
    async.ts
    queries.ts
    errors.ts
    vitest/
      config.ts
      setup.ts
      plugin.ts
  test/
    render-component.test.ts
    fixture.test.ts
    providers.test.ts
    inputs-outputs.test.ts
    router.test.ts
    forms.test.ts
    compatibility.test.ts
```

## API Target

### Fast Harness API

```ts
import { renderComponent } from 'angular-go-testbed';

const fixture = await renderComponent(AppComponent, {
  imports: [],
  providers: [],
  inputs: {
    title: 'Hello'
  }
});

expect(fixture.element.textContent).toContain('Hello');

await fixture.setInput('title', 'Updated');
await fixture.detectChanges();
fixture.destroy();
```

### Fixture API

```ts
interface GoComponentFixture<T> {
  component: T;
  element: HTMLElement;
  hostElement: HTMLElement;
  nativeElement: HTMLElement;
  injector: EnvironmentInjector;
  detectChanges(): Promise<void> | void;
  whenStable(): Promise<void>;
  setInput(name: string, value: unknown): Promise<void> | void;
  query(selector: string): HTMLElement | null;
  queryAll(selector: string): HTMLElement[];
  destroy(): void;
}
```

### Compatibility API

```ts
import { GoTestBed } from 'angular-go-testbed';

beforeEach(async () => {
  await GoTestBed.configureTestingModule({
    imports: [AppComponent],
    providers: []
  }).compileComponents();
});

it('creates component', () => {
  const fixture = GoTestBed.createComponent(AppComponent);
  fixture.detectChanges();
  expect(fixture.componentInstance).toBeTruthy();
});
```

Compatibility API is intentionally phased. The fast harness should be implemented first.

## Phase 0 - Baseline Audit And Guardrails

### Tasks

- Audit current `new-demo-app` unit test setup:
  - `src/app/app.spec.ts`
  - `tsconfig.spec.json`
  - `package.json` test scripts
  - current Vitest/JSDOM dependencies
- Audit `vite-plugin-angular-go` compile/link behavior for test mode:
  - daemon lifecycle
  - in-memory output path resolution
  - source map behavior
  - linker cache behavior
- Decide exact test environments:
  - `jsdom` for fast default
  - browser mode only for selected cases
- Add a roadmap status section to this file when implementation starts.

### Acceptance Criteria

- Documented baseline command matrix:
  - Go compiler tests
  - Vite plugin tests
  - builder tests
  - new testbed tests
  - demo app tests
- No code behavior change required in this phase.

## Phase 1 - Package Skeleton

### Tasks

- Create `angular-packages/angular-go-testbed`.
- Configure ESM TypeScript output.
- Add exports:
  - root package exports
  - `./vitest`
  - `./setup`
- Add scripts:
  - `build`
  - `test`
  - `typecheck`
  - `prepack`
- Add package dependencies:
  - peer dependencies: `@angular/core`, `@angular/common`, `@angular/platform-browser`, `rxjs`
  - dev dependencies: `typescript`, `vitest`, `vite`, `jsdom`
- Avoid depending on `@angular/compiler-cli` in the runtime testbed package.

### Acceptance Criteria

- `npm run build` passes inside `angular-go-testbed`.
- Package can be imported from `new-demo-app` through `file:../angular-go-testbed`.

## Phase 2 - Vitest Preset And Vite Plugin Integration

### Tasks

- Add `createAngularGoVitestConfig()` helper.
- Compose Vite plugins:
  - `angularGoCompile({ mode: 'server', compile: true })`
  - `angularGoLinker({ link: true })`
- Support options:
  - `project`
  - `compilerPath`
  - `compilationMode`
  - `hmr: false` by default in test mode
  - `setupFiles`
  - `testEnvironment: 'jsdom' | 'browser'`
- Add a `setupAngularGoTestbed()` setup function.
- Ensure daemon is closed after Vitest run when possible.

### Acceptance Criteria

- A minimal Vitest config can run a standalone Angular component spec through `go-ngc`.
- No disk output is required for compiled component JS.
- Failed compiler diagnostics surface as Vitest transform errors with file/line info.

## Phase 3 - Runtime Environment

### Tasks

- Implement test environment initialization:
  - create or reuse Angular platform context as needed
  - create root test host DOM node
  - create `EnvironmentInjector`
  - support global providers
- Implement teardown:
  - destroy component refs
  - destroy application refs/injectors where applicable
  - remove host DOM nodes
  - reset internal registries between tests
- Support optional automatic cleanup in `afterEach`.

### Acceptance Criteria

- Multiple component tests can run in one Vitest process without state leakage.
- Repeated `renderComponent()` calls do not leave detached DOM nodes.
- Providers are not shared across tests unless explicitly configured globally.

## Phase 4 - `renderComponent` MVP

### Tasks

- Implement:
  - `renderComponent(component, options)`
  - root host creation
  - Angular `createComponent`
  - `ApplicationRef.attachView`
  - initial change detection
  - component destruction
- Support options:
  - `providers`
  - `imports`
  - `inputs`
  - `host`
  - `autoDetectChanges`
- Provide query helpers:
  - `query`
  - `queryAll`
  - `text`

### Acceptance Criteria

- Convert `new-demo-app/src/app/app.spec.ts` to `renderComponent`.
- Tests pass with Vitest and Go compiler.
- Rendered DOM matches the Angular runtime output for basic components.

## Phase 5 - Inputs, Outputs, Signals, And Change Detection

### Tasks

- Implement `setInput(name, value)`.
- Support Angular signal inputs where runtime exposes them.
- Support classic `@Input`.
- Support `@Output` subscription helpers:
  - `output(name)`
  - `emitted(name)`
  - `trigger(selector, event)`
- Implement `detectChanges`.
- Implement `whenStable` with a pragmatic async queue strategy.
- Decide whether zone.js is required or optional.

### Acceptance Criteria

- Tests cover:
  - classic input
  - signal input
  - output event emitter
  - DOM event listener
  - async promise update
  - repeated change detection

## Phase 6 - Dependency Injection And Providers

### Tasks

- Support:
  - plain providers
  - environment providers
  - `inject()` inside component/provider code
  - provider override in fast harness options
- Add `inject(token)` helper for tests.
- Add local injector per rendered component.
- Add global provider setup for test suites.

### Acceptance Criteria

- Tests cover:
  - service injection
  - provider override per test
  - multi providers
  - environment providers such as router providers

## Phase 7 - Router Support

### Tasks

- Add `provideTestRouter(routes, options)` helper.
- Support route initialization before first detect changes.
- Support navigation helpers:
  - `navigateByUrl`
  - `currentUrl`
  - `findByRoute`
- Integrate with Angular `provideRouter`.

### Acceptance Criteria

- Demo app can test routed components without Angular `TestBed`.
- Router tests can wait for navigation stability.
- Basic guards/resolvers can be tested or explicitly classified as unsupported.

## Phase 8 - Forms Support

### Tasks

- Validate template-driven forms.
- Validate reactive forms.
- Add user-event helpers:
  - `type`
  - `clear`
  - `select`
  - `check`
  - `dispatch`
- Ensure DOM event payloads match Angular expectations.

### Acceptance Criteria

- Tests cover:
  - `[(ngModel)]`
  - `formControl`
  - `formGroup`
  - checkbox input
  - select/dropdown where jsdom allows it
- Existing forms regressions from compiler work are represented as testbed specs.

## Phase 9 - Content Projection, Templates, And Structural Directives

### Tasks

- Add host-wrapper rendering:
  - render a host template that uses the component under test
  - pass projected content
  - bind host inputs/outputs
- Support tests for:
  - `ng-content`
  - `ng-template`
  - `NgTemplateOutlet`
  - `@if`
  - `@for`
  - `@switch`
  - deferred blocks where runtime allows

### Acceptance Criteria

- Prior regressions around `TemplateRef.createEmbeddedViewImpl` are covered.
- Structural directive rendering works in the fast harness.

## Phase 10 - `GoTestBed` Compatibility Layer

### Tasks

- Implement subset:
  - `configureTestingModule`
  - `compileComponents`
  - `createComponent`
  - `inject`
  - `resetTestingModule`
  - `overrideProvider`
- Return a fixture shape close to Angular:
  - `componentInstance`
  - `nativeElement`
  - `debugElement` as minimal adapter
  - `detectChanges`
  - `whenStable`
  - `destroy`
- Do not implement unsupported APIs silently.
- Throw explicit unsupported errors for:
  - `overrideTemplate`
  - `overrideComponent`
  - `overrideDirective`
  - `fakeAsync` if not implemented

### Acceptance Criteria

- Simple Angular `TestBed` specs can migrate by import rename only:
  - `TestBed` -> `GoTestBed`
- Unsupported APIs fail fast with actionable messages.

## Phase 11 - Async, Zone, And Stability Semantics

### Tasks

- Decide official default:
  - zoneless fast mode
  - optional zone compatibility mode
- Implement `whenStable` semantics:
  - microtask flush
  - macrotask best effort
  - router navigation wait
  - fixture-level pending tasks if Angular exposes them
- Evaluate whether `fakeAsync`, `tick`, and `flush` should be wrappers around Vitest fake timers or explicitly unsupported.

### Acceptance Criteria

- Async tests are deterministic.
- Timer behavior is documented.
- Unsupported zone APIs fail fast.

## Phase 12 - Browser Mode And Third-Party UI Libraries

### Tasks

- Add Vitest browser mode preset.
- Support browser-only tests for:
  - overlays
  - focus management
  - positioning
  - animations
  - PrimeNG/CDK components
- Keep jsdom mode as default for speed.

### Acceptance Criteria

- PrimeNG dropdown/menu/table smoke tests run in browser mode.
- Browser-mode tests can reuse the same testbed fixture API.

## Phase 13 - Compiler/Linker Parity Test Matrix

### Tasks

- Add paired tests:
  - Angular TestBed result
  - GoTestBed result
- Compare:
  - DOM output
  - input propagation
  - output emission
  - DI behavior
  - router behavior
  - forms behavior
  - structural directive behavior
- Store known divergences with explicit classification:
  - compiler bug
  - runtime harness bug
  - unsupported Angular TestBed feature
  - jsdom/browser limitation

### Acceptance Criteria

- Every historical compiler regression receives a runtime test where practical.
- No known divergence is left unclassified.

## Phase 14 - Performance Benchmarking

### Tasks

- Add benchmarks for:
  - cold start first test
  - warm watch-mode rerun
  - single component test
  - 100 component tests
  - 1000 component tests
  - spec file edit
  - component TS edit
  - template edit
  - dependency edit
- Measure phase timings:
  - Vitest startup
  - Vite transform
  - go-ngc build
  - linker
  - runtime render
  - fixture teardown
- Compare with Angular TestBed baseline.

### Acceptance Criteria

- Benchmark command is stable and documented.
- Results are written to a machine-readable JSON file.
- Roadmap includes performance thresholds before prod claim.

## Phase 15 - Developer Experience

### Tasks

- Improve error messages:
  - compiler diagnostics
  - linker diagnostics
  - unsupported testbed APIs
  - missing DOM environment
  - daemon startup failure
- Add source map support for stack traces.
- Add typed helper APIs.
- Add migration examples:
  - Angular TestBed to `renderComponent`
  - Angular TestBed to `GoTestBed`
  - router tests
  - forms tests
  - PrimeNG browser tests

### Acceptance Criteria

- A developer can diagnose compile/runtime/testbed errors without reading daemon logs.
- Documentation includes realistic app examples.

## Phase 16 - Builder Integration

### Tasks

- Add `angular-go-build:test` builder.
- Read `angular.json` unit-test target options.
- Map to Vitest config:
  - `tsConfig`
  - `polyfills`
  - `assets`
  - `styles`
  - `scripts`
  - `include`
  - `exclude`
  - `watch`
  - `codeCoverage`
  - `reporters`
  - `browsers` or browser-mode equivalent if supported
- Provide `ng test` support through the Go builder.

### Acceptance Criteria

- `npm test` or `ng test` in `new-demo-app` runs with the Go testbed.
- Angular CLI users do not need a hand-written Vitest config for common cases.

## Phase 17 - Production Readiness Gate

### Required Before "Prod Testbed" Claim

- Fast harness is stable for standalone components.
- DI, router, forms, outputs, content projection, and structural directives have tests.
- `GoTestBed` compatibility subset is documented and fail-fast for unsupported APIs.
- jsdom/browser-mode split is explicit.
- Performance benchmark shows meaningful improvement over Angular TestBed for common component tests.
- Compiler daemon does not crash the Vitest process on unsupported compiler paths.
- Testbed can run in CI without orphaned daemon processes.
- E2E smoke tests cover:
  - app component
  - routed page
  - forms page
  - PrimeNG dropdown/table/menu behavior

## Non-Goals For Initial Prod Claim

- Full clone of Angular `TestBed`.
- Full zone.js testing API parity.
- Perfect support for every `override*` API.
- Replacement for Playwright visual/browser behavior tests.
- Replacement for compiler golden/parity tests.

## Implementation Order

Recommended execution order:

1. Phase 0
2. Phase 1
3. Phase 2
4. Phase 3
5. Phase 4
6. Phase 5
7. Phase 6
8. Phase 8
9. Phase 9
10. Phase 7
11. Phase 10
12. Phase 13
13. Phase 14
14. Phase 15
15. Phase 16
16. Phase 12
17. Phase 17

Router and browser mode can be moved earlier if app coverage requires them, but the fastest path to value is:

```text
package skeleton -> Vitest preset -> renderComponent -> inputs/outputs -> DI -> forms/templates -> compatibility layer
```

## Current Decision

Build `angular-go-testbed` as a new package, not inside `vite-plugin-angular-go`.

Reason:

- The plugin should remain build/transform/linker infrastructure.
- The testbed is runtime testing API and should be versioned/documented independently.
- The builder can later depend on both packages to implement `ng test`.
