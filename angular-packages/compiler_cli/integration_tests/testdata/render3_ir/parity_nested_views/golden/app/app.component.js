import { Component } from '@angular/core';
import * as i0 from "@angular/core";
function AppComponent_ng_template_1_ng_template_1_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵdomElementStart(0, "span");
    i0.ɵɵtext(1);
    i0.ɵɵdomElementEnd();
} if (rf & 2) {
    const ctx_r0 = i0.ɵɵnextContext(2);
    i0.ɵɵadvance();
    i0.ɵɵtextInterpolate1("Nested ", ctx_r0.value);
} }
function AppComponent_ng_template_1_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵdomElementStart(0, "div");
    i0.ɵɵdomTemplate(1, AppComponent_ng_template_1_ng_template_1_Template, 2, 1, "ng-template");
    i0.ɵɵdomElementEnd();
} }
export class AppComponent {
    value = 42;
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 2, vars: 0, template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵdomElementStart(0, "div");
            i0.ɵɵdomTemplate(1, AppComponent_ng_template_1_Template, 2, 0, "ng-template");
            i0.ɵɵdomElementEnd();
        } }, encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                template: `
    <div>
      <ng-template>
        <div>
          <ng-template>
            <span>Nested {{ value }}</span>
          </ng-template>
        </div>
      </ng-template>
    </div>
  `,
                standalone: true
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 18 }); })();
