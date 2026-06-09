package annotations_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestPipeDecoratorHandler_DetectAndAnalyze(t *testing.T) {
	sourceText := `
@Pipe({
	name: 'truncate',
	standalone: true
})
export class TruncatePipe {}
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
	handler := annotations.NewPipeDecoratorHandler(host, metaRegistry)

	// Test Name
	assert.Equal(t, "PipeDecoratorHandler", handler.Name())

	// Test Detect
	decs := host.GetDecoratorsOfDeclaration(classNode.AsNode())
	detected := handler.Detect(classNode, decs)
	assert.NotNil(t, detected)
	assert.Equal(t, "Pipe", detected.Name)

	// Test Analyze
	analysis, diags := handler.Analyze(classNode, detected)
	assert.Empty(t, diags)
	assert.NotNil(t, analysis)

	pipeAnalysis, ok := analysis.(*annotations.PipeAnalysis)
	assert.True(t, ok)
	assert.Equal(t, "truncate", pipeAnalysis.Name)
	assert.True(t, pipeAnalysis.IsStandalone)
}
