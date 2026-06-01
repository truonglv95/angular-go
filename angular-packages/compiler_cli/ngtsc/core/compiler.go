package core

import (
	"context"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/internal/compiler"
)

type NgCompiler struct {
	tsProgram     *compiler.Program
	checker       any // will cast to typechecker
	reflector     reflection.ReflectionHost
	handlers      []transform.DecoratorHandler
	traitCompiler *transform.TraitCompiler

	metaRegistry  *metadata.LocalMetadataRegistry
	scopeRegistry *scope.LocalModuleScopeRegistry

	analyzed bool
}

func NewNgCompiler(tsProgram *compiler.Program) (*NgCompiler, error) {
	ctx := context.Background()
	chk, _ := tsProgram.GetTypeChecker(ctx)
	refHost := reflection.NewTypeScriptReflectionHost(chk)

	localMetaRegistry := metadata.NewLocalMetadataRegistry()
	dtsMetaReader := metadata.NewDtsMetadataReader(chk)
	compoundMetaReader := metadata.NewCompoundMetadataReader([]metadata.MetadataReader{localMetaRegistry, dtsMetaReader})
	
	scopeRegistry := scope.NewLocalModuleScopeRegistry(compoundMetaReader)

	handlers := []transform.DecoratorHandler{
		annotations.NewComponentDecoratorHandler(refHost, false, localMetaRegistry, scopeRegistry),
		annotations.NewDirectiveDecoratorHandler(refHost, localMetaRegistry),
		annotations.NewPipeDecoratorHandler(refHost, localMetaRegistry),
		annotations.NewInjectableDecoratorHandler(refHost),
		annotations.NewNgModuleDecoratorHandler(refHost, localMetaRegistry, scopeRegistry),
	}
	traitCompiler := transform.NewTraitCompiler(handlers, refHost)

	return &NgCompiler{
		tsProgram:     tsProgram,
		checker:       chk,
		reflector:     refHost,
		handlers:      handlers,
		traitCompiler: traitCompiler,
		metaRegistry:  localMetaRegistry,
		scopeRegistry: scopeRegistry,
	}, nil
}

func (c *NgCompiler) AnalyzeSync() []*ast.Diagnostic {
	for _, sf := range c.tsProgram.SourceFiles() {
		c.traitCompiler.AnalyzeSync(sf)
	}
	c.analyzed = true
	return nil
}

func (c *NgCompiler) Resolve() []*ast.Diagnostic {
	c.traitCompiler.Resolve()
	return nil
}

func (c *NgCompiler) PrepareEmit() []*ast.Diagnostic {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	for _, sf := range c.tsProgram.SourceFiles() {
		c.traitCompiler.UpdateSourceFile(sf, factory)
	}
	return nil
}
