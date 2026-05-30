package compiler

import "github.com/microsoft/typescript-go/angular-packages/compiler/core"

// Port of angular/packages/compiler/src/compiler_facade_interface.ts
//
// A set of interfaces which are shared between @angular/core and @angular/compiler to allow
// for late binding of @angular/compiler for JIT purposes.

// ExportedCompilerFacade is the global interface for Angular's exported compiler.
type ExportedCompilerFacade struct {
	CompilerFacade CompilerFacade
}

// CompilerFacade is the interface that the Angular compiler exposes to the runtime.
type CompilerFacade interface {
	CompilePipe(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3PipeMetadataFacade) interface{}
	CompilePipeDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, declaration R3DeclarePipeFacade) interface{}
	CompileInjectable(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3InjectableMetadataFacade) interface{}
	CompileInjectableDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3DeclareInjectableFacade) interface{}
	CompileInjector(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3InjectorMetadataFacade) interface{}
	CompileInjectorDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, declaration R3DeclareInjectorFacade) interface{}
	CompileNgModule(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3NgModuleMetadataFacade) interface{}
	CompileNgModuleDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, declaration R3DeclareNgModuleFacade) interface{}
	CompileDirective(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3DirectiveMetadataFacade) interface{}
	CompileDirectiveDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, declaration R3DeclareDirectiveFacade) interface{}
	CompileComponent(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3ComponentMetadataFacade) interface{}
	CompileComponentDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, declaration R3DeclareComponentFacade) interface{}
	CompileFactory(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3FactoryDefMetadataFacade) interface{}
	CompileFactoryDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3DeclareFactoryFacade) interface{}
	CompileService(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3ServiceMetadataFacade) interface{}
	CompileServiceDeclaration(angularCoreEnv CoreEnvironment, sourceMapUrl string, meta R3DeclareServiceFacade) interface{}
	CreateParseSourceSpan(kind string, typeName string, sourceUrl string) FacadeParseSourceSpan
}

// CoreEnvironment is a map of @angular/core symbol names to their values.
type CoreEnvironment map[string]interface{}

// FacadeResourceLoader corresponds to the ResourceLoader type in the facade interface.
type FacadeResourceLoader interface {
	Get(url string) (string, error)
}

// OpaqueValue is any value from the outside world (TypeScript type system).
type OpaqueValue = interface{}

// FacadeType corresponds to TypeScript's Function type for Angular types.
type FacadeType = interface{}

// anyFacade corresponds to the any enum in the facade interface.
// Note: any is already defined in injectable_compiler_2.go.

// R3DependencyMetadataFacade holds dependency injection metadata from the JIT facade.
type R3DependencyMetadataFacade struct {
	Token     OpaqueValue
	Attribute *string
	Host      bool
	Optional  bool
	Self      bool
	SkipSelf  bool
}

// R3DeclareDependencyMetadataFacade holds dependency injection metadata for partial pipeline.
type R3DeclareDependencyMetadataFacade struct {
	Token     OpaqueValue
	Attribute *bool
	Host      *bool
	Optional  *bool
	Self      *bool
	SkipSelf  *bool
}

// R3PipeMetadataFacade holds pipe metadata from the facade.
type R3PipeMetadataFacade struct {
	Name         string
	Type         FacadeType
	PipeName     *string
	Pure         bool
	IsStandalone bool
}

// R3InjectableMetadataFacade holds injectable metadata from the facade.
type R3InjectableMetadataFacade struct {
	Name              string
	Type              FacadeType
	TypeArgumentCount int
	ProvidedIn        OpaqueValue // Type | 'root' | 'platform' | 'any' | nil
	UseClass          OpaqueValue
	UseFactory        OpaqueValue
	UseExisting       OpaqueValue
	UseValue          OpaqueValue
	Deps              []R3DependencyMetadataFacade
}

// R3ServiceMetadataFacade holds service metadata from the facade.
type R3ServiceMetadataFacade struct {
	Name              string
	Type              FacadeType
	TypeArgumentCount int
	AutoProvided      *bool
	Factory           OpaqueValue
}

// R3NgModuleMetadataFacade holds NgModule metadata from the facade.
type R3NgModuleMetadataFacade struct {
	Type         FacadeType
	Bootstrap    []FacadeType
	Declarations []FacadeType
	Imports      []FacadeType
	Exports      []FacadeType
	Schemas      []struct{ Name string }
	Id           *string
}

// R3InjectorMetadataFacade holds injector metadata from the facade.
type R3InjectorMetadataFacade struct {
	Name      string
	Type      FacadeType
	Providers []OpaqueValue
	Imports   []OpaqueValue
}

