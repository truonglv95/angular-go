package ml_parser

import (
	"regexp"
	"strings"
)

const PRESERVE_WS_ATTR_NAME = "ngPreserveWhitespaces"

var (
	skipWsTrimTags = map[string]bool{
		"pre":      true,
		"template": true,
		"textarea": true,
		"script":   true,
		"style":    true,
	}

	wsChars         = " \f\n\r\t\v\u1680\u180e\u2000-\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"
	noWsRegexp      = regexp.MustCompile(`[^` + wsChars + `]`)
	wsReplaceRegexp = regexp.MustCompile(`[` + wsChars + `]{2,}`)
)

func hasPreserveWhitespacesAttr(attrs []*Attribute) bool {
	for _, attr := range attrs {
		if attr.Name == PRESERVE_WS_ATTR_NAME {
			return true
		}
	}
	return false
}

func ReplaceNgsp(value string) string {
	return strings.ReplaceAll(value, "\uE500", " ")
}

type SiblingVisitorContext struct {
	Prev Node
	Next Node
}

type WhitespaceVisitor struct {
	preserveSignificantWhitespace bool
	originalNodeMap               map[Node]Node
	requireContext                bool
	icuExpansionDepth             int
}

func NewWhitespaceVisitor(preserveSignificantWhitespace bool) *WhitespaceVisitor {
	return &WhitespaceVisitor{
		preserveSignificantWhitespace: preserveSignificantWhitespace,
		requireContext:                true,
	}
}

func NewWhitespaceVisitorWithMap(preserveSignificantWhitespace bool, originalNodeMap map[Node]Node) *WhitespaceVisitor {
	return &WhitespaceVisitor{
		preserveSignificantWhitespace: preserveSignificantWhitespace,
		originalNodeMap:               originalNodeMap,
		requireContext:                true,
	}
}

func (v *WhitespaceVisitor) VisitElement(element *Element, context any) any {
	if skipWsTrimTags[element.Name] || hasPreserveWhitespacesAttr(element.Attrs) {
		newElement := &Element{
			BaseNode:        BaseNode{SourceSpan: element.SourceSpan, I18n: element.I18n},
			Name:            element.Name,
			Attrs:           VisitAllWithSiblingsForAttributes(v, element.Attrs),
			Directives:      element.Directives,
			Children:        element.Children,
			StartSourceSpan: element.StartSourceSpan,
			EndSourceSpan:   element.EndSourceSpan,
			IsVoid:          element.IsVoid,
			IsSelfClosing:   element.IsSelfClosing,
		}
		if v.originalNodeMap != nil {
			v.originalNodeMap[newElement] = element
		}
		return newElement
	}

	newElement := &Element{
		BaseNode:        BaseNode{SourceSpan: element.SourceSpan, I18n: element.I18n},
		Name:            element.Name,
		Attrs:           element.Attrs,
		Directives:      element.Directives,
		Children:        VisitAllWithSiblingsForNodes(v, element.Children),
		StartSourceSpan: element.StartSourceSpan,
		EndSourceSpan:   element.EndSourceSpan,
		IsVoid:          element.IsVoid,
		IsSelfClosing:   element.IsSelfClosing,
	}
	if v.originalNodeMap != nil {
		v.originalNodeMap[newElement] = element
	}
	return newElement
}

func (v *WhitespaceVisitor) VisitComponent(component *Component, context any) any {
	el := &component.Element
	var newElement *Element
	if skipWsTrimTags[el.Name] || hasPreserveWhitespacesAttr(el.Attrs) {
		newElement = &Element{
			BaseNode:        BaseNode{SourceSpan: el.SourceSpan, I18n: el.I18n},
			Name:            el.Name,
			Attrs:           VisitAllWithSiblingsForAttributes(v, el.Attrs),
			Directives:      el.Directives,
			Children:        el.Children,
			StartSourceSpan: el.StartSourceSpan,
			EndSourceSpan:   el.EndSourceSpan,
			IsVoid:          el.IsVoid,
			IsSelfClosing:   el.IsSelfClosing,
		}
	} else {
		newElement = &Element{
			BaseNode:        BaseNode{SourceSpan: el.SourceSpan, I18n: el.I18n},
			Name:            el.Name,
			Attrs:           el.Attrs,
			Directives:      el.Directives,
			Children:        VisitAllWithSiblingsForNodes(v, el.Children),
			StartSourceSpan: el.StartSourceSpan,
			EndSourceSpan:   el.EndSourceSpan,
			IsVoid:          el.IsVoid,
			IsSelfClosing:   el.IsSelfClosing,
		}
	}
	newComponent := &Component{
		Element:       *newElement,
		ComponentName: component.ComponentName,
		TagName:       component.TagName,
		FullName:      component.FullName,
	}
	if v.originalNodeMap != nil {
		v.originalNodeMap[newComponent] = component
	}
	return newComponent
}

