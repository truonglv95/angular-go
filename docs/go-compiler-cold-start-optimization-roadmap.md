# Go Compiler Cold Start Optimization Roadmap

Date: 2026-06-02

## Goal

Improve cold compile performance before relying on cache or incremental compilation.

Cache and incremental mode are still important for Vite/HMR and production builder UX,
but they should come after the cold path is strong. Cold compile remains the baseline for:

- first install
- CI
- production build without warm cache
- cache miss
- compiler version bump
- dependency version bump

## Current Baseline

Latest 2000-component synthetic app benchmark:

| Compiler | Time |
| --- | ---: |
| Go global | `680.59ms` |
| Go local | `631.47ms` |
| ngtsc | `6351.74ms` |
| Go global speedup | `9.33x` |
| Go local speedup | `10.06x` |

Latest large-app phase profile:

| Phase | Global | Local |
| --- | ---: | ---: |
| `compile.total` | `665.32ms` | `586.89ms` |
| `ts.emit_total` | `386.53ms` | `396.67ms` |
| `angular.prepare_emit` | `217.28ms` | `206.66ms` |
| `angular.resolve` | `170.86ms` | `125.82ms` |
| `render3.template_parse` | `125.41ms` | `117.80ms` |
| `render3.pipeline_transform` | `123.38ms` | `112.91ms` |
| `ts.type_checker_create` | `9.21ms` | `0ms` |

CPU profile summary:

- `emitJSFile` / `printSourceFile` are now major cumulative costs.
- file writing and `os.OpenFile` are visible in pprof.
- GC/runtime costs are high: `runtime.madvise`, `runtime.scanObject`,
  `runtime.tryDeferToSpanScan`, `runtime.memclrNoHeapPointers`.
- Local mode no longer creates a type checker, but still pays resolve/prepare/emit costs.

## Priority 1: Instrument Emit Path in Detail

### Why

`ts.emit_total` is now the largest wall-clock phase:

| Mode | Cost |
| --- | ---: |
| Global | `386.53ms` |
| Local | `396.67ms` |

Current profile is too coarse. We need to split emit before optimizing it.

### Tasks

- Add phase timers in `internal/compiler/emitter.go`:
  - `emit.js_file`
  - `emit.declaration_file`
  - `emit.run_script_transformers`
  - `emit.print_source_file`
  - `emit.write_text`
  - `emit.write_js`
  - `emit.write_dts`
  - `emit.source_map`
  - `emit.custom_transformers`
- Add write-path timers in VFS:
  - `emit.fs_ensure_dir`
  - `emit.fs_open`
  - `emit.fs_write_bytes`
  - `emit.fs_close`
- Keep profiling gated by `GO_NGC_PHASE_PROFILE`.

### Success Criteria

We can answer precisely how much cold compile time is spent in:

- AST transform
- printer
- JS output write
- declaration output write
- source map output
- filesystem open/write/close

## Priority 2: Optimize Disk Write Path

### Why

The 2000-component benchmark writes about 2002 emitted outputs. pprof shows substantial
time in:

- `WriteFile`
- `writeFileEnsuringDir`
- `os.OpenFile`
- syscall

This scales linearly with component count.

### Tasks

- Cache directory existence during emit so repeated files do not repeatedly ensure the
  same output directory.
- Avoid repeated path normalization where the path is already normalized.
- Avoid writing unchanged content when safe.
  - This is not full incremental compilation.
  - It is a cold-path write optimization for repeated benchmark/build runs.
- Measure `emit-to-memory` vs `emit-to-disk`.
  - Add a temporary benchmark mode or test helper that discards output.
  - Use this to separate compiler CPU from disk IO.
- Check whether declaration files/source maps are being emitted in the benchmark/builder
  mode when they are not needed.

### Success Criteria

- File write cumulative time decreases.
- Cold compile wall-clock decreases on 2000 components.
- Output correctness remains unchanged.

## Priority 3: Optimize Printer and Emit Allocation

### Why

pprof shows `emitJSFile` and `printSourceFile` as major cumulative costs. GC is also high,
which suggests printer/emit allocation is now limiting performance.

### Tasks

- Profile allocations around:
  - printer text writer
  - source map writer
  - comment emission
  - synthetic comments for pure annotations
  - AST transform output
- Reuse printer buffers where safe.
- Pre-size output buffers when source size is known.
- Avoid unnecessary string concatenation in printer hot paths.
- Ensure pure annotation custom transformer does not traverse files without Angular
  define calls if this is still measurable.
