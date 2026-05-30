package i18n

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

// To preserve structural parity, we define VisitNodeFn
type VisitNodeFn func(htmlNode ml_parser.Node, i18nNode Node) Node

type I18nMessageFactory func(nodes []ml_parser.Node, meaning *string, description *string, customId *string, visitNodeFn VisitNodeFn) *Message

var expParser = expression_parser.NewParser(expression_parser.Lexer{}, false)

func CreateI18nMessageFactory(retainEmptyTokens bool, preserveExpressionWhitespace bool) I18nMessageFactory {
	visitor := NewI18nVisitor(expParser, retainEmptyTokens, preserveExpressionWhitespace)
	return func(nodes []ml_parser.Node, meaning *string, description *string, customId *string, visitNodeFn VisitNodeFn) *Message {
		m := ""
		if meaning != nil {
			m = *meaning
		}
		d := ""
		if description != nil {
			d = *description
		}
		c := ""
		if customId != nil {
			c = *customId
		}
		return visitor.ToI18nMessage(nodes, m, d, c, visitNodeFn)
	}
}

type I18nMessageVisitorContext struct {
	IsIcu                bool
	IcuDepth             int
	PlaceholderRegistry  *PlaceholderRegistry
	PlaceholderToContent map[string]*MessagePlaceholder
	PlaceholderToMessage map[string]*Message
	VisitNodeFn          VisitNodeFn
}

func noopVisitNodeFn(htmlNode ml_parser.Node, i18nNode Node) Node {
	return i18nNode
}

type I18nVisitor struct {
	expressionParser             *expression_parser.Parser
	retainEmptyTokens            bool
	preserveExpressionWhitespace bool
}

func NewI18nVisitor(expressionParser *expression_parser.Parser, retainEmptyTokens bool, preserveExpressionWhitespace bool) *I18nVisitor {
	return &I18nVisitor{
		expressionParser:             expressionParser,
		retainEmptyTokens:            retainEmptyTokens,
		preserveExpressionWhitespace: preserveExpressionWhitespace,
	}
}

func visitAll(visitor ml_parser.Visitor, nodes []ml_parser.Node, context any) []Node {
	var result []Node
	for _, node := range nodes {
		res := node.Visit(visitor, context)
		if res != nil {
			result = append(result, res.(Node))
		}
	}
	return result
}

func (v *I18nVisitor) ToI18nMessage(nodes []ml_parser.Node, meaning string, description string, customId string, visitNodeFn VisitNodeFn) *Message {
	isIcu := false
	if len(nodes) == 1 {
		_, ok := nodes[0].(*ml_parser.Expansion)
		isIcu = ok
	}

	if visitNodeFn == nil {
		visitNodeFn = noopVisitNodeFn
	}

	context := &I18nMessageVisitorContext{
		IsIcu:                isIcu,
		IcuDepth:             0,
		PlaceholderRegistry:  NewPlaceholderRegistry(),
		PlaceholderToContent: make(map[string]*MessagePlaceholder),
		PlaceholderToMessage: make(map[string]*Message),
		VisitNodeFn:          visitNodeFn,
	}

	i18nodes := visitAll(v, nodes, context)

	return NewMessage(
		i18nodes,
		context.PlaceholderToContent,
		context.PlaceholderToMessage,
		meaning,
		description,
		customId,
	)
}

func (v *I18nVisitor) VisitElement(el *ml_parser.Element, context any) any {
	ctx := context.(*I18nMessageVisitorContext); _ = ctx
	return v.visitElementLike(el, ctx)
}

func (v *I18nVisitor) VisitAttribute(attribute *ml_parser.Attribute, context any) any {
	ctx := context.(*I18nMessageVisitorContext)
	var node Node
	span := attribute.ValueSpan
	if span == nil {
		span = attribute.SourceSpan
	}
	if len(attribute.ValueTokens) == 0 || len(attribute.ValueTokens) == 1 {
		node = NewText(attribute.Value, span)
	} else {
		node = v.visitTextWithInterpolation(attribute.ValueTokens, span, ctx, nil)
	}
	return ctx.VisitNodeFn(attribute, node)
}

