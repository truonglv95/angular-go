package file_system_test

import (
	"runtime"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/stretchr/testify/assert"
)

func TestHelpers_AbsoluteFrom(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})

	// Should not panic on absolute path
	assert.NotPanics(t, func() {
		file_system.AbsoluteFrom("/test.txt")
	})

	if runtime.GOOS == "windows" {
		assert.Equal(t, file_system.AbsoluteFsPath("C:/test.txt"), file_system.AbsoluteFrom("C:\\test.txt"))
		assert.Equal(t, file_system.AbsoluteFsPath("C:/test.txt"), file_system.AbsoluteFrom("C:/test.txt"))
		assert.Equal(t, file_system.AbsoluteFsPath("D:/foo/test.txt"), file_system.AbsoluteFrom("D:\\foo\\test.txt"))
	}

	// Should panic on relative path
	assert.Panics(t, func() {
		file_system.AbsoluteFrom("test.txt")
	})
}

func TestHelpers_RelativeFrom(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})

	// Should not panic on relative path
	assert.NotPanics(t, func() {
		file_system.RelativeFrom("a/b/c.txt")
	})

	// Should panic on absolute path
	assert.Panics(t, func() {
		file_system.RelativeFrom("/a/b/c.txt")
	})

	if runtime.GOOS == "windows" {
		assert.Panics(t, func() {
			file_system.RelativeFrom("C:/a/b/c.txt")
		})
	}
}
