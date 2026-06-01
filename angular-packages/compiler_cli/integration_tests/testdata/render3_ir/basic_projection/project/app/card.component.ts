import {Component} from '@angular/core';

@Component({
  selector: 'app-card',
  standalone: true,
  template: `
    <section class="card">
      <header>
        <ng-content select="[card-title]"></ng-content>
      </header>
      <main>
        <ng-content></ng-content>
      </main>
    </section>
  `,
})
export class CardComponent {}
