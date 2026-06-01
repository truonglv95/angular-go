package ir

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type LocalRef struct {
	Name   string
	Target string
}

type ElementOrContainerOpBase struct {
	OpBase
	ConsumesSlotOpTrait
	Xref            XrefId
	SlotHandle      *SlotHandle
	NumSlotsUsed    int
	Attributes      any
	LocalRefsField  any
	NonBindable     bool
	StartSourceSpan *parse_util.ParseSourceSpan
	WholeSourceSpan *parse_util.ParseSourceSpan
}

func (o *ElementOrContainerOpBase) SetAttributes(attrs any) {
	o.Attributes = attrs
}

func (o *ElementOrContainerOpBase) Handle() *SlotHandle {
	if o.SlotHandle == nil {
		o.SlotHandle = NewSlotHandle()
	}
	return o.SlotHandle
}

func (o *ElementOrContainerOpBase) AddNumSlotsUsed(n int) {
	o.NumSlotsUsed += n
}

func (o *ElementOrContainerOpBase) GetNumSlotsUsed() int {
	return o.NumSlotsUsed
}

func (o *ElementOrContainerOpBase) LocalRefs() []LocalRef {
	if o.LocalRefsField == nil {
		return nil
	}
	if refs, ok := o.LocalRefsField.([]LocalRef); ok {
		return refs
	}
	return nil
}

func (o *ElementOrContainerOpBase) GetLocalRefs() any {
	return o.LocalRefsField
}

func (o *ElementOrContainerOpBase) SetLocalRefs(refs any) {
	o.LocalRefsField = refs
}

func (o *ElementOrContainerOpBase) SetLocalRefsExpr(expr output.Expression) {}

func (o *ElementOrContainerOpBase) Kind() OpKind {
	return 0 // TODO: Fix correct kind
}

type ElementOpBase struct {
	ElementOrContainerOpBase
	Tag       *string
	Namespace any
}

type ElementStartOp struct {
	ElementOpBase
	I18nPlaceholder any
	kind            OpKind
}

func (o *ElementStartOp) Kind() OpKind {
	if o.kind != 0 {
		return o.kind
	}
	return OpKindElementStart
}

func (o *ElementStartOp) SetKind(kind OpKind) {
	o.kind = kind
}

func CreateElementStartOp(
	tag string,
	xref XrefId,
	namespace any,
	i18nPlaceholder any,
	startSourceSpan *parse_util.ParseSourceSpan,
	wholeSourceSpan *parse_util.ParseSourceSpan,
) *ElementStartOp {
	return &ElementStartOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:            xref,
				SlotHandle:      NewSlotHandle(),
				NumSlotsUsed:    1,
				LocalRefsField:  []LocalRef{},
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       &tag,
			Namespace: namespace,
		},
		I18nPlaceholder: i18nPlaceholder,
	}
}

type ForeignComponentOp struct {
	OpBase
	ConsumesSlotOpTrait
	Xref                XrefId
	ForeignComponentRef output.Expression
	Props               output.Expression
	SourceSpan          *parse_util.ParseSourceSpan
}

func (o *ForeignComponentOp) Kind() OpKind {
	return OpKindForeignComponent
}

func CreateForeignComponentOp(
	xref XrefId,
	foreignComponentRef output.Expression,
	props output.Expression,
	sourceSpan *parse_util.ParseSourceSpan,
) *ForeignComponentOp {
	return &ForeignComponentOp{
		Xref:                xref,
		ForeignComponentRef: foreignComponentRef,
		Props:               props,
		SourceSpan:          sourceSpan,
	}
}

type ElementOp struct {
	ElementOpBase
	I18nPlaceholder any
}

func (o *ElementOp) Kind() OpKind {
	return OpKindElement
}

type TemplateOp struct {
	ElementOpBase
	TemplateKind       any
	Decls              *int
	Vars               *int
	FunctionNameSuffix *string
	I18nPlaceholder    any
}

func (o *TemplateOp) Kind() OpKind {
	return OpKindTemplate
}

