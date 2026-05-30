package template_parser

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
)

// TemplateParser is responsible for parsing a component's HTML template
// and combining ml_parser (HTML AST) with expression_parser (Angular AST).
type TemplateParser struct {
	// exprParser *expression_parser.Parser
}

func NewTemplateParser() *TemplateParser {
	return &TemplateParser{}
}

// ParsedTemplate represents the final Template AST
type ParsedTemplate struct {
	Nodes  []TemplateNode
	Errors []error
}

type TemplateNode interface {
	isTemplateNode()
}

// Element node with bindings
type ElementNode struct {
	Name       string
	Attributes []*ml_parser.Attribute
	Inputs     []BoundInput  // e.g., [property]="expression"
	Outputs    []BoundOutput // e.g., (event)="action"
	Children   []TemplateNode
}

func (e *ElementNode) isTemplateNode() {}

// Bound input property
type BoundInput struct {
	Name  string
	Value string
}

// Bound output event
type BoundOutput struct {
	Name   string
	Action string
}

// Text node with optional interpolation
type TextNode struct {
	Value         string
	Interpolation string // Present if {{ }} is found
}

func (t *TextNode) isTemplateNode() {}

// Parse parses the raw HTML string into a Template AST.
func (tp *TemplateParser) Parse(template string, sourceUrl string) ParsedTemplate {
	// 1. Run ml_parser to get the HTML AST
	htmlResult := ml_parser.ParseHTML(template, sourceUrl)
	nodes := htmlResult.RootNodes
	var err error
	if len(htmlResult.Errors) > 0 { err = htmlResult.Errors[0] }

	result := ParsedTemplate{}
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("HTML Parse Error: %v", err))
		return result
	}

	// 2. Traverse ML AST and link expression_parser
	var visitNodes func(nodes []ml_parser.Node) []TemplateNode
	visitNodes = func(nodes []ml_parser.Node) []TemplateNode {
		var templateNodes []TemplateNode
		for _, node := range nodes {
			switch n := node.(type) {
			case *ml_parser.Element:
				el := &ElementNode{
					Name:       n.Name,
					Attributes: []*ml_parser.Attribute{},
					Inputs:     []BoundInput{},
					Outputs:    []BoundOutput{},
				}

				// Process attributes to identify bindings
				for _, attr := range n.Attrs {
					name := attr.Name
					val := attr.Value

					if strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]") {
						// Property Binding: [prop]="expr"
						propName := name[1 : len(name)-1]
						// exprAst := tp.exprParser.ParseBinding(val, sourceUrl, 0, nil)
						el.Inputs = append(el.Inputs, BoundInput{
							Name:  propName,
							Value: val,
						})
					} else if strings.HasPrefix(name, "(") && strings.HasSuffix(name, ")") {
						// Event Binding: (event)="action"
						eventName := name[1 : len(name)-1]
						// actionAst := tp.exprParser.ParseAction(val, sourceUrl, 0, nil)
						el.Outputs = append(el.Outputs, BoundOutput{
							Name:   eventName,
							Action: val,
						})
					} else {
						// Regular attribute
						el.Attributes = append(el.Attributes, attr)
					}
				}

				// Recursively process children
				el.Children = visitNodes(n.Children)
				templateNodes = append(templateNodes, el)

			case *ml_parser.Text:
				textNode := &TextNode{
					Value: n.Value,
				}
				// Check if text contains interpolation {{ }}
				if strings.Contains(n.Value, "{{") && strings.Contains(n.Value, "}}") {
					// It's interpolation! Use the expression parser to parse it
					// interpAst := tp.exprParser.ParseInterpolation(n.Value, sourceUrl, 0, nil)
					textNode.Interpolation = n.Value
				}
				templateNodes = append(templateNodes, textNode)
			}
		}
		return templateNodes
	}

	result.Nodes = visitNodes(nodes)
	return result
}
