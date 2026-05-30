package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

const version = 3
const jsB64Prefix = "# sourceMappingURL=data:application/json;base64,"
const b64Digits = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

type segment struct {
	col0        int
	sourceUrl   *string
	sourceLine0 *int
	sourceCol0  *int
}

type SourceMap struct {
	Version        int       `json:"version"`
	File           string    `json:"file,omitempty"`
	SourceRoot     string    `json:"sourceRoot"`
	Sources        []string  `json:"sources"`
	SourcesContent []*string `json:"sourcesContent"`
	Mappings       string    `json:"mappings"`
}

type SourceMapGenerator struct {
	file           *string
	sourcesContent map[string]*string
	sources        []string
	lines          [][]segment
	lastCol0       int
	hasMappings    bool
}

func NewSourceMapGenerator(file *string) *SourceMapGenerator {
	return &SourceMapGenerator{
		file:           file,
		sourcesContent: make(map[string]*string),
		sources:        []string{},
		lines:          [][]segment{},
	}
}

func (g *SourceMapGenerator) AddSource(url string, content *string) *SourceMapGenerator {
	if _, ok := g.sourcesContent[url]; !ok {
		g.sourcesContent[url] = content
		g.sources = append(g.sources, url)
	}
	return g
}

func (g *SourceMapGenerator) AddLine() *SourceMapGenerator {
	g.lines = append(g.lines, []segment{})
	g.lastCol0 = 0
	return g
}

func (g *SourceMapGenerator) AddMapping(col0 int, sourceUrl *string, sourceLine0 *int, sourceCol0 *int) *SourceMapGenerator {
	if len(g.lines) == 0 {
		panic("A line must be added before mappings can be added")
	}
	if sourceUrl != nil {
		if _, ok := g.sourcesContent[*sourceUrl]; !ok {
			panic(fmt.Sprintf("Unknown source file \"%s\"", *sourceUrl))
		}
	}
	if col0 < g.lastCol0 {
		panic("Mapping should be added in output order")
	}
	if sourceUrl != nil && (sourceLine0 == nil || sourceCol0 == nil) {
		panic("The source location must be provided when a source url is provided")
	}

	g.hasMappings = true
	g.lastCol0 = col0
	g.lines[len(g.lines)-1] = append(g.lines[len(g.lines)-1], segment{
		col0:        col0,
		sourceUrl:   sourceUrl,
		sourceLine0: sourceLine0,
		sourceCol0:  sourceCol0,
	})
	return g
}

func (g *SourceMapGenerator) ToJSON() *SourceMap {
	if !g.hasMappings {
		return nil
	}

	sourcesIndex := make(map[string]int)
	var sourcesContent []*string

	for i, url := range g.sources {
		sourcesIndex[url] = i
		sourcesContent = append(sourcesContent, g.sourcesContent[url])
	}

	mappings := ""
	lastCol0 := 0
	lastSourceIndex := 0
	lastSourceLine0 := 0
	lastSourceCol0 := 0

	for _, segments := range g.lines {
		lastCol0 = 0
		var segmentStrs []string

		for _, seg := range segments {
			segAsStr := toBase64VLQ(seg.col0 - lastCol0)
			lastCol0 = seg.col0

			if seg.sourceUrl != nil {
				segAsStr += toBase64VLQ(sourcesIndex[*seg.sourceUrl] - lastSourceIndex)
				lastSourceIndex = sourcesIndex[*seg.sourceUrl]
				segAsStr += toBase64VLQ(*seg.sourceLine0 - lastSourceLine0)
				lastSourceLine0 = *seg.sourceLine0
				segAsStr += toBase64VLQ(*seg.sourceCol0 - lastSourceCol0)
				lastSourceCol0 = *seg.sourceCol0
			}
			segmentStrs = append(segmentStrs, segAsStr)
		}
		mappings += strings.Join(segmentStrs, ",")
		mappings += ";"
	}

	if len(mappings) > 0 {
		mappings = mappings[:len(mappings)-1]
	}

	file := ""
	if g.file != nil {
		file = *g.file
	}

	return &SourceMap{
		File:           file,
		Version:        version,
		SourceRoot:     "",
		Sources:        g.sources,
		SourcesContent: sourcesContent,
		Mappings:       mappings,
	}
}

func (g *SourceMapGenerator) ToJsComment() string {
	if !g.hasMappings {
		return ""
	}
	jsonBytes, err := json.Marshal(g.ToJSON())
	if err != nil {
		panic(err)
	}
	return "//" + jsB64Prefix + ToBase64String(string(jsonBytes))
}

func ToBase64String(value string) string {
	b64 := ""
	encoded := []byte(value)
	for i := 0; i < len(encoded); {
		i1 := int(encoded[i])
		i++
		var i2 *int
		if i < len(encoded) {
			val := int(encoded[i])
			i2 = &val
			i++
		}
		var i3 *int
		if i < len(encoded) {
			val := int(encoded[i])
			i3 = &val
			i++
		}

		b64 += toBase64Digit(i1 >> 2)
		i2Val := 0
		if i2 != nil {
			i2Val = *i2 >> 4
		}
		b64 += toBase64Digit(((i1 & 3) << 4) | i2Val)

		if i2 == nil {
			b64 += "="
		} else {
			i3Val := 0
			if i3 != nil {
				i3Val = *i3 >> 6
			}
			b64 += toBase64Digit(((*i2 & 15) << 2) | i3Val)
		}

		if i2 == nil || i3 == nil {
			b64 += "="
		} else {
			b64 += toBase64Digit(*i3 & 63)
		}
	}
	return b64
}

func toBase64VLQ(value int) string {
	if value < 0 {
		value = (-value << 1) + 1
	} else {
		value = value << 1
	}

	out := ""
	for {
		digit := value & 31
		value = value >> 5
		if value > 0 {
			digit = digit | 32
		}
		out += toBase64Digit(digit)
		if value <= 0 {
			break
		}
	}
	return out
}

func toBase64Digit(value int) string {
	if value < 0 || value >= 64 {
		panic("Can only encode value in the range [0, 63]")
	}
	return string(b64Digits[value])
}
