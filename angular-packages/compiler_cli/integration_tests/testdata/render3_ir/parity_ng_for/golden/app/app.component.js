import { Component } from '@angular/core';
import { NgFor } from '@angular/common';
import * as i0 from "@angular/core";
function AppComponent_li_1_Template(rf, ctx) { if (rf & 1) {
    i0.ɵɵelementStart(0, "li");
    i0.ɵɵtext(1);
    i0.ɵɵelementEnd();
} if (rf & 2) {
    const item_r1 = ctx.$implicit;
    const i_r2 = ctx.index;
    i0.ɵɵadvance();
    i0.ɵɵtextInterpolate2(" ", i_r2, ": ", item_r1.name, " ");
} }
export class AppComponent {
    items = [{ id: 1, name: 'A' }, { id: 2, name: 'B' }];
    trackById(index, item) { return item.id; }
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 2, vars: 2, consts: [[4, "ngFor", "ngForOf", "ngForTrackBy"]], template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵelementStart(0, "ul");
            i0.ɵɵtemplate(1, AppComponent_li_1_Template, 2, 2, "li", 0);
            i0.ɵɵelementEnd();
        } if (rf & 2) {
            i0.ɵɵadvance();
            i0.ɵɵproperty("ngForOf", ctx.items)("ngForTrackBy", ctx.trackById);
        } }, dependencies: [NgFor], encapsulation: 2 });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                imports: [NgFor],
                template: `
    <ul>
      <li *ngFor="let item of items; let i = index; trackBy: trackById">
        {{ i }}: {{ item.name }}
      </li>
    </ul>
  `,
                standalone: true
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 16 }); })();
