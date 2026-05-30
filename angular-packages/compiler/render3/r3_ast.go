package render3

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	expression_parser "github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

type Node interface {
	GetSourceSpan() parse_util.ParseSourceSpan
	Visit(visitor Visitor) interface{}
}

type Comment struct {
	Value      string
	SourceSpan parse_util.ParseSourceSpan
}

func (c *Comment) GetSourceSpan() parse_util.ParseSourceSpan { return c.SourceSpan }
func (c *Comment) Visit(visitor Visitor) interface{} {
	panic("visit() not implemented for Comment")
}

type Text struct {
	Value      string
	SourceSpan parse_util.ParseSourceSpan
}

func (t *Text) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Text) Visit(visitor Visitor) interface{} {
	return visitor.VisitText(t)
}

type BoundText struct {
	Value      expression_parser.AST
	SourceSpan parse_util.ParseSourceSpan
	I18n       i18n.I18nMeta
}

func (t *BoundText) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *BoundText) Visit(visitor Visitor) interface{} {
	return visitor.VisitBoundText(t)
}

type TextAttribute struct {
	Name       string
	Value      string
	SourceSpan parse_util.ParseSourceSpan
	KeySpan    *parse_util.ParseSourceSpan
	ValueSpan  *parse_util.ParseSourceSpan
	I18n       i18n.I18nMeta
}

func (t *TextAttribute) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *TextAttribute) Visit(visitor Visitor) interface{} {
	return visitor.VisitTextAttribute(t)
}

type BoundAttribute struct {
	Name            string
	Type            int
	SecurityContext core.SecurityContext
	Value           expression_parser.AST
	Unit            *string
	SourceSpan      parse_util.ParseSourceSpan
	KeySpan         parse_util.ParseSourceSpan
	ValueSpan       *parse_util.ParseSourceSpan
	I18n            i18n.I18nMeta
}

func toParseSourceSpan(span expression_parser.ParseSourceSpan, baseSpan *parse_util.ParseSourceSpan) parse_util.ParseSourceSpan {
	if baseSpan == nil {
		return parse_util.ParseSourceSpan{}
	}
	start := baseSpan.Start.MoveBy(span.Start - baseSpan.Start.Offset)
	end := baseSpan.Start.MoveBy(span.End - baseSpan.Start.Offset)
	return *parse_util.NewParseSourceSpan(start, end, start, nil)
}

func toParseSourceSpanPtr(span *expression_parser.ParseSourceSpan, baseSpan *parse_util.ParseSourceSpan) *parse_util.ParseSourceSpan {
	if span == nil {
		return nil
	}
	res := toParseSourceSpan(*span, baseSpan)
	return &res
}

func BoundAttributeFromBoundElementProperty(prop any, i18nVal any, baseSpan *parse_util.ParseSourceSpan) *BoundAttribute {
	p := prop.(template_parser.BoundElementProperty)
	var i18nMeta i18n.I18nMeta = nil
	if i18nVal != nil {
		i18nMeta = i18nVal.(i18n.I18nMeta)
	}
	return &BoundAttribute{
		Name:            p.Name,
		Type:            int(p.Type),
		SecurityContext: p.SecurityContext,
		Value:           &p.Expression,
		Unit:            p.Unit,
		SourceSpan:      toParseSourceSpan(p.SourceSpan, baseSpan),
		KeySpan:         toParseSourceSpan(p.KeySpan, baseSpan),
		ValueSpan:       toParseSourceSpanPtr(p.ValueSpan, baseSpan),
		I18n:            i18nMeta,
	}
}

func (t *BoundAttribute) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *BoundAttribute) Visit(visitor Visitor) interface{} {
	return visitor.VisitBoundAttribute(t)
}

type BoundEvent struct {
	Name        string
	Type        template_parser.ParsedEventType
	Handler     expression_parser.AST
	Target      *string
	Phase       *string
	SourceSpan  parse_util.ParseSourceSpan
	HandlerSpan parse_util.ParseSourceSpan
	KeySpan     parse_util.ParseSourceSpan
}

