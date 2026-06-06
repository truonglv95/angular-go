package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations_local"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
)

type NgCompiler struct {
	tsProgram              *compiler.Program
	checker                *checker.Checker
	reflector              reflection.ReflectionHost
	handlers               []transform.DecoratorHandler
	traitCompiler          *transform.TraitCompiler
	incrementalCompilation *incremental.IncrementalCompilation
	affectedFiles          map[string]bool

	metaRegistry  *metadata.LocalMetadataRegistry
	scopeRegistry *scope.LocalModuleScopeRegistry

	compilationMode string
	options         NgCompilerOptions

	analyzed bool
	resolved bool
	prepared bool
}

func NewNgCompiler(tsProgram *compiler.Program, options NgCompilerOptions, oldCompiler *NgCompiler) (*NgCompiler, error) {
	ctx := context.Background()
	var chk *checker.Checker
	if options.CompilationMode != "local" {
		func() {
			chk, _ = tsProgram.GetTypeChecker(ctx)
		}()
	}
	refHost := reflection.NewTypeScriptReflectionHost(chk)

	localMetaRegistry := metadata.NewLocalMetadataRegistry()
	dtsMetaReader := metadata.NewDtsMetadataReader(chk)
	compoundMetaReader := metadata.NewCompoundMetadataReader([]metadata.MetadataReader{localMetaRegistry, dtsMetaReader})

	scopeRegistry := scope.NewLocalModuleScopeRegistry(compoundMetaReader)

	localRefHost := reflection.NewTypeScriptReflectionHost(nil)

	var handlers []transform.DecoratorHandler
	if options.CompilationMode == "local" {
		handlers = []transform.DecoratorHandler{
			annotations_local.NewComponentLocalDecoratorHandler(localRefHost, false, localMetaRegistry, scopeRegistry, options.EnableHmr),
			annotations_local.NewDirectiveLocalDecoratorHandler(localRefHost, localMetaRegistry),
			annotations_local.NewPipeLocalDecoratorHandler(localRefHost, localMetaRegistry), // LOCAL HANDLER
			annotations_local.NewInjectableLocalDecoratorHandler(localRefHost),
			annotations_local.NewNgModuleLocalDecoratorHandler(localRefHost, localMetaRegistry, scopeRegistry), // LOCAL HANDLER
		}
	} else {
		handlers = []transform.DecoratorHandler{
			annotations.NewComponentDecoratorHandler(refHost, false, localMetaRegistry, scopeRegistry, options.EnableHmr),
			annotations.NewDirectiveDecoratorHandler(refHost, localMetaRegistry),
			annotations.NewPipeDecoratorHandler(refHost, localMetaRegistry),
			annotations.NewInjectableDecoratorHandler(refHost), // GLOBAL HANDLER
			annotations.NewNgModuleDecoratorHandler(refHost, localMetaRegistry, scopeRegistry),
		}
	}

	var oldTc *transform.TraitCompiler
	var oldGraph *semantic_graph.SemanticDepGraph
	var oldIncrementalState *incremental.IncrementalState
	if oldCompiler != nil {
		oldTc = oldCompiler.traitCompiler
		if oldCompiler.incrementalCompilation != nil && oldCompiler.incrementalCompilation.State != nil {
			oldIncrementalState = oldCompiler.incrementalCompilation.State
			if sg, ok := oldIncrementalState.SemanticGraph.(*semantic_graph.SemanticDepGraph); ok {
				oldGraph = sg
			}
		}
	}
	traitCompiler := transform.NewTraitCompiler(handlers, refHost, localRefHost, oldTc)
	traitCompiler.SemanticUpdater = semantic_graph.NewSemanticDepGraphUpdater(oldGraph)

	newVersions := getFileVersions(tsProgram)
	var incrementalComp *incremental.IncrementalCompilation
	if oldIncrementalState != nil {
		incrementalComp = incremental.Incremental(
			tsProgram,
			newVersions,
			nil,
			oldIncrementalState,
			options.InvalidatedFiles,
			nil,
		)
	} else {
		incrementalComp = incremental.Fresh(newVersions)
	}
	traitCompiler.DepTracker = incrementalComp

	return &NgCompiler{
		tsProgram:              tsProgram,
		checker:                chk,
		reflector:              refHost,
		handlers:               handlers,
		traitCompiler:          traitCompiler,
		incrementalCompilation: incrementalComp,
		metaRegistry:           localMetaRegistry,
		scopeRegistry:          scopeRegistry,
		compilationMode:        options.CompilationMode,
		options:                options,
	}, nil
}

