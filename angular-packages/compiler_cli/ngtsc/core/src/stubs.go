package src

type CompilerHost interface {
	GetSourceFile(fileName string, languageVersion any, onError any, shouldCreateNewSourceFile any) any
	FileExists(fileName string) bool
}
type Program = any
type CustomTransformers = any
type CompilerOptions = any
type CancellationToken = any
