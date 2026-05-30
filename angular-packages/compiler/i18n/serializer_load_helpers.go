package i18n

import (
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

var nonAlphaNumRE = regexp.MustCompile(`[^a-z0-9]+`)

func errorsFromML(errs []error) []*parse_util.ParseError {
	out := make([]*parse_util.ParseError, 0, len(errs))
	for _, err := range errs {
		if pe, ok := err.(*parse_util.ParseError); ok {
			out = append(out, pe)
		}
	}
	return out
}

func joinParseErrors(errs []*parse_util.ParseError) string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "\n")
}

func findMLAttr(attrs []*ml_parser.Attribute, name string) *ml_parser.Attribute {
	for _, attr := range attrs {
		if attr.Name == name {
			return attr
		}
	}
	return nil
}

func innerText(element *ml_parser.Element) string {
	if element.EndSourceSpan == nil {
		return ""
	}
	start := element.StartSourceSpan.End.Offset
	end := element.EndSourceSpan.Start.Offset
	if end < start {
		return ""
	}
	content := element.StartSourceSpan.Start.File.Content
	return content[start:end]
}

func appendNodes(dst []Node, visited any) []Node {
	switch v := visited.(type) {
	case nil:
		return dst
	case Node:
		return append(dst, v)
	case []Node:
		return append(dst, v...)
	default:
		return dst
	}
}

func expansionCasesToNodes(cases []*ml_parser.ExpansionCase) []ml_parser.Node {
	nodes := make([]ml_parser.Node, len(cases))
	for i, c := range cases {
		nodes[i] = c
	}
	return nodes
}
