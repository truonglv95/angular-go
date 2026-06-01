import { Component } from '@angular/core';
import * as i0 from "@angular/core";
const AppComponent_Defer_2_DepsFn = () => [import("./heavy.component").then(m => m.HeavyComponent)];
function AppComponent_Defer_0_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵelement(0, "app-heavy");
} }
function AppComponent_DeferPlaceholder_1_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵelementStart(0, "div");
    i0.ɵɵtext(1, "Loading...");
    i0.ɵɵelementEnd();
} }
export class AppComponent {
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 4, vars: 0, template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵdomTemplate(0, AppComponent_Defer_0_Template, 1, 0)(1, AppComponent_DeferPlaceholder_1_Template, 2, 0);
            i0.ɵɵdefer(2, 0, AppComponent_Defer_2_DepsFn, null, 1);
            i0.ɵɵdeferOnViewport(0, -1);
        } }, encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadataAsync(AppComponent, () => [import("./heavy.component").then(m => m.HeavyComponent)], HeavyComponent => { i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                imports: [HeavyComponent],
                template: `
    @defer (on viewport) {
      <app-heavy />
    } @placeholder {
      <div>Loading...</div>
    }
  `,
                standalone: true
            }]
    }], null, null); }); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 17 }); })();
