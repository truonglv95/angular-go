package render3

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/schema"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

var LEADING_TRIVIA_CHARS = []string{" ", "\n", "\r", "\t"}

type ParseTemplateOptions struct {
	PreserveWhitespaces                *bool
	PreserveLineEndings                *bool
	PreserveSignificantWhitespace      *bool
	Range                              *ml_parser.LexerRange
	EscapedString                      *bool
	LeadingTriviaChars                 []string
	EnableI18nLegacyMessageIdFormat    *bool
	I18nNormalizeLineEndingsInICUs     *bool
	AlwaysAttemptHtmlToR3AstConversion *bool
	CollectCommentNodes                *bool
	EnableBlockSyntax                  *bool
	EnableLetSyntax                    *bool
	EnableSelectorless                 *bool
}

type ParsedTemplate struct {
	PreserveWhitespaces *bool
	Errors              []parse_util.ParseError
	Nodes               []Node
	StyleUrls           []string
	Styles              []string
	NgContentSelectors  []string
	CommentNodes        []*Comment
}



type schemaRegistryAdapter struct {
	registry schema.ElementSchemaRegistry
}

func (s *schemaRegistryAdapter) HasElement(name string, schemaCtx []string) bool {
	return s.registry.HasElement(name, nil)
}

func (s *schemaRegistryAdapter) HasProperty(tagName string, propName string, schemaMetas []core.SchemaMetadata) bool {
	return s.registry.HasProperty(tagName, propName, schemaMetas)
}

func (s *schemaRegistryAdapter) SecurityContext(name string, propName string, isAttribute bool) core.SecurityContext {
	return s.registry.SecurityContext(name, propName, isAttribute)
}

func (s *schemaRegistryAdapter) AllKnownElementNames() []string {
	return s.registry.AllKnownElementNames()
}

func (s *schemaRegistryAdapter) GetMappedPropName(name string) string {
	return s.registry.GetMappedPropName(name)
}

func (s *schemaRegistryAdapter) ValidateProperty(name string) struct {
	Error bool
	Msg   *string
} {
	res := s.registry.ValidateProperty(name)
	return struct {
		Error bool
		Msg   *string
	}{
		Error: res.Error,
		Msg:   res.Msg,
	}
}

func (s *schemaRegistryAdapter) ValidateAttribute(name string) struct {
	Error bool
	Msg   *string
} {
	res := s.registry.ValidateAttribute(name)
	return struct {
		Error bool
		Msg   *string
	}{
		Error: res.Error,
		Msg:   res.Msg,
	}
}

var ElementRegistry = &schemaRegistryAdapter{registry: schema.NewDomElementSchemaRegistry()}

func ParseTemplate(template string, templateUrl string, options *ParseTemplateOptions) ParsedTemplate {
	preserveWhitespaces := false
	collectCommentNodes := false
	selectorlessEnabled := false
	enableBlockSyntax := true
	enableLetSyntax := true
	var leadingTrivia []string

	if options != nil {
		if options.PreserveWhitespaces != nil {
			preserveWhitespaces = *options.PreserveWhitespaces
		}
		if options.CollectCommentNodes != nil {
			collectCommentNodes = *options.CollectCommentNodes
		}
		if options.EnableSelectorless != nil {
			selectorlessEnabled = *options.EnableSelectorless
		}
		if options.EnableBlockSyntax != nil {
			enableBlockSyntax = *options.EnableBlockSyntax
		}
		if options.EnableLetSyntax != nil {
			enableLetSyntax = *options.EnableLetSyntax
		}
		leadingTrivia = options.LeadingTriviaChars
	}

	if len(leadingTrivia) == 0 {
		leadingTrivia = LEADING_TRIVIA_CHARS
	}

	htmlParser := ml_parser.NewHtmlParser()
	i18nParser := i18n.NewI18NHtmlParser(htmlParser, nil, nil, core.MissingTranslationStrategyWarning, nil)
	
	tokenizeOptions := &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: true,
		LeadingTriviaChars:     leadingTrivia,
		SelectorlessEnabled:    selectorlessEnabled,
		TokenizeBlocks:         &enableBlockSyntax,
		TokenizeLet:            &enableLetSyntax,
	}

	url := templateUrl
	if url == "" {
		url = "path:://to/template"
	}
	parseResult := i18nParser.Parse(template, url, tokenizeOptions)

	htmlNodes := parseResult.RootNodes

	if !preserveWhitespaces {
		htmlNodes = ml_parser.VisitAllWithSiblingsForNodes(ml_parser.NewWhitespaceVisitor(true), htmlNodes)
	}

	lexer := expression_parser.Lexer{}
	exprParser := expression_parser.NewParser(lexer, false)
	bindingParser := template_parser.NewBindingParser(exprParser, ElementRegistry, []template_parser.ParseError{})

	r3Result := HtmlAstToRender3Ast(htmlNodes, bindingParser, Render3ParseOptions{CollectCommentNodes: collectCommentNodes})

	var errors []parse_util.ParseError
	for _, err := range parseResult.Errors {
		if pe, ok := err.(*parse_util.ParseError); ok {
			errors = append(errors, *pe)
		} else {
			errors = append(errors, parse_util.ParseError{
				Msg: err.Error(),
			})
		}
	}

	var baseSpan *parse_util.ParseSourceSpan
	if len(htmlNodes) > 0 {
		baseSpan = htmlNodes[0].GetSourceSpan()
	}

	for _, e := range bindingParser.Errors {
		spanVal := toParseSourceSpan(e.Span, baseSpan)
		errors = append(errors, parse_util.ParseError{
			Span: &spanVal,
			Msg:  e.Message,
		})
	}

	return ParsedTemplate{
		PreserveWhitespaces: &preserveWhitespaces,
		Errors:              errors,
		Nodes:               r3Result.Nodes,
		StyleUrls:           r3Result.StyleUrls,
		Styles:              r3Result.Styles,
		NgContentSelectors:  r3Result.NgContentSelectors,
		CommentNodes:        r3Result.CommentNodes,
	}
}

func MakeBindingParser(selectorlessEnabled bool) *template_parser.BindingParser {
	lexer := expression_parser.Lexer{}
	parser := expression_parser.NewParser(lexer, false)
	return template_parser.NewBindingParser(parser, ElementRegistry, nil)
}
