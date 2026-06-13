import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { ModuleDashboardComponent } from './module-dashboard.component';

@NgModule({
  declarations: [ModuleDashboardComponent],
  imports: [
    CommonModule,
    RouterModule.forChild([
      {
        path: '',
        component: ModuleDashboardComponent
      }
    ])
  ]
})
export class ModuleDashboardModule {}
