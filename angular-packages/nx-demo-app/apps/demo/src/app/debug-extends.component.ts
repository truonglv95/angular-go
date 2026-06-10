import { Component, Input } from '@angular/core';

@Component({ template: '' })
export class BaseComponent {
  @Input() someBaseProp = 'base';
}

@Component({
  selector: 'app-child',
  standalone: true,
  template: '<div>{{ someBaseProp }}</div>',
})
export class ChildComponent extends BaseComponent {
  @Input() someChildProp = 'child';
}
