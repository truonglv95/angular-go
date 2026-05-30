package reflection_test

import (
	"sort"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findFirstImportDeclaration walks the AST of a source file to find the first import declaration.
func findFirstImportDeclaration(sf *ast.SourceFile) *ast.Node {
	var found *ast.Node
	var visit func(node *ast.Node) bool
	visit = func(node *ast.Node) bool {
		if found != nil {
			return true
		}
		if ast.IsImportDeclaration(node) {
			found = node
			return true
		}
		return node.ForEachChild(visit)
	}
	sf.AsNode().ForEachChild(visit)
	return found
}

// ─── getConstructorParameters ───────────────────────────────────────────────

func TestGetConstructorParameters_SingleArgument(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			class Bar {}

			class Foo {
			  constructor(bar: Bar) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 1)
	expectParameter(t, args[0], "bar", "Bar", "", "")
}

func TestGetConstructorParameters_DecoratedArgument(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/dec.ts",
			Contents: `
			export function dec(target: any, key: string, index: number) {}
			`,
		},
		{
			Name: "/entry.ts",
			Contents: `
			import {dec} from './dec';
			class Bar {}

			class Foo {
			  constructor(@dec bar: Bar) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 1)
	expectParameter(t, args[0], "bar", "Bar", "dec", "./dec")
}

func TestGetConstructorParameters_DecoratedArgumentWithACall(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/dec.ts",
			Contents: `
			export function dec(target: any, key: string, index: number) {}
			`,
		},
		{
			Name: "/entry.ts",
			Contents: `
			import {dec} from './dec';
			class Bar {}

			class Foo {
			  constructor(@dec bar: Bar) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 1)
	expectParameter(t, args[0], "bar", "Bar", "dec", "./dec")
}

func TestGetConstructorParameters_DecoratedArgumentWithIndirection(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/bar.ts",
			Contents: `
			export class Bar {}
			`,
		},
		{
			Name: "/entry.ts",
			Contents: `
			import {Bar} from './bar';
			import * as star from './bar';

			class Foo {
			  constructor(bar: Bar, otherBar: star.Bar) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 2)
	expectParameterImported(t, args[0], "bar", "./bar", "Bar")
	expectParameterImported(t, args[1], "otherBar", "./bar", "Bar")
}

func TestGetConstructorParameters_AliasedImportArgument(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/bar.ts",
			Contents: `
			export class Bar {}
			`,
		},
		{
			Name: "/entry.ts",
			Contents: `
			import {Bar as LocalBar} from './bar';

			class Foo {
			  constructor(bar: LocalBar) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 1)
	expectParameterImported(t, args[0], "bar", "./bar", "Bar")
}

func TestGetConstructorParameters_NamespaceDeclarationsArgument(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			export declare class Bar {}
			declare namespace i1 {
			  export {
			    Bar,
			  }
			}

			class Foo {
			  constructor(bar: i1.Bar) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 1)
	expectParameter(t, args[0], "bar", "i1.Bar", "", "")
}

func TestGetConstructorParameters_DefaultImportArgument(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/bar.ts",
			Contents: `
			export default class Bar {}
			`,
		},
		{
			Name: "/entry.ts",
			Contents: `
			import Bar from './bar';

			class Foo {
			  constructor(bar: Bar) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 1)

	tvr := args[0].TypeValueReference
	require.NotNil(t, tvr, "typeValueReference should not be nil")
	assert.Equal(t, reflection.Local, tvr.GetKind(), "Expected LOCAL kind")

	local, ok := tvr.(*reflection.LocalTypeValueReference)
	require.True(t, ok, "Expected LocalTypeValueReference")
	assert.NotNil(t, local.DefaultImportStatement, "defaultImportStatement should not be nil")
}

func TestGetConstructorParameters_NullableArgument(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/bar.ts",
			Contents: `
			export class Bar {}
			`,
		},
		{
			Name: "/entry.ts",
			Contents: `
			import {Bar} from './bar';

			class Foo {
			  constructor(bar: Bar|null) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 1)
	expectParameterImported(t, args[0], "bar", "./bar", "Bar")
}

func TestGetConstructorParameters_OverloadedConstructor(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			class Bar {}
			class Baz {}

			class Foo {
			  constructor(bar: Bar);
			  constructor(bar: Bar, baz?: Baz) {}
			}
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	args := host.GetConstructorParameters(clazz)
	require.Len(t, args, 2)
	expectParameter(t, args[0], "bar", "Bar", "", "")
	expectParameter(t, args[1], "baz", "Baz", "", "")
}

// ─── getImportOfIdentifier ───────────────────────────────────────────────────

func TestGetImportOfIdentifier_DirectImport(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{Name: "/node_modules/absolute/index.ts", Contents: "export class Target {}"},
		{
			Name: "/entry.ts",
			Contents: `
			import {Target} from 'absolute';
			let foo: Target;
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")

	fooNode := ngtsctest.FindVariableDeclaration(sf, "foo")
	require.NotNil(t, fooNode, "variable 'foo' not found")

	varDecl := fooNode.AsVariableDeclaration()
	require.NotNil(t, varDecl.Type, "foo.type should not be nil")
	require.True(t, ast.IsTypeReferenceNode(varDecl.Type), "foo.type should be a TypeReferenceNode")

	typeRef := varDecl.Type.AsTypeReferenceNode()
	require.True(t, ast.IsIdentifier(typeRef.TypeName), "typeName should be an identifier")
	target := typeRef.TypeName

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	directImport := host.GetImportOfIdentifier(target)
	require.NotNil(t, directImport, "directImport should not be nil")

	assert.Equal(t, "Target", directImport.Name)
	assert.Equal(t, "absolute", directImport.From)

	importDecl := findFirstImportDeclaration(sf)
	assert.Equal(t, importDecl, directImport.Node)
}

func TestGetImportOfIdentifier_NamespacedImport(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{Name: "/node_modules/absolute/index.ts", Contents: "export class Target {}"},
		{
			Name: "/entry.ts",
			Contents: `
			import * as abs from 'absolute';
			let foo: abs.Target;
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")

	fooNode := ngtsctest.FindVariableDeclaration(sf, "foo")
	require.NotNil(t, fooNode, "variable 'foo' not found")

	varDecl := fooNode.AsVariableDeclaration()
	require.NotNil(t, varDecl.Type, "foo.type should not be nil")
	require.True(t, ast.IsTypeReferenceNode(varDecl.Type), "foo.type should be a TypeReferenceNode")

	typeRef := varDecl.Type.AsTypeReferenceNode()
	require.True(t, ast.IsQualifiedName(typeRef.TypeName), "typeName should be a QualifiedName")
	target := typeRef.TypeName.AsQualifiedName().Right

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	namespacedImport := host.GetImportOfIdentifier(target)
	require.NotNil(t, namespacedImport, "namespacedImport should not be nil")

	assert.Equal(t, "Target", namespacedImport.Name)
	assert.Equal(t, "absolute", namespacedImport.From)

	importDecl := findFirstImportDeclaration(sf)
	assert.Equal(t, importDecl, namespacedImport.Node)
}

// ─── getDeclarationOfIdentifier ──────────────────────────────────────────────

func TestGetDeclarationOfIdentifier_ReExport(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{Name: "/node_modules/absolute/index.ts", Contents: "export class Target {}"},
		{Name: "/local1.ts", Contents: `export {Target as AliasTarget} from 'absolute';`},
		{Name: "/local2.ts", Contents: `export {AliasTarget as Target} from './local1';`},
		{
			Name: "/entry.ts",
			Contents: `
			import {Target} from './local2';
			import {Target as DirectTarget} from 'absolute';

			const target = Target;
			const directTarget = DirectTarget;
			`,
		},
	})
	defer result.Release()
	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")

	targetVarNode := ngtsctest.FindVariableDeclaration(sf, "target")
	require.NotNil(t, targetVarNode, "'target' variable not found")
	targetVarDecl := targetVarNode.AsVariableDeclaration()
	require.NotNil(t, targetVarDecl.Initializer, "target.initializer should not be nil")
	require.True(t, ast.IsIdentifier(targetVarDecl.Initializer), "target.initializer should be an identifier")
	targetId := targetVarDecl.Initializer

	directTargetVarNode := ngtsctest.FindVariableDeclaration(sf, "directTarget")
	require.NotNil(t, directTargetVarNode, "'directTarget' variable not found")
	directTargetVarDecl := directTargetVarNode.AsVariableDeclaration()
	require.NotNil(t, directTargetVarDecl.Initializer, "directTarget.initializer should not be nil")
	require.True(t, ast.IsIdentifier(directTargetVarDecl.Initializer), "directTarget.initializer should be an identifier")
	directTargetId := directTargetVarDecl.Initializer

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	targetDecl := host.GetDeclarationOfIdentifier(targetId)
	directTargetDecl := host.GetDeclarationOfIdentifier(directTargetId)

	require.NotNil(t, targetDecl, "No declaration found for Target")
	require.NotNil(t, directTargetDecl, "No declaration found for DirectTarget")

	// The target declaration should be in absolute/index.ts
	absIndexSF := ngtsctest.RequireSourceFileByName(t, result, "absolute/index.ts")
	// Walk up parent chain to find the source file of the declaration
	declSFNode := findSourceFileNode(targetDecl.Node)
	assert.NotNil(t, declSFNode, "could not find source file of target declaration")
	assert.Same(t, absIndexSF.AsNode(), declSFNode,
		"target declaration should be in absolute/index.ts")
	assert.True(t, ast.IsClassDeclaration(targetDecl.Node), "target declaration should be a class declaration")

	assert.Equal(t, "absolute", directTargetDecl.ViaModule)
	assert.Same(t, targetDecl.Node, directTargetDecl.Node)
}

