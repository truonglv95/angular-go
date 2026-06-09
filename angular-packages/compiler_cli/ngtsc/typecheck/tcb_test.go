package typecheck

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTcb_BasicBinding(t *testing.T) {
	parsed := render3.ParseTemplate("{{hello}} {{world}}", "/test.html", nil)
	tcbStr, _ := GenerateTcb(
		"MyComp",
		&parsed,
		func(node render3.Node) []DirectiveInfo { return nil },
		func(node render3.Node, binding any) (string, string, bool) { return "", "", false },
		nil,
	)

	assert.Contains(t, tcbStr, "function _tcb_MyComp(this: MyComp) {")
	assert.Contains(t, tcbStr, "this.hello")
	assert.Contains(t, tcbStr, "this.world")
}

func TestGenerateTcb_ElementAndDirective(t *testing.T) {
	parsed := render3.ParseTemplate(`<div dir [class]="foo"></div>`, "/test.html", nil)
	tcbStr, _ := GenerateTcb(
		"MyComp",
		&parsed,
		func(node render3.Node) []DirectiveInfo {
			if el, ok := node.(*render3.Element); ok && el.Name == "div" {
				return []DirectiveInfo{
					{ClassName: "Dir", OwningModule: "@angular/core"},
				}
			}
			return nil
		},
		func(node render3.Node, binding any) (string, string, bool) {
			if b, ok := binding.(*render3.BoundAttribute); ok && b.Name == "class" {
				return "Dir", "className", true
			}
			return "", "", false
		},
		nil,
	)

	assert.Contains(t, tcbStr, "var _el1: HTMLDivElement = null!;")
	assert.Contains(t, tcbStr, "var _dir1: import('@angular/core').Dir = null!;")
	assert.Contains(t, tcbStr, "_assign(_dir1.className, (this.foo));")
}

func TestGenerateTcb_LocalReference(t *testing.T) {
	parsed := render3.ParseTemplate(`<div #myDiv></div>`, "/test.html", nil)
	tcbStr, _ := GenerateTcb(
		"MyComp",
		&parsed,
		func(node render3.Node) []DirectiveInfo { return nil },
		func(node render3.Node, binding any) (string, string, bool) { return "", "", false },
		nil,
	)

	assert.Contains(t, tcbStr, "var _el1: HTMLDivElement = null!;")
	assert.Contains(t, tcbStr, "var myDiv = _el1;")
}

func TestGenerateTcb_Pipes(t *testing.T) {
	parsed := render3.ParseTemplate(`{{ value | json }}`, "/test.html", nil)
	pipes := map[string]string{
		"json": "JsonPipe",
	}
	tcbStr, _ := GenerateTcb(
		"MyComp",
		&parsed,
		func(node render3.Node) []DirectiveInfo { return nil },
		func(node render3.Node, binding any) (string, string, bool) { return "", "", false },
		pipes,
	)

	assert.Contains(t, tcbStr, "var _pipe_json: JsonPipe = null!;")
	assert.Contains(t, tcbStr, "_pipe_json.transform(this.value)")
}

func TestGenerateTcb_SwitchBlock(t *testing.T) {
	parsed := render3.ParseTemplate(`@switch (x) { @case (1) { Case 1 } @default { Default Case } }`, "/test.html", nil)
	tcbStr, _ := GenerateTcb(
		"MyComp",
		&parsed,
		func(node render3.Node) []DirectiveInfo { return nil },
		func(node render3.Node, binding any) (string, string, bool) { return "", "", false },
		nil,
	)

	assert.Contains(t, tcbStr, "switch (this.x) {")
	assert.Contains(t, tcbStr, "case 1:")
	assert.Contains(t, tcbStr, "default:")
}
