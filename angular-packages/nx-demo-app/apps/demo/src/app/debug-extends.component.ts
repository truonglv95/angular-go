import { Component, inject } from '@angular/core';
import { BaseDirective } from '@demo/base.directive';

@Component({
  selector: 'app-child',
  standalone: true,
  template: '<div>child</div>',
})
export class ChildComponent extends BaseDirective {
  ngOnInit() {
    console.log(this.fb);
    const form = this.fb.group({
      name: ['']
    });
  }
}
