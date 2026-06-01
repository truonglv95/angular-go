package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/internal/ast"
)

type LinkerEnvironment struct {
	Host AstHost
}

func NewLinkerEnvironment(host AstHost) LinkerEnvironment {
	if host == nil {
		host = NewTypeScriptAstHost()
	}
	return LinkerEnvironment{Host: host}
}

// FileLinker links all partial declarations found in a single file. The AST
// rewrite/insertion layer is intentionally kept outside this type; callers pass
// the declaration call they want linked and receive a compiler output AST.
type FileLinker struct {
	environment    LinkerEnvironment
	linkerSelector *PartialLinkerSelector
	sourceURL      string
	code           string
}

func NewFileLinker(environment LinkerEnvironment, sourceURL string, code string) *FileLinker {
	return &FileLinker{
		environment:    environment,
		linkerSelector: NewPartialLinkerSelector(sourceURL, code),
		sourceURL:      sourceURL,
		code:           code,
	}
}

func (l *FileLinker) IsPartialDeclaration(calleeName string) bool {
	return l.linkerSelector.SupportsDeclaration(calleeName)
}

func (l *FileLinker) LinkPartialDeclaration(declarationFn string, args []*ast.Node, constantPool render3.ConstantPool) (LinkedDefinition, error) {
	if len(args) != 1 {
		return LinkedDefinition{}, linkerError(nil, "Invalid function call: expected exactly one object literal argument, got %d.", len(args))
	}
	metaObj, err := ParseAstObject(args[0], l.environment.Host)
	if err != nil {
		return LinkedDefinition{}, err
	}
	minVersion, err := metaObj.GetString("minVersion")
	if err != nil {
		return LinkedDefinition{}, err
	}
	version, err := metaObj.GetString("version")
	if err != nil {
		return LinkedDefinition{}, err
	}
	linker, err := l.linkerSelector.GetLinker(declarationFn, minVersion, version)
	if err != nil {
		return LinkedDefinition{}, err
	}
	return linker.LinkPartialDeclaration(constantPool, metaObj, version)
}

func (l *FileLinker) SourceURL() string {
	return l.sourceURL
}

func (l *FileLinker) Code() string {
	return l.code
}
