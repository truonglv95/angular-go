package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type PartialInjectableLinkerVersion1 struct {
}

func NewPartialInjectableLinkerVersion1() *PartialInjectableLinkerVersion1 {
	return &PartialInjectableLinkerVersion1{}
}

func (l *PartialInjectableLinkerVersion1) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	meta, err := l.ToR3InjectableMeta(metaObj, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	compiled := render3.CompileInjectableFromMetadata(meta)
	return linkedDefinitionFromCompiled(compiled), nil
}

func (l *PartialInjectableLinkerVersion1) ToR3InjectableMeta(metaObj *AstObject, version string) (render3.R3InjectableMetadata, error) {
	typeExpr, err := metaObj.GetOpaque("type")
	if err != nil {
		return render3.R3InjectableMetadata{}, err
	}

	name := "unknown"

	var providedIn output.Expression
	if metaObj.Has("providedIn") {
		providedIn, _ = metaObj.GetOpaque("providedIn")
	}

	getOpaque := func(prop string) output.Expression {
		if metaObj.Has(prop) {
			val, _ := metaObj.GetOpaque(prop)
			return val
		}
		return nil
	}

	return render3.R3InjectableMetadata{
		Name:        name,
		Type:        render3.R3Reference{Value: typeExpr},
		ProvidedIn:  providedIn,
		UseClass:    getOpaque("useClass"),
		UseFactory:  getOpaque("useFactory"),
		UseExisting: getOpaque("useExisting"),
		UseValue:    getOpaque("useValue"),
	}, nil
}
