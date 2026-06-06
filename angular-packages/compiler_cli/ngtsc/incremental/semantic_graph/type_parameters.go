package semantic_graph

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type SemanticTypeParameter interface {
	GetName() string
}

type semanticTypeParamImpl struct {
	name string
}

func (s *semanticTypeParamImpl) GetName() string {
	return s.name
}

func ExtractSemanticTypeParameters(node any) []SemanticTypeParameter {
	n, ok := node.(*ast.Node)
	if !ok || n == nil {
		return nil
	}
	tps := n.TypeParameters()
	if len(tps) == 0 {
		return nil
	}
	var res []SemanticTypeParameter
	for _, tp := range tps {
		if tp == nil {
			continue
		}
		name := ""
		if tp.Name() != nil {
			name = tp.Name().AsIdentifier().Text
		} else {
			name = tp.Text()
		}
		res = append(res, &semanticTypeParamImpl{name: name})
	}
	return res
}

func AreTypeParametersEqual(current []SemanticTypeParameter, previous []SemanticTypeParameter) bool {
	if len(current) != len(previous) {
		return false
	}
	for i := range current {
		if current[i].GetName() != previous[i].GetName() {
			return false
		}
	}
	return true
}
