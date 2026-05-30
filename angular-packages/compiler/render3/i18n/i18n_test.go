package i18n

import (
	"fmt"
	"strings"
	"testing"

	ci18n "github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	o "github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/stretchr/testify/assert"
)

func parseI18nMessage(html string) *ci18n.Message {
	htmlParser := ml_parser.NewHtmlParser()
	parseResult := htmlParser.Parse(html, "i18n spec", &ml_parser.TokenizeOptions{TokenizeExpansionForms: true})
	if len(parseResult.Errors) > 0 {
		var errs []string
		for _, e := range parseResult.Errors {
			errs = append(errs, e.Error())
		}
		panic("Parse errors: " + strings.Join(errs, "; "))
	}
	if len(parseResult.RootNodes) == 0 {
		panic("No root nodes parsed")
	}
	el := parseResult.RootNodes[0].(*ml_parser.Element)
	factory := ci18n.CreateI18nMessageFactory(true, false)
	meaning := ""
	description := ""
	customId := ""
	return factory(el.Children, &meaning, &description, &customId, nil)
}

func TestFormatI18nPlaceholderName(t *testing.T) {
	cases := [][]string{
		{"", ""},
		{"ICU", "icu"},
		{"ICU_1", "icu_1"},
		{"ICU_1000", "icu_1000"},
		{"START_TAG_NG-CONTAINER", "startTagNgContainer"},
		{"START_TAG_NG-CONTAINER_1", "startTagNgContainer_1"},
		{"CLOSE_TAG_ITALIC", "closeTagItalic"},
		{"CLOSE_TAG_BOLD_1", "closeTagBold_1"},
	}
	for _, tc := range cases {
		input, expected := tc[0], tc[1]
		t.Run(input, func(t *testing.T) {
			actual := FormatI18nPlaceholderName(input, true)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestParseI18nMeta(t *testing.T) {
	meta := func(customId, meaning, description string) I18nMeta {
		return I18nMeta{CustomId: customId, Meaning: meaning, Description: description}
	}

	assert.Equal(t, meta("", "", ""), ParseI18nMeta(""))
	assert.Equal(t, meta("", "", "desc"), ParseI18nMeta("desc"))
	assert.Equal(t, meta("id", "", "desc"), ParseI18nMeta("desc@@id"))
	assert.Equal(t, meta("", "meaning", "desc"), ParseI18nMeta("meaning|desc"))
	assert.Equal(t, meta("id", "meaning", "desc"), ParseI18nMeta("meaning|desc@@id"))
	assert.Equal(t, meta("id", "", ""), ParseI18nMeta("@@id"))

	assert.Equal(t, meta("", "", ""), ParseI18nMeta("\n   "))
	assert.Equal(t, meta("", "", "desc"), ParseI18nMeta("\n   desc\n   "))
	assert.Equal(t, meta("id", "", "desc"), ParseI18nMeta("\n   desc@@id\n   "))
	assert.Equal(t, meta("", "meaning", "desc"), ParseI18nMeta("\n   meaning|desc\n   "))
	assert.Equal(t, meta("id", "meaning", "desc"), ParseI18nMeta("\n   meaning|desc@@id\n   "))
	assert.Equal(t, meta("id", "", ""), ParseI18nMeta("\n   @@id\n   "))
}

func TestI18nMetaToJSDoc(t *testing.T) {
	t.Run("generates with description", func(t *testing.T) {
		tags := I18nMetaToJSDoc(I18nMeta{Description: "desc"})
		assert.Len(t, tags, 1)
		assert.Equal(t, o.JSDocTag{TagName: "desc", Text: "desc"}, tags[0])
	})

	t.Run("generates with no description suppressed", func(t *testing.T) {
		tags := I18nMetaToJSDoc(I18nMeta{})
		assert.Len(t, tags, 1)
		assert.Equal(t, o.JSDocTag{TagName: "suppress", Text: "{msgDescriptions}"}, tags[0])
	})

	t.Run("generates with description and meaning", func(t *testing.T) {
		tags := I18nMetaToJSDoc(I18nMeta{Meaning: "meaning", Description: "desc"})
		assert.Contains(t, tags, o.JSDocTag{TagName: "meaning", Text: "meaning"})
		assert.Contains(t, tags, o.JSDocTag{TagName: "desc", Text: "desc"})
	})
}

func TestSerializeI18nHead(t *testing.T) {
	metaBlock := func(customId, meaning, description string, legacyIds ...string) I18nMeta {
		return testMeta(customId, meaning, description, legacyIds...)
	}

	runTest := func(meta I18nMeta, messageParts []o.LiteralPiece, expectedCooked, expectedRaw string) {
		localized := o.NewLocalizedString(meta, messageParts, nil, nil, o.NewSourceSpanAdapter(createDummySourceSpan()), nil)
		res := localized.SerializeI18nHead()
		assert.Equal(t, expectedCooked, res.Cooked)
		assert.Equal(t, expectedRaw, res.Raw)
		assert.NotNil(t, res.Range)
	}

	runTest(metaBlock("", "", ""), []o.LiteralPiece{literal("")}, "", "")
	runTest(metaBlock("", "", "desc"), []o.LiteralPiece{literal("")}, ":desc:", ":desc:")
	runTest(metaBlock("id", "", "desc"), []o.LiteralPiece{literal("")}, ":desc@@id:", ":desc@@id:")
	runTest(metaBlock("", "meaning", "desc"), []o.LiteralPiece{literal("")}, ":meaning|desc:", ":meaning|desc:")
	runTest(metaBlock("id", "meaning", "desc"), []o.LiteralPiece{literal("")}, ":meaning|desc@@id:", ":meaning|desc@@id:")
	runTest(metaBlock("id", "", ""), []o.LiteralPiece{literal("")}, ":@@id:", ":@@id:")

	// Escaping colons (block markers)
	runTest(metaBlock("id:sub_id", "meaning", "desc"), []o.LiteralPiece{literal("")}, ":meaning|desc@@id:sub_id:", ":meaning|desc@@id\\:sub_id:")
	runTest(metaBlock("id", "meaning:sub_meaning", "desc"), []o.LiteralPiece{literal("")}, ":meaning:sub_meaning|desc@@id:", ":meaning\\:sub_meaning|desc@@id:")
	runTest(metaBlock("id", "meaning", "desc:sub_desc"), []o.LiteralPiece{literal("")}, ":meaning|desc:sub_desc@@id:", ":meaning|desc\\:sub_desc@@id:")
	runTest(metaBlock("id", "meaning", "desc"), []o.LiteralPiece{literal("message source")}, ":meaning|desc@@id:message source", ":meaning|desc@@id:message source")
	runTest(metaBlock("id", "meaning", "desc"), []o.LiteralPiece{literal(":message source")}, ":meaning|desc@@id::message source", ":meaning|desc@@id::message source")
	runTest(metaBlock("", "", ""), []o.LiteralPiece{literal("message source")}, "message source", "message source")
	runTest(metaBlock("", "", ""), []o.LiteralPiece{literal(":message source")}, ":message source", "\\:message source")
}

func TestSerializeI18nPlaceholderBlock(t *testing.T) {
	runTest := func(meta I18nMeta, messageParts []o.LiteralPiece, placeHolderNames []o.PlaceholderPiece, partIndex int, expectedCooked, expectedRaw string) {
		localized := o.NewLocalizedString(meta, messageParts, placeHolderNames, nil, o.NewSourceSpanAdapter(createDummySourceSpan()), nil)
		res := localized.SerializeI18nTemplatePart(partIndex)
		assert.Equal(t, expectedCooked, res.Cooked)
		assert.Equal(t, expectedRaw, res.Raw)
		assert.NotNil(t, res.Range)
	}

	runTest(testMeta("", "", ""), []o.LiteralPiece{literal(""), literal("")}, []o.PlaceholderPiece{placeholder("")}, 1, "", "")
	runTest(testMeta("", "", ""), []o.LiteralPiece{literal(""), literal("")}, []o.PlaceholderPiece{placeholder("abc")}, 1, ":abc:", ":abc:")
	runTest(testMeta("", "", ""), []o.LiteralPiece{literal(""), literal("message")}, []o.PlaceholderPiece{placeholder("")}, 1, "message", "message")
	runTest(testMeta("", "", ""), []o.LiteralPiece{literal(""), literal("message")}, []o.PlaceholderPiece{placeholder("abc")}, 1, ":abc:message", ":abc:message")
	runTest(testMeta("", "", ""), []o.LiteralPiece{literal(""), literal(":message")}, []o.PlaceholderPiece{placeholder("")}, 1, ":message", "\\:message")
	runTest(testMeta("", "", ""), []o.LiteralPiece{literal(""), literal(":message")}, []o.PlaceholderPiece{placeholder("abc")}, 1, ":abc::message", ":abc::message")
}

func TestSerializeI18nMessageForGetMsg(t *testing.T) {
	serialize := func(input string) string {
		msg := parseI18nMessage("<div i18n>" + input + "</div>")
		return SerializeI18nMessageForGetMsg(msg)
	}

	t.Run("should serialize plain text for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "Some text", serialize("Some text"))
	})

	t.Run("should serialize text with interpolation for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "Some text {$interpolation} and {$interpolation_1}", serialize("Some text {{ valueA }} and {{ valueB + valueC }}"))
	})

	t.Run("should serialize interpolation with named placeholder for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "{$placeholderName}", serialize("{{ valueB + valueC // i18n(ph=\"PLACEHOLDER NAME\") }}"))
	})

	t.Run("should serialize content with HTML tags for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "A {$startTagSpan}B{$startTagDiv}C{$closeTagDiv}{$closeTagSpan} D", serialize("A <span>B<div>C</div></span> D"))
	})

	t.Run("should serialize simple ICU for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "{VAR_PLURAL, plural, 10 {ten} other {other}}", serialize("{age, plural, 10 {ten} other {other}}"))
	})

	t.Run("should serialize nested ICUs for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "{VAR_PLURAL, plural, 10 {ten {VAR_SELECT, select, 1 {one} 2 {two} other {2+}}} other {other}}",
			serialize("{age, plural, 10 {ten {size, select, 1 {one} 2 {two} other {2+}}} other {other}}"))
	})

	t.Run("should serialize ICU with nested HTML for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "{VAR_PLURAL, plural, 10 {{START_BOLD_TEXT}ten{CLOSE_BOLD_TEXT}} other {{START_TAG_DIV}other{CLOSE_TAG_DIV}}}",
			serialize("{age, plural, 10 {<b>ten</b>} other {<div class=\"A\">other</div>}}"))
	})

	t.Run("should serialize ICU with nested HTML containing further ICUs for GetMsg()", func(t *testing.T) {
		assert.Equal(t, "{$icu}{$startTagDiv}{$icu}{$closeTagDiv}",
			serialize("{gender, select, male {male} female {female} other {other}}<div>{gender, select, male {male} female {female} other {other}}</div>"))
	})
}

