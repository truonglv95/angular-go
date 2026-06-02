# NgTemplateOutlet TemplateRef slot bug fix plan

## Runtime symptom

The demo fails at runtime with:

```text
ERROR TypeError: templateRef.createEmbeddedViewImpl is not a function
    at _R3ViewContainerRef.createEmbeddedView (...)
    at _NgTemplateOutlet.ngOnChanges (...)
```

This means `NgTemplateOutlet` received a value that is not Angular's internal `TemplateRef` instance. `ViewContainerRef.createEmbeddedView()` calls the internal method `createEmbeddedViewImpl()`, which only real `TemplateRef` objects have.

## Affected path

Current demo source:

- `angular-packages/new-demo-app/src/app/app.html`
- `angular-packages/new-demo-app/src/app/app.ts`

The app uses:

```html
<p-menu [model]="items" styleClass="w-full"></p-menu>
```

The failing `NgTemplateOutlet` path is inside PrimeNG Menu, not directly in app code.

Relevant generated/cache files:

- `angular-packages/new-demo-app/node_modules/.vite/deps/primeng_menu.js`
- `angular-packages/new-demo-app/node_modules/primeng/fesm2022/primeng-menu.mjs`

PrimeNG Menu has this internal template:

```html
<ng-container *ngTemplateOutlet="itemContent; context: { $implicit: item }"></ng-container>

<ng-template #itemContent>
  ...
</ng-template>
```

So `itemContent` must resolve to the `TemplateRef` of the local `<ng-template #itemContent>`.

## Evidence

I compared our Go linker output with Angular linker oracle for:

```text
node_modules/primeng/fesm2022/primeng-menu.mjs
```

### Angular linker output

Angular emits the local ref array first in `consts`:

```js
consts: [
  ["itemContent", ""],
  ["htmlLabel", ""],
  [3, "click", "pBind"],
  ...
]
```

And the `ng-template #itemContent` creation uses const index `0` with `templateRefExtractor`:

```js
i0.ɵɵtemplate(
  3,
  MenuItemContent_ng_template_3_Template,
  5,
  4,
  "ng-template",
  null,
  0,
  i0.ɵɵtemplateRefExtractor
);
```

Inside nested embedded views Angular references slot `4`:

```js
const itemContent_r2 = i0.ɵɵreference(4);
```

### Go linker output

Go currently emits local ref arrays at the end of `consts`:

```js
consts: [
  [3, "click", "pBind"],
  [4, "ngIf"],
  ...
  ["itemContent", ""],
  ["htmlLabel", ""]
]
```

The `ng-template #itemContent` creation is still on tNode slot `3`:

```js
i0.ɵɵtemplate(
  3,
  MenuItemContent_ng_template_3_Template,
  5,
  4,
  "ng-template",
  null,
  14,
  i0.ɵɵtemplateRefExtractor
);
```

But nested views reference slot `3`:

```js
const itemContent_r2 = i0.ɵɵreference(3);
```

This is wrong. Slot `3` is the `TContainer`/template node slot. The local ref value slot is after the node slot.

For one local ref:

```text
template tNode slot: 3
local ref #itemContent slot: 4
```

So `ɵɵreference(3)` can return the wrong runtime value, and `NgTemplateOutlet` later receives a non-TemplateRef object.

## Root cause

The compiler pipeline already records local ref offsets, but reify ignores them.

Relevant code:

- `angular-packages/compiler/template/pipeline/phases/generate_variables.go`
- `angular-packages/compiler/template/pipeline/phases/reify.go`
- `angular-packages/compiler/template/pipeline/phases/local_refs_and_resolve_contexts.go`
- `angular-packages/compiler/template/pipeline/phases/slot_allocation.go`

`generate_variables.go` records each local ref with an offset:

```go
Initializer: ir.NewReferenceExpr(ref.TargetId, ref.TargetSlot, ref.Offset)
```

But `reify.go` currently emits only the target slot:

```go
case ir.ExpressionKindReference:
    ref := irExpr.(*ir.ReferenceExpr)
    slot := 0
    if ref.TargetSlot != nil && ref.TargetSlot.Slot != nil {
        slot = *ref.TargetSlot.Slot
    }
    args := []output.Expression{output.NewLiteralExpr(slot, nil, nil, nil)}
    return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
```

