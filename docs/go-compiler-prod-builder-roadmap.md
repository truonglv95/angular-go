# Go Angular Compiler Production Builder Roadmap

This roadmap tracks the work needed to move the current Go Angular compiler/linker from an experimental Vite integration toward a production-grade builder.

Current baseline:

- Angular baseline: `@angular/compiler-cli@21.2.15`.
- Go compiler binary: `go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go`.
- Local full gate:

```bash
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH ./ci.sh
```

Latest verified state:

- Render3 parity: `exact=0 normalized=19 semantic=0 missing-go=0 missing-ngtsc=0 compile-failed=0`.
- Linker parity: `exact=0 normalized=8 semantic=0 missing-go=0 missing-ngtsc=0 compile-failed=0`.
- Vite production build: passes with chunk-size warning only.
- Playwright runtime smoke: `6 passed`.

## Production Readiness Definition

The builder can be considered production-ready only when these are true:

- Render3 parity has no semantic diffs across the supported corpus.
- Linker parity has no semantic diffs across Angular packages and selected third-party libraries.
- Runtime smoke passes in both Vite dev-server and production-preview modes.
- Diagnostics include file, line, column, and useful compiler context.
- Sourcemaps are emitted and usable in browser/debugger workflows.
- Cache invalidation is graph-based enough to avoid stale compiled output.
- CI runs from a clean checkout without manual local state.
- Unsupported compiler/linker cases fail explicitly instead of emitting wrong JavaScript.

## Phase 0: Repository Hygiene And Baseline Freeze

**Status: Done for current workspace cleanup**

Goal: make the current working state reproducible before expanding scope.

Tasks:

- Group current changes into clear areas:
  - compiler/render3
  - linker
  - parity tools and fixtures
  - Vite plugin
  - demo app and smoke tests
  - docs
- Remove temporary demo files such as ad-hoc `temp*.js`, `test*.ts/js`, `a.js`, and invalid local experiments.
- Decide whether stale `angular-packages/new-demo-app/vite.config.js` should be removed or regenerated.
- Ensure `./ci.sh` passes after cleanup.
- Create a checkpoint commit or branch before starting the next technical phase.

Done:

- Removed ad-hoc demo experiments and generated source-adjacent JavaScript from `angular-packages/new-demo-app`.
- Removed stale `angular-packages/new-demo-app/vite.config.js`; Vite and Playwright now use `vite.config.ts`.
- Removed local fixture `project/node_modules` symlink artifacts under render3 testdata.
- Removed temporary patch files and the untracked `angular-packages/go-ngc` binary artifact.
- Removed staged demo-only artifacts `angular-packages/new-demo-app/test.js` and `angular-packages/new-demo-app/go-ngc` from the worktree/index.
- Verified the cleaned workspace with:

```bash
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH ./ci.sh
```

Latest Phase 0 gate result:

- Go unit/integration tests: passed.
- Render3 parity strict: `exact=0 normalized=19 semantic=0 missing-go=0 missing-ngtsc=0 compile-failed=0`.
- Linker parity strict: `exact=0 normalized=8 semantic=0 missing-go=0 missing-ngtsc=0 compile-failed=0`.
- Vite plugin build: passed.
- Vite production build: passed with chunk-size warning only.
- Playwright smoke: `6 passed`.

Acceptance:

- `git status` contains only intentional source, fixture, config, binary rebuild, and doc changes: met for the current workspace.
- `./ci.sh` passes from the cleaned state: met.
- The expected Node and Go versions are documented: met.

## Phase 1: Lock IR Pipeline Architecture

**Status: Next**

Goal: remove the biggest internal compiler architecture risk: mixed assumptions around `OpList`.

Decision:

- Use the slice-backed `OpList` model as the canonical Go implementation.
- Do not reintroduce linked-list semantics unless a benchmark proves a real need.

Tasks:

- Audit all template pipeline phases for linked-list assumptions.
- Add invariant tests for:
  - op insert/remove/prepend ordering
  - slot allocation after op mutation
  - nested embedded view ordering
  - listener handler op ordering
  - defer child-view ordering
- Add microbenchmarks for representative pipeline mutations.
- Document the Go-specific `OpList` contract.

Acceptance:

- No phase depends on hidden linked-list behavior.
- Op ordering and slot tests pass deterministically.
- Benchmark results are recorded before any later optimization.

## Phase 2: Reduce Normalized Parity Debt

**Status: Done**

Goal: move parity from "normalized pass" toward exact or explicitly accepted formatting-only differences.

Tasks:

- Classify every current normalizer rule as permanent or temporary:
  - `class-debug-file-path`
  - `class-metadata`
  - `class-metadata-async`
  - `class-debug-info`
  - `listener-function-names`
  - `line-endings`
  - `trim-final-newline`
- Fix temporary compiler/emitter differences where practical.
- Add fixture-level `--fail-normalized` enforcement once a fixture reaches exact parity.
- Keep `domElement*` vs `element*` outside the normalizer permanently.

Acceptance:

- The parity report shows which rules keep each fixture from exact parity.
- At least the first subset of render3 fixtures can pass with `--fail-normalized`.
- No semantic difference is hidden by a broad normalizer.

## Phase 3: Expand Render3 Semantic Corpus

**Status: Done**

Goal: make render3 coverage broad enough to catch Angular template runtime bugs before manual demo testing.

Fixture groups:

