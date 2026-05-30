package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// EmitNamespaceChanges detects transitions between HTML, SVG, and MathML namespaces, emitting corresponding NamespaceOp constructs.
func EmitNamespaceChanges(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		activeNamespace := ir.NamespaceHTML
		var newOps []ir.Op

		for _, op := range unit.GetCreate().Elements() {
			elementStart, ok := op.(*ir.ElementStartOp)
			if !ok {
				newOps = append(newOps, op)
				continue
			}

			ns := ir.NamespaceHTML
			if elementStart.Namespace != nil {
				if castedNs, ok := elementStart.Namespace.(ir.Namespace); ok {
					ns = castedNs
				}
			}

			if ns != activeNamespace {
				namespaceOp := &ir.NamespaceOp{Active: ns}
				newOps = append(newOps, namespaceOp)
				activeNamespace = ns
			}
			newOps = append(newOps, op)
		}
		unit.GetCreate().Ops = newOps
	}
}
