# Go ngtsc Parity Roadmap

This document tracks where the Go Angular compiler/linker currently stands against `ngtsc`, and what remains before it can be treated as a production builder.

Baseline for current parity decisions:

- Angular npm baseline: `@angular/compiler-cli@21.2.15`.
- Local ngtsc command: `node angular-packages/new-demo-app/node_modules/@angular/compiler-cli/bundles/src/bin/ngc.js -p <tsconfig>`.
- Go compiler command: `go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go`.
- Main test command:

```bash
source ~/.gvm/scripts/gvm && gvm use go1.26.3 >/dev/null
go test ./angular-packages/compiler_cli/integration_tests ./angular-packages/compiler/template/pipeline/phases
```

## Status Legend

- **Done**: implementation and tests are in place for the currently scoped behavior.
- **Partial**: important cases work, but coverage is not broad enough to call it production-ready.
- **Debt**: accepted normalizations or architectural mismatch still hide real future risk.
- **Next**: the next concrete work needed to move the phase forward.

## Current Verified State

The render3 golden suite has grown beyond the original basic fixtures. It now covers:

- `parity_ng_if`
- `parity_ng_for`
- `parity_ng_template_outlet`
- `parity_template_refs`
- `parity_nested_templates`
- `parity_nested_views`
- `parity_listeners`
- `parity_forms`
- `parity_forms_form_control`
- `parity_forms_form_control_name`
- `parity_pipes`
- `parity_animations`
- `parity_defer`
- `parity_defer_triggers`
- `parity_defer_blocks`
- `parity_defer_nested`

Semantic fixes already landed:

- Standalone directive/pipe dependencies are filtered by actual template usage.
- Direct standalone imports are preferred for `dependencies`, for example `NgIf`, `NgFor`, `NgTemplateOutlet`, and local pipes.
- Pipe binding emits `ɵɵpipeBindX(slot, varOffset, ...)`.
- Pipe creation uses one declaration slot.
- Structural template consts no longer include empty marker values such as `[4, "ngFor", "", ...]`.
- Component `animations` are passed into render3 metadata and emit `data.animation`.
- Forms control instructions are aligned to Angular 21.2.15 for the covered fixtures: normal Forms usage does not emit `ɵɵcontrolCreate/ɵɵcontrol`; only `[formField]` is currently eligible in the pipeline rule.
- Deferred component imports are removed from eager imports and emitted through defer dependency functions.
- Deferred component deps force full template mode so defer child views do not fall into incorrect DomOnly output.
- `ɵɵtemplate` create calls are chainable like ngtsc.
- Template local refs, embedded view contexts, and `NgTemplateOutlet` behavior have enough parity to remove the previous PrimeNG dropdown/menu `TemplateRef` runtime failure class in the tested paths.
- The linker has targeted parity coverage for partial declarations, host listeners with `$event`, queries, animation names, undefined literals, NgModule metadata, facade options, deps variants, local refs, forward refs, projected template refs, and nested template context depth.
- The Vite integration has been split into two plugin entry points: `angularGoCompile(...)` and `angularGoLinker(...)`, with a combined default wrapper.

Accepted golden normalizations for now:

- Class metadata/debug-info formatting.
- Listener function generated names.
- Dependency array order.
- `ɵɵdomElement*` vs `ɵɵelement*` aliases where runtime behavior is equivalent.

These normalizations are useful for reducing noise, but they must not become the final definition of parity.

## Phase 1: Lock Baseline Comparison

**Status: Done for scoped parity fixtures**

Goal: every parity decision is made against real `ngc`, not against stale goldens.

Done:

- Current baseline is explicitly `@angular/compiler-cli@21.2.15`.
- Several previous fixes were made by comparing Go output with ngtsc output for forms, defer, pipes, TemplateRef, and linker cases.
- `compare_linker_output.mjs` exists for linker output comparison and can now optionally write the ngtsc-linked output.
- Added `angular-packages/compiler_cli/tools/parity_compare`.
- Added single-fixture and `--all` modes for render3 parity fixtures.
- Added linker mode for the selected Angular/PrimeNG FESM corpus.
- Added deterministic output under `.tmp/ngtsc-parity`.
- Added `report.json`, raw output copies, normalized output copies, raw diff, normalized diff, and command stdout/stderr files.
- Added `--strict` and `--fail-normalized` gates.
- Shared the golden JS normalizer through `angular-packages/compiler_cli/integration_tests/parity`.

Debt:

- All current render3 parity fixtures classify as `normalized`, not `exact`.
- All selected linker corpus files classify as `normalized`, not `exact`.
- `--update-golden` is intentionally not implemented yet; it should wait until Phase 2 normalizer tests are in place.
- Non-parity legacy render3 fixtures are excluded from default `--all`. They can be audited with `--include-legacy`, but current output path differences expose existing fixture/golden debt.

Next:

- Use the parity runner as the source of truth before changing goldens:

```bash
go run ./angular-packages/compiler_cli/tools/parity_compare \
  --fixture parity_ng_template_outlet \
  --kind render3 \
  --angular-version 21.2.15 \
  --out .tmp/ngtsc-parity
```

- Render3 parity corpus:

```bash
go run ./angular-packages/compiler_cli/tools/parity_compare --kind render3 --all
go run ./angular-packages/compiler_cli/tools/parity_compare --kind render3 --all --strict
```

- Legacy render3 fixture audit:

```bash
go run ./angular-packages/compiler_cli/tools/parity_compare --kind render3 --all --include-legacy
```

- Linker corpus:

```bash
go run ./angular-packages/compiler_cli/tools/parity_compare --kind linker --all
go run ./angular-packages/compiler_cli/tools/parity_compare --kind linker --all --strict
```

- For render3 fixtures, the runner now:
  - locate `testdata/render3_ir/<fixture>/project/tsconfig.json`
  - run Go compiler into a temp output directory
  - run ngtsc into a separate temp output directory
  - collect emitted `.js` files from both outputs
  - pair files by relative path
  - compare raw output first
  - compare normalized output second
  - classify the result
- For linker fixtures, the runner now:
  - invoke the Go linker with `go-ngc --link <input.mjs>`
  - invoke the ngtsc linker comparison path through `compare_linker_output.mjs`
  - store both raw outputs and the comparison summary
- Output is written under a deterministic temp layout:

```text
.tmp/ngtsc-parity/
  render3/
    parity_ng_template_outlet/
      go/
      ngtsc/
      normalized/
      diff.raw.patch
      diff.normalized.patch
      report.json
  linker/
    <case>/
      go.mjs
      ngtsc.mjs
      diff.raw.patch
      report.json
```

- Classify each compared file into exactly one category:
  - `exact`: raw Go output equals raw ngtsc output
  - `normalized`: raw output differs, but accepted normalizer makes it equal
  - `semantic`: normalized output still differs
  - `missing-go-output`: ngtsc emitted a file that Go did not emit
  - `missing-ngtsc-output`: Go emitted a file that ngtsc did not emit
  - `compile-failed`: either compiler failed before output comparison
- Make the report machine-readable:

```json
{
  "baseline": {
    "angular": "21.2.15",
    "goCompiler": "./go-ngc"
  },
  "kind": "render3",
  "fixture": "parity_ng_template_outlet",
  "status": "normalized",
  "files": [
    {
      "path": "src/app/app.component.js",
      "status": "normalized",
      "normalizations": ["class-metadata", "listener-names"]
    }
  ]
}
```

- Keep the normalizer boundary explicit. The parity runner may use normalizers only for reporting and classification. It must always preserve raw output and raw diff.
- Add a `--strict` mode for CI:
  - fail on `semantic`
  - fail on missing output
  - fail on compiler failures
  - optionally fail on `normalized` once a phase declares that normalization retired
- Add `--update-golden` later, only after Phase 2 has dedicated normalizer tests. That mode must require an existing `exact` or approved `normalized` classification before writing fixture goldens.

