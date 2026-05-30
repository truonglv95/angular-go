package api

import "github.com/microsoft/typescript-go/internal/ast"

type CompilerOptions map[string]interface{}

type EmitFlags int

const (
	EmitFlags_DTS EmitFlags = 1 << iota
	EmitFlags_JS
	EmitFlags_Metadata
	EmitFlags_I18nBundle
	EmitFlags_Codegen

	EmitFlags_Default = EmitFlags_DTS | EmitFlags_JS | EmitFlags_Codegen
	EmitFlags_All     = EmitFlags_DTS | EmitFlags_JS | EmitFlags_Metadata | EmitFlags_I18nBundle | EmitFlags_Codegen
)

const UNKNOWN_ERROR_CODE = 500

type CustomTransformers struct {
	BeforeTs []TransformerFactory
	AfterTs  []TransformerFactory
}

type Program interface {
	GetTsProgram() Program_ts
	GetTsOptionDiagnostics(cancellationToken CancellationToken) []ast.Diagnostic
	GetNgOptionDiagnostics(cancellationToken CancellationToken) []ast.Diagnostic
	GetTsSyntacticDiagnostics(sourceFile ast.SourceFile, cancellationToken CancellationToken) []ast.Diagnostic
	GetNgStructuralDiagnostics(cancellationToken CancellationToken) []ast.Diagnostic
	GetTsSemanticDiagnostics(sourceFile ast.SourceFile, cancellationToken CancellationToken) []ast.Diagnostic
	GetNgSemanticDiagnostics(fileName string, cancellationToken CancellationToken) []ast.Diagnostic
	LoadNgStructureAsync() error
	ListLazyRoutes(entryRoute string) []LazyRoute
	Emit(opts *EmitOptions) EmitResult
	GetEmittedSourceFiles() map[string]ast.SourceFile
}

type CompilerHost interface {
	CompilerHost_ts
}

type TsEmitCallback func(args TsEmitArguments) EmitResult
type TsMergeEmitResultsCallback func(results []EmitResult) EmitResult

type TsEmitArguments struct {
	Program            Program_ts
	Options            CompilerOptions
	TargetSourceFile   ast.SourceFile
	WriteFile          WriteFileCallback
	CancellationToken  CancellationToken
	EmitOnlyDtsFiles   bool
	CustomTransformers *CustomTransformers_ts
}

type LazyRoute struct {
	Route            string
	Module           struct{ Name, FilePath string }
	ReferencedModule struct{ Name, FilePath string }
}

type EmitOptions struct {
	EmitFlags                EmitFlags
	ForceEmit                bool
	CancellationToken        CancellationToken
	CustomTransformers       *CustomTransformers
	EmitCallback             TsEmitCallback
	MergeEmitResultsCallback TsMergeEmitResultsCallback
}
