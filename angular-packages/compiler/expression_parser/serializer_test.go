package expression_parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

func parse(expression string) expression_parser.ASTWithSource {
	parser := expression_parser.NewParser(expression_parser.Lexer{}, false)
	return parser.ParseBinding(expression, expression_parser.ParseSourceSpan{}, 0)
}

func parseAction(expression string) expression_parser.ASTWithSource {
	parser := expression_parser.NewParser(expression_parser.Lexer{}, false)
	return parser.ParseAction(expression, expression_parser.ParseSourceSpan{}, 0)
}

func serialize(expression expression_parser.ASTWithSource) string {
	return expression_parser.Serialize(expression.Ast)
}

func TestSerializer(t *testing.T) {
	t.Run("serialize", func(t *testing.T) {
		t.Run("serializes unary plus", func(t *testing.T) {
			assert.Equal(t, "+1234", serialize(parse(" + 1234 ")))
		})

		t.Run("serializes unary negative", func(t *testing.T) {
			assert.Equal(t, "-1234", serialize(parse(" - 1234 ")))
		})

		t.Run("serializes binary operations", func(t *testing.T) {
			assert.Equal(t, "1234 + 4321", serialize(parse(" 1234   +   4321 ")))
		})

		t.Run("serializes exponentiation", func(t *testing.T) {
			assert.Equal(t, "1 * 2 ** 3", serialize(parse(" 1  *  2  **  3 ")))
		})

		t.Run("serializes chains", func(t *testing.T) {
			assert.Equal(t, "1234; 4321", serialize(parseAction(" 1234;   4321 ")))
		})

		t.Run("serializes conditionals", func(t *testing.T) {
			assert.Equal(t, "cond ? 1234 : 4321", serialize(parse(" cond   ?   1234   :   4321 ")))
		})

		t.Run("serializes `this`", func(t *testing.T) {
			assert.Equal(t, "this", serialize(parse(" this ")))
		})

		t.Run("serializes keyed reads", func(t *testing.T) {
			assert.Equal(t, "foo[bar]", serialize(parse(" foo   [bar] ")))
		})

		t.Run("serializes keyed write", func(t *testing.T) {
			assert.Equal(t, "foo[bar] = baz", serialize(parse(" foo   [bar]   =   baz ")))
		})

		t.Run("serializes array literals", func(t *testing.T) {
			assert.Equal(t, "[foo, bar, baz]", serialize(parse(" [   foo,   bar,   baz   ] ")))
		})

		t.Run("serializes object literals", func(t *testing.T) {
			assert.Equal(t, "{foo: bar, baz: test}", serialize(parse(" {   foo:   bar,   baz:   test   } ")))
		})

		t.Run("serializes primitives", func(t *testing.T) {
			assert.Equal(t, "'test'", serialize(parse(` 'test' `)))
			assert.Equal(t, "'test'", serialize(parse(` "test" `)))
			assert.Equal(t, "true", serialize(parse(" true ")))
			assert.Equal(t, "false", serialize(parse(" false ")))
			assert.Equal(t, "1234", serialize(parse(" 1234 ")))
			assert.Equal(t, "null", serialize(parse(" null ")))
			assert.Equal(t, "undefined", serialize(parse(" undefined ")))
		})

		t.Run("escapes string literals", func(t *testing.T) {
			assert.Equal(t, `'Hello, \'World\'...'`, serialize(parse(` 'Hello, \'World\'...' `)))
			assert.Equal(t, `'Hello, "World"...'`, serialize(parse(` 'Hello, \"World\"...' `)))
		})

		t.Run("serializes pipes", func(t *testing.T) {
			assert.Equal(t, "foo | pipe", serialize(parse(" foo   |   pipe ")))
		})

		t.Run("serializes not prefixes", func(t *testing.T) {
			assert.Equal(t, "!foo", serialize(parse(" !   foo ")))
		})

		t.Run("serializes non-null assertions", func(t *testing.T) {
			assert.Equal(t, "foo!", serialize(parse(" foo   ! ")))
		})

		t.Run("serializes property reads", func(t *testing.T) {
			assert.Equal(t, "foo.bar", serialize(parse(" foo   .   bar ")))
		})

		t.Run("serializes property writes", func(t *testing.T) {
			assert.Equal(t, "foo.bar = baz", serialize(parseAction(" foo   .   bar   =   baz ")))
		})

		t.Run("serializes safe property reads", func(t *testing.T) {
			assert.Equal(t, "foo?.bar", serialize(parse(" foo   ?.   bar ")))
		})

		t.Run("serializes safe keyed reads", func(t *testing.T) {
			assert.Equal(t, "foo?.[bar]", serialize(parse(" foo   ?.   [   bar   ] ")))
		})

		t.Run("serializes calls", func(t *testing.T) {
			assert.Equal(t, "foo()", serialize(parse(" foo   (   ) ")))
			assert.Equal(t, "foo(bar)", serialize(parse(" foo   (   bar   ) ")))
			assert.Equal(t, "foo(bar, )", serialize(parse(" foo   (   bar   ,   ) ")))
			assert.Equal(t, "foo(bar, baz)", serialize(parse(" foo   (   bar   ,   baz   ) ")))
		})

		t.Run("serializes safe calls", func(t *testing.T) {
			assert.Equal(t, "foo?.()", serialize(parse(" foo   ?.   (   ) ")))
			assert.Equal(t, "foo?.(bar)", serialize(parse(" foo   ?.   (   bar   ) ")))
			assert.Equal(t, "foo?.(bar, )", serialize(parse(" foo   ?.   (   bar   ,   ) ")))
			assert.Equal(t, "foo?.(bar, baz)", serialize(parse(" foo   ?.   (   bar   ,   baz   ) ")))
		})

		t.Run("serializes void expressions", func(t *testing.T) {
			assert.Equal(t, "void 0", serialize(parse(" void   0 ")))
		})

		t.Run("serializes in expressions", func(t *testing.T) {
			assert.Equal(t, "foo in bar", serialize(parse(" foo   in   bar ")))
		})

		t.Run("serializes instanceof expressions", func(t *testing.T) {
			assert.Equal(t, "foo instanceof Bar", serialize(parse(" foo   instanceof   Bar ")))
		})
	})
}
