package component_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/component"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
)

func TestComponentDecoratorHandler_Analyze(t *testing.T) {
	sourceText := `
@Component({
	selector: 'app-root',
	standalone: true,
	template: '<h1>Hello</h1>'
})
export class AppComponent {}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/app.component.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)
	if sourceFile == nil {
		t.Fatal("Failed to parse source file")
	}

	var classNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) {
			classNode = stmt
			break
		}
	}

	if classNode == nil {
		t.Fatal("Could not find class in AST")
	}

	host := reflection.NewTypeScriptReflectionHost(nil)
	evaluator := partial_evaluator.NewPartialEvaluator(host, nil, nil)
	handler := component.NewComponentDecoratorHandler(host, evaluator)

	data, err := handler.Analyze(classNode)
	if err != nil {
		t.Fatalf("Analyze failed with error: %v", err)
	}

	if data == nil {
		t.Fatal("Expected ComponentAnalysisData, got nil")
	}

	if data.Selector != "app-root" {
		t.Errorf("Expected selector 'app-root', got '%s'", data.Selector)
	}

	if data.Standalone != true {
		t.Errorf("Expected standalone true, got false")
	}

	if data.Template != "<h1>Hello</h1>" {
		t.Errorf("Expected template '<h1>Hello</h1>', got '%s'", data.Template)
	}
}

func TestComponentDecoratorHandler_PrivateIdentifierPanic(t *testing.T) {
	sourceText := `
@Component({
	selector: 'app-root',
	standalone: true,
	template: '<h1>Hello</h1>',
	animations: [
		this.#trigger()
	]
})
export class AppComponent {}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/app.component.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)
	if sourceFile == nil {
		t.Fatal("Failed to parse source file")
	}

	var classNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) {
			classNode = stmt
			break
		}
	}

	if classNode == nil {
		t.Fatal("Could not find class in AST")
	}

	host := reflection.NewTypeScriptReflectionHost(nil)
	evaluator := partial_evaluator.NewPartialEvaluator(host, nil, nil)
	handler := component.NewComponentDecoratorHandler(host, evaluator)

	// This should run without panic
	_, _ = handler.Analyze(classNode)
}