// findSourceFileNode walks up the parent chain to find the SourceFile node.
func findSourceFileNode(node *ast.Node) *ast.Node {
	for node != nil {
		if ast.IsSourceFile(node) {
			return node
		}
		node = node.Parent
	}
	return nil
}


func TestGetDeclarationOfIdentifier_DirectImport(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{Name: "/node_modules/absolute/index.ts", Contents: "export class Target {}"},
		{
			Name: "/entry.ts",
			Contents: `
			import {Target} from 'absolute';
			let foo: Target;
			`,
		},
	})
	defer result.Release()

	absSF := ngtsctest.RequireSourceFile(t, result, "/node_modules/absolute/index.ts")
	targetDeclNode := ngtsctest.FindNamedClassDeclaration(absSF, "Target")
	require.NotNil(t, targetDeclNode, "Target class not found in absolute/index.ts")

	entrySF := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	fooNode := ngtsctest.FindVariableDeclaration(entrySF, "foo")
	require.NotNil(t, fooNode, "foo not found")

	varDecl := fooNode.AsVariableDeclaration()
	require.NotNil(t, varDecl.Type)
	require.True(t, ast.IsTypeReferenceNode(varDecl.Type))
	typeRef := varDecl.Type.AsTypeReferenceNode()
	require.True(t, ast.IsIdentifier(typeRef.TypeName))
	target := typeRef.TypeName

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	decl := host.GetDeclarationOfIdentifier(target)
	require.NotNil(t, decl)

	assert.Same(t, targetDeclNode, decl.Node)
	assert.Equal(t, "absolute", decl.ViaModule)
}