func CreateTemplateOp(
	xref XrefId,
	templateKind any,
	tag *string,
	functionNameSuffix string,
	namespace any,
	i18nPlaceholder any,
	startSourceSpan *parse_util.ParseSourceSpan,
	wholeSourceSpan *parse_util.ParseSourceSpan,
) *TemplateOp {
	return &TemplateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:            xref,
				SlotHandle:      NewSlotHandle(),
				NumSlotsUsed:    1,
				LocalRefsField:  []LocalRef{},
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tag,
			Namespace: namespace,
		},
		TemplateKind:       templateKind,
		FunctionNameSuffix: &functionNameSuffix,
		I18nPlaceholder:    i18nPlaceholder,
	}
}

type ConditionalCreateOp struct {
	ElementOpBase
	TemplateKind       any
	Decls              *int
	Vars               *int
	FunctionNameSuffix *string
	I18nPlaceholder    any
}

func (o *ConditionalCreateOp) Kind() OpKind {
	return OpKindConditionalCreate
}

func CreateConditionalCreateOp(
	xref XrefId,
	templateKind any,
	tag *string,
	functionNameSuffix string,
	namespace any,
	i18nPlaceholder any,
	startSourceSpan *parse_util.ParseSourceSpan,
	wholeSourceSpan *parse_util.ParseSourceSpan,
) *ConditionalCreateOp {
	return &ConditionalCreateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:            xref,
				SlotHandle:      NewSlotHandle(),
				NumSlotsUsed:    1,
				LocalRefsField:  []LocalRef{},
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tag,
			Namespace: namespace,
		},
		TemplateKind:       templateKind,
		FunctionNameSuffix: &functionNameSuffix,
		I18nPlaceholder:    i18nPlaceholder,
	}
}

type ConditionalBranchCreateOp struct {
	ElementOpBase
	TemplateKind       any
	Decls              *int
	Vars               *int
	FunctionNameSuffix *string
	I18nPlaceholder    any
}

func (o *ConditionalBranchCreateOp) Kind() OpKind {
	return OpKindConditionalBranchCreate
}

func CreateConditionalBranchCreateOp(
	xref XrefId,
	templateKind any,
	tag *string,
	functionNameSuffix string,
	namespace any,
	i18nPlaceholder any,
	startSourceSpan *parse_util.ParseSourceSpan,
	wholeSourceSpan *parse_util.ParseSourceSpan,
) *ConditionalBranchCreateOp {
	return &ConditionalBranchCreateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:            xref,
				SlotHandle:      NewSlotHandle(),
				NumSlotsUsed:    1,
				LocalRefsField:  []LocalRef{},
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tag,
			Namespace: namespace,
		},
		TemplateKind:       templateKind,
		FunctionNameSuffix: &functionNameSuffix,
		I18nPlaceholder:    i18nPlaceholder,
	}
}

type RepeaterCreateOp struct {
	ElementOpBase
	Decls                 *int
	Vars                  *int
	EmptyView             XrefId
	Track                 output.Expression
	TrackByOps            any
	TrackByFn             output.Expression
	VarNames              any
	UsesComponentInstance bool
	FunctionNameSuffix    string
	EmptyTag              *string
	EmptyAttributes       any
	I18nPlaceholder       any
	EmptyI18nPlaceholder  any
}

func (o *RepeaterCreateOp) Kind() OpKind {
	return OpKindRepeaterCreate
}

func (o *RepeaterCreateOp) ConsumesVars() bool {
	return true
}

func CreateRepeaterCreateOp(
	repeaterView XrefId,
	emptyView *XrefId,
	tag *string,
	track output.Expression,
	varNames any,
	emptyTag *string,
	i18nPlaceholder any,
	emptyI18nPlaceholder any,
	startSourceSpan *parse_util.ParseSourceSpan,
	wholeSourceSpan *parse_util.ParseSourceSpan,
) *RepeaterCreateOp {
	var ev XrefId
	numSlotsUsed := 2
	if emptyView != nil {
		ev = *emptyView
		numSlotsUsed = 3
	}
	return &RepeaterCreateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				Xref:            repeaterView,
				SlotHandle:      NewSlotHandle(),
				NumSlotsUsed:    numSlotsUsed,
				LocalRefsField:  []LocalRef{},
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag: tag,
		},
		EmptyView:            ev,
		Track:                track,
		VarNames:             varNames,
		FunctionNameSuffix:   "For",
		EmptyTag:             emptyTag,
		I18nPlaceholder:      i18nPlaceholder,
		EmptyI18nPlaceholder: emptyI18nPlaceholder,
	}
}

