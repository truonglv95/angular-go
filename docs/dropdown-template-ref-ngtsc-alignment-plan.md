# Dropdown TemplateRef ngtsc Alignment Plan

## Current failure

Opening the PrimeNG select/dropdown panel still fails at runtime:

```text
ERROR TypeError: templateRef.createEmbeddedViewImpl is not a function
    at _R3ViewContainerRef.createEmbeddedView
    at _NgTemplateOutlet.ngOnChanges
```

This means an `NgTemplateOutlet` input receives a non-`TemplateRef` value. For `p-select`, the important path is:

```text
p-select -> p-overlay -> ContentChild('content') -> contentTemplate -> NgTemplateOutlet
```

There are also internal `p-select` outlets such as `buildInItems`, `loader`, `empty`, `emptyFilter`, `itemTemplate`, etc. Any of these can fail if local refs or embedded-view context are emitted incorrectly.

## What was verified

Direct Go linker output for the obvious `p-select` / `p-overlay` path now looks close to ngtsc:

```js
i0.ɵɵelementStart(7, "p-overlay", 21, 1);
i0.ɵɵtemplate(9, Select_ng_template_9_Template, 13, 23, "ng-template", null, 2, i0.ɵɵtemplateRefExtractor);
```

`p-overlay` content query is also semantically close:

```js
i0.ɵɵcontentQuery(dirIndex, ["content"], 4);
i0.ɵɵqueryRefresh(tmp = i0.ɵɵloadQuery()) && (ctx.contentTemplate = tmp.first);
```

However, "looks close" is not enough. Runtime still reports a non-`TemplateRef`, so the remaining issue is likely in pipeline semantics, not a single hard-coded instruction.

## Key ngtsc finding

The Go template pipeline order is materially different from Angular's current pipeline.

Angular `angular-src/packages/compiler/src/template/pipeline/src/emit.ts` runs relevant phases in this order:

```text
generateVariables
saveAndRestoreView
resolveNames
resolveContexts
resolveSanitizers
liftLocalRefs
expandSafeReads
generateTemporaryVariables
optimizeVariables
optimizeStoreLet
allocateSlots
collectConstExpressions
collectElementConsts
countVariables
generateAdvance
nameFunctionsAndVariables
mergeNextContextExpressions
generateNgContainerOps
collapseEmptyInstructions
reify
chain
```

The Go pipeline currently differs in critical ways:

```text
GenerateNgContainerOps            // too early
...
ReserveLocalRefs                  // custom workaround
CollectElementConsts              // too early
...
LiftLocalRefs                     // before ResolveContexts, unlike ngtsc
...
AllocateSlots
GenerateAdvance
GenerateTemporaryVariables        // too late
CollapseEmptyInstructions
OptimizeVariables
NormalizeSequentialNextContext... // custom workaround
NameFunctionsAndVariables
CountVariables
Reify
Chain
```

This explains why each fix has moved the failure to a new runtime surface:

- local-ref const ordering was patched with `ReserveLocalRefs`;
- `ng-container` refs were patched by widening local-ref handling;
- context depth was patched by normalizing `nextContext`;
- but the compiler is still not preserving the same IR lifecycle as ngtsc.

The safest fix is to align phase ordering and delete workarounds that only exist because the order is wrong.

## Root-cause hypothesis

The `TemplateRef` failure is caused by local refs being lifted/serialized after IR has already diverged from ngtsc:

1. `GenerateNgContainerOps` runs at the start in Go, but ngtsc runs it near the end.
2. That changes whether local refs live on `Template`, `ElementStart`, or `ContainerStart` ops during scope generation and local-ref lifting.
3. Go then compensates by including `ContainerStart` / `Container` in local-ref phases, but ngtsc's `liftLocalRefs` only handles:

```text
ElementStart
ConditionalCreate
ConditionalBranchCreate
Template
```

4. This can produce correct-looking local-ref const arrays while still producing different runtime local-ref values in embedded/template/query paths.

## Fix plan

### Phase 1: Add a phase-order parity test

Create a focused test that links a component using the exact PrimeNG pattern:

```html
<x-overlay #overlay>
  <ng-template #content>
    <ng-container *ngTemplateOutlet="buildInItems; context: { $implicit: items }"></ng-container>
    <ng-template #buildInItems let-items>...</ng-template>
  </ng-template>
</x-overlay>
```

Assertions:

