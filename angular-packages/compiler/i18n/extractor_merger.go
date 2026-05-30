package i18n

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

var (
	i18nAttr           = "i18n"
	i18nAttrPrefix     = "i18n-"
	meaningSeparator   = "|"
	idSeparator        = "@@"
	i18nCommentsWarned = false
)

type ExtractionResult struct {
	Messages []*Message
	Errors   []*parse_util.ParseError
}

func ExtractMessages(
	nodes []ml_parser.Node,
	implicitTags []string,
	implicitAttrs map[string][]string,
	preserveSignificantWhitespace bool,
) *ExtractionResult {
	visitor := newVisitor(implicitTags, implicitAttrs, preserveSignificantWhitespace)
	return visitor.extract(nodes)
}

func MergeTranslations(
	nodes []ml_parser.Node,
	translations *TranslationBundle,
	implicitTags []string,
	implicitAttrs map[string][]string,
) *ml_parser.ParseTreeResult {
	visitor := newVisitor(implicitTags, implicitAttrs, true)
	return visitor.merge(nodes, translations)
}

type visitorMode int

const (
	visitorModeExtract visitorMode = iota
	visitorModeMerge
)

type extractorMergerVisitor struct {
	implicitTags                  []string
	implicitAttrs                 map[string][]string
	preserveSignificantWhitespace bool

	depth                  int
	inI18nNode             bool
	inImplicitNode         bool
	inI18nBlock            bool
	blockMeaningAndDesc    string
	blockChildren          []ml_parser.Node
	blockStartDepth        int
	inIcu                  bool
	msgCountAtSectionStart *int
	errors                 []*parse_util.ParseError
	mode                   visitorMode
	messages               []*Message
	translations           *TranslationBundle
	createI18nMessage      I18nMessageFactory
}

func newVisitor(implicitTags []string, implicitAttrs map[string][]string, preserveSignificantWhitespace bool) *extractorMergerVisitor {
	return &extractorMergerVisitor{
		implicitTags:                  implicitTags,
		implicitAttrs:                 implicitAttrs,
		preserveSignificantWhitespace: preserveSignificantWhitespace,
	}
}

func (v *extractorMergerVisitor) extract(nodes []ml_parser.Node) *ExtractionResult {
	v.init(visitorModeExtract)
	for _, node := range nodes {
		node.Visit(v, nil)
	}
	if v.inI18nBlock {
		v.reportError(nodes[len(nodes)-1], "Unclosed block")
	}
	return &ExtractionResult{Messages: v.messages, Errors: v.errors}
}

func (v *extractorMergerVisitor) merge(nodes []ml_parser.Node, translations *TranslationBundle) *ml_parser.ParseTreeResult {
	v.init(visitorModeMerge)
	v.translations = translations

	wrapper := &ml_parser.Element{

		Children: nodes,
	}

	translatedNode := wrapper.Visit(v, nil).(*ml_parser.Element)

	if v.inI18nBlock {
		v.reportError(nodes[len(nodes)-1], "Unclosed block")
	}

	var errs []error
	for _, e := range v.errors {
		errs = append(errs, e)
	}

	return &ml_parser.ParseTreeResult{
		RootNodes: translatedNode.Children,
		Errors:    errs,
	}
}

func (v *extractorMergerVisitor) VisitExpansionCase(icuCase *ml_parser.ExpansionCase, context any) any {
	expression := visitAllForMlNodes(v, icuCase.Expression, context)
	if v.mode == visitorModeMerge {
		return &ml_parser.ExpansionCase{
			Value:      icuCase.Value,
			Expression: expression,
		}
	}
	return nil
}

func (v *extractorMergerVisitor) VisitExpansion(icu *ml_parser.Expansion, context any) any {
	v.mayBeAddBlockChildren(icu)
	wasInIcu := v.inIcu

	if !v.inIcu {
		if v.isInTranslatableSection() {
			v.addMessage([]ml_parser.Node{icu}, "")
		}
		v.inIcu = true
	}

	cases := visitAllExpansionCases(v, icu.Cases, context)

	var result *ml_parser.Expansion = icu
	if v.mode == visitorModeMerge {
		result = &ml_parser.Expansion{
			SwitchValue: icu.SwitchValue,
			Type:        icu.Type,
			Cases:       cases,
		}
		result.SourceSpan = icu.SourceSpan
	}

	v.inIcu = wasInIcu
	return result
}

