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
) *render3.R3CompiledExpression { panic("unimplemented") }

func CreateInjectableType(typ output.Expression, typeArgumentCount int) output.Type { panic("unimplemented") }

func DelegateToFactory(
	typ output.Expression,
	useType output.Expression,
	unwrapForwardRefs bool,
) output.Expression { panic("unimplemented") }

func createFactoryFunction(typ output.Expression) *output.ArrowFunctionExpr { panic("unimplemented") }
