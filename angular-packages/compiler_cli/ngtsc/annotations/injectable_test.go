package annotations_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestInjectableDecoratorHandler_DetectAndAnalyze(t *testing.T) {
	sourceText := `
@Injectable({providedIn: 'root'})
export class MyService {}
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
	handler := annotations.NewInjectableDecoratorHandler(host)

	// Test Name
	assert.Equal(t, "InjectableDecoratorHandler", handler.Name())

	// Test Detect
	decs := host.GetDecoratorsOfDeclaration(classNode.AsNode())
	detected := handler.Detect(classNode, decs)
	assert.NotNil(t, detected)
	assert.Equal(t, "Injectable", detected.Name)

	// Test Analyze
	analysis, diags := handler.Analyze(classNode, detected)
	assert.Empty(t, diags)
	assert.NotNil(t, analysis)

	injectableAnalysis, ok := analysis.(*annotations.InjectableAnalysis)
	assert.True(t, ok)
	assert.NotNil(t, injectableAnalysis.ProvidedIn)
	wrappedExpr, isWrapped := injectableAnalysis.ProvidedIn.(*output.WrappedNodeExpr)
	assert.True(t, isWrapped)
	node, ok := wrappedExpr.Node.(*ast.Node)
	assert.True(t, ok)
	assert.Equal(t, ast.KindStringLiteral, node.Kind)
	assert.Equal(t, "root", node.AsStringLiteral().Text)
}