func BoundEventFromParsedEvent(event any, baseSpan *parse_util.ParseSourceSpan) *BoundEvent {
	e := event.(*template_parser.ParsedEvent)
	var target *string = nil
	if e.Type == template_parser.ParsedEventTypeRegular {
		target = e.TargetOrPhase
	}
	var phase *string = nil
	if e.Type == template_parser.ParsedEventTypeLegacyAnimation {
		phase = e.TargetOrPhase
	}
	return &BoundEvent{
		Name:        e.Name,
		Type:        e.Type,
		Handler:     &e.Handler,
		Target:      target,
		Phase:       phase,
		SourceSpan:  toParseSourceSpan(e.SourceSpan, baseSpan),
		HandlerSpan: toParseSourceSpan(e.HandlerSpan, baseSpan),
		KeySpan:     toParseSourceSpan(e.KeySpan, baseSpan),
	}
}

func (t *BoundEvent) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *BoundEvent) Visit(visitor Visitor) interface{} {
	return visitor.VisitBoundEvent(t)
}

type Element struct {
	Name            string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	Directives      []*Directive
	Children        []Node
	References      []*Reference
	IsSelfClosing   bool
	SourceSpan      parse_util.ParseSourceSpan
	StartSourceSpan parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
	IsVoid          bool
	I18n            i18n.I18nMeta
}

func (t *Element) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Element) Visit(visitor Visitor) interface{} {
	return visitor.VisitElement(t)
}

type DeferredTrigger interface {
	Node
	IsDeferredTrigger()
}

type BaseDeferredTrigger struct {
	NameSpan           *parse_util.ParseSourceSpan
	SourceSpan         parse_util.ParseSourceSpan
	PrefetchSpan       *parse_util.ParseSourceSpan
	WhenOrOnSourceSpan *parse_util.ParseSourceSpan
	HydrateSpan        *parse_util.ParseSourceSpan
}

func (t *BaseDeferredTrigger) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *BaseDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}
func (t *BaseDeferredTrigger) IsDeferredTrigger() {}

type BoundDeferredTrigger struct {
	BaseDeferredTrigger
	Value expression_parser.AST
}

func NewBoundDeferredTrigger(value expression_parser.AST, sourceSpan parse_util.ParseSourceSpan, prefetchSpan *parse_util.ParseSourceSpan, whenSourceSpan parse_util.ParseSourceSpan, hydrateSpan *parse_util.ParseSourceSpan) *BoundDeferredTrigger {
	return &BoundDeferredTrigger{
		BaseDeferredTrigger: BaseDeferredTrigger{
			NameSpan:           nil,
			SourceSpan:         sourceSpan,
			PrefetchSpan:       prefetchSpan,
			WhenOrOnSourceSpan: &whenSourceSpan,
			HydrateSpan:        hydrateSpan,
		},
		Value: value,
	}
}

