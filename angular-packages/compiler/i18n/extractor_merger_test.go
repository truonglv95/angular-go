package i18n

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
)

type htmlSerializerVisitor struct{}

func (v *htmlSerializerVisitor) VisitElement(element *ml_parser.Element, context any) any {
	attrsStr := v.visitAllAttrs(element.Attrs, " ", " ")
	tagDef := ml_parser.GetHtmlTagDefinition(element.Name)
	if tagDef.IsVoid() {
		return fmt.Sprintf("<%s%s/>", element.Name, attrsStr)
	}
	return fmt.Sprintf("<%s%s>%s</%s>", element.Name, attrsStr, v.visitAll(element.Children, "", ""), element.Name)
}

func (v *htmlSerializerVisitor) VisitAttribute(attribute *ml_parser.Attribute, context any) any {
	return fmt.Sprintf("%s=\"%s\"", attribute.Name, attribute.Value)
}

func (v *htmlSerializerVisitor) VisitText(text *ml_parser.Text, context any) any {
	return text.Value
}

func (v *htmlSerializerVisitor) VisitComment(comment *ml_parser.Comment, context any) any {
	return fmt.Sprintf("<!--%s-->", comment.Value)
}

func (v *htmlSerializerVisitor) VisitExpansion(expansion *ml_parser.Expansion, context any) any {
	cases := v.visitAllCases(expansion.Cases, "", "")
	return fmt.Sprintf("{%s, %s,%s}", expansion.SwitchValue, expansion.Type, cases)
}

func (v *htmlSerializerVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	expr := v.visitAll(expansionCase.Expression, "", "")
	return fmt.Sprintf(" %s {%s}", expansionCase.Value, expr)
}

func (v *htmlSerializerVisitor) VisitBlock(block *ml_parser.Block, context any) any {
	params := " "
	if len(block.Parameters) > 0 {
		params = fmt.Sprintf(" (%s) ", v.visitAllBlockParams(block.Parameters, ";", " "))
	}
	return fmt.Sprintf("@%s%s{%s}", block.Name, params, v.visitAll(block.Children, "", ""))
}

func (v *htmlSerializerVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return parameter.Expression
}

