package render3

// Port of angular/packages/compiler/src/render3/r3_injector_any

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// R3InjectorMetadata holds metadata for injector pipeline.
type R3InjectorMetadata struct {
	Name      string
	Type      R3Reference
	Providers output.Expression // nil if not set
	Imports   []output.Expression
}

// CompileInjector compiles an R3InjectorMetadata into an R3CompiledExpression.
func CompileInjector(meta R3InjectorMetadata) R3CompiledExpression {
	definitionMap := NewDefinitionMap()

	if meta.Providers != nil {
		definitionMap.Set("providers", meta.Providers)
	}

	if len(meta.Imports) > 0 {
		definitionMap.Set("imports", LiteralArr(meta.Imports))
	}

	expression := ImportExpr(*Identifiers.DefineInjector).
		CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, true, nil)
	type_ := CreateInjectorType(meta)
	return R3CompiledExpression{Expression: expression, Type: type_, Statements: []output.Statement{}}
}

// CreateInjectorType creates the type expression for an injector.
func CreateInjectorType(meta R3InjectorMetadata) output.Type {
	return output.NewExpressionType(
		output.NewExternalExpr(*Identifiers.InjectorDeclaration,
			nil,
			[]output.Type{output.NewExpressionType(meta.Type.Type)},
			nil,
			nil,
		),
	)
}
