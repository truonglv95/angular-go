package parity

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeGoldenJSWithRulesLineEndings(t *testing.T) {
	got := NormalizeGoldenJSWithRules("const a = 1;\r\nconst b = 2;\r\n")
	assertNormalized(t, got, "const a = 1;\nconst b = 2;\n", []string{"line-endings"})
}

func TestNormalizeGoldenJSWithRulesClassDebugFilePath(t *testing.T) {
	got := NormalizeGoldenJSWithRules(`(() => { i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "/tmp/project/src/app/app.component.ts", lineNumber: 1 }); })();`)
	assertNormalized(t, got, `(() => { i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 1 }); })();
`, []string{"class-debug-file-path", "trim-final-newline"})
}

func TestNormalizeGoldenJSWithRulesListenerFunctionNames(t *testing.T) {
	got := NormalizeGoldenJSWithRules(`i0.ɵɵlistener("click", function AppComponent_button_0_listener($event) { return ctx.save($event); });`)
	assertNormalized(t, got, `i0.ɵɵlistener("click", function ($event) { return ctx.save($event); });
`, []string{"listener-function-names", "trim-final-newline"})
}

func TestNormalizeGoldenJSWithRulesClassMetadata(t *testing.T) {
	input := `export class AppComponent {}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(AppComponent, [{
        type: Component,
        args: [{ selector: "app-root" }]
    }], null, null); })();
`
	got := NormalizeGoldenJSWithRules(input)
	assertNormalized(t, got, "export class AppComponent {}\n", []string{"class-metadata"})
}

func TestNormalizeGoldenJSWithRulesClassMetadataAsync(t *testing.T) {
	input := `export class AppComponent {}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadataAsync(AppComponent, () => [import("./cmp")], (cmp) => [{
        type: Component,
        args: [{ imports: [cmp.HeavyComponent] }]
    }], null, null); })();
`
	got := NormalizeGoldenJSWithRules(input)
	assertNormalized(t, got, "export class AppComponent {}\n", []string{"class-metadata-async"})
}

func TestNormalizeGoldenJSWithRulesClassDebugInfo(t *testing.T) {
	input := `export class AppComponent {}
(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassDebugInfo(AppComponent, { className: "AppComponent", filePath: "app/app.component.ts", lineNumber: 1 }); })();
`
	got := NormalizeGoldenJSWithRules(input)
	assertNormalized(t, got, "export class AppComponent {}\n", []string{"class-debug-info"})
}

func TestNormalizeGoldenJSWithRulesTrimFinalNewline(t *testing.T) {
	got := NormalizeGoldenJSWithRules("  const a = 1;\n\n")
	assertNormalized(t, got, "const a = 1;\n", []string{"trim-final-newline"})
}

func TestNormalizeGoldenJSPreservesSemanticDifferences(t *testing.T) {
	left := NormalizeGoldenJS(`i0.ɵɵpipeBind1(1, 2, ctx.value);`)
	right := NormalizeGoldenJS(`i0.ɵɵpipeBind1(1, 3, ctx.value);`)
	if left == right {
		t.Fatal("normalizer hid a pipeBind var offset semantic difference")
	}
}

func TestNormalizeGoldenJSPreservesDomElementInstructionDifferences(t *testing.T) {
	left := NormalizeGoldenJS(`i0.ɵɵdomElementStart(0, "div");`)
	right := NormalizeGoldenJS(`i0.ɵɵelementStart(0, "div");`)
	if left == right {
		t.Fatal("normalizer hid a domElement vs element instruction difference")
	}
}

func TestMergeRuleNames(t *testing.T) {
	got := MergeRuleNames([]string{"class-metadata", "line-endings"}, []string{"line-endings", "class-debug-info"})
	want := []string{"class-metadata", "line-endings", "class-debug-info"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeRuleNames() = %#v, want %#v", got, want)
	}
}

func assertNormalized(t *testing.T, got NormalizedJS, wantText string, wantRules []string) {
	t.Helper()
	if got.Text != wantText {
		t.Fatalf("Text mismatch\n--- got ---\n%s\n--- want ---\n%s", got.Text, wantText)
	}
	if !reflect.DeepEqual(got.Rules, wantRules) {
		t.Fatalf("Rules = %#v, want %#v", got.Rules, wantRules)
	}
	if strings.Contains(got.Text, "\r\n") {
		t.Fatalf("normalized text still contains CRLF: %q", got.Text)
	}
}
