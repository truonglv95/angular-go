package phases

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// RemoveContentSelectors removes ng-content attribute select bindings.
func RemoveContentSelectors(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		elements := createOpXrefMap(unit)
		var newOps []ir.Op
		for _, op := range unit.GetUpdate().Elements() {
			bindingOp, ok := op.(*ir.BindingOp)
			if !ok {
				newOps = append(newOps, op)
				continue
			}
			target, exists := elements[bindingOp.Target]
			if exists && target.Kind() == ir.OpKindProjection && strings.ToLower(bindingOp.Name) == "select" {
				continue
			}
			newOps = append(newOps, op)
		}
		unit.GetUpdate().Ops = newOps
	}
}
