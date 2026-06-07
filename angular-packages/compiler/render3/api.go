package render3

type Expression any
type ParseSourceSpan any

type DeferBlockDepsEmitMode int

const (
	DeferBlockDepsEmitMode_PerBlock DeferBlockDepsEmitMode = iota
	DeferBlockDepsEmitMode_PerComponent
)

type DeclarationListEmitMode int

const (
	DeclarationListEmitMode_Direct DeclarationListEmitMode = iota
	DeclarationListEmitMode_Closure
	DeclarationListEmitMode_ClosureResolved
	DeclarationListEmitMode_RuntimeResolved
)

type R3DirectiveMetadata struct {
	Name              string
	Type              R3Reference
	TypeArgumentCount int
	TypeSourceSpan    ParseSourceSpan
	Deps              interface{} // []R3DependencyMetadata | "invalid" | null
	Selector          *string
	Queries           []R3QueryMetadata
	ViewQueries       []R3QueryMetadata
	Host              R3HostMetadata
	Lifecycle         struct {
		UsesOnChanges bool
	}
	Inputs          map[string]R3InputMetadata
	InputProperties   []string
	Outputs         map[string]string
	OutputProperties  []string
	UsesInheritance bool
	ControlCreate   *struct {
		PassThroughInput *string
	}
	ExportAs               []string
	Providers              Expression
	IsStandalone           bool
	IsSignal               bool
	HostDirectives         []R3HostDirectiveMetadata
	LegacyOptionalChaining bool
}

type R3ComponentMetadata[DeclarationT R3TemplateDependencyMetadata] struct {
	R3DirectiveMetadata
	Template                 Template
	Declarations             []DeclarationT
	Defer                    R3ComponentDeferMetadata
	DeclarationListEmitMode  DeclarationListEmitMode
	Styles                   []string
	ExternalStyles           []string
	Encapsulation            int // ViewEncapsulation
	Animations               Expression
	ViewProviders            Expression
	RelativeContextFilePath  string
	I18nUseExternalIds       bool
	ChangeDetection          Expression // ChangeDetectionStrategy | Expression | null
	RelativeTemplatePath     *string
	HasDirectiveDependencies bool
	RawImports               Expression
	ForeignImports           []R3ForeignComponentMetadata
}

type R3ComponentDeferMetadata struct {
	Mode           DeferBlockDepsEmitMode
	Blocks         map[any]Expression // for PerBlock
	DependenciesFn Expression         // for PerComponent
}

type R3InputMetadata struct {
	ClassPropertyName   string
	BindingPropertyName string
	Required            bool
	IsSignal            bool
	TransformFunction   Expression
}

type R3TemplateDependencyKind int

const (
	R3TemplateDependencyKind_Directive R3TemplateDependencyKind = iota
	R3TemplateDependencyKind_Pipe
	R3TemplateDependencyKind_NgModule
)

type R3TemplateDependency struct {
	Kind R3TemplateDependencyKind
	Type Expression
}

type R3TemplateDependencyMetadata interface {
	GetKind() R3TemplateDependencyKind
}

type R3DirectiveDependencyMetadata struct {
	R3TemplateDependency
	Selector    string
	Inputs      []string
	Outputs     []string
	ExportAs    []string
	IsComponent bool
}

func (m R3DirectiveDependencyMetadata) GetKind() R3TemplateDependencyKind {
	return m.Kind
}

type R3PipeDependencyMetadata struct {
	R3TemplateDependency
	Name string
}

func (m R3PipeDependencyMetadata) GetKind() R3TemplateDependencyKind {
	return m.Kind
}

type R3NgModuleDependencyMetadata struct {
	R3TemplateDependency
}

func (m R3NgModuleDependencyMetadata) GetKind() R3TemplateDependencyKind {
	return m.Kind
}

type R3ForeignComponentMetadata struct {
	Name      string
	Component Expression
}

type R3QueryMetadata struct {
	PropertyName            string
	First                   bool
	Predicate               interface{} // MaybeForwardRefExpression | []string
	Descendants             bool
	EmitDistinctChangesOnly bool
	Read                    Expression
	Static                  bool
	IsSignal                bool
}

type SpecialAttributes struct {
	StyleAttr *string
	ClassAttr *string
}

type R3HostMetadata struct {
	Attributes        map[string]Expression
	Listeners         map[string]string
	Properties        map[string]string
	SpecialAttributes SpecialAttributes
}

type R3HostDirectiveMetadata struct {
	Directive          R3Reference
	IsForwardReference bool
	Inputs             map[string]string
	Outputs            map[string]string
}

type R3DeferResolverFunctionMetadata struct {
	Mode         DeferBlockDepsEmitMode
	Dependencies interface{} // []R3DeferPerBlockDependency | []R3DeferPerComponentDependency
}

type R3DeferPerBlockDependency struct {
	TypeReference   Expression
	SymbolName      string
	IsDeferrable    bool
	ImportPath      *string
	IsDefaultImport bool
}

type R3DeferPerComponentDependency struct {
	SymbolName      string
	ImportPath      string
	LocalName       string
	IsDefaultImport bool
}

func (m R3TemplateDependency) GetKind() R3TemplateDependencyKind {
	return m.Kind
}
