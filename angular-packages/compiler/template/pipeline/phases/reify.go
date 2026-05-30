package phases

import (
	"fmt"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// DOM properties that need to be remapped.
var domPropertyRemapping = map[string]string{
	"class":      "className",
	"for":        "htmlFor",
	"formaction": "formAction",
	"innerHtml":  "innerHTML",
	"readonly":   "readOnly",
	"tabindex":   "tabIndex",
}

// Reify compiles semantic IR operations into actual runtime call statements.
// After reification, the create/update operation lists should only contain StatementOps.
func Reify(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		reifyCreateOperations(unit)
		reifyUpdateOperations(unit)
	}
}

// reifyCreateOperations converts create-phase IR ops to StatementOps.
func reifyCreateOperations(unit compilation.CompilationUnit) {
	ops := make([]ir.Op, len(unit.GetCreate().Elements()))
	copy(ops, unit.GetCreate().Elements())
	
	fmt.Printf("reifyCreateOperations ops:\n")
	for i, op := range ops {
		fmt.Printf("  %d: %v\n", i, op.Kind())
	}
	
	for _, op := range ops {
		// First transform any IR expressions within the op to output expressions.
		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			return reifyIrExpression(unit, expr)
		}, ir.VisitorContextFlagNone)

		// Then convert the op itself to statement(s).
		reifyCreateOp(unit, op)
	}
}

// reifyUpdateOperations converts update-phase IR ops to StatementOps.
func reifyUpdateOperationsList(unit compilation.CompilationUnit, opList *ir.OpList) {
	ops := make([]ir.Op, len(opList.Elements()))
	copy(ops, opList.Elements())
	for _, op := range ops {
		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			return reifyIrExpression(unit, expr)
		}, ir.VisitorContextFlagNone)

		reifyUpdateOp(unit, opList, op)
	}
}

func reifyUpdateOperations(unit compilation.CompilationUnit) {
	reifyUpdateOperationsList(unit, unit.GetUpdate())
}

func reifyListenerHandler(unit compilation.CompilationUnit, name string, handlerOps *ir.OpList, consumesDollarEvent bool) *output.FunctionExpr {
	reifyUpdateOperationsList(unit, handlerOps)
	var handlerStmts []output.Statement
	for _, op := range handlerOps.Elements() {
		if op.Kind() != ir.OpKindStatement {
			panic(fmt.Sprintf("AssertionError: expected reified statements, but found op %v", op.Kind()))
		}
		stmtOp := op.(*ir.StatementOp)
		handlerStmts = append(handlerStmts, stmtOp.Statement)
	}
	var params []*output.FnParam
	if consumesDollarEvent {
		params = append(params, &output.FnParam{Name: "$event"})
	}
	return output.NewFunctionExpr(params, handlerStmts, nil, nil, &name, nil)
}

