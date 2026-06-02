# PrimeNG Dropdown TemplateRef Fix Plan

## Current failure

When opening the PrimeNG dropdown/select demo:

```text
ERROR TypeError: templateRef.createEmbeddedViewImpl is not a function
```

The demo path is:

```text
App -> p-select -> p-overlay -> NgTemplateOutlet -> ViewContainerRef.createEmbeddedView()
```

`p-select` projects an internal template into `p-overlay`:

```html
<p-overlay #overlay ...>
  <ng-template #content>...</ng-template>
</p-overlay>
```

`p-overlay` reads that projected template with:

```ts
@ContentChild('content') contentTemplate: TemplateRef<any>;
```

and renders it through:

```html
<ng-container *ngTemplateOutlet="contentTemplate || _contentTemplate; context: ..."></ng-container>
```

The runtime error means `contentTemplate || _contentTemplate` is not an Angular `TemplateRef`. It is likely an `ElementRef`, directive instance, or another local-reference value.

## Findings so far

1. The app uses `p-select` in `src/app/app.html` under the `form` view:

```html
<p-select [options]="cities" [(ngModel)]="selectedCity" optionLabel="name" placeholder="Select a City" styleClass="w-64" />
```

2. Direct compare of `primeng-select.mjs` shows the critical area:

Go output:

```js
i0.ɵɵelementStart(7, "p-overlay", 5, 37);
i0.ɵɵtemplate(9, Select_ng_template_9_Template, 13, 23, "ng-template", null, 38, i0.ɵɵtemplateRefExtractor);
```

ngtsc output:

```js
i0.ɵɵelementStart(7, "p-overlay", 21, 1);
i0.ɵɵtemplate(9, Select_ng_template_9_Template, 13, 23, "ng-template", null, 2, i0.ɵɵtemplateRefExtractor);
```

Both use `ɵɵtemplateRefExtractor`, but Go emits local-ref consts at the end of `consts`, while ngtsc emits local-ref consts first. The indexes are adjusted, so this may be valid, but it must be verified against projection/query behavior.

3. Direct compare of `primeng-overlay.mjs` shows `Overlay` queries local ref `content`:

Go:

```js
i0.ɵɵcontentQuery(dirIndex, ["content"], 4);
i0.ɵɵviewQuery(["overlay"], 5);
i0.ɵɵviewQuery(["content"], 5);
```

ngtsc:

```js
i0.ɵɵcontentQuery(dirIndex, _c0, 4);
i0.ɵɵviewQuery(_c1, 5)(_c0, 5);
```

Semantically these look equivalent, but the value returned by the `content` content query must be proven to be a `TemplateRef`.

4. There is also an optimizer-blocking encoding failure:

```text
Expected ";" but found "�"
node_modules/primeng/fesm2022/primeng-datepicker.mjs:4751
i0.ɵ��template(...)
```

This must be handled in the same workstream because dependency optimization cannot be trusted if linked output can corrupt Angular private identifiers like `ɵɵtemplate`.

## Working hypotheses

### H1: Projected `ng-template #content` local ref resolves to the wrong value

The most likely cause is local-ref value slot allocation or local-ref metadata around projected `ng-template` nodes. If `Overlay`'s content query receives the wrong local-ref value, it passes a non-TemplateRef to `NgTemplateOutlet`.

Expected behavior:

```js
ɵɵtemplate(slot, ..., "ng-template", null, localRefConstIndex, ɵɵtemplateRefExtractor)
```

and `ContentChild('content')` should receive the extractor-produced `TemplateRef`.

### H2: Const pool/local-ref ordering is valid for direct references, but broken for content query matching

Go currently reorders local-ref const arrays compared with ngtsc. Direct `ɵɵreference(n)` can still work if indexes are adjusted, but content queries against local-ref names may depend on `TNode.localNames` shape/order more strictly.

### H3: `ɵ` corruption comes from linker/plugin output encoding, not Angular semantics

The `i0.ɵ��template` syntax error suggests one of:

- Go linker emits invalid UTF-8 bytes.
- The Vite plugin captures/processes stdout in a way that corrupts multi-byte UTF-8.
- esbuild receives already-corrupted linked contents.
- A stale/corrupted `.vite/deps` file is being reused after a failed link.

## Implementation plan

### Phase 1: Reproduce with a minimal fixture

Create a linker parity test that models the exact pattern:

```ts
@Component({
  selector: 'x-overlay',
  template: `<ng-container *ngTemplateOutlet="contentTemplate"></ng-container>`
})
class OverlayLike {
  @ContentChild('content') contentTemplate: TemplateRef<any>;
}

@Component({
  selector: 'x-select',
  template: `
    <x-overlay>
      <ng-template #content>content</ng-template>
    </x-overlay>
  `
})
class SelectLike {}
```

Assertions:

- The projected `ng-template #content` is emitted with `ɵɵtemplateRefExtractor`.
- `OverlayLike` content query matches the `content` local ref.
- No projected local ref resolves to an element/directive value.

