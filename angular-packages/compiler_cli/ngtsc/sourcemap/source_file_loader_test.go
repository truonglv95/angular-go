package sourcemap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockLogger struct {
	Warns []string
}

func (m *MockLogger) Warn(msg ...any) {
	m.Warns = append(m.Warns, fmt.Sprint(msg...))
}

func createRawSourceMap(custom RawSourceMap) RawSourceMap {
	res := RawSourceMap{
		Version:        3,
		SourceRoot:     "",
		Sources:        []string{},
		SourcesContent: nil,
		Names:          []string{},
		Mappings:       "",
	}
	if custom.Version != 0 {
		res.Version = custom.Version
	}
	if custom.File != "" {
		res.File = custom.File
	}
	if custom.SourceRoot != "" {
		res.SourceRoot = custom.SourceRoot
	}
	if len(custom.Sources) > 0 {
		res.Sources = custom.Sources
	}
	if len(custom.SourcesContent) > 0 {
		res.SourcesContent = custom.SourcesContent
	}
	if len(custom.Names) > 0 {
		res.Names = custom.Names
	}
	if custom.Mappings != "" {
		res.Mappings = custom.Mappings
	}
	return res
}

func mapToComment(m RawSourceMap) string {
	b, _ := json.Marshal(m)
	enc := base64.StdEncoding.EncodeToString(b)
	return "\n//# sourceMappingURL=data:application/json;charset=utf-8;base64," + enc
}

func TestLoadSourceFile_NoSourceMapInline(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", "some inline content")
	require.NotNil(t, sourceFile)
	assert.Equal(t, "some inline content", sourceFile.Contents)
	assert.Equal(t, "/foo/src/index.js", sourceFile.SourcePath)
	assert.Nil(t, sourceFile.RawMap)
	assert.Empty(t, sourceFile.Sources)
}

func TestLoadSourceFile_NoSourceMapDisk(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")
	fs.WriteFile("/foo/src/index.js", []byte("some external content"), false)
	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFile("/foo/src/index.js")
	require.NotNil(t, sourceFile)
	assert.Equal(t, "some external content", sourceFile.Contents)
	assert.Equal(t, "/foo/src/index.js", sourceFile.SourcePath)
	assert.Nil(t, sourceFile.RawMap)
	assert.Empty(t, sourceFile.Sources)
}

func TestLoadSourceFile_ExternalSourceMap(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")
	sourceMap := createRawSourceMap(RawSourceMap{File: "index.js"})
	sourceMapBytes, _ := json.Marshal(sourceMap)
	fs.WriteFile("/foo/src/external.js.map", sourceMapBytes, false)
	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents(
		"/foo/src/index.js",
		"some inline content\n//# sourceMappingURL=external.js.map",
	)
	require.NotNil(t, sourceFile)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, sourceMap, sourceFile.RawMap.Map)
}

func TestLoadSourceFile_LastLineOnly(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")
	sourceMap := createRawSourceMap(RawSourceMap{File: "index.js"})
	sourceMapBytes, _ := json.Marshal(sourceMap)
	fs.WriteFile("/foo/src/external.js.map", sourceMapBytes, false)
	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	content := strings.Join([]string{
		"some content",
		"//# sourceMappingURL=bad.js.map",
		"some more content",
		"//# sourceMappingURL=external.js.map",
	}, "\n")

	sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", content)
	require.NotNil(t, sourceFile)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, sourceMap, sourceFile.RawMap.Map)
}

