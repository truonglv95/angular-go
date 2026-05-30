package partial_evaluator

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type StaticInterpreter struct {
}

func (recv *StaticInterpreter) Visit(node ast.Expression, context any) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *StaticInterpreter) VisitType(node ast.Node, context any) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}
