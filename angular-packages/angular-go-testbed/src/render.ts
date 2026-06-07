import { createApplication } from '@angular/platform-browser';
import {
  createComponent,
  Type,
  Provider,
  runInInjectionContext,
  EnvironmentInjector,
  ProviderToken,
  importProvidersFrom,
  EnvironmentProviders
} from '@angular/core';
import { GoComponentFixture, GoComponentFixtureImpl } from './fixture.js';

export interface RenderComponentOptions {
  providers?: Provider[];
  imports?: any[];
  declarations?: any[];
  inputs?: Record<string, any>;
  host?: HTMLElement | string;
  autoDetectChanges?: boolean;
}

const activeFixtures = new Set<GoComponentFixture<any>>();
let lastActiveInjector: EnvironmentInjector | null = null;

export function normalizeTestProviders(options: { providers?: Provider[]; imports?: any[]; declarations?: any[] }): Array<Provider | EnvironmentProviders> {
  if (options.declarations?.length) {
    throw new Error(
      'angular-go-testbed does not support dynamic declarations. Use standalone components compiled by go-ngc, or render a precompiled host component.'
    );
  }

  const providers: Array<Provider | EnvironmentProviders> = [...(options.providers ?? [])];
  const moduleImports = (options.imports ?? []).filter((entry) => shouldImportProvidersFrom(entry));
  if (moduleImports.length > 0) {
    providers.push(importProvidersFrom(...moduleImports));
  }
  return providers;
}

export async function renderComponent<T>(
  component: Type<T>,
  options: RenderComponentOptions = {}
): Promise<GoComponentFixture<T>> {
  const appRef = await createApplication({
    providers: normalizeTestProviders(options)
  });

  const hostElement = resolveHostElement(options.host);

  // Run component creation inside the application's injection context
  const componentRef = runInInjectionContext(appRef.injector, () => {
    return createComponent(component, {
      environmentInjector: appRef.injector,
      hostElement: hostElement
    });
  });

  appRef.attachView(componentRef.hostView);

  if (options.inputs) {
    for (const [key, value] of Object.entries(options.inputs)) {
      componentRef.setInput(key, value);
    }
  }

  if (options.autoDetectChanges !== false) {
    componentRef.changeDetectorRef.detectChanges();
  }

  let fixture!: GoComponentFixture<T>;
  fixture = new GoComponentFixtureImpl(componentRef, appRef, hostElement, () => {
    activeFixtures.delete(fixture);
  });
  activeFixtures.add(fixture);
  lastActiveInjector = componentRef.injector as EnvironmentInjector;

  return fixture;
}

export function inject<T>(token: ProviderToken<T>): T {
  if (!lastActiveInjector) {
    throw new Error('No active testbed environment. Call renderComponent() first.');
  }
  return lastActiveInjector.get(token);
}

export function cleanupTestbed() {
  for (const fixture of activeFixtures) {
    try {
      fixture.destroy();
    } catch {
      // ignore
    }
  }
  activeFixtures.clear();
  lastActiveInjector = null;
}

function resolveHostElement(host?: HTMLElement | string): HTMLElement {
  if (host instanceof HTMLElement) {
    if (!host.parentNode) {
      document.body.appendChild(host);
    }
    return host;
  }

  const hostElement = document.createElement('div');
  if (typeof host === 'string') {
    hostElement.setAttribute('data-go-testbed-host', host);
  }
  document.body.appendChild(hostElement);
  return hostElement;
}

function shouldImportProvidersFrom(entry: any): boolean {
  if (!entry) return false;
  if (entry.ngModule) return true;
  if (entry.ɵmod) return true;
  if (entry.ɵinj && !entry.ɵcmp && !entry.ɵdir && !entry.ɵpipe) return true;
  return false;
}
