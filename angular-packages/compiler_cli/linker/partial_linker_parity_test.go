package linker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ngcompiler "github.com/microsoft/typescript-go/angular-packages/compiler"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/emit"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ingest"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/printer"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func TestPartialLinkerParityHarnessMinimalDirective(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class Dir {}
Dir.ɵdir = i0.ɵɵngDeclareDirective({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: Dir,
  isStandalone: true,
  selector: "[dir]",
  inputs: {
    value: "value",
    flag: ["flag", "flag", i0.booleanAttribute],
    signalValue: {
      classPropertyName: "signalValue",
      publicName: "signalValue",
      isSignal: true,
      isRequired: false,
      transformFunction: null
    }
  },
  host: {
    properties: {
      "class.active": "flag"
    }
  },
  ngImport: i0
});
`

	text := linkPartialSource(t, "minimal-directive.mjs", source)

	assertContains(t, text, "ɵɵdefineDirective")
	assertNotContains(t, text, "ɵɵngDeclareDirective")
	assertContains(t, text, `selectors: [["", "dir", ""]]`)
	assertContains(t, text, `value: "value"`)
	assertContains(t, text, `flag: "flag"`)
	assertContains(t, text, `signalValue: [1, "signalValue", "signalValue"]`)
	assertContains(t, text, "hostBindings")
}

func TestPartialLinkerHostListenersPassDollarEvent(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class Dir {
  update(value) {}
  blur() {}
}

Dir.ɵdir = i0.ɵɵngDeclareDirective({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: Dir,
  isStandalone: true,
  selector: "[dir]",
  host: {
    listeners: {
      "input": "update($any($event.target).value)",
      "blur": "blur()"
    }
  },
  ngImport: i0
});
`

	text := linkPartialSource(t, "host-listener-event.mjs", source)

	assertContains(t, text, `("input", function Dir_input_HostBindingHandler($event)`)
	assertContains(t, text, `ctx.update($event.target.value)`)
	assertContains(t, text, `("blur", function Dir_blur_HostBindingHandler()`)
	assertNotContains(t, text, `ctx.$event`)
	assertNotContains(t, text, `ctx.$any`)
}

func TestPartialLinkerEmitsContentAndViewQueries(t *testing.T) {
	source := `
import * as i0 from "@angular/core";
import { ElementRef } from "@angular/core";

class Footer {}
class Panel {
  id = "panel-id";
}

Panel.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "16.1.0",
  version: "20.3.15",
  type: Panel,
  isStandalone: true,
  selector: "p-panel",
  queries: [
    { propertyName: "footerFacet", first: true, predicate: Footer, descendants: true },
    { propertyName: "templates", predicate: ["pTemplate"], emitDistinctChangesOnly: false }
  ],
  viewQueries: [
    { propertyName: "contentWrapperViewChild", first: true, predicate: ["contentWrapper"], descendants: true, read: ElementRef, static: true }
  ],
  ngImport: i0,
  template: ` + "`" + `<div #contentWrapper><ng-content></ng-content></div>` + "`" + `,
  isInline: true
});
`

	text := linkPartialSource(t, "query-component.mjs", source)

	assertContains(t, text, "contentQueries")
	assertContains(t, text, "viewQuery")
	assertContains(t, text, "ɵɵcontentQuery")
	assertContains(t, text, "ɵɵviewQuery")
	assertContains(t, text, "Footer")
	assertContains(t, text, `"pTemplate"`)
	assertContains(t, text, `"contentWrapper"`)
	assertContains(t, text, "ElementRef")
	assertContains(t, text, "footerFacet")
	assertContains(t, text, "contentWrapperViewChild")
}

