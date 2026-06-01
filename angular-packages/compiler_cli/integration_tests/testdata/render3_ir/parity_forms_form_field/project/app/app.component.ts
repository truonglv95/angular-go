import {Component, Directive, Input} from '@angular/core';

@Directive({
  selector: '[formField]',
  standalone: true
})
export class FormFieldDirective {
  @Input() formField = '';
}

@Component({
  selector: 'app-root',
  imports: [FormFieldDirective],
  template: `
    <input [formField]="field" />
  `,
  standalone: true
})
export class AppComponent {
  field = 'name';
}
