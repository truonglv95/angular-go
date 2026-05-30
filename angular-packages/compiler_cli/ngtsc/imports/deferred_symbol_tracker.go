package imports

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type DeferredSymbolTracker struct {
}

func (recv *DeferredSymbolTracker) GetNonRemovableDeferredImports(sourceFile ast.SourceFile, classDecl ast.Node) []ast.Node {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DeferredSymbolTracker) MarkAsDeferrableCandidate(identifier ast.Identifier, importDecl ast.Node, componentClassDecl ast.Node, isExplicitlyDeferred bool) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DeferredSymbolTracker) CanDefer(importDecl ast.Node) bool {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DeferredSymbolTracker) GetDeferrableImportDecls() map[string]bool {
	// TODO: stub
	panic("unimplemented")
}