- `#content` is emitted on an `ng-template` with `ɵɵtemplateRefExtractor`.
- `ContentChild('content')` receives the extracted template ref.
- `#buildInItems` is emitted with `ɵɵtemplateRefExtractor`.
- `ɵɵreference(...)` slot indexes match ngtsc.
- No local ref is lifted from `ContainerStart` / `Container` unless ngtsc does the same for that fixture.

### Phase 2: Reorder Go pipeline to match ngtsc

Move phases toward Angular's `emit.ts` order:

- Move `GenerateNgContainerOps` from the beginning to near the end, after `MergeNextContextExpressions` and before `CollapseEmptyInstructions`.
- Move `CollectElementConsts` from early pipeline to after `CollectConstExpressions` / equivalent late const phase.
- Move `LiftLocalRefs` to after `ResolveSanitizers` and before `ExpandSafeReads`.
- Move `GenerateTemporaryVariables` before `OptimizeVariables`.
- Move `AllocateSlots` closer to ngtsc's position after store-let optimization and before advance generation.
- Ensure `CountVariables` runs before reify and after late const collection.

Do this conservatively, one phase group at a time, with tests after each group.

### Phase 3: Remove local-ref workarounds after phase alignment

After phase order is corrected:

- Remove `ReserveLocalRefs`.
- Remove `ReservedLocalRefsConst` from `ElementOrContainerOpBase`.
- Revert `LiftLocalRefs` op coverage to ngtsc parity:

```text
ElementStart
ConditionalCreate
ConditionalBranchCreate
Template
```

- Re-evaluate whether `GenerateVariables` should include `ContainerStart` / `Container`. If ngtsc does not, remove it unless a failing parity fixture proves otherwise.

### Phase 4: Replace context workaround with ngtsc-equivalent merge

The custom `NormalizeSequentialNextContextExpressions` fixed `p-menu`, but it should be validated against ngtsc's `mergeNextContextExpressions`.

Plan:

- Compare Go implementation with `angular-src/packages/compiler/src/template/pipeline/src/phases/next_context_merging.ts`.
- Keep only behavior that matches ngtsc.
- Add tests for nested `@if` + `*ngFor` + `*ngIf` where root context and loop context are both used.

### Phase 5: PrimeNG linker parity harness

Add a test or script that links and compares these PrimeNG files against ngtsc:

```text
primeng-overlay.mjs
primeng-select.mjs
primeng-menu.mjs
primeng-datepicker.mjs
```

For each file, compare:

- `consts` local-ref arrays
- every instruction with `ɵɵtemplateRefExtractor`
- every `ɵɵreference(...)`
- every `ɵɵcontentQuery(...)`
- every `ɵɵviewQuery(...)`
- every `ɵɵnextContext(...)`
- every `ngTemplateOutlet` binding source

The comparison does not need byte-for-byte equality, but it must fail if slot indexes, local-ref indexes, or context depth differ.

### Phase 6: Runtime debug guard for this class of failure

Temporarily add a dev-only Vite/plugin diagnostic mode:

- Detect linked modules containing `NgTemplateOutlet`.
- Inject a tiny wrapper or source-map assisted log around `ngTemplateOutlet` property writes.
- Log whether the value has `createEmbeddedViewImpl`.
- Include the component/template function name and source module.

This should be behind an option, e.g.:

```ts
angularGo({ debugTemplateRefs: true })
```

This is not the production fix, but it makes the next runtime failure directly identify the bad producer instead of guessing from stack traces.

### Phase 7: Rebuild and verify in Vite dev mode

After implementation:

```bash
source ~/.gvm/scripts/gvm && gvm use go1.26.3 >/dev/null
go test ./angular-packages/compiler_cli/linker ./angular-packages/compiler/template/pipeline/...
go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go
cd angular-packages/new-demo-app
rm -rf node_modules/.vite
npx vite --host 127.0.0.1 --port 4200
```

Manual smoke:

- click `p-menu`
- open `p-select`
- select an option
- open/close datepicker
- switch views in the demo app

## Acceptance criteria

- `p-select` panel opens without `templateRef.createEmbeddedViewImpl` error.
- `p-menu` still has no `ctx.model` or `menuitemId` context errors.
- `primeng-select` local-ref/template/query output matches ngtsc for all template-ref related instructions.
- No PrimeNG file contains replacement characters (`�`) after linking.
- The fix removes or justifies every custom workaround introduced for local refs/context depth.
