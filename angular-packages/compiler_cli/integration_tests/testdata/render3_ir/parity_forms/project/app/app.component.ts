import {Component} from '@angular/core';
import {FormsModule} from '@angular/forms';

@Component({
  selector: 'app-root',
  imports: [FormsModule],
  template: `
    <form>
      <input name="username" [(ngModel)]="username" />
    </form>
  `,
  standalone: true
})
export class AppComponent {
  username = '';
}
