package phases

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ResolveNames resolves lexical references in views (LexicalReadExpr) to either a target variable
// or to property reads on the top-level component context.
// Also matches RestoreViewExpr expressions with the variables of their corresponding saved views.
func ResolveNames(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, fnExpr := range unit.GetFunctions() {
			if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok {
				ops := arrowFn.Ops.Elements()
				processLexicalScope(unit, ops, nil)
			}
		}
		processLexicalScope(unit, unit.GetCreate().Elements(), nil)
		processLexicalScope(unit, unit.GetUpdate().Elements(), nil)
	}
}

type savedViewInfo struct {
	view     ir.XrefId
	variable ir.XrefId
}

func processLexicalScope(unit compilation.CompilationUnit, ops []ir.Op, sv *savedViewInfo) {
	scope := map[string]ir.XrefId{}
	localDefinitions := map[string]ir.XrefId{}

	for _, op := range ops {
		switch op.Kind() {
		case ir.OpKindVariable:
			if v, ok := op.(*ir.VariableOp); ok {
				variable := v.Variable
				switch variable.Kind {
				case ir.SemanticVariableKindIdentifier:
					if variable.Local {
						if _, exists := localDefinitions[variable.Identifier]; exists {
							continue
						}
						localDefinitions[variable.Identifier] = v.Xref
					} else if _, exists := scope[variable.Identifier]; exists {
						continue
					}
					scope[variable.Identifier] = v.Xref
				case ir.SemanticVariableKindAlias:
					if _, exists := scope[variable.Identifier]; exists {
						continue
					}
					scope[variable.Identifier] = v.Xref
				case ir.SemanticVariableKindSavedView:
					sv = &savedViewInfo{
						view:     variable.View,
						variable: v.Xref,
					}
				}
			}
		case ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindListener, ir.OpKindTwoWayListener:
			if lOp, ok := op.(ir.ListenerTrait); ok {
				processLexicalScope(unit, lOp.HandlerOps().Elements(), sv)
			}
		case ir.OpKindRepeaterCreate:
			if r, ok := op.(*ir.RepeaterCreateOp); ok {
				if r.TrackByOps != nil {
					if trackByOpList, ok2 := r.TrackByOps.(*ir.OpList); ok2 {
						processLexicalScope(unit, trackByOpList.Elements(), sv)
					}
				}
			}
		}
	}

	// Resolve LexicalReadExprs and RestoreViewExprs.
	for _, op := range ops {
		switch op.Kind() {
		case ir.OpKindListener, ir.OpKindTwoWayListener, ir.OpKindAnimation, ir.OpKindAnimationListener:
			continue
		}
		ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			if lex, ok := expr.(*ir.LexicalReadExpr); ok {
				if xref, found := localDefinitions[lex.Name]; found {
					return ir.NewReadVariableExpr(xref)
				} else if xref, found := scope[lex.Name]; found {
					return ir.NewReadVariableExpr(xref)
				} else {
					ctxExpr := ir.NewContextExpr(unit.GetJob().GetRoot().GetXref())
					return output.NewReadPropExpr(ctxExpr, lex.Name, nil, nil, nil, false)
				}
			} else if restore, ok := expr.(*ir.RestoreViewExpr); ok {
				if restore.ViewXref != nil {
					viewId := *restore.ViewXref
					if sv == nil || sv.view != viewId {
						panic(fmt.Sprintf("AssertionError: no saved view %v from view %v", viewId, unit.GetXref()))
					}
					restore.ViewXref = nil
					restore.ViewExpr = ir.NewReadVariableExpr(sv.variable)
					return restore
				}
			}
			return expr
		}, ir.VisitorContextFlagNone)
	}

	// Verify no LexicalReadExprs remain.
	for _, op := range ops {
		ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
			if lex, ok := expr.(*ir.LexicalReadExpr); ok {
				panic(fmt.Sprintf("AssertionError: no lexical reads should remain, but found read of %s", lex.Name))
			}
		})
	}
}