func (v *I18nVisitor) VisitText(text *ml_parser.Text, context any) any {
	ctx := context.(*I18nMessageVisitorContext)
	var node Node
	if len(text.Tokens) == 1 {
		node = NewText(text.Value, text.SourceSpan)
	} else {
		node = v.visitTextWithInterpolation(text.Tokens, text.SourceSpan, ctx, nil)
	}
	return ctx.VisitNodeFn(text, node)
}

func (v *I18nVisitor) VisitComment(comment *ml_parser.Comment, context any) any {
	return nil
}

func (v *I18nVisitor) VisitExpansion(icu *ml_parser.Expansion, context any) any {
	ctx := context.(*I18nMessageVisitorContext)
	ctx.IcuDepth++
	var caseOrders []string
	i18nIcuCases := make(map[string]Node)
	for _, caze := range icu.Cases {
		i18nIcuCases[caze.Value] = NewContainer(visitAll(v, caze.Expression, ctx), nil)
		caseOrders = append(caseOrders, caze.Value)
	}
	i18nIcu := NewIcu(icu.SwitchValue, icu.Type, i18nIcuCases, caseOrders, icu.SourceSpan, "")
	ctx.IcuDepth--

	if ctx.IsIcu || ctx.IcuDepth > 0 {
		expPh := ctx.PlaceholderRegistry.GetUniquePlaceholder(fmt.Sprintf("VAR_%s", icu.Type))
		i18nIcu.ExpressionPlaceholder = expPh
		ctx.PlaceholderToContent[expPh] = &MessagePlaceholder{
			Text:       icu.SwitchValue,
			SourceSpan: nil,
		}
		return ctx.VisitNodeFn(icu, i18nIcu)
	}

	phName := ctx.PlaceholderRegistry.GetPlaceholderName("ICU", icu.SourceSpan.ToString())
	ctx.PlaceholderToMessage[phName] = v.ToI18nMessage([]ml_parser.Node{icu}, "", "", "", nil)
	node := NewIcuPlaceholder(i18nIcu, phName, icu.SourceSpan)
	return ctx.VisitNodeFn(icu, node)
}

func (v *I18nVisitor) VisitExpansionCase(icuCase *ml_parser.ExpansionCase, context any) any {
	panic("Unreachable code")
}

func (v *I18nVisitor) VisitBlock(block *ml_parser.Block, context any) any {
	ctx := context.(*I18nMessageVisitorContext)
	children := visitAll(v, block.Children, ctx)

	if block.Name == "switch" {
		return NewContainer(children, block.SourceSpan)
	}

	var parameters []string
	for _, param := range block.Parameters {
		parameters = append(parameters, param.Expression)
	}

	startPhName := ctx.PlaceholderRegistry.GetStartBlockPlaceholderName(block.Name, parameters)
	closePhName := ctx.PlaceholderRegistry.GetCloseBlockPlaceholderName(block.Name)

	ctx.PlaceholderToContent[startPhName] = &MessagePlaceholder{
		Text:       block.StartSourceSpan.ToString(),
		SourceSpan: block.StartSourceSpan,
	}

	endText := "}"
	if block.EndSourceSpan != nil {
		endText = block.EndSourceSpan.ToString()
	}

	endSpan := block.EndSourceSpan
	if endSpan == nil {
		endSpan = block.SourceSpan
	}

	ctx.PlaceholderToContent[closePhName] = &MessagePlaceholder{
		Text:       endText,
		SourceSpan: endSpan,
	}

	node := NewBlockPlaceholder(
		block.Name,
		parameters,
		startPhName,
		closePhName,
		children,
		block.SourceSpan,
		block.StartSourceSpan,
		block.EndSourceSpan,
	)
	return ctx.VisitNodeFn(block, node)
}