func TestPartialLinkerEmitsLegacyAnimationNames(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class Cmp {
  state = "open";
  done(event) {}
}

Cmp.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "16.1.0",
  version: "20.3.15",
  type: Cmp,
  isStandalone: true,
  selector: "x-cmp",
  ngImport: i0,
  template: ` + "`" + `
    <div [@fade]="state" (@fade.done)="done($event)"></div>
    <section [@slide]="state" (@slide.start)="done($event)"></section>
  ` + "`" + `,
  isInline: true,
  animations: []
});
`

	text := linkPartialSource(t, "legacy-animation-component.mjs", source)

	assertContains(t, text, `ɵɵlistener("@fade.done"`)
	assertContains(t, text, `ɵɵproperty("@fade"`)
	assertContains(t, text, `ɵɵlistener("@slide.start"`)
	assertContains(t, text, `ɵɵproperty("@slide"`)
	assertNotContains(t, text, `ɵɵdomListener("fade"`)
	assertNotContains(t, text, `ɵɵproperty("fade"`)
}

func TestPartialLinkerEmitsUndefinedKeyword(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class Cmp {
  collapsed = false;
}

Cmp.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "16.1.0",
  version: "20.3.15",
  type: Cmp,
  isStandalone: true,
  selector: "x-cmp",
  ngImport: i0,
  template: ` + "`" + `
    <div [attr.tabindex]="collapsed ? '-1' : undefined"></div>
    <section [title]="'undefined'"></section>
  ` + "`" + `,
  isInline: true
});
`

	text := linkPartialSource(t, "undefined-keyword-component.mjs", source)

	assertContains(t, text, `ctx.collapsed ? "-1" : undefined`)
	assertContains(t, text, `ɵɵdomProperty("title", "undefined")`)
	assertNotContains(t, text, `ctx.collapsed ? "-1" : "undefined"`)
}

func TestPartialLinkerEmitsNgModuleMetadata(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class Cmp {}
class Dir {}
class SharedModule {}
class PanelModule {}

PanelModule.ɵmod = i0.ɵɵngDeclareNgModule({
  minVersion: "14.0.0",
  version: "20.3.15",
  ngImport: i0,
  type: PanelModule,
  declarations: [Cmp, Dir],
  imports: [SharedModule],
  exports: [Cmp]
});
`

	text := linkPartialSource(t, "ng-module.mjs", source)

	assertContains(t, text, "ɵɵdefineNgModule")
	assertNotContains(t, text, "ɵɵngDeclareNgModule")
	assertContains(t, text, "type: PanelModule")
	assertContains(t, text, "declarations: [Cmp, Dir]")
	assertContains(t, text, "imports: [SharedModule]")
	assertContains(t, text, "exports: [Cmp]")
	assertNotContains(t, text, "type: null")
	assertNotContains(t, text, "imports: null")
	assertNotContains(t, text, "exports: null")
}

func TestPartialLinkerEmitsComponentFacadeOptions(t *testing.T) {
	source := `
import * as i0 from "@angular/core";
import { ChangeDetectionStrategy, ViewEncapsulation } from "@angular/core";

class Cmp {}

Cmp.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "16.1.0",
  version: "20.3.15",
  type: Cmp,
  isStandalone: true,
  selector: "x-cmp",
  ngImport: i0,
  template: ` + "`" + `<span>cmp</span>` + "`" + `,
  isInline: true,
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.ShadowDom
});
`

	text := linkPartialSource(t, "component-options.mjs", source)

	assertContains(t, text, "changeDetection: 0")
	assertContains(t, text, "encapsulation: 3")
	assertNotContains(t, text, "changeDetection: null")
}

func TestPartialLinkerFactoryDepsNullUsesInheritedFactory(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class BaseCmp {}

BaseCmp.ɵfac = i0.ɵɵngDeclareFactory({
  minVersion: "12.0.0",
  version: "20.3.15",
  ngImport: i0,
  type: BaseCmp,
  deps: null,
  target: i0.ɵɵFactoryTarget.Component
});
`

	text := linkPartialSource(t, "factory-deps-null.mjs", source)

	assertContains(t, text, "BaseCmp_Factory")
	assertContains(t, text, "ɵBaseCmp_BaseFactory")
	assertContains(t, text, "ɵɵgetInheritedFactory(BaseCmp)")
	assertNotContains(t, text, "new (__ngFactoryType__ || BaseCmp)()")
}

func TestPartialLinkerFactoryEmptyDepsUsesConstructorFactory(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class EmptyDepsCmp {}

EmptyDepsCmp.ɵfac = i0.ɵɵngDeclareFactory({
  minVersion: "12.0.0",
  version: "20.3.15",
  ngImport: i0,
  type: EmptyDepsCmp,
  deps: [],
  target: i0.ɵɵFactoryTarget.Component
});
`

	text := linkPartialSource(t, "factory-empty-deps.mjs", source)

	assertContains(t, text, "EmptyDepsCmp_Factory")
	assertContains(t, text, "new (__ngFactoryType__ || EmptyDepsCmp)()")
	assertNotContains(t, text, "ɵɵgetInheritedFactory(EmptyDepsCmp)")
}

func TestPartialLinkerFactoryInvalidDepsEmitsInvalidFactory(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class InvalidDepsService {}

InvalidDepsService.ɵfac = i0.ɵɵngDeclareFactory({
  minVersion: "12.0.0",
  version: "20.3.15",
  ngImport: i0,
  type: InvalidDepsService,
  deps: "invalid",
  target: i0.ɵɵFactoryTarget.Injectable
});
`

	text := linkPartialSource(t, "factory-invalid-deps.mjs", source)

	assertContains(t, text, "InvalidDepsService_Factory")
	assertContains(t, text, "ɵɵinvalidFactory()")
	assertNotContains(t, text, "new (__ngFactoryType__ || InvalidDepsService)()")
	assertNotContains(t, text, "ɵɵgetInheritedFactory(InvalidDepsService)")
}

func TestPartialLinkerLocalRefsUseReferenceValueSlots(t *testing.T) {
	source := `
import * as i0 from "@angular/core";
import { NgTemplateOutlet } from "@angular/common";

class Cmp {}

Cmp.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: Cmp,
  isStandalone: true,
  selector: "x-cmp",
  ngImport: i0,
  template: ` + "`" + `
    <ng-container *ngTemplateOutlet="primary"></ng-container>
    <ng-container *ngTemplateOutlet="secondary"></ng-container>
    <ng-template #primary #secondary>Projected</ng-template>
  ` + "`" + `,
  isInline: true,
  dependencies: [
    { kind: "directive", type: NgTemplateOutlet, selector: "[ngTemplateOutlet]", inputs: ["ngTemplateOutletContext", "ngTemplateOutlet", "ngTemplateOutletInjector"] }
  ]
});
`

	text := linkPartialSource(t, "local-ref-slots.mjs", source)

	assertContains(t, text, `ɵɵtemplateRefExtractor`)
	assertContains(t, text, `ɵɵreference(3)`)
	assertContains(t, text, `ɵɵreference(4)`)
	assertNotContains(t, text, `ɵɵreference(2)`)
}

func TestPartialLinkerResolvesForwardRefTemplateDependencies(t *testing.T) {
	source := `
import * as i0 from "@angular/core";
import { NgIf } from "@angular/common";

class Cmp {}

Cmp.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: Cmp,
  isStandalone: false,
  selector: "x-cmp",
  ngImport: i0,
  template: ` + "`" + `<div *ngIf="visible"></div>` + "`" + `,
  isInline: true,
  dependencies: [
    { kind: "directive", type: i0.forwardRef(() => NgIf), selector: "[ngIf]", inputs: ["ngIf", "ngIfThen", "ngIfElse"] }
  ]
});
`

	text := linkPartialSource(t, "forward-ref-deps.mjs", source)

	assertContains(t, text, `dependencies: () => [i0.forwardRef(() => NgIf)].map(i0.resolveForwardRef)`)
	assertNotContains(t, text, `dependencies: [i0.forwardRef(() => NgIf)]`)
}

func TestPartialLinkerResolvesConstArrayTemplateDependencies(t *testing.T) {
	source := `
import * as i0 from "@angular/core";
import { NgIf } from "@angular/common";

const CMP_DEPS = [
  { kind: "directive", type: NgIf, selector: "[ngIf]", inputs: ["ngIf", "ngIfThen", "ngIfElse"] }
];

class Cmp {
  visible = true;
}

Cmp.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: Cmp,
  isStandalone: true,
  selector: "x-cmp",
  ngImport: i0,
  template: ` + "`" + `<div *ngIf="visible"></div>` + "`" + `,
  isInline: true,
  dependencies: CMP_DEPS
});
`

	text := linkPartialSource(t, "const-array-deps.mjs", source)

	assertContains(t, text, `dependencies: () => [NgIf].map(i0.resolveForwardRef)`)
	assertContains(t, text, `ɵɵtemplate`)
	assertNotContains(t, text, `ɵɵngDeclareComponent`)
}

func TestPartialLinkerKeepsProjectedTemplateLocalRefsBeforeAttrs(t *testing.T) {
	source := `
import * as i0 from "@angular/core";

class OverlayLike {}
OverlayLike.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: OverlayLike,
  isStandalone: true,
  selector: "x-overlay",
  ngImport: i0,
  template: ` + "`" + `<ng-content></ng-content>` + "`" + `,
  isInline: true
});

class SelectLike {
  visible = false;
}

SelectLike.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: SelectLike,
  isStandalone: true,
  selector: "x-select",
  ngImport: i0,
  template: ` + "`" + `
    <x-overlay #overlay [visible]="visible">
      <ng-template #content>Content</ng-template>
    </x-overlay>
  ` + "`" + `,
  isInline: true,
  dependencies: [
    { kind: "component", type: OverlayLike, selector: "x-overlay", inputs: ["visible"] }
  ]
});
`

	text := linkPartialSource(t, "projected-template-local-refs.mjs", source)

	assertContains(t, text, `consts: [["overlay", ""], ["content", ""], [3, "visible"]]`)
	assertContains(t, text, `ɵɵelementStart(0, "x-overlay", 2, 0)`)
	assertContains(t, text, `ɵɵtemplate(2, SelectLike_ng_template_2_Template, 1, 0, "ng-template", null, 1, i0.ɵɵtemplateRefExtractor)`)
}

func TestPartialLinkerUsesAncestorContextDepthForNestedTemplates(t *testing.T) {
	source := `
import * as i0 from "@angular/core";
import { NgForOf, NgIf } from "@angular/common";

class MenuLike {
  visible = true;
  model = [];
  id = 'menu';
  menuitemId(item, id, index) { return id + '_' + index; }
}

MenuLike.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "17.1.0",
  version: "20.3.15",
  type: MenuLike,
  isStandalone: true,
  selector: "x-menu-like",
  ngImport: i0,
  template: ` + "`" + `
    @if (visible) {
      <ul>
        <ng-template ngFor let-item let-i="index" [ngForOf]="model" *ngIf="visible">
          <li *ngIf="item.visible !== false">{{ menuitemId(item, id, i) }}</li>
        </ng-template>
      </ul>
    }
  ` + "`" + `,
  isInline: true,
  dependencies: [
    { kind: "directive", type: NgIf, selector: "[ngIf]", inputs: ["ngIf", "ngIfThen", "ngIfElse"] },
    { kind: "directive", type: NgForOf, selector: "[ngFor][ngForOf]", inputs: ["ngForOf", "ngForTrackBy", "ngForTemplate"] }
  ]
});
`

	text := linkPartialSource(t, "nested-template-context-depth.mjs", source)

	assertContains(t, text, `ɵɵnextContext`)
	assertContains(t, text, `ngForOf`)
	assertContains(t, text, `.model`)
	assertContains(t, text, `.menuitemId`)
}

func TestPartialLinkerParityHarnessPanelLikeKnownGaps(t *testing.T) {
	if os.Getenv("GO_LINKER_AUDIT_KNOWN_GAPS") == "" {
		t.Skip("set GO_LINKER_AUDIT_KNOWN_GAPS=1 to run the known-gap parity audit")
	}

	source := `
import * as i0 from "@angular/core";
import { Component, ChangeDetectionStrategy, ViewEncapsulation } from "@angular/core";

class Footer {}
class PrimeTemplate {}
class PanelStyle {}
class Bind {}
class Panel {
  id = "panel-id";
  collapsed = false;
  toggleable = false;
  styleClass;
  animating() { return false; }
  transitionOptions = "400ms";
  dataP() { return ""; }
  cx(value) { return value; }
  cn(value) { return value; }
  ptm(value) { return value; }
  onToggleDone(event) {}
}

Panel.ɵcmp = i0.ɵɵngDeclareComponent({
  minVersion: "16.1.0",
  version: "20.3.15",
  type: Panel,
  isStandalone: true,
  selector: "p-panel",
  inputs: {
    id: "id",
    toggleable: ["toggleable", "toggleable", i0.booleanAttribute],
    collapsed: ["collapsed", "collapsed", i0.booleanAttribute],
    styleClass: "styleClass"
  },
  host: {
    properties: {
      "id": "id",
      "class": "cn(cx('root'), styleClass)",
      "attr.data-p": "dataP()"
    }
  },
  providers: [PanelStyle],
  queries: [
    { propertyName: "footerFacet", first: true, predicate: Footer, descendants: true },
    { propertyName: "templates", predicate: PrimeTemplate }
  ],
  viewQueries: [
    { propertyName: "contentWrapperViewChild", first: true, predicate: ["contentWrapper"], descendants: true }
  ],
  hostDirectives: [{ directive: Bind }],
  ngImport: i0,
  template: ` + "`" + `
    <div
      [id]="id + '_content'"
      [attr.tabindex]="collapsed ? '-1' : undefined"
      [@panelContent]="collapsed ? 'hidden' : 'visible'"
      (@panelContent.done)="onToggleDone($event)"
    >
      <div #contentWrapper><ng-content></ng-content></div>
    </div>
  ` + "`" + `,
  isInline: true,
  animations: [],
  changeDetection: ChangeDetectionStrategy.OnPush,
  encapsulation: ViewEncapsulation.None
});
`

	text := linkPartialSource(t, "panel-like.mjs", source)

	assertContains(t, text, "contentQueries")
	assertContains(t, text, "viewQuery")
	assertContains(t, text, `"@panelContent"`)
	assertContains(t, text, `"@panelContent.done"`)
	assertContains(t, text, `: undefined`)
	assertContains(t, text, "changeDetection")
}

func linkPartialSource(t *testing.T, fileName string, source string) string {
	t.Helper()

	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		t.Fatalf("filepath.Abs(%s): %v", fileName, err)
	}
	absFileName = tspath.NormalizePath(absFileName)

	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: absFileName}, source, core.ScriptKindJS)
	if sf == nil {
		t.Fatalf("ParseSourceFile(%s) returned nil", fileName)
	}

	result := LinkSourceFile(sf, ngcompiler.NewConstantPool(false))
	if len(result.Errors) > 0 {
		t.Fatalf("LinkSourceFile(%s) errors: %v", fileName, result.Errors)
	}
	if !result.Changed {
		t.Fatalf("LinkSourceFile(%s) did not report changes", fileName)
	}

	p := printer.NewPrinter(printer.PrinterOptions{NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, nil)
	return p.EmitSourceFile(result.SourceFile)
}

func assertContains(t *testing.T, text string, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("linked output does not contain %q\n\n%s", want, text)
	}
}

func assertNotContains(t *testing.T, text string, want string) {
	t.Helper()
	if strings.Contains(text, want) {
		t.Fatalf("linked output unexpectedly contains %q\n\n%s", want, text)
	}
}
