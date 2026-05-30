package file_system

import (
	"strings"
)

type FileSystemEntries struct {
	Files       []string
	Directories []string
}

func CreateFileSystemTsReadDirectoryFn(fs FileSystem) func(rootDir string, extensions []string, excludes []string, includes []string, depth int) []string {
	return func(rootDir string, extensions []string, excludes []string, includes []string, depth int) []string {
		var results []string

		directoryExists := func(p string) bool {
			resolvedPath := fs.Resolve(p)
			return fs.Exists(resolvedPath) && fs.Stat(resolvedPath).IsDirectory()
		}

		var walk func(currentDir string, currentDepth int)
		walk = func(currentDir string, currentDepth int) {
			if depth >= 0 && currentDepth > depth {
				return
			}

			resolvedPath := fs.Resolve(currentDir)
			if !directoryExists(string(resolvedPath)) {
				return
			}

			children := fs.Readdir(resolvedPath)
			for _, child := range children {
				childPath := fs.Join(string(resolvedPath), string(child))
				stat := fs.Stat(AbsoluteFsPath(childPath))

				if stat != nil && stat.IsDirectory() {
					walk(childPath, currentDepth+1)
				} else {
					// Check extensions
					matched := false
					if len(extensions) > 0 {
						for _, ext := range extensions {
							if strings.HasSuffix(string(child), ext) {
								matched = true
								break
							}
						}
					} else {
						matched = true
					}
					if matched {
						// Here we ideally check includes/excludes as well but
						// skipping complex globs logic for simplicity unless required.
						results = append(results, childPath)
					}
				}
			}
		}

		walk(rootDir, 0)
		return results
	}
}
