package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// LiftLocalRefs lifts local reference declarations on element-like structures within each view
// into an entry in the consts array for the whole component.
func LiftLocalRefs(job *compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindElementStart, ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate, ir.OpKindTemplate:
				if lr, ok := op.(interface {
					GetLocalRefs() any
					SetLocalRefs(any)
					AddNumSlotsUsed(int)
				}); ok {
					refs, isSlice := lr.GetLocalRefs().([]ir.LocalRef)
					if !isSlice {
						panic("AssertionError: expected localRefs to be an array still")
					}
					lr.AddNumSlotsUsed(len(refs))
					if len(refs) > 0 {
						constIdx := job.AddConst(serializeLocalRefs(refs), nil)
						lr.SetLocalRefs(constIdx)
					} else {
						lr.SetLocalRefs(nil)
					}
				}
			}
		}
	}
}

func serializeLocalRefs(refs []ir.LocalRef) output.Expression {
	constRefs := make([]output.Expression, 0, len(refs)*2)
	for _, ref := range refs {
		constRefs = append(constRefs,
			output.NewLiteralExpr(ref.Name, nil, nil, nil),
			output.NewLiteralExpr(ref.Target, nil, nil, nil),
		)
	}
	return output.NewLiteralArrayExpr(constRefs, nil, nil, nil)
}

// ResolveContexts resolves ir.ContextExpr expressions to either the ctx parameter to component
// functions (for the current view context) or to variables that store those contexts.
func ResolveContexts(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, fnExpr := range unit.GetFunctions() {
			if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok {
				resolveContextsInScope(unit, arrowFn.Ops.Elements())
			}
		}
		resolveContextsInScope(unit, unit.GetCreate().Elements())
		resolveContextsInScope(unit, unit.GetUpdate().Elements())
	}
}

const ctxContextName = "ctx"

func resolveContextsInScope(view compilation.CompilationUnit, ops []ir.Op) {
	scope := map[ir.XrefId]output.Expression{}
	scope[view.GetXref()] = output.NewReadVarExpr(ctxContextName, nil, nil, nil)

	for _, op := range ops {
		switch op.Kind() {
		case ir.OpKindVariable:
			if v, ok := op.(*ir.VariableOp); ok {
				if v.Variable.Kind == ir.SemanticVariableKindContext {
					scope[v.Variable.View] = ir.NewReadVariableExpr(v.Xref)
				}
			}
		case ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindListener, ir.OpKindTwoWayListener:
			if lOp, ok := op.(ir.ListenerTrait); ok {
				resolveContextsInScope(view, lOp.GetHandlerOps().Elements())
			}
		case ir.OpKindRepeaterCreate:
			if r, ok := op.(*ir.RepeaterCreateOp); ok && r.TrackByOps != nil {
				if tbo, ok2 := r.TrackByOps.(*ir.OpList); ok2 {
					resolveContextsInScope(view, tbo.Elements())
				}
			}
		}
	}

	// Prefer ctx of the root view.
	if view.GetXref() == view.GetJob().GetRoot().GetXref() {
		scope[view.GetXref()] = output.NewReadVarExpr(ctxContextName, nil, nil, nil)
	}

	for _, op := range ops {
		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			if ctxExpr, ok := expr.(*ir.ContextExpr); ok {
				resolved, found := scope[ctxExpr.View]
				if !found {
					panic("No context found for reference to view from view")
				}
				return resolved
			}
			return expr
		}, ir.VisitorContextFlagNone)
	}
}