type RepeaterVarNames struct {
}

type ElementEndOp struct {
	OpBase
	Xref       XrefId
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *ElementEndOp) Kind() OpKind {
	return OpKindElementEnd
}

func CreateElementEndOp(
	xref XrefId,
	sourceSpan *parse_util.ParseSourceSpan,
) *ElementEndOp {
	return &ElementEndOp{
		Xref:       xref,
		SourceSpan: sourceSpan,
	}
}

type ContainerStartOp struct {
	ElementOrContainerOpBase
	kind OpKind
}

func (o *ContainerStartOp) Kind() OpKind {
	if o.kind != 0 {
		return o.kind
	}
	return OpKindContainerStart
}

func (o *ContainerStartOp) SetKind(kind OpKind) {
	o.kind = kind
}

type ContainerOp struct {
	ElementOrContainerOpBase
}

func (o *ContainerOp) Kind() OpKind {
	return OpKindContainer
}

type ContainerEndOp struct {
	OpBase
	Xref       XrefId
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *ContainerEndOp) Kind() OpKind {
	return OpKindContainerEnd
}

type DisableBindingsOp struct {
	OpBase
	Xref XrefId
}

func (o *DisableBindingsOp) Kind() OpKind {
	return OpKindDisableBindings
}

type EnableBindingsOp struct {
	OpBase
	Xref XrefId
}

func (o *EnableBindingsOp) Kind() OpKind {
	return OpKindEnableBindings
}

type TextOp struct {
	OpBase
	ConsumesSlotOpTrait
	Xref           XrefId
	SlotHandle     *SlotHandle
	NumSlotsUsed   int
	InitialValue   string
	IcuPlaceholder *string
	SourceSpan     *parse_util.ParseSourceSpan
}

func (o *TextOp) Kind() OpKind {
	return OpKindText
}

func (o *TextOp) Handle() *SlotHandle {
	if o.SlotHandle == nil {
		o.SlotHandle = NewSlotHandle()
	}
	return o.SlotHandle
}

func (o *TextOp) AddNumSlotsUsed(n int) {
	o.NumSlotsUsed += n
}

func (o *TextOp) GetNumSlotsUsed() int {
	return o.NumSlotsUsed
}

func CreateTextOp(
	xref XrefId,
	initialValue string,
	icuPlaceholder *string,
	sourceSpan *parse_util.ParseSourceSpan,
) *TextOp {
	return &TextOp{
		Xref:           xref,
		SlotHandle:     NewSlotHandle(),
		NumSlotsUsed:   1,
		InitialValue:   initialValue,
		IcuPlaceholder: icuPlaceholder,
		SourceSpan:     sourceSpan,
	}
}

type AnimationStringOp struct {
	OpBase
	Target          XrefId
	Name            string
	AnimationKind   any
	Expression      output.Expression
	I18nMessage     XrefId
	SecurityContext any
	Sanitizer       output.Expression
	SourceSpan      *parse_util.ParseSourceSpan
}

func (o *AnimationStringOp) Kind() OpKind {
	return OpKindAnimationString
}

type AnimationOp struct {
	OpBase
	Target          XrefId
	Name            string
	AnimationKind   any
	HandlerOps      any
	HandlerFnName   *string
	I18nMessage     XrefId
	SecurityContext any
	Sanitizer       output.Expression
	SourceSpan      *parse_util.ParseSourceSpan
}

func (o *AnimationOp) Kind() OpKind {
	return OpKindAnimation
}