func (v *I18nVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	panic("Unreachable code")
}

func (v *I18nVisitor) visitElementLike(node ml_parser.Node, ctx *I18nMessageVisitorContext) Node {
	var children []Node
	var attrs map[string]string = make(map[string]string)
	var nodeName string
	var isVoid bool
	var sourceSpan, startSourceSpan, endSourceSpan *parse_util.ParseSourceSpan

	if el, ok := node.(*ml_parser.Element); ok {
		children = visitAll(v, el.Children, ctx)
		nodeName = el.Name
		isVoid = ml_parser.GetHtmlTagDefinition(el.Name).IsVoid()
		sourceSpan = el.SourceSpan
		startSourceSpan = el.StartSourceSpan
		endSourceSpan = el.EndSourceSpan
		for _, attr := range el.Attrs {
			attrs[attr.Name] = attr.Value
		}
	}

	startPhName := ctx.PlaceholderRegistry.GetStartTagPlaceholderName(nodeName, attrs, isVoid)
	ctx.PlaceholderToContent[startPhName] = &MessagePlaceholder{
		Text:       startSourceSpan.ToString(),
		SourceSpan: startSourceSpan,
	}

	closePhName := ""
	if !isVoid {
		closePhName = ctx.PlaceholderRegistry.GetCloseTagPlaceholderName(nodeName)
		endSpan := endSourceSpan
		if endSpan == nil {
			endSpan = sourceSpan
		}
		ctx.PlaceholderToContent[closePhName] = &MessagePlaceholder{
			Text:       fmt.Sprintf("</%s>", nodeName),
			SourceSpan: endSpan,
		}
	}

	i18nNode := NewTagPlaceholder(
		nodeName,
		attrs,
		startPhName,
		closePhName,
		children,
		isVoid,
		sourceSpan,
		startSourceSpan,
		endSourceSpan,
	)
	return ctx.VisitNodeFn(node, i18nNode)
}

func (v *I18nVisitor) visitTextWithInterpolation(
	tokens []ml_parser.Token,
	sourceSpan *parse_util.ParseSourceSpan,
	context *I18nMessageVisitorContext,
	previousI18n any,
) Node {
	var nodes []Node
	hasInterpolation := false
	for _, token := range tokens {
		switch token.Type {
		case ml_parser.TokenTypeInterpolation, ml_parser.TokenTypeAttrValueInterpolation:
			hasInterpolation = true
			startMarker := token.Parts[0]
			expression := token.Parts[1]
			endMarker := token.Parts[2]
			baseName := extractPlaceholderName(expression)
			if baseName == "" {
				baseName = "INTERPOLATION"
			}
			phName := context.PlaceholderRegistry.GetPlaceholderName(baseName, expression)

			if v.preserveExpressionWhitespace {
				context.PlaceholderToContent[phName] = &MessagePlaceholder{
					Text:       strings.Join(token.Parts, ""),
					SourceSpan: token.SourceSpan,
				}
				nodes = append(nodes, NewPlaceholder(expression, phName, token.SourceSpan))
			} else {
				normalized := v.normalizeExpression(token)
				context.PlaceholderToContent[phName] = &MessagePlaceholder{
					Text:       startMarker + normalized + endMarker,
					SourceSpan: token.SourceSpan,
				}
				nodes = append(nodes, NewPlaceholder(normalized, phName, token.SourceSpan))
			}
		default:
			if len(token.Parts[0]) > 0 || v.retainEmptyTokens {
				previousIndex := len(nodes) - 1
				if previousIndex >= 0 {
					if prevText, ok := nodes[previousIndex].(*Text); ok {
						prevText.Value += token.Parts[0]
						prevText.SourceSpan = parse_util.NewParseSourceSpan(
							prevText.SourceSpan.Start,
							token.SourceSpan.End,
							prevText.SourceSpan.FullStart,
							prevText.SourceSpan.Details,
						)
					} else {
						nodes = append(nodes, NewText(token.Parts[0], token.SourceSpan))
					}
				} else {
					nodes = append(nodes, NewText(token.Parts[0], token.SourceSpan))
				}
			} else {
				if v.retainEmptyTokens {
					nodes = append(nodes, NewText(token.Parts[0], token.SourceSpan))
				}
			}
		}
	}

	if hasInterpolation {
		return NewContainer(nodes, sourceSpan)
	} else {
		if len(nodes) > 0 {
			return nodes[0]
		}
		return nil
	}
}

