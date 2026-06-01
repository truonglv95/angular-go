import { Component } from '@angular/core';
import * as i0 from "@angular/core";
export class AppComponent {
    submit(val) { }
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 4, vars: 0, consts: [["myInput", ""], ["type", "text"], [3, "click"]], template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            const _r1 = i0.ɵɵgetCurrentView();
            i0.ɵɵdomElement(0, "input", 1, 0);
            i0.ɵɵdomElementStart(2, "button", 2);
            i0.ɵɵdomListener("click", function AppComponent_Template_button_click_2_listener() { i0.ɵɵrestoreView(_r1); const myInput_r2 = i0.ɵɵreference(1); return i0.ɵɵresetView(ctx.submit(myInput_r2.value)); });
            i0.ɵɵtext(3, "Submit");
            i0.ɵɵdomElementEnd();
        } }, encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                template: `
    <input #myInput type="text" />
    <button (click)="submit(myInput.value)">Submit</button>
  `,
                standalone: true
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 11 }); })();
