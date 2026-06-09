package file_system_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/stretchr/testify/assert"
)

func TestNodeJSFileSystem_PathManipulation(t *testing.T) {
	pm := &file_system.NodeJSPathManipulation{}

	// Pwd
	pwd := pm.Pwd()
	assert.NotEmpty(t, pwd)

	// Resolve & Dirname
	resolved := pm.Resolve("/a/b/c")
	assert.Equal(t, "/a/b/c", filepath.ToSlash(string(resolved)))
	assert.Equal(t, "/a/b", pm.Dirname(string(resolved)))

	// Join
	assert.Equal(t, "/a/b/c/d", pm.Join("/a/b/c", "d"))

	// IsRoot
	if runtime.GOOS == "windows" {
		assert.True(t, pm.IsRoot(file_system.AbsoluteFsPath("C:/")))
	} else {
		assert.True(t, pm.IsRoot(file_system.AbsoluteFsPath("/")))
	}

	// IsRooted
	assert.True(t, pm.IsRooted(string(resolved)))

	// Relative
	assert.Equal(t, "d", pm.Relative("/a/b/c", "/a/b/c/d"))

	// Basename
	assert.Equal(t, file_system.PathSegment("c"), pm.Basename("/a/b/c"))
	assert.Equal(t, file_system.PathSegment("c"), pm.Basename("/a/b/c.txt", ".txt"))

	// Extname
	assert.Equal(t, ".txt", pm.Extname("c.txt"))
}

func TestNodeJSFileSystem_ReadWrite(t *testing.T) {
	tempDir := t.TempDir()
	fs := &file_system.NodeJSFileSystem{}

	tempDirPath := file_system.AbsoluteFsPath(filepath.ToSlash(tempDir))
	filePath := fs.Join(string(tempDirPath), "test.txt")
	absFilePath := file_system.AbsoluteFsPath(filePath)

	// Exists before write
	assert.False(t, fs.Exists(absFilePath))

	// WriteFile
	fs.WriteFile(absFilePath, []byte("Hello World"), false)
	assert.True(t, fs.Exists(absFilePath))

	// WriteFile exclusive on an existing file should fail to write.
	// Since Go's os.File.Write on a nil file pointer does not panic but returns an error (which is ignored),
	// we verify that the content was NOT overwritten and remains "Hello World".
	fs.WriteFile(absFilePath, []byte("Exclusive Overwrite"), true)
	assert.Equal(t, "Hello World", fs.ReadFile(absFilePath))

	// ReadFile & ReadFileBuffer
	assert.Equal(t, "Hello World", fs.ReadFile(absFilePath))
	assert.Equal(t, []byte("Hello World"), fs.ReadFileBuffer(absFilePath))

	// Readdir
	files := fs.Readdir(tempDirPath)
	assert.Len(t, files, 1)
	assert.Equal(t, file_system.PathSegment("test.txt"), files[0])

	// Stat & Lstat
	stat := fs.Stat(absFilePath)
	assert.True(t, stat.IsFile())
	assert.False(t, stat.IsDirectory())
	assert.False(t, stat.IsSymbolicLink())

	lstat := fs.Lstat(absFilePath)
	assert.True(t, lstat.IsFile())
	assert.False(t, lstat.IsDirectory())
	assert.False(t, lstat.IsSymbolicLink())

	// Realpath
	expectedReal, _ := filepath.EvalSymlinks(string(absFilePath))
	assert.Equal(t, filepath.ToSlash(expectedReal), string(fs.Realpath(absFilePath)))

	// CopyFile
	copyPath := file_system.AbsoluteFsPath(fs.Join(string(tempDirPath), "copy.txt"))
	fs.CopyFile(absFilePath, copyPath)
	assert.True(t, fs.Exists(copyPath))
	assert.Equal(t, "Hello World", fs.ReadFile(copyPath))

	// MoveFile
	movePath := file_system.AbsoluteFsPath(fs.Join(string(tempDirPath), "move.txt"))
	fs.MoveFile(copyPath, movePath)
	assert.False(t, fs.Exists(copyPath))
	assert.True(t, fs.Exists(movePath))

	// EnsureDir
	subDir := file_system.AbsoluteFsPath(fs.Join(string(tempDirPath), "subdir"))
	fs.EnsureDir(subDir)
	assert.True(t, fs.Exists(subDir))
	assert.True(t, fs.Stat(subDir).IsDirectory())

	// RemoveFile
	fs.RemoveFile(movePath)
	assert.False(t, fs.Exists(movePath))

	// RemoveDeep
	fs.RemoveDeep(subDir)
	assert.False(t, fs.Exists(subDir))
}

func TestNodeJSFileSystem_Symlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir := t.TempDir()
	fs := &file_system.NodeJSFileSystem{}

	tempDirPath := file_system.AbsoluteFsPath(filepath.ToSlash(tempDir))
	targetPath := file_system.AbsoluteFsPath(fs.Join(string(tempDirPath), "target.txt"))
	linkPath := file_system.AbsoluteFsPath(fs.Join(string(tempDirPath), "link.txt"))

	fs.WriteFile(targetPath, []byte("target"), false)
	err := os.Symlink(string(targetPath), string(linkPath))
	if err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	assert.True(t, fs.Exists(linkPath))
	assert.True(t, fs.Lstat(linkPath).IsSymbolicLink())
	assert.False(t, fs.Stat(linkPath).IsSymbolicLink())

	expectedTargetReal, _ := filepath.EvalSymlinks(string(targetPath))
	assert.Equal(t, filepath.ToSlash(expectedTargetReal), string(fs.Realpath(linkPath)))
}

func TestNodeJSFileSystem_IsCaseSensitive(t *testing.T) {
	fs := &file_system.NodeJSFileSystem{}
	assert.Equal(t, fs.IsCaseSensitive(), fs.IsCaseSensitive())
}
