package ir

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type InterpolateTextOp struct {
	OpBase
	Target        XrefId
	Interpolation any
	SourceSpan    *parse_util.ParseSourceSpan
}

func (o *InterpolateTextOp) Kind() OpKind {
	return OpKindInterpolateText
}

func (o *InterpolateTextOp) ConsumesVars() bool {
	return true
}

func (o *InterpolateTextOp) GetTarget() XrefId {
	return o.Target
}

type Interpolation struct {
	output.Expression
	Strings          []string
	Expressions      []output.Expression
	I18nPlaceholders []string
}

func NewInterpolation(strings []string, expressions []output.Expression, i18nPlaceholders []string) *Interpolation {
	if len(i18nPlaceholders) != 0 && len(i18nPlaceholders) != len(expressions) {
		panic("Expected interpolation placeholder count to match expression count")
	}
	return &Interpolation{
		Expression:       output.NewLiteralExpr(nil, nil, nil, nil),
		Strings:          strings,
		Expressions:      expressions,
		I18nPlaceholders: i18nPlaceholders,
	}
}

func CreateInterpolateTextOp(
	target XrefId,
	interpolation *Interpolation,
	sourceSpan *parse_util.ParseSourceSpan,
) *InterpolateTextOp {
	return &InterpolateTextOp{
		Target:        target,
		Interpolation: interpolation,
		SourceSpan:    sourceSpan,
	}
}

type BindingOp struct {
	OpBase
	Target                        XrefId
	BindingKind                   any
	Name                          string
	Expression                    output.Expression
	Unit                          *string
	SecurityContext               any
	IsTextAttribute               bool
	IsStructuralTemplateAttribute bool
	TemplateKind                  any
	I18nContext                   XrefId
	I18nMessage                   any
	SourceSpan                    *parse_util.ParseSourceSpan
}

func (o *BindingOp) Kind() OpKind {
	return OpKindBinding
}

func CreateBindingOp(
	target XrefId,
	bindingKind any,
	name string,
	expression output.Expression,
	unit *string,
	securityContext any,
	isTextAttribute bool,
	isStructuralTemplateAttribute bool,
	templateKind any,
	i18nMessage any,
	sourceSpan *parse_util.ParseSourceSpan,
) *BindingOp {
	return &BindingOp{
		Target:                        target,
		BindingKind:                   bindingKind,
		Name:                          name,
		Expression:                    expression,
		Unit:                          unit,
		SecurityContext:               securityContext,
		IsTextAttribute:               isTextAttribute,
		IsStructuralTemplateAttribute: isStructuralTemplateAttribute,
		TemplateKind:                  templateKind,
		I18nMessage:                   i18nMessage,
		SourceSpan:                    sourceSpan,
	}
}

type PropertyOp struct {
	OpBase
	Target                        XrefId
	Name                          string
	Expression                    output.Expression
	BindingKind                   any
	SecurityContext               any
	Sanitizer                     output.Expression
	IsStructuralTemplateAttribute bool
	TemplateKind                  any
	I18nContext                   XrefId
	I18nMessage                   any
	SourceSpan                    *parse_util.ParseSourceSpan
}

func (o *PropertyOp) Kind() OpKind {
	return OpKindProperty
}

func (o *PropertyOp) GetTarget() XrefId { return o.Target }

func (o *PropertyOp) ConsumesVars() bool {
	return true
}

type TwoWayPropertyOp struct {
	OpBase
	Target                        XrefId
	Name                          string
	Expression                    output.Expression
	SecurityContext               any
	Sanitizer                     output.Expression
	IsStructuralTemplateAttribute bool
	TemplateKind                  any
	I18nContext                   XrefId
	I18nMessage                   any
	SourceSpan                    *parse_util.ParseSourceSpan
}

func (o *TwoWayPropertyOp) Kind() OpKind {
	return OpKindTwoWayProperty
}

func (o *TwoWayPropertyOp) GetTarget() XrefId { return o.Target }

func (o *TwoWayPropertyOp) ConsumesVars() bool {
	return true
}

type StylePropOp struct {
	OpBase
	Target     XrefId
	Name       string
	Expression output.Expression
	Unit       *string
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *StylePropOp) Kind() OpKind {
	return OpKindStyleProp
}

func (o *StylePropOp) GetTarget() XrefId { return o.Target }

func (o *StylePropOp) ConsumesVars() bool {
	return true
}

type ClassPropOp struct {
	OpBase
	Target     XrefId
	Name       string
	Expression output.Expression
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *ClassPropOp) Kind() OpKind {
	return OpKindClassProp
}

func (o *ClassPropOp) GetTarget() XrefId { return o.Target }

func (o *ClassPropOp) ConsumesVars() bool {
	return true
}

type StyleMapOp struct {
	OpBase
	Target     XrefId
	Expression output.Expression
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *StyleMapOp) Kind() OpKind {
	return OpKindStyleMap
}

func (o *StyleMapOp) GetTarget() XrefId { return o.Target }

func (o *StyleMapOp) ConsumesVars() bool {
	return true
}

type ClassMapOp struct {
	OpBase
	Target     XrefId
	Expression output.Expression
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *ClassMapOp) Kind() OpKind {
	return OpKindClassMap
}

