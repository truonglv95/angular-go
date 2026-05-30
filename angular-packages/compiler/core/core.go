package core

const EmitDistinctChangesOnlyDefaultValue = true

type ViewEncapsulation int

const (
	ViewEncapsulationEmulated                      ViewEncapsulation = 0
	ViewEncapsulationNone                          ViewEncapsulation = 2
	ViewEncapsulationShadowDom                     ViewEncapsulation = 3
	ViewEncapsulationExperimentalIsolatedShadowDom ViewEncapsulation = 4
)

type ChangeDetectionStrategy int

const (
	ChangeDetectionStrategyOnPush  ChangeDetectionStrategy = 0
	ChangeDetectionStrategyDefault ChangeDetectionStrategy = 1
	ChangeDetectionStrategyEager   ChangeDetectionStrategy = 1
)

type Input struct {
	Alias     *string
	Required  *bool
	Transform func(value any) any
	IsSignal  bool
}

type InputFlags int

const (
	InputFlagsNone                       InputFlags = 0
	InputFlagsSignalBased                InputFlags = 1 << 0
	InputFlagsHasDecoratorInputTransform InputFlags = 1 << 1
)

type Output struct {
	Alias *string
}

type HostBinding struct {
	HostPropertyName *string
}

type HostListener struct {
	EventName *string
	Args      []string
}

type SchemaMetadata struct {
	Name string
}

var CustomElementsSchema = SchemaMetadata{
	Name: "custom-elements",
}

var NoErrorsSchema = SchemaMetadata{
	Name: "no-errors-schema",
}

type Type any

type InjectFlags int

const (
	InjectFlagsDefault  InjectFlags = 0
	InjectFlagsHost     InjectFlags = 1 << 0
	InjectFlagsSelf     InjectFlags = 1 << 1
	InjectFlagsSkipSelf InjectFlags = 1 << 2
	InjectFlagsOptional InjectFlags = 1 << 3
	InjectFlagsForPipe  InjectFlags = 1 << 4
)

type MissingTranslationStrategy int

const (
	MissingTranslationStrategyError   MissingTranslationStrategy = 0
	MissingTranslationStrategyWarning MissingTranslationStrategy = 1
	MissingTranslationStrategyIgnore  MissingTranslationStrategy = 2
)

type SelectorFlags int

const (
	SelectorFlagsNot       SelectorFlags = 0b0001
	SelectorFlagsAttribute SelectorFlags = 0b0010
	SelectorFlagsElement   SelectorFlags = 0b0100
	SelectorFlagsClass     SelectorFlags = 0b1000
)

type RenderFlags int

const (
	RenderFlagsCreate RenderFlags = 0b01
	RenderFlagsUpdate RenderFlags = 0b10
)

type AttributeMarker int

const (
	AttributeMarkerNamespaceURI AttributeMarker = 0
	AttributeMarkerClasses      AttributeMarker = 1
	AttributeMarkerStyles       AttributeMarker = 2
	AttributeMarkerBindings     AttributeMarker = 3
	AttributeMarkerTemplate     AttributeMarker = 4
	AttributeMarkerProjectAs    AttributeMarker = 5
	AttributeMarkerI18n         AttributeMarker = 6
)

var SVG_NAMESPACE = "svg"
var MATH_ML_NAMESPACE = "math"

type SecurityContext int

const (
	SecurityContextNone SecurityContext = iota
	SecurityContextHTML
	SecurityContextStyle
	SecurityContextScript
	SecurityContextURL
	SecurityContextResourceURL
	SecurityContextAttributeNoBinding
)
