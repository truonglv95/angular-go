package compiler

import (
	"reflect"
	"regexp"
	"testing"
)

func TestUtil(t *testing.T) {
	t.Run("splitAtColon", func(t *testing.T) {
		tests := []struct {
			name string
			in   string
			def  []string
			want []string
		}{
			{name: "should split when a single colon is present", in: "a:b", def: []string{}, want: []string{"a", "b"}},
			{name: "should trim parts", in: " a : b ", def: []string{}, want: []string{"a", "b"}},
			{name: "should support multiple colons", in: "a:b:c", def: []string{}, want: []string{"a", "b:c"}},
			{name: "should use the default value when no colon is present", in: "ab", def: []string{"c", "d"}, want: []string{"c", "d"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := SplitAtColon(tt.in, tt.def); !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("SplitAtColon(%q) = %#v, want %#v", tt.in, got, tt.want)
				}
			})
		}
	})

	t.Run("RegExp", func(t *testing.T) {
		if !regexp.MustCompile(EscapeRegExp("b")).MatchString("abc") {
			t.Fatal("escaped b should match abc")
		}
		if regexp.MustCompile(EscapeRegExp("b")).MatchString("adc") {
			t.Fatal("escaped b should not match adc")
		}
		if !regexp.MustCompile(EscapeRegExp("a.b")).MatchString("a.b") {
			t.Fatal("escaped a.b should match literal a.b")
		}
		if regexp.MustCompile(EscapeRegExp("a.b")).MatchString("axb") {
			t.Fatal("escaped a.b should not match axb")
		}
	})

	t.Run("utf8encode", func(t *testing.T) {
		w := func(bytes ...byte) string { return string(bytes) }
		tests := []struct {
			input string
			want  string
		}{
			{"abc", "abc"},
			{"\x00", "\x00"},
			{"\u0080", w(0xc2, 0x80)},
			{"\u05ca", w(0xd7, 0x8a)},
			{"\u07ff", w(0xdf, 0xbf)},
			{"\u0800", w(0xe0, 0xa0, 0x80)},
			{"\u2c3c", w(0xe2, 0xb0, 0xbc)},
			{"\uffff", w(0xef, 0xbf, 0xbf)},
			{"\U00010000", w(0xf0, 0x90, 0x80, 0x80)},
			{"\U0001d306", w(0xf0, 0x9d, 0x8c, 0x86)},
			{"\U0010ffff", w(0xf4, 0x8f, 0xbf, 0xbf)},
			{w(0xed, 0xa0, 0x80), w(0xed, 0xa0, 0x80)},
			{w(0xed, 0xa0, 0x80, 0xed, 0xa0, 0x80), w(0xed, 0xa0, 0x80, 0xed, 0xa0, 0x80)},
			{w(0xed, 0xa0, 0x80) + "A", w(0xed, 0xa0, 0x80) + "A"},
			{w(0xed, 0xa0, 0x80) + "\U0001d306" + w(0xed, 0xa0, 0x80), w(0xed, 0xa0, 0x80) + w(0xf0, 0x9d, 0x8c, 0x86) + w(0xed, 0xa0, 0x80)},
			{w(0xed, 0xa6, 0xaf), w(0xed, 0xa6, 0xaf)},
			{w(0xed, 0xaf, 0xbf), w(0xed, 0xaf, 0xbf)},
			{w(0xed, 0xb0, 0x80), w(0xed, 0xb0, 0x80)},
			{w(0xed, 0xb0, 0x80, 0xed, 0xb0, 0x80), w(0xed, 0xb0, 0x80, 0xed, 0xb0, 0x80)},
			{w(0xed, 0xb0, 0x80) + "A", w(0xed, 0xb0, 0x80) + "A"},
			{w(0xed, 0xb0, 0x80) + "\U0001d306" + w(0xed, 0xb0, 0x80), w(0xed, 0xb0, 0x80) + w(0xf0, 0x9d, 0x8c, 0x86) + w(0xed, 0xb0, 0x80)},
			{w(0xed, 0xbb, 0xae), w(0xed, 0xbb, 0xae)},
			{w(0xed, 0xbf, 0xbf), w(0xed, 0xbf, 0xbf)},
		}
		for _, tt := range tests {
			if got := string(Utf8Encode(tt.input)); got != tt.want {
				t.Fatalf("Utf8Encode(%q) = % x, want % x", tt.input, []byte(got), []byte(tt.want))
			}
		}
	})

	t.Run("stringify", func(t *testing.T) {
		if got := Stringify(map[string]any{}); got != "object" {
			t.Fatalf("Stringify(map[string]any{}) = %q, want object", got)
		}
	})
}
