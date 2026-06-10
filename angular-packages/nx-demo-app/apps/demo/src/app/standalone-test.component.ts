import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-stand-b',
  standalone: true,
  template: '<div>stand b {{ propB }}</div>'
})
export class StandaloneBComponent {
  @Input() propB = 'b';
}

@Component({
  selector: 'app-stand-a',
  standalone: true,
  imports: [StandaloneBComponent],
  template: '<app-stand-b [propB]="val"></app-stand-b>'
})
export class StandaloneAComponent {
  val = 'a';
}
