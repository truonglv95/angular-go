import {Component} from '@angular/core';
import {HeavyComponent} from './heavy.component';

@Component({
  selector: 'app-root',
  imports: [HeavyComponent],
  template: `
    @defer (on viewport) {
      <app-heavy />
    } @placeholder {
      <div>Loading viewport...</div>
    }
    @defer (on idle) {
      <app-heavy />
    }
    @defer (on interaction) {
      <app-heavy />
    } @placeholder {
      <button>Click me</button>
    }
    @defer (when isReady) {
      <app-heavy />
    }
  `,
  standalone: true
})
export class AppComponent {
  isReady = false;
}
