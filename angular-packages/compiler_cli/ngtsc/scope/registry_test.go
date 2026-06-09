package scope_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestLocalModuleScopeRegistry_Basic(t *testing.T) {
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	moduleNode := &ast.Node{}
	compNode := &ast.Node{}
	dirNode := &ast.Node{}
	pipeNode := &ast.Node{}

	// Register metadata
	dirMeta := &metadata.DirectiveMeta{
		Kind:     metadata.MetaKindDirective,
		Ref:      metadata.Reference{Node: dirNode},
		Selector: "app-dir",
	}
	metaRegistry.RegisterDirective(dirNode, dirMeta)

	pipeMeta := &metadata.PipeMeta{
		Ref:  metadata.Reference{Node: pipeNode},
		Name: "app-pipe",
	}
	metaRegistry.RegisterPipe(pipeNode, pipeMeta)

	moduleMeta := &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleNode},
		Declarations: []metadata.Reference{{Node: compNode}},
		Imports:      []metadata.Reference{{Node: dirNode}, {Node: pipeNode}},
	}
	metaRegistry.RegisterNgModule(moduleNode, moduleMeta)

	scopeRegistry.RegisterComponentDeclaration(compNode, moduleNode)

	compilationScope := scopeRegistry.GetCompilationScope(compNode)
	assert.NotNil(t, compilationScope)
	assert.Len(t, compilationScope.Directives, 1)
	assert.Equal(t, "app-dir", compilationScope.Directives[0].Selector)
	assert.Len(t, compilationScope.Pipes, 1)
	assert.Equal(t, "app-pipe", compilationScope.Pipes[0].Name)
}

func TestLocalModuleScopeRegistry_TransitiveModuleChain(t *testing.T) {
	// Tests complex nested modules:
	// ModuleA imports ModuleB
	// ModuleB exports ModuleC and DirB
	// ModuleC exports DirCE
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	dirA := &ast.Node{}
	dirB := &ast.Node{}
	dirCE := &ast.Node{}

	moduleA := &ast.Node{}
	moduleB := &ast.Node{}
	moduleC := &ast.Node{}

	compA := &ast.Node{} // Declared in ModuleA

	// Register directives
	metaRegistry.RegisterDirective(dirA, &metadata.DirectiveMeta{Selector: "dir-a"})
	metaRegistry.RegisterDirective(dirB, &metadata.DirectiveMeta{Selector: "dir-b"})
	metaRegistry.RegisterDirective(dirCE, &metadata.DirectiveMeta{Selector: "dir-ce"})

	// ModuleC: declarations=[DirCE], exports=[DirCE]
	metaRegistry.RegisterNgModule(moduleC, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleC},
		Declarations: []metadata.Reference{{Node: dirCE}},
		Exports:      []metadata.Reference{{Node: dirCE}},
	})

	// ModuleB: declarations=[DirB], exports=[ModuleC, DirB]
	metaRegistry.RegisterNgModule(moduleB, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleB},
		Declarations: []metadata.Reference{{Node: dirB}},
		Exports:      []metadata.Reference{{Node: moduleC}, {Node: dirB}},
	})

	// ModuleA: declarations=[CompA], imports=[ModuleB]
	metaRegistry.RegisterNgModule(moduleA, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleA},
		Declarations: []metadata.Reference{{Node: compA}},
		Imports:      []metadata.Reference{{Node: moduleB}},
	})

	scopeRegistry.RegisterComponentDeclaration(compA, moduleA)

	compilationScope := scopeRegistry.GetCompilationScope(compA)
	assert.NotNil(t, compilationScope)

	// Directives visible in CompA template should be:
	// - DirB (exported by ModuleB)
	// - DirCE (exported by ModuleC, since ModuleC is exported by ModuleB)
	assert.Len(t, compilationScope.Directives, 2)

	var selectors []string
	for _, d := range compilationScope.Directives {
		selectors = append(selectors, d.Selector)
	}
	assert.Contains(t, selectors, "dir-b")
	assert.Contains(t, selectors, "dir-ce")
}

