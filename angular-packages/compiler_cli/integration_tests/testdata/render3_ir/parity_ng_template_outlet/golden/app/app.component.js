import { Component } from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';
import * as i0 from "@angular/core";
const _c0 = () => ({ $implicit: "world" });
function AppComponent_ng_container_0_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵelementContainer(0);
} }
function AppComponent_ng_template_1_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵtext(0);
} if (rf & 2) {
    const msg_r1 = ctx.$implicit;
    i0.ɵɵtextInterpolate1(" Hello ", msg_r1, "! ");
} }
export class AppComponent {
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 3, vars: 3, consts: [["myTpl", ""], [4, "ngTemplateOutlet", "ngTemplateOutletContext"]], template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵtemplate(0, AppComponent_ng_container_0_Template, 1, 0, "ng-container", 1)(1, AppComponent_ng_template_1_Template, 1, 1, "ng-template", null, 0, i0.ɵɵtemplateRefExtractor);
        } if (rf & 2) {
            const myTpl_r2 = i0.ɵɵreference(2);
            i0.ɵɵproperty("ngTemplateOutlet", myTpl_r2)("ngTemplateOutletContext", i0.ɵɵpureFunction0(2, _c0));
        } }, dependencies: [NgTemplateOutlet], encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                imports: [NgTemplateOutlet],
                template: `
    <ng-container *ngTemplateOutlet="myTpl; context: { $implicit: 'world' }"></ng-container>
    <ng-template #myTpl let-msg>
      Hello {{ msg }}!
    </ng-template>
  `,
                standalone: true
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 15 }); })();
