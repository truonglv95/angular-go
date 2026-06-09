package file_system_test

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/stretchr/testify/assert"
)

func TestLogicalFileSystem_SingleRoot(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})

	getCanonicalFn := func(fileName string) string { return fileName }

	fs := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{
			file_system.AbsoluteFsPath("/test"),
		},
		getCanonicalFn,
	)

	p1 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/foo/foo.ts"))
	assert.NotNil(t, p1)
	assert.Equal(t, file_system.LogicalProjectPath("/foo/foo"), *p1)

	p2 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/bar/bar.ts"))
	assert.NotNil(t, p2)
	assert.Equal(t, file_system.LogicalProjectPath("/bar/bar"), *p2)

	p3 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/not-test/bar.ts"))
	assert.Nil(t, p3)
}

func TestLogicalFileSystem_MultiRoot(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})

	getCanonicalFn := func(fileName string) string { return fileName }

	fs := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{
			file_system.AbsoluteFsPath("/test/foo"),
			file_system.AbsoluteFsPath("/test/bar"),
		},
		getCanonicalFn,
	)

	p1 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/foo/foo.ts"))
	assert.NotNil(t, p1)
	assert.Equal(t, file_system.LogicalProjectPath("/foo"), *p1)

	p2 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/bar/bar.ts"))
	assert.NotNil(t, p2)
	assert.Equal(t, file_system.LogicalProjectPath("/bar"), *p2)
}

func TestLogicalFileSystem_NestedRoots(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})

	getCanonicalFn := func(fileName string) string { return fileName }

	fs := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{
			file_system.AbsoluteFsPath("/test"),
			file_system.AbsoluteFsPath("/test/dist"),
		},
		getCanonicalFn,
	)

	p1 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/foo.ts"))
	assert.NotNil(t, p1)
	assert.Equal(t, file_system.LogicalProjectPath("/foo"), *p1)

	p2 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/dist/foo.ts"))
	assert.NotNil(t, p2)
	assert.Equal(t, file_system.LogicalProjectPath("/foo"), *p2)
}

func TestLogicalFileSystem_AlwaysReturnSlashPrefixed(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})
	getCanonicalFn := func(fileName string) string { return fileName }

	rootFs := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{file_system.AbsoluteFsPath("/")},
		getCanonicalFn,
	)
	p1 := rootFs.LogicalPathOfFile(file_system.AbsoluteFsPath("/foo/foo.ts"))
	assert.NotNil(t, p1)
	assert.Equal(t, file_system.LogicalProjectPath("/foo/foo"), *p1)

	nonRootFs := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{file_system.AbsoluteFsPath("/test/")},
		getCanonicalFn,
	)
	p2 := nonRootFs.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/foo/foo.ts"))
	assert.NotNil(t, p2)
	assert.Equal(t, file_system.LogicalProjectPath("/foo/foo"), *p2)
}

func TestLogicalFileSystem_MaintainCasing(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})
	getCanonicalFn := func(fileName string) string { return fileName }

	fs := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{file_system.AbsoluteFsPath("/Test")},
		getCanonicalFn,
	)

	p1 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/Test/foo/Foo.ts"))
	assert.NotNil(t, p1)
	assert.Equal(t, file_system.LogicalProjectPath("/foo/Foo"), *p1)

	p2 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/Test/foo/foo.ts"))
	assert.NotNil(t, p2)
	assert.Equal(t, file_system.LogicalProjectPath("/foo/foo"), *p2)

	p3 := fs.LogicalPathOfFile(file_system.AbsoluteFsPath("/Test/bar/bAR.ts"))
	assert.NotNil(t, p3)
	assert.Equal(t, file_system.LogicalProjectPath("/bar/bAR"), *p3)
}

func TestLogicalFileSystem_CaseSensitivityMatching(t *testing.T) {
	file_system.SetFileSystem(&file_system.NodeJSFileSystem{})

	// Case-sensitive matching
	fsSensitive := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{file_system.AbsoluteFsPath("/Test")},
		func(fileName string) string { return fileName },
	)
	assert.Nil(t, fsSensitive.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/car/CAR.ts")))

	// Case-insensitive matching
	fsInsensitive := file_system.NewLogicalFileSystem(
		[]file_system.AbsoluteFsPath{file_system.AbsoluteFsPath("/Test")},
		func(fileName string) string { return strings.ToLower(fileName) },
	)
	p := fsInsensitive.LogicalPathOfFile(file_system.AbsoluteFsPath("/test/car/CAR.ts"))
	assert.NotNil(t, p)
	assert.Equal(t, file_system.LogicalProjectPath("/car/CAR"), *p)
}

func TestLogicalFileSystem_RelativePathBetween(t *testing.T) {
	r1 := file_system.LogicalProjectPathRelativePathBetween(
		file_system.LogicalProjectPath("/foo"),
		file_system.LogicalProjectPath("/bar"),
	)
	assert.Equal(t, file_system.PathSegment("./bar"), r1)

	r2 := file_system.LogicalProjectPathRelativePathBetween(
		file_system.LogicalProjectPath("/foo/index"),
		file_system.LogicalProjectPath("/bar/index"),
	)
	assert.Equal(t, file_system.PathSegment("../bar/index"), r2)

	r3 := file_system.LogicalProjectPathRelativePathBetween(
		file_system.LogicalProjectPath("/fOO"),
		file_system.LogicalProjectPath("/bAR"),
	)
	assert.Equal(t, file_system.PathSegment("./bAR"), r3)
}
