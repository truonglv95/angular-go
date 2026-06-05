package compiler_cli

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/emit"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ingest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc"
	ngtsc_core "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/locale"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
	"github.com/microsoft/typescript-go/internal/vfs/cachedvfs"
	"github.com/microsoft/typescript-go/internal/vfs/osvfs"
)

type ParsedConfiguration struct {
	Project         string
	Options         *core.CompilerOptions
	RootNames       []string
	Errors          []*ast.Diagnostic
	Locale          locale.Locale
	CompilationMode string
	EnableHmr       bool
	DiscardOutput   bool
	Write           bool
	Format          string
}

type osSys struct {
	writer             io.Writer
	fs                 vfs.FS
	defaultLibraryPath string
	cwd                string
	start              time.Time
}

type profiledCompilerHost struct {
	compiler.CompilerHost
}

func (h *profiledCompilerHost) ClearSourceCache() {
	if cc, ok := h.CompilerHost.(CacheClearable); ok {
		cc.ClearSourceCache()
	}
}

// H2 FIX: Delegate ClearSourceFile so that the type assertion
// state.Host.(CacheClearable) in server.go succeeds. Without this method,
// profiledCompilerHost does not satisfy CacheClearable even though the
// embedded cachedCompilerHost does, causing per-file invalidation to silently
// fall back to a full cache clear on every change.
func (h *profiledCompilerHost) ClearSourceFile(filePath string) {
	if cc, ok := h.CompilerHost.(CacheClearable); ok {
		cc.ClearSourceFile(filePath)
	}
}

func (h *profiledCompilerHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	return h.CompilerHost.GetSourceFile(opts)
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
	parsed.Options.SingleThreaded = core.BoolToTristate(false)
	parsed.RootNames = configParseResult.FileNames()
	parsed.Locale = configParseResult.Locale()
	return parsed
}

type OutputFile struct {
	Path string `json:"path"`
	Text string `json:"text"`
	Hash string `json:"hash,omitempty"`
	Kind string `json:"kind"`
}

type PerformCompilationResult struct {
	Diagnostics []*ast.Diagnostic
	Status      tsc.ExitStatus
	Outputs     []OutputFile
}

func PerformCompilation(config *ParsedConfiguration) *PerformCompilationResult {
	sys := newSystem(config.DiscardOutput)

	if len(config.Errors) > 0 {
		return &PerformCompilationResult{
			Diagnostics: config.Errors,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}
	}

	host := CreateCompilerHost(sys)
	res, _ := PerformCompilationWithHost(config, host, nil)
	return res
}

type cachedSourceFile struct {
	file  *ast.SourceFile
	mtime time.Time
	size  int64
}

type cachedCompilerHost struct {
	compiler.CompilerHost
	mu              sync.RWMutex
	sourceFileCache map[string]*cachedSourceFile
}

func (h *cachedCompilerHost) ClearSourceCache() {
	h.mu.Lock()
	h.sourceFileCache = make(map[string]*cachedSourceFile)
	h.mu.Unlock()
}

// B#6 FIX: ClearSourceFile evicts a single file from the source cache instead
// of nuking the entire cache on every file-change event.
func (h *cachedCompilerHost) ClearSourceFile(filePath string) {
	h.mu.Lock()
	delete(h.sourceFileCache, filePath)
	h.mu.Unlock()
}

type CacheClearable interface {
	ClearSourceCache()
	// ClearSourceFile evicts a single file; implementors that only support
	// full-clear may implement this as a no-op or fall back to ClearSourceCache.
	ClearSourceFile(filePath string)
}

func (h *cachedCompilerHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	h.mu.RLock()
	cached, ok := h.sourceFileCache[opts.FileName]
	h.mu.RUnlock()

	var mtime time.Time
	var size int64
	if stat := h.FS().Stat(opts.FileName); stat != nil {
		mtime = stat.ModTime()
		size = stat.Size()
	}

	if ok && cached.mtime == mtime && cached.size == size && cached.file != nil {
		return cached.file
	}

	file := h.CompilerHost.GetSourceFile(opts)

	h.mu.Lock()
	if h.sourceFileCache == nil {
		h.sourceFileCache = make(map[string]*cachedSourceFile)
	}
	h.sourceFileCache[opts.FileName] = &cachedSourceFile{
		file:  file,
		mtime: mtime,
		size:  size,
	}
	h.mu.Unlock()

	return file
}

func CreateCompilerHost(sys *osSys) compiler.CompilerHost {
	host := compiler.NewCompilerHost(
		sys.GetCurrentDirectory(),
		cachedvfs.From(sys.FS()),
		sys.DefaultLibraryPath(),
		&tsc.ExtendedConfigCache{},
		nil,
	)
	cachedHost := &cachedCompilerHost{
		CompilerHost:    host,
		sourceFileCache: make(map[string]*cachedSourceFile),
	}
	return &profiledCompilerHost{CompilerHost: cachedHost}
}

func PerformCompilationWithHost(config *ParsedConfiguration, host compiler.CompilerHost, oldProgram *ngtsc.NgtscProgram) (*PerformCompilationResult, *ngtsc.NgtscProgram) {
	if len(config.Errors) > 0 {
		return &PerformCompilationResult{
			Diagnostics: config.Errors,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}, nil
	}

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
		ngProgram, err = ngtsc.NewNgtscProgram(
			config.RootNames,
			parsedCommandLine,
			host,
			ngtsc_core.NgCompilerOptions{
				CompilationMode: config.CompilationMode,
				EnableHmr:       config.EnableHmr,
			},
			oldProgram,
		)
	}()
	if err != nil {
		return &PerformCompilationResult{Diagnostics: nil}, nil
	}
	var diags []*ast.Diagnostic
	func() {
		ngProgram.LoadNgStructureAsync(ctx)
	}()
	diags = append(diags, ngProgram.GetNgDiagnostics()...)

	func() {
		diags = append(diags, ngProgram.GetTsProgram().GetConfigFileParsingDiagnostics()...)
		diags = append(diags, ngProgram.GetTsProgram().GetSyntacticDiagnostics(nil, nil)...)
	}()

	if len(diags) > 0 {
		return &PerformCompilationResult{
			Diagnostics: diags,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}, ngProgram
	}

	var emitResult *compiler.EmitResult
	var outputs []OutputFile
	var outputsMutex sync.Mutex

	func() {

		// Throttle concurrent file writes to avoid disk thrashing
		writeSemaphore := make(chan struct{}, 16)

		emitResult = ngProgram.Emit(ctx, compiler.EmitOptions{
			WriteFile: func(fileName string, text string, data *compiler.WriteFileData) error {
				writeSemaphore <- struct{}{}
				defer func() { <-writeSemaphore }()

				var linkErr error
				func() {
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

				kind := "other"
				if strings.HasSuffix(fileName, ".js") {
					kind = "js"
				} else if strings.HasSuffix(fileName, ".d.ts") {
					kind = "dts"
				} else if strings.HasSuffix(fileName, ".map") {
					kind = "map"
				}

				hash := sha256.Sum256([]byte(text))
				hashStr := fmt.Sprintf("%x", hash)

				outputsMutex.Lock()
				outputs = append(outputs, OutputFile{
					Path: fileName,
					Text: text,
					Hash: hashStr,
					Kind: kind,
				})
				outputsMutex.Unlock()

				if config.Write {
					return host.FS().WriteFile(fileName, text)
				}
				return nil
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
		Outputs:     outputs,
	}, ngProgram
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
