package phases

import (
	"errors"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// DeleteAnyCasts finds any function calls to `$any`, excluding `this.$any`, and deletes them,
// since they have no runtime effects.
func DeleteAnyCasts(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			ir.TransformExpressionsInOp(op, removeAnys, ir.VisitorContextFlagNone)
		}
	}
}

func removeAnys(e output.Expression, flags ir.VisitorContextFlag) output.Expression {
	if invoke, ok := e.(*output.InvokeFunctionExpr); ok {
		if lex, ok := invoke.Fn.(*ir.LexicalReadExpr); ok && lex.Name == "$any" {
			if len(invoke.Args) != 1 {
				// We don't panic strictly, but throw an error?
				// Since signature is func(e output.Expression) output.Expression, we can panic.
				panic(errors.New("The $any builtin function expects exactly one argument."))
			}
			return invoke.Args[0]
		}
	}
	return e
}
