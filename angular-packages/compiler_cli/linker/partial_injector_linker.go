package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type PartialInjectorLinkerVersion1 struct {
}

func NewPartialInjectorLinkerVersion1() *PartialInjectorLinkerVersion1 {
	return &PartialInjectorLinkerVersion1{}
}

func (l *PartialInjectorLinkerVersion1) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	meta, err := l.ToR3InjectorMeta(metaObj, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	compiled := render3.CompileInjector(meta)
	return linkedDefinitionFromCompiled(compiled), nil
}

func (l *PartialInjectorLinkerVersion1) ToR3InjectorMeta(metaObj *AstObject, version string) (render3.R3InjectorMetadata, error) {
	typeExpr, err := metaObj.GetOpaque("type")
	if err != nil {
		return render3.R3InjectorMetadata{}, err
	}

	name := "unknown"

	var providers output.Expression
	if metaObj.Has("providers") {
		providers, _ = metaObj.GetOpaque("providers")
	}

	var imports []output.Expression
	if metaObj.Has("imports") {
		importsArray, err := metaObj.GetArray("imports")
		if err == nil {
			imports = make([]output.Expression, len(importsArray))
			for i, imp := range importsArray {
				impExpr := imp.GetOpaque()
				imports[i] = impExpr
			}
		}
	}

	return render3.R3InjectorMetadata{
		Name:      name,
		Type:      render3.R3Reference{Value: typeExpr},
		Providers: providers,
		Imports:   imports,
	}, nil
}
