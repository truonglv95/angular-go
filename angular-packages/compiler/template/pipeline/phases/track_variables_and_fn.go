package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// GenerateTrackVariables replaces LexicalReadExprs in track expressions with $index/$item reads.
func GenerateTrackVariables(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindRepeaterCreate {
				continue
			}
			repeater, ok := op.(*ir.RepeaterCreateOp)
			if !ok || repeater.Track == nil {
				continue
			}

			varNames, ok := repeater.VarNames.(struct {
				Index    map[string]bool
				Implicit string
			})

			repeater.Track = ir.TransformExpressionsInExpression(repeater.Track, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				lex, isLex := expr.(*ir.LexicalReadExpr)
				if !isLex {
					return expr
				}
				if ok && varNames.Index[lex.Name] {
					return output.NewReadVarExpr("$index", nil, nil, nil)
				}
				if ok && lex.Name == varNames.Implicit {
					return output.NewReadVarExpr("$item", nil, nil, nil)
				}
				return expr
			}, ir.VisitorContextFlagNone)
		}
	}
}

// OptimizeTrackFns optimizes track functions in for repeaters where possible.
func OptimizeTrackFns(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindRepeaterCreate {
				continue
			}
			repeater, ok := op.(*ir.RepeaterCreateOp)
			if !ok || repeater.Track == nil {
				continue
			}

			if rv, ok := repeater.Track.(*output.ReadVarExpr); ok && rv.Name == "$index" {
				// Track by $index → use repeaterTrackByIndex builtin (stored as nil placeholder; reify resolves).
				repeater.TrackByFn = output.NewReadVarExpr("ɵɵrepeaterTrackByIndex", nil, nil, nil)
			} else if rv, ok := repeater.Track.(*output.ReadVarExpr); ok && rv.Name == "$item" {
				// Track by $item → use repeaterTrackByIdentity builtin.
				repeater.TrackByFn = output.NewReadVarExpr("ɵɵrepeaterTrackByIdentity", nil, nil, nil)
			} else if isTrackByFunctionCall(job.GetRoot().GetXref(), repeater.Track) {
				repeater.UsesComponentInstance = true
				if invoke, ok := repeater.Track.(*output.InvokeFunctionExpr); ok {
					if readProp, ok2 := invoke.Fn.(*output.ReadPropExpr); ok2 {
						if ctxExpr, ok3 := readProp.Receiver.(*ir.ContextExpr); ok3 {
							if ctxExpr.View == unit.GetXref() {
								repeater.TrackByFn = readProp
							} else {
								// Need component instance.
								compInst := output.NewReadVarExpr("ɵɵcomponentInstance", nil, nil, nil)
								repeater.TrackByFn = output.NewReadPropExpr(compInst, readProp.Name, nil, nil, nil, false)
								repeater.Track = repeater.TrackByFn
							}
						}
					}
				}
			} else {
				// Non-optimizable: replace ContextExprs with TrackContextExprs.
				repeater.Track = ir.TransformExpressionsInExpression(repeater.Track, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
					switch expr.(type) {
					case *ir.PipeBindingExpr, *ir.PipeBindingVariadicExpr:
						panic("Illegal State: Pipes are not allowed in this context")
					}
					if ctxExpr, ok := expr.(*ir.ContextExpr); ok {
						repeater.UsesComponentInstance = true
						return ir.NewTrackContextExpr(ctxExpr.View)
					}
					return expr
				}, ir.VisitorContextFlagNone)

				// Wrap in an OpList for the track fn body.
				trackOpList := ir.NewOpList()
				retStmt := &ir.StatementOp{
					Statement: &output.ReturnStatement{Value: repeater.Track},
				}
				trackOpList.Push(retStmt)
				repeater.TrackByOps = trackOpList
			}
		}
	}
}

// isTrackByFunctionCall checks if the expression is a top-level method call `fn($index, item)`.
func isTrackByFunctionCall(rootView ir.XrefId, expr output.Expression) bool {
	invoke, ok := expr.(*output.InvokeFunctionExpr)
	if !ok || len(invoke.Args) == 0 || len(invoke.Args) > 2 {
		return false
	}
	readProp, ok := invoke.Fn.(*output.ReadPropExpr)
	if !ok {
		return false
	}
	ctxExpr, ok := readProp.Receiver.(*ir.ContextExpr)
	if !ok || ctxExpr.View != rootView {
		return false
	}
	if len(invoke.Args) == 0 {
		return false
	}
	arg0, ok := invoke.Args[0].(*output.ReadVarExpr)
	if !ok || arg0.Name != "$index" {
		return false
	}
	if len(invoke.Args) == 1 {
		return true
	}
	arg1, ok := invoke.Args[1].(*output.ReadVarExpr)
	return ok && arg1.Name == "$item"
}
