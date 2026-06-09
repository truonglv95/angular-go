package compiler_cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/emit"
	_ "github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ingest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc"
	ngtsc_core "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/perf"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/locale"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
	"github.com/microsoft/typescript-go/internal/vfs/cachedvfs"
	"github.com/microsoft/typescript-go/internal/vfs/osvfs"
)

type ParsedConfiguration struct {
	Project           string
	Options           *core.CompilerOptions
	RootNames         []string
	Errors            []*ast.Diagnostic
	Locale            locale.Locale
	CompilationMode   string
	EnableHmr         bool
	DiscardOutput     bool
	Write             bool
	Format            string
	StrictTemplates   bool
	StyleIncludePaths []string
}

func EnablePreserveImports(config *ParsedConfiguration) {
	if config != nil && config.Options != nil {
		config.Options.VerbatimModuleSyntax = core.TSTrue
	}
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

func stripComments(data []byte) []byte {
	var clean []byte
	inLineComment := false
	inBlockComment := false
	inString := false
	var stringChar byte

	for i := 0; i < len(data); i++ {
		b := data[i]

		if inLineComment {
			if b == '\n' || b == '\r' {
				inLineComment = false
				clean = append(clean, b)
			}
			continue
		}

		if inBlockComment {
			if b == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				i++ // skip '/'
			}
			continue
		}

		if inString {
			clean = append(clean, b)
			if b == '\\' && i+1 < len(data) {
				clean = append(clean, data[i+1])
				i++
			} else if b == stringChar {
				inString = false
			}
			continue
		}

		// Check for comment starts or string starts
		if b == '"' || b == '\'' || b == '`' {
			inString = true
			stringChar = b
			clean = append(clean, b)
			continue
		}

		if b == '/' && i+1 < len(data) {
			if data[i+1] == '/' {
				inLineComment = true
				i++ // skip second '/'
				continue
			} else if data[i+1] == '*' {
				inBlockComment = true
				i++ // skip '*'
				continue
			}
		}

		clean = append(clean, b)
	}

	return clean
}

func parseStrictTemplates(tsconfigPath string) bool {
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return false
	}

	cleanData := stripComments(data)

	var parsed struct {
		Extends                string `json:"extends"`
		AngularCompilerOptions struct {
			StrictTemplates *bool `json:"strictTemplates"`
		} `json:"angularCompilerOptions"`
	}
	if err := json.Unmarshal(cleanData, &parsed); err != nil {
		return false
	}

	if parsed.AngularCompilerOptions.StrictTemplates != nil {
		return *parsed.AngularCompilerOptions.StrictTemplates
	}

	if parsed.Extends != "" {
		parentPath := filepath.Join(filepath.Dir(tsconfigPath), parsed.Extends)
		if !strings.HasSuffix(parentPath, ".json") {
			parentPath += ".json"
		}
		return parseStrictTemplates(parentPath)
	}

	return false
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
	parsed.StrictTemplates = parseStrictTemplates(resolvedProject)
	return parsed
}

type OutputFile struct {
	Path string `json:"path"`
	Text string `json:"text,omitempty"`
	Hash string `json:"hash,omitempty"`
	Kind string `json:"kind"`
}

type PerformCompilationResult struct {
	Diagnostics []*ast.Diagnostic
	Status      tsc.ExitStatus
	Outputs     []OutputFile
	PerfPhases  map[string]int64 `json:"perfPhases,omitempty"`
}

