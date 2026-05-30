package ml_parser

import "github.com/microsoft/typescript-go/angular-packages/compiler/tags"

type XmlParser struct {
	*Parser
}

func NewXmlParser() *XmlParser {
	return &XmlParser{
		Parser: NewParser(func(tagName string) tags.TagDefinition {
			return GetXmlTagDefinition(tagName)
		}),
	}
}

func (p *XmlParser) Parse(source string, url string, options *TokenizeOptions) *ParseTreeResult {
	falseVal := false
	opts := TokenizeOptions{
		TokenizeBlocks:      &falseVal,
		TokenizeLet:         &falseVal,
		SelectorlessEnabled: false,
	}

	if options != nil {
		opts.TokenizeExpansionForms = options.TokenizeExpansionForms
	}

	return p.Parser.Parse(source, url, &opts)
}
