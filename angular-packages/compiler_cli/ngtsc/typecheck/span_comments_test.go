package typecheck

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/stretchr/testify/assert"
)

func tcbWithSpans(template string, declarations []TestDeclaration, pipes map[string]string) string {
	parsed := render3.ParseTemplate(template, "synthetic.html", nil)

	getDirectives := func(node render3.Node) []DirectiveInfo {
		var result []DirectiveInfo
		el, ok := node.(*render3.Element)
		if !ok {
			return nil
		}
		for _, decl := range declarations {
			if decl.Type == "directive" {
				if matchesSelector(el, decl.Selector) {
					result = append(result, DirectiveInfo{
						ClassName:    decl.Name,
						OwningModule: decl.File,
					})
				}
			}
		}
		return result
	}

	getBindingConsumer := func(node render3.Node, binding any) (string, string, bool) {
		el, ok := node.(*render3.Element)
		if !ok {
			return "", "", false
		}
		boundAttr, isBoundAttr := binding.(*render3.BoundAttribute)
		if !isBoundAttr {
			return "", "", false
		}

		for _, decl := range declarations {
			if decl.Type == "directive" && matchesSelector(el, decl.Selector) {
				if decl.Inputs != nil {
					switch inputs := decl.Inputs.(type) {
					case map[string]string:
						if propName, ok := inputs[boundAttr.Name]; ok {
							return decl.Name, propName, true
						}
					case map[string]any:
						if val, ok := inputs[boundAttr.Name]; ok {
							if strVal, ok := val.(string); ok {
								return decl.Name, strVal, true
							}
						}
					}
				}
			}
		}
		return "", "", false
	}

	tcbStr, _ := GenerateTcbWithOptions(
		"Test",
		&parsed,
		getDirectives,
		getBindingConsumer,
		pipes,
		true, // emitSpans
		0,    // absoluteOffset
	)

	fields := strings.Fields(tcbStr)
	return strings.Join(fields, " ")
}

