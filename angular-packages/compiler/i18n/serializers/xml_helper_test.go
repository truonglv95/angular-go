package serializers

import "testing"

func TestXMLHelper(t *testing.T) {
	t.Run("should serialize XML declaration", func(t *testing.T) {
		if got := Serialize([]Node{NewDeclaration(map[string]string{"version": "1.0"})}); got != `<?xml version="1.0" ?>` {
			t.Fatalf("serialize declaration = %q", got)
		}
	})

	t.Run("should serialize text node", func(t *testing.T) {
		if got := Serialize([]Node{NewText("foo bar")}); got != "foo bar" {
			t.Fatalf("serialize text = %q", got)
		}
	})

	t.Run("should escape text nodes", func(t *testing.T) {
		if got := Serialize([]Node{NewText("<>")}); got != "&lt;&gt;" {
			t.Fatalf("escaped text = %q", got)
		}
	})

	t.Run("should serialize xml nodes without children", func(t *testing.T) {
		if got := Serialize([]Node{NewTag("el", map[string]string{"foo": "bar"}, nil)}); got != `<el foo="bar"/>` {
			t.Fatalf("serialize tag = %q", got)
		}
	})

	t.Run("should serialize xml nodes with children", func(t *testing.T) {
		got := Serialize([]Node{
			NewTag("parent", map[string]string{}, []Node{
				NewTag("child", map[string]string{}, []Node{NewText("content")}),
			}),
		})
		if got != "<parent><child>content</child></parent>" {
			t.Fatalf("serialize nested tag = %q", got)
		}
	})

	t.Run("should serialize node lists", func(t *testing.T) {
		got := Serialize([]Node{
			NewTag("el", map[string]string{"order": "0"}, nil),
			NewTag("el", map[string]string{"order": "1"}, nil),
		})
		if got != `<el order="0"/><el order="1"/>` {
			t.Fatalf("serialize node list = %q", got)
		}
	})

	t.Run("should escape attribute values", func(t *testing.T) {
		got := Serialize([]Node{NewTag("el", map[string]string{"foo": `<>\"`}, nil)})
		if got != `<el foo="&lt;&gt;\&quot;"/>` {
			t.Fatalf("serialize escaped attrs = %q", got)
		}
	})
}
