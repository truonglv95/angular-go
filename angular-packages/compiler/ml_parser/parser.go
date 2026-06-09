package ml_parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
)

type TreeError struct {
	*parse_util.ParseError
	ElementName *string
}

func NewTreeError(elementName *string, span *parse_util.ParseSourceSpan, msg string) *TreeError {
	return &TreeError{
		ParseError:  parse_util.NewParseError(span, msg, nil, nil),
		ElementName: elementName,
	}
}

type ParseTreeResult struct {
	RootNodes []Node
	Errors    []error
}

type Parser struct {
	GetTagDefinition func(tagName string) tags.TagDefinition
}

func NewParser(getTagDefinition func(tagName string) tags.TagDefinition) *Parser {
	return &Parser{GetTagDefinition: getTagDefinition}
}

func (p *Parser) Parse(source string, url string, options *TokenizeOptions) *ParseTreeResult {
	tokenizeResult := Tokenize(source, url, p.GetTagDefinition, options)
	builder := newTreeBuilder(tokenizeResult.Tokens, p.GetTagDefinition)
	builder.build()

	var errors []error
	for _, err := range tokenizeResult.Errors {
		errors = append(errors, err)
	}
	for _, err := range builder.errors {
		errors = append(errors, err)
	}

	return &ParseTreeResult{
		RootNodes: builder.rootNodes,
		Errors:    errors,
	}
}

type nodeContainer interface {
	getChildren() []Node
	setChildren([]Node)
	getName() string
	getSourceSpan() *parse_util.ParseSourceSpan
	setEndSourceSpan(*parse_util.ParseSourceSpan)
	isBlock() bool
	isElement() bool
	isComponent() bool
}

// Implement nodeContainer for Element, Block, Component in ast.go

type treeBuilder struct {
	index                 int
	peek                  *Token
	containerStack        []nodeContainer
	rootNodes             []Node
	errors                []*TreeError
	tokens                []Token
	tagDefinitionResolver func(string) tags.TagDefinition
}

func newTreeBuilder(tokens []Token, tagDefinitionResolver func(string) tags.TagDefinition) *treeBuilder {
	tb := &treeBuilder{
		index:                 -1,
		tokens:                tokens,
		tagDefinitionResolver: tagDefinitionResolver,
		containerStack:        []nodeContainer{},
		rootNodes:             []Node{},
		errors:                []*TreeError{},
	}
	tb.advance()
	return tb
}

func (b *treeBuilder) build() {
	for b.peek.Type != TokenTypeEOF {
		if b.peek.Type == TokenTypeTagOpenStart || b.peek.Type == TokenTypeIncompleteTagOpen {
			b.consumeElementStartTag(b.advance())
		} else if b.peek.Type == TokenTypeTagClose {
			b.consumeElementEndTag(b.advance())
		} else if b.peek.Type == TokenTypeCdataStart {
			b.closeVoidElement()
			b.consumeCdata(b.advance())
		} else if b.peek.Type == TokenTypeCommentStart {
			b.closeVoidElement()
			b.consumeComment(b.advance())
		} else if b.peek.Type == TokenTypeText || b.peek.Type == TokenTypeRawText || b.peek.Type == TokenTypeEscapableRawText {
			b.closeVoidElement()
			b.consumeText(b.advance())
		} else if b.peek.Type == TokenTypeExpansionFormStart {
			b.consumeExpansion(b.advance())
		} else if b.peek.Type == TokenTypeBlockOpenStart {
			b.closeVoidElement()
			b.consumeBlockOpen(b.advance())
		} else if b.peek.Type == TokenTypeBlockClose {
			b.closeVoidElement()
			b.consumeBlockClose(b.advance())
		} else if b.peek.Type == TokenTypeIncompleteBlockOpen {
			b.closeVoidElement()
			b.consumeIncompleteBlock(b.advance())
		} else if b.peek.Type == TokenTypeLetStart {
			b.closeVoidElement()
			b.consumeLet(b.advance())
		} else if b.peek.Type == TokenTypeIncompleteLet {
			b.closeVoidElement()
			b.consumeIncompleteLet(b.advance())
		} else if b.peek.Type == TokenTypeComponentOpenStart || b.peek.Type == TokenTypeIncompleteComponentOpen {
			b.consumeComponentStartTag(b.advance())
		} else if b.peek.Type == TokenTypeComponentClose {
			b.consumeComponentEndTag(b.advance())
		} else {
			b.advance()
		}
	}

	for _, leftoverContainer := range b.containerStack {
		if leftoverContainer.isBlock() {
			name := leftoverContainer.getName()
			b.errors = append(b.errors, NewTreeError(
				&name,
				leftoverContainer.getSourceSpan(),
				"Unclosed block \""+name+"\"",
			))
		}
	}
}

