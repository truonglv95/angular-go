package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CollapseSingletonInterpolations collapses interpolation expressions with all empty strings (Strings length is 2 and all are "").
func CollapseSingletonInterpolations(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetUpdate().Elements() {
			switch op.Kind() {
			case ir.OpKindAttribute:
				attrOp := op.(*ir.AttributeOp)
				if interp, ok := attrOp.Expression.(*ir.Interpolation); ok {
					if len(interp.Strings) == 2 && allStringsEmpty(interp.Strings) {
						attrOp.Expression = interp.Expressions[0]
					}
				}
			case ir.OpKindStyleProp:
				stylePropOp := op.(*ir.StylePropOp)
				if interp, ok := stylePropOp.Expression.(*ir.Interpolation); ok {
					if len(interp.Strings) == 2 && allStringsEmpty(interp.Strings) {
						stylePropOp.Expression = interp.Expressions[0]
					}
				}
			case ir.OpKindStyleMap:
				styleMapOp := op.(*ir.StyleMapOp)
				if interp, ok := styleMapOp.Expression.(*ir.Interpolation); ok {
					if len(interp.Strings) == 2 && allStringsEmpty(interp.Strings) {
						styleMapOp.Expression = interp.Expressions[0]
					}
				}
			case ir.OpKindClassMap:
				classMapOp := op.(*ir.ClassMapOp)
				if interp, ok := classMapOp.Expression.(*ir.Interpolation); ok {
					if len(interp.Strings) == 2 && allStringsEmpty(interp.Strings) {
						classMapOp.Expression = interp.Expressions[0]
					}
				}
			}
		}
	}
}

func allStringsEmpty(strs []string) bool {
	for _, s := range strs {
		if s != "" {
			return false
		}
	}
	return true
}
