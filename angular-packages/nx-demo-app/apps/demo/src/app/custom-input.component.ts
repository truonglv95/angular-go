import { Component, forwardRef, Input, signal, computed, effect } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR, FormsModule } from '@angular/forms';
import { format } from 'date-fns';
import { BaseComponent } from './base.component';

@Component({
  selector: 'app-custom-input',
  standalone: true,
  imports: [FormsModule],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => CustomInputComponent),
      multi: true
    }
  ],
  template: `
    <div class="custom-input-container">
      <label i18n="@@customInputLabel">{{ baseTitle }} (YYYY-MM-DD):</label>
      <input 
        [ngModel]="value()" 
        (ngModelChange)="onInput($event)"
        (blur)="onTouched()"
        [disabled]="disabled()"
      />
      <p class="formatted-date" i18n="@@formattedDate">Formatted: {{ formattedDate() }}</p>
    </div>
  `,
  styles: [`
    .custom-input-container { padding: 10px; border: 1px solid #ccc; border-radius: 4px; }
    .formatted-date { color: #007bff; font-weight: bold; }
  `]
})
export class CustomInputComponent extends BaseComponent implements ControlValueAccessor {
  
  // Internal State with Signals
  value = signal<string>('');
  disabled = signal<boolean>(false);
  
  // Computed Signal with 3rd party lib
  formattedDate = computed(() => {
    try {
      if (!this.value()) return 'N/A';
      const d = new Date(this.value());
      if (isNaN(d.getTime())) return 'Invalid Date';
      return format(d, 'EEEE, MMMM do, yyyy');
    } catch {
      return 'Error';
    }
  });

  onChange = (val: any) => {};
  onTouched = () => {};

  constructor() {
    super();
    effect(() => {
      console.log('Value changed to:', this.value());
    });
  }

  onInput(val: string) {
    this.value.set(val);
    this.onChange(val);
  }

  writeValue(val: any): void {
    if (val !== undefined && val !== null) {
      this.value.set(val);
    } else {
      this.value.set('');
    }
  }
  
  registerOnChange(fn: any): void {
    this.onChange = fn;
  }
  
  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }
  
  setDisabledState(isDisabled: boolean): void {
    this.disabled.set(isDisabled);
  }
}
