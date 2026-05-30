package pipe_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/pipe"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeDecoratorHandler_Analyze(t *testing.T) {
	sourceText := `
@Pipe({
	name: 'customTruncate',
	pure: false,
	standalone: true
})
export class TruncatePipe {}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/truncate.pipe.ts",
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
	handler := pipe.NewPipeDecoratorHandler(host, evaluator)

	data, err := handler.Analyze(classNode)
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Equal(t, "customTruncate", data.Name)
	assert.False(t, data.Pure)
	assert.True(t, data.Standalone)
}
