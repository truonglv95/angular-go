package imports

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
)

type ModuleResolver struct {
	program *compiler.Program
}

func NewModuleResolver(program *compiler.Program) *ModuleResolver {
	return &ModuleResolver{
		program: program,
	}
}

func (recv *ModuleResolver) ResolveModule(moduleName string, containingFile string) *ast.SourceFile {
	sf := recv.program.GetSourceFile(containingFile)
	if sf == nil {
		return nil
	}
	mode := recv.program.Options().Module
	resolved := recv.program.GetResolvedModule(sf, moduleName, mode)
	if resolved == nil || resolved.ResolvedFileName == "" {
		return nil
	}
	return recv.program.GetSourceFile(resolved.ResolvedFileName)
}
