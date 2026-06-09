package annotations_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestComponentDecoratorHandler_DetectAndAnalyze(t *testing.T) {
	sourceText := `
@Component({
	selector: 'app-root',
	standalone: true,
	template: '<h1>Hello</h1>'
})
export class AppComponent {}
`
	sourceFile := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/entry.ts"}, sourceText, core.ScriptKindTS)
	if sourceFile == nil || len(sourceFile.Statements.Nodes) == 0 {
		t.Fatal("Failed to parse source file")
	}

	var classNode *ast.ClassDeclaration
	for _, stmt := range sourceFile.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) {
			classNode = stmt.AsClassDeclaration()
			break
		}
	}
	if classNode == nil {
		t.Fatal("Could not find class in AST")
	}

	host := reflection.NewTypeScriptReflectionHost(nil)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)
	handler := annotations.NewComponentDecoratorHandler(host, false, metaRegistry, scopeRegistry, nil, false)

	// Test Name
	assert.Equal(t, "ComponentDecoratorHandler", handler.Name())

	// Test Detect
	decs := host.GetDecoratorsOfDeclaration(classNode.AsNode())
	detected := handler.Detect(classNode, decs)
	assert.NotNil(t, detected)
	assert.Equal(t, "Component", detected.Name)

	// Test Analyze
	analysis, diags := handler.Analyze(classNode, detected)
	assert.Empty(t, diags)
	assert.NotNil(t, analysis)

	compAnalysis, ok := analysis.(*annotations.ComponentAnalysis)
	assert.True(t, ok)
	assert.Equal(t, "app-root", compAnalysis.Selector)
	assert.True(t, compAnalysis.IsStandalone)
	assert.Equal(t, "<h1>Hello</h1>", compAnalysis.Template)
}
