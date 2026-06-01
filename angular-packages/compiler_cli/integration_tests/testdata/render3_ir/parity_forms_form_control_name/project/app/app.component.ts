import {Component} from '@angular/core';
import {ReactiveFormsModule, FormGroup, FormControl} from '@angular/forms';

@Component({
  selector: 'app-root',
  imports: [ReactiveFormsModule],
  template: `
    <form [formGroup]="form">
      <input formControlName="username" />
    </form>
  `,
  standalone: true
})
export class AppComponent {
  form = new FormGroup({
    username: new FormControl('')
  });
}
