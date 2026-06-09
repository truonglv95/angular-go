package compiler

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type R3InjectableMetadata struct {
	Name              string
	Type              *render3.R3Reference
	TypeArgumentCount int
	ProvidedIn        *render3.MaybeForwardRefExpression
	UseClass          *render3.MaybeForwardRefExpression
	UseFactory        output.Expression
	UseExisting       *render3.MaybeForwardRefExpression
	UseValue          *render3.MaybeForwardRefExpression
	Deps              []*render3.R3DependencyMetadata
}

func CompileInjectable(
	meta *R3InjectableMetadata,
	resolveForwardRefs bool,
) *render3.R3CompiledExpression {
	if meta == nil || meta.Type == nil {
		return nil
	}

	compiled := render3.CompileInjectableFromMetadata(render3.R3InjectableMetadata{
		Name:              meta.Name,
		Type:              *meta.Type,
		TypeArgumentCount: meta.TypeArgumentCount,
		ProvidedIn:        convertMaybeForwardRef(meta.ProvidedIn),
		UseClass:          convertMaybeForwardRef(meta.UseClass),
		UseFactory:        meta.UseFactory,
		UseExisting:       convertMaybeForwardRef(meta.UseExisting),
		UseValue:          convertMaybeForwardRef(meta.UseValue),
	})
	return &compiled
}

func CreateInjectableType(typ output.Expression, typeArgumentCount int) output.Type {
	return output.NewExpressionType(
		output.NewExternalExpr(*render3.Identifiers.InjectableDeclaration, nil, []output.Type{
			render3.TypeWithParameters(typ, typeArgumentCount),
		}, nil, nil),
	)
}

func DelegateToFactory(
	typ output.Expression,
	useType output.Expression,
	unwrapForwardRefs bool,
) output.Expression {
	if useType != nil {
		return output.NewReadPropExpr(useType, "ɵfac", nil, nil, nil, false)
	}
	if typ != nil {
		return output.NewReadPropExpr(typ, "ɵfac", nil, nil, nil, false)
	}
	return nil
}

func createFactoryFunction(typ output.Expression) *output.ArrowFunctionExpr {
	var body any
	if typ != nil {
		body = output.NewInstantiateExpr(typ, nil, nil, nil, nil)
	}
	return output.NewArrowFunctionExpr(nil, body, nil, nil, nil)
}

func convertMaybeForwardRef(expr *render3.MaybeForwardRefExpression) output.Expression {
	if expr == nil {
		return nil
	}
	return render3.ConvertFromMaybeForwardRefExpression(*expr)
}
