package incremental_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncrementalReconciliation_ChangedVersion(t *testing.T) {
	fooPath := "/foo.ts"
	opts := ast.SourceFileParseOptions{
		FileName: fooPath,
	}

	sf := parser.ParseSourceFile(opts, "export const FOO = true;", core.ScriptKindTS)
	require.NotNil(t, sf)

	versionMapFirst := map[string]string{fooPath: "version.1"}
	firstCompilation := incremental.Fresh(versionMapFirst)
	firstCompilation.RecordSuccessfulAnalysis(nil)
	firstCompilation.RecordSuccessfulEmit(sf)

	versionMapSecond := map[string]string{fooPath: "version.2"}
	secondCompilation := incremental.Incremental(
		nil,
		versionMapSecond,
		nil,
		firstCompilation.State,
		make(map[string]bool),
		nil,
	)

	secondCompilation.RecordSuccessfulAnalysis(nil)
	assert.False(t, secondCompilation.SafeToSkipEmit(sf), "Should NOT be safe to skip emit because version has changed")
}

func TestFileDependencyGraph_Basic(t *testing.T) {
	graph := incremental.NewFileDependencyGraph()
	graph.AddDependency("/a.ts", "/b.ts")
	graph.AddDependency("/a.ts", "/c.ts")

	changed := map[string]bool{"/c.ts": true}
	affected := graph.UpdateWithPhysicalChanges(*graph, changed, nil, nil)
	assert.True(t, affected["/c.ts"])
	assert.True(t, affected["/a.ts"], "Transitive dependency: a.ts depends on c.ts, so changing c.ts must affect a.ts")
}

func TestFileDependencyGraph_ResourceDependency(t *testing.T) {
	graph := incremental.NewFileDependencyGraph()
	graph.AddResourceDependency("/a.ts", "/a.html")

	deps := graph.GetResourceDependencies("/a.ts")
	assert.Contains(t, deps, "/a.html")

	changedRes := map[string]bool{"/a.html": true}
	affected := graph.UpdateWithPhysicalChanges(*graph, nil, nil, changedRes)
	assert.True(t, affected["/a.ts"])
}

func TestFileDependencyGraph_FailedAnalysis(t *testing.T) {
	graph := incremental.NewFileDependencyGraph()
	graph.RecordDependencyAnalysisFailure("/a.ts")

	affected := graph.UpdateWithPhysicalChanges(*graph, nil, nil, nil)
	assert.True(t, affected["/a.ts"])
}

func TestIncrementalBuildStrategy_Tracked(t *testing.T) {
	strategy := incremental.NewTrackedIncrementalBuildStrategy()

	state := strategy.GetIncrementalState(nil)
	assert.Empty(t, state.Versions)

	newState := incremental.IncrementalState{
		Versions:     map[string]string{"/a.ts": "v1"},
		EmittedFiles: map[string]bool{"/a.ts": true},
	}
	strategy.SetIncrementalState(newState, nil)

	retrieved := strategy.GetIncrementalState(nil)
	assert.Equal(t, "v1", retrieved.Versions["/a.ts"])
	assert.True(t, retrieved.EmittedFiles["/a.ts"])
}

func TestIncrementalReconciliation_SkipEmitRequiresSemanticReferences(t *testing.T) {
	fooPath := "/foo.ts"
	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: fooPath}, "export const FOO = true;", core.ScriptKindTS)
	require.NotNil(t, sf)

	firstCompilation := incremental.Fresh(map[string]string{fooPath: "version.1"})
	firstCompilation.RecordSuccessfulEmit(sf)

	secondCompilation := incremental.Incremental(
		nil,
		map[string]string{fooPath: "version.1"},
		nil,
		firstCompilation.State,
		make(map[string]bool),
		nil,
	)
	assert.False(t, secondCompilation.SafeToSkipEmit(sf), "Should not skip emit until semantic graph references prove the file is unaffected")

	graph := semantic_graph.NewSemanticDepGraph()
	updater := semantic_graph.NewSemanticDepGraphUpdater(graph)
	symA := semantic_graph.SemanticSymbol{Path: "/foo.ts", Identifier: "Foo", Kind: "component", Selector: "foo"}
	symB := semantic_graph.SemanticSymbol{Path: "/bar.ts", Identifier: "Bar", Kind: "directive", Selector: "bar"}
	updater.RegisterSymbolForDecl("/foo.ts@Foo", symA)
	updater.RegisterSymbolForDecl("/bar.ts@Bar", symB)
	updater.RegisterReference("/foo.ts@Foo", semantic_graph.SymbolKey("/bar.ts", "Bar"))
	updater.Finalize()
	firstCompilation.State.SemanticGraph = graph

	thirdCompilation := incremental.Incremental(
		nil,
		map[string]string{fooPath: "version.1"},
		nil,
		firstCompilation.State,
		make(map[string]bool),
		nil,
	)
	assert.True(t, thirdCompilation.SafeToSkipEmit(sf), "Should skip unchanged emit once prior semantic references are available")
}

