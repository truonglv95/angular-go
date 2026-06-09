package translator_test

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func printNode(node *ast.Node) string {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	var stmts []*ast.Node
	if node.Kind == ast.KindSourceFile {
		return printer.NewPrinter(printer.PrinterOptions{NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, nil).EmitSourceFile(node.AsSourceFile())
	}
	if node.Kind == ast.KindBlock {
		stmts = node.AsBlock().Statements.Nodes
	} else if isStatement(node) {
		stmts = []*ast.Node{node}
	} else {
		// expression
		stmts = []*ast.Node{factory.NewExpressionStatement(node)}
	}
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "", factory.NewNodeList(stmts), factory.NewToken(ast.KindEndOfFile))
	text := printer.NewPrinter(printer.PrinterOptions{NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, nil).EmitSourceFile(sf.AsSourceFile())
	return strings.TrimSpace(text)
}

func isStatement(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindVariableStatement, ast.KindFunctionDeclaration, ast.KindExpressionStatement, ast.KindReturnStatement, ast.KindIfStatement, ast.KindBlock:
		return true
	default:
		return false
	}
}

func TestTranslator_VariableDeclaration(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	mgr := imports.NewImportManager(nil, factory)

	// const x = 42;
	valExpr := output.NewLiteralExpr(42, nil, nil, nil)
	stmt := output.NewDeclareVarStmt("x", valExpr, nil, output.StmtModifierFinal, nil, nil)

	visitor := translator.NewExpressionTranslatorVisitor(factory, mgr, nil, translator.TranslatorOptions{})
	translated := stmt.VisitStatement(visitor, translator.Context{IsStatementMode: true}).(*ast.Node)

	require.NotNil(t, translated)
	printed := printNode(translated)
	assert.Equal(t, "const x = 42;", printed)
}

func TestTranslator_LiteralExpressions(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	mgr := imports.NewImportManager(nil, factory)
	visitor := translator.NewExpressionTranslatorVisitor(factory, mgr, nil, translator.TranslatorOptions{})

	tests := []struct {
		expr     output.Expression
		expected string
	}{
		{output.NewLiteralExpr(42, nil, nil, nil), "42;"},
		{output.NewLiteralExpr(3.14, nil, nil, nil), "3.14;"},
		{output.NewLiteralExpr("hello", nil, nil, nil), `"hello";`},
		{output.NewLiteralExpr(true, nil, nil, nil), "true;"},
		{output.NewLiteralExpr(false, nil, nil, nil), "false;"},
		{output.NewLiteralExpr(nil, nil, nil, nil), "null;"},
	}

	for _, tc := range tests {
		translated := tc.expr.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
		printed := printNode(translated)
		assert.Equal(t, tc.expected, printed)
	}
}

func TestTranslator_BinaryOperator(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	mgr := imports.NewImportManager(nil, factory)
	visitor := translator.NewExpressionTranslatorVisitor(factory, mgr, nil, translator.TranslatorOptions{})

	lhs := output.NewReadVarExpr("a", nil, nil, nil)
	rhs := output.NewLiteralExpr(10, nil, nil, nil)
	expr := output.NewBinaryOperatorExpr(output.BinaryOperatorPlus, lhs, rhs, nil, nil, nil)

	translated := expr.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	printed := printNode(translated)
	assert.Equal(t, "a + 10;", printed)
}

func TestTranslator_ConditionalExpr(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	mgr := imports.NewImportManager(nil, factory)
	visitor := translator.NewExpressionTranslatorVisitor(factory, mgr, nil, translator.TranslatorOptions{})

	cond := output.NewReadVarExpr("isTrue", nil, nil, nil)
	trueCase := output.NewLiteralExpr("yes", nil, nil, nil)
	falseCase := output.NewLiteralExpr("no", nil, nil, nil)
	expr := output.NewConditionalExpr(cond, trueCase, falseCase, nil, nil, nil)

	translated := expr.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	printed := printNode(translated)
	assert.Equal(t, `isTrue ? "yes" : "no";`, printed)
}

func TestTranslator_DynamicImportExpr(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	mgr := imports.NewImportManager(nil, factory)
	visitor := translator.NewExpressionTranslatorVisitor(factory, mgr, nil, translator.TranslatorOptions{})

	expr := output.NewDynamicImportExpr("./module", nil, nil, nil)

	translated := expr.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	printed := printNode(translated)
	assert.Equal(t, `import("./module");`, printed)
}
