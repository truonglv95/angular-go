import {Component} from '@angular/core';
import {TrackDirective} from './track.directive';

@Component({
  selector: 'host-cmp',
  standalone: true,
  imports: [TrackDirective],
  template: '<button [appTrack]="name">Pick</button>',
})
export class HostComponent {
  name = 'alpha';
}
