# Go NGC Incremental + Semantic Graph Roadmap

Date: 2026-06-05

## Current State

The current Go compiler already has a useful daemon-backed builder path, but the `ngtsc/incremental` package is still mostly a scaffold.

Implemented today:

- `Fresh()` creates a minimal `IncrementalCompilation`.
- `Incremental()` carries a previous `IncrementalState`.
- `RecordSuccessfulEmit()` records emitted source files.
- `SafeToSkipEmit()` only checks source file version equality and prior emitted state.
- One basic test exists for changed file version invalidating emit skip.

Still stubbed with `panic("unimplemented")`:

- `FileDependencyGraph` in `dependency_tracking.go`
- `IncrementalBuildStrategy` implementations in `strategy.go`
- `SemanticSymbol` comparison methods in `semantic_graph/api.go`
- `SemanticDepGraph` and `SemanticDepGraphUpdater` in `semantic_graph/graph.go`
- semantic equality helpers in `semantic_graph/util.go`
- type parameter extraction/equality in `semantic_graph/type_parameters.go`

This means the daemon can avoid process startup and reuse some in-memory state, but it cannot yet make ngtsc-grade decisions such as:

- source file unchanged, skip analysis;
- resource-only change, re-analyze only affected components;
- public API changed, invalidate dependents;
- private implementation changed, emit only local file;
- template/type-check API changed, invalidate TCB consumers;
- unchanged semantic graph, safely reuse analysis/type-check/emit results.

## Goal

Implement an incremental engine that is generic, ngtsc-aligned, and useful for both:

- local Vite dev server rebuild/HMR;
- global production builder rebuilds.

The first production target is not full Angular CLI parity. The first target is a safe incremental builder that never skips work incorrectly and then becomes more precise phase by phase.

## Non-Goals

- Do not optimize by string matching emitted JS.
- Do not special-case PrimeNG/demo-app components.
- Do not depend on Vite-only behavior inside `ngtsc/incremental`.
- Do not make semantic graph APIs use untyped `any` forever. Temporary adapters are allowed, but the final graph should be typed around source file paths, declaration keys, semantic symbols, and references.

## Phase 0: Stabilize Incremental Contracts

Objective: replace the current ambiguous `any`-heavy API surface with explicit contracts while keeping existing behavior.

Tasks:

1. Define stable identity types:
   - `FilePath`
   - `ResourcePath`
   - `DeclarationKey`
   - `SemanticSymbolKey`
   - `BuildVersion`
2. Expand `IncrementalState` to include:
   - `Versions map[string]string`
   - `EmittedFiles map[string]bool`
   - `Analysis map[string]FileAnalysisState`
   - `TypeCheck map[string]TypeCheckState`
   - `DepGraph FileDependencyGraph`
   - `SemanticGraph semantic_graph.SemanticDepGraph`
3. Add invariants:
   - no method may panic for unsupported incremental data;
   - unknown state must conservatively force rebuild;
   - deleted files must invalidate all dependents;
   - failed dependency analysis must invalidate dependents conservatively.
4. Update `NOOP_INCREMENTAL_BUILD` so it is a real object that always returns no reusable state, never panics.

Acceptance:

- `go test ./angular-packages/compiler_cli/ngtsc/incremental/...` passes.
- No `panic("unimplemented")` remains in `ngtsc/incremental` except deliberately guarded fatal lifecycle checks.
- Existing compiler behavior is unchanged when no previous state is supplied.

## Phase 1: File Dependency Graph

Objective: track TS and resource dependencies so changed files/resources can produce an affected-file set.

Current stubs:

- `FileDependencyGraph.AddDependency`
- `AddResourceDependency`
- `RecordDependencyAnalysisFailure`
- `GetResourceDependencies`
- `UpdateWithPhysicalChanges`

Tasks:

1. Implement graph storage:
   - `depsByFile[from] -> set(on)`
   - `reverseDeps[on] -> set(from)`
   - `resourceDepsByFile[from] -> set(resource)`
   - `reverseResourceDeps[resource] -> set(from)`
   - `failedAnalysisFiles`
