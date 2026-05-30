package partial_evaluator

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type ForeignFunctionResolver = any

type ForeignTypeResolver = any

type PartialEvaluator struct {
}

func (recv *PartialEvaluator) Evaluate(expr ast.Expression, foreignFunctionResolver ForeignFunctionResolver) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *PartialEvaluator) EvaluateType(typeNode ast.Node, owningModule any, foreignFunctionResolver ForeignFunctionResolver, foreignTypeResolver ForeignTypeResolver) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}