`ref.Offset` is ignored.

For local refs, Angular stores ref values in slots after the declaration node:

```text
actualRefSlot = targetSlot + 1 + ref.Offset
```

Current Go output uses:

```text
actualRefSlot = targetSlot
```

That explains the exact mismatch:

```text
expected: ɵɵreference(4)
actual:   ɵɵreference(3)
```

## Fix plan

### Phase 1: Add a focused regression test

Add a compiler/linker parity test for a component template with:

```html
<ng-container *ngTemplateOutlet="itemContent"></ng-container>
<ng-template #itemContent>Item</ng-template>
```

Acceptance:

- Go output must contain `ɵɵtemplate(... "ng-template", null, <localRefConst>, ɵɵtemplateRefExtractor)`.
- Go output must emit `ɵɵreference(<templateSlot + 1>)`, not `ɵɵreference(<templateSlot>)`.
- The test should fail before the fix.

### Phase 2: Fix `ReferenceExpr` reification

Update `angular-packages/compiler/template/pipeline/phases/reify.go`.

For `ir.ExpressionKindReference`, compute:

```go
slot = *ref.TargetSlot.Slot + 1 + ref.Offset
```

instead of:

```go
slot = *ref.TargetSlot.Slot
```

This must be generic for all local refs:

```html
<div #a #b></div>
```

Expected slots:

```text
#a => targetSlot + 1
#b => targetSlot + 2
```

Acceptance:

- Single local ref works.
- Multiple local refs use increasing offsets.
- Existing `ReferenceExpr.Offset` becomes meaningful.

### Phase 3: Verify local-ref const handling

The local ref const order differs from Angular oracle, but this is not necessarily fatal because the creation instruction passes the matching const index.

Still verify:

- `LiftLocalRefs()` adds `len(refs)` to slot usage.
- `TemplateOp.NumSlotsUsed` includes node slot + local ref slots.
- `reify.go` passes `templateRefExtractor` for `<ng-template>` local refs.
- Element refs and container refs still use correct slots.

Acceptance:

- Const order may differ if semantic output is correct.
- Slot references must match runtime layout.

### Phase 4: Re-link PrimeNG Menu and compare with Angular oracle

Run:

```bash
./go-ngc --link angular-packages/new-demo-app/node_modules/primeng/fesm2022/primeng-menu.mjs > /tmp/go-menu.js
```

Run Angular linker oracle for the same input.

Compare specifically:

```js
const itemContent_r2 = ɵɵreference(...)
const htmlLabel_r3 = ɵɵreference(...)
```

Expected after fix:

```js
itemContent_r2 = ɵɵreference(4)
htmlLabel_r3 = ɵɵreference(3)
```

depending on each template's local declaration slot.

Acceptance:

- PrimeNG `MenuItemContent` no longer emits `ɵɵreference(3)` for `#itemContent` when the template slot is `3`.
- It emits `ɵɵreference(4)`.

### Phase 5: Runtime validation in Vite

Rebuild:

```bash
source ~/.gvm/scripts/gvm
gvm use go1.26.3
go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go
```

Clear Vite cache:

```bash
cd angular-packages/new-demo-app
rm -rf node_modules/.vite
```

Run Vite with Node 20.19+:

```bash
node ./node_modules/vite/bin/vite.js --host 127.0.0.1 --port 4200
```

Acceptance:

- `p-menu` renders without `templateRef.createEmbeddedViewImpl is not a function`.
- Button components still render.
- No regression in previously fixed linker cases:
  - `deps: null`
  - `deps: []`
  - `deps: "invalid"`

### Phase 6: Test sweep

Run:

```bash
go test ./angular-packages/compiler_cli/linker
```

Also run focused template pipeline tests for local references if available. If broader template pipeline tests still fail on unrelated stale expectations, keep that separate and document it.

## Definition of done

- A regression test covers local ref slot offsets.
- `reify.go` uses `targetSlot + 1 + offset` for `ɵɵreference`.
- PrimeNG Menu linked output references the correct `TemplateRef` slot.
- Vite demo no longer throws `createEmbeddedViewImpl is not a function`.
- No hard-coded PrimeNG-specific behavior is added.
