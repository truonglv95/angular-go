import { Injectable, NgModule, Component, Inject, InjectionToken, forwardRef, Optional } from '@angular/core';

// 1. Service provider cấp Component (không có @Injectable root)
@Injectable()
export class MyProvidedService {
  doSomething() { return 'ok'; }
}

@Component({
  selector: 'app-di-case1',
  template: `<div>Case 11</div>`,
  standalone: true,
  providers: [MyProvidedService]
})
export class Case1Component {
  constructor(private srv: MyProvidedService) {}
}

// 2. InjectionToken
export const MY_TOKEN = new InjectionToken<string>('MY_TOKEN');

@Component({
  selector: 'app-di-case2',
  template: `<div>Case 224</div>`,
  standalone: true,
  providers: [{provide: MY_TOKEN, useValue: 'hello'}]
})
export class Case2Component {
  constructor(@Inject(MY_TOKEN) private tokenVal: string) {}
}

// 3. forwardRef
@Component({
  selector: 'app-di-case3',
  template: `<div>Case 3</div>`,
  standalone: true,
  providers: [forwardRef(() => ForwardService)]
})
export class Case3Component {
  constructor(@Inject(forwardRef(() => ForwardService)) private srv: ForwardService) {}
}

@Injectable()
export class ForwardService {}

// 4. Interface with @Optional and @Inject
export interface SomeInterface {}
export const SOME_INTERFACE_TOKEN = new InjectionToken<SomeInterface>('SomeInterface');

@Component({
  selector: 'app-di-case4',
  template: `<div>Case 4</div>`,
  standalone: true
})
export class Case4Component {
  constructor(@Optional() @Inject(SOME_INTERFACE_TOKEN) private srv: SomeInterface) {}
}