func (b *treeBuilder) advance() *Token {
	prev := b.peek
	if b.index < len(b.tokens)-1 {
		b.index++
	}
	b.peek = &b.tokens[b.index]
	return prev
}

func (b *treeBuilder) advanceIf(tokenType TokenType) *Token {
	if b.peek.Type == tokenType {
		return b.advance()
	}
	return nil
}

func (b *treeBuilder) getContainer() nodeContainer {
	if len(b.containerStack) > 0 {
		return b.containerStack[len(b.containerStack)-1]
	}
	return nil
}

func (b *treeBuilder) addToParent(node Node) {
	parent := b.getContainer()
	if parent != nil {
		parent.setChildren(append(parent.getChildren(), node))
	} else {
		b.rootNodes = append(b.rootNodes, node)
	}
}

func (b *treeBuilder) consumeCdata(startToken *Token) {
	b.consumeText(b.advance())
	b.advanceIf(TokenTypeCdataEnd)
}

func (b *treeBuilder) consumeComment(token *Token) {
	text := b.advanceIf(TokenTypeRawText)
	endToken := b.advanceIf(TokenTypeCommentEnd)
	value := ""
	if text != nil {
		value = strings.TrimSpace(text.Parts[0])
	}

	sourceSpan := token.SourceSpan
	if endToken != nil {
		sourceSpan = parse_util.NewParseSourceSpan(token.SourceSpan.Start, endToken.SourceSpan.End, token.SourceSpan.FullStart, nil)
	}
	b.addToParent(&Comment{
		BaseNode: BaseNode{SourceSpan: sourceSpan},
		Value:    value,
	})
}

func (b *treeBuilder) consumeExpansion(token *Token) {
	switchValue := b.advance()
	typeToken := b.advance()
	cases := []*ExpansionCase{}

	for b.peek.Type == TokenTypeExpansionCaseValue {
		expCase := b.parseExpansionCase()
		if expCase == nil {
			return
		}
		cases = append(cases, expCase)
	}

	if b.peek.Type != TokenTypeExpansionFormEnd {
		b.errors = append(b.errors, NewTreeError(nil, b.peek.SourceSpan, "Invalid ICU message. Missing '}'."))
		return
	}

	sourceSpan := parse_util.NewParseSourceSpan(token.SourceSpan.Start, b.peek.SourceSpan.End, token.SourceSpan.FullStart, nil)
	b.addToParent(&Expansion{
		BaseNode:    BaseNode{SourceSpan: sourceSpan},
		SwitchValue: switchValue.Parts[0],
		Type:        typeToken.Parts[0],
		Cases:       cases,
	})
	b.advance()
}

