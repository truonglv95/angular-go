package imports_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/imports"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultImportTracker(t *testing.T) {
	expr := &ast.Node{}
	importDecl := &ast.Node{}

	// Verify initial state
	assert.Nil(t, imports.GetDefaultImportDeclaration(expr))

	// Attach and verify
	imports.AttachDefaultImportDeclaration(expr, importDecl)
	assert.Equal(t, importDecl, imports.GetDefaultImportDeclaration(expr))

	// Record used import
	tracker := imports.NewDefaultImportTracker()
	tracker.RecordUsedImport(nil) // shouldn't panic

	// Create a real SourceFile
	opts := ast.SourceFileParseOptions{
		FileName: "/test.ts",
	}
	sf := parser.ParseSourceFile(opts, "import Foo from './dep';", core.ScriptKindTS)
	require.NotNil(t, sf)

	var node *ast.Node
	for _, stmt := range sf.Statements.Nodes {
		if ast.IsImportDeclaration(stmt) {
			node = stmt
			break
		}
	}
	require.NotNil(t, node)

	tracker.RecordUsedImport(node)
}
