import { Component } from '@angular/core';

@Component({
  selector: 'app-component-a',
  template: '<app-component-c [data]="myData"></app-component-c>',
  standalone: false
})
export class ComponentA {
  myData = 'hello';
}
