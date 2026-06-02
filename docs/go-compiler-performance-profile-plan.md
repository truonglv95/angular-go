# Go Angular Compiler Performance Profile Plan

Date: 2026-06-02

## Scope

This report profiles the Go compiler by phase:

- TypeScript config/program/source parsing
- Angular analyze/resolve/prepare emit
- template parse
- render3 ingest/transform/emit
- TypeScript emit and post-emit processing
- linker parse/link/print

The profiler is gated behind `GO_NGC_PHASE_PROFILE=1`. JSON output is written with
`GO_NGC_PHASE_PROFILE_OUT=/path/to/profile.json`.

## Measurement Notes

Phase timers are cumulative. Phases invoked across many files or goroutines can exceed
`compile.total`, which is wall-clock time for the whole compilation.

This is especially visible for:

- `ts.parse_source_file`
- `emit.write_file`
- `emit.fs_write`

Use `compile.total`, `ts.emit_total`, and `linker.total` as wall-clock anchors. Use
per-file phase totals to find allocation/contention hotspots.

## Current Results

### Demo App, Global Compilation

Input: `angular-packages/new-demo-app/tsconfig.app.json`

| Phase | Time | Count | Notes |
| --- | ---: | ---: | --- |
| `compile.total` | 79.90ms | 1 | wall-clock compile |
| `compile.program_create` | 59.43ms | 1 | TS program graph creation |
| `ts.parse_source_file` | 1114.28ms | 580 | cumulative parse across source/dependency files |
| `ts.type_checker_create` | 13.08ms | 1 | checker creation |
| `angular.analyze` | 0.90ms | 1 | currently not the bottleneck |
| `angular.resolve` | 7.27ms | 1 | scope/metadata resolution |
| `render3.template_parse` | 8.48ms | 4 | component template parsing |
| `render3.pipeline_transform` | 2.70ms | 2 | render3 pipeline |
| `angular.prepare_emit` | 7.51ms | 1 | AST update before emit |
| `ts.emit_total` | 12.24ms | 1 | wall-clock emit |
| `emit.pure_parse_js` | 0.99ms | 8 | emitted JS reparsing |
| `emit.linker` | 0.16ms | 8 | linker fast path mostly skipped |

### Synthetic Large App, 2000 Components, Global Compilation

Input: `.tmp/bench/large_app/tsconfig.json`

| Phase | Time | Count | Notes |
| --- | ---: | ---: | --- |
| `compile.total` | 821.51ms | 1 | wall-clock compile |
| `compile.program_create` | 107.43ms | 1 | TS program graph creation |
| `ts.parse_source_file` | 1691.43ms | 2273 | cumulative source parsing |
| `ts.type_checker_create` | 15.19ms | 1 | small relative cost |
| `angular.analyze` | 9.58ms | 1 | small relative cost |
| `angular.resolve` | 155.71ms | 1 | scope/metadata resolution cost grows |
| `render3.template_parse` | 232.24ms | 4002 | major Angular-specific cost |
| `render3.pipeline_transform` | 150.03ms | 2001 | major render3 cost |
| `angular.prepare_emit` | 360.64ms | 1 | largest Angular-side wall phase |
| `ts.emit_total` | 548.50ms | 1 | largest wall phase |
| `emit.pure_parse_js` | 268.82ms | 2002 | major avoidable post-emit cost |
| `emit.fs_write` | 686.83ms | 2002 | cumulative, likely IO/cache/lock contention |
| `emit.linker` | 16.97ms | 2002 | not the main compile bottleneck |

### Synthetic Large App, 2000 Components, Local Compilation

Input: `.tmp/bench/large_app/tsconfig.json --compilationMode=local`

| Phase | Time | Count | Notes |
| --- | ---: | ---: | --- |
| `compile.total` | 788.99ms | 1 | only about 4% faster than global in this run |
| `compile.program_create` | 83.86ms | 1 | lower than global |
| `ts.parse_source_file` | 1298.77ms | 2273 | cumulative source parsing |
| `ts.type_checker_create` | 9.78ms | 1 | still present in local mode |
| `angular.analyze` | 1.83ms | 1 | local analyze is cheap |
| `angular.resolve` | 128.66ms | 1 | still significant |
| `render3.template_parse` | 229.54ms | 4002 | unchanged bottleneck |
| `render3.pipeline_transform` | 140.19ms | 2001 | unchanged bottleneck |
| `angular.prepare_emit` | 345.86ms | 1 | still large |
| `ts.emit_total` | 574.43ms | 1 | dominates wall time |
| `emit.pure_parse_js` | 232.90ms | 2002 | major avoidable cost |
| `emit.fs_write` | 385.26ms | 2002 | cumulative |
| `emit.linker` | 13.01ms | 2002 | not the main bottleneck |

### Standalone Linker, PrimeNG Button

Input: `node_modules/primeng/fesm2022/primeng-button.mjs`

| Phase | Time | Count | Notes |
| --- | ---: | ---: | --- |
| `linker.total` | 7.75ms | 1 | wall-clock linker |
| `linker.file_read` | 0.05ms | 1 | negligible |
| `linker.parse_js` | 0.96ms | 1 | JS AST parse |
| `linker.link_ast` | 5.46ms | 1 | main linker cost |
| `render3.template_parse` | 4.15ms | 1 | partial declaration template parse inside linker |
| `render3.pipeline_transform` | 0.67ms | 1 | render3 lowering inside linker |
| `linker.print_js` | 1.32ms | 1 | output printer |

## Findings

1. `angular.analyze` is not the main performance issue anymore.
   On the 2000-component app it is only 9.58ms global and 1.83ms local.

