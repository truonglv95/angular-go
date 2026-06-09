package imports

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportManager_AddImportSymbol(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	manager := NewImportManager(nil, factory)

	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "", core.ScriptKindTS)
	require.NotNil(t, sf)
	sfNode := sf.AsNode()

	symbolName := "input"
	ref := manager.AddImport(ImportRequest{
		ExportModuleSpecifier: "@angular/core",
		ExportSymbolName:      &symbolName,
		RequestedFile:         sfNode,
	})

	require.NotNil(t, ref)
	assert.Equal(t, ast.KindIdentifier, ref.Kind)
	assert.Equal(t, "input", ref.AsIdentifier().Text)

	declarations := manager.GetAllImports(sfNode)
	require.Len(t, declarations, 1)

	decl := declarations[0]
	assert.Equal(t, ast.KindImportDeclaration, decl.Kind)

	importDecl := decl.AsImportDeclaration()
	require.NotNil(t, importDecl.ImportClause)

	clause := importDecl.ImportClause.AsImportClause()
	require.NotNil(t, clause.NamedBindings)
	assert.Equal(t, ast.KindNamedImports, clause.NamedBindings.Kind)

	namedImports := clause.NamedBindings.AsNamedImports()
	require.Len(t, namedImports.Elements.Nodes, 1)

	specifier := namedImports.Elements.Nodes[0].AsImportSpecifier()
	assert.Equal(t, "input", specifier.Name().AsIdentifier().Text)
	assert.Nil(t, specifier.PropertyName)
}

func TestImportManager_AddImportNamespace(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	manager := NewImportManager(nil, factory)

	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "", core.ScriptKindTS)
	require.NotNil(t, sf)
	sfNode := sf.AsNode()

	ref := manager.AddImport(ImportRequest{
		ExportModuleSpecifier: "@angular/core",
		ExportSymbolName:      nil,
		RequestedFile:         sfNode,
	})

	require.NotNil(t, ref)
	assert.Equal(t, ast.KindIdentifier, ref.Kind)
	assert.Equal(t, "i0", ref.AsIdentifier().Text)

	declarations := manager.GetAllImports(sfNode)
	require.Len(t, declarations, 1)

	decl := declarations[0]
	assert.Equal(t, ast.KindImportDeclaration, decl.Kind)

	importDecl := decl.AsImportDeclaration()
	require.NotNil(t, importDecl.ImportClause)

	clause := importDecl.ImportClause.AsImportClause()
	require.NotNil(t, clause.NamedBindings)
	assert.Equal(t, ast.KindNamespaceImport, clause.NamedBindings.Kind)

	nsImport := clause.NamedBindings.AsNamespaceImport()
	assert.Equal(t, "i0", nsImport.Name().AsIdentifier().Text)
}

func TestImportManager_ForceGenerateNamespaces(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	config := PresetImportManagerForceNamespaceImports
	manager := NewImportManager(&config, factory)

	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "", core.ScriptKindTS)
	require.NotNil(t, sf)
	sfNode := sf.AsNode()

	symbolName := "input"
	ref := manager.AddImport(ImportRequest{
		ExportModuleSpecifier: "@angular/core",
		ExportSymbolName:      &symbolName,
		RequestedFile:         sfNode,
	})

	require.NotNil(t, ref)
	assert.Equal(t, ast.KindPropertyAccessExpression, ref.Kind)

	pae := ref.AsPropertyAccessExpression()
	assert.Equal(t, "i0", pae.Expression.AsIdentifier().Text)
	assert.Equal(t, "input", pae.Name().AsIdentifier().Text)

	declarations := manager.GetAllImports(sfNode)
	require.Len(t, declarations, 1)

	decl := declarations[0]
	importDecl := decl.AsImportDeclaration()
	clause := importDecl.ImportClause.AsImportClause()
	assert.Equal(t, ast.KindNamespaceImport, clause.NamedBindings.Kind)
}

func TestImportManager_AddSideEffectImport(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	manager := NewImportManager(nil, factory)

	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "", core.ScriptKindTS)
	require.NotNil(t, sf)
	sfNode := sf.AsNode()

	manager.AddSideEffectImport(sfNode, "@angular/core")

	declarations := manager.GetAllImports(sfNode)
	require.Len(t, declarations, 1)

	decl := declarations[0]
	importDecl := decl.AsImportDeclaration()
	assert.Nil(t, importDecl.ImportClause)
	assert.Equal(t, "@angular/core", importDecl.ModuleSpecifier.AsStringLiteral().Text)
}

func TestImportManager_RemoveImport(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	manager := NewImportManager(nil, factory)

	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "", core.ScriptKindTS)
	require.NotNil(t, sf)
	sfNode := sf.AsNode()

	assert.False(t, manager.IsRemoved(sfNode, "input", "@angular/core"))

	manager.RemoveImport(sfNode, "input", "@angular/core")
	assert.True(t, manager.IsRemoved(sfNode, "input", "@angular/core"))
}
