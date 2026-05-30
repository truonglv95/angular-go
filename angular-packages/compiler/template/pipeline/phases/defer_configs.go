package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ConfigureDeferInstructions collects the timing configuration arrays into the component's consts.
func ConfigureDeferInstructions(job compilation.CompilationJob) {
	cJob, ok := job.(*compilation.ComponentCompilationJob)
	if !ok {
		return
	}

	for _, unit := range cJob.GetUnits() {
		for _, op := range unit.GetCreate().Ops {
			deferOp, ok := op.(*ir.DeferOp)
			if !ok {
				continue
			}

			if deferOp.PlaceholderMinimumTime != nil {
				collected := &ir.ConstCollectedExpr{
					Expr: deferConfigArray(deferOp.PlaceholderMinimumTime),
				}
				collected.Self = collected
				deferOp.PlaceholderConfig = collected
			}

			if deferOp.LoadingMinimumTime != nil || deferOp.LoadingAfterTime != nil {
				collected := &ir.ConstCollectedExpr{
					Expr: deferConfigArray(deferOp.LoadingMinimumTime, deferOp.LoadingAfterTime),
				}
				collected.Self = collected
				deferOp.LoadingConfig = collected
			}
		}
	}
}

func deferConfigArray(times ...*int) output.Expression {
	entries := make([]output.Expression, len(times))
	for i, t := range times {
		if t == nil {
			entries[i] = output.NewLiteralExpr(nil, nil, nil, nil)
		} else {
			entries[i] = output.NewLiteralExpr(*t, nil, nil, nil)
		}
	}
	return output.NewLiteralArrayExpr(entries, nil, nil, nil)
}
