package file_system

import "github.com/microsoft/typescript-go/internal/ast"

type CompilerHost = interface{}
type CompilerOptions = interface{}
type ScriptTarget = interface{}

var ScriptKind_Unknown = 0

type SourceFile = ast.SourceFile
type Program interface {
	GetSourceFile(string) SourceFile
	GetSourceFiles() []SourceFile
}

func CreateSourceFile(fileName string, sourceText string, languageVersion ScriptTarget, setParentNodes bool, scriptKind int) SourceFile {
	return ast.SourceFile{}
}
func GetDefaultLibFileName(options CompilerOptions) string { return "" }

var NewLineKind_CarriageReturnLineFeed = 0
var NewLineKind_LineFeed = 1
