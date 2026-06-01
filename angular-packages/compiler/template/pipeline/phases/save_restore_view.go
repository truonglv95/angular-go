package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func SaveAndRestoreView(job *compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, fnExpr := range unit.GetFunctions() {
			arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr)
			if !ok {
				continue
			}
			if needsRestoreView(job, unit, arrowFn.Ops) {
				restoreViewTarget := output.NewReadVarExpr(arrowFn.CurrentViewName, nil, nil, nil)
				addSaveRestoreViewOperation(unit, arrowFn.Ops, restoreViewTarget)
			}
		}

		savedViewVar := ir.SemanticVariable{
			Kind: ir.SemanticVariableKindSavedView,
			View: unit.GetXref(),
		}

		unit.GetCreate().Prepend([]ir.Op{
			ir.NewCreateVariableOp(
				unit.GetJob().AllocateXrefId(),
				savedViewVar,
				ir.NewGetCurrentViewExpr(),
				ir.VariableFlagsNone,
			).(ir.Op),
		})

		for _, op := range unit.GetCreate().Ops {
			var handlerOps *ir.OpList
			switch o := op.(type) {
			case *ir.ListenerOp:
				if h, ok := o.HandlerOps.(*ir.OpList); ok {
					handlerOps = h
				}
			case *ir.TwoWayListenerOp:
				handlerOps = o.HandlerOps
			case *ir.AnimationOp:
				if h, ok := o.HandlerOps.(*ir.OpList); ok {
					handlerOps = h
				}
			case *ir.AnimationListenerOp:
				if h, ok := o.HandlerOps.(*ir.OpList); ok {
					handlerOps = h
				}
			}

			if handlerOps != nil {
				if needsRestoreView(job, unit, handlerOps) {
					addSaveRestoreViewOperation(unit, handlerOps, unit.GetXref())
				}
			}
		}
	}
}

func needsRestoreView(
	job *compilation.ComponentCompilationJob,
	unit compilation.CompilationUnit,
	opList *ir.OpList,
) bool {
	result := unit.GetXref() != job.Root.GetXref()

	if !result {
		for _, innerOp := range opList.Ops {
			ir.VisitExpressionsInOp(innerOp, func(expr ir.Expression) {
				if _, ok := expr.(*ir.ReferenceExpr); ok {
					result = true
				} else if _, ok := expr.(*ir.ContextLetReferenceExpr); ok {
					result = true
				}
			})
		}
	}

	return result
}

func addSaveRestoreViewOperation(
	unit compilation.CompilationUnit,
	opList *ir.OpList,
	restoreViewTarget any,
) {
	var restoreViewExpr *ir.RestoreViewExpr
	if xref, ok := restoreViewTarget.(ir.XrefId); ok {
		restoreViewExpr = ir.NewRestoreViewExpr(xref)
	} else if expr, ok := restoreViewTarget.(output.Expression); ok {
		restoreViewExpr = ir.NewRestoreViewExprFromExpr(expr)
	}

	semanticVar := ir.SemanticVariable{
		Kind: ir.SemanticVariableKindContext,
		View: unit.GetXref(),
	}

	opList.Prepend([]ir.Op{
		ir.NewVariableOp(
			unit.GetJob().AllocateXrefId(),
			semanticVar,
			restoreViewExpr,
			ir.VariableFlagsNone,
		).(ir.Op),
	})

	for _, handlerOp := range opList.Ops {
		if stmtOp, ok := handlerOp.(*ir.StatementOp); ok {
			if retStmt, ok := stmtOp.Statement.(*output.ReturnStatement); ok {
				retStmt.Value = ir.NewResetViewExpr(retStmt.Value)
			}
		}
	}
}
