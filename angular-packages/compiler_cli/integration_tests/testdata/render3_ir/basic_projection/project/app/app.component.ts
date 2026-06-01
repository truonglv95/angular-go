import {Component} from '@angular/core';
import {CardComponent} from './card.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CardComponent],
  template: `
    <app-card>
      <h1 card-title>Dashboard</h1>
      <p>Ready</p>
    </app-card>
  `,
})
export class AppComponent {}