func (t *BoundDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type NeverDeferredTrigger struct {
	BaseDeferredTrigger
}

func (t *NeverDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type IdleDeferredTrigger struct {
	BaseDeferredTrigger
	Timeout *float64
}

func NewIdleDeferredTrigger(nameSpan parse_util.ParseSourceSpan, sourceSpan parse_util.ParseSourceSpan, prefetchSpan *parse_util.ParseSourceSpan, onSourceSpan *parse_util.ParseSourceSpan, hydrateSpan *parse_util.ParseSourceSpan, timeout *float64) *IdleDeferredTrigger {
	return &IdleDeferredTrigger{
		BaseDeferredTrigger: BaseDeferredTrigger{
			NameSpan:           &nameSpan,
			SourceSpan:         sourceSpan,
			PrefetchSpan:       prefetchSpan,
			WhenOrOnSourceSpan: onSourceSpan,
			HydrateSpan:        hydrateSpan,
		},
		Timeout: timeout,
	}
}

func (t *IdleDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type ImmediateDeferredTrigger struct {
	BaseDeferredTrigger
}

func (t *ImmediateDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type HoverDeferredTrigger struct {
	BaseDeferredTrigger
	Reference *string
}

func NewHoverDeferredTrigger(reference *string, nameSpan parse_util.ParseSourceSpan, sourceSpan parse_util.ParseSourceSpan, prefetchSpan *parse_util.ParseSourceSpan, onSourceSpan *parse_util.ParseSourceSpan, hydrateSpan *parse_util.ParseSourceSpan) *HoverDeferredTrigger {
	return &HoverDeferredTrigger{
		BaseDeferredTrigger: BaseDeferredTrigger{
			NameSpan:           &nameSpan,
			SourceSpan:         sourceSpan,
			PrefetchSpan:       prefetchSpan,
			WhenOrOnSourceSpan: onSourceSpan,
			HydrateSpan:        hydrateSpan,
		},
		Reference: reference,
	}
}

func (t *HoverDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type TimerDeferredTrigger struct {
	BaseDeferredTrigger
	Delay float64
}

func NewTimerDeferredTrigger(delay float64, nameSpan parse_util.ParseSourceSpan, sourceSpan parse_util.ParseSourceSpan, prefetchSpan *parse_util.ParseSourceSpan, onSourceSpan *parse_util.ParseSourceSpan, hydrateSpan *parse_util.ParseSourceSpan) *TimerDeferredTrigger {
	return &TimerDeferredTrigger{
		BaseDeferredTrigger: BaseDeferredTrigger{
			NameSpan:           &nameSpan,
			SourceSpan:         sourceSpan,
			PrefetchSpan:       prefetchSpan,
			WhenOrOnSourceSpan: onSourceSpan,
			HydrateSpan:        hydrateSpan,
		},
		Delay: delay,
	}
}

func (t *TimerDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type InteractionDeferredTrigger struct {
	BaseDeferredTrigger
	Reference *string
}

func NewInteractionDeferredTrigger(reference *string, nameSpan parse_util.ParseSourceSpan, sourceSpan parse_util.ParseSourceSpan, prefetchSpan *parse_util.ParseSourceSpan, onSourceSpan *parse_util.ParseSourceSpan, hydrateSpan *parse_util.ParseSourceSpan) *InteractionDeferredTrigger {
	return &InteractionDeferredTrigger{
		BaseDeferredTrigger: BaseDeferredTrigger{
			NameSpan:           &nameSpan,
			SourceSpan:         sourceSpan,
			PrefetchSpan:       prefetchSpan,
			WhenOrOnSourceSpan: onSourceSpan,
			HydrateSpan:        hydrateSpan,
		},
		Reference: reference,
	}
}

func (t *InteractionDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type ViewportDeferredTrigger struct {
	BaseDeferredTrigger
	Reference *string
}

func NewViewportDeferredTrigger(reference *string, nameSpan parse_util.ParseSourceSpan, sourceSpan parse_util.ParseSourceSpan, prefetchSpan *parse_util.ParseSourceSpan, onSourceSpan *parse_util.ParseSourceSpan, hydrateSpan *parse_util.ParseSourceSpan) *ViewportDeferredTrigger {
	return &ViewportDeferredTrigger{
		BaseDeferredTrigger: BaseDeferredTrigger{
			NameSpan:           &nameSpan,
			SourceSpan:         sourceSpan,
			PrefetchSpan:       prefetchSpan,
			WhenOrOnSourceSpan: onSourceSpan,
			HydrateSpan:        hydrateSpan,
		},
		Reference: reference,
	}
}

func (t *ViewportDeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(t)
}

type BlockNode struct {
	NameSpan        parse_util.ParseSourceSpan
	SourceSpan      parse_util.ParseSourceSpan
	StartSourceSpan parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
}

type DeferredBlockPlaceholder struct {
	BlockNode
	Children    []Node
	MinimumTime *float64
	I18n        i18n.I18nMeta
}

func (t *DeferredBlockPlaceholder) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *DeferredBlockPlaceholder) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlockPlaceholder(t)
}

type DeferredBlockLoading struct {
	BlockNode
	Children    []Node
	AfterTime   *float64
	MinimumTime *float64
	I18n        i18n.I18nMeta
}

func (t *DeferredBlockLoading) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *DeferredBlockLoading) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlockLoading(t)
}

type DeferredBlockError struct {
	BlockNode
	Children []Node
	I18n     i18n.I18nMeta
}

func (t *DeferredBlockError) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *DeferredBlockError) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlockError(t)
}

type DeferredBlockTriggers struct {
	When        *BoundDeferredTrigger
	Idle        *IdleDeferredTrigger
	Immediate   *ImmediateDeferredTrigger
	Hover       *HoverDeferredTrigger
	Timer       *TimerDeferredTrigger
	Interaction *InteractionDeferredTrigger
	Viewport    *ViewportDeferredTrigger
	Never       *NeverDeferredTrigger
}

type DeferredBlock struct {
	BlockNode
	Children         []Node
	Triggers         DeferredBlockTriggers
	PrefetchTriggers DeferredBlockTriggers
	HydrateTriggers  DeferredBlockTriggers
	Placeholder      *DeferredBlockPlaceholder
	Loading          *DeferredBlockLoading
	Error            *DeferredBlockError
	MainBlockSpan    parse_util.ParseSourceSpan
	I18n             i18n.I18nMeta
}

func (t *DeferredBlock) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *DeferredBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlock(t)
}