func (b *treeBuilder) parseExpansionCase() *ExpansionCase {
	value := b.advance()

	if b.peek.Type != TokenTypeExpansionCaseExpStart {
		b.errors = append(b.errors, NewTreeError(nil, b.peek.SourceSpan, "Invalid ICU message. Missing '{'."))
		return nil
	}

	start := b.advance()
	exp := b.collectExpansionExpTokens(start)
	if exp == nil {
		return nil
	}

	end := b.advance()
	exp = append(exp, Token{Type: TokenTypeEOF, Parts: []string{}, SourceSpan: end.SourceSpan})

	expansionCaseParser := newTreeBuilder(exp, b.tagDefinitionResolver)
	expansionCaseParser.build()
	if len(expansionCaseParser.errors) > 0 {
		b.errors = append(b.errors, expansionCaseParser.errors...)
		return nil
	}

	sourceSpan := parse_util.NewParseSourceSpan(value.SourceSpan.Start, end.SourceSpan.End, value.SourceSpan.FullStart, nil)
	return &ExpansionCase{
		BaseNode:   BaseNode{SourceSpan: sourceSpan},
		Value:      value.Parts[0],
		Expression: expansionCaseParser.rootNodes,
	}
}

func (b *treeBuilder) collectExpansionExpTokens(start *Token) []Token {
	exp := []Token{}
	expansionFormStack := []TokenType{TokenTypeExpansionCaseExpStart}

	for {
		if b.peek.Type == TokenTypeExpansionFormStart || b.peek.Type == TokenTypeExpansionCaseExpStart {
			expansionFormStack = append(expansionFormStack, b.peek.Type)
		}

		if b.peek.Type == TokenTypeExpansionCaseExpEnd {
			if len(expansionFormStack) > 0 && expansionFormStack[len(expansionFormStack)-1] == TokenTypeExpansionCaseExpStart {
				expansionFormStack = expansionFormStack[:len(expansionFormStack)-1]
				if len(expansionFormStack) == 0 {
					return exp
				}
			} else {
				b.errors = append(b.errors, NewTreeError(nil, start.SourceSpan, "Invalid ICU message. Missing '}'."))
				return nil
			}
		}

		if b.peek.Type == TokenTypeExpansionFormEnd {
			if len(expansionFormStack) > 0 && expansionFormStack[len(expansionFormStack)-1] == TokenTypeExpansionFormStart {
				expansionFormStack = expansionFormStack[:len(expansionFormStack)-1]
			} else {
				b.errors = append(b.errors, NewTreeError(nil, start.SourceSpan, "Invalid ICU message. Missing '}'."))
				return nil
			}
		}

		if b.peek.Type == TokenTypeEOF {
			b.errors = append(b.errors, NewTreeError(nil, start.SourceSpan, "Invalid ICU message. Missing '}'."))
			return nil
		}

		exp = append(exp, *b.advance())
	}
}

var entityRegex = regexp.MustCompile(`&([^;]+);`)

func decodeEntity(match string, entity string) string {
	if val, ok := NAMED_ENTITIES[entity]; ok {
		return val
	}
	if strings.HasPrefix(strings.ToLower(entity), "#x") {
		code, err := strconv.ParseInt(entity[2:], 16, 32)
		if err == nil {
			return string(rune(code))
		}
	} else if strings.HasPrefix(entity, "#") {
		code, err := strconv.ParseInt(entity[1:], 10, 32)
		if err == nil {
			return string(rune(code))
		}
	}
	return match
}

