package ml_parser_test

import (
	"reflect"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

func humanizeErrors(errors []error) []any {
	var result []any
	for _, e := range errors {
		if te, ok := e.(*ml_parser.TreeError); ok {
			var elemName string
			if te.ElementName != nil {
				elemName = *te.ElementName
			}
			result = append(result, []any{
				elemName,
				te.Msg,
				humanizeLineColumn(te.Span.Start),
			})
		} else if pe, ok := e.(*parse_util.ParseError); ok {
			result = append(result, []any{
				pe.Msg,
				humanizeLineColumn(pe.Span.Start),
			})
		} else {
			result = append(result, []any{
				e.Error(),
			})
		}
	}
	if result == nil {
		return []any{}
	}
	return result
}

func TestHtmlParser(t *testing.T) {
	parser := ml_parser.NewHtmlParser()

	t.Run("parse", func(t *testing.T) {
		t.Run("text nodes", func(t *testing.T) {
			t.Run("should parse root level text nodes", func(t *testing.T) {
				got := humanizeDom(parser.Parse("a", "TestComp", nil))
				expected := []any{
					[]any{"Text", "a", 0, []string{"a"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse text nodes inside regular elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div>a</div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Text", "a", 1, []string{"a"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse text nodes inside <ng-template> elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<ng-template>a</ng-template>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "ng-template", 0},
					[]any{"Text", "a", 1, []string{"a"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse CDATA", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<![CDATA[text]]>", "TestComp", nil))
				expected := []any{
					[]any{"Text", "text", 0, []string{"text"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse text nodes with HTML entities (5+ hex digits)", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div>&#x1F6C8;</div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Text", "\U0001F6C8", 1, []string{""}, []string{"\U0001F6C8", "&#x1F6C8;"}, []string{""}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse text nodes with decimal HTML entities (5+ digits)", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div>&#128712;</div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Text", "\U0001F6C8", 1, []string{""}, []string{"\U0001F6C8", "&#128712;"}, []string{""}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse named HTML entities containing digits", func(t *testing.T) {
				got1 := humanizeDom(parser.Parse("<div>&sup1;</div>", "TestComp", nil))
				expected1 := []any{
					[]any{"Element", "div", 0},
					[]any{"Text", "\u00B9", 1, []string{""}, []string{"\u00B9", "&sup1;"}, []string{""}},
				}
				if !reflect.DeepEqual(got1, expected1) {
					t.Errorf("got %v, want %v", got1, expected1)
				}

				got2 := humanizeDom(parser.Parse("<div>&frac12;</div>", "TestComp", nil))
				expected2 := []any{
					[]any{"Element", "div", 0},
					[]any{"Text", "\u00BD", 1, []string{""}, []string{"\u00BD", "&frac12;"}, []string{""}},
				}
				if !reflect.DeepEqual(got2, expected2) {
					t.Errorf("got %v, want %v", got2, expected2)
				}
			})

			t.Run("should normalize line endings within CDATA", func(t *testing.T) {
				parsed := parser.Parse("<![CDATA[ line 1 \r\n line 2 ]]>", "TestComp", nil)
				got := humanizeDom(parsed)
				expected := []any{
					[]any{"Text", " line 1 \n line 2 ", 0, []string{" line 1 \n line 2 "}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
				if len(parsed.Errors) != 0 {
					t.Errorf("expected no errors, got %v", parsed.Errors)
				}
			})
		})

		t.Run("elements", func(t *testing.T) {
			t.Run("should parse root level elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse elements inside of regular elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div><span></span></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Element", "span", 1},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse elements inside <ng-template> elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<ng-template><span></span></ng-template>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "ng-template", 0},
					[]any{"Element", "span", 1},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should support void elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<link rel=\"author license\" href=\"/about\">", "TestComp", nil))
				expected := []any{
					[]any{"Element", "link", 0},
					[]any{"Attribute", "rel", "author license", []string{"author license"}},
					[]any{"Attribute", "href", "/about", []string{"/about"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should indicate whether an element is void", func(t *testing.T) {
				res := parser.Parse("<input><div></div>", "TestComp", nil)
				nodes := res.RootNodes
				el0 := nodes[0].(*ml_parser.Element)
				el1 := nodes[1].(*ml_parser.Element)
				if el0.Name != "input" || !el0.IsVoid {
					t.Errorf("expected input void=true, got name=%s void=%t", el0.Name, el0.IsVoid)
				}
				if el1.Name != "div" || el1.IsVoid {
					t.Errorf("expected div void=false, got name=%s void=%t", el1.Name, el1.IsVoid)
				}
			})

			t.Run("should not error on void elements from HTML5 spec", func(t *testing.T) {
				htmlList := []string{
					"<map><area></map>",
					"<div><br></div>",
					"<colgroup><col></colgroup>",
					"<div><embed></div>",
					"<div><hr></div>",
					"<div><img></div>",
					"<div><input></div>",
					"<object><param>/<object>",
					"<audio><source></audio>",
					"<audio><track></audio>",
					"<p><wbr></p>",
				}
				for _, h := range htmlList {
					res := parser.Parse(h, "TestComp", nil)
					if len(res.Errors) > 0 {
						t.Errorf("expected no errors for %s, got %v", h, res.Errors)
					}
				}
			})

			t.Run("should close void elements on text nodes", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<p>before<br>after</p>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "p", 0},
					[]any{"Text", "before", 1, []string{"before"}},
					[]any{"Element", "br", 1},
					[]any{"Text", "after", 1, []string{"after"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should support optional end tags", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div><p>1<p>2</div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Element", "p", 1},
					[]any{"Text", "1", 2, []string{"1"}},
					[]any{"Element", "p", 1},
					[]any{"Text", "2", 2, []string{"2"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should support nested elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<ul><li><ul><li></li></ul></li></ul>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "ul", 0},
					[]any{"Element", "li", 1},
					[]any{"Element", "ul", 2},
					[]any{"Element", "li", 3},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should not wraps elements in a required parent", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div><tr></tr></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Element", "tr", 1},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should support explicit namespace", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<myns:div></myns:div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", ":myns:div", 0},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should support implicit namespace", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<svg></svg>", "TestComp", nil))
				expected := []any{
					[]any{"Element", ":svg:svg", 0},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should propagate the namespace", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<myns:div><p></p></myns:div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", ":myns:div", 0},
					[]any{"Element", ":myns:p", 1},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should match closing tags case sensitive", func(t *testing.T) {
				res := parser.Parse("<DiV><P></p></dIv>", "TestComp", nil)
				gotErrors := humanizeErrors(res.Errors)
				expectedErrors := []any{
					[]any{
						"p",
						"Unexpected closing tag \"p\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags",
						"0:8",
					},
					[]any{
						"dIv",
						"Unexpected closing tag \"dIv\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags",
						"0:12",
					},
				}
				if !reflect.DeepEqual(gotErrors, expectedErrors) {
					t.Errorf("got %v, want %v", gotErrors, expectedErrors)
				}
			})

			t.Run("should support self closing void elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<input />", "TestComp", nil))
				expected := []any{
					[]any{"Element", "input", 0, "#selfClosing"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should support self closing foreign elements", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<math />", "TestComp", nil))
				expected := []any{
					[]any{"Element", ":math:math", 0, "#selfClosing"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should ignore LF immediately after textarea, pre and listing", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<p>\n</p><textarea>\n</textarea><pre>\n\n</pre><listing>\n\n</listing>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "p", 0},
					[]any{"Text", "\n", 1, []string{"\n"}},
					[]any{"Element", "textarea", 0},
					[]any{"Element", "pre", 0},
					[]any{"Text", "\n", 1, []string{"\n"}},
					[]any{"Element", "listing", 0},
					[]any{"Text", "\n", 1, []string{"\n"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should normalize line endings in text", func(t *testing.T) {
				p1 := parser.Parse("<title> line 1 \r\n line 2 </title>", "TestComp", nil)
				got1 := humanizeDom(p1)
				expected1 := []any{
					[]any{"Element", "title", 0},
					[]any{"Text", " line 1 \n line 2 ", 1, []string{" line 1 \n line 2 "}},
				}
				if !reflect.DeepEqual(got1, expected1) {
					t.Errorf("got %v, want %v", got1, expected1)
				}
				if len(p1.Errors) > 0 {
					t.Errorf("unexpected errors: %v", p1.Errors)
				}

				p2 := parser.Parse("<script> line 1 \r\n line 2 </script>", "TestComp", nil)
				got2 := humanizeDom(p2)
				expected2 := []any{
					[]any{"Element", "script", 0},
					[]any{"Text", " line 1 \n line 2 ", 1, []string{" line 1 \n line 2 "}},
				}
				if !reflect.DeepEqual(got2, expected2) {
					t.Errorf("got %v, want %v", got2, expected2)
				}
				if len(p2.Errors) > 0 {
					t.Errorf("unexpected errors: %v", p2.Errors)
				}

				p3 := parser.Parse("<div> line 1 \r\n line 2 </div>", "TestComp", nil)
				got3 := humanizeDom(p3)
				expected3 := []any{
					[]any{"Element", "div", 0},
					[]any{"Text", " line 1 \n line 2 ", 1, []string{" line 1 \n line 2 "}},
				}
				if !reflect.DeepEqual(got3, expected3) {
					t.Errorf("got %v, want %v", got3, expected3)
				}
				if len(p3.Errors) > 0 {
					t.Errorf("unexpected errors: %v", p3.Errors)
				}

				p4 := parser.Parse("<span> line 1 \r\n line 2 </span>", "TestComp", nil)
				got4 := humanizeDom(p4)
				expected4 := []any{
					[]any{"Element", "span", 0},
					[]any{"Text", " line 1 \n line 2 ", 1, []string{" line 1 \n line 2 "}},
				}
				if !reflect.DeepEqual(got4, expected4) {
					t.Errorf("got %v, want %v", got4, expected4)
				}
				if len(p4.Errors) > 0 {
					t.Errorf("unexpected errors: %v", p4.Errors)
				}
			})

			t.Run("should parse element with JavaScript keyword tag name", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<constructor></constructor>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "constructor", 0},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("attributes", func(t *testing.T) {
			t.Run("should parse attributes on regular elements case sensitive", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div kEy=\"v\" key2=v2></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Attribute", "kEy", "v", []string{"v"}},
					[]any{"Attribute", "key2", "v2", []string{"v2"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse attributes containing interpolation", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div foo=\"1{{message}}2\"></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Attribute", "foo", "1{{message}}2", []string{"1"}, []string{"{{", "message", "}}"}, []string{"2"}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse attributes containing unquoted interpolation", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div foo={{message}}></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Attribute", "foo", "{{message}}", []string{""}, []string{"{{", "message", "}}"}, []string{""}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse bound inputs with expressions containing newlines", func(t *testing.T) {
				html := "<app-component\n                        [attr]=\"[\n                        {text: 'some text',url:'//www.google.com'},\n                        {text:'other text',url:'//www.google.com'}]\"></app-component>"
				got := humanizeDom(parser.Parse(html, "TestComp", nil))
				expected := []any{
					[]any{"Element", "app-component", 0},
					[]any{
						"Attribute",
						"[attr]",
						"[\n                        {text: 'some text',url:'//www.google.com'},\n                        {text:'other text',url:'//www.google.com'}]",
						[]string{"[\n                        {text: 'some text',url:'//www.google.com'},\n                        {text:'other text',url:'//www.google.com'}]"},
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse attributes containing encoded entities", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div foo=\"&amp;\"></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Attribute", "foo", "&", []string{""}, []string{"&", "&amp;"}, []string{""}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse attributes containing encoded entities (5+ hex digits)", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div foo=\"&#x1F6C8;\"></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Attribute", "foo", "\U0001F6C8", []string{""}, []string{"\U0001F6C8", "&#x1F6C8;"}, []string{""}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should parse attributes containing encoded decimal entities (5+ digits)", func(t *testing.T) {
				got := humanizeDom(parser.Parse("<div foo=\"&#128712;\"></div>", "TestComp", nil))
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Attribute", "foo", "\U0001F6C8", []string{""}, []string{"\U0001F6C8", "&#128712;"}, []string{""}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should decode HTML entities in interpolated attributes", func(t *testing.T) {
				got := humanizeDomSourceSpans(parser.Parse("<div foo=\"{{&amp;}}\"></div>", "TestComp", nil))
				expected := []any{
					[]any{
						"Element",
						"div",
						0,
						"<div foo=\"{{&amp;}}\"></div>",
						"<div foo=\"{{&amp;}}\">",
						"</div>",
					},
					[]any{"Attribute", "foo", "{{&}}", []string{""}, []string{"{{", "&amp;", "}}"}, []string{""}, "foo=\"{{&amp;}}\""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should normalize line endings within attribute values", func(t *testing.T) {
				res := parser.Parse("<div key=\"  \r\n line 1 \r\n   line 2  \"></div>", "TestComp", nil)
				got := humanizeDom(res)
				expected := []any{
					[]any{"Element", "div", 0},
					[]any{"Attribute", "key", "  \n line 1 \n   line 2  ", []string{"  \n line 1 \n   line 2  "}},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
				if len(res.Errors) > 0 {
					t.Errorf("unexpected errors: %v", res.Errors)
				}
			})
		})
	})
}
