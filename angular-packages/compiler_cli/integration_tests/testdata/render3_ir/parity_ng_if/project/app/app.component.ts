import {Component} from '@angular/core';
import {NgIf} from '@angular/common';

@Component({
  selector: 'app-root',
  imports: [NgIf],
  template: `
    <div *ngIf="condition; else elseBlock">
      Hello IF!
    </div>
    <ng-template #elseBlock>
      Hello ELSE!
    </ng-template>
  `,
  standalone: true
})
export class AppComponent {
  condition = true;
}