// reifyIrExpression converts an IR expression to a plain output.Expression.
func reifyIrExpression(unit compilation.CompilationUnit, expr output.Expression) output.Expression {
	irExpr, ok := expr.(ir.Expression)
	if !ok {
		return expr
	}
	switch irExpr.ExprKind() {
	case ir.ExpressionKindLexicalRead:
		// LexicalRead should have been replaced in earlier phases.
		panic("AssertionError: LexicalReadExpr should have been resolved before reify")
	case ir.ExpressionKindSlotLiteralExpr:
		slotExpr := irExpr.(*ir.SlotLiteralExpr)
		slotVal := -1
		if slotExpr.SlotHandle != nil && slotExpr.SlotHandle.Slot != nil {
			slotVal = *slotExpr.SlotHandle.Slot
		}
		return output.NewLiteralExpr(slotVal, nil, nil, nil)
	case ir.ExpressionKindContext:
		ctxExpr := irExpr.(*ir.ContextExpr)
		_ = ctxExpr
		return output.NewReadVarExpr("ctx", nil, nil, nil)
	case ir.ExpressionKindGetCurrentView:
		return output.NewInvokeFunctionExpr(output.NewReadVarExpr("ɵɵgetCurrentView", nil, nil, nil), nil, nil, nil, false, nil, false)
	case ir.ExpressionKindNextContext:
		nc := irExpr.(*ir.NextContextExpr)
		fn := output.NewReadVarExpr("\u0275\u0275nextContext", nil, nil, nil)
		var args []output.Expression
		if nc.Steps > 1 {
			args = append(args, output.NewLiteralExpr(nc.Steps, nil, nil, nil))
		}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindReference:
		ref := irExpr.(*ir.ReferenceExpr)
		fn := output.NewReadVarExpr("ɵɵreference", nil, nil, nil)
		slot := 0
		if ref.TargetSlot != nil && ref.TargetSlot.Slot != nil {
			slot = *ref.TargetSlot.Slot
		}
		args := []output.Expression{output.NewLiteralExpr(slot, nil, nil, nil)}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindStoreLet:
		sl := irExpr.(*ir.StoreLetExpr)
		fn := output.NewReadVarExpr("\u0275\u0275storeLet", nil, nil, nil)
		args := []output.Expression{sl.Value}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindContextLetReference:
		clr := irExpr.(*ir.ContextLetReferenceExpr)
		fn := output.NewReadVarExpr("ɵɵreadContextLet", nil, nil, nil)
		args := []output.Expression{output.NewLiteralExpr(clr.Target, nil, nil, nil)}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindRestoreView:
		rv := irExpr.(*ir.RestoreViewExpr)
		fn := output.NewReadVarExpr("ɵɵrestoreView", nil, nil, nil)
		var viewArg output.Expression
		if rv.ViewExpr != nil {
			viewArg = rv.ViewExpr
		} else if rv.ViewXref != nil {
			viewArg = ir.NewReadVariableExpr(*rv.ViewXref)
		} else {
			viewArg = output.NULL_EXPR
		}
		args := []output.Expression{viewArg}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindResetView:
		rsv := irExpr.(*ir.ResetViewExpr)
		fn := output.NewReadVarExpr("\u0275\u0275resetView", nil, nil, nil)
		args := []output.Expression{rsv.Expr}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindReadVariable:
		rv := irExpr.(*ir.ReadVariableExpr)
		if rv.Name != nil {
			return output.NewReadVarExpr(*rv.Name, nil, nil, nil)
		}
		return output.NewReadVarExpr("_unknownVar", nil, nil, nil)
	case ir.ExpressionKindAssignTemporaryExpr:
		at := irExpr.(*ir.AssignTemporaryExpr)
		if at.Name != nil {
			lhs := output.NewReadVarExpr(*at.Name, nil, nil, nil)
			return output.NewBinaryOperatorExpr(output.BinaryOperatorAssign, lhs, at.Expr, nil, nil, nil)
		}
		return at.Expr
	case ir.ExpressionKindReadTemporaryExpr:
		rt := irExpr.(*ir.ReadTemporaryExpr)
		if rt.Name != nil {
			return output.NewReadVarExpr(*rt.Name, nil, nil, nil)
		}
		return output.NewReadVarExpr("_tmp", nil, nil, nil)
	case ir.ExpressionKindTrackContext:
		tc := irExpr.(*ir.TrackContextExpr)
		fn := output.NewReadVarExpr("\u0275\u0275componentInstance", nil, nil, nil)
		_ = tc
		return output.NewInvokeFunctionExpr(fn, nil, nil, nil, false, nil, false)
	case ir.ExpressionKindPureFunctionExpr:
		pf := irExpr.(*ir.PureFunctionExpr)
		if pf.Fn != nil {
			fn := output.NewReadVarExpr("\u0275\u0275pureFunction0", nil, nil, nil)
			args := append([]output.Expression{output.NewLiteralExpr(len(pf.Args), nil, nil, nil), pf.Fn}, pf.Args...)
			return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		}
		return output.NULL_EXPR
	case ir.ExpressionKindPipeBinding:
		pb := irExpr.(*ir.PipeBindingExpr)
		fnName := "\u0275\u0275pipeBind1"
		if len(pb.Args) > 1 {
			fnName = "\u0275\u0275pipeBind" + string(rune('0'+len(pb.Args)))
		}
		fn := output.NewReadVarExpr(fnName, nil, nil, nil)
		slotVal := 0
		if pb.TargetSlot != nil && pb.TargetSlot.Slot != nil {
			slotVal = *pb.TargetSlot.Slot
		}
		args := append([]output.Expression{output.NewLiteralExpr(slotVal, nil, nil, nil)}, pb.Args...)
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindTwoWayBindingSet:
		tb := irExpr.(*ir.TwoWayBindingSetExpr)
		fn := output.NewReadVarExpr("\u0275\u0275twoWayBindingSet", nil, nil, nil)
		args := []output.Expression{tb.TargetExpr, tb.Value}
		return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
	case ir.ExpressionKindSafeNavigationMigration:
		snm := irExpr.(*ir.SafeNavigationMigrationExpr)
		return snm.Expr
	case ir.ExpressionKindSafeTernaryExpr:
		// SafeTernaryExpr should have been expanded by ExpandSafeReads.
		st := irExpr.(*ir.SafeTernaryExpr)
		_ = st
		panic("AssertionError: SafeTernaryExpr should have been expanded before reify")
	default:
		return expr
	}
}