func (b *treeBuilder) consumeText(token *Token) {
	tokens := []Token{*token}
	startSpan := token.SourceSpan
	text := token.Parts[0]

	if len(text) > 0 && text[0] == '\n' {
		parent := b.getContainer()
		if parent != nil && len(parent.getChildren()) == 0 {
			tagDef := b.getTagDefinition(parent)
			if tagDef != nil && tagDef.IgnoreFirstLf() {
				text = text[1:]
				tokens[0] = Token{Type: token.Type, SourceSpan: token.SourceSpan, Parts: []string{text}}
			}
		}
	}

	for b.peek.Type == TokenTypeInterpolation || b.peek.Type == TokenTypeText || b.peek.Type == TokenTypeEncodedEntity {
		token = b.advance()
		tokens = append(tokens, *token)
		if token.Type == TokenTypeInterpolation {
			text += strings.Join(token.Parts, "") // regex replace not fully matched here
		} else if token.Type == TokenTypeEncodedEntity {
			text += token.Parts[0]
		} else {
			text += strings.Join(token.Parts, "")
		}
	}

	if len(text) > 0 {
		endSpan := token.SourceSpan
		b.addToParent(&Text{
			BaseNode: BaseNode{SourceSpan: parse_util.NewParseSourceSpan(startSpan.Start, endSpan.End, startSpan.FullStart, startSpan.Details)},
			Value:    text,
			Tokens:   tokens,
		})
	}
}

func (b *treeBuilder) closeVoidElement() {
	el := b.getContainer()
	if el != nil {
		tagDef := b.getTagDefinition(el)
		if tagDef != nil && tagDef.IsVoid() {
			b.containerStack = b.containerStack[:len(b.containerStack)-1]
		}
	}
}

func (b *treeBuilder) consumeDirectivesAndAttributes(directives *[]*Directive, attributes *[]*Attribute) {
	for b.peek.Type == TokenTypeAttrName || b.peek.Type == TokenTypeDirectiveName {
		if b.peek.Type == TokenTypeDirectiveName {
			dirToken := b.advance()
			dirName := dirToken.Parts[0]
			var dirAttrs []*Attribute

			startSpan := dirToken.SourceSpan
			endSpan := dirToken.SourceSpan

			if b.peek.Type == TokenTypeDirectiveOpen {
				b.advance() // Consume TokenTypeDirectiveOpen
				for b.peek.Type == TokenTypeAttrName {
					attrToken := b.advance()
					dirAttrs = append(dirAttrs, b.consumeAttr(attrToken))
				}
				if b.peek.Type == TokenTypeDirectiveClose {
					closeToken := b.advance() // Consume TokenTypeDirectiveClose
					endSpan = closeToken.SourceSpan
				}
			}

			span := parse_util.NewParseSourceSpan(startSpan.Start, endSpan.End, startSpan.FullStart, nil)
			*directives = append(*directives, &Directive{
				Name:            dirName,
				Attrs:           dirAttrs,
				SourceSpan:      *span,
				StartSourceSpan: startSpan,
				EndSourceSpan:   endSpan,
			})
		} else {
			attrName := b.advance()
			*attributes = append(*attributes, b.consumeAttr(attrName))
		}
	}
}

func (b *treeBuilder) consumeElementStartTag(startTagToken *Token) {
	attrs := []*Attribute{}
	var directives []*Directive
	b.consumeDirectivesAndAttributes(&directives, &attrs)

	fullName := b.getElementFullName(startTagToken, b.getClosestElementLikeParent())
	tagDef := b.tagDefinitionResolver(fullName)
	selfClosing := false

	if b.peek.Type == TokenTypeTagOpenEndVoid {
		b.advance()
		selfClosing = true
		prefix := tags.GetNsPrefix(&fullName)
		if tagDef == nil || (!tagDef.CanSelfClose() && prefix == nil && !tagDef.IsVoid()) {
			b.errors = append(b.errors, NewTreeError(
				&fullName,
				startTagToken.SourceSpan,
				"Only void, custom and foreign elements can be self closed \""+startTagToken.Parts[1]+"\"",
			))
		}
	} else if b.peek.Type == TokenTypeTagOpenEnd {
		b.advance()
		selfClosing = false
	}

	end := b.peek.SourceSpan.FullStart
	span := parse_util.NewParseSourceSpan(startTagToken.SourceSpan.Start, end, startTagToken.SourceSpan.FullStart, nil)
	startSpan := parse_util.NewParseSourceSpan(startTagToken.SourceSpan.Start, end, startTagToken.SourceSpan.FullStart, nil)

	isVoid := false
	if tagDef != nil {
		isVoid = tagDef.IsVoid()
	}

	el := &Element{
		BaseNode:        BaseNode{SourceSpan: span},
		Name:            fullName,
		Attrs:           attrs,
		Directives:      directives,
		Children:        []Node{},
		StartSourceSpan: startSpan,
		IsVoid:          isVoid,
		IsSelfClosing:   selfClosing,
	}

	parent := b.getContainer()
	isClosedByChild := false
	if parent != nil {
		pDef := b.getTagDefinition(parent)
		if pDef != nil {
			isClosedByChild = pDef.IsClosedByChild(el.Name)
		}
	}
	b.pushContainer(el, isClosedByChild)

	if selfClosing {
		b.popContainer(&fullName, true, span)
	} else if startTagToken.Type == TokenTypeIncompleteTagOpen {
		b.popContainer(&fullName, true, nil)
		b.errors = append(b.errors, NewTreeError(&fullName, span, "Opening tag \""+fullName+"\" not terminated."))
	}
}

