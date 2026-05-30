package file_system

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
)

var tsDtsTsxJsExtension = regexp.MustCompile(`(?:\.d)?\.ts$|\.tsx$|\.js$`)

// Convert Windows-style separators to POSIX separators.
func NormalizeSeparators(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// Remove a .ts, .d.ts, .tsx, or .js extension from a file name.
func StripExtension(path string) string {
	return tsDtsTsxJsExtension.ReplaceAllString(path, "")
}

func GetSourceFileOrError(program Program, fileName AbsoluteFsPath) ast.SourceFile {
	sf := program.GetSourceFile(string(fileName))
	if sf.FileName() == "" {
		availableFiles := []string{}
		for _, sf := range program.GetSourceFiles() {
			availableFiles = append(availableFiles, sf.FileName())
		}
		panic(fmt.Sprintf("Program does not contain \"%s\" - available files are %s", fileName, strings.Join(availableFiles, ", ")))
	}
	return sf
}
