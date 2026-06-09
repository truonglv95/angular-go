package sourcemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMappings_Empty(t *testing.T) {
	mappings := ParseMappings(nil, nil, nil)
	assert.Nil(t, mappings)

	raw := &RawSourceMap{Mappings: "", Names: []string{}, Sources: []string{}, Version: 3}
	mappings = ParseMappings(raw, []*SourceFile{}, []int{})
	assert.Nil(t, mappings)
}

func TestParseMappings_Valid(t *testing.T) {
	// "AAAA" maps line 0, col 0 to source 0, line 0, col 0
	raw := &RawSourceMap{
		Version:  3,
		Sources:  []string{"a.js"},
		Mappings: "AAAA",
	}

	sourceA := NewSourceFile("/foo/src/a.js", "abcdefg", nil, nil)
	mappings := ParseMappings(raw, []*SourceFile{sourceA}, []int{0, 8})

	require.Len(t, mappings, 1)
	assert.Equal(t, 0, mappings[0].GeneratedSegment.Line)
	assert.Equal(t, 0, mappings[0].GeneratedSegment.Column)
	assert.Equal(t, 0, mappings[0].OriginalSegment.Line)
	assert.Equal(t, 0, mappings[0].OriginalSegment.Column)
	assert.Equal(t, sourceA, mappings[0].OriginalSource)
}

func TestExtractOriginalSegments(t *testing.T) {
	sourceA := NewSourceFile("/foo/src/a.js", "abcdefg", nil, nil)
	raw := &RawSourceMap{
		Version:  3,
		Sources:  []string{"a.js"},
		Mappings: "AAAA,EAAG,EAAD", // multiple mappings: [0,0,0,0], [2,0,0,3], [4,0,0,2]
	}

	mappings := ParseMappings(raw, []*SourceFile{sourceA}, []int{0, 8})
	originalSegments := ExtractOriginalSegments(mappings)

	segments := originalSegments[sourceA]
	require.Len(t, segments, 3)
	assert.Equal(t, 0, segments[0].Column)
	assert.Equal(t, 2, segments[1].Column)
	assert.Equal(t, 3, segments[2].Column)
}

func TestFindLastMappingIndexBefore(t *testing.T) {
	marker5 := &SegmentMarker{Line: 0, Column: 50, Position: 50}
	marker4 := &SegmentMarker{Line: 0, Column: 40, Position: 40, Next: marker5}
	marker3 := &SegmentMarker{Line: 0, Column: 30, Position: 30, Next: marker4}
	marker2 := &SegmentMarker{Line: 0, Column: 20, Position: 20, Next: marker3}
	marker1 := &SegmentMarker{Line: 0, Column: 10, Position: 10, Next: marker2}

	mappings := []Mapping{
		{GeneratedSegment: marker1},
		{GeneratedSegment: marker2},
		{GeneratedSegment: marker3},
		{GeneratedSegment: marker4},
		{GeneratedSegment: marker5},
	}

	// Exact match inclusive
	idx := FindLastMappingIndexBefore(mappings, &SegmentMarker{Line: 0, Column: 30, Position: 30}, false, 0)
	assert.Equal(t, 2, idx)

	// No exact match, below
	idx = FindLastMappingIndexBefore(mappings, &SegmentMarker{Line: 0, Column: 35, Position: 35}, false, 0)
	assert.Equal(t, 2, idx)

	// Below all mappings
	idx = FindLastMappingIndexBefore(mappings, &SegmentMarker{Line: 0, Column: 5, Position: 5}, false, 0)
	assert.Equal(t, -1, idx)

	// Above all mappings
	idx = FindLastMappingIndexBefore(mappings, &SegmentMarker{Line: 0, Column: 60, Position: 60}, false, 0)
	assert.Equal(t, 4, idx)
}

func TestRenderFlattenedSourceMap(t *testing.T) {
	sourceA := NewSourceFile("/foo/src/a.js", "abcdefg", nil, nil)
	raw := &RawSourceMap{
		Version:  3,
		Sources:  []string{"src/a.js"},
		Mappings: "AAAA",
	}

	sf := NewSourceFile("/foo/out.js", "abcdefg", &SourceMapInfo{
		MapAndPath: MapAndPath{
			Map: *raw,
		},
		Origin: ContentOriginProvided,
	}, []*SourceFile{sourceA})

	rendered := sf.RenderFlattenedSourceMap()
	require.NotNil(t, rendered)
	assert.Equal(t, 3, rendered.Version)
	assert.Contains(t, rendered.Sources, "src/a.js")
	assert.Equal(t, "AAAA", rendered.Mappings)
}