func (b *treeBuilder) consumeComponentStartTag(startTagToken *Token) {
	attrs := []*Attribute{}
	var directives []*Directive
	b.consumeDirectivesAndAttributes(&directives, &attrs)

	compName := startTagToken.Parts[0]
	var tagName string
	if len(startTagToken.Parts) > 2 && startTagToken.Parts[2] != "" {
		tagName = startTagToken.Parts[2]
	} else {
		tagName = "span" // Fallback tag name
	}

	fullName := b.getComponentFullName(startTagToken, b.getClosestElementLikeParent())
	tagDef := b.tagDefinitionResolver(tagName)
	selfClosing := false

	if b.peek.Type == TokenTypeComponentOpenEndVoid {
		b.advance()
		selfClosing = true
	} else if b.peek.Type == TokenTypeComponentOpenEnd {
		b.advance()
		selfClosing = false
	}

	end := b.peek.SourceSpan.FullStart
	span := parse_util.NewParseSourceSpan(startTagToken.SourceSpan.Start, end, startTagToken.SourceSpan.FullStart, nil)
	startSpan := parse_util.NewParseSourceSpan(startTagToken.SourceSpan.Start, end, startTagToken.SourceSpan.FullStart, nil)

	isVoid := false
	if tagDef != nil {
		isVoid = tagDef.IsVoid()
	}

	comp := &Component{
		Element: Element{
			BaseNode:        BaseNode{SourceSpan: span},
			Name:            fullName,
			Attrs:           attrs,
			Directives:      directives,
			Children:        []Node{},
			StartSourceSpan: startSpan,
			IsVoid:          isVoid,
			IsSelfClosing:   selfClosing,
		},
		ComponentName: compName,
		TagName:       tagName,
		FullName:      fullName,
	}

	parent := b.getContainer()
	isClosedByChild := false
	if parent != nil {
		pDef := b.getTagDefinition(parent)
		if pDef != nil {
			isClosedByChild = pDef.IsClosedByChild(comp.TagName)
		}
	}
	b.pushContainer(comp, isClosedByChild)

	if selfClosing {
		b.popContainer(&fullName, true, span)
	} else if startTagToken.Type == TokenTypeIncompleteComponentOpen {
		b.popContainer(&fullName, true, nil)
		b.errors = append(b.errors, NewTreeError(&fullName, span, "Opening tag \""+fullName+"\" not terminated."))
	}
}

func (b *treeBuilder) consumeComponentEndTag(endToken *Token) {
	fullName := b.getComponentFullName(endToken, b.getClosestElementLikeParent())
	if !b.popContainer(&fullName, true, endToken.SourceSpan) {
		errMsg := "Unexpected closing tag \"" + fullName + "\". It may happen when the tag has already been closed by another tag."
		b.errors = append(b.errors, NewTreeError(&fullName, endToken.SourceSpan, errMsg))
	}
}