func (v *WhitespaceVisitor) VisitAttribute(attribute *Attribute, context any) any {
	if attribute.Name != PRESERVE_WS_ATTR_NAME {
		return attribute
	}
	return nil
}

func (v *WhitespaceVisitor) VisitText(text *Text, context any) any {
	ctx, ok := context.(*SiblingVisitorContext)
	if !ok {
		ctx = nil
	}

	isNotBlank := noWsRegexp.MatchString(text.Value)
	hasExpansionSibling := false
	if ctx != nil {
		if _, isExpansion := ctx.Prev.(*Expansion); isExpansion {
			hasExpansionSibling = true
		}
		if _, isExpansion := ctx.Next.(*Expansion); isExpansion {
			hasExpansionSibling = true
		}
	}

	inIcuExpansion := v.icuExpansionDepth > 0
	if inIcuExpansion && v.preserveSignificantWhitespace {
		return text
	}

	if isNotBlank || hasExpansionSibling {
		tokens := make([]Token, len(text.Tokens))
		for i, token := range text.Tokens {
			if token.Type == TokenTypeText {
				tokens[i] = createWhitespaceProcessedTextToken(token)
			} else {
				tokens[i] = token
			}
		}

		if !v.preserveSignificantWhitespace && len(tokens) > 0 {
			firstToken := tokens[0]
			tokens[0] = trimLeadingWhitespace(firstToken, ctx)

			lastIdx := len(tokens) - 1
			lastToken := tokens[lastIdx]
			tokens[lastIdx] = trimTrailingWhitespace(lastToken, ctx)
		}

		processed := processWhitespace(text.Value)
		var value string
		if v.preserveSignificantWhitespace {
			value = processed
		} else {
			value = trimLeadingAndTrailingWhitespace(processed, ctx)
		}

		result := &Text{
			Value:  value,
			Tokens: tokens,
		}
		result.SourceSpan = text.SourceSpan
		result.I18n = text.I18n

		if v.originalNodeMap != nil {
			v.originalNodeMap[result] = text
		}
		return result
	}

	return nil
}

func (v *WhitespaceVisitor) VisitComment(comment *Comment, context any) any {
	return comment
}

func (v *WhitespaceVisitor) VisitExpansion(expansion *Expansion, context any) any {
	v.icuExpansionDepth++
	defer func() {
		v.icuExpansionDepth--
	}()

	newCases := VisitAllWithSiblingsForExpansionCases(v, expansion.Cases)

	newExpansion := &Expansion{
		SwitchValue:           expansion.SwitchValue,
		Type:                  expansion.Type,
		Cases:                 newCases,
	}
	newExpansion.SourceSpan = expansion.SourceSpan
	newExpansion.I18n = expansion.I18n

	if v.originalNodeMap != nil {
		v.originalNodeMap[newExpansion] = expansion
	}

	return newExpansion
}

func (v *WhitespaceVisitor) VisitExpansionCase(expansionCase *ExpansionCase, context any) any {
	newExpansionCase := &ExpansionCase{
		Value:           expansionCase.Value,
		Expression:      VisitAllWithSiblingsForNodes(v, expansionCase.Expression),
	}
	newExpansionCase.SourceSpan = expansionCase.SourceSpan

	if v.originalNodeMap != nil {
		v.originalNodeMap[newExpansionCase] = expansionCase
	}
	return newExpansionCase
}

func (v *WhitespaceVisitor) VisitBlock(block *Block, context any) any {
	newBlock := &Block{
		Name:            block.Name,
		Parameters:      block.Parameters,
		Children:        VisitAllWithSiblingsForNodes(v, block.Children),
		StartSourceSpan: block.StartSourceSpan,
		EndSourceSpan:   block.EndSourceSpan,
	}
	newBlock.SourceSpan = block.SourceSpan

	if v.originalNodeMap != nil {
		v.originalNodeMap[newBlock] = block
	}
	return newBlock
}