func TestSerializeI18nMessageForLocalize(t *testing.T) {
	serialize := func(input string) ([]o.LiteralPiece, []o.PlaceholderPiece) {
		msg := parseI18nMessage("<div i18n>" + input + "</div>")
		return SerializeI18nMessageForLocalize(msg)
	}

	t.Run("should serialize plain text for $localize()", func(t *testing.T) {
		parts, phs := serialize("Some text")
		assertEqualPieces(t, []o.LiteralPiece{literal("Some text")}, nil, parts, phs)
	})

	t.Run("should serialize text with interpolation for $localize()", func(t *testing.T) {
		parts, phs := serialize("Some text {{ valueA }} and {{ valueB + valueC }} done")
		assertEqualPieces(t, []o.LiteralPiece{literal("Some text "), literal(" and "), literal(" done")}, []o.PlaceholderPiece{placeholder("INTERPOLATION"), placeholder("INTERPOLATION_1")}, parts, phs)
	})

	t.Run("should compute source-spans when serializing text with interpolation for $localize()", func(t *testing.T) {
		parts, phs := serialize("Some text {{ valueA }} and {{ valueB + valueC }} done")
		assert.Equal(t, "Some text ", parts[0].Text)
		assert.Equal(t, "Some text ", spanToString(parts[0].SourceSpan))
		assert.Equal(t, " and ", parts[1].Text)
		assert.Equal(t, " and ", spanToString(parts[1].SourceSpan))
		assert.Equal(t, " done", parts[2].Text)
		assert.Equal(t, " done", spanToString(parts[2].SourceSpan))

		assert.Equal(t, "INTERPOLATION", phs[0].Text)
		assert.Equal(t, "{{ valueA }}", spanToString(phs[0].SourceSpan))
		assert.Equal(t, "INTERPOLATION_1", phs[1].Text)
		assert.Equal(t, "{{ valueB + valueC }}", spanToString(phs[1].SourceSpan))
	})

	t.Run("should serialize text with interpolation at start for $localize()", func(t *testing.T) {
		parts, phs := serialize("{{ valueA }} and {{ valueB + valueC }} done")
		assertEqualPieces(t, []o.LiteralPiece{literal(""), literal(" and "), literal(" done")}, []o.PlaceholderPiece{placeholder("INTERPOLATION"), placeholder("INTERPOLATION_1")}, parts, phs)
	})

	t.Run("should serialize text with interpolation at end for $localize()", func(t *testing.T) {
		parts, phs := serialize("Some text {{ valueA }} and {{ valueB + valueC }}")
		assertEqualPieces(t, []o.LiteralPiece{literal("Some text "), literal(" and "), literal("")}, []o.PlaceholderPiece{placeholder("INTERPOLATION"), placeholder("INTERPOLATION_1")}, parts, phs)
	})

	t.Run("should serialize only interpolation for $localize()", func(t *testing.T) {
		parts, phs := serialize("{{ valueB + valueC }}")
		assertEqualPieces(t, []o.LiteralPiece{literal(""), literal("")}, []o.PlaceholderPiece{placeholder("INTERPOLATION")}, parts, phs)
	})

	t.Run("should serialize interpolation with named placeholder for $localize()", func(t *testing.T) {
		parts, phs := serialize("{{ valueB + valueC // i18n(ph=\"PLACEHOLDER NAME\") }}")
		assertEqualPieces(t, []o.LiteralPiece{literal(""), literal("")}, []o.PlaceholderPiece{placeholder("PLACEHOLDER_NAME")}, parts, phs)
	})

	t.Run("should serialize content with HTML tags for $localize()", func(t *testing.T) {
		parts, phs := serialize("A <span>B<div>C</div></span> D")
		assertEqualPieces(t, []o.LiteralPiece{literal("A "), literal("B"), literal("C"), literal(""), literal(" D")}, []o.PlaceholderPiece{
			placeholder("START_TAG_SPAN"),
			placeholder("START_TAG_DIV"),
			placeholder("CLOSE_TAG_DIV"),
			placeholder("CLOSE_TAG_SPAN"),
		}, parts, phs)
	})

	t.Run("should compute source-spans when serializing content with HTML tags for $localize()", func(t *testing.T) {
		parts, phs := serialize("A <span>B<div>C</div></span> D")
		assert.Equal(t, "A ", parts[0].Text)
		assert.Equal(t, "A ", spanToString(parts[0].SourceSpan))
		assert.Equal(t, "B", parts[1].Text)
		assert.Equal(t, "B", spanToString(parts[1].SourceSpan))
		assert.Equal(t, "C", parts[2].Text)
		assert.Equal(t, "C", spanToString(parts[2].SourceSpan))
		assert.Equal(t, "", parts[3].Text)
		assert.Equal(t, "", spanToString(parts[3].SourceSpan))
		assert.Equal(t, " D", parts[4].Text)
		assert.Equal(t, " D", spanToString(parts[4].SourceSpan))

		assert.Equal(t, "START_TAG_SPAN", phs[0].Text)
		assert.Equal(t, "<span>", spanToString(phs[0].SourceSpan))
		assert.Equal(t, "START_TAG_DIV", phs[1].Text)
		assert.Equal(t, "<div>", spanToString(phs[1].SourceSpan))
		assert.Equal(t, "CLOSE_TAG_DIV", phs[2].Text)
		assert.Equal(t, "</div>", spanToString(phs[2].SourceSpan))
		assert.Equal(t, "CLOSE_TAG_SPAN", phs[3].Text)
		assert.Equal(t, "</span>", spanToString(phs[3].SourceSpan))
	})

	t.Run("should create the correct source-spans when there are two placeholders next to each other", func(t *testing.T) {
		parts, phs := serialize("<b>{{value}}</b>")
		assert.Equal(t, "", parts[0].Text)
		assert.Equal(t, `"" (10-10)`, humanizeSourceSpan(parts[0].SourceSpan))
		assert.Equal(t, "", parts[1].Text)
		assert.Equal(t, `"" (13-13)`, humanizeSourceSpan(parts[1].SourceSpan))
		assert.Equal(t, "", parts[2].Text)
		assert.Equal(t, `"" (22-22)`, humanizeSourceSpan(parts[2].SourceSpan))
		assert.Equal(t, "", parts[3].Text)
		assert.Equal(t, `"" (26-26)`, humanizeSourceSpan(parts[3].SourceSpan))

		assert.Equal(t, "START_BOLD_TEXT", phs[0].Text)
		assert.Equal(t, `"<b>" (10-13)`, humanizeSourceSpan(phs[0].SourceSpan))
		assert.Equal(t, "INTERPOLATION", phs[1].Text)
		assert.Equal(t, `"{{value}}" (13-22)`, humanizeSourceSpan(phs[1].SourceSpan))
		assert.Equal(t, "CLOSE_BOLD_TEXT", phs[2].Text)
		assert.Equal(t, `"</b>" (22-26)`, humanizeSourceSpan(phs[2].SourceSpan))
	})

	t.Run("should create the correct placeholder source-spans when there is skipped leading whitespace", func(t *testing.T) {
		parts, phs := serialize("<b>   {{value}}</b>")
		assert.Equal(t, "", parts[0].Text)
		assert.Equal(t, `"" (10-10)`, humanizeSourceSpan(parts[0].SourceSpan))
		assert.Equal(t, "   ", parts[1].Text)
		assert.Equal(t, `"   " (13-16)`, humanizeSourceSpan(parts[1].SourceSpan))
		assert.Equal(t, "", parts[2].Text)
		assert.Equal(t, `"" (25-25)`, humanizeSourceSpan(parts[2].SourceSpan))
		assert.Equal(t, "", parts[3].Text)
		assert.Equal(t, `"" (29-29)`, humanizeSourceSpan(parts[3].SourceSpan))

		assert.Equal(t, "START_BOLD_TEXT", phs[0].Text)
		assert.Equal(t, `"<b>" (10-13)`, humanizeSourceSpan(phs[0].SourceSpan))
		assert.Equal(t, "INTERPOLATION", phs[1].Text)
		assert.Equal(t, `"{{value}}" (16-25)`, humanizeSourceSpan(phs[1].SourceSpan))
		assert.Equal(t, "CLOSE_BOLD_TEXT", phs[2].Text)
		assert.Equal(t, `"</b>" (25-29)`, humanizeSourceSpan(phs[2].SourceSpan))
	})

	t.Run("should serialize simple ICU for $localize()", func(t *testing.T) {
		parts, phs := serialize("{age, plural, 10 {ten} other {other}}")
		assertEqualPieces(t, []o.LiteralPiece{literal("{VAR_PLURAL, plural, 10 {ten} other {other}}")}, nil, parts, phs)
	})

	t.Run("should serialize nested ICUs for $localize()", func(t *testing.T) {
		parts, phs := serialize("{age, plural, 10 {ten {size, select, 1 {one} 2 {two} other {2+}}} other {other}}")
		assertEqualPieces(t, []o.LiteralPiece{literal("{VAR_PLURAL, plural, 10 {ten {VAR_SELECT, select, 1 {one} 2 {two} other {2+}}} other {other}}")}, nil, parts, phs)
	})

	t.Run("should serialize ICU with embedded HTML for $localize()", func(t *testing.T) {
		parts, phs := serialize("{age, plural, 10 {<b>ten</b>} other {<div class=\"A\">other</div>}}")
		assertEqualPieces(t, []o.LiteralPiece{literal("{VAR_PLURAL, plural, 10 {{START_BOLD_TEXT}ten{CLOSE_BOLD_TEXT}} other {{START_TAG_DIV}other{CLOSE_TAG_DIV}}}")}, nil, parts, phs)
	})

	t.Run("should serialize ICU with embedded interpolation for $localize()", func(t *testing.T) {
		parts, phs := serialize("{age, plural, 10 {<b>ten</b>} other {{{age}} years old}}")
		assertEqualPieces(t, []o.LiteralPiece{literal("{VAR_PLURAL, plural, 10 {{START_BOLD_TEXT}ten{CLOSE_BOLD_TEXT}} other {{INTERPOLATION} years old}}")}, nil, parts, phs)
	})

	t.Run("should serialize ICU with nested HTML containing further ICUs for $localize()", func(t *testing.T) {
		parts, phs := serialize("{gender, select, male {male} female {female} other {other}}<div>{gender, select, male {male} female {female} other {other}}</div>")
		icu := placeholder("ICU")
		assertEqualPieces(t, []o.LiteralPiece{literal(""), literal(""), literal(""), literal(""), literal("")}, []o.PlaceholderPiece{icu, placeholder("START_TAG_DIV"), icu, placeholder("CLOSE_TAG_DIV")}, parts, phs)
	})

	t.Run("should serialize nested ICUs with embedded interpolation for $localize()", func(t *testing.T) {
		parts, phs := serialize("{age, plural, 10 {ten {size, select, 1 {{{ varOne }}} 2 {{{ varTwo }}} other {2+}}} other {other}}")
		assertEqualPieces(t, []o.LiteralPiece{literal("{VAR_PLURAL, plural, 10 {ten {VAR_SELECT, select, 1 {{INTERPOLATION}} 2 {{INTERPOLATION_1}} other {2+}}} other {other}}")}, nil, parts, phs)
	})
}

