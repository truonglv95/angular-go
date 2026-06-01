import { Routes } from '@angular/router';
import { Component } from '@angular/core';

@Component({
  selector: 'dummy-route',
  template: '<div class="dummy-route-content">Routed Component Successfully Loaded!</div>'
})
export class DummyRouteComponent {}

export const routes: Routes = [
  { path: 'dummy', component: DummyRouteComponent }
];
