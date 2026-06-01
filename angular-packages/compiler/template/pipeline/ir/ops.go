package ir

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"

	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

// XrefId is a branded integer type for cross-reference IDs used in IR.
type XrefId int

// ConstIndex is a branded integer type for an index into the component's consts array.
type ConstIndex int

type Op interface {
	// Kind returns the OpKind discriminant.
	Kind() OpKind
	// GetKind returns the OpKind discriminant (alias for Kind).
	GetKind() OpKind
}

// OpBase is the base struct embedded into all Op implementations.
type OpBase struct {
}

func (o *OpBase) GetKind() OpKind { return 0 }

// CreateAdvanceOp creates an AdvanceOp.
func CreateAdvanceOp(delta int, sourceSpan *parse_util.ParseSourceSpan) *AdvanceOp {
	return &AdvanceOp{Delta: delta, SourceSpan: sourceSpan}
}

// DisableBindingsOp disables bindings on an element's descendants (non-bindable).

// CreateDisableBindingsOp creates a DisableBindingsOp.
func CreateDisableBindingsOp(xref XrefId) *DisableBindingsOp {
	return &DisableBindingsOp{Xref: xref}
}

// EnableBindingsOp re-enables bindings after a non-bindable element.

// CreateEnableBindingsOp creates an EnableBindingsOp.
func CreateEnableBindingsOp(xref XrefId) *EnableBindingsOp {
	return &EnableBindingsOp{Xref: xref}
}

// VisitExpressionsInOp calls visitor for every expression contained in the op.
func VisitExpressionsInOp(op Op, visitor func(expr Expression)) {
	TransformExpressionsInOp(op, func(expr output.Expression, flags VisitorContextFlag) output.Expression {
		if irExpr, ok := expr.(Expression); ok {
			visitor(irExpr)
		}
		return expr
	}, VisitorContextFlagNone)
}

