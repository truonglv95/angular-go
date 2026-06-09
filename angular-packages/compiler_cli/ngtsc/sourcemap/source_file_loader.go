package sourcemap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
)

// Logger is a simple logging interface for the source file loader.
type Logger interface {
	Warn(msg ...any)
}

var schemeMatcher = regexp.MustCompile(`(?i)^([a-z][a-z0-9.-]*):\/\/`)

var inlineCommentRegex = regexp.MustCompile(`(?i)sourceMappingURL=data:(?:application|text)/json;(?:charset[:=][^;]+;)?base64,([A-Za-z0-9+/=\-_]+)`)

var externalCommentRegexSlashSlash = regexp.MustCompile("//[@#][ \\t]+sourceMappingURL=([^\\s'\"`]+)")
var externalCommentRegexSlashStar  = regexp.MustCompile(`/\*[@#][ \t]+sourceMappingURL=([^\*]+?)[ \t]*\*/`)

// SourceFileLoader loads a source file, its associated source map and any upstream sources.
type SourceFileLoader struct {
	fs           file_system.ReadonlyFileSystem
	logger       Logger
	schemeMap    map[string]file_system.AbsoluteFsPath
	currentPaths []file_system.AbsoluteFsPath
}

// NewSourceFileLoader creates a new SourceFileLoader.
func NewSourceFileLoader(
	fs file_system.ReadonlyFileSystem,
	logger Logger,
	schemeMap map[string]file_system.AbsoluteFsPath,
) *SourceFileLoader {
	normalizedSchemeMap := make(map[string]file_system.AbsoluteFsPath)
	for k, v := range schemeMap {
		normalizedSchemeMap[strings.ToLower(k)] = v
	}
	return &SourceFileLoader{
		fs:        fs,
		logger:    logger,
		schemeMap: normalizedSchemeMap,
	}
}

// LoadSourceFile loads a source file from the file-system, compute its source map,
// and recursively load any referenced source files.
func (l *SourceFileLoader) LoadSourceFile(sourcePath file_system.AbsoluteFsPath) *SourceFile {
	return l.loadSourceFileInternal(sourcePath, nil, ContentOriginFileSystem, nil)
}

// LoadSourceFileWithContents loads a source file from the provided content,
// compute its source map, and recursively load any referenced source files.
func (l *SourceFileLoader) LoadSourceFileWithContents(sourcePath file_system.AbsoluteFsPath, contents string) *SourceFile {
	return l.loadSourceFileInternal(sourcePath, &contents, ContentOriginProvided, nil)
}

// LoadSourceFileWithMap loads a source file from the provided content and source map,
// and recursively load any referenced source files.
func (l *SourceFileLoader) LoadSourceFileWithMap(sourcePath file_system.AbsoluteFsPath, contents string, mapAndPath MapAndPath) *SourceFile {
	sourceMapInfo := &SourceMapInfo{
		MapAndPath: mapAndPath,
		Origin:     ContentOriginProvided,
	}
	return l.loadSourceFileInternal(sourcePath, &contents, ContentOriginProvided, sourceMapInfo)
}

func (l *SourceFileLoader) loadSourceFileInternal(
	sourcePath file_system.AbsoluteFsPath,
	contents *string,
	sourceOrigin ContentOrigin,
	sourceMapInfo *SourceMapInfo,
) *SourceFile {
	previousPaths := make([]file_system.AbsoluteFsPath, len(l.currentPaths))
	copy(previousPaths, l.currentPaths)
	defer func() {
		l.currentPaths = previousPaths
	}()

	var actualContents string
	if contents == nil {
		if !l.fs.Exists(sourcePath) {
			return nil
		}
		var err error
		actualContents, err = l.readSourceFile(sourcePath)
		if err != nil {
			l.logger.Warn(fmt.Sprintf("Unable to fully load %s for source-map flattening: %s", string(sourcePath), err.Error()))
			return nil
		}
	} else {
		actualContents = *contents
	}

	if sourceMapInfo == nil {
		var err error
		sourceMapInfo, err = l.loadSourceMap(sourcePath, actualContents, sourceOrigin)
		if err != nil {
			l.logger.Warn(fmt.Sprintf("Unable to fully load %s for source-map flattening: %s", string(sourcePath), err.Error()))
			return nil
		}
	}

	var sources []*SourceFile
	if sourceMapInfo != nil {
		basePath := sourcePath
		if sourceMapInfo.MapPath != nil {
			basePath = file_system.AbsoluteFsPath(*sourceMapInfo.MapPath)
		}
		sources = l.processSources(basePath, sourceMapInfo)
	}

	return NewSourceFile(string(sourcePath), actualContents, sourceMapInfo, sources)
}

func (l *SourceFileLoader) readSourceFile(sourcePath file_system.AbsoluteFsPath) (string, error) {
	if err := l.trackPath(sourcePath); err != nil {
		return "", err
	}
	return l.fs.ReadFile(sourcePath), nil
}

func (l *SourceFileLoader) readRawSourceMap(mapPath file_system.AbsoluteFsPath) (RawSourceMap, error) {
	if err := l.trackPath(mapPath); err != nil {
		return RawSourceMap{}, err
	}
	content := l.fs.ReadFile(mapPath)
	var rawMap RawSourceMap
	if err := json.Unmarshal([]byte(content), &rawMap); err != nil {
		return RawSourceMap{}, err
	}
	return rawMap, nil
}

