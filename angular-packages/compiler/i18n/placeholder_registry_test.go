package i18n

import "testing"

func TestPlaceholderRegistry(t *testing.T) {
	var reg *PlaceholderRegistry
	reset := func() {
		reg = NewPlaceholderRegistry()
	}

	t.Run("tag placeholder", func(t *testing.T) {
		t.Run("should generate names for well known tags", func(t *testing.T) {
			reset()
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{}, false); got != "START_PARAGRAPH" {
				t.Fatalf("start tag placeholder = %q", got)
			}
			if got := reg.GetCloseTagPlaceholderName("p"); got != "CLOSE_PARAGRAPH" {
				t.Fatalf("close tag placeholder = %q", got)
			}
		})

		t.Run("should generate names for custom tags", func(t *testing.T) {
			reset()
			if got := reg.GetStartTagPlaceholderName("my-cmp", map[string]string{}, false); got != "START_TAG_MY-CMP" {
				t.Fatalf("start tag placeholder = %q", got)
			}
			if got := reg.GetCloseTagPlaceholderName("my-cmp"); got != "CLOSE_TAG_MY-CMP" {
				t.Fatalf("close tag placeholder = %q", got)
			}
		})

		t.Run("should generate the same name for the same tag", func(t *testing.T) {
			reset()
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{}, false); got != "START_PARAGRAPH" {
				t.Fatalf("first placeholder = %q", got)
			}
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{}, false); got != "START_PARAGRAPH" {
				t.Fatalf("second placeholder = %q", got)
			}
		})

		t.Run("should be case sensitive for tag name", func(t *testing.T) {
			reset()
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{}, false); got != "START_PARAGRAPH" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetStartTagPlaceholderName("P", map[string]string{}, false); got != "START_PARAGRAPH_1" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetCloseTagPlaceholderName("p"); got != "CLOSE_PARAGRAPH" {
				t.Fatalf("close placeholder = %q", got)
			}
			if got := reg.GetCloseTagPlaceholderName("P"); got != "CLOSE_PARAGRAPH_1" {
				t.Fatalf("close placeholder = %q", got)
			}
		})

		t.Run("should generate the same name for the same tag with the same attributes", func(t *testing.T) {
			reset()
			attrs1 := map[string]string{"foo": "a", "bar": "b"}
			attrs2 := map[string]string{"bar": "b", "foo": "a"}
			for i, attrs := range []map[string]string{attrs1, attrs1, attrs2} {
				if got := reg.GetStartTagPlaceholderName("p", attrs, false); got != "START_PARAGRAPH" {
					t.Fatalf("call %d placeholder = %q", i, got)
				}
			}
		})

		t.Run("should generate different names for the same tag with different attributes", func(t *testing.T) {
			reset()
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{"foo": "a", "bar": "b"}, false); got != "START_PARAGRAPH" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{"foo": "a"}, false); got != "START_PARAGRAPH_1" {
				t.Fatalf("placeholder = %q", got)
			}
		})

		t.Run("should be case sensitive for attributes", func(t *testing.T) {
			reset()
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{"foo": "a", "bar": "b"}, false); got != "START_PARAGRAPH" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{"fOo": "a", "bar": "b"}, false); got != "START_PARAGRAPH_1" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{"fOo": "a", "bAr": "b"}, false); got != "START_PARAGRAPH_2" {
				t.Fatalf("placeholder = %q", got)
			}
		})

		t.Run("should support void tags", func(t *testing.T) {
			reset()
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{}, true); got != "PARAGRAPH" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{}, true); got != "PARAGRAPH" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetStartTagPlaceholderName("p", map[string]string{"other": "true"}, true); got != "PARAGRAPH_1" {
				t.Fatalf("placeholder = %q", got)
			}
		})
	})

	t.Run("arbitrary placeholders", func(t *testing.T) {
		t.Run("should generate the same name given the same name and content", func(t *testing.T) {
			reset()
			if got := reg.GetPlaceholderName("name", "content"); got != "NAME" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetPlaceholderName("name", "content"); got != "NAME" {
				t.Fatalf("placeholder = %q", got)
			}
		})

		t.Run("should generate a different name given different content", func(t *testing.T) {
			reset()
			if got := reg.GetPlaceholderName("name", "content1"); got != "NAME" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetPlaceholderName("name", "content2"); got != "NAME_1" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetPlaceholderName("name", "content3"); got != "NAME_2" {
				t.Fatalf("placeholder = %q", got)
			}
		})

		t.Run("should generate a different name given different names", func(t *testing.T) {
			reset()
			if got := reg.GetPlaceholderName("name1", "content"); got != "NAME1" {
				t.Fatalf("placeholder = %q", got)
			}
			if got := reg.GetPlaceholderName("name2", "content"); got != "NAME2" {
				t.Fatalf("placeholder = %q", got)
			}
		})
	})

	t.Run("block placeholders", func(t *testing.T) {
		t.Run("should generate placeholders for a plain block", func(t *testing.T) {
			reset()
			if got := reg.GetStartBlockPlaceholderName("if", nil); got != "START_BLOCK_IF" {
				t.Fatalf("start block placeholder = %q", got)
			}
			if got := reg.GetCloseBlockPlaceholderName("if"); got != "CLOSE_BLOCK_IF" {
				t.Fatalf("close block placeholder = %q", got)
			}
		})

		t.Run("should generate placeholders for a block with spaces in its name", func(t *testing.T) {
			reset()
			if got := reg.GetStartBlockPlaceholderName("else if", nil); got != "START_BLOCK_ELSE_IF" {
				t.Fatalf("start block placeholder = %q", got)
			}
			if got := reg.GetCloseBlockPlaceholderName("else if"); got != "CLOSE_BLOCK_ELSE_IF" {
				t.Fatalf("close block placeholder = %q", got)
			}
		})
	})
}
