package entry_point

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

type PrivateExportChecker struct {}

func NewPrivateExportChecker(entryPoint ast.Node, checker any, reflector reflection.ReflectionHost) *PrivateExportChecker {
	return &PrivateExportChecker{}
}

func (c *PrivateExportChecker) Check(decl reflection.DeclarationNode) []ast.Diagnostic {
	return nil
}
