package file_system

import (
	"errors"
)

// The default `FileSystem` that will always fail.
type InvalidFileSystem struct{}

func makeError() error {
	return errors.New("FileSystem has not been configured. Please call `SetFileSystem()` before calling this method.")
}

func (i *InvalidFileSystem) Exists(path AbsoluteFsPath) bool {
	panic(makeError())
}

func (i *InvalidFileSystem) ReadFile(path AbsoluteFsPath) string {
	panic(makeError())
}

func (i *InvalidFileSystem) ReadFileBuffer(path AbsoluteFsPath) []byte {
	panic(makeError())
}

func (i *InvalidFileSystem) WriteFile(path AbsoluteFsPath, data []byte, exclusive bool) {
	panic(makeError())
}

func (i *InvalidFileSystem) RemoveFile(path AbsoluteFsPath) {
	panic(makeError())
}

func (i *InvalidFileSystem) Symlink(target AbsoluteFsPath, path AbsoluteFsPath) {
	panic(makeError())
}

func (i *InvalidFileSystem) Readdir(path AbsoluteFsPath) []PathSegment {
	panic(makeError())
}

func (i *InvalidFileSystem) Lstat(path AbsoluteFsPath) FileStats {
	panic(makeError())
}

func (i *InvalidFileSystem) Stat(path AbsoluteFsPath) FileStats {
	panic(makeError())
}

func (i *InvalidFileSystem) Pwd() AbsoluteFsPath {
	panic(makeError())
}

func (i *InvalidFileSystem) Chdir(path AbsoluteFsPath) {
	panic(makeError())
}

func (i *InvalidFileSystem) Extname(path string) string {
	panic(makeError())
}

func (i *InvalidFileSystem) CopyFile(from AbsoluteFsPath, to AbsoluteFsPath) {
	panic(makeError())
}

func (i *InvalidFileSystem) MoveFile(from AbsoluteFsPath, to AbsoluteFsPath) {
	panic(makeError())
}

func (i *InvalidFileSystem) EnsureDir(path AbsoluteFsPath) {
	panic(makeError())
}

func (i *InvalidFileSystem) RemoveDeep(path AbsoluteFsPath) {
	panic(makeError())
}

func (i *InvalidFileSystem) IsCaseSensitive() bool {
	panic(makeError())
}

func (i *InvalidFileSystem) Resolve(paths ...string) AbsoluteFsPath {
	panic(makeError())
}

func (i *InvalidFileSystem) Dirname(file string) string {
	panic(makeError())
}

func (i *InvalidFileSystem) Join(basePath string, paths ...string) string {
	panic(makeError())
}

func (i *InvalidFileSystem) IsRoot(path AbsoluteFsPath) bool {
	panic(makeError())
}

func (i *InvalidFileSystem) IsRooted(path string) bool {
	panic(makeError())
}

func (i *InvalidFileSystem) Relative(from string, to string) string {
	panic(makeError())
}

func (i *InvalidFileSystem) Basename(filePath string, extension ...string) PathSegment {
	panic(makeError())
}

func (i *InvalidFileSystem) Realpath(filePath AbsoluteFsPath) AbsoluteFsPath {
	panic(makeError())
}

func (i *InvalidFileSystem) GetDefaultLibLocation() AbsoluteFsPath {
	panic(makeError())
}

func (i *InvalidFileSystem) Normalize(path string) string {
	panic(makeError())
}
