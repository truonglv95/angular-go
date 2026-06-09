import { Component, Input, booleanAttribute, model, computed, effect, Directive, ElementRef, HostListener, Type } from '@angular/core';
import { ReactiveFormsModule, FormBuilder, Validators, FormGroup } from '@angular/forms';
import { NgComponentOutlet, NgIf } from '@angular/common';
import { Button } from 'primeng/button';
import { InputText } from 'primeng/inputtext';
import { Card } from 'primeng/card';

// 1. Directives & Host Directives
@Directive({
  selector: '[appHighlight]',
  standalone: true
})
export class HighlightDirective {
  @Input() appHighlight = '';

  constructor(private el: ElementRef) {}

  @HostListener('mouseenter') onMouseEnter() {
    this.el.nativeElement.style.backgroundColor = this.appHighlight || 'cyan';
  }
  @HostListener('mouseleave') onMouseLeave() {
    this.el.nativeElement.style.backgroundColor = '';
  }
}

@Directive({
  selector: '[appComplexHost]',
  standalone: true,
  hostDirectives: [
    {
      directive: HighlightDirective,
      inputs: ['appHighlight: highlightColor']
    }
  ]
})
export class ComplexHostDirective {
}

// 2. Dynamic Component
@Component({
  selector: 'app-dynamic-widget',
  standalone: true,
  template: `<div class="p-3 bg-purple-100 rounded border-purple-300 border">Hello from Dynamic Widget Component!</div>`
})
export class DynamicWidgetComponent {}

// 3. Child Component with Model Signal, Content Projection, Input Transforms
@Component({
  selector: 'app-complex-child',
  standalone: true,
  imports: [Button],
  template: `
    <div class="border p-4 rounded bg-surface-100">
      <h5 class="text-xl font-bold">Child Component</h5>
      <p>Is Special (Transformed boolean): <span class="font-semibold">{{ isSpecial }}</span></p>
      <div class="flex items-center gap-4 my-2">
        <p>Model Signal Counter: {{ counterModel() }}</p>
        <p-button label="Increment Model" (onClick)="incrementModel()" size="small" />
      </div>
      
      <div class="mt-4 p-2 bg-surface-200 border-l-4 border-primary">
        <strong class="block mb-2">Projected Header:</strong>
        <ng-content select="[header]"></ng-content>
      </div>
      
      <div class="mt-2 p-2 bg-white">
        <strong class="block mb-2">Projected Body:</strong>
        <ng-content></ng-content>
      </div>

      <div class="mt-2 p-2 bg-surface-200 border-r-4 border-primary text-right">
        <strong class="block mb-2">Projected Footer:</strong>
        <ng-content select="[footer]"></ng-content>
      </div>
    </div>
  `
})
export class ComplexChildComponent {
  @Input({ transform: booleanAttribute }) isSpecial: boolean = false;
  counterModel = model(0);

  incrementModel() {
    this.counterModel.update(c => c + 1);
  }
}

// 4. Main Complex Wrapper Component
@Component({
  selector: 'app-complex-feature',
  standalone: true,
  imports: [
    ReactiveFormsModule, NgComponentOutlet, NgIf,
    Button, InputText, Card,
    ComplexHostDirective, ComplexChildComponent
  ],
  template: `
    <div class="flex flex-col gap-6">
      
      <!-- Reactive Forms -->
      <p-card header="1. Reactive Forms & Signals">
        <form [formGroup]="myForm" (ngSubmit)="onSubmit()" class="flex flex-col gap-4 max-w-sm">
          <div class="flex flex-col gap-2">
            <label for="name">Name</label>
            <input pInputText id="name" formControlName="name" />
          </div>
          <div class="flex flex-col gap-2">
            <label for="email">Email</label>
            <input pInputText id="email" formControlName="email" />
          </div>
          <p-button type="submit" label="Submit Form" [disabled]="myForm.invalid" />
        </form>
        <p class="mt-4">Form Value: Name: {{ myForm.value.name }}, Email: {{ myForm.value.email }}</p>
        <p>Form Status: {{ myForm.status }}</p>
        
        <hr class="my-4" />
        
        <p class="font-semibold text-lg">Computed Signal: {{ computedState() }}</p>
      </p-card>

      <!-- Host Directives -->
      <p-card header="2. Host Directives">
        <div appComplexHost highlightColor="#ffeb3b" class="p-4 border rounded cursor-pointer inline-block">
          Hover over me! I use a Host Directive that maps "highlightColor" to "appHighlight" input of HighlightDirective.
        </div>
      </p-card>

      <!-- Model Signals & Content Projection -->
      <p-card header="3. Model Signals & Content Projection">
        <app-complex-child isSpecial="true" [(counterModel)]="parentCounter">
          <div header class="text-blue-600 font-medium">This is the Header Slot content</div>
          
          <p>This is the default content projection body. Parent counter: {{ parentCounter }}</p>
          <p-button label="Increment from Parent" (onClick)="parentCounter = parentCounter + 1" size="small" />

          <div footer class="text-gray-500 italic">This is the Footer Slot content</div>
        </app-complex-child>
      </p-card>

      <!-- Dynamic Components -->
      <p-card header="4. Dynamic Component Loading">
        <p-button label="Load Dynamic Widget" (onClick)="loadWidget()" size="small" class="mb-4" />
        <div class="mt-4">
          <ng-container *ngComponentOutlet="dynamicComponent"></ng-container>
        </div>
      </p-card>

      <!-- Defer with complex triggers -->
      <p-card header="5. Complex Defer Blocks">
        <div #hoverTrigger class="p-4 bg-gray-200 border border-gray-400 rounded cursor-pointer inline-block mb-4">
          Hover over this box to trigger &#64;defer loading
        </div>
        
        @defer (on hover(hoverTrigger)) {
          <div class="p-4 bg-green-100 border border-green-400 rounded">
            Successfully loaded via hover trigger!
          </div>
        } @placeholder {
          <div class="p-4 bg-surface-100 border border-surface-300 rounded text-gray-500">
            Waiting to be triggered... (Placeholder)
          </div>
        } @loading {
          <div class="p-4 bg-blue-100 border border-blue-400 rounded">
            Loading...
          </div>
        }
      </p-card>

    </div>
  `
})
export class ComplexFeatureComponent {
  myForm!: FormGroup;
  parentCounter = 10;
  dynamicComponent: Type<any> | null = null;

  // Computed signal based on form status and parent counter
  computedState = computed(() => {
    return `Counter: ${this.parentCounter} | Form Valid: ${this.myForm?.valid}`;
  });

  constructor(private fb: FormBuilder) {
    this.myForm = this.fb.group({
      name: ['', Validators.required],
      email: ['', [Validators.required, Validators.email]]
    });

    effect(() => {
      console.log('Effect triggered! Parent counter is now:', this.parentCounter);
    });
  }

  onSubmit() {
    console.log('Form Submitted!', this.myForm.value);
  }

  loadWidget() {
    this.dynamicComponent = DynamicWidgetComponent;
  }
}
 
// trigger incremental build
