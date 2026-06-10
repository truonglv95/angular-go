import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-component-c',
  template: '<div>{{ data }}</div>',
  standalone: false
})
export class ComponentC {
  @Input() data: string = '';
}
