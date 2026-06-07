import { ComponentRef, ApplicationRef, EnvironmentInjector } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { filter, first } from 'rxjs/operators';
import { type, clear, check, select, dispatch } from './events.js';

export interface GoComponentFixture<T> {
  component: T;
  componentRef: ComponentRef<T>;
  element: HTMLElement;
  hostElement: HTMLElement;
  nativeElement: HTMLElement;
  injector: EnvironmentInjector;
  detectChanges(): Promise<void> | void;
  whenStable(): Promise<void>;
  setInput(name: string, value: unknown): Promise<void> | void;
  query(selector: string): HTMLElement | null;
  queryAll(selector: string): HTMLElement[];
  output(name: string): any;
  emitted(name: string): any[];
  trigger(selector: string, eventName: string, eventObj?: any): void;
  type(selector: string, text: string): void;
  clear(selector: string): void;
  check(selector: string, checked?: boolean): void;
  select(selector: string, value: string): void;
  destroy(): void;
}

export class GoComponentFixtureImpl<T> implements GoComponentFixture<T> {
  private outputEmissions = new Map<string, any[]>();
  private subscriptions: any[] = [];
  private destroyed = false;

  constructor(
    public componentRef: ComponentRef<T>,
    private appRef: ApplicationRef,
    public hostElement: HTMLElement,
    private onDestroy?: () => void
  ) {
    this.setupOutputTracking();
  }

  get component(): T {
    return this.componentRef.instance;
  }

  get element(): HTMLElement {
    return this.hostElement;
  }

  get nativeElement(): HTMLElement {
    return this.componentRef.location.nativeElement;
  }

  get injector(): EnvironmentInjector {
    return this.componentRef.injector as EnvironmentInjector;
  }

  detectChanges(): Promise<void> | void {
    this.assertNotDestroyed();
    this.componentRef.changeDetectorRef.detectChanges();
  }

  async whenStable(): Promise<void> {
    const isStable$ = this.appRef.isStable;
    await firstValueFrom(
      isStable$.pipe(
        filter((stable) => stable),
        first()
      )
    );
  }

  setInput(name: string, value: unknown): Promise<void> | void {
    this.assertNotDestroyed();
    this.componentRef.setInput(name, value);
  }

  query(selector: string): HTMLElement | null {
    return this.nativeElement.querySelector(selector);
  }

  queryAll(selector: string): HTMLElement[] {
    return Array.from(this.nativeElement.querySelectorAll(selector));
  }

  output(name: string): any {
    this.assertNotDestroyed();
    return (this.component as any)[name];
  }

  emitted(name: string): any[] {
    this.assertNotDestroyed();
    return this.outputEmissions.get(name) ?? [];
  }

  trigger(selector: string, eventName: string, eventObj?: any): void {
    const element = this.query(selector);
    if (!element) {
      throw new Error(`Element not found for selector: "${selector}"`);
    }
    dispatch(element, eventName, eventObj);
    this.detectChanges();
  }

  type(selector: string, text: string): void {
    const element = this.query(selector);
    if (!element) {
      throw new Error(`Element not found for selector: "${selector}"`);
    }
    type(element, text);
    this.detectChanges();
  }

  clear(selector: string): void {
    const element = this.query(selector);
    if (!element) {
      throw new Error(`Element not found for selector: "${selector}"`);
    }
    clear(element);
    this.detectChanges();
  }

  check(selector: string, checked = true): void {
    const element = this.query(selector);
    if (!element) {
      throw new Error(`Element not found for selector: "${selector}"`);
    }
    check(element, checked);
    this.detectChanges();
  }

  select(selector: string, value: string): void {
    const element = this.query(selector);
    if (!element) {
      throw new Error(`Element not found for selector: "${selector}"`);
    }
    select(element, value);
    this.detectChanges();
  }

  private setupOutputTracking() {
    const cmpDef = (this.component as any).constructor?.ɵcmp;
    if (cmpDef && cmpDef.outputs) {
      for (const [propName, templateName] of Object.entries(cmpDef.outputs)) {
        const emitter = (this.component as any)[propName];
        if (emitter && typeof emitter.subscribe === 'function') {
          const emissions: any[] = [];
          this.outputEmissions.set(propName, emissions);
          if (templateName !== propName) {
            this.outputEmissions.set(templateName as string, emissions);
          }
          const sub = emitter.subscribe((val: any) => {
            emissions.push(val);
          });
          this.subscriptions.push(sub);
        }
      }
    }
  }

  destroy(): void {
    if (this.destroyed) return;
    this.destroyed = true;
    for (const sub of this.subscriptions) {
      if (sub && typeof sub.unsubscribe === 'function') {
        sub.unsubscribe();
      }
    }
    this.subscriptions = [];
    this.componentRef.destroy();
    this.appRef.destroy();
    if (this.hostElement.parentNode) {
      this.hostElement.parentNode.removeChild(this.hostElement);
    }
    this.onDestroy?.();
  }

  private assertNotDestroyed(): void {
    if (this.destroyed) {
      throw new Error('Cannot use a destroyed GoComponentFixture.');
    }
  }
}
