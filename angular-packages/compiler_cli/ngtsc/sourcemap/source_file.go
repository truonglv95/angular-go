package sourcemap

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// -------------------------------------------------------------------------
// Segment Marker
// -------------------------------------------------------------------------

// SegmentMarker represents a marker that indicates the start of a segment in a mapping.
type SegmentMarker struct {
	Line     int
	Column   int
	Position int
	Next     *SegmentMarker
}

// CompareSegments compares two segment-markers, for use in a search or sorting algorithm.
func CompareSegments(a, b *SegmentMarker) int {
	return a.Position - b.Position
}

// OffsetSegment returns a new segment-marker that is offset by the given number of characters.
func OffsetSegment(startOfLinePositions []int, marker *SegmentMarker, offset int) *SegmentMarker {
	if offset == 0 {
		return marker
	}

	line := marker.Line
	position := marker.Position + offset
	for line < len(startOfLinePositions)-1 && startOfLinePositions[line+1] <= position {
		line++
	}
	for line > 0 && startOfLinePositions[line] > position {
		line--
	}
	column := position - startOfLinePositions[line]
	return &SegmentMarker{
		Line:     line,
		Column:   column,
		Position: position,
		Next:     nil,
	}
}

// -------------------------------------------------------------------------
// Types needed for SourceFile and Mappings
// -------------------------------------------------------------------------

type RawSourceMap struct {
	Version        int      `json:"version"`
	File           string   `json:"file,omitempty"`
	SourceRoot     string   `json:"sourceRoot,omitempty"`
	Sources        []string `json:"sources"`
	Names          []string `json:"names"`
	SourcesContent []string `json:"sourcesContent,omitempty"`
	Mappings       string   `json:"mappings"`
}

type MapAndPath struct {
	MapPath *string
	Map     RawSourceMap
}

type ContentOrigin int

const (
	ContentOriginProvided ContentOrigin = iota
	ContentOriginInline
	ContentOriginFileSystem
)

type SourceMapInfo struct {
	MapAndPath
	Origin ContentOrigin
}

// Mapping consists of two segment markers: one in the generated source and one in the original
// source, which indicate the start of each segment.
type Mapping struct {
	GeneratedSegment *SegmentMarker
	OriginalSource   *SourceFile
	OriginalSegment  *SegmentMarker
	Name             *string
}

// OriginalLocation holds the mapped original location
type OriginalLocation struct {
	File   string
	Line   int
	Column int
}

// -------------------------------------------------------------------------
// Source File
// -------------------------------------------------------------------------

var sourceMapCommentRegexp = regexp.MustCompile(`(?m)(?://|/\*)[@#]\s+sourceMappingURL=.*?(?:\*/)?$`)

func RemoveSourceMapComments(contents string) string {
	res := sourceMapCommentRegexp.ReplaceAllString(contents, "")
	if strings.HasSuffix(res, "\n\n") {
		res = res[:len(res)-1]
	}
	return res
}

func ComputeStartOfLinePositions(str string) []int {
	const NEWLINE_MARKER_OFFSET = 1
	lines := strings.Split(str, "\n")
	lineLengths := make([]int, len(lines))
	for i, s := range lines {
		lineLengths[i] = len(s)
	}

	startPositions := make([]int, 0, len(lineLengths))
	startPositions = append(startPositions, 0)
	for i := 0; i < len(lineLengths)-1; i++ {
		startPositions = append(startPositions, startPositions[i]+lineLengths[i]+NEWLINE_MARKER_OFFSET)
	}
	return startPositions
}

type SourceFile struct {
	SourcePath           string
	Contents             string
	RawMap               *SourceMapInfo
	Sources              []*SourceFile
	StartOfLinePositions []int
	FlattenedMappings    []Mapping
}

func NewSourceFile(sourcePath string, contents string, rawMap *SourceMapInfo, sources []*SourceFile) *SourceFile {
	contents = RemoveSourceMapComments(contents)
	startOfLinePositions := ComputeStartOfLinePositions(contents)

	sf := &SourceFile{
		SourcePath:           sourcePath,
		Contents:             contents,
		RawMap:               rawMap,
		Sources:              sources,
		StartOfLinePositions: startOfLinePositions,
	}
	sf.FlattenedMappings = sf.flattenMappings()
	return sf
}

