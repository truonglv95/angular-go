package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// MergeNextContextExpressions merges logically sequential NextContextExpr operations.
func MergeNextContextExpressions(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, fnExpr := range unit.GetFunctions() {
			if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok {
				mergeNextContextsInOps(arrowFn.Ops)
			}
		}
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindListener, ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindTwoWayListener:
				if lOp, ok := op.(ir.ListenerTrait); ok {
					mergeNextContextsInOps(lOp.GetHandlerOps())
				}
			}
		}
		mergeNextContextsInOps(unit.GetUpdate())
	}
}

func mergeNextContextsInOps(ops *ir.OpList) {
	for _, op := range ops.Elements() {
		if op.Kind() != ir.OpKindStatement {
			continue
		}
		stmtOp, ok := op.(interface{ GetStatement() output.Statement })
		if !ok {
			continue
		}
		exprStmt, ok := stmtOp.GetStatement().(*output.ExpressionStatement)
		if !ok {
			continue
		}
		nextCtx, ok := exprStmt.Expr.(*ir.NextContextExpr)
		if !ok {
			continue
		}
		mergeSteps := nextCtx.Steps

		tryToMerge := true
		candidates := ops.Elements()
		start := -1
		for i, candidate := range candidates {
			if candidate == op {
				start = i + 1
				break
			}
		}
		if start < 0 {
			continue
		}
		for _, candidate := range candidates[start:] {
			if !tryToMerge {
				break
			}
			ir.TransformExpressionsInOp(candidate, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				irExpr, ok := expr.(ir.Expression)
				if !ok {
					return expr
				}
				if !tryToMerge {
					return expr
				}
				if flags&ir.VisitorContextFlagInChildOperation != 0 {
					return expr
				}
				switch e := irExpr.(type) {
				case *ir.NextContextExpr:
					e.Steps += mergeSteps
					ops.Remove(op)
					tryToMerge = false
				case *ir.GetCurrentViewExpr:
					tryToMerge = false
				case *ir.ReferenceExpr:
					tryToMerge = false
				case *ir.ContextLetReferenceExpr:
					tryToMerge = false
				}
				return expr
			}, ir.VisitorContextFlagNone)
		}
	}
}
