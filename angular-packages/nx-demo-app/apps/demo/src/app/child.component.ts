import { Component, Input } from '@angular/core';
import { BaseComponent } from './base.component';

@Component({
  selector: 'app-child',
  standalone: true,
  template: '<div>{{ baseTitle }}</div><div [title]="baseTitle"></div>',
})
export class ChildComponent extends BaseComponent {
  @Input() someChildProp = 'child';
}
