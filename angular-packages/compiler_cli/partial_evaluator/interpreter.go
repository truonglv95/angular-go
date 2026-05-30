package partial_evaluator

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
)

type StaticInterpreter struct {}

func NewStaticInterpreter(host reflection.ReflectionHost, checker any) *StaticInterpreter {
	return &StaticInterpreter{}
}

func (si *StaticInterpreter) Visit(node any, context any) any {
	return nil
}

func (si *StaticInterpreter) evaluate(node any, env any) any {
	return nil
}
