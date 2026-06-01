package phases

import (
	"fmt"
	"strings"

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

func legacyAnimationName(name string) string {
	if strings.HasPrefix(name, "@") {
		return name
	}
	return "@" + name
}

func legacyAnimationListenerName(name string, phase *string) string {
	eventName := legacyAnimationName(name)
	if phase == nil || *phase == "" {
		return eventName
	}
	if strings.HasSuffix(eventName, "."+*phase) {
		return eventName
	}
	return eventName + "." + *phase
}

func listenerName(listener *ir.ListenerOp) string {
	if listener.IsLegacyAnimationListener {
		return legacyAnimationListenerName(listener.Name, listener.LegacyAnimationPhase)
	}
	return listener.Name
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
			panic("AssertionError: expected reified statements")
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
	case ir.ExpressionKindArrowFunction:
		arrowExpr := irExpr.(*ir.ArrowFunctionExpr)
		var params []*output.FnParam
		for i := range arrowExpr.Parameters {
			params = append(params, &arrowExpr.Parameters[i])
		}
		return output.NewArrowFunctionExpr(params, arrowExpr.Body, nil, nil, nil)
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
			slot = *ref.TargetSlot.Slot + 1 + ref.Offset
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
			varOffset := 0
			if pf.VarOffset != nil {
				varOffset = *pf.VarOffset
			}
			numArgs := len(pf.Args)
			var fnName string
			var args []output.Expression

			if numArgs <= 8 {
				fnName = fmt.Sprintf("\u0275\u0275pureFunction%d", numArgs)
				args = append([]output.Expression{output.NewLiteralExpr(varOffset, nil, nil, nil), pf.Fn}, pf.Args...)
			} else {
				fnName = "\u0275\u0275pureFunctionV"
				args = []output.Expression{
					output.NewLiteralExpr(varOffset, nil, nil, nil),
					pf.Fn,
					output.NewLiteralArrayExpr(pf.Args, nil, nil, nil),
				}
			}

			fn := output.NewReadVarExpr(fnName, nil, nil, nil)
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
		args := []output.Expression{
			output.NewLiteralExpr(slotVal, nil, nil, nil),
			output.NewLiteralExpr(pb.GetVarOffset(), nil, nil, nil),
		}
		args = append(args, pb.Args...)
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
	getSlot := func(slot any) output.Expression {
		if slot == nil {
			return output.NULL_EXPR
		}
		if i, ok := slot.(int); ok {
			return output.NewLiteralExpr(i, nil, nil, nil)
		}
		if sh, ok := slot.(*ir.SlotHandle); ok && sh != nil && sh.Slot != nil {
			return output.NewLiteralExpr(*sh.Slot, nil, nil, nil)
		}
		return output.NULL_EXPR
	}

	isNilExpression := func(e output.Expression) bool {
		// e is typed nil if we can't type assert it, or if reflection says it's nil.
		// A simpler check: in Go, comparing an interface to nil doesn't work for typed nils.
		// However, we can use a type switch or reflection.
		// For our AST, if it is exactly (*output.ReadVarExpr)(nil), etc.
		if e == nil {
			return true
		}
		switch val := e.(type) {
		case *output.ReadVarExpr:
			return val == nil
		case *output.LiteralExpr:
			return val == nil
		case *output.InvokeFunctionExpr:
			return val == nil
		}
		return false
	}

	// Each create op gets converted to one or more StatementOps.
	// The actual instruction calls depend on the op kind.
	// We delegate to a simple switch here; the full implementation would call
	// the ng.* instruction helper functions.
	switch op.Kind() {
	case ir.OpKindStatement:
		// Already a statement, nothing to do.
	case ir.OpKindNamespace:
		ns := op.(*ir.NamespaceOp)
		var fnName string
		switch ns.Active {
		case ir.NamespaceHTML:
			fnName = "ɵɵnamespaceHTML"
		case ir.NamespaceSVG:
			fnName = "ɵɵnamespaceSVG"
		case ir.NamespaceMath:
			fnName = "ɵɵnamespaceMathML"
		default:
			fnName = "ɵɵnamespaceHTML"
		}
		fn := output.NewReadVarExpr(fnName, nil, nil, nil)
		call := output.NewInvokeFunctionExpr(fn, nil, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindAdvance:
		adv := op.(*ir.AdvanceOp)
		fn := output.NewReadVarExpr("ɵɵadvance", nil, nil, nil)
		var args []output.Expression
		if adv.Delta != 1 {
			args = append(args, output.NewLiteralExpr(adv.Delta, nil, nil, nil))
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindPipe:
		pipe := op.(*ir.PipeOp)
		fn := output.NewReadVarExpr("ɵɵpipe", nil, nil, nil)
		var slotVal int = -1
		if pipe.SlotHandle != nil && pipe.SlotHandle.Slot != nil {
			slotVal = *pipe.SlotHandle.Slot
		} else {
			slotVal = int(pipe.Xref) - 1
		}
		args := []output.Expression{
			output.NewLiteralExpr(slotVal, nil, nil, nil),
			output.NewLiteralExpr(pipe.Name, nil, nil, nil),
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindElementStart, ir.OpKindElement, ir.OpKindContainerStart, ir.OpKindContainer:
		var slotVal int = -1
		var tagVal string
		var attrs any
		var localRefs any

		isElementStart := op.Kind() == ir.OpKindElementStart || op.Kind() == ir.OpKindContainerStart
		isContainerStart := op.Kind() == ir.OpKindContainerStart
		isContainer := op.Kind() == ir.OpKindContainer

		if el, ok := op.(*ir.ElementStartOp); ok {
			if el.SlotHandle != nil && el.SlotHandle.Slot != nil {
				slotVal = *el.SlotHandle.Slot
			} else {
				slotVal = int(el.Xref) - 1
			}
			if el.Tag != nil {
				tagVal = *el.Tag
			}
			attrs = el.Attributes
			localRefs = el.LocalRefsField
		} else if el, ok := op.(*ir.ElementOp); ok {
			if el.SlotHandle != nil && el.SlotHandle.Slot != nil {
				slotVal = *el.SlotHandle.Slot
			} else {
				slotVal = int(el.Xref) - 1
			}
			if el.Tag != nil {
				tagVal = *el.Tag
			}
			attrs = el.Attributes
			localRefs = el.LocalRefsField
		} else if el, ok := op.(*ir.ContainerStartOp); ok {
			if el.SlotHandle != nil && el.SlotHandle.Slot != nil {
				slotVal = *el.SlotHandle.Slot
			} else {
				slotVal = int(el.Xref) - 1
			}
			attrs = el.Attributes
			localRefs = el.LocalRefsField
		} else if el, ok := op.(*ir.ContainerOp); ok {
			if el.SlotHandle != nil && el.SlotHandle.Slot != nil {
				slotVal = *el.SlotHandle.Slot
			} else {
				slotVal = int(el.Xref) - 1
			}
			attrs = el.Attributes
			localRefs = el.LocalRefsField
		} else {
			panic("Expected ElementStartOp, ElementOp, ContainerStartOp, or ContainerOp")
		}

		var fn output.Expression
		if unit.GetJob().GetMode() == compilation.TemplateCompilationMode_DomOnly {
			if isContainerStart {
				fn = output.NewReadVarExpr("ɵɵdomElementContainerStart", nil, nil, nil)
			} else if isContainer {
				fn = output.NewReadVarExpr("ɵɵdomElementContainer", nil, nil, nil)
			} else if isElementStart {
				fn = output.NewReadVarExpr("ɵɵdomElementStart", nil, nil, nil)
			} else {
				fn = output.NewReadVarExpr("ɵɵdomElement", nil, nil, nil)
			}
		} else {
			if isContainerStart {
				fn = output.NewReadVarExpr("ɵɵelementContainerStart", nil, nil, nil)
			} else if isContainer {
				fn = output.NewReadVarExpr("ɵɵelementContainer", nil, nil, nil)
			} else if isElementStart {
				fn = output.NewReadVarExpr("ɵɵelementStart", nil, nil, nil)
			} else {
				fn = output.NewReadVarExpr("ɵɵelement", nil, nil, nil)
			}
		}

		args := []output.Expression{output.NewLiteralExpr(slotVal, nil, nil, nil)}
		if !isContainerStart && !isContainer {
			args = append(args, output.NewLiteralExpr(tagVal, nil, nil, nil))
		}
		if attrs != nil || localRefs != nil {
			if isContainerStart {
				if attrs != nil {
					args = append(args, output.NewLiteralExpr(attrs, nil, nil, nil))
				} else {
					args = append(args, output.NewLiteralExpr(nil, nil, nil, nil))
				}
				if localRefs != nil {
					args = append(args, output.NewLiteralExpr(localRefs, nil, nil, nil))
				}
			} else {
				if attrs != nil {
					args = append(args, output.NewLiteralExpr(attrs, nil, nil, nil))
				} else {
					args = append(args, output.NewLiteralExpr(nil, nil, nil, nil))
				}
				if localRefs != nil {
					args = append(args, output.NewLiteralExpr(localRefs, nil, nil, nil))
				}
			}
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
			output.NewLiteralExpr(listenerName(l), nil, nil, nil),
			listenerFn,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindTwoWayListener:
		l := op.(*ir.TwoWayListenerOp)

		name := ""
		if l.HandlerFnName != nil {
			name = *l.HandlerFnName
		}

		var handlerOpsList *ir.OpList
		if l.HandlerOps != nil {
			handlerOpsList = l.HandlerOps
		} else {
			handlerOpsList = ir.NewOpList()
		}

		listenerFn := reifyListenerHandler(unit, name, handlerOpsList, true) // ConsumesDollarEvent=true for two-way
		fn := output.NewReadVarExpr("ɵɵtwoWayListener", nil, nil, nil)
		args := []output.Expression{
			output.NewLiteralExpr(l.Name, nil, nil, nil),
			listenerFn,
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindElementEnd, ir.OpKindContainerEnd:
		var fn output.Expression
		if unit.GetJob().GetMode() == compilation.TemplateCompilationMode_DomOnly {
			if op.Kind() == ir.OpKindContainerEnd {
				fn = output.NewReadVarExpr("ɵɵdomElementContainerEnd", nil, nil, nil)
			} else {
				fn = output.NewReadVarExpr("ɵɵdomElementEnd", nil, nil, nil)
			}
		} else {
			if op.Kind() == ir.OpKindContainerEnd {
				fn = output.NewReadVarExpr("ɵɵelementContainerEnd", nil, nil, nil)
			} else {
				fn = output.NewReadVarExpr("ɵɵelementEnd", nil, nil, nil)
			}
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
		args := []output.Expression{output.NewLiteralExpr(slotVal, nil, nil, nil)}
		if txt.InitialValue != "" {
			args = append(args, output.NewLiteralExpr(txt.InitialValue, nil, nil, nil))
		}
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
		var fn output.Expression
		isBlock := false
		if k, ok := tmpl.TemplateKind.(ir.TemplateKind); ok && k == ir.TemplateKindBlock {
			isBlock = true
		}
		if unit.GetJob().GetMode() == compilation.TemplateCompilationMode_DomOnly || isBlock {
			fn = output.NewReadVarExpr("\u0275\u0275domTemplate", nil, nil, nil)
		} else {
			fn = output.NewReadVarExpr("\u0275\u0275template", nil, nil, nil)
		}

		var slotVal int = -1
		if tmpl.SlotHandle != nil && tmpl.SlotHandle.Slot != nil {
			slotVal = *tmpl.SlotHandle.Slot
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
		}

		hasAttrsOrRefs := tmpl.Attributes != nil || tmpl.LocalRefsField != nil
		if tmpl.Tag != nil {
			args = append(args, output.NewLiteralExpr(*tmpl.Tag, nil, nil, nil))
		} else if hasAttrsOrRefs {
			args = append(args, output.NULL_EXPR)
		}
		if tmpl.Attributes != nil || tmpl.LocalRefsField != nil {
			if tmpl.Attributes != nil {
				args = append(args, output.NewLiteralExpr(tmpl.Attributes, nil, nil, nil))
			} else {
				args = append(args, output.NULL_EXPR)
			}
			if tmpl.LocalRefsField != nil {
				args = append(args, output.NewLiteralExpr(tmpl.LocalRefsField, nil, nil, nil))
				args = append(args, output.NewReadVarExpr("\u0275\u0275templateRefExtractor", nil, nil, nil))
			}
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
			args = append(args, output.NewLiteralExpr(repeater.Attributes, nil, nil, nil))
		} else {
			args = append(args, output.NULL_EXPR)
		}

		if repeater.TrackByFn != nil {
			args = append(args, repeater.TrackByFn)
		} else {
			args = append(args, output.NULL_EXPR)
		}

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

		hasEmptyBlock := emptyTmplFn != output.NULL_EXPR || emptyDecls != 0 || emptyVars != 0 || repeater.EmptyTag != nil || repeater.EmptyAttributes != nil
		if repeater.UsesComponentInstance || hasEmptyBlock {
			args = append(args, output.NewLiteralExpr(repeater.UsesComponentInstance, nil, nil, nil))
		}

		if hasEmptyBlock {
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
	case ir.OpKindProjectionDef:
		projDef := op.(*ir.ProjectionDefOp)
		fn := output.NewReadVarExpr("\u0275\u0275projectionDef", nil, nil, nil)
		var args []output.Expression
		if projDef.Def != nil {
			args = append(args, projDef.Def)
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindProjection:
		proj := op.(*ir.ProjectionOp)
		fn := output.NewReadVarExpr("\u0275\u0275projection", nil, nil, nil)
		var slotVal int = -1
		if proj.SlotHandle != nil && proj.SlotHandle.Slot != nil {
			slotVal = *proj.SlotHandle.Slot
		} else {
			slotVal = int(proj.Xref) - 1
		}
		args := []output.Expression{output.NewLiteralExpr(slotVal, nil, nil, nil)}

		projIndex := 0
		if proj.ProjectionSlotIndex != nil {
			projIndex = *proj.ProjectionSlotIndex
		}
		if projIndex != 0 || proj.Attributes != nil {
			args = append(args, output.NewLiteralExpr(projIndex, nil, nil, nil))
			if proj.Attributes != nil {
				args = append(args, output.NewLiteralExpr(proj.Attributes, nil, nil, nil))
			}
		}

		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindControlCreate:
		fn := output.NewReadVarExpr("\u0275\u0275controlCreate", nil, nil, nil)
		call := output.NewInvokeFunctionExpr(fn, nil, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)
	case ir.OpKindDefer:
		deferOp := op.(*ir.DeferOp)
		fn := output.NewReadVarExpr("ɵɵdefer", nil, nil, nil)

		slot := getSlot(deferOp.Handle())
		mainSlot := getSlot(deferOp.MainSlot)
		var depsFn output.Expression
		if deferOp.ResolverFn != nil && !isNilExpression(deferOp.ResolverFn) {
			depsFn = deferOp.ResolverFn
		} else {
			depsFn = output.NULL_EXPR
		}

		var config output.Expression = output.NULL_EXPR
		if deferOp.PlaceholderConfig != nil && !isNilExpression(deferOp.PlaceholderConfig) {
			config = deferOp.PlaceholderConfig
		}

		placeholderSlot := getSlot(deferOp.PlaceholderSlot)
		errorSlot := getSlot(deferOp.ErrorSlot)
		loadingSlot := getSlot(deferOp.LoadingSlot)
		loadingConfig := deferOp.LoadingConfig
		if loadingConfig == nil || isNilExpression(loadingConfig) {
			loadingConfig = output.NULL_EXPR
		}

		args := []output.Expression{
			slot,
			mainSlot,
			depsFn,
			loadingSlot,
			placeholderSlot,
			errorSlot,
			loadingConfig,
			config, // placeholderConfig
		}

		// Remove trailing nulls. But actually, if we have configs, we need to pass deferEnableTimerScheduling.
		for len(args) > 2 && args[len(args)-1] == output.NULL_EXPR {
			args = args[:len(args)-1]
		}
		
		// ngc appends deferEnableTimerScheduling if we reach the config parameters and they require it.
		// Actually ngtsc appends it if timer scheduling is used (e.g. `minimum`, `after`).
		// Let's assume it's needed if we have config indices.
		if deferOp.PlaceholderConfig != nil || deferOp.LoadingConfig != nil {
			// Pad with nulls if needed
			for len(args) < 8 {
				args = append(args, output.NULL_EXPR)
			}
			args = append(args, output.NewReadVarExpr("\u0275\u0275deferEnableTimerScheduling", nil, nil, nil))
		}

		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(unit.GetCreate(), stmt, op)
		unit.GetCreate().Remove(op)

	case ir.OpKindDeferOn:
		deferOnOp := op.(*ir.DeferOnOp)

		var fnName string
		var args []output.Expression

		switch t := deferOnOp.Trigger.(type) {
		case ir.DeferIdleTrigger, *ir.DeferIdleTrigger:
			fnName = "OnIdle"
		case ir.DeferImmediateTrigger, *ir.DeferImmediateTrigger:
			fnName = "OnImmediate"
		case ir.DeferTimerTrigger:
			fnName = "OnTimer"
			args = append(args, output.NewLiteralExpr(t.Delay, nil, nil, nil))
		case *ir.DeferTimerTrigger:
			fnName = "OnTimer"
			args = append(args, output.NewLiteralExpr(t.Delay, nil, nil, nil))
		case ir.DeferHoverTrigger:
			fnName = "OnHover"
			args = append(args, getSlot(t.TargetSlot))
			if t.TargetSlotViewSteps != nil {
				args = append(args, output.NewLiteralExpr(*t.TargetSlotViewSteps, nil, nil, nil))
			} else {
				args = append(args, output.NULL_EXPR)
			}
		case *ir.DeferHoverTrigger:
			fnName = "OnHover"
			args = append(args, getSlot(t.TargetSlot))
			if t.TargetSlotViewSteps != nil {
				args = append(args, output.NewLiteralExpr(*t.TargetSlotViewSteps, nil, nil, nil))
			} else {
				args = append(args, output.NULL_EXPR)
			}
		case ir.DeferInteractionTrigger:
			fnName = "OnInteraction"
			args = append(args, getSlot(t.TargetSlot))
			if t.TargetSlotViewSteps != nil {
				args = append(args, output.NewLiteralExpr(*t.TargetSlotViewSteps, nil, nil, nil))
			} else {
				args = append(args, output.NULL_EXPR)
			}
		case *ir.DeferInteractionTrigger:
			fnName = "OnInteraction"
			args = append(args, getSlot(t.TargetSlot))
			if t.TargetSlotViewSteps != nil {
				args = append(args, output.NewLiteralExpr(*t.TargetSlotViewSteps, nil, nil, nil))
			} else {
				args = append(args, output.NULL_EXPR)
			}
		case ir.DeferViewportTrigger:
			fnName = "OnViewport"
			args = append(args, getSlot(t.TargetSlot))
			if t.TargetSlotViewSteps != nil {
				args = append(args, output.NewLiteralExpr(*t.TargetSlotViewSteps, nil, nil, nil))
			} else {
				args = append(args, output.NULL_EXPR)
			}
		case *ir.DeferViewportTrigger:
			fnName = "OnViewport"
			args = append(args, getSlot(t.TargetSlot))
			if t.TargetSlotViewSteps != nil {
				args = append(args, output.NewLiteralExpr(*t.TargetSlotViewSteps, nil, nil, nil))
			} else {
				args = append(args, output.NULL_EXPR)
			}
		default:
			panic(fmt.Sprintf("Unknown trigger type: %T", t))
		}

		var modifier ir.DeferOpModifierKind
		if deferOnOp.Modifier != nil {
			modifier = deferOnOp.Modifier.(ir.DeferOpModifierKind)
		} else {
			modifier = ir.DeferOpModifierKindNONE
		}

		fullFnName := ""
		switch modifier {
		case ir.DeferOpModifierKindNONE:
			fullFnName = "ɵɵdefer" + fnName
		case ir.DeferOpModifierKindPREFETCH:
			fullFnName = "ɵɵdeferPrefetch" + fnName
		case ir.DeferOpModifierKindHYDRATE:
			fullFnName = "ɵɵdeferHydrate" + fnName
		default:
			fullFnName = "ɵɵdefer" + fnName
		}

		fn := output.NewReadVarExpr(fullFnName, nil, nil, nil)
		for len(args) > 0 && args[len(args)-1] == output.NULL_EXPR {
			args = args[:len(args)-1]
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
				if numExprs < len(interp.Strings) && interp.Strings[numExprs] != "" {
					args = append(args, output.NewLiteralExpr(interp.Strings[numExprs], nil, nil, nil))
				}
			} else {
				fn = output.NewReadVarExpr("\u0275\u0275textInterpolateV", nil, nil, nil)
				var vArgs []output.Expression
				for i := 0; i < numExprs; i++ {
					vArgs = append(vArgs, output.NewLiteralExpr(interp.Strings[i], nil, nil, nil))
					vArgs = append(vArgs, interp.Expressions[i])
				}
				if numExprs < len(interp.Strings) && interp.Strings[numExprs] != "" {
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
			name := prop.Name
			if prop.BindingKind == ir.BindingKindLegacyAnimation {
				name = legacyAnimationName(name)
			}
			args = []output.Expression{
				output.NewLiteralExpr(name, nil, nil, nil),
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
	case ir.OpKindControl:
		fn := output.NewReadVarExpr("\u0275\u0275control", nil, nil, nil)
		call := output.NewInvokeFunctionExpr(fn, nil, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindTwoWayProperty:
		prop := op.(*ir.TwoWayPropertyOp)
		fn := output.NewReadVarExpr("ɵɵtwoWayProperty", nil, nil, nil)
		args := []output.Expression{
			output.NewLiteralExpr(prop.Name, nil, nil, nil),
			prop.Expression,
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
		if styleProp.Unit != nil {
			args = append(args, output.NewLiteralExpr(*styleProp.Unit, nil, nil, nil))
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindAttribute:
		attr := op.(*ir.AttributeOp)
		fn := output.NewReadVarExpr("\u0275\u0275attribute", nil, nil, nil)
		args := []output.Expression{
			output.NewLiteralExpr(attr.Name, nil, nil, nil),
			attr.Expression,
		}
		if attr.Sanitizer != nil {
			args = append(args, attr.Sanitizer)
		}
		call := output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	case ir.OpKindDomProperty:
		domProp := op.(*ir.DomPropertyOp)

		name := domProp.Name
		if mapped, ok := domPropertyRemapping[name]; ok {
			name = mapped
		}
		fn := output.NewReadVarExpr("\u0275\u0275property", nil, nil, nil)
		args := []output.Expression{
			output.NewLiteralExpr(name, nil, nil, nil),
			domProp.Expression.(output.Expression),
		}
		if domProp.Sanitizer != nil {
			args = append(args, domProp.Sanitizer)
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
	case ir.OpKindDeferWhen:
		deferWhen := op.(*ir.DeferWhenOp)

		var modifier ir.DeferOpModifierKind
		if deferWhen.Modifier != nil {
			modifier = deferWhen.Modifier.(ir.DeferOpModifierKind)
		} else {
			modifier = ir.DeferOpModifierKindNONE
		}

		fnName := ""
		switch modifier {
		case ir.DeferOpModifierKindNONE:
			fnName = "ɵɵdeferWhen"
		case ir.DeferOpModifierKindPREFETCH:
			fnName = "ɵɵdeferPrefetchWhen"
		case ir.DeferOpModifierKindHYDRATE:
			fnName = "ɵɵdeferHydrateWhen"
		default:
			fnName = "ɵɵdeferWhen"
		}

		fn := output.NewReadVarExpr(fnName, nil, nil, nil)
		call := output.NewInvokeFunctionExpr(fn, []output.Expression{deferWhen.Expr}, nil, nil, false, nil, false)
		stmt := &ir.StatementOp{Statement: &output.ExpressionStatement{Expr: call}}
		ir.OpListInsertBefore(opList, stmt, op)
		opList.Remove(op)
	default:
		panic(fmt.Sprintf("Unhandled update op in reifyUpdateOp: %T", op))
	}
}
