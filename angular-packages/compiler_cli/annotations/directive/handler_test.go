package directive_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/directive"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectiveDecoratorHandler_Analyze(t *testing.T) {
	sourceText := `
@Directive({
	selector: '[app-highlight]',
	standalone: true,
	inputs: ['color', 'defaultColor'],
	outputs: ['hover'],
	host: {
		'(mouseenter)': 'onMouseEnter()',
		'(mouseleave)': 'onMouseLeave()'
	},
	exportAs: 'appHighlight'
})
export class HighlightDirective {}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/highlight.directive.ts",
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
	handler := directive.NewDirectiveDecoratorHandler(host, evaluator)

	data, err := handler.Analyze(classNode)
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Equal(t, "[app-highlight]", data.Selector)
	assert.True(t, data.Standalone)
	assert.Equal(t, map[string]string{"color": "color", "defaultColor": "defaultColor"}, data.Inputs)
	assert.Equal(t, map[string]string{"hover": "hover"}, data.Outputs)
	assert.Equal(t, map[string]string{
		"(mouseenter)": "onMouseEnter()",
		"(mouseleave)": "onMouseLeave()",
	}, data.HostBindings)
	assert.Equal(t, []string{"appHighlight"}, data.ExportAs)
}
