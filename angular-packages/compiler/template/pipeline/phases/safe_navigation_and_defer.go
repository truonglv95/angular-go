package phases

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// RemoveSafeNavigationMigration finds and removes calls to $safeNavigationMigration,
// marking the argument to use legacy null-returning safe navigation semantics.
func RemoveSafeNavigationMigration(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				return convertSafeNavMigrationCall(expr)
			}, ir.VisitorContextFlagNone)
		}
	}
}

func convertSafeNavMigrationCall(e output.Expression) output.Expression {
	invoke, ok := e.(*output.InvokeFunctionExpr)
	if !ok {
		return e
	}
	lex, ok := invoke.Fn.(*ir.LexicalReadExpr)
	if !ok || lex.Name != "$safeNavigationMigration" {
		return e
	}
	if len(invoke.Args) != 1 {
		panic("The $safeNavigationMigration builtin function expects exactly one argument.")
	}
	snm := &ir.SafeNavigationMigrationExpr{Expr: invoke.Args[0]}
	snm.Self = snm
	return snm
}

// ResolveDeferDepsFns resolves the dependency function of a deferred block.
func ResolveDeferDepsFns(job *compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindDefer {
				continue
			}
			deferOp, ok := op.(*ir.DeferOp)
			if !ok {
				continue
			}
			if deferOp.ResolverFn != nil {
				// Handle typed nil
				if rve, ok := deferOp.ResolverFn.(*output.ReadVarExpr); !ok || rve != nil {
					continue
				}
			}
			if deferOp.OwnResolverFn == nil {
				continue
			}
			if rve, ok := deferOp.OwnResolverFn.(*output.ReadVarExpr); ok && rve == nil {
				continue
			}
			var slot int
			sh := deferOp.Handle()
			if sh != nil && sh.Slot != nil {
				slot = *sh.Slot
			} else {
				panic("AssertionError: slot must be assigned before extracting defer deps functions")
			}
			fnName := ""
			if unit.GetFnName() != nil {
				fnName = *unit.GetFnName()
			}
			// Remove _Template suffix.
			fullPathName := fnName
			const templateSuffix = "_Template"
			if len(fullPathName) > len(templateSuffix) && fullPathName[len(fullPathName)-len(templateSuffix):] == templateSuffix {
				fullPathName = fullPathName[:len(fullPathName)-len(templateSuffix)]
			}
			if pool, ok2 := job.GetPool().(interface {
				GetSharedFunctionReference(fn output.Expression, name string, unique bool) output.Expression
			}); ok2 {
				depsFnName := fmt.Sprintf("%s_Defer_%d_DepsFn", fullPathName, slot)

				var reify func(expr output.Expression) output.Expression
				reify = func(expr output.Expression) output.Expression {
					return ir.TransformExpressionsInExpression(expr, func(e output.Expression, flags ir.VisitorContextFlag) output.Expression {
						if irArrow, ok := e.(*ir.ArrowFunctionExpr); ok {
							var params []*output.FnParam
							for i := range irArrow.Parameters {
								params = append(params, &irArrow.Parameters[i])
							}
							body := reify(irArrow.Body)
							return output.NewArrowFunctionExpr(params, body, nil, nil, nil)
						}
						return e
					}, ir.VisitorContextFlagNone)
				}

				resolvedFn := reify(deferOp.OwnResolverFn)

				deferOp.ResolverFn = pool.GetSharedFunctionReference(resolvedFn, depsFnName, false)
			}
		}
	}
}
