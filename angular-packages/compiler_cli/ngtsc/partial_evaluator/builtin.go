package partial_evaluator

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type ArraySliceBuiltinFn struct {
}

func (recv *ArraySliceBuiltinFn) Evaluate(node ast.Node, args ResolvedValueArray) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}

type ArrayConcatBuiltinFn struct {
}

func (recv *ArrayConcatBuiltinFn) Evaluate(node ast.Node, args ResolvedValueArray) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}

type StringConcatBuiltinFn struct {
}

func (recv *StringConcatBuiltinFn) Evaluate(node ast.Node, args ResolvedValueArray) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}
