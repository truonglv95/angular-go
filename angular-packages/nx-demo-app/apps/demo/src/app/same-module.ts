import { Component, Input, NgModule } from '@angular/core';

@Component({
  selector: 'non-standalone',
  template: '<div>Non Standalone</div>',
  standalone: false
})
export class NonStandaloneComponent {}

@NgModule({
  declarations: [NonStandaloneComponent],
  exports: [NonStandaloneComponent]
})
export class NonStandaloneModule {}

@Component({
  selector: 'app-comp-d',
  standalone: true,
  template: '<div>D</div>'
})
export class ComponentD {
  @Input() dInput!: string;
}

@Component({
  selector: 'app-comp-c',
  standalone: true,
  imports: [ComponentD],
  template: '<app-comp-d [dInput]="\'test\'"></app-comp-d>'
})
export class ComponentC {
}


