package annotations_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	ngdiagnostics "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestNgModuleDecoratorHandler_DetectAndAnalyze(t *testing.T) {
	sourceText := `
@NgModule({
	declarations: [MyComp],
	imports: [CommonModule],
	exports: [MyComp]
})
export class MyModule {}
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
	handler := annotations.NewNgModuleDecoratorHandler(host, metaRegistry, scopeRegistry)

	// Test Name
	assert.Equal(t, "NgModuleDecoratorHandler", handler.Name())

	// Test Detect
	decs := host.GetDecoratorsOfDeclaration(classNode.AsNode())
	detected := handler.Detect(classNode, decs)
	assert.NotNil(t, detected)
	assert.Equal(t, "NgModule", detected.Name)

	// Test Analyze
	analysis, diags := handler.Analyze(classNode, detected)
	assert.Empty(t, diags)
	assert.NotNil(t, analysis)

	moduleAnalysis, ok := analysis.(*annotations.NgModuleAnalysis)
	assert.True(t, ok)
	assert.Len(t, moduleAnalysis.Declarations, 1)
	assert.Len(t, moduleAnalysis.Imports, 1)
	assert.Len(t, moduleAnalysis.Exports, 1)
}

func TestNgModuleDecoratorHandler_ResolveDiagnostics(t *testing.T) {
	// Case 1: Duplicate declarations
	sourceTextDuplicate := `
export class MyComp {}
@NgModule({
	declarations: [MyComp, MyComp],
})
export class MyModule {}
`
	sourceFile := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/entry.ts"}, sourceTextDuplicate, core.ScriptKindTS)
	var myModuleClass, myCompClass *ast.ClassDeclaration
	for _, stmt := range sourceFile.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) {
			decl := stmt.AsClassDeclaration()
			if decl.Name().AsIdentifier().Text == "MyModule" {
				myModuleClass = decl
			} else if decl.Name().AsIdentifier().Text == "MyComp" {
				myCompClass = decl
			}
		}
	}

	host := reflection.NewTypeScriptReflectionHost(nil)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)
	handler := annotations.NewNgModuleDecoratorHandler(host, metaRegistry, scopeRegistry)

	// Register MyComp directive
	metaRegistry.RegisterDirective(myCompClass.AsNode(), &metadata.DirectiveMeta{
		Name: "MyComp",
		Ref:  metadata.Reference{Node: myCompClass.AsNode(), Name: "MyComp"},
	})

	decs := host.GetDecoratorsOfDeclaration(myModuleClass.AsNode())
	detected := handler.Detect(myModuleClass, decs)
	analysis, _ := handler.Analyze(myModuleClass, detected)

	_, diags := handler.Resolve(myModuleClass, analysis)
	assert.Len(t, diags, 1)
	assert.Equal(t, int32(ngdiagnostics.NgErrorCode(ngdiagnostics.ErrorCode_NGMODULE_INVALID_DECLARATION)), diags[0].Code())
	assert.Contains(t, diags[0].MessageArgs()[0], "declared multiple times")

	// Case 2: Standalone declaration in NgModule declarations
	sourceTextStandalone := `
export class StandaloneComp {}
@NgModule({
	declarations: [StandaloneComp],
})
export class MyModule {}
`
	sourceFile2 := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/entry2.ts"}, sourceTextStandalone, core.ScriptKindTS)
	var myModuleClass2, standaloneClass *ast.ClassDeclaration
	for _, stmt := range sourceFile2.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) {
			decl := stmt.AsClassDeclaration()
			if decl.Name().AsIdentifier().Text == "MyModule" {
				myModuleClass2 = decl
			} else if decl.Name().AsIdentifier().Text == "StandaloneComp" {
				standaloneClass = decl
			}
		}
	}

	metaRegistry2 := metadata.NewLocalMetadataRegistry()
	scopeRegistry2 := scope.NewLocalModuleScopeRegistry(metaRegistry2)
	handler2 := annotations.NewNgModuleDecoratorHandler(host, metaRegistry2, scopeRegistry2)

	metaRegistry2.RegisterDirective(standaloneClass.AsNode(), &metadata.DirectiveMeta{
		Name:       "StandaloneComp",
		Ref:        metadata.Reference{Node: standaloneClass.AsNode(), Name: "StandaloneComp"},
		Standalone: true,
	})

	decs2 := host.GetDecoratorsOfDeclaration(myModuleClass2.AsNode())
	detected2 := handler2.Detect(myModuleClass2, decs2)
	analysis2, _ := handler2.Analyze(myModuleClass2, detected2)

	_, diags2 := handler2.Resolve(myModuleClass2, analysis2)
	assert.Len(t, diags2, 1)
	assert.Equal(t, int32(ngdiagnostics.NgErrorCode(ngdiagnostics.ErrorCode_NGMODULE_DECLARATION_IS_STANDALONE)), diags2[0].Code())
	assert.Contains(t, diags2[0].MessageArgs()[0], "is standalone and cannot be declared")

	// Case 3: Invalid imports (non-standalone component imported directly)
	sourceTextClassicImport := `
export class ClassicComp {}
@NgModule({
	imports: [ClassicComp],
})
export class MyModule {}
`
	sourceFile3 := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/entry3.ts"}, sourceTextClassicImport, core.ScriptKindTS)
	var myModuleClass3, classicClass *ast.ClassDeclaration
	for _, stmt := range sourceFile3.Statements.Nodes {
		if ast.IsClassDeclaration(stmt) {
			decl := stmt.AsClassDeclaration()
			if decl.Name().AsIdentifier().Text == "MyModule" {
				myModuleClass3 = decl
			} else if decl.Name().AsIdentifier().Text == "ClassicComp" {
				classicClass = decl
			}
		}
	}

	metaRegistry3 := metadata.NewLocalMetadataRegistry()
	scopeRegistry3 := scope.NewLocalModuleScopeRegistry(metaRegistry3)
	handler3 := annotations.NewNgModuleDecoratorHandler(host, metaRegistry3, scopeRegistry3)

	metaRegistry3.RegisterDirective(classicClass.AsNode(), &metadata.DirectiveMeta{
		Name:       "ClassicComp",
		Ref:        metadata.Reference{Node: classicClass.AsNode(), Name: "ClassicComp"},
		Standalone: false,
	})

	decs3 := host.GetDecoratorsOfDeclaration(myModuleClass3.AsNode())
	detected3 := handler3.Detect(myModuleClass3, decs3)
	analysis3, _ := handler3.Analyze(myModuleClass3, detected3)

	_, diags3 := handler3.Resolve(myModuleClass3, analysis3)
	assert.Len(t, diags3, 1)
	assert.Equal(t, int32(ngdiagnostics.NgErrorCode(ngdiagnostics.ErrorCode_NGMODULE_INVALID_IMPORT)), diags3[0].Code())
	assert.Contains(t, diags3[0].MessageArgs()[0], "is not standalone and cannot be imported directly")
}
