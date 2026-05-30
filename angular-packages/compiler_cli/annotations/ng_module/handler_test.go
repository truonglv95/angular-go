package ng_module_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/ng_module"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNgModuleDecoratorHandler_Analyze(t *testing.T) {
	sourceText := `
@NgModule({
	declarations: [TestComp],
	imports: [CommonModule],
	exports: [TestComp, CommonModule],
	bootstrap: [TestComp],
	providers: [TestService]
})
export class TestModule {}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/test.module.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)
	require.NotNil(t, sourceFile, "Failed to parse source file")

	var classNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) {
			classNode = stmt
			break
		}
	}
	require.NotNil(t, classNode, "Could not find class in AST")

	host := reflection.NewTypeScriptReflectionHost(nil)
	evaluator := partial_evaluator.NewPartialEvaluator(host, nil, nil)
	handler := ng_module.NewNgModuleDecoratorHandler(host, evaluator)

	data, err := handler.Analyze(classNode)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Since we are not running a full type checker program here, the unresolved identifiers
	// like TestComp, CommonModule resolve to DynamicValue or the identifier name as text.
	// We can inspect that our structure is correctly resolved.
	assert.Len(t, data.Declarations, 1)
	assert.Len(t, data.Imports, 1)
	assert.Len(t, data.Exports, 2)
	assert.Len(t, data.Bootstrap, 1)
	assert.NotNil(t, data.Providers)
}
