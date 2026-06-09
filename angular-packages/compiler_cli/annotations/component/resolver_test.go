package component

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
)

func TestLegacyAnimationTriggerResolver_PrivateIdentifier(t *testing.T) {
	sourceText := `this.#trigger()`
	opts := ast.SourceFileParseOptions{
		FileName: "/test.ts",
	}
	sourceFile := parser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)
	if sourceFile == nil || len(sourceFile.Statements.Nodes) == 0 {
		t.Fatal("Failed to parse source text")
	}

	exprStmt := sourceFile.Statements.Nodes[0].AsExpressionStatement()
	callExpr := exprStmt.Expression.AsCallExpression()
	fn := callExpr.Expression

	resolve := func(expr *ast.Node) partial_evaluator.ResolvedValue {
		return nil
	}
	unresolvable := &partial_evaluator.DynamicValue{Node: fn}

	// This call should not panic even when fn.AsPropertyAccessExpression().Name() is a PrivateIdentifier.
	res := legacyAnimationTriggerResolver(fn, callExpr, resolve, unresolvable)
	if res != unresolvable {
		t.Errorf("Expected unresolvable, got %v", res)
	}
}