func TestGetDeclarationOfIdentifier_NamespacedImport(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{Name: "/node_modules/absolute/index.ts", Contents: "export class Target {}"},
		{
			Name: "/entry.ts",
			Contents: `
			import * as abs from 'absolute';
			let foo: abs.Target;
			`,
		},
	})
	defer result.Release()

	absSF := ngtsctest.RequireSourceFile(t, result, "/node_modules/absolute/index.ts")
	targetDeclNode := ngtsctest.FindNamedClassDeclaration(absSF, "Target")
	require.NotNil(t, targetDeclNode, "Target class not found in absolute/index.ts")

	entrySF := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	fooNode := ngtsctest.FindVariableDeclaration(entrySF, "foo")
	require.NotNil(t, fooNode, "foo not found")

	varDecl := fooNode.AsVariableDeclaration()
	require.NotNil(t, varDecl.Type)
	require.True(t, ast.IsTypeReferenceNode(varDecl.Type))
	typeRef := varDecl.Type.AsTypeReferenceNode()
	require.True(t, ast.IsQualifiedName(typeRef.TypeName))
	target := typeRef.TypeName.AsQualifiedName().Right

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	decl := host.GetDeclarationOfIdentifier(target)
	require.NotNil(t, decl)

	assert.Same(t, targetDeclNode, decl.Node)
	assert.Equal(t, "absolute", decl.ViaModule)
}

