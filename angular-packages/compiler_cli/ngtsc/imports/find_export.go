package imports

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// FindExportedNameOfNode finds the name, if any, by which a node is exported from a given file.
func FindExportedNameOfNode(target *ast.Node, file *ast.SourceFile, reflector reflection.ReflectionHost) *string {
	exports := reflector.GetExportsOfModule(file.AsNode())
	if exports == nil {
		return nil
	}

	var declaredName *string
	if id := identifierOfNode(target); id != nil {
		text := id.AsIdentifier().Text
		declaredName = &text
	}

	var foundExportName *string
	for exportName, declaration := range exports {
		if declaration.Node != target {
			continue
		}

		if declaredName != nil && exportName == *declaredName {
			res := exportName
			return &res
		}

		res := exportName
		foundExportName = &res
	}
	return foundExportName
}
