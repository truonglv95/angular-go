package render3

import (
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

func assertContainsSpan(t *testing.T, nodes []Node, expectedExpr string, expectedSpan expression_parser.AbsoluteSourceSpan) {
	humanized := humanizeExpressionSource(nodes)
	found := false
	for _, h := range humanized {
		if h.AST == expectedExpr && h.Span == expectedSpan {
			found = true
			break
		}
	}
	if !found {
		var list []string
		for _, h := range humanized {
			list = append(list, fmt.Sprintf("[%q, {%d, %d}]", h.AST, h.Span.Start, h.Span.End))
		}
		t.Errorf("Expected to find [%q, {%d, %d}] in:\n%s", expectedExpr, expectedSpan.Start, expectedSpan.End, strings.Join(list, "\n"))
	}
}

func assertContainsSpans(t *testing.T, nodes []Node, expected []HumanizedExpressionSource) {
	humanized := humanizeExpressionSource(nodes)
	for _, exp := range expected {
		found := false
		for _, h := range humanized {
			if h.AST == exp.AST && h.Span == exp.Span {
				found = true
				break
			}
		}
		if !found {
			var list []string
			for _, h := range humanized {
				list = append(list, fmt.Sprintf("[%q, {%d, %d}]", h.AST, h.Span.Start, h.Span.End))
			}
			t.Errorf("Expected to find [%q, {%d, %d}] in:\n%s", exp.AST, exp.Span.Start, exp.Span.End, strings.Join(list, "\n"))
		}
	}
}

func TestAbsoluteSpan(t *testing.T) {
	t.Run("should handle comment in interpolation", func(t *testing.T) {
		nodes := parseR3("{{foo // comment}}", ParseR3Options{PreserveWhitespaces: true}).Nodes
		assertContainsSpan(t, nodes, "foo", expression_parser.NewAbsoluteSourceSpan(2, 5))
	})

	t.Run("should handle whitespace in interpolation", func(t *testing.T) {
		nodes := parseR3("{{  foo  }}", ParseR3Options{PreserveWhitespaces: true}).Nodes
		assertContainsSpan(t, nodes, "foo", expression_parser.NewAbsoluteSourceSpan(4, 7))
	})

	t.Run("should handle whitespace and comment in interpolation", func(t *testing.T) {
		nodes := parseR3("{{  foo // comment  }}", ParseR3Options{PreserveWhitespaces: true}).Nodes
		assertContainsSpan(t, nodes, "foo", expression_parser.NewAbsoluteSourceSpan(4, 7))
	})

	t.Run("should handle comment in an action binding", func(t *testing.T) {
		nodes := parseR3("<button (click)=\"foo = true // comment\">Save</button>", ParseR3Options{PreserveWhitespaces: true}).Nodes
		assertContainsSpan(t, nodes, "foo = true", expression_parser.NewAbsoluteSourceSpan(17, 27))
	})

	t.Run("should provide absolute offsets with arbitrary whitespace", func(t *testing.T) {
		nodes := parseR3("<div>\n  \n{{foo}}</div>", ParseR3Options{PreserveWhitespaces: true}).Nodes
		assertContainsSpan(t, nodes, "\n  \n{{ foo }}", expression_parser.NewAbsoluteSourceSpan(5, 16))
	})

	t.Run("should provide absolute offsets of an expression in a bound text", func(t *testing.T) {
		nodes := parseR3("<div>{{foo}}</div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "{{ foo }}", expression_parser.NewAbsoluteSourceSpan(5, 12))
	})

	t.Run("should provide absolute offsets of an expression in a bound event", func(t *testing.T) {
		nodes1 := parseR3("<div (click)=\"foo();bar();\"></div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes1, "foo(); bar();", expression_parser.NewAbsoluteSourceSpan(14, 26))

		nodes2 := parseR3("<div on-click=\"foo();bar();\"></div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes2, "foo(); bar();", expression_parser.NewAbsoluteSourceSpan(15, 27))
	})

	t.Run("should provide absolute offsets of an expression in a bound attribute", func(t *testing.T) {
		nodes1 := parseR3("<input [disabled]=\"condition ? true : false\" />", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes1, "condition ? true : false", expression_parser.NewAbsoluteSourceSpan(19, 43))

		nodes2 := parseR3("<input bind-disabled=\"condition ? true : false\" />", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes2, "condition ? true : false", expression_parser.NewAbsoluteSourceSpan(22, 46))
	})

	t.Run("should provide absolute offsets of an expression in a template attribute", func(t *testing.T) {
		nodes := parseR3("<div *ngIf=\"value | async\"></div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "(value | async)", expression_parser.NewAbsoluteSourceSpan(12, 25))
	})

	t.Run("binary expression - should provide absolute offsets of a binary expression", func(t *testing.T) {
		nodes := parseR3("<div>{{1 + 2}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "1 + 2", expression_parser.NewAbsoluteSourceSpan(7, 12))
	})

	t.Run("binary expression - should provide absolute offsets of expressions in a binary expression", func(t *testing.T) {
		nodes := parseR3("<div>{{1 + 2}}<div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "1", Span: expression_parser.NewAbsoluteSourceSpan(7, 8)},
			{AST: "2", Span: expression_parser.NewAbsoluteSourceSpan(11, 12)},
		})
	})

	t.Run("conditional - should provide absolute offsets of a conditional", func(t *testing.T) {
		nodes := parseR3("<div>{{bool ? 1 : 0}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "bool ? 1 : 0", expression_parser.NewAbsoluteSourceSpan(7, 19))
	})

	t.Run("conditional - should provide absolute offsets of expressions in a conditional", func(t *testing.T) {
		nodes := parseR3("<div>{{bool ? 1 : 0}}<div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "bool", Span: expression_parser.NewAbsoluteSourceSpan(7, 11)},
			{AST: "1", Span: expression_parser.NewAbsoluteSourceSpan(14, 15)},
			{AST: "0", Span: expression_parser.NewAbsoluteSourceSpan(18, 19)},
		})
	})

	t.Run("chain - should provide absolute offsets of a chain", func(t *testing.T) {
		nodes := parseR3("<div (click)=\"a(); b();\"><div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "a(); b();", expression_parser.NewAbsoluteSourceSpan(14, 23))
	})

	t.Run("chain - should provide absolute offsets of expressions in a chain", func(t *testing.T) {
		nodes := parseR3("<div (click)=\"a(); b();\"><div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "a()", Span: expression_parser.NewAbsoluteSourceSpan(14, 17)},
			{AST: "b()", Span: expression_parser.NewAbsoluteSourceSpan(19, 22)},
		})
	})

	t.Run("function call - should provide absolute offsets of a function call", func(t *testing.T) {
		nodes := parseR3("<div>{{fn()()}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "fn()()", expression_parser.NewAbsoluteSourceSpan(7, 13))
	})

	t.Run("function call - should provide absolute offsets of expressions in a function call", func(t *testing.T) {
		nodes := parseR3("<div>{{fn()(param)}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "param", expression_parser.NewAbsoluteSourceSpan(12, 17))
	})

	t.Run("should provide absolute offsets of an implicit receiver", func(t *testing.T) {
		nodes := parseR3("<div>{{a.b}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "", expression_parser.NewAbsoluteSourceSpan(7, 7))
	})

	t.Run("interpolation - should provide absolute offsets of an interpolation", func(t *testing.T) {
		nodes := parseR3("<div>{{1 + foo.length}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "{{ 1 + foo.length }}", expression_parser.NewAbsoluteSourceSpan(5, 23))
	})

	t.Run("interpolation - should provide absolute offsets of expressions in an interpolation", func(t *testing.T) {
		nodes := parseR3("<div>{{1 + 2}}<div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "1", Span: expression_parser.NewAbsoluteSourceSpan(7, 8)},
			{AST: "2", Span: expression_parser.NewAbsoluteSourceSpan(11, 12)},
		})
	})

	t.Run("interpolation - should handle HTML entity before interpolation", func(t *testing.T) {
		nodes := parseR3("&nbsp;{{abc}}", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "abc", Span: expression_parser.NewAbsoluteSourceSpan(8, 11)},
		})
	})

	t.Run("interpolation - should handle many HTML entities and many interpolations", func(t *testing.T) {
		nodes := parseR3("&quot;{{abc}}&quot;{{def}}&nbsp;{{ghi}}", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "abc", Span: expression_parser.NewAbsoluteSourceSpan(8, 11)},
			{AST: "def", Span: expression_parser.NewAbsoluteSourceSpan(21, 24)},
			{AST: "ghi", Span: expression_parser.NewAbsoluteSourceSpan(34, 37)},
		})
	})

	t.Run("interpolation - should handle interpolation in attribute", func(t *testing.T) {
		nodes := parseR3("<div class=\"{{abc}}\"><div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "abc", Span: expression_parser.NewAbsoluteSourceSpan(14, 17)},
		})
	})

	t.Run("interpolation - should handle interpolation preceded by HTML entity in attribute", func(t *testing.T) {
		nodes := parseR3("<div class=\"&nbsp;{{abc}}\"><div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "abc", Span: expression_parser.NewAbsoluteSourceSpan(20, 23)},
		})
	})

	t.Run("interpolation - should handle many interpolation with HTML entities in attribute", func(t *testing.T) {
		nodes := parseR3("<div class=\"&quot;{{abc}}&quot;&nbsp;{{def}}\"><div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "abc", Span: expression_parser.NewAbsoluteSourceSpan(20, 23)},
			{AST: "def", Span: expression_parser.NewAbsoluteSourceSpan(39, 42)},
		})
	})

	t.Run("keyed read - should provide absolute offsets of a keyed read", func(t *testing.T) {
		nodes := parseR3("<div>{{obj[key]}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "obj[key]", expression_parser.NewAbsoluteSourceSpan(7, 15))
	})

	t.Run("keyed read - should provide absolute offsets of expressions in a keyed read", func(t *testing.T) {
		nodes := parseR3("<div>{{obj[key]}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "key", expression_parser.NewAbsoluteSourceSpan(11, 14))
	})

	t.Run("keyed write - should provide absolute offsets of a keyed write", func(t *testing.T) {
		nodes := parseR3("<div>{{obj[key] = 0}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "obj[key] = 0", expression_parser.NewAbsoluteSourceSpan(7, 19))
	})

	t.Run("keyed write - should provide absolute offsets of expressions in a keyed write", func(t *testing.T) {
		nodes := parseR3("<div>{{obj[key] = 0}}<div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "key", Span: expression_parser.NewAbsoluteSourceSpan(11, 14)},
			{AST: "0", Span: expression_parser.NewAbsoluteSourceSpan(18, 19)},
		})
	})

	t.Run("should provide absolute offsets of a literal primitive", func(t *testing.T) {
		nodes := parseR3("<div>{{100}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "100", expression_parser.NewAbsoluteSourceSpan(7, 10))
	})

	t.Run("literal array - should provide absolute offsets of a literal array", func(t *testing.T) {
		nodes := parseR3("<div>{{[0, 1, 2]}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "[0, 1, 2]", expression_parser.NewAbsoluteSourceSpan(7, 16))
	})

	t.Run("literal array - should provide absolute offsets of expressions in a literal array", func(t *testing.T) {
		nodes := parseR3("<div>{{[0, 1, 2]}}<div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "0", Span: expression_parser.NewAbsoluteSourceSpan(8, 9)},
			{AST: "1", Span: expression_parser.NewAbsoluteSourceSpan(11, 12)},
			{AST: "2", Span: expression_parser.NewAbsoluteSourceSpan(14, 15)},
		})
	})

	t.Run("literal map - should provide absolute offsets of a literal map", func(t *testing.T) {
		nodes := parseR3("<div>{{ {a: 0} }}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "{a: 0}", expression_parser.NewAbsoluteSourceSpan(8, 14))
	})

	t.Run("literal map - should provide absolute offsets of expressions in a literal map", func(t *testing.T) {
		nodes := parseR3("<div>{{ {a: 0} }}<div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "0", Span: expression_parser.NewAbsoluteSourceSpan(12, 13)},
		})
	})

	t.Run("method call - should provide absolute offsets of a method call", func(t *testing.T) {
		nodes := parseR3("<div>{{method()}}</div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "method()", expression_parser.NewAbsoluteSourceSpan(7, 15))
	})

	t.Run("method call - should provide absolute offsets of expressions in a method call", func(t *testing.T) {
		nodes := parseR3("<div>{{method(param)}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "param", expression_parser.NewAbsoluteSourceSpan(14, 19))
	})

	t.Run("non-null assert - should provide absolute offsets of a non-null assert", func(t *testing.T) {
		nodes := parseR3("<div>{{prop!}}</div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop!", expression_parser.NewAbsoluteSourceSpan(7, 12))
	})

	t.Run("non-null assert - should provide absolute offsets of expressions in a non-null assert", func(t *testing.T) {
		nodes := parseR3("<div>{{prop!}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop", expression_parser.NewAbsoluteSourceSpan(7, 11))
	})

	t.Run("pipe - should provide absolute offsets of a pipe", func(t *testing.T) {
		nodes := parseR3("<div>{{prop | pipe}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "(prop | pipe)", expression_parser.NewAbsoluteSourceSpan(7, 18))
	})

	t.Run("pipe - should provide absolute offsets expressions in a pipe", func(t *testing.T) {
		nodes := parseR3("<div>{{prop | pipe}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop", expression_parser.NewAbsoluteSourceSpan(7, 11))
	})

	t.Run("property read - should provide absolute offsets of a property read", func(t *testing.T) {
		nodes := parseR3("<div>{{prop.obj}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop.obj", expression_parser.NewAbsoluteSourceSpan(7, 15))
	})

	t.Run("property read - should provide absolute offsets of expressions in a property read", func(t *testing.T) {
		nodes := parseR3("<div>{{prop.obj}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop", expression_parser.NewAbsoluteSourceSpan(7, 11))
	})

	t.Run("property write - should provide absolute offsets of a property write", func(t *testing.T) {
		nodes := parseR3("<div (click)=\"prop = 0\"></div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop = 0", expression_parser.NewAbsoluteSourceSpan(14, 22))
	})

	t.Run("property write - should provide absolute offsets of an accessed property write", func(t *testing.T) {
		nodes := parseR3("<div (click)=\"prop.inner = 0\"></div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop.inner = 0", expression_parser.NewAbsoluteSourceSpan(14, 28))
	})

	t.Run("property write - should provide absolute offsets of expressions in a property write", func(t *testing.T) {
		nodes := parseR3("<div (click)=\"prop = 0\"></div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "0", expression_parser.NewAbsoluteSourceSpan(21, 22))
	})

	t.Run("\"not\" prefix - should provide absolute offsets of a \"not\" prefix", func(t *testing.T) {
		nodes := parseR3("<div>{{!prop}}</div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "!prop", expression_parser.NewAbsoluteSourceSpan(7, 12))
	})

	t.Run("\"not\" prefix - should provide absolute offsets of expressions in a \"not\" prefix", func(t *testing.T) {
		nodes := parseR3("<div>{{!prop}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop", expression_parser.NewAbsoluteSourceSpan(8, 12))
	})

	t.Run("safe method call - should provide absolute offsets of a safe method call", func(t *testing.T) {
		nodes := parseR3("<div>{{prop?.safe()}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop?.safe()", expression_parser.NewAbsoluteSourceSpan(7, 19))
	})

	t.Run("safe method call - should provide absolute offsets of expressions in safe method call", func(t *testing.T) {
		nodes := parseR3("<div>{{prop?.safe()}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop", expression_parser.NewAbsoluteSourceSpan(7, 11))
	})

	t.Run("safe property read - should provide absolute offsets of a safe property read", func(t *testing.T) {
		nodes := parseR3("<div>{{prop?.safe}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop?.safe", expression_parser.NewAbsoluteSourceSpan(7, 17))
	})

	t.Run("safe property read - should provide absolute offsets of expressions in safe property read", func(t *testing.T) {
		nodes := parseR3("<div>{{prop?.safe}}<div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "prop", expression_parser.NewAbsoluteSourceSpan(7, 11))
	})

	t.Run("absolute offsets for template expressions - should work for simple cases", func(t *testing.T) {
		nodes := parseR3("<div *ngFor=\"let item of items\">{{item}}</div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "items", expression_parser.NewAbsoluteSourceSpan(25, 30))
	})

	t.Run("absolute offsets for template expressions - should work with multiple bindings", func(t *testing.T) {
		nodes := parseR3("<div *ngFor=\"let a of As; let b of Bs\"></div>", ParseR3Options{}).Nodes
		assertContainsSpans(t, nodes, []HumanizedExpressionSource{
			{AST: "As", Span: expression_parser.NewAbsoluteSourceSpan(22, 24)},
			{AST: "Bs", Span: expression_parser.NewAbsoluteSourceSpan(35, 37)},
		})
	})

	t.Run("ICU expressions - is correct for variables and placeholders", func(t *testing.T) {
		nodes := parseR3("<span i18n>{item.var, plural, other { {{item.placeholder}} items } }</span>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "item.var", expression_parser.NewAbsoluteSourceSpan(12, 20))
		assertContainsSpan(t, nodes, "item.placeholder", expression_parser.NewAbsoluteSourceSpan(40, 56))
	})

	t.Run("ICU expressions - is correct for variables and placeholders in nested ICUs", func(t *testing.T) {
		nodes := parseR3("<span i18n>{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }</span>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "item.var", expression_parser.NewAbsoluteSourceSpan(12, 20))
		assertContainsSpan(t, nodes, "item.placeholder", expression_parser.NewAbsoluteSourceSpan(40, 56))
		assertContainsSpan(t, nodes, "nestedVar", expression_parser.NewAbsoluteSourceSpan(60, 69))
		assertContainsSpan(t, nodes, "nestedPlaceholder", expression_parser.NewAbsoluteSourceSpan(89, 106))
	})

	t.Run("object literal - is correct for object literals with shorthand property declarations", func(t *testing.T) {
		nodes := parseR3("<div (click)=\"test({a: 1, b, c: 3, foo})\"></div>", ParseR3Options{}).Nodes
		assertContainsSpan(t, nodes, "{a: 1, b: b, c: 3, foo: foo}", expression_parser.NewAbsoluteSourceSpan(19, 39))
		assertContainsSpan(t, nodes, "b", expression_parser.NewAbsoluteSourceSpan(26, 27))
		assertContainsSpan(t, nodes, "foo", expression_parser.NewAbsoluteSourceSpan(35, 38))
	})
}