func (v *WhitespaceVisitor) VisitBlockParameter(parameter *BlockParameter, context any) any {
	return parameter
}

func (v *WhitespaceVisitor) VisitLetDeclaration(decl *LetDeclaration, context any) any {
	return decl
}

func trimLeadingWhitespace(token Token, context *SiblingVisitorContext) Token {
	if token.Type != TokenTypeText {
		return token
	}

	isFirstTokenInTag := context == nil || context.Prev == nil
	if !isFirstTokenInTag {
		return token
	}

	return transformTextToken(token, func(text string) string {
		return strings.TrimLeft(text, wsChars)
	})
}

func trimTrailingWhitespace(token Token, context *SiblingVisitorContext) Token {
	if token.Type != TokenTypeText {
		return token
	}

	isLastTokenInTag := context == nil || context.Next == nil
	if !isLastTokenInTag {
		return token
	}

	return transformTextToken(token, func(text string) string {
		return strings.TrimRight(text, wsChars)
	})
}

func trimLeadingAndTrailingWhitespace(text string, context *SiblingVisitorContext) string {
	isFirstTokenInTag := context == nil || context.Prev == nil
	isLastTokenInTag := context == nil || context.Next == nil

	maybeTrimmedStart := text
	if isFirstTokenInTag {
		maybeTrimmedStart = strings.TrimLeft(text, wsChars)
	}

	maybeTrimmed := maybeTrimmedStart
	if isLastTokenInTag {
		maybeTrimmed = strings.TrimRight(maybeTrimmedStart, wsChars)
	}
	return maybeTrimmed
}

func createWhitespaceProcessedTextToken(token Token) Token {
	newToken := Token{
		Type:       token.Type,
		Parts:      []string{processWhitespace(token.Parts[0])},
		SourceSpan: token.SourceSpan,
	}
	return newToken
}

func transformTextToken(token Token, transform func(string) string) Token {
	newToken := Token{
		Type:       token.Type,
		Parts:      []string{transform(token.Parts[0])},
		SourceSpan: token.SourceSpan,
	}
	return newToken
}

func processWhitespace(text string) string {
	return wsReplaceRegexp.ReplaceAllString(ReplaceNgsp(text), " ")
}

func RemoveWhitespaces(htmlAstWithErrors *ParseTreeResult, preserveSignificantWhitespace bool) *ParseTreeResult {
	visitor := NewWhitespaceVisitor(preserveSignificantWhitespace)
	return &ParseTreeResult{
		RootNodes: VisitAllWithSiblingsForNodes(visitor, htmlAstWithErrors.RootNodes),
		Errors:    htmlAstWithErrors.Errors,
	}
}

func VisitAllWithSiblingsForNodes(visitor *WhitespaceVisitor, nodes []Node) []Node {
	var result []Node
	for i, ast := range nodes {
		ctx := &SiblingVisitorContext{}
		if i > 0 {
			ctx.Prev = nodes[i-1]
		}
		if i < len(nodes)-1 {
			ctx.Next = nodes[i+1]
		}

		astResult := ast.Visit(visitor, ctx)
		if astResult != nil {
			result = append(result, astResult.(Node))
		}
	}
	return result
}

func VisitAllWithSiblingsForAttributes(visitor *WhitespaceVisitor, nodes []*Attribute) []*Attribute {
	var result []*Attribute
	for _, ast := range nodes {
		ctx := &SiblingVisitorContext{}
		astResult := visitor.VisitAttribute(ast, ctx)
		if astResult != nil {
			result = append(result, astResult.(*Attribute))
		}
	}
	return result
}

func VisitAllWithSiblingsForExpansionCases(visitor *WhitespaceVisitor, nodes []*ExpansionCase) []*ExpansionCase {
	var result []*ExpansionCase
	for _, ast := range nodes {
		ctx := &SiblingVisitorContext{}
		astResult := visitor.VisitExpansionCase(ast, ctx)
		if astResult != nil {
			result = append(result, astResult.(*ExpansionCase))
		}
	}
	return result
}
