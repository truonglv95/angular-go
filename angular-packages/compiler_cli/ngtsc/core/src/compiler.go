package src

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type LazyCompilationState struct {
	IsCore                              bool
	TraitCompiler                       any // TraitCompiler
	Reflector                           any // TypeScriptReflectionHost
	MetaReader                          any // MetadataReader
	ScopeRegistry                       any // LocalModuleScopeRegistry
	TypeCheckScopeRegistry              any // TypeCheckScopeRegistry
	ExportReferenceGraph                any // ReferenceGraph
	DtsTransforms                       any // DtsTransformRegistry
	AliasingHost                        any // AliasingHost
	RefEmitter                          any // ReferenceEmitter
	TemplateTypeChecker                 any // TemplateTypeChecker
	ResourceRegistry                    any // ResourceRegistry
	ExtendedTemplateChecker             any // ExtendedTemplateChecker
	TemplateSemanticsChecker            any // TemplateSemanticsChecker
	SourceFileValidator                 any // SourceFileValidator
	JitDeclarationRegistry              any // JitDeclarationRegistry
	SupportJitMode                      bool
	LocalCompilationExtraImportsTracker any // LocalCompilationExtraImportsTracker
}

type CompilationTicketKind int

const (
	CompilationTicketKindFresh CompilationTicketKind = iota
	CompilationTicketKindIncrementalTypeScript
	CompilationTicketKindIncrementalResource
)

type FreshCompilationTicket struct {
	Kind                      CompilationTicketKind
	Options                   any // NgCompilerOptions
	IncrementalBuildStrategy  any // IncrementalBuildStrategy
	ProgramDriver             any // ProgramDriver
	EnableTemplateTypeChecker bool
	UsePoisonedData           bool
	TsProgram                 Program
	PerfRecorder              any // ActivePerfRecorder
}

type IncrementalTypeScriptCompilationTicket struct {
	Kind                      CompilationTicketKind
	Options                   any // NgCompilerOptions
	NewProgram                Program
	IncrementalBuildStrategy  any // IncrementalBuildStrategy
	IncrementalCompilation    any // IncrementalCompilation
	ProgramDriver             any // ProgramDriver
	EnableTemplateTypeChecker bool
	UsePoisonedData           bool
	PerfRecorder              any // ActivePerfRecorder
}

type IncrementalResourceCompilationTicket struct {
	Kind                  CompilationTicketKind
	Compiler              *NgCompiler
	ModifiedResourceFiles map[string]struct{}
	PerfRecorder          any // ActivePerfRecorder
}

type CompilationTicket interface {
	GetKind() CompilationTicketKind
}

func (t *FreshCompilationTicket) GetKind() CompilationTicketKind                 { return t.Kind }
func (t *IncrementalTypeScriptCompilationTicket) GetKind() CompilationTicketKind { return t.Kind }
func (t *IncrementalResourceCompilationTicket) GetKind() CompilationTicketKind   { return t.Kind }

type NgCompiler struct {
	compilation             *LazyCompilationState
	constructionDiagnostics []ast.Diagnostic
	nonTemplateDiagnostics  []ast.Diagnostic
	closureCompilerEnabled  bool
	currentProgram          Program
	entryPoint              ast.SourceFile
	moduleResolver          any // ModuleResolver
	resourceManager         any // AdapterResourceLoader
	cycleAnalyzer           any // CycleAnalyzer

	IgnoreForDiagnostics map[string]struct{}
	IgnoreForEmit        map[string]struct{}

	EnableTemplateTypeChecker bool
	enableBlockSyntax         bool
	enableLetSyntax           bool
	angularCoreVersion        *string
	enableHmr                 bool
	implicitStandaloneValue   bool
	enableSelectorless        bool
	emitDeclarationOnly       bool

	delegatingPerfRecorder any // DelegatingPerfRecorder

	adapter                any // NgCompilerAdapter
	Options                any // NgCompilerOptions
	inputProgram           Program
	programDriver          any // ProgramDriver
	incrementalStrategy    any // IncrementalBuildStrategy
	incrementalCompilation any // IncrementalCompilation
	UsePoisonedData        bool
	livePerfRecorder       any // ActivePerfRecorder
}