2. Normalize all paths through the same path utility used by compiler host/Vite plugin.
3. Implement `UpdateWithPhysicalChanges`:
   - direct changed TS files are affected;
   - deleted TS files are affected and invalidate reverse dependents;
   - changed resources invalidate files that reference those resources;
   - files with prior dependency-analysis failure are affected;
   - propagate through reverse dependency graph until fixed point.
4. Wire dependency tracker into analysis:
   - component template/style URLs;
   - imports/dependencies discovered by compiler analysis;
   - partial evaluator references where available.

Acceptance:

- Unit tests for changed TS, deleted TS, resource-only change, transitive dependency, and failed analysis.
- Vite template/style edit produces an affected set limited to owning component files.
- Conservative fallback still rebuilds all when dependency information is missing.

## Phase 2: Incremental Build Strategy

Objective: make build strategies store and retrieve incremental state consistently across fresh, tracked, and patched-program modes.

Current stubs:

- `NoopIncrementalBuildStrategy`
- `TrackedIncrementalBuildStrategy`
- `PatchedProgramIncrementalBuildStrategy`

Tasks:

1. Implement `NoopIncrementalBuildStrategy`:
   - `GetIncrementalState` returns empty state;
   - `SetIncrementalState` is no-op;
   - `ToNextBuildStrategy` returns itself.
2. Implement `TrackedIncrementalBuildStrategy`:
   - stores last `IncrementalState`;
   - returns previous state for next build;
   - swaps current state after successful build only.
3. Implement `PatchedProgramIncrementalBuildStrategy`:
   - attaches state to program/driver wrapper where possible;
   - falls back to tracked state if program metadata is unavailable.
4. Add a daemon context field for current incremental strategy/state.
5. Update server build flow:
   - first build uses fresh state;
   - subsequent build uses prior state;
   - failed builds do not replace prior known-good state.

Acceptance:

- Daemon no-change rebuild can prove it reuses prior state.
- Failed build followed by fix does not reuse corrupted state.
- Unit tests cover strategy transitions.

## Phase 3: Analysis Result Reuse

Objective: skip Angular analysis for files that are version-identical and unaffected.

Tasks:

1. Define `FileAnalysisState`:
   - source file path/version;
   - detected traits;
   - analyzed metadata;
   - diagnostics;
   - dependency records;
2. Make `RecordSuccessfulAnalysis(traitCompiler)` collect per-file analysis outputs.
3. Implement `PriorAnalysisFor(sf)`:
   - return prior analysis if version unchanged and file not affected;
   - return nil if resource/dependency/semantic invalidation affects file.
4. Extend `TraitCompiler` to accept prior analysis per source file.
5. Ensure global mode remains checker-safe:
   - do not parallelize checker-dependent reuse incorrectly;
   - reuse must not access stale AST nodes from old program without validation.

Acceptance:

- No-change daemon rebuild skips analysis for all app files.
- Single template/resource edit re-analyzes only owning component and dependent scope files if required.
- Existing render3/linker tests still pass.

## Phase 4: Semantic Symbol Model

Objective: represent Angular semantic API changes separately from physical file changes.

Current stubs:

- `SemanticSymbol.IsPublicApiAffected`
- `IsEmitAffected`
- `IsTypeCheckApiAffected`
- `IsTypeCheckBlockAffected`

Tasks:

1. Define concrete symbol kinds:
   - component;
   - directive;
   - pipe;
   - injectable;
   - NgModule;
2. Expand `SemanticSymbol` fields:
   - declaration key;
   - export/public name;
   - selector;
   - inputs/outputs/model inputs;
   - queries;
   - host directives;
   - standalone/imports/exports/declarations;
   - pipe name/pure flag;
   - DI provider/factory-relevant metadata;
   - template type-check API shape;
   - emit-relevant metadata.
3. Implement comparison categories:
   - public API affected;
   - emit affected;
   - type-check API affected;
   - type-check block affected.