func (sf *SourceFile) flattenMappings() []Mapping {
	var rawMap *RawSourceMap
	if sf.RawMap != nil {
		rawMap = &sf.RawMap.Map
	}

	mappings := ParseMappings(rawMap, sf.Sources, sf.StartOfLinePositions)
	EnsureOriginalSegmentLinks(mappings)

	var flattenedMappings []Mapping

	for i := 0; i < len(mappings); i++ {
		aToBmapping := mappings[i]
		bSource := aToBmapping.OriginalSource

		if len(bSource.FlattenedMappings) == 0 {
			flattenedMappings = append(flattenedMappings, aToBmapping)
			continue
		}

		incomingStart := aToBmapping.OriginalSegment
		incomingEnd := incomingStart.Next

		outgoingStartIndex := FindLastMappingIndexBefore(bSource.FlattenedMappings, incomingStart, false, 0)
		if outgoingStartIndex < 0 {
			outgoingStartIndex = 0
		}

		outgoingEndIndex := len(bSource.FlattenedMappings) - 1
		if incomingEnd != nil {
			outgoingEndIndex = FindLastMappingIndexBefore(bSource.FlattenedMappings, incomingEnd, true, outgoingStartIndex)
		}

		for bToCmappingIndex := outgoingStartIndex; bToCmappingIndex <= outgoingEndIndex; bToCmappingIndex++ {
			bToCmapping := bSource.FlattenedMappings[bToCmappingIndex]
			flattenedMappings = append(flattenedMappings, MergeMappings(sf, aToBmapping, bToCmapping))
		}
	}

	return flattenedMappings
}

func (sf *SourceFile) GetOriginalLocation(line, column int) *OriginalLocation {
	if len(sf.FlattenedMappings) == 0 {
		return nil
	}

	var position int
	if line < len(sf.StartOfLinePositions) {
		position = sf.StartOfLinePositions[line] + column
	} else {
		position = len(sf.Contents)
	}

	locationSegment := &SegmentMarker{
		Line:     line,
		Column:   column,
		Position: position,
		Next:     nil,
	}

	mappingIndex := FindLastMappingIndexBefore(sf.FlattenedMappings, locationSegment, false, 0)
	if mappingIndex < 0 {
		mappingIndex = 0
	}

	mapping := sf.FlattenedMappings[mappingIndex]
	offset := locationSegment.Position - mapping.GeneratedSegment.Position
	offsetOriginalSegment := OffsetSegment(mapping.OriginalSource.StartOfLinePositions, mapping.OriginalSegment, offset)

	return &OriginalLocation{
		File:   mapping.OriginalSource.SourcePath,
		Line:   offsetOriginalSegment.Line,
		Column: offsetOriginalSegment.Column,
	}
}

func (sf *SourceFile) RenderFlattenedSourceMap() *RawSourceMap {
	sources := NewIndexedMap()
	names := NewIndexedSet()
	var mappings [][][]int
	sourcePathDir := filepath.ToSlash(filepath.Dir(sf.SourcePath))

	relativeSourcePathCache := make(map[string]string)
	getRelativePath := func(input string) string {
		if val, ok := relativeSourcePathCache[input]; ok {
			return val
		}
		rel, err := filepath.Rel(sourcePathDir, input)
		if err != nil {
			rel = input
		} else {
			rel = filepath.ToSlash(rel)
		}
		relativeSourcePathCache[input] = rel
		return rel
	}

	for _, mapping := range sf.FlattenedMappings {
		sourceIndex := sources.Set(getRelativePath(mapping.OriginalSource.SourcePath), mapping.OriginalSource.Contents)
		mappingArray := []int{
			mapping.GeneratedSegment.Column,
			sourceIndex,
			mapping.OriginalSegment.Line,
			mapping.OriginalSegment.Column,
		}

		if mapping.Name != nil {
			nameIndex := names.Add(*mapping.Name)
			mappingArray = append(mappingArray, nameIndex)
		}

		line := mapping.GeneratedSegment.Line
		for line >= len(mappings) {
			mappings = append(mappings, [][]int{})
		}
		mappings[line] = append(mappings[line], mappingArray)
	}

	fileRel, _ := filepath.Rel(sourcePathDir, sf.SourcePath)

	return &RawSourceMap{
		Version:        3,
		File:           filepath.ToSlash(fileRel),
		Sources:        sources.Keys,
		Names:          names.Values,
		Mappings:       EncodeMappings(mappings),
		SourcesContent: sources.Values,
	}
}

// -------------------------------------------------------------------------
// Mapping Helpers
// -------------------------------------------------------------------------

func FindLastMappingIndexBefore(mappings []Mapping, marker *SegmentMarker, exclusive bool, lowerIndex int) int {
	if len(mappings) == 0 {
		return -1
	}
	if lowerIndex >= len(mappings) {
		return -1
	}

	upperIndex := len(mappings) - 1
	test := 0
	if exclusive {
		test = -1
	}

	if CompareSegments(mappings[lowerIndex].GeneratedSegment, marker) > test {
		return -1
	}

	matchingIndex := -1
	for lowerIndex <= upperIndex {
		index := (upperIndex + lowerIndex) >> 1
		if CompareSegments(mappings[index].GeneratedSegment, marker) <= test {
			matchingIndex = index
			lowerIndex = index + 1
		} else {
			upperIndex = index - 1
		}
	}
	return matchingIndex
}