func NgCompilerFromTicket(ticket CompilationTicket, adapter any) *NgCompiler {
	switch t := ticket.(type) {
	case *FreshCompilationTicket:
		return NewNgCompiler(
			adapter, t.Options, t.TsProgram, t.ProgramDriver,
			t.IncrementalBuildStrategy, nil, t.EnableTemplateTypeChecker,
			t.UsePoisonedData, t.PerfRecorder,
		)
	case *IncrementalTypeScriptCompilationTicket:
		return NewNgCompiler(
			adapter, t.Options, t.NewProgram, t.ProgramDriver,
			t.IncrementalBuildStrategy, t.IncrementalCompilation, t.EnableTemplateTypeChecker,
			t.UsePoisonedData, t.PerfRecorder,
		)
	case *IncrementalResourceCompilationTicket:
		compiler := t.Compiler
		compiler.UpdateWithChangedResources(t.ModifiedResourceFiles, t.PerfRecorder)
		return compiler
	}
	return nil
}

func NewNgCompiler(
	adapter any,
	options any,
	inputProgram Program,
	programDriver any,
	incrementalStrategy any,
	incrementalCompilation any,
	enableTemplateTypeChecker bool,
	usePoisonedData bool,
	livePerfRecorder any,
) *NgCompiler {
	c := &NgCompiler{
		adapter:                   adapter,
		Options:                   options,
		inputProgram:              inputProgram,
		programDriver:             programDriver,
		incrementalStrategy:       incrementalStrategy,
		incrementalCompilation:    incrementalCompilation,
		UsePoisonedData:           usePoisonedData,
		livePerfRecorder:          livePerfRecorder,
		EnableTemplateTypeChecker: enableTemplateTypeChecker,
	}
	// minimal constructor logic for the structure
	return c
}

func (c *NgCompiler) UpdateWithChangedResources(changedResources map[string]struct{}, perfRecorder any) {
	c.livePerfRecorder = perfRecorder
	// ... c.delegatingPerfRecorder.target = perfRecorder
}

func (c *NgCompiler) GetResourceDependencies(file ast.SourceFile) []string {
	return nil
}

func (c *NgCompiler) GetDiagnostics() []ast.Diagnostic {
	return nil
}

func (c *NgCompiler) GetDiagnosticsForFile(file ast.SourceFile, optimizeFor any) []ast.Diagnostic {
	return nil
}

func (c *NgCompiler) GetDiagnosticsForComponent(component ast.ClassDeclaration) []ast.Diagnostic {
	return nil
}

func (c *NgCompiler) GetOptionDiagnostics() []ast.Diagnostic {
	return c.constructionDiagnostics
}

func (c *NgCompiler) GetCurrentProgram() Program {
	return c.currentProgram
}

func (c *NgCompiler) AnalyzeAsync() error {
	return nil
}

type PrepareEmitResult struct {
	Transformers CustomTransformers
}

func (c *NgCompiler) PrepareEmit() PrepareEmitResult {
	return PrepareEmitResult{}
}

func (c *NgCompiler) GetIndexedComponents() map[any]any {
	return nil
}

func (c *NgCompiler) GetApiDocumentation(entryPoint string, privateModules map[string]struct{}) any {
	return nil
}

func (c *NgCompiler) Xi18n(ctx any) {
}

func (c *NgCompiler) EmitHmrUpdateModule(node any) *string {
	return nil
}

// Setup logic translation
func (c *NgCompiler) setup() {
	// ... constructionDiagnostics
	// c.moduleResolver = new ModuleResolver(c.inputProgram, c.Options, c.adapter, moduleResolutionCache)
	// c.resourceManager = new AdapterResourceLoader(c.adapter, c.Options)
	// c.cycleAnalyzer = new CycleAnalyzer(...)
	// c.incrementalStrategy.setIncrementalState(...)

	// c.IgnoreForDiagnostics = new Set(...)
	// c.IgnoreForEmit = c.adapter.ignoreForEmit
}
