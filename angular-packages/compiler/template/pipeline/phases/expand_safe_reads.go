package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ExpandSafeReads expands safe navigation reads into ternary expressions (null-coalescing guards).
// For native optional chaining support, it converts SafePropertyReadExpr/SafeKeyedReadExpr to
// the corresponding output.ReadPropExpr/ReadKeyExpr with isOptional=true.
// For legacy null semantics, it expands to SafeTernaryExpr.
func ExpandSafeReads(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		legacyOptional := job.GetLegacyOptionalChaining()

		// Create/update ops.
		processOpList(unit.GetCreate(), job, legacyOptional)
		processOpList(unit.GetUpdate(), job, legacyOptional)

		// Listener handler ops.
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindListener, ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindTwoWayListener:
				if lOp, ok := op.(ir.ListenerTrait); ok {
					processOpList(lOp.HandlerOps(), job, legacyOptional)
				}
			}
		}

		// Inline functions.
		for _, fnExpr := range unit.GetFunctions() {
			if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok {
				processOpList(arrowFn.Ops, job, legacyOptional)
			}
		}

		// Second pass: convert SafeTernaryExpr to output ConditionalExpr.
		convertSafeTernaries(unit.GetCreate())
		convertSafeTernaries(unit.GetUpdate())
	}
}

func processOpList(ops *ir.OpList, job compilation.CompilationJob, legacyOptional bool) {
	for _, op := range ops.Elements() {
		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			return safeTransform(expr, job, flags, legacyOptional)
		}, ir.VisitorContextFlagNone)
	}
}

// safeTransform converts safe navigation IR expressions.
func safeTransform(expr output.Expression, job compilation.CompilationJob, flags ir.VisitorContextFlag, legacyOptional bool) output.Expression {
	// SafeNavigationMigrationExpr: unwrap to inner expression.
	if snm, ok := expr.(*ir.SafeNavigationMigrationExpr); ok {
		return snm.Expr
	}

	useNullSemantics := legacyOptional || (flags&ir.VisitorContextFlagInSafeNavMigration != 0)

	if !useNullSemantics {
		// Native optional chaining: set isOptional=true on output AST nodes.
		if sp, ok := expr.(*ir.SafePropertyReadExpr); ok {
			return output.NewReadPropExpr(sp.Receiver, sp.PropName, nil, nil, nil, true)
		}
		if sk, ok := expr.(*ir.SafeKeyedReadExpr); ok {
			return output.NewReadKeyExpr(sk.Receiver, sk.Index, nil, nil, nil, true)
		}
		return expr
	}

	// Legacy null semantics: expand to SafeTernaryExpr.
	if sp, ok := expr.(*ir.SafePropertyReadExpr); ok {
		return expandSafePropertyRead(sp, job)
	}
	if sk, ok := expr.(*ir.SafeKeyedReadExpr); ok {
		return expandSafeKeyedRead(sk, job)
	}
	// Optional function call.
	if invoke, ok := expr.(*output.InvokeFunctionExpr); ok && invoke.IsOptional {
		return expandSafeInvoke(invoke, job)
	}

	return expr
}

func expandSafePropertyRead(sp *ir.SafePropertyReadExpr, job compilation.CompilationJob) output.Expression {
	guard, bodyReceiver := maybeTemporary(sp.Receiver, job)
	body := output.NewReadPropExpr(bodyReceiver, sp.PropName, nil, nil, nil, false)
	return &ir.SafeTernaryExpr{Guard: guard, Expr: body}
}

func expandSafeKeyedRead(sk *ir.SafeKeyedReadExpr, job compilation.CompilationJob) output.Expression {
	guard, bodyReceiver := maybeTemporary(sk.Receiver, job)
	body := output.NewReadKeyExpr(bodyReceiver, sk.Index, nil, nil, nil, false)
	return &ir.SafeTernaryExpr{Guard: guard, Expr: body}
}

func expandSafeInvoke(invoke *output.InvokeFunctionExpr, job compilation.CompilationJob) output.Expression {
	guard, bodyFn := maybeTemporary(invoke.Fn, job)
	body := output.NewInvokeFunctionExpr(bodyFn, invoke.Args, nil, nil, false, nil, false)
	return &ir.SafeTernaryExpr{Guard: guard, Expr: body}
}

// maybeTemporary generates a temporary variable if needed to avoid double-evaluation.
func maybeTemporary(expr output.Expression, job compilation.CompilationJob) (guard output.Expression, body output.Expression) {
	if needsTemporaryForSafeAccess(expr) {
		xref := job.AllocateXrefId()
		assignExpr := &ir.AssignTemporaryExpr{Expr: expr, Xref: xref}
		readExpr := &ir.ReadTemporaryExpr{Xref: xref}
		return assignExpr, readExpr
	}
	return expr, expr.Clone()
}

// needsTemporaryForSafeAccess checks if an expression needs a temporary to avoid double evaluation.
func needsTemporaryForSafeAccess(expr output.Expression) bool {
	switch e := expr.(type) {
	case *output.UnaryOperatorExpr:
		return needsTemporaryForSafeAccess(e.Expr)
	case *output.BinaryOperatorExpr:
		return needsTemporaryForSafeAccess(e.Lhs) || needsTemporaryForSafeAccess(e.Rhs)
	case *output.ConditionalExpr:
		if e.FalseCase != nil && needsTemporaryForSafeAccess(e.FalseCase) {
			return true
		}
		return needsTemporaryForSafeAccess(e.Condition) || needsTemporaryForSafeAccess(e.TrueCase)
	case *output.ReadPropExpr:
		return needsTemporaryForSafeAccess(e.Receiver)
	case *output.ReadKeyExpr:
		return needsTemporaryForSafeAccess(e.Receiver) || needsTemporaryForSafeAccess(e.Index)
	case *ir.AssignTemporaryExpr:
		return needsTemporaryForSafeAccess(e.Expr)
	case *ir.SafeNavigationMigrationExpr:
		return needsTemporaryForSafeAccess(e.Expr)
	case *output.InvokeFunctionExpr, *output.LiteralArrayExpr, *output.LiteralMapExpr:
		return true
	case *ir.PipeBindingExpr:
		return true
	}
	return false
}

// convertSafeTernaries converts SafeTernaryExpr to ConditionalExpr in the output AST.
func convertSafeTernaries(ops *ir.OpList) {
	for _, op := range ops.Elements() {
		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			st, ok := expr.(*ir.SafeTernaryExpr)
			if !ok {
				return expr
			}
			// (guard == null) ? null : expr
			nullExpr := output.NULL_EXPR
			guard := output.NewBinaryOperatorExpr(
				output.BinaryOperatorEquals,
				st.Guard,
				nullExpr,
				nil, nil, nil,
			)
			cond := output.NewConditionalExpr(guard, nullExpr, st.Expr, nil, nil, nil)
			return output.NewParenthesizedExpr(cond, nil, nil, nil)
		}, ir.VisitorContextFlagNone)
	}
}
