import {
  Component,
  Input,
  Output,
  EventEmitter,
  inject as angularInject,
  Injectable,
  InjectionToken,
  NgModule,
  input,
  model,
  HostBinding,
  HostListener,
  ViewChild,
  ViewChildren,
  ElementRef,
  QueryList,
  Pipe,
  PipeTransform,
  Type
} from '@angular/core';
import { GoTestBed, renderComponent } from '@angular-go/build/testbed';
import { TestBed } from '@angular/core/testing';
import { BrowserTestingModule, platformBrowserTesting } from '@angular/platform-browser/testing';
import { FormsModule, ReactiveFormsModule, FormControl } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { describe, it, expect, beforeAll } from 'vitest';

// --- Initialize TestBed Environment ---
beforeAll(() => {
  try {
    TestBed.initTestEnvironment(
      BrowserTestingModule,
      platformBrowserTesting()
    );
  } catch {
    // already initialized
  }
});

// --- Test Components ---

@Component({
  selector: 'app-matrix-input-output',
  standalone: true,
  template: `
    <div class="title">{{ title }}</div>
    <button (click)="notify.emit('clicked')">Click Me</button>
  `
})
class InputOutputComponent {
  @Input() title = 'Default';
  @Output() notify = new EventEmitter<string>();
}

@Injectable({ providedIn: 'root' })
class DataService {
  getData() {
    return 'real-data';
  }
}

@Component({
  selector: 'app-matrix-di',
  standalone: true,
  template: `<div class="data">{{ service.getData() }}</div>`
})
class DiComponent {
  service = angularInject(DataService);
}