func (l *SourceFileLoader) trackPath(path file_system.AbsoluteFsPath) error {
	for _, p := range l.currentPaths {
		if p == path {
			var chain []string
			for _, cp := range l.currentPaths {
				chain = append(chain, string(cp))
			}
			return fmt.Errorf("Circular source file mapping dependency: %s -> %s", strings.Join(chain, " -> "), string(path))
		}
	}
	l.currentPaths = append(l.currentPaths, path)
	return nil
}

func (l *SourceFileLoader) loadSourceMap(
	sourcePath file_system.AbsoluteFsPath,
	sourceContents string,
	sourceOrigin ContentOrigin,
) (*SourceMapInfo, error) {
	lastLine := l.getLastNonEmptyLine(sourceContents)

	if matches := inlineCommentRegex.FindStringSubmatch(lastLine); len(matches) > 1 {
		base64Data := matches[1]
		decodedBytes, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			decodedBytes, err = base64.URLEncoding.DecodeString(base64Data)
		}
		if err != nil {
			return nil, err
		}
		var rawMap RawSourceMap
		if err := json.Unmarshal(decodedBytes, &rawMap); err != nil {
			return nil, err
		}
		return &SourceMapInfo{
			MapAndPath: MapAndPath{
				Map:     rawMap,
				MapPath: nil,
			},
			Origin: ContentOriginInline,
		}, nil
	}

	if sourceOrigin == ContentOriginInline {
		return nil, nil
	}

	var fileName string
	if matches := externalCommentRegexSlashSlash.FindStringSubmatch(lastLine); len(matches) > 1 {
		fileName = strings.TrimSpace(matches[1])
	} else if matches := externalCommentRegexSlashStar.FindStringSubmatch(lastLine); len(matches) > 1 {
		fileName = strings.TrimSpace(matches[1])
	}

	if fileName != "" {
		externalMapPath := l.resolve(l.fs.Dirname(string(sourcePath)), fileName)
		rawMap, err := l.readRawSourceMap(externalMapPath)
		if err != nil {
			l.logger.Warn(fmt.Sprintf("Unable to fully load %s for source-map flattening: %s", string(sourcePath), err.Error()))
			return nil, nil
		}
		mapPathStr := string(externalMapPath)
		return &SourceMapInfo{
			MapAndPath: MapAndPath{
				Map:     rawMap,
				MapPath: &mapPathStr,
			},
			Origin: ContentOriginFileSystem,
		}, nil
	}

	impliedMapPath := l.fs.Resolve(string(sourcePath) + ".map")
	if l.fs.Exists(impliedMapPath) {
		rawMap, err := l.readRawSourceMap(impliedMapPath)
		if err != nil {
			return nil, err
		}
		mapPathStr := string(impliedMapPath)
		return &SourceMapInfo{
			MapAndPath: MapAndPath{
				Map:     rawMap,
				MapPath: &mapPathStr,
			},
			Origin: ContentOriginFileSystem,
		}, nil
	}

	return nil, nil
}

func (l *SourceFileLoader) processSources(
	basePath file_system.AbsoluteFsPath,
	info *SourceMapInfo,
) []*SourceFile {
	sourceRoot := l.resolve(
		l.fs.Dirname(string(basePath)),
		l.replaceSchemeWithPath(info.Map.SourceRoot),
	)

	sources := make([]*SourceFile, len(info.Map.Sources))
	for i, source := range info.Map.Sources {
		path := l.resolve(string(sourceRoot), l.replaceSchemeWithPath(source))
		var content *string
		if i < len(info.Map.SourcesContent) && info.Map.SourcesContent[i] != "" {
			content = &info.Map.SourcesContent[i]
		}

		sourceOrigin := ContentOriginFileSystem
		if content != nil && info.Origin != ContentOriginProvided {
			sourceOrigin = ContentOriginInline
		}

		sources[i] = l.loadSourceFileInternal(path, content, sourceOrigin, nil)
	}
	return sources
}

func (l *SourceFileLoader) getLastNonEmptyLine(contents string) string {
	if contents == "" {
		return ""
	}
	trailingWhitespaceIndex := len(contents) - 1
	for trailingWhitespaceIndex > 0 && (contents[trailingWhitespaceIndex] == '\n' || contents[trailingWhitespaceIndex] == '\r') {
		trailingWhitespaceIndex--
	}
	lastRealLineIndex := -1
	if trailingWhitespaceIndex-1 >= 0 {
		lastRealLineIndex = strings.LastIndex(contents[:trailingWhitespaceIndex], "\n")
	}
	if lastRealLineIndex == -1 {
		lastRealLineIndex = 0
	}
	start := lastRealLineIndex + 1
	end := trailingWhitespaceIndex + 1
	if start > len(contents) {
		start = len(contents)
	}
	if end > len(contents) {
		end = len(contents)
	}
	if start > end {
		return ""
	}
	return contents[start:end]
}

func (l *SourceFileLoader) replaceSchemeWithPath(path string) string {
	matches := schemeMatcher.FindStringSubmatch(path)
	if len(matches) > 1 {
		scheme := strings.ToLower(matches[1])
		replacement := ""
		if mapped, ok := l.schemeMap[scheme]; ok {
			replacement = string(mapped)
		}
		return strings.Replace(path, matches[0], replacement, 1)
	}
	return path
}

func (l *SourceFileLoader) resolve(base string, path string) file_system.AbsoluteFsPath {
	if l.fs.IsRooted(path) {
		return l.fs.Resolve(path)
	}
	return l.fs.Resolve(base, path)
}
