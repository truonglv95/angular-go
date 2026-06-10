import { NgModule } from '@angular/core';
import { ModuleB } from './module-b.module';
import { ComponentA } from './component-a.component';

const SHARED_MODULES = [ModuleB];

@NgModule({
  imports: [...SHARED_MODULES],
  declarations: [ComponentA],
  exports: [ComponentA]
})
export class ModuleA {}
