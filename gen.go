t.Run("HtmlLexer", func(t *testing.T) {
t.Run("line/column numbers", func(t *testing.T) {
t.Run("should work without newlines", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "0:0"},
    []any{TokenTypeTagOpenEnd, "0:2"},
    []any{TokenTypeText, "0:3"},
    []any{TokenTypeTagClose, "0:4"},
    []any{TokenTypeEOF, "0:8"},
}, tokenizeAndHumanizeLineColumn("<t>a</t>", nil))
});
t.Run("should work with one newline", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "0:0"},
    []any{TokenTypeTagOpenEnd, "0:2"},
    []any{TokenTypeText, "0:3"},
    []any{TokenTypeTagClose, "1:1"},
    []any{TokenTypeEOF, "1:5"},
}, tokenizeAndHumanizeLineColumn("<t>\na</t>", nil))
});
t.Run("should work with multiple newlines", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "0:0"},
    []any{TokenTypeTagOpenEnd, "1:0"},
    []any{TokenTypeText, "1:1"},
    []any{TokenTypeTagClose, "2:1"},
    []any{TokenTypeEOF, "2:5"},
}, tokenizeAndHumanizeLineColumn("<t\n>\na</t>", nil))
});
t.Run("should work with CR and LF", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "0:0"},
    []any{TokenTypeTagOpenEnd, "1:0"},
    []any{TokenTypeText, "1:1"},
    []any{TokenTypeTagClose, "2:1"},
    []any{TokenTypeEOF, "2:5"},
}, tokenizeAndHumanizeLineColumn("<t\n>\r\na\r</t>", nil))
});
t.Run("should skip over leading trivia for source-span start", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "0:0", "0:0"},
    []any{TokenTypeTagOpenEnd, "0:2", "0:2"},
    []any{TokenTypeText, "1:3", "0:3"},
    []any{TokenTypeTagClose, "1:4", "1:4"},
    []any{TokenTypeEOF, "1:8", "1:8"},
}, tokenizeAndHumanizeFullStart("<t>\n \t a</t>", &TokenizeOptions{LeadingTriviaChars: []string{"\n", " ", "\t"}}))
});
});
t.Run("content ranges", func(t *testing.T) {
t.Run("should only process the text within the range", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "line 1\nline 2\nline 3"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("pre 1\npre 2\npre 3 `line 1\nline 2\nline 3` post 1\n post 2\n post 3", &TokenizeOptions{Range: &LexerRange{StartPos: 19, StartLine: 2, StartCol: 7, EndPos: 39}}))
});
t.Run("should take into account preceding (non-processed) lines and columns", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "2:7"},
    []any{TokenTypeEOF, "4:6"},
}, tokenizeAndHumanizeLineColumn("pre 1\npre 2\npre 3 `line 1\nline 2\nline 3` post 1\n post 2\n post 3", &TokenizeOptions{Range: &LexerRange{StartPos: 19, StartLine: 2, StartCol: 7, EndPos: 39}}))
});
});
t.Run("comments", func(t *testing.T) {
t.Run("should parse comments", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeCommentStart},
    []any{TokenTypeRawText, "t\ne\ns\nt"},
    []any{TokenTypeCommentEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<!--t\ne\rs\r\nt-->", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeCommentStart, "<!--"},
    []any{TokenTypeRawText, "t\ne\rs\r\nt"},
    []any{TokenTypeCommentEnd, "-->"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<!--t\ne\rs\r\nt-->", nil))
});
t.Run("should report <!- without -", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"a\"", "0:3"},
}, tokenizeAndHumanizeErrors("<!-a", nil))
});
t.Run("should report missing end comment", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:4"},
}, tokenizeAndHumanizeErrors("<!--", nil))
});
t.Run("should accept comments finishing by too many dashes (even number)", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeCommentStart, "<!--"},
    []any{TokenTypeRawText, " test --"},
    []any{TokenTypeCommentEnd, "-->"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<!-- test ---->", nil))
});
t.Run("should accept comments finishing by too many dashes (odd number)", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeCommentStart, "<!--"},
    []any{TokenTypeRawText, " test -"},
    []any{TokenTypeCommentEnd, "-->"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<!-- test --->", nil))
});
});
t.Run("doctype", func(t *testing.T) {
t.Run("should parse doctypes", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeDocType, "DOCTYPE html"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<!DOCTYPE html>", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeDocType, "<!DOCTYPE html>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<!DOCTYPE html>", nil))
});
t.Run("should report missing end doctype", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:2"},
}, tokenizeAndHumanizeErrors("<!", nil))
});
});
t.Run("CDATA", func(t *testing.T) {
t.Run("should parse CDATA", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeCdataStart},
    []any{TokenTypeRawText, "t\ne\ns\nt"},
    []any{TokenTypeCdataEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<![CDATA[t\ne\rs\r\nt]]>", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeCdataStart, "<![CDATA["},
    []any{TokenTypeRawText, "t\ne\rs\r\nt"},
    []any{TokenTypeCdataEnd, "]]>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<![CDATA[t\ne\rs\r\nt]]>", nil))
});
t.Run("should report <![ without CDATA[", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"a\"", "0:3"},
}, tokenizeAndHumanizeErrors("<![a", nil))
});
t.Run("should report missing end cdata", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:9"},
}, tokenizeAndHumanizeErrors("<![CDATA[", nil))
});
});
t.Run("open tags", func(t *testing.T) {
t.Run("should parse open tags without prefix", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "test"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<test>", nil))
});
t.Run("should parse namespace prefix", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "ns1", "test"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<ns1:test>", nil))
});
t.Run("should parse void tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "test"},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<test/>", nil))
});
t.Run("should allow whitespace after the tag name", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "test"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<test >", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<test"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<test>", nil))
});
t.Run("tags", func(t *testing.T) {
t.Run("terminated with EOF", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "<div"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<div", nil))
});
t.Run("after tag name", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "<div"},
    []any{TokenTypeTagOpenStart, "<span"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeIncompleteTagOpen, "<div"},
    []any{TokenTypeTagClose, "</span>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<div<span><div</span>", nil))
});
t.Run("in attribute", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "<div"},
    []any{TokenTypeAttrName, "class"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "hi"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "sty"},
    []any{TokenTypeTagOpenStart, "<span"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeTagClose, "</span>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<div class=\"hi\" sty<span></span>", nil))
});
t.Run("after quote", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "<div"},
    []any{TokenTypeText, "\""},
    []any{TokenTypeTagOpenStart, "<span"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeTagClose, "</span>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<div \"<span></span>", nil))
});
});
t.Run("component tags", func(t *testing.T) {
options := {selectorlessEnabled: true}
t.Run("should parse a basic component tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeComponentOpenStart, "MyComp", "", ""},
    []any{TokenTypeComponentOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeComponentClose, "MyComp", "", ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<MyComp>hello</MyComp>", nil))
});
t.Run("should parse a component tag with a tag name", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeComponentOpenStart, "MyComp", "", "button"},
    []any{TokenTypeComponentOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeComponentClose, "MyComp", "", "button"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<MyComp:button>hello</MyComp:button>", nil))
});
t.Run("should parse a component tag with a tag name and namespace", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeComponentOpenStart, "MyComp", "svg", "title"},
    []any{TokenTypeComponentOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeComponentClose, "MyComp", "svg", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<MyComp:svg:title>hello</MyComp:svg:title>", nil))
});
t.Run("should parse a self-closing component tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeComponentOpenStart, "MyComp", "", ""},
    []any{TokenTypeComponentOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<MyComp/>", nil))
});
t.Run("should produce spans for component tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeComponentOpenStart, "<MyComp:svg:title"},
    []any{TokenTypeComponentOpenEnd, ">"},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeComponentClose, "</MyComp:svg:title>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<MyComp:svg:title>hello</MyComp:svg:title>", nil))
});
t.Run("should parse an incomplete component open tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteComponentOpen, "MyComp", "", "span"},
    []any{TokenTypeAttrName, "", "class"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "hi"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "sty"},
    []any{TokenTypeTagOpenStart, "", "span"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "span"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<MyComp:span class=\"hi\" sty<span></span>", nil))
});
t.Run("should parse a component tag with raw text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeComponentOpenStart, "MyComp", "", "script"},
    []any{TokenTypeComponentOpenEnd},
    []any{TokenTypeRawText, "t\ne\ns\nt"},
    []any{TokenTypeComponentClose, "MyComp", "", "script"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<MyComp:script>t\ne\rs\r\nt</MyComp:script>", nil))
});
t.Run("should parse a component tag with escapable raw text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeComponentOpenStart, "MyComp", "", "title"},
    []any{TokenTypeComponentOpenEnd},
    []any{TokenTypeEscapableRawText, "t\ne\ns\nt"},
    []any{TokenTypeComponentClose, "MyComp", "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<MyComp:title>t\ne\rs\r\nt</MyComp:title>", nil))
});
});
t.Run("selectorless directives", func(t *testing.T) {
options := {selectorlessEnabled: true}
t.Run("should parse a basic directive", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeDirectiveName, "MyDir"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div @MyDir></div>", nil))
});
t.Run("should parse a directive with parentheses, but no attributes", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeDirectiveName, "MyDir"},
    []any{TokenTypeDirectiveOpen},
    []any{TokenTypeDirectiveClose},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div @MyDir()></div>", nil))
});
t.Run("should parse a directive with a single attribute without a value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeDirectiveName, "MyDir"},
    []any{TokenTypeDirectiveOpen},
    []any{TokenTypeAttrName, "", "foo"},
    []any{TokenTypeDirectiveClose},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div @MyDir(foo)></div>", nil))
});
t.Run("should parse a directive with attributes", func(t *testing.T) {
tokens := tokenizeAndHumanizeParts("<div @MyDir(static=\"one\" [bound]=\"expr\" [(twoWay)]=\"expr\" #ref=\"name\" (click)=\"handler()\")></div>", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeDirectiveName, "MyDir"},
    []any{TokenTypeDirectiveOpen},
    []any{TokenTypeAttrName, "", "static"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "one"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "[bound]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "[(twoWay)]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "#ref"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "name"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "(click)"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "handler()"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeDirectiveClose},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokens)
});
t.Run("should parse a directive mixed in with other attributes", func(t *testing.T) {
tokens := tokenizeAndHumanizeParts("<div before=\"value\" @OneDir([one]=\"1\" two=\"2\") middle @TwoDir @ThreeDir((three)=\"handleThree()\") after=\"value\"></div>", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeAttrName, "", "before"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "value"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeDirectiveName, "OneDir"},
    []any{TokenTypeDirectiveOpen},
    []any{TokenTypeAttrName, "", "[one]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "1"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "two"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "2"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeDirectiveClose},
    []any{TokenTypeAttrName, "", "middle"},
    []any{TokenTypeDirectiveName, "TwoDir"},
    []any{TokenTypeDirectiveName, "ThreeDir"},
    []any{TokenTypeDirectiveOpen},
    []any{TokenTypeAttrName, "", "(three)"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "handleThree()"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeDirectiveClose},
    []any{TokenTypeAttrName, "", "after"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "value"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokens)
});
t.Run("should not pick up selectorless-like text inside a tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "@MyDir()"},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div>@MyDir()</div>", nil))
});
t.Run("should not pick up selectorless-like text inside an attribute", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeAttrName, "", "hello"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "@MyDir"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div hello=\"@MyDir\"></div>", nil))
});
t.Run("should produce spans for directives", func(t *testing.T) {
tokens := tokenizeAndHumanizeSourceSpans("<div @Empty @NoAttrs() @WithAttr([one]=\"1\" two=\"2\") @WithSimpleAttr(simple)></div>", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<div"},
    []any{TokenTypeDirectiveName, "@Empty"},
    []any{TokenTypeDirectiveName, "@NoAttrs"},
    []any{TokenTypeDirectiveOpen, "("},
    []any{TokenTypeDirectiveClose, ")"},
    []any{TokenTypeDirectiveName, "@WithAttr"},
    []any{TokenTypeDirectiveOpen, "("},
    []any{TokenTypeAttrName, "[one]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "1"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "two"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "2"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeDirectiveClose, ")"},
    []any{TokenTypeDirectiveName, "@WithSimpleAttr"},
    []any{TokenTypeDirectiveOpen, "("},
    []any{TokenTypeAttrName, "simple"},
    []any{TokenTypeDirectiveClose, ")"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeTagClose, "</div>"},
    []any{TokenTypeEOF, ""},
}, tokens)
});
t.Run("should not capture whitespace in directive spans", func(t *testing.T) {
tokens := tokenizeAndHumanizeSourceSpans("<div    @Dir   (  one=\"1\"    (two)=\"handleTwo()\"     )     ></div>", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<div"},
    []any{TokenTypeDirectiveName, "@Dir"},
    []any{TokenTypeDirectiveOpen, "("},
    []any{TokenTypeAttrName, "one"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "1"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "(two)"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "handleTwo()"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeDirectiveClose, ")"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeTagClose, "</div>"},
    []any{TokenTypeEOF, ""},
}, tokens)
});
});
t.Run("escapable raw text", func(t *testing.T) {
t.Run("should parse text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "t\ne\ns\nt"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>t\ne\rs\r\nt</title>", nil))
});
t.Run("should detect entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, ""},
    []any{TokenTypeEncodedEntity, "&", "&amp;"},
    []any{TokenTypeEscapableRawText, ""},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>&amp;</title>", nil))
});
t.Run("should ignore other opening tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "a<div>"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>a<div></title>", nil))
});
t.Run("should ignore other closing tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "a</test>"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>a</test></title>", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<title"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeEscapableRawText, "a"},
    []any{TokenTypeTagClose, "</title>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<title>a</title>", nil))
});
});
t.Run("parsable data", func(t *testing.T) {
t.Run("should parse an SVG <title> tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "svg", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "test"},
    []any{TokenTypeTagClose, "svg", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<svg:title>test</svg:title>", nil))
});
t.Run("should parse an SVG <title> tag with children", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "svg", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagOpenStart, "", "f"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "test"},
    []any{TokenTypeTagClose, "", "f"},
    []any{TokenTypeTagClose, "svg", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<svg:title><f>test</f></svg:title>", nil))
});
});
t.Run("expansion forms", func(t *testing.T) {
t.Run("should parse an expansion form", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=5"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "five"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "foo"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "bar"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 {four} =5 {five} foo {bar} }", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion form with text elements surrounding it", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "before"},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "after"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("before{one.two, three, =4 {four}}after", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion form as a tag single child", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagOpenStart, "", "span"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "a"},
    []any{TokenTypeRawText, "b"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "c"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeTagClose, "", "span"},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div><span>{a, b, =4 {c}}</span></div>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion form with whitespace surrounding it", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagOpenStart, "", "span"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, " "},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "a"},
    []any{TokenTypeRawText, "b"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "c"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, " "},
    []any{TokenTypeTagClose, "", "span"},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div><span> {a, b, =4 {c}} </span></div>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion forms with elements in it", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four "},
    []any{TokenTypeTagOpenStart, "", "b"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "a"},
    []any{TokenTypeTagClose, "", "b"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 {four <b>a</b>}}", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion forms containing an interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four "},
    []any{TokenTypeInterpolation, "{{", "a", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 {four {{a}}}}", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse nested expansion forms", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "xx"},
    []any{TokenTypeRawText, "yy"},
    []any{TokenTypeExpansionCaseValue, "=x"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "one"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, " "},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 { {xx, yy, =x {one}} }}", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("[line ending normalization", func(t *testing.T) {
t.Run("{escapedString: true}", func(t *testing.T) {
t.Run("should normalize line-endings in expansion forms if `i18nNormalizeLineEndingsInICUs` is true", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, []Token(nil), result.NonNormalizedIcuExpressions)
});
t.Run("should not normalize line-endings in ICU expressions when `i18nNormalizeLineEndingsInICUs` is not defined", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n    messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
});
t.Run("should not normalize line endings in nested expansion forms when `i18nNormalizeLineEndingsInICUs` is not defined", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n  messages.length, plural,\r\n  =0 { zero \r\n       {\r\n         p.gender, select,\r\n         male {m}\r\n       }\r\n     }\r\n}", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n  messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "zero \n       "},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n         p.gender"},
    []any{TokenTypeRawText, "select"},
    []any{TokenTypeExpansionCaseValue, "male"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "m"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n     "},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n  messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
assert.Equal(t, '\r\n         p.gender', result.NonNormalizedIcuExpressions![1].sourceSpan.toString())
});
});
t.Run("{escapedString: false}", func(t *testing.T) {
t.Run("should normalize line-endings in expansion forms if `i18nNormalizeLineEndingsInICUs` is true", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, []Token(nil), result.NonNormalizedIcuExpressions)
});
t.Run("should not normalize line-endings in ICU expressions when `i18nNormalizeLineEndingsInICUs` is not defined (escapedString:false)", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n    messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
});
t.Run("should not normalize line endings in nested expansion forms when `i18nNormalizeLineEndingsInICUs` is not defined", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n  messages.length, plural,\r\n  =0 { zero \r\n       {\r\n         p.gender, select,\r\n         male {m}\r\n       }\r\n     }\r\n}", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n  messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "zero \n       "},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n         p.gender"},
    []any{TokenTypeRawText, "select"},
    []any{TokenTypeExpansionCaseValue, "male"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "m"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n     "},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n  messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
assert.Equal(t, '\r\n         p.gender', result.NonNormalizedIcuExpressions![1].sourceSpan.toString())
});
});
});
});
t.Run("errors", func(t *testing.T) {
t.Run("should report unescaped \"{\" on error", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\" (Do you have an unescaped \"{\" in your template? Use \"{{ '{' }}\") to escape it.)", "0:21"},
}, tokenizeAndHumanizeErrors("<p>before { after</p>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should report unescaped \"{\" as an error, even after a prematurely terminated interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\" (Do you have an unescaped \"{\" in your template? Use \"{{ '{' }}\") to escape it.)", "0:56"},
}, tokenizeAndHumanizeErrors("<code>{{b}<!---->}</code><pre>import {a} from 'a';</pre>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should include 2 lines of context in message", func(t *testing.T) {
src := "111\n222\n333\nE\n444\n555\n666\n"
file := new ParseSourceFile(src, 'file://')
location := new ParseLocation(file, 12, 123, 456)
span := new ParseSourceSpan(location, location)
error := new ParseError(span, '**ERROR**')
assert.Equal(t, `**ERROR** ("\n222\n333\n[ERROR ->]E\n444\n555\n"): file://@123:456`, error.toString())
});
});
t.Run("unicode characters", func(t *testing.T) {
t.Run("should support unicode characters", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<p"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "İ"},
    []any{TokenTypeTagClose, "</p>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<p>İ</p>", nil))
});
});
t.Run("(processing escaped strings)", func(t *testing.T) {
t.Run("should unescape standard escape sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "' ' '"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\' \\' \\'", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\" \" \""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\\" \\\" \\\"", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "` ` `"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\` \\` \\`", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\\ \\ \\"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\\\ \\\\ \\\\", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\n \n \n"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\n \\n \\n", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\n"},
    []any{TokenTypeInterpolation, "{{", "\n", "}}"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\r{{\\r}}\\r", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u000b \u000b \u000b"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\v \\v \\v", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\t \t \t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\t \\t \\t", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\b \b \b"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\b \\b \\b", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\f \f \f"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\f \\f \\f", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "' \" ` \\ \n \n \u000b \t \b \f"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\' \\\" \\` \\\\ \\n \\r \\v \\t \\b \\f", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape null sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\0", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\09", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape octal sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u0001 \u0001 \u0001 \n  \u00019 4 999"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\001 \\01 \\1 \\12 \\223 \\19 \\2234 \\999", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape hex sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u0012 O Ü"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\x12 \\x4F \\xDC", &TokenizeOptions{EscapedString: true}))
});
t.Run("should report an error on an invalid hex sequence", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:2"},
}, tokenizeAndHumanizeErrors("\\xGG", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:6"},
}, tokenizeAndHumanizeErrors("abc \\x xyz", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:5"},
}, tokenizeAndHumanizeErrors("abc\\x", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape fixed length Unicode sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "ģ ꯍ"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\u0123 \\uABCD", &TokenizeOptions{EscapedString: true}))
});
t.Run("should error on an invalid fixed length Unicode sequence", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:2"},
}, tokenizeAndHumanizeErrors("\\uGGGG", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape variable length Unicode sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u0001 ઼ ሴ 𒎫"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\u{01} \\u{ABC} \\u{1234} \\u{123AB}", &TokenizeOptions{EscapedString: true}))
});
t.Run("should error on an invalid variable length Unicode sequence", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:3"},
}, tokenizeAndHumanizeErrors("\\u{GG}", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape line continuations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "abcdef"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("abc\\\ndef", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "xy"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\\nx\\\ny\\\n", &TokenizeOptions{EscapedString: true}))
});
t.Run("should remove backslash from \"non-escape\" sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a g ~"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("a g ~", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape sequences in plain text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "abc\ndef\nghi\tjkl`'\"mno"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("abc\ndef\\nghi\\tjkl\\`\\'\\\"mno", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape sequences in raw text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "script"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeRawText, "abc\ndef\nghi\tjkl`'\"mno"},
    []any{TokenTypeTagClose, "", "script"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<script>abc\ndef\\nghi\\tjkl\\`\\'\\\"mno</script>", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape sequences in escapable raw text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "abc\ndef\nghi\tjkl`'\"mno"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>abc\ndef\\nghi\\tjkl\\`\\'\\\"mno</title>", &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escape sequences in tag definitions", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "c"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeAttrValueText, "d"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\\\"b\\\" \\n c=\\'d\\'>", &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escaped new line in tag definitions", func(t *testing.T) {
text := "<t\\n></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escaped characters in tag definitions", func(t *testing.T) {
text := "<t\u0013></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape characters in tag names", func(t *testing.T) {
text := "<t\\x64></t\\x64>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "td"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "td"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<t\\x64"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeTagClose, "</t\\x64>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape characters in attributes", func(t *testing.T) {
text := "<t \\x64=\"\\x65\"></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "d"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "e"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escaped new line in attribute values", func(t *testing.T) {
text := "<t a=b\\n></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should tokenize the correct span when there are escape sequences", func(t *testing.T) {
text := "selector: \"app-root\",\ntemplate: \"line 1\\n\\\"line 2\\\"\\nline 3\",\ninputs: []"
range := {
          startPos: 33,
          startLine: 1,
          startCol: 10,
          endPos: 59,
        }
assert.Equal(t, [][]any{
    []any{TokenTypeText, "line 1\n\"line 2\"\nline 3"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{Range: nil, EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "line 1\\n\\\"line 2\\\"\\nline 3"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans(text, &TokenizeOptions{Range: nil, EscapedString: true}))
});
t.Run("should account for escape sequences when computing source spans ", func(t *testing.T) {
text := "<t>line 1</t>\n<t>line 2</t>\\n<t>line 3\\\n</t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "line 1"},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "line 2"},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "line 3"},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "0:0"},
    []any{TokenTypeTagOpenEnd, "0:2"},
    []any{TokenTypeText, "0:3"},
    []any{TokenTypeTagClose, "0:9"},
    []any{TokenTypeText, "0:13"},
    []any{TokenTypeTagOpenStart, "1:0"},
    []any{TokenTypeTagOpenEnd, "1:2"},
    []any{TokenTypeText, "1:3"},
    []any{TokenTypeTagClose, "1:9"},
    []any{TokenTypeText, "1:13"},
    []any{TokenTypeTagOpenStart, "1:15"},
    []any{TokenTypeTagOpenEnd, "1:17"},
    []any{TokenTypeText, "1:18"},
    []any{TokenTypeTagClose, "2:0"},
    []any{TokenTypeEOF, "2:4"},
}, tokenizeAndHumanizeLineColumn(text, &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<t"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "line 1"},
    []any{TokenTypeTagClose, "</t>"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeTagOpenStart, "<t"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "line 2"},
    []any{TokenTypeTagClose, "</t>"},
    []any{TokenTypeText, "\\n"},
    []any{TokenTypeTagOpenStart, "<t"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "line 3\\\n"},
    []any{TokenTypeTagClose, "</t>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans(text, &TokenizeOptions{EscapedString: true}))
});
});
});
t.Run("@let declarations", func(t *testing.T) {
t.Run("should parse a @let declaration", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "123 + 456"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = 123 + 456;", nil))
});
t.Run("should parse @let declarations with arbitrary number of spaces", func(t *testing.T) {
expected := [
        [TokenType.LET_START, 'foo'],
        [TokenType.LET_VALUE, '123 + 456'],
        [TokenType.LET_END],
        [TokenType.EOF],
      ]
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let               foo       =          123 + 456;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let foo=123 + 456;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let foo =123 + 456;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let foo=   123 + 456;", nil))
});
t.Run("should parse a @let declaration with newlines before/after its name", func(t *testing.T) {
expected := [
        [TokenType.LET_START, 'foo'],
        [TokenType.LET_VALUE, '123'],
        [TokenType.LET_END],
        [TokenType.EOF],
      ]
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let\nfoo = 123;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let    \nfoo = 123;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let    \n              foo = 123;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let foo\n= 123;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let foo\n       = 123;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let foo   \n   = 123;", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@let  \n   foo   \n   = 123;", nil))
});
t.Run("should parse a @let declaration with new lines in its value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "123 + \n 456 + \n789\n"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = \n123 + \n 456 + \n789\n;", nil))
});
t.Run("should parse a @let declaration inside of a block", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "defer"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "123 + 456"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@defer {@let foo = 123 + 456;}", nil))
});
t.Run("should parse @let declaration using semicolon inside of a string", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "'a; b'"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = 'a; b';", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "\"';'\""},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = \"';'\";", nil))
});
t.Run("should parse @let declaration using escaped quotes in a string", func(t *testing.T) {
markup := "@let foo = '\\';\\'' + \"\\\",\";"
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "'\\';\\'' + \"\\\",\""},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(markup, nil))
});
t.Run("should parse @let declaration using function calls in its value", func(t *testing.T) {
markup := "@let foo = fn(a, b) + fn2(c, d, e);"
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "fn(a, b) + fn2(c, d, e)"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(markup, nil))
});
t.Run("should parse @let declarations using array literals in their value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "[1, 2, 3]"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = [1, 2, 3];", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "[0, [foo[1]], 3]"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = [0, [foo[1]], 3];", nil))
});
t.Run("should parse @let declarations using object literals", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "{a: 1, b: {c: something + 2}}"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = {a: 1, b: {c: something + 2}};", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "{}"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = {};", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "{foo: \";\"}"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = {foo: \";\"};", nil))
});
t.Run("should parse a @let declaration containing complex expression", func(t *testing.T) {
markup := "@let foo = fn({a: 1, b: [otherFn([{c: \";\"}], 321, {d: [',']})]});"
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "fn({a: 1, b: [otherFn([{c: \";\"}], 321, {d: [',']})]})"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(markup, nil))
});
t.Run("should handle @let declaration with invalid syntax in the value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:13"},
}, tokenizeAndHumanizeErrors("@let foo = \";", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "{a: 1,"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = {a: 1,;", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "[1, "},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = [1, ;", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, "fn("},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = fn(;", nil))
});
t.Run("should parse a @let declaration without a value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "foo"},
    []any{TokenTypeLetValue, ""},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo =;", nil))
});
t.Run("should handle no space after @let", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, "@let"},
    []any{TokenTypeText, "Foo = 123;"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@letFoo = 123;", nil))
});
t.Run("should handle unsupported characters in the name of @let", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, "foo"},
    []any{TokenTypeText, "\\bar = 123;"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo\\bar = 123;", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, ""},
    []any{TokenTypeText, "#foo = 123;"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let #foo = 123;", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, "foo"},
    []any{TokenTypeText, "bar = 123;"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo\nbar = 123;", nil))
});
t.Run("should handle digits in the name of an @let", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeLetStart, "a123"},
    []any{TokenTypeLetValue, "foo"},
    []any{TokenTypeLetEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let a123 = foo;", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, ""},
    []any{TokenTypeText, "123a = 123;"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let 123a = 123;", nil))
});
t.Run("should handle an @let declaration without an ending token", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, "foo"},
    []any{TokenTypeLetValue, "123 + 456"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = 123 + 456", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, "foo"},
    []any{TokenTypeLetValue, "123 + 456                  "},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = 123 + 456                  ", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteLet, "foo"},
    []any{TokenTypeLetValue, "123, bar = 456"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@let foo = 123, bar = 456", nil))
});
t.Run("should not parse @let inside an interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " @let foo = 123; ", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ @let foo = 123; }}", nil))
});
});
t.Run("attributes", func(t *testing.T) {
t.Run("should parse attributes without prefix", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a>", nil))
});
t.Run("should parse attributes with interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrValueInterpolation, "{{", "v", "}}"},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "b"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "s"},
    []any{TokenTypeAttrValueInterpolation, "{{", "m", "}}"},
    []any{TokenTypeAttrValueText, "e"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "c"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "s"},
    []any{TokenTypeAttrValueInterpolation, "{{", "m//c", "}}"},
    []any{TokenTypeAttrValueText, "e"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"{{v}}\" b=\"s{{m}}e\" c=\"s{{m//c}}e\">", nil))
});
t.Run("should end interpolation on an unescaped matching quote", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrValueInterpolation, "{{", " a \\\" ' b "},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"{{ a \\\" ' b \">", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrValueInterpolation, "{{", " a \" \\' b "},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a='{{ a \" \\' b '>", nil))
});
t.Run("should parse attributes with prefix", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "ns1", "a"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t ns1:a>", nil))
});
t.Run("should parse attributes whose prefix is not valid", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "(ns1:a)"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t (ns1:a)>", nil))
});
t.Run("should parse attributes with single quote value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a='b'>", nil))
});
t.Run("should parse attributes with double quote value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"b\">", nil))
});
t.Run("should parse attributes with unquoted value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=b>", nil))
});
t.Run("should parse attributes with unquoted interpolation value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "a"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrValueInterpolation, "{{", "link.text", "}}"},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<a a={{link.text}}>", nil))
});
t.Run("should parse bound inputs with expressions containing newlines", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "app-component"},
    []any{TokenTypeAttrName, "", "[attr]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "[\n        {text: 'some text',url:'//www.google.com'},\n        {text:'other text',url:'//www.google.com'}]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<app-component\n        [attr]=\"[\n        {text: 'some text',url:'//www.google.com'},\n        {text:'other text',url:'//www.google.com'}]\">", nil))
});
t.Run("should parse attributes with empty quoted value", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"\">", nil))
});
t.Run("should allow whitespace", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a = b >", nil))
});
t.Run("should parse attributes with entities in values", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeEncodedEntity, "A", "&#65;"},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeEncodedEntity, "A", "&#x41;"},
    []any{TokenTypeAttrValueText, ""},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"&#65;&#x41;\">", nil))
});
t.Run("should not decode entities without trailing \";\"", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "&amp"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "b"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "c&&d"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"&amp\" b=\"c&&d\">", nil))
});
t.Run("should parse attributes with \"&\" in values", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "b && c &"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"b && c &\">", nil))
});
t.Run("should parse values with CR and LF", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeAttrValueText, "t\ne\ns\nt"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a='t\ne\rs\r\nt'>", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<t"},
    []any{TokenTypeAttrName, "a"},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<t a=b>", nil))
});
t.Run("should report missing closing single quote", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:8"},
}, tokenizeAndHumanizeErrors("<t a='b>", nil))
});
t.Run("should report missing closing double quote", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:8"},
}, tokenizeAndHumanizeErrors("<t a=\"b>", nil))
});
t.Run("should permit more characters in square-bracketed attributes", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.text-primary/80]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.text-primary/80]=\"expr\"/>", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.data-active:text-green-300/80]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.data-active:text-green-300/80]=\"expr\"/>", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.data-[size='large']:p-8]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.data-[size='large']:p-8] = \"expr\"/>", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.data-[size='large']:p-8]"},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.data-[size='large']:p-8]/>", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.data-[size='hello white space']]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.data-[size='hello white space']]=\"expr\"/>", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.text-primary/80]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "[class.data-active:text-green-300/80]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr2"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "[class.data-[size='large']:p-8]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr3"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "some-attr"},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.text-primary/80]=\"expr\" [class.data-active:text-green-300/80]=\"expr2\" [class.data-[size='large']:p-8] = \"expr3\" some-attr/>", nil))
});
t.Run("should allow mismatched square brackets in attribute name", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.a]b]c]"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "expr"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.a]b]c]=\"expr\"/>", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.a[]][[]]b]][c]"},
    []any{TokenTypeTagOpenEndVoid},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.a[]][[]]b]][c]/>", nil))
});
t.Run("should stop permissive parsing of square brackets on new line", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "", "foo"},
    []any{TokenTypeAttrName, "", "[class.text-"},
    []any{TokenTypeAttrName, "", "primary"},
    []any{TokenTypeText, "80]=\"expr\"/>"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<foo [class.text-\nprimary/80]=\"expr\"/>", nil))
});
});
t.Run("closing tags", func(t *testing.T) {
t.Run("should parse closing tags without prefix", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagClose, "", "test"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("</test>", nil))
});
t.Run("should parse closing tags with prefix", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagClose, "ns1", "test"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("</ns1:test>", nil))
});
t.Run("should allow whitespace", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagClose, "", "test"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("</ test >", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagClose, "</test>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("</test>", nil))
});
t.Run("should report missing name after </", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:2"},
}, tokenizeAndHumanizeErrors("</", nil))
});
t.Run("should report missing >", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:6"},
}, tokenizeAndHumanizeErrors("</test", nil))
});
});
t.Run("entities", func(t *testing.T) {
t.Run("should parse named entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a"},
    []any{TokenTypeEncodedEntity, "&", "&amp;"},
    []any{TokenTypeText, "b"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("a&amp;b", nil))
});
t.Run("should parse named entities containing digits", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "¹", "&sup1;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("&sup1;", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "½", "&frac12;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("&frac12;", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "▓", "&blk34;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("&blk34;", nil))
});
t.Run("should parse hexadecimal entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "A", "&#x41;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "A", "&#X41;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("&#x41;&#X41;", nil))
});
t.Run("should parse decimal entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "A", "&#65;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("&#65;", nil))
});
t.Run("should parse entities with more than 4 hex digits", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "🛈", "&#x1F6C8;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("&#x1F6C8;", nil))
});
t.Run("should parse entities with more than 4 decimal digits", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeEncodedEntity, "🛈", "&#128712;"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("&#128712;", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a"},
    []any{TokenTypeEncodedEntity, "&amp;"},
    []any{TokenTypeText, "b"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("a&amp;b", nil))
});
t.Run("should report malformed/unknown entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unknown entity \"tbo\" - use the \"&#<decimal>;\" or  \"&#x<hex>;\" syntax", "0:0"},
}, tokenizeAndHumanizeErrors("&tbo;", nil))
assert.Equal(t, [][]any{
    []any{"Unable to parse entity \"&#3s\" - decimal character reference entities must end with \";\"", "0:4"},
}, tokenizeAndHumanizeErrors("&#3sdf;", nil))
assert.Equal(t, [][]any{
    []any{"Unable to parse entity \"&#xas\" - hexadecimal character reference entities must end with \";\"", "0:5"},
}, tokenizeAndHumanizeErrors("&#xasdf;", nil))
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:6"},
}, tokenizeAndHumanizeErrors("&#xABC", nil))
});
t.Run("should not parse js object methods", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unknown entity \"valueOf\" - use the \"&#<decimal>;\" or  \"&#x<hex>;\" syntax", "0:0"},
}, tokenizeAndHumanizeErrors("&valueOf;", nil))
});
});
t.Run("regular text", func(t *testing.T) {
t.Run("should parse text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("a", nil))
});
t.Run("should parse interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " a ", "}}"},
    []any{TokenTypeText, "b"},
    []any{TokenTypeInterpolation, "{{", " c // comment ", "}}"},
    []any{TokenTypeText, "d"},
    []any{TokenTypeInterpolation, "{{", " e \"}} ' \" f ", "}}"},
    []any{TokenTypeText, "g"},
    []any{TokenTypeInterpolation, "{{", " h // \" i ", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ a }}b{{ c // comment }}d{{ e \"}} ' \" f }}g{{ h // \" i }}", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{ a }}"},
    []any{TokenTypeText, "b"},
    []any{TokenTypeInterpolation, "{{ c // comment }}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("{{ a }}b{{ c // comment }}", nil))
});
t.Run("should handle CR & LF in text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "t\ne\ns\nt"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("t\ne\rs\r\nt", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "t\ne\rs\r\nt"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("t\ne\rs\r\nt", nil))
});
t.Run("should handle CR & LF in interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", "t\ne\ns\nt", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{t\ne\rs\r\nt}}", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{t\ne\rs\r\nt}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("{{t\ne\rs\r\nt}}", nil))
});
t.Run("should parse entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a"},
    []any{TokenTypeEncodedEntity, "&", "&amp;"},
    []any{TokenTypeText, "b"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("a&amp;b", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a"},
    []any{TokenTypeEncodedEntity, "&amp;"},
    []any{TokenTypeText, "b"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("a&amp;b", nil))
});
t.Run("should parse text starting with \"&\"", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a && b &"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("a && b &", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("a", nil))
});
t.Run("should allow \"<\" in text nodes", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " a < b ? c : d ", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ a < b ? c : d }}", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<p"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "a"},
    []any{TokenTypeIncompleteTagOpen, "<b"},
    []any{TokenTypeTagClose, "</p>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<p>a<b</p>", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "< a>"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("< a>", nil))
});
t.Run("should break out of interpolation in text token on valid start tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " a "},
    []any{TokenTypeText, ""},
    []any{TokenTypeTagOpenStart, "", "b"},
    []any{TokenTypeAttrName, "", "&&"},
    []any{TokenTypeAttrName, "", "c"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, " d "},
    []any{TokenTypeBlockClose},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ a <b && c > d }}", nil))
});
t.Run("should break out of interpolation in text token on valid comment", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " a }"},
    []any{TokenTypeText, ""},
    []any{TokenTypeCommentStart},
    []any{TokenTypeRawText, ""},
    []any{TokenTypeCommentEnd},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ a }<!---->}", nil))
});
t.Run("should end interpolation on a valid closing tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "p"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " a "},
    []any{TokenTypeText, ""},
    []any{TokenTypeTagClose, "", "p"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<p>{{ a </p>", nil))
});
t.Run("should break out of interpolation in text token on valid CDATA", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " a }"},
    []any{TokenTypeText, ""},
    []any{TokenTypeCdataStart},
    []any{TokenTypeRawText, ""},
    []any{TokenTypeCdataEnd},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ a }<![CDATA[]]>}", nil))
});
t.Run("should ignore invalid start tag in interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "code"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", "'<={'", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeTagClose, "", "code"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<code>{{'<={'}}</code>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse start tags quotes in place of an attribute name as text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "", "t"},
    []any{TokenTypeText, "\">"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t \">", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "", "t"},
    []any{TokenTypeText, "'>"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t '>", nil))
});
t.Run("should parse start tags quotes in place of an attribute name (after a valid attribute)", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeText, "\">"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\"b\" \">", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteTagOpen, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeText, "'>"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a='b' '>", nil))
});
t.Run("should be able to escape {", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " \"{\" ", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ \"{\" }}", nil))
});
t.Run("should be able to escape {{", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " \"{{\" ", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ \"{{\" }}", nil))
});
t.Run("should capture everything up to the end of file in the interpolation expression part if there are mismatched quotes", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", " \"{{a}}' }}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{{ \"{{a}}' }}", nil))
});
t.Run("should treat expansion form as text when they are not parsed", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "span"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "{a, b, =4 {c}}"},
    []any{TokenTypeTagClose, "", "span"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<span>{a, b, =4 {c}}</span>", &TokenizeOptions{TokenizeExpansionForms: false, TokenizeBlocks: false}))
});
});
t.Run("raw text", func(t *testing.T) {
t.Run("should parse text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "script"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeRawText, "t\ne\ns\nt"},
    []any{TokenTypeTagClose, "", "script"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<script>t\ne\rs\r\nt</script>", nil))
});
t.Run("should not detect entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "script"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeRawText, "&amp;"},
    []any{TokenTypeTagClose, "", "script"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<script>&amp;</SCRIPT>", nil))
});
t.Run("should ignore other opening tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "script"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeRawText, "a<div>"},
    []any{TokenTypeTagClose, "", "script"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<script>a<div></script>", nil))
});
t.Run("should ignore other closing tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "script"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeRawText, "a</test>"},
    []any{TokenTypeTagClose, "", "script"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<script>a</test></script>", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<script"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeRawText, "a"},
    []any{TokenTypeTagClose, "</script>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<script>a</script>", nil))
});
});
t.Run("escapable raw text", func(t *testing.T) {
t.Run("should parse text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "t\ne\ns\nt"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>t\ne\rs\r\nt</title>", nil))
});
t.Run("should detect entities", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, ""},
    []any{TokenTypeEncodedEntity, "&", "&amp;"},
    []any{TokenTypeEscapableRawText, ""},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>&amp;</title>", nil))
});
t.Run("should ignore other opening tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "a<div>"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>a<div></title>", nil))
});
t.Run("should ignore other closing tags", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "a</test>"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>a</test></title>", nil))
});
t.Run("should store the locations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<title"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeEscapableRawText, "a"},
    []any{TokenTypeTagClose, "</title>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<title>a</title>", nil))
});
});
t.Run("parsable data", func(t *testing.T) {
t.Run("should parse an SVG <title> tag", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "svg", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "test"},
    []any{TokenTypeTagClose, "svg", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<svg:title>test</svg:title>", nil))
});
t.Run("should parse an SVG <title> tag with children", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "svg", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagOpenStart, "", "f"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "test"},
    []any{TokenTypeTagClose, "", "f"},
    []any{TokenTypeTagClose, "svg", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<svg:title><f>test</f></svg:title>", nil))
});
});
t.Run("expansion forms", func(t *testing.T) {
t.Run("should parse an expansion form", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=5"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "five"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "foo"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "bar"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 {four} =5 {five} foo {bar} }", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion form with text elements surrounding it", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "before"},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "after"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("before{one.two, three, =4 {four}}after", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion form as a tag single child", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagOpenStart, "", "span"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "a"},
    []any{TokenTypeRawText, "b"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "c"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeTagClose, "", "span"},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div><span>{a, b, =4 {c}}</span></div>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion form with whitespace surrounding it", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagOpenStart, "", "span"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, " "},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "a"},
    []any{TokenTypeRawText, "b"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "c"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, " "},
    []any{TokenTypeTagClose, "", "span"},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div><span> {a, b, =4 {c}} </span></div>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion forms with elements in it", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four "},
    []any{TokenTypeTagOpenStart, "", "b"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "a"},
    []any{TokenTypeTagClose, "", "b"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 {four <b>a</b>}}", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse an expansion forms containing an interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four "},
    []any{TokenTypeInterpolation, "{{", "a", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 {four {{a}}}}", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should parse nested expansion forms", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "xx"},
    []any{TokenTypeRawText, "yy"},
    []any{TokenTypeExpansionCaseValue, "=x"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "one"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, " "},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("{one.two, three, =4 { {xx, yy, =x {one}} }}", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("[line ending normalization", func(t *testing.T) {
t.Run("{escapedString: true}", func(t *testing.T) {
t.Run("should normalize line-endings in expansion forms if `i18nNormalizeLineEndingsInICUs` is true", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, []Token(nil), result.NonNormalizedIcuExpressions)
});
t.Run("should not normalize line-endings in ICU expressions when `i18nNormalizeLineEndingsInICUs` is not defined", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n    messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
});
t.Run("should not normalize line endings in nested expansion forms when `i18nNormalizeLineEndingsInICUs` is not defined", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n  messages.length, plural,\r\n  =0 { zero \r\n       {\r\n         p.gender, select,\r\n         male {m}\r\n       }\r\n     }\r\n}", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n  messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "zero \n       "},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n         p.gender"},
    []any{TokenTypeRawText, "select"},
    []any{TokenTypeExpansionCaseValue, "male"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "m"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n     "},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n  messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
assert.Equal(t, '\r\n         p.gender', result.NonNormalizedIcuExpressions![1].sourceSpan.toString())
});
});
t.Run("{escapedString: false}", func(t *testing.T) {
t.Run("should normalize line-endings in expansion forms if `i18nNormalizeLineEndingsInICUs` is true", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, []Token(nil), result.NonNormalizedIcuExpressions)
});
t.Run("should not normalize line-endings in ICU expressions when `i18nNormalizeLineEndingsInICUs` is not defined (escapeString: false)", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n    messages.length,\r\n    plural,\r\n    =0 {You have \r\nno\r\n messages}\r\n    =1 {One {{message}}}}\r\n", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n    messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "You have \nno\n messages"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=1"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "One "},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n    messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
});
t.Run("should not normalize line endings in nested expansion forms when `i18nNormalizeLineEndingsInICUs` is not defined", func(t *testing.T) {
result := tokenizeWithoutErrors("{\r\n  messages.length, plural,\r\n  =0 { zero \r\n       {\r\n         p.gender, select,\r\n         male {m}\r\n       }\r\n     }\r\n}", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n  messages.length"},
    []any{TokenTypeRawText, "plural"},
    []any{TokenTypeExpansionCaseValue, "=0"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "zero \n       "},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "\r\n         p.gender"},
    []any{TokenTypeRawText, "select"},
    []any{TokenTypeExpansionCaseValue, "male"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "m"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeText, "\n     "},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeEOF},
}, humanizeParts(result.Tokens))
assert.Equal(t, '\r\n  messages.length', result.NonNormalizedIcuExpressions![0].sourceSpan.toString())
assert.Equal(t, '\r\n         p.gender', result.NonNormalizedIcuExpressions![1].sourceSpan.toString())
});
});
});
});
t.Run("errors", func(t *testing.T) {
t.Run("should report unescaped \"{\" on error", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\" (Do you have an unescaped \"{\" in your template? Use \"{{ '{' }}\") to escape it.)", "0:21"},
}, tokenizeAndHumanizeErrors("<p>before { after</p>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should report unescaped \"{\" as an error, even after a prematurely terminated interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\" (Do you have an unescaped \"{\" in your template? Use \"{{ '{' }}\") to escape it.)", "0:56"},
}, tokenizeAndHumanizeErrors("<code>{{b}<!---->}</code><pre>import {a} from 'a';</pre>", &TokenizeOptions{TokenizeExpansionForms: true}))
});
t.Run("should include 2 lines of context in message", func(t *testing.T) {
src := "111\n222\n333\nE\n444\n555\n666\n"
file := new ParseSourceFile(src, 'file://')
location := new ParseLocation(file, 12, 123, 456)
span := new ParseSourceSpan(location, location)
error := new ParseError(span, '**ERROR**')
assert.Equal(t, `**ERROR** ("\n222\n333\n[ERROR ->]E\n444\n555\n"): file://@123:456`, error.toString())
});
});
t.Run("unicode characters", func(t *testing.T) {
t.Run("should support unicode characters", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<p"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "İ"},
    []any{TokenTypeTagClose, "</p>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans("<p>İ</p>", nil))
});
});
t.Run("(processing escaped strings)", func(t *testing.T) {
t.Run("should unescape standard escape sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "' ' '"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\' \\' \\'", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\" \" \""},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\\" \\\" \\\"", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "` ` `"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\` \\` \\`", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\\ \\ \\"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\\\ \\\\ \\\\", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\n \n \n"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\n \\n \\n", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\n"},
    []any{TokenTypeInterpolation, "{{", "\n", "}}"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\r{{\\r}}\\r", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u000b \u000b \u000b"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\v \\v \\v", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\t \t \t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\t \\t \\t", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\b \b \b"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\b \\b \\b", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\f \f \f"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\f \\f \\f", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "' \" ` \\ \n \n \u000b \t \b \f"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\' \\\" \\` \\\\ \\n \\r \\v \\t \\b \\f", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape null sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\0", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\09", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape octal sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u0001 \u0001 \u0001 \n  \u00019 4 999"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\001 \\01 \\1 \\12 \\223 \\19 \\2234 \\999", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape hex sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u0012 O Ü"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\x12 \\x4F \\xDC", &TokenizeOptions{EscapedString: true}))
});
t.Run("should report an error on an invalid hex sequence", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:2"},
}, tokenizeAndHumanizeErrors("\\xGG", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:6"},
}, tokenizeAndHumanizeErrors("abc \\x xyz", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:5"},
}, tokenizeAndHumanizeErrors("abc\\x", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape fixed length Unicode sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "ģ ꯍ"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\u0123 \\uABCD", &TokenizeOptions{EscapedString: true}))
});
t.Run("should error on an invalid fixed length Unicode sequence", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:2"},
}, tokenizeAndHumanizeErrors("\\uGGGG", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape variable length Unicode sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "\u0001 ઼ ሴ 𒎫"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\u{01} \\u{ABC} \\u{1234} \\u{123AB}", &TokenizeOptions{EscapedString: true}))
});
t.Run("should error on an invalid variable length Unicode sequence", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Invalid hexadecimal escape sequence", "0:3"},
}, tokenizeAndHumanizeErrors("\\u{GG}", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape line continuations", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "abcdef"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("abc\\\ndef", &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "xy"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("\\\nx\\\ny\\\n", &TokenizeOptions{EscapedString: true}))
});
t.Run("should remove backslash from \"non-escape\" sequences", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "a g ~"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("a g ~", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape sequences in plain text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "abc\ndef\nghi\tjkl`'\"mno"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("abc\ndef\\nghi\\tjkl\\`\\'\\\"mno", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape sequences in raw text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "script"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeRawText, "abc\ndef\nghi\tjkl`'\"mno"},
    []any{TokenTypeTagClose, "", "script"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<script>abc\ndef\\nghi\\tjkl\\`\\'\\\"mno</script>", &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape sequences in escapable raw text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "title"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEscapableRawText, "abc\ndef\nghi\tjkl`'\"mno"},
    []any{TokenTypeTagClose, "", "title"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<title>abc\ndef\\nghi\\tjkl\\`\\'\\\"mno</title>", &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escape sequences in tag definitions", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "c"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeAttrValueText, "d"},
    []any{TokenTypeAttrQuote, "'"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<t a=\\\"b\\\" \\n c=\\'d\\'>", &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escaped new line in tag definitions", func(t *testing.T) {
text := "<t\\n></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escaped characters in tag definitions", func(t *testing.T) {
text := "<t\u0013></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape characters in tag names", func(t *testing.T) {
text := "<t\\x64></t\\x64>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "td"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "td"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<t\\x64"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeTagClose, "</t\\x64>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should unescape characters in attributes", func(t *testing.T) {
text := "<t \\x64=\"\\x65\"></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "d"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "e"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should parse over escaped new line in attribute values", func(t *testing.T) {
text := "<t a=b\\n></t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrValueText, "b"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
});
t.Run("should tokenize the correct span when there are escape sequences", func(t *testing.T) {
text := "selector: \"app-root\",\ntemplate: \"line 1\\n\\\"line 2\\\"\\nline 3\",\ninputs: []"
range := {
        startPos: 33,
        startLine: 1,
        startCol: 10,
        endPos: 59,
      }
assert.Equal(t, [][]any{
    []any{TokenTypeText, "line 1\n\"line 2\"\nline 3"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{Range: nil, EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeText, "line 1\\n\\\"line 2\\\"\\nline 3"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans(text, &TokenizeOptions{Range: nil, EscapedString: true}))
});
t.Run("should account for escape sequences when computing source spans ", func(t *testing.T) {
text := "<t>line 1</t>\n<t>line 2</t>\\n<t>line 3\\\n</t>"
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "line 1"},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "line 2"},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeTagOpenStart, "", "t"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "line 3"},
    []any{TokenTypeTagClose, "", "t"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(text, &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "0:0"},
    []any{TokenTypeTagOpenEnd, "0:2"},
    []any{TokenTypeText, "0:3"},
    []any{TokenTypeTagClose, "0:9"},
    []any{TokenTypeText, "0:13"},
    []any{TokenTypeTagOpenStart, "1:0"},
    []any{TokenTypeTagOpenEnd, "1:2"},
    []any{TokenTypeText, "1:3"},
    []any{TokenTypeTagClose, "1:9"},
    []any{TokenTypeText, "1:13"},
    []any{TokenTypeTagOpenStart, "1:15"},
    []any{TokenTypeTagOpenEnd, "1:17"},
    []any{TokenTypeText, "1:18"},
    []any{TokenTypeTagClose, "2:0"},
    []any{TokenTypeEOF, "2:4"},
}, tokenizeAndHumanizeLineColumn(text, &TokenizeOptions{EscapedString: true}))
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "<t"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "line 1"},
    []any{TokenTypeTagClose, "</t>"},
    []any{TokenTypeText, "\n"},
    []any{TokenTypeTagOpenStart, "<t"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "line 2"},
    []any{TokenTypeTagClose, "</t>"},
    []any{TokenTypeText, "\\n"},
    []any{TokenTypeTagOpenStart, "<t"},
    []any{TokenTypeTagOpenEnd, ">"},
    []any{TokenTypeText, "line 3\\\n"},
    []any{TokenTypeTagClose, "</t>"},
    []any{TokenTypeEOF, ""},
}, tokenizeAndHumanizeSourceSpans(text, &TokenizeOptions{EscapedString: true}))
});
});
t.Run("blocks", func(t *testing.T) {
t.Run("should parse a block without parameters", func(t *testing.T) {
expected := [
        [TokenType.BLOCK_OPEN_START, 'if'],
        [TokenType.BLOCK_OPEN_END],
        [TokenType.TEXT, 'hello'],
        [TokenType.BLOCK_CLOSE],
        [TokenType.EOF],
      ]
assert.Equal(t, expected, tokenizeAndHumanizeParts("@if {hello}", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@if () {hello}", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@if(){hello}", nil))
});
t.Run("should parse @default never;", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "default never"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@default never;", nil))
});
t.Run("should parse @default never(expr);", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "default never"},
    []any{TokenTypeBlockParameter, "expr"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@default never(expr);", nil))
});
t.Run("should parse @default never ;", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "default never"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@default never ;", nil))
});
t.Run("should parse a block with parameters", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "for"},
    []any{TokenTypeBlockParameter, "item of items"},
    []any{TokenTypeBlockParameter, "track item.id"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@for (item of items; track item.id) {hello}", nil))
});
t.Run("should parse a block with a trailing semicolon after the parameters", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "for"},
    []any{TokenTypeBlockParameter, "item of items"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@for (item of items;) {hello}", nil))
});
t.Run("should parse a block with a space in its name", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "else if"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@else if {hello}", nil))
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "else if"},
    []any{TokenTypeBlockParameter, "foo !== 2"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@else if (foo !== 2) {hello}", nil))
});
t.Run("should parse a block with an arbitrary amount of spaces around the parentheses", func(t *testing.T) {
expected := [
        [TokenType.BLOCK_OPEN_START, 'for'],
        [TokenType.BLOCK_PARAMETER, 'a'],
        [TokenType.BLOCK_PARAMETER, 'b'],
        [TokenType.BLOCK_PARAMETER, 'c'],
        [TokenType.BLOCK_OPEN_END],
        [TokenType.TEXT, 'hello'],
        [TokenType.BLOCK_CLOSE],
        [TokenType.EOF],
      ]
assert.Equal(t, expected, tokenizeAndHumanizeParts("@for(a; b; c){hello}", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@for      (a; b; c)      {hello}", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@for(a; b; c)      {hello}", nil))
assert.Equal(t, expected, tokenizeAndHumanizeParts("@for      (a; b; c){hello}", nil))
});
t.Run("should parse a block with multiple trailing semicolons", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "for"},
    []any{TokenTypeBlockParameter, "item of items"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@for (item of items;;;;;) {hello}", nil))
});
t.Run("should parse a block with trailing whitespace", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "defer"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@defer                        {hello}", nil))
});
t.Run("should parse a block with no trailing semicolon", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "for"},
    []any{TokenTypeBlockParameter, "item of items"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@for (item of items){hello}", nil))
});
t.Run("should handle semicolons, braces and parentheses used in a block parameter", func(t *testing.T) {
input := "@for (a === \";\"; b === ')'; c === \"(\"; d === '}'; e === \"{\") {hello}"
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "for"},
    []any{TokenTypeBlockParameter, "a === \";\""},
    []any{TokenTypeBlockParameter, "b === ')'"},
    []any{TokenTypeBlockParameter, "c === \"(\""},
    []any{TokenTypeBlockParameter, "d === '}'"},
    []any{TokenTypeBlockParameter, "e === \"{\""},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(input, nil))
});
t.Run("should handle object literals and function calls in block parameters", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "defer"},
    []any{TokenTypeBlockParameter, "on a({a: 1, b: 2}, false, {c: 3})"},
    []any{TokenTypeBlockParameter, "when b({d: 4})"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@defer (on a({a: 1, b: 2}, false, {c: 3}); when b({d: 4})) {hello}", nil))
});
t.Run("should parse block with unclosed parameters", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteBlockOpen, "if"},
    []any{TokenTypeBlockParameter, "a === b {hello}"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if (a === b {hello}", nil))
});
t.Run("should parse block with stray parentheses in the parameter position", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteBlockOpen, "if a"},
    []any{TokenTypeText, "=== b) {hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if a === b) {hello}", nil))
});
t.Run("should report invalid quotes in a parameter", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:21"},
}, tokenizeAndHumanizeErrors("@if (a === \") {hello}", nil))
assert.Equal(t, [][]any{
    []any{"Unexpected character \"EOF\"", "0:24"},
}, tokenizeAndHumanizeErrors("@if (a === \"hi') {hello}", nil))
});
t.Run("should report unclosed object literal inside a parameter", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeIncompleteBlockOpen, "if"},
    []any{TokenTypeBlockParameter, "{invalid: true"},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if ({invalid: true) hello}", nil))
});
t.Run("should handle a semicolon used in a nested string inside a block parameter", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockParameter, "condition === \"';'\""},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if (condition === \"';'\") {hello}", nil))
});
t.Run("should handle a semicolon next to an escaped quote used in a block parameter", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockParameter, "condition === \"\\\";\""},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if (condition === \"\\\";\") {hello}", nil))
});
t.Run("should parse mixed text and html content in a block", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockParameter, "a === 1"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "foo "},
    []any{TokenTypeTagOpenStart, "", "b"},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeText, "bar"},
    []any{TokenTypeTagClose, "", "b"},
    []any{TokenTypeText, " baz"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if (a === 1) {foo <b>bar</b> baz}", nil))
});
t.Run("should parse HTML tags with attributes containing curly braces inside blocks", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockParameter, "a === 1"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "}"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrName, "", "b"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "{"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if (a === 1) {<div a=\"}\" b=\"{\"></div>}", nil))
});
t.Run("should parse HTML tags with attribute containing block syntax", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeTagOpenStart, "", "div"},
    []any{TokenTypeAttrName, "", "a"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeAttrValueText, "@if (foo) {}"},
    []any{TokenTypeAttrQuote, "\""},
    []any{TokenTypeTagOpenEnd},
    []any{TokenTypeTagClose, "", "div"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("<div a=\"@if (foo) {}\"></div>", nil))
});
t.Run("should parse nested blocks", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockParameter, "a"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello a"},
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello unnamed"},
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockParameter, "b"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello b"},
    []any{TokenTypeBlockOpenStart, "if"},
    []any{TokenTypeBlockParameter, "c"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, "hello c"},
    []any{TokenTypeBlockClose},
    []any{TokenTypeBlockClose},
    []any{TokenTypeBlockClose},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@if (a) {hello a@if {hello unnamed@if (b) {hello b@if (c) {hello c}}}}", nil))
});
t.Run("should parse a block containing an expansion", func(t *testing.T) {
result := tokenizeAndHumanizeParts("@defer {{one.two, three, =4 {four} =5 {five} foo {bar} }}", nil)
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "defer"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeExpansionFormStart},
    []any{TokenTypeRawText, "one.two"},
    []any{TokenTypeRawText, "three"},
    []any{TokenTypeExpansionCaseValue, "=4"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "four"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "=5"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "five"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionCaseValue, "foo"},
    []any{TokenTypeExpansionCaseExpStart},
    []any{TokenTypeText, "bar"},
    []any{TokenTypeExpansionCaseExpEnd},
    []any{TokenTypeExpansionFormEnd},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, result)
});
t.Run("should parse a block containing an interpolation", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeBlockOpenStart, "defer"},
    []any{TokenTypeBlockOpenEnd},
    []any{TokenTypeText, ""},
    []any{TokenTypeInterpolation, "{{", "message", "}}"},
    []any{TokenTypeText, ""},
    []any{TokenTypeBlockClose},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@defer {{{message}}}", nil))
});
t.Run("should parse an incomplete block start without parameters with surrounding text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "My email frodo"},
    []any{TokenTypeIncompleteBlockOpen, "for"},
    []any{TokenTypeText, ".com"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("My email frodo@for.com", nil))
});
t.Run("should parse an incomplete block start at the end of the input", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "My favorite console is "},
    []any{TokenTypeIncompleteBlockOpen, "switch"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("My favorite console is @switch", nil))
});
t.Run("should parse an incomplete block start with parentheses but without params", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "Use the "},
    []any{TokenTypeIncompleteBlockOpen, "for"},
    []any{TokenTypeText, "block"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("Use the @for() block", nil))
});
t.Run("should parse an incomplete block start with parentheses and params", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "This is the "},
    []any{TokenTypeIncompleteBlockOpen, "if"},
    []any{TokenTypeBlockParameter, "{alias: \"foo\"}"},
    []any{TokenTypeText, "expression"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("This is the @if({alias: \"foo\"}) expression", nil))
});
t.Run("should parse @ as text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "@"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@", nil))
});
t.Run("should parse space followed by @ as text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, " @"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts(" @", nil))
});
t.Run("should parse @ followed by space as text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "@ "},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@ ", nil))
});
t.Run("should parse @ followed by newline and text as text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "@\nfoo"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@\nfoo", nil))
});
t.Run("should parse @ in the middle of text as text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "foo bar @ baz clink"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("foo bar @ baz clink", nil))
});
t.Run("should parse incomplete block with space, then name as text", func(t *testing.T) {
assert.Equal(t, [][]any{
    []any{TokenTypeText, "@ if"},
    []any{TokenTypeEOF},
}, tokenizeAndHumanizeParts("@ if", nil))
});
});
});
