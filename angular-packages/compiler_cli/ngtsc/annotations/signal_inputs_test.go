package annotations

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
)

func parseExpression(t *testing.T, sourceText string) *ast.Node {
	t.Helper()
	sourceFile := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/entry.ts"}, sourceText, core.ScriptKindTS)
	if sourceFile == nil || len(sourceFile.Statements.Nodes) == 0 {
		t.Fatal("Failed to parse source text")
	}
	exprStmt := sourceFile.Statements.Nodes[0].AsExpressionStatement()
	return exprStmt.Expression
}

func TestParseSignalInput(t *testing.T) {
	// 1. input(1)
	expr1 := parseExpression(t, `input(1)`)
	meta1, ok1 := parseSignalInput(expr1, "prop")
	assert.True(t, ok1)
	assert.Equal(t, "prop", meta1.ClassPropertyName)
	assert.Equal(t, "prop", meta1.BindingPropertyName)
	assert.False(t, meta1.Required)
	assert.True(t, meta1.IsSignal)

	// 2. input.required()
	expr2 := parseExpression(t, `input.required()`)
	meta2, ok2 := parseSignalInput(expr2, "prop")
	assert.True(t, ok2)
	assert.Equal(t, "prop", meta2.ClassPropertyName)
	assert.Equal(t, "prop", meta2.BindingPropertyName)
	assert.True(t, meta2.Required)
	assert.True(t, meta2.IsSignal)

	// 3. input(1, {alias: 'customAlias'})
	expr3 := parseExpression(t, `input(1, {alias: 'customAlias'})`)
	meta3, ok3 := parseSignalInput(expr3, "prop")
	assert.True(t, ok3)
	assert.Equal(t, "prop", meta3.ClassPropertyName)
	assert.Equal(t, "customAlias", meta3.BindingPropertyName)
	assert.False(t, meta3.Required)

	// 4. input.required({alias: 'customAliasRequired'})
	expr4 := parseExpression(t, `input.required({alias: 'customAliasRequired'})`)
	meta4, ok4 := parseSignalInput(expr4, "prop")
	assert.True(t, ok4)
	assert.Equal(t, "prop", meta4.ClassPropertyName)
	assert.Equal(t, "customAliasRequired", meta4.BindingPropertyName)
	assert.True(t, meta4.Required)
}

func TestParseModelInput(t *testing.T) {
	// 1. model(1)
	expr1 := parseExpression(t, `model(1)`)
	meta1, ok1 := parseModelInput(expr1, "prop")
	assert.True(t, ok1)
	assert.Equal(t, "prop", meta1.ClassPropertyName)
	assert.Equal(t, "prop", meta1.BindingPropertyName)
	assert.False(t, meta1.Required)
	assert.True(t, meta1.IsSignal)

	// 2. model.required()
	expr2 := parseExpression(t, `model.required()`)
	meta2, ok2 := parseModelInput(expr2, "prop")
	assert.True(t, ok2)
	assert.Equal(t, "prop", meta2.ClassPropertyName)
	assert.Equal(t, "prop", meta2.BindingPropertyName)
	assert.True(t, meta2.Required)
	assert.True(t, meta2.IsSignal)

	// 3. model(1, {alias: 'customAlias'})
	expr3 := parseExpression(t, `model(1, {alias: 'customAlias'})`)
	meta3, ok3 := parseModelInput(expr3, "prop")
	assert.True(t, ok3)
	assert.Equal(t, "prop", meta3.ClassPropertyName)
	assert.Equal(t, "customAlias", meta3.BindingPropertyName)
	assert.False(t, meta3.Required)

	// 4. model.required({alias: 'customAliasRequired'})
	expr4 := parseExpression(t, `model.required({alias: 'customAliasRequired'})`)
	meta4, ok4 := parseModelInput(expr4, "prop")
	assert.True(t, ok4)
	assert.Equal(t, "prop", meta4.ClassPropertyName)
	assert.Equal(t, "customAliasRequired", meta4.BindingPropertyName)
	assert.True(t, meta4.Required)
}
