import {Component, TemplateRef, ViewChild, ViewContainerRef} from '@angular/core';
import {NgTemplateOutlet} from '@angular/common';

@Component({
  selector: 'app-root',
  imports: [NgTemplateOutlet],
  template: `
    <ng-template #outerTpl let-outerMsg="msg" let-user>
      <div>Outer: {{ outerMsg }} - {{ user }}</div>
      <ng-template #innerTpl let-innerMsg="msg">
        <div>Inner: {{ innerMsg }} inside {{ outerMsg }}</div>
        <button (click)="clickInner(innerMsg, outerMsg, user)">Click</button>
      </ng-template>
      <ng-container *ngTemplateOutlet="innerTpl; context: { msg: 'inner-message' }"></ng-container>
    </ng-template>
    
    <ng-container *ngTemplateOutlet="outerTpl; context: { msg: 'outer-message', $implicit: 'John' }"></ng-container>
  `,
  standalone: true
})
export class AppComponent {
  @ViewChild('outerTpl', {static: true}) outerTpl!: TemplateRef<any>;
  
  clickInner(inner: string, outer: string, user: string) {}
}