// reifyCreateOp converts a create-phase op to statement(s).
func reifyCreateOp(unit compilation.CompilationUnit, op ir.Op) {
	// Each create op gets converted to one or more StatementOps.
	// The actual instruction calls depend on the op kind.
	// We delegate to a simple switch here; the full implementation would call
	// the ng.* instruction helper functions.
	switch op.Kind() {
	case ir.OpKindStatement:
		// Already a statement, nothing to do.
	case ir.OpKindAdvance:
		adv := op.(*ir.AdvanceOp)
		fn := output.NewReadVarExpr("\u0275\u0275advance", nil, nil, nil)
		var args []output.Expression
		if adv.Delta != 1 {
			args = append(args, output.NewLiteralExpr(adv.Delta, nil, nil, nil))
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindElementStart:
		el := op.(*ir.ElementStartOp)
		var fn output.Expression
		if unit.GetJob().GetMode() == compilation.TemplateCompilationMode_DomOnly {
			fn = output.NewReadVarExpr("\u0275\u0275domElementStart", nil, nil, nil)
		} else {
			fn = output.NewReadVarExpr("\u0275\u0275elementStart", nil, nil, nil)
		}
		var slotVal int = -1
		if el.SlotHandle != nil && el.SlotHandle.Slot != nil {
			slotVal = *el.SlotHandle.Slot
		} else {
			slotVal = int(el.Xref) - 1
		}
		var tagVal string
		if el.Tag != nil {
			tagVal = *el.Tag
		}
		args := []output.Expression{output.NewLiteralExpr(slotVal, nil, nil, nil), output.NewLiteralExpr(tagVal, nil, nil, nil)}
		if el.Attributes != nil {
			args = append(args, output.NewLiteralExpr(el.Attributes, nil, nil, nil))
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindListener:
		l := op.(*ir.ListenerOp)
		
		name := ""
		if l.HandlerFnName != nil {
			name = *l.HandlerFnName
		}
		
		var handlerOpsList *ir.OpList
		if l.HandlerOps != nil {
			handlerOpsList = l.HandlerOps.(*ir.OpList)
		} else {
			handlerOpsList = ir.NewOpList()
		}

		listenerFn := reifyListenerHandler(unit, name, handlerOpsList, l.ConsumesDollarEvent)
		var fn output.Expression
		if unit.GetJob().GetMode() == compilation.TemplateCompilationMode_DomOnly && !l.HostListener && !l.IsLegacyAnimationListener {
			fn = output.NewReadVarExpr("\u0275\u0275domListener", nil, nil, nil)
		} else {
			fn = output.NewReadVarExpr("\u0275\u0275listener", nil, nil, nil)
		}
		args := []output.Expression{
			output.NewLiteralExpr(l.Name, nil, nil, nil),
			listenerFn,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindElementEnd:
		var fn output.Expression
		if unit.GetJob().GetMode() == compilation.TemplateCompilationMode_DomOnly {
			fn = output.NewReadVarExpr("\u0275\u0275domElementEnd", nil, nil, nil)
		} else {
			fn = output.NewReadVarExpr("\u0275\u0275elementEnd", nil, nil, nil)
		}
		call := output.NewInvokeFunctionExpr(fn, nil, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindText:
		txt := op.(*ir.TextOp)
		fn := output.NewReadVarExpr("\u0275\u0275text", nil, nil, nil)
		var slotVal int = -1
		if txt.SlotHandle != nil && txt.SlotHandle.Slot != nil {
			slotVal = *txt.SlotHandle.Slot
		} else {
			slotVal = int(txt.Xref) - 1
		}
		args := []output.Expression{output.NewLiteralExpr(slotVal, nil, nil, nil), output.NewLiteralExpr(txt.InitialValue, nil, nil, nil)}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindVariable:
		var name string
		var init output.Expression
		if v, ok := op.(*ir.VariableOp); ok {
			if v.Variable != nil && v.Variable.Name != nil {
				name = *v.Variable.Name
			} else {
				name = "_var"
			}
			init = v.Initializer
		} else if cv, ok := op.(*ir.CreateVariableOp); ok {
			if cv.Variable != nil && cv.Variable.Name != nil {
				name = *cv.Variable.Name
			} else {
				name = "_var"
			}
			init = cv.Initializer
		}
		stmt := &ir.StatementOp{Statement: output.NewDeclareVarStmt(name, init, nil, output.StmtModifierFinal, nil, nil)}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindTemplate:
		tmpl := op.(*ir.TemplateOp)
		fn := output.NewReadVarExpr("\u0275\u0275template", nil, nil, nil)
		
		var slotVal int = -1
		if tmpl.SlotHandle != nil && tmpl.SlotHandle.Slot != nil {
			slotVal = *tmpl.SlotHandle.Slot
		}
		
		tagVal := ""
		if tmpl.Tag != nil {
			tagVal = *tmpl.Tag
		}
		
		var tmplFn output.Expression = output.NULL_EXPR
		if job, ok := unit.GetJob().(*compilation.ComponentCompilationJob); ok {
			if childView, exists := job.Views[tmpl.Xref]; exists && childView.FnName != nil {
				tmplFn = output.NewReadVarExpr(*childView.FnName, nil, nil, nil)
			}
		}
		
		var decls, vars int
		if tmpl.Decls != nil {
			decls = *tmpl.Decls
		}
		if tmpl.Vars != nil {
			vars = *tmpl.Vars
		}
		
		args := []output.Expression{
			output.NewLiteralExpr(slotVal, nil, nil, nil),
			tmplFn,
			output.NewLiteralExpr(decls, nil, nil, nil),
			output.NewLiteralExpr(vars, nil, nil, nil),
			output.NewLiteralExpr(tagVal, nil, nil, nil),
		}
		if tmpl.Attributes != nil {
			args = append(args, output.NewLiteralExpr(tmpl.Attributes, nil, nil, nil))
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindConditionalCreate:
		cond := op.(*ir.ConditionalCreateOp)
		fn := output.NewReadVarExpr("\u0275\u0275conditionalCreate", nil, nil, nil)
		var slotVal int = -1
		if cond.SlotHandle != nil && cond.SlotHandle.Slot != nil {
			slotVal = *cond.SlotHandle.Slot
		}
		
		decls := 0
		if cond.Decls != nil {
			decls = *cond.Decls
		}
		vars := 0
		if cond.Vars != nil {
			vars = *cond.Vars
		}
		
		var tmplFn output.Expression = output.NULL_EXPR
		if job, ok := unit.GetJob().(*compilation.ComponentCompilationJob); ok {
			if childView, exists := job.Views[cond.Xref]; exists {
				if childView.FnName != nil {
					tmplFn = output.NewReadVarExpr(*childView.FnName, nil, nil, nil)
				}
				if cond.Decls == nil && childView.Decls != nil {
					decls = *childView.Decls
				}
				if cond.Vars == nil && childView.Vars != nil {
					vars = *childView.Vars
				}
			}
		}
		
		args := []output.Expression{
			output.NewLiteralExpr(slotVal, nil, nil, nil),
			tmplFn,
			output.NewLiteralExpr(decls, nil, nil, nil),
			output.NewLiteralExpr(vars, nil, nil, nil),
		}
		if cond.Tag != nil {
			args = append(args, output.NewLiteralExpr(*cond.Tag, nil, nil, nil))
			if cond.Attributes != nil {
				args = append(args, output.NewLiteralExpr(cond.Attributes, nil, nil, nil))
			}
		} else if cond.Attributes != nil {
			args = append(args, output.NULL_EXPR)
			args = append(args, output.NewLiteralExpr(cond.Attributes, nil, nil, nil))
		}
		var call output.Expression = output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		
		// Gather ConditionalBranchCreateOps by looking ahead in the actual slice
		elements := unit.GetCreate().Elements()
		idx := -1
		for i, e := range elements {
			if e == op {
				idx = i
				break
			}
		}
		
		if idx != -1 {
			for i := idx + 1; i < len(elements); i++ {
				next := elements[i]
				if next.Kind() == ir.OpKindExtractedAttribute {
					// Skip extracted attributes
					continue
				}
				if next.Kind() == ir.OpKindConditionalBranchCreate {
					branch := next.(*ir.ConditionalBranchCreateOp)
					var branchSlotVal int = -1
					if branch.SlotHandle != nil && branch.SlotHandle.Slot != nil {
						branchSlotVal = *branch.SlotHandle.Slot
					}
					branchDecls := 0
					if branch.Decls != nil {
						branchDecls = *branch.Decls
					}
					branchVars := 0
					if branch.Vars != nil {
						branchVars = *branch.Vars
					}
					
					var branchTmplFn output.Expression = output.NULL_EXPR
					if job, ok := unit.GetJob().(*compilation.ComponentCompilationJob); ok {
						if childView, exists := job.Views[branch.Xref]; exists {
							if childView.FnName != nil {
								branchTmplFn = output.NewReadVarExpr(*childView.FnName, nil, nil, nil)
							}
							if branch.Decls == nil && childView.Decls != nil {
								branchDecls = *childView.Decls
							}
							if branch.Vars == nil && childView.Vars != nil {
								branchVars = *childView.Vars
							}
						}
					}
					
					branchArgs := []output.Expression{
						output.NewLiteralExpr(branchSlotVal, nil, nil, nil),
						branchTmplFn,
						output.NewLiteralExpr(branchDecls, nil, nil, nil),
						output.NewLiteralExpr(branchVars, nil, nil, nil),
					}
					if branch.Tag != nil {
						branchArgs = append(branchArgs, output.NewLiteralExpr(*branch.Tag, nil, nil, nil))
						if branch.Attributes != nil {
							branchArgs = append(branchArgs, output.NewLiteralExpr(branch.Attributes, nil, nil, nil))
						}
					} else if branch.Attributes != nil {
						branchArgs = append(branchArgs, output.NULL_EXPR)
						branchArgs = append(branchArgs, output.NewLiteralExpr(branch.Attributes, nil, nil, nil))
					}
					call = output.NewInvokeFunctionExpr(call, branchArgs, nil, nil, false, nil, false)
					unit.GetCreate().Remove(next)
					// Since we removed an element from the slice, we need to adjust our index and elements list
					elements = unit.GetCreate().Elements()
					i--
				} else {
					break
				}
			}
		}
		
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindConditionalBranchCreate:
		// Should be handled by peeking in ConditionalCreate
		unit.GetCreate().Remove(op)
	case ir.OpKindRepeaterCreate:
		repeater := op.(*ir.RepeaterCreateOp)
		fn := output.NewReadVarExpr("\u0275\u0275repeaterCreate", nil, nil, nil)
		
		var slotVal int = -1
		if repeater.SlotHandle != nil && repeater.SlotHandle.Slot != nil {
			slotVal = *repeater.SlotHandle.Slot
		}
		
		var tmplFn output.Expression = output.NULL_EXPR
		if job, ok := unit.GetJob().(*compilation.ComponentCompilationJob); ok {
			if childView, exists := job.Views[repeater.Xref]; exists && childView.FnName != nil {
				tmplFn = output.NewReadVarExpr(*childView.FnName, nil, nil, nil)
			}
		}

		var decls, vars int
		if repeater.Decls != nil {
			decls = *repeater.Decls
		} else if job, ok := unit.GetJob().(*compilation.ComponentCompilationJob); ok {
			if childView, exists := job.Views[repeater.Xref]; exists {
				if childView.Decls != nil {
					decls = *childView.Decls
				}
				if childView.Vars != nil {
					vars = *childView.Vars
				}
			}
		}
		if repeater.Vars != nil {
			vars = *repeater.Vars
		}
		
		args := []output.Expression{
			output.NewLiteralExpr(slotVal, nil, nil, nil),
			tmplFn,
			output.NewLiteralExpr(decls, nil, nil, nil),
			output.NewLiteralExpr(vars, nil, nil, nil),
		}
		
		var tagVal string
		if repeater.Tag != nil {
			tagVal = *repeater.Tag
		}
		args = append(args, output.NewLiteralExpr(tagVal, nil, nil, nil))
		
		if repeater.Attributes != nil {
			println(fmt.Sprintf("reify: RepeaterCreate attributes is %T: %v", repeater.Attributes, repeater.Attributes))
			args = append(args, output.NewLiteralExpr(repeater.Attributes, nil, nil, nil))
		} else {
			args = append(args, output.NULL_EXPR)
		}
		
		if repeater.TrackByFn != nil {
			args = append(args, repeater.TrackByFn)
		} else {
			args = append(args, output.NULL_EXPR)
		}
		
		args = append(args, output.NewLiteralExpr(repeater.UsesComponentInstance, nil, nil, nil))
		
		var emptyTmplFn output.Expression = output.NULL_EXPR
		var emptyDecls, emptyVars int
		if repeater.EmptyView != 0 {
			if job, ok := unit.GetJob().(*compilation.ComponentCompilationJob); ok {
				if emptyView, exists := job.Views[repeater.EmptyView]; exists {
					if emptyView.FnName != nil {
						emptyTmplFn = output.NewReadVarExpr(*emptyView.FnName, nil, nil, nil)
					}
					if emptyView.Decls != nil {
						emptyDecls = *emptyView.Decls
					}
					if emptyView.Vars != nil {
						emptyVars = *emptyView.Vars
					}
				}
			}
		}
		
		if emptyTmplFn != output.NULL_EXPR || emptyDecls != 0 || emptyVars != 0 || repeater.EmptyTag != nil || repeater.EmptyAttributes != nil {
			args = append(args, emptyTmplFn)
			args = append(args, output.NewLiteralExpr(emptyDecls, nil, nil, nil))
			args = append(args, output.NewLiteralExpr(emptyVars, nil, nil, nil))
			var emptyTagVal string
			if repeater.EmptyTag != nil {
				emptyTagVal = *repeater.EmptyTag
			}
			args = append(args, output.NewLiteralExpr(emptyTagVal, nil, nil, nil))
			if repeater.EmptyAttributes != nil {
				args = append(args, output.NewLiteralExpr(repeater.EmptyAttributes, nil, nil, nil))
			}
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	default:
		// For unhandled ops, log or remove them to prevent panics in emit, or leave them for debugging.
		// For now, let's remove them so emit doesn't panic on unimplemented ops.
		unit.GetCreate().Remove(op)
	}
}

// reifyUpdateOp converts an update-phase op to statement(s).
func reifyUpdateOp(unit compilation.CompilationUnit, opList *ir.OpList, op ir.Op) {
	switch op.Kind() {
	case ir.OpKindStatement:
		// Already a statement.
	case ir.OpKindAdvance:
		adv := op.(*ir.AdvanceOp)
		fn := output.NewReadVarExpr("\u0275\u0275advance", nil, nil, nil)
		var args []output.Expression
		if adv.Delta != 1 {
			args = append(args, output.NewLiteralExpr(adv.Delta, nil, nil, nil))
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindVariable:
		var name string
		var init output.Expression
		if v, ok := op.(*ir.VariableOp); ok {
			if v.Variable != nil && v.Variable.Name != nil {
				name = *v.Variable.Name
			} else {
				name = "_var"
			}
			init = v.Initializer
		} else if cv, ok := op.(*ir.CreateVariableOp); ok {
			if cv.Variable != nil && cv.Variable.Name != nil {
				name = *cv.Variable.Name
			} else {
				name = "_var"
			}
			init = cv.Initializer
		}
		stmt := &ir.StatementOp{Statement: output.NewDeclareVarStmt(name, init, nil, output.StmtModifierFinal, nil, nil)}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindInterpolateText:
		txt := op.(*ir.InterpolateTextOp)
		var interp *ir.Interpolation
		if i, ok := txt.Interpolation.(*ir.Interpolation); ok {
			interp = i
		} else {
			// fallback
		}
		
		var fn output.Expression
		var args []output.Expression
		
		if interp != nil {
			numExprs := len(interp.Expressions)
			
			if numExprs == 1 && len(interp.Strings) == 2 && interp.Strings[0] == "" && interp.Strings[1] == "" {
				fn = output.NewReadVarExpr("\u0275\u0275textInterpolate", nil, nil, nil)
				args = []output.Expression{interp.Expressions[0]}
			} else if numExprs >= 1 && numExprs <= 8 {
				fn = output.NewReadVarExpr(fmt.Sprintf("\u0275\u0275textInterpolate%d", numExprs), nil, nil, nil)
				for i := 0; i < numExprs; i++ {
					args = append(args, output.NewLiteralExpr(interp.Strings[i], nil, nil, nil))
					args = append(args, interp.Expressions[i])
				}
				if numExprs < len(interp.Strings) {
					args = append(args, output.NewLiteralExpr(interp.Strings[numExprs], nil, nil, nil))
				}
			} else {
				fn = output.NewReadVarExpr("\u0275\u0275textInterpolateV", nil, nil, nil)
				var vArgs []output.Expression
				for i := 0; i < numExprs; i++ {
					vArgs = append(vArgs, output.NewLiteralExpr(interp.Strings[i], nil, nil, nil))
					vArgs = append(vArgs, interp.Expressions[i])
				}
				if numExprs < len(interp.Strings) {
					vArgs = append(vArgs, output.NewLiteralExpr(interp.Strings[numExprs], nil, nil, nil))
				}
				args = []output.Expression{output.NewLiteralArrayExpr(vArgs, nil, nil, nil)}
			}
		}
		
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindProperty:
		prop := op.(*ir.PropertyOp)
		var fn output.Expression
		var args []output.Expression
		if unit.GetJob().GetMode() == compilation.TemplateCompilationMode_DomOnly && prop.BindingKind != ir.BindingKindAnimation && prop.BindingKind != ir.BindingKindLegacyAnimation {
			fn = output.NewReadVarExpr("\u0275\u0275domProperty", nil, nil, nil)
			name := prop.Name
			if mapped, ok := domPropertyRemapping[name]; ok {
				name = mapped
			}
			args = []output.Expression{
				output.NewLiteralExpr(name, nil, nil, nil),
				prop.Expression,
			}
		} else {
			fn = output.NewReadVarExpr("\u0275\u0275property", nil, nil, nil)
			args = []output.Expression{
				output.NewLiteralExpr(prop.Name, nil, nil, nil),
				prop.Expression,
			}
		}
		if prop.Sanitizer != nil {
			args = append(args, prop.Sanitizer)
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindClassProp:
		classProp := op.(*ir.ClassPropOp)
		fn := output.NewReadVarExpr("\u0275\u0275classProp", nil, nil, nil)
		args := []output.Expression{
			output.NewLiteralExpr(classProp.Name, nil, nil, nil),
			classProp.Expression,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindStyleProp:
		styleProp := op.(*ir.StylePropOp)
		fn := output.NewReadVarExpr("\u0275\u0275styleProp", nil, nil, nil)
		args := []output.Expression{
			output.NewLiteralExpr(styleProp.Name, nil, nil, nil),
			styleProp.Expression,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindConditional:
		cond := op.(*ir.ConditionalOp)
		fn := output.NewReadVarExpr("\u0275\u0275conditional", nil, nil, nil)
		
		args := []output.Expression{
			cond.Processed,
		}
		if cond.ContextValue != nil {
			args = append(args, cond.ContextValue)
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindRepeater:
		repeater := op.(*ir.RepeaterOp)
		fn := output.NewReadVarExpr("\u0275\u0275repeater", nil, nil, nil)
		args := []output.Expression{
			repeater.Collection,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindClassMap:
		classMap := op.(*ir.ClassMapOp)
		fn := output.NewReadVarExpr("\u0275\u0275classMap", nil, nil, nil)
		args := []output.Expression{
			classMap.Expression,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindStyleMap:
		styleMap := op.(*ir.StyleMapOp)
		fn := output.NewReadVarExpr("\u0275\u0275styleMap", nil, nil, nil)
		args := []output.Expression{
			styleMap.Expression,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	default:
		// For unhandled ops, log or remove them to prevent panics in emit, or leave them for debugging.
		// For now, let's remove them so emit doesn't panic on unimplemented ops.
		opList.Remove(op)
	}
}
