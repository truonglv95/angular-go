package i18n

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
)

type placeholderOrderVisitor struct {
	names []string
	seen  map[string]bool
}

func (v *placeholderOrderVisitor) add(name string) {
	if !v.seen[name] {
		v.seen[name] = true
		v.names = append(v.names, name)
	}
}

func (v *placeholderOrderVisitor) VisitText(text *Text, context any) any {
	return nil
}

func (v *placeholderOrderVisitor) VisitContainer(container *Container, context any) any {
	for _, child := range container.Children {
		child.Visit(v, nil)
	}
	return nil
}

func (v *placeholderOrderVisitor) VisitIcu(icu *Icu, context any) any {
	if icu.ExpressionPlaceholder != "" {
		v.add(icu.ExpressionPlaceholder)
	}
	for _, name := range icu.CaseOrders {
		if c, ok := icu.Cases[name]; ok {
			c.Visit(v, nil)
		}
	}
	return nil
}

func (v *placeholderOrderVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	v.add(ph.StartName)
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	if !ph.IsVoid {
		v.add(ph.CloseName)
	}
	return nil
}

func (v *placeholderOrderVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	v.add(ph.Name)
	return nil
}

func (v *placeholderOrderVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	v.add(ph.Name)
	ph.Value.Visit(v, nil)
	return nil
}

func (v *placeholderOrderVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	v.add(ph.StartName)
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	v.add(ph.CloseName)
	return nil
}

func getPlaceholderNamesInOrder(msg *Message) []string {
	v := &placeholderOrderVisitor{
		seen: make(map[string]bool),
	}
	for _, node := range msg.Nodes {
		node.Visit(v, nil)
	}
	var ordered []string
	for _, name := range v.names {
		if _, ok := msg.Placeholders[name]; ok {
			ordered = append(ordered, name)
		}
	}
	var remaining []string
	for name := range msg.Placeholders {
		found := false
		for _, o := range ordered {
			if o == name {
				found = true
				break
			}
		}
		if !found {
			remaining = append(remaining, name)
		}
	}
	sort.Strings(remaining)
	return append(ordered, remaining...)
}

func extractMessages(
	html string,
	implicitTags []string,
	implicitAttrs map[string][]string,
	preserveSignificantWhitespace bool,
) []*Message {
	htmlParser := ml_parser.NewHtmlParser()
	parseResult := htmlParser.Parse(html, "extractor spec", &ml_parser.TokenizeOptions{TokenizeExpansionForms: true})
	if len(parseResult.Errors) > 1 {
		var errStrings []string
		for _, err := range parseResult.Errors {
			errStrings = append(errStrings, err.Error())
		}
		panic(fmt.Sprintf("unexpected parse errors: %s", strings.Join(errStrings, "\n")))
	}
	return ExtractMessages(
		parseResult.RootNodes,
		implicitTags,
		implicitAttrs,
		preserveSignificantWhitespace,
	).Messages
}

func humanizeMessages(
	html string,
	implicitTags []string,
	implicitAttrs map[string][]string,
	preserveSignificantWhitespace bool,
) [][]any {
	messages := extractMessages(html, implicitTags, implicitAttrs, preserveSignificantWhitespace)
	var result [][]any
	for _, msg := range messages {
		result = append(result, []any{
			SerializeNodes(msg.Nodes),
			msg.Meaning,
			msg.Description,
			msg.Id,
		})
	}
	if result == nil {
		return [][]any{}
	}
	return result
}

func humanizePlaceholders(
	html string,
	implicitTags []string,
	implicitAttrs map[string][]string,
	preserveSignificantWhitespace bool,
) []string {
	messages := extractMessages(html, implicitTags, implicitAttrs, preserveSignificantWhitespace)
	var result []string
	for _, msg := range messages {
		orderedNames := getPlaceholderNamesInOrder(msg)
		var pairs []string
		for _, name := range orderedNames {
			ph := msg.Placeholders[name]
			pairs = append(pairs, fmt.Sprintf("%s=%s", name, ph.Text))
		}
		result = append(result, strings.Join(pairs, ", "))
	}
	if result == nil {
		return []string{}
	}
	return result
}

func humanizePlaceholdersToMessage(
	html string,
	implicitTags []string,
	implicitAttrs map[string][]string,
	preserveSignificantWhitespace bool,
) []string {
	messages := extractMessages(html, implicitTags, implicitAttrs, preserveSignificantWhitespace)
	var result []string
	for _, msg := range messages {
		var keys []string
		for k := range msg.PlaceholderToMessage {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var pairs []string
		for _, k := range keys {
			subMsg := msg.PlaceholderToMessage[k]
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, Digest(subMsg)))
		}
		result = append(result, strings.Join(pairs, ", "))
	}
	if result == nil {
		return []string{}
	}
	return result
}