4. Keep all comparisons structural and deterministic.

Acceptance:

- Tests show private class body changes do not invalidate public API.
- Selector/input/output changes invalidate dependents.
- Template-only changes affect emit/TCB but not unrelated public symbols.

## Phase 5: Semantic Dependency Graph

Objective: track symbol-to-symbol references and compute invalidation from semantic changes.

Current stubs:

- `SemanticDepGraph.RegisterSymbol`
- `GetEquivalentSymbol`
- `GetSymbolByDecl`
- `SemanticDepGraphUpdater.RegisterSymbol`
- `Finalize`
- `GetSemanticReference`
- `GetSymbol`

Tasks:

1. Implement graph storage:
   - `symbolsByKey`
   - `symbolsByDecl`
   - `referencesBySymbol`
   - `reverseReferencesBySymbol`
2. Implement `SemanticReference` as a typed value:
   - source symbol key;
   - target symbol key;
   - reference kind: emit, public API, type-check API, type-check block.
3. Implement updater:
   - collect symbols/references during analysis/emit planning;
   - compare current graph with previous graph in `Finalize`;
   - compute affected symbol sets for public API, emit, type-check API, TCB.
4. Feed graph result into `IncrementalCompilation`.

Acceptance:

- Changing a directive input invalidates components that match/use it.
- Changing a pipe name invalidates templates that reference it.
- Changing private implementation of a component does not invalidate unrelated symbols.

## Phase 6: Type Parameter Equality

Objective: correctly compare generic type parameters that affect public/type-check shape.

Current stubs:

- `ExtractSemanticTypeParameters`
- `AreTypeParametersEqual`

Tasks:

1. Extract type parameter data from class/function declarations:
   - name;
   - constraint;
   - default type;
   - variance-like shape if represented;
2. Normalize type expressions into stable strings/structural forms.
3. Compare arrays by length/order/constraint/default.
4. Integrate into component/directive/pipe semantic symbols.

Acceptance:

- Generic directive/component type parameter changes invalidate type-check consumers.
- Renaming without semantic change is handled according to ngtsc behavior, not by raw text alone.

## Phase 7: Type-Check Result Reuse

Objective: reuse TCB/type-check outputs when semantic type-check API is unchanged.

Tasks:

1. Define `TypeCheckState`:
   - source file;
   - TCB dependencies;
   - diagnostics;
   - generated TCB IDs/hashes.
2. Implement `RecordSuccessfulTypeCheck`.
3. Implement `PriorTypeCheckingResultsFor`.
4. Use semantic graph result:
   - unchanged type-check API: reuse;
   - changed type-check block: regenerate local block;
   - changed dependency API: invalidate dependent TCBs.

Acceptance:

- No-change rebuild reuses prior type-check state.
- Template edit regenerates only relevant TCBs.
- Directive input type change invalidates templates that bind to it.

## Phase 8: Emit Skip and Output Cache

Objective: safely skip emit or reuse emitted JS for unaffected files.

Current `SafeToSkipEmit` only checks version equality and prior emitted file.

Tasks:

1. Extend `SafeToSkipEmit` to consider:
   - physical version changes;
   - resource changes;
   - dependency graph affected files;
   - semantic emit affected symbols;
   - linker/HMR/options salt;
2. Store emitted outputs by source path:
   - JS;
   - sourcemap;
   - d.ts if applicable;
   - hash;
3. In daemon mode, return cached memory outputs for skipped files.
4. In disk mode, avoid touching output files when hash unchanged.

Acceptance:

- No-change daemon rebuild avoids re-emitting unchanged files.
- Disk builder does not rewrite unchanged outputs.
- Changing compiler options invalidates output cache.

## Phase 9: Resource-Only Incremental Flow

Objective: handle HTML/CSS/resource edits without recreating full TS program where possible.

Tasks:

