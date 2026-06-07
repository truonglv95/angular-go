import { Component, Input, Output, EventEmitter, inject as angularInject, Injectable, InjectionToken, NgModule } from '@angular/core';
import { GoTestBed, renderComponent } from 'angular-go/testbed';
import { FormsModule, ReactiveFormsModule, FormControl } from '@angular/forms';
import { CommonModule } from '@angular/common';

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
