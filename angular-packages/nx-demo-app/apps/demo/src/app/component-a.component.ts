import { Component, Input } from '@angular/core';
import { ExternalBase } from 'fake-lib';

@Component({
  selector: 'app-component-a',
  template: 'A',
  standalone: false
})
export class ComponentA extends ExternalBase {
  @Input() title = 'Component A';
}