func (v *extractorMergerVisitor) VisitComment(comment *ml_parser.Comment, context any) any {
	isOpening := isOpeningComment(comment)
	if isOpening && v.isInTranslatableSection() {
		v.reportError(comment, "Could not start a block inside a translatable section")
		return nil
	}

	isClosing := isClosingComment(comment)
	if isClosing && !v.inI18nBlock {
		v.reportError(comment, "Trying to close an unopened block")
		return nil
	}

	if !v.inI18nNode && !v.inIcu {
		if !v.inI18nBlock {
			if isOpening {
				if !i18nCommentsWarned {
					i18nCommentsWarned = true
					// fmt.Println("WARNING: I18n comments are deprecated, use an <ng-container> element instead")
				}
				v.inI18nBlock = true
				v.blockStartDepth = v.depth
				v.blockChildren = nil
				val := ""
				if comment.Value != "" {
					val = comment.Value
				}
				if strings.HasPrefix(val, "i18n:") {
					val = val[5:]
				} else if strings.HasPrefix(val, "i18n") {
					val = val[4:]
				}
				v.blockMeaningAndDesc = strings.TrimSpace(val)
				v.openTranslatableSection(comment)
			}
		} else {
			if isClosing {
				if v.depth == v.blockStartDepth {
					v.closeTranslatableSection(comment, v.blockChildren)
					v.inI18nBlock = false
					message := v.addMessage(v.blockChildren, v.blockMeaningAndDesc)
					nodes := v.translateMessage(comment, message)
					var res []ml_parser.Node
					for _, n := range nodes {
						visited := n.Visit(v, nil)
						if visited != nil {
							res = append(res, visited.(ml_parser.Node))
						}
					}
					return res
				} else {
					v.reportError(comment, "I18N blocks should not cross element boundaries")
					return nil
				}
			}
		}
	}
	return nil
}

func (v *extractorMergerVisitor) VisitText(text *ml_parser.Text, context any) any {
	if v.isInTranslatableSection() {
		v.mayBeAddBlockChildren(text)
	}
	return text
}

func (v *extractorMergerVisitor) VisitElement(el *ml_parser.Element, context any) any {
	return v.visitElementLike(el, context)
}

func (v *extractorMergerVisitor) VisitAttribute(attribute *ml_parser.Attribute, context any) any {
	panic("unreachable code")
}

func (v *extractorMergerVisitor) VisitBlock(block *ml_parser.Block, context any) any {
	v.mayBeAddBlockChildren(block)

	if v.mode == visitorModeExtract {
		for _, child := range block.Children {
			child.Visit(v, context)
		}
	} else if v.mode == visitorModeMerge {
		var childNodes []ml_parser.Node
		for _, child := range block.Children {
			visited := child.Visit(v, context)
			if visited != nil {
				if n, ok := visited.(ml_parser.Node); ok {
					childNodes = append(childNodes, n)
				} else if nodes, ok := visited.([]ml_parser.Node); ok {
					childNodes = append(childNodes, nodes...)
				}
			}
		}

		res := &ml_parser.Block{
			Name:       block.Name,
			Parameters: block.Parameters,
			Children:   childNodes,
		}
		res.SourceSpan = block.SourceSpan
		res.StartSourceSpan = block.StartSourceSpan
		res.EndSourceSpan = block.EndSourceSpan
		return res
	}

	return nil
}

func (v *extractorMergerVisitor) VisitBlockParameter(param *ml_parser.BlockParameter, context any) any {
	if v.mode == visitorModeMerge {
		return param
	}
	return nil
}

func (v *extractorMergerVisitor) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	if v.mode == visitorModeMerge {
		return decl
	}
	return nil
}

func (v *extractorMergerVisitor) VisitComponent(component *ml_parser.Component, context any) any {
	return v.visitElementLike(&component.Element, context)
}

func (v *extractorMergerVisitor) VisitDirective(directive *ml_parser.Directive, context any) any {
	panic("unreachable code")
}