2. Local compilation gives limited wall-clock improvement because the hot path moved to
   `ts.emit_total`, `angular.prepare_emit`, template parsing, render3 transform, and
   post-emit JS parsing.

3. `emit.pure_parse_js` is a high-value optimization target.
   We currently reparse every emitted JS file only to insert `/*@__PURE__*/` comments.
   On the 2000-component app this costs 232-269ms cumulative.

4. `angular.prepare_emit` is now one of the biggest Angular-owned wall phases.
   It updates every source file and triggers most render3 compilation work.

5. `render3.template_parse` and `render3.pipeline_transform` are the biggest template-side
   costs. Together they account for roughly 370-382ms cumulative on the 2000-component app.

6. Standalone linker cost is mostly inside `linker.link_ast`, and that includes render3
   work for partial declarations. JS parse/print are secondary but still visible.

7. `ts.parse_source_file` is large cumulatively but heavily parallelized. It is still worth
   optimizing allocations, dependency parsing, and file cache reuse, but it is not directly
   equal to wall-clock compile time.

## Optimization Plan

### Phase 1: Make Performance Profiling a Stable Tool

- Keep `GO_NGC_PHASE_PROFILE` gated and disabled by default.
- Add a benchmark summary script that runs:
  - demo app compile
  - synthetic large app global compile
  - synthetic large app local compile
  - representative standalone linker files
- Emit a compact Markdown/JSON summary with wall phases and cumulative phases separated.
- Add optional allocation profile capture for large app runs.

Expected value: makes performance regression visible before optimizing further.

### Phase 2: Remove Post-Emit JS Reparse for Pure Annotations

- Stop reparsing emitted JS in `emit.pure_parse_js`.
- Move pure annotation emission into Angular codegen/printer path where define calls are
  originally emitted.
- If printer-level support is hard, add a structured emit marker in generated AST instead
  of scanning printed JS.
- Add golden tests that assert pure comments on:
  - `ɵɵdefineComponent`
  - `ɵɵdefineDirective`
  - `ɵɵdefineNgModule`
  - `ɵɵdefinePipe`
  - `ɵɵdefineInjectable`

Expected value: remove about 230-270ms cumulative on 2000-component app.

### Phase 3: Reduce `angular.prepare_emit` Cost

- Profile `TraitCompiler.UpdateSourceFile` and each decorator handler's compile path.
- Split `prepare_emit` into:
  - trait update
  - component compile
  - directive/pipe/module/injectable compile
  - source-file transform
- Avoid revisiting source files with no Angular traits.
- Cache static metadata that is currently recomputed during update.
- Ensure local mode does not run global-only resolution/update work.

Expected value: this is the largest Angular-owned wall phase after TS emit.

### Phase 4: Optimize Template Parse

- Add allocation profile around `render3.template_parse`.
- Audit `ml_parser`, whitespace removal, ICU/i18n visitors, and source-span creation.
- Pool or reuse short-lived parser structs where safe.
- Avoid duplicate parse of inline templates between analyze and emit.
- Cache external template parse result by file path and content hash.

Expected value: reduce the 229-232ms cumulative template parse cost on 2000 components.

### Phase 5: Optimize Render3 Pipeline Transform

- Finish the `OpList` model decision and remove mixed slice/linked-list debt.
- Benchmark common operations:
  - append
  - insert before/after
  - remove
  - scan by kind
  - repeated phase traversal
- Combine cheap linear phases where it does not hurt correctness or parity with ngtsc.
- Reduce temporary ops/expressions allocated per template.

Expected value: reduce the 140-150ms cumulative transform cost and lower GC pressure.

### Phase 6: Linker Hot Path

- Profile representative packages, not only PrimeNG button:
  - `@angular/router`
  - `primeng/menu`
  - `primeng/dropdown`
  - `primeng/datepicker`
  - `@angular/forms`
- Split `linker.link_ast` into declaration extraction, metadata conversion, template
  compilation, AST rewrite.
- Add a content-hash cache in the Vite linker plugin to avoid relinking unchanged files.
- Keep `NeedsLinking` as a very cheap string scan before AST parsing.
- Investigate printer cost only after `link_ast` is reduced.

Expected value: improves Vite dependency optimization and dev server cold start.

### Phase 7: Local Mode Without Unnecessary Type Checker Work

- Audit why local mode still creates a type checker.
- Replace checker-dependent import/metadata reads with syntax-level import tables where
  local compilation semantics allow it.
- Keep global mode behavior unchanged.
- Add parity tests for local-mode metadata emission.

Expected value: local mode currently improves compile wall time only modestly. This phase
turns local mode into a real fast path.

### Phase 8: Vite Builder-Level Performance

- Separate compiler plugin and linker plugin caches:
  - compiled TS/component output cache
  - linked dependency output cache
- Cache key must include:
  - file content hash
  - `go-ngc` version/build hash
  - tsconfig relevant options
  - Angular/compiler option relevant flags
  - dependency package version for linker
- Avoid invoking `go-ngc --link` per request if an optimized dependency cache exists.
- Measure:
  - Vite startup
  - first page render
  - HMR component edit
  - dependency optimize time
  - production build

Expected value: builder performance will be dominated by cache boundaries, not raw compiler speed.

## Recommended Next Work

Start with Phase 2, then Phase 3.

Reason:

- Phase 2 removes a clearly avoidable cost with low semantic risk.
- Phase 3 attacks the largest Angular-owned wall phase.
- Template parser and render3 pipeline optimization should come after more granular
  sub-phase profiling, otherwise we risk optimizing the wrong allocations.

