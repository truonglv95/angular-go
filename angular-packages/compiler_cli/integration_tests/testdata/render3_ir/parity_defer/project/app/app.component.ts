import {Component} from '@angular/core';
import {HeavyComponent} from './heavy.component';


@Component({
  selector: 'app-root',
  imports: [HeavyComponent],
  template: `
    @defer (on viewport) {
      <app-heavy />
    } @placeholder {
      <div>Loading...</div>
    }
  `,
  standalone: true
})
export class AppComponent {
}
