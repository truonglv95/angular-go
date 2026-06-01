import { Component } from '@angular/core';
import { trigger, state, style, transition, animate } from '@angular/animations';
import * as i0 from "@angular/core";
export class AppComponent {
    isOpen = true;
    static ɵfac = function AppComponent_Factory(__ngFactoryType__) { return new (__ngFactoryType__ || AppComponent)(); };
    static ɵcmp = /*@__PURE__*/ i0.ɵɵdefineComponent({ type: AppComponent, selectors: [["app-root"]], decls: 2, vars: 1, template: function AppComponent_Template(rf, ctx) { if (rf & 1) {
            i0.ɵɵdomElementStart(0, "div");
            i0.ɵɵtext(1, " Animate me ");
            i0.ɵɵdomElementEnd();
        } if (rf & 2) {
            i0.ɵɵproperty("@openClose", ctx.isOpen ? "open" : "closed");
        } }, encapsulation: 2, data: { animation: [
                trigger('openClose', [
                    state('open', style({ height: '200px' })),
                    state('closed', style({ height: '100px' })),
                    transition('open => closed', [animate('1s')]),
                    transition('closed => open', [animate('0.5s')])
                ])
            ] } });
}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{
                selector: 'app-root',
                template: `
    <div [@openClose]="isOpen ? 'open' : 'closed'">
      Animate me
    </div>
  `,
                standalone: true,
                animations: [
                    trigger('openClose', [
                        state('open', style({ height: '200px' })),
                        state('closed', style({ height: '100px' })),
                        transition('open => closed', [animate('1s')]),
                        transition('closed => open', [animate('0.5s')])
                    ])
                ]
            }]
    }], null, null); })();
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 21 }); })();
