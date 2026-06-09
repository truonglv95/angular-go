package shims

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHost struct {
	files map[string]*ast.SourceFile
}

func (h *mockHost) GetSourceFile(fileName string, languageVersion any, onError func(message string), shouldCreateNewSourceFile bool) *ast.SourceFile {
	return h.files[fileName]
}

func (h *mockHost) FileExists(fileName string) bool {
	_, ok := h.files[fileName]
	return ok
}

type mockProgram struct {
	files []*ast.SourceFile
}

func (p *mockProgram) GetSourceFiles() []*ast.SourceFile {
	return p.files
}

type mockPerFileGenerator struct {
	extensionPrefix string
}

func (g *mockPerFileGenerator) ExtensionPrefix() string {
	return g.extensionPrefix
}

func (g *mockPerFileGenerator) ShouldEmit() bool {
	return false
}

func (g *mockPerFileGenerator) GenerateShimForFile(sf *ast.SourceFile, genFilePath string, priorShimSf *ast.SourceFile) *ast.SourceFile {
	if priorShimSf != nil {
		return priorShimSf
	}
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	return factory.NewSourceFile(ast.SourceFileParseOptions{FileName: genFilePath}, "export const SHIM_FOR_FILE = '"+sf.FileName()+"';", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()
}

func TestShimAdapter_RecognizeBasicShim(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	testSf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "export class A {}", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	host := &mockHost{
		files: map[string]*ast.SourceFile{
			"/test.ts": testSf,
		},
	}

	adapter := NewShimAdapter(host, nil, nil, []PerFileShimGenerator{&mockPerFileGenerator{extensionPrefix: "testshim"}}, nil)
	shimSf := adapter.MaybeGenerate("/test.testshim.ts")

	require.NotNil(t, shimSf)
	assert.Equal(t, "/test.testshim.ts", shimSf.FileName())
	assert.Contains(t, shimSf.Text(), "SHIM_FOR_FILE")
}

func TestShimAdapter_NotRecognizeNormalFile(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	testSf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "export class A {}", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	host := &mockHost{
		files: map[string]*ast.SourceFile{
			"/test.ts": testSf,
		},
	}

	adapter := NewShimAdapter(host, nil, nil, []PerFileShimGenerator{&mockPerFileGenerator{extensionPrefix: "testshim"}}, nil)
	shimSf := adapter.MaybeGenerate("/test.ts")

	assert.Nil(t, shimSf)
}

func TestShimAdapter_NotRecognizeShimWithoutSourceFile(t *testing.T) {
	host := &mockHost{
		files: map[string]*ast.SourceFile{},
	}

	adapter := NewShimAdapter(host, nil, nil, []PerFileShimGenerator{&mockPerFileGenerator{extensionPrefix: "testshim"}}, nil)
	shimSf := adapter.MaybeGenerate("/other.testshim.ts")

	assert.Nil(t, shimSf)
}

func TestShimAdapter_DetectPriorShim(t *testing.T) {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	testSf := factory.NewSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts"}, "export class A {}", factory.NewNodeList(nil), factory.NewToken(ast.KindEndOfFile)).AsSourceFile()

	host := &mockHost{
		files: map[string]*ast.SourceFile{
			"/test.ts": testSf,
		},
	}

	adapter1 := NewShimAdapter(host, nil, nil, []PerFileShimGenerator{&mockPerFileGenerator{extensionPrefix: "testshim"}}, nil)
	originalShim := adapter1.MaybeGenerate("/test.testshim.ts")
	require.NotNil(t, originalShim)

	oldProg := &mockProgram{
		files: []*ast.SourceFile{testSf, originalShim},
	}

	adapter2 := NewShimAdapter(host, nil, nil, []PerFileShimGenerator{&mockPerFileGenerator{extensionPrefix: "testshim"}}, oldProg)
	newShim := adapter2.MaybeGenerate("/test.testshim.ts")

	assert.Equal(t, originalShim, newShim)
}
