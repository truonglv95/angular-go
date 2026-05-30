package render3

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

const CORE = "@angular/core"

// Identifiers for Ivy runtime instructions.
// Maps to angular/packages/compiler/src/render3/r3_identifiers.ts
func s(str string) *string { return &str }

var coreStr = s(CORE)

var Identifiers = struct {
	Core                              *output.ExternalReference
	NamespaceHTML                     *output.ExternalReference
	NamespaceMathML                   *output.ExternalReference
	NamespaceSVG                      *output.ExternalReference
	Element                           *output.ExternalReference
	ElementStart                      *output.ExternalReference
	ElementEnd                        *output.ExternalReference
	ForeignComponent                  *output.ExternalReference
	DomElement                        *output.ExternalReference
	DomElementStart                   *output.ExternalReference
	DomElementEnd                     *output.ExternalReference
	DomElementContainer               *output.ExternalReference
	DomElementContainerStart          *output.ExternalReference
	DomElementContainerEnd            *output.ExternalReference
	DomTemplate                       *output.ExternalReference
	DomListener                       *output.ExternalReference
	Advance                           *output.ExternalReference
	SyntheticHostProperty             *output.ExternalReference
	SyntheticHostListener             *output.ExternalReference
	Attribute                         *output.ExternalReference
	ClassProp                         *output.ExternalReference
	ElementContainerStart             *output.ExternalReference
	ElementContainerEnd               *output.ExternalReference
	ElementContainer                  *output.ExternalReference
	StyleMap                          *output.ExternalReference
	ClassMap                          *output.ExternalReference
	StyleProp                         *output.ExternalReference
	Interpolate                       *output.ExternalReference
	Interpolate1                      *output.ExternalReference
	Interpolate2                      *output.ExternalReference
	Interpolate3                      *output.ExternalReference
	Interpolate4                      *output.ExternalReference
	Interpolate5                      *output.ExternalReference
	Interpolate6                      *output.ExternalReference
	Interpolate7                      *output.ExternalReference
	Interpolate8                      *output.ExternalReference
	InterpolateV                      *output.ExternalReference
	NextContext                       *output.ExternalReference
	ResetView                         *output.ExternalReference
	TemplateCreate                    *output.ExternalReference
	Defer                             *output.ExternalReference
	DeferWhen                         *output.ExternalReference
	DeferOnIdle                       *output.ExternalReference
	DeferOnImmediate                  *output.ExternalReference
	DeferOnTimer                      *output.ExternalReference
	DeferOnHover                      *output.ExternalReference
	DeferOnInteraction                *output.ExternalReference
	DeferOnViewport                   *output.ExternalReference
	DeferPrefetchWhen                 *output.ExternalReference
	DeferPrefetchOnIdle               *output.ExternalReference
	DeferPrefetchOnImmediate          *output.ExternalReference
	DeferPrefetchOnTimer              *output.ExternalReference
	DeferPrefetchOnHover              *output.ExternalReference
	DeferPrefetchOnInteraction        *output.ExternalReference
	DeferPrefetchOnViewport           *output.ExternalReference
	DeferHydrateWhen                  *output.ExternalReference
	DeferHydrateNever                 *output.ExternalReference
	DeferHydrateOnIdle                *output.ExternalReference
	DeferHydrateOnImmediate           *output.ExternalReference
	DeferHydrateOnTimer               *output.ExternalReference
	DeferHydrateOnHover               *output.ExternalReference
	DeferHydrateOnInteraction         *output.ExternalReference
	DeferHydrateOnViewport            *output.ExternalReference
	DeferEnableTimerScheduling        *output.ExternalReference
	EnableIncrementalHydrationRuntime *output.ExternalReference
	ConditionalCreate                 *output.ExternalReference
	ConditionalBranchCreate           *output.ExternalReference
	Conditional                       *output.ExternalReference
	Repeater                          *output.ExternalReference
	RepeaterCreate                    *output.ExternalReference
	RepeaterTrackByIndex              *output.ExternalReference
	RepeaterTrackByIdentity           *output.ExternalReference
	ComponentInstance                 *output.ExternalReference
	Text                              *output.ExternalReference
	EnableBindings                    *output.ExternalReference
	DisableBindings                   *output.ExternalReference
	GetCurrentView                    *output.ExternalReference
	TextInterpolate                   *output.ExternalReference
	TextInterpolate1                  *output.ExternalReference
	TextInterpolate2                  *output.ExternalReference
	TextInterpolate3                  *output.ExternalReference
	TextInterpolate4                  *output.ExternalReference
	TextInterpolate5                  *output.ExternalReference
	TextInterpolate6                  *output.ExternalReference
	TextInterpolate7                  *output.ExternalReference
	TextInterpolate8                  *output.ExternalReference
	TextInterpolateV                  *output.ExternalReference
	RestoreView                       *output.ExternalReference
	PureFunction0                     *output.ExternalReference
	PureFunction1                     *output.ExternalReference
	PureFunction2                     *output.ExternalReference
	PureFunction3                     *output.ExternalReference
	PureFunction4                     *output.ExternalReference
	PureFunction5                     *output.ExternalReference
	PureFunction6                     *output.ExternalReference
	PureFunction7                     *output.ExternalReference
	PureFunction8                     *output.ExternalReference
	PureFunctionV                     *output.ExternalReference
	PipeBind1                         *output.ExternalReference
	PipeBind2                         *output.ExternalReference
	PipeBind3                         *output.ExternalReference
	PipeBind4                         *output.ExternalReference
	PipeBindV                         *output.ExternalReference
	DomProperty                       *output.ExternalReference
	AriaProperty                      *output.ExternalReference
	Property                          *output.ExternalReference
	Control                           *output.ExternalReference
	ControlCreate                     *output.ExternalReference
	AnimationEnterListener            *output.ExternalReference
	AnimationLeaveListener            *output.ExternalReference
	AnimationEnter                    *output.ExternalReference
	AnimationLeave                    *output.ExternalReference
	I18n                              *output.ExternalReference
	I18nAttributes                    *output.ExternalReference
	I18nExp                           *output.ExternalReference
	I18nStart                         *output.ExternalReference
	I18nEnd                           *output.ExternalReference
	I18nApply                         *output.ExternalReference
	I18nPostprocess                   *output.ExternalReference
	Pipe                              *output.ExternalReference
	Projection                        *output.ExternalReference
	ProjectionDef                     *output.ExternalReference
	Reference                         *output.ExternalReference
	Inject                            *output.ExternalReference
	InjectAttribute                   *output.ExternalReference
	DirectiveInject                   *output.ExternalReference
	InvalidFactory                    *output.ExternalReference
	InvalidFactoryDep                 *output.ExternalReference
	TemplateRefExtractor              *output.ExternalReference
	ForwardRef                        *output.ExternalReference
	ResolveForwardRef                 *output.ExternalReference
	ReplaceMetadata                   *output.ExternalReference
	GetReplaceMetadataURL             *output.ExternalReference
	ƟɵdefineInjectable                *output.ExternalReference
	DeclareInjectable                 *output.ExternalReference
	InjectableDeclaration             *output.ExternalReference
	DefineService                     *output.ExternalReference
	DeclareService                    *output.ExternalReference
	ResolveWindow                     *output.ExternalReference
	ResolveDocument                   *output.ExternalReference
	ResolveBody                       *output.ExternalReference
	GetComponentDepsFactory           *output.ExternalReference
	DefineComponent                   *output.ExternalReference
	DeclareComponent                  *output.ExternalReference
	SetComponentScope                 *output.ExternalReference
	ChangeDetectionStrategy           *output.ExternalReference
	ViewEncapsulation                 *output.ExternalReference
	ComponentDeclaration              *output.ExternalReference
	FactoryDeclaration                *output.ExternalReference
	DeclareFactory                    *output.ExternalReference
	FactoryTarget                     *output.ExternalReference
	DefineDirective                   *output.ExternalReference
	DeclareDirective                  *output.ExternalReference
	DirectiveDeclaration              *output.ExternalReference
	InjectorDef                       *output.ExternalReference
	InjectorDeclaration               *output.ExternalReference
	DefineInjector                    *output.ExternalReference
	DeclareInjector                   *output.ExternalReference
	NgModuleDeclaration               *output.ExternalReference
	ModuleWithProviders               *output.ExternalReference
	DefineNgModule                    *output.ExternalReference
	DeclareNgModule                   *output.ExternalReference
	SetNgModuleScope                  *output.ExternalReference
	RegisterNgModuleType              *output.ExternalReference
	PipeDeclaration                   *output.ExternalReference
	DefinePipe                        *output.ExternalReference
	DeclarePipe                       *output.ExternalReference
	DeclareClassMetadata              *output.ExternalReference
	DeclareClassMetadataAsync         *output.ExternalReference
	SetClassMetadata                  *output.ExternalReference
	SetClassMetadataAsync             *output.ExternalReference
	SetClassDebugInfo                 *output.ExternalReference
	QueryRefresh                      *output.ExternalReference
	ViewQuery                         *output.ExternalReference
	LoadQuery                         *output.ExternalReference
	ContentQuery                      *output.ExternalReference
	ViewQuerySignal                   *output.ExternalReference
	ContentQuerySignal                *output.ExternalReference
	QueryAdvance                      *output.ExternalReference
	TwoWayProperty                    *output.ExternalReference
	TwoWayBindingSet                  *output.ExternalReference
	TwoWayListener                    *output.ExternalReference
	DeclareLet                        *output.ExternalReference
	StoreLet                          *output.ExternalReference
	ReadContextLet                    *output.ExternalReference
	ArrowFunction                     *output.ExternalReference
	AttachSourceLocations             *output.ExternalReference
	NgOnChangesFeature                *output.ExternalReference
	ControlFeature                    *output.ExternalReference
	InheritDefinitionFeature          *output.ExternalReference
	ProvidersFeature                  *output.ExternalReference
	HostDirectivesFeature             *output.ExternalReference
	ExternalStylesFeature             *output.ExternalReference
	Listener                          *output.ExternalReference
	GetInheritedFactory               *output.ExternalReference
	SanitizeHtml                      *output.ExternalReference
	SanitizeStyle                     *output.ExternalReference
	ValidateAttribute                 *output.ExternalReference
	SanitizeResourceUrl               *output.ExternalReference
	SanitizeScript                    *output.ExternalReference
	SanitizeUrl                       *output.ExternalReference
	SanitizeUrlOrResourceUrl          *output.ExternalReference
	TrustConstantHtml                 *output.ExternalReference
	TrustConstantResourceUrl          *output.ExternalReference
	InputDecorator                    *output.ExternalReference
	OutputDecorator                   *output.ExternalReference
	ViewChildDecorator                *output.ExternalReference
	ViewChildrenDecorator             *output.ExternalReference
	ContentChildDecorator             *output.ExternalReference
	ContentChildrenDecorator          *output.ExternalReference
	InputSignalBrandWriteType         *output.ExternalReference
	UnwrapDirectiveSignalInputs       *output.ExternalReference
	UnwrapWritableSignal              *output.ExternalReference
	AssertType                        *output.ExternalReference
}{
	Core:                              &output.ExternalReference{Name: nil, ModuleName: coreStr},
	NamespaceHTML:                     &output.ExternalReference{Name: s("ɵɵnamespaceHTML"), ModuleName: coreStr},
	NamespaceMathML:                   &output.ExternalReference{Name: s("ɵɵnamespaceMathML"), ModuleName: coreStr},
	NamespaceSVG:                      &output.ExternalReference{Name: s("ɵɵnamespaceSVG"), ModuleName: coreStr},
	Element:                           &output.ExternalReference{Name: s("ɵɵelement"), ModuleName: coreStr},
	ElementStart:                      &output.ExternalReference{Name: s("ɵɵelementStart"), ModuleName: coreStr},
	ElementEnd:                        &output.ExternalReference{Name: s("ɵɵelementEnd"), ModuleName: coreStr},
	ForeignComponent:                  &output.ExternalReference{Name: s("ɵɵforeignComponent"), ModuleName: coreStr},
	DomElement:                        &output.ExternalReference{Name: s("ɵɵdomElement"), ModuleName: coreStr},
	DomElementStart:                   &output.ExternalReference{Name: s("ɵɵdomElementStart"), ModuleName: coreStr},
	DomElementEnd:                     &output.ExternalReference{Name: s("ɵɵdomElementEnd"), ModuleName: coreStr},
	DomElementContainer:               &output.ExternalReference{Name: s("ɵɵdomElementContainer"), ModuleName: coreStr},
	DomElementContainerStart:          &output.ExternalReference{Name: s("ɵɵdomElementContainerStart"), ModuleName: coreStr},
	DomElementContainerEnd:            &output.ExternalReference{Name: s("ɵɵdomElementContainerEnd"), ModuleName: coreStr},
	DomTemplate:                       &output.ExternalReference{Name: s("ɵɵdomTemplate"), ModuleName: coreStr},
	DomListener:                       &output.ExternalReference{Name: s("ɵɵdomListener"), ModuleName: coreStr},
	Advance:                           &output.ExternalReference{Name: s("ɵɵadvance"), ModuleName: coreStr},
	SyntheticHostProperty:             &output.ExternalReference{Name: s("ɵɵsyntheticHostProperty"), ModuleName: coreStr},
	SyntheticHostListener:             &output.ExternalReference{Name: s("ɵɵsyntheticHostListener"), ModuleName: coreStr},
	Attribute:                         &output.ExternalReference{Name: s("ɵɵattribute"), ModuleName: coreStr},
	ClassProp:                         &output.ExternalReference{Name: s("ɵɵclassProp"), ModuleName: coreStr},
	ElementContainerStart:             &output.ExternalReference{Name: s("ɵɵelementContainerStart"), ModuleName: coreStr},
	ElementContainerEnd:               &output.ExternalReference{Name: s("ɵɵelementContainerEnd"), ModuleName: coreStr},
	ElementContainer:                  &output.ExternalReference{Name: s("ɵɵelementContainer"), ModuleName: coreStr},
	StyleMap:                          &output.ExternalReference{Name: s("ɵɵstyleMap"), ModuleName: coreStr},
	ClassMap:                          &output.ExternalReference{Name: s("ɵɵclassMap"), ModuleName: coreStr},
	StyleProp:                         &output.ExternalReference{Name: s("ɵɵstyleProp"), ModuleName: coreStr},
	Interpolate:                       &output.ExternalReference{Name: s("ɵɵinterpolate"), ModuleName: coreStr},
	Interpolate1:                      &output.ExternalReference{Name: s("ɵɵinterpolate1"), ModuleName: coreStr},
	Interpolate2:                      &output.ExternalReference{Name: s("ɵɵinterpolate2"), ModuleName: coreStr},
	Interpolate3:                      &output.ExternalReference{Name: s("ɵɵinterpolate3"), ModuleName: coreStr},
	Interpolate4:                      &output.ExternalReference{Name: s("ɵɵinterpolate4"), ModuleName: coreStr},
	Interpolate5:                      &output.ExternalReference{Name: s("ɵɵinterpolate5"), ModuleName: coreStr},
	Interpolate6:                      &output.ExternalReference{Name: s("ɵɵinterpolate6"), ModuleName: coreStr},
	Interpolate7:                      &output.ExternalReference{Name: s("ɵɵinterpolate7"), ModuleName: coreStr},
	Interpolate8:                      &output.ExternalReference{Name: s("ɵɵinterpolate8"), ModuleName: coreStr},
	InterpolateV:                      &output.ExternalReference{Name: s("ɵɵinterpolateV"), ModuleName: coreStr},
	NextContext:                       &output.ExternalReference{Name: s("ɵɵnextContext"), ModuleName: coreStr},
	ResetView:                         &output.ExternalReference{Name: s("ɵɵresetView"), ModuleName: coreStr},
	TemplateCreate:                    &output.ExternalReference{Name: s("ɵɵtemplate"), ModuleName: coreStr},
	Defer:                             &output.ExternalReference{Name: s("ɵɵdefer"), ModuleName: coreStr},
	DeferWhen:                         &output.ExternalReference{Name: s("ɵɵdeferWhen"), ModuleName: coreStr},
	DeferOnIdle:                       &output.ExternalReference{Name: s("ɵɵdeferOnIdle"), ModuleName: coreStr},
	DeferOnImmediate:                  &output.ExternalReference{Name: s("ɵɵdeferOnImmediate"), ModuleName: coreStr},
	DeferOnTimer:                      &output.ExternalReference{Name: s("ɵɵdeferOnTimer"), ModuleName: coreStr},
	DeferOnHover:                      &output.ExternalReference{Name: s("ɵɵdeferOnHover"), ModuleName: coreStr},
	DeferOnInteraction:                &output.ExternalReference{Name: s("ɵɵdeferOnInteraction"), ModuleName: coreStr},
	DeferOnViewport:                   &output.ExternalReference{Name: s("ɵɵdeferOnViewport"), ModuleName: coreStr},
	DeferPrefetchWhen:                 &output.ExternalReference{Name: s("ɵɵdeferPrefetchWhen"), ModuleName: coreStr},
	DeferPrefetchOnIdle:               &output.ExternalReference{Name: s("ɵɵdeferPrefetchOnIdle"), ModuleName: coreStr},
	DeferPrefetchOnImmediate:          &output.ExternalReference{Name: s("ɵɵdeferPrefetchOnImmediate"), ModuleName: coreStr},
	DeferPrefetchOnTimer:              &output.ExternalReference{Name: s("ɵɵdeferPrefetchOnTimer"), ModuleName: coreStr},
	DeferPrefetchOnHover:              &output.ExternalReference{Name: s("ɵɵdeferPrefetchOnHover"), ModuleName: coreStr},
	DeferPrefetchOnInteraction:        &output.ExternalReference{Name: s("ɵɵdeferPrefetchOnInteraction"), ModuleName: coreStr},
	DeferPrefetchOnViewport:           &output.ExternalReference{Name: s("ɵɵdeferPrefetchOnViewport"), ModuleName: coreStr},
	DeferHydrateWhen:                  &output.ExternalReference{Name: s("ɵɵdeferHydrateWhen"), ModuleName: coreStr},
	DeferHydrateNever:                 &output.ExternalReference{Name: s("ɵɵdeferHydrateNever"), ModuleName: coreStr},
	DeferHydrateOnIdle:                &output.ExternalReference{Name: s("ɵɵdeferHydrateOnIdle"), ModuleName: coreStr},
	DeferHydrateOnImmediate:           &output.ExternalReference{Name: s("ɵɵdeferHydrateOnImmediate"), ModuleName: coreStr},
	DeferHydrateOnTimer:               &output.ExternalReference{Name: s("ɵɵdeferHydrateOnTimer"), ModuleName: coreStr},
	DeferHydrateOnHover:               &output.ExternalReference{Name: s("ɵɵdeferHydrateOnHover"), ModuleName: coreStr},
	DeferHydrateOnInteraction:         &output.ExternalReference{Name: s("ɵɵdeferHydrateOnInteraction"), ModuleName: coreStr},
	DeferHydrateOnViewport:            &output.ExternalReference{Name: s("ɵɵdeferHydrateOnViewport"), ModuleName: coreStr},
	DeferEnableTimerScheduling:        &output.ExternalReference{Name: s("ɵɵdeferEnableTimerScheduling"), ModuleName: coreStr},
	EnableIncrementalHydrationRuntime: &output.ExternalReference{Name: s("ɵɵenableIncrementalHydrationRuntime"), ModuleName: coreStr},
	ConditionalCreate:                 &output.ExternalReference{Name: s("ɵɵconditionalCreate"), ModuleName: coreStr},
	ConditionalBranchCreate:           &output.ExternalReference{Name: s("ɵɵconditionalBranchCreate"), ModuleName: coreStr},
	Conditional:                       &output.ExternalReference{Name: s("ɵɵconditional"), ModuleName: coreStr},
	Repeater:                          &output.ExternalReference{Name: s("ɵɵrepeater"), ModuleName: coreStr},
	RepeaterCreate:                    &output.ExternalReference{Name: s("ɵɵrepeaterCreate"), ModuleName: coreStr},
	RepeaterTrackByIndex:              &output.ExternalReference{Name: s("ɵɵrepeaterTrackByIndex"), ModuleName: coreStr},
	RepeaterTrackByIdentity:           &output.ExternalReference{Name: s("ɵɵrepeaterTrackByIdentity"), ModuleName: coreStr},
	ComponentInstance:                 &output.ExternalReference{Name: s("ɵɵcomponentInstance"), ModuleName: coreStr},
	Text:                              &output.ExternalReference{Name: s("ɵɵtext"), ModuleName: coreStr},
	EnableBindings:                    &output.ExternalReference{Name: s("ɵɵenableBindings"), ModuleName: coreStr},
	DisableBindings:                   &output.ExternalReference{Name: s("ɵɵdisableBindings"), ModuleName: coreStr},
	GetCurrentView:                    &output.ExternalReference{Name: s("ɵɵgetCurrentView"), ModuleName: coreStr},
	TextInterpolate:                   &output.ExternalReference{Name: s("ɵɵtextInterpolate"), ModuleName: coreStr},
	TextInterpolate1:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate1"), ModuleName: coreStr},
	TextInterpolate2:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate2"), ModuleName: coreStr},
	TextInterpolate3:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate3"), ModuleName: coreStr},
	TextInterpolate4:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate4"), ModuleName: coreStr},
	TextInterpolate5:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate5"), ModuleName: coreStr},
	TextInterpolate6:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate6"), ModuleName: coreStr},
	TextInterpolate7:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate7"), ModuleName: coreStr},
	TextInterpolate8:                  &output.ExternalReference{Name: s("ɵɵtextInterpolate8"), ModuleName: coreStr},
	TextInterpolateV:                  &output.ExternalReference{Name: s("ɵɵtextInterpolateV"), ModuleName: coreStr},
	RestoreView:                       &output.ExternalReference{Name: s("ɵɵrestoreView"), ModuleName: coreStr},
	PureFunction0:                     &output.ExternalReference{Name: s("ɵɵpureFunction0"), ModuleName: coreStr},
	PureFunction1:                     &output.ExternalReference{Name: s("ɵɵpureFunction1"), ModuleName: coreStr},
	PureFunction2:                     &output.ExternalReference{Name: s("ɵɵpureFunction2"), ModuleName: coreStr},
	PureFunction3:                     &output.ExternalReference{Name: s("ɵɵpureFunction3"), ModuleName: coreStr},
	PureFunction4:                     &output.ExternalReference{Name: s("ɵɵpureFunction4"), ModuleName: coreStr},
	PureFunction5:                     &output.ExternalReference{Name: s("ɵɵpureFunction5"), ModuleName: coreStr},
	PureFunction6:                     &output.ExternalReference{Name: s("ɵɵpureFunction6"), ModuleName: coreStr},
	PureFunction7:                     &output.ExternalReference{Name: s("ɵɵpureFunction7"), ModuleName: coreStr},
	PureFunction8:                     &output.ExternalReference{Name: s("ɵɵpureFunction8"), ModuleName: coreStr},
	PureFunctionV:                     &output.ExternalReference{Name: s("ɵɵpureFunctionV"), ModuleName: coreStr},
	PipeBind1:                         &output.ExternalReference{Name: s("ɵɵpipeBind1"), ModuleName: coreStr},
	PipeBind2:                         &output.ExternalReference{Name: s("ɵɵpipeBind2"), ModuleName: coreStr},
	PipeBind3:                         &output.ExternalReference{Name: s("ɵɵpipeBind3"), ModuleName: coreStr},
	PipeBind4:                         &output.ExternalReference{Name: s("ɵɵpipeBind4"), ModuleName: coreStr},
	PipeBindV:                         &output.ExternalReference{Name: s("ɵɵpipeBindV"), ModuleName: coreStr},
	DomProperty:                       &output.ExternalReference{Name: s("ɵɵdomProperty"), ModuleName: coreStr},
	AriaProperty:                      &output.ExternalReference{Name: s("ɵɵariaProperty"), ModuleName: coreStr},
	Property:                          &output.ExternalReference{Name: s("ɵɵproperty"), ModuleName: coreStr},
	Control:                           &output.ExternalReference{Name: s("ɵɵcontrol"), ModuleName: coreStr},
	ControlCreate:                     &output.ExternalReference{Name: s("ɵɵcontrolCreate"), ModuleName: coreStr},
	AnimationEnterListener:            &output.ExternalReference{Name: s("ɵɵanimateEnterListener"), ModuleName: coreStr},
	AnimationLeaveListener:            &output.ExternalReference{Name: s("ɵɵanimateLeaveListener"), ModuleName: coreStr},
	AnimationEnter:                    &output.ExternalReference{Name: s("ɵɵanimateEnter"), ModuleName: coreStr},
	AnimationLeave:                    &output.ExternalReference{Name: s("ɵɵanimateLeave"), ModuleName: coreStr},
	I18n:                              &output.ExternalReference{Name: s("ɵɵi18n"), ModuleName: coreStr},
	I18nAttributes:                    &output.ExternalReference{Name: s("ɵɵi18nAttributes"), ModuleName: coreStr},
	I18nExp:                           &output.ExternalReference{Name: s("ɵɵi18nExp"), ModuleName: coreStr},
	I18nStart:                         &output.ExternalReference{Name: s("ɵɵi18nStart"), ModuleName: coreStr},
	I18nEnd:                           &output.ExternalReference{Name: s("ɵɵi18nEnd"), ModuleName: coreStr},
	I18nApply:                         &output.ExternalReference{Name: s("ɵɵi18nApply"), ModuleName: coreStr},
	I18nPostprocess:                   &output.ExternalReference{Name: s("ɵɵi18nPostprocess"), ModuleName: coreStr},
	Pipe:                              &output.ExternalReference{Name: s("ɵɵpipe"), ModuleName: coreStr},
	Projection:                        &output.ExternalReference{Name: s("ɵɵprojection"), ModuleName: coreStr},
	ProjectionDef:                     &output.ExternalReference{Name: s("ɵɵprojectionDef"), ModuleName: coreStr},
	Reference:                         &output.ExternalReference{Name: s("ɵɵreference"), ModuleName: coreStr},
	Inject:                            &output.ExternalReference{Name: s("ɵɵinject"), ModuleName: coreStr},
	InjectAttribute:                   &output.ExternalReference{Name: s("ɵɵinjectAttribute"), ModuleName: coreStr},
	DirectiveInject:                   &output.ExternalReference{Name: s("ɵɵdirectiveInject"), ModuleName: coreStr},
	InvalidFactory:                    &output.ExternalReference{Name: s("ɵɵinvalidFactory"), ModuleName: coreStr},
	InvalidFactoryDep:                 &output.ExternalReference{Name: s("ɵɵinvalidFactoryDep"), ModuleName: coreStr},
	TemplateRefExtractor:              &output.ExternalReference{Name: s("ɵɵtemplateRefExtractor"), ModuleName: coreStr},
	ForwardRef:                        &output.ExternalReference{Name: s("forwardRef"), ModuleName: coreStr},
	ResolveForwardRef:                 &output.ExternalReference{Name: s("resolveForwardRef"), ModuleName: coreStr},
	ReplaceMetadata:                   &output.ExternalReference{Name: s("ɵɵreplaceMetadata"), ModuleName: coreStr},
	GetReplaceMetadataURL:             &output.ExternalReference{Name: s("ɵɵgetReplaceMetadataURL"), ModuleName: coreStr},
	ƟɵdefineInjectable:                &output.ExternalReference{Name: s("ɵɵdefineInjectable"), ModuleName: coreStr},
	DeclareInjectable:                 &output.ExternalReference{Name: s("ɵɵngDeclareInjectable"), ModuleName: coreStr},
	InjectableDeclaration:             &output.ExternalReference{Name: s("ɵɵInjectableDeclaration"), ModuleName: coreStr},
	DefineService:                     &output.ExternalReference{Name: s("ɵɵdefineService"), ModuleName: coreStr},
	DeclareService:                    &output.ExternalReference{Name: s("ɵɵngDeclareService"), ModuleName: coreStr},
	ResolveWindow:                     &output.ExternalReference{Name: s("ɵɵresolveWindow"), ModuleName: coreStr},
	ResolveDocument:                   &output.ExternalReference{Name: s("ɵɵresolveDocument"), ModuleName: coreStr},
	ResolveBody:                       &output.ExternalReference{Name: s("ɵɵresolveBody"), ModuleName: coreStr},
	GetComponentDepsFactory:           &output.ExternalReference{Name: s("ɵɵgetComponentDepsFactory"), ModuleName: coreStr},
	DefineComponent:                   &output.ExternalReference{Name: s("ɵɵdefineComponent"), ModuleName: coreStr},
	DeclareComponent:                  &output.ExternalReference{Name: s("ɵɵngDeclareComponent"), ModuleName: coreStr},
	SetComponentScope:                 &output.ExternalReference{Name: s("ɵɵsetComponentScope"), ModuleName: coreStr},
	ChangeDetectionStrategy:           &output.ExternalReference{Name: s("ChangeDetectionStrategy"), ModuleName: coreStr},
	ViewEncapsulation:                 &output.ExternalReference{Name: s("ViewEncapsulation"), ModuleName: coreStr},
	ComponentDeclaration:              &output.ExternalReference{Name: s("ɵɵComponentDeclaration"), ModuleName: coreStr},
	FactoryDeclaration:                &output.ExternalReference{Name: s("ɵɵFactoryDeclaration"), ModuleName: coreStr},
	DeclareFactory:                    &output.ExternalReference{Name: s("ɵɵngDeclareFactory"), ModuleName: coreStr},
	FactoryTarget:                     &output.ExternalReference{Name: s("ɵɵFactoryTarget"), ModuleName: coreStr},
	DefineDirective:                   &output.ExternalReference{Name: s("ɵɵdefineDirective"), ModuleName: coreStr},
	DeclareDirective:                  &output.ExternalReference{Name: s("ɵɵngDeclareDirective"), ModuleName: coreStr},
	DirectiveDeclaration:              &output.ExternalReference{Name: s("ɵɵDirectiveDeclaration"), ModuleName: coreStr},
	InjectorDef:                       &output.ExternalReference{Name: s("ɵɵInjectorDef"), ModuleName: coreStr},
	InjectorDeclaration:               &output.ExternalReference{Name: s("ɵɵInjectorDeclaration"), ModuleName: coreStr},
	DefineInjector:                    &output.ExternalReference{Name: s("ɵɵdefineInjector"), ModuleName: coreStr},
	DeclareInjector:                   &output.ExternalReference{Name: s("ɵɵngDeclareInjector"), ModuleName: coreStr},
	NgModuleDeclaration:               &output.ExternalReference{Name: s("ɵɵNgModuleDeclaration"), ModuleName: coreStr},
	ModuleWithProviders:               &output.ExternalReference{Name: s("ModuleWithProviders"), ModuleName: coreStr},
	DefineNgModule:                    &output.ExternalReference{Name: s("ɵɵdefineNgModule"), ModuleName: coreStr},
	DeclareNgModule:                   &output.ExternalReference{Name: s("ɵɵngDeclareNgModule"), ModuleName: coreStr},
	SetNgModuleScope:                  &output.ExternalReference{Name: s("ɵɵsetNgModuleScope"), ModuleName: coreStr},
	RegisterNgModuleType:              &output.ExternalReference{Name: s("ɵɵregisterNgModuleType"), ModuleName: coreStr},
	PipeDeclaration:                   &output.ExternalReference{Name: s("ɵɵPipeDeclaration"), ModuleName: coreStr},
	DefinePipe:                        &output.ExternalReference{Name: s("ɵɵdefinePipe"), ModuleName: coreStr},
	DeclarePipe:                       &output.ExternalReference{Name: s("ɵɵngDeclarePipe"), ModuleName: coreStr},
	DeclareClassMetadata:              &output.ExternalReference{Name: s("ɵɵngDeclareClassMetadata"), ModuleName: coreStr},
	DeclareClassMetadataAsync:         &output.ExternalReference{Name: s("ɵɵngDeclareClassMetadataAsync"), ModuleName: coreStr},
	SetClassMetadata:                  &output.ExternalReference{Name: s("ɵsetClassMetadata"), ModuleName: coreStr},
	SetClassMetadataAsync:             &output.ExternalReference{Name: s("ɵsetClassMetadataAsync"), ModuleName: coreStr},
	SetClassDebugInfo:                 &output.ExternalReference{Name: s("ɵsetClassDebugInfo"), ModuleName: coreStr},
	QueryRefresh:                      &output.ExternalReference{Name: s("ɵɵqueryRefresh"), ModuleName: coreStr},
	ViewQuery:                         &output.ExternalReference{Name: s("ɵɵviewQuery"), ModuleName: coreStr},
	LoadQuery:                         &output.ExternalReference{Name: s("ɵɵloadQuery"), ModuleName: coreStr},
	ContentQuery:                      &output.ExternalReference{Name: s("ɵɵcontentQuery"), ModuleName: coreStr},
	ViewQuerySignal:                   &output.ExternalReference{Name: s("ɵɵviewQuerySignal"), ModuleName: coreStr},
	ContentQuerySignal:                &output.ExternalReference{Name: s("ɵɵcontentQuerySignal"), ModuleName: coreStr},
	QueryAdvance:                      &output.ExternalReference{Name: s("ɵɵqueryAdvance"), ModuleName: coreStr},
	TwoWayProperty:                    &output.ExternalReference{Name: s("ɵɵtwoWayProperty"), ModuleName: coreStr},
	TwoWayBindingSet:                  &output.ExternalReference{Name: s("ɵɵtwoWayBindingSet"), ModuleName: coreStr},
	TwoWayListener:                    &output.ExternalReference{Name: s("ɵɵtwoWayListener"), ModuleName: coreStr},
	DeclareLet:                        &output.ExternalReference{Name: s("ɵɵdeclareLet"), ModuleName: coreStr},
	StoreLet:                          &output.ExternalReference{Name: s("ɵɵstoreLet"), ModuleName: coreStr},
	ReadContextLet:                    &output.ExternalReference{Name: s("ɵɵreadContextLet"), ModuleName: coreStr},
	ArrowFunction:                     &output.ExternalReference{Name: s("ɵɵarrowFunction"), ModuleName: coreStr},
	AttachSourceLocations:             &output.ExternalReference{Name: s("ɵɵattachSourceLocations"), ModuleName: coreStr},
	NgOnChangesFeature:                &output.ExternalReference{Name: s("ɵɵNgOnChangesFeature"), ModuleName: coreStr},
	ControlFeature:                    &output.ExternalReference{Name: s("ɵɵControlFeature"), ModuleName: coreStr},
	InheritDefinitionFeature:          &output.ExternalReference{Name: s("ɵɵInheritDefinitionFeature"), ModuleName: coreStr},
	ProvidersFeature:                  &output.ExternalReference{Name: s("ɵɵProvidersFeature"), ModuleName: coreStr},
	HostDirectivesFeature:             &output.ExternalReference{Name: s("ɵɵHostDirectivesFeature"), ModuleName: coreStr},
	ExternalStylesFeature:             &output.ExternalReference{Name: s("ɵɵExternalStylesFeature"), ModuleName: coreStr},
	Listener:                          &output.ExternalReference{Name: s("ɵɵlistener"), ModuleName: coreStr},
	GetInheritedFactory:               &output.ExternalReference{Name: s("ɵɵgetInheritedFactory"), ModuleName: coreStr},
	SanitizeHtml:                      &output.ExternalReference{Name: s("ɵɵsanitizeHtml"), ModuleName: coreStr},
	SanitizeStyle:                     &output.ExternalReference{Name: s("ɵɵsanitizeStyle"), ModuleName: coreStr},
	ValidateAttribute:                 &output.ExternalReference{Name: s("ɵɵvalidateAttribute"), ModuleName: coreStr},
	SanitizeResourceUrl:               &output.ExternalReference{Name: s("ɵɵsanitizeResourceUrl"), ModuleName: coreStr},
	SanitizeScript:                    &output.ExternalReference{Name: s("ɵɵsanitizeScript"), ModuleName: coreStr},
	SanitizeUrl:                       &output.ExternalReference{Name: s("ɵɵsanitizeUrl"), ModuleName: coreStr},
	SanitizeUrlOrResourceUrl:          &output.ExternalReference{Name: s("ɵɵsanitizeUrlOrResourceUrl"), ModuleName: coreStr},
	TrustConstantHtml:                 &output.ExternalReference{Name: s("ɵɵtrustConstantHtml"), ModuleName: coreStr},
	TrustConstantResourceUrl:          &output.ExternalReference{Name: s("ɵɵtrustConstantResourceUrl"), ModuleName: coreStr},
	InputDecorator:                    &output.ExternalReference{Name: s("Input"), ModuleName: coreStr},
	OutputDecorator:                   &output.ExternalReference{Name: s("Output"), ModuleName: coreStr},
	ViewChildDecorator:                &output.ExternalReference{Name: s("ViewChild"), ModuleName: coreStr},
	ViewChildrenDecorator:             &output.ExternalReference{Name: s("ViewChildren"), ModuleName: coreStr},
	ContentChildDecorator:             &output.ExternalReference{Name: s("ContentChild"), ModuleName: coreStr},
	ContentChildrenDecorator:          &output.ExternalReference{Name: s("ContentChildren"), ModuleName: coreStr},
	InputSignalBrandWriteType:         &output.ExternalReference{Name: s("ɵINPUT_SIGNAL_BRAND_WRITE_TYPE"), ModuleName: coreStr},
	UnwrapDirectiveSignalInputs:       &output.ExternalReference{Name: s("ɵUnwrapDirectiveSignalInputs"), ModuleName: coreStr},
	UnwrapWritableSignal:              &output.ExternalReference{Name: s("ɵunwrapWritableSignal"), ModuleName: coreStr},
	AssertType:                        &output.ExternalReference{Name: s("ɵassertType"), ModuleName: coreStr},
}
