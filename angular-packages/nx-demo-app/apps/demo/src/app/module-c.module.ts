import { NgModule } from '@angular/core';
import { ComponentC } from './component-c.component';

@NgModule({
  declarations: [ComponentC],
  exports: [ComponentC]
})
export class ModuleC {}