// ─── getExportsOfModule ───────────────────────────────────────────────────────

func TestGetExportsOfModule_SimpleExports(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			export const x = 10;
			export function foo() {}
			export type T = string;
			export interface I {}
			export enum E {}
			`,
		},
	})
	defer result.Release()

	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	exportedDecls := host.GetExportsOfModule(sf.AsNode())
	require.NotNil(t, exportedDecls)

	keys := make([]string, 0, len(exportedDecls))
	for k := range exportedDecls {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// The TypeScript spec expects keys in declaration order: ['foo', 'x', 'T', 'I', 'E']
	// Our implementation returns a map so we just check the set of names and all viaModule are empty
	expectedKeys := []string{"E", "I", "T", "foo", "x"}
	assert.Equal(t, expectedKeys, keys, "Exported names should match")

	for _, decl := range exportedDecls {
		assert.Empty(t, decl.ViaModule, "ViaModule should be empty for local declarations")
	}
}

func TestGetExportsOfModule_ReExports(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{Name: "/node_modules/absolute/index.ts", Contents: "export class Target {}"},
		{Name: "/local1.ts", Contents: `export {Target as AliasTarget} from 'absolute';`},
		{Name: "/local2.ts", Contents: `export {AliasTarget as Target} from './local1';`},
		{
			Name: "/entry.ts",
			Contents: `
			export {Target as Target1} from 'absolute';
			export {AliasTarget} from './local1';
			export {Target as AliasTarget2} from './local2';
			export * from 'absolute';
			`,
		},
	})
	defer result.Release()

	sf := ngtsctest.RequireSourceFile(t, result, "/entry.ts")
	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	exportedDecls := host.GetExportsOfModule(sf.AsNode())
	require.NotNil(t, exportedDecls)

	// The TypeScript spec expects ['Target1', 'AliasTarget', 'AliasTarget2', 'Target']
	expectedNames := map[string]bool{
		"Target1":     true,
		"AliasTarget": true,
		"AliasTarget2": true,
		"Target":      true,
	}
	for k := range exportedDecls {
		assert.True(t, expectedNames[k], "Unexpected export: %s", k)
	}
	assert.Len(t, exportedDecls, 4, "Expected 4 exports")

	for _, decl := range exportedDecls {
		assert.Empty(t, decl.ViaModule, "ViaModule should be empty")
	}
}

// ─── getMembersOfClass ────────────────────────────────────────────────────────

func TestGetMembersOfClass_StringLiteralMembers(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			class Foo {
			  'string-literal-property-member' = 'my value';
			}
			`,
		},
	})
	defer result.Release()
	members := getMembersHelper(t, result, "/entry.ts")
	require.Len(t, members, 1)
	expectMember(t, members[0], "string-literal-property-member", reflection.Property)
}

func TestGetMembersOfClass_MethodMembers(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			class Foo {
			  myMethod(): void {
			  }
			}
			`,
		},
	})
	defer result.Release()
	members := getMembersHelper(t, result, "/entry.ts")
	require.Len(t, members, 1)
	expectMember(t, members[0], "myMethod", reflection.Method)
}

func TestGetMembersOfClass_ConstructorAsMember(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			class Foo {
			  constructor() {}
			}
			`,
		},
	})
	defer result.Release()
	members := getMembersHelper(t, result, "/entry.ts")
	require.Len(t, members, 1)
	expectMember(t, members[0], "constructor", reflection.Constructor)
}