func TestSerializeIcuNode(t *testing.T) {
	serialize := func(input string) string {
		msg := parseI18nMessage("<div i18n>" + input + "</div>")
		icu := msg.Nodes[0].(*ci18n.Icu)
		return SerializeIcuNode(icu)
	}

	t.Run("should serialize a simple ICU", func(t *testing.T) {
		assert.Equal(t, "{VAR_PLURAL, plural, 10 {ten} other {other}}", serialize("{age, plural, 10 {ten} other {other}}"))
	})

	t.Run("should serialize a nested ICU", func(t *testing.T) {
		assert.Equal(t, "{VAR_PLURAL, plural, 10 {ten {VAR_SELECT, select, 1 {one} 2 {two} other {2+}}} other {other}}",
			serialize("{age, plural, 10 {ten {size, select, 1 {one} 2 {two} other {2+}}} other {other}}"))
	})

	t.Run("should serialize ICU with nested HTML", func(t *testing.T) {
		assert.Equal(t, "{VAR_PLURAL, plural, 10 {{START_BOLD_TEXT}ten{CLOSE_BOLD_TEXT}} other {{START_TAG_DIV}other{CLOSE_TAG_DIV}}}",
			serialize("{age, plural, 10 {<b>ten</b>} other {<div class=\"A\">other</div>}}"))
	})

	t.Run("should serialize an ICU with embedded interpolations", func(t *testing.T) {
		assert.Equal(t, "{VAR_SELECT, select, 10 {ten} other {{INTERPOLATION} years old}}",
			serialize("{age, select, 10 {ten} other {{{age}} years old}}"))
	})
}

