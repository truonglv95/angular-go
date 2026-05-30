package imports_test

import (
	"fmt"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})
}

func TestEmitter_AbsoluteModuleStrategy(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	// Case 1: should not generate an import for a reference without owning module
	res := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name:     "/node_modules/external/index.d.ts",
			Contents: `export declare class Foo {}`,
		},
		{
			Name:     "/context.ts",
			Contents: `import {Foo} from 'external'; export class Context {}`,
		},
	})
	defer res.Release()

	sfExternal := ngtsctest.RequireSourceFile(t, res, "/node_modules/external/index.d.ts")
	declFoo := ngtsctest.FindNamedClassDeclaration(sfExternal, "Foo")
	require.NotNil(t, declFoo)

	contextSf := ngtsctest.RequireSourceFile(t, res, "/context.ts")

	resolver := imports.NewModuleResolver(res.Program)
	refHost := reflection.NewTypeScriptReflectionHost(res.Checker)
	strategy := imports.NewAbsoluteModuleStrategy(resolver, refHost)

	// Reference has no owning module guess
	ref := imports.NewReference(declFoo, nil)
	emitted := strategy.Emit(ref, contextSf, imports.None)
	assert.Nil(t, emitted)

	// Case 2: should generate an import using the exported name of the declaration
	refWithModule := imports.NewReference(declFoo, &imports.OwningModule{
		Specifier:         "external",
		ResolutionContext: contextSf.FileName(),
	})

	emitted2 := strategy.Emit(refWithModule, contextSf, imports.None)
	require.NotNil(t, emitted2)
	if emitted2.GetKind() == imports.Failed {
		t.Fatalf("Emission failed. Reason: %s", emitted2.(*imports.FailedEmitResult).Reason)
	}
	assert.Equal(t, imports.Success, emitted2.GetKind())

	emittedRef := emitted2.(*imports.EmittedReference)
	extExpr, ok := emittedRef.Expression.(*output.ExternalExpr)
	require.True(t, ok)
	assert.Equal(t, "Foo", *extExpr.Value.Name)
	assert.Equal(t, "external", *extExpr.Value.ModuleName)
}

func TestEmitter_LocalIdentifierStrategy(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	res := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name:     "/context.ts",
			Contents: `export class Foo {}`,
		},
	})
	defer res.Release()

	contextSf := ngtsctest.RequireSourceFile(t, res, "/context.ts")
	declFoo := ngtsctest.FindNamedClassDeclaration(contextSf, "Foo")
	require.NotNil(t, declFoo)

	strategy := imports.NewLocalIdentifierStrategy()
	ref := imports.NewReference(declFoo, nil)

	emitted := strategy.Emit(ref, contextSf, imports.None)
	require.NotNil(t, emitted)
	assert.Equal(t, imports.Success, emitted.GetKind())

	emittedRef := emitted.(*imports.EmittedReference)
	wrappedExpr, ok := emittedRef.Expression.(*output.WrappedNodeExpr)
	require.True(t, ok)
	assert.NotNil(t, wrappedExpr.Node)
}

func TestEmitter_LogicalProjectStrategy(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	res := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name:     "/app/index.ts",
			Contents: `export class Foo {}`,
		},
		{
			Name:     "/app/context.ts",
			Contents: `export class Context {}`,
		},
	})
	defer res.Release()

	sfIndex := ngtsctest.RequireSourceFile(t, res, "/app/index.ts")
	declFoo := ngtsctest.FindNamedClassDeclaration(sfIndex, "Foo")
	require.NotNil(t, declFoo)

	contextSf := ngtsctest.RequireSourceFile(t, res, "/app/context.ts")

	refHost := reflection.NewTypeScriptReflectionHost(res.Checker)
	logicalFs := file_system.NewLogicalFileSystem([]file_system.AbsoluteFsPath{file_system.AbsoluteFsPath("/")}, func(fileName string) string { return fileName })
	strategy := imports.NewLogicalProjectStrategy(refHost, logicalFs)

	ref := imports.NewReference(declFoo, nil)
	emitted := strategy.Emit(ref, contextSf, imports.None)
	require.NotNil(t, emitted)
	if emitted.GetKind() == imports.Failed {
		fmt.Printf("LogicalProjectStrategy Emit failed: %s\n", emitted.(*imports.FailedEmitResult).Reason)
	}
	assert.Equal(t, imports.Success, emitted.GetKind())

	emittedRef := emitted.(*imports.EmittedReference)
	extExpr, ok := emittedRef.Expression.(*output.ExternalExpr)
	require.True(t, ok)
	assert.Equal(t, "Foo", *extExpr.Value.Name)
	assert.Equal(t, "./index", *extExpr.Value.ModuleName)
}