func newCompilerInternalErrorDiagnostic(recovered any) *ast.Diagnostic {
	return ast.NewCompilerDiagnostic(
		diagnostics.Could_not_write_file_0_Colon_1,
		"Angular compiler",
		fmt.Sprintf("internal error: %v\nStack: %s", recovered, string(debug.Stack())),
	)
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
	res, _ := PerformCompilationWithHost(config, host, nil, nil)
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

func PerformCompilationWithHost(config *ParsedConfiguration, host compiler.CompilerHost, oldProgram *ngtsc.NgtscProgram, invalidatedFiles map[string]bool) (result *PerformCompilationResult, ngProgramResult *ngtsc.NgtscProgram) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = &PerformCompilationResult{
				Diagnostics: []*ast.Diagnostic{newCompilerInternalErrorDiagnostic(recovered)},
				Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
			}
			ngProgramResult = nil
		}
	}()

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
				CompilationMode:   config.CompilationMode,
				EnableHmr:         config.EnableHmr,
				InvalidatedFiles:  invalidatedFiles,
				StrictTemplates:   config.StrictTemplates,
				StyleIncludePaths: config.StyleIncludePaths,
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
		var rawDiags []*ast.Diagnostic
		if len(invalidatedFiles) > 0 {
			for file := range invalidatedFiles {
				absPath := tspath.GetNormalizedAbsolutePath(file, host.GetCurrentDirectory())
				canonicalPath := tspath.GetCanonicalFileName(absPath, host.FS().UseCaseSensitiveFileNames())
				sf := ngProgram.GetTsProgram().GetSourceFileByPath(tspath.Path(canonicalPath))
				if sf != nil {
					rawDiags = append(rawDiags, ngProgram.GetTsProgram().GetSyntacticDiagnostics(ctx, sf)...)
					rawDiags = append(rawDiags, ngProgram.GetTsProgram().GetBindDiagnostics(ctx, sf)...)
					rawDiags = append(rawDiags, ngProgram.GetTsProgram().GetSemanticDiagnostics(ctx, sf)...)
				}
			}
		} else {
			rawDiags = append(rawDiags, ngProgram.GetTsProgram().GetSyntacticDiagnostics(ctx, nil)...)
			rawDiags = append(rawDiags, ngProgram.GetTsProgram().GetBindDiagnostics(ctx, nil)...)
			rawDiags = append(rawDiags, ngProgram.GetTsProgram().GetSemanticDiagnostics(ctx, nil)...)
		}
		for _, d := range rawDiags {
			if d.Code() != 1484 {
				diags = append(diags, d)
			}
		}
	}()

	diags = filterDiagnostics(diags)
	if len(diags) > 0 {
		return &PerformCompilationResult{
			Diagnostics: diags,
			Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
		}, ngProgram
	}

	var emitResult *compiler.EmitResult
	var outputs []OutputFile
	var outputsMutex sync.Mutex

	type PendingFile struct {
		FileName string
		Text     string
	}
	var pendingFiles []PendingFile
	var pendingFilesMutex sync.Mutex

	affected := ngProgram.GetAffectedFiles()
	if oldProgram != nil {
		// Warm rebuild
		var skippedEmit bool = false
		if len(invalidatedFiles) > 0 && affected != nil {
			for file := range affected {
				absPath := tspath.GetNormalizedAbsolutePath(file, host.GetCurrentDirectory())
				canonicalPath := tspath.GetCanonicalFileName(absPath, host.FS().UseCaseSensitiveFileNames())
				sf := ngProgram.GetTsProgram().GetSourceFileByPath(tspath.Path(canonicalPath))
				if sf == nil {
					continue
				}
				var singleEmitResult *compiler.EmitResult
				func() {
					singleEmitResult = ngProgram.Emit(ctx, compiler.EmitOptions{
						TargetSourceFile: sf,
						EmitOnly:         compiler.EmitOnlyJs,
						WriteFile: func(fileName string, text string, data *compiler.WriteFileData) error {
							pendingFilesMutex.Lock()
							pendingFiles = append(pendingFiles, PendingFile{FileName: fileName, Text: text})
							pendingFilesMutex.Unlock()
							return nil
						},
					})
				}()
				if singleEmitResult != nil {
					diags = append(diags, singleEmitResult.Diagnostics...)
					if singleEmitResult.EmitSkipped {
						skippedEmit = true
					}
				}
			}
		}
		emitResult = &compiler.EmitResult{EmitSkipped: skippedEmit}
	} else {
		// Cold start: emit everything
		func() {
			emitResult = ngProgram.Emit(ctx, compiler.EmitOptions{
				WriteFile: func(fileName string, text string, data *compiler.WriteFileData) error {
					pendingFilesMutex.Lock()
					pendingFiles = append(pendingFiles, PendingFile{FileName: fileName, Text: text})
					pendingFilesMutex.Unlock()
					return nil
				},
			})
		}()
		if emitResult != nil {
			diags = append(diags, emitResult.Diagnostics...)
		}
	}

	if !emitResult.EmitSkipped && len(diags) == 0 {
		// Run Linker in parallel
		rec := ngProgram.PerfRecorder()
		var prevPhase perf.PerfPhase
		if rec != nil {
			prevPhase = rec.Phase(perf.PerfPhase_Linker)
		}

		var wg sync.WaitGroup
		errs := make(chan error, len(pendingFiles))
		outputs = make([]OutputFile, len(pendingFiles))

		numWorkers := runtime.NumCPU()
		if numWorkers > 16 {
			numWorkers = 16
		}
		if numWorkers < 1 {
			numWorkers = 1
		}
		workerSem := make(chan struct{}, numWorkers)

		for i, f := range pendingFiles {
			wg.Add(1)
			go func(idx int, file PendingFile) {
				defer wg.Done()
				defer func() {
					if recovered := recover(); recovered != nil {
						errs <- fmt.Errorf("Angular linker internal error in %s: %v", file.FileName, recovered)
					}
				}()
				workerSem <- struct{}{}
				defer func() { <-workerSem }()

				var text = file.Text
				var linkErr error
				var linkedText string
				var changed bool

				linkedText, changed, linkErr = linkPartialDeclarationsInEmittedJavaScript(file.FileName, text)
				if linkErr != nil {
					errs <- linkErr
					return
				}
				if changed {
					text = linkedText
				}

				if config.Write {
					writeErr := host.FS().WriteFile(file.FileName, text)
					if writeErr != nil {
						errs <- writeErr
						return
					}
				}

				kind := "other"
				if strings.HasSuffix(file.FileName, ".js") {
					kind = "js"
				} else if strings.HasSuffix(file.FileName, ".d.ts") {
					kind = "dts"
				} else if strings.HasSuffix(file.FileName, ".map") {
					kind = "map"
				}

				hash := sha256.Sum256([]byte(text))
				hashStr := fmt.Sprintf("%x", hash)

				outputsMutex.Lock()
				outputs[idx] = OutputFile{
					Path: file.FileName,
					Text: text,
					Hash: hashStr,
					Kind: kind,
				}
				outputsMutex.Unlock()
			}(i, f)
		}

		wg.Wait()
		close(errs)

		if rec != nil {
			rec.Phase(prevPhase)
		}

		for err := range errs {
			if err != nil {
				diags = append(diags, ast.NewCompilerDiagnostic(diagnostics.Could_not_write_file_0_Colon_1, "", err.Error()))
			}
		}
	}

	diags = filterDiagnostics(diags)
	status := tsc.ExitStatusSuccess
	if emitResult.EmitSkipped || len(diags) > 0 {
		status = tsc.ExitStatusDiagnosticsPresent_OutputsSkipped
	}

	return &PerformCompilationResult{
		Diagnostics: diags,
		Status:      status,
		Outputs:     outputs,
		PerfPhases:  ngProgram.GetPerfResults(),
	}, ngProgram
}

