package ml_parser

import "github.com/microsoft/typescript-go/angular-packages/compiler/tags"

type HtmlParser struct {
	*Parser
}

func NewHtmlParser() *HtmlParser {
	return &HtmlParser{
		Parser: NewParser(func(tagName string) tags.TagDefinition {
			return GetHtmlTagDefinition(tagName)
		}),
	}
}

func (p *HtmlParser) Parse(source string, url string, options *TokenizeOptions) *ParseTreeResult {
	return p.Parser.Parse(source, url, options)
}

func ParseHTML(html string, url string, options ...any) ParseTreeResult {
	var tokenizeOptions *TokenizeOptions
	if len(options) > 0 {
		if parsedOptions, ok := options[0].(*TokenizeOptions); ok {
			tokenizeOptions = parsedOptions
		}
	}

	result := NewHtmlParser().Parse(html, url, tokenizeOptions)
	if result == nil {
		return ParseTreeResult{}
	}
	return *result
}

func IsNgTemplate(tagName string) bool {
	return tags.IsNgTemplate(tagName)
}
