package compiler_cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/internal/perf"

	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/emit"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ingest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/locale"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
	"github.com/microsoft/typescript-go/internal/vfs/osvfs"
)

type ParsedConfiguration struct {
	Project         string
	Options         *core.CompilerOptions
	RootNames       []string
	Errors          []*ast.Diagnostic
	Locale          locale.Locale
	CompilationMode string
	DiscardOutput   bool
}

type osSys struct {
	writer             io.Writer
	fs                 vfs.FS
	defaultLibraryPath string
	cwd                string
	start              time.Time
}

func (s *osSys) FS() vfs.FS                                { return s.fs }
func (s *osSys) DefaultLibraryPath() string                { return s.defaultLibraryPath }
func (s *osSys) GetCurrentDirectory() string               { return s.cwd }
func (s *osSys) Writer() io.Writer                         { return s.writer }
func (s *osSys) WriteOutputIsTTY() bool                    { return false }
func (s *osSys) GetWidthOfTerminal() int                   { return 80 }
func (s *osSys) GetEnvironmentVariable(name string) string { return os.Getenv(name) }
func (s *osSys) Now() time.Time                            { return time.Now() }
func (s *osSys) SinceStart() time.Duration                 { return time.Since(s.start) }

func newSystem(discardOutput bool) *osSys {
	cwd, _ := os.Getwd()
	fsys := bundled.WrapFS(osvfs.FS())
	fsys = &optimizedFS{fsys}
	if discardOutput {
		fsys = &discardFS{fsys}
	}
	return &osSys{
		cwd:                tspath.NormalizePath(cwd),
		fs:                 fsys,
		defaultLibraryPath: bundled.LibPath(),
		writer:             os.Stdout,
		start:              time.Now(),
	}
}

type optimizedFS struct {
	vfs.FS
}

func (o *optimizedFS) WriteFile(path string, data string) error {
	if existing, ok := o.FS.ReadFile(path); ok && existing == data {
		return nil
	}
	defer perf.Time("emit.fs_write")()
	return o.FS.WriteFile(path, data)
}

type discardFS struct {
	vfs.FS
}

func (d *discardFS) WriteFile(path string, data string) error {
	return nil
}

func (d *discardFS) AppendFile(path string, data string) error {
	return nil
}

func ReadConfiguration(project string) *ParsedConfiguration {
	defer perf.Time("config.read")()
	sys := newSystem(false)
	resolvedProject := tspath.CombinePaths(sys.GetCurrentDirectory(), project)
	if sys.FS().DirectoryExists(resolvedProject) {
		resolvedProject = tspath.CombinePaths(resolvedProject, "tsconfig.json")
	}

	extendedConfigCache := &tsc.ExtendedConfigCache{}
	configParseResult, errors := tsoptions.GetParsedCommandLineOfConfigFile(
		resolvedProject,
		&core.CompilerOptions{},
		nil,
		sys,
		extendedConfigCache,
	)

	parsed := &ParsedConfiguration{
		Project: resolvedProject,
	}

	if len(errors) > 0 {
		parsed.Errors = errors
		return parsed
	}

	parsed.Options = configParseResult.CompilerOptions()
	parsed.RootNames = configParseResult.FileNames()
	parsed.Locale = configParseResult.Locale()
	return parsed
}

type PerformCompilationResult struct {
	Diagnostics []*ast.Diagnostic
	Status      tsc.ExitStatus
}

