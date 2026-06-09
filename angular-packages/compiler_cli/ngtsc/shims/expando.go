package shims

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

var shimsMap = make(map[string]bool)
var originalReferencedFilesMap = make(map[string][]*ast.FileReference)
var fileShimDataMap = make(map[string]bool)

func IsShim(sf *ast.SourceFile) bool {
	if sf == nil {
		return false
	}
	return shimsMap[sf.FileName()]
}

func SetIsShim(sf *ast.SourceFile, val bool) {
	if sf != nil {
		shimsMap[sf.FileName()] = val
	}
}

func IsFileShimSourceFile(sf *ast.SourceFile) bool {
	if sf == nil {
		return false
	}
	return fileShimDataMap[sf.FileName()]
}

func SetIsFileShimSourceFile(sf *ast.SourceFile, val bool) {
	if sf != nil {
		fileShimDataMap[sf.FileName()] = val
	}
}

func UntagTsFile(sf *ast.SourceFile) {
	if sf == nil {
		return
	}
	if orig, ok := originalReferencedFilesMap[sf.FileName()]; ok {
		sf.ReferencedFiles = orig
	}
}

func RetagTsFile(sf *ast.SourceFile) {
	// Simple stub for RetagTsFile
}

func CopyFileShimData(from *ast.SourceFile, to *ast.SourceFile) any {
	return nil
}

func UntagAllTsFiles(program any) any {
	return nil
}

func RetagAllTsFiles(program any) any {
	return nil
}

func ClearExpandoState() {
	shimsMap = make(map[string]bool)
	originalReferencedFilesMap = make(map[string][]*ast.FileReference)
	fileShimDataMap = make(map[string]bool)
}