func TestLoadSourceFile_LastNonBlankLine(t *testing.T) {
	for _, eolMarker := range []string{"\n", "\r\n"} {
		t.Run("EOLMarker_"+strings.ReplaceAll(eolMarker, "\r", "R"), func(t *testing.T) {
			fs := file_system.NewMockFileSystem(true)
			fs.EnsureDir("/foo/src")
			sourceMap := createRawSourceMap(RawSourceMap{File: "index.js"})
			sourceMapBytes, _ := json.Marshal(sourceMap)
			fs.WriteFile("/foo/src/external.js.map", sourceMapBytes, false)
			logger := &MockLogger{}
			registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

			content := strings.Join([]string{
				"some content",
				"//# sourceMappingURL=bad.js.map",
				"some more content",
				"//# sourceMappingURL=external.js.map",
				"",
				"",
			}, eolMarker)

			sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", content)
			require.NotNil(t, sourceFile)
			require.NotNil(t, sourceFile.RawMap)
			assert.Equal(t, sourceMap, sourceFile.RawMap.Map)
		})
	}
}

func TestLoadSourceFile_MissingExternalSourceMap(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")
	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents(
		"/foo/src/index.js",
		"some inline content\n//# sourceMappingURL=external.js.map",
	)
	require.NotNil(t, sourceFile)
	assert.Nil(t, sourceFile.RawMap)
}

func TestLoadSourceFile_InlineEncodedSourceMap(t *testing.T) {
	sourceMap := createRawSourceMap(RawSourceMap{File: "index.js"})
	sourceMapBytes, _ := json.Marshal(sourceMap)
	encodedSourceMap := base64.StdEncoding.EncodeToString(sourceMapBytes)
	logger := &MockLogger{}
	registry := NewSourceFileLoader(file_system.NewMockFileSystem(true), logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents(
		"/foo/src/index.js",
		"some inline content\n//# sourceMappingURL=data:application/json;charset=utf-8;base64,"+encodedSourceMap,
	)
	require.NotNil(t, sourceFile)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, sourceMap, sourceFile.RawMap.Map)
}

func TestLoadSourceFile_ImpliedSourceMap(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")
	sourceMap := createRawSourceMap(RawSourceMap{File: "index.js"})
	sourceMapBytes, _ := json.Marshal(sourceMap)
	fs.WriteFile("/foo/src/index.js.map", sourceMapBytes, false)
	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", "some inline content")
	require.NotNil(t, sourceFile)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, sourceMap, sourceFile.RawMap.Map)
}

func TestLoadSourceFile_MissingImpliedSourceMap(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")
	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", "some inline content")
	require.NotNil(t, sourceFile)
	assert.Nil(t, sourceFile.RawMap)
}

func TestLoadSourceFile_RecurseSources(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")

	indexSourceMap := createRawSourceMap(RawSourceMap{
		File:           "index.js",
		Sources:        []string{"x.js", "y.js", "z.js"},
		SourcesContent: []string{"", "", "z content"},
	})
	indexSourceMapBytes, _ := json.Marshal(indexSourceMap)
	fs.WriteFile("/foo/src/index.js.map", indexSourceMapBytes, false)

	fs.WriteFile("/foo/src/x.js", []byte("x content"), false)

	ySourceMap := createRawSourceMap(RawSourceMap{
		File:    "y.js",
		Sources: []string{"a.js"},
	})
	ySourceMapBytes, _ := json.Marshal(ySourceMap)
	fs.WriteFile("/foo/src/y.js", []byte("y content"), false)
	fs.WriteFile("/foo/src/y.js.map", ySourceMapBytes, false)
	fs.WriteFile("/foo/src/z.js", []byte("z content"), false)
	fs.WriteFile("/foo/src/a.js", []byte("a content"), false)

	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", "index content")
	require.NotNil(t, sourceFile)

	assert.Equal(t, "index content", sourceFile.Contents)
	assert.Equal(t, "/foo/src/index.js", sourceFile.SourcePath)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, indexSourceMap, sourceFile.RawMap.Map)

	require.Len(t, sourceFile.Sources, 3)

	assert.Equal(t, "x content", sourceFile.Sources[0].Contents)
	assert.Equal(t, "/foo/src/x.js", sourceFile.Sources[0].SourcePath)
	assert.Nil(t, sourceFile.Sources[0].RawMap)
	assert.Empty(t, sourceFile.Sources[0].Sources)

	assert.Equal(t, "y content", sourceFile.Sources[1].Contents)
	assert.Equal(t, "/foo/src/y.js", sourceFile.Sources[1].SourcePath)
	require.NotNil(t, sourceFile.Sources[1].RawMap)
	assert.Equal(t, ySourceMap, sourceFile.Sources[1].RawMap.Map)

	require.Len(t, sourceFile.Sources[1].Sources, 1)
	assert.Equal(t, "a content", sourceFile.Sources[1].Sources[0].Contents)
	assert.Equal(t, "/foo/src/a.js", sourceFile.Sources[1].Sources[0].SourcePath)
	assert.Nil(t, sourceFile.Sources[1].Sources[0].RawMap)
	assert.Empty(t, sourceFile.Sources[1].Sources[0].Sources)

	assert.Equal(t, "z content", sourceFile.Sources[2].Contents)
	assert.Equal(t, "/foo/src/z.js", sourceFile.Sources[2].SourcePath)
	assert.Nil(t, sourceFile.Sources[2].RawMap)
	assert.Empty(t, sourceFile.Sources[2].Sources)
}

