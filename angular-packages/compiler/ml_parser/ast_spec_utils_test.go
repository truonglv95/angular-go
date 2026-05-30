package ml_parser_test

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

func humanizeDom(parseResult *ml_parser.ParseTreeResult, addSourceSpan ...bool) []any {
	if len(parseResult.Errors) > 0 {
		var errorStrings []string
		for _, err := range parseResult.Errors {
			errorStrings = append(errorStrings, err.Error())
		}
		errorString := strings.Join(errorStrings, "\n")
		panic(fmt.Sprintf("Unexpected parse errors:\n%s", errorString))
	}
	hasSpan := false
	if len(addSourceSpan) > 0 {
		hasSpan = addSourceSpan[0]
	}
	return humanizeNodes(parseResult.RootNodes, hasSpan)
}

func humanizeDomSourceSpans(parseResult *ml_parser.ParseTreeResult) []any {
	return humanizeDom(parseResult, true)
}

func humanizeNodes(nodes []ml_parser.Node, addSourceSpan bool) []any {
	humanizer := &_Humanizer{
		result: make([]any, 0),
		includeSourceSpan: addSourceSpan,
	}
	for _, node := range nodes { node.Visit(humanizer, nil) }
	return humanizer.result
}

func humanizeLineColumn(location *parse_util.ParseLocation) string {
	return fmt.Sprintf("%d:%d", location.Line, location.Col)
}

type _Humanizer struct {
	result            []any
	elDepth           int
	includeSourceSpan bool
}

func (h *_Humanizer) appendContext(ast ml_parser.Node, input []any) []any {
	if !h.includeSourceSpan {
		return input
	}
	input = append(input, ast.GetSourceSpan().ToString())
	if ast.GetSourceSpan().FullStart.Offset != ast.GetSourceSpan().Start.Offset {
		input = append(input, ast.GetSourceSpan().FullStart.File.Content[ast.GetSourceSpan().FullStart.Offset:ast.GetSourceSpan().End.Offset])
	}
	return input
}

func (h *_Humanizer) VisitElement(element *ml_parser.Element, context any) any {
	res := h.appendContext(element, []any{
		"Element",
		element.Name,
		h.elDepth,
	})
	h.elDepth++
	if element.IsSelfClosing {
		res = append(res, "#selfClosing")
	}
	
	if h.includeSourceSpan {
		if element.StartSourceSpan != nil {
			res = append(res, element.StartSourceSpan.ToString())
		} else {
			res = append(res, nil)
		}
		if element.EndSourceSpan != nil {
			res = append(res, element.EndSourceSpan.ToString())
		} else {
			res = append(res, nil)
		}
	}
	h.result = append(h.result, res)

	// Since Attributes/Directives/Children might not match `[]Node`, we might need to iterate
	for _, attr := range element.Attrs {
		h.VisitAttribute(attr, nil)
	}
	// Directives - check if they exist in Go Element
	// HTML parser doesn't really have Directives yet, let's see. Wait, we should just visit children.
	for _, node := range element.Children { node.Visit(h, nil) }
	h.elDepth--
	return nil
}

func (h *_Humanizer) VisitAttribute(attribute *ml_parser.Attribute, context any) any {
	res := []any{"Attribute", attribute.Name, attribute.Value}
	for _, token := range attribute.ValueTokens {
		res = append(res, token.Parts)
	}
	h.result = append(h.result, h.appendContext(attribute, res))
	return nil
}

func (h *_Humanizer) VisitText(text *ml_parser.Text, context any) any {
	res := []any{"Text", text.Value, h.elDepth}
	for _, token := range text.Tokens {
		res = append(res, token.Parts)
	}
	h.result = append(h.result, h.appendContext(text, res))
	return nil
}

func (h *_Humanizer) VisitComment(comment *ml_parser.Comment, context any) any {
	res := []any{"Comment", comment.Value, h.elDepth}
	h.result = append(h.result, h.appendContext(comment, res))
	return nil
}

func (h *_Humanizer) VisitExpansion(expansion *ml_parser.Expansion, context any) any {
	res := []any{"Expansion", expansion.SwitchValue, expansion.Type, h.elDepth}
	h.elDepth++
	h.result = append(h.result, h.appendContext(expansion, res))
	for _, c := range expansion.Cases {
		h.VisitExpansionCase(c, nil)
	}
	h.elDepth--
	return nil
}

func (h *_Humanizer) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	res := []any{"ExpansionCase", expansionCase.Value, h.elDepth}
	h.result = append(h.result, h.appendContext(expansionCase, res))
	return nil
}