func (t *DeferredBlock) VisitAll(visitor Visitor) {
	visitTriggers(t.HydrateTriggers, visitor)
	visitTriggers(t.Triggers, visitor)
	visitTriggers(t.PrefetchTriggers, visitor)
	VisitAll(visitor, t.Children)

	var remainingBlocks []Node
	if t.Placeholder != nil {
		remainingBlocks = append(remainingBlocks, t.Placeholder)
	}
	if t.Loading != nil {
		remainingBlocks = append(remainingBlocks, t.Loading)
	}
	if t.Error != nil {
		remainingBlocks = append(remainingBlocks, t.Error)
	}
	VisitAll(visitor, remainingBlocks)
}

func visitTriggers(triggers DeferredBlockTriggers, visitor Visitor) {
	if triggers.When != nil {
		triggers.When.Visit(visitor)
	}
	if triggers.Idle != nil {
		triggers.Idle.Visit(visitor)
	}
	if triggers.Immediate != nil {
		triggers.Immediate.Visit(visitor)
	}
	if triggers.Hover != nil {
		triggers.Hover.Visit(visitor)
	}
	if triggers.Timer != nil {
		triggers.Timer.Visit(visitor)
	}
	if triggers.Interaction != nil {
		triggers.Interaction.Visit(visitor)
	}
	if triggers.Viewport != nil {
		triggers.Viewport.Visit(visitor)
	}
	if triggers.Never != nil {
		triggers.Never.Visit(visitor)
	}
}

type SwitchBlock struct {
	BlockNode
	Expression      expression_parser.AST
	Groups          []*SwitchBlockCaseGroup
	UnknownBlocks   []*UnknownBlock
	ExhaustiveCheck *SwitchExhaustiveCheck
}

func (t *SwitchBlock) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *SwitchBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitSwitchBlock(t)
}

type SwitchBlockCase struct {
	BlockNode
	Expression expression_parser.AST
}

func (t *SwitchBlockCase) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *SwitchBlockCase) Visit(visitor Visitor) interface{} {
	return visitor.VisitSwitchBlockCase(t)
}

type SwitchBlockCaseGroup struct {
	BlockNode
	Cases    []*SwitchBlockCase
	Children []Node
	I18n     i18n.I18nMeta
}

func (t *SwitchBlockCaseGroup) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *SwitchBlockCaseGroup) Visit(visitor Visitor) interface{} {
	return visitor.VisitSwitchBlockCaseGroup(t)
}

type SwitchExhaustiveCheck struct {
	BlockNode
	Expression expression_parser.AST
}

func (t *SwitchExhaustiveCheck) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *SwitchExhaustiveCheck) Visit(visitor Visitor) interface{} {
	return visitor.VisitSwitchExhaustiveCheck(t)
}

type ForLoopBlock struct {
	BlockNode
	Item             *Variable
	Expression       expression_parser.ASTWithSource
	TrackBy          *expression_parser.ASTWithSource
	TrackKeywordSpan *parse_util.ParseSourceSpan
	ContextVariables []*Variable
	Children         []Node
	Empty            *ForLoopBlockEmpty
	MainBlockSpan    parse_util.ParseSourceSpan
	I18n             i18n.I18nMeta
}

func (t *ForLoopBlock) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *ForLoopBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitForLoopBlock(t)
}

type ForLoopBlockEmpty struct {
	BlockNode
	Children []Node
	I18n     i18n.I18nMeta
}

func (t *ForLoopBlockEmpty) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *ForLoopBlockEmpty) Visit(visitor Visitor) interface{} {
	return visitor.VisitForLoopBlockEmpty(t)
}

type IfBlock struct {
	BlockNode
	Branches []*IfBlockBranch
}

