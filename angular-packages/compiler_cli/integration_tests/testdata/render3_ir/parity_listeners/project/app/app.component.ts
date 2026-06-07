import {Component} from '@angular/core';

@Component({
  selector: 'app-root',
  template: `
    <button (click)="onClick($event)">Click me</button>
    <input (input)="onInput($event.target)" />
  `,
  standalone: true
})
export class AppComponent {
  onClick(event: Event) {
    console.log(event);
  }
  onInput(target: any) {
    console.log(target);
  }
}
