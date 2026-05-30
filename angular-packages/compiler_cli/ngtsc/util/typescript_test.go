// Package util_test ports typescript_spec.ts from ngtsc 1:1.
package util_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/util"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/stretchr/testify/assert"
)

// mockCompilerHost is a minimal implementation of CompilerHostLike for testing.
type mockCompilerHost struct {
	canonicalFileName func(val string) string
	currentDirectory  func() string
}

func (h *mockCompilerHost) GetCanonicalFileName(val string) string {
	return h.canonicalFileName(val)
}

func (h *mockCompilerHost) GetCurrentDirectory() string {
	return h.currentDirectory()
}

// TestGetRootDirs_RelativeRootDirectory mirrors:
// it('should allow relative root directories', ...)
func TestGetRootDirs_RelativeRootDirectory(t *testing.T) {
	host := &mockCompilerHost{
		canonicalFileName: func(val string) string { return val },
		currentDirectory:  func() string { return "/fs-root/projects" },
	}
	result := util.GetRootDirs(host, &core.CompilerOptions{
		RootDir: "./test-project-root",
	})
	// fs.resolve('/fs-root/projects', './test-project-root') = '/fs-root/projects/test-project-root'
	assert.Equal(t, []string{"/fs-root/projects/test-project-root"}, result)
}
