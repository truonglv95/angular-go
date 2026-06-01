import {Component, Pipe, PipeTransform} from '@angular/core';

@Pipe({name: 'uppercase', standalone: true})
export class UppercasePipe implements PipeTransform {
  transform(value: string) { return value.toUpperCase(); }
}

@Component({
  selector: 'app-root',
  imports: [UppercasePipe],
  template: `
    <div>{{ title | uppercase }}</div>
  `,
  standalone: true
})
export class AppComponent {
  title = 'hello';
}
