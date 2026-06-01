import {Component} from '@angular/core';
import {NgIf, NgTemplateOutlet} from '@angular/common';

@Component({
  selector: 'app-root',
  imports: [NgIf, NgTemplateOutlet],
  template: `
    <ng-container *ngTemplateOutlet="rowTpl; context: { $implicit: item, active: enabled }"></ng-container>

    <ng-template #rowTpl let-row let-active="active">
      <section *ngIf="active">
        <input #labelInput [value]="row.label" />
        <button (click)="select(row.id, labelInput.value)">Select</button>
      </section>
    </ng-template>
  `,
  standalone: true
})
export class AppComponent {
  enabled = true;
  item = {id: 7, label: 'Seven'};

  select(id: number, label: string) {}
}