func TestGetMembersOfClass_DecoratorsOfMember(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			declare var Input;

			class Foo {
			  @Input()
			  prop: string;
			}
			`,
		},
	})
	defer result.Release()
	members := getMembersHelper(t, result, "/entry.ts")
	require.Len(t, members, 1)
	require.NotNil(t, members[0].Decorators)
	require.Greater(t, len(members[0].Decorators), 0)
	assert.Equal(t, "Input", members[0].Decorators[0].Name)
}

func TestGetMembersOfClass_StaticMembers(t *testing.T) {
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/entry.ts",
			Contents: `
			class Foo {
			  static staticMember = '';
			}
			`,
		},
	})
	defer result.Release()
	members := getMembersHelper(t, result, "/entry.ts")
	require.Len(t, members, 1)
	assert.True(t, members[0].IsStatic)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func getMembersHelper(t *testing.T, result *ngtsctest.ProgramResult, fileName string) []reflection.ClassMember {
	t.Helper()
	sf := ngtsctest.RequireSourceFile(t, result, fileName)
	clazz := ngtsctest.FindNamedClassDeclaration(sf, "Foo")
	require.NotNil(t, clazz, "class Foo not found")
	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	return host.GetMembersOfClass(clazz)
}

func expectMember(t *testing.T, member reflection.ClassMember, name string, kind reflection.ClassMemberKind) {
	t.Helper()
	assert.Equal(t, name, member.Name)
	assert.Equal(t, kind, member.Kind)
}

// expectParameter checks a LOCAL-kind parameter by expression string.
// If decoratorName != "", also verifies the decorator with its import.
func expectParameter(t *testing.T, param reflection.CtorParameter, name string, typeStr string, decoratorName string, decoratorFrom string) {
	t.Helper()
	assert.Equal(t, name, param.Name)

	tvr := param.TypeValueReference
	require.NotNil(t, tvr, "typeValueReference should not be nil for param %s", name)
	require.NotEqual(t, reflection.Unavailable, tvr.GetKind(),
		"Expected parameter %s to have a typeValueReference (got Unavailable)", name)
	require.Equal(t, reflection.Local, tvr.GetKind(),
		"Expected LOCAL kind for parameter %s, got kind=%v", name, tvr.GetKind())

	local, ok := tvr.(*reflection.LocalTypeValueReference)
	require.True(t, ok, "Expected *LocalTypeValueReference for parameter %s", name)
	expr, err := ngtsctest.ArgExpressionToString(local.Expression)
	require.NoError(t, err, "ArgExpressionToString failed for param %s", name)
	assert.Equal(t, typeStr, expr, "Type expression mismatch for param %s", name)

	if decoratorName != "" {
		require.NotNil(t, param.Decorators)
		found := false
		for _, dec := range param.Decorators {
			if dec.Name == decoratorName && dec.Import != nil && dec.Import.From == decoratorFrom {
				found = true
				break
			}
		}
		assert.True(t, found, "decorator %s from %s not found on param %s", decoratorName, decoratorFrom, name)
	}
}

// expectParameterImported checks an IMPORTED-kind parameter.
func expectParameterImported(t *testing.T, param reflection.CtorParameter, name string, moduleName string, importedName string) {
	t.Helper()
	assert.Equal(t, name, param.Name)

	tvr := param.TypeValueReference
	require.NotNil(t, tvr, "typeValueReference should not be nil for param %s", name)
	require.NotEqual(t, reflection.Unavailable, tvr.GetKind(),
		"Expected parameter %s to have a typeValueReference (got Unavailable)", name)
	require.Equal(t, reflection.Imported, tvr.GetKind(),
		"Expected IMPORTED kind for parameter %s, got kind=%v", name, tvr.GetKind())

	imp, ok := tvr.(*reflection.ImportedTypeValueReference)
	require.True(t, ok, "Expected *ImportedTypeValueReference for parameter %s", name)
	assert.Equal(t, moduleName, imp.ModuleName, "ModuleName mismatch for param %s", name)
	assert.Equal(t, importedName, imp.ImportedName, "ImportedName mismatch for param %s", name)
}
