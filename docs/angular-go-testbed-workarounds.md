# Angular TestBed to GoTestBed Migration & Workaround Guide

This guide helps Angular developers migrate existing test suites from Angular's standard zone-based `TestBed` to `angular-go`'s fast, zone-less `GoTestBed` running under Vitest.

---

## 1. Summary of Major Differences

| Feature | Standard TestBed | GoTestBed (angular-go) | Workaround / Mitigation |
| :--- | :--- | :--- | :--- |
| **Component Type** | Supports declared (non-standalone) and standalone components. | Supports **standalone components** only. | Convert components to `standalone: true`, or compile them beforehand. |
| **Zone Semantics** | Zone-based by default (tracks microtasks/macrotasks via `zone.js`). | **Zone-less** (checks `isStable` via `ApplicationRef`). | Use standard `async/await` or Vitest time-mocking utilities. |
| **Async UT utilities** | `fakeAsync`, `tick()`, `flush()` | Unsupported. | Use Vitest `vi.useFakeTimers()` and `vi.advanceTimersByTime()`. |
| **Dynamic Overrides** | `overrideTemplate`, `overrideComponent`, `overrideDirective` | Unsupported at runtime. | Use dependency injection overrides via `overrideProvider()`. |

---

## 2. Migration Examples & Workarounds

### 2.1. Workaround for Dynamic Declarations (DIV-001)

In standard `TestBed`, you often declare non-standalone components under test on-the-fly:

```typescript
// ❌ Standard TestBed (Dynamic Declaration)
await TestBed.configureTestingModule({
  declarations: [MyComponent, MockChildComponent],
}).compileComponents();
```

`GoTestBed` requires components to be compiled beforehand. Non-standalone components cannot be dynamically declared in the test module:

```typescript
//  GoTestBed Workaround (Standalone Conversion)
// Ensure MyComponent is compiled as standalone: true
await GoTestBed.configureTestingModule({
  imports: [MyComponent], // Import it instead of declaring
}).compileComponents();
```

---

### 2.2. Workaround for Runtime Template Overrides (DIV-002)

Standard tests often replace template elements or child components dynamically:

```typescript
// ❌ Standard TestBed (Template Override)
TestBed.overrideComponent(MyComponent, {
  set: { template: '<div>Mocked Template</div>' }
});
```

Because `go-ngc` performs full AOT template compilation, templates cannot be modified at runtime. Instead, use dependency injection overrides to swap out child service dependencies or providers:

```typescript
//  GoTestBed Workaround (Provider Override)
await GoTestBed.configureTestingModule({
  imports: [MyComponent],
})
.overrideProvider(DataService, { useValue: mockDataService })
.compileComponents();
```

If you must mock a component's visual structure, create a separate standalone mock component class and configure it in your imports:

```typescript
@Component({
  selector: 'app-child',
  standalone: true,
  template: '<div>Mocked Child Component</div>'
})
class MockChildComponent {}
```

---

### 2.3. Workaround for `fakeAsync` and `tick()` (DIV-004)

Standard tests rely on Zone.js to fast-forward asynchronous code:

```typescript
// ❌ Standard TestBed (fakeAsync)
it('fetches data async', fakeAsync(() => {
  component.loadData();
  tick(500);
  expect(component.data).toBe(expected);
}));
```

Since `GoTestBed` runs zone-less, use Vitest's built-in time-mocking helpers instead:

```typescript
//  GoTestBed Workaround (Vitest Fake Timers)
it('fetches data async', async () => {
  vi.useFakeTimers();
  
  component.loadData();
  vi.advanceTimersByTime(500);
  
  expect(component.data).toBe(expected);
  
  vi.useRealTimers();
});
```

---

### 2.4. Handling Async Change Detection (`whenStable`)

When testing asynchronous changes (like HTTP request bindings or router events), await `fixture.whenStable()` using standard promises:

```typescript
it('resolves async bindings', async () => {
  const fixture = GoTestBed.createComponent(MyComponent);
  fixture.detectChanges();
  
  // Trigger async action
  fixture.componentInstance.triggerRequest();
  
  // Wait for all microtasks to resolve and component to stabilize
  await fixture.whenStable();
  
  // Run assertions
  expect(fixture.nativeElement.textContent).toContain('Data Resolved');
});
```
