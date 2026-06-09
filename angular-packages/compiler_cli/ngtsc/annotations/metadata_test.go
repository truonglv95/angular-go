package annotations_test

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
)

func compileAndPrintMetadata(t *testing.T, contents string) string {
	res := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/node_modules/@angular/core/index.d.ts",
			Contents: `
				export declare function Input(...args: any[]): any;
				export declare function Inject(...args: any[]): any;
				export declare function Component(...args: any[]): any;
				export declare function Directive(...args: any[]): any;
				export declare class Injector {}
			`,
		},
		{
			Name: "/index.ts",
			Contents: contents,
		},
	})
	defer res.Release()

	host := reflection.NewTypeScriptReflectionHost(res.Checker)
	sf := ngtsctest.RequireSourceFile(t, res, "/index.ts")
	target := ngtsctest.RequireDeclaration(t, sf, "Target", ast.IsClassDeclaration)

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	meta := extractClassMetadata(t, target, host, false, factory)
	if meta == nil {
		return ""
	}

	mgr := imports.NewImportManager(&imports.ImportManagerConfig{
		ForceGenerateNamespacesForNewImports: true,
	}, factory)

	classMetaExpr := render3.CompileClassMetadata(*meta)
	stmt := classMetaExpr.ToStmt(nil)

	visitor := translator.NewExpressionTranslatorVisitor(factory, mgr, nil, translator.TranslatorOptions{})
	translated := stmt.VisitStatement(visitor, translator.Context{IsStatementMode: true}).(*ast.Node)

	printed := printNode(translated)
	return strings.Join(strings.Fields(printed), " ")
}

func cloneNodeSynthetically(node *ast.Node, factory *ast.NodeFactory) *ast.Node {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case ast.KindIdentifier:
		return factory.NewIdentifier(node.AsIdentifier().Text)
	case ast.KindPropertyAccessExpression:
		pae := node.AsPropertyAccessExpression()
		expr := cloneNodeSynthetically(pae.Expression, factory)
		name := cloneNodeSynthetically(pae.Name(), factory)
		return factory.NewPropertyAccessExpression((*ast.Expression)(expr), nil, (*ast.MemberName)(name), 0)
	case ast.KindStringLiteral:
		return factory.NewStringLiteral(node.AsStringLiteral().Text, ast.TokenFlags(0))
	case ast.KindNoSubstitutionTemplateLiteral:
		return factory.NewNoSubstitutionTemplateLiteral(node.AsNoSubstitutionTemplateLiteral().Text, ast.TokenFlags(0))
	case ast.KindQualifiedName:
		qn := node.AsQualifiedName()
		left := cloneNodeSynthetically(qn.Left, factory)
		right := cloneNodeSynthetically(qn.Right.AsNode(), factory)
		return factory.NewQualifiedName((*ast.EntityName)(left), (*ast.IdentifierNode)(right))
	case ast.KindArrayLiteralExpression:
		var elements []*ast.Node
		if node.AsArrayLiteralExpression().Elements != nil {
			for _, el := range node.AsArrayLiteralExpression().Elements.Nodes {
				elements = append(elements, cloneNodeSynthetically(el, factory))
			}
		}
		return factory.NewArrayLiteralExpression(factory.NewNodeList(elements), false).AsNode()
	case ast.KindObjectLiteralExpression:
		var props []*ast.Node
		if node.AsObjectLiteralExpression().Properties != nil {
			for _, p := range node.AsObjectLiteralExpression().Properties.Nodes {
				props = append(props, cloneNodeSynthetically(p, factory))
			}
		}
		return factory.NewObjectLiteralExpression(factory.NewNodeList(props), false).AsNode()
	case ast.KindPropertyAssignment:
		pa := node.AsPropertyAssignment()
		name := cloneNodeSynthetically(pa.Name(), factory)
		init := cloneNodeSynthetically(pa.Initializer, factory)
		return factory.NewPropertyAssignment(nil, (*ast.PropertyName)(name), nil, nil, (*ast.Expression)(init))
	case ast.KindNumericLiteral:
		return factory.NewNumericLiteral(node.AsNumericLiteral().Text, 0).AsNode()
	case ast.KindPrefixUnaryExpression:
		pue := node.AsPrefixUnaryExpression()
		operand := cloneNodeSynthetically(pue.Operand, factory)
		return factory.NewPrefixUnaryExpression(pue.Operator, operand).AsNode()
	default:
		return node
	}
}