func MergeMappings(generatedSource *SourceFile, ab Mapping, bc Mapping) Mapping {
	name := bc.Name
	if name == nil {
		name = ab.Name
	}

	diff := CompareSegments(bc.GeneratedSegment, ab.OriginalSegment)
	if diff > 0 {
		return Mapping{
			Name:             name,
			GeneratedSegment: OffsetSegment(generatedSource.StartOfLinePositions, ab.GeneratedSegment, diff),
			OriginalSource:   bc.OriginalSource,
			OriginalSegment:  bc.OriginalSegment,
		}
	} else {
		return Mapping{
			Name:             name,
			GeneratedSegment: ab.GeneratedSegment,
			OriginalSource:   bc.OriginalSource,
			OriginalSegment:  OffsetSegment(bc.OriginalSource.StartOfLinePositions, bc.OriginalSegment, -diff),
		}
	}
}

func ParseMappings(rawMap *RawSourceMap, sources []*SourceFile, generatedSourceStartOfLinePositions []int) []Mapping {
	if rawMap == nil {
		return nil
	}

	rawMappings := DecodeMappings(rawMap.Mappings)
	if rawMappings == nil {
		return nil
	}

	var mappings []Mapping

	for generatedLine, generatedLineMappings := range rawMappings {
		for _, rawMapping := range generatedLineMappings {
			if len(rawMapping) >= 4 {
				sourceIndex := rawMapping[1]
				if sourceIndex >= len(sources) || sources[sourceIndex] == nil {
					continue
				}

				originalSource := sources[sourceIndex]
				generatedColumn := rawMapping[0]

				var name *string
				if len(rawMapping) == 5 && len(rawMap.Names) > 0 && rawMapping[4] < len(rawMap.Names) {
					n := rawMap.Names[rawMapping[4]]
					name = &n
				}

				line := rawMapping[2]
				column := rawMapping[3]

				generatedSegment := &SegmentMarker{
					Line:     generatedLine,
					Column:   generatedColumn,
					Position: generatedSourceStartOfLinePositions[generatedLine] + generatedColumn,
					Next:     nil,
				}

				originalSegment := &SegmentMarker{
					Line:     line,
					Column:   column,
					Position: originalSource.StartOfLinePositions[line] + column,
					Next:     nil,
				}

				mappings = append(mappings, Mapping{
					Name:             name,
					GeneratedSegment: generatedSegment,
					OriginalSegment:  originalSegment,
					OriginalSource:   originalSource,
				})
			}
		}
	}

	return mappings
}

func ExtractOriginalSegments(mappings []Mapping) map[*SourceFile][]*SegmentMarker {
	originalSegments := make(map[*SourceFile][]*SegmentMarker)
	for _, mapping := range mappings {
		originalSource := mapping.OriginalSource
		originalSegments[originalSource] = append(originalSegments[originalSource], mapping.OriginalSegment)
	}

	for _, segmentMarkers := range originalSegments {
		sort.SliceStable(segmentMarkers, func(i, j int) bool {
			return CompareSegments(segmentMarkers[i], segmentMarkers[j]) < 0
		})
	}

	return originalSegments
}

func EnsureOriginalSegmentLinks(mappings []Mapping) {
	segmentsBySource := ExtractOriginalSegments(mappings)
	for _, markers := range segmentsBySource {
		for i := 0; i < len(markers)-1; i++ {
			markers[i].Next = markers[i+1]
		}
	}
}

// -------------------------------------------------------------------------
// Helper Data Structures
// -------------------------------------------------------------------------

type IndexedMap struct {
	map_   map[string]int
	Keys   []string
	Values []string
}

func NewIndexedMap() *IndexedMap {
	return &IndexedMap{
		map_: make(map[string]int),
	}
}

func (im *IndexedMap) Set(key string, value string) int {
	if idx, ok := im.map_[key]; ok {
		return idx
	}
	index := len(im.Values)
	im.Values = append(im.Values, value)
	im.Keys = append(im.Keys, key)
	im.map_[key] = index
	return index
}

type IndexedSet struct {
	map_   map[string]int
	Values []string
}

func NewIndexedSet() *IndexedSet {
	return &IndexedSet{
		map_: make(map[string]int),
	}
}

func (is *IndexedSet) Add(value string) int {
	if idx, ok := is.map_[value]; ok {
		return idx
	}
	index := len(is.Values)
	is.Values = append(is.Values, value)
	is.map_[value] = index
	return index
}

// -------------------------------------------------------------------------
// Stubs for Codec
// -------------------------------------------------------------------------

func DecodeMappings(mappings string) [][][]int {
	// TODO: implement actual VLQ decoding of source map mappings.
	return nil
}

func EncodeMappings(mappings [][][]int) string {
	// TODO: implement actual VLQ encoding of source map mappings.
	return ""
}