// TransformExpressionsInOp transforms all expressions in an op using the given function.
func TransformExpressionsInOp(op Op, transform func(expr output.Expression, flags VisitorContextFlag) output.Expression, flags VisitorContextFlag) {
	if op == nil {
		return
	}
	switch o := op.(type) {
	case *StylePropOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *StyleMapOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *ClassPropOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *ClassMapOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *AnimationStringOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *AnimationBindingOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *BindingOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *PropertyOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
		if o.Sanitizer != nil {
			o.Sanitizer = TransformExpressionsInExpression(o.Sanitizer, transform, flags)
		}
	case *DomPropertyOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if expr, ok := o.Expression.(output.Expression); ok && expr != nil {
			o.Expression = TransformExpressionsInExpression(expr, transform, flags)
		}
		if o.Sanitizer != nil {
			o.Sanitizer = TransformExpressionsInExpression(o.Sanitizer, transform, flags)
		}
	case *AttributeOp:
		if interp, ok := o.Expression.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		} else if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
		if o.Sanitizer != nil {
			o.Sanitizer = TransformExpressionsInExpression(o.Sanitizer, transform, flags)
		}
	case *TwoWayPropertyOp:
		if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
		if o.Sanitizer != nil {
			o.Sanitizer = TransformExpressionsInExpression(o.Sanitizer, transform, flags)
		}
	case *I18nExpressionOp:
		if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
	case *InterpolateTextOp:
		if interp, ok := o.Interpolation.(*Interpolation); ok {
			transformExpressionsInInterpolation(interp, transform, flags)
		}
	case *StatementOp:
		if o.Statement != nil {
			TransformExpressionsInStatement(o.Statement, transform, flags)
		}
	case *CreateStatementOp:
		if o.Statement != nil {
			TransformExpressionsInStatement(o.Statement, transform, flags)
		}
	case *VariableOp:
		if o.Initializer != nil {
			o.Initializer = TransformExpressionsInExpression(o.Initializer, transform, flags)
		}
	case *CreateVariableOp:
		if o.Initializer != nil {
			o.Initializer = TransformExpressionsInExpression(o.Initializer, transform, flags)
		}
	case *ConditionalOp:
		if conds, ok := o.Conditions.([]*ConditionalCaseExpr); ok {
			for _, cond := range conds {
				if cond.Expr != nil {
					cond.Expr = TransformExpressionsInExpression(cond.Expr, transform, flags)
				}
			}
		}
		if o.Processed != nil {
			o.Processed = TransformExpressionsInExpression(o.Processed, transform, flags)
		}
		if o.ContextValue != nil {
			o.ContextValue = TransformExpressionsInExpression(o.ContextValue, transform, flags)
		}
	case *AnimationOp:
		if ops, ok := o.HandlerOps.(*OpList); ok && ops != nil {
			for _, innerOp := range ops.Ops {
				TransformExpressionsInOp(innerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		}
	case *AnimationListenerOp:
		if ops, ok := o.HandlerOps.(*OpList); ok && ops != nil {
			for _, innerOp := range ops.Ops {
				TransformExpressionsInOp(innerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		}
	case *ListenerOp:
		if ops, ok := o.HandlerOps.(*OpList); ok && ops != nil {
			for _, innerOp := range ops.Ops {
				TransformExpressionsInOp(innerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		}
	case *TwoWayListenerOp:
		if o.HandlerOps != nil {
			for _, innerOp := range o.HandlerOps.Ops {
				TransformExpressionsInOp(innerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		}
	case *ExtractedAttributeOp:
		if o.Expression != nil {
			o.Expression = TransformExpressionsInExpression(o.Expression, transform, flags)
		}
		if o.TrustedValueFn != nil {
			o.TrustedValueFn = TransformExpressionsInExpression(o.TrustedValueFn, transform, flags)
		}
	case *RepeaterCreateOp:
		if o.TrackByOps == nil {
			if o.Track != nil {
				o.Track = TransformExpressionsInExpression(o.Track, transform, flags)
			}
		} else if ops, ok := o.TrackByOps.(*OpList); ok && ops != nil {
			for _, innerOp := range ops.Ops {
				TransformExpressionsInOp(innerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		}
		if o.TrackByFn != nil {
			o.TrackByFn = TransformExpressionsInExpression(o.TrackByFn, transform, flags)
		}
	case *RepeaterOp:
		if o.Collection != nil {
			o.Collection = TransformExpressionsInExpression(o.Collection, transform, flags)
		}
	case *DeferOp:
		if o.LoadingConfig != nil {
			o.LoadingConfig = TransformExpressionsInExpression(o.LoadingConfig, transform, flags)
		}
		if o.PlaceholderConfig != nil {
			o.PlaceholderConfig = TransformExpressionsInExpression(o.PlaceholderConfig, transform, flags)
		}
		if o.ResolverFn != nil {
			o.ResolverFn = TransformExpressionsInExpression(o.ResolverFn, transform, flags)
		}
	case *DeferWhenOp:
		if o.Expr != nil {
			o.Expr = TransformExpressionsInExpression(o.Expr, transform, flags)
		}
	case *StoreLetOp:
		if o.Value != nil {
			o.Value = TransformExpressionsInExpression(o.Value, transform, flags)
		}
	case *ForeignComponentOp:
		if o.Props != nil {
			o.Props = TransformExpressionsInExpression(o.Props, transform, flags)
		}
	}
}

// TransformExpressionsInStatement transforms all expressions in a statement.
func TransformExpressionsInStatement(stmt output.Statement, transform ExpressionTransform, flags VisitorContextFlag) {
	switch s := stmt.(type) {
	case *output.ExpressionStatement:
		if s.Expr != nil {
			s.Expr = TransformExpressionsInExpression(s.Expr, transform, flags)
		}
	case *output.ReturnStatement:
		if s.Value != nil {
			s.Value = TransformExpressionsInExpression(s.Value, transform, flags)
		}
	case *output.DeclareVarStmt:
		if s.Value != nil {
			s.Value = TransformExpressionsInExpression(s.Value, transform, flags)
		}
	case *output.IfStmt:
		if s.Condition != nil {
			s.Condition = TransformExpressionsInExpression(s.Condition, transform, flags)
		}
		for _, caseStmt := range s.TrueCase {
			TransformExpressionsInStatement(caseStmt, transform, flags)
		}
		for _, caseStmt := range s.FalseCase {
			TransformExpressionsInStatement(caseStmt, transform, flags)
		}
	}
}

func transformExpressionsInInterpolation(interp *Interpolation, transform ExpressionTransform, flags VisitorContextFlag) {
	for i, expr := range interp.Expressions {
		interp.Expressions[i] = TransformExpressionsInExpression(expr, transform, flags)
	}
}

// i18n package usage marker (prevents unused import error)
var _ = (*i18n.Message)(nil)