type ListenerOp struct {
	OpBase
	Target                    XrefId
	TargetSlot                any
	HostListener              bool
	Name                      string
	Tag                       *string
	HandlerOps                any
	HandlerFnName             *string
	ConsumesDollarEvent       bool
	IsLegacyAnimationListener bool
	LegacyAnimationPhase      *string
	EventTarget               *string
	SourceSpan                *parse_util.ParseSourceSpan
}

func (o *ListenerOp) Kind() OpKind {
	return OpKindListener
}

func CreateListenerOp(
	target XrefId,
	targetSlot *SlotHandle,
	name string,
	tag *string,
	handlerOps any,
	phase *string,
	eventTarget *string,
	hostListener bool,
	sourceSpan *parse_util.ParseSourceSpan,
) *ListenerOp {
	return &ListenerOp{
		Target:                    target,
		TargetSlot:                targetSlot,
		Name:                      name,
		Tag:                       tag,
		HandlerOps:                handlerOps,
		EventTarget:               eventTarget,
		HostListener:              hostListener,
		SourceSpan:                sourceSpan,
		LegacyAnimationPhase:      phase,
		IsLegacyAnimationListener: phase != nil,
	}
}

type AnimationListenerOp struct {
	OpBase
	Target              XrefId
	TargetSlot          any
	HostListener        bool
	Name                string
	AnimationKind       any
	Tag                 *string
	HandlerOps          any
	HandlerFnName       *string
	ConsumesDollarEvent bool
	EventTarget         *string
	SourceSpan          *parse_util.ParseSourceSpan
}

func (o *AnimationListenerOp) Kind() OpKind {
	return OpKindAnimationListener
}

func CreateAnimationListenerOp(
	target XrefId,
	targetSlot *SlotHandle,
	name string,
	tag *string,
	handlerOps any,
	animationKind any,
	eventTarget *string,
	hostListener bool,
	sourceSpan *parse_util.ParseSourceSpan,
) *AnimationListenerOp {
	return &AnimationListenerOp{
		Target:        target,
		TargetSlot:    targetSlot,
		Name:          name,
		Tag:           tag,
		HandlerOps:    handlerOps,
		AnimationKind: animationKind,
		EventTarget:   eventTarget,
		HostListener:  hostListener,
		SourceSpan:    sourceSpan,
	}
}

type TwoWayListenerOp struct {
	OpBase
	Target        XrefId
	TargetSlot    any
	Name          string
	Tag           *string
	HandlerOps    *OpList
	HandlerFnName *string
	SourceSpan    *parse_util.ParseSourceSpan
}

func (o *TwoWayListenerOp) Kind() OpKind {
	return OpKindTwoWayListener
}

func (o *TwoWayListenerOp) GetHandlerOps() *OpList {
	return o.HandlerOps
}

func (o *TwoWayListenerOp) GetHandlerFnName() *string {
	return o.HandlerFnName
}

func (o *TwoWayListenerOp) SetHandlerFnName(name string) {
	o.HandlerFnName = &name
}

func (o *TwoWayListenerOp) GetTag() *string {
	return o.Tag
}

func (o *TwoWayListenerOp) GetName() string {
	return o.Name
}

func (o *TwoWayListenerOp) GetTargetSlot() SlotHandle {
	if slot, ok := o.TargetSlot.(*SlotHandle); ok {
		return *slot
	}
	if slot, ok := o.TargetSlot.(SlotHandle); ok {
		return slot
	}
	return SlotHandle{}
}

func CreateTwoWayListenerOp(
	target XrefId,
	targetSlot *SlotHandle,
	name string,
	tag *string,
	handlerOps *OpList,
	sourceSpan *parse_util.ParseSourceSpan,
) *TwoWayListenerOp {
	return &TwoWayListenerOp{
		Target:     target,
		TargetSlot: targetSlot,
		Name:       name,
		Tag:        tag,
		HandlerOps: handlerOps,
		SourceSpan: sourceSpan,
	}
}