func (v *extractorMergerVisitor) init(mode visitorMode) {
	v.mode = mode
	v.inI18nBlock = false
	v.inI18nNode = false
	v.depth = 0
	v.inIcu = false
	v.msgCountAtSectionStart = nil
	v.errors = nil
	v.messages = nil
	v.inImplicitNode = false
	v.createI18nMessage = CreateI18nMessageFactory(!v.preserveSignificantWhitespace, v.preserveSignificantWhitespace)
}

func (v *extractorMergerVisitor) visitElementLike(node *ml_parser.Element, context any) *ml_parser.Element {
	v.mayBeAddBlockChildren(node)
	v.depth++
	wasInI18nNode := v.inI18nNode
	wasInImplicitNode := v.inImplicitNode
	var childNodes []ml_parser.Node
	var translatedChildNodes []ml_parser.Node

	nodeName := node.Name
	i18nAttr := getI18nAttr(node)
	i18nMeta := ""
	if i18nAttr != nil {
		i18nMeta = i18nAttr.Value
	}

	isImplicit := false
	for _, t := range v.implicitTags {
		if nodeName == t {
			isImplicit = true
			break
		}
	}
	isImplicit = isImplicit && !v.inIcu && !v.isInTranslatableSection()
	isTopLevelImplicit := !wasInImplicitNode && isImplicit
	v.inImplicitNode = wasInImplicitNode || isImplicit

	if !v.isInTranslatableSection() && !v.inIcu {
		if i18nAttr != nil || isTopLevelImplicit {
			v.inI18nNode = true
			message := v.addMessage(node.Children, i18nMeta)
			translatedChildNodes = v.translateMessage(node, message)
		}

		if v.mode == visitorModeExtract {
			isTranslatable := i18nAttr != nil || isTopLevelImplicit
			if isTranslatable {
				v.openTranslatableSection(node)
			}
			for _, child := range node.Children {
				child.Visit(v, nil)
			}
			if isTranslatable {
				v.closeTranslatableSection(node, node.Children)
			}
		}
	} else {
		if i18nAttr != nil || isTopLevelImplicit {
			v.reportError(node, "Could not mark an element as translatable inside a translatable section")
		}

		if v.mode == visitorModeExtract {
			for _, child := range node.Children {
				child.Visit(v, nil)
			}
		}
	}

	if v.mode == visitorModeMerge {
		visitNodes := node.Children
		if translatedChildNodes != nil {
			visitNodes = translatedChildNodes
		}
		for _, child := range visitNodes {
			visited := child.Visit(v, context)
			if visited != nil && !v.isInTranslatableSection() {
				if n, ok := visited.(ml_parser.Node); ok {
					childNodes = append(childNodes, n)
				} else if nodes, ok := visited.([]ml_parser.Node); ok {
					childNodes = append(childNodes, nodes...)
				}
			}
		}
	}

	v.visitAttributesOf(node)

	v.depth--
	v.inI18nNode = wasInI18nNode
	v.inImplicitNode = wasInImplicitNode

	if v.mode == visitorModeMerge {
		var attrs []*ml_parser.Attribute
		for _, attr := range v.translateAttributes(node) {
			copyAttr := attr
			attrs = append(attrs, &copyAttr)
		}

		return &ml_parser.Element{
			BaseNode:        node.BaseNode,
			Name:            node.Name,
			Attrs:           attrs,
			Children:        childNodes,
			EndSourceSpan:   node.EndSourceSpan,
			StartSourceSpan: node.StartSourceSpan,
			IsVoid:          node.IsVoid,
			IsSelfClosing:   node.IsSelfClosing,
		}
	}
	return nil
}

func (v *extractorMergerVisitor) visitAttributesOf(el *ml_parser.Element) {
	explicitAttrNameToValue := make(map[string]string)
	implicitAttrNames := v.implicitAttrs[el.Name]

	for _, attr := range el.Attrs {
		if strings.HasPrefix(attr.Name, i18nAttrPrefix) {
			explicitAttrNameToValue[attr.Name[len(i18nAttrPrefix):]] = attr.Value
		}
	}

	for _, attr := range el.Attrs {
		if val, ok := explicitAttrNameToValue[attr.Name]; ok {
			v.addMessage([]ml_parser.Node{attr}, val)
		} else {
			for _, name := range implicitAttrNames {
				if attr.Name == name {
					v.addMessage([]ml_parser.Node{attr}, "")
					break
				}
			}
		}
	}
}