### Phase 2: Add byte-level encoding regression

Add a direct linker test around `primeng-datepicker`-like output or a smaller source that emits `ɵɵtemplate`.

Assertions:

- Linked output is valid UTF-8.
- Output does not contain Unicode replacement char `�`.
- Output contains `ɵɵtemplate`, not corrupted `ɵ��template`.

Also add a Vite plugin-level check before returning linked contents:

```ts
if (linked.includes('\uFFFD')) {
  throw new Error('go-ngc linker emitted replacement characters');
}
```

This fails fast instead of letting esbuild report a misleading syntax error.

### Phase 3: Compare Go vs ngtsc for `Select` and `Overlay`

Generate paired outputs:

```bash
./go-ngc --link angular-packages/new-demo-app/node_modules/primeng/fesm2022/primeng-select.mjs
./go-ngc --link angular-packages/new-demo-app/node_modules/primeng/fesm2022/primeng-overlay.mjs
```

and Angular linker output via `@angular/compiler-cli/linker/babel`.

Compare:

- `consts` local-ref arrays.
- `ɵɵtemplate(..., localRefsIndex, ɵɵtemplateRefExtractor)`.
- `ɵɵcontentQuery(...)`.
- `ɵɵviewQuery(...)`.
- `ɵɵreference(...)`.
- Any `ngTemplateOutlet` input source.

### Phase 4: Fix generic local-ref/query semantics

Do not special-case PrimeNG.

Potential fixes to validate:

- Preserve ngtsc-like local-ref const ordering in const collection.
- Ensure `ng-template` local refs always allocate declaration slot + extracted value slot.
- Ensure `ReferenceExpr.Offset` is applied for all consumers, including projected templates and content-query paths.
- Ensure content queries for string predicates read the local-ref value, not the declaration node.

### Phase 5: Rebuild and dependency optimizer validation

After code fix:

```bash
source ~/.gvm/scripts/gvm && gvm use go1.26.3 >/dev/null
go test ./angular-packages/compiler_cli/linker
go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go
```

Then regenerate Vite deps:

```bash
cd angular-packages/new-demo-app
rm -rf node_modules/.vite
node ./node_modules/vite/bin/vite.js --host 127.0.0.1 --port 4200
```

Validate optimized files:

- `primeng_select.js` has no `�`.
- `primeng_datepicker.js` has no `�`.
- `p-select` overlay content template is emitted with `templateRefExtractor`.
- The dropdown opens without `createEmbeddedViewImpl` error.

### Phase 6: Broaden PrimeNG smoke coverage

Run the same parity/smoke checks for components using overlays/templates:

- `primeng/select`
- `primeng/datepicker`
- `primeng/multiselect`
- `primeng/dialog`
- `primeng/table`
- `primeng/tree`

The goal is to catch all variants of:

- projected `ng-template`
- `ContentChild('name')`
- `ViewChild('name')`
- `NgTemplateOutlet`
- template ref inside nested embedded views

## Expected outcome

After this plan, Go linker output should match ngtsc behavior for projected TemplateRefs and overlay-driven dropdowns. The fix should be generic across PrimeNG and Angular libraries, not tied to `p-select`.

## Implementation status

Implemented.

- Added a generic `ReserveLocalRefs` phase before attribute const collection, so local-ref const arrays are emitted before attribute const arrays like ngtsc.
- Kept the existing `LiftLocalRefs` phase after `ResolveNames`, so local refs remain available for scope generation and template name resolution.
- Stored the reserved const index on `ElementOrContainerOpBase` and reused it during `LiftLocalRefs`.
- Included `ng-container` local refs in scope generation and local-ref lifting, covering PrimeNG cases such as `#emptyFilter` and `#empty` inside embedded views.
- Added a linker parity regression for projected `ng-template #content` through an overlay-like component.

Validation run:

```bash
source ~/.gvm/scripts/gvm && gvm use go1.26.3 >/dev/null && go test ./angular-packages/compiler_cli/linker ./angular-packages/compiler/template/pipeline/...
source ~/.gvm/scripts/gvm && gvm use go1.26.3 >/dev/null && go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go
./go-ngc --link angular-packages/new-demo-app/node_modules/primeng/fesm2022/primeng-select.mjs
./go-ngc --link angular-packages/new-demo-app/node_modules/primeng/fesm2022/primeng-datepicker.mjs
cd angular-packages/new-demo-app && rm -rf node_modules/.vite && node ./node_modules/vite/bin/vite.js build
```

Key compare result for `primeng-select` now matches ngtsc in the critical area:

```js
consts: [["elseBlock", ""], ["overlay", ""], ["content", ""], ... ["emptyFilter", ""], ["empty", ""], ...]
i0.ɵɵelementStart(7, "p-overlay", 21, 1);
i0.ɵɵtemplate(9, Select_ng_template_9_Template, 13, 23, "ng-template", null, 2, i0.ɵɵtemplateRefExtractor);
```

`primeng-datepicker` linked output was also checked for `�`/`ɵ��` and no replacement characters were found.
