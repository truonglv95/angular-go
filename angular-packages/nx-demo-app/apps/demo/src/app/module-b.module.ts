import { NgModule } from '@angular/core';
import { ModuleC } from './module-c.module';

@NgModule({
  imports: [ModuleC],
  exports: [ModuleC]
})
export class ModuleB {}
