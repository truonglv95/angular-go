package annotations_test

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/printer"
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
