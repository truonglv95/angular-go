import {Component} from '@angular/core';
import {ShoutPipe} from './shout.pipe';

@Component({
  selector: 'pipe-cmp',
  standalone: true,
  imports: [ShoutPipe],
  template: '<p>{{ name | shout }}</p><input [value]="name">',
})
export class PipeComponent {
  name = 'ready';
}