- New control flow: `@if`, `@for`, `@switch`, nested `@defer`.
- Forms: validators, async validators, custom value accessors, nested form groups.
- Template contexts: `NgTemplateOutlet`, `let-*`, `$implicit`, projected templates, nested structural directives.
- Queries: content/view query edge cases.
- Host bindings/listeners: directive and component combinations.
- Styles: external styles, encapsulation, `styleUrl` and `styleUrls`.
- i18n minimal path: interpolated text, attributes, and ICU if production scope requires it.

Acceptance:

- New fixtures compare against real `ngtsc`.
- All semantic diffs are either fixed or explicitly marked unsupported with a failing issue fixture.
- Runtime-impacting cases get matching Playwright coverage when applicable.

## Phase 4: Linker Corpus And Partial Evaluator

**Status: Done**

Goal: make the linker generic enough for real Angular and third-party libraries.

Tasks:

- Build a linker corpus runner for:
  - `@angular/common`
  - `@angular/forms`
  - `@angular/router`
  - `@angular/animations`
  - selected PrimeNG FESM files
- Extend partial evaluator support for:
  - same-file const arrays
  - spread arrays and object spreads
  - imported const arrays
  - `forwardRef`
  - conditional expressions
  - simple function return values
  - enum/static property access if encountered in corpus
- Add regression fixtures for every unsupported metadata shape before fixing it.
- Improve linker diagnostics with file/range information.

Acceptance:

- No `Expected array literal` failures on the selected corpus.
- No invalid JavaScript output for esbuild/Vite.
- Unsupported expressions fail explicitly and point to the metadata expression.

## Phase 5: Vite Builder Abstraction

**Status: Done**

Goal: evolve the Vite integration from a wrapper around `go-ngc -p` into a reliable builder path.

Tasks:

- Emit real sourcemaps instead of `map: null`.
- Parse compiler diagnostics into Vite-friendly errors.
- Track dependency graph edges:
  - `templateUrl`
  - `styleUrl` / `styleUrls`
  - standalone imports
  - route/lazy imports where possible
- Replace broad HMR invalidation with component/subgraph invalidation.
- Add separate tests for:
  - compile-only
  - linker-only
  - combined dev
  - combined production build
  - production preview smoke
- Improve compile/link cache keys using dependency graph data, not only project-wide file stats.

Acceptance:

- Editing a template/style recompiles only the affected component path.
- Vite overlay shows actionable compiler diagnostics.
- Production build and dev server use the same compiler/linker semantics.
- No stale output after repeated edits.

## Phase 6: Runtime Smoke Expansion

**Status: Planned**

Goal: automate the browser failures that static parity cannot catch.

Current smoke coverage:

- menu navigation
- input `ngModel`
- checkbox/radio events
- PrimeNG select panel
- table rendering
- tabs
- defer trigger

Next smoke coverage:

- PrimeNG dialog open/close
- datepicker popup and selection
- tree/treeTable expansion
- accordion/panel toggle
- picklist/orderlist movement
- splitter rendering
- router navigation
- production-preview mode

Acceptance:

- Browser console remains clean for all supported smoke paths.
- The suite fails on Angular runtime errors such as:
  - `ExpressionChangedAfterItHasBeenCheckedError`
  - injector bloom assertion failures
  - `templateRef.createEmbeddedViewImpl is not a function`
  - `$event.target` undefined
  - unknown `ngIf/ngFor` binding errors

## Phase 7: Performance And Stability Benchmarks

**Status: Done**

Goal: quantify whether the Go builder is faster, stable, and worth using over ngtsc for supported cases.

Benchmarks:

- Cold compile: Go vs ngtsc.
- Incremental template edit.
- Incremental TypeScript component edit.
- Linker throughput over selected `node_modules`.
- Peak memory.
- Vite dev startup.
- Vite production build.

Tasks:

- Add benchmark scripts that write JSON reports under `.tmp/bench`.
- Keep ngtsc as the baseline for every benchmark.
- Track performance before and after major pipeline/linker changes.

Acceptance:

- Benchmark output is reproducible locally.
- Any major slowdown has a visible report before merge.
- Performance numbers are separated from semantic correctness gates.

## Phase 8: CI And Release Hardening

**Status: Planned**

Goal: make production readiness enforceable outside one local machine.

Tasks:

- Wire `ci.sh` into the project CI runner.
- Ensure CI starts from a clean checkout and installs dependencies deterministically.
- Cache only safe dependencies, not compiler outputs that can hide stale-output bugs.
- Add release/version compatibility checks for the Angular baseline.
- Publish builder usage docs for:
  - compile-only
  - linker-only
  - combined Vite builder
  - known unsupported cases

Acceptance:

- CI fails on semantic parity regression.
- CI fails on runtime smoke console errors.
- CI fails on unsupported linker expressions in the selected corpus.
- A user can run the builder from docs without local hidden state.

## Recommended Execution Order

1. Phase 0: clean and freeze the current baseline.
2. Phase 1: lock `OpList`/IR architecture.
3. Phase 4: expand linker corpus and partial evaluator.
4. Phase 2: reduce normalized parity debt.
5. Phase 3: expand render3 semantic corpus.
6. Phase 5: harden Vite builder behavior.
7. Phase 6: expand runtime smoke.
8. Phase 7: add performance benchmarks.
9. Phase 8: wire clean CI and release docs.

## Stop Conditions

Stop and fix before continuing if any of these appear:

- A semantic parity diff appears in render3 or linker output.
- Linker emits invalid JavaScript.
- Runtime smoke reports any Angular console error.
- A cache hit serves stale compiled output.
- Unsupported metadata is silently ignored.
- A normalizer rule hides an instruction, slot, dependency, or context difference.
