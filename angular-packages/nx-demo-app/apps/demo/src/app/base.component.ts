import { Directive, Input } from '@angular/core';

@Directive()
export class BaseComponent {
  @Input() baseTitle: string = '';
}
