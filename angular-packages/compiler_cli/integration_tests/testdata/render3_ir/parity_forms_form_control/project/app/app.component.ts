import {Component} from '@angular/core';
import {ReactiveFormsModule, FormControl} from '@angular/forms';

@Component({
  selector: 'app-root',
  imports: [ReactiveFormsModule],
  template: `
    <input [formControl]="ctrl" />
  `,
  standalone: true
})
export class AppComponent {
  ctrl = new FormControl('');
}
