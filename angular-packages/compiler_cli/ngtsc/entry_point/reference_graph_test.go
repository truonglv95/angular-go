// Package entry_point_test ports reference_graph_spec.ts from ngtsc 1:1.
package entry_point_test

import (
	"sort"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/entry_point"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nodeRegistry maps string names to stable fake *ast.Node pointers for use as
// DeclarationNode (= *ast.Node) keys in the ReferenceGraph test.
// This mirrors the TypeScript test's use of plain strings as graph nodes.
type nodeRegistry struct {
	nodes map[string]*ast.Node
}

func newNodeRegistry() *nodeRegistry {
	return &nodeRegistry{nodes: make(map[string]*ast.Node)}
}

func (r *nodeRegistry) get(name string) *ast.Node {
	if n, ok := r.nodes[name]; ok {
		return n
	}
	// Allocate a unique *ast.Node per name using a synthetic SourceFile node.
	// We only need pointer identity here; the node is never walked.
	n := &ast.Node{}
	r.nodes[name] = n
	return n
}

// sortedTransitiveRefs returns the sorted string names of transitive references of `from`.
func sortedTransitiveRefs(graph *entry_point.ReferenceGraph, reg *nodeRegistry, from string) []string {
	refs := graph.TransitiveReferencesOf(reflection.DeclarationNode(reg.get(from)))
	// Reverse-map pointer → name for readable comparison.
	ptrToName := make(map[*ast.Node]string)
	for name, ptr := range reg.nodes {
		ptrToName[ptr] = name
	}
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ptrToName[(*ast.Node)(ref)])
	}
	sort.Strings(names)
	return names
}

// pathNames returns the string names along a path, or nil if no path exists.
func pathNames(graph *entry_point.ReferenceGraph, reg *nodeRegistry, from, to string) []string {
	path := graph.PathFrom(
		reflection.DeclarationNode(reg.get(from)),
		reflection.DeclarationNode(reg.get(to)),
	)
	if path == nil {
		return nil
	}
	ptrToName := make(map[*ast.Node]string)
	for name, ptr := range reg.nodes {
		ptrToName[ptr] = name
	}
	result := make([]string, len(path))
	for i, n := range path {
		result[i] = ptrToName[(*ast.Node)(n)]
	}
	return result
}

// makeBaseGraph creates the base graph used in the beforeEach:
//
//	origin -> alpha -> beta -> gamma
func makeBaseGraph(t *testing.T) (*entry_point.ReferenceGraph, *nodeRegistry) {
	t.Helper()
	reg := newNodeRegistry()
	graph := entry_point.NewReferenceGraph()
	graph.Add(reflection.DeclarationNode(reg.get("origin")), reflection.DeclarationNode(reg.get("alpha")))
	graph.Add(reflection.DeclarationNode(reg.get("alpha")), reflection.DeclarationNode(reg.get("beta")))
	graph.Add(reflection.DeclarationNode(reg.get("beta")), reflection.DeclarationNode(reg.get("gamma")))
	return graph, reg
}

// TestReferenceGraph_SimpleChain mirrors:
// it('should track a simple chain of references', ...)
func TestReferenceGraph_SimpleChain(t *testing.T) {
	graph, reg := makeBaseGraph(t)
	// origin -> alpha -> beta -> gamma
	require.Equal(t, []string{"alpha", "beta", "gamma"}, sortedTransitiveRefs(graph, reg, "origin"))
	require.Equal(t, []string{"gamma"}, sortedTransitiveRefs(graph, reg, "beta"))
}

// TestReferenceGraph_CycleNoCrash mirrors:
// it('should not crash on a cycle', ...)
func TestReferenceGraph_CycleNoCrash(t *testing.T) {
	graph, reg := makeBaseGraph(t)
	// Add cycle: beta -> origin
	graph.Add(reflection.DeclarationNode(reg.get("beta")), reflection.DeclarationNode(reg.get("origin")))
	assert.Equal(t, []string{"alpha", "beta", "gamma", "origin"}, sortedTransitiveRefs(graph, reg, "origin"))
}

// TestReferenceGraph_PathBetweenNodes mirrors:
// it('should report a path between two nodes in the graph', ...)
func TestReferenceGraph_PathBetweenNodes(t *testing.T) {
	graph, reg := makeBaseGraph(t)
	//             ,------------------------\
	// origin -> alpha -> beta -> gamma -> delta
	//                      \----------------^
	graph.Add(reflection.DeclarationNode(reg.get("beta")), reflection.DeclarationNode(reg.get("delta")))
	graph.Add(reflection.DeclarationNode(reg.get("delta")), reflection.DeclarationNode(reg.get("alpha")))

	pathOriginGamma := pathNames(graph, reg, "origin", "gamma")
	require.Equal(t, []string{"origin", "alpha", "beta", "gamma"}, pathOriginGamma)

	pathBetaAlpha := pathNames(graph, reg, "beta", "alpha")
	require.Equal(t, []string{"beta", "delta", "alpha"}, pathBetaAlpha)
}

// TestReferenceGraph_NoPath mirrors:
// it("should not report a path that doesn't exist", ...)
func TestReferenceGraph_NoPath(t *testing.T) {
	graph, reg := makeBaseGraph(t)
	path := pathNames(graph, reg, "gamma", "beta")
	assert.Nil(t, path)
}
