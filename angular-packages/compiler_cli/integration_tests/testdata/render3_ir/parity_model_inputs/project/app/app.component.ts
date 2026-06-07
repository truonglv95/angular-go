import {Component, model} from '@angular/core';

@Component({
  selector: 'app-root',
  template: `
    <div>{{ value() }}</div>
    <div>{{ count() }}</div>
    <div>{{ customVal() }}</div>
  `,
  standalone: true
})
export class AppComponent {
  value = model<string>();
  count = model.required<number>();
  customVal = model(0, {alias: 'customName'});
}