type PipeOp struct {
	OpBase
	Xref         XrefId
	Name         string
	SlotHandle   *SlotHandle
	NumSlotsUsed int
}

func (o *PipeOp) Kind() OpKind {
	return OpKindPipe
}

func (o *PipeOp) Handle() *SlotHandle {
	return o.SlotHandle
}

func (o *PipeOp) AddNumSlotsUsed(n int) {
	o.NumSlotsUsed += n
}

func (o *PipeOp) GetNumSlotsUsed() int {
	return o.NumSlotsUsed
}

type NamespaceOp struct {
	OpBase
	Active any
}

func (o *NamespaceOp) Kind() OpKind {
	return OpKindNamespace
}

type ProjectionDefOp struct {
	OpBase
	Def output.Expression
}

func (o *ProjectionDefOp) Kind() OpKind {
	return OpKindProjectionDef
}

type EnableIncrementalHydrationRuntimeOp struct {
	OpBase
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *EnableIncrementalHydrationRuntimeOp) Kind() OpKind {
	return OpKindEnableIncrementalHydrationRuntime
}

type ProjectionOp struct {
	OpBase
	ConsumesSlotOpTrait
	Xref                        XrefId
	SlotHandle                  *SlotHandle
	NumSlotsUsed                int
	ProjectionSlotIndex         *int
	Attributes                  any
	LocalRefs                   *string
	Selector                    *string
	I18nPlaceholder             any
	SourceSpan                  *parse_util.ParseSourceSpan
	FallbackView                XrefId
	FallbackViewI18nPlaceholder any
}

func (o *ProjectionOp) Handle() *SlotHandle {
	if o.SlotHandle == nil {
		o.SlotHandle = NewSlotHandle()
	}
	return o.SlotHandle
}

func (o *ProjectionOp) AddNumSlotsUsed(n int) {
	o.NumSlotsUsed += n
}

func (o *ProjectionOp) GetNumSlotsUsed() int {
	return o.NumSlotsUsed
}

func (o *ProjectionOp) Kind() OpKind {
	return OpKindProjection
}

func CreateProjectionOp(
	xref XrefId,
	selector *string,
	i18nPlaceholder any,
	fallbackView *XrefId,
	sourceSpan *parse_util.ParseSourceSpan,
) *ProjectionOp {
	var fv XrefId
	if fallbackView != nil {
		fv = *fallbackView
	}
	return &ProjectionOp{
		Xref:            xref,
		SlotHandle:      NewSlotHandle(),
		NumSlotsUsed:    1,
		Selector:        selector,
		I18nPlaceholder: i18nPlaceholder,
		FallbackView:    fv,
		SourceSpan:      sourceSpan,
	}
}

type ExtractedAttributeOp struct {
	OpBase
	Target          XrefId
	BindingKind     any
	Namespace       *string
	Name            string
	Expression      output.Expression
	I18nContext     XrefId
	SecurityContext any
	TrustedValueFn  output.Expression
	I18nMessage     any
}

func (o *ExtractedAttributeOp) Kind() OpKind {
	return OpKindExtractedAttribute
}

func CreateExtractedAttributeOp(
	target XrefId,
	bindingKind any,
	namespace *string,
	name string,
	expression output.Expression,
	trustedValueFn output.Expression,
	i18nMessage any,
	securityContext any,
) *ExtractedAttributeOp {
	return &ExtractedAttributeOp{
		Target:          target,
		BindingKind:     bindingKind,
		Namespace:       namespace,
		Name:            name,
		Expression:      expression,
		TrustedValueFn:  trustedValueFn,
		I18nMessage:     i18nMessage,
		SecurityContext: securityContext,
	}
}

type DeferOp struct {
	OpBase
	Xref                   XrefId
	SlotHandle             *SlotHandle
	NumSlotsUsed           int
	MainView               XrefId
	MainSlot               any
	LoadingView            XrefId
	LoadingSlot            any
	PlaceholderView        XrefId
	PlaceholderSlot        any
	ErrorView              XrefId
	ErrorSlot              any
	PlaceholderMinimumTime *int
	LoadingMinimumTime     *int
	LoadingAfterTime       *int
	PlaceholderConfig      output.Expression
	LoadingConfig          output.Expression
	OwnResolverFn          output.Expression
	ResolverFn             output.Expression
	Flags                  any
	SourceSpan             *parse_util.ParseSourceSpan
}