func extractClassMetadata(t *testing.T, clazz *ast.Node, host reflection.ReflectionHost, isCore bool, factory *ast.NodeFactory) *render3.R3ClassMetadata {
	if !host.IsClass(clazz) {
		return nil
	}

	classDecorators := host.GetDecoratorsOfDeclaration(clazz)
	if classDecorators == nil {
		return nil
	}

	var ngClassDecorators []output.Expression
	for _, dec := range classDecorators {
		if isAngularDecorator(dec, isCore) {
			meta := decoratorToMetadata(dec, factory)
			ngClassDecorators = append(ngClassDecorators, meta)
		}
	}

	if len(ngClassDecorators) == 0 {
		return nil
	}

	metaDecorators := output.NewLiteralArrayExpr(ngClassDecorators, nil, nil, nil)

	// constructor parameters
	var metaCtorParameters output.Expression
	classCtorParameters := host.GetConstructorParameters(clazz)
	if classCtorParameters != nil {
		var ctorParameters []output.Expression
		for _, param := range classCtorParameters {
			ctorParameters = append(ctorParameters, ctorParameterToMetadata(param, isCore, factory))
		}
		metaCtorParameters = output.NewArrowFunctionExpr(
			nil,
			output.NewLiteralArrayExpr(ctorParameters, nil, nil, nil),
			nil, nil, nil,
		)
	}

	// property decorators
	var metaPropDecorators output.Expression
	classMembers := host.GetMembersOfClass(clazz)
	var decoratedMembers []output.LiteralMapEntry

	for _, member := range classMembers {
		if member.IsStatic {
			continue
		}
		if member.AccessLevel == reflection.EcmaScriptPrivate {
			continue
		}

		if member.Decorators != nil && len(member.Decorators) > 0 {
			var memberNgDecorators []output.Expression
			for _, dec := range member.Decorators {
				if isAngularDecorator(dec, isCore) {
					memberNgDecorators = append(memberNgDecorators, decoratorToMetadata(dec, factory))
				}
			}
			if len(memberNgDecorators) > 0 {
				val := output.NewLiteralArrayExpr(memberNgDecorators, nil, nil, nil)
				quoted := false
				if member.NameNode != nil && (ast.IsStringLiteral(member.NameNode) || ast.IsNoSubstitutionTemplateLiteral(member.NameNode)) {
					quoted = true
				}
				decoratedMembers = append(decoratedMembers, output.NewLiteralMapPropertyAssignment(member.Name, val, quoted))
			}
		}
	}

	if len(decoratedMembers) > 0 {
		metaPropDecorators = output.NewLiteralMapExpr(decoratedMembers, nil, nil, nil)
	}

	className := ""
	if clazz.AsClassDeclaration().Name() != nil {
		className = clazz.AsClassDeclaration().Name().AsIdentifier().Text
	}

	return &render3.R3ClassMetadata{
		Type:           output.NewReadVarExpr(className, nil, nil, nil),
		Decorators:     metaDecorators,
		CtorParameters: metaCtorParameters,
		PropDecorators: metaPropDecorators,
	}
}

func isAngularDecorator(decorator reflection.Decorator, isCore bool) bool {
	return isCore || (decorator.Import != nil && decorator.Import.From == "@angular/core")
}

func decoratorToMetadata(decorator reflection.Decorator, factory *ast.NodeFactory) output.Expression {
	var props []output.LiteralMapEntry
	clonedIdent := cloneNodeSynthetically(decorator.Identifier, factory)
	props = append(props, output.NewLiteralMapPropertyAssignment(
		"type",
		output.NewWrappedNodeExpr(clonedIdent, nil, nil, nil),
		false,
	))

	if len(decorator.Args) > 0 {
		var args []output.Expression
		for _, arg := range decorator.Args {
			clonedArg := cloneNodeSynthetically(arg, factory)
			args = append(args, output.NewWrappedNodeExpr(clonedArg, nil, nil, nil))
		}
		props = append(props, output.NewLiteralMapPropertyAssignment(
			"args",
			output.NewLiteralArrayExpr(args, nil, nil, nil),
			false,
		))
	}

	return output.NewLiteralMapExpr(props, nil, nil, nil)
}

