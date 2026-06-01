package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

const ngContainerTag = "ng-container"

// GenerateNgContainerOps replaces an Element or ElementStart whose tag is ng-container
// with a Container/ContainerStart op.
func GenerateNgContainerOps(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		updatedElementXrefs := map[ir.XrefId]bool{}

		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() == ir.OpKindElementStart {
				if el, ok := op.(*ir.ElementStartOp); ok {
					tag := el.Tag
					if tag != nil && *tag == ngContainerTag {
						newOp := &ir.ContainerStartOp{ElementOrContainerOpBase: el.ElementOrContainerOpBase}
						ir.OpListInsertBefore(unit.GetCreate(), newOp, op)
						unit.GetCreate().Remove(op)
						updatedElementXrefs[el.Xref] = true
					}
				}
			}
			if op.Kind() == ir.OpKindElement {
				if el, ok := op.(*ir.ElementOp); ok {
					tag := el.Tag
					if tag != nil && *tag == ngContainerTag {
						newOp := &ir.ContainerOp{ElementOrContainerOpBase: el.ElementOrContainerOpBase}
						ir.OpListInsertBefore(unit.GetCreate(), newOp, op)
						unit.GetCreate().Remove(op)
						updatedElementXrefs[el.Xref] = true
					}
				}
			}
			if op.Kind() == ir.OpKindElementEnd {
				if end, ok := op.(*ir.ElementEndOp); ok && updatedElementXrefs[end.Xref] {
					newOp := &ir.ContainerEndOp{Xref: end.Xref, SourceSpan: end.SourceSpan}
					ir.OpListInsertBefore(unit.GetCreate(), newOp, op)
					unit.GetCreate().Remove(op)
				}
			}
		}
	}
}

// DisableBindings emits disableBindings and enableBindings instructions for ng-non-bindable
// containers and all their descendants.
func DisableBindings(job compilation.CompilationJob) {
	elements := map[ir.XrefId]ir.Op{}
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if ir.IsElementOrContainerOp(op) {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					elements[xg.GetXref()] = op
				}
			}
		}
	}

	for _, unit := range job.GetUnits() {
		var newOps []ir.Op
		for _, op := range unit.GetCreate().Elements() {
			nonBindable := false
			if nb, ok := op.(interface{ GetNonBindable() bool }); ok {
				nonBindable = nb.GetNonBindable()
			}

			if op.Kind() == ir.OpKindElementEnd || op.Kind() == ir.OpKindContainerEnd {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					if elem, exists := elements[xg.GetXref()]; exists {
						if nb, ok2 := elem.(interface{ GetNonBindable() bool }); ok2 && nb.GetNonBindable() {
							newOps = append(newOps, ir.CreateEnableBindingsOp(xg.GetXref()))
						}
					}
				}
			}

			newOps = append(newOps, op)

			if (op.Kind() == ir.OpKindElementStart || op.Kind() == ir.OpKindContainerStart) && nonBindable {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					newOps = append(newOps, ir.CreateDisableBindingsOp(xg.GetXref()))
				}
			}
		}
		unit.GetCreate().Ops = newOps
	}
}
