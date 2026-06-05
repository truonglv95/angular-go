package core

import (
	"context"
	"runtime"
	"sync"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations_local"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
)

type NgCompiler struct {
	tsProgram     *compiler.Program
	checker       *checker.Checker
	reflector     reflection.ReflectionHost
	handlers      []transform.DecoratorHandler
	traitCompiler *transform.TraitCompiler

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
	if oldCompiler != nil {
		oldTc = oldCompiler.traitCompiler
	}
	traitCompiler := transform.NewTraitCompiler(handlers, refHost, localRefHost, oldTc)

	return &NgCompiler{
		tsProgram:       tsProgram,
		checker:         chk,
		reflector:       refHost,
		handlers:        handlers,
		traitCompiler:   traitCompiler,
		metaRegistry:    localMetaRegistry,
		scopeRegistry:   scopeRegistry,
		compilationMode: options.CompilationMode,
		options:         options,
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

	c.analyzed = true
	return nil
}

func (c *NgCompiler) Resolve() []*ast.Diagnostic {
	if c.resolved {
		return nil
	}
	c.traitCompiler.Resolve()
	c.resolved = true
	return nil
}

func (c *NgCompiler) PrepareEmit() []*ast.Diagnostic {
	if c.prepared {
		return nil
	}

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	for _, sf := range c.tsProgram.SourceFiles() {
		c.traitCompiler.UpdateSourceFile(sf, factory)
	}

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
