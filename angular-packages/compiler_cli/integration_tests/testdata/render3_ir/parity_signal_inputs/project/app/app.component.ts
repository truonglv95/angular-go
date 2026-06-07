import {Component, input} from '@angular/core';

function uppercaseTransform(v: string): string {
  return v.toUpperCase();
}

@Component({
  selector: 'app-root',
  template: `
    <div>{{ name() }}</div>
    <div>{{ count() }}</div>
    <div>{{ aliasedName() }}</div>
    <div>{{ transformedName() }}</div>
  `,
  standalone: true
})
export class AppComponent {
  name = input<string>();
  count = input.required<number>();
  aliasedName = input('', {alias: 'customName'});
  transformedName = input('', {transform: uppercaseTransform});
}
