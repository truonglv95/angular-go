package entry_point_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/entry_point"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/stretchr/testify/assert"
)

func TestFindFlatIndexEntryPoint_SingleFile(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})
	// should use the only source file if only a single one is specified
	input := []file_system.AbsoluteFsPath{
		file_system.AbsoluteFsPath("/src/index.ts"),
	}
	res := entry_point.FindFlatIndexEntryPoint(input)
	assert.NotNil(t, res)
	assert.Equal(t, file_system.AbsoluteFsPath("/src/index.ts"), *res)
}

func TestFindFlatIndexEntryPoint_MultipleFiles(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})
	// should use the shortest source file ending with "index.ts" for multiple files
	input := []file_system.AbsoluteFsPath{
		file_system.AbsoluteFsPath("/src/deep/index.ts"),
		file_system.AbsoluteFsPath("/src/index.ts"),
		file_system.AbsoluteFsPath("/index.ts"),
	}
	res := entry_point.FindFlatIndexEntryPoint(input)
	assert.NotNil(t, res)
	assert.Equal(t, file_system.AbsoluteFsPath("/index.ts"), *res)
}