@Component({
  selector: 'app-matrix-structural',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div *ngIf="show" class="content">Visible</div>
    <ul>
      <li *ngFor="let item of items">{{ item }}</li>
    </ul>
  `
})
class StructuralComponent {
  @Input() show = false;
  @Input() items: string[] = [];
}

@Component({
  selector: 'app-matrix-forms',
  standalone: true,
  imports: [FormsModule, ReactiveFormsModule],
  template: `
    <input [formControl]="control" class="text-input" />
    <div class="val">{{ control.value }}</div>
  `
})
class FormsComponent {
  control = new FormControl('');
}

@Component({
  selector: 'app-matrix-signals',
  standalone: true,
  template: `
    <div class="sig-val">{{ sigInput() }}</div>
    <div class="model-val">{{ modelVal() }}</div>
  `
})
class SignalsComponent {
  sigInput = input<string>('default-sig');
  modelVal = model<number>(0);
}

@Component({
  selector: 'app-matrix-host-bindings',
  standalone: true,
  template: `<div>Host Binding</div>`
})
class HostBindingsComponent {
  @HostBinding('attr.role') role = 'button';
  @HostBinding('class.active') isActive = true;
  clickCount = 0;

  @HostListener('click')
  onClick() {
    this.clickCount++;
  }
}

@Component({
  selector: 'app-matrix-child',
  standalone: true,
  template: `<div class="child">{{ name }}</div>`
})
class ChildComponent {
  @Input() name = '';
}

@Component({
  selector: 'app-matrix-queries',
  standalone: true,
  imports: [ChildComponent],
  template: `
    <app-matrix-child name="first" #childRef></app-matrix-child>
    <app-matrix-child name="second"></app-matrix-child>
    <div #divRef class="my-div">Div Content</div>
  `
})
class QueriesComponent {
  @ViewChild('childRef') firstChild!: ChildComponent;
  @ViewChild('divRef') divElement!: ElementRef<HTMLDivElement>;
  @ViewChildren(ChildComponent) allChildren!: QueryList<ChildComponent>;
}

@Component({
  selector: 'app-matrix-control-flow',
  standalone: true,
  template: `
    @if (show) {
      <div class="cf-visible">Visible</div>
    } @else {
      <div class="cf-hidden">Hidden</div>
    }
    @for (item of items; track item) {
      <span class="item">{{ item }}</span>
    }
  `
})
class ControlFlowComponent {
  @Input() show = true;
  @Input() items: string[] = ['A', 'B'];
}

// --- Paired Verification Helper ---
async function verifyParity<T>(
  component: Type<T>,
  options: {
    inputs?: Record<string, any>;
    providers?: any[];
  } = {},
  verifyFn: (fixture: {
    nativeElement: HTMLElement;
    component: T;
    setInput: (name: string, value: any) => void;
    detectChanges: () => void;
  }) => void | Promise<void>
) {
  // 1. GoTestBed (angular-go)
  GoTestBed.resetTestingModule();
  const goConfig: any = { imports: [component] };
  if (options.providers) {
    goConfig.providers = options.providers;
  }
  await GoTestBed.configureTestingModule(goConfig).compileComponents();
  const goFixture = GoTestBed.createComponent(component);
  if (options.inputs) {
    for (const [k, v] of Object.entries(options.inputs)) {
      goFixture.fixture.setInput(k, v);
    }
  }
  goFixture.detectChanges();

  // 2. Standard Angular TestBed
  TestBed.resetTestingModule();
  const stdConfig: any = { imports: [component] };
  if (options.providers) {
    stdConfig.providers = options.providers;
  }
  TestBed.configureTestingModule(stdConfig);
  const stdFixture = TestBed.createComponent(component);
  if (options.inputs) {
    for (const [k, v] of Object.entries(options.inputs)) {
      stdFixture.componentRef.setInput(k, v);
    }
  }
  stdFixture.detectChanges();

  // 3. Verify side-by-side
  await verifyFn({
    nativeElement: goFixture.nativeElement,
    component: goFixture.componentInstance || goFixture.component,
    setInput: (name: string, value: any) => {
      goFixture.fixture.setInput(name, value);
    },
    detectChanges: () => goFixture.detectChanges()
  });

  await verifyFn({
    nativeElement: stdFixture.nativeElement,
    component: stdFixture.componentInstance,
    setInput: (name: string, value: any) => {
      stdFixture.componentRef.setInput(name, value);
    },
    detectChanges: () => stdFixture.detectChanges()
  });

  // 4. Assert identical DOM output
  expect(goFixture.nativeElement.innerHTML.replace(/\s+/g, ' ').trim()).toBe(
    stdFixture.nativeElement.innerHTML.replace(/\s+/g, ' ').trim()
  );

  goFixture.destroy();
  stdFixture.destroy();
}

// --- Parity Matrix Tests ---
describe('Phase 13 Compiler/Linker Parity Test Matrix', () => {

  it('DOM Output & Input/Output Parity', async () => {
    let emitted: string[] = [];

    await verifyParity(
      InputOutputComponent,
      { inputs: { title: 'Parity Title' } },
      async ({ nativeElement, component, detectChanges }) => {
        expect(nativeElement.querySelector('.title')?.textContent).toBe('Parity Title');
        
        const sub = component.notify.subscribe((val: string) => emitted.push(val));
        const btn = nativeElement.querySelector('button');
        btn?.click();
        detectChanges();
        sub.unsubscribe();
      }
    );

    expect(emitted).toEqual(['clicked', 'clicked']);
  });

  it('Input propagation changes DOM identical', async () => {
    await verifyParity(
      InputOutputComponent,
      { inputs: { title: 'First' } },
      async ({ nativeElement, setInput, detectChanges }) => {
        expect(nativeElement.querySelector('.title')?.textContent).toBe('First');
        setInput('title', 'Second');
        detectChanges();
        expect(nativeElement.querySelector('.title')?.textContent).toBe('Second');
      }
    );
  });

  it('Dependency Injection & Provider Override Parity', async () => {
    const mockService = {
      getData() {
        return 'mocked-parity-data';
      }
    };

    await verifyParity(
      DiComponent,
      {
        providers: [{ provide: DataService, useValue: mockService }]
      },
      async ({ nativeElement }) => {
        expect(nativeElement.querySelector('.data')?.textContent).toBe('mocked-parity-data');
      }
    );
  });

  it('Structural Directives (*ngIf, *ngFor) Parity', async () => {
    await verifyParity(
      StructuralComponent,
      { inputs: { show: true, items: ['X', 'Y', 'Z'] } },
      async ({ nativeElement, setInput, detectChanges }) => {
        expect(nativeElement.querySelector('.content')?.textContent).toBe('Visible');
        expect(nativeElement.querySelectorAll('li').length).toBe(3);

        setInput('show', false);
        setInput('items', ['X']);
        detectChanges();

        expect(nativeElement.querySelector('.content')).toBeNull();
        expect(nativeElement.querySelectorAll('li').length).toBe(1);
      }
    );
  });

  it('Forms & Interactive State Parity', async () => {
    await verifyParity(FormsComponent, {}, async ({ nativeElement, detectChanges }) => {
      expect(nativeElement.querySelector('.val')?.textContent).toBe('');
      
      const inputEl = nativeElement.querySelector('.text-input') as HTMLInputElement;
      inputEl.value = 'Typed Val';
      inputEl.dispatchEvent(new Event('input'));
      detectChanges();
      
      expect(nativeElement.querySelector('.val')?.textContent).toBe('Typed Val');
    });
  });

  it('Signal Inputs & Model Inputs Parity', async () => {
    await verifyParity(
      SignalsComponent,
      { inputs: { sigInput: 'signal-parity', modelVal: 99 } },
      async ({ nativeElement, setInput, detectChanges }) => {
        expect(nativeElement.querySelector('.sig-val')?.textContent).toBe('signal-parity');
        expect(nativeElement.querySelector('.model-val')?.textContent).toBe('99');

        setInput('modelVal', 101);
        detectChanges();
        expect(nativeElement.querySelector('.model-val')?.textContent).toBe('101');
      }
    );
  });

  it('Host Bindings & Host Listeners Parity', async () => {
    await verifyParity(HostBindingsComponent, {}, async ({ nativeElement, component, detectChanges }) => {
      expect(nativeElement.getAttribute('role')).toBe('button');
      expect(nativeElement.classList.contains('active')).toBe(true);

      nativeElement.click();
      detectChanges();
      expect(component.clickCount).toBe(1);
    });
  });

  it('View & Content Queries Parity', async () => {
    await verifyParity(QueriesComponent, {}, async ({ component }) => {
      expect(component.firstChild).toBeDefined();
      expect(component.firstChild.name).toBe('first');
      expect(component.divElement.nativeElement.textContent).toBe('Div Content');
      expect(component.allChildren.length).toBe(2);
    });
  });

  it('Modern Control Flow (@if, @for) Parity', async () => {
    await verifyParity(ControlFlowComponent, {}, async ({ nativeElement, setInput, detectChanges }) => {
      expect(nativeElement.querySelector('.cf-visible')).not.toBeNull();
      expect(nativeElement.querySelectorAll('.item').length).toBe(2);

      setInput('show', false);
      setInput('items', ['A', 'B', 'C']);
      detectChanges();

      expect(nativeElement.querySelector('.cf-visible')).toBeNull();
      expect(nativeElement.querySelector('.cf-hidden')).not.toBeNull();
      expect(nativeElement.querySelectorAll('.item').length).toBe(3);
    });
  });
});
