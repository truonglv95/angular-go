package output

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptrStr(s string) *string { return &s }
func ptrInt(i int) *int       { return &i }

func TestSourceMapGenerator(t *testing.T) {
	t.Run("generation", func(t *testing.T) {
		t.Run("should generate a valid source map", func(t *testing.T) {
			outJs := "out.js"
			mapGen := NewSourceMapGenerator(&outJs).
				AddSource("a.js", nil).
				AddLine().
				AddMapping(0, ptrStr("a.js"), ptrInt(0), ptrInt(0)).
				AddMapping(4, ptrStr("a.js"), ptrInt(0), ptrInt(6)).
				AddMapping(5, ptrStr("a.js"), ptrInt(0), ptrInt(7)).
				AddMapping(8, ptrStr("a.js"), ptrInt(0), ptrInt(22)).
				AddMapping(9, ptrStr("a.js"), ptrInt(0), ptrInt(23)).
				AddMapping(10, ptrStr("a.js"), ptrInt(0), ptrInt(24)).
				AddLine().
				AddMapping(0, ptrStr("a.js"), ptrInt(1), ptrInt(0)).
				AddMapping(4, ptrStr("a.js"), ptrInt(1), ptrInt(6)).
				AddMapping(5, ptrStr("a.js"), ptrInt(1), ptrInt(7)).
				AddMapping(8, ptrStr("a.js"), ptrInt(1), ptrInt(10)).
				AddMapping(9, ptrStr("a.js"), ptrInt(1), ptrInt(11)).
				AddMapping(10, ptrStr("a.js"), ptrInt(1), ptrInt(12)).
				AddLine().
				AddMapping(0, ptrStr("a.js"), ptrInt(3), ptrInt(0)).
				AddMapping(2, ptrStr("a.js"), ptrInt(3), ptrInt(2)).
				AddMapping(3, ptrStr("a.js"), ptrInt(3), ptrInt(3)).
				AddMapping(10, ptrStr("a.js"), ptrInt(3), ptrInt(10)).
				AddMapping(11, ptrStr("a.js"), ptrInt(3), ptrInt(11)).
				AddMapping(21, ptrStr("a.js"), ptrInt(3), ptrInt(11)).
				AddMapping(22, ptrStr("a.js"), ptrInt(3), ptrInt(12)).
				AddLine().
				AddMapping(4, ptrStr("a.js"), ptrInt(4), ptrInt(4)).
				AddMapping(11, ptrStr("a.js"), ptrInt(4), ptrInt(11)).
				AddMapping(12, ptrStr("a.js"), ptrInt(4), ptrInt(12)).
				AddMapping(15, ptrStr("a.js"), ptrInt(4), ptrInt(15)).
				AddMapping(16, ptrStr("a.js"), ptrInt(4), ptrInt(16)).
				AddMapping(21, ptrStr("a.js"), ptrInt(4), ptrInt(21)).
				AddMapping(22, ptrStr("a.js"), ptrInt(4), ptrInt(22)).
				AddMapping(23, ptrStr("a.js"), ptrInt(4), ptrInt(23)).
				AddLine().
				AddMapping(0, ptrStr("a.js"), ptrInt(5), ptrInt(0)).
				AddMapping(1, ptrStr("a.js"), ptrInt(5), ptrInt(1)).
				AddMapping(2, ptrStr("a.js"), ptrInt(5), ptrInt(2)).
				AddMapping(3, ptrStr("a.js"), ptrInt(5), ptrInt(2))

			assert.Equal(t,
				"AAAA,IAAM,CAAC,GAAe,CAAC,CAAC;AACxB,IAAM,CAAC,GAAG,CAAC,CAAC;AAEZ,EAAE,CAAC,OAAO,CAAC,UAAA,CAAC;IACR,OAAO,CAAC,GAAG,CAAC,KAAK,CAAC,CAAC;AACvB,CAAC,CAAC,CAAA",
				mapGen.ToJSON().Mappings)
		})

		t.Run("should include the files and their contents", func(t *testing.T) {
			inline := "inline"
			outJs := "out.js"
			mapGen := NewSourceMapGenerator(&outJs).
				AddSource("inline.ts", &inline).
				AddSource("inline.ts", &inline).
				AddSource("url.ts", nil).
				AddLine().
				AddMapping(0, ptrStr("inline.ts"), ptrInt(0), ptrInt(0)).
				ToJSON()

			assert.Equal(t, "out.js", mapGen.File)
			assert.Equal(t, []string{"inline.ts", "url.ts"}, mapGen.Sources)
			assert.Equal(t, []*string{&inline, nil}, mapGen.SourcesContent)
		})

		t.Run("should not generate source maps when there is no mapping", func(t *testing.T) {
			outJs := "out.js"
			inline := "inline"
			smg := NewSourceMapGenerator(&outJs).AddSource("inline.ts", &inline).AddLine()
			assert.Nil(t, smg.ToJSON())
			assert.Equal(t, "", smg.ToJsComment())
		})
	})

	t.Run("encodeB64String", func(t *testing.T) {
		t.Run("should return the b64 encoded value", func(t *testing.T) {
			cases := []struct {
				src string
				b64 string
			}{
				{"", ""},
				{"a", "YQ=="},
				{"Foo", "Rm9v"},
				{"Foo1", "Rm9vMQ=="},
				{"Foo12", "Rm9vMTI="},
				{"Foo123", "Rm9vMTIz"},
			}
			for _, tc := range cases {
				assert.Equal(t, tc.b64, base64.StdEncoding.EncodeToString([]byte(tc.src)))
			}
		})
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("should throw when mappings are added out of order", func(t *testing.T) {
			outJs := "out.js"
			assert.PanicsWithValue(t, "Mapping should be added in output order", func() {
				NewSourceMapGenerator(&outJs).
					AddSource("in.js", nil).
					AddLine().
					AddMapping(10, ptrStr("in.js"), ptrInt(0), ptrInt(0)).
					AddMapping(0, ptrStr("in.js"), ptrInt(0), ptrInt(0))
			})
		})

		t.Run("should throw when adding segments before any line is created", func(t *testing.T) {
			outJs := "out.js"
			assert.PanicsWithValue(t, "A line must be added before mappings can be added", func() {
				NewSourceMapGenerator(&outJs).
					AddSource("in.js", nil).
					AddMapping(0, ptrStr("in.js"), ptrInt(0), ptrInt(0))
			})
		})

		t.Run("should throw when adding segments referencing unknown sources", func(t *testing.T) {
			outJs := "out.js"
			assert.PanicsWithValue(t, "Unknown source file \"in_.js\"", func() {
				NewSourceMapGenerator(&outJs).
					AddSource("in.js", nil).
					AddLine().
					AddMapping(0, ptrStr("in_.js"), ptrInt(0), ptrInt(0))
			})
		})
	})
}
