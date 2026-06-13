import { Route } from '@angular/router';

export const appRoutes: Route[] = [
  {
    path: 'widgets/module',
    loadChildren: () =>
      import('@demo/widgets/module-dashboard/module-dashboard.module').then(
        (m) => m.ModuleDashboardModule
      ),
  },
  {
    path: 'widgets/standalone',
    loadComponent: () =>
      import('@demo/widgets/standalone-insights/standalone-insights.component').then(
        (m) => m.StandaloneInsightsComponent
      ),
  },
  {
    path: 'compiler-stress',
    loadComponent: () =>
      import('./compiler-stress.component').then((m) => m.CompilerStressComponent),
  },
];
