package partial_evaluator

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type ResolvedValue = int

type ResolvedValueArray interface {
}

type ResolvedValueMap interface {
}

type ResolvedModule struct {
}

func (recv *ResolvedModule) GetExport(name string) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *ResolvedModule) GetExports() ResolvedValueMap {
	// TODO: stub
	panic("unimplemented")
}

type EnumValue struct {
}

type KnownFn struct {
}

func (recv *KnownFn) Evaluate(node ast.Node, args ResolvedValueArray) ResolvedValue {
	// TODO: stub
	panic("unimplemented")
}