func TestLocalModuleScopeRegistry_StandaloneComponent(t *testing.T) {
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	standaloneComp := &ast.Node{}
	importedDir := &ast.Node{}
	importedPipe := &ast.Node{}
	importedModule := &ast.Node{}
	exportedDirFromModule := &ast.Node{}

	// Register metadata for the imported elements
	metaRegistry.RegisterDirective(importedDir, &metadata.DirectiveMeta{
		Selector: "imported-dir",
	})
	metaRegistry.RegisterPipe(importedPipe, &metadata.PipeMeta{
		Name: "imported-pipe",
	})
	metaRegistry.RegisterDirective(exportedDirFromModule, &metadata.DirectiveMeta{
		Selector: "exported-dir",
	})
	metaRegistry.RegisterNgModule(importedModule, &metadata.NgModuleMeta{
		Ref:     metadata.Reference{Node: importedModule},
		Exports: []metadata.Reference{{Node: exportedDirFromModule}},
	})

	// Standalone component imports: [importedDir, importedPipe, importedModule]
	metaRegistry.RegisterDirective(standaloneComp, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindComponent,
		Selector:    "standalone-comp",
		Standalone:  true,
		IsComponent: true,
		Imports: []metadata.Reference{
			{Node: importedDir},
			{Node: importedPipe},
			{Node: importedModule},
		},
	})

	compilationScope := scopeRegistry.GetCompilationScope(standaloneComp)
	assert.NotNil(t, compilationScope)

	// Visible directives in standaloneComp should be importedDir and exportedDirFromModule
	assert.Len(t, compilationScope.Directives, 2)
	var selectors []string
	for _, d := range compilationScope.Directives {
		selectors = append(selectors, d.Selector)
	}
	assert.Contains(t, selectors, "imported-dir")
	assert.Contains(t, selectors, "exported-dir")

	// Visible pipes in standaloneComp should be importedPipe
	assert.Len(t, compilationScope.Pipes, 1)
	assert.Equal(t, "imported-pipe", compilationScope.Pipes[0].Name)
}

func TestLocalModuleScopeRegistry_NotTreatExportedAsImported(t *testing.T) {
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	moduleA := &ast.Node{}
	moduleB := &ast.Node{}
	dirB := &ast.Node{}
	compA := &ast.Node{}

	// Register metadata
	metaRegistry.RegisterDirective(dirB, &metadata.DirectiveMeta{
		Selector: "dir-b",
	})
	metaRegistry.RegisterNgModule(moduleB, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleB},
		Declarations: []metadata.Reference{{Node: dirB}},
		Exports:      []metadata.Reference{{Node: dirB}},
	})
	// ModuleA exports ModuleB, but imports is empty!
	metaRegistry.RegisterNgModule(moduleA, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleA},
		Declarations: []metadata.Reference{{Node: compA}},
		Exports:      []metadata.Reference{{Node: moduleB}},
	})

	scopeRegistry.RegisterComponentDeclaration(compA, moduleA)

	compilationScope := scopeRegistry.GetCompilationScope(compA)
	assert.NotNil(t, compilationScope)
	// Since ModuleB is exported but NOT imported by ModuleA,
	// compA (declared in ModuleA) should NOT see dirB in its template compilation scope!
	assert.Empty(t, compilationScope.Directives)
}

func TestLocalModuleScopeRegistry_DeduplicateDeclarationsAndExports(t *testing.T) {
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	moduleA := &ast.Node{}
	moduleB := &ast.Node{}
	moduleC := &ast.Node{}
	dirA := &ast.Node{}
	dirB := &ast.Node{}
	compA := &ast.Node{}

	metaRegistry.RegisterDirective(dirA, &metadata.DirectiveMeta{
		Selector: "dir-a",
	})
	metaRegistry.RegisterDirective(dirB, &metadata.DirectiveMeta{
		Selector: "dir-b",
	})

	metaRegistry.RegisterNgModule(moduleB, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleB},
		Declarations: []metadata.Reference{{Node: dirB}},
		Exports:      []metadata.Reference{{Node: dirB}},
	})
	metaRegistry.RegisterNgModule(moduleC, &metadata.NgModuleMeta{
		Ref:     metadata.Reference{Node: moduleC},
		Exports: []metadata.Reference{{Node: moduleB}},
	})

	// ModuleA imports both ModuleB and ModuleC (which also exports ModuleB).
	// ModuleA also declares dirA twice in its declarations (deduplication check).
	metaRegistry.RegisterNgModule(moduleA, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: moduleA},
		Declarations: []metadata.Reference{{Node: dirA}, {Node: dirA}},
		Imports:      []metadata.Reference{{Node: moduleB}, {Node: moduleC}},
	})

	scopeRegistry.RegisterComponentDeclaration(compA, moduleA)

	compilationScope := scopeRegistry.GetCompilationScope(compA)
	assert.NotNil(t, compilationScope)

	// Directives should be deduplicated: only 1 dir-a and 1 dir-b should be present.
	assert.Len(t, compilationScope.Directives, 2)
	var selectors []string
	for _, d := range compilationScope.Directives {
		selectors = append(selectors, d.Selector)
	}
	assert.Contains(t, selectors, "dir-a")
	assert.Contains(t, selectors, "dir-b")
}
