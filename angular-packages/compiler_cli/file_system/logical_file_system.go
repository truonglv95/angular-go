package file_system

import (
	"path/filepath"
	"sort"
	"strings"
)

type LogicalProjectPath string

func LogicalProjectPathBetween(o PathManipulation, from, to LogicalProjectPath) PathSegment {
	relativePath := o.Relative(o.Dirname(string(o.Resolve(string(from)))), string(o.Resolve(string(to))))
	return toRelativeImport(relativePath)
}

func toRelativeImport(path string) PathSegment {
	if !strings.HasPrefix(path, ".") {
		path = "./" + path
	}
	return PathSegment(path)
}

type CanonicalFileNameFunc func(fileName string) string

type LogicalFileSystem struct {
	rootDirs             []AbsoluteFsPath
	canonicalRootDirs    []AbsoluteFsPath
	getCanonicalFileName CanonicalFileNameFunc
	cache                map[AbsoluteFsPath]*LogicalProjectPath
}

func NewLogicalFileSystem(rootDirs []AbsoluteFsPath, getCanonicalFileName CanonicalFileNameFunc) *LogicalFileSystem {
	// Make a copy and sort it by length in reverse order (longest first)
	sortedRootDirs := make([]AbsoluteFsPath, len(rootDirs))
	copy(sortedRootDirs, rootDirs)
	sort.Slice(sortedRootDirs, func(i, j int) bool {
		return len(sortedRootDirs[i]) > len(sortedRootDirs[j])
	})

	canonicalRootDirs := make([]AbsoluteFsPath, len(sortedRootDirs))
	for i, dir := range sortedRootDirs {
		canonicalRootDirs[i] = AbsoluteFsPath(getCanonicalFileName(string(dir)))
	}

	return &LogicalFileSystem{
		rootDirs:             sortedRootDirs,
		canonicalRootDirs:    canonicalRootDirs,
		getCanonicalFileName: getCanonicalFileName,
		cache:                make(map[AbsoluteFsPath]*LogicalProjectPath),
	}
}

func (l *LogicalFileSystem) LogicalPathOfFile(physicalFile AbsoluteFsPath, o PathManipulation) *LogicalProjectPath {
	if val, ok := l.cache[physicalFile]; ok {
		return val
	}

	canonicalFilePath := AbsoluteFsPath(l.getCanonicalFileName(string(physicalFile)))
	var logicalFile *LogicalProjectPath

	for i := 0; i < len(l.rootDirs); i++ {
		rootDir := l.rootDirs[i]
		canonicalRootDir := l.canonicalRootDirs[i]
		if isWithinBasePath(canonicalRootDir, canonicalFilePath, o) {
			path := l.createLogicalProjectPath(physicalFile, rootDir)
			logicalFile = &path
			// The logical project does not include any special "node_modules" nested directories.
			if strings.Contains(string(*logicalFile), "/node_modules/") {
				logicalFile = nil
			} else {
				break
			}
		}
	}

	l.cache[physicalFile] = logicalFile
	return logicalFile
}

func (l *LogicalFileSystem) createLogicalProjectPath(file AbsoluteFsPath, rootDir AbsoluteFsPath) LogicalProjectPath {
	logicalPath := stripExtension(string(file)[len(rootDir):])
	if !strings.HasPrefix(logicalPath, "/") {
		logicalPath = "/" + logicalPath
	}
	return LogicalProjectPath(logicalPath)
}

func isWithinBasePath(base AbsoluteFsPath, path AbsoluteFsPath, o PathManipulation) bool {
	return isLocalRelativePath(o.Relative(string(base), string(path)))
}

func isLocalRelativePath(relPath string) bool {
	return !strings.HasPrefix(relPath, "..") && !filepath.IsAbs(relPath)
}

func stripExtension(path string) string {
	ext := filepath.Ext(path)
	if ext != "" {
		return path[:len(path)-len(ext)]
	}
	return path
}