func (v *I18nVisitor) normalizeExpression(token ml_parser.Token) string {
	expression := token.Parts[1]
	var start, end, fullStart int
	if token.SourceSpan != nil {
		start = token.SourceSpan.Start.Offset
		end = token.SourceSpan.End.Offset
		fullStart = token.SourceSpan.FullStart.Offset
	}
	expr := v.expressionParser.ParseBinding(
		expression,
		expression_parser.ParseSourceSpan{
			Start:     start,
			End:       end,
			FullStart: fullStart,
		},
		start,
	)
	return expression_parser.Serialize(expr.Ast)
}

var customPhExp = regexp.MustCompile(`//[\s\S]*?i18n[\s\S]*?\(.*?ph\s*=\s*(?:"([^"]*?)"|'([^']*?)').*?\)`)

func extractPlaceholderName(input string) string {
	matches := customPhExp.FindStringSubmatch(input)
	if len(matches) > 0 {
		if matches[1] != "" {
			return matches[1]
		}
		if matches[2] != "" {
			return matches[2]
		}
	}
	return ""
}

var tagToPlaceholderNames = map[string]string{
	"A":     "LINK",
	"B":     "BOLD_TEXT",
	"BR":    "LINE_BREAK",
	"EM":    "EMPHASISED_TEXT",
	"H1":    "HEADING_LEVEL1",
	"H2":    "HEADING_LEVEL2",
	"H3":    "HEADING_LEVEL3",
	"H4":    "HEADING_LEVEL4",
	"H5":    "HEADING_LEVEL5",
	"H6":    "HEADING_LEVEL6",
	"HR":    "HORIZONTAL_RULE",
	"I":     "ITALIC_TEXT",
	"LI":    "LIST_ITEM",
	"LINK":  "MEDIA_LINK",
	"OL":    "ORDERED_LIST",
	"P":     "PARAGRAPH",
	"Q":     "QUOTATION",
	"S":     "STRIKETHROUGH_TEXT",
	"SMALL": "SMALL_TEXT",
	"SUB":   "SUBSTRIPT",
	"SUP":   "SUPERSCRIPT",
	"TBODY": "TABLE_BODY",
	"TD":    "TABLE_CELL",
	"TFOOT": "TABLE_FOOTER",
	"TH":    "TABLE_HEADER_CELL",
	"THEAD": "TABLE_HEADER",
	"TR":    "TABLE_ROW",
	"TT":    "MONOSPACED_TEXT",
	"U":     "UNDERLINED_TEXT",
	"UL":    "UNORDERED_LIST",
}

// PlaceholderRegistry Implementation Placeholder
type PlaceholderRegistry struct {
	placeHolderNameCounts map[string]int
	signatureToName       map[string]string
}

func NewPlaceholderRegistry() *PlaceholderRegistry {
	return &PlaceholderRegistry{
		placeHolderNameCounts: make(map[string]int),
		signatureToName:       make(map[string]string),
	}
}