func TestLoadSourceFile_MissingReferencedSourceFile(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")

	indexSourceMap := createRawSourceMap(RawSourceMap{
		File:           "index.js",
		Sources:        []string{"x.js"},
		SourcesContent: []string{""},
	})
	indexSourceMapBytes, _ := json.Marshal(indexSourceMap)
	fs.WriteFile("/foo/src/index.js.map", indexSourceMapBytes, false)

	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", "index content")
	require.NotNil(t, sourceFile)

	assert.Equal(t, "index content", sourceFile.Contents)
	assert.Equal(t, "/foo/src/index.js", sourceFile.SourcePath)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, indexSourceMap, sourceFile.RawMap.Map)

	require.Len(t, sourceFile.Sources, 1)
	assert.Nil(t, sourceFile.Sources[0])
}

func TestLoadSourceFile_CyclicDependencySourceFiles(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")

	aMap := createRawSourceMap(RawSourceMap{File: "a.js", Sources: []string{"b.js"}})
	aPath := "/foo/src/a.js"
	fs.WriteFile(file_system.AbsoluteFsPath(aPath), []byte("a content"+mapToComment(aMap)), false)

	bPath := "/foo/src/b.js"
	bMap := createRawSourceMap(RawSourceMap{File: "b.js", Sources: []string{"c.js"}})
	fs.WriteFile(file_system.AbsoluteFsPath(bPath), []byte("b content"+mapToComment(bMap)), false)

	cPath := "/foo/src/c.js"
	cMap := createRawSourceMap(RawSourceMap{File: "c.js", Sources: []string{"a.js"}})
	fs.WriteFile(file_system.AbsoluteFsPath(cPath), []byte("c content"+mapToComment(cMap)), false)

	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFile(file_system.AbsoluteFsPath(aPath))
	require.NotNil(t, sourceFile)
	assert.Equal(t, "a content\n", sourceFile.Contents)
	assert.Equal(t, aPath, sourceFile.SourcePath)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, aMap, sourceFile.RawMap.Map)
	require.Len(t, sourceFile.Sources, 1)

	require.NotEmpty(t, logger.Warns)
	assert.Contains(t, logger.Warns[0], fmt.Sprintf("Circular source file mapping dependency: %s -> %s -> %s -> %s", aPath, bPath, cPath, aPath))
}

