import { Component } from '@angular/core';
import * as i0 from "@angular/core";
export class HeavyComponent {
    static ɵfac = function HeavyComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || HeavyComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: HeavyComponent, selectors: [["app-heavy"]], decls: 1, vars: 0, template: function HeavyComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵtext(0, "Heavy!");
        } }, encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(HeavyComponent, [{
        type: Component,
        args: [{
                selector: 'app-heavy',
                template: `Heavy!`,
                standalone: true
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(HeavyComponent, { className: "HeavyComponent", filePath: "app/heavy.component.ts", lineNumber: 8 }); })();
