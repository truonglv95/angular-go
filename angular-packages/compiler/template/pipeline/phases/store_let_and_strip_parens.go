package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// OptimizeStoreLet removes any storeLet calls that aren't referenced outside of the current view.
func OptimizeStoreLet(job compilation.CompilationJob) {
	letUsedExternally := map[ir.XrefId]bool{}
	declareLetOps := map[ir.XrefId]ir.Op{}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			if op.Kind() == ir.OpKindDeclareLet {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					declareLetOps[xg.GetXref()] = op
				}
			}
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				if clr, ok := expr.(*ir.ContextLetReferenceExpr); ok {
					letUsedExternally[clr.Target] = true
				}
			})
		}
	}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetUpdate().Elements() {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				if sl, ok := expr.(*ir.StoreLetExpr); ok && !letUsedExternally[sl.Target] {
					if !storeLetHasPipe(sl) {
						if declOp, exists := declareLetOps[sl.Target]; exists {
							unit.GetCreate().Remove(declOp)
						}
					}
					return sl.Value
				}
				return expr
			}, ir.VisitorContextFlagNone)
		}
	}
}

func storeLetHasPipe(root *ir.StoreLetExpr) bool {
	result := false
	ir.TransformExpressionsInExpression(root, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
		switch expr.(type) {
		case *ir.PipeBindingExpr, *ir.PipeBindingVariadicExpr:
			result = true
		}
		return expr
	}, ir.VisitorContextFlagNone)
	return result
}

// StripNonrequiredParentheses removes user-added parentheses that are not required by JS semantics.
func StripNonrequiredParentheses(job compilation.CompilationJob) {
	requiredParens := map[*output.ParenthesizedExpr]bool{}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				if binExpr, ok := expr.(*output.BinaryOperatorExpr); ok {
					switch binExpr.Operator {
					case output.BinaryOperatorExponentiation:
						checkExponentiationParens(binExpr, requiredParens)
					case output.BinaryOperatorNullishCoalesce:
						checkNullishCoalescingParens(binExpr, requiredParens)
					case output.BinaryOperatorAnd, output.BinaryOperatorOr:
						checkAndOrParens(binExpr, requiredParens)
					}
				}
				return expr
			}, ir.VisitorContextFlagNone)
		}
	}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				if paren, ok := expr.(*output.ParenthesizedExpr); ok {
					if requiredParens[paren] {
						return paren
					}
					return paren.Expr
				}
				return expr
			}, ir.VisitorContextFlagNone)
		}
	}
}

func checkExponentiationParens(expr *output.BinaryOperatorExpr, required map[*output.ParenthesizedExpr]bool) {
	if lhsParen, ok := expr.Lhs.(*output.ParenthesizedExpr); ok {
		if _, isUnary := lhsParen.Expr.(*output.UnaryOperatorExpr); isUnary {
			required[lhsParen] = true
		}
	}
}

func checkNullishCoalescingParens(expr *output.BinaryOperatorExpr, required map[*output.ParenthesizedExpr]bool) {
	if lhsParen, ok := expr.Lhs.(*output.ParenthesizedExpr); ok {
		if isLogicalAndOrExpr(lhsParen.Expr) {
			required[lhsParen] = true
		} else if _, isCond := lhsParen.Expr.(*output.ConditionalExpr); isCond {
			required[lhsParen] = true
		}
	}
	if rhsParen, ok := expr.Rhs.(*output.ParenthesizedExpr); ok {
		if isLogicalAndOrExpr(rhsParen.Expr) {
			required[rhsParen] = true
		} else if _, isCond := rhsParen.Expr.(*output.ConditionalExpr); isCond {
			required[rhsParen] = true
		}
	}
}

func checkAndOrParens(expr *output.BinaryOperatorExpr, required map[*output.ParenthesizedExpr]bool) {
	if lhsParen, ok := expr.Lhs.(*output.ParenthesizedExpr); ok {
		if binExpr, ok2 := lhsParen.Expr.(*output.BinaryOperatorExpr); ok2 {
			if binExpr.Operator == output.BinaryOperatorNullishCoalesce {
				required[lhsParen] = true
			}
		}
	}
}

func isLogicalAndOrExpr(expr output.Expression) bool {
	if binExpr, ok := expr.(*output.BinaryOperatorExpr); ok {
		return binExpr.Operator == output.BinaryOperatorAnd || binExpr.Operator == output.BinaryOperatorOr
	}
	return false
}
