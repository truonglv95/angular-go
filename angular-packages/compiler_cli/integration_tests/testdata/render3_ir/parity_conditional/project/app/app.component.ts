import {Component} from '@angular/core';

@Component({
  selector: 'app-root',
  template: `
    @if (val === 1) {
      <div>One</div>
    } @else if (val === 2) {
      <div>Two</div>
    } @else {
      <div>Other</div>
    }
  `,
  standalone: true
})
export class AppComponent {
  val = 1;
}
