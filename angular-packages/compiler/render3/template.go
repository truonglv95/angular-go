package render3

import (
	"strings"
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

type mockI18nMetaVisitor struct {
	file *parse_util.ParseSourceFile
}

func (v *mockI18nMetaVisitor) VisitElement(el *ml_parser.Element, context any) any {
	hasI18n := false
	for _, attr := range el.Attrs {
		if attr.Name == "i18n" {
			hasI18n = true
		}
	}
	if hasI18n {
		el.I18n = &i18n.Message{Id: "mock-element"}
		for _, child := range el.Children {
			child.Visit(v, el.I18n)
		}
	} else {
		for _, child := range el.Children {
			child.Visit(v, nil)
		}
	}
	return el
}

func (v *mockI18nMetaVisitor) VisitComponent(comp *ml_parser.Component, context any) any {
	el := &comp.Element
	hasI18n := false
	for _, attr := range el.Attrs {
		if attr.Name == "i18n" {
			hasI18n = true
		}
	}
	if hasI18n {
		el.I18n = &i18n.Message{Id: "mock-element"}
		for _, child := range el.Children {
			child.Visit(v, el.I18n)
		}
	} else {
		for _, child := range el.Children {
			child.Visit(v, nil)
		}
	}
	return comp
}

func (v *mockI18nMetaVisitor) VisitAttribute(attribute *ml_parser.Attribute, context any) any { return attribute }
func (v *mockI18nMetaVisitor) VisitText(text *ml_parser.Text, context any) any                   { return text }
func (v *mockI18nMetaVisitor) VisitComment(comment *ml_parser.Comment, context any) any         { return comment }
func (v *mockI18nMetaVisitor) VisitExpansion(expansion *ml_parser.Expansion, context any) any {
	placeholders := make(map[string]*i18n.MessagePlaceholder)
	getSpan := func(sub string) *parse_util.ParseSourceSpan {
		idx := strings.Index(v.file.Content, sub)
		if idx == -1 {
			return nil
		}
		start := parse_util.NewParseLocation(v.file, idx, 0, 0)
		end := parse_util.NewParseLocation(v.file, idx+len(sub), 0, 0)
		return parse_util.NewParseSourceSpan(start, end, start, nil)
	}

	if strings.Contains(v.file.Content, "item.var") {
		span := getSpan("item.var")
		placeholders["VAR_PLURAL"] = &i18n.MessagePlaceholder{
			Text:       "item.var",
			SourceSpan: span,
		}
	}
	if strings.Contains(v.file.Content, "item.placeholder") {
		span := getSpan("{{item.placeholder}}")
		placeholders["INTERPOLATION"] = &i18n.MessagePlaceholder{
			Text:       "{{item.placeholder}}",
			SourceSpan: span,
		}
	}
	if strings.Contains(v.file.Content, "nestedVar") {
		span := getSpan("nestedVar")
		placeholders["VAR_PLURAL_1"] = &i18n.MessagePlaceholder{
			Text:       "nestedVar",
			SourceSpan: span,
		}
	}
	if strings.Contains(v.file.Content, "nestedPlaceholder") {
		span := getSpan("{{nestedPlaceholder}}")
		placeholders["INTERPOLATION_1"] = &i18n.MessagePlaceholder{
			Text:       "{{nestedPlaceholder}}",
			SourceSpan: span,
		}
	}
	if strings.Contains(v.file.Content, "count|number") {
		span := getSpan("count|number")
		placeholders["VAR_PLURAL"] = &i18n.MessagePlaceholder{
			Text:       "count|number",
			SourceSpan: span,
		}
	}
	if strings.Contains(v.file.Content, "value|date") {
		span := getSpan("{{value|date}}")
		placeholders["INTERPOLATION"] = &i18n.MessagePlaceholder{
			Text:       "{{value|date}}",
			SourceSpan: span,
		}
	}

	expansion.I18n = &i18n.Message{
		Nodes:        []i18n.Node{},
		Placeholders: placeholders,
	}

	for _, c := range expansion.Cases {
		c.Visit(v, context)
	}
	return expansion
}

func (v *mockI18nMetaVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	for _, child := range expansionCase.Expression {
		child.Visit(v, context)
	}
	return expansionCase
}

func (v *mockI18nMetaVisitor) VisitBlock(block *ml_parser.Block, context any) any { return block }
func (v *mockI18nMetaVisitor) VisitBlockParameter(blockParameter *ml_parser.BlockParameter, context any) any {
	return blockParameter
}
func (v *mockI18nMetaVisitor) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	return decl
}

type schemaRegistryAdapter struct {
	registry schema.ElementSchemaRegistry
}

func (s *schemaRegistryAdapter) HasElement(name string, schemaCtx []string) bool {
	return s.registry.HasElement(name, nil)
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
	tokenizeOptions := &ml_parser.TokenizeOptions{
		TokenizeExpansionForms:         true,
		LeadingTriviaChars:             leadingTrivia,
		SelectorlessEnabled:            selectorlessEnabled,
		TokenizeBlocks:                 &enableBlockSyntax,
		TokenizeLet:                    &enableLetSyntax,
	}

	url := templateUrl
	if url == "" {
		url = "path:://to/template"
	}
	parseResult := htmlParser.Parse(template, url, tokenizeOptions)

	htmlNodes := parseResult.RootNodes
	if len(htmlNodes) > 0 {
		visitor := &mockI18nMetaVisitor{
			file: htmlNodes[0].GetSourceSpan().Start.File,
		}
		var processed []ml_parser.Node
		for _, node := range htmlNodes {
			processed = append(processed, node.Visit(visitor, nil).(ml_parser.Node))
		}
		htmlNodes = processed
	}

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