func (v *extractorMergerVisitor) addMessage(ast []ml_parser.Node, msgMeta string) *Message {
	if len(ast) == 0 || isEmptyAttributeValue(ast) || isPlaceholderOnlyAttributeValue(ast) {
		return nil
	}

	meaning, description, id := parseMessageMeta(msgMeta)
	message := v.createI18nMessage(ast, &meaning, &description, &id, nil)
	v.messages = append(v.messages, message)
	return message
}

func isEmptyAttributeValue(ast []ml_parser.Node) bool {
	if len(ast) != 1 {
		return false
	}
	if attr, ok := ast[0].(*ml_parser.Attribute); ok {
		return strings.TrimSpace(attr.Value) == ""
	}
	return false
}

func isPlaceholderOnlyAttributeValue(ast []ml_parser.Node) bool {
	if len(ast) != 1 {
		return false
	}
	attr, ok := ast[0].(*ml_parser.Attribute)
	if !ok {
		return false
	}

	interpolations := 0
	plainText := ""
	for _, token := range attr.ValueTokens {
		if token.Type == ml_parser.TokenTypeAttrValueInterpolation {
			interpolations++
		} else if token.Type == ml_parser.TokenTypeAttrValueText {
			plainText += strings.TrimSpace(token.Parts[0])
		}
	}

	return interpolations == 1 && plainText == ""
}

func (v *extractorMergerVisitor) translateMessage(el ml_parser.Node, message *Message) []ml_parser.Node {
	if message != nil && v.mode == visitorModeMerge {
		nodes := v.translations.Get(message)
		if nodes != nil {
			return nodes
		}
		v.reportError(el, fmt.Sprintf("Translation unavailable for message id=\"%s\"", v.translations.Digest(message)))
	}
	return nil
}

func (v *extractorMergerVisitor) translateAttributes(node *ml_parser.Element) []ml_parser.Attribute {
	type metaData struct {
		meaning     string
		description string
		id          string
	}
	i18nParsedMessageMeta := make(map[string]metaData)
	var translatedAttributes []ml_parser.Attribute

	for _, attr := range node.Attrs {
		if strings.HasPrefix(attr.Name, i18nAttrPrefix) {
			meaning, description, id := parseMessageMeta(attr.Value)
			i18nParsedMessageMeta[attr.Name[len(i18nAttrPrefix):]] = metaData{meaning, description, id}
		}
	}

	for _, attr := range node.Attrs {
		if attr.Name == i18nAttr || strings.HasPrefix(attr.Name, i18nAttrPrefix) {
			continue
		}

		if attr.Value != "" {
			if meta, ok := i18nParsedMessageMeta[attr.Name]; ok {
				message := v.createI18nMessage([]ml_parser.Node{attr}, &meta.meaning, &meta.description, &meta.id, nil)
				nodes := v.translations.Get(message)
				if nodes != nil {
					if len(nodes) == 0 {
						translatedAttributes = append(translatedAttributes, ml_parser.Attribute{
							BaseNode:    attr.BaseNode,
							Name:        attr.Name,
							Value:       "",
							KeySpan:     attr.KeySpan,
							ValueSpan:   attr.ValueSpan,
							ValueTokens: attr.ValueTokens,
						})
					} else if textNode, ok := nodes[0].(*ml_parser.Text); ok {
						translatedAttributes = append(translatedAttributes, ml_parser.Attribute{
							BaseNode:    attr.BaseNode,
							Name:        attr.Name,
							Value:       textNode.Value,
							KeySpan:     attr.KeySpan,
							ValueSpan:   attr.ValueSpan,
							ValueTokens: attr.ValueTokens,
						})
					} else {
						id := meta.id
						if id == "" {
							id = v.translations.Digest(message)
						}
						v.reportError(node, fmt.Sprintf("Unexpected translation for attribute \"%s\" (id=\"%s\")", attr.Name, id))
					}
				} else {
					id := meta.id
					if id == "" {
						id = v.translations.Digest(message)
					}
					v.reportError(node, fmt.Sprintf("Translation unavailable for attribute \"%s\" (id=\"%s\")", attr.Name, id))
				}
				continue
			}
		}
		translatedAttributes = append(translatedAttributes, *attr)
	}

	return translatedAttributes
}

