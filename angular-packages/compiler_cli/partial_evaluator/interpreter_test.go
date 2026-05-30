package partial_evaluator_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
)

func TestPartialEvaluator_EvaluateObjectLiteral(t *testing.T) {
	sourceText := `
const config = { selector: 'my-app', standalone: true, items: ['a', 'b'] };
`
	opts := ast.SourceFileParseOptions{
		FileName: "/test.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)
	if sourceFile == nil {
		t.Fatal("Failed to parse source file")
	}

	// Find the object literal expression
	var objNode *ast.Node
	// Simply walk the statements and use ForEachChild or type casts
	for _, stmt := range sourceFile.Statements.Nodes {
		stmt.ForEachChild(func(child *ast.Node) bool {
			child.ForEachChild(func(subChild *ast.Node) bool {
				if ast.IsObjectLiteralExpression(subChild) {
					objNode = subChild
				}
				subChild.ForEachChild(func(grandChild *ast.Node) bool {
					if ast.IsObjectLiteralExpression(grandChild) {
						objNode = grandChild
					}
					return false
				})
				return false
			})
			return false
		})
	}

	if objNode == nil {
		t.Fatal("Could not find object literal in AST")
	}

	host := reflection.NewTypeScriptReflectionHost(nil)
	evaluator := partial_evaluator.NewPartialEvaluator(host, nil, nil)

	result := evaluator.Evaluate(objNode, nil)
	if result == nil {
		t.Fatal("Evaluator returned nil")
	}

	resolvedMap, ok := result.(partial_evaluator.ResolvedValueMap)
	if !ok {
		t.Fatalf("Expected ResolvedValueMap, got %T", result)
	}

	if selector, ok := resolvedMap["selector"].(string); !ok || selector != "my-app" {
		t.Errorf("Expected selector 'my-app', got %v", resolvedMap["selector"])
	}

	if standalone, ok := resolvedMap["standalone"].(bool); !ok || !standalone {
		t.Errorf("Expected standalone true, got %v", resolvedMap["standalone"])
	}

	items, ok := resolvedMap["items"].(partial_evaluator.ResolvedValueArray)
	if !ok || len(items) != 2 {
		t.Fatalf("Expected ResolvedValueArray of length 2, got %v", resolvedMap["items"])
	}

	if items[0] != "a" || items[1] != "b" {
		t.Errorf("Expected items ['a', 'b'], got %v", items)
	}
}
