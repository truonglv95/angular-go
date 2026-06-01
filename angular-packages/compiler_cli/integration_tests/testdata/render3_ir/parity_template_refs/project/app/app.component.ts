import {Component} from '@angular/core';

@Component({
  selector: 'app-root',
  template: `
    <input #myInput type="text" />
    <button (click)="submit(myInput.value)">Submit</button>
  `,
  standalone: true
})
export class AppComponent {
  submit(val: string) {}
}