// AnalyzeSync scans all source files and runs decorator detection + analysis.
// B#1 FIX: Both global and local modes now run in parallel using a worker pool.
func (c *NgCompiler) AnalyzeSync() []*ast.Diagnostic {
	if c.analyzed {
		return nil
	}

	sourceFiles := c.tsProgram.SourceFiles()
	if len(sourceFiles) == 0 {
		c.analyzed = true
		return nil
	}

	// Feed files into a channel for workers to consume
	ch := make(chan *ast.SourceFile, len(sourceFiles))
	for _, sf := range sourceFiles {
		ch <- sf
	}
	close(ch)

	// Use min(NumCPU, 16, len(files)) workers to avoid excessive goroutine overhead
	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}
	if numWorkers > len(sourceFiles) {
		numWorkers = len(sourceFiles)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	if c.compilationMode == "local" {
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for sf := range ch {
					c.traitCompiler.AnalyzeSyncLocal(sf)
				}
			}()
		}
	} else {
		// Global mode: AnalyzeSync calls Checker.GetSymbolAtLocation which writes
		// into LinkStore maps — the TypeScript Checker is NOT goroutine-safe.
		// Run all source files through a SINGLE worker to avoid concurrent map writes.
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sf := range ch {
				c.traitCompiler.AnalyzeSync(sf)
			}
		}()
	}
	wg.Wait()

	if c.traitCompiler.SemanticUpdater != nil {
		res := c.traitCompiler.SemanticUpdater.Finalize()
		if c.incrementalCompilation != nil && c.incrementalCompilation.State != nil {
			c.incrementalCompilation.State.SemanticGraph = c.traitCompiler.SemanticUpdater.GetGraph()
		}
		c.mergeSemanticAffectedFiles(res)
	}

	if c.incrementalCompilation != nil {
		c.incrementalCompilation.RecordSuccessfulAnalysis(c.traitCompiler)
	}

	c.analyzed = true
	return nil
}

func (c *NgCompiler) Resolve() []*ast.Diagnostic {
	if c.resolved {
		return nil
	}
	c.traitCompiler.Resolve()

	if c.traitCompiler.SemanticUpdater != nil {
		res := c.traitCompiler.SemanticUpdater.Finalize()
		if c.incrementalCompilation != nil && c.incrementalCompilation.State != nil {
			c.incrementalCompilation.State.SemanticGraph = c.traitCompiler.SemanticUpdater.GetGraph()
		}
		c.mergeSemanticAffectedFiles(res)
	}

	if c.incrementalCompilation != nil {
		fileDiags := c.traitCompiler.GetDiagnostics()
		res := make(map[string]any)
		for file, diags := range fileDiags {
			res[file] = diags
		}
		c.incrementalCompilation.RecordSuccessfulTypeCheck(res)
	}

	c.resolved = true
	return nil
}

func (c *NgCompiler) mergeSemanticAffectedFiles(res semantic_graph.SemanticDependencyResult) {
	if res == nil {
		return
	}
	if c.affectedFiles == nil {
		c.affectedFiles = make(map[string]bool)
	}
	for _, f := range res.GetAffectedFiles() {
		c.affectedFiles[f] = true
	}
}

func (c *NgCompiler) PrepareEmit() []*ast.Diagnostic {
	if c.prepared {
		return nil
	}

	sourceFiles := c.tsProgram.SourceFiles()
	if len(sourceFiles) == 0 {
		c.prepared = true
		return nil
	}

	ch := make(chan *ast.SourceFile, len(sourceFiles))
	for _, sf := range sourceFiles {
		ch <- sf
	}
	close(ch)

	numWorkers := runtime.NumCPU()
	if numWorkers > 16 {
		numWorkers = 16
	}
	if numWorkers > len(sourceFiles) {
		numWorkers = len(sourceFiles)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
			for sf := range ch {
				isAffected := false
				if c.affectedFiles != nil && c.affectedFiles[sf.FileName()] {
					isAffected = true
				}
				if !isAffected && c.incrementalCompilation != nil && c.incrementalCompilation.SafeToSkipEmit(sf) {
					c.incrementalCompilation.RecordSuccessfulEmit(sf)
					continue
				}
				c.traitCompiler.UpdateSourceFile(sf, factory)
				if c.incrementalCompilation != nil {
					c.incrementalCompilation.RecordSuccessfulEmit(sf)
				}
			}
		}()
	}
	wg.Wait()

	c.prepared = true
	return nil
}

func (c *NgCompiler) GetHmrUpdate(componentId string) string {
	if c.traitCompiler != nil {
		return c.traitCompiler.GetHmrUpdate(componentId)
	}
	return ""
}

func (c *NgCompiler) GetHmrComponentIds() []string {
	if c.traitCompiler != nil {
		return c.traitCompiler.GetHmrComponentIds()
	}
	return nil
}

func getFileVersions(tsProgram *compiler.Program) map[string]string {
	versions := make(map[string]string)
	for _, sf := range tsProgram.SourceFiles() {
		hash := sha256.Sum256([]byte(sf.Text()))
		versions[sf.FileName()] = fmt.Sprintf("%x", hash)
	}
	return versions
}
