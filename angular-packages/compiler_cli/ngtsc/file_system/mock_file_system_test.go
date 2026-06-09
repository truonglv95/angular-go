package file_system_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/stretchr/testify/assert"
)

func TestMockFileSystem_Symlinks(t *testing.T) {
	for _, caseSensitive := range []bool{true, false} {
		t.Run(string(map[bool]string{true: "case-sensitive", false: "case-insensitive"}[caseSensitive]), func(t *testing.T) {
			fs := file_system.NewMockFileSystem(caseSensitive)

			targetPath := fs.Resolve("/link/target")
			symlinkPath := fs.Resolve("/src/symlink")
			packageJsonInTarget := fs.Resolve(string(targetPath), "package.json")
			packageJsonUsingSymlink := fs.Resolve(string(symlinkPath), "package.json")

			fs.EnsureDir(targetPath)
			fs.EnsureDir(fs.Resolve("/src"))
			fs.WriteFile(packageJsonInTarget, []byte("{}"), false)
			fs.Symlink(targetPath, symlinkPath)

			assert.True(t, fs.Exists(packageJsonUsingSymlink))
			assert.False(t, fs.Exists(fs.Resolve(string(symlinkPath), "unknown.json")))
			assert.Equal(t, "{}", fs.ReadFile(packageJsonInTarget))

			assert.Equal(t, packageJsonInTarget, fs.Realpath(packageJsonUsingSymlink))
			assert.Equal(t, packageJsonInTarget, fs.Realpath(packageJsonInTarget))

			assert.False(t, fs.Stat(symlinkPath).IsSymbolicLink())
			assert.True(t, fs.Lstat(symlinkPath).IsSymbolicLink())

			assert.False(t, fs.Stat(packageJsonUsingSymlink).IsSymbolicLink())
			assert.False(t, fs.Lstat(packageJsonUsingSymlink).IsSymbolicLink())
		})
	}
}
