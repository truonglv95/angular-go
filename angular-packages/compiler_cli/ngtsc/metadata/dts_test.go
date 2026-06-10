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
  static ɵcmp: i0.ɵɵComponentDeclaration<MyComp, "my-comp-selector", ["cmp"], {foo: "fooAlias"}, {closed: "closedAlias"}, never, never, true>;
}
export declare class MyDir {
  static ɵdir: i0.ɵɵDirectiveDeclaration<MyDir, "my-dir-selector", ["dir"], {bar: {alias: "barAlias", required: true}}, {saved: "savedAlias"}, never, never, false>;
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
	assert.Equal(t, []string{"cmp"}, compMeta.ExportAs)
	assert.Equal(t, map[string]string{"foo": "fooAlias"}, compMeta.Inputs)
	assert.Equal(t, map[string]string{"closed": "closedAlias"}, compMeta.Outputs)

	// 2. Verify Directive Metadata
	dirMeta := reader.GetDirectiveMetadata(dirNode)
	assert.NotNil(t, dirMeta)
	assert.Equal(t, "my-dir-selector", dirMeta.Selector)
	assert.False(t, dirMeta.IsComponent)
	assert.False(t, dirMeta.Standalone)
	assert.Equal(t, metadata.MetaKindDirective, dirMeta.Kind)
	assert.Equal(t, []string{"dir"}, dirMeta.ExportAs)
	assert.Equal(t, map[string]string{"bar": "barAlias"}, dirMeta.Inputs)
	assert.Equal(t, map[string]string{"saved": "savedAlias"}, dirMeta.Outputs)

	compSymbol := reader.GetSemanticSymbol(compNode)
	assert.NotNil(t, compSymbol)
	assert.Equal(t, "/lib.d.ts", compSymbol.Path)
	assert.Equal(t, "MyComp", compSymbol.Identifier)
	assert.Equal(t, "component", compSymbol.Kind)
	assert.Equal(t, compMeta.Inputs, compSymbol.Inputs)
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

func TestDtsMetadataReader_NgModuleImportAliases(t *testing.T) {
	dtsSource := `
import DefaultModule from './default';
import { RealDir as AliasDir, RealPipe } from './shared';
import * as shared from './namespace';

export declare class MyModule {
  static ɵmod: i0.ɵɵNgModuleDeclaration<MyModule, [typeof AliasDir], [typeof DefaultModule, typeof shared.NamespaceModule], [typeof RealPipe, typeof shared.NamespaceDir]>;
}
`
	sourceFile := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/alias-module.d.ts"}, dtsSource, core.ScriptKindTS)
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
	assert.NotNil(t, moduleNode)

	reader := metadata.NewDtsMetadataReader(nil)
	moduleMeta := reader.GetNgModuleMetadata(moduleNode)
	assert.NotNil(t, moduleMeta)

	assert.Len(t, moduleMeta.Declarations, 1)
	assert.Equal(t, "RealDir", moduleMeta.Declarations[0].Name)
	assert.Equal(t, "./shared", moduleMeta.Declarations[0].OwningModule)

	assert.Len(t, moduleMeta.Imports, 2)
	assert.Equal(t, "default", moduleMeta.Imports[0].Name)
	assert.Equal(t, "./default", moduleMeta.Imports[0].OwningModule)
	assert.Equal(t, "NamespaceModule", moduleMeta.Imports[1].Name)
	assert.Equal(t, "./namespace", moduleMeta.Imports[1].OwningModule)

	assert.Len(t, moduleMeta.Exports, 2)
	assert.Equal(t, "RealPipe", moduleMeta.Exports[0].Name)
	assert.Equal(t, "./shared", moduleMeta.Exports[0].OwningModule)
	assert.Equal(t, "NamespaceDir", moduleMeta.Exports[1].Name)
	assert.Equal(t, "./namespace", moduleMeta.Exports[1].OwningModule)
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

func TestDtsMetadataReader_MalformedAST_PrivateIdentifierName(t *testing.T) {
	// A class with a private identifier name (#PrivatePipe) is invalid TS,
	// but we test that our metadata reader does not panic if such a node occurs.
	dtsSource := `
export declare class #PrivatePipe {
  static ɵpipe: i0.ɵɵPipeDeclaration<any, "my-pipe-name", false, true>;
}
`
	opts := ast.SourceFileParseOptions{
		FileName: "/malformed.d.ts",
	}

	sourceFile := parser.ParseSourceFile(opts, dtsSource, core.ScriptKindTS)
	if sourceFile == nil {
		t.Skip("Parser did not produce a file")
	}

	var pipeNode *ast.Node
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind == ast.KindClassDeclaration {
			pipeNode = stmt
			break
		}
	}

	if pipeNode != nil {
		reader := metadata.NewDtsMetadataReader(nil)
		// This should not panic!
		pipeMeta := reader.GetPipeMetadata(pipeNode)
		_ = pipeMeta
	}
}
