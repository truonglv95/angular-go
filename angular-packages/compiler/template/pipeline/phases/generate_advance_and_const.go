package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// GenerateAdvance generates ir.AdvanceOps in between ir.UpdateOps to ensure the runtime's
// implicit slot context will be advanced correctly.
func GenerateAdvance(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// Build a map of all declarations in the view with assigned slots.
		slotMap := map[ir.XrefId]int{}
		for _, op := range unit.GetCreate().Elements() {
			trait, ok := op.(ir.ConsumesSlotTrait)
			if !ok {
				continue
			}
			handle := trait.Handle()
			if handle.Slot == nil {
				panic("AssertionError: expected slots to have been allocated before generating advance() calls")
			}
			slotMap[trait.Xref()] = *handle.Slot
		}

		// Step through update ops and generate AdvanceOps as needed.
		slotContext := 0
		for _, op := range unit.GetUpdate().Elements() {
			var consumer ir.DependsOnSlotContextOpTrait
			if depOp, ok := op.(ir.DependsOnSlotContextOpTrait); ok {
				consumer = depOp
			} else {
				ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
					if consumer == nil {
						if depExpr, ok2 := expr.(ir.DependsOnSlotContextOpTrait); ok2 {
							consumer = depExpr
						}
					}
				})
			}

			if consumer == nil {
				continue
			}

			slot, exists := slotMap[consumer.GetTarget()]
			if !exists {
				panic("AssertionError: reference to unknown slot for target")
			}

			if slotContext != slot {
				delta := slot - slotContext
				if delta < 0 {
					panic("AssertionError: slot counter should never need to move backwards")
				}
				ir.InsertBefore(ir.CreateAdvanceOp(delta, nil), op)
				slotContext = slot
			}
		}
	}
}

// CollectConstExpressions lifts ir.ConstCollectedExpr instances into the component const array.
func CollectConstExpressions(job *compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				cce, ok := expr.(*ir.ConstCollectedExpr)
				if !ok {
					return expr
				}
				constIdx := job.AddConst(cce.Expr, nil)
				return output.NewLiteralExpr(constIdx, nil, nil, nil)
			}, ir.VisitorContextFlagNone)
		}
	}
}
