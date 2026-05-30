package util

import (
	"path/filepath"
	"strings"
)

// RelativePathBetween attempts to find a relative path between two paths.
func RelativePathBetween(from, to string) *string {
	// Equivalent to stripExtension(relative(dirname(resolve(from)), resolve(to)))
	// Since we don't have the exact file_system module from Angular here, we mock the logic.
	fromDir := filepath.Dir(from)
	rel, err := filepath.Rel(fromDir, to)
	if err != nil {
		return nil
	}

	ext := filepath.Ext(rel)
	relNoExt := strings.TrimSuffix(rel, ext)

	if relNoExt != "" {
		// toRelativeImport logic: prepend ./ if not starting with . or /
		if !strings.HasPrefix(relNoExt, ".") && !strings.HasPrefix(relNoExt, "/") {
			relNoExt = "./" + relNoExt
		}
		return &relNoExt
	}
	return nil
}

func NormalizeSeparators(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

type CompilerHost interface {
	GetCanonicalFileName(fileName string) string
}

func GetProjectRelativePath(fileName string, rootDirs []string, compilerHost CompilerHost) *string {
	filePath := compilerHost.GetCanonicalFileName(fileName)

	for _, rootDir := range rootDirs {
		canonRootDir := compilerHost.GetCanonicalFileName(rootDir)
		rel, err := filepath.Rel(canonRootDir, filePath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return &rel
		}
	}

	return nil
}
