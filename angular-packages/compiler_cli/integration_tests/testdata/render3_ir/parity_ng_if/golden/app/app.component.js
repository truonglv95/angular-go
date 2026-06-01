import { Component } from '@angular/core';
import { NgIf } from '@angular/common';
import * as i0 from "@angular/core";
function AppComponent_div_0_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵelementStart(0, "div");
    i0.ɵɵtext(1, " Hello IF! ");
    i0.ɵɵelementEnd();
} }
function AppComponent_ng_template_1_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵtext(0, " Hello ELSE! ");
} }
export class AppComponent {
    condition = true;
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 3, vars: 2, consts: [["elseBlock", ""], [4, "ngIf", "ngIfElse"]], template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵtemplate(0, AppComponent_div_0_Template, 2, 0, "div", 1)(1, AppComponent_ng_template_1_Template, 1, 0, "ng-template", null, 0, i0.ɵɵtemplateRefExtractor);
        } if (rf & 2) {
            const elseBlock_r1 = i0.ɵɵreference(2);
            i0.ɵɵproperty("ngIf", ctx.condition)("ngIfElse", elseBlock_r1);
        } }, dependencies: [NgIf], encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                imports: [NgIf],
                template: `
    <div *ngIf="condition; else elseBlock">
      Hello IF!
    </div>
    <ng-template #elseBlock>
      Hello ELSE!
    </ng-template>
  `,
                standalone: true
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 17 }); })();