func (b *treeBuilder) pushContainer(node nodeContainer, isClosedByChild bool) {
	if isClosedByChild {
		b.containerStack = b.containerStack[:len(b.containerStack)-1]
	}
	b.addToParent(node.(Node))
	b.containerStack = append(b.containerStack, node)
}

func (b *treeBuilder) consumeElementEndTag(endTagToken *Token) {
	fullName := b.getElementFullName(endTagToken, b.getClosestElementLikeParent())

	tagDef := b.tagDefinitionResolver(fullName)
	if tagDef != nil && tagDef.IsVoid() {
		b.errors = append(b.errors, NewTreeError(
			&fullName,
			endTagToken.SourceSpan,
			"Void elements do not have end tags \""+endTagToken.Parts[1]+"\"",
		))
	} else if !b.popContainer(&fullName, true, endTagToken.SourceSpan) {
		errMsg := "Unexpected closing tag \"" + fullName + "\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags"
		b.errors = append(b.errors, NewTreeError(&fullName, endTagToken.SourceSpan, errMsg))
	}
}

func (b *treeBuilder) popContainer(expectedName *string, isElement bool, endSourceSpan *parse_util.ParseSourceSpan) bool {
	unexpectedCloseTagDetected := false
	for stackIndex := len(b.containerStack) - 1; stackIndex >= 0; stackIndex-- {
		node := b.containerStack[stackIndex]
		nodeName := node.getName()

		if (expectedName == nil || nodeName == *expectedName) && node.isElement() == isElement {
			node.setEndSourceSpan(endSourceSpan)
			b.containerStack = b.containerStack[:stackIndex]
			return !unexpectedCloseTagDetected
		}

		nodeTagDef := b.getTagDefinition(node)
		if node.isBlock() || nodeTagDef == nil || !nodeTagDef.ClosedByParent() {
			unexpectedCloseTagDetected = true
		}
	}
	return false
}

func (b *treeBuilder) consumeAttr(attrName *Token) *Attribute {
	fullName := tags.MergeNsAndName(attrName.Parts[0], attrName.Parts[1])
	attrEnd := attrName.SourceSpan.End

	if b.peek.Type == TokenTypeAttrQuote {
		b.advance()
	}

	value := ""
	valueTokens := []Token{}
	var valueStartSpan *parse_util.ParseSourceSpan = nil
	var valueEnd *parse_util.ParseLocation = nil

	if b.peek.Type == TokenTypeAttrValueText {
		valueStartSpan = b.peek.SourceSpan
		valueEnd = b.peek.SourceSpan.End
		for b.peek.Type == TokenTypeAttrValueText || b.peek.Type == TokenTypeAttrValueInterpolation || b.peek.Type == TokenTypeEncodedEntity {
			valueToken := b.advance()
			valueTokens = append(valueTokens, *valueToken)
			if valueToken.Type == TokenTypeAttrValueInterpolation {
				interpVal := strings.Join(valueToken.Parts, "")
				interpVal = entityRegex.ReplaceAllStringFunc(interpVal, func(m string) string {
					sub := entityRegex.FindStringSubmatch(m)
					if len(sub) > 1 {
						return decodeEntity(m, sub[1])
					}
					return m
				})
				value += interpVal
			} else if valueToken.Type == TokenTypeEncodedEntity {
				value += valueToken.Parts[0]
			} else {
				value += strings.Join(valueToken.Parts, "")
			}
			valueEnd = valueToken.SourceSpan.End
			attrEnd = valueToken.SourceSpan.End
		}
	}

	if b.peek.Type == TokenTypeAttrQuote {
		quoteToken := b.advance()
		attrEnd = quoteToken.SourceSpan.End
	}

	var valueSpan *parse_util.ParseSourceSpan
	if valueStartSpan != nil && valueEnd != nil {
		valueSpan = parse_util.NewParseSourceSpan(valueStartSpan.Start, valueEnd, valueStartSpan.FullStart, nil)
	}

	return &Attribute{
		BaseNode:    BaseNode{SourceSpan: parse_util.NewParseSourceSpan(attrName.SourceSpan.Start, attrEnd, attrName.SourceSpan.FullStart, nil)},
		Name:        fullName,
		Value:       value,
		KeySpan:     attrName.SourceSpan,
		ValueSpan:   valueSpan,
		ValueTokens: valueTokens,
	}
}