var linkerCache sync.Map // Thread-safe global cache mapping unlinked text hash to linked text string.

func linkPartialDeclarationsInEmittedJavaScript(fileName string, text string) (string, bool, error) {
	if !isJavaScriptOutput(fileName) {
		return text, false, nil
	}

	absFileName, err := filepath.Abs(fileName)
	if err != nil {
		absFileName = fileName
	}

	// Check cache by input unlinked text hash to avoid redundant parses
	hash := sha256.Sum256([]byte(text))
	key := hex.EncodeToString(hash[:])
	if val, ok := linkerCache.Load(key); ok {
		cachedStr := val.(string)
		return cachedStr, cachedStr != text, nil
	}

	linkedText, changed, linkErr := LinkJavaScriptText(absFileName, text, core.ScriptKindJS)
	if linkErr != nil {
		return text, false, linkErr
	}

	if changed {
		linkerCache.Store(key, linkedText)
	} else {
		linkerCache.Store(key, text)
	}

	return linkedText, changed, nil
}

func isJavaScriptOutput(fileName string) bool {
	switch filepath.Ext(fileName) {
	case ".js", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func filterDiagnostics(diags []*ast.Diagnostic) []*ast.Diagnostic {
	var filtered []*ast.Diagnostic
	for _, d := range diags {
		if d.Code() != 1484 {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
