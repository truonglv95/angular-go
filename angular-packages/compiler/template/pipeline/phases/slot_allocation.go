package phases

import (
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
			handle.Slot = &slotCount
			slotMap[trait.Xref()] = slotCount
			// NumSlotsUsed is a concrete field, use a duck-typed interface to retrieve it.
			type numSlotsUsedGetter interface {
				GetNumSlotsUsed() int
			}
			numSlots := 1
			if getter, ok := op.(numSlotsUsedGetter); ok {
				numSlots = getter.GetNumSlotsUsed()
			}
			slotCount += numSlots
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
