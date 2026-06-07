import { Type, Provider, EnvironmentInjector, ProviderToken } from '@angular/core';
import { createApplication } from '@angular/platform-browser';
import { createComponent, runInInjectionContext } from '@angular/core';
import { GoComponentFixture, GoComponentFixtureImpl } from './fixture.js';
import { normalizeTestProviders } from './render.js';

export interface TestModuleMetadata {
  declarations?: any[];
  imports?: any[];
  providers?: Provider[];
}

export class GoTestBedFixture<T> {
  constructor(public fixture: GoComponentFixture<T>) {}

  get componentInstance(): T {
    return this.fixture.component;
  }

  get nativeElement(): HTMLElement {
    return this.fixture.nativeElement;
  }

  get debugElement() {
    return {
      nativeElement: this.fixture.nativeElement,
      query: (selector: string) => this.fixture.query(selector),
      queryAll: (selector: string) => this.fixture.queryAll(selector),
      componentInstance: this.fixture.component
    };
  }

  detectChanges(): void {
    this.fixture.detectChanges();
  }

  async whenStable(): Promise<void> {
    await this.fixture.whenStable();
  }

  destroy(): void {
    this.fixture.destroy();
  }
}

export class GoTestBedImpl {
  private providers: Provider[] = [];
  private imports: any[] = [];
  private declarations: any[] = [];
  private preparedAppRef: any = null;
  private fixtures = new Set<GoTestBedFixture<any>>();

  configureTestingModule(moduleDef: TestModuleMetadata): this {
    if (moduleDef.declarations) {
      this.declarations.push(...moduleDef.declarations);
    }
    if (moduleDef.providers) {
      this.providers.push(...moduleDef.providers);
    }
    if (moduleDef.imports) {
      this.imports.push(...moduleDef.imports);
    }
    return this;
  }

  async compileComponents(): Promise<void> {
    if (this.preparedAppRef) {
      return;
    }
    this.preparedAppRef = await createApplication({
      providers: normalizeTestProviders({
        providers: this.providers,
        imports: this.imports,
        declarations: this.declarations
      })
    });
  }

  createComponent<T>(component: Type<T>): GoTestBedFixture<T> {
    if (!this.preparedAppRef) {
      throw new Error(
        'GoTestBed: You must call compileComponents() and await its completion before calling createComponent().'
      );
    }

    const hostElement = document.createElement('div');
    document.body.appendChild(hostElement);

    const componentRef = runInInjectionContext(this.preparedAppRef.injector, () => {
      return createComponent(component, {
        environmentInjector: this.preparedAppRef.injector,
        hostElement: hostElement
      });
    });

    this.preparedAppRef.attachView(componentRef.hostView);
    componentRef.changeDetectorRef.detectChanges();

    let wrappedFixture!: GoTestBedFixture<T>;
    wrappedFixture = new GoTestBedFixture<T>(
      new GoComponentFixtureImpl(componentRef, this.preparedAppRef, hostElement, () => {
        this.fixtures.delete(wrappedFixture);
      })
    );
    this.fixtures.add(wrappedFixture);
    return wrappedFixture;
  }

  inject<T>(token: ProviderToken<T>): T {
    if (!this.preparedAppRef) {
      throw new Error('GoTestBed: Test module not compiled yet. Call compileComponents() first.');
    }
    return this.preparedAppRef.injector.get(token);
  }

  resetTestingModule(): this {
    for (const fixture of this.fixtures) {
      try {
        fixture.destroy();
      } catch {
        // ignore
      }
    }
    this.fixtures.clear();
    this.providers = [];
    this.imports = [];
    this.declarations = [];
    if (this.preparedAppRef) {
      try {
        this.preparedAppRef.destroy();
      } catch {
        // ignore
      }
      this.preparedAppRef = null;
    }
    return this;
  }

  overrideProvider(token: any, overrides: { useValue?: any; useClass?: any; useFactory?: any; useExisting?: any }): this {
    this.providers = this.providers.filter((p) => {
      if (p === token) return false;
      if (typeof p === 'object' && p !== null && 'provide' in p && p.provide === token) return false;
      return true;
    });

    if (overrides.useValue !== undefined) {
      this.providers.push({ provide: token, useValue: overrides.useValue });
    } else if (overrides.useClass !== undefined) {
      this.providers.push({ provide: token, useClass: overrides.useClass });
    } else if (overrides.useFactory !== undefined) {
      this.providers.push({ provide: token, useFactory: overrides.useFactory });
    } else if (overrides.useExisting !== undefined) {
      this.providers.push({ provide: token, useExisting: overrides.useExisting });
    }
    return this;
  }

  overrideTemplate(): never {
    throw new Error('GoTestBed: overrideTemplate() is not supported in fast harness mode.');
  }

  overrideComponent(): never {
    throw new Error('GoTestBed: overrideComponent() is not supported in fast harness mode.');
  }

  overrideDirective(): never {
    throw new Error('GoTestBed: overrideDirective() is not supported in fast harness mode.');
  }
}

export const GoTestBed = new GoTestBedImpl();
