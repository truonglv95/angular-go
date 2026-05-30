package semantic_graph

type SemanticSymbol struct {
	Path       string
	Identifier string
}

func (recv *SemanticSymbol) IsPublicApiAffected(previousSymbol SemanticSymbol) bool {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticSymbol) IsEmitAffected(previousSymbol SemanticSymbol, publicApiAffected map[string]bool) bool {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticSymbol) IsTypeCheckApiAffected(previousSymbol SemanticSymbol) bool {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticSymbol) IsTypeCheckBlockAffected(previousSymbol SemanticSymbol, typeCheckApiAffected map[string]bool) bool {
	// TODO: stub
	panic("unimplemented")
}

type SemanticReference interface {
}
