package ngtsc

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core/src"
)

type NgtscProgram struct {
	compiler               *src.NgCompiler
	host                   *src.NgCompilerHost
	options                any // NgCompilerOptions
	tsProgram              any
	reuseTsProgram         any
	closureCompilerEnabled bool
}

func NewNgtscProgram(
	rootNames []string,
	options any, // NgCompilerOptions
	delegateHost any,
	oldProgram *NgtscProgram,
) *NgtscProgram {
	// 1:1 structural setup
	return &NgtscProgram{}
}

func (p *NgtscProgram) GetTsProgram() any {
	return p.tsProgram
}

func (p *NgtscProgram) GetReuseTsProgram() any {
	return p.reuseTsProgram
}

func (p *NgtscProgram) GetTsOptionDiagnostics(cancellationToken any) []any {
	return nil
}

func (p *NgtscProgram) GetTsSyntacticDiagnostics(sourceFile any, cancellationToken any) []any {
	// ...
	return nil
}

func (p *NgtscProgram) GetTsSemanticDiagnostics(sourceFile any, cancellationToken any) []any {
	// ...
	return nil
}

func (p *NgtscProgram) GetNgOptionDiagnostics(cancellationToken any) []any {
	return nil
}

func (p *NgtscProgram) GetNgStructuralDiagnostics(cancellationToken any) []any {
	return nil
}

func (p *NgtscProgram) GetNgSemanticDiagnostics(fileName string, cancellationToken any) []any {
	return nil
}

func (p *NgtscProgram) LoadNgStructureAsync() error {
	return p.compiler.AnalyzeAsync()
}

func (p *NgtscProgram) ListLazyRoutes(entryRoute string) []any {
	return nil
}

func (p *NgtscProgram) EmitXi18n() {
	// p.compiler.xi18n(ctx)
}

func (p *NgtscProgram) Emit(opts any) any {
	// p.compiler.perfRecorder.inPhase(...)
	return nil
}

func (p *NgtscProgram) GetIndexedComponents() map[any]any {
	return nil // p.compiler.getIndexedComponents()
}

func (p *NgtscProgram) GetApiDocumentation(entryPoint string, privateModules map[string]struct{}) any {
	return nil // p.compiler.getApiDocumentation(...)
}

func (p *NgtscProgram) GetEmittedSourceFiles() map[string]any {
	panic("Method not implemented.")
}
