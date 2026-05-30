package render3

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

type MockSchemaRegistry struct {
	existingProperties map[string]bool
	attrPropMapping    map[string]string
	existingElements   map[string]bool
	invalidProperties  []string
	invalidAttributes  []string
}

func NewMockSchemaRegistry() *MockSchemaRegistry {
	return &MockSchemaRegistry{
		existingProperties: map[string]bool{"invalidProp": false},
		attrPropMapping:    map[string]string{"mappedAttr": "mappedProp"},
		existingElements:   map[string]bool{"unknown": false, "un-known": false},
		invalidProperties:  []string{"onEvent"},
		invalidAttributes:  []string{"onEvent"},
	}
}

func (m *MockSchemaRegistry) HasProperty(tagName string, propName string, schemaMetas []core.SchemaMetadata) bool {
	val, ok := m.existingProperties[propName]
	if !ok {
		return true
	}
	return val
}

func (m *MockSchemaRegistry) HasElement(name string, schemaCtx []string) bool {
	val, ok := m.existingElements[strings.ToLower(name)]
	if !ok {
		return true
	}
	return val
}

func (m *MockSchemaRegistry) SecurityContext(name string, propName string, isAttribute bool) core.SecurityContext {
	return core.SecurityContextNone
}

func (m *MockSchemaRegistry) AllKnownElementNames() []string {
	var names []string
	for k := range m.existingElements {
		names = append(names, k)
	}
	return names
}

func (m *MockSchemaRegistry) GetMappedPropName(propName string) string {
	if val, ok := m.attrPropMapping[propName]; ok {
		return val
	}
	return propName
}

func (m *MockSchemaRegistry) GetDefaultComponentElementName() string {
	return "ng-component"
}

func (m *MockSchemaRegistry) ValidateProperty(name string) struct {
	Error bool
	Msg   *string
} {
	for _, p := range m.invalidProperties {
		if p == name {
			msg := "Binding to property '" + name + "' is disallowed for security reasons"
			return struct {
				Error bool
				Msg   *string
			}{Error: true, Msg: &msg}
		}
	}
	return struct {
		Error bool
		Msg   *string
	}{Error: false}
}

func (m *MockSchemaRegistry) ValidateAttribute(name string) struct {
	Error bool
	Msg   *string
} {
	for _, a := range m.invalidAttributes {
		if a == name {
			msg := "Binding to attribute '" + name + "' is disallowed for security reasons"
			return struct {
				Error bool
				Msg   *string
			}{Error: true, Msg: &msg}
		}
	}
	return struct {
		Error bool
		Msg   *string
	}{Error: false}
}

func (m *MockSchemaRegistry) NormalizeAnimationStyleProperty(propName string) string {
	return propName
}

func (m *MockSchemaRegistry) NormalizeAnimationStyleValue(camelCaseProp string, userProvidedProp string, val any) struct {
	Error string
	Value string
} {
	var valStr string
	if s, ok := val.(string); ok {
		valStr = s
	} else {
		valStr = fmt.Sprintf("%v", val)
	}
	return struct {
		Error string
		Value string
	}{Error: "", Value: valStr}
}

type ParseR3Options struct {
	PreserveWhitespaces bool
	LeadingTriviaChars  []string
	IgnoreError         bool
	SelectorlessEnabled bool
	CollectCommentNodes bool
}

func parseR3(input string, options ParseR3Options) Render3ParseResult {
	htmlParser := ml_parser.NewHtmlParser()
	leadingTrivia := options.LeadingTriviaChars
	if len(leadingTrivia) == 0 {
		leadingTrivia = LEADING_TRIVIA_CHARS
	}

	tokenizeOptions := &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: true,
		LeadingTriviaChars:     leadingTrivia,
		SelectorlessEnabled:    options.SelectorlessEnabled,
	}

	parseResult := htmlParser.Parse(input, "path:://to/template", tokenizeOptions)

	if len(parseResult.Errors) > 0 && !options.IgnoreError {
		var msgs []string
		for _, e := range parseResult.Errors {
			msgs = append(msgs, e.Error())
		}
		panic(strings.Join(msgs, "\n"))
	}

	htmlNodes := processI18nMeta(parseResult)
	if !options.PreserveWhitespaces {
		htmlNodes = ml_parser.VisitAllWithSiblingsForNodes(ml_parser.NewWhitespaceVisitor(true), htmlNodes)
	}

	lexer := expression_parser.Lexer{}
	parser := expression_parser.NewParser(lexer, false)
	schemaRegistry := NewMockSchemaRegistry()
	bindingParser := template_parser.NewBindingParser(parser, schemaRegistry, nil)

	r3Result := HtmlAstToRender3Ast(htmlNodes, bindingParser, Render3ParseOptions{CollectCommentNodes: options.CollectCommentNodes})

	for _, err := range parseResult.Errors {
		if pe, ok := err.(*parse_util.ParseError); ok {
			r3Result.Errors = append(r3Result.Errors, pe)
		} else {
			r3Result.Errors = append(r3Result.Errors, &parse_util.ParseError{
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
		r3Result.Errors = append(r3Result.Errors, &parse_util.ParseError{
			Span: &spanVal,
			Msg:  e.Message,
		})
	}

	if len(r3Result.Errors) > 0 && !options.IgnoreError {
		var msgs []string
		for _, e := range r3Result.Errors {
			msgs = append(msgs, e.Error())
		}
		panic(strings.Join(msgs, "\n"))
	}

	return r3Result
}

func processI18nMeta(result *ml_parser.ParseTreeResult) []ml_parser.Node {
	if len(result.RootNodes) == 0 {
		return result.RootNodes
	}
	visitor := &mockI18nMetaVisitor{
		file: result.RootNodes[0].GetSourceSpan().Start.File,
	}
	var processed []ml_parser.Node
	for _, node := range result.RootNodes {
		processed = append(processed, node.Visit(visitor, nil).(ml_parser.Node))
	}
	return processed
}


