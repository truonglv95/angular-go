package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// Fence is a bitfield indicating optimization constraints for expressions.
type Fence int

const (
	FenceNone            Fence = 0b000
	FenceViewContextRead  Fence = 0b001
	FenceViewContextWrite Fence = 0b010
	FenceSideEffectful    Fence = 0b100
)

const contextName = "ctx"

// OptimizeVariables optimizes variable declarations in the IR:
// - removes unused variables with no side effects
// - transforms unused variables with side effects to expression statements
// - inlines variables that are used exactly once when semantically safe
func OptimizeVariables(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// Process arrow function bodies.
		for _, fnExpr := range unit.GetFunctions() {
			if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok {
				inlineAlwaysInlineVariables(arrowFn.Ops)
			}
		}
		inlineAlwaysInlineVariables(unit.GetCreate())
		inlineAlwaysInlineVariables(unit.GetUpdate())

		// Inline in listener handler ops.
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindListener, ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindTwoWayListener:
				if lOp, ok := op.(ir.ListenerTrait); ok {
					inlineAlwaysInlineVariables(lOp.HandlerOps())
				}
			case ir.OpKindRepeaterCreate:
				if r, ok := op.(*ir.RepeaterCreateOp); ok {
					if opList, ok2 := r.TrackByOps.(*ir.OpList); ok2 {
						inlineAlwaysInlineVariables(opList)
					}
				}
			}
		}

		// Optimize variables in functions.
		for _, fnExpr := range unit.GetFunctions() {
			if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok {
				optimizeVariablesInOpList(arrowFn.Ops, nil)
				optimizeSaveRestoreView(arrowFn.Ops)
			}
		}

		// Optimize in listener ops.
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindListener, ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindTwoWayListener:
				if lOp, ok := op.(ir.ListenerTrait); ok {
					optimizeVariablesInOpList(lOp.HandlerOps(), skipArrowFnOps)
					optimizeSaveRestoreView(lOp.HandlerOps())
				}
			case ir.OpKindRepeaterCreate:
				if r, ok := op.(*ir.RepeaterCreateOp); ok {
					if opList, ok2 := r.TrackByOps.(*ir.OpList); ok2 {
						optimizeVariablesInOpList(opList, skipArrowFnOps)
					}
				}
			}
		}

		optimizeVariablesInOpList(unit.GetCreate(), skipArrowFnOps)
		optimizeVariablesInOpList(unit.GetUpdate(), skipArrowFnOps)
	}
}

// skipArrowFnOps is a predicate that skips expressions inside arrow function operations.
func skipArrowFnOps(flags ir.VisitorContextFlag) bool {
	return flags&ir.VisitorContextFlagInArrowFunction == 0
}

// inlineAlwaysInlineVariables inlines variables marked with AlwaysInline.
func inlineAlwaysInlineVariables(ops *ir.OpList) {
	vars := map[ir.XrefId]*ir.VariableOp{}
	for _, op := range ops.Elements() {
		varOp, ok := op.(*ir.VariableOp)
		if !ok {
			continue
		}
		if varOp.Flags&ir.VariableFlagsAlwaysInline != 0 {
			// Validate no fence-sensitive expressions.
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				if ir.IsIrExpression(expr) && fencesForIrExpr(expr) != FenceNone {
					panic("AssertionError: A context-sensitive variable was marked AlwaysInline")
				}
			})
			vars[varOp.Xref] = varOp
		}
	}

	if len(vars) == 0 {
		return
	}

	// Replace ReadVariableExpr with the initializer.
	for _, op := range ops.Elements() {
		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			rv, ok := expr.(*ir.ReadVariableExpr)
			if !ok {
				return expr
			}
			if varOp, exists := vars[rv.Xref]; exists {
				return varOp.Initializer
			}
			return expr
		}, ir.VisitorContextFlagNone)
	}

	// Remove the inlined variable ops.
	for _, varOp := range vars {
		ops.Remove(varOp)
	}
}

