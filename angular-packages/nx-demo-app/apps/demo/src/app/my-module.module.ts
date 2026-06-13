import { NgModule } from '@angular/core';
import { SharedModule } from './shared.module';
import { RouterModule } from '@angular/router';
import { NonStandaloneComponent } from './non-standalone.component';

@NgModule({
  imports: [SharedModule.forRoot(), RouterModule.forChild([])],
  declarations: [NonStandaloneComponent],
  exports: [NonStandaloneComponent]
})
export class MyModule {}
