package file_system

// A fully qualified path in the file system, in POSIX form.
type AbsoluteFsPath string

// A path that's relative to another (unspecified) root.
// This does not necessarily have to refer to a physical file.
type PathSegment string

// An abstraction over the path manipulation aspects of a file-system.
type PathManipulation interface {
	Extname(path string) string
	IsRoot(path AbsoluteFsPath) bool
	IsRooted(path string) bool
	Dirname(file string) string
	Join(basePath string, paths ...string) string

	// Compute the relative path between `from` and `to`.
	Relative(from string, to string) string
	Basename(filePath string, extension ...string) PathSegment
	Normalize(path string) string
	Resolve(paths ...string) AbsoluteFsPath
	Pwd() AbsoluteFsPath
	Chdir(path AbsoluteFsPath)
}

// Information about an object in the FileSystem.
type FileStats interface {
	IsFile() bool
	IsDirectory() bool
	IsSymbolicLink() bool
}

// An abstraction over the read-only aspects of a file-system.
type ReadonlyFileSystem interface {
	PathManipulation
	IsCaseSensitive() bool
	Exists(path AbsoluteFsPath) bool
	ReadFile(path AbsoluteFsPath) string
	ReadFileBuffer(path AbsoluteFsPath) []byte
	Readdir(path AbsoluteFsPath) []PathSegment
	Lstat(path AbsoluteFsPath) FileStats
	Stat(path AbsoluteFsPath) FileStats
	Realpath(filePath AbsoluteFsPath) AbsoluteFsPath
	GetDefaultLibLocation() AbsoluteFsPath
}

// A basic interface to abstract the underlying file-system.
type FileSystem interface {
	ReadonlyFileSystem
	WriteFile(path AbsoluteFsPath, data []byte, exclusive bool)
	RemoveFile(path AbsoluteFsPath)
	Symlink(target AbsoluteFsPath, path AbsoluteFsPath)
	CopyFile(from AbsoluteFsPath, to AbsoluteFsPath)
	MoveFile(from AbsoluteFsPath, to AbsoluteFsPath)
	EnsureDir(path AbsoluteFsPath)
	RemoveDeep(path AbsoluteFsPath)
}