// R3HostDirectiveMetadataFacade holds host directive metadata from the facade.
type R3HostDirectiveMetadataFacade struct {
	Directive FacadeType
	Inputs    []string
	Outputs   []string
}

// R3DirectiveMetadataFacade holds directive metadata from the facade.
type R3DirectiveMetadataFacade struct {
	Name                   string
	Type                   FacadeType
	TypeSourceSpan         FacadeParseSourceSpan
	Selector               *string
	Queries                []R3QueryMetadataFacade
	Host                   map[string]string
	PropMetadata           map[string][]OpaqueValue
	Lifecycle              struct{ UsesOnChanges bool }
	Inputs                 []interface{} // string | {name, alias?, required?}
	Outputs                []string
	UsesInheritance        bool
	ControlCreate          *struct{ PassThroughInput *string }
	ExportAs               []string
	Providers              []OpaqueValue
	ViewQueries            []R3QueryMetadataFacade
	IsStandalone           bool
	HostDirectives         []R3HostDirectiveMetadataFacade
	IsSignal               bool
	LegacyOptionalChaining bool
}

// R3ComponentMetadataFacade extends R3DirectiveMetadataFacade with component-specific metadata.
type R3ComponentMetadataFacade struct {
	R3DirectiveMetadataFacade
	Template                 string
	PreserveWhitespaces      bool
	Animations               []OpaqueValue
	Declarations             []R3TemplateDependencyFacade
	Styles                   []string
	Encapsulation            core.ViewEncapsulation
	ViewProviders            []OpaqueValue
	ChangeDetection          *core.ChangeDetectionStrategy
	HasDirectiveDependencies bool
}

// LegacyInputPartialMapping is a string or array for backward-compatible input mappings.
// Corresponds to: type LegacyInputPartialMapping = string | [bindingPropertyName: string, classPropertyName: string, transformFunction?: Function]
type LegacyInputPartialMapping interface{} // string | []interface{}

// R3DeclareDirectiveFacade holds partial directive declaration metadata.
type R3DeclareDirectiveFacade struct {
	Selector               *string
	Type                   FacadeType
	Version                string
	Inputs                 map[string]interface{} // various input mapping types
	Outputs                map[string]string
	Host                   *R3DeclareDirectiveHostFacade
	Queries                []R3DeclareQueryMetadataFacade
	ViewQueries            []R3DeclareQueryMetadataFacade
	Providers              OpaqueValue
	ExportAs               []string
	UsesInheritance        *bool
	UsesOnChanges          *bool
	ControlCreate          *struct{ PassThroughInput *string }
	IsStandalone           *bool
	IsSignal               *bool
	HostDirectives         []R3HostDirectiveMetadataFacade
	LegacyOptionalChaining *bool
}

// R3DeclareDirectiveHostFacade holds host metadata for partial declaration.
type R3DeclareDirectiveHostFacade struct {
	Attributes     map[string]OpaqueValue
	Listeners      map[string]string
	Properties     map[string]string
	ClassAttribute *string
	StyleAttribute *string
}

// R3DeclareComponentFacade extends R3DeclareDirectiveFacade with component-specific declaration metadata.
type R3DeclareComponentFacade struct {
	R3DeclareDirectiveFacade
	Template               string
	IsInline               *bool
	Styles                 []string
	Dependencies           []R3DeclareTemplateDependencyFacade
	Components             []R3DeclareDirectiveDependencyFacade
	Directives             []R3DeclareDirectiveDependencyFacade
	Pipes                  map[string]interface{} // pipe name → type or func
	DeferBlockDependencies []interface{}
	ViewProviders          OpaqueValue
	Animations             OpaqueValue
	ChangeDetection        *core.ChangeDetectionStrategy
	Encapsulation          *core.ViewEncapsulation
	PreserveWhitespaces    *bool
}

// R3DeclareTemplateDependencyFacade is a union of directive/pipe/ngmodule dependency facades.
type R3DeclareTemplateDependencyFacade struct {
	Kind string
	// Embedded fields for each dependency kind:
	R3DeclareDirectiveDependencyFacade
	R3DeclarePipeDependencyFacade
	R3DeclareNgModuleDependencyFacade
}

// R3DeclareDirectiveDependencyFacade holds metadata for a directive dependency in partial pipeline.
type R3DeclareDirectiveDependencyFacade struct {
	Kind     *string // "directive" | "component" | nil
	Selector string
	Type     interface{} // OpaqueValue | func() OpaqueValue
	Inputs   []string
	Outputs  []string
	ExportAs []string
}