Fixture scope for the first implementation:

- `parity_ng_if`
- `parity_ng_for`
- `parity_ng_template_outlet`
- `parity_template_refs`
- `parity_nested_templates`
- `parity_nested_views`
- `parity_listeners`
- `parity_forms`
- `parity_forms_form_control`
- `parity_forms_form_control_name`
- `parity_pipes`
- `parity_animations`
- `parity_defer`
- `parity_defer_triggers`
- `parity_defer_blocks`
- `parity_defer_nested`

Implementation result:

1. Render3 runner for one fixture and one emitted `.js` file: done.
2. Multi-file pairing and missing-output detection: done.
3. Shared golden normalizer for report-only classification: done.
4. `report.json` plus raw and normalized patch files: done.
5. `--all` fixture discovery for parity fixtures: done.
6. Linker mode wrapping the existing linker comparison flow: done.
7. `--strict` for CI-style failure behavior: done.
8. Optional `--update-golden`: deferred to Phase 2.

Acceptance criteria status:

- `parity_forms`, `parity_defer`, `parity_pipes`, `parity_ng_if`, `parity_ng_for`, and `parity_ng_template_outlet` can be compared with one command: met.
- The current render3 parity fixture list can be compared with `--all`: met.
- Raw Go and ngtsc outputs are always persisted for inspection: met.
- Every file is classified as `exact`, `normalized`, `semantic`, missing output, or compile failure: met.
- A semantic diff cannot be hidden by the normalizer because raw diff and normalized diff are both persisted: met.
- CI can consume `report.json` and fail only on semantic or execution failures: met for the runner; formal CI wiring remains Phase 10.

## Phase 2: Harden Golden Test Infrastructure

**Status: Done for current normalizer scope**

Goal: golden failures should be signal, not formatting noise.

Done:

- `normalizeGoldenJS` has been moved out of `render3_ir_golden_test.go` into `angular-packages/compiler_cli/integration_tests/parity`.
- The normalizer now exposes named rules through `NormalizeGoldenJSWithRules`.
- The parity runner now records concrete rule names in `report.json` instead of the previous coarse `golden-js` marker.
- Unit tests cover each implemented normalization rule:
  - `line-endings`
  - `class-debug-file-path`
  - `listener-function-names`
  - `class-metadata`
  - `class-metadata-async`
  - `class-debug-info`
  - `trim-final-newline`
- Unit tests verify that a semantic `pipeBind` var-offset difference is not hidden by the normalizer.
- Golden fixture path handling has already been fixed so failures are not polluted by incorrect paths.

Debt:

- Current render3 parity reports still classify as `normalized`, not `exact`, mostly because class metadata/debug-info formatting still differs.
- Dependency array ordering and `domElement*` aliases remain broader parity debt, but they should not be widened casually in the normalizer without rule-specific tests and explicit approval.
- `--update-golden` is still deferred until we decide whether golden updates should accept `normalized` output or require `exact` output fixture by fixture.

Next:

- Add rule-level summary counts to the parity runner output so the CLI summary can show which normalizations are still keeping each fixture from `exact`.
- Decide per rule whether it is permanently accepted formatting noise or a temporary compiler debt:
  - `class-debug-file-path`: likely permanent environment normalization.
  - `class-metadata` / `class-metadata-async`: probably formatting/declaration emit debt.
  - `class-debug-info`: likely formatting/declaration emit debt.
  - `listener-function-names`: likely acceptable generated-name noise unless Angular relies on function names.
  - `trim-final-newline` / `line-endings`: permanent formatting normalization.
- Only after that decision, add guarded `--update-golden`.

Acceptance criteria status:

- Normalizer behavior is covered by tests: met.
- Parity reports list concrete normalization rules: met.
- Golden tests still fail for semantic differences such as `pipeBind` var offset: covered by unit test.
- Broader semantic fail coverage for slot, `decls`, `vars`, `defer`, `templateRefExtractor`, context, and dependency identity remains part of future fixture expansion.

