package shims

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type TopLevelShimGenerator interface {
	ShouldEmit() bool
	MakeTopLevelShim() ast.SourceFile
}

type PerFileShimGenerator interface {
	ExtensionPrefix() string
	ShouldEmit() bool
	GenerateShimForFile(sf ast.SourceFile, genFilePath string, priorShimSf ast.SourceFile) ast.SourceFile
}