func TestSemanticSymbol_IsPublicApiAffected(t *testing.T) {
	s1 := semantic_graph.SemanticSymbol{
		Path:       "/foo.ts",
		Identifier: "MyComponent",
		Kind:       "component",
		Selector:   "my-comp",
		Inputs:     map[string]string{"input1": "input1_alias"},
		Outputs:    map[string]string{"output1": "output1_alias"},
		ExportAs:   []string{"myCompAlias"},
	}

	// 1. Identical symbol should not affect public API
	assert.False(t, s1.IsPublicApiAffected(s1))

	// 2. Change selector
	s2 := s1
	s2.Selector = "new-comp"
	assert.True(t, s2.IsPublicApiAffected(s1))

	// 3. Change inputs
	s3 := s1
	s3.Inputs = map[string]string{"input1": "new_alias"}
	assert.True(t, s3.IsPublicApiAffected(s1))

	// 4. Change outputs
	s4 := s1
	s4.Outputs = map[string]string{"output1": "new_alias"}
	assert.True(t, s4.IsPublicApiAffected(s1))

	// 5. Change exportAs
	s5 := s1
	s5.ExportAs = []string{"newAlias"}
	assert.True(t, s5.IsPublicApiAffected(s1))

	// 6. Change identifier/path/kind
	s6 := s1
	s6.Identifier = "OtherComponent"
	assert.True(t, s6.IsPublicApiAffected(s1))
}

func TestSemanticDepGraphUpdater_Finalize(t *testing.T) {
	graph := semantic_graph.NewSemanticDepGraph()
	updater1 := semantic_graph.NewSemanticDepGraphUpdater(graph)

	sym1 := semantic_graph.SemanticSymbol{
		Path:       "/foo.ts",
		Identifier: "CompA",
		Kind:       "component",
		Selector:   "comp-a",
	}
	updater1.RegisterSymbol(sym1)
	res1 := updater1.Finalize()
	// Since there was no prior symbol, this should be in the affected files
	assert.Contains(t, res1.GetAffectedFiles(), "/foo.ts")

	// Next compilation, unchanged CompA
	updater2 := semantic_graph.NewSemanticDepGraphUpdater(graph)
	updater2.RegisterSymbol(sym1)
	res2 := updater2.Finalize()
	assert.Empty(t, res2.GetAffectedFiles(), "Should not be affected since CompA hasn't changed")

	// Next compilation, changed CompA
	sym1Changed := sym1
	sym1Changed.Selector = "comp-a-new"
	updater3 := semantic_graph.NewSemanticDepGraphUpdater(graph)
	updater3.RegisterSymbol(sym1Changed)
	res3 := updater3.Finalize()
	assert.Contains(t, res3.GetAffectedFiles(), "/foo.ts", "Should be affected because selector changed")
}

func TestSemanticDepGraphUpdater_PropagatesReferenceInvalidation(t *testing.T) {
	graph := semantic_graph.NewSemanticDepGraph()
	component := semantic_graph.SemanticSymbol{Path: "/cmp.ts", Identifier: "Cmp", Kind: "component", Selector: "cmp"}
	directive := semantic_graph.SemanticSymbol{Path: "/dir.ts", Identifier: "Dir", Kind: "directive", Selector: "[dir]"}

	updater1 := semantic_graph.NewSemanticDepGraphUpdater(graph)
	updater1.RegisterSymbolForDecl("/cmp.ts@Cmp", component)
	updater1.RegisterSymbolForDecl("/dir.ts@Dir", directive)
	updater1.RegisterReference("/cmp.ts@Cmp", semantic_graph.SymbolKey("/dir.ts", "Dir"))
	updater1.Finalize()

	updater2 := semantic_graph.NewSemanticDepGraphUpdater(graph)
	updater2.RegisterSymbolForDecl("/cmp.ts@Cmp", component)
	changedDirective := directive
	changedDirective.Inputs = map[string]string{"value": "dirValue"}
	updater2.RegisterSymbolForDecl("/dir.ts@Dir", changedDirective)
	updater2.RegisterReference("/cmp.ts@Cmp", semantic_graph.SymbolKey("/dir.ts", "Dir"))
	res := updater2.Finalize()

	assert.Contains(t, res.GetAffectedFiles(), "/dir.ts")
	assert.Contains(t, res.GetAffectedFiles(), "/cmp.ts", "Component should be affected when a referenced directive public API changes")
}
