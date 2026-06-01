import { Component, Pipe } from '@angular/core';
import * as i0 from "@angular/core";
export class UppercasePipe {
    transform(value) { return value.toUpperCase(); }
    static ɵfac = function UppercasePipe_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || UppercasePipe)(); };
    static ɵpipe = /*@__PURE__*/ i0.ɵɵdefinePipe({ name: "uppercase", type: UppercasePipe, pure: true });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(UppercasePipe, [{
        type: Pipe,
        args: [{ name: 'uppercase', standalone: true }]
    }], null, null); })();
export class AppComponent {
    title = 'hello';
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 3, vars: 3, template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵdomElementStart(0, "div");
            i0.ɵɵtext(1);
            i0.ɵɵpipe(2, "uppercase");
            i0.ɵɵdomElementEnd();
        } if (rf & 2) {
            i0.ɵɵadvance();
            i0.ɵɵtextInterpolate(i0.ɵɵpipeBind1(2, 1, ctx.title));
        } }, dependencies: [UppercasePipe], encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                imports: [UppercasePipe],
                template: `
    <div>{{ title | uppercase }}</div>
  `,
                standalone: true
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 16 }); })();
