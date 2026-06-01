import {Component} from '@angular/core';
import {HeavyComponent as HeavyAlias} from './deferred';

@Component({
  selector: 'app-root',
  imports: [HeavyAlias],
  template: `
    @defer (on interaction(trigger)) {
      <app-heavy />
    } @placeholder {
      <button #trigger type="button">Load</button>
    }
  `,
  standalone: true
})
export class AppComponent {
}
