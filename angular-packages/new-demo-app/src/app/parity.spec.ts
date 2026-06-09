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
  PipeTransform
} from '@angular/core';
import { GoTestBed, renderComponent } from '@angular-go/build/testbed';
import { FormsModule, ReactiveFormsModule, FormControl } from '@angular/forms';
import { CommonModule, UpperCasePipe } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { Button } from 'primeng/button';

// --- Test Components ---

@Injectable({ providedIn: 'root' })
class DataService {
  getData() {
    return 'real-data';
  }
}

@Component({
  selector: 'app-input-output',
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

@Component({
  selector: 'app-di',
  standalone: true,
  template: `<div class="data">{{ service.getData() }}</div>`
})
class DiComponent {
  service = angularInject(DataService);
}

@Component({
  selector: 'app-structural',
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
  selector: 'app-forms',
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

const MODULE_TOKEN = new InjectionToken<string>('MODULE_TOKEN');

@NgModule({
  providers: [{ provide: MODULE_TOKEN, useValue: 'module-provider' }]
})
class ProviderModule {}

@Component({
  selector: 'app-module-provider',
  standalone: true,
  template: `<div class="module-provider">{{ value }}</div>`
})
class ModuleProviderComponent {
  value = angularInject(MODULE_TOKEN);
}

@Component({
  selector: 'app-projection-card',
  standalone: true,
  template: `<section class="card"><ng-content></ng-content></section>`
})
class ProjectionCardComponent {}

@Component({
  selector: 'app-projection-host',
  standalone: true,
  imports: [ProjectionCardComponent],
  template: `<app-projection-card><span class="projected">Projected</span></app-projection-card>`
})
class ProjectionHostComponent {}

// --- New Components for Phase 11 Runtime Parity ---

@Component({
  selector: 'app-signals',
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
  selector: 'app-host-bindings',
  standalone: true,
  template: `<div>Host Binding Test</div>`
})
class HostBindingsComponent {
  @HostBinding('attr.role') role = 'button';
  @HostBinding('class.active') isActive = true;

  clickCount = 0;

  @HostListener('click', ['$event'])
  onClick(event: Event) {
    this.clickCount++;
  }
}

@Component({
  selector: 'app-child-item',
  standalone: true,
  template: `<div class="item-text">{{ name }}</div>`
})
class ChildItemComponent {
  @Input() name = '';
}

@Component({
  selector: 'app-queries',
  standalone: true,
  imports: [ChildItemComponent],
  template: `
    <app-child-item name="first" #firstRef></app-child-item>
    <app-child-item name="second"></app-child-item>
    <div #divRef class="my-div">Div Content</div>
  `
})
class QueriesComponent {
  @ViewChild('firstRef') firstChild!: ChildItemComponent;
  @ViewChild('divRef') divElement!: ElementRef<HTMLDivElement>;
  @ViewChildren(ChildItemComponent) allChildren!: QueryList<ChildItemComponent>;
}

@Component({
  selector: 'app-modern-control-flow',
  standalone: true,
  template: `
    @if (show) {
      <div class="visible-cf">VisibleCF</div>
    } @else {
      <div class="hidden-cf">HiddenCF</div>
    }

    @for (item of items; track item) {
      <div class="cf-item">{{ item }}</div>
    }

    @switch (status) {
      @case ('active') {
        <div class="status-active">Active</div>
      }
      @case ('inactive') {
        <div class="status-inactive">Inactive</div>
      }
      @default {
        <div class="status-unknown">Unknown</div>
      }
    }
  `
})
class ModernControlFlowComponent {
  show = true;
  items: string[] = ['one', 'two'];
  status = 'active';
}

@Component({
  selector: 'app-defer-block',
  standalone: true,
  template: `
    @defer (on timer(50ms)) {
      <div class="deferred-content">Deferred content loaded</div>
    } @placeholder {
      <div class="placeholder-content">Placeholder</div>
    } @loading {
      <div class="loading-content">Loading...</div>
    }
  `
})
class DeferBlockComponent {}

@Pipe({
  name: 'customPipe',
  standalone: true
})
class CustomPipe implements PipeTransform {
  transform(value: string, suffix: string = ''): string {
    return value.toUpperCase() + suffix;
  }
}

@Component({
  selector: 'app-pipe-test',
  standalone: true,
  imports: [UpperCasePipe, CustomPipe],
  template: `
    <div class="builtin">{{ text | uppercase }}</div>
    <div class="custom">{{ text | customPipe:'!!!' }}</div>
  `
})
class PipeTestComponent {
  text = 'hello';
}

@Component({
  selector: 'app-router-target',
  standalone: true,
  template: `<div class="target-route">Target Content</div>`
})
class RouterTargetComponent {}

@Component({
  selector: 'app-router-host',
  standalone: true,
  imports: [RouterModule],
  template: `
    <router-outlet></router-outlet>
  `
})
class RouterHostComponent {
  router = angularInject(Router);
}

@Component({
  selector: 'app-primeng-test',
  standalone: true,
  imports: [Button],
  template: `
    <p-button label="PrimeButton" class="p-btn"></p-button>
  `
})
class PrimeNgTestComponent {}

// --- Parity Specs ---

describe('Angular Go Testbed Parity Matrix', () => {
  describe('Input & Output Propagation', () => {
    it('should propagate inputs and emit outputs correctly', async () => {
      const fixture = await renderComponent(InputOutputComponent, {
        inputs: { title: 'Initial Title' }
      });

      expect(fixture.nativeElement.querySelector('.title')?.textContent).toBe('Initial Title');

      fixture.trigger('button', 'click');
      expect(fixture.emitted('notify')).toEqual(['clicked']);

      fixture.setInput('title', 'Updated Title');
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('.title')?.textContent).toBe('Updated Title');

      fixture.destroy();
    });
  });

  describe('Dependency Injection & Provider Override', () => {
    it('should inject real service by default', async () => {
      const fixture = await renderComponent(DiComponent);
      expect(fixture.nativeElement.querySelector('.data')?.textContent).toBe('real-data');
      fixture.destroy();
    });

    it('should support NgModule provider imports in render options', async () => {
      const fixture = await renderComponent(ModuleProviderComponent, {
        imports: [ProviderModule]
      });

      expect(fixture.nativeElement.querySelector('.module-provider')?.textContent).toBe('module-provider');
      fixture.destroy();
    });

    it('should allow overriding providers in options', async () => {
      const mockService = {
        getData() {
          return 'mocked-data';
        }
      };

      const fixture = await renderComponent(DiComponent, {
        providers: [{ provide: DataService, useValue: mockService }]
      });

      expect(fixture.nativeElement.querySelector('.data')?.textContent).toBe('mocked-data');
      fixture.destroy();
    });

    it('should allow overriding providers in GoTestBed', async () => {
      const mockService = {
        getData() {
          return 'mocked-testbed';
        }
      };

      GoTestBed.resetTestingModule();
      await GoTestBed.configureTestingModule({
        imports: [DiComponent],
        providers: [DataService]
      })
      .overrideProvider(DataService, { useValue: mockService })
      .compileComponents();

      const fixture = GoTestBed.createComponent(DiComponent);
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('.data')?.textContent).toBe('mocked-testbed');
      fixture.destroy();
    });
  });

  describe('Structural Directives (*ngIf, *ngFor)', () => {
    it('should render structural directives correctly', async () => {
      const fixture = await renderComponent(StructuralComponent, {
        inputs: { show: false, items: ['A', 'B'] }
      });

      expect(fixture.nativeElement.querySelector('.content')).toBeNull();
      expect(fixture.nativeElement.querySelectorAll('li').length).toBe(2);

      fixture.setInput('show', true);
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('.content')?.textContent).toBe('Visible');

      fixture.setInput('items', ['A', 'B', 'C']);
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelectorAll('li').length).toBe(3);

      fixture.destroy();
    });
  });

  describe('Content Projection', () => {
    it('should render projected content through a precompiled host component', async () => {
      const fixture = await renderComponent(ProjectionHostComponent);

      expect(fixture.nativeElement.querySelector('.card .projected')?.textContent).toBe('Projected');
      fixture.destroy();
    });
  });

  describe('Forms Support & Event Simulator', () => {
    it('should update reactive forms control value when typed', async () => {
      const fixture = await renderComponent(FormsComponent);

      fixture.type('.text-input', 'Hello Go Testbed');
      expect(fixture.component.control.value).toBe('Hello Go Testbed');
      expect(fixture.nativeElement.querySelector('.val')?.textContent).toBe('Hello Go Testbed');

      fixture.clear('.text-input');
      expect(fixture.component.control.value).toBe('');
      expect(fixture.nativeElement.querySelector('.val')?.textContent).toBe('');

      fixture.destroy();
    });
  });

  describe('Signal & Model Inputs', () => {
    it('should support signal inputs and model inputs', async () => {
      const fixture = await renderComponent(SignalsComponent, {
        inputs: { sigInput: 'hello-signal', modelVal: 42 }
      });

      expect(fixture.nativeElement.querySelector('.sig-val')?.textContent).toBe('hello-signal');
      expect(fixture.nativeElement.querySelector('.model-val')?.textContent).toBe('42');

      fixture.component.modelVal.set(100);
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('.model-val')?.textContent).toBe('100');

      fixture.destroy();
    });
  });

  describe('Host Bindings & Host Listeners', () => {
    it('should support host bindings and host listeners', async () => {
      const fixture = await renderComponent(HostBindingsComponent);
      
      expect(fixture.nativeElement.getAttribute('role')).toBe('button');
      expect(fixture.nativeElement.classList.contains('active')).toBe(true);
      expect(fixture.component.clickCount).toBe(0);

      fixture.nativeElement.click();
      fixture.detectChanges();
      expect(fixture.component.clickCount).toBe(1);

      fixture.destroy();
    });
  });

  describe('View & Content Queries', () => {
    it('should resolve view queries correctly', async () => {
      const fixture = await renderComponent(QueriesComponent);
      
      expect(fixture.component.firstChild).toBeDefined();
      expect(fixture.component.firstChild.name).toBe('first');
      
      expect(fixture.component.divElement).toBeDefined();
      expect(fixture.component.divElement.nativeElement.textContent).toBe('Div Content');
      
      expect(fixture.component.allChildren).toBeDefined();
      expect(fixture.component.allChildren.length).toBe(2);
      
      fixture.destroy();
    });
  });

  describe('Modern Control Flow (@if, @for, @switch)', () => {
    it('should render modern control flow correctly', async () => {
      const fixture = await renderComponent(ModernControlFlowComponent);
      
      expect(fixture.nativeElement.querySelector('.visible-cf')).not.toBeNull();
      expect(fixture.nativeElement.querySelector('.hidden-cf')).toBeNull();
      expect(fixture.nativeElement.querySelectorAll('.cf-item').length).toBe(2);
      expect(fixture.nativeElement.querySelector('.status-active')?.textContent).toBe('Active');

      fixture.component.show = false;
      fixture.component.items = ['one', 'two', 'three'];
      fixture.component.status = 'inactive';
      fixture.detectChanges();

      expect(fixture.nativeElement.querySelector('.visible-cf')).toBeNull();
      expect(fixture.nativeElement.querySelector('.hidden-cf')).not.toBeNull();
      expect(fixture.nativeElement.querySelectorAll('.cf-item').length).toBe(3);
      expect(fixture.nativeElement.querySelector('.status-inactive')?.textContent).toBe('Inactive');
      
      fixture.destroy();
    });
  });

  describe('Defer Blocks (@defer)', () => {
    it('should render defer block stages correctly', async () => {
      const fixture = await renderComponent(DeferBlockComponent);
      
      expect(fixture.nativeElement.querySelector('.placeholder-content')?.textContent).toBe('Placeholder');
      expect(fixture.nativeElement.querySelector('.deferred-content')).toBeNull();

      await new Promise(resolve => setTimeout(resolve, 100));
      fixture.detectChanges();

      expect(fixture.nativeElement.querySelector('.deferred-content')?.textContent).toBe('Deferred content loaded');
      expect(fixture.nativeElement.querySelector('.placeholder-content')).toBeNull();
      
      fixture.destroy();
    });
  });

  describe('Pipes Support', () => {
    it('should support built-in and custom pipes', async () => {
      const fixture = await renderComponent(PipeTestComponent);
      
      expect(fixture.nativeElement.querySelector('.builtin')?.textContent).toBe('HELLO');
      expect(fixture.nativeElement.querySelector('.custom')?.textContent).toBe('HELLO!!!');
      
      fixture.destroy();
    });
  });

  describe('Router Support', () => {
    it('should render router outlet and support navigation', async () => {
      const { provideRouter } = await import('@angular/router');
      
      const fixture = await renderComponent(RouterHostComponent, {
        providers: [
          provideRouter([
            { path: 'target', component: RouterTargetComponent }
          ])
        ]
      });

      const router = fixture.component.router;
      await router.navigateByUrl('/target');
      fixture.detectChanges();
      
      await fixture.whenStable();
      
      expect(fixture.nativeElement.querySelector('.target-route')?.textContent).toBe('Target Content');
      fixture.destroy();
    });
  });

  describe('PrimeNG Integration', () => {
    it('should render PrimeNG button correctly', async () => {
      const fixture = await renderComponent(PrimeNgTestComponent);
      const button = fixture.nativeElement.querySelector('.p-btn button');
      
      expect(button).not.toBeNull();
      expect(button?.textContent).toContain('PrimeButton');
      fixture.destroy();
    });
  });

  describe('Unsupported dynamic testing module APIs', () => {
    it('should fail fast for dynamic declarations', async () => {
      await expect(
        renderComponent(InputOutputComponent, {
          declarations: [InputOutputComponent]
        })
      ).rejects.toThrow('does not support dynamic declarations');
    });
  });
});