func (r *PlaceholderRegistry) GetStartTagPlaceholderName(tag string, attrs map[string]string, isVoid bool) string {
	signature := r.hashTag(tag, attrs, isVoid)
	if name, ok := r.signatureToName[signature]; ok {
		return name
	}

	upperTag := strings.ToUpper(tag)
	baseName, ok := tagToPlaceholderNames[upperTag]
	if !ok {
		baseName = "TAG_" + upperTag
	}
	name := ""
	if isVoid {
		name = r.generateUniqueName(baseName)
	} else {
		name = r.generateUniqueName("START_" + baseName)
	}

	r.signatureToName[signature] = name
	return name
}

func (r *PlaceholderRegistry) GetCloseTagPlaceholderName(tag string) string {
	signature := r.hashClosingTag(tag)
	if name, ok := r.signatureToName[signature]; ok {
		return name
	}

	upperTag := strings.ToUpper(tag)
	baseName, ok := tagToPlaceholderNames[upperTag]
	if !ok {
		baseName = "TAG_" + upperTag
	}
	name := r.generateUniqueName("CLOSE_" + baseName)

	r.signatureToName[signature] = name
	return name
}

func (r *PlaceholderRegistry) GetPlaceholderName(name string, content string) string {
	upperName := strings.ToUpper(name)
	signature := fmt.Sprintf("PH: %s=%s", upperName, content)
	if val, ok := r.signatureToName[signature]; ok {
		return val
	}

	uniqueName := r.generateUniqueName(upperName)
	r.signatureToName[signature] = uniqueName
	return uniqueName
}

func (r *PlaceholderRegistry) GetUniquePlaceholder(name string) string {
	return r.generateUniqueName(strings.ToUpper(name))
}

func (r *PlaceholderRegistry) GetStartBlockPlaceholderName(name string, parameters []string) string {
	signature := r.hashBlock(name, parameters)
	if val, ok := r.signatureToName[signature]; ok {
		return val
	}

	placeholder := r.generateUniqueName("START_BLOCK_" + r.toSnakeCase(name))
	r.signatureToName[signature] = placeholder
	return placeholder
}

func (r *PlaceholderRegistry) GetCloseBlockPlaceholderName(name string) string {
	signature := r.hashClosingBlock(name)
	if val, ok := r.signatureToName[signature]; ok {
		return val
	}

	placeholder := r.generateUniqueName("CLOSE_BLOCK_" + r.toSnakeCase(name))
	r.signatureToName[signature] = placeholder
	return placeholder
}

func (r *PlaceholderRegistry) hashTag(tag string, attrs map[string]string, isVoid bool) string {
	start := "<" + tag
	var keys []string
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var strAttrs []string
	for _, k := range keys {
		strAttrs = append(strAttrs, fmt.Sprintf(" %s=%s", k, attrs[k]))
	}
	end := ""
	if isVoid {
		end = "/>"
	} else {
		end = fmt.Sprintf("></%s>", tag)
	}
	return start + strings.Join(strAttrs, "") + end
}

func (r *PlaceholderRegistry) hashClosingTag(tag string) string {
	return r.hashTag("/"+tag, nil, false)
}

func (r *PlaceholderRegistry) hashBlock(name string, parameters []string) string {
	params := ""
	if len(parameters) > 0 {
		sortedParams := make([]string, len(parameters))
		copy(sortedParams, parameters)
		sort.Strings(sortedParams)
		params = fmt.Sprintf(" (%s)", strings.Join(sortedParams, "; "))
	}
	return fmt.Sprintf("@%s%s {}", name, params)
}

func (r *PlaceholderRegistry) hashClosingBlock(name string) string {
	return r.hashBlock("close_"+name, nil)
}

func (r *PlaceholderRegistry) toSnakeCase(name string) string {
	var sb strings.Builder
	for _, c := range strings.ToUpper(name) {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			sb.WriteRune(c)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

func (r *PlaceholderRegistry) generateUniqueName(base string) string {
	count, ok := r.placeHolderNameCounts[base]
	if !ok {
		r.placeHolderNameCounts[base] = 1
		return base
	}
	r.placeHolderNameCounts[base] = count + 1
	return fmt.Sprintf("%s_%d", base, count)
}
