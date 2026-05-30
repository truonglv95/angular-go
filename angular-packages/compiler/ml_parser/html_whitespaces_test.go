package ml_parser_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/stretchr/testify/assert"
)

func parseAndRemoveWS(template string, options *ml_parser.TokenizeOptions) []any {
	parser := ml_parser.NewHtmlParser()
	parsed := parser.Parse(template, "TestComp", options)
	// removeWhitespaces returns the array of nodes (modified)
	return humanizeDom(ml_parser.RemoveWhitespaces(parsed, true))
}

func TestRemoveWhitespaces(t *testing.T) {
	t.Run("should remove blank text nodes", func(t *testing.T) {
		assert.Equal(t, []any{}, parseAndRemoveWS(" ", nil))
		assert.Equal(t, []any{}, parseAndRemoveWS("\n", nil))
		assert.Equal(t, []any{}, parseAndRemoveWS("\t", nil))
		assert.Equal(t, []any{}, parseAndRemoveWS("    \t    \n ", nil))
	})

	t.Run("should remove whitespaces (space, tab, new line) between elements", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "br", 0},
			[]any{"Element", "br", 0},
			[]any{"Element", "br", 0},
			[]any{"Element", "br", 0},
		}, parseAndRemoveWS("<br>  <br>\t<br>\n<br>", nil))
	})

	t.Run("should remove whitespaces from child text nodes", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "div", 0},
			[]any{"Element", "span", 1},
		}, parseAndRemoveWS("<div><span> </span></div>", nil))
	})

	t.Run("should remove whitespaces from the beginning and end of a template", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "br", 0},
		}, parseAndRemoveWS(" <br>\t", nil))
	})

	t.Run("should convert &ngsp; to a space and preserve it", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "div", 0},
			[]any{"Element", "span", 1},
			[]any{"Text", "foo", 2, []string{"foo"}},
			[]any{"Text", " ", 1, []string{""}, []string{ml_parser.NGSP_UNICODE, "&ngsp;"}, []string{""}},
			[]any{"Element", "span", 1},
			[]any{"Text", "bar", 2, []string{"bar"}},
		}, parseAndRemoveWS("<div><span>foo</span>&ngsp;<span>bar</span></div>", nil))
	})

	t.Run("should replace multiple whitespaces with one space", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Text", " foo ", 0, []string{" foo "}},
		}, parseAndRemoveWS("\n\n\nfoo\t\t\t", nil))
		assert.Equal(t, []any{
			[]any{"Text", " foo ", 0, []string{" foo "}},
		}, parseAndRemoveWS("   \n foo  \t ", nil))
	})

	t.Run("should remove whitespace inside of blocks", func(t *testing.T) {
		markup := "@if (cond) {<br>  <br>\t<br>\n<br>}"
		assert.Equal(t, []any{
			[]any{"Block", "if", 0},
			[]any{"BlockParameter", "cond"},
			[]any{"Element", "br", 1},
			[]any{"Element", "br", 1},
			[]any{"Element", "br", 1},
			[]any{"Element", "br", 1},
		}, parseAndRemoveWS(markup, nil))
	})

	t.Run("should not replace &nbsp;", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Text", "\u00a0", 0, []string{""}, []string{"\u00a0", "&nbsp;"}, []string{""}},
		}, parseAndRemoveWS("&nbsp;", nil))
	})

	t.Run("should not replace sequences of &nbsp;", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Text", "\u00a0\u00a0foo\u00a0\u00a0", 0, []string{""}, []string{"\u00a0", "&nbsp;"}, []string{""}, []string{"\u00a0", "&nbsp;"}, []string{"foo"}, []string{"\u00a0", "&nbsp;"}, []string{""}, []string{"\u00a0", "&nbsp;"}, []string{""}},
		}, parseAndRemoveWS("&nbsp;&nbsp;foo&nbsp;&nbsp;", nil))
	})

	t.Run("should not replace single tab and newline with spaces", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Text", "\nfoo", 0, []string{"\nfoo"}},
		}, parseAndRemoveWS("\nfoo", nil))
		assert.Equal(t, []any{
			[]any{"Text", "\tfoo", 0, []string{"\tfoo"}},
		}, parseAndRemoveWS("\tfoo", nil))
	})

	t.Run("should preserve single whitespaces between interpolations", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Text", "{{fooExp}} {{barExp}}", 0, []string{""}, []string{"{{", "fooExp", "}}"}, []string{" "}, []string{"{{", "barExp", "}}"}, []string{""}},
		}, parseAndRemoveWS("{{fooExp}} {{barExp}}", nil))
		assert.Equal(t, []any{
			[]any{"Text", "{{fooExp}}\t{{barExp}}", 0, []string{""}, []string{"{{", "fooExp", "}}"}, []string{"\t"}, []string{"{{", "barExp", "}}"}, []string{""}},
		}, parseAndRemoveWS("{{fooExp}}\t{{barExp}}", nil))
		assert.Equal(t, []any{
			[]any{"Text", "{{fooExp}}\n{{barExp}}", 0, []string{""}, []string{"{{", "fooExp", "}}"}, []string{"\n"}, []string{"{{", "barExp", "}}"}, []string{""}},
		}, parseAndRemoveWS("{{fooExp}}\n{{barExp}}", nil))
	})

	t.Run("should preserve whitespaces around interpolations", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Text", " {{exp}} ", 0, []string{" "}, []string{"{{", "exp", "}}"}, []string{" "}},
		}, parseAndRemoveWS(" {{exp}} ", nil))
	})

	t.Run("should preserve whitespaces around ICU expansions", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "span", 0},
			[]any{"Text", " ", 1, []string{" "}},
			[]any{"Expansion", "a", "b", 1},
			[]any{"ExpansionCase", "=4", 2},
			[]any{"Text", " ", 1, []string{" "}},
		}, parseAndRemoveWS("<span> {a, b, =4 {c}} </span>", &ml_parser.TokenizeOptions{TokenizeExpansionForms: true}))
	})

	t.Run("should preserve whitespaces inside <pre> elements", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "pre", 0},
			[]any{"Element", "strong", 1},
			[]any{"Text", "foo", 2, []string{"foo"}},
			[]any{"Text", "\n", 1, []string{"\n"}},
			[]any{"Element", "strong", 1},
			[]any{"Text", "bar", 2, []string{"bar"}},
		}, parseAndRemoveWS("<pre><strong>foo</strong>\n<strong>bar</strong></pre>", nil))
	})

	t.Run("should skip whitespace trimming in <textarea>", func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "textarea", 0},
			[]any{"Text", "foo\n\n  bar", 1, []string{"foo\n\n  bar"}},
		}, parseAndRemoveWS("<textarea>foo\n\n  bar</textarea>", nil))
	})

	t.Run("should preserve whitespaces inside elements annotated with "+ml_parser.PRESERVE_WS_ATTR_NAME, func(t *testing.T) {
		assert.Equal(t, []any{
			[]any{"Element", "div", 0},
			[]any{"Element", "img", 1},
			[]any{"Text", " ", 1, []string{" "}},
			[]any{"Element", "img", 1},
		}, parseAndRemoveWS("<div "+ml_parser.PRESERVE_WS_ATTR_NAME+"><img> <img></div>", nil))
	})
}
