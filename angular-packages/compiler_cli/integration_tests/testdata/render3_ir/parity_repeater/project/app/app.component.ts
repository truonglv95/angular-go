import {Component} from '@angular/core';

@Component({
  selector: 'app-root',
  template: `
    @for (item of items; track item.id; let idx = $index, e = $even) {
      <div>{{idx}}: {{item.name}} (even: {{e}})</div>
    } @empty {
      <div>No items</div>
    }
  `,
  standalone: true
})
export class AppComponent {
  items = [
    {id: 1, name: 'one'},
    {id: 2, name: 'two'}
  ];
}
