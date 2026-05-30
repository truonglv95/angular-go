package i18n

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
)

type I18NHtmlParser struct {
	htmlParser        *ml_parser.HtmlParser
	translationBundle *TranslationBundle
}

func NewI18NHtmlParser(
	htmlParser *ml_parser.HtmlParser,
	translations *string,
	translationsFormat *string,
	missingTranslation core.MissingTranslationStrategy,
	console any,
) *I18NHtmlParser {
	var translationBundle *TranslationBundle

	if translations != nil {
		serializer := createSerializer(translationsFormat)
		translationBundle = TranslationBundleLoad(*translations, "i18n", serializer, missingTranslation, console)
	} else {
		translationBundle = NewTranslationBundle(
			NewEagerTranslationStore(map[string][]Node{}),
			nil,
			Digest,
			nil,
			missingTranslation,
			console,
		)
	}

	return &I18NHtmlParser{
		htmlParser:        htmlParser,
		translationBundle: translationBundle,
	}
}

func (p *I18NHtmlParser) Parse(source string, url string, options *ml_parser.TokenizeOptions) *ml_parser.ParseTreeResult {
	parseResult := p.htmlParser.Parse(source, url, options)

	if len(parseResult.Errors) > 0 {
		return &ml_parser.ParseTreeResult{RootNodes: parseResult.RootNodes, Errors: parseResult.Errors}
	}

	return MergeTranslations(parseResult.RootNodes, p.translationBundle, nil, nil)
}

func createSerializer(format *string) Serializer {
	f := "xlf"
	if format != nil {
		f = strings.ToLower(*format)
	}

	switch f {
	case "xmb":
		return NewXmb()
	case "xtb":
		return NewXtb()
	case "xliff2", "xlf2":
		return NewXliff2()
	case "xliff", "xlf":
		fallthrough
	default:
		return NewXliff()
	}
}
