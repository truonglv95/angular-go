package ml_parser_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/stretchr/testify/assert"
)

func TestNodeSerializer(t *testing.T) {
	var parser *ml_parser.HtmlParser

	setup := func() {
		parser = ml_parser.NewHtmlParser()
	}

	t.Run("should support element", func(t *testing.T) {
		setup()
		html := "<p></p>"
		ast := parser.Parse(html, "url", nil)
		assert.Equal(t, []string{html}, serializeNodes(ast.RootNodes))
	})

	t.Run("should support attributes", func(t *testing.T) {
		setup()
		html := "<p k=\"value\"></p>"
		ast := parser.Parse(html, "url", nil)
		assert.Equal(t, []string{html}, serializeNodes(ast.RootNodes))
	})

	t.Run("should support text", func(t *testing.T) {
		setup()
		html := "some text"
		ast := parser.Parse(html, "url", nil)
		assert.Equal(t, []string{html}, serializeNodes(ast.RootNodes))
	})

	t.Run("should support expansion", func(t *testing.T) {
		setup()
		html := "{number, plural, =0 {none} =1 {one} other {many}}"
		opts := &ml_parser.TokenizeOptions{TokenizeExpansionForms: true}
		ast := parser.Parse(html, "url", opts)
		assert.Equal(t, []string{html}, serializeNodes(ast.RootNodes))
	})

	t.Run("should support comment", func(t *testing.T) {
		setup()
		html := "<!--comment-->"
		opts := &ml_parser.TokenizeOptions{TokenizeExpansionForms: true}
		ast := parser.Parse(html, "url", opts)
		assert.Equal(t, []string{html}, serializeNodes(ast.RootNodes))
	})

	t.Run("should support nesting", func(t *testing.T) {
		setup()
		html := `<div i18n="meaning|desc">
        <span>{{ interpolation }}</span>
        <!--comment-->
        <p expansion="true">
          {number, plural, =0 {{sex, select, other {<b>?</b>}}}}
        </p>
      </div>`
		opts := &ml_parser.TokenizeOptions{TokenizeExpansionForms: true}
		ast := parser.Parse(html, "url", opts)
		assert.Equal(t, []string{html}, serializeNodes(ast.RootNodes))
	})
}