func (h *_Humanizer) VisitBlock(block *ml_parser.Block, context any) any {
	res := h.appendContext(block, []any{"Block", block.Name, h.elDepth})
	h.elDepth++
	if h.includeSourceSpan {
		if block.StartSourceSpan != nil {
			res = append(res, block.StartSourceSpan.ToString())
		} else {
			res = append(res, nil)
		}
		if block.EndSourceSpan != nil {
			res = append(res, block.EndSourceSpan.ToString())
		} else {
			res = append(res, nil)
		}
	}
	h.result = append(h.result, res)
	for _, param := range block.Parameters {
		h.VisitBlockParameter(param, nil)
	}
	for _, node := range block.Children { node.Visit(h, nil) }
	h.elDepth--
	return nil
}

func (h *_Humanizer) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	h.result = append(h.result, h.appendContext(parameter, []any{"BlockParameter", parameter.Expression}))
	return nil
}


func VisitAll(visitor ml_parser.Visitor, nodes []ml_parser.Node, context any) []any {
	var result []any
	for _, node := range nodes {
		res := node.Visit(visitor, context)
		if res != nil {
			result = append(result, res)
		}
	}
	return result
}

type serializerVisitor struct{}

func (v *serializerVisitor) VisitElement(element *ml_parser.Element, context any) any {
	attrsStr := v.visitAllAttrs(element.Attrs, " ", " ")
	tagDef := ml_parser.GetHtmlTagDefinition(element.Name)
	if tagDef.IsVoid() {
		return fmt.Sprintf("<%s%s/>", element.Name, attrsStr)
	}
	return fmt.Sprintf("<%s%s>%s</%s>", element.Name, attrsStr, v.visitAll(element.Children, "", ""), element.Name)
}

func (v *serializerVisitor) VisitAttribute(attribute *ml_parser.Attribute, context any) any {
	return fmt.Sprintf("%s=\"%s\"", attribute.Name, attribute.Value)
}

func (v *serializerVisitor) VisitText(text *ml_parser.Text, context any) any {
	return text.Value
}

func (v *serializerVisitor) VisitComment(comment *ml_parser.Comment, context any) any {
	return fmt.Sprintf("<!--%s-->", comment.Value)
}

func (v *serializerVisitor) VisitExpansion(expansion *ml_parser.Expansion, context any) any {
	cases := v.visitAllCases(expansion.Cases, "", "")
	return fmt.Sprintf("{%s, %s,%s}", expansion.SwitchValue, expansion.Type, cases)
}

func (v *serializerVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	expr := v.visitAll(expansionCase.Expression, "", "")
	return fmt.Sprintf(" %s {%s}", expansionCase.Value, expr)
}

func (v *serializerVisitor) VisitBlock(block *ml_parser.Block, context any) any {
	params := " "
	if len(block.Parameters) > 0 {
		params = fmt.Sprintf(" (%s) ", v.visitAllBlockParams(block.Parameters, ";", " "))
	}
	return fmt.Sprintf("@%s%s{%s}", block.Name, params, v.visitAll(block.Children, "", ""))
}

func (v *serializerVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return parameter.Expression
}

func (v *serializerVisitor) visitAll(nodes []ml_parser.Node, separator, prefix string) string {
	if len(nodes) == 0 {
		return ""
	}
	parts := make([]string, len(nodes))
	for i, node := range nodes {
		parts[i] = node.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func (v *serializerVisitor) visitAllAttrs(attrs []*ml_parser.Attribute, separator, prefix string) string {
	if len(attrs) == 0 {
		return ""
	}
	parts := make([]string, len(attrs))
	for i, attr := range attrs {
		parts[i] = attr.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func (v *serializerVisitor) visitAllCases(cases []*ml_parser.ExpansionCase, separator, prefix string) string {
	if len(cases) == 0 {
		return ""
	}
	parts := make([]string, len(cases))
	for i, c := range cases {
		parts[i] = c.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func (v *serializerVisitor) visitAllBlockParams(params []*ml_parser.BlockParameter, separator, prefix string) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.Visit(v, nil).(string)
	}
	return prefix + strings.Join(parts, separator)
}

func serializeNodes(nodes []ml_parser.Node) []string {
	v := &serializerVisitor{}
	result := make([]string, len(nodes))
	for i, node := range nodes {
		result[i] = node.Visit(v, nil).(string)
	}
	return result
}
