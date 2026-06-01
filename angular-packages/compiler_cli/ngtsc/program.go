package ngtsc

import (
	"context"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/tsoptions"
)

type NgtscProgram struct {
	compiler  *core.NgCompiler
	tsProgram *compiler.Program
}

func NewNgtscProgram(
	rootNames []string,
	tsConfig *tsoptions.ParsedCommandLine,
	delegateHost compiler.CompilerHost,
) (*NgtscProgram, error) {
	// Create the TypeScript program inside NgtscProgram
	tsProgram := compiler.NewProgram(compiler.ProgramOptions{
		Config: tsConfig,
		Host:   delegateHost,
	})

	ngCompiler, err := core.NewNgCompiler(tsProgram)
	if err != nil {
		return nil, err
	}

	return &NgtscProgram{
		compiler:  ngCompiler,
		tsProgram: tsProgram,
	}, nil
}

func (p *NgtscProgram) GetTsProgram() *compiler.Program {
	return p.tsProgram
}

func (p *NgtscProgram) LoadNgStructureAsync(ctx context.Context) error {
	p.compiler.AnalyzeSync()
	p.compiler.Resolve()
	return nil
}

func (p *NgtscProgram) GetNgDiagnostics() []*ast.Diagnostic {
	var diags []*ast.Diagnostic
	diags = append(diags, p.compiler.AnalyzeSync()...)
	diags = append(diags, p.compiler.Resolve()...)
	return diags
}

func (p *NgtscProgram) Emit(ctx context.Context, opts compiler.EmitOptions) *compiler.EmitResult {
	p.compiler.PrepareEmit()
	return p.tsProgram.Emit(ctx, opts)
}
