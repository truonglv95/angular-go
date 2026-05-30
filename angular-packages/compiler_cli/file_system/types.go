package file_system

// AbsoluteFsPath is a string representing a fully qualified path in the file system, in POSIX form.
type AbsoluteFsPath string

// PathSegment is a string representing a path that's relative to another (unspecified) root.
type PathSegment string

// FileStats provides information about an object in the FileSystem.
type FileStats interface {
	IsFile() bool
	IsDirectory() bool
	IsSymbolicLink() bool
}

// PathManipulation is an abstraction over the path manipulation aspects of a file-system.
type PathManipulation interface {
	Extname(path string) string
	IsRoot(path AbsoluteFsPath) bool
	IsRooted(path string) bool
	Dirname(file string) string
	Join(basePath string, paths ...string) string
	Relative(from string, to string) string
	Basename(filePath string, extension string) PathSegment
	Normalize(path string) string
	Resolve(paths ...string) AbsoluteFsPath
	Pwd() AbsoluteFsPath
	Chdir(path AbsoluteFsPath) error
}

// ReadonlyFileSystem is an abstraction over the read-only aspects of a file-system.
type ReadonlyFileSystem interface {
	PathManipulation
	IsCaseSensitive() bool
	Exists(path AbsoluteFsPath) bool
	ReadFile(path AbsoluteFsPath) (string, error)
	ReadFileBuffer(path AbsoluteFsPath) ([]byte, error)
	Readdir(path AbsoluteFsPath) ([]PathSegment, error)
	Lstat(path AbsoluteFsPath) (FileStats, error)
	Stat(path AbsoluteFsPath) (FileStats, error)
	Realpath(filePath AbsoluteFsPath) (AbsoluteFsPath, error)
	GetDefaultLibLocation() AbsoluteFsPath
}

// FileSystem is a basic interface to abstract the underlying file-system.
type FileSystem interface {
	ReadonlyFileSystem
	WriteFile(path AbsoluteFsPath, data []byte, exclusive bool) error
	RemoveFile(path AbsoluteFsPath) error
	Symlink(target AbsoluteFsPath, path AbsoluteFsPath) error
	CopyFile(from AbsoluteFsPath, to AbsoluteFsPath) error
	MoveFile(from AbsoluteFsPath, to AbsoluteFsPath) error
	EnsureDir(path AbsoluteFsPath) error
	RemoveDeep(path AbsoluteFsPath) error
}