func (o *DeferOp) Kind() OpKind {
	return OpKindDefer
}



func (o *DeferOp) Handle() *SlotHandle {
	if o.SlotHandle == nil {
		o.SlotHandle = NewSlotHandle()
	}
	return o.SlotHandle
}

func (o *DeferOp) AddNumSlotsUsed(n int) {
	o.NumSlotsUsed += n
}

func (o *DeferOp) GetNumSlotsUsed() int {
	return o.NumSlotsUsed
}

func CreateDeferOp(
	xref XrefId,
	mainView XrefId,
	mainSlot *SlotHandle,
	ownResolverFn output.Expression,
	resolverFn output.Expression,
	sourceSpan *parse_util.ParseSourceSpan,
) *DeferOp {
	return &DeferOp{
		Xref:          xref,
		SlotHandle:    NewSlotHandle(),
		NumSlotsUsed:  2,
		MainView:      mainView,
		MainSlot:      mainSlot,
		OwnResolverFn: ownResolverFn,
		ResolverFn:    resolverFn,
		SourceSpan:    sourceSpan,
	}
}

type DeferTriggerBase struct {
	Kind DeferTriggerKind
}

func (t DeferTriggerBase) GetKind() DeferTriggerKind {
	return t.Kind
}

type DeferIdleTrigger struct {
	DeferTriggerBase
	Timeout *float64
}

type DeferImmediateTrigger struct {
	DeferTriggerBase
}

type DeferNeverTrigger struct {
	DeferTriggerBase
}

type DeferTimerTrigger struct {
	DeferTriggerBase
	Delay float64
}

type DeferTriggerWithTargetBase struct {
	DeferTriggerBase
	TargetName          *string
	TargetXref          *XrefId
	TargetSlot          *SlotHandle
	TargetView          *XrefId
	TargetSlotViewSteps *int
}

func (t DeferTriggerWithTargetBase) GetTargetName() *string {
	return t.TargetName
}

func (t *DeferTriggerWithTargetBase) SetTarget(xref XrefId, view XrefId, slot *SlotHandle, steps int) {
	t.TargetXref = &xref
	t.TargetView = &view
	t.TargetSlot = slot
	t.TargetSlotViewSteps = &steps
}

type DeferHoverTrigger struct {
	DeferTriggerWithTargetBase
}

type DeferInteractionTrigger struct {
	DeferTriggerWithTargetBase
}

type DeferViewportTrigger struct {
	DeferTriggerWithTargetBase
	Options output.Expression
}

type DeferOnOp struct {
	OpBase
	Defer      XrefId
	Trigger    any
	Modifier   any
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *DeferOnOp) Kind() OpKind {
	return OpKindDeferOn
}

func CreateDeferOnOp(
	deferId XrefId,
	trigger any,
	modifier any,
	sourceSpan *parse_util.ParseSourceSpan,
) *DeferOnOp {
	return &DeferOnOp{
		Defer:      deferId,
		Trigger:    trigger,
		Modifier:   modifier,
		SourceSpan: sourceSpan,
	}
}

type DeclareLetOp struct {
	OpBase
	ConsumesSlotOpTrait
	Xref         XrefId
	SlotHandle   *SlotHandle
	NumSlotsUsed int
	SourceSpan   *parse_util.ParseSourceSpan
	DeclaredName string
}

func (o *DeclareLetOp) Handle() *SlotHandle {
	if o.SlotHandle == nil {
		o.SlotHandle = NewSlotHandle()
	}
	return o.SlotHandle
}

func (o *DeclareLetOp) AddNumSlotsUsed(n int) {
	o.NumSlotsUsed += n
}

func (o *DeclareLetOp) GetNumSlotsUsed() int {
	return o.NumSlotsUsed
}

