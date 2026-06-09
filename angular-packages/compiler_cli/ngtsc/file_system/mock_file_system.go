package file_system

import (
	"fmt"
	"path"
	"strings"
)

type MockFileEntry struct {
	Data          []byte
	IsDir         bool
	SymlinkTarget string // if non-empty, this entry is a symlink
}

type MockFileSystem struct {
	caseSensitive bool
	files         map[AbsoluteFsPath]*MockFileEntry
	cwd           AbsoluteFsPath
}

func NewMockFileSystem(caseSensitive bool) *MockFileSystem {
	return &MockFileSystem{
		caseSensitive: caseSensitive,
		files:         make(map[AbsoluteFsPath]*MockFileEntry),
		cwd:           "/",
	}
}

func (m *MockFileSystem) IsCaseSensitive() bool {
	return m.caseSensitive
}

func (m *MockFileSystem) canonicalize(path AbsoluteFsPath) AbsoluteFsPath {
	if m.caseSensitive {
		return path
	}
	return AbsoluteFsPath(strings.ToLower(string(path)))
}

func (m *MockFileSystem) resolvePath(path AbsoluteFsPath, followTerminalSymlink bool) AbsoluteFsPath {
	if followTerminalSymlink {
		return m.Realpath(path)
	}
	resolved := m.Resolve(string(path))
	parent := m.Dirname(string(resolved))
	if parent != string(resolved) && parent != "/" {
		resolvedParent := m.Realpath(AbsoluteFsPath(parent))
		return m.Resolve(string(resolvedParent), string(m.Basename(string(resolved))))
	}
	return resolved
}

func (m *MockFileSystem) Exists(path AbsoluteFsPath) bool {
	resolved := m.resolvePath(path, false)
	_, exists := m.files[m.canonicalize(resolved)]
	return exists
}

func (m *MockFileSystem) ReadFile(path AbsoluteFsPath) string {
	resolved := m.resolvePath(path, true)
	entry, exists := m.files[m.canonicalize(resolved)]
	if !exists || entry.IsDir {
		return ""
	}
	return string(entry.Data)
}

func (m *MockFileSystem) ReadFileBuffer(path AbsoluteFsPath) []byte {
	resolved := m.resolvePath(path, true)
	entry, exists := m.files[m.canonicalize(resolved)]
	if !exists || entry.IsDir {
		return nil
	}
	return entry.Data
}

func (m *MockFileSystem) WriteFile(path AbsoluteFsPath, data []byte, exclusive bool) {
	resolved := m.resolvePath(path, true)
	canonical := m.canonicalize(resolved)
	if exclusive {
		if _, exists := m.files[canonical]; exists {
			panic(fmt.Sprintf("EEXIST: file already exists: %s", path))
		}
	}
	// Ensure parent dir exists
	parent := m.Dirname(string(resolved))
	if parent != "/" && parent != string(resolved) {
		m.EnsureDir(AbsoluteFsPath(parent))
	}

	m.files[canonical] = &MockFileEntry{
		Data:  data,
		IsDir: false,
	}
}

func (m *MockFileSystem) RemoveFile(path AbsoluteFsPath) {
	resolved := m.resolvePath(path, false)
	delete(m.files, m.canonicalize(resolved))
}

func (m *MockFileSystem) Symlink(target AbsoluteFsPath, path AbsoluteFsPath) {
	resolvedPath := m.resolvePath(path, false)
	// Ensure parent dir exists
	parent := m.Dirname(string(resolvedPath))
	if parent != "/" && parent != string(resolvedPath) {
		m.EnsureDir(AbsoluteFsPath(parent))
	}

	m.files[m.canonicalize(resolvedPath)] = &MockFileEntry{
		IsDir:         false,
		SymlinkTarget: string(target),
	}
}

func (m *MockFileSystem) CopyFile(from AbsoluteFsPath, to AbsoluteFsPath) {
	resolvedFrom := m.resolvePath(from, true)
	resolvedTo := m.resolvePath(to, true)
	entry, exists := m.files[m.canonicalize(resolvedFrom)]
	if exists && !entry.IsDir {
		m.WriteFile(resolvedTo, entry.Data, false)
	}
}

func (m *MockFileSystem) MoveFile(from AbsoluteFsPath, to AbsoluteFsPath) {
	m.CopyFile(from, to)
	m.RemoveFile(from)
}

func (m *MockFileSystem) EnsureDir(path AbsoluteFsPath) {
	resolved := m.resolvePath(path, true)
	canonical := m.canonicalize(resolved)
	if entry, exists := m.files[canonical]; exists && !entry.IsDir {
		panic(fmt.Sprintf("Folder already exists as a file: %s", resolved))
	}
	if resolved == "/" {
		return
	}
	// Ensure parent dir exists
	parent := m.Dirname(string(resolved))
	if parent != "/" && parent != string(resolved) {
		m.EnsureDir(AbsoluteFsPath(parent))
	}

	m.files[canonical] = &MockFileEntry{
		IsDir: true,
	}
}

func (m *MockFileSystem) RemoveDeep(path AbsoluteFsPath) {
	resolved := m.resolvePath(path, false)
	resolvedCanonical := m.canonicalize(resolved)
	prefix := string(resolvedCanonical)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for p := range m.files {
		if p == resolvedCanonical || strings.HasPrefix(string(p), prefix) {
			delete(m.files, p)
		}
	}
}


func (m *MockFileSystem) Extname(filePath string) string {
	return path.Ext(filePath)
}

