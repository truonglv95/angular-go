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
		candidate := op.Next()
		for candidate != nil && candidate.Kind() != ir.OpKindListEnd && tryToMerge {
			// Visit expressions in the candidate op (without flags - simplified).
			// We check the expressions manually to avoid in-child operation visits.
			ir.VisitExpressionsInOp(candidate, func(expr ir.Expression) {
				if !tryToMerge {
					return
				}
				switch e := expr.(type) {
				case *ir.NextContextExpr:
					e.Steps += mergeSteps
					ops.Remove(op)
					tryToMerge = false
				case *ir.GetCurrentViewExpr:
					_ = e
					tryToMerge = false
				case *ir.ReferenceExpr:
					_ = e
					tryToMerge = false
				case *ir.ContextLetReferenceExpr:
					_ = e
					tryToMerge = false
				}
			})
			if tryToMerge {
				candidate = candidate.Next()
			}
		}
	}
}
