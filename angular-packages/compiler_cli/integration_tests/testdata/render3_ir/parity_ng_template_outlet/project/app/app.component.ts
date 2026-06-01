import {Component} from '@angular/core';
import {NgTemplateOutlet} from '@angular/common';

@Component({
  selector: 'app-root',
  imports: [NgTemplateOutlet],
  template: `
    <ng-container *ngTemplateOutlet="myTpl; context: { $implicit: 'world' }"></ng-container>
    <ng-template #myTpl let-msg>
      Hello {{ msg }}!
    </ng-template>
  `,
  standalone: true
})
export class AppComponent {
}
