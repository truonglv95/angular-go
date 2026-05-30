package file_system

import (
	"runtime"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
)

type NgtscCompilerHost struct {
	CompilerHost
	fs      FileSystem
	options CompilerOptions
}

func NewNgtscCompilerHost(fs FileSystem, options CompilerOptions) *NgtscCompilerHost {
	return &NgtscCompilerHost{
		fs:      fs,
		options: options,
	}
}

func (h *NgtscCompilerHost) GetSourceFile(fileName string, languageVersion ScriptTarget, onError func(message string), shouldCreateNewSourceFile bool) ast.SourceFile {
	text := h.ReadFile(fileName)
	if text != "" {
		return CreateSourceFile(fileName, text, languageVersion, true, ScriptKind_Unknown)
	}
	return ast.SourceFile{}
}

func (h *NgtscCompilerHost) GetDefaultLibFileName(options CompilerOptions) string {
	return h.fs.Join(h.GetDefaultLibLocation(), GetDefaultLibFileName(options))
}

func (h *NgtscCompilerHost) GetDefaultLibLocation() string {
	return string(h.fs.GetDefaultLibLocation())
}

func (h *NgtscCompilerHost) WriteFile(fileName string, data string, writeByteOrderMark bool, onError func(message string), sourceFiles []ast.SourceFile) {
	path := AbsoluteFrom(fileName)
	h.fs.EnsureDir(AbsoluteFsPath(h.fs.Dirname(string(path))))
	h.fs.WriteFile(path, []byte(data), false)
}

func (h *NgtscCompilerHost) GetCurrentDirectory() string {
	return string(h.fs.Pwd())
}

func (h *NgtscCompilerHost) GetCanonicalFileName(fileName string) string {
	if h.UseCaseSensitiveFileNames() {
		return fileName
	}
	return strings.ToLower(fileName)
}

func (h *NgtscCompilerHost) UseCaseSensitiveFileNames() bool {
	return h.fs.IsCaseSensitive()
}

func (h *NgtscCompilerHost) GetNewLine() string {
	switch 0 {
	case NewLineKind_CarriageReturnLineFeed:
		return "\r\n"
	case NewLineKind_LineFeed:
		return "\n"
	default:
		if runtime.GOOS == "windows" {
			return "\r\n"
		}
		return "\n"
	}
}

func (h *NgtscCompilerHost) FileExists(fileName string) bool {
	absPath := h.fs.Resolve(fileName)
	return h.fs.Exists(absPath) && h.fs.Stat(absPath).IsFile()
}

func (h *NgtscCompilerHost) ReadFile(fileName string) string {
	absPath := h.fs.Resolve(fileName)
	if !h.FileExists(string(absPath)) {
		return ""
	}
	return h.fs.ReadFile(absPath)
}

func (h *NgtscCompilerHost) Realpath(path string) string {
	return string(h.fs.Realpath(h.fs.Resolve(path)))
}
