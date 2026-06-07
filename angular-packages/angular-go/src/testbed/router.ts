import { provideRouter, Routes, RouterFeatures, Router, ActivatedRoute } from '@angular/router';
import { inject } from './render.js';

export function provideTestRouter(routes: Routes, ...features: RouterFeatures[]) {
  return provideRouter(routes, ...features);
}

export async function navigateByUrl(url: string): Promise<boolean> {
  const router = inject(Router);
  return router.navigateByUrl(url);
}

export function currentUrl(): string {
  const router = inject(Router);
  return router.url;
}

export function findByRoute(): ActivatedRoute {
  return inject(ActivatedRoute);
}