## Phase 3: Remove `domElement*` Normalization Debt

**Status: Verified for scoped parity fixtures**

Goal: align instruction selection with Angular 21.2.15 instead of normalizing it away.

Done:

- Runtime-impacting issues around pipes, listeners, defer, and structural views have been fixed enough for current golden coverage.
- The current normalizer does not rewrite `ɵɵdomElement*` or `ɵɵelement*`.
- Added a normalizer unit guard proving `ɵɵdomElementStart(...)` and `ɵɵelementStart(...)` remain different after normalization.
- Current scoped parity fixtures compare `domElement*`/`element*` as real semantic output, not accepted normalized output.
- `parity_pipes` and `parity_template_refs` show matching instruction families between Go and ngtsc in the raw compare output.
- The parity runner now prints rule-level normalization counts, which makes it clear that current render3 `normalized` status is metadata/debug/listener-name noise, not `domElement*` masking.

Debt:

- Broader non-parity and future fixtures may still expose real `TemplateCompilationMode_DomOnly` vs full-mode instruction-selection gaps.
- Legacy fixtures should be audited with `--include-legacy` after their output path/rootDir debt is cleaned up.

Next:

- Keep `domElement*` out of the normalizer permanently.
- Add future fixtures for any newly discovered full-mode/DomOnly transition bug.
- When legacy fixture paths are normalized, run:

```bash
go run ./angular-packages/compiler_cli/tools/parity_compare --kind render3 --all --include-legacy --strict
```

Acceptance criteria status:

- `parity_pipes` matches `ngc` without `domElement*` normalization: met for scoped fixture.
- Defer fixtures emit the same instruction family as `ngc`: met for current scoped fixtures.
- Normalizer cannot hide `domElement*` vs `element*`: met by unit guard.
- Vite runtime remains a Phase 9 smoke-suite responsibility.

## Phase 4: Defer Block Parity

**Status: Partial, expanded**

Goal: defer output should match ngtsc for dependency resolution, block shape, slots, triggers, and nested views.

Done:

- Fixtures exist for basic defer, triggers, placeholder/loading/error blocks, and nested defer.
- Added `parity_defer_alias_barrel` to cover aliased standalone imports through a barrel re-export.
- Deferred component deps are emitted through dependency resolver functions.
- Deferred component imports are removed from eager imports.
- Deferred deps force full template compilation mode.
- Deferred alias/barrel deps now preserve the import surface used by the component (`import("./deferred").then(m => m.HeavyComponent)`) instead of falling back to the declaration file.
- Deferred alias/barrel eager import removal now uses the local alias for filtering/removal while preserving the exported symbol for dynamic import.
- Explicit defer trigger targets inside placeholder views now resolve to the placeholder slot shape expected by ngtsc.

Debt:

- Defer output is covered by the Phase 1 parity runner, but not by browser-trigger runtime smoke yet.
- Complex import shapes still need more coverage: type-only imports, namespace imports, default exports, same-symbol different-module cases, and external package exports.
- Hydration/prefetch/timer/viewport edge cases are not yet broad enough.

Next:

- Add fixture cases for type-only/default/namespace deferred imports.
- Add explicit ngtsc compare for more `ɵɵdeferPrefetch*` and hydration trigger forms.
- Add runtime smoke for trigger activation.

Acceptance criteria:

- No eager import remains for deferred-only component deps: met for current direct and alias/barrel fixtures.
- Placeholder/loading/error slots match ngtsc for covered cases: met for current fixtures.
- Nested defer does not corrupt view indexes or resolver functions: met for current fixture.
- Alias/barrel deferred imports preserve the component import surface: met.

## Phase 5: Forms And Control Directive Parity

**Status: Partial, expanded**

Goal: Forms output should match Angular 21.2.15 and be ready for future Angular changes.

Done:

