import { Attribute, Component } from '@angular/core';

@Component({
  selector: 'x-attr',
  standalone: true,
  template: '{{ mode }}',
})
export class AttrCmp {
  constructor(@Attribute('data-mode') readonly mode: string | null) {}
}
