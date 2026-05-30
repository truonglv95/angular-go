package ir

type OpKind int

const (
	OpKindListEnd OpKind = iota
	OpKindStatement
	OpKindVariable
	OpKindElementStart
	OpKindElement
	OpKindForeignComponent
	OpKindTemplate
	OpKindElementEnd
	OpKindContainerStart
	OpKindContainer
	OpKindContainerEnd
	OpKindDisableBindings
	OpKindConditionalCreate
	OpKindConditionalBranchCreate
	OpKindConditional
	OpKindEnableBindings
	OpKindText
	OpKindListener
	OpKindInterpolateText
	OpKindBinding
	OpKindProperty
	OpKindStyleProp
	OpKindClassProp
	OpKindStyleMap
	OpKindClassMap
	OpKindAdvance
	OpKindPipe
	OpKindAttribute
	OpKindExtractedAttribute
	OpKindDefer
	OpKindDeferOn
	OpKindDeferWhen
	OpKindI18nMessage
	OpKindDomProperty
	OpKindNamespace
	OpKindProjectionDef
	OpKindEnableIncrementalHydrationRuntime
	OpKindProjection
	OpKindRepeaterCreate
	OpKindRepeater
	OpKindTwoWayProperty
	OpKindTwoWayListener
	OpKindDeclareLet
	OpKindStoreLet
	OpKindI18nStart
	OpKindI18n
	OpKindI18nEnd
	OpKindI18nExpression
	OpKindI18nApply
	OpKindIcuStart
	OpKindIcuEnd
	OpKindIcuPlaceholder
	OpKindI18nContext
	OpKindI18nAttributes
	OpKindSourceLocation
	OpKindAnimation
	OpKindAnimationString
	OpKindAnimationBinding
	OpKindAnimationListener
	OpKindControl
	OpKindControlCreate
)

type ExpressionKind int

const (
	ExpressionKindLexicalRead ExpressionKind = iota
	ExpressionKindContext
	ExpressionKindTrackContext
	ExpressionKindReadVariable
	ExpressionKindNextContext
	ExpressionKindReference
	ExpressionKindStoreLet
	ExpressionKindContextLetReference
	ExpressionKindGetCurrentView
	ExpressionKindRestoreView
	ExpressionKindResetView
	ExpressionKindPureFunctionExpr
	ExpressionKindPureFunctionParameterExpr
	ExpressionKindPipeBinding
	ExpressionKindPipeBindingVariadic
	ExpressionKindSafePropertyRead
	ExpressionKindSafeKeyedRead
	ExpressionKindSafeNavigationMigration
	ExpressionKindSafeTernaryExpr
	ExpressionKindEmptyExpr
	ExpressionKindAssignTemporaryExpr
	ExpressionKindReadTemporaryExpr
	ExpressionKindSlotLiteralExpr
	ExpressionKindConditionalCase
	ExpressionKindConstCollected
	ExpressionKindTwoWayBindingSet
	ExpressionKindArrowFunction
)

type VariableFlags int

const (
	VariableFlagsNone         VariableFlags = 0b0000
	VariableFlagsAlwaysInline VariableFlags = 0b0001
)

type SemanticVariableKind int

const (
	SemanticVariableKindContext SemanticVariableKind = iota
	SemanticVariableKindIdentifier
	SemanticVariableKindSavedView
	SemanticVariableKindAlias
)

type BindingKind int

const (
	BindingKindAttribute BindingKind = iota
	BindingKindClassName
	BindingKindStyleProperty
	BindingKindProperty
	BindingKindTemplate
	BindingKindI18n
	BindingKindLegacyAnimation
	BindingKindTwoWayProperty
	BindingKindAnimation
)

type I18nParamResolutionTime int

const (
	I18nParamResolutionTimeCreation I18nParamResolutionTime = iota
	I18nParamResolutionTimePostproccessing
)

type I18nExpressionFor int

const (
	I18nExpressionForI18nText I18nExpressionFor = iota
	I18nExpressionForI18nAttribute
)

type I18nParamValueFlags int

const (
	I18nParamValueFlagsNone            I18nParamValueFlags = 0b0000
	I18nParamValueFlagsElementTag      I18nParamValueFlags = 0b0001
	I18nParamValueFlagsTemplateTag     I18nParamValueFlags = 0b0010
	I18nParamValueFlagsOpenTag         I18nParamValueFlags = 0b0100
	I18nParamValueFlagsCloseTag        I18nParamValueFlags = 0b1000
	I18nParamValueFlagsExpressionIndex I18nParamValueFlags = 0b10000
)

type Namespace int

const (
	NamespaceHTML Namespace = iota
	NamespaceSVG
	NamespaceMath
)

type DeferTriggerKind int

const (
	DeferTriggerKindIdle DeferTriggerKind = iota
	DeferTriggerKindImmediate
	DeferTriggerKindTimer
	DeferTriggerKindHover
	DeferTriggerKindInteraction
	DeferTriggerKindViewport
	DeferTriggerKindNever
)

type I18nContextKind int

const (
	I18nContextKindRootI18n I18nContextKind = iota
	I18nContextKindIcu
	I18nContextKindAttr
)

type TemplateKind int

const (
	TemplateKindNgTemplate TemplateKind = iota
	TemplateKindStructural
	TemplateKindBlock
)

type AnimationKind string

const (
	AnimationKindENTER AnimationKind = "enter"
	AnimationKindLEAVE AnimationKind = "leave"
)

type AnimationBindingKind int

const (
	AnimationBindingKindSTRING AnimationBindingKind = iota
	AnimationBindingKindVALUE
)

type DeferOpModifierKind string

const (
	DeferOpModifierKindNONE     DeferOpModifierKind = "none"
	DeferOpModifierKindPREFETCH DeferOpModifierKind = "prefetch"
	DeferOpModifierKindHYDRATE  DeferOpModifierKind = "hydrate"
)

type TDeferDetailsFlags int

const (
	TDeferDetailsFlagsDefault            TDeferDetailsFlags = 0
	TDeferDetailsFlagsHasHydrateTriggers TDeferDetailsFlags = 1 << 0
)