// R3DeclarePipeDependencyFacade holds metadata for a pipe dependency in partial pipeline.
type R3DeclarePipeDependencyFacade struct {
	Kind *string // "pipe" | nil
	Name string
	Type interface{} // OpaqueValue | func() OpaqueValue
}

// R3DeclareNgModuleDependencyFacade holds metadata for an NgModule dependency in partial pipeline.
type R3DeclareNgModuleDependencyFacade struct {
	Kind string      // "ngmodule"
	Type interface{} // OpaqueValue | func() OpaqueValue
}

// R3TemplateDependencyKindFacade enumerates template dependency kinds.
type R3TemplateDependencyKindFacade int

const (
	R3TemplateDependencyKindFacadeDirective R3TemplateDependencyKindFacade = 0
	R3TemplateDependencyKindFacadePipe      R3TemplateDependencyKindFacade = 1
	R3TemplateDependencyKindFacadeNgModule  R3TemplateDependencyKindFacade = 2
)

// R3TemplateDependencyFacade holds template dependency metadata.
type R3TemplateDependencyFacade struct {
	Kind R3TemplateDependencyKindFacade
	Type interface{} // OpaqueValue | func() OpaqueValue
}

// R3FactoryDefMetadataFacade holds factory def metadata from the facade.
type R3FactoryDefMetadataFacade struct {
	Name              string
	Type              FacadeType
	TypeArgumentCount int
	Deps              []R3DependencyMetadataFacade
	Target            any
}

// R3DeclareFactoryFacade holds partial factory declaration metadata.
type R3DeclareFactoryFacade struct {
	Type   FacadeType
	Deps   interface{} // []R3DeclareDependencyMetadataFacade | "invalid" | nil
	Target any
}

// R3DeclareInjectableFacade holds partial injectable declaration metadata.
type R3DeclareInjectableFacade struct {
	Type        FacadeType
	ProvidedIn  OpaqueValue // Type | 'root' | 'platform' | 'any' | nil
	UseClass    OpaqueValue
	UseFactory  OpaqueValue
	UseExisting OpaqueValue
	UseValue    OpaqueValue
	Deps        []R3DeclareDependencyMetadataFacade
}

// R3DeclareServiceFacade holds partial service declaration metadata.
type R3DeclareServiceFacade struct {
	Type         FacadeType
	AutoProvided *bool
	Factory      OpaqueValue
}

// ViewEncapsulationFacade corresponds to the core.ViewEncapsulation enum in the facade.
// Note: core.ViewEncapsulation is already defined in core.go.

// R3QueryMetadataFacade holds query metadata from the facade.
type R3QueryMetadataFacade struct {
	PropertyName            string
	First                   bool
	Predicate               interface{} // OpaqueValue | []string
	Descendants             bool
	EmitDistinctChangesOnly bool
	Read                    OpaqueValue
	Static                  bool
	IsSignal                bool
}

// R3DeclareQueryMetadataFacade holds query metadata for partial declaration.
type R3DeclareQueryMetadataFacade struct {
	PropertyName            string
	First                   *bool
	Predicate               interface{} // OpaqueValue | []string
	Descendants             *bool
	Read                    OpaqueValue
	Static                  *bool
	EmitDistinctChangesOnly *bool
	IsSignal                *bool
}

// R3DeclareInjectorFacade holds partial injector declaration metadata.
type R3DeclareInjectorFacade struct {
	Type      FacadeType
	Imports   []OpaqueValue
	Providers []OpaqueValue
}

// R3DeclareNgModuleFacade holds partial NgModule declaration metadata.
type R3DeclareNgModuleFacade struct {
	Type         FacadeType
	Bootstrap    interface{} // []OpaqueValue | func() []OpaqueValue
	Declarations interface{} // []OpaqueValue | func() []OpaqueValue
	Imports      interface{} // []OpaqueValue | func() []OpaqueValue
	Exports      interface{} // []OpaqueValue | func() []OpaqueValue
	Schemas      []OpaqueValue
	Id           OpaqueValue
}

// R3DeclarePipeFacade holds partial pipe declaration metadata.
type R3DeclarePipeFacade struct {
	Type         FacadeType
	Version      string
	Name         string
	Pure         *bool
	IsStandalone *bool
}

// FacadeParseSourceSpan represents a source span used in JIT pipeline.
type FacadeParseSourceSpan struct {
	Start     interface{}
	End       interface{}
	Details   interface{}
	FullStart interface{}
}