- Fixtures exist for `[(ngModel)]`, `[formControl]`, and `formControlName`.
- Added a `[formField]` fixture with a standalone directive so both Go and ngtsc compile the binding.
- Current Angular 21.2.15 behavior is reflected for those fixtures: normal forms controls do not emit `ɵɵcontrolCreate/ɵɵcontrol`.
- `[formField]` emits `ɵɵcontrolCreate/ɵɵcontrol` and matches ngtsc semantically.
- Host listener `$event` handling has been fixed in linker paths so value accessors receive the real event.
- The Go pipeline has an explicit `formField` branch for control instructions.
- Directive input/output literal-map keys now emit identifier-safe keys unquoted, matching ngtsc more closely and reducing non-semantic diffs.

Debt:

- Forms dependency arrays are still normalized for ordering.
- Future Angular control-instruction behavior needs version/capability gating instead of assumptions.

Next:

- Add negative fixtures for validators and unused value accessors.
- Add raw ngtsc compare for dependency identity/order before removing dependency-order normalization.

Acceptance criteria:

- `[(ngModel)]`, `[formControl]`, and `formControlName` do not emit `ɵɵcontrolCreate/ɵɵcontrol` for Angular 21.2.15: met for current fixtures.
- `[formField]` does emit control instructions: met.
- Dependencies are complete and do not include unused validators.

## Phase 6: TemplateRef And Embedded View Context Parity

**Status: Partial, expanded**

Goal: remove the class of runtime errors around `TemplateRef`, `NgTemplateOutlet`, PrimeNG overlays, and nested templates.

Done:

- Fixtures exist for `NgTemplateOutlet`, template refs, nested templates, nested views, and listener cases.
- Added `parity_template_outlet_structural_refs`, combining `NgTemplateOutlet`, `$implicit`/named context values, `NgIf`, DOM local refs, and event listeners inside an instantiated template.
- Linker tests cover local refs, projected template local refs before attrs, and ancestor context depth in nested templates.
- The implementation has been moved closer to ngtsc behavior instead of preserving the earlier workaround path.
- The previous chain of runtime issues around `templateRef.createEmbeddedViewImpl`, `ctx.model`, and `ctx_r1.menuitemId` has driven targeted parity fixes.

Debt:

- This is still not "complete" until runtime smoke tests cover PrimeNG dropdown/menu/table/datepicker automatically.
- Context resolution needs broader coverage for `let-*`, `$implicit`, `nextContext`, projected templates, and nested control-flow combinations.
- Any remaining workaround should be removed once phase ordering fully matches ngtsc.

Next:

- Add more golden fixtures that combine `NgTemplateOutlet`, structural directives, local refs, and nested embedded views.
- Add browser smoke assertions for PrimeNG dropdown/menu/table/datepicker.
- Compare local-ref indexes and template ref extractor slots against ngtsc output.

Acceptance criteria:

- No `templateRef.createEmbeddedViewImpl is not a function` in automated smoke tests.
- `ɵɵreference(...)` indexes match `ngc` for covered fixtures.
- `TemplateRef` values are not confused with DOM element refs.
- `NgTemplateOutlet` plus structural directives and DOM local refs has no semantic parity diff for the covered fixture.

## Phase 7: Linker Parity For Libraries

**Status: Partial**

Goal: generically link Angular and third-party partial-compiled libraries.

Done:

- Linker has targeted tests for:
  - minimal directives
  - host listeners and `$event`
  - content/view queries
  - animation host/template names
  - `undefined` literals
  - NgModule metadata
  - component facade options
  - factory deps variants
  - local refs
  - forward refs
  - projected template local refs before attrs
  - ancestor context depth in nested templates
  - panel-like known gaps
- Partial linker now resolves same-file const array metadata such as `const CMP_DEPS = [...]` followed by `dependencies: CMP_DEPS`.
- Linker golden comparison is wired through `compare_linker_output.mjs`.
- Previous failures such as `Expected array literal` and corrupted `ɵɵtemplate` output were addressed for the seen packages.
- Vite dependency optimization is configured to force ASCII escaping to avoid invalid non-ASCII token output in esbuild.

