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
		var textXrefs []ir.XrefId
		for _, op := range unit.GetCreate().Elements() {
			trait, ok := op.(ir.ConsumesSlotTrait)
			if !ok {
				continue
			}
			handle := trait.Handle()
			if handle.Slot == nil {
				panic("AssertionError: expected slots to have been allocated before generating advance() calls")
			}
			slotMap[trait.GetXref()] = *handle.Slot
			if op.Kind() == ir.OpKindText {
				textXrefs = append(textXrefs, trait.GetXref())
			}
		}

		// Step through update ops and generate AdvanceOps as needed.
		slotContext := 0
		lastInterpolateTextTarget := ir.XrefId(-1)
		var newUpdateOps []ir.Op
		for _, op := range unit.GetUpdate().Elements() {
			if it, ok := op.(*ir.InterpolateTextOp); ok && it.Target == lastInterpolateTextTarget {
				for i, textXref := range textXrefs {
					if textXref == it.Target && i+1 < len(textXrefs) {
						it.Target = textXrefs[i+1]
						break
					}
				}
			}

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
				newUpdateOps = append(newUpdateOps, op)
				continue
			}

			slot, exists := slotMap[consumer.GetTarget()]
			if !exists {
				// We might be trying to advance to an element that doesn't have a slot.
				// E.g. a host binding.
				newUpdateOps = append(newUpdateOps, op)
				continue
			}

			if slotContext != slot {
				delta := slot - slotContext
				if delta < 0 {
					panic("AssertionError: slot counter should never need to move backwards")
				}
				newUpdateOps = append(newUpdateOps, ir.CreateAdvanceOp(delta, nil))
				slotContext = slot
			}
			newUpdateOps = append(newUpdateOps, op)
			if it, ok := op.(*ir.InterpolateTextOp); ok {
				lastInterpolateTextTarget = it.Target
			}
		}
		unit.GetUpdate().Ops = newUpdateOps
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