- Consider marking Angular-generated define calls at creation time instead of scanning
  source file AST during emit.

### Success Criteria

- Lower GC samples in CPU profile.
- Lower allocation profile for large app emit.
- Lower `emit.print_source_file` time.

## Priority 4: Parallelize `prepare_emit`

### Why

`angular.prepare_emit` is still a large wall phase:

| Mode | Cost |
| --- | ---: |
| Global | `217.28ms` |
| Local | `206.66ms` |

`UpdateSourceFile` is mostly per-file work and should be parallelizable if shared mutable
state is isolated.

### Tasks

- Split `prepare_emit` profile further:
  - `prepare_emit.file_scan`
  - `prepare_emit.update_source_file`
  - `prepare_emit.compile_traits`
  - `prepare_emit.import_manager`
  - `prepare_emit.class_update`
- Audit thread safety:
  - use one `NodeFactory` per goroutine if needed
  - keep `ConstantPool` per file
  - keep `ImportManager` per file
  - do not mutate shared trait state during compile
- Parallelize source-file updates for files with Angular traits.
- Preserve deterministic output order.

### Success Criteria

- `angular.prepare_emit` wall time drops significantly on 2000 components.
- Global and local both improve.
- Render3 and linker parity tests remain stable.

## Priority 5: Reduce Render3 Pipeline Allocation

### Why

Render3 pipeline has improved but still costs:

| Phase | Global | Local |
| --- | ---: | ---: |
| `render3.template_parse` | `125.41ms` | `117.80ms` |
| `render3.pipeline_transform` | `123.38ms` | `112.91ms` |

The remaining cost likely includes allocation/GC pressure.

### Tasks

- Add allocation profile for:
  - template parse
  - ingest
  - pipeline transform
  - emit template function
- Preallocate `OpList` where operation counts are predictable.
- Avoid temporary slices during common phase traversals.
- Remove remaining mixed slice/linked-list patterns.
- Reuse small maps/slices carefully where lifetime is clear.
- Keep phase merge work conservative and parity-driven.

### Success Criteria

- Lower `render3.pipeline_transform` time.
- Lower GC pressure in large app CPU profile.
- No semantic drift against ngtsc parity fixtures.

## Priority 6: Trim Local Resolve Path

### Why

Local mode no longer creates a type checker, but still spends:

| Phase | Cost |
| --- | ---: |
| `angular.resolve` | `125.82ms` |
| `angular.prepare_emit` | `206.66ms` |
| `ts.emit_total` | `396.67ms` |

Local mode should avoid global-style dependency resolution where runtime-resolved imports
are enough.

### Tasks

- Audit local component dependency resolution.
- Use raw `imports` expression for local runtime-resolved dependency emission.
- Skip DtsMetadataReader/scope metadata paths when local semantics do not require them.
- Avoid global-only scope work in local mode.
- Keep global mode unchanged.

### Success Criteria

- `angular.resolve` drops in local mode.
- Local output remains compatible with Angular runtime.
- Existing local/global parity fixtures continue to pass.

## Priority 7: Improve Benchmark Reliability

### Why

Some phase timers are cumulative across goroutines and can exceed wall-clock time. Some
wall-clock runs also have outliers due to filesystem/cache state.

### Tasks

- Add median and p95 to benchmark scripts, not only average.
- Record all runs in Markdown summary.
- Separate:
  - cold clean output
  - warm filesystem cache
  - emit-to-memory
  - emit-to-disk
- Record Go version, Node version, CPU count, and GOGC.
- Add optional `GOMAXPROCS` matrix.

### Success Criteria

- Benchmark results are stable enough for regressions.
- We can distinguish real compiler wins from filesystem/cache noise.

## Deferred: Cache and Incremental

Cache and incremental compilation should come after the cold path work above.

Future builder tasks:

- Vite compiler plugin cache by file content hash.
- Vite linker plugin cache by package file content hash.
- HMR local-mode compilation only for touched components.
- persistent dependency linker cache.

These will improve dev loop and production builder UX, but they should not hide cold
compile bottlenecks.

## Recommended Next Implementation Order

1. Add detailed emit/write profiling.
2. Optimize disk write path and measure emit-to-memory vs emit-to-disk.
3. Optimize printer/emit allocation.
4. Parallelize `prepare_emit`.
5. Reduce render3 pipeline allocation.
6. Trim local resolve path.
7. Improve benchmark script reliability.

