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

func TestDirectiveDecoratorHandler_DetectAndAnalyze(t *testing.T) {
	sourceText := `
@Directive({
	selector: '[appHighlight]',
	standalone: true
})
export class HighlightDirective {}
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
	handler := annotations.NewDirectiveDecoratorHandler(host, metaRegistry)

	// Test Name
	assert.Equal(t, "DirectiveDecoratorHandler", handler.Name())

	// Test Detect
	decs := host.GetDecoratorsOfDeclaration(classNode.AsNode())
	detected := handler.Detect(classNode, decs)
	assert.NotNil(t, detected)
	assert.Equal(t, "Directive", detected.Name)

	// Test Analyze
	analysis, diags := handler.Analyze(classNode, detected)
	assert.Empty(t, diags)
	assert.NotNil(t, analysis)

	dirAnalysis, ok := analysis.(*annotations.DirectiveAnalysis)
	assert.True(t, ok)
	assert.Equal(t, "[appHighlight]", dirAnalysis.Selector)
	assert.True(t, dirAnalysis.IsStandalone)
}

func TestDirectiveDecoratorHandler_AnalyzeExtends(t *testing.T) {
	sourceText := `
class BaseDirective {
	@Input() baseProp = 'base';
}

@Directive({
	selector: '[appHighlight]',
	standalone: true
})
export class HighlightDirective extends BaseDirective {
	@Input() childProp = 'child';
}
`
	sourceFile := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/entry.ts"}, sourceText, core.ScriptKindTS)
	var classNode *ast.ClassDeclaration
	for _, stmt := range sourceFile.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) && stmt.AsClassDeclaration().Name() != nil && stmt.AsClassDeclaration().Name().AsIdentifier().Text == "HighlightDirective" {
			classNode = stmt.AsClassDeclaration()
			break
		}
	}
	if classNode == nil {
		t.Fatal("Could not find class in AST")
	}

	host := reflection.NewTypeScriptReflectionHost(nil)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	handler := annotations.NewDirectiveDecoratorHandler(host, metaRegistry)

	decs := host.GetDecoratorsOfDeclaration(classNode.AsNode())
	detected := handler.Detect(classNode, decs)
	
	analysis, _ := handler.Analyze(classNode, detected)
	dirAnalysis := analysis.(*annotations.DirectiveAnalysis)
	
	if _, ok := dirAnalysis.Inputs["childProp"]; !ok {
		t.Error("Missing childProp")
	}
	if _, ok := dirAnalysis.Inputs["baseProp"]; !ok {
		t.Error("Missing baseProp")
	}
}