func PerformCompilation(config *ParsedConfiguration) *PerformCompilationResult {
	defer perf.Time("compile.total")()
	sys := newSystem(config.DiscardOutput)

	if len(config.Errors) > 0 {
		return &PerformCompilationResult{
			Diagnostics: config.Errors,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}
	}

	var host compiler.CompilerHost
	func() {
		defer perf.Time("compile.host_create")()
		host = compiler.NewCachedFSCompilerHost(
			sys.GetCurrentDirectory(),
			sys.FS(),
			sys.DefaultLibraryPath(),
			&tsc.ExtendedConfigCache{},
			nil,
		)
	}()

	// Build a standard parsed command line for the program creation
	parsedConfig := &core.ParsedOptions{
		FileNames:       config.RootNames,
		CompilerOptions: config.Options,
	}
	parsedCommandLine := &tsoptions.ParsedCommandLine{
		ParsedConfig: parsedConfig,
	}

	// Run Custom Angular AST transformations on all source files before emission
	ctx := context.Background()

	var ngProgram *ngtsc.NgtscProgram
	var err error
	func() {
		defer perf.Time("compile.program_create")()
		ngProgram, err = ngtsc.NewNgtscProgram(
			config.RootNames,
			parsedCommandLine,
			host,
			config.CompilationMode,
		)
	}()
	if err != nil {
		return &PerformCompilationResult{Diagnostics: nil}
	}
	var diags []*ast.Diagnostic
	func() {
		defer perf.Time("angular.load_structure")()
		ngProgram.LoadNgStructureAsync(ctx)
	}()
	diags = append(diags, ngProgram.GetNgDiagnostics()...)

	func() {
		defer perf.Time("ts.diagnostics")()
		diags = append(diags, ngProgram.GetTsProgram().GetConfigFileParsingDiagnostics()...)
		diags = append(diags, ngProgram.GetTsProgram().GetSyntacticDiagnostics(nil, nil)...)
	}()

	if len(diags) > 0 {
		return &PerformCompilationResult{
			Diagnostics: diags,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}
	}

	var emitResult *compiler.EmitResult
	func() {
		defer perf.Time("ts.emit_total")()
		emitResult = ngProgram.Emit(ctx, compiler.EmitOptions{
			WriteFile: func(fileName string, text string, data *compiler.WriteFileData) error {
				defer perf.Time("emit.write_file")()
				var linkErr error
				func() {
					defer perf.Time("emit.linker")()
					var linkedText string
					var changed bool
					linkedText, changed, linkErr = linkPartialDeclarationsInEmittedJavaScript(fileName, text)
					if linkErr != nil {
						return
					}
					if changed {
						text = linkedText
					}
				}()
				if linkErr != nil {
					return linkErr
				}

				// Insert pure annotations using Regex instead of full AST parsing to save 395ms
				func() {
					defer perf.Time("emit.custom_transformers")()
					re := regexp.MustCompile(`(i\d+\.ɵɵdefine(?:Component|Directive|NgModule|Pipe|Injectable)\()`)
					text = re.ReplaceAllString(text, "/*@__PURE__*/ $1")
				}()

				timerName := "emit.write_text"
				if strings.HasSuffix(fileName, ".js") {
					timerName = "emit.write_js"
				} else if strings.HasSuffix(fileName, ".d.ts") {
					timerName = "emit.write_dts"
				} else if strings.HasSuffix(fileName, ".map") {
					timerName = "emit.source_map"
				}
				defer perf.Time(timerName)()

				return host.FS().WriteFile(fileName, text)
			},
		})
	}()

	diags = append(diags, emitResult.Diagnostics...)

	status := tsc.ExitStatusSuccess
	if emitResult.EmitSkipped || len(diags) > 0 {
		status = tsc.ExitStatusDiagnosticsPresent_OutputsSkipped
	}

	return &PerformCompilationResult{
		Diagnostics: diags,
		Status:      status,
	}
}

func linkPartialDeclarationsInEmittedJavaScript(fileName string, text string) (string, bool, error) {
	if !isJavaScriptOutput(fileName) {
		return text, false, nil
	}

	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		absFileName = fileName
	}
	return LinkJavaScriptText(absFileName, text, core.ScriptKindJS)
}

func isJavaScriptOutput(fileName string) bool {
	switch filepath.Ext(fileName) {
	case ".js", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}
