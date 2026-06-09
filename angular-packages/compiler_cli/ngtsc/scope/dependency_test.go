package scope_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestLocalModuleScopeRegistry_ResolveDependencyNgModule(t *testing.T) {
	// 1. should produce an accurate scope for a basic NgModule
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	moduleNode := &ast.Node{}
	dirNode := &ast.Node{}
	compNode := &ast.Node{}

	// Register Dir with owning module 'test' (acting as dependency)
	metaRegistry.RegisterDirective(dirNode, &metadata.DirectiveMeta{
		Selector: "[dir]",
		Ref: metadata.Reference{
			Node:         dirNode,
			OwningModule: "test",
		},
	})

	// Register Module with owning module 'test'
	metaRegistry.RegisterNgModule(moduleNode, &metadata.NgModuleMeta{
		Ref: metadata.Reference{
			Node:         moduleNode,
			OwningModule: "test",
		},
		Declarations: []metadata.Reference{
			{Node: dirNode, OwningModule: "test"},
		},
		Exports: []metadata.Reference{
			{Node: dirNode, OwningModule: "test"},
		},
	})

	// Local component imports Module
	metaRegistry.RegisterDirective(compNode, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindComponent,
		Selector:    "local-comp",
		Standalone:  true,
		IsComponent: true,
		Imports: []metadata.Reference{
			{Node: moduleNode, OwningModule: "test"},
		},
	})

	compilationScope := scopeRegistry.GetCompilationScope(compNode)
	assert.NotNil(t, compilationScope)
	assert.Len(t, compilationScope.Directives, 1)
	assert.Equal(t, "[dir]", compilationScope.Directives[0].Selector)
	assert.Equal(t, "test", compilationScope.Directives[0].Ref.OwningModule)
}

func TestLocalModuleScopeRegistry_ResolveDependencyNgModuleExported(t *testing.T) {
	// 2. should produce an accurate scope when a module is exported
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	moduleANode := &ast.Node{}
	moduleBNode := &ast.Node{}
	dirNode := &ast.Node{}
	compNode := &ast.Node{}

	// Register metadata with owning module 'test'
	metaRegistry.RegisterDirective(dirNode, &metadata.DirectiveMeta{
		Selector: "[dir]",
		Ref: metadata.Reference{
			Node:         dirNode,
			OwningModule: "test",
		},
	})

	// ModuleA: declarations=[Dir], exports=[Dir]
	metaRegistry.RegisterNgModule(moduleANode, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleANode, OwningModule: "test"},
		Declarations: []metadata.Reference{{Node: dirNode, OwningModule: "test"}},
		Exports:      []metadata.Reference{{Node: dirNode, OwningModule: "test"}},
	})

	// ModuleB: exports=[ModuleA]
	metaRegistry.RegisterNgModule(moduleBNode, &metadata.NgModuleMeta{
		Ref:     metadata.Reference{Node: moduleBNode, OwningModule: "test"},
		Exports: []metadata.Reference{{Node: moduleANode, OwningModule: "test"}},
	})

	// Local component imports ModuleB
	metaRegistry.RegisterDirective(compNode, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindComponent,
		Selector:    "local-comp",
		Standalone:  true,
		IsComponent: true,
		Imports: []metadata.Reference{
			{Node: moduleBNode, OwningModule: "test"},
		},
	})

	compilationScope := scopeRegistry.GetCompilationScope(compNode)
	assert.NotNil(t, compilationScope)
	assert.Len(t, compilationScope.Directives, 1)
	assert.Equal(t, "[dir]", compilationScope.Directives[0].Selector)
	assert.Equal(t, "test", compilationScope.Directives[0].Ref.OwningModule)
}

func TestLocalModuleScopeRegistry_ResolveAcrossModules(t *testing.T) {
	// 3. should resolve correctly across modules (different owning modules)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	moduleANode := &ast.Node{}
	moduleBNode := &ast.Node{}
	dirNode := &ast.Node{}
	compNode := &ast.Node{}

	// Register Dir from 'declaration' module
	metaRegistry.RegisterDirective(dirNode, &metadata.DirectiveMeta{
		Selector: "[dir]",
		Ref: metadata.Reference{
			Node:         dirNode,
			OwningModule: "declaration",
		},
	})

	// ModuleA from 'declaration' module: declarations=[Dir], exports=[Dir]
	metaRegistry.RegisterNgModule(moduleANode, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleANode, OwningModule: "declaration"},
		Declarations: []metadata.Reference{{Node: dirNode, OwningModule: "declaration"}},
		Exports:      []metadata.Reference{{Node: dirNode, OwningModule: "declaration"}},
	})

	// ModuleB from 'exported' module: exports=[ModuleA]
	metaRegistry.RegisterNgModule(moduleBNode, &metadata.NgModuleMeta{
		Ref:     metadata.Reference{Node: moduleBNode, OwningModule: "exported"},
		Exports: []metadata.Reference{{Node: moduleANode, OwningModule: "declaration"}},
	})

	// Local component imports ModuleB
	metaRegistry.RegisterDirective(compNode, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindComponent,
		Selector:    "local-comp",
		Standalone:  true,
		IsComponent: true,
		Imports: []metadata.Reference{
			{Node: moduleBNode, OwningModule: "exported"},
		},
	})

	compilationScope := scopeRegistry.GetCompilationScope(compNode)
	assert.NotNil(t, compilationScope)
	assert.Len(t, compilationScope.Directives, 1)
	assert.Equal(t, "[dir]", compilationScope.Directives[0].Selector)
	// Verified: directive retains its original best guess owning module 'declaration'
	assert.Equal(t, "declaration", compilationScope.Directives[0].Ref.OwningModule)
}
