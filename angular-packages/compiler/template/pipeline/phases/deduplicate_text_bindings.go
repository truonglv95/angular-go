package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// DeduplicateTextBindings discards duplicate style or class bindings, prioritizing the final specified bindings.
func DeduplicateTextBindings(job compilation.CompilationJob) {
	seen := make(map[ir.XrefId]map[string]bool)
	for _, unit := range job.GetUnits() {
		ops := unit.GetUpdate().Elements()
		removeSet := make(map[ir.Op]bool)

		for i := len(ops) - 1; i >= 0; i-- {
			op := ops[i]
			bindingOp, ok := op.(*ir.BindingOp)
			if !ok {
				continue
			}

			if bindingOp.IsTextAttribute {
				seenForElement := seen[bindingOp.Target]
				if seenForElement == nil {
					seenForElement = make(map[string]bool)
					seen[bindingOp.Target] = seenForElement
				}

				if seenForElement[bindingOp.Name] && (bindingOp.Name == "style" || bindingOp.Name == "class") {
					removeSet[op] = true
				}
				seenForElement[bindingOp.Name] = true
			}
		}

		var newOps []ir.Op
		for _, op := range ops {
			if !removeSet[op] {
				newOps = append(newOps, op)
			}
		}
		unit.GetUpdate().Ops = newOps
	}
}
