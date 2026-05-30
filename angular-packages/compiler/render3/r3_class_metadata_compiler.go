package render3

// Port of angular/packages/compiler/src/render3/r3_class_metadata_any

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// CompileClassMetadataFn is a function type that compiles class metadata.
type CompileClassMetadataFn func(metadata R3ClassMetadata) output.Expression

// R3ClassMetadata holds metadata of a class which captures the original Angular decorators.
type R3ClassMetadata struct {
	// The class type for which the metadata is captured.
	Type output.Expression

	// An expression representing the Angular decorators that were applied on the class.
	Decorators output.Expression

	// An expression representing the Angular decorators applied to constructor parameters,
	// or nil if there is no constructor.
	CtorParameters output.Expression // nil if no constructor

	// An expression representing the Angular decorators applied on the properties of the class,
	// or nil if no properties have decorators.
	PropDecorators output.Expression // nil if no properties
}

// CompileClassMetadata compiles class metadata into an ngDevMode-guarded expression.
func CompileClassMetadata(metadata R3ClassMetadata) *output.InvokeFunctionExpr {
	fnCall := internalCompileClassMetadata(metadata)
	iife := output.NewArrowFunctionExpr(
		nil,
		[]output.Statement{DevOnlyGuardedExpression(fnCall).ToStmt(nil)},
		nil,
		nil,
		nil,
	)
	return iife.CallFn(nil, nil, false, nil)
}

// internalCompileClassMetadata compiles only the setClassMetadata call without additional wrappers.
func internalCompileClassMetadata(metadata R3ClassMetadata) *output.InvokeFunctionExpr {
	ctorParams := metadata.CtorParameters
	if ctorParams == nil {
		ctorParams = output.NewLiteralExpr(nil, nil, nil, nil)
	}
	propDecorators := metadata.PropDecorators
	if propDecorators == nil {
		propDecorators = output.NewLiteralExpr(nil, nil, nil, nil)
	}
	return ImportExpr(*Identifiers.SetClassMetadata).
		CallFn([]output.Expression{
			metadata.Type,
			metadata.Decorators,
			ctorParams,
			propDecorators,
		}, nil, false, nil)
}

// CompileComponentClassMetadata compiles class metadata for a component, potentially
// wrapping it with setClassMetadataAsync if there are deferrable dependencies.
func CompileComponentClassMetadata(
	metadata R3ClassMetadata,
	dependencies []R3DeferPerComponentDependency,
) output.Expression {
	if len(dependencies) == 0 {
		// If there are no deferrable symbols - just generate a regular setClassMetadata call.
		return CompileClassMetadata(metadata)
	}

	wrapperParams := make([]*output.FnParam, len(dependencies))
	for i, dep := range dependencies {
		wrapperParams[i] = output.NewFnParam(dep.SymbolName, output.DYNAMIC_TYPE)
	}

	return internalCompileSetClassMetadataAsync(
		metadata,
		wrapperParams,
		CompileComponentMetadataAsyncResolver(dependencies),
	)
}

// CompileOpaqueAsyncClassMetadata compiles class metadata for cases where we can't analyze
// deferred block dependencies but have a reference to the compiled dependency resolver function.
func CompileOpaqueAsyncClassMetadata(
	metadata R3ClassMetadata,
	deferResolver output.Expression,
	deferredDependencyNames []string,
) output.Expression {
	wrapperParams := make([]*output.FnParam, len(deferredDependencyNames))
	for i, name := range deferredDependencyNames {
		wrapperParams[i] = output.NewFnParam(name, output.DYNAMIC_TYPE)
	}
	return internalCompileSetClassMetadataAsync(metadata, wrapperParams, deferResolver)
}

// internalCompileSetClassMetadataAsync compiles a setClassMetadataAsync call.
func internalCompileSetClassMetadataAsync(
	metadata R3ClassMetadata,
	wrapperParams []*output.FnParam,
	dependencyResolverFn output.Expression,
) output.Expression {
	// Omit the wrapper since it'll be added around setClassMetadataAsync instead.
	setClassMetadataCall := internalCompileClassMetadata(metadata)
	setClassMetaWrapper := output.NewArrowFunctionExpr(
		wrapperParams,
		[]output.Statement{setClassMetadataCall.ToStmt(nil)},
		nil,
		nil,
		nil,
	)
	setClassMetaAsync := ImportExpr(*Identifiers.SetClassMetadataAsync).
		CallFn([]output.Expression{
			metadata.Type,
			dependencyResolverFn,
			setClassMetaWrapper,
		}, nil, false, nil)

	iife := output.NewArrowFunctionExpr(
		nil,
		[]output.Statement{DevOnlyGuardedExpression(setClassMetaAsync).ToStmt(nil)},
		nil,
		nil,
		nil,
	)
	return iife.CallFn(nil, nil, false, nil)
}

// CompileComponentMetadataAsyncResolver compiles the function that loads the dependencies
// for the entire component in setClassMetadataAsync.
func CompileComponentMetadataAsyncResolver(
	dependencies []R3DeferPerComponentDependency,
) *output.ArrowFunctionExpr {
	dynamicImports := make([]output.Expression, len(dependencies))
	for i, dep := range dependencies {
		symbolName := dep.SymbolName
		importPath := dep.ImportPath
		isDefaultImport := dep.IsDefaultImport

		// e.g. `(m) => m.CmpA`
		var propName string
		if isDefaultImport {
			propName = "default"
		} else {
			propName = symbolName
		}
		innerFn := output.NewArrowFunctionExpr(
			[]*output.FnParam{output.NewFnParam("m", output.DYNAMIC_TYPE)},
			VariableExpr("m").Prop(propName, nil),
			nil,
			nil,
			nil,
		)

		// e.g. `import('./cmp-a').then(...)`
		tsIgnore := TsIgnoreComment()
		dynamicImports[i] = output.NewDynamicImportExpr(importPath, nil, nil, nil).
			Prop("then", nil).
			CallFn([]output.Expression{innerFn}, nil, false, []output.LeadingComment{tsIgnore})
	}

	// e.g. `() => [ ... ];`
	return output.NewArrowFunctionExpr(nil, LiteralArr(dynamicImports), nil, nil, nil)
}
