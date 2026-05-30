package file_system

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// PathManipulation methods

func (o *OSFileSystem) Extname(path string) string {
	return filepath.Ext(path)
}

func (o *OSFileSystem) IsRoot(path AbsoluteFsPath) bool {
	return o.Dirname(string(path)) == o.Normalize(string(path))
}

func (o *OSFileSystem) IsRooted(path string) bool {
	return filepath.IsAbs(path)
}

func (o *OSFileSystem) Dirname(file string) string {
	return o.Normalize(filepath.Dir(file))
}

func (o *OSFileSystem) Join(basePath string, paths ...string) string {
	allPaths := append([]string{basePath}, paths...)
	return o.Normalize(filepath.Join(allPaths...))
}

func (o *OSFileSystem) Relative(from string, to string) string {
	rel, err := filepath.Rel(from, to)
	if err != nil {
		return to // fallback
	}
	return o.Normalize(rel)
}

func (o *OSFileSystem) Basename(filePath string, extension string) PathSegment {
	base := filepath.Base(filePath)
	if extension != "" && strings.HasSuffix(base, extension) {
		base = base[:len(base)-len(extension)]
	}
	return PathSegment(base)
}

func (o *OSFileSystem) Normalize(path string) string {
	// Convert backslashes to forward slashes
	return strings.ReplaceAll(path, "\\", "/")
}

func (o *OSFileSystem) Resolve(paths ...string) AbsoluteFsPath {
	if len(paths) == 0 {
		return o.Pwd()
	}
	// If the first path is not absolute, resolve from PWD
	absPath := paths[0]
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(string(o.Pwd()), absPath)
	}
	allPaths := append([]string{absPath}, paths[1:]...)
	return AbsoluteFsPath(o.Normalize(filepath.Join(allPaths...)))
}

func (o *OSFileSystem) Pwd() AbsoluteFsPath {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return AbsoluteFsPath(o.Normalize(dir))
}

func (o *OSFileSystem) Chdir(path AbsoluteFsPath) error {
	return os.Chdir(string(path))
}

// ReadonlyFileSystem methods

func (o *OSFileSystem) IsCaseSensitive() bool {
	// We do the file toggle case check like NodeJS version to detect case sensitivity.
	pwd := string(o.Pwd())
	if pwd == "" {
		return true
	}
	toggled := toggleCase(pwd)
	_, err := os.Stat(toggled)
	return errors.Is(err, fs.ErrNotExist)
}

func toggleCase(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			result.WriteRune(r - 32)
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune(r + 32)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func (o *OSFileSystem) Exists(path AbsoluteFsPath) bool {
	_, err := os.Stat(string(path))
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}

func (o *OSFileSystem) ReadFile(path AbsoluteFsPath) (string, error) {
	data, err := os.ReadFile(string(path))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (o *OSFileSystem) ReadFileBuffer(path AbsoluteFsPath) ([]byte, error) {
	return os.ReadFile(string(path))
}

func (o *OSFileSystem) Readdir(path AbsoluteFsPath) ([]PathSegment, error) {
	entries, err := os.ReadDir(string(path))
	if err != nil {
		return nil, err
	}
	var segments []PathSegment
	for _, entry := range entries {
		segments = append(segments, PathSegment(entry.Name()))
	}
	return segments, nil
}

func (o *OSFileSystem) Lstat(path AbsoluteFsPath) (FileStats, error) {
	info, err := os.Lstat(string(path))
	if err != nil {
		return nil, err
	}
	return &osFileStats{info: info}, nil
}

func (o *OSFileSystem) Stat(path AbsoluteFsPath) (FileStats, error) {
	info, err := os.Stat(string(path))
	if err != nil {
		return nil, err
	}
	return &osFileStats{info: info}, nil
}

func (o *OSFileSystem) Realpath(filePath AbsoluteFsPath) (AbsoluteFsPath, error) {
	resolved, err := filepath.EvalSymlinks(string(filePath))
	if err != nil {
		return "", err
	}
	return AbsoluteFsPath(o.Normalize(resolved)), nil
}

func (o *OSFileSystem) GetDefaultLibLocation() AbsoluteFsPath {
	// TODO: To be implemented based on typescript-go lib location
	return ""
}

// FileSystem methods

func (o *OSFileSystem) WriteFile(path AbsoluteFsPath, data []byte, exclusive bool) error {
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if exclusive {
		flag |= os.O_EXCL
	}
	f, err := os.OpenFile(string(path), flag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (o *OSFileSystem) RemoveFile(path AbsoluteFsPath) error {
	return os.Remove(string(path))
}

func (o *OSFileSystem) Symlink(target AbsoluteFsPath, path AbsoluteFsPath) error {
	return os.Symlink(string(target), string(path))
}

func (o *OSFileSystem) CopyFile(from AbsoluteFsPath, to AbsoluteFsPath) error {
	data, err := os.ReadFile(string(from))
	if err != nil {
		return err
	}
	return os.WriteFile(string(to), data, 0644)
}

func (o *OSFileSystem) MoveFile(from AbsoluteFsPath, to AbsoluteFsPath) error {
	return os.Rename(string(from), string(to))
}

func (o *OSFileSystem) EnsureDir(path AbsoluteFsPath) error {
	return os.MkdirAll(string(path), 0755)
}

func (o *OSFileSystem) RemoveDeep(path AbsoluteFsPath) error {
	return os.RemoveAll(string(path))
}

// osFileStats implements FileStats
type osFileStats struct {
	info os.FileInfo
}

func (s *osFileStats) IsFile() bool {
	return !s.info.IsDir()
}

func (s *osFileStats) IsDirectory() bool {
	return s.info.IsDir()
}

func (s *osFileStats) IsSymbolicLink() bool {
	return s.info.Mode()&os.ModeSymlink != 0
}
