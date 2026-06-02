# Go Compiler Performance Optimization Plan

Date: 2026-06-02

This plan is based on the phase profile in
`docs/go-compiler-performance-profile-plan.md`.

Goal: improve performance for both global and local compilation without introducing
short-term workarounds or diverging from ngtsc semantics.

## Priority Order

1. Remove post-emit JS reparse for pure annotations.
2. Split and optimize `angular.prepare_emit`.
3. Optimize `render3.template_parse`.
4. Optimize `render3.pipeline_transform` and settle `OpList`.
5. Make local mode a real no-checker fast path.

## 1. Remove `emit.pure_parse_js`

### Why First

Current large-app profile:

| Mode | Cost |
| --- | ---: |
| Global | `268.82ms` cumulative |
| Local | `232.90ms` cumulative |

This cost affects both modes and is mostly avoidable. The compiler currently emits JS,
then reparses every emitted JS file only to insert `/*@__PURE__*/` before Angular define
calls.

### Best Fix

- Move pure annotation generation into the Angular codegen/printer path.
- Mark these calls as pure when they are created, not after JS has been printed:
  - `ɵɵdefineComponent`
  - `ɵɵdefineDirective`
  - `ɵɵdefineNgModule`
  - `ɵɵdefinePipe`
  - `ɵɵdefineInjectable`
- Prefer structured AST/printer support over string scanning.
- Keep linker behavior unchanged.

### Tests

- Golden tests comparing output shape with ngtsc.
- Cases for component, directive, module, pipe, injectable.
- Ensure minifiers still see the pure comment before the call expression.

### Expected Impact

High impact for both global and local compilation. This should reduce unnecessary
post-emit work and also lower allocation/GC pressure.

## 2. Split and Optimize `angular.prepare_emit`

### Why Second

Current large-app profile:

| Mode | Cost |
| --- | ---: |
| Global | `360.64ms` wall phase |
| Local | `345.86ms` wall phase |

This is the largest Angular-owned wall phase. It is nearly the same in global and local
mode, so improving it benefits both.

### First Step: Add Sub-Phase Profiling

Split `angular.prepare_emit` into:

- `prepare_emit.update_source_file`
- `prepare_emit.component_compile`
- `prepare_emit.directive_compile`
- `prepare_emit.pipe_compile`
- `prepare_emit.ngmodule_compile`
- `prepare_emit.injectable_compile`
- `prepare_emit.source_transform`

### Best Fixes

- Skip source files that have no Angular traits.
- Cache static metadata already computed during analyze/resolve.
- Avoid recomputing directive/component metadata during update.
- Ensure local mode avoids global-only scope and metadata work.
- Keep AST updates deterministic and compatible with ngtsc output.

### Tests

- Existing integration/golden tests.
- Dedicated tests for mixed files:
  - files with no Angular decorators
  - files with multiple Angular classes
  - standalone components
  - NgModule-declared components

### Expected Impact

High impact for both modes. This is the best compiler-owned wall-clock target after
removing pure JS reparse.

## 3. Optimize `render3.template_parse`

### Why Third

Current large-app profile:

| Mode | Cost |
| --- | ---: |
| Global | `232.24ms` cumulative |
| Local | `229.54ms` cumulative |

This cost scales with component count and is effectively identical in global/local mode.

### Best Fixes

- Verify whether inline templates are parsed more than once across analyze/emit.
- Cache template parse result by:
  - inline template content hash
  - external template file path + content hash
  - relevant parser options
- Add allocation profile around:
  - `ml_parser`
  - whitespace removal
  - ICU/i18n visitors
  - source span creation
- Reduce short-lived allocations before changing parser behavior.

### Tests

- Golden tests for:
  - control flow
  - nested templates
  - content projection
  - i18n markers
  - ICU
  - external templates
- Cache invalidation tests for external template content changes.

### Expected Impact

Medium-to-high impact, especially for component-heavy apps. This should improve global
and local almost equally.

## 4. Optimize `render3.pipeline_transform` and Settle `OpList`

### Why Fourth

Current large-app profile:

| Mode | Cost |
| --- | ---: |
| Global | `150.03ms` cumulative |
| Local | `140.19ms` cumulative |

The codebase still has debt around mixed slice and linked-list `OpList` models. This is
a correctness and performance risk.

### Best Fixes

- Decide one primary `OpList` representation based on benchmark data.
- Benchmark real operations:
  - append
  - scan by kind
  - insert before/after
  - remove
  - repeated phase traversal
- Prefer a slice-first model unless benchmarked workloads prove linked-list wins.
- Keep linked structures only where mutation density clearly requires them.
- Merge simple linear phases only when parity and debuggability remain clear.
- Reduce temporary operation/expression allocation.

### Tests

- Existing render3 pipeline phase tests.
- Golden output tests against ngtsc for:
  - embedded views
  - listener generation
  - `TemplateRef`
  - forms host listeners
  - defer blocks
  - PrimeNG-style templates

### Expected Impact

Medium impact for both modes. Main value is lower cumulative transform time and lower GC
pressure.

## 5. Local Mode No-Checker Fast Path

### Why Fifth

Current large-app profile:

| Phase | Global | Local |
| --- | ---: | ---: |
| `ts.type_checker_create` | `15.19ms` | `9.78ms` |
| `angular.analyze` | `9.58ms` | `1.83ms` |
| `compile.total` | `821.51ms` | `788.99ms` |

Local mode already reduces analyze cost, but total compile time improves only modestly.
The bigger costs are emit, prepare emit, template parse, and render3 transform. Therefore
local no-checker work should come after the shared bottlenecks above.

### Best Fixes

- Audit why local compilation still creates a type checker.
- Replace checker-dependent import and metadata reads with syntax-level import tables
  where local compilation semantics allow it.
- Keep global mode unchanged.
- Avoid weakening semantic validation in global mode.
- Make local mode output deterministic and comparable with ngtsc local compilation.

### Tests

- Local-mode parity fixtures.
- Imports and aliases.
- Standalone imports.
- NgModule metadata.
- Directive/pipe dependencies.
- Re-exported symbols where local mode can legally support them.

### Expected Impact

Lower immediate impact than steps 1-4, but important for a production builder because it
creates a clearer fast path for Vite/HMR and isolated file transforms.

## Recommended Implementation Sequence

### Phase A: Quick High-ROI Optimization

- Implement step 1.
- Verify with golden tests and large-app profile.
- Expected profile improvement should be visible in both global and local mode.

### Phase B: Compiler Wall-Clock Optimization

- Add sub-phase profiling for step 2.
- Optimize skip/caching behavior in `prepare_emit`.
- Reprofile global/local large app.

### Phase C: Template Scale Optimization

- Implement template parse cache and allocation reductions.
- Reprofile component-heavy app.

### Phase D: Render3 Pipeline Cleanup

- Benchmark and settle `OpList`.
- Optimize transform traversal/allocation.
- Keep ngtsc parity tests strict.

### Phase E: Local Builder Fast Path

- Remove unnecessary checker dependency from local mode.
- Integrate with Vite compile plugin cache.
- Measure HMR and isolated component edits.

## Success Metrics

For the 2000-component synthetic app:

- Global compile should improve first by removing `emit.pure_parse_js`.
- Local compile should improve by a similar amount.
- `angular.prepare_emit` should become measurable by sub-phase, then reduced.
- Template parse and pipeline transform cumulative cost should trend down without golden
  drift.

For Vite builder usage:

- Cold startup should rely on dependency linker cache.
- Component edits should avoid full program work where local semantics allow it.
- Linker and compiler plugin caches must have separate cache keys.

