import { Directive, inject } from '@angular/core';
import { FormBuilder } from '@angular/forms';

@Directive()
export class BaseDirective {
  protected readonly fb = inject(FormBuilder);
}
