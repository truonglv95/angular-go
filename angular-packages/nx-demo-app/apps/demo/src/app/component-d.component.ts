import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-component-d',
  template: '<div>component d {{ propD }}</div>',
  standalone: false
})
export class ComponentD {
  @Input() propD = 'd';
}