func (b *treeBuilder) consumeBlockOpen(token *Token) {
	parameters := []*BlockParameter{}

	for b.peek.Type == TokenTypeBlockParameter {
		paramToken := b.advance()
		parameters = append(parameters, &BlockParameter{
			BaseNode:   BaseNode{SourceSpan: paramToken.SourceSpan},
			Expression: paramToken.Parts[0],
		})
	}

	if b.peek.Type == TokenTypeBlockOpenEnd {
		b.advance()
	}

	end := b.peek.SourceSpan.FullStart
	span := parse_util.NewParseSourceSpan(token.SourceSpan.Start, end, token.SourceSpan.FullStart, nil)
	startSpan := parse_util.NewParseSourceSpan(token.SourceSpan.Start, end, token.SourceSpan.FullStart, nil)

	block := &Block{
		BaseNode:        BaseNode{SourceSpan: span},
		Name:            token.Parts[0],
		Parameters:      parameters,
		Children:        []Node{},
		StartSourceSpan: startSpan,
	}
	b.pushContainer(block, false)
}

func (b *treeBuilder) consumeBlockClose(token *Token) {
	initialStackLength := len(b.containerStack)
	topNode := b.containerStack[initialStackLength-1]

	if !b.popContainer(nil, false, token.SourceSpan) {
		if len(b.containerStack) < initialStackLength {
			nodeName := topNode.getName()
			b.errors = append(b.errors, NewTreeError(
				nil,
				token.SourceSpan,
				"Unexpected closing block. The block may have been closed earlier. Did you forget to close the <"+nodeName+"> element? If you meant to write the `}` character, you should use the \"&#125;\" HTML entity instead.",
			))
			return
		}

		b.errors = append(b.errors, NewTreeError(
			nil,
			token.SourceSpan,
			"Unexpected closing block. The block may have been closed earlier. If you meant to write the `}` character, you should use the \"&#125;\" HTML entity instead.",
		))
	}
}

func (b *treeBuilder) consumeIncompleteBlock(token *Token) {
	b.consumeBlockOpen(token) // Just open it
}

func (b *treeBuilder) consumeLet(token *Token) {
	name := ""
	if len(token.Parts) > 0 {
		name = token.Parts[0]
	}

	var valToken *Token
	if b.peek.Type == TokenTypeLetValue {
		valToken = b.advance()
	}

	value := ""
	var valueSpan *parse_util.ParseSourceSpan
	if valToken != nil && len(valToken.Parts) > 0 {
		value = valToken.Parts[0]
		valueSpan = valToken.SourceSpan
	}

	var endToken *Token
	if b.peek.Type == TokenTypeLetEnd {
		endToken = b.advance()
	}

	sourceSpan := token.SourceSpan
	if endToken != nil {
		sourceSpan = parse_util.NewParseSourceSpan(token.SourceSpan.Start, endToken.SourceSpan.End, token.SourceSpan.FullStart, nil)
	} else if valToken != nil {
		sourceSpan = parse_util.NewParseSourceSpan(token.SourceSpan.Start, valToken.SourceSpan.End, token.SourceSpan.FullStart, nil)
	}

	var nameSpan parse_util.ParseSourceSpan
	if token.SourceSpan != nil {
		nameSpan = *token.SourceSpan
	}

	var span parse_util.ParseSourceSpan
	if sourceSpan != nil {
		span = *sourceSpan
	}

	letNode := &LetDeclaration{
		Name:       name,
		Value:      value,
		SourceSpan: span,
		NameSpan:   &nameSpan,
		ValueSpan:  valueSpan,
	}

	b.addToParent(letNode)
}

