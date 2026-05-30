package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// RemoveEmptyBindings deletes any binding operation targeting an ir.EmptyExpr.
func RemoveEmptyBindings(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		var newOps []ir.Op
		for _, op := range unit.GetUpdate().Elements() {
			keep := true
			switch op.Kind() {
			case ir.OpKindAttribute:
				attrOp := op.(*ir.AttributeOp)
				if _, ok := attrOp.Expression.(*ir.EmptyExpr); ok {
					keep = false
				}
			case ir.OpKindBinding:
				bindingOp := op.(*ir.BindingOp)
				if _, ok := bindingOp.Expression.(*ir.EmptyExpr); ok {
					keep = false
				}
			case ir.OpKindClassProp:
				classPropOp := op.(*ir.ClassPropOp)
				if _, ok := classPropOp.Expression.(*ir.EmptyExpr); ok {
					keep = false
				}
			case ir.OpKindClassMap:
				classMapOp := op.(*ir.ClassMapOp)
				if _, ok := classMapOp.Expression.(*ir.EmptyExpr); ok {
					keep = false
				}
			case ir.OpKindProperty:
				propertyOp := op.(*ir.PropertyOp)
				if _, ok := propertyOp.Expression.(*ir.EmptyExpr); ok {
					keep = false
				}
			case ir.OpKindStyleProp:
				stylePropOp := op.(*ir.StylePropOp)
				if _, ok := stylePropOp.Expression.(*ir.EmptyExpr); ok {
					keep = false
				}
			case ir.OpKindStyleMap:
				styleMapOp := op.(*ir.StyleMapOp)
				if _, ok := styleMapOp.Expression.(*ir.EmptyExpr); ok {
					keep = false
				}
			}
			if keep {
				newOps = append(newOps, op)
			}
		}
		unit.GetUpdate().Ops = newOps
	}
}
