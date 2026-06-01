import {Component} from '@angular/core';
import {trigger, state, style, transition, animate} from '@angular/animations';

@Component({
  selector: 'app-root',
  template: `
    <div [@openClose]="isOpen ? 'open' : 'closed'">
      Animate me
    </div>
  `,
  standalone: true,
  animations: [
    trigger('openClose', [
      state('open', style({ height: '200px' })),
      state('closed', style({ height: '100px' })),
      transition('open => closed', [animate('1s')]),
      transition('closed => open', [animate('0.5s')])
    ])
  ]
})
export class AppComponent {
  isOpen = true;
}
