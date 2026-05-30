package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// InsertIncrementalHydrationRuntime inserts EnableIncrementalHydrationRuntimeOp before the first DeferOp with hydrate triggers.
func InsertIncrementalHydrationRuntime(job compilation.CompilationJob) {
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

			if hasHydrateTriggers(deferOp) {
				runtimeOp := &ir.EnableIncrementalHydrationRuntimeOp{
					SourceSpan: deferOp.SourceSpan,
				}
				ir.OpListInsertBefore(unit.GetCreate(), runtimeOp, deferOp)
				// Only the first hydrating defer in the view needs the activator.
				break
			}
		}
	}
}

func hasHydrateTriggers(op *ir.DeferOp) bool {
	if op.Flags == nil {
		return false
	}
	switch f := op.Flags.(type) {
	case ir.TDeferDetailsFlags:
		return (f & ir.TDeferDetailsFlagsHasHydrateTriggers) != 0
	case int:
		return (ir.TDeferDetailsFlags(f) & ir.TDeferDetailsFlagsHasHydrateTriggers) != 0
	}
	return false
}