func createDummySourceSpan() *parse_util.ParseSourceSpan {
	sourceFile := parse_util.NewParseSourceFile("content", "file.html")
	start := parse_util.NewParseLocation(sourceFile, 0, 0, 0)
	end := parse_util.NewParseLocation(sourceFile, 7, 0, 7)
	return parse_util.NewParseSourceSpan(start, end, start, nil)
}

func literal(text string) o.LiteralPiece {
	return o.LiteralPiece{
		Text:       text,
		SourceSpan: o.NewSourceSpanAdapter(createDummySourceSpan()),
	}
}

func placeholder(name string, associatedMsg ...*ci18n.Message) o.PlaceholderPiece {
	var assoc o.Message = nil
	if len(associatedMsg) > 0 && associatedMsg[0] != nil {
		assoc = associatedMsg[0]
	}
	return o.PlaceholderPiece{
		Text:              name,
		SourceSpan:        o.NewSourceSpanAdapter(createDummySourceSpan()),
		AssociatedMessage: assoc,
	}
}

func testMeta(customId, meaning, description string, legacyIds ...string) I18nMeta {
	return I18nMeta{CustomId: customId, Meaning: meaning, Description: description, LegacyIds: legacyIds}
}

func humanizeSourceSpan(span o.ParseSourceSpan) string {
	if span == nil {
		return ""
	}
	adapter, ok := span.(*o.SourceSpanAdapter)
	if !ok || adapter.Span == nil {
		return ""
	}
	s := adapter.Span
	return fmt.Sprintf(`"%s" (%d-%d)`, s.ToString(), s.Start.Offset, s.End.Offset)
}

