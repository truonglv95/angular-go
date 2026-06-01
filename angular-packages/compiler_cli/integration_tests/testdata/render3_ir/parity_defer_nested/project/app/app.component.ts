import {Component} from '@angular/core';
import {HeavyComponent} from './heavy.component';

@Component({
  selector: 'app-root',
  imports: [HeavyComponent],
  template: `
    @defer {
      <div>
        Outer block
        @defer {
          <app-heavy />
        }
      </div>
    }
  `,
  standalone: true
})
export class AppComponent {
}
