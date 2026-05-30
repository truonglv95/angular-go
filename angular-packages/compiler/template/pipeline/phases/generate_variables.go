package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

type Scope struct {
	View                ir.XrefId
	ViewContextVariable *ir.SemanticVariable
	ContextVariables    map[string]*ir.SemanticVariable
	Aliases             []ir.Expression
	References          []ScopeReference
	LetDeclarations     []ScopeLetDeclaration
	Parent              *Scope
}

type ScopeReference struct {
	Name       string
	TargetId   ir.XrefId
	TargetSlot *ir.SlotHandle
	Offset     int
	Variable   *ir.SemanticVariable
}

type ScopeLetDeclaration struct {
	TargetId   ir.XrefId
	TargetSlot *ir.SlotHandle
	Variable   *ir.SemanticVariable
}

func GenerateVariables(job *compilation.ComponentCompilationJob) {
	recursivelyProcessView(job.Root, nil)
}

func recursivelyProcessView(view *compilation.ViewCompilationUnit, parentScope *Scope) {
	scope := getScopeForView(view, parentScope)

	for _, op := range view.GetCreate().Ops {
		switch op.Kind() {
		case ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate, ir.OpKindTemplate:
			xref := getXrefFromOp(op)
			if childView, ok := view.Job.Views[xref]; ok && childView != nil {
				recursivelyProcessView(childView, scope)
			}
		case ir.OpKindProjection:
			if projOp, ok := op.(*ir.ProjectionOp); ok && projOp.FallbackView != 0 {
				if childView, ok := view.Job.Views[projOp.FallbackView]; ok && childView != nil {
					recursivelyProcessView(childView, scope)
				}
			}
		case ir.OpKindRepeaterCreate:
			if repOp, ok := op.(*ir.RepeaterCreateOp); ok {
				if childView, ok := view.Job.Views[repOp.Xref]; ok && childView != nil {
					recursivelyProcessView(childView, scope)
				}
				if repOp.EmptyView != 0 {
					if emptyView, ok := view.Job.Views[repOp.EmptyView]; ok && emptyView != nil {
						recursivelyProcessView(emptyView, scope)
					}
				}
				if repOp.TrackByOps != nil {
					if trackOps, ok := repOp.TrackByOps.(*ir.OpList); ok && trackOps != nil {
						trackOps.Prepend(generateVariablesInScopeForView(view, scope, false))
					}
				}
			}
		case ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindListener, ir.OpKindTwoWayListener:
			if listener, ok := op.(ir.ListenerTrait); ok {
				if handlerOps := listener.HandlerOps(); handlerOps != nil {
					handlerOps.Prepend(generateVariablesInScopeForView(view, scope, true))
				}
			}
		}
	}

	view.GetUpdate().Prepend(generateVariablesInScopeForView(view, scope, false))

	for _, fnExpr := range view.GetFunctions() {
		if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok && arrowFn != nil {
			if arrowFn.Ops != nil {
				arrowFn.Ops.Prepend(generateVariablesInScopeForView(view, getScopeForView(view, parentScope), true))
			}
		}
	}
}

func getXrefFromOp(op ir.Op) ir.XrefId {
	if trait, ok := op.(ir.ConsumesSlotOpTrait); ok {
		return trait.GetXref()
	}
	return 0
}