Debt:

- This is not full library-corpus parity yet.
- PrimeNG and Angular package coverage is still targeted, not exhaustive.
- Partial evaluator behavior is still the biggest open linker risk: spread arrays, conditional expressions, imported consts, forward refs, and complex metadata expressions need broader coverage.
- Same-file const arrays are supported, but spread arrays and imported const arrays are still open.

Next:

- Build a corpus runner for `@angular/common`, `@angular/forms`, `@angular/router`, and selected PrimeNG FESM files.
- Classify each file as exact, normalized, supported semantic diff, or unsupported.
- Add regression fixtures for every unsupported expression shape found in the corpus.
- Extend partial evaluator coverage for spread arrays and imported const arrays.

Acceptance criteria:

- No invalid token output such as corrupted `ɵɵtemplate`.
- No `Expected array literal` failures on the selected Angular/PrimeNG corpus.
- PrimeNG linked output runs in Vite smoke app.

## Phase 8: Vite Builder Integration

**Status: Partial, hardened**

Goal: provide a builder path that can compile app code and link dependencies independently.

Done:

- The plugin is split into:
  - `angularGoCompile(...)` for app/source compilation.
  - `angularGoLinker(...)` for partial declaration linking.
  - default `angularGo(...)` wrapper for combined usage.
- Linker cache keys account for input file mtime/size, in-memory code hash, and compiler binary mtime/size salt.
- `optimizeDeps.force` is enabled by default while linker output is still evolving.
- Esbuild charset is forced to ASCII by default to avoid invalid token output in optimized dependencies.
- Plugin return types were loosened to avoid Vite type identity conflicts across package-local `node_modules`.
- Compile cache keys now include the `go-ngc` binary salt, compile args, Angular core/compiler-cli versions, and project input stats for TS/templates/styles/config files.
- Linker cache keys now include compiler and Angular package salt, not only input file/code identity.
- Compile plugin registers project inputs with Vite watch mode and skips unchanged compiles when output already exists.
- Missing emitted JS during module load forces a real recompile instead of trusting an unchanged project cache key.
- Compiler failures are surfaced through a dedicated `GoNgcError` with stdout/stderr included for Vite overlay/log output.
- Production Vite build has been verified with Node 22.12.0 using `vite.config.ts` after the compile-cache hardening.

Debt:

- Compile plugin still maps `.ts` to emitted `.js` from `out-tsc/app`; it is not yet a full production builder abstraction.
- Sourcemaps are returned as `null`.
- Diagnostics include compiler stdout/stderr but are not yet parsed into rich Vite source locations.
- HMR uses broad invalidation/full reload instead of precise module invalidation.
- Cache keys include project input stats, but not a semantic imported standalone dependency graph.
- Demo folder currently has a stale `vite.config.js`; production build verification must use `--config vite.config.ts` unless that stale JS file is removed or regenerated.
- Build output is verified, but dev HMR is still manual and should become an automated smoke path before production builder usage.

Next:

- Add dependency invalidation for `templateUrl`, `styleUrl`, and imported standalone deps.
- Add sourcemap support.
- Format compiler diagnostics into Vite-friendly errors.
- Add separate compile-only, linker-only, combined dev, and production build tests.
- Remove or stop generating stale `vite.config.js` in the demo app.

Acceptance criteria:

- Compile-only mode works.
- Linker-only mode works.
- Combined mode works.
- Vite dev server handles edits without stale compiled output.
- Vite production build succeeds with Node 22.12.0 and `vite.config.ts`.

## Phase 9: Runtime Smoke Suite

**Status: Done for baseline automation**

Goal: catch browser/runtime failures that static golden tests cannot see.

Done:

- Manual Vite testing has exposed and validated fixes for checkbox/input events, unknown `ngIf`, PrimeNG menu/dropdown, TemplateRef, and linker output corruption.
- Added Playwright smoke coverage for menu navigation, input `ngModel`, checkbox/radio events, PrimeNG select panel rendering, table rendering, tabs, and defer trigger loading.
- Playwright web server now launches Vite with `--config vite.config.ts`, so the smoke suite does not accidentally load the stale demo `vite.config.js`.
- Runtime smoke currently passes with Node 22.12.0:

