package imports

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type ImportedSymbolsTracker struct {
}

func (recv *ImportedSymbolsTracker) IsPotentialReferenceToNamedImport(node ast.Identifier, exportedName string, moduleName string) bool {
	// TODO: stub
	panic("unimplemented")
}

func (recv *ImportedSymbolsTracker) IsPotentialReferenceToNamespaceImport(node ast.Identifier, moduleName string) bool {
	// TODO: stub
	panic("unimplemented")
}

func (recv *ImportedSymbolsTracker) HasNamedImport(sourceFile ast.SourceFile, exportedName string, moduleName string) bool {
	// TODO: stub
	panic("unimplemented")
}

func (recv *ImportedSymbolsTracker) HasNamespaceImport(sourceFile ast.SourceFile, moduleName string) bool {
	// TODO: stub
	panic("unimplemented")
}
