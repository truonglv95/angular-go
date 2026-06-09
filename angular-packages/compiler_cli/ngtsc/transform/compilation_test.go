package transform

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failDecoratorHandler struct {
	t *testing.T
}

func (h *failDecoratorHandler) Name() string {
	return "FailDecoratorHandler"
}

func (h *failDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	h.t.Fatal("Detect should not have been called")
	return nil
}

func (h *failDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	h.t.Fatal("Analyze should not have been called")
	return nil, nil
}

func (h *failDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysis any) (any, []ast.Diagnostic) {
	h.t.Fatal("Resolve should not have been called")
	return nil, nil
}

func (h *failDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysis any, resolution any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]CompileResult, []ast.Diagnostic) {
	h.t.Fatal("CompileFull should not have been called")
	return nil, nil
}

func TestTraitCompiler_DeclarationFiles(t *testing.T) {
	// Decorator handlers should not run against declaration files (.d.ts).
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/test.d.ts",
			Contents: `
			export declare function MyDecorator(target: any): void;
			@MyDecorator
			export declare class SomeDirective {}
			`,
		},
	})
	defer result.Release()

	sf := ngtsctest.RequireSourceFile(t, result, "/test.d.ts")
	require.True(t, sf.IsDeclarationFile)

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	localHost := reflection.NewTypeScriptReflectionHost(nil)
	tc := NewTraitCompiler([]DecoratorHandler{&failDecoratorHandler{t: t}}, host, localHost, nil)

	tc.AnalyzeSync(sf)

	assert.Empty(t, tc.GetClasses())
}

type spyDecoratorHandler struct {
	t             *testing.T
	detected      bool
	analyzed      bool
	resolved      bool
	compiled      bool
	compResultVal string
}

func (h *spyDecoratorHandler) Name() string {
	return "SpyDecoratorHandler"
}

func (h *spyDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	if node.Name() != nil && node.Name().AsIdentifier().Text == "Cmp" {
		h.detected = true
		return &decorators[0]
	}
	return nil
}

func (h *spyDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	h.analyzed = true
	return "analysis-data", nil
}

func (h *spyDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysis any) (any, []ast.Diagnostic) {
	h.resolved = true
	return "resolution-data", nil
}

func (h *spyDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysis any, resolution any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]CompileResult, []ast.Diagnostic) {
	h.compiled = true
	initializer := factory.NewToken(ast.KindTrueKeyword)
	return []CompileResult{
		{
			PropertyName: "ɵcmp",
			Initializer:  initializer,
		},
	}, nil
}

func TestTraitCompiler_CompilationLifecycle(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/test.ts",
			Contents: `
			export function MyDecorator(target: any) {}
			@MyDecorator
			export class Cmp {}
			`,
		},
	})
	defer result.Release()

	sf := ngtsctest.RequireSourceFile(t, result, "/test.ts")
	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	localHost := reflection.NewTypeScriptReflectionHost(nil)
	handler := &spyDecoratorHandler{t: t}
	tc := NewTraitCompiler([]DecoratorHandler{handler}, host, localHost, nil)

	// 1. Analyze
	tc.AnalyzeSync(sf)
	assert.True(t, handler.detected)
	assert.True(t, handler.analyzed)

	classes := tc.GetClasses()
	require.Len(t, classes, 1)

	var traits []*Trait
	for _, trs := range classes {
		traits = trs
	}
	require.Len(t, traits, 1)
	assert.Equal(t, TraitStateAnalyzed, traits[0].State)

	// 2. Resolve
	tc.Resolve()
	assert.True(t, handler.resolved)
	assert.Equal(t, TraitStateResolved, traits[0].State)

	// 3. Compile/UpdateSourceFile
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	tc.UpdateSourceFile(sf, factory)
	assert.True(t, handler.compiled)

	// Verify the static property is injected
	classDecl := ngtsctest.FindNamedClassDeclaration(sf, "Cmp").AsClassDeclaration()
	require.NotNil(t, classDecl)

	var foundCmp bool
	for _, member := range classDecl.Members.Nodes {
		if ast.IsPropertyDeclaration(member) {
			prop := member.AsPropertyDeclaration()
			if prop.Name().AsIdentifier().Text == "ɵcmp" {
				foundCmp = true
				break
			}
		}
	}
	assert.True(t, foundCmp, "Static property ɵcmp was not injected into class Cmp")
}

func TestTraitCompiler_NonExportedClasses(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/test.ts",
			Contents: `
			export function MyDecorator(target: any) {}
			@MyDecorator
			class Cmp {} // Not exported!
			`,
		},
	})
	defer result.Release()

	sf := ngtsctest.RequireSourceFile(t, result, "/test.ts")
	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	localHost := reflection.NewTypeScriptReflectionHost(nil)
	handler := &spyDecoratorHandler{t: t}
	tc := NewTraitCompiler([]DecoratorHandler{handler}, host, localHost, nil)

	tc.AnalyzeSync(sf)
	assert.True(t, handler.detected)
	assert.True(t, handler.analyzed)

	classes := tc.GetClasses()
	require.Len(t, classes, 1)
}