func TestI18nParser(t *testing.T) {
	t.Run("elements", func(t *testing.T) {
		t.Run("should extract from elements", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\">text</div>", nil, nil, true)
			expected := [][]any{
				{[]string{"text"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract from nested elements", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\">text<span><b>nested</b></span></div>", nil, nil, true)
			expected := [][]any{
				{
					[]string{
						"text",
						"<ph tag name=\"START_TAG_SPAN\"><ph tag name=\"START_BOLD_TEXT\">nested</ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_TAG_SPAN\">",
					},
					"m",
					"d",
					"",
				},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should not create a message for empty elements", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\"></div>", nil, nil, true)
			expected := [][]any{}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should not create a message for plain elements", func(t *testing.T) {
			got := humanizeMessages("<div></div>", nil, nil, true)
			expected := [][]any{}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should support void elements", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\"><p><br></p></div>", nil, nil, true)
			expected := [][]any{
				{
					[]string{
						"<ph tag name=\"START_PARAGRAPH\"><ph tag name=\"LINE_BREAK\"/></ph name=\"CLOSE_PARAGRAPH\">",
					},
					"m",
					"d",
					"",
				},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should trim whitespace from custom ids (but not meanings)", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"\n   m|d@@id\n   \">text</div>", nil, nil, true)
			expected := [][]any{
				{[]string{"text"}, "\n   m", "d", "id"},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	})

	t.Run("attributes", func(t *testing.T) {
		t.Run("should extract from attributes outside of translatable section", func(t *testing.T) {
			got := humanizeMessages("<div i18n-title=\"m|d\" title=\"msg\"></div>", nil, nil, true)
			expected := [][]any{
				{[]string{"msg"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract from attributes in translatable element", func(t *testing.T) {
			got := humanizeMessages("<div i18n><p><b i18n-title=\"m|d\" title=\"msg\"></b></p></div>", nil, nil, true)
			expected := [][]any{
				{
					[]string{
						"<ph tag name=\"START_PARAGRAPH\"><ph tag name=\"START_BOLD_TEXT\"></ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">",
					},
					"",
					"",
					"",
				},
				{[]string{"msg"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract from attributes in translatable block", func(t *testing.T) {
			got := humanizeMessages("<!-- i18n --><p><b i18n-title=\"m|d\" title=\"msg\"></b></p><!-- /i18n -->", nil, nil, true)
			expected := [][]any{
				{[]string{"msg"}, "m", "d", ""},
				{
					[]string{
						"<ph tag name=\"START_PARAGRAPH\"><ph tag name=\"START_BOLD_TEXT\"></ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">",
					},
					"",
					"",
					"",
				},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract from attributes in translatable ICU", func(t *testing.T) {
			got := humanizeMessages("<!-- i18n -->{count, plural, =0 {<p><b i18n-title=\"m|d\" title=\"msg\"></b></p>}}<!-- /i18n -->", nil, nil, true)
			expected := [][]any{
				{[]string{"msg"}, "m", "d", ""},
				{
					[]string{
						"{count, plural, =0 {[<ph tag name=\"START_PARAGRAPH\"><ph tag name=\"START_BOLD_TEXT\"></ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">]}}",
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

		t.Run("should extract from attributes in non translatable ICU", func(t *testing.T) {
			got := humanizeMessages("{count, plural, =0 {<p><b i18n-title=\"m|d\" title=\"msg\"></b></p>}}", nil, nil, true)
			expected := [][]any{
				{[]string{"msg"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should not create a message for empty attributes", func(t *testing.T) {
			got := humanizeMessages("<div i18n-title=\"m|d\" title></div>", nil, nil, true)
			expected := [][]any{}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	})

	t.Run("interpolation", func(t *testing.T) {
		t.Run("should replace interpolation with placeholder", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\">before{{ exp }}after</div>", nil, nil, true)
			expected := [][]any{
				{[]string{"[before, <ph name=\"INTERPOLATION\"> exp </ph>, after]"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should support named interpolation", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\">before{{ exp //i18n(ph=\"teSt\") }}after</div>", nil, nil, true)
			expected := [][]any{
				{[]string{"[before, <ph name=\"TEST\"> exp //i18n(ph=\"teSt\") </ph>, after]"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}

			got2 := humanizeMessages("<div i18n='m|d'>before{{ exp //i18n(ph='teSt') }}after</div>", nil, nil, true)
			expected2 := [][]any{
				{[]string{"[before, <ph name=\"TEST\"> exp //i18n(ph='teSt') </ph>, after]"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got2, expected2) {
				t.Errorf("got %v, want %v", got2, expected2)
			}
		})
	})

	t.Run("blocks", func(t *testing.T) {
		t.Run("should extract from blocks", func(t *testing.T) {
			got := humanizeMessages(`<!-- i18n: meaning1|desc1 -->message1<!-- /i18n -->
         <!-- i18n: desc2 -->message2<!-- /i18n -->
         <!-- i18n -->message3<!-- /i18n -->`, nil, nil, true)
			expected := [][]any{
				{[]string{"message1"}, "meaning1", "desc1", ""},
				{[]string{"message2"}, "", "desc2", ""},
				{[]string{"message3"}, "", "", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract all siblings", func(t *testing.T) {
			got := humanizeMessages(`<!-- i18n -->text<p>html<b>nested</b></p><!-- /i18n -->`, nil, nil, true)
			expected := [][]any{
				{
					[]string{
						"text",
						"<ph tag name=\"START_PARAGRAPH\">html, <ph tag name=\"START_BOLD_TEXT\">nested</ph name=\"CLOSE_BOLD_TEXT\"></ph name=\"CLOSE_PARAGRAPH\">",
					},
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

	t.Run("ICU messages", func(t *testing.T) {
		t.Run("should extract as ICU when single child of an element", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\">{count, plural, =0 {zero}}</div>", nil, nil, true)
			expected := [][]any{
				{[]string{"{count, plural, =0 {[zero]}}"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract as ICU + ph when not single child of an element", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\">b{count, plural, =0 {zero}}a</div>", nil, nil, true)
			expected := [][]any{
				{[]string{"b", "<ph icu name=\"ICU\">{count, plural, =0 {[zero]}}</ph>", "a"}, "m", "d", ""},
				{[]string{"{count, plural, =0 {[zero]}}"}, "", "", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract as ICU + ph when wrapped in whitespace in an element", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\"> {count, plural, =0 {zero}} </div>", nil, nil, true)
			expected := [][]any{
				{[]string{" ", "<ph icu name=\"ICU\">{count, plural, =0 {[zero]}}</ph>", " "}, "m", "d", ""},
				{[]string{"{count, plural, =0 {[zero]}}"}, "", "", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract as ICU when single child of a block", func(t *testing.T) {
			got := humanizeMessages("<!-- i18n:m|d -->{count, plural, =0 {zero}}<!-- /i18n -->", nil, nil, true)
			expected := [][]any{
				{[]string{"{count, plural, =0 {[zero]}}"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should extract as ICU + ph when not single child of a block", func(t *testing.T) {
			got := humanizeMessages("<!-- i18n:m|d -->b{count, plural, =0 {zero}}a<!-- /i18n -->", nil, nil, true)
			expected := [][]any{
				{[]string{"{count, plural, =0 {[zero]}}"}, "", "", ""},
				{[]string{"b", "<ph icu name=\"ICU\">{count, plural, =0 {[zero]}}</ph>", "a"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should not extract nested ICU messages", func(t *testing.T) {
			got := humanizeMessages("<div i18n=\"m|d\">b{count, plural, =0 {{sex, select, male {m}}}}a</div>", nil, nil, true)
			expected := [][]any{
				{
					[]string{"b", "<ph icu name=\"ICU\">{count, plural, =0 {[{sex, select, male {[m]}}]}}</ph>", "a"},
					"m",
					"d",
					"",
				},
				{[]string{"{count, plural, =0 {[{sex, select, male {[m]}}]}}"}, "", "", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should preserve whitespace when preserving significant whitespace", func(t *testing.T) {
			html := "<div i18n=\"m|d\">{count, plural, =0 {{{   foo   }}}}</div>"
			got := humanizeMessages(html, nil, nil, true)
			expected := [][]any{
				{[]string{"{count, plural, =0 {[[<ph name=\"INTERPOLATION\">   foo   </ph>]]}}"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should normalize whitespace when not preserving significant whitespace", func(t *testing.T) {
			html := "<div i18n=\"m|d\">{count, plural, =0 {{{   foo   }}}}</div>"
			got := humanizeMessages(html, nil, nil, false)
			expected := [][]any{
				{[]string{"{count, plural, =0 {[[, <ph name=\"INTERPOLATION\">foo</ph>, ]]}}"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	})

	t.Run("implicit elements", func(t *testing.T) {
		t.Run("should extract from implicit elements", func(t *testing.T) {
			got := humanizeMessages("<b>bold</b><i>italic</i>", []string{"b"}, nil, true)
			expected := [][]any{
				{[]string{"bold"}, "", "", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	})

	t.Run("implicit attributes", func(t *testing.T) {
		t.Run("should extract implicit attributes", func(t *testing.T) {
			got := humanizeMessages("<b title=\"bb\">bold</b><i title=\"ii\">italic</i>", []string{}, map[string][]string{"b": {"title"}}, true)
			expected := [][]any{
				{[]string{"bb"}, "", "", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	})

	t.Run("placeholders", func(t *testing.T) {
		t.Run("should reuse the same placeholder name for tags", func(t *testing.T) {
			html := "<div i18n=\"m|d\"><p>one</p><p>two</p><p other>three</p></div>"
			got := humanizeMessages(html, nil, nil, true)
			expected := [][]any{
				{
					[]string{
						"<ph tag name=\"START_PARAGRAPH\">one</ph name=\"CLOSE_PARAGRAPH\">",
						"<ph tag name=\"START_PARAGRAPH\">two</ph name=\"CLOSE_PARAGRAPH\">",
						"<ph tag name=\"START_PARAGRAPH_1\">three</ph name=\"CLOSE_PARAGRAPH\">",
					},
					"m",
					"d",
					"",
				},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}

			gotPhs := humanizePlaceholders(html, nil, nil, true)
			expectedPhs := []string{
				"START_PARAGRAPH=<p>, CLOSE_PARAGRAPH=</p>, START_PARAGRAPH_1=<p other>",
			}
			if !reflect.DeepEqual(gotPhs, expectedPhs) {
				t.Errorf("got %q, want %q", gotPhs, expectedPhs)
			}
		})

		t.Run("should reuse the same placeholder name for interpolations", func(t *testing.T) {
			html := "<div i18n=\"m|d\">{{ a }}{{ a }}{{ b }}</div>"
			got := humanizeMessages(html, nil, nil, true)
			expected := [][]any{
				{
					[]string{
						"[<ph name=\"INTERPOLATION\"> a </ph>, <ph name=\"INTERPOLATION\"> a </ph>, <ph name=\"INTERPOLATION_1\"> b </ph>]",
					},
					"m",
					"d",
					"",
				},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}

			gotPhs := humanizePlaceholders(html, nil, nil, true)
			expectedPhs := []string{
				"INTERPOLATION={{ a }}, INTERPOLATION_1={{ b }}",
			}
			if !reflect.DeepEqual(gotPhs, expectedPhs) {
				t.Errorf("got %q, want %q", gotPhs, expectedPhs)
			}
		})

		t.Run("should reuse the same placeholder name for icu messages", func(t *testing.T) {
			html := "<div i18n=\"m|d\">{count, plural, =0 {0}}{count, plural, =0 {0}}{count, plural, =1 {1}}</div>"
			got := humanizeMessages(html, nil, nil, true)
			expected := [][]any{
				{
					[]string{
						"<ph icu name=\"ICU\">{count, plural, =0 {[0]}}</ph>",
						"<ph icu name=\"ICU\">{count, plural, =0 {[0]}}</ph>",
						"<ph icu name=\"ICU_1\">{count, plural, =1 {[1]}}</ph>",
					},
					"m",
					"d",
					"",
				},
				{[]string{"{count, plural, =0 {[0]}}"}, "", "", ""},
				{[]string{"{count, plural, =0 {[0]}}"}, "", "", ""},
				{[]string{"{count, plural, =1 {[1]}}"}, "", "", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}

			gotPhs := humanizePlaceholders(html, nil, nil, true)
			expectedPhs := []string{
				"",
				"VAR_PLURAL=count",
				"VAR_PLURAL=count",
				"VAR_PLURAL=count",
			}
			if !reflect.DeepEqual(gotPhs, expectedPhs) {
				t.Errorf("got %q, want %q", gotPhs, expectedPhs)
			}

			gotPhsToMsg := humanizePlaceholdersToMessage(html, nil, nil, true)
			expectedPhsToMsg := []string{
				"ICU=f0f76923009914f1b05f41042a5c7231b9496504, ICU_1=73693d1f78d0fc882f0bcbce4cb31a0aa1995cfe",
				"",
				"",
				"",
			}
			if !reflect.DeepEqual(gotPhsToMsg, expectedPhsToMsg) {
				t.Errorf("got %q, want %q", gotPhsToMsg, expectedPhsToMsg)
			}
		})

		t.Run("should preserve whitespace when preserving significant whitespace", func(t *testing.T) {
			html := "<div i18n=\"m|d\">hello {{   foo   }}</div>"
			got := humanizeMessages(html, nil, nil, true)
			expected := [][]any{
				{[]string{"[hello , <ph name=\"INTERPOLATION\">   foo   </ph>]"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})

		t.Run("should normalize whitespace when not preserving significant whitespace", func(t *testing.T) {
			html := "<div i18n=\"m|d\">hello {{   foo   }}</div>"
			got := humanizeMessages(html, nil, nil, false)
			expected := [][]any{
				{[]string{"[hello , <ph name=\"INTERPOLATION\">foo</ph>, ]"}, "m", "d", ""},
			}
			if !reflect.DeepEqual(got, expected) {
				t.Errorf("got %v, want %v", got, expected)
			}
		})
	})
}
