package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// SpecializeStyleBindings elevates plain generic class/style BindingOp structures to specific high-performance update operations.
func SpecializeStyleBindings(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		var newOps []ir.Op
		for _, op := range unit.GetUpdate().Elements() {
			bindingOp, ok := op.(*ir.BindingOp)
			if !ok {
				newOps = append(newOps, op)
				continue
			}

			bindingKind, ok := bindingOp.BindingKind.(ir.BindingKind)
			if !ok {
				newOps = append(newOps, op)
				continue
			}

			var finalOp ir.Op = op
			switch bindingKind {
			case ir.BindingKindClassName:
				if _, isInterpolation := bindingOp.Expression.(*ir.Interpolation); isInterpolation {
					panic("Unexpected interpolation in ClassName binding")
				}
				finalOp = &ir.ClassPropOp{
					Target:     bindingOp.Target,
					Name:       bindingOp.Name,
					Expression: bindingOp.Expression,
					SourceSpan: bindingOp.SourceSpan,
				}

			case ir.BindingKindStyleProperty:
				finalOp = &ir.StylePropOp{
					Target:     bindingOp.Target,
					Name:       bindingOp.Name,
					Expression: bindingOp.Expression,
					Unit:       bindingOp.Unit,
					SourceSpan: bindingOp.SourceSpan,
				}

			case ir.BindingKindProperty, ir.BindingKindTemplate:
				if bindingOp.Name == "style" {
					finalOp = &ir.StyleMapOp{
						Target:     bindingOp.Target,
						Expression: bindingOp.Expression,
						SourceSpan: bindingOp.SourceSpan,
					}
				} else if bindingOp.Name == "class" {
					finalOp = &ir.ClassMapOp{
						Target:     bindingOp.Target,
						Expression: bindingOp.Expression,
						SourceSpan: bindingOp.SourceSpan,
					}
				}
			}
			newOps = append(newOps, finalOp)
		}
		unit.GetUpdate().Ops = newOps
	}
}
