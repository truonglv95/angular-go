package util

import (
	"fmt"
	"path"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
)

// CompilerHostLike is the minimal interface needed by GetRootDirs.
// It mirrors the Pick<ts.CompilerHost, 'getCurrentDirectory' | 'getCanonicalFileName'> in TS.
type CompilerHostLike interface {
	GetCurrentDirectory() string
	GetCanonicalFileName(fileName string) string
}

// GetRootDirs returns the resolved root directories from the compiler options.
// It mirrors the TypeScript getRootDirs() in ngtsc/util/src/typescript.ts.
// Relative paths are resolved against the current directory of the host.
func GetRootDirs(host CompilerHostLike, options *core.CompilerOptions) []string {
	var rootDirs []string
	cwd := host.GetCurrentDirectory()

	if len(options.RootDirs) > 0 {
		rootDirs = append(rootDirs, options.RootDirs...)
	} else if options.RootDir != "" {
		rootDirs = append(rootDirs, options.RootDir)
	} else {
		rootDirs = append(rootDirs, cwd)
	}

	// Resolve each root dir against cwd, applying canonical file name transform.
	result := make([]string, len(rootDirs))
	for i, rootDir := range rootDirs {
		canonical := host.GetCanonicalFileName(rootDir)
		if path.IsAbs(canonical) {
			result[i] = path.Clean(canonical)
		} else {
			result[i] = path.Clean(path.Join(cwd, canonical))
		}
	}
	return result
}

func IsDtsPath(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".d.ts")
}

func IsNonDeclarationTsPath(filePath string) bool {
	lower := strings.ToLower(filePath)
	return (strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx")) && !IsDtsPath(filePath)
}

func IsFromDtsFile(node *ast.Node) bool {
	sf := ast.GetSourceFileOfNode(node)
	return sf != nil && sf.IsDeclarationFile
}

func NodeNameForError(node *ast.Node) string {
	if node == nil {
		return "unknown"
	}
	sf := ast.GetSourceFileOfNode(node)
	if sf != nil {
		return fmt.Sprintf("%v@0:0", node.Kind)
	}
	return fmt.Sprintf("%v", node.Kind)
}

func GetSourceFile(node *ast.Node) *ast.SourceFile {
	return ast.GetSourceFileOfNode(node)
}

type Program interface {
	GetSourceFile(fileName string) *ast.SourceFile
}

func GetSourceFileOrNull(program Program, fileName string) *ast.SourceFile {
	return program.GetSourceFile(fileName)
}

func IsDeclaration(node *ast.Node) bool {
	return IsValueDeclaration(node) || IsTypeDeclaration(node)
}

func IsValueDeclaration(node *ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case ast.KindClassDeclaration, ast.KindFunctionDeclaration, ast.KindVariableDeclaration:
		return true
	}
	return false
}

func IsTypeDeclaration(node *ast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case ast.KindEnumDeclaration, ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration:
		return true
	}
	return false
}

func IsNamedDeclaration(node *ast.Node) bool {
	return false
}

func IsExported(node *ast.Node) bool {
	return false
}

func NodeDebugInfo(node *ast.Node) string {
	sf := ast.GetSourceFileOfNode(node)
	if sf != nil {
		return fmt.Sprintf("[%s: %v @ 0:0]", sf.FileName(), node.Kind)
	}
	return fmt.Sprintf("[%v]", node.Kind)
}

func IsAssignment(node *ast.Node) bool {
	// TS-Go doesn't export BinaryExpression easily through type cast if it's node.data
	// For now we'll do:
	if node.Kind == ast.KindBinaryExpression {
		return true // approximate
	}
	return false
}

func ToUnredirectedSourceFile(sf *ast.SourceFile) *ast.SourceFile {
	return sf
}
