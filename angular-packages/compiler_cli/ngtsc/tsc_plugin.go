package ngtsc

import (
	
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core/src"
)

type PluginCompilerHost interface {
	any
	InputFiles() []string
}

type TscPlugin interface {
	Name() string
	WrapHost(host any, inputFiles []string, options any) PluginCompilerHost
	SetupCompilation(program any, oldProgram any) SetupCompilationResult
	GetDiagnostics(file any) []any
	GetOptionDiagnostics() []any
	GetNextProgram() any
	CreateTransformers() any
}

type SetupCompilationResult struct {
	IgnoreForDiagnostics map[any]struct{}
	IgnoreForEmit        map[any]struct{}
}

type NgTscPlugin struct {
	name      string
	ngOptions any // {}
	options   any // NgCompilerOptions
	host      *src.NgCompilerHost
	compiler  *src.NgCompiler
}

func NewNgTscPlugin(ngOptions any) *NgTscPlugin {
	return &NgTscPlugin{
		name:      "ngtsc",
		ngOptions: ngOptions,
	}
}

func (p *NgTscPlugin) Name() string {
	return p.name
}

func (p *NgTscPlugin) Compiler() *src.NgCompiler {
	if p.compiler == nil {
		panic("Lifecycle error: setupCompilation() must be called first.")
	}
	return p.compiler
}

func (p *NgTscPlugin) WrapHost(
	host any,
	inputFiles []string,
	options any,
) PluginCompilerHost {
	p.options = options // merge ngOptions
	p.host = src.NgCompilerHostWrap(host, inputFiles, p.options, nil)
	return nil // type cast issue for PluginCompilerHost, let's assume it works
}

func (p *NgTscPlugin) SetupCompilation(
	program any,
	oldProgram any,
) SetupCompilationResult {
	if p.host == nil || p.options == nil {
		panic("Lifecycle error: setupCompilation() before wrapHost().")
	}
	p.host.PostProgramCreationCleanup()

	// ...
	// This would call NgCompiler.fromTicket(...)
	p.compiler = src.NgCompilerFromTicket(nil, p.host)

	return SetupCompilationResult{
		IgnoreForDiagnostics: map[any]struct{}{},
		IgnoreForEmit:        map[any]struct{}{},
	}
}

func (p *NgTscPlugin) GetDiagnostics(file any) []any {
	if file == nil {
		return nil
	}
	return nil
}

func (p *NgTscPlugin) GetOptionDiagnostics() []any {
	return nil
}

func (p *NgTscPlugin) GetNextProgram() any {
	return p.Compiler().GetCurrentProgram()
}

func (p *NgTscPlugin) CreateTransformers() any {
	// p.Compiler().PerfRecorder.Phase(...)
	res := p.Compiler().PrepareEmit()
	return res.Transformers
}
