package shims

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShimReferenceTagger_TagSourceFile(t *testing.T) {
	tagger := NewShimReferenceTagger([]string{"test1", "test2"})

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_tag.ts"}, "export const x = 1;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	assert.Len(t, sf.ReferencedFiles, 0)

	tagger.Tag(sf)
	require.Len(t, sf.ReferencedFiles, 2)
	assert.Equal(t, "/file_tag.test1.ts", sf.ReferencedFiles[0].FileName)
	assert.Equal(t, "/file_tag.test2.ts", sf.ReferencedFiles[1].FileName)
}

func TestShimReferenceTagger_DoNotTagDeclarationFiles(t *testing.T) {
	tagger := NewShimReferenceTagger([]string{"test1", "test2"})

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_decl.d.ts"}, "export declare const x: number;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()
	sf.IsDeclarationFile = true

	tagger.Tag(sf)
	assert.Len(t, sf.ReferencedFiles, 0)
}

func TestShimReferenceTagger_DoNotTagJsFiles(t *testing.T) {
	tagger := NewShimReferenceTagger([]string{"test1", "test2"})

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_js.js"}, "const x = 1;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	tagger.Tag(sf)
	assert.Len(t, sf.ReferencedFiles, 0)
}

func TestShimReferenceTagger_DoNotTagShimFiles(t *testing.T) {
	tagger := NewShimReferenceTagger([]string{"test1", "test2"})

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_shim.test1.ts"}, "export const x = 1;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	SetIsShim(sf, true)

	tagger.Tag(sf)
	assert.Len(t, sf.ReferencedFiles, 0)
}

func TestShimReferenceTagger_DoNotTagAfterFinalize(t *testing.T) {
	tagger := NewShimReferenceTagger([]string{"test1", "test2"})
	tagger.Finalize()

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_finalize.ts"}, "export const x = 1;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	tagger.Tag(sf)
	assert.Len(t, sf.ReferencedFiles, 0)
}

func TestShimReferenceTagger_DoNotOverwriteOriginal(t *testing.T) {
	tagger := NewShimReferenceTagger([]string{"test"})

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_overwrite.ts"}, "export const x = 1;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()
	sf.ReferencedFiles = []*ast.FileReference{
		{FileName: "/other.ts"},
	}

	tagger.Tag(sf)
	require.Len(t, sf.ReferencedFiles, 2)
	assert.Equal(t, "/other.ts", sf.ReferencedFiles[0].FileName)
	assert.Equal(t, "/file_overwrite.test.ts", sf.ReferencedFiles[1].FileName)
}

func TestShimReferenceTagger_AlwaysTagOriginal(t *testing.T) {
	tagger1 := NewShimReferenceTagger([]string{"test1"})
	tagger2 := NewShimReferenceTagger([]string{"test2"})

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_always.ts"}, "export const x = 1;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	tagger1.Tag(sf)
	tagger2.Tag(sf)

	require.Len(t, sf.ReferencedFiles, 1)
	assert.Equal(t, "/file_always.test2.ts", sf.ReferencedFiles[0].FileName)
}

func TestShimReferenceTagger_UntagAndRetag(t *testing.T) {
	tagger := NewShimReferenceTagger([]string{"test"})

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	sf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/file_untag.ts"}, "export const x = 1;", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()
	sf.ReferencedFiles = []*ast.FileReference{
		{FileName: "/other.ts"},
	}

	tagger.Tag(sf)
	require.Len(t, sf.ReferencedFiles, 2)
	assert.Equal(t, "/other.ts", sf.ReferencedFiles[0].FileName)
	assert.Equal(t, "/file_untag.test.ts", sf.ReferencedFiles[1].FileName)

	UntagTsFile(sf)
	require.Len(t, sf.ReferencedFiles, 1)
	assert.Equal(t, "/other.ts", sf.ReferencedFiles[0].FileName)
}
