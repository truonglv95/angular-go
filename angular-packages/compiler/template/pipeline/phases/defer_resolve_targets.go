package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// resolveDeferTargetScope holds the map from local ref names to their element targets.
type resolveDeferTargetScope struct {
	targets map[string]deferTargetInfo
}

type deferTargetInfo struct {
	xref ir.XrefId
	slot *ir.SlotHandle
}

// deferTriggerWithTarget is the interface that concrete trigger types implement if they
// support target resolution.
type deferTriggerWithTarget interface {
	GetKind() ir.DeferTriggerKind
	GetTargetName() *string
	SetTarget(xref ir.XrefId, view ir.XrefId, slot *ir.SlotHandle, steps int)
}

// ResolveDeferTargetNames resolves defer trigger target names to xrefs.
func ResolveDeferTargetNames(job *compilation.ComponentCompilationJob) {
	scopes := map[ir.XrefId]*resolveDeferTargetScope{}

	getScopeForView := func(viewXref ir.XrefId) *resolveDeferTargetScope {
		if scope, ok := scopes[viewXref]; ok {
			return scope
		}
		scope := &resolveDeferTargetScope{targets: map[string]deferTargetInfo{}}
		view, exists := job.Views[viewXref]
		if !exists {
			scopes[viewXref] = scope
			return scope
		}
		for _, op := range view.GetCreate().Elements() {
			if !ir.IsElementOrContainerOp(op) {
				continue
			}
			xg, ok := op.(interface{ GetXref() ir.XrefId })
			if !ok {
				continue
			}
			localRefs, ok2 := op.(interface{ GetLocalRefs() []ir.LocalRef })
			if !ok2 {
				continue
			}
			for _, ref := range localRefs.GetLocalRefs() {
				var slotHandle *ir.SlotHandle
				if slot, ok3 := op.(ir.ConsumesSlotTrait); ok3 {
					slotHandle = slot.Handle()
				}
				scope.targets[ref.Name] = deferTargetInfo{xref: xg.GetXref(), slot: slotHandle}
			}
		}
		scopes[viewXref] = scope
		return scope
	}

	resolveTrigger := func(deferOwnerView *compilation.ViewCompilationUnit, op *ir.DeferOnOp, placeholderView ir.XrefId) {
		trigger, ok := op.Trigger.(deferTriggerWithTarget)
		if !ok {
			return
		}
		switch trigger.GetKind() {
		case ir.DeferTriggerKindIdle, ir.DeferTriggerKindNever, ir.DeferTriggerKindImmediate, ir.DeferTriggerKindTimer:
			return
		case ir.DeferTriggerKindHover, ir.DeferTriggerKindInteraction, ir.DeferTriggerKindViewport:
			if trigger.GetTargetName() == nil {
				if placeholderView == 0 {
					panic("defer on trigger with no target name must have a placeholder block")
				}
				placeholder, exists := job.Views[placeholderView]
				if !exists {
					panic("AssertionError: could not find placeholder view for defer on trigger")
				}
				for _, phOp := range placeholder.GetCreate().Elements() {
					if !ir.HasConsumesSlotTrait(phOp) {
						continue
					}
					if !ir.IsElementOrContainerOp(phOp) && phOp.Kind() != ir.OpKindProjection {
						continue
					}
					xg, ok2 := phOp.(interface{ GetXref() ir.XrefId })
					if !ok2 {
						continue
					}
					var slotHandle *ir.SlotHandle
					if slot, ok3 := phOp.(ir.ConsumesSlotTrait); ok3 {
						slotHandle = slot.Handle()
					}
					trigger.SetTarget(xg.GetXref(), placeholderView, slotHandle, -1)
					return
				}
				return
			}
			var view *compilation.ViewCompilationUnit
			step := 0
			if placeholderView != 0 {
				view = job.Views[placeholderView]
				step = -1
			} else {
				view = deferOwnerView
			}
			for view != nil {
				scope := getScopeForView(view.GetXref())
				if tgt, exists := scope.targets[*trigger.GetTargetName()]; exists {
					trigger.SetTarget(tgt.xref, view.GetXref(), tgt.slot, step)
					return
				}
				if view.Parent != nil {
					view = job.Views[*view.Parent]
					step++
				} else {
					break
				}
			}
			if placeholderView != 0 {
				placeholder, exists := job.Views[placeholderView]
				if !exists {
					return
				}
				for _, phOp := range placeholder.GetCreate().Elements() {
					if !ir.HasConsumesSlotTrait(phOp) {
						continue
					}
					if !ir.IsElementOrContainerOp(phOp) && phOp.Kind() != ir.OpKindProjection {
						continue
					}
					xg, ok2 := phOp.(interface{ GetXref() ir.XrefId })
					if !ok2 {
						continue
					}
					var slotHandle *ir.SlotHandle
					if slot, ok3 := phOp.(ir.ConsumesSlotTrait); ok3 {
						slotHandle = slot.Handle()
					}
					trigger.SetTarget(xg.GetXref(), placeholderView, slotHandle, -1)
					return
				}
			}
		}
	}

	for _, unit := range job.GetUnits() {
		viewUnit, ok := unit.(*compilation.ViewCompilationUnit)
		if !ok {
			continue
		}
		defers := map[ir.XrefId]*ir.DeferOp{}
		for _, op := range viewUnit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindDefer:
				if deferOp, ok2 := op.(*ir.DeferOp); ok2 {
					defers[deferOp.Xref] = deferOp
				}
			case ir.OpKindDeferOn:
				if deferOnOp, ok2 := op.(*ir.DeferOnOp); ok2 {
					deferOp, exists := defers[deferOnOp.Defer]
					if !exists {
						continue
					}
					placeholderView := deferOp.PlaceholderView
					resolveTrigger(viewUnit, deferOnOp, placeholderView)
				}
			}
		}
	}
}
