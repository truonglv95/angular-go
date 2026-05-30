package semantic_graph

import "github.com/microsoft/typescript-go/angular-packages/compiler/output"

type SemanticDependencyResult interface {
}

type SemanticDepGraph struct {
	Files        any
	SymbolByDecl map[string]any
}

func (recv *SemanticDepGraph) RegisterSymbol(symbol SemanticSymbol) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticDepGraph) GetEquivalentSymbol(symbol SemanticSymbol) SemanticSymbol {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticDepGraph) GetSymbolByDecl(decl any) SemanticSymbol {
	// TODO: stub
	panic("unimplemented")
}

type SemanticDepGraphUpdater struct {
}

func (recv *SemanticDepGraphUpdater) RegisterSymbol(symbol SemanticSymbol) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticDepGraphUpdater) Finalize() SemanticDependencyResult {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticDepGraphUpdater) GetSemanticReference(decl any, expr output.Expression) SemanticReference {
	// TODO: stub
	panic("unimplemented")
}

func (recv *SemanticDepGraphUpdater) GetSymbol(decl any) SemanticSymbol {
	// TODO: stub
	panic("unimplemented")
}
