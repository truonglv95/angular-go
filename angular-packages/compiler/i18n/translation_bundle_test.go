package i18n_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/stretchr/testify/assert"
)

type htmlSerializerVisitor struct{}

func (v *htmlSerializerVisitor) VisitElement(element *ml_parser.Element, context any) any {
	var attrs []string
	for _, attr := range element.Attrs {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", attr.Name, attr.Value))
	}
	attrsStr := ""
	if len(attrs) > 0 {
		attrsStr = " " + strings.Join(attrs, " ")
	}
	if element.IsSelfClosing {
		return fmt.Sprintf("<%s%s/>", element.Name, attrsStr)
	}
	var children []string
	for _, child := range element.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("<%s%s>%s</%s>", element.Name, attrsStr, strings.Join(children, ""), element.Name)
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
	var cases []string
	for _, c := range expansion.Cases {
		cases = append(cases, c.Visit(v, nil).(string))
	}
	return fmt.Sprintf("{%s, %s,%s}", expansion.SwitchValue, expansion.Type, strings.Join(cases, ""))
}

func (v *htmlSerializerVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	var children []string
	for _, child := range expansionCase.Expression {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf(" %s {%s}", expansionCase.Value, strings.Join(children, ""))
}

func (v *htmlSerializerVisitor) VisitBlock(block *ml_parser.Block, context any) any {
	var params []string
	for _, param := range block.Parameters {
		params = append(params, param.Expression)
	}
	paramsStr := ""
	if len(params) > 0 {
		paramsStr = fmt.Sprintf(" (%s) ", strings.Join(params, "; "))
	}
	var children []string
	for _, child := range block.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("@%s%s{%s}", block.Name, paramsStr, strings.Join(children, ""))
}

func (v *htmlSerializerVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return parameter.Expression
}

func serializeNodes(nodes []ml_parser.Node) []string {
	v := &htmlSerializerVisitor{}
	var result []string
	for _, node := range nodes {
		result = append(result, node.Visit(v, nil).(string))
	}
	return result
}

func extractMessagesHelper(htmlStr string) []*i18n.Message {
	htmlParser := ml_parser.NewHtmlParser()
	tokenizeBlocks := true
	parseResult := htmlParser.Parse(htmlStr, "test", &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: true,
		TokenizeBlocks:         &tokenizeBlocks,
	})
	if len(parseResult.Errors) > 0 {
		var errMsgs []string
		for _, err := range parseResult.Errors {
			errMsgs = append(errMsgs, err.Error())
		}
		panic("unexpected HTML errors: " + strings.Join(errMsgs, "\n"))
	}
	res := i18n.ExtractMessages(parseResult.RootNodes, []string{}, make(map[string][]string), true)
	if len(res.Errors) > 0 {
		var errMsgs []string
		for _, err := range res.Errors {
			errMsgs = append(errMsgs, err.Error())
		}
		panic("unexpected i18n extraction errors: " + strings.Join(errMsgs, "\n"))
	}
	return res.Messages
}

type mockConsole struct {
	warnings []string
}

func (c *mockConsole) Warn(msg string) {
	c.warnings = append(c.warnings, msg)
}

