package imports

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type LocalCompilationExtraImportsTracker struct {
}

func (recv *LocalCompilationExtraImportsTracker) MarkFileForExtraImportGeneration(sf ast.SourceFile) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *LocalCompilationExtraImportsTracker) AddImportForFile(sf ast.SourceFile, moduleName string) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *LocalCompilationExtraImportsTracker) AddGlobalImportFromIdentifier(node ast.Node) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *LocalCompilationExtraImportsTracker) GetImportsForFile(sf ast.SourceFile) []string {
	// TODO: stub
	panic("unimplemented")
}
