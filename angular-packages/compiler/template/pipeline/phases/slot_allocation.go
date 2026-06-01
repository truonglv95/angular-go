package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// AllocateSlots assigns data slots for all operations which implement ConsumesSlotOpTrait,
// and propagates the assigned data slots into any expressions which reference them.
func AllocateSlots(job *compilation.ComponentCompilationJob) {
	// Map of all declarations in all views within the component which require an assigned slot index.
	slotMap := map[ir.XrefId]int{}

	// Process all views in the component and assign slot indexes.
	for _, view := range job.Views {
		slotCount := 0

		for _, op := range view.GetCreate().Elements() {
			trait, hasTrait := op.(ir.ConsumesSlotTrait)
			if !hasTrait {
				continue
			}
			handle := trait.Handle()
			val := slotCount
			handle.Slot = &val
			slotMap[trait.GetXref()] = slotCount
			numSlots := trait.GetNumSlotsUsed()
			slotCount += numSlots
		}

		for _, op := range view.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindListener:
				if l, ok := op.(*ir.ListenerOp); ok {
					if !l.HostListener {
						if slot, found := slotMap[l.Target]; found {
							l.TargetSlot = ir.SlotHandle{Slot: &slot}
						} else {
							panic("AssertionError: expected Listener to have a target with an allocated slot")
						}
					}
				}
			}
		}
		for _, op := range view.Ops() {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				if trait, ok := expr.(ir.DependsOnSlotContextTrait); ok {
					if slot, found := slotMap[trait.Target()]; found {
						// Assuming the trait allows setting the target slot, but actually we need to switch on the type
						switch e := expr.(type) {
						case *ir.PipeBindingExpr:
							if e.TargetSlot == nil {
								e.TargetSlot = &ir.SlotHandle{}
							}
							slotVal := slot
							e.TargetSlot.Slot = &slotVal
						case *ir.ContextLetReferenceExpr:
							if e.TargetSlot == nil {
								e.TargetSlot = &ir.SlotHandle{}
							}
							slotVal := slot
							e.TargetSlot.Slot = &slotVal
						case *ir.RestoreViewExpr:
							// handle if needed
						}
					} else {
						panic("AssertionError: expected slot to be allocated for expression")
					}
				}
				return expr
			}, ir.VisitorContextFlagNone)
		}
		
		view.Decls = &slotCount
	}

	// After slot assignment, propagate slot assignments into TemplateOps.
	for _, view := range job.Views {
		for _, op := range view.Ops() {
			switch op.Kind() {
			case ir.OpKindTemplate, ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate, ir.OpKindRepeaterCreate:
				if tmpl, ok := op.(interface {
					GetXref() ir.XrefId
					SetDecls(int)
				}); ok {
					if childView, exists := job.Views[tmpl.GetXref()]; exists && childView.Decls != nil {
						tmpl.SetDecls(*childView.Decls)
					}
				}
			}
		}
	}
}
