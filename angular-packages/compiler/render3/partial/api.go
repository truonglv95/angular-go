package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type R3PartialDeclaration struct {
	MinVersion string
	Version    string
	NgImport   output.Expression
	Type       output.Expression
}

type LegacyInputPartialMapping interface{} // string | []interface{}

type R3DeclareDirectiveMetadata struct {
	R3PartialDeclaration
	Selector        *string
	Inputs          map[string]interface{} // map of string -> map[string]interface{} | LegacyInputPartialMapping
	Outputs         map[string]string
	Host            *R3DeclareDirectiveHostMetadata
	Queries         []R3DeclareQueryMetadata
	ViewQueries     []R3DeclareQueryMetadata
	Providers       output.Expression
	ExportAs        []string
	UsesInheritance *bool
	UsesOnChanges   *bool
	ControlCreate   *R3DeclareDirectiveControlCreateMetadata
	IsStandalone    *bool
	IsSignal        *bool
	HostDirectives  []R3DeclareHostDirectiveMetadata
}

type R3DeclareDirectiveHostMetadata struct {
	Attributes     map[string]output.Expression
	Listeners      map[string]string
	Properties     map[string]string
	ClassAttribute *string
	StyleAttribute *string
}

type R3DeclareDirectiveControlCreateMetadata struct {
	PassThroughInput *string
}

type R3DeclareComponentMetadata struct {
	R3DeclareDirectiveMetadata
	Template               output.Expression
	IsInline               *bool
	Styles                 []string
	Components             []R3DeclareDirectiveDependencyMetadata
	Directives             []R3DeclareDirectiveDependencyMetadata
	Dependencies           []R3DeclareTemplateDependencyMetadata
	DeferBlockDependencies []output.Expression
	Pipes                  map[string]interface{} // output.Expression | func() output.Expression
	ViewProviders          output.Expression
	Animations             output.Expression
	ChangeDetection        *core.ChangeDetectionStrategy
	Encapsulation          *core.ViewEncapsulation
	PreserveWhitespaces    *bool
}

type R3DeclareTemplateDependencyMetadata interface{} // R3DeclareDirectiveDependencyMetadata | R3DeclarePipeDependencyMetadata | R3DeclareNgModuleDependencyMetadata

type R3DeclareDirectiveDependencyMetadata struct {
	Kind     string // "directive" | "component"
	Selector string
	Type     interface{} // output.Expression | func() output.Expression
	Inputs   []string
	Outputs  []string
	ExportAs []string
}

type R3DeclarePipeDependencyMetadata struct {
	Kind string // "pipe"
	Name string
	Type interface{} // output.Expression | func() output.Expression
}

type R3DeclareNgModuleDependencyMetadata struct {
	Kind string      // "ngmodule"
	Type interface{} // output.Expression | func() output.Expression
}

type R3DeclareQueryMetadata struct {
	PropertyName            string
	First                   *bool
	Predicate               interface{} // output.Expression | []string
	Descendants             *bool
	EmitDistinctChangesOnly *bool
	Read                    output.Expression
	Static                  *bool
	IsSignal                bool
}

type R3DeclareNgModuleMetadata struct {
	R3PartialDeclaration
	Bootstrap    []output.Expression
	Declarations []output.Expression
	Imports      []output.Expression
	Exports      []output.Expression
	Schemas      []output.Expression
	Id           output.Expression
}

type R3DeclareInjectorMetadata struct {
	R3PartialDeclaration
	Providers output.Expression
	Imports   []output.Expression
}

type R3DeclarePipeMetadata struct {
	R3PartialDeclaration
	Name         string
	Pure         *bool
	IsStandalone *bool
}

type R3DeclareFactoryMetadata struct {
	R3PartialDeclaration
	Deps   interface{} // []R3DeclareDependencyMetadata | "invalid" | null
	Target render3.FactoryTarget
}

type R3DeclareInjectableMetadata struct {
	R3PartialDeclaration
	ProvidedIn  output.Expression
	UseClass    output.Expression
	UseFactory  output.Expression
	UseExisting output.Expression
	UseValue    output.Expression
	Deps        []R3DeclareDependencyMetadata
}

type R3DeclareDependencyMetadata struct {
	Token     output.Expression // null is allowed
	Attribute *bool
	Host      *bool
	Optional  *bool
	Self      *bool
	SkipSelf  *bool
}

type R3DeclareClassMetadata struct {
	R3PartialDeclaration
	Decorators     output.Expression
	CtorParameters output.Expression
	PropDecorators output.Expression
}

type R3DeclareClassMetadataAsync struct {
	R3PartialDeclaration
	ResolveDeferredDeps output.Expression
	ResolveMetadata     output.Expression
}

type R3DeclareHostDirectiveMetadata struct {
	Directive output.Expression
	Inputs    []string
	Outputs   []string
}

type R3DeclareServiceMetadata struct {
	R3PartialDeclaration
	AutoProvided *bool
	Factory      output.Expression
}
