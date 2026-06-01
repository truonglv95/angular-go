package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type PartialPipeLinkerVersion1 struct{}

func NewPartialPipeLinkerVersion1() *PartialPipeLinkerVersion1 {
	return &PartialPipeLinkerVersion1{}
}

func (l *PartialPipeLinkerVersion1) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	_ = constantPool
	meta, err := ToR3PipeMeta(metaObj, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	compiled := render3.CompilePipeFromMetadata(meta)
	return linkedDefinitionFromCompiled(compiled), nil
}

func ToR3PipeMeta(metaObj *AstObject, version string) (render3.R3PipeMetadata, error) {
	typeValue, err := metaObj.GetValue("type")
	if err != nil {
		return render3.R3PipeMetadata{}, err
	}
	typeName := typeValue.GetSymbolName()
	if typeName == "" {
		return render3.R3PipeMetadata{}, linkerError(typeValue.Node(), "Unsupported type, its name could not be determined.")
	}
	pipeName, err := metaObj.GetString("name")
	if err != nil {
		return render3.R3PipeMetadata{}, err
	}
	pure := true
	if metaObj.Has("pure") {
		pure, err = metaObj.GetBoolean("pure")
		if err != nil {
			return render3.R3PipeMetadata{}, err
		}
	}
	isStandalone := getDefaultStandaloneValue(version)
	if metaObj.Has("isStandalone") {
		isStandalone, err = metaObj.GetBoolean("isStandalone")
		if err != nil {
			return render3.R3PipeMetadata{}, err
		}
	}
	return render3.R3PipeMetadata{
		Name: typeName,
		Type: render3.R3Reference{
			Value: output.NewWrappedNodeExpr(typeValue.Node(), nil, nil, nil),
		},
		TypeArgumentCount: 0,
		PipeName:          &pipeName,
		Deps:              nil,
		Pure:              pure,
		IsStandalone:      isStandalone,
	}, nil
}