func getScopeForView(view *compilation.ViewCompilationUnit, parent *Scope) *Scope {
	scope := &Scope{
		View: view.Xref,
		ViewContextVariable: &ir.SemanticVariable{
			Kind: ir.SemanticVariableKindContext,
			View: view.Xref,
		},
		ContextVariables: make(map[string]*ir.SemanticVariable),
		Aliases:          view.Aliases,
		Parent:           parent,
	}

	for identifier := range view.ContextVariables {
		scope.ContextVariables[identifier] = &ir.SemanticVariable{
			Kind:       ir.SemanticVariableKindIdentifier,
			Identifier: identifier,
			Local:      false,
		}
	}

	for _, op := range view.GetCreate().Ops {
		kind := op.Kind()
		if kind == ir.OpKindElementStart || kind == ir.OpKindConditionalCreate || kind == ir.OpKindConditionalBranchCreate || kind == ir.OpKindTemplate {
			var localRefs []ir.LocalRef
			if trait, ok := op.(ir.LocalRefsTrait); ok {
				localRefs = trait.LocalRefs()
			}

			var xref ir.XrefId
			var handle *ir.SlotHandle
			if slot, ok := op.(interface {
				GetXref() ir.XrefId
				Handle() *ir.SlotHandle
			}); ok {
				xref = slot.GetXref()
				handle = slot.Handle()
			}

			for offset, ref := range localRefs {
				scope.References = append(scope.References, ScopeReference{
					Name:       ref.Name,
					TargetId:   xref,
					TargetSlot: handle,
					Offset:     offset,
					Variable: &ir.SemanticVariable{
						Kind:       ir.SemanticVariableKindIdentifier,
						Identifier: ref.Name,
						Local:      false,
					},
				})
			}
		} else if kind == ir.OpKindDeclareLet {
			if letOp, ok := op.(*ir.DeclareLetOp); ok {
				scope.LetDeclarations = append(scope.LetDeclarations, ScopeLetDeclaration{
					TargetId:   letOp.Xref,
					TargetSlot: letOp.Handle(),
					Variable: &ir.SemanticVariable{
						Kind:       ir.SemanticVariableKindIdentifier,
						Identifier: letOp.DeclaredName,
						Local:      false,
					},
				})
			}
		}
	}

	return scope
}

func generateVariablesInScopeForView(
	view *compilation.ViewCompilationUnit,
	scope *Scope,
	isCallback bool,
) []ir.Op {
	var newOps []ir.Op

	if scope.View != view.Xref {
		newOps = append(newOps, &ir.VariableOp{
			Xref:        view.Job.AllocateXrefId(),
			Variable:    scope.ViewContextVariable,
			Initializer: ir.NewNextContextExpr(),
			Flags:       ir.VariableFlagsNone,
		})
	}

	scopeView := view.Job.Views[scope.View]
	if scopeView != nil {
		for name, value := range scopeView.ContextVariables {
			context := ir.NewContextExpr(scope.View)
			var variable output.Expression
			if value == ir.CTX_REF {
				variable = context
			} else {
				variable = output.NewReadPropExpr(context, value, nil, nil, nil, false)
			}
			newOps = append(newOps, &ir.VariableOp{
				Xref:        view.Job.AllocateXrefId(),
				Variable:    scope.ContextVariables[name],
				Initializer: variable,
				Flags:       ir.VariableFlagsNone,
			})
		}

		for _, aliasExpr := range scopeView.Aliases {
			if aliasVar, ok := any(aliasExpr).(*ir.SemanticVariable); ok && aliasVar != nil {
				newOps = append(newOps, &ir.VariableOp{
					Xref:        view.Job.AllocateXrefId(),
					Variable:    aliasVar,
					Initializer: aliasVar.Expression.Clone(),
					Flags:       ir.VariableFlagsAlwaysInline,
				})
			}
		}
	}

	for _, ref := range scope.References {
		newOps = append(newOps, &ir.VariableOp{
			Xref:        view.Job.AllocateXrefId(),
			Variable:    ref.Variable,
			Initializer: ir.NewReferenceExpr(ref.TargetId, ref.TargetSlot, ref.Offset),
			Flags:       ir.VariableFlagsNone,
		})
	}

	if scope.View != view.Xref || isCallback {
		for _, decl := range scope.LetDeclarations {
			newOps = append(newOps, &ir.VariableOp{
				Xref:        view.Job.AllocateXrefId(),
				Variable:    decl.Variable,
				Initializer: ir.NewContextLetReferenceExpr(decl.TargetId, decl.TargetSlot),
				Flags:       ir.VariableFlagsNone,
			})
		}
	}

	if scope.Parent != nil {
		newOps = append(newOps, generateVariablesInScopeForView(view, scope.Parent, false)...)
	}

	return newOps
}