func TestLoadSourceFile_CyclicDependencySourceMaps(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")

	aPath := "/foo/src/a.js"
	fs.WriteFile(file_system.AbsoluteFsPath(aPath), []byte("a.js content\n//# sourceMappingURL=a.js.map"), false)

	aMap := createRawSourceMap(RawSourceMap{File: "a.js", Sources: []string{"b.js"}})
	aMapPath := "/foo/src/a.js.map"
	aMapBytes, _ := json.Marshal(aMap)
	fs.WriteFile(file_system.AbsoluteFsPath(aMapPath), aMapBytes, false)

	bPath := "/foo/src/b.js"
	fs.WriteFile(file_system.AbsoluteFsPath(bPath), []byte("b.js content\n//# sourceMappingURL=a.js.map"), false)

	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFile(file_system.AbsoluteFsPath(aPath))
	require.NotNil(t, sourceFile)
	assert.Equal(t, "a.js content\n", sourceFile.Contents)
	assert.Equal(t, aPath, sourceFile.SourcePath)
	require.NotNil(t, sourceFile.RawMap)
	assert.Equal(t, aMap, sourceFile.RawMap.Map)
	require.Len(t, sourceFile.Sources, 1)

	require.NotEmpty(t, logger.Warns)
	assert.Contains(t, logger.Warns[0], fmt.Sprintf("Circular source file mapping dependency: %s -> %s -> %s -> %s", aPath, aMapPath, bPath, aMapPath))

	innerSourceFile := sourceFile.Sources[0]
	require.NotNil(t, innerSourceFile)
	assert.Equal(t, "b.js content\n", innerSourceFile.Contents)
	assert.Equal(t, bPath, innerSourceFile.SourcePath)
	assert.Nil(t, innerSourceFile.RawMap)
	assert.Empty(t, innerSourceFile.Sources)
}

func TestLoadSourceFile_InlineLooksLikeCycle(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")

	aPath := "/foo/src/a.js"
	aMap := createRawSourceMap(RawSourceMap{
		File:           "a.js",
		Sources:        []string{"a.js"},
		SourcesContent: []string{"inline original a.js content"},
	})
	fs.WriteFile(file_system.AbsoluteFsPath(aPath), []byte("a content"+mapToComment(aMap)), false)

	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFile(file_system.AbsoluteFsPath(aPath))
	require.NotNil(t, sourceFile)
	require.Len(t, sourceFile.Sources, 1)
	assert.Equal(t, "inline original a.js content", sourceFile.Sources[0].Contents)
	assert.Equal(t, aPath, sourceFile.Sources[0].SourcePath)
	assert.Nil(t, sourceFile.Sources[0].RawMap)
	assert.Empty(t, sourceFile.Sources[0].Sources)

	assert.Empty(t, logger.Warns)
}

func TestLoadSourceFile_NoLoadSourceMapIfInlineSource(t *testing.T) {
	fs := file_system.NewMockFileSystem(true)
	fs.EnsureDir("/foo/src")

	aPath := "/foo/src/a.js"
	fs.WriteFile(file_system.AbsoluteFsPath(aPath), []byte("a.js content\n//# sourceMappingURL=a.js.map"), false)
	aMapPath := "/foo/src/a.js.map"
	aMap := createRawSourceMap(RawSourceMap{File: "a.js", Sources: []string{"b.js"}})
	aMapBytes, _ := json.Marshal(aMap)
	fs.WriteFile(file_system.AbsoluteFsPath(aMapPath), aMapBytes, false)

	bPath := "/foo/src/b.js"
	fs.WriteFile(file_system.AbsoluteFsPath(bPath), []byte("b.js content\n//# sourceMappingURL=b.js.map"), false)
	bMapPath := "/foo/src/b.js.map"
	bMap := createRawSourceMap(RawSourceMap{
		File:           "b.js",
		Sources:        []string{"c.js"},
		SourcesContent: []string{"c content\n//# sourceMappingURL=c.js.map"},
	})
	bMapBytes, _ := json.Marshal(bMap)
	fs.WriteFile(file_system.AbsoluteFsPath(bMapPath), bMapBytes, false)

	cMapPath := "/foo/src/c.js.map"
	cMap := createRawSourceMap(RawSourceMap{File: "c.js", Sources: []string{"d.js"}})
	cMapBytes, _ := json.Marshal(cMap)
	fs.WriteFile(file_system.AbsoluteFsPath(cMapPath), cMapBytes, false)

	logger := &MockLogger{}
	registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

	sourceFile := registry.LoadSourceFile(file_system.AbsoluteFsPath(aPath))
	require.NotNil(t, sourceFile)

	bSource := sourceFile.Sources[0]
	require.NotNil(t, bSource)

	cSource := bSource.Sources[0]
	require.NotNil(t, cSource)

	assert.Nil(t, cSource.RawMap)
	assert.Empty(t, cSource.Sources)

	assert.Empty(t, logger.Warns)
}