func TestSpanComments(t *testing.T) {
	t.Run("unary ops", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ -a }}", nil, nil), "(-(this.a /*4,5*/) /*4,5*/) /*3,5*/")
	})

	t.Run("binary ops", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a + b }}", nil, nil), "(((this.a /*3,4*/) /*3,4*/)) + (((this.b /*7,8*/) /*7,8*/)) /*3,8*/")
	})

	t.Run("conditions", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a ? b : c }}", nil, nil), "((((this.a /*3,4*/) /*3,4*/)) ? ((this.b /*7,8*/) /*7,8*/) : (((this.c /*11,12*/) /*11,12*/))) /*3,12*/")
	})

	t.Run("interpolations", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ hello }} {{ world }}", nil, nil), `(((this.hello /*3,8*/) /*3,8*/)) + (((this.world /*15,20*/) /*15,20*/))`)
	})

	t.Run("literal map expressions", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ m({foo: a, bar: b}) }}", nil, nil), `this.m /*3,4*/((({ "foo" /*6,9*/: ((this.a /*11,12*/) /*11,12*/), "bar" /*14,17*/: ((this.b /*19,20*/) /*19,20*/) }) /*5,21*/)) /*3,22*/`)
	})

	t.Run("literal map expressions with shorthand declarations", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ m({a, b}) }}", nil, nil), `this.m /*3,4*/((({ "a" /*6,7*/: ((this.a /*6,7*/) /*6,7*/), "b" /*9,10*/: ((this.b /*9,10*/) /*9,10*/) }) /*5,11*/)) /*3,12*/`)
	})

	t.Run("literal array expressions", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ [a, b] }}", nil, nil), `([((this.a /*4,5*/) /*4,5*/), ((this.b /*7,8*/) /*7,8*/)]) /*3,9*/`)
	})

	t.Run("literals", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ 123 }}", nil, nil), `123 /*3,6*/`)
	})

	t.Run("non-null assertions", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a! }}", nil, nil), `(((this.a /*3,4*/) /*3,4*/))! /*3,5*/`)
	})

	t.Run("prefix not", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ !a }}", nil, nil), `(!(((this.a /*4,5*/) /*4,5*/))) /*3,5*/`)
	})

	t.Run("method calls", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ method(a, b) }}", nil, nil), `this.method /*3,9*/((this.a /*10,11*/) /*10,11*/, (this.b /*13,14*/) /*13,14*/) /*3,15*/`)
	})

	t.Run("safe calls", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ method?.(a, b) }}", nil, nil), `(0 as any ? ((this.method /*3,9*/) /*3,9*/)!((this.a /*12,13*/) /*12,13*/, (this.b /*15,16*/) /*15,16*/) : undefined) /*3,17*/`)
	})

	t.Run("method calls of variables", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("<ng-template let-method>{{ method(a, b) }}</ng-template>", nil, nil), `method /*3,9*/((this.a /*34,35*/) /*10,11*/, (this.b /*37,38*/) /*13,14*/) /*3,15*/`)
	})

	t.Run("function calls", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ method(a)(b, c) }}", nil, nil), `this.method /*3,9*/((this.a /*10,11*/) /*10,11*/) /*3,12*/((this.b /*13,14*/) /*13,14*/, (this.c /*16,17*/) /*16,17*/) /*3,18*/`)
	})

	t.Run("property access", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a.b.c }}", nil, nil), `(((((this.a /*3,4*/) /*3,4*/).b /*5,6*/) /*3,6*/).c /*7,8*/) /*3,8*/`)
	})

	t.Run("property writes", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("<div (click)='a.b.c = d'></div>", nil, nil), `(((((((this.a /*14,15*/) /*0,1*/).b /*16,17*/) /*0,3*/).c /*6,7*/) /*0,5*/)) = (((this.d /*22,23*/) /*8,9*/)) /*0,9*/`)
	})

	t.Run("$event property writes", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("<div (click)='a = $event'></div>", nil, nil), `(((this.a /*14,15*/) /*0,1*/)) = (_event /*4,10*/) /*0,10*/`)
	})

	t.Run("keyed property access", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a[b] }}", nil, nil), `((this.a /*3,4*/) /*3,4*/)[((this.b /*5,6*/) /*5,6*/)] /*3,7*/`)
	})

	t.Run("keyed property writes", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans(`<div (click)="a[b] = c"></div>`, nil, nil), `(((this.a /*14,15*/) /*0,1*/)[((this.b /*16,17*/) /*2,3*/)] /*0,4*/) = (((this.c /*9,10*/) /*7,8*/)) /*0,8*/`)
	})

	t.Run("safe property access", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a?.b }}", nil, nil), `(((this.a /*3,4*/) /*3,4*/)?.b /*6,7*/ /*3,7*/)`)
	})

	t.Run("safe method calls", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a?.method(b) }}", nil, nil), `((0 as any ? ((this.a /*3,4*/) /*3,4*/)?.method /*6,12*/ /*3,12*/!((this.b /*13,14*/) /*13,14*/) : undefined) /*3,15*/)`)
	})

	t.Run("safe keyed reads", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ a?.[0] }}", nil, nil), `(((this.a /*3,4*/) /*3,4*/)?.[0 /*7,8*/] /*3,9*/)`)
	})

	t.Run("$any casts", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("{{ $any(a) }}", nil, nil), `((this.a /*8,9*/) /*8,9*/ as any) /*3,10*/`)
	})

	t.Run("chained expressions", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("<div (click)='a; b; c'></div>", nil, nil), `(((this.a /*14,15*/) /*0,1*/), ((this.b /*17,18*/) /*3,4*/), ((this.c /*6,7*/) /*6,7*/)) /*0,7*/`)
	})

	t.Run("pipe usages", func(t *testing.T) {
		declarations := []TestDeclaration{
			{
				Type:     "pipe",
				Name:     "TestPipe",
				PipeName: "test",
			},
		}
		pipes := map[string]string{
			"test": "i0.TestPipe",
		}
		block := tcbWithSpans("{{ a | test:b }}", declarations, pipes)
		assert.Contains(t, block, "var _pipe1 = null! as i0.TestPipe;")
		assert.Contains(t, block, "_pipe1.transform /*7,11*/((this.a /*3,4*/) /*3,4*/, (this.b /*12,13*/) /*12,13*/) /*3,13*/")
	})

	t.Run("element refs", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("<span #a></span>{{ a || a }}", nil, nil), "((a /*3,4*/) || (a /*8,9*/) /*3,9*/)")
	})

	t.Run("template vars", func(t *testing.T) {
		assert.Contains(t, tcbWithSpans("<ng-template let-a='b'>{{ a || a }}</ng-template>", nil, nil), "((a /*3,4*/) || (a /*8,9*/) /*3,9*/)")
	})

	t.Run("directive refs", func(t *testing.T) {
		declarations := []TestDeclaration{
			{
				Type:        "directive",
				Name:        "MyComponent",
				Selector:    "my-cmp",
				IsStandalone: true,
			},
		}
		block := tcbWithSpans("<my-cmp #a></my-cmp>{{ a || a }}", declarations, nil)
		assert.Contains(t, block, "((a /*3,4*/) || (a /*8,9*/) /*3,9*/)")
	})

	t.Run("generic directive inputs", func(t *testing.T) {
		declarations := []TestDeclaration{
			{
				Type:        "directive",
				Name:        "MyComponent",
				Selector:    "my-cmp",
				IsStandalone: true,
				IsGeneric:    true,
				Inputs:       map[string]string{"inputA": "inputA"},
			},
		}
		block := tcbWithSpans(`<my-cmp [inputA]="''"></my-cmp>`, declarations, nil)
		assert.Contains(t, block, `_assign(_dir1.inputA, ("" /*0,2*/));`)
	})

	t.Run("control flow @for", func(t *testing.T) {
		template := `@for (user of users; track user; let i = $index) { {{i}} }`
		block := tcbWithSpans(template, nil, nil)
		assert.Contains(t, block, "for (let user of ((this.users /*14,19*/) /*0,5*/))")
	})

	t.Run("control flow @if", func(t *testing.T) {
		template := `@if (x; as alias) { {{alias}} }`
		block := tcbWithSpans(template, nil, nil)
		assert.Contains(t, block, "if ((this.x /*5,6*/) /*0,1*/)")
	})
}
