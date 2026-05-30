package phases

import (
	"reflect"
	"testing"
)

func TestStyleParsing(t *testing.T) {
	t.Run("should parse empty or blank strings", func(t *testing.T) {
		assertParsedStyles(t, "", []string{})
		assertParsedStyles(t, "    ", []string{})
	})

	t.Run("should parse a string into a key/value map", func(t *testing.T) {
		assertParsedStyles(t, "width:100px;height:200px;opacity:0", []string{
			"width", "100px", "height", "200px", "opacity", "0",
		})
	})

	t.Run("should allow empty values", func(t *testing.T) {
		assertParsedStyles(t, "width:;height:   ;", []string{"width", "", "height", ""})
	})

	t.Run("should trim values and properties", func(t *testing.T) {
		assertParsedStyles(t, "width :333px ; height:666px    ; opacity: 0.5;", []string{
			"width", "333px", "height", "666px", "opacity", "0.5",
		})
	})

	t.Run("should not mess up with quoted strings that contain [:;] values", func(t *testing.T) {
		assertParsedStyles(t, `content: "foo; man: guy"; width: 100px`, []string{
			"content", `"foo; man: guy"`, "width", "100px",
		})
	})

	t.Run("should not mess up with quoted strings that contain inner quote values", func(t *testing.T) {
		quoteStr := `"one 'two' three "four" five"`
		assertParsedStyles(t, "content: "+quoteStr+"; width: 123px", []string{
			"content", quoteStr, "width", "123px",
		})
	})

	t.Run("should respect parenthesis that are placed within a style", func(t *testing.T) {
		assertParsedStyles(t, `background-image: url("foo.jpg")`, []string{
			"background-image", `url("foo.jpg")`,
		})
	})

	t.Run("should respect multi-level parenthesis that contain special [:;] characters", func(t *testing.T) {
		assertParsedStyles(t, "color: rgba(calc(50 * 4), var(--cool), :5;); height: 100px;", []string{
			"color", "rgba(calc(50 * 4), var(--cool), :5;)", "height", "100px",
		})
	})

	t.Run("should hyphenate style properties from camel case", func(t *testing.T) {
		assertParsedStyles(t, "borderWidth: 200px", []string{"border-width", "200px"})
	})

	t.Run("should not remove quotes from string data types", func(t *testing.T) {
		assertParsedStyles(t, `content: "foo"`, []string{"content", `"foo"`})
	})

	t.Run("should not remove quotes that changes the value context from invalid to valid", func(t *testing.T) {
		assertParsedStyles(t, `width: "1px"`, []string{"width", `"1px"`})
	})
}

func TestStyleHyphenate(t *testing.T) {
	t.Run("should convert a camel-cased value to a hyphenated value", func(t *testing.T) {
		tests := map[string]string{
			"fooBar":      "foo-bar",
			"fooBarMan":   "foo-bar-man",
			"-fooBar-man": "-foo-bar-man",
		}
		for input, want := range tests {
			if got := hyphenate(input); got != want {
				t.Fatalf("hyphenate(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("should make everything lowercase", func(t *testing.T) {
		if got := hyphenate("-WebkitAnimation"); got != "-webkit-animation" {
			t.Fatalf("hyphenate(%q) = %q, want %q", "-WebkitAnimation", got, "-webkit-animation")
		}
	})
}

func assertParsedStyles(t *testing.T, input string, want []string) {
	t.Helper()
	if got := parseStyles(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseStyles(%q) = %#v, want %#v", input, got, want)
	}
}