func (t *IfBlock) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *IfBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitIfBlock(t)
}

type IfBlockBranch struct {
	BlockNode
	Expression      expression_parser.AST
	Children        []Node
	ExpressionAlias *Variable
	I18n            i18n.I18nMeta
}

func (t *IfBlockBranch) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *IfBlockBranch) Visit(visitor Visitor) interface{} {
	return visitor.VisitIfBlockBranch(t)
}

type UnknownBlock struct {
	Name       string
	SourceSpan parse_util.ParseSourceSpan
	NameSpan   parse_util.ParseSourceSpan
}

func (t *UnknownBlock) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *UnknownBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitUnknownBlock(t)
}

type LetDeclaration struct {
	Name       string
	Value      expression_parser.AST
	SourceSpan parse_util.ParseSourceSpan
	NameSpan   parse_util.ParseSourceSpan
	ValueSpan  parse_util.ParseSourceSpan
}

func (t *LetDeclaration) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *LetDeclaration) Visit(visitor Visitor) interface{} {
	return visitor.VisitLetDeclaration(t)
}

type Component struct {
	ComponentName   string
	TagName         *string
	FullName        string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	Directives      []*Directive
	Children        []Node
	References      []*Reference
	IsSelfClosing   bool
	SourceSpan      parse_util.ParseSourceSpan
	StartSourceSpan parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
	I18n            i18n.I18nMeta
}

func (t *Component) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Component) Visit(visitor Visitor) interface{} {
	return visitor.VisitComponent(t)
}

type Directive struct {
	Name            string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	References      []*Reference
	SourceSpan      parse_util.ParseSourceSpan
	StartSourceSpan parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
	I18n            i18n.I18nMeta
}

func (t *Directive) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Directive) Visit(visitor Visitor) interface{} {
	return visitor.VisitDirective(t)
}

type Template struct {
	TagName         *string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	Directives      []*Directive
	TemplateAttrs   []Node // BoundAttribute | TextAttribute
	Children        []Node
	References      []*Reference
	Variables       []*Variable
	IsSelfClosing   bool
	SourceSpan      parse_util.ParseSourceSpan
	StartSourceSpan parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
	I18n            i18n.I18nMeta
}

func (t *Template) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Template) Visit(visitor Visitor) interface{} {
	return visitor.VisitTemplate(t)
}

type Content struct {
	Name            string
	Selector        string
	Attributes      []*TextAttribute
	Children        []Node
	IsSelfClosing   bool
	SourceSpan      parse_util.ParseSourceSpan
	StartSourceSpan parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
	I18n            i18n.I18nMeta
}

func NewContent(selector string, attributes []*TextAttribute, children []Node, isSelfClosing bool, sourceSpan parse_util.ParseSourceSpan, startSourceSpan parse_util.ParseSourceSpan, endSourceSpan *parse_util.ParseSourceSpan, i18n i18n.I18nMeta) *Content {
	return &Content{
		Name:            "ng-content",
		Selector:        selector,
		Attributes:      attributes,
		Children:        children,
		IsSelfClosing:   isSelfClosing,
		SourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		I18n:            i18n,
	}
}

func (t *Content) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Content) Visit(visitor Visitor) interface{} {
	return visitor.VisitContent(t)
}

type Variable struct {
	Name       string
	Value      string
	SourceSpan parse_util.ParseSourceSpan
	KeySpan    parse_util.ParseSourceSpan
	ValueSpan  *parse_util.ParseSourceSpan
}

func (t *Variable) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Variable) Visit(visitor Visitor) interface{} {
	return visitor.VisitVariableExpr(t)
}

type Reference struct {
	Name       string
	Value      string
	SourceSpan parse_util.ParseSourceSpan
	KeySpan    parse_util.ParseSourceSpan
	ValueSpan  *parse_util.ParseSourceSpan
}

func (t *Reference) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Reference) Visit(visitor Visitor) interface{} {
	return visitor.VisitReference(t)
}

type Icu struct {
	Vars         map[string]*BoundText
	Placeholders map[string]Node // Text | BoundText
	SourceSpan   parse_util.ParseSourceSpan
	I18n         i18n.I18nMeta
}

func (t *Icu) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Icu) Visit(visitor Visitor) interface{} {
	return visitor.VisitIcu(t)
}

