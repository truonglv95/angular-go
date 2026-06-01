package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type LinkedDefinition struct {
	Expression output.Expression
	Statements []output.Statement
}

type PartialLinker interface {
	LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error)
}

type PartialLinkerSelector struct {
	linkers                           map[string]PartialLinker
	UnknownDeclarationVersionHandling string
}

func NewPartialLinkerSelector(sourceURL string, code string) *PartialLinkerSelector {
	return &PartialLinkerSelector{
		UnknownDeclarationVersionHandling: "error",
		linkers: map[string]PartialLinker{
			DeclareDirective:  NewPartialDirectiveLinkerVersion1(sourceURL, code),
			DeclareComponent:  NewPartialComponentLinkerVersion1(sourceURL, code),
			DeclarePipe:       NewPartialPipeLinkerVersion1(),
			DeclareFactory:    NewPartialFactoryLinkerVersion1(),
			DeclareInjectable: NewPartialInjectableLinkerVersion1(),
			DeclareInjector:   NewPartialInjectorLinkerVersion1(),
			DeclareNgModule:   NewPartialNgModuleLinkerVersion1(),
			DeclareClassMetadata:      &DropPartialLinker{},
			DeclareClassMetadataAsync: &DropPartialLinker{},
			DeclareService: &UnsupportedPartialLinker{Name: DeclareService},
		},
	}
}

func (s *PartialLinkerSelector) SupportsDeclaration(functionName string) bool {
	_, ok := s.linkers[functionName]
	return ok
}

func (s *PartialLinkerSelector) GetLinker(functionName string, minVersion string, version string) (PartialLinker, error) {
	linker, ok := s.linkers[functionName]
	if !ok {
		return nil, linkerError(nil, "Unknown partial declaration function %s.", functionName)
	}
	// Version range dispatch is intentionally centralized here. There is only a
	// v1 linker in this port right now, so newer declarations fall through to v1
	// exactly where future range entries should be added.
	_ = minVersion
	_ = version
	return linker, nil
}

type UnsupportedPartialLinker struct {
	Name string
}

func (l *UnsupportedPartialLinker) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	return LinkedDefinition{}, linkerError(nil, "Partial linker %s is not implemented yet.", l.Name)
}

type DropPartialLinker struct{}

func (l *DropPartialLinker) LinkPartialDeclaration(constantPool render3.ConstantPool, metaObj *AstObject, version string) (LinkedDefinition, error) {
	return LinkedDefinition{Expression: nil}, nil
}
