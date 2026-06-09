package typecheck

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/internal/ast"
)

type OptimizeFor int

const (
	OptimizeFor_WholeProgram OptimizeFor = iota
	OptimizeFor_SingleFile
)

type TemplateTypeChecker interface {
	GetDiagnosticsForFile(sf *ast.SourceFile, mode OptimizeFor) []*ast.Diagnostic
	GetTypeCheckBlock(classDecl *ast.Node) *ast.Node
	GetTemplate(classDecl *ast.Node) []render3.Node
}