func (o *DeclareLetOp) Kind() OpKind {
	return OpKindDeclareLet
}

func CreateDeclareLetOp(
	xref XrefId,
	declaredName string,
	sourceSpan *parse_util.ParseSourceSpan,
) *DeclareLetOp {
	return &DeclareLetOp{
		Xref:         xref,
		SlotHandle:   NewSlotHandle(),
		NumSlotsUsed: 1,
		DeclaredName: declaredName,
		SourceSpan:   sourceSpan,
	}
}

type I18nParamValue struct {
	Value string
}

type I18nMessageOp struct {
	OpBase
	Xref                 XrefId
	I18nContext          XrefId
	I18nBlock            XrefId
	Message              any
	MessagePlaceholder   *string
	NeedsPostprocessing  bool
	Params               string
	PostprocessingParams string
	SubMessages          XrefId
}

func (o *I18nMessageOp) Kind() OpKind {
	return OpKindI18nMessage
}

type I18nOpBase struct {
	OpBase
	ConsumesSlotOpTrait
	Xref             XrefId
	Root             XrefId
	Message          any
	MessageIndex     any
	SubTemplateIndex *int
	Context          XrefId
	SourceSpan       *parse_util.ParseSourceSpan
}

func (o *I18nOpBase) Kind() OpKind {
	return OpKindI18n
}

type I18nOp struct {
	I18nOpBase
}

func (o *I18nOp) Kind() OpKind {
	return OpKindI18n
}

type I18nStartOp struct {
	I18nOpBase
	kind OpKind
}

func (o *I18nStartOp) Kind() OpKind {
	if o.kind != 0 {
		return o.kind
	}
	return OpKindI18nStart
}

func (o *I18nStartOp) SetKind(kind OpKind) {
	o.kind = kind
}

func CreateI18nStartOp(
	xref XrefId,
	message any,
	messageIndex any,
	sourceSpan *parse_util.ParseSourceSpan,
) *I18nStartOp {
	return &I18nStartOp{
		I18nOpBase: I18nOpBase{
			Xref:         xref,
			Message:      message,
			MessageIndex: messageIndex,
			SourceSpan:   sourceSpan,
		},
	}
}

type I18nEndOp struct {
	OpBase
	Xref       XrefId
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *I18nEndOp) Kind() OpKind {
	return OpKindI18nEnd
}

func CreateI18nEndOp(
	xref XrefId,
	sourceSpan *parse_util.ParseSourceSpan,
) *I18nEndOp {
	return &I18nEndOp{
		Xref:       xref,
		SourceSpan: sourceSpan,
	}
}

type IcuStartOp struct {
	OpBase
	Xref               XrefId
	Message            any
	MessagePlaceholder string
	Context            XrefId
	SourceSpan         *parse_util.ParseSourceSpan
}

func (o *IcuStartOp) Kind() OpKind {
	return OpKindIcuStart
}

func CreateIcuStartOp(
	xref XrefId,
	message any,
	messagePlaceholder string,
	sourceSpan *parse_util.ParseSourceSpan,
) *IcuStartOp {
	return &IcuStartOp{
		Xref:               xref,
		Message:            message,
		MessagePlaceholder: messagePlaceholder,
		SourceSpan:         sourceSpan,
	}
}

type IcuEndOp struct {
	OpBase
	Xref XrefId
}

func (o *IcuEndOp) Kind() OpKind {
	return OpKindIcuEnd
}

func CreateIcuEndOp(
	xref XrefId,
) *IcuEndOp {
	return &IcuEndOp{
		Xref: xref,
	}
}

type IcuPlaceholderOp struct {
	OpBase
	Xref                   XrefId
	Name                   string
	Strings                string
	ExpressionPlaceholders any
}

func (o *IcuPlaceholderOp) Kind() OpKind {
	return OpKindIcuPlaceholder
}

type I18nContextOp struct {
	OpBase
	ContextKind          any
	Xref                 XrefId
	I18nBlock            XrefId
	Message              any
	Params               string
	PostprocessingParams string
	SourceSpan           *parse_util.ParseSourceSpan
}

