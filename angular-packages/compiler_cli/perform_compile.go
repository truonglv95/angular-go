package compiler_cli

import (
	"context"
	"io"
	"os"
	"sort"
	"time"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/astnav"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/locale"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
	"github.com/microsoft/typescript-go/internal/vfs/osvfs"
)

type ParsedConfiguration struct {
	Project   string
	Options   *core.CompilerOptions
	RootNames []string
	Errors    []*ast.Diagnostic
	Locale    locale.Locale
}

type osSys struct {
	writer             io.Writer
	fs                 vfs.FS
	defaultLibraryPath string
	cwd                string
	start              time.Time
}

func (s *osSys) FS() vfs.FS                                 { return s.fs }
func (s *osSys) DefaultLibraryPath() string                 { return s.defaultLibraryPath }
func (s *osSys) GetCurrentDirectory() string                 { return s.cwd }
func (s *osSys) Writer() io.Writer                          { return s.writer }
func (s *osSys) WriteOutputIsTTY() bool                     { return false }
func (s *osSys) GetWidthOfTerminal() int                    { return 80 }
func (s *osSys) GetEnvironmentVariable(name string) string { return os.Getenv(name) }
func (s *osSys) Now() time.Time                            { return time.Now() }
func (s *osSys) SinceStart() time.Duration                  { return time.Since(s.start) }

func newSystem() *osSys {
	cwd, _ := os.Getwd()
	return &osSys{
		cwd:                tspath.NormalizePath(cwd),
		fs:                 bundled.WrapFS(osvfs.FS()),
		defaultLibraryPath: bundled.LibPath(),
		writer:             os.Stdout,
		start:              time.Now(),
	}
}

func ReadConfiguration(project string) *ParsedConfiguration {
	sys := newSystem()
	resolvedProject := tspath.NormalizePath(project)
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
	sys := newSystem()

	if len(config.Errors) > 0 {
		return &PerformCompilationResult{
			Diagnostics: config.Errors,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}
	}

	host := compiler.NewCachedFSCompilerHost(
		sys.GetCurrentDirectory(),
		sys.FS(),
		sys.DefaultLibraryPath(),
		&tsc.ExtendedConfigCache{},
		nil,
	)

	// Build a standard parsed command line for the program creation
	parsedConfig := &core.ParsedOptions{
		FileNames:       config.RootNames,
		CompilerOptions: config.Options,
	}
	parsedCommandLine := &tsoptions.ParsedCommandLine{
		ParsedConfig: parsedConfig,
	}

	program := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsedCommandLine,
		Host:   host,
	})

	// Run Custom Angular AST transformations on all source files before emission
	ctx := context.Background()
	chk, release := program.GetTypeChecker(ctx)
	refHost := reflection.NewTypeScriptReflectionHost(chk)

	for _, sf := range program.SourceFiles() {
		transform.TransformSourceFile(sf, refHost)
	}

	// Release the type checker back to the pool
	release()

	// Perform compiler emission with the modified AST directly
	emitResult := program.Emit(ctx, compiler.EmitOptions{
		WriteFile: func(fileName string, text string, data *compiler.WriteFileData) error {
			// Parse the emitted JS into a temporary AST using the typescript-go parser
			sf := parser.ParseSourceFile(ast.SourceFileParseOptions{
				FileName: fileName,
			}, text, core.ScriptKindJS)
			if sf != nil {
				var pureOffsets []int
				var findPureCalls func(node *ast.Node)
				findPureCalls = func(node *ast.Node) {
					if node == nil {
						return
					}
					if node.Kind == ast.KindCallExpression {
						call := node.AsCallExpression()
						if call.Expression.Kind == ast.KindPropertyAccessExpression {
							pa := call.Expression.AsPropertyAccessExpression()
							if pa.Name().Kind == ast.KindIdentifier {
								name := pa.Name().AsIdentifier().Text
								if name == "ɵɵdefineComponent" || name == "ɵɵdefineDirective" || name == "ɵɵdefineNgModule" || name == "ɵɵdefinePipe" || name == "ɵɵdefineInjectable" {
									pureOffsets = append(pureOffsets, astnav.GetStartOfNode(node, sf, false))
								}
							}
						}
					}
					node.ForEachChild(func(child *ast.Node) bool {
						findPureCalls(child)
						return false
					})
				}
				findPureCalls(sf.AsNode())

				// Insert comments at the exact AST-determined positions in reverse order
				if len(pureOffsets) > 0 {
					sort.Slice(pureOffsets, func(i, j int) bool {
						return pureOffsets[i] > pureOffsets[j]
					})
					for _, offset := range pureOffsets {
						text = text[:offset] + "/*@__PURE__*/ " + text[offset:]
					}
				}
			}

			return host.FS().WriteFile(fileName, text)
		},
	})

	var diags []*ast.Diagnostic
	diags = append(diags, program.GetConfigFileParsingDiagnostics()...)
	diags = append(diags, emitResult.Diagnostics...)

	status := tsc.ExitStatusSuccess
	if emitResult.EmitSkipped {
		status = tsc.ExitStatusDiagnosticsPresent_OutputsSkipped
	}

	return &PerformCompilationResult{
		Diagnostics: diags,
		Status:      status,
	}
}