func (b *treeBuilder) consumeIncompleteLet(token *Token) {
	b.errors = append(b.errors, NewTreeError(
		nil,
		token.SourceSpan,
		"Incomplete @let declaration",
	))
}

func (b *treeBuilder) getClosestElementLikeParent() nodeContainer {
	for i := len(b.containerStack) - 1; i >= 0; i-- {
		if b.containerStack[i].isElement() || b.containerStack[i].isComponent() {
			return b.containerStack[i]
		}
	}
	return nil
}

func (b *treeBuilder) getElementFullName(token *Token, parent nodeContainer) string {
	var prefix, tagName string
	if token.Type == TokenTypeComponentOpenStart || token.Type == TokenTypeComponentClose || token.Type == TokenTypeIncompleteComponentOpen {
		prefix = token.Parts[1]
		tagName = token.Parts[2]
		if tagName == "" {
			tagName = "span" // Fallback tag name
		}
	} else {
		prefix = token.Parts[0]
		tagName = token.Parts[1]
	}

	if prefix == "" {
		tagDef := b.tagDefinitionResolver(tagName)
		if tagDef != nil {
			imp := tagDef.ImplicitNamespacePrefix()
			if imp != nil {
				prefix = *imp
			}
		}
	}

	if prefix == "" && parent != nil && parent.isElement() {
		parentName := parent.getName()
		_, parentTagName, _ := tags.SplitNsName(parentName, false)
		parentTagDef := b.tagDefinitionResolver(parentTagName)
		if parentTagDef == nil || !parentTagDef.PreventNamespaceInheritance() {
			parentPrefix := tags.GetNsPrefix(&parentName)
			if parentPrefix != nil {
				prefix = *parentPrefix
			}
		}
	}

	if prefix != "" {
		return tags.MergeNsAndName(prefix, tagName)
	}
	return tagName
}

func (b *treeBuilder) getComponentFullName(token *Token, parent nodeContainer) string {
	compName := token.Parts[0]
	var prefix string
	var tagName string
	if len(token.Parts) > 1 {
		prefix = token.Parts[1]
	}
	if len(token.Parts) > 2 {
		tagName = token.Parts[2]
	}

	if tagName == "" {
		return compName
	}

	if prefix == "" {
		tagDef := b.tagDefinitionResolver(tagName)
		if tagDef != nil {
			imp := tagDef.ImplicitNamespacePrefix()
			if imp != nil {
				prefix = *imp
			}
		}
	}

	if prefix == "" && parent != nil && parent.isElement() {
		parentName := parent.getName()
		_, parentTagName, _ := tags.SplitNsName(parentName, false)
		parentTagDef := b.tagDefinitionResolver(parentTagName)
		if parentTagDef == nil || !parentTagDef.PreventNamespaceInheritance() {
			parentPrefix := tags.GetNsPrefix(&parentName)
			if parentPrefix != nil {
				prefix = *parentPrefix
			}
		}
	}

	var nsTagName string
	if prefix == "" {
		nsTagName = tagName
	} else {
		nsTagName = prefix + ":" + tagName
	}

	if strings.HasPrefix(nsTagName, ":") {
		return compName + nsTagName
	}
	return compName + ":" + nsTagName
}

func (b *treeBuilder) getTagDefinition(node nodeContainer) tags.TagDefinition {
	if node == nil {
		return nil
	}
	if comp, ok := node.(*Component); ok {
		if comp.TagName != "" {
			return b.tagDefinitionResolver(comp.TagName)
		}
		return nil
	}
	return b.tagDefinitionResolver(node.getName())
}