func (o *I18nContextOp) Kind() OpKind {
	return OpKindI18nContext
}

type I18nAttributesOp struct {
	OpBase
	ConsumesSlotOpTrait
	Target               XrefId
	I18nAttributesConfig any
}

func (o *I18nAttributesOp) Kind() OpKind {
	return OpKindI18nAttributes
}

func CreateI18nAttributesOp(
	xref XrefId,
	slotHandle *SlotHandle,
	target XrefId,
) *I18nAttributesOp {
	return &I18nAttributesOp{
		Target: target,
	}
}

type ElementSourceLocation struct {
	TargetSlot any
	Offset     int
	Line       int
	Column     int
}

type SourceLocationOp struct {
	OpBase
	TemplatePath string
	Locations    any
}

func (o *SourceLocationOp) Kind() OpKind {
	return OpKindSourceLocation
}

type ControlCreateOp struct {
	OpBase
	SourceSpan *parse_util.ParseSourceSpan
}

func (o *ControlCreateOp) Kind() OpKind {
	return OpKindControlCreate
}
func (o *ListenerOp) GetKind() OpKind { return o.Kind() }

func (o *ListenerOp) GetHandlerOps() *OpList {
	if list, ok := o.HandlerOps.(*OpList); ok {
		return list
	}
	return nil
}

func (o *ListenerOp) GetHandlerFnName() *string          { return o.HandlerFnName }
func (o *ListenerOp) SetHandlerFnName(st string)         { o.HandlerFnName = &st }
func (o *ListenerOp) SetConsumesDollarEvent(v bool)      { o.ConsumesDollarEvent = v }
func (o *ListenerOp) GetHostListener() bool              { return o.HostListener }
func (o *ListenerOp) GetName() string                    { return o.Name }
func (o *ListenerOp) SetName(st string)                  { o.Name = st }
func (o *ListenerOp) GetIsLegacyAnimationListener() bool { return o.IsLegacyAnimationListener }
func (o *ListenerOp) GetLegacyAnimationPhase() string {
	if o.LegacyAnimationPhase != nil {
		return *o.LegacyAnimationPhase
	}
	return ""
}
func (o *ListenerOp) GetTag() *string { return o.Tag }
func (o *ListenerOp) GetTargetSlot() SlotHandle {
	if slot, ok := o.TargetSlot.(SlotHandle); ok {
		return slot
	}
	return SlotHandle{}
}

func (o *ElementOrContainerOpBase) GetXref() XrefId {
	return o.Xref
}

func (o *ForeignComponentOp) GetXref() XrefId {
	return o.Xref
}

func (o *TextOp) GetXref() XrefId {
	return o.Xref
}

func (o *PipeOp) GetXref() XrefId {
	return o.Xref
}

func (o *ProjectionOp) GetXref() XrefId {
	return o.Xref
}

func (o *DeferOp) GetXref() XrefId {
	return o.Xref
}

func (o *DeclareLetOp) GetXref() XrefId {
	return o.Xref
}

func (o *I18nOpBase) GetXref() XrefId {
	return o.Xref
}

func (o *I18nAttributesOp) GetXref() XrefId {
	return o.Target
}

func (o *TemplateOp) SetDecls(decls int) {
	o.Decls = &decls
}

func (o *TemplateOp) SetVars(vars int) {
	o.Vars = &vars
}

func (o *RepeaterCreateOp) SetDecls(decls int) {
	o.Decls = &decls
}

func (o *RepeaterCreateOp) SetVars(vars int) {
	o.Vars = &vars
}

func (o *ConditionalCreateOp) SetDecls(decls int) {
	o.Decls = &decls
}

func (o *ConditionalCreateOp) SetVars(vars int) {
	o.Vars = &vars
}

func (o *ConditionalBranchCreateOp) SetDecls(decls int) {
	o.Decls = &decls
}

func (o *ConditionalBranchCreateOp) SetVars(vars int) {
	o.Vars = &vars
}