func TestTranslationBundle(t *testing.T) {
	file := parse_util.NewParseSourceFile("content", "url")
	startLocation := parse_util.NewParseLocation(file, 0, 0, 0)
	endLocation := parse_util.NewParseLocation(file, 7, 0, 7)
	span := parse_util.NewParseSourceSpan(startLocation, endLocation, nil, nil)
	srcNode := i18n.NewText("src", span)

	it("should translate a plain text", func() {
		msgMap := map[string][]i18n.Node{
			"foo": {i18n.NewText("bar", nil)},
		}
		tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(msgMap), nil, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyWarning, nil)
		msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")
		assert.Equal(t, []string{"bar"}, serializeNodes(tb.Get(msg)))
	})

	it("should translate html-like plain text", func() {
		msgMap := map[string][]i18n.Node{
			"foo": {i18n.NewText("<p>bar</p>", nil)},
		}
		tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(msgMap), nil, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyWarning, nil)
		msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")
		nodes := tb.Get(msg)
		assert.Equal(t, 1, len(nodes))
		textNode, ok := nodes[0].(*ml_parser.Text)
		assert.True(t, ok)
		assert.Equal(t, "<p>bar</p>", textNode.Value)
	})

	it("should translate a message with placeholder", func() {
		msgMap := map[string][]i18n.Node{
			"foo": {i18n.NewText("bar", nil), i18n.NewPlaceholder("", "ph1", nil)},
		}
		phMap := map[string]*i18n.MessagePlaceholder{
			"ph1": createPlaceholder("*phContent*"),
		}
		tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(msgMap), nil, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyWarning, nil)
		msg := i18n.NewMessage([]i18n.Node{srcNode}, phMap, make(map[string]*i18n.Message), "m", "d", "i")
		assert.Equal(t, []string{"bar*phContent*"}, serializeNodes(tb.Get(msg)))
	})

	it("should translate a message with placeholder referencing messages", func() {
		msgMap := map[string][]i18n.Node{
			"foo": {
				i18n.NewText("--", nil),
				i18n.NewPlaceholder("", "ph1", nil),
				i18n.NewText("++", nil),
			},
			"ref": {i18n.NewText("*refMsg*", nil)},
		}
		refMsg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")
		msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), map[string]*i18n.Message{"ph1": refMsg}, "m", "d", "i")
		count := 0
		digest := func(_ *i18n.Message) string {
			if count > 0 {
				return "ref"
			}
			count++
			return "foo"
		}
		tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(msgMap), nil, digest, nil, core.MissingTranslationStrategyWarning, nil)
		assert.Equal(t, []string{"--*refMsg*++"}, serializeNodes(tb.Get(msg)))
	})

	it("should use the original message or throw when a translation is not found", func() {
		src := "<some-tag>some text{{ some_expression }}</some-tag>{count, plural, =0 {no} few {a <b>few</b>}}"
		messages := extractMessagesHelper("<div i18n>" + src + "</div>")

		digest := func(_ *i18n.Message) string { return "no matching id" }

		// Empty message map -> use source messages in Ignore mode
		tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(make(map[string][]i18n.Node)), nil, digest, nil, core.MissingTranslationStrategyIgnore, nil)
		assert.Equal(t, src, strings.Join(serializeNodes(tb.Get(messages[0])), ""))

		// Empty message map -> use source messages in Warning mode
		tb = i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(make(map[string][]i18n.Node)), nil, digest, nil, core.MissingTranslationStrategyWarning, nil)
		assert.Equal(t, src, strings.Join(serializeNodes(tb.Get(messages[0])), ""))

		// Empty message map -> throw in Error mode
		tb = i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(make(map[string][]i18n.Node)), nil, digest, nil, core.MissingTranslationStrategyError, nil)
		assert.Panics(t, func() {
			tb.Get(messages[0])
		})
	})

	t.Run("errors reporting", func(t *testing.T) {
		it("should report unknown placeholders", func() {
			msgMap := map[string][]i18n.Node{
				"foo": {i18n.NewText("bar", nil), i18n.NewPlaceholder("", "ph1", span)},
			}
			tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(msgMap), nil, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyWarning, nil)
			msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")
			assertPanicContains(t, "Unknown placeholder \"ph1\"", func() {
				tb.Get(msg)
			})
		})

		it("should report missing translation", func() {
			tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(make(map[string][]i18n.Node)), nil, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyError, nil)
			msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")
			assertPanicContains(t, "Missing translation for message \"foo\"", func() {
				tb.Get(msg)
			})
		})

		it("should report missing translation with MissingTranslationStrategy.Warning", func() {
			console := &mockConsole{}
			locale := "en"
			tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(make(map[string][]i18n.Node)), &locale, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyWarning, console)
			msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")

			assert.NotPanics(t, func() {
				tb.Get(msg)
			})
			assert.Equal(t, 1, len(console.warnings))
			assert.Contains(t, console.warnings[0], "Missing translation for message \"foo\" for locale \"en\"")
		})

		it("should not report missing translation with MissingTranslationStrategy.Ignore", func() {
			tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(make(map[string][]i18n.Node)), nil, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyIgnore, nil)
			msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")
			assert.NotPanics(t, func() {
				tb.Get(msg)
			})
		})

		it("should report missing referenced message", func() {
			msgMap := map[string][]i18n.Node{
				"foo": {i18n.NewPlaceholder("", "ph1", span)},
			}
			refMsg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), make(map[string]*i18n.Message), "m", "d", "i")
			msg := i18n.NewMessage([]i18n.Node{srcNode}, make(map[string]*i18n.MessagePlaceholder), map[string]*i18n.Message{"ph1": refMsg}, "m", "d", "i")
			count := 0
			digest := func(_ *i18n.Message) string {
				if count > 0 {
					return "ref"
				}
				count++
				return "foo"
			}
			tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(msgMap), nil, digest, nil, core.MissingTranslationStrategyError, nil)
			assertPanicContains(t, "Missing translation for message \"ref\"", func() {
				tb.Get(msg)
			})
		})

		it("should report invalid translated html", func() {
			msgMap := map[string][]i18n.Node{
				"foo": {i18n.NewText("text", nil), i18n.NewPlaceholder("", "ph1", nil)},
			}
			phMap := map[string]*i18n.MessagePlaceholder{
				"ph1": createPlaceholder("</b>"),
			}
			tb := i18n.NewTranslationBundle(i18n.NewEagerTranslationStore(msgMap), nil, func(_ *i18n.Message) string { return "foo" }, nil, core.MissingTranslationStrategyWarning, nil)
			msg := i18n.NewMessage([]i18n.Node{srcNode}, phMap, make(map[string]*i18n.Message), "m", "d", "i")
			assertPanicContains(t, "Unexpected closing tag \"b\"", func() {
				tb.Get(msg)
			})
		})
	})
}

func createPlaceholder(text string) *i18n.MessagePlaceholder {
	file := parse_util.NewParseSourceFile(text, "file://test")
	start := parse_util.NewParseLocation(file, 0, 0, 0)
	end := parse_util.NewParseLocation(file, len(text), 0, len(text))
	return &i18n.MessagePlaceholder{
		Text:       text,
		SourceSpan: parse_util.NewParseSourceSpan(start, end, nil, nil),
	}
}

func assertPanicContains(t *testing.T, expected string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, but did not panic", expected)
		}
		rStr := fmt.Sprint(r)
		assert.Contains(t, rStr, expected)
	}()
	fn()
}

func it(name string, fn func()) {
	// Simple test run helper for flat it() syntax in specs
	fn()
}