type HostElement struct {
	TagNames   []string
	Bindings   []*BoundAttribute
	Listeners  []*BoundEvent
	SourceSpan parse_util.ParseSourceSpan
}

func NewHostElement(tagNames []string, bindings []*BoundAttribute, listeners []*BoundEvent, sourceSpan parse_util.ParseSourceSpan) *HostElement {
	if len(tagNames) == 0 {
		panic("HostElement must have at least one tag name.")
	}
	return &HostElement{
		TagNames:   tagNames,
		Bindings:   bindings,
		Listeners:  listeners,
		SourceSpan: sourceSpan,
	}
}

func (t *HostElement) GetSourceSpan() parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *HostElement) Visit(visitor Visitor) interface{} {
	panic("HostElement cannot be visited")
}

type Visitor interface {
	Visit(node Node) interface{}
	VisitElement(element *Element) interface{}
	VisitTemplate(template *Template) interface{}
	VisitContent(content *Content) interface{}
	VisitVariableExpr(variable *Variable) interface{}
	VisitReference(reference *Reference) interface{}
	VisitTextAttribute(attribute *TextAttribute) interface{}
	VisitBoundAttribute(attribute *BoundAttribute) interface{}
	VisitBoundEvent(attribute *BoundEvent) interface{}
	VisitText(text *Text) interface{}
	VisitBoundText(text *BoundText) interface{}
	VisitIcu(icu *Icu) interface{}
	VisitDeferredBlock(deferred *DeferredBlock) interface{}
	VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) interface{}
	VisitDeferredBlockError(block *DeferredBlockError) interface{}
	VisitDeferredBlockLoading(block *DeferredBlockLoading) interface{}
	VisitDeferredTrigger(trigger DeferredTrigger) interface{}
	VisitSwitchBlock(block *SwitchBlock) interface{}
	VisitSwitchBlockCase(block *SwitchBlockCase) interface{}
	VisitSwitchBlockCaseGroup(block *SwitchBlockCaseGroup) interface{}
	VisitSwitchExhaustiveCheck(block *SwitchExhaustiveCheck) interface{}
	VisitForLoopBlock(block *ForLoopBlock) interface{}
	VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) interface{}
	VisitIfBlock(block *IfBlock) interface{}
	VisitIfBlockBranch(block *IfBlockBranch) interface{}
	VisitUnknownBlock(block *UnknownBlock) interface{}
	VisitLetDeclaration(decl *LetDeclaration) interface{}
	VisitComponent(component *Component) interface{}
	VisitDirective(directive *Directive) interface{}
}

func VisitAll(visitor Visitor, nodes []Node) []interface{} {
	var result []interface{}
	for _, node := range nodes {
		var newNode interface{}
		if visitor.Visit(node) != nil {
			newNode = visitor.Visit(node)
		} else {
			newNode = node.Visit(visitor)
		}
		if newNode != nil {
			result = append(result, newNode)
		}
	}
	return result
}

type RecursiveVisitor struct {
	Impl Visitor
}

func (v *RecursiveVisitor) visitor() Visitor {
	if v.Impl != nil {
		return v.Impl
	}
	return v
}

func (v *RecursiveVisitor) Visit(node Node) interface{} { return nil }

