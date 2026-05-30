package ml_parser_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
	"github.com/stretchr/testify/assert"
)

func TestInlineCommentsInAttributes(t *testing.T) {
	parser := ml_parser.NewParser(func(tagName string) tags.TagDefinition {
		return &mockTagDefinition{}
	})

	t.Run("should ignore single line comments between attributes", func(t *testing.T) {
		source := `
      <div 
        // comment 1
        attr1="value1"
        // comment 2
        attr2="value2"
      ></div>
    `
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 2, len(element.Attrs))
		assert.Equal(t, "attr1", element.Attrs[0].Name)
		assert.Equal(t, "attr2", element.Attrs[1].Name)
	})

	t.Run("should ignore single line comments between inputs and outputs", func(t *testing.T) {
		source := `
      <div 
        // comment 1
        [input]="value1"
        // comment 2
        (output)="handler()"
      ></div>
    `
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 2, len(element.Attrs))
		assert.Equal(t, "[input]", element.Attrs[0].Name)
		assert.Equal(t, "(output)", element.Attrs[1].Name)
	})

	t.Run("should ignore single line comments at the end of tag", func(t *testing.T) {
		source := `<div attr1="value1" // comment 
    ></div>`
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 1, len(element.Attrs))
		assert.Equal(t, "attr1", element.Attrs[0].Name)
	})

	t.Run("should handle commented out attribute", func(t *testing.T) {
		source := `<div /* attr1="value1" */ attr2="value2"></div>`
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 1, len(element.Attrs))
		assert.Equal(t, "attr2", element.Attrs[0].Name)
	})

	t.Run("should comment an attribute with a // on a new line", func(t *testing.T) {
		source := `<div
    // attr1="value1"
    attr2="value2"></div>`
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 1, len(element.Attrs))
		assert.Equal(t, "attr2", element.Attrs[0].Name)
	})

	t.Run("should ignore multi-line comments between attributes", func(t *testing.T) {
		source := `
      <div 
        /* comment 1 */
        attr1="value1"
        /* 
           comment 2 
           spanning multiple lines
        */
        attr2="value2"
      ></div>
    `
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 2, len(element.Attrs))
		assert.Equal(t, "attr1", element.Attrs[0].Name)
		assert.Equal(t, "attr2", element.Attrs[1].Name)
	})

	t.Run("should ignore multi-line comments at the end of tag", func(t *testing.T) {
		source := `<div attr1="value1" /* comment */ ></div>`
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 1, len(element.Attrs))
		assert.Equal(t, "attr1", element.Attrs[0].Name)
	})

	t.Run("should handle * inside multi-line comments", func(t *testing.T) {
		source := `<div attr1="value1" /* comment with * inside */ attr2="value2"></div>`
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 2, len(element.Attrs))
		assert.Equal(t, "attr1", element.Attrs[0].Name)
		assert.Equal(t, "attr2", element.Attrs[1].Name)
	})

	t.Run("should maintain correct source spans with comments", func(t *testing.T) {
		source := `<div attr1="a" /* comment */ attr2="b"></div>`
		result := parser.Parse(source, "url", nil)
		assert.Equal(t, 0, len(result.Errors))
		elements := findElements(result.RootNodes)
		assert.Equal(t, 1, len(elements))
		element := elements[0]
		assert.Equal(t, 2, len(element.Attrs))

		attr1 := element.Attrs[0]
		assert.Equal(t, "attr1", attr1.Name)
		assert.Equal(t, 5, attr1.SourceSpan.Start.Offset)
		assert.Equal(t, 14, attr1.SourceSpan.End.Offset)

		attr2 := element.Attrs[1]
		assert.Equal(t, "attr2", attr2.Name)
		assert.Equal(t, 29, attr2.SourceSpan.Start.Offset)
		assert.Equal(t, 38, attr2.SourceSpan.End.Offset)
	})
}

type mockTagDefinition struct{}

func (m *mockTagDefinition) ClosedByParent() bool                  { return false }
func (m *mockTagDefinition) ImplicitNamespacePrefix() *string      { return nil }
func (m *mockTagDefinition) IsVoid() bool                          { return false }
func (m *mockTagDefinition) IgnoreFirstLf() bool                   { return false }
func (m *mockTagDefinition) CanSelfClose() bool                    { return false }
func (m *mockTagDefinition) PreventNamespaceInheritance() bool     { return false }
func (m *mockTagDefinition) IsClosedByChild(name string) bool      { return false }
func (m *mockTagDefinition) GetContentType(prefix *string) tags.TagContentType {
	return tags.TagContentTypeParsableData
}

func findElements(nodes []ml_parser.Node) []*ml_parser.Element {
	var elements []*ml_parser.Element
	for _, node := range nodes {
		if el, ok := node.(*ml_parser.Element); ok {
			elements = append(elements, el)
		}
	}
	return elements
}
