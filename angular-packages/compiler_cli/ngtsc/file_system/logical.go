package file_system

import (
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
)

type LogicalProjectPath string

func LogicalProjectPathRelativePathBetween(from LogicalProjectPath, to LogicalProjectPath) PathSegment {
	relativePath := Relative(Dirname(string(Resolve(string(from)))), string(Resolve(string(to))))
	return PathSegment(ToRelativeImport(relativePath))
}

type LogicalFileSystem struct {
	rootDirs          []AbsoluteFsPath
	canonicalRootDirs []AbsoluteFsPath
	cache             map[AbsoluteFsPath]*LogicalProjectPath
	getCanonicalFn    func(fileName string) string
}

func NewLogicalFileSystem(rootDirs []AbsoluteFsPath, getCanonicalFn func(fileName string) string) *LogicalFileSystem {
	sortedRootDirs := make([]AbsoluteFsPath, len(rootDirs))
	copy(sortedRootDirs, rootDirs)
	sort.Slice(sortedRootDirs, func(i, j int) bool {
		return len(sortedRootDirs[i]) > len(sortedRootDirs[j])
	})

	canonicalRootDirs := make([]AbsoluteFsPath, len(sortedRootDirs))
	for i, dir := range sortedRootDirs {
		canonicalRootDirs[i] = AbsoluteFsPath(getCanonicalFn(string(dir)))
	}

	return &LogicalFileSystem{
		rootDirs:          sortedRootDirs,
		canonicalRootDirs: canonicalRootDirs,
		cache:             make(map[AbsoluteFsPath]*LogicalProjectPath),
		getCanonicalFn:    getCanonicalFn,
	}
}

func (l *LogicalFileSystem) LogicalPathOfSf(sf ast.SourceFile) *LogicalProjectPath {
	return l.LogicalPathOfFile(AbsoluteFromSourceFile(&sf))
}

func (l *LogicalFileSystem) LogicalPathOfFile(physicalFile AbsoluteFsPath) *LogicalProjectPath {
	if cached, ok := l.cache[physicalFile]; ok {
		return cached
	}

	canonicalFilePath := AbsoluteFsPath(l.getCanonicalFn(string(physicalFile)))
	var logicalFile *LogicalProjectPath

	for i := 0; i < len(l.rootDirs); i++ {
		rootDir := l.rootDirs[i]
		canonicalRootDir := l.canonicalRootDirs[i]
		if isWithinBasePath(canonicalRootDir, canonicalFilePath) {
			path := l.createLogicalProjectPath(physicalFile, rootDir)
			if strings.Contains(string(path), "/node_modules/") {
				logicalFile = nil
			} else {
				logicalFile = &path
				break
			}
		}
	}

	l.cache[physicalFile] = logicalFile
	return logicalFile
}

func (l *LogicalFileSystem) createLogicalProjectPath(file AbsoluteFsPath, rootDir AbsoluteFsPath) LogicalProjectPath {
	logicalPath := StripExtension(string(file)[len(rootDir):])
	if strings.HasPrefix(logicalPath, "/") {
		return LogicalProjectPath(logicalPath)
	}
	return LogicalProjectPath("/" + logicalPath)
}

func isWithinBasePath(base AbsoluteFsPath, path AbsoluteFsPath) bool {
	return IsLocalRelativePath(Relative(string(base), string(path)))
}