func (o *ClassMapOp) GetTarget() XrefId { return o.Target }

func (o *ClassMapOp) ConsumesVars() bool {
	return true
}

type AttributeOp struct {
	OpBase
	Target                        XrefId
	Namespace                     *string
	Name                          string
	Expression                    output.Expression
	SecurityContext               any
	Sanitizer                     output.Expression
	IsTextAttribute               bool
	IsStructuralTemplateAttribute bool
	TemplateKind                  any
	I18nContext                   XrefId
	I18nMessage                   any
	SourceSpan                    *parse_util.ParseSourceSpan
}

func (o *AttributeOp) Kind() OpKind {
	return OpKindAttribute
}

func (o *AttributeOp) GetTarget() XrefId { return o.Target }

func (o *AttributeOp) ConsumesVars() bool {
	return true
}

type AdvanceOp struct {
	OpBase
	Delta      int
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *AdvanceOp) Kind() OpKind {
	return OpKindAdvance
}

type ConditionalOp struct {
	OpBase
	Target       XrefId
	Test         output.Expression
	Conditions   any
	Processed    output.Expression
	ContextValue output.Expression
	SourceSpan   *parse_util.ParseSourceSpan
}

func (o *ConditionalOp) Kind() OpKind {
	return OpKindConditional
}

func (o *ConditionalOp) GetTarget() XrefId { return o.Target }

func (o *ConditionalOp) ConsumesVars() bool {
	return true
}

func CreateConditionalOp(
	target XrefId,
	test output.Expression,
	conditions any,
	sourceSpan *parse_util.ParseSourceSpan,
) *ConditionalOp {
	return &ConditionalOp{
		Target:     target,
		Test:       test,
		Conditions: conditions,
		SourceSpan: sourceSpan,
	}
}

type RepeaterOp struct {
	OpBase
	Target     XrefId
	TargetSlot any
	Collection output.Expression
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *RepeaterOp) Kind() OpKind {
	return OpKindRepeater
}

func (o *RepeaterOp) GetTarget() XrefId { return o.Target }

func CreateRepeaterOp(
	target XrefId,
	targetSlot *SlotHandle,
	collection output.Expression,
	sourceSpan *parse_util.ParseSourceSpan,
) *RepeaterOp {
	return &RepeaterOp{
		Target:     target,
		TargetSlot: targetSlot,
		Collection: collection,
		SourceSpan: sourceSpan,
	}
}

type AnimationBindingOp struct {
	OpBase
	Name                 string
	Target               XrefId
	AnimationKind        any
	Expression           output.Expression
	I18nMessage          XrefId
	SecurityContext      any
	Sanitizer            output.Expression
	SourceSpan           *parse_util.ParseSourceSpan
	AnimationBindingKind any
}

func (o *AnimationBindingOp) Kind() OpKind {
	return OpKindAnimationBinding
}

type DeferWhenOp struct {
	OpBase
	Target     XrefId
	Expr       output.Expression
	Modifier   any
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *DeferWhenOp) Kind() OpKind {
	return OpKindDeferWhen
}

func (o *DeferWhenOp) GetTarget() XrefId { return o.Target }

func (o *DeferWhenOp) ConsumesVars() bool {
	return true
}

func CreateDeferWhenOp(
	target XrefId,
	expr output.Expression,
	modifier any,
	sourceSpan *parse_util.ParseSourceSpan,
) *DeferWhenOp {
	return &DeferWhenOp{
		Target:     target,
		Expr:       expr,
		Modifier:   modifier,
		SourceSpan: sourceSpan,
	}
}

type I18nExpressionOp struct {
	OpBase
	Context         XrefId
	Target          XrefId
	I18nOwner       XrefId
	Handle          any
	Expression      output.Expression
	IcuPlaceholder  XrefId
	I18nPlaceholder *string
	ResolutionTime  any
	Usage           any
	Name            string
	SourceSpan      *parse_util.ParseSourceSpan
}

func (o *I18nExpressionOp) Kind() OpKind {
	return OpKindI18nExpression
}

func (o *I18nExpressionOp) GetTarget() XrefId { return o.Target }

func (o *I18nExpressionOp) ConsumesVars() bool {
	return true
}

type I18nApplyOp struct {
	OpBase
	Owner      XrefId
	Handle     any
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *I18nApplyOp) Kind() OpKind {
	return OpKindI18nApply
}

type StoreLetOp struct {
	OpBase
	SourceSpan   *parse_util.ParseSourceSpan
	DeclaredName string
	Target       XrefId
	Value        output.Expression
}

func (o *StoreLetOp) Kind() OpKind {
	return OpKindStoreLet
}

func (o *StoreLetOp) ConsumesVars() bool {
	return true
}

func CreateStoreLetOp(
	target XrefId,
	declaredName string,
	value output.Expression,
	sourceSpan *parse_util.ParseSourceSpan,
) *StoreLetOp {
	return &StoreLetOp{
		Target:       target,
		DeclaredName: declaredName,
		Value:        value,
		SourceSpan:   sourceSpan,
	}
}

type ControlOp struct {
	OpBase
	Target     XrefId
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *ControlOp) Kind() OpKind {
	return OpKindControl
}

func (o *ControlOp) GetTarget() XrefId { return o.Target }
