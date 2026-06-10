import { Component } from '@angular/core';

@Component({
  selector: 'icu-demo',
  template: `<span i18n="@@itemCount">{count, plural, =0 {no items} =1 {one item} other {{{count}} items}}</span>`
})
export class IcuDemoComponent {
  count = 0;
}
