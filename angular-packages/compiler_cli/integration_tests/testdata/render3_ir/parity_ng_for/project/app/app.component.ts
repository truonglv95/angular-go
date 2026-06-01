import {Component} from '@angular/core';
import {NgFor} from '@angular/common';

@Component({
  selector: 'app-root',
  imports: [NgFor],
  template: `
    <ul>
      <li *ngFor="let item of items; let i = index; trackBy: trackById">
        {{ i }}: {{ item.name }}
      </li>
    </ul>
  `,
  standalone: true
})
export class AppComponent {
  items = [{id: 1, name: 'A'}, {id: 2, name: 'B'}];
  trackById(index: number, item: any) { return item.id; }
}