func (m *MockFileSystem) IsRoot(p AbsoluteFsPath) bool {
	resolved := m.Resolve(string(p))
	return resolved == "/"
}

func (m *MockFileSystem) IsRooted(p string) bool {
	return strings.HasPrefix(p, "/") || (len(p) >= 3 && p[1] == ':' && p[2] == '/')
}

func (m *MockFileSystem) Dirname(file string) string {
	cleaned := m.Normalize(file)
	dir := path.Dir(cleaned)
	return dir
}

func (m *MockFileSystem) Join(basePath string, paths ...string) string {
	cleanedBase := m.Normalize(basePath)
	all := append([]string{cleanedBase}, paths...)
	return m.Normalize(path.Join(all...))
}

func (m *MockFileSystem) Relative(from string, to string) string {
	fromClean := string(m.Resolve(from))
	toClean := string(m.Resolve(to))

	if fromClean == toClean {
		return "."
	}

	fromParts := strings.Split(strings.Trim(fromClean, "/"), "/")
	toParts := strings.Split(strings.Trim(toClean, "/"), "/")

	if fromClean == "/" {
		fromParts = nil
	}
	if toClean == "/" {
		toParts = nil
	}

	common := 0
	for common < len(fromParts) && common < len(toParts) && fromParts[common] == toParts[common] {
		common++
	}

	var relParts []string
	for i := common; i < len(fromParts); i++ {
		relParts = append(relParts, "..")
	}
	for i := common; i < len(toParts); i++ {
		relParts = append(relParts, toParts[i])
	}

	return strings.Join(relParts, "/")
}

func (m *MockFileSystem) Basename(filePath string, extension ...string) PathSegment {
	base := path.Base(filePath)
	if len(extension) > 0 {
		ext := extension[0]
		if strings.HasSuffix(base, ext) {
			base = base[:len(base)-len(ext)]
		}
	}
	return PathSegment(base)
}

func (m *MockFileSystem) Normalize(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return path.Clean(p)
}

func (m *MockFileSystem) Resolve(paths ...string) AbsoluteFsPath {
	var joined string
	if len(paths) > 0 && m.IsRooted(paths[0]) {
		joined = path.Join(paths...)
	} else if len(paths) > 0 {
		joined = path.Join(string(m.cwd), path.Join(paths...))
	} else {
		joined = string(m.cwd)
	}
	return AbsoluteFsPath(m.Normalize(joined))
}

func (m *MockFileSystem) Pwd() AbsoluteFsPath {
	return m.cwd
}

func (m *MockFileSystem) Chdir(path AbsoluteFsPath) {
	m.cwd = m.Resolve(string(path))
}

func (m *MockFileSystem) Readdir(path AbsoluteFsPath) []PathSegment {
	resolved := m.Realpath(path)
	resolvedCanonical := m.canonicalize(resolved)
	prefix := string(resolvedCanonical)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	var segments []PathSegment
	seen := make(map[string]bool)
	for p := range m.files {
		pStr := string(p)
		if strings.HasPrefix(pStr, prefix) {
			rel := pStr[len(prefix):]
			parts := strings.Split(rel, "/")
			if len(parts) > 0 && parts[0] != "" {
				segment := parts[0]
				canonicalSeg := string(m.canonicalize(AbsoluteFsPath(segment)))
				if !seen[canonicalSeg] {
					seen[canonicalSeg] = true
					segments = append(segments, PathSegment(segment))
				}
			}
		}
	}
	return segments
}

func (m *MockFileSystem) Realpath(filePath AbsoluteFsPath) AbsoluteFsPath {
	resolved := m.Resolve(string(filePath))
	entry, exists := m.files[m.canonicalize(resolved)]
	if exists && entry.SymlinkTarget != "" {
		target := entry.SymlinkTarget
		if !m.IsRooted(target) {
			target = m.Join(string(m.Dirname(string(resolved))), target)
		}
		return m.Realpath(AbsoluteFsPath(target))
	}
	parent := m.Dirname(string(resolved))
	if parent != string(resolved) && parent != "/" {
		resolvedParent := m.Realpath(AbsoluteFsPath(parent))
		return m.Resolve(string(resolvedParent), string(m.Basename(string(resolved))))
	}
	return resolved
}

func (m *MockFileSystem) GetDefaultLibLocation() AbsoluteFsPath {
	return "/node_modules/typescript/lib"
}

type MockFileStats struct {
	isDir     bool
	isSymlink bool
}

func (s *MockFileStats) IsFile() bool {
	return !s.isDir && !s.isSymlink
}

func (s *MockFileStats) IsDirectory() bool {
	return s.isDir
}

func (s *MockFileStats) IsSymbolicLink() bool {
	return s.isSymlink
}

func (m *MockFileSystem) Lstat(path AbsoluteFsPath) FileStats {
	resolved := m.resolvePath(path, false)
	entry, exists := m.files[m.canonicalize(resolved)]
	if !exists {
		panic(fmt.Sprintf("lstat failed: file not found: %s", path))
	}
	return &MockFileStats{
		isDir:     entry.IsDir,
		isSymlink: entry.SymlinkTarget != "",
	}
}

func (m *MockFileSystem) Stat(path AbsoluteFsPath) FileStats {
	resolved := m.resolvePath(path, true)
	entry, exists := m.files[m.canonicalize(resolved)]
	if !exists {
		panic(fmt.Sprintf("stat failed: file not found: %s", path))
	}
	return &MockFileStats{
		isDir:     entry.IsDir,
		isSymlink: false,
	}
}