// optimizeVariablesInOpList performs full variable optimization on a list of ops.
func optimizeVariablesInOpList(ops *ir.OpList, predicate func(ir.VisitorContextFlag) bool) {
	// Collect all variable declarations.
	varOps := map[ir.XrefId]*ir.VariableOp{}
	varFences := map[ir.XrefId]Fence{}
	varUsages := map[ir.XrefId]int{}
	varRemoteUsage := map[ir.XrefId]bool{}

	elements := ops.Elements()
	for _, op := range elements {
		if varOp, ok := op.(*ir.VariableOp); ok {
			if varOp.Flags&ir.VariableFlagsAlwaysInline != 0 {
				continue
			}
			varOps[varOp.Xref] = varOp
			varFences[varOp.Xref] = collectFences(varOp.Initializer)
			varUsages[varOp.Xref] = 0
		}
	}

	// Count usages.
	for _, op := range elements {
		ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
			if predicate != nil && !ir.IsIrExpression(expr) {
				return
			}
			rv, ok := expr.(*ir.ReadVariableExpr)
			if !ok {
				return
			}
			if _, isLocal := varOps[rv.Xref]; !isLocal {
				return
			}
			varUsages[rv.Xref]++
		})
	}

	// Optimize each variable.
	for _, varOp := range varOps {
		xref := varOp.Xref
		usages := varUsages[xref]
		fences := varFences[xref]

		if usages == 0 {
			if fences&FenceSideEffectful != 0 {
				// Keep as expression statement.
				exprStmt := &output.ExpressionStatement{Expr: varOp.Initializer}
				stmtOp := &ir.StatementOp{Statement: exprStmt}
				ir.InsertBefore(stmtOp, varOp)
			}
			ops.Remove(varOp)
		} else if usages == 1 && !varRemoteUsage[xref] {
			// Try inlining.
			if tryInlineVariable(xref, varOp.Initializer, fences, ops, elements, predicate) {
				ops.Remove(varOp)
			}
		}
	}
}

// tryInlineVariable attempts to inline a variable into its single usage site.
func tryInlineVariable(xref ir.XrefId, init output.Expression, declFences Fence, ops *ir.OpList, elements []ir.Op, predicate func(ir.VisitorContextFlag) bool) bool {
	inlined := false
	inliningAllowed := true

	for _, op := range elements {
		if _, ok := op.(*ir.VariableOp); ok && !inlined {
			if vop, ok2 := op.(*ir.VariableOp); ok2 && vop.Xref == xref {
				continue
			}
		}

		if inlined {
			break
		}

		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			if !inliningAllowed || inlined {
				return expr
			}
			if predicate != nil {
				if !predicate(flags) {
					return expr
				}
			}
			if flags&ir.VisitorContextFlagInChildOperation != 0 && declFences&FenceViewContextRead != 0 {
				return expr
			}
			if rv, ok := expr.(*ir.ReadVariableExpr); ok && rv.Xref == xref {
				inlined = true
				return init
			}
			// Check if crossing a fence.
			if ir.IsIrExpression(expr) {
				exprFences := fencesForIrExpr(expr.(ir.Expression))
				inliningAllowed = inliningAllowed && safeToInlinePastFences(exprFences, declFences)
			}
			return expr
		}, ir.VisitorContextFlagNone)

		if !inliningAllowed {
			break
		}
	}
	return inlined
}

// optimizeSaveRestoreView removes unnecessary saveView/restoreView pairs.
func optimizeSaveRestoreView(ops *ir.OpList) {
	elements := ops.Elements()
	if len(elements) < 2 {
		return
	}
	head := elements[0]
	tail := elements[len(elements)-1]

	// Check: head is a restoreView statement, tail is a return with resetView.
	headStmt, ok1 := head.(*ir.StatementOp)
	tailStmt, ok2 := tail.(*ir.StatementOp)
	if !ok1 || !ok2 {
		return
	}

	if len(elements) != 2 {
		return
	}

	headExprStmt, ok := headStmt.Statement.(*output.ExpressionStatement)
	if !ok {
		return
	}
	if _, ok := headExprStmt.Expr.(*ir.RestoreViewExpr); !ok {
		return
	}

	tailRetStmt, ok := tailStmt.Statement.(*output.ReturnStatement)
	if !ok {
		return
	}
	resetView, ok := tailRetStmt.Value.(*ir.ResetViewExpr)
	if !ok {
		return
	}

	// Remove the restoreView call and unwrap resetView.
	ops.Remove(head)
	tailRetStmt.Value = resetView.Expr
}

// fencesForIrExpr returns the Fence flags for an IR expression.
func fencesForIrExpr(expr ir.Expression) Fence {
	switch expr.ExprKind() {
	case ir.ExpressionKindNextContext:
		return FenceViewContextRead | FenceViewContextWrite
	case ir.ExpressionKindRestoreView:
		return FenceViewContextRead | FenceViewContextWrite | FenceSideEffectful
	case ir.ExpressionKindStoreLet:
		return FenceSideEffectful
	case ir.ExpressionKindReference, ir.ExpressionKindContextLetReference:
		return FenceViewContextRead
	default:
		return FenceNone
	}
}

// collectFences returns Fence flags by scanning an expression for IR expressions.
func collectFences(expr output.Expression) Fence {
	f := FenceNone
	if irExpr, ok := expr.(ir.Expression); ok {
		f |= fencesForIrExpr(irExpr)
	}
	return f
}

// safeToInlinePastFences checks if inlining across a fence is safe.
func safeToInlinePastFences(fences, declFences Fence) bool {
	if fences&FenceViewContextWrite != 0 {
		if declFences&FenceViewContextRead != 0 {
			return false
		}
	} else if fences&FenceViewContextRead != 0 {
		if declFences&FenceViewContextWrite != 0 {
			return false
		}
	}
	return true
}