1. Separate TS source invalidation from resource invalidation in server RPC.
2. Store resource versions/hashes in incremental state.
3. For template/style changes:
   - invalidate owning component analysis/emit;
   - keep TypeScript AST/checker state when possible;
   - regenerate HMR output only for affected components when HMR enabled.
4. Update Vite plugin to pass resource-specific invalidation metadata to daemon.

Acceptance:

- HTML-only edit rebuilds component output without full TS parse/program recreation.
- CSS-only edit updates component styles/HMR path.
- Non-component global CSS still triggers normal Vite handling.

## Phase 10: Integration With Existing Daemon/Vite Builder

Objective: make incremental state visible and measurable in the real builder path.

Tasks:

1. Add daemon build result metadata:
   - `affectedFiles`
   - `reusedAnalysisCount`
   - `reusedTypeCheckCount`
   - `skippedEmitCount`
   - `cacheHitOutputs`
2. Add verbose Vite logs for incremental reuse.
3. Extend `run_builder_matrix.mjs`:
   - no-change rebuild;
   - TS implementation edit;
   - component template edit;
   - component CSS edit;
   - public API change;
   - dependency change.
4. Add regression thresholds after stable baseline is known.

Acceptance:

- Benchmark report shows per-phase incremental reuse.
- Vite E2E still passes dev/prod.
- Rebuild correctness is validated against cold build output hashes.

## Phase 11: Tests and Parity Gates

Objective: make incremental correctness hard to regress.

Tasks:

1. Unit tests:
   - dependency graph;
   - semantic graph;
   - type parameter equality;
   - strategy state transitions;
2. Integration tests:
   - cold build vs incremental build output equality;
   - resource-only edit;
   - public API invalidation;
   - deleted file invalidation;
3. Vite E2E:
   - dev no-HMR full reload path;
   - dev HMR opt-in path;
   - production build after incremental daemon session;
4. Race tests:
   - concurrent daemon build requests;
   - HMR read while build is running;
   - invalidation while build is running.

Acceptance:

- `go test ./angular-packages/compiler_cli/ngtsc/incremental/...`
- `go test ./angular-packages/compiler_cli ./angular-packages/compiler_cli/integration_tests ./angular-packages/compiler_cli/linker`
- `go test -race` on daemon/incremental/linker critical packages
- Vite dev/prod E2E pass

## Recommended Implementation Order

1. Phase 0: contracts and no-panic baseline.
2. Phase 1: file/resource dependency graph.
3. Phase 2: build strategies and daemon state retention.
4. Phase 3: analysis reuse.
5. Phase 8: emit skip/output cache for immediate builder performance win.
6. Phase 4 and 5: semantic symbols and semantic dependency graph.
7. Phase 6 and 7: type parameters and type-check reuse.
8. Phase 9 and 10: resource-only fast path and observability.
9. Phase 11: broad parity/race gates.

This order gives useful Vite builder wins early without requiring full ngtsc semantic graph parity on day one.

## Risk Register

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Reusing old AST/checker nodes incorrectly | Runtime or emit corruption | Store stable keys/hashes, not raw stale nodes, unless validated against current program. |
| Under-invalidating semantic changes | Incorrect production output | Conservative fallback: unknown semantic comparison means affected. |
| Resource-only path bypasses TS scope changes | Stale templates/directives | Resource-only path must still consult scope/dependency graph. |
| Global checker is not goroutine-safe | Races/panics | Keep global analysis serialized unless checker access is proven read-only. |
| HMR and incremental state diverge | Dev-only runtime errors | HMR output must derive from the same affected component set as normal emit. |

## Definition of Done

Incremental/semantic graph work is production-builder ready when:

- no unimplemented panics remain in `ngtsc/incremental`;
- no-change daemon rebuild reuses analysis and emit safely;
- template/style edits rebuild only affected components;
- public API changes invalidate dependents;
- cold build and incremental build outputs match byte-for-byte after normalization;
- dev/prod Vite E2E pass;
- race tests pass for daemon + incremental paths;
- benchmark matrix records stable improvements for no-change and local edit cases.