func TestLoadSourceFile_ProtocolMappedPaths(t *testing.T) {
	testCases := []struct {
		scheme     string
		mappedPath string
	}{
		{scheme: "WEBPACK://", mappedPath: "/foo/src/index.ts"},
		{scheme: "webpack://", mappedPath: "/foo/src/index.ts"},
		{scheme: "missing://", mappedPath: "/src/index.ts"},
	}

	for _, tc := range testCases {
		t.Run("Scheme_"+strings.ReplaceAll(tc.scheme, ":", "_"), func(t *testing.T) {
			fs := file_system.NewMockFileSystem(true)
			fs.EnsureDir("/foo/src")

			indexSourceMap := createRawSourceMap(RawSourceMap{
				File:           "index.js",
				Sources:        []string{tc.scheme + "/src/index.ts"},
				SourcesContent: []string{"original content"},
			})
			indexSourceMapBytes, _ := json.Marshal(indexSourceMap)
			fs.WriteFile("/foo/src/index.js.map", indexSourceMapBytes, false)

			logger := &MockLogger{}
			registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

			sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", "generated content")
			require.NotNil(t, sourceFile)

			originalSource := sourceFile.Sources[0]
			require.NotNil(t, originalSource)
			assert.Equal(t, "original content", originalSource.Contents)
			assert.Equal(t, tc.mappedPath, originalSource.SourcePath)
			assert.Nil(t, originalSource.RawMap)
			assert.Empty(t, originalSource.Sources)
		})
	}
}

func TestLoadSourceFile_ProtocolMappedSourceRoots(t *testing.T) {
	testCases := []struct {
		scheme     string
		mappedPath string
	}{
		{scheme: "WEBPACK://", mappedPath: "/foo/src/index.ts"},
		{scheme: "webpack://", mappedPath: "/foo/src/index.ts"},
		{scheme: "missing://", mappedPath: "/src/index.ts"},
	}

	for _, tc := range testCases {
		t.Run("Scheme_"+strings.ReplaceAll(tc.scheme, ":", "_"), func(t *testing.T) {
			fs := file_system.NewMockFileSystem(true)
			fs.EnsureDir("/foo/src")

			indexSourceMap := createRawSourceMap(RawSourceMap{
				File:           "index.js",
				Sources:        []string{"index.ts"},
				SourcesContent: []string{"original content"},
				SourceRoot:     tc.scheme + "/src",
			})
			indexSourceMapBytes, _ := json.Marshal(indexSourceMap)
			fs.WriteFile("/foo/src/index.js.map", indexSourceMapBytes, false)

			logger := &MockLogger{}
			registry := NewSourceFileLoader(fs, logger, map[string]file_system.AbsoluteFsPath{"webpack": "/foo"})

			sourceFile := registry.LoadSourceFileWithContents("/foo/src/index.js", "generated content")
			require.NotNil(t, sourceFile)

			originalSource := sourceFile.Sources[0]
			require.NotNil(t, originalSource)
			assert.Equal(t, "original content", originalSource.Contents)
			assert.Equal(t, tc.mappedPath, originalSource.SourcePath)
			assert.Nil(t, originalSource.RawMap)
			assert.Empty(t, originalSource.Sources)
		})
	}
}
