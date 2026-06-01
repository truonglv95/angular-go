import {Component} from '@angular/core';
import {HeavyComponent} from './heavy.component';

@Component({
  selector: 'app-root',
  imports: [HeavyComponent],
  template: `
    @defer {
      <app-heavy />
    } @placeholder (minimum 500ms) {
      <div>Placeholder...</div>
    } @loading (after 100ms; minimum 1s) {
      <div>Loading...</div>
    } @error {
      <div>Error loading heavy component</div>
    }
  `,
  standalone: true
})
export class AppComponent {
}
