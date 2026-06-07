# Angular Go AOT Component Parity Roadmap

Date: 2026-06-07

## Goal

Bring `angular-go` AOT component compilation from "supported for covered SPA/demo cases" to practical parity with Angular's official `@angular/compiler-cli`.

Target baseline:

- Angular compiler baseline: `@angular/compiler-cli` `21.2.x`.
- Runtime baseline: Angular `21.2.x`.
- Primary scope: browser SPA AOT component compilation.
- Secondary scope: diagnostics, type check blocks, incremental rebuilds, and runtime parity.

This roadmap intentionally does not claim full Angular platform parity for SSR, i18n/localize, service worker, or library packaging. Those should remain separate roadmaps.

## Definition Of Done

`angular-go` can claim AOT component parity when:

- `go test ./angular-packages/compiler_cli/...` is green.
- AOT compliance golden suite passes against the selected Angular baseline.
- Runtime parity tests pass through `angular-go/testbed` and Vite demo apps.
- Forms, router, PrimeNG template, content projection, structural control flow, host binding/listener, and query regressions are covered.
- No reachable `panic("unimplemented")` remains on the component AOT path.
- Diagnostics match Angular behavior closely enough for real projects:
  - same error category;
  - same or mapped error code;
  - same file/span where practical;
  - same fatal/nonfatal behavior.
- Incremental rebuild tests do not produce stale output.
- Performance is measured for cold and warm builds, and no phase regresses beyond accepted thresholds.

## Implementation Order

Recommended order:

1. Phase 0 - Baseline And Inventory
2. Phase 1 - Golden Harness
3. Phase 2 - Component Metadata Parity
4. Phase 3 - Scope Resolution
5. Phase 4 - Template Parser And Binder
6. Phase 5 - Render3 IR Parity
7. Phase 6 - Instruction Coverage
8. Phase 7 - Modern Angular Features
9. Phase 8 - Diagnostics Parity
10. Phase 9 - Type Check Blocks
11. Phase 10 - Incremental AOT
12. Phase 11 - Runtime Parity Suite
13. Phase 12 - Performance Gate

Do not skip Phase 1. Without a stable golden harness, fixes will continue to jump from one runtime error to another.

## Phase 0 - Baseline And Inventory

### Tasks

- Lock the Angular comparison version:
  - `@angular/compiler-cli`
  - `@angular/compiler`
  - `@angular/core`
  - `@angular/common`
  - `@angular/forms`
  - `@angular/router`
- Record the exact version in this roadmap.
- Inventory Angular compliance cases from:
  - `angular-packages/angular-src/packages/compiler-cli/test/compliance/test_cases`
  - `angular-packages/angular-src/packages/compiler-cli/test/ngtsc`
  - `angular-packages/angular-src/packages/compiler/test`
- Create a case classification table:
  - component metadata;
  - directive/pipe;
  - NgModule scope;
  - standalone scope;
  - template binding;
  - control flow;
  - defer;
  - forms;
  - host bindings/listeners;
  - content projection;
  - queries;
  - diagnostics;
  - type checking.
- Identify currently known local regressions:
  - `TemplateRef.createEmbeddedViewImpl`;
  - event `$event.target`;
  - `ngIf` not recognized;
  - PrimeNG dropdown/template contexts;
  - forms listener/value accessor behavior;
  - import elision;
  - defer output drift.
- List all reachable `panic("unimplemented")` from the component AOT path.

### Acceptance Criteria

- There is a checked-in inventory document or test manifest.
- Every compliance family has an owner status:
  - supported;
  - partial;
  - not started;
  - intentionally out of scope.
- Known regressions are mapped to test cases.

## Phase 1 - Golden Harness

### Tasks

- Build a harness that compiles the same fixture with:
  - official Angular `ngtsc`;
  - `go-ngc`.
- Support fixture input:
  - project root;
  - `tsconfig.json`;
  - expected output paths.
- Compare:
  - JS emit;
  - `.d.ts` emit where applicable;
  - diagnostics;
  - linker output for partial declarations.
- Add controlled normalization:
  - line endings;
  - source map paths;
  - harmless import order only if semantically equivalent;
  - generated temporary suffixes only if proven nonsemantic.
- Do not normalize:
  - instruction order;
  - slot indices;
  - context variable access;
  - generated dependencies;
  - directive/pipe scope contents;
  - listener payloads.
- Add update command for intentional golden updates.
- Add clear failure output:
  - fixture name;
  - file path;
  - diff;
  - classification hint.

### Acceptance Criteria

- A new fixture can be added with one command.
- Golden failures distinguish:
  - formatting drift;
  - semantic emit mismatch;
  - missing diagnostic;
  - extra diagnostic;
  - compiler crash.
- Existing golden fixture path issues are fixed permanently.

## Phase 2 - Component Metadata Parity

### Tasks

- Compare `angular-go` component decorator analysis with Angular's:
  - `angular-src/packages/compiler-cli/src/ngtsc/annotations/component`.
