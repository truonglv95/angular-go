package core

import (
	"context"
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
	"github.com/microsoft/typescript-go/internal/perf"
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

	analyzed bool
	resolved bool
	prepared bool
}

func NewNgCompiler(tsProgram *compiler.Program, compilationMode string) (*NgCompiler, error) {
	ctx := context.Background()
	var chk *checker.Checker
	if compilationMode != "local" {
		func() {
			defer perf.Time("ts.type_checker_create")()
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
	if compilationMode == "local" {
		handlers = []transform.DecoratorHandler{
			annotations_local.NewComponentLocalDecoratorHandler(refHost, false, localMetaRegistry, scopeRegistry),
			annotations_local.NewDirectiveLocalDecoratorHandler(refHost, localMetaRegistry),
			annotations_local.NewPipeLocalDecoratorHandler(refHost, localMetaRegistry),
			annotations_local.NewInjectableLocalDecoratorHandler(refHost), // LOCAL HANDLER
			annotations_local.NewNgModuleLocalDecoratorHandler(refHost, localMetaRegistry, scopeRegistry),
		}
	} else {
		handlers = []transform.DecoratorHandler{
			annotations.NewComponentDecoratorHandler(refHost, false, localMetaRegistry, scopeRegistry),
			annotations.NewDirectiveDecoratorHandler(refHost, localMetaRegistry),
			annotations.NewPipeDecoratorHandler(refHost, localMetaRegistry),
			annotations.NewInjectableDecoratorHandler(refHost), // GLOBAL HANDLER
			annotations.NewNgModuleDecoratorHandler(refHost, localMetaRegistry, scopeRegistry),
		}
	}
	traitCompiler := transform.NewTraitCompiler(handlers, refHost, localRefHost)

	return &NgCompiler{
		tsProgram:       tsProgram,
		checker:         chk,
		reflector:       refHost,
		handlers:        handlers,
		traitCompiler:   traitCompiler,
		metaRegistry:    localMetaRegistry,
		scopeRegistry:   scopeRegistry,
		compilationMode: compilationMode,
	}, nil
}

func (c *NgCompiler) AnalyzeSync() []*ast.Diagnostic {
	if c.analyzed {
		return nil
	}
	defer perf.Time("angular.analyze")()
	if c.compilationMode == "local" {
		var wg sync.WaitGroup
		for _, sf := range c.tsProgram.SourceFiles() {
			wg.Add(1)
			go func(file *ast.SourceFile) {
				defer wg.Done()
				c.traitCompiler.AnalyzeSyncLocal(file)
			}(sf)
		}
		wg.Wait()
	} else {
		for _, sf := range c.tsProgram.SourceFiles() {
			c.traitCompiler.AnalyzeSync(sf)
		}
	}
	c.analyzed = true
	return nil
}

func (c *NgCompiler) Resolve() []*ast.Diagnostic {
	if c.resolved {
		return nil
	}
	defer perf.Time("angular.resolve")()
	c.traitCompiler.Resolve()
	c.resolved = true
	return nil
}

func (c *NgCompiler) PrepareEmit() []*ast.Diagnostic {
	if c.prepared {
		return nil
	}
	defer perf.Time("angular.prepare_emit")()
	var wg sync.WaitGroup
	for _, sf := range c.tsProgram.SourceFiles() {
		sf := sf
		wg.Add(1)
		go func() {
			defer wg.Done()
			factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
			c.traitCompiler.UpdateSourceFile(sf, factory)
		}()
	}
	wg.Wait()
	c.prepared = true
	return nil
}