```bash
cd angular-packages/new-demo-app
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH npx playwright test
```

Debt:

- Console-clean checks are automated for the covered paths, but the matrix is still narrow.
- The smoke suite does not yet cover all large PrimeNG surfaces in the demo, such as dialog, datepicker, tree, advanced panel/picklist/orderlist/treetable/splitter interactions.
- The browser smoke is not yet wired into a formal CI command.

Next:

- Expand smoke interactions for datepicker, dialog, tree, advanced panel, picklist, orderlist, treetable, splitter, and router.
- Add a CI-facing script that rebuilds `go-ngc`, builds the Vite plugin, and then runs the Playwright smoke suite.
- Consider adding a production-preview smoke path in addition to dev-server smoke.

Acceptance criteria:

- No `ExpressionChangedAfterItHasBeenCheckedError`.
- No injector bloom assertion failure.
- No `templateRef.createEmbeddedViewImpl` failure.
- No `$event.target` undefined failure.
- No unknown `ngIf/ngFor` binding errors.

## Phase 10: CI Gate

**Status: Done**

Goal: prevent regression before using this as a production builder.

Implemented gate:

```bash
PATH=/Users/truong/.nvm/versions/node/v22.12.0/bin:$PATH ./ci.sh
```

The script currently runs:

- Go unit/integration checks for template pipeline phases, render3 integration tests, and linker tests.
- `go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go`.
- Render3 parity strict for all parity fixtures.
- Linker parity strict for the selected Angular/PrimeNG corpus.
- Vite plugin TypeScript build.
- New demo app Vite production build through `vite.config.ts`.
- Playwright runtime smoke.

Latest local result:

- `render3`: `exact=0 normalized=19 semantic=0 missing-go=0 missing-ngtsc=0 compile-failed=0`.
- `linker`: `exact=0 normalized=8 semantic=0 missing-go=0 missing-ngtsc=0 compile-failed=0`.
- Vite production build: passed with only chunk-size warning.
- Playwright smoke: `6 passed`.

Debt:

- The gate is a local script; it is not yet wired into GitHub Actions or another CI runner.
- The gate still accepts normalized parity. Exact parity remains a later hardening target.
- Browser smoke coverage is baseline only and should expand with every newly supported runtime surface.

Acceptance criteria:

- CI fails on semantic parity regression: done in `ci.sh` through `--strict`.
- CI does not fail on accepted formatting-only differences: done through normalized classification.
- `go-ngc` can be rebuilt from source and used by the Vite plugin: done.
- Vite builder smoke passes in CI: done locally through `ci.sh`; pending remote CI wiring.

## Recommended Next Priority

1. Finish Phase 2 first so all later failures are trustworthy.
2. Remove Phase 3 `domElement*` normalization debt, because it hides real pipeline-mode divergence.
3. Expand Phase 4 defer and Phase 7 linker corpus together, because libraries often combine partial linking with deferred/lazy shapes.
4. Harden Phase 8 Vite builder integration only after compile/link output is stable enough to cache safely.
5. Add Phase 9 runtime smoke before calling the builder production-ready.
6. Convert Phase 10 into a CI gate once golden, linker, builder, and browser smoke all run deterministically.

## Non-Negotiable Semantic Gates

Before production builder usage, these must be true without relying on broad normalization:

- No eager imports for defer-only dependencies.
- Pipe create/bind slot semantics match ngtsc.
- Forms control instructions match the selected Angular baseline.
- TemplateRef/local-ref slots are stable across nested embedded views.
- Linker output is always valid JavaScript for esbuild and Vite.
- Partial evaluator failures are explicit and actionable, not silent wrong output.
- Runtime smoke has zero Angular console errors for the supported demo matrix.
