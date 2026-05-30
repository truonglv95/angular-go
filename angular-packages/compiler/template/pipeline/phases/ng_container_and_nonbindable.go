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
				if tagged, ok := op.(interface{ GetTag() *string }); ok {
					tag := tagged.GetTag()
					if tag != nil && *tag == ngContainerTag {
						op.(interface{ SetKind(ir.OpKind) }).SetKind(ir.OpKindContainerStart)
						if xg, ok2 := op.(interface{ GetXref() ir.XrefId }); ok2 {
							updatedElementXrefs[xg.GetXref()] = true
						}
					}
				}
			}
			if op.Kind() == ir.OpKindElementEnd {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok && updatedElementXrefs[xg.GetXref()] {
					op.(interface{ SetKind(ir.OpKind) }).SetKind(ir.OpKindContainerEnd)
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
		for _, op := range unit.GetCreate().Elements() {
			nonBindable := false
			if nb, ok := op.(interface{ GetNonBindable() bool }); ok {
				nonBindable = nb.GetNonBindable()
			}

			if (op.Kind() == ir.OpKindElementStart || op.Kind() == ir.OpKindContainerStart) && nonBindable {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					ir.InsertAfter(ir.CreateDisableBindingsOp(xg.GetXref()), op)
				}
			}
			if op.Kind() == ir.OpKindElementEnd || op.Kind() == ir.OpKindContainerEnd {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					if elem, exists := elements[xg.GetXref()]; exists {
						if nb, ok2 := elem.(interface{ GetNonBindable() bool }); ok2 && nb.GetNonBindable() {
							ir.InsertBefore(ir.CreateEnableBindingsOp(xg.GetXref()), op)
						}
					}
				}
			}
		}
	}
}
