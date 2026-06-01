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
	assertContains(t, text, `"value": "value"`)
	assertContains(t, text, `"flag": "flag"`)
	assertContains(t, text, `"signalValue": [1, "signalValue", "signalValue"]`)
	assertContains(t, text, "hostBindings")
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
