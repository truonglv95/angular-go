package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CountVariables counts the number of variable slots used within each view, stores that on the
// view itself, and propagates it to the ir.TemplateOp for embedded views.
func CountVariables(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		varCount := 0

		// Count variables on top-level ops first.
		for _, op := range unit.GetCreate().Elements() {
			if cv, ok := op.(ir.ConsumesVarsTrait); ok && cv.ConsumesVars() {
				varCount += varsUsedByOp(op)
			}
		}
		for _, op := range unit.GetUpdate().Elements() {
			if cv, ok := op.(ir.ConsumesVarsTrait); ok && cv.ConsumesVars() {
				varCount += varsUsedByOp(op)
			}
		}

		// First pass: count non-pure-function expressions.
		for _, op := range unit.GetCreate().Elements() {
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				if !ir.IsIrExpression(expr) {
					return
				}
				if _, isPure := expr.(*ir.PureFunctionExpr); isPure {
					return
				}
				if uvo, ok := expr.(ir.UsesVarOffsetTrait); ok {
					uvo.SetVarOffset(varCount)
				}
				if cv, ok := expr.(ir.ConsumesVarsTrait); ok && cv.ConsumesVars() {
					varCount += varsUsedByIrExpression(expr)
				}
			})
		}
		for _, op := range unit.GetUpdate().Elements() {
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				if !ir.IsIrExpression(expr) {
					return
				}
				if _, isPure := expr.(*ir.PureFunctionExpr); isPure {
					return
				}
				if uvo, ok := expr.(ir.UsesVarOffsetTrait); ok {
					uvo.SetVarOffset(varCount)
				}
				if cv, ok := expr.(ir.ConsumesVarsTrait); ok && cv.ConsumesVars() {
					varCount += varsUsedByIrExpression(expr)
				}
			})
		}

		// Second pass: pure function expressions only.
		for _, op := range unit.GetCreate().Elements() {
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				if !ir.IsIrExpression(expr) {
					return
				}
				if _, isPure := expr.(*ir.PureFunctionExpr); !isPure {
					return
				}
				if uvo, ok := expr.(ir.UsesVarOffsetTrait); ok {
					uvo.SetVarOffset(varCount)
				}
				if cv, ok := expr.(ir.ConsumesVarsTrait); ok && cv.ConsumesVars() {
					varCount += varsUsedByIrExpression(expr)
				}
			})
		}
		for _, op := range unit.GetUpdate().Elements() {
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				if !ir.IsIrExpression(expr) {
					return
				}
				if _, isPure := expr.(*ir.PureFunctionExpr); !isPure {
					return
				}
				if uvo, ok := expr.(ir.UsesVarOffsetTrait); ok {
					uvo.SetVarOffset(varCount)
				}
				if cv, ok := expr.(ir.ConsumesVarsTrait); ok && cv.ConsumesVars() {
					varCount += varsUsedByIrExpression(expr)
				}
			})
		}

		varCountCopy := varCount
		unit.SetVars(&varCountCopy)
	}

	if cj, ok := job.(*compilation.ComponentCompilationJob); ok {
		for _, view := range cj.Views {
			for _, op := range view.GetCreate().Elements() {
				switch op.Kind() {
				case ir.OpKindTemplate, ir.OpKindRepeaterCreate, ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate:
					if xg, ok2 := op.(interface{ GetXref() ir.XrefId }); ok2 {
						if childView, exists := cj.Views[xg.GetXref()]; exists && childView.GetVars() != nil {
							if setter, ok3 := op.(interface{ SetVars(int) }); ok3 {
								setter.SetVars(*childView.GetVars())
							}
						}
					}
				}
			}
		}
	}
}

func varsUsedByOp(op ir.Op) int {
	switch op.Kind() {
	case ir.OpKindI18nExpression, ir.OpKindConditional, ir.OpKindDeferWhen, ir.OpKindStoreLet:
		return 1
	case ir.OpKindProperty, ir.OpKindDomProperty, ir.OpKindAttribute, ir.OpKindTwoWayProperty:
		var expr any
		if propOp, ok := op.(*ir.PropertyOp); ok {
			expr = propOp.Expression
		} else if attrOp, ok := op.(*ir.AttributeOp); ok {
			expr = attrOp.Expression
		} else if twoWayOp, ok := op.(*ir.TwoWayPropertyOp); ok {
			expr = twoWayOp.Expression
		}
		if interp, ok := expr.(*ir.Interpolation); ok {
			return 1 + len(interp.Expressions)
		}
		return 1
	case ir.OpKindStyleProp, ir.OpKindClassProp, ir.OpKindStyleMap, ir.OpKindClassMap:
		slots := 2
		var expr any
		if sp, ok := op.(*ir.StylePropOp); ok {
			expr = sp.Expression
		}
		if cp, ok := op.(*ir.ClassPropOp); ok {
			expr = cp.Expression
		}
		if sm, ok := op.(*ir.StyleMapOp); ok {
			expr = sm.Expression
		}
		if cm, ok := op.(*ir.ClassMapOp); ok {
			expr = cm.Expression
		}
		if interp, ok := expr.(*ir.Interpolation); ok {
			slots += len(interp.Expressions)
		}
		return slots
	case ir.OpKindInterpolateText:
		if interp, ok := op.(*ir.InterpolateTextOp); ok {
			if inner, ok2 := interp.Interpolation.(*ir.Interpolation); ok2 {
				return len(inner.Expressions)
			}
		}
		return 1
	case ir.OpKindRepeaterCreate:
		if rop, ok := op.(*ir.RepeaterCreateOp); ok && rop.EmptyView != 0 {
			return 1
		}
		return 0
	}
	return 0
}

func varsUsedByIrExpression(expr ir.Expression) int {
	switch e := expr.(type) {
	case *ir.PureFunctionExpr:
		return 1 + len(e.Args)
	case *ir.PipeBindingExpr:
		return 1 + len(e.Args)
	case *ir.PipeBindingVariadicExpr:
		return 1 + e.NumArgs
	case *ir.StoreLetExpr:
		_ = e
		return 1
	case *ir.ArrowFunctionExpr:
		_ = e
		return 1
	default:
		return 0
	}
}