func spanToString(span o.ParseSourceSpan) string {
	if span == nil {
		return ""
	}
	if adapter, ok := span.(*o.SourceSpanAdapter); ok && adapter.Span != nil {
		return adapter.Span.ToString()
	}
	return ""
}

func assertEqualPieces(t *testing.T, expectedParts []o.LiteralPiece, expectedPhs []o.PlaceholderPiece, actualParts []o.LiteralPiece, actualPhs []o.PlaceholderPiece) {
	if !assert.Len(t, actualParts, len(expectedParts), "messageParts length mismatch") {
		return
	}
	for i := range expectedParts {
		assert.Equal(t, expectedParts[i].Text, actualParts[i].Text, "messageParts[%d].Text mismatch", i)
		assert.NotNil(t, actualParts[i].SourceSpan, "messageParts[%d].SourceSpan should not be nil", i)
	}

	if !assert.Len(t, actualPhs, len(expectedPhs), "placeHolders length mismatch") {
		return
	}
	for i := range expectedPhs {
		assert.Equal(t, expectedPhs[i].Text, actualPhs[i].Text, "placeHolders[%d].Text mismatch", i)
		assert.NotNil(t, actualPhs[i].SourceSpan, "placeHolders[%d].SourceSpan should not be nil", i)
		if expectedPhs[i].AssociatedMessage != nil {
			assert.NotNil(t, actualPhs[i].AssociatedMessage, "placeHolders[%d].AssociatedMessage should not be nil", i)
		}
	}
}
