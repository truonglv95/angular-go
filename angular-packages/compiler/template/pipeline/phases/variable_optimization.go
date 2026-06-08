package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// Fence is a bitfield indicating optimization constraints for expressions.
type Fence int

const (
	FenceNone             Fence = 0b000
	FenceViewContextRead  Fence = 0b001
	FenceViewContextWrite Fence = 0b010
	FenceSideEffectful    Fence = 0b100
)

const contextName = "ctx"

type opInfo struct {
	variablesUsed map[ir.XrefId]bool
	fences        Fence
}

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
					inlineAlwaysInlineVariables(lOp.GetHandlerOps())
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
					optimizeVariablesInOpList(lOp.GetHandlerOps(), skipArrowFnOps)
					optimizeSaveRestoreView(lOp.GetHandlerOps())
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
	vars := map[ir.XrefId]ir.Op{}
	varInits := map[ir.XrefId]output.Expression{}
	for _, op := range ops.Elements() {
		var flags ir.VariableFlags
		var init output.Expression
		var xref ir.XrefId
		if varOp, ok := op.(*ir.VariableOp); ok {
			flags = varOp.Flags
			init = varOp.Initializer
			xref = varOp.Xref
		} else if cvOp, ok := op.(*ir.CreateVariableOp); ok {
			flags = cvOp.Flags
			init = cvOp.Initializer
			xref = cvOp.Xref
		} else {
			continue
		}
		if flags&ir.VariableFlagsAlwaysInline != 0 {
			// Validate no fence-sensitive expressions.
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				if ir.IsIrExpression(expr) && fencesForIrExpr(expr) != FenceNone {
					panic("AssertionError: A context-sensitive variable was marked AlwaysInline")
				}
			})
			vars[xref] = op
			varInits[xref] = init
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
			if init, exists := varInits[rv.Xref]; exists {
				return init
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
	varOps := map[ir.XrefId]ir.Op{}
	varKinds := map[ir.XrefId]ir.SemanticVariableKind{}
	varInits := map[ir.XrefId]output.Expression{}
	varFences := map[ir.XrefId]Fence{}
	varUsages := map[ir.XrefId]int{}
	varRemoteUsage := map[ir.XrefId]bool{}
	opInfos := map[ir.Op]opInfo{}

	elements := make([]ir.Op, len(ops.Elements()))
	copy(elements, ops.Elements())
	for _, op := range elements {
		var flags ir.VariableFlags
		var init output.Expression
		var xref ir.XrefId
		var kind ir.SemanticVariableKind
		if varOp, ok := op.(*ir.VariableOp); ok {
			flags = varOp.Flags
			init = varOp.Initializer
			xref = varOp.Xref
			kind = varOp.Variable.Kind
		} else if cvOp, ok := op.(*ir.CreateVariableOp); ok {
			flags = cvOp.Flags
			init = cvOp.Initializer
			xref = cvOp.Xref
			kind = cvOp.Variable.Kind
		} else {
			continue
		}
		if flags&ir.VariableFlagsAlwaysInline != 0 {
			continue
		}
		varOps[xref] = op
		varKinds[xref] = kind
		varInits[xref] = init
		varFences[xref] = collectFences(init)
		varUsages[xref] = 0
	}
	inlineRestoredContextIntoIdentifierInitializers(ops, varOps, varKinds, varInits, varFences)

	for _, op := range elements {
		opInfos[op] = collectOpInfo(op, predicate)
		countVariableUsages(op, varUsages, varRemoteUsage, predicate)
	}

	contextIsUsed := false
	for i := len(elements) - 1; i >= 0; i-- {
		op := elements[i]
		info := opInfos[op]
		var xref ir.XrefId
		if varOp, ok := op.(*ir.VariableOp); ok {
			xref = varOp.Xref
		} else if cvOp, ok := op.(*ir.CreateVariableOp); ok {
			xref = cvOp.Xref
		}

		if xref != 0 {
			if varOp, ok := varOps[xref]; ok && varUsages[xref] == 0 {
				fences := info.fences
				if (contextIsUsed && fences&FenceViewContextWrite != 0) || fences&FenceSideEffectful != 0 {
					exprStmt := &output.ExpressionStatement{Expr: varInits[xref]}
					stmtOp := &ir.StatementOp{Statement: exprStmt}
					ir.OpListInsertBefore(ops, stmtOp, varOp)
					opInfos[stmtOp] = info
				} else {
					uncountVariableUsages(op, varUsages)
				}
				ops.Remove(varOp)
				delete(opInfos, op)
				delete(varOps, xref)
				delete(varKinds, xref)
				delete(varInits, xref)
				delete(varFences, xref)
				delete(varUsages, xref)
				continue
			}
		}

		if info.fences&FenceViewContextRead != 0 {
			contextIsUsed = true
		}
	}

	// Optimize each remaining variable in op order.
	for _, candidate := range elements {
		var xref ir.XrefId
		if varOp, ok := candidate.(*ir.VariableOp); ok {
			xref = varOp.Xref
		} else if cvOp, ok := candidate.(*ir.CreateVariableOp); ok {
			xref = cvOp.Xref
		} else {
			continue
		}
		varOp, ok := varOps[xref]
		if !ok {
			continue
		}
		usages := varUsages[xref]
		fences := varFences[xref]
		init := varInits[xref]
		kind := varKinds[xref]

		if kind == ir.SemanticVariableKindSavedView {
			continue
		} else if usages == 1 && !varRemoteUsage[xref] {
			// Try inlining. For Context kind variables, only inline into other Variable ops
			// (matching ngtsc's allowConservativeInlining which prevents inlining Context
			// variables into general update ops like repeater, classProp, etc.).
			if tryInlineVariable(xref, init, kind, fences, varOp, ops, opInfos) {
				ops.Remove(varOp)
				delete(varOps, xref)
			}
		}
	}
}

