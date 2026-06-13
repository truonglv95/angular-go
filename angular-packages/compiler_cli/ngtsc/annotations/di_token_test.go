package annotations

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractDITokenFromForwardRefInjectDecorator(t *testing.T) {
	sourceText := `
class Target {
	constructor(@Inject(forwardRef(() => ForwardService)) srv: ForwardService) {}
}
class ForwardService {}
`
	sourceFile := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/entry.ts"}, sourceText, core.ScriptKindTS)
	require.NotNil(t, sourceFile)

	var classNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) && stmt.Name().AsIdentifier().Text == "Target" {
			classNode = stmt
			break
		}
	}
	require.NotNil(t, classNode)

	host := reflection.NewTypeScriptReflectionHost(nil)
	params := host.GetConstructorParameters(classNode)
	require.Len(t, params, 1)
	require.Len(t, params[0].Decorators, 1)
	require.Equal(t, "Inject", params[0].Decorators[0].Name)
	require.Len(t, params[0].Decorators[0].Args, 1)

	token := extractDITokenFromNode(params[0].Decorators[0].Args[0])
	forwardRefCall, ok := token.(*output.InvokeFunctionExpr)
	require.True(t, ok)

	fn, ok := forwardRefCall.Fn.(*output.ReadVarExpr)
	require.True(t, ok)
	assert.Equal(t, "forwardRef", fn.Name)
	require.Len(t, forwardRefCall.Args, 1)

	forwardRefFn, ok := forwardRefCall.Args[0].(*output.FunctionExpr)
	require.True(t, ok)
	require.Len(t, forwardRefFn.Statements, 1)

	ret, ok := forwardRefFn.Statements[0].(*output.ReturnStatement)
	require.True(t, ok)
	retValue, ok := ret.Value.(*output.ReadVarExpr)
	require.True(t, ok)
	assert.Equal(t, "ForwardService", retValue.Name)
}
