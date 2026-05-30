package entry_point

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/util"
)

func FindFlatIndexEntryPoint(rootFiles []file_system.AbsoluteFsPath) *file_system.AbsoluteFsPath {
	var tsFiles []file_system.AbsoluteFsPath
	for _, file := range rootFiles {
		if util.IsNonDeclarationTsPath(string(file)) {
			tsFiles = append(tsFiles, file)
		}
	}

	var resolvedEntryPoint *file_system.AbsoluteFsPath

	if len(tsFiles) == 1 {
		resolvedEntryPoint = &tsFiles[0]
	} else {
		for _, tsFile := range tsFiles {
			if file_system.GetFileSystem().Basename(string(tsFile)) == "index.ts" &&
				(resolvedEntryPoint == nil || len(tsFile) <= len(*resolvedEntryPoint)) {
				// We need a local variable to take its address safely in loop,
				// but tsFile is passed by value (it's an alias to string).
				f := tsFile
				resolvedEntryPoint = &f
			}
		}
	}

	return resolvedEntryPoint
}