func (v *RecursiveVisitor) VisitElement(element *Element) interface{} {
	vis := v.visitor()
	for _, attr := range element.Attributes {
		attr.Visit(vis)
	}
	for _, input := range element.Inputs {
		input.Visit(vis)
	}
	for _, output := range element.Outputs {
		output.Visit(vis)
	}
	for _, directive := range element.Directives {
		directive.Visit(vis)
	}
	for _, child := range element.Children {
		child.Visit(vis)
	}
	for _, ref := range element.References {
		ref.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitTemplate(template *Template) interface{} {
	vis := v.visitor()
	for _, attr := range template.Attributes {
		attr.Visit(vis)
	}
	for _, input := range template.Inputs {
		input.Visit(vis)
	}
	for _, output := range template.Outputs {
		output.Visit(vis)
	}
	for _, directive := range template.Directives {
		directive.Visit(vis)
	}
	for _, child := range template.Children {
		child.Visit(vis)
	}
	for _, ref := range template.References {
		ref.Visit(vis)
	}
	for _, variable := range template.Variables {
		variable.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitDeferredBlock(deferred *DeferredBlock) interface{} {
	deferred.VisitAll(v.visitor())
	return nil
}

func (v *RecursiveVisitor) VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) interface{} {
	vis := v.visitor()
	for _, child := range block.Children {
		child.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitDeferredBlockError(block *DeferredBlockError) interface{} {
	vis := v.visitor()
	for _, child := range block.Children {
		child.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitDeferredBlockLoading(block *DeferredBlockLoading) interface{} {
	vis := v.visitor()
	for _, child := range block.Children {
		child.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitSwitchBlock(block *SwitchBlock) interface{} {
	vis := v.visitor()
	for _, group := range block.Groups {
		group.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitSwitchBlockCase(block *SwitchBlockCase) interface{} { return nil }

func (v *RecursiveVisitor) VisitSwitchBlockCaseGroup(block *SwitchBlockCaseGroup) interface{} {
	vis := v.visitor()
	for _, c := range block.Cases {
		c.Visit(vis)
	}
	for _, child := range block.Children {
		child.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitSwitchExhaustiveCheck(block *SwitchExhaustiveCheck) interface{} {
	return nil
}

func (v *RecursiveVisitor) VisitForLoopBlock(block *ForLoopBlock) interface{} {
	vis := v.visitor()
	if block.Item != nil {
		block.Item.Visit(vis)
	}
	for _, cv := range block.ContextVariables {
		cv.Visit(vis)
	}
	for _, child := range block.Children {
		child.Visit(vis)
	}
	if block.Empty != nil {
		block.Empty.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) interface{} {
	vis := v.visitor()
	for _, child := range block.Children {
		child.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitIfBlock(block *IfBlock) interface{} {
	vis := v.visitor()
	for _, branch := range block.Branches {
		branch.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitIfBlockBranch(block *IfBlockBranch) interface{} {
	vis := v.visitor()
	for _, child := range block.Children {
		child.Visit(vis)
	}
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitContent(content *Content) interface{} {
	vis := v.visitor()
	for _, child := range content.Children {
		child.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitComponent(component *Component) interface{} {
	vis := v.visitor()
	for _, attr := range component.Attributes {
		attr.Visit(vis)
	}
	for _, input := range component.Inputs {
		input.Visit(vis)
	}
	for _, output := range component.Outputs {
		output.Visit(vis)
	}
	for _, directive := range component.Directives {
		directive.Visit(vis)
	}
	for _, child := range component.Children {
		child.Visit(vis)
	}
	for _, ref := range component.References {
		ref.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitDirective(directive *Directive) interface{} {
	vis := v.visitor()
	for _, attr := range directive.Attributes {
		attr.Visit(vis)
	}
	for _, input := range directive.Inputs {
		input.Visit(vis)
	}
	for _, output := range directive.Outputs {
		output.Visit(vis)
	}
	for _, ref := range directive.References {
		ref.Visit(vis)
	}
	return nil
}

func (v *RecursiveVisitor) VisitVariableExpr(variable *Variable) interface{}          { return nil }
func (v *RecursiveVisitor) VisitReference(reference *Reference) interface{}           { return nil }
func (v *RecursiveVisitor) VisitTextAttribute(attribute *TextAttribute) interface{}   { return nil }
func (v *RecursiveVisitor) VisitBoundAttribute(attribute *BoundAttribute) interface{} { return nil }
func (v *RecursiveVisitor) VisitBoundEvent(attribute *BoundEvent) interface{}         { return nil }
func (v *RecursiveVisitor) VisitText(text *Text) interface{}                          { return nil }
func (v *RecursiveVisitor) VisitBoundText(text *BoundText) interface{}                { return nil }
func (v *RecursiveVisitor) VisitIcu(icu *Icu) interface{}                             { return nil }
func (v *RecursiveVisitor) VisitDeferredTrigger(trigger DeferredTrigger) interface{}  { return nil }
func (v *RecursiveVisitor) VisitUnknownBlock(block *UnknownBlock) interface{}         { return nil }
func (v *RecursiveVisitor) VisitLetDeclaration(decl *LetDeclaration) interface{}      { return nil }


func (v *Variable) GetName() string  { return v.Name }
func (r *Reference) GetName() string { return r.Name }
func (l *LetDeclaration) GetName() string { return l.Name }
