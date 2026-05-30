package i18n_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	messageFactory := i18n.CreateI18nMessageFactory(false /* retainEmptyTokens */, true /* preserveExpressionWhitespace */)

	t.Run("messageText()", func(t *testing.T) {
		t.Run("should serialize simple text", func(t *testing.T) {
			message := messageFactory(parseHtml("abc\ndef"), nil, nil, nil, nil)
			assert.Equal(t, "abc\ndef", message.MessageString)
		})

		t.Run("should serialize text with interpolations", func(t *testing.T) {
			message := messageFactory(parseHtml("abc {{ 123 }}{{ 456 }} def"), nil, nil, nil, nil)
			assert.Equal(t, "abc {$INTERPOLATION}{$INTERPOLATION_1} def", message.MessageString)
		})

		t.Run("should serialize HTML elements", func(t *testing.T) {
			message := messageFactory(
				parseHtml("abc <span>foo</span><span>bar</span> def"),
				nil, nil, nil, nil,
			)
			assert.Equal(
				t,
				"abc {$START_TAG_SPAN}foo{$CLOSE_TAG_SPAN}{$START_TAG_SPAN}bar{$CLOSE_TAG_SPAN} def",
				message.MessageString,
			)
		})

		t.Run("should serialize ICU placeholders", func(t *testing.T) {
			message := messageFactory(
				parseHtml("abc {value, select, case1 {value1} case2 {value2} case3 {value3}} def"),
				nil, nil, nil, nil,
			)
			assert.Equal(t, "abc {$ICU} def", message.MessageString)
		})

		t.Run("should serialize ICU expressions", func(t *testing.T) {
			message := messageFactory(
				parseHtml("{value, select, case1 {value1} case2 {value2} case3 {value3}}"),
				nil, nil, nil, nil,
			)
			assert.Equal(
				t,
				"{VAR_SELECT, select, case1 {value1} case2 {value2} case3 {value3}}",
				message.MessageString,
			)
		})

		t.Run("should serialize nested ICU expressions", func(t *testing.T) {
			message := messageFactory(
				parseHtml(`{gender, select,
            male {male of age: {age, select, 10 {ten} 20 {twenty} 30 {thirty} other {other}}}
            female {female}
            other {other}
          }`),
				nil, nil, nil, nil,
			)
			assert.Equal(
				t,
				"{VAR_SELECT_1, select, male {male of age: {VAR_SELECT, select, 10 {ten} 20 {twenty} 30 {thirty} other {other}}} female {female} other {other}}",
				message.MessageString,
			)
		})

		t.Run("should serialize blocks", func(t *testing.T) {
			message := messageFactory(
				parseHtml("abc @if (foo) {foo} @else if (bar) {bar} @else {baz} def"),
				nil, nil, nil, nil,
			)

			assert.Equal(
				t,
				"abc {$START_BLOCK_IF}foo{$CLOSE_BLOCK_IF} {$START_BLOCK_ELSE_IF}bar{$CLOSE_BLOCK_ELSE_IF} {$START_BLOCK_ELSE}baz{$CLOSE_BLOCK_ELSE} def",
				message.MessageString,
			)
		})
	})
}

func parseHtml(htmlStr string) []ml_parser.Node {
	htmlParser := ml_parser.NewHtmlParser()
	tokenizeBlocks := true
	parseResult := htmlParser.Parse(htmlStr, "i18n_ast spec", &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: true,
		TokenizeBlocks:         &tokenizeBlocks,
	})
	if len(parseResult.Errors) > 0 {
		var errMsgs []string
		for _, err := range parseResult.Errors {
			errMsgs = append(errMsgs, err.Error())
		}
		panic(fmt.Sprintf("unexpected parse errors: %s", strings.Join(errMsgs, "\n")))
	}
	return parseResult.RootNodes
}
