package file_system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

type NodeJSPathManipulation struct{}

func (p *NodeJSPathManipulation) Pwd() AbsoluteFsPath {
	cwd, _ := os.Getwd()
	return AbsoluteFsPath(p.Normalize(cwd))
}

func (p *NodeJSPathManipulation) Chdir(dir AbsoluteFsPath) {
	os.Chdir(string(dir))
}

func (p *NodeJSPathManipulation) Resolve(paths ...string) AbsoluteFsPath {
	// In Go, filepath.Abs + filepath.Join does a lot of this.
	var joined string
	if len(paths) == 1 && filepath.IsAbs(paths[0]) {
		joined = paths[0]
	} else if len(paths) > 0 {
		joined = filepath.Join(paths...)
		if !filepath.IsAbs(joined) {
			abs, _ := filepath.Abs(joined)
			joined = abs
		}
	} else {
		joined, _ = filepath.Abs(".")
	}
	return AbsoluteFsPath(p.Normalize(joined))
}

func (p *NodeJSPathManipulation) Dirname(file string) string {
	return p.Normalize(filepath.Dir(file))
}

func (p *NodeJSPathManipulation) Join(basePath string, paths ...string) string {
	allPaths := append([]string{basePath}, paths...)
	return p.Normalize(filepath.Join(allPaths...))
}

func (p *NodeJSPathManipulation) IsRoot(path AbsoluteFsPath) bool {
	return p.Dirname(string(path)) == p.Normalize(string(path))
}

func (p *NodeJSPathManipulation) IsRooted(path string) bool {
	return filepath.IsAbs(path)
}

func (p *NodeJSPathManipulation) Relative(from string, to string) string {
	rel, _ := filepath.Rel(from, to)
	return p.Normalize(rel)
}

func (p *NodeJSPathManipulation) Basename(filePath string, extension ...string) PathSegment {
	base := filepath.Base(filePath)
	if len(extension) > 0 {
		ext := extension[0]
		if strings.HasSuffix(base, ext) {
			base = base[:len(base)-len(ext)]
		}
	}
	return PathSegment(base)
}

func (p *NodeJSPathManipulation) Extname(path string) string {
	return filepath.Ext(path)
}

func (p *NodeJSPathManipulation) Normalize(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

type NodeJSReadonlyFileSystem struct {
	NodeJSPathManipulation
	caseSensitive *bool
}

func (r *NodeJSReadonlyFileSystem) IsCaseSensitive() bool {
	if r.caseSensitive == nil {
		isSensitive := true
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			isSensitive = false
		}
		r.caseSensitive = &isSensitive
	}
	return *r.caseSensitive
}

func (r *NodeJSReadonlyFileSystem) Exists(path AbsoluteFsPath) bool {
	_, err := os.Stat(string(path))
	return err == nil
}

func (r *NodeJSReadonlyFileSystem) ReadFile(path AbsoluteFsPath) string {
	bytes, _ := os.ReadFile(string(path))
	return string(bytes)
}

func (r *NodeJSReadonlyFileSystem) ReadFileBuffer(path AbsoluteFsPath) []byte {
	bytes, _ := os.ReadFile(string(path))
	return bytes
}

func (r *NodeJSReadonlyFileSystem) Readdir(path AbsoluteFsPath) []PathSegment {
	entries, _ := os.ReadDir(string(path))
	result := make([]PathSegment, len(entries))
	for i, entry := range entries {
		result[i] = PathSegment(entry.Name())
	}
	return result
}

type fileStats struct {
	fi os.FileInfo
}

func (s *fileStats) IsFile() bool {
	return !s.fi.IsDir()
}

func (s *fileStats) IsDirectory() bool {
	return s.fi.IsDir()
}

func (s *fileStats) IsSymbolicLink() bool {
	return s.fi.Mode()&os.ModeSymlink != 0
}

func (r *NodeJSReadonlyFileSystem) Lstat(path AbsoluteFsPath) FileStats {
	fi, _ := os.Lstat(string(path))
	return &fileStats{fi: fi}
}

func (r *NodeJSReadonlyFileSystem) Stat(path AbsoluteFsPath) FileStats {
	fi, _ := os.Stat(string(path))
	return &fileStats{fi: fi}
}

func (r *NodeJSReadonlyFileSystem) Realpath(path AbsoluteFsPath) AbsoluteFsPath {
	real, _ := filepath.EvalSymlinks(string(path))
	return r.Resolve(real)
}

func (r *NodeJSReadonlyFileSystem) GetDefaultLibLocation() AbsoluteFsPath {
	// Not straightforward in Go, hardcoding or panicking for now depending on use case.
	// We'll return an empty path or panic.
	return r.Resolve("")
}

type NodeJSFileSystem struct {
	NodeJSReadonlyFileSystem
}

func (fs *NodeJSFileSystem) WriteFile(path AbsoluteFsPath, data []byte, exclusive bool) {
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if exclusive {
		flag |= os.O_EXCL
	}
	file, _ := os.OpenFile(string(path), flag, 0666)
	file.Write(data)
	file.Close()
}

func (fs *NodeJSFileSystem) RemoveFile(path AbsoluteFsPath) {
	os.Remove(string(path))
}

func (fs *NodeJSFileSystem) Symlink(target AbsoluteFsPath, path AbsoluteFsPath) {
	os.Symlink(string(target), string(path))
}

func (fs *NodeJSFileSystem) CopyFile(from AbsoluteFsPath, to AbsoluteFsPath) {
	input, _ := os.ReadFile(string(from))
	os.WriteFile(string(to), input, 0666)
}

func (fs *NodeJSFileSystem) MoveFile(from AbsoluteFsPath, to AbsoluteFsPath) {
	os.Rename(string(from), string(to))
}

func (fs *NodeJSFileSystem) EnsureDir(path AbsoluteFsPath) {
	os.MkdirAll(string(path), 0777)
}

func (fs *NodeJSFileSystem) RemoveDeep(path AbsoluteFsPath) {
	os.RemoveAll(string(path))
}

func toggleCase(str string) string {
	var result strings.Builder
	for _, ch := range str {
		if unicode.IsUpper(ch) {
			result.WriteRune(unicode.ToLower(ch))
		} else if unicode.IsLower(ch) {
			result.WriteRune(unicode.ToUpper(ch))
		} else {
			result.WriteRune(ch)
		}
	}
	return result.String()
}
