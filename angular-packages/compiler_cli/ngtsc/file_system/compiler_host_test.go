package file_system_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/stretchr/testify/assert"
)

func TestNgtscCompilerHost(t *testing.T) {
	for _, caseSensitive := range []bool{true, false} {
		t.Run(string(map[bool]string{true: "case-sensitive", false: "case-insensitive"}[caseSensitive]), func(t *testing.T) {
			fs := file_system.NewMockFileSystem(caseSensitive)
			directory := fs.Resolve("/a/b/c")
			fs.EnsureDir(directory)

			host := file_system.NewNgtscCompilerHost(fs, nil)

			// fileExists() should return false for an existing directory
			assert.False(t, host.FileExists(string(directory)))

			// readFile() should return empty string for an existing directory
			assert.Equal(t, "", host.ReadFile(string(directory)))

			// getSourceFile() should return a zero-value ast.SourceFile (with empty FileName) for an existing directory
			sourceFile := host.GetSourceFile(string(directory), nil, nil, false)
			assert.Equal(t, "", sourceFile.FileName())

			// useCaseSensitiveFileNames() should return the same as fs.IsCaseSensitive()
			assert.Equal(t, fs.IsCaseSensitive(), host.UseCaseSensitiveFileNames())

			// getCanonicalFileName() should return original filename if FS is case-sensitive or lower case otherwise
			if fs.IsCaseSensitive() {
				assert.Equal(t, "AbCd.ts", host.GetCanonicalFileName("AbCd.ts"))
			} else {
				assert.Equal(t, "abcd.ts", host.GetCanonicalFileName("AbCd.ts"))
			}
		})
	}
}
