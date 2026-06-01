package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CreateVariadicPipes converts pipes that accept more than 4 arguments to variadic pipes.
func CreateVariadicPipes(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetUpdate().Ops {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				binding, ok := expr.(*ir.PipeBindingExpr)
				if !ok {
					return expr
				}

				// Pipes are variadic if they have more than 4 arguments.
				if len(binding.Args) <= 4 {
					return expr
				}

				variadicPipe := &ir.PipeBindingVariadicExpr{
					TargetXref: binding.Target(),
					TargetSlot: binding.TargetSlot,
					PipeName:   binding.PipeName,
					Args:       output.NewLiteralArrayExpr(binding.Args, nil, nil, nil),
					NumArgs:    len(binding.Args),
				}
				variadicPipe.Self = variadicPipe
				return variadicPipe
			}, ir.VisitorContextFlagNone)
		}
	}
}
