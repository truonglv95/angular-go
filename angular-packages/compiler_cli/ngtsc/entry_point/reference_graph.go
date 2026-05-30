package entry_point

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/reflection"
)

type ReferenceGraph struct {
	references map[reflection.DeclarationNode]map[reflection.DeclarationNode]bool
}

func NewReferenceGraph() *ReferenceGraph {
	return &ReferenceGraph{
		references: make(map[reflection.DeclarationNode]map[reflection.DeclarationNode]bool),
	}
}

func (g *ReferenceGraph) Add(from reflection.DeclarationNode, to reflection.DeclarationNode) {
	if g.references[from] == nil {
		g.references[from] = make(map[reflection.DeclarationNode]bool)
	}
	g.references[from][to] = true
}

func (g *ReferenceGraph) TransitiveReferencesOf(target reflection.DeclarationNode) []reflection.DeclarationNode {
	set := make(map[reflection.DeclarationNode]bool)
	g.collectTransitiveReferences(set, target)

	var result []reflection.DeclarationNode
	for node := range set {
		result = append(result, node)
	}
	return result
}

func (g *ReferenceGraph) PathFrom(source reflection.DeclarationNode, target reflection.DeclarationNode) []reflection.DeclarationNode {
	seen := make(map[reflection.DeclarationNode]bool)
	return g.collectPathFrom(source, target, seen)
}

func (g *ReferenceGraph) collectPathFrom(source reflection.DeclarationNode, target reflection.DeclarationNode, seen map[reflection.DeclarationNode]bool) []reflection.DeclarationNode {
	if source == target {
		return []reflection.DeclarationNode{target}
	} else if seen[source] {
		return nil
	}
	seen[source] = true

	edges := g.references[source]
	if len(edges) == 0 {
		return nil
	}

	for edge := range edges {
		partialPath := g.collectPathFrom(edge, target, seen)
		if partialPath != nil {
			candidatePath := make([]reflection.DeclarationNode, 0, len(partialPath)+1)
			candidatePath = append(candidatePath, source)
			candidatePath = append(candidatePath, partialPath...)
			return candidatePath
		}
	}

	return nil
}

func (g *ReferenceGraph) collectTransitiveReferences(set map[reflection.DeclarationNode]bool, decl reflection.DeclarationNode) {
	edges := g.references[decl]
	if len(edges) > 0 {
		for ref := range edges {
			if !set[ref] {
				set[ref] = true
				g.collectTransitiveReferences(set, ref)
			}
		}
	}
}
