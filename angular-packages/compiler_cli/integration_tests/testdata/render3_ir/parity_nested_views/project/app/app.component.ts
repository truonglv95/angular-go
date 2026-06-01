import {Component} from '@angular/core';

@Component({
  selector: 'app-root',
  template: `
    <div>
      <ng-template>
        <div>
          <ng-template>
            <span>Nested {{ value }}</span>
          </ng-template>
        </div>
      </ng-template>
    </div>
  `,
  standalone: true
})
export class AppComponent {
  value = 42;
}