- Implement/verify metadata fields:
  - `selector`;
  - `standalone`;
  - `imports`;
  - `host`;
  - `inputs`;
  - aliased inputs;
  - required inputs;
  - signal inputs;
  - model inputs;
  - `outputs`;
  - aliased outputs;
  - `queries`;
  - `viewQueries`;
  - `providers`;
  - `viewProviders`;
  - `encapsulation`;
  - `changeDetection`;
  - `styles`;
  - `styleUrls`;
  - `template`;
  - `templateUrl`;
  - `preserveWhitespaces`;
  - `schemas`;
  - `animations` fail-fast or support.
- Implement forwardRef and partially evaluated metadata shapes.
- Add metadata golden fixtures for each field family.

### Acceptance Criteria

- Component metadata output matches Angular for supported cases.
- Unsupported metadata fails with explicit diagnostic or documented unsupported error.
- No metadata field is silently ignored if Angular uses it for emit/runtime behavior.

## Phase 3 - Scope Resolution

### Tasks

- Implement standalone scope parity:
  - component imports;
  - standalone directive imports;
  - standalone pipe imports;
  - imported NgModule exports;
  - transitive imports;
  - duplicate imports;
  - invalid imports diagnostics.
- Implement NgModule scope parity:
  - declarations;
  - imports;
  - exports;
  - bootstrap;
  - schemas;
  - transitive compilation scope;
  - transitive export scope;
  - duplicate declarations;
  - declaration in multiple modules;
  - standalone component declared in NgModule diagnostics.
- Support package re-export patterns:
  - direct export;
  - barrel export;
  - alias export;
  - `.d.ts` metadata import aliases.
- Add tests for:
  - `CommonModule` directives/pipes;
  - `FormsModule`;
  - `ReactiveFormsModule`;
  - router directives;
  - third-party library modules such as PrimeNG.

### Acceptance Criteria

- `ngIf`, `ngFor`, pipes, forms directives, and PrimeNG components are resolved through the same scope shape as Angular.
- Scope changes invalidate affected components.
- Unknown element/property diagnostics match Angular category and span.

## Phase 4 - Template Parser And Binder

### Tasks

- Verify parser parity for:
  - interpolation;
  - property binding;
  - attribute binding;
  - class/style binding;
  - event binding;
  - two-way binding;
  - references;
  - variables;
  - pipes;
  - safe navigation;
  - non-null assertions;
  - `$event`;
  - `$any`;
  - SVG/MathML namespaces.
- Verify binder parity for:
  - directive matching;
  - input matching;
  - output matching;
  - local refs;
  - `exportAs`;
  - template refs;
  - embedded view contexts;
  - nested view context restore;
  - pipe lookup;
  - defer dependencies.
- Add fixtures for:
  - nested templates;
  - template refs passed to inputs;
  - `NgTemplateOutlet`;
  - structural directives;
  - form controls;
  - component outputs.

### Acceptance Criteria

- Template binder output drives the same directive/pipe/instruction choices as Angular.
- Previous context-related runtime errors have regression fixtures.
- `$event.target` and listener payload behavior match Angular.

## Phase 5 - Render3 IR Parity

### Tasks

- Audit current pipeline model:
  - slice-based operations;
  - linked-list operation list;
  - mutation phases;
  - slot allocation.
- Decide and document the canonical operation model.
- Remove or isolate mixed OpList debt.
- Compare Render3 phases against Angular:
  - ingest;
  - binding specialization;
  - i18n phases if in scope;
  - defer phases;
  - pure function extraction;
  - slot allocation;
  - namespace handling;
  - projection;
  - pipe handling;
  - listener handling;
  - view restore.
- Add golden tests for each phase family.

### Acceptance Criteria

- IR phase output is deterministic.
- Slot allocation matches Angular for all covered cases.
- No runtime workaround is needed for `TemplateRef`, view context, or slot restore.

## Phase 6 - Instruction Coverage

### Tasks

- Add instruction golden/runtime coverage for:
  - `ɵɵelementStart`;
  - `ɵɵelementEnd`;
  - `ɵɵelement`;
  - `ɵɵtext`;
  - `ɵɵtextInterpolate*`;
  - `ɵɵtemplate`;
  - `ɵɵconditional`;
  - `ɵɵrepeater`;
  - `ɵɵlistener`;
  - `ɵɵproperty`;
  - `ɵɵattribute`;
  - `ɵɵclassProp`;
  - `ɵɵstyleProp`;
  - `ɵɵadvance`;
  - `ɵɵpipe`;
  - `ɵɵpipeBind*`;
  - `ɵɵprojection`;
  - `ɵɵprojectionDef`;
  - `ɵɵreference`;
  - `ɵɵqueryRefresh`;
  - `ɵɵviewQuery`;
  - `ɵɵcontentQuery`;
  - host binding instructions.
- Add forms-specific listener/value accessor output tests.
- Add PrimeNG template context instruction tests.

### Acceptance Criteria

- Each instruction family has at least one golden fixture and one runtime test where practical.
- Emitted create/update block structure matches Angular for covered cases.

## Phase 7 - Modern Angular Features

### Tasks

- Implement and test signal input support:
  - `input()`;
  - `input.required()`;
  - aliases;
  - transforms.