func ctorParameterToMetadata(param reflection.CtorParameter, isCore bool, factory *ast.NodeFactory) output.Expression {
	var typeExpr output.Expression
	if param.TypeValueReference.GetKind() != reflection.Unavailable {
		typeExpr = valueReferenceToExpression(param.TypeValueReference, factory)
	} else {
		typeExpr = output.NewReadVarExpr("undefined", nil, nil, nil)
	}

	var mapEntries []output.LiteralMapEntry
	mapEntries = append(mapEntries, output.NewLiteralMapPropertyAssignment("type", typeExpr, false))

	if param.Decorators != nil && len(param.Decorators) > 0 {
		var decorators []output.Expression
		for _, dec := range param.Decorators {
			if isAngularDecorator(dec, isCore) {
				decorators = append(decorators, decoratorToMetadata(dec, factory))
			}
		}
		if len(decorators) > 0 {
			val := output.NewLiteralArrayExpr(decorators, nil, nil, nil)
			mapEntries = append(mapEntries, output.NewLiteralMapPropertyAssignment("decorators", val, false))
		}
	}

	return output.NewLiteralMapExpr(mapEntries, nil, nil, nil)
}

func valueReferenceToExpression(ref reflection.TypeValueReference, factory *ast.NodeFactory) output.Expression {
	switch r := ref.(type) {
	case *reflection.LocalTypeValueReference:
		clonedExpr := cloneNodeSynthetically(r.Expression, factory)
		return output.NewWrappedNodeExpr(clonedExpr, nil, nil, nil)
	case *reflection.ImportedTypeValueReference:
		module := r.ModuleName
		name := r.ImportedName
		return output.NewExternalExpr(output.ExternalReference{
			ModuleName: &module,
			Name:       &name,
		}, nil, nil, nil, nil)
	default:
		return output.NewLiteralExpr(nil, nil, nil, nil)
	}
}

func TestSetClassMetadataConverter(t *testing.T) {
	t.Run("should convert decorated class metadata", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			import {Component} from '@angular/core';

			@Component('metadata') class Target {}
		`)
		assert.Equal(t, `(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(Target, [{ type: Component, args: ["metadata"] }], null, null); })();`, res)
	})

	t.Run("should convert namespaced decorated class metadata", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			import * as core from '@angular/core';

			@core.Component('metadata') class Target {}
		`)
		assert.Equal(t, `(() => { (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(Target, [{ type: core.Component, args: ["metadata"] }], null, null); })();`, res)
	})

	t.Run("should convert decorated class constructor parameter metadata", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			import {Component, Inject, Injector} from '@angular/core';
			const FOO = 'foo';

			@Component('metadata') class Target {
				constructor(@Inject(FOO) foo: any, bar: Injector) {}
			}
		`)
		assert.Contains(t, res, `() => [{ type: undefined, decorators: [{ type: Inject, args: [FOO] }] }, { type: i0.Injector }]`)
	})

	t.Run("should convert decorated field metadata", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			import {Component, Input} from '@angular/core';

			@Component('metadata') class Target {
				@Input() foo: string;

				@Input('value') bar: string;

				notDecorated: string;
			}
		`)
		assert.Contains(t, res, `{ foo: [{ type: Input }], bar: [{ type: Input, args: ["value"] }] })`)
	})

	t.Run("should convert decorated field getter/setter metadata", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			import {Component, Input} from '@angular/core';

			@Component('metadata') class Target {
				@Input() get foo() { return this._foo; }
				set foo(value: string) { this._foo = value; }
				private _foo: string;

				get bar() { return this._bar; }
				@Input('value') set bar(value: string) { this._bar = value; }
				private _bar: string;
			}
		`)
		assert.Contains(t, res, `{ foo: [{ type: Input }], bar: [{ type: Input, args: ["value"] }] })`)
	})

	t.Run("should not convert non-angular decorators to metadata", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			declare function NotAComponent(...args: any[]): any;

			@NotAComponent('metadata') class Target {}
		`)
		assert.Equal(t, "", res)
	})

	t.Run("should preserve quotes around class member names", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			import {Component, Input} from '@angular/core';

			@Component('metadata') class Target {
				@Input() 'has-dashes-in-name' = 123;
				@Input() noDashesInName = 456;
			}
		`)
		assert.Contains(t, res, `{ "has-dashes-in-name": [{ type: Input }], noDashesInName: [{ type: Input }] })`)
	})

	t.Run("should not emit metadata for ECMAScript private fields", func(t *testing.T) {
		res := compileAndPrintMetadata(t, `
			import {Directive, Input} from '@angular/core';

			@Directive() class Target {
				@Input() #privateName = 123;
				@Input() publicName = 456;
			}
		`)
		assert.Contains(t, res, `{ publicName: [{ type: Input }] })`)
	})
}
