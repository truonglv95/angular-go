package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type PartialNgModuleLinkerVersion1 struct {
}

func NewPartialNgModuleLinkerVersion1() *PartialNgModuleLinkerVersion1 {
	return &PartialNgModuleLinkerVersion1{}
}

func (l *PartialNgModuleLinkerVersion1) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	meta, err := l.ToR3NgModuleMeta(metaObj, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	expr := render3.CompileNgModuleDeclarationExpression(meta)
	compiled := render3.R3CompiledExpression{
		Expression: expr,
		Statements: []output.Statement{},
	}
	return linkedDefinitionFromCompiled(compiled), nil
}

func (l *PartialNgModuleLinkerVersion1) ToR3NgModuleMeta(metaObj *AstObject, version string) (render3.R3NgModuleDeclarationFacade, error) {
	typeExpr, err := metaObj.GetOpaque("type")
	if err != nil {
		return render3.R3NgModuleDeclarationFacade{}, err
	}

	getOpaqueIfExists := func(prop string) interface{} {
		if metaObj.Has(prop) {
			val, _ := metaObj.GetOpaque(prop)
			return val
		}
		return nil
	}

	return render3.R3NgModuleDeclarationFacade{
		Type:         typeExpr,
		Bootstrap:    getOpaqueIfExists("bootstrap"),
		Declarations: getOpaqueIfExists("declarations"),
		Imports:      getOpaqueIfExists("imports"),
		Exports:      getOpaqueIfExists("exports"),
		Schemas:      getOpaqueIfExists("schemas"),
		Id:           getOpaqueIfExists("id"),
	}, nil
}