- Implement and test model input support:
  - `model()`;
  - aliases;
  - two-way binding emit.
- Implement and test modern control flow:
  - `@if`;
  - `@else if`;
  - `@else`;
  - alias expressions;
  - `@for`;
  - `track`;
  - contextual variables;
  - `@empty`;
  - `@switch`;
  - `@case`;
  - `@default`.
- Implement and test defer:
  - `@defer`;
  - `@placeholder`;
  - `@loading`;
  - `@error`;
  - triggers;
  - dependencies;
  - generated imports.

### Acceptance Criteria

- Modern Angular feature output compares directly against current `ngtsc`.
- Defer output has stable golden fixtures and runtime coverage.

## Phase 8 - Diagnostics Parity

### Tasks

- Add diagnostic comparison harness.
- Match diagnostics for:
  - unknown element;
  - unknown property;
  - unknown event;
  - missing pipe;
  - missing directive;
  - invalid two-way binding;
  - duplicate input;
  - duplicate output;
  - required input missing;
  - invalid host metadata;
  - invalid standalone imports;
  - invalid NgModule declarations;
  - private export errors;
  - cyclic imports where Angular reports them.
- Track:
  - code;
  - category;
  - source span;
  - related information if available.

### Acceptance Criteria

- Diagnostic tests are stable and not over-normalized.
- Real app template errors point to useful source locations.

## Phase 9 - Type Check Blocks

### Tasks

- Implement/complete TCB generation for:
  - template expressions;
  - directive inputs;
  - required inputs;
  - outputs;
  - DOM events;
  - local refs;
  - pipes;
  - generic directives;
  - generic components;
  - embedded views;
  - control flow blocks;
  - defer blocks.
- Support compiler options:
  - `strictTemplates`;
  - `fullTemplateTypeCheck`;
  - nullability modes;
  - input access modifiers if applicable.
- Add fixtures for:
  - generic components;
  - generic directives;
  - typed forms;
  - `$event` inference;
  - pipe return types.

### Acceptance Criteria

- Template type checking catches the same class of errors as Angular for covered strict mode cases.
- TCB output or diagnostics can be compared with Angular.

## Phase 10 - Incremental AOT

### Tasks

- Complete semantic graph implementation for component compilation.
- Track dependencies:
  - component TS file;
  - external template;
  - external style;
  - imported directive;
  - imported pipe;
  - NgModule scope;
  - standalone import graph;
  - `.d.ts` metadata.
- Add invalidation tests for:
  - TS edit;
  - HTML edit;
  - CSS/SCSS edit;
  - directive selector edit;
  - input/output edit;
  - pipe name edit;
  - NgModule import/export edit;
  - provider-only edit.
- Ensure daemon memory output is consistent after incremental updates.

### Acceptance Criteria

- No stale JS output after dependency edits.
- No unnecessary full program rebuild for local edits where Angular can be incremental.
- HMR affected component list is accurate.

## Phase 11 - Runtime Parity Suite

### Tasks

- Add runtime tests through `angular-go/testbed` for:
  - component creation;
  - inputs;
  - outputs;
  - signal inputs;
  - model inputs;
  - host bindings;
  - host listeners;
  - content projection;
  - queries;
  - structural directives;
  - control flow;
  - defer;
  - pipes;
  - forms;
  - router;
  - PrimeNG dropdown;
  - PrimeNG table;
  - PrimeNG menu.
- Add Playwright/Vite runtime tests for browser-only behavior:
  - overlays;
  - focus;
  - keyboard events;
  - animation/transition dependent UI.

### Acceptance Criteria

- Historical runtime regressions are covered.
- Runtime suite clearly distinguishes compiler bugs from testbed/browser limitations.

## Phase 12 - Performance Gate

### Tasks

- Add benchmark scenarios:
  - cold 1 component;
  - cold 100 components;
  - cold 1000 components;
  - cold 2000 components;
  - warm daemon no-op;
  - TS edit;
  - HTML edit;
  - CSS edit;
  - directive dependency edit;
  - NgModule scope edit.
- Record phase timings:
  - TS parse;
  - Angular analyze;
  - template parse;
  - binder;
  - render3 transform;
  - emit;
  - linker;
  - Vite transform;
  - testbed runtime render.
- Store benchmark output as JSON.
- Define thresholds:
  - no phase regression above accepted percentage;
  - cold start target;
  - warm rebuild target;
  - HMR target.

### Acceptance Criteria

- Every major parity phase can be checked against performance baseline.
- Performance regressions are visible in CI or local benchmark reports.

## Current Priority Queue

Start with these tasks:

1. Build Phase 1 golden harness or harden the existing harness if already present.
2. Add component metadata golden cases for all `@Component` fields.
3. Add standalone scope fixtures for `CommonModule`, `FormsModule`, and PrimeNG.
4. Add binder fixtures for nested templates, `NgTemplateOutlet`, and `$event.target`.
5. Add Render3 instruction fixtures for forms and projection.
6. Add diagnostics parity tests for unknown property/directive/pipe.
7. Add TCB smoke tests for strict template mode.
8. Add incremental invalidation tests for `.ts`, `.html`, `.css`, and imported directive edits.

