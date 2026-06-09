package ngtsc

import (
	"context"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/perf"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	tscCore "github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
)

type NgtscProgram struct {
	compiler  *core.NgCompiler
	tsProgram *compiler.Program
}

func NewNgtscProgram(
	rootNames []string,
	tsConfig *tsoptions.ParsedCommandLine,
	delegateHost compiler.CompilerHost,
	options core.NgCompilerOptions,
	oldProgram *NgtscProgram,
) (*NgtscProgram, error) {
	var rec *perf.ActivePerfRecorder
	if oldProgram != nil && oldProgram.compiler != nil && oldProgram.compiler.PerfRecorder() != nil {
		rec = oldProgram.compiler.PerfRecorder()
		rec.Reset()
	} else {
		rec = perf.ActivePerfRecorderZeroedToNow()
	}

	rec.Phase(perf.PerfPhase_TypeScriptProgramCreate)
	
	if len(options.InvalidatedFiles) > 0 {
		canonicalInvalidated := make(map[string]bool)
		for file, val := range options.InvalidatedFiles {
			absPath := tspath.GetNormalizedAbsolutePath(file, delegateHost.GetCurrentDirectory())
			canonicalPath := tspath.GetCanonicalFileName(absPath, delegateHost.FS().UseCaseSensitiveFileNames())
			canonicalInvalidated[canonicalPath] = val
		}
		options.InvalidatedFiles = canonicalInvalidated
	}

	var tsProgram *compiler.Program
	if oldProgram != nil && oldProgram.tsProgram != nil && len(options.InvalidatedFiles) > 0 {
		tsProgram = oldProgram.tsProgram
		var ok bool
		for file := range options.InvalidatedFiles {
			absPath := tspath.GetNormalizedAbsolutePath(file, delegateHost.GetCurrentDirectory())
			canonicalPath := tspath.GetCanonicalFileName(absPath, delegateHost.FS().UseCaseSensitiveFileNames())
			if tsProgram.GetSourceFileByPath(tspath.Path(canonicalPath)) == nil {
				continue
			}
			tsProgram, ok = tsProgram.UpdateProgram(tspath.Path(canonicalPath), delegateHost, nil)
			if !ok {
				tsProgram = nil
				break
			}
		}
	}

	if tsProgram == nil {
		// Create the TypeScript program inside NgtscProgram
		tsProgram = compiler.NewProgram(compiler.ProgramOptions{
			Config:         tsConfig,
			Host:           delegateHost,
			SingleThreaded: tscCore.TSFalse,
		})
	}

	rec.Phase(perf.PerfPhase_Setup)
	var oldCompiler *core.NgCompiler
	if oldProgram != nil {
		oldCompiler = oldProgram.compiler
	}
	ngCompiler, err := core.NewNgCompiler(tsProgram, options, oldCompiler)
	if err != nil {
		return nil, err
	}
	ngCompiler.SetPerfRecorder(rec)

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
	rec := p.compiler.PerfRecorder()
	if rec != nil {
		rec.Phase(perf.PerfPhase_TypeScriptEmit)
		defer rec.Phase(perf.PerfPhase_Unaccounted)
	}
	return p.tsProgram.Emit(ctx, opts)
}

func (p *NgtscProgram) GetAffectedFiles() map[string]bool {
	if p.compiler != nil {
		return p.compiler.AffectedFiles()
	}
	return nil
}

func (p *NgtscProgram) GetHmrUpdate(componentId string) string {
	if p.compiler != nil {
		return p.compiler.GetHmrUpdate(componentId)
	}
	return ""
}

func (p *NgtscProgram) GetHmrComponentIds() []string {
	if p.compiler != nil {
		return p.compiler.GetHmrComponentIds()
	}
	return nil
}

func (p *NgtscProgram) GetPerfResults() map[string]int64 {
	if p.compiler != nil {
		return p.compiler.GetPerfResults()
	}
	return nil
}

func (p *NgtscProgram) PerfRecorder() *perf.ActivePerfRecorder {
	if p.compiler != nil {
		return p.compiler.PerfRecorder()
	}
	return nil
}

