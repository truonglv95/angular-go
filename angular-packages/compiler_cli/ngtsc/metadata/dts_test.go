package metadata_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
)

func TestDtsMetadataReader_ComponentAndDirective(t *testing.T) {
	dtsSource := `
export declare class MyComp {
  static ɵcmp: i0.ɵɵComponentDeclaration<MyComp, "my-comp-selector", never, {}, {}, never, never, true>;
}
export declare class MyDir {
  static ɵdir: i0.ɵɵDirectiveDeclaration<MyDir, "my-dir-selector", never, {}, {}, never, never, false>;
}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/lib.d.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, dtsSource, core.ScriptKindTS)
	if sourceFile == nil {
		t.Fatal("Failed to parse .d.ts source file")
	}

	var compNode, dirNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind == ast.KindClassDeclaration {
			name := stmt.AsClassDeclaration().Name().AsIdentifier().Text
			if name == "MyComp" {
				compNode = stmt
			} else if name == "MyDir" {
				dirNode = stmt
			}
		}
	}

	assert.NotNil(t, compNode, "Could not find MyComp class")
	assert.NotNil(t, dirNode, "Could not find MyDir class")

	reader := metadata.NewDtsMetadataReader(nil)

	// 1. Verify Component Metadata
	compMeta := reader.GetDirectiveMetadata(compNode)
	assert.NotNil(t, compMeta)
	assert.Equal(t, "my-comp-selector", compMeta.Selector)
	assert.True(t, compMeta.IsComponent)
	assert.True(t, compMeta.Standalone)
	assert.Equal(t, metadata.MetaKindComponent, compMeta.Kind)

	// 2. Verify Directive Metadata
	dirMeta := reader.GetDirectiveMetadata(dirNode)
	assert.NotNil(t, dirMeta)
	assert.Equal(t, "my-dir-selector", dirMeta.Selector)
	assert.False(t, dirMeta.IsComponent)
	assert.False(t, dirMeta.Standalone)
	assert.Equal(t, metadata.MetaKindDirective, dirMeta.Kind)
}

func TestDtsMetadataReader_Pipe(t *testing.T) {
	dtsSource := `
export declare class MyPipe {
  static ɵpipe: i0.ɵɵPipeDeclaration<MyPipe, "my-pipe-name", false, true>;
}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/pipe.d.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, dtsSource, core.ScriptKindTS)
	if sourceFile == nil {
		t.Fatal("Failed to parse .d.ts source file")
	}

	var pipeNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind == ast.KindClassDeclaration {
			pipeNode = stmt
			break
		}
	}

	assert.NotNil(t, pipeNode, "Could not find MyPipe class")

	reader := metadata.NewDtsMetadataReader(nil)
	pipeMeta := reader.GetPipeMetadata(pipeNode)
	assert.NotNil(t, pipeMeta)
	assert.Equal(t, "my-pipe-name", pipeMeta.Name)
	assert.False(t, pipeMeta.Pure)
	assert.True(t, pipeMeta.Standalone)
}

func TestDtsMetadataReader_NgModule(t *testing.T) {
	dtsSource := `
export declare class MyModule {
  static ɵmod: i0.ɵɵNgModuleDeclaration<MyModule, [typeof MyComp], [typeof OtherModule], [typeof MyComp]>;
}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/module.d.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, dtsSource, core.ScriptKindTS)
	if sourceFile == nil {
		t.Fatal("Failed to parse .d.ts source file")
	}

	var moduleNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind == ast.KindClassDeclaration {
			moduleNode = stmt
			break
		}
	}

	assert.NotNil(t, moduleNode, "Could not find MyModule class")

	reader := metadata.NewDtsMetadataReader(nil)
	moduleMeta := reader.GetNgModuleMetadata(moduleNode)
	assert.NotNil(t, moduleMeta)

	// Verify declarations, imports, exports lists are parsed
	assert.Len(t, moduleMeta.Declarations, 1)
	assert.Len(t, moduleMeta.Imports, 1)
	assert.Len(t, moduleMeta.Exports, 1)

	declExprName := moduleMeta.Declarations[0].Node
	assert.Equal(t, ast.KindIdentifier, declExprName.Kind)
	assert.Equal(t, "MyComp", declExprName.AsIdentifier().Text)
}

func TestCompoundMetadataReader(t *testing.T) {
	localRegistry := metadata.NewLocalMetadataRegistry()
	dtsReader := metadata.NewDtsMetadataReader(nil)
	compound := metadata.NewCompoundMetadataReader([]metadata.MetadataReader{localRegistry, dtsReader})

	localNode := &ast.Node{}
	localDirMeta := &metadata.DirectiveMeta{
		Selector: "local-selector",
	}
	localRegistry.RegisterDirective(localNode, localDirMeta)

	// Test querying from local registry
	meta := compound.GetDirectiveMetadata(localNode)
	assert.Equal(t, localDirMeta, meta)

	// Test querying from dts reader with a dts Component class node
	dtsSource := `
export declare class MyComp {
  static ɵcmp: i0.ɵɵComponentDeclaration<MyComp, "dts-selector", never, {}, {}, never, never, true>;
}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/compound.d.ts",
	}
	sourceFile := parser.ParseSourceFile(opts, dtsSource, core.ScriptKindTS)
	var dtsNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind == ast.KindClassDeclaration {
			dtsNode = stmt
			break
		}
	}

	meta2 := compound.GetDirectiveMetadata(dtsNode)
	assert.NotNil(t, meta2)
	assert.Equal(t, "dts-selector", meta2.Selector)
}
