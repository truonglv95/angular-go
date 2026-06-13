import { Component } from '@angular/core';
import { Observable, of } from 'rxjs';

@Component({
  selector: 'app-non-standalone',
  standalone: false,
  template: '<div>{{ data$ | async }} - final verify</div>'
})
export class NonStandaloneComponent {
  data$: Observable<string> = of('hello async');
}