func (v *extractorMergerVisitor) translateDirectives(node *ml_parser.Element) []ml_parser.Directive {
	return nil
}

func (v *extractorMergerVisitor) mayBeAddBlockChildren(node ml_parser.Node) {
	if v.inI18nBlock && !v.inIcu && v.depth == v.blockStartDepth {
		v.blockChildren = append(v.blockChildren, node)
	}
}

func (v *extractorMergerVisitor) openTranslatableSection(node ml_parser.Node) {
	if v.isInTranslatableSection() {
		v.reportError(node, "Unexpected section start")
	} else {
		count := len(v.messages)
		v.msgCountAtSectionStart = &count
	}
}

func (v *extractorMergerVisitor) isInTranslatableSection() bool {
	return v.msgCountAtSectionStart != nil
}

func (v *extractorMergerVisitor) closeTranslatableSection(node ml_parser.Node, directChildren []ml_parser.Node) {
	if !v.isInTranslatableSection() {
		v.reportError(node, "Unexpected section end")
		return
	}

	startIndex := *v.msgCountAtSectionStart
	significantChildren := 0
	for _, n := range directChildren {
		if _, isComment := n.(*ml_parser.Comment); !isComment {
			significantChildren++
		}
	}

	if significantChildren == 1 {
		for i := len(v.messages) - 1; i >= startIndex; i-- {
			ast := v.messages[i].Nodes
			if len(ast) == 1 {
				if _, isText := ast[0].(*Text); isText {
					continue
				}
			}
			v.messages = append(v.messages[:i], v.messages[i+1:]...)
			break
		}
	}

	v.msgCountAtSectionStart = nil
}

func (v *extractorMergerVisitor) reportError(node ml_parser.Node, msg string) {
	var span *parse_util.ParseSourceSpan
	if node != nil {
		span = node.GetSourceSpan()
	}
	v.errors = append(v.errors, parse_util.NewParseError(span, msg, nil, nil))
}

func isOpeningComment(n ml_parser.Node) bool {
	if comment, ok := n.(*ml_parser.Comment); ok {
		return comment.Value != "" && strings.HasPrefix(comment.Value, "i18n")
	}
	return false
}

func isClosingComment(n ml_parser.Node) bool {
	if comment, ok := n.(*ml_parser.Comment); ok {
		return comment.Value != "" && comment.Value == "/i18n"
	}
	return false
}

func getI18nAttr(p *ml_parser.Element) *ml_parser.Attribute {
	for _, attr := range p.Attrs {
		if attr.Name == i18nAttr {
			return attr
		}
	}
	return nil
}

func parseMessageMeta(i18n string) (meaning string, description string, id string) {
	if i18n == "" {
		return "", "", ""
	}

	idIndex := strings.Index(i18n, idSeparator)
	descIndex := strings.Index(i18n, meaningSeparator)

	meaningAndDesc := i18n
	if idIndex > -1 {
		meaningAndDesc = i18n[:idIndex]
		id = i18n[idIndex+2:]
	}

	if descIndex > -1 && (idIndex == -1 || descIndex < idIndex) {
		meaning = meaningAndDesc[:descIndex]
		description = meaningAndDesc[descIndex+1:]
	} else {
		meaning = ""
		description = meaningAndDesc
	}

	return meaning, description, strings.TrimSpace(id)
}

func visitAllForMlNodes(visitor ml_parser.Visitor, nodes []ml_parser.Node, context any) []ml_parser.Node {
	var result []ml_parser.Node
	for _, node := range nodes {
		if node == nil {
			continue
		}
		res := node.Visit(visitor, context)
		if res != nil {
			if n, ok := res.(ml_parser.Node); ok {
				result = append(result, n)
			}
		}
	}
	return result
}
func visitAllExpansionCases(visitor ml_parser.Visitor, cases []*ml_parser.ExpansionCase, context any) []*ml_parser.ExpansionCase {
	var result []*ml_parser.ExpansionCase
	for _, icuCase := range cases {
		if icuCase == nil {
			continue
		}
		res := icuCase.Visit(visitor, context)
		if res != nil {
			if c, ok := res.(*ml_parser.ExpansionCase); ok {
				result = append(result, c)
			}
		}
	}
	return result
}
