package file_system

import (
	"fmt"
	"strings"
)

var fs FileSystem = &InvalidFileSystem{}

func GetFileSystem() FileSystem {
	return fs
}

func SetFileSystem(fileSystem FileSystem) {
	fs = fileSystem
}

// Convert the path `path` to an `AbsoluteFsPath`, throwing an error if it's not an absolute path.
func AbsoluteFrom(path string) AbsoluteFsPath {
	if !fs.IsRooted(path) {
		panic(fmt.Sprintf("Internal Error: absoluteFrom(%s): path is not absolute", path))
	}
	return fs.Resolve(path)
}

// Extract an `AbsoluteFsPath` from a `ts.SourceFile`-like object.
func AbsoluteFromSourceFile(sf interface{ FileName() string }) AbsoluteFsPath {
	return fs.Resolve(sf.FileName())
}

// Convert the path `path` to a `PathSegment`, throwing an error if it's not a relative path.
func RelativeFrom(path string) PathSegment {
	normalized := NormalizeSeparators(path)
	if fs.IsRooted(normalized) {
		panic(fmt.Sprintf("Internal Error: relativeFrom(%s): path is not relative", path))
	}
	return PathSegment(normalized)
}

func Dirname(file string) string {
	return fs.Dirname(file)
}

func Join(basePath string, paths ...string) string {
	return fs.Join(basePath, paths...)
}

func Resolve(basePath string, paths ...string) AbsoluteFsPath {
	allPaths := append([]string{basePath}, paths...)
	return fs.Resolve(allPaths...)
}

func IsRoot(path AbsoluteFsPath) bool {
	return fs.IsRoot(path)
}

func IsRooted(path string) bool {
	return fs.IsRooted(path)
}

func Relative(from string, to string) string {
	return fs.Relative(from, to)
}

func Basename(filePath string, extension ...string) PathSegment {
	return fs.Basename(filePath, extension...)
}

func IsLocalRelativePath(relativePath string) bool {
	return !IsRooted(relativePath) && !strings.HasPrefix(relativePath, "..")
}

func ToRelativeImport(relativePath string) string {
	if IsLocalRelativePath(relativePath) {
		return "./" + relativePath
	}
	return relativePath
}