func collectOpInfo(op ir.Op, predicate func(ir.VisitorContextFlag) bool) opInfo {
	fences := FenceNone
	variablesUsed := map[ir.XrefId]bool{}
	ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
		irExpr, ok := expr.(ir.Expression)
		if !ok || (predicate != nil && !predicate(flags)) {
			return expr
		}
		if rv, ok := irExpr.(*ir.ReadVariableExpr); ok {
			variablesUsed[rv.Xref] = true
		} else {
			fences |= fencesForIrExpr(irExpr)
		}
		return expr
	}, ir.VisitorContextFlagNone)
	return opInfo{variablesUsed: variablesUsed, fences: fences}
}

func countVariableUsages(op ir.Op, varUsages map[ir.XrefId]int, varRemoteUsage map[ir.XrefId]bool, predicate func(ir.VisitorContextFlag) bool) {
	ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
		irExpr, ok := expr.(ir.Expression)
		if !ok || (predicate != nil && !predicate(flags)) {
			return expr
		}
		rv, ok := irExpr.(*ir.ReadVariableExpr)
		if !ok {
			return expr
		}
		if _, exists := varUsages[rv.Xref]; !exists {
			return expr
		}
		varUsages[rv.Xref]++
		if flags&ir.VisitorContextFlagInChildOperation != 0 {
			varRemoteUsage[rv.Xref] = true
		}
		return expr
	}, ir.VisitorContextFlagNone)
}

func uncountVariableUsages(op ir.Op, varUsages map[ir.XrefId]int) {
	ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
		rv, ok := expr.(*ir.ReadVariableExpr)
		if !ok {
			return
		}
		if count, exists := varUsages[rv.Xref]; exists && count > 0 {
			varUsages[rv.Xref] = count - 1
		}
	})
}