func (v *htmlSerializerVisitor) visitAll(nodes []ml_parser.Node, separator, prefix string) string {
	if len(nodes) == 0 {
		return ""
	}
	parts := make([]string, len(nodes))
	for i, node := range nodes {
		parts[i] = node.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func (v *htmlSerializerVisitor) visitAllAttrs(attrs []*ml_parser.Attribute, separator, prefix string) string {
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, len(attrs))
	for i, attr := range attrs {
		parts[i] = attr.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func (v *htmlSerializerVisitor) visitAllCases(cases []*ml_parser.ExpansionCase, separator, prefix string) string {
	if len(cases) == 0 {
		return ""
	}
	parts := make([]string, len(cases))
	for i, c := range cases {
		parts[i] = c.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func (v *htmlSerializerVisitor) visitAllBlockParams(params []*ml_parser.BlockParameter, separator, prefix string) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func serializeHtmlNodes(nodes []ml_parser.Node) []string {
	v := &htmlSerializerVisitor{}
	result := make([]string, len(nodes))
	for i, node := range nodes {
		result[i] = node.Visit(v, nil).(string)
	}
	return result
}

func parseHtml(html string) []ml_parser.Node {
	htmlParser := ml_parser.NewHtmlParser()
	parseResult := htmlParser.Parse(html, "extractor spec", &ml_parser.TokenizeOptions{TokenizeExpansionForms: true})
	if len(parseResult.Errors) > 1 {
		var errStrs []string
		for _, err := range parseResult.Errors {
			errStrs = append(errStrs, err.Error())
		}
		panic("unexpected parse errors: " + strings.Join(errStrs, "\n"))
	}
	return parseResult.RootNodes
}

func extract(
	html string,
	implicitTags []string,
	implicitAttrs map[string][]string,
) [][]any {
	nodes := parseHtml(html)
	result := ExtractMessages(nodes, implicitTags, implicitAttrs, true)
	if len(result.Errors) > 0 {
		var errStrs []string
		for _, err := range result.Errors {
			errStrs = append(errStrs, err.Error())
		}
		panic("unexpected errors: " + strings.Join(errStrs, "\n"))
	}
	var res [][]any
	for _, msg := range result.Messages {
		res = append(res, []any{
			SerializeNodes(msg.Nodes),
			msg.Meaning,
			msg.Description,
			msg.Id,
		})
	}
	if res == nil {
		return [][]any{}
	}
	return res
}

func extractErrors(
	html string,
	implicitTags []string,
	implicitAttrs map[string][]string,
) [][]string {
	nodes := parseHtml(html)
	result := ExtractMessages(nodes, implicitTags, implicitAttrs, true)
	var errors [][]string
	for _, err := range result.Errors {
		errors = append(errors, []string{err.Msg, err.Span.ToString()})
	}
	if errors == nil {
		return [][]string{}
	}
	return errors
}

func fakeTranslate(
	content string,
	implicitTags []string,
	implicitAttrs map[string][]string,
) string {
	htmlNodes := parseHtml(content)
	res := ExtractMessages(htmlNodes, implicitTags, implicitAttrs, true)

	i18nMsgMap := make(map[string][]Node)

	for _, message := range res.Messages {
		id := Digest(message)
		text := strings.Join(SerializeNodes(message.Nodes), "")
		text = strings.ReplaceAll(text, "<", "[")
		i18nMsgMap[id] = []Node{NewText(fmt.Sprintf("**%s**", text), nil)}
	}

	translationBundle := NewTranslationBundle(NewEagerTranslationStore(i18nMsgMap), nil, Digest, nil, core.MissingTranslationStrategyWarning, nil)
	output := MergeTranslations(htmlNodes, translationBundle, implicitTags, implicitAttrs)

	if len(output.Errors) > 0 {
		var errStrs []string
		for _, err := range output.Errors {
			errStrs = append(errStrs, err.Error())
		}
		panic("unexpected errors: " + strings.Join(errStrs, "\n"))
	}

	return strings.Join(serializeHtmlNodes(output.RootNodes), "")
}

func fakeNoTranslate(
	content string,
	implicitTags []string,
	implicitAttrs map[string][]string,
) string {
	htmlNodes := parseHtml(content)
	translationBundle := NewTranslationBundle(nil, nil, Digest, nil, core.MissingTranslationStrategyIgnore, nil)
	output := MergeTranslations(htmlNodes, translationBundle, implicitTags, implicitAttrs)

	if len(output.Errors) > 0 {
		var errStrs []string
		for _, err := range output.Errors {
			errStrs = append(errStrs, err.Error())
		}
		panic("unexpected errors: " + strings.Join(errStrs, "\n"))
	}

	return strings.Join(serializeHtmlNodes(output.RootNodes), "")
}

func TestExtractorAndMerger(t *testing.T) {
	t.Run("Extractor", func(t *testing.T) {
		t.Run("elements", func(t *testing.T) {
			t.Run("should extract from elements", func(t *testing.T) {
				got := extract("<div i18n=\"m|d|e\">text<span>nested</span></div>", nil, nil)
				expected := [][]any{
					{
						[]string{"text", "<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"},
						"m",
						"d|e",
						"",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes", func(t *testing.T) {
				got := extract("<div i18n=\"m1|d1\"><span i18n-title=\"m2|d2\" title=\"single child\">nested</span></div>", nil, nil)
				expected := [][]any{
					{[]string{"<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"}, "m1", "d1", ""},
					{[]string{"single child"}, "m2", "d2", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes with id", func(t *testing.T) {
				got := extract("<div i18n=\"m1|d1@@i1\"><span i18n-title=\"m2|d2@@i2\" title=\"single child\">nested</span></div>", nil, nil)
				expected := [][]any{
					{[]string{"<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"}, "m1", "d1", "i1"},
					{[]string{"single child"}, "m2", "d2", "i2"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should trim whitespace from custom ids (but not meanings)", func(t *testing.T) {
				got := extract("<div i18n=\"\n   m1|d1@@i1\n   \">test</div>", nil, nil)
				expected := [][]any{
					{[]string{"test"}, "\n   m1", "d1", "i1"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes without meaning and with id", func(t *testing.T) {
				got := extract("<div i18n=\"d1@@i1\"><span i18n-title=\"d2@@i2\" title=\"single child\">nested</span></div>", nil, nil)
				expected := [][]any{
					{[]string{"<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"}, "", "d1", "i1"},
					{[]string{"single child"}, "", "d2", "i2"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes with id only", func(t *testing.T) {
				got := extract("<div i18n=\"@@i1\"><span i18n-title=\"@@i2\" title=\"single child\">nested</span></div>", nil, nil)
				expected := [][]any{
					{[]string{"<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"}, "", "", "i1"},
					{[]string{"single child"}, "", "", "i2"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from ICU messages", func(t *testing.T) {
				got := extract("<div i18n=\"m|d\">{count, plural, =0 { <p i18n-title i18n-desc title=\"title\" desc=\"desc\"></p>}}</div>", nil, nil)
				expected := [][]any{
					{
						[]string{"{count, plural, =0 {[<ph tag name=\"START_PARAGRAPH\"></ph name=\"CLOSE_PARAGRAPH\">]}}"},
						"m",
						"d",
						"",
					},
					{[]string{"title"}, "", "", ""},
					{[]string{"desc"}, "", "", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should not create a message for empty elements", func(t *testing.T) {
				got := extract("<div i18n=\"m|d\"></div>", nil, nil)
				expected := [][]any{}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should not create a message for placeholder-only elements", func(t *testing.T) {
				got := extract("<div i18n=\"m|d\">{{ foo }}</div>", nil, nil)
				expected := [][]any{
					{[]string{"[<ph name=\"INTERPOLATION\"> foo </ph>]"}, "m", "d", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should ignore implicit elements in translatable elements", func(t *testing.T) {
				got := extract("<div i18n=\"m|d\"><p></p></div>", []string{"p"}, nil)
				expected := [][]any{
					{[]string{"<ph tag name=\"START_PARAGRAPH\"></ph name=\"CLOSE_PARAGRAPH\">"}, "m", "d", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("blocks", func(t *testing.T) {
			t.Run("should extract from elements inside blocks", func(t *testing.T) {
				got := extract(
					"@switch (value) {"+
						"@case (1) {<div i18n=\"a|b|c\">one <span>nested</span></div>}"+
						"@case (2) {<strong i18n=\"d|e|f\">two <span>nested</span></strong>}"+
						"@default {<strong i18n=\"g|h|i\">default <span>nested</span></strong>}"+
						"}",
					nil, nil,
				)
				expected := [][]any{
					{
						[]string{"one ", "<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"},
						"a",
						"b|c",
						"",
					},
					{
						[]string{"two ", "<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"},
						"d",
						"e|f",
						"",
					},
					{
						[]string{"default ", "<ph tag name=\"START_TAG_SPAN\">nested</ph name=\"CLOSE_TAG_SPAN\">"},
						"g",
						"h|i",
						"",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from i18n comment blocks inside blocks", func(t *testing.T) {
				got := extract(
					"@switch (value) {"+
						"@case (1) {<!-- i18n: oneMeaning|oneDesc -->one message<!-- /i18n -->}"+
						"@case (2) {<!-- i18n: twoMeaning|twoDesc -->two message<!-- /i18n -->}"+
						"@default {<!-- i18n: defaultMeaning|defaultDesc -->default message<!-- /i18n -->}"+
						"}",
					nil, nil,
				)
				expected := [][]any{
					{[]string{"one message"}, "oneMeaning", "oneDesc", ""},
					{[]string{"two message"}, "twoMeaning", "twoDesc", ""},
					{[]string{"default message"}, "defaultMeaning", "defaultDesc", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract ICUs from elements inside blocks", func(t *testing.T) {
				got := extract(
					"@switch (value) {"+
						"@case (1) {<div i18n=\"a|b\">{count, plural, =0 {oneText}}</div>}"+
						"@case (2) {<div i18n=\"c|d\">{count, plural, =0 {twoText}}</div>}"+
						"@default {<div i18n=\"e|f\">{count, plural, =0 {defaultText}}</div>}"+
						"}",
					nil, nil,
				)
				expected := [][]any{
					{[]string{"{count, plural, =0 {[oneText]}}"}, "a", "b", ""},
					{[]string{"{count, plural, =0 {[twoText]}}"}, "c", "d", ""},
					{[]string{"{count, plural, =0 {[defaultText]}}"}, "e", "f", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should not extract messages from ICUs directly inside blocks", func(t *testing.T) {
				expression := "{count, plural, =0 {text}}"
				got := extract(
					"@switch (value) {"+
						"@case (1) {"+expression+"}"+
						"@case (2) {"+expression+"}"+
						"@default {"+expression+"}"+
						"}",
					nil, nil,
				)
				expected := [][]any{}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should handle blocks inside of translated elements", func(t *testing.T) {
				got := extract("<span i18n=\"a|b|c\">@if (cond) {main content} @else {else content}</span>", nil, nil)
				expected := [][]any{
					{
						[]string{
							"<ph block name=\"START_BLOCK_IF\">main content</ph name=\"CLOSE_BLOCK_IF\">",
							" ",
							"<ph block name=\"START_BLOCK_ELSE\">else content</ph name=\"CLOSE_BLOCK_ELSE\">",
						},
						"a",
						"b|c",
						"",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("i18n comment blocks", func(t *testing.T) {
			t.Run("should extract from blocks", func(t *testing.T) {
				got := extract(`<!-- i18n: meaning1|desc1 -->message1<!-- /i18n -->
         <!-- i18n: desc2 -->message2<!-- /i18n -->
         <!-- i18n -->message3<!-- /i18n -->
         <!-- i18n: meaning4|desc4@@id4 -->message4<!-- /i18n -->
         <!-- i18n: @@id5 -->message5<!-- /i18n -->`, nil, nil)
				expected := [][]any{
					{[]string{"message1"}, "meaning1", "desc1", ""},
					{[]string{"message2"}, "", "desc2", ""},
					{[]string{"message3"}, "", "", ""},
					{[]string{"message4"}, "meaning4", "desc4", "id4"},
					{[]string{"message5"}, "", "", "id5"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should ignore implicit elements in blocks", func(t *testing.T) {
				got := extract("<!-- i18n:m|d --><p></p><!-- /i18n -->", []string{"p"}, nil)
				expected := [][]any{
					{[]string{"<ph tag name=\"START_PARAGRAPH\"></ph name=\"CLOSE_PARAGRAPH\">"}, "m", "d", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract siblings", func(t *testing.T) {
				got := extract(`<!-- i18n -->text<p>html<b>nested</b></p>{count, plural, =0 {<span>html</span>}}{{interp}}<!-- /i18n -->`, nil, nil)
				expected := [][]any{
					{
						[]string{"{count, plural, =0 {[<ph tag name=\"START_TAG_SPAN\">html</ph name=\"CLOSE_TAG_SPAN\">]}}"},
						"",
						"",
						"",
					},
					{
						[]string{
							"text",
							"<ph tag name=\"START_PARAGRAPH\">html, <ph tag name=\"START_BOLD_TEXT\">nested</ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">",
							"<ph icu name=\"ICU\">{count, plural, =0 {[<ph tag name=\"START_TAG_SPAN\">html</ph name=\"CLOSE_TAG_SPAN\">]}}</ph>",
							"[<ph name=\"INTERPOLATION\">interp</ph>]",
						},
						"",
						"",
						"",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got \n%v\nwant \n%v", got, expected)
				}
			})

			t.Run("should ignore other comments", func(t *testing.T) {
				got := extract(`<!-- i18n: meaning1|desc1@@id1 --><!-- other -->message1<!-- /i18n -->`, nil, nil)
				expected := [][]any{
					{[]string{"message1"}, "meaning1", "desc1", "id1"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should not create a message for empty blocks", func(t *testing.T) {
				got := extract(`<!-- i18n: meaning1|desc1 --><!-- /i18n -->`, nil, nil)
				expected := [][]any{}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("ICU messages", func(t *testing.T) {
			t.Run("should extract ICU messages from translatable elements", func(t *testing.T) {
				got1 := extract("<div i18n=\"m|d\">{count, plural, =0 {text}}</div>", nil, nil)
				expected1 := [][]any{
					{[]string{"{count, plural, =0 {[text]}}"}, "m", "d", ""},
				}
				if !reflect.DeepEqual(got1, expected1) {
					t.Errorf("got %v, want %v", got1, expected1)
				}

				got2 := extract("<div>{count, plural, =0 {text}}</div>", []string{"div"}, nil)
				expected2 := [][]any{
					{[]string{"{count, plural, =0 {[text]}}"}, "", "", ""},
				}
				if !reflect.DeepEqual(got2, expected2) {
					t.Errorf("got %v, want %v", got2, expected2)
				}

				got3 := extract("<div i18n=\"m|d@@i\">before{count, plural, =0 {text}}after</div>", nil, nil)
				expected3 := [][]any{
					{
						[]string{"before", "<ph icu name=\"ICU\">{count, plural, =0 {[text]}}</ph>", "after"},
						"m",
						"d",
						"i",
					},
					{[]string{"{count, plural, =0 {[text]}}"}, "", "", ""},
				}
				if !reflect.DeepEqual(got3, expected3) {
					t.Errorf("got %v, want %v", got3, expected3)
				}
			})

			t.Run("should extract ICU messages from translatable block", func(t *testing.T) {
				got1 := extract("<!-- i18n:m|d -->{count, plural, =0 {text}}<!-- /i18n -->", nil, nil)
				expected1 := [][]any{
					{[]string{"{count, plural, =0 {[text]}}"}, "m", "d", ""},
				}
				if !reflect.DeepEqual(got1, expected1) {
					t.Errorf("got %v, want %v", got1, expected1)
				}

				got2 := extract("<!-- i18n:m|d -->before{count, plural, =0 {text}}after<!-- /i18n -->", nil, nil)
				expected2 := [][]any{
					{[]string{"{count, plural, =0 {[text]}}"}, "", "", ""},
					{
						[]string{"before", "<ph icu name=\"ICU\">{count, plural, =0 {[text]}}</ph>", "after"},
						"m",
						"d",
						"",
					},
				}
				if !reflect.DeepEqual(got2, expected2) {
					t.Errorf("got %v, want %v", got2, expected2)
				}
			})

			t.Run("should not extract ICU messages outside of i18n sections", func(t *testing.T) {
				got := extract("{count, plural, =0 {text}}", nil, nil)
				expected := [][]any{}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should ignore nested ICU messages", func(t *testing.T) {
				got := extract("<div i18n=\"m|d\">{count, plural, =0 { {sex, select, male {m}} }}</div>", nil, nil)
				expected := [][]any{
					{[]string{"{count, plural, =0 {[{sex, select, male {[m]}},  ]}}"}, "m", "d", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should ignore implicit elements in non translatable ICU messages", func(t *testing.T) {
				got := extract("<div i18n=\"m|d@@i\">{count, plural, =0 { {sex, select, male {<p>ignore</p>}} }}</div>", []string{"p"}, nil)
				expected := [][]any{
					{
						[]string{"{count, plural, =0 {[{sex, select, male {[<ph tag name=\"START_PARAGRAPH\">ignore</ph name=\"CLOSE_PARAGRAPH\">]}},  ]}}"},
						"m",
						"d",
						"i",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should ignore implicit elements in non translatable ICU messages 2", func(t *testing.T) {
				got := extract("{count, plural, =0 { {sex, select, male {<p>ignore</p>}} }}", []string{"p"}, nil)
				expected := [][]any{}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("attributes", func(t *testing.T) {
			t.Run("should extract from attributes outside of translatable sections", func(t *testing.T) {
				got := extract("<div i18n-title=\"m|d@@i\" title=\"msg\"></div>", nil, nil)
				expected := [][]any{
					{[]string{"msg"}, "m", "d", "i"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes in translatable elements", func(t *testing.T) {
				got := extract("<div i18n><p><b i18n-title=\"m|d@@i\" title=\"msg\"></b></p></div>", nil, nil)
				expected := [][]any{
					{
						[]string{"<ph tag name=\"START_PARAGRAPH\"><ph tag name=\"START_BOLD_TEXT\"></ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">"},
						"",
						"",
						"",
					},
					{[]string{"msg"}, "m", "d", "i"},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes in translatable blocks", func(t *testing.T) {
				got := extract("<!-- i18n --><p><b i18n-title=\"m|d\" title=\"msg\"></b></p><!-- /i18n -->", nil, nil)
				expected := [][]any{
					{[]string{"msg"}, "m", "d", ""},
					{
						[]string{"<ph tag name=\"START_PARAGRAPH\"><ph tag name=\"START_BOLD_TEXT\"></ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">"},
						"",
						"",
						"",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes in translatable ICUs", func(t *testing.T) {
				got := extract("<!-- i18n -->{count, plural, =0 {<p><b i18n-title=\"m|d@@i\" title=\"msg\"></b></p>}}<!-- /i18n -->", nil, nil)
				expected := [][]any{
					{[]string{"msg"}, "m", "d", "i"},
					{
						[]string{"{count, plural, =0 {[<ph tag name=\"START_PARAGRAPH\"><ph tag name=\"START_BOLD_TEXT\"></ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">]}}"},
						"",
						"",
						"",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should extract from attributes in non translatable ICUs", func(t *testing.T) {
				got := extract("{count, plural, =0 {<p><b i18n-title=\"m|d\" title=\"msg\"></b></p>}}", nil, nil)
				expected := [][]any{
					{[]string{"msg"}, "m", "d", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should not create a message for empty attributes", func(t *testing.T) {
				got := extract("<div i18n-title=\"m|d\" title></div>", nil, nil)
				expected := [][]any{}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should not create a message for placeholder-only attributes", func(t *testing.T) {
				got := extract("<div i18n-title=\"m|d\" title=\"{{ foo }}\"></div>", nil, nil)
				expected := [][]any{}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("implicit elements", func(t *testing.T) {
			t.Run("should extract from implicit elements", func(t *testing.T) {
				got := extract("<b>bold</b><i>italic</i>", []string{"b"}, nil)
				expected := [][]any{
					{[]string{"bold"}, "", "", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})

			t.Run("should allow nested implicit elements", func(t *testing.T) {
				got := extract("<div>outer<div>inner</div></div>", []string{"div"}, nil)
				expected := [][]any{
					{
						[]string{"outer", "<ph tag name=\"START_TAG_DIV\">inner</ph name=\"CLOSE_TAG_DIV\">"},
						"",
						"",
						"",
					},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("implicit attributes", func(t *testing.T) {
			t.Run("should extract implicit attributes", func(t *testing.T) {
				got := extract("<b title=\"bb\">bold</b><i title=\"ii\">italic</i>", nil, map[string][]string{"b": {"title"}})
				expected := [][]any{
					{[]string{"bb"}, "", "", ""},
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("got %v, want %v", got, expected)
				}
			})
		})

		t.Run("errors", func(t *testing.T) {
			t.Run("elements", func(t *testing.T) {
				t.Run("should report nested translatable elements", func(t *testing.T) {
					got := extractErrors(`<p i18n><b i18n></b></p>`, nil, nil)
					expected := [][]string{
						{
							"Could not mark an element as translatable inside a translatable section",
							"<b i18n></b>",
						},
					}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("got %v, want %v", got, expected)
					}
				})

				t.Run("should report translatable elements in implicit elements", func(t *testing.T) {
					got := extractErrors(`<p><b i18n></b></p>`, []string{"p"}, nil)
					expected := [][]string{
						{
							"Could not mark an element as translatable inside a translatable section",
							"<b i18n></b>",
						},
					}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("got %v, want %v", got, expected)
					}
				})

				t.Run("should report translatable elements in translatable blocks", func(t *testing.T) {
					got := extractErrors(`<!-- i18n --><b i18n></b><!-- /i18n -->`, nil, nil)
					expected := [][]string{
						{
							"Could not mark an element as translatable inside a translatable section",
							"<b i18n></b>",
						},
					}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("got %v, want %v", got, expected)
					}
				})
			})

			t.Run("blocks", func(t *testing.T) {
				t.Run("should report nested blocks", func(t *testing.T) {
					got := extractErrors(`<!-- i18n --><!-- i18n --><!-- /i18n --><!-- /i18n -->`, nil, nil)
					expected := [][]string{
						{"Could not start a block inside a translatable section", "<!-- i18n -->"},
						{"Trying to close an unopened block", "<!-- /i18n -->"},
					}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("got %v, want %v", got, expected)
					}
				})

				t.Run("should report unclosed blocks", func(t *testing.T) {
					got := extractErrors(`<!-- i18n -->`, nil, nil)
					expected := [][]string{
						{"Unclosed block", "<!-- i18n -->"},
					}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("got %v, want %v", got, expected)
					}
				})

				t.Run("should report translatable blocks in translatable elements", func(t *testing.T) {
					got := extractErrors(`<p i18n><!-- i18n --><!-- /i18n --></p>`, nil, nil)
					expected := [][]string{
						{"Could not start a block inside a translatable section", "<!-- i18n -->"},
						{"Trying to close an unopened block", "<!-- /i18n -->"},
					}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("got %v, want %v", got, expected)
					}
				})

				t.Run("should report translatable blocks in implicit elements", func(t *testing.T) {
					got := extractErrors(`<p><!-- i18n --><!-- /i18n --></p>`, []string{"p"}, nil)
					expected := [][]string{
						{"Could not start a block inside a translatable section", "<!-- i18n -->"},
						{"Trying to close an unopened block", "<!-- /i18n -->"},
					}
					if !reflect.DeepEqual(got, expected) {
						t.Errorf("got %v, want %v", got, expected)
					}
				})

				t.Run("should report when start and end of a block are not at the same level", func(t *testing.T) {
					got1 := extractErrors(`<!-- i18n --><p><!-- /i18n --></p>`, nil, nil)
					expected1 := [][]string{
						{"I18N blocks should not cross element boundaries", "<!-- /i18n -->"},
						{"Unclosed block", "<p><!-- /i18n --></p>"},
					}
					if !reflect.DeepEqual(got1, expected1) {
						t.Errorf("got %v, want %v", got1, expected1)
					}

					got2 := extractErrors(`<p><!-- i18n --></p><!-- /i18n -->`, nil, nil)
					expected2 := [][]string{
						{"I18N blocks should not cross element boundaries", "<!-- /i18n -->"},
						{"Unclosed block", "<!-- /i18n -->"},
					}
					if !reflect.DeepEqual(got2, expected2) {
						t.Errorf("got %v, want %v", got2, expected2)
					}
				})
			})
		})
	})

	t.Run("Merger", func(t *testing.T) {
		t.Run("elements", func(t *testing.T) {
			t.Run("should merge elements", func(t *testing.T) {
				got := fakeTranslate(`<p i18n="m|d">foo</p>`, nil, nil)
				expected := "<p>**foo**</p>"
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should merge nested elements", func(t *testing.T) {
				got := fakeTranslate(`<div>before<p i18n="m|d">foo</p><!-- comment --></div>`, nil, nil)
				expected := "<div>before<p>**foo**</p></div>"
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should merge empty messages", func(t *testing.T) {
				HTML := `<div i18n>some element</div>`
				htmlNodes := parseHtml(HTML)
				messages := ExtractMessages(htmlNodes, nil, nil, true).Messages

				if len(messages) != 1 {
					t.Errorf("expected 1 message, got %d", len(messages))
				}

				i18nMsgMap := make(map[string][]Node)
				i18nMsgMap[Digest(messages[0])] = []Node{}
				translations := NewTranslationBundle(NewEagerTranslationStore(i18nMsgMap), nil, Digest, nil, core.MissingTranslationStrategyWarning, nil)

				output := MergeTranslations(htmlNodes, translations, nil, nil)
				if len(output.Errors) > 0 {
					t.Errorf("unexpected errors: %v", output.Errors)
				}

				got := strings.Join(serializeHtmlNodes(output.RootNodes), "")
				expected := `<div></div>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})
		})

		t.Run("i18n comment blocks", func(t *testing.T) {
			t.Run("should merge blocks", func(t *testing.T) {
				HTML := `before<!-- i18n --><p>foo</p><span><i>bar</i></span><!-- /i18n -->after`
				got := fakeTranslate(HTML, nil, nil)
				expected := `before**[ph tag name="START_PARAGRAPH">foo[/ph name="CLOSE_PARAGRAPH">[ph tag name="START_TAG_SPAN">[ph tag name="START_ITALIC_TEXT">bar[/ph name="CLOSE_ITALIC_TEXT">[/ph name="CLOSE_TAG_SPAN">**after`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should merge nested blocks", func(t *testing.T) {
				HTML := `<div>before<!-- i18n --><p>foo</p><span><i>bar</i></span><!-- /i18n -->after</div>`
				got := fakeTranslate(HTML, nil, nil)
				expected := `<div>before**[ph tag name="START_PARAGRAPH">foo[/ph name="CLOSE_PARAGRAPH">[ph tag name="START_TAG_SPAN">[ph tag name="START_ITALIC_TEXT">bar[/ph name="CLOSE_ITALIC_TEXT">[/ph name="CLOSE_TAG_SPAN">**after</div>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})
		})

		t.Run("attributes", func(t *testing.T) {
			t.Run("should merge attributes", func(t *testing.T) {
				got := fakeTranslate(`<p i18n-title="m|d" title="foo"></p>`, nil, nil)
				expected := `<p title="**foo**"></p>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should merge attributes with ids", func(t *testing.T) {
				got := fakeTranslate(`<p i18n-title="@@id" title="foo"></p>`, nil, nil)
				expected := `<p title="**foo**"></p>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should merge nested attributes", func(t *testing.T) {
				got := fakeTranslate(`<div>{count, plural, =0 {<p i18n-title title="foo"></p>}}</div>`, nil, nil)
				expected := `<div>{count, plural, =0 {<p title="**foo**"></p>}}</div>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should merge attributes without values", func(t *testing.T) {
				got := fakeTranslate(`<p i18n-title="m|d" title=""></p>`, nil, nil)
				expected := `<p title=""></p>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should merge empty attributes", func(t *testing.T) {
				HTML := `<div i18n-title title="some attribute">some element</div>`
				htmlNodes := parseHtml(HTML)
				messages := ExtractMessages(htmlNodes, nil, nil, true).Messages

				if len(messages) != 1 {
					t.Errorf("expected 1 message, got %d", len(messages))
				}

				i18nMsgMap := make(map[string][]Node)
				i18nMsgMap[Digest(messages[0])] = []Node{}
				translations := NewTranslationBundle(NewEagerTranslationStore(i18nMsgMap), nil, Digest, nil, core.MissingTranslationStrategyWarning, nil)

				output := MergeTranslations(htmlNodes, translations, nil, nil)
				if len(output.Errors) > 0 {
					t.Errorf("unexpected errors: %v", output.Errors)
				}

				got := strings.Join(serializeHtmlNodes(output.RootNodes), "")
				expected := `<div title="">some element</div>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})
		})

		t.Run("no translations", func(t *testing.T) {
			t.Run("should remove i18n attributes", func(t *testing.T) {
				got := fakeNoTranslate(`<p i18n="m|d">foo</p>`, nil, nil)
				expected := "<p>foo</p>"
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should remove i18n- attributes", func(t *testing.T) {
				got := fakeNoTranslate(`<p i18n-title="m|d" title="foo"></p>`, nil, nil)
				expected := `<p title="foo"></p>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should remove i18n comment blocks", func(t *testing.T) {
				got := fakeNoTranslate(`before<!-- i18n --><p>foo</p><span><i>bar</i></span><!-- /i18n -->after`, nil, nil)
				expected := `before<p>foo</p><span><i>bar</i></span>after`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})

			t.Run("should remove nested i18n markup", func(t *testing.T) {
				got := fakeNoTranslate(`<!-- i18n --><span someAttr="ok">foo</span><div>{count, plural, =0 {<p i18n-title title="foo"></p>}}</div><!-- /i18n -->`, nil, nil)
				expected := `<span someAttr="ok">foo</span><div>{count, plural, =0 {<p title="foo"></p>}}</div>`
				if got != expected {
					t.Errorf("got %q, want %q", got, expected)
				}
			})
		})
	})
}