func inlineRestoredContextIntoIdentifierInitializers(
	ops *ir.OpList,
	varOps map[ir.XrefId]ir.Op,
	varKinds map[ir.XrefId]ir.SemanticVariableKind,
	varInits map[ir.XrefId]output.Expression,
	varFences map[ir.XrefId]Fence,
) {
	restoreContextUsage := map[ir.XrefId]int{}
	for xref := range varOps {
		if varKinds[xref] != ir.SemanticVariableKindIdentifier {
			continue
		}
		readProp, ok := varInits[xref].(*output.ReadPropExpr)
		if !ok {
			continue
		}
		readVar, ok := readProp.Receiver.(*ir.ReadVariableExpr)
		if !ok || varKinds[readVar.Xref] != ir.SemanticVariableKindContext {
			continue
		}
		if _, ok := varInits[readVar.Xref].(*ir.RestoreViewExpr); ok {
			restoreContextUsage[readVar.Xref]++
		}
	}

	for xref, op := range varOps {
		if varKinds[xref] != ir.SemanticVariableKindIdentifier {
			continue
		}
		readProp, ok := varInits[xref].(*output.ReadPropExpr)
		if !ok {
			continue
		}
		readVar, ok := readProp.Receiver.(*ir.ReadVariableExpr)
		if !ok || varKinds[readVar.Xref] != ir.SemanticVariableKindContext {
			continue
		}
		if _, ok := varInits[readVar.Xref].(*ir.RestoreViewExpr); !ok {
			continue
		}
		if restoreContextUsage[readVar.Xref] != 1 {
			continue
		}

		readProp.Receiver = varInits[readVar.Xref].Clone()
		varInits[xref] = readProp
		
		// Remove the RestoreViewExpr from varOps and ops since it has been completely inlined.
		if ctxOp, exists := varOps[readVar.Xref]; exists {
			ops.Remove(ctxOp)
			delete(varOps, readVar.Xref)
			delete(varKinds, readVar.Xref)
			delete(varInits, readVar.Xref)
			delete(varFences, readVar.Xref)
		}

		switch varOp := op.(type) {
		case *ir.VariableOp:
			varOp.Initializer = readProp
		case *ir.CreateVariableOp:
			varOp.Initializer = readProp
		}
	}
}

// tryInlineVariable attempts to inline a variable into its single usage site.
// For Context kind variables (nextContext results), only inlining into other VariableOps
// is allowed (conservative inlining, matching ngtsc's allowConservativeInlining).
func tryInlineVariable(xref ir.XrefId, init output.Expression, kind ir.SemanticVariableKind, declFences Fence, decl ir.Op, ops *ir.OpList, opInfos map[ir.Op]opInfo) bool {
	elements := ops.Elements()
	declIndex := -1
	for i, op := range elements {
		if op == decl {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		return false
	}

	for _, target := range elements[declIndex+1:] {
		info := opInfos[target]
		if !info.variablesUsed[xref] {
			if !safeToInlinePastFences(info.fences, declFences) {
				break
			}
			continue
		}
		if !allowConservativeInlining(kind, init, target) {
			break
		}
		return tryInlineVariableInitializer(xref, init, target, declFences)
	}
	return false
}

func allowConservativeInlining(kind ir.SemanticVariableKind, init output.Expression, target ir.Op) bool {
	switch kind {
	case ir.SemanticVariableKindIdentifier:
		if readVar, ok := init.(*output.ReadVarExpr); ok && readVar.Name == contextName {
			return true
		}
		return false
	case ir.SemanticVariableKindContext:
		_, isVarOp := target.(*ir.VariableOp)
		_, isCreateVarOp := target.(*ir.CreateVariableOp)
		return isVarOp || isCreateVarOp
	default:
		return true
	}
}

func tryInlineVariableInitializer(xref ir.XrefId, init output.Expression, target ir.Op, declFences Fence) bool {
	inlined := false
	inliningAllowed := true

	ir.TransformExpressionsInOp(target, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
		irExpr, ok := expr.(ir.Expression)
		if !ok || !inliningAllowed || inlined {
			return expr
		}
		if flags&ir.VisitorContextFlagInChildOperation != 0 && declFences&FenceViewContextRead != 0 {
			return expr
		}
		if rv, ok := irExpr.(*ir.ReadVariableExpr); ok && rv.Xref == xref {
			inlined = true
			return init
		}
		inliningAllowed = inliningAllowed && safeToInlinePastFences(fencesForIrExpr(irExpr), declFences)
		return expr
	}, ir.VisitorContextFlagNone)
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
	ir.TransformExpressionsInExpression(expr, func(inner output.Expression, flags ir.VisitorContextFlag) output.Expression {
		if irExpr, ok := inner.(ir.Expression); ok {
			f |= fencesForIrExpr(irExpr)
		}
		return inner
	}, ir.VisitorContextFlagNone)
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
