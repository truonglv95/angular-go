package expression_parser

import (
	"fmt"
	"strings"
)

type ParseFlags int

const (
	ParseFlagsNone   ParseFlags = 0
	ParseFlagsAction ParseFlags = 1 << 0
)

type ParseContextFlags int

const (
	ParseContextFlagsNone     ParseContextFlags = 0
	ParseContextFlagsWritable ParseContextFlags = 1 << 0
)

var EOFToken = Token{Type: TokenTypeCharacter}

var SUPPORTED_REGEX_FLAGS = map[string]bool{
	"d": true, "g": true, "i": true, "m": true, "s": true, "u": true, "v": true, "y": true,
}

func getLocation(span ParseSourceSpan) string {
	if true {
		return fmt.Sprintf("%d", 0)
	}
	return "(unknown)"
}

func getParseError(message string, input string, locationText string, parseSourceSpan ParseSourceSpan) ParseError {
	if len(locationText) > 0 {
		locationText = " " + locationText + " "
	}
	location := getLocation(parseSourceSpan)
	errorMsg := fmt.Sprintf("Parser Error: %s%s[%s] in %s", message, locationText, input, location)
	return NewParseError(parseSourceSpan, errorMsg)
}

type SplitInterpolation struct {
	Strings     []InterpolationPiece
	Expressions []InterpolationPiece
	Offsets     []int
}

type InterpolationPiece struct {
	Text  string
	Start int
	End   int
}

type Parser struct {
	lexer                        Lexer
	supportsDirectPipeReferences bool
}

func NewParser(lexer Lexer, supportsDirectPipeReferences bool) *Parser {
	return &Parser{
		lexer:                        lexer,
		supportsDirectPipeReferences: supportsDirectPipeReferences,
	}
}

func (p *Parser) ParseAction(input string, parseSourceSpan ParseSourceSpan, absoluteOffset int) ASTWithSource {
	errors := []ParseError{}
	p.checkNoInterpolation(&errors, input, parseSourceSpan)
	stripped, _ := p.stripComments(input)
	tokens := p.lexer.Tokenize(stripped)

	astNode := newParseAST(
		input,
		parseSourceSpan,
		absoluteOffset,
		tokens,
		ParseFlagsAction,
		&errors,
		0,
		p.supportsDirectPipeReferences,
	).parseChain()

	return NewASTWithSource(astNode, input, getLocation(parseSourceSpan), absoluteOffset, errors)
}

func (p *Parser) ParseBinding(input string, parseSourceSpan ParseSourceSpan, absoluteOffset int) ASTWithSource {
	errors := []ParseError{}
	astNode := p.parseBindingAst(input, parseSourceSpan, absoluteOffset, &errors)
	return NewASTWithSource(astNode, input, getLocation(parseSourceSpan), absoluteOffset, errors)
}

func (p *Parser) checkSimpleExpression(astNode AST) []string {
	checker := NewSimpleExpressionChecker()
	astNode.Visit(checker, nil)
	return checker.Errors
}

func (p *Parser) ParseSimpleBinding(input string, parseSourceSpan ParseSourceSpan, absoluteOffset int) ASTWithSource {
	errors := []ParseError{}
	astNode := p.parseBindingAst(input, parseSourceSpan, absoluteOffset, &errors)
	simpleExpressionErrors := p.checkSimpleExpression(astNode)

	if len(simpleExpressionErrors) > 0 {
		errors = append(errors, getParseError(
			fmt.Sprintf("Host binding expression cannot contain %s", strings.Join(simpleExpressionErrors, " ")),
			input,
			"",
			parseSourceSpan,
		))
	}
	return NewASTWithSource(astNode, input, getLocation(parseSourceSpan), absoluteOffset, errors)
}

func (p *Parser) parseBindingAst(input string, parseSourceSpan ParseSourceSpan, absoluteOffset int, errors *[]ParseError) AST {
	p.checkNoInterpolation(errors, input, parseSourceSpan)
	stripped, _ := p.stripComments(input)
	tokens := p.lexer.Tokenize(stripped)
	return newParseAST(
		input,
		parseSourceSpan,
		absoluteOffset,
		tokens,
		ParseFlagsNone,
		errors,
		0,
		p.supportsDirectPipeReferences,
	).parseChain()
}

func (p *Parser) ParseTemplateBindings(templateKey string, templateValue string, parseSourceSpan ParseSourceSpan, absoluteKeyOffset int, absoluteValueOffset int) TemplateBindingParseResult {
	tokens := p.lexer.Tokenize(templateValue)
	errors := []ParseError{}
	parser := newParseAST(
		templateValue,
		parseSourceSpan,
		absoluteValueOffset,
		tokens,
		ParseFlagsNone,
		&errors,
		0,
		p.supportsDirectPipeReferences,
	)
	return parser.parseTemplateBindings(TemplateBindingIdentifier{
		Source: templateKey,
		Span:   ParseSpan{Start: absoluteKeyOffset, End: absoluteKeyOffset + len(templateKey)},
	})
}

func (p *Parser) ParseInterpolation(input string, parseSourceSpan ParseSourceSpan, absoluteOffset int, interpolatedTokens []interface{}) ASTWithSource {
	errors := []ParseError{}
	split := p.SplitInterpolation(input, parseSourceSpan, &errors, interpolatedTokens)
	if len(split.Expressions) == 0 {
		return ASTWithSource{}
	}

	expressionNodes := []AST{}

	for i, exprPiece := range split.Expressions {
		var expressionSpan ParseSourceSpan
		var hasExpressionSpan bool
		// interpolatedTokens handling is mocked since the types can be InterpolatedAttributeToken or InterpolatedTextToken
		// Assuming we can access the span if available
		if interpolatedTokens != nil && i*2+1 < len(interpolatedTokens) {
			if token, ok := interpolatedTokens[i*2+1].(interface {
				GetSpanOffsets() (int, int)
			}); ok {
				start, end := token.GetSpanOffsets()
				expressionSpan = ParseSourceSpan{Start: start, End: end}
				hasExpressionSpan = true
			}
		}

		expressionText := exprPiece.Text
		stripped, hasComments := p.stripComments(expressionText)
		tokens := p.lexer.Tokenize(stripped)

		if hasComments && len(strings.TrimSpace(stripped)) == 0 && len(tokens) == 0 {
			*&errors = append(errors, getParseError(
				"Interpolation expression cannot only contain a comment",
				input,
				fmt.Sprintf("at column %d in", exprPiece.Start),
				parseSourceSpan,
			))
			continue
		}

		spanToUse := parseSourceSpan
		textToUse := input
		if hasExpressionSpan {
			spanToUse = expressionSpan
			textToUse = expressionText
		}

		astNode := newParseAST(
			textToUse,
			spanToUse,
			absoluteOffset,
			tokens,
			ParseFlagsNone,
			&errors,
			split.Offsets[i],
			p.supportsDirectPipeReferences,
		).parseChain()

		expressionNodes = append(expressionNodes, astNode)
	}

	stringsList := make([]string, len(split.Strings))
	for i, s := range split.Strings {
		stringsList[i] = s.Text
	}

	return p.createInterpolationAst(
		stringsList,
		expressionNodes,
		input,
		getLocation(parseSourceSpan),
		absoluteOffset,
		errors,
	)
}

func (p *Parser) ParseInterpolationExpression(expression string, parseSourceSpan ParseSourceSpan, absoluteOffset int) ASTWithSource {
	stripped, _ := p.stripComments(expression)
	tokens := p.lexer.Tokenize(stripped)
	errors := []ParseError{}
	astNode := newParseAST(
		expression,
		parseSourceSpan,
		absoluteOffset,
		tokens,
		ParseFlagsNone,
		&errors,
		0,
		p.supportsDirectPipeReferences,
	).parseChain()

	stringsList := []string{"", ""}
	return p.createInterpolationAst(
		stringsList,
		[]AST{astNode},
		expression,
		getLocation(parseSourceSpan),
		absoluteOffset,
		errors,
	)
}

func (p *Parser) createInterpolationAst(stringsList []string, expressions []AST, input string, location string, absoluteOffset int, errors []ParseError) ASTWithSource {
	stringsAny := make([]any, len(stringsList))
	for i, s := range stringsList {
		stringsAny[i] = s
	}
	span := ParseSpan{Start: 0, End: len(input)}
	interpolation := &Interpolation{
		ASTNodeBase: ASTNodeBase{
			SpanData:   span,
			SourceSpan: AbsoluteSourceSpan{Start: absoluteOffset, End: absoluteOffset + len(input)},
		},
		Strings:     stringsAny,
		Expressions: expressions,
	}
	return NewASTWithSource(interpolation, input, location, absoluteOffset, errors)
}

func (p *Parser) SplitInterpolation(input string, parseSourceSpan ParseSourceSpan, errors *[]ParseError, interpolatedTokens []interface{}) SplitInterpolation {
	stringsList := []InterpolationPiece{}
	expressions := []InterpolationPiece{}
	offsets := []int{}

	var inputToTemplateIndexMap map[int]int
	if interpolatedTokens != nil {
		inputToTemplateIndexMap = getIndexMapForOriginalTemplate(interpolatedTokens)
	}

	i := 0
	atInterpolation := false
	extendLastString := false
	interpStart := "{{"
	interpEnd := "}}"

	for i < len(input) {
		if !atInterpolation {
			start := i
			idx := strings.Index(input[i:], interpStart)
			if idx == -1 {
				i = len(input)
			} else {
				i += idx
			}
			text := input[start:i]
			stringsList = append(stringsList, InterpolationPiece{Text: text, Start: start, End: i})
			atInterpolation = true
		} else {
			fullStart := i
			exprStart := fullStart + len(interpStart)
			exprEnd := p.getInterpolationEndIndex(input, interpEnd, exprStart)
			if exprEnd == -1 {
				atInterpolation = false
				extendLastString = true
				break
			}
			fullEnd := exprEnd + len(interpEnd)
			text := input[exprStart:exprEnd]

			if len(strings.TrimSpace(text)) == 0 {
				*errors = append(*errors, getParseError(
					"Blank expressions are not allowed in interpolated strings",
					input,
					fmt.Sprintf("at column %d in", i),
					parseSourceSpan,
				))
			}

			expressions = append(expressions, InterpolationPiece{Text: text, Start: fullStart, End: fullEnd})

			startInOriginalTemplate := fullStart
			if inputToTemplateIndexMap != nil {
				if val, ok := inputToTemplateIndexMap[fullStart]; ok {
					startInOriginalTemplate = val
				}
			}

			offset := startInOriginalTemplate + len(interpStart)
			offsets = append(offsets, offset)

			i = fullEnd
			atInterpolation = false
		}
	}

	if !atInterpolation {
		if extendLastString {
			if len(stringsList) > 0 {
				piece := &stringsList[len(stringsList)-1]
				piece.Text += input[i:]
				piece.End = len(input)
			}
		} else {
			stringsList = append(stringsList, InterpolationPiece{Text: input[i:], Start: i, End: len(input)})
		}
	}

	return SplitInterpolation{Strings: stringsList, Expressions: expressions, Offsets: offsets}
}

func (p *Parser) WrapLiteralPrimitive(input *string, sourceSpanOrLocation interface{}, absoluteOffset int) ASTWithSource {
	length := 0
	if input != nil {
		length = len(*input)
	}
	span := ParseSpan{Start: 0, End: length}

	location := ""
	var srcSpan AbsoluteSourceSpan
	if str, ok := sourceSpanOrLocation.(string); ok {
		location = str
		srcSpan = AbsoluteSourceSpan{Start: absoluteOffset, End: absoluteOffset + length}
	} else if spanObj, ok := sourceSpanOrLocation.(ParseSourceSpan); ok {
		location = getLocation(spanObj)
		srcSpan = AbsoluteSourceSpan{Start: spanObj.Start, End: spanObj.End}
	} else if spanObj, ok := sourceSpanOrLocation.(AbsoluteSourceSpan); ok {
		srcSpan = spanObj
	}

	var val interface{} = nil
	source := ""
	if input != nil {
		val = *input
		source = *input
	}

	return NewASTWithSource(
		&LiteralPrimitive{
			ASTNodeBase: ASTNodeBase{
				SpanData:   span,
				SourceSpan: srcSpan,
			},
			Value: val,
		},
		source,
		location,
		absoluteOffset,
		[]ParseError{},
	)
}

func (p *Parser) stripComments(input string) (stripped string, hasComments bool) {
	i := p.commentStart(input)
	if i != -1 {
		return input[:i], true
	}
	return input, false
}

func (p *Parser) commentStart(input string) int {
	outerQuote := -1
	for i := 0; i < len(input)-1; i++ {
		char := input[i]
		nextChar := input[i+1]

		if char == '/' && nextChar == '/' && outerQuote == -1 {
			return i
		}

		if outerQuote == int(char) {
			outerQuote = -1
		} else if outerQuote == -1 && isQuote(int(char)) {
			outerQuote = int(char)
		}
	}
	return -1
}

func (p *Parser) checkNoInterpolation(errors *[]ParseError, input string, parseSourceSpan ParseSourceSpan) {
	startIndex := -1
	endIndex := -1

	p.forEachUnquotedChar(input, 0, func(charIndex int) bool {
		if startIndex == -1 {
			if strings.HasPrefix(input[charIndex:], "{{") {
				startIndex = charIndex
			}
		} else {
			endIndex = p.getInterpolationEndIndex(input, "}}", charIndex)
			if endIndex > -1 {
				return false
			}
		}
		return true
	})

	if startIndex > -1 && endIndex > -1 {
		*errors = append(*errors, getParseError(
			"Got interpolation ({{}}) where expression was expected",
			input,
			fmt.Sprintf("at column %d in", startIndex),
			parseSourceSpan,
		))
	}
}

func (p *Parser) getInterpolationEndIndex(input string, expressionEnd string, start int) int {
	res := -1
	p.forEachUnquotedChar(input, start, func(charIndex int) bool {
		if strings.HasPrefix(input[charIndex:], expressionEnd) {
			res = charIndex
			return false
		}
		if strings.HasPrefix(input[charIndex:], "//") {
			idx := strings.Index(input[charIndex:], expressionEnd)
			if idx != -1 {
				res = charIndex + idx
			}
			return false
		}
		return true
	})
	return res
}

func (p *Parser) forEachUnquotedChar(input string, start int, cb func(int) bool) {
	outerQuote := -1
	escapeCount := 0
	for i := start; i < len(input); i++ {
		char := int(input[i])
		if isQuote(char) && (outerQuote == -1 || outerQuote == char) && escapeCount%2 == 0 {
			if outerQuote == -1 {
				outerQuote = char
			} else {
				outerQuote = -1
			}
		} else if outerQuote == -1 {
			if !cb(i) {
				return
			}
		}
		if input[i] == '\\' {
			escapeCount++
		} else {
			escapeCount = 0
		}
	}
}

func isQuote(code int) bool {
	return code == '\'' || code == '"' || code == '`'
}

type parseAST struct {
	rparensExpected              int
	rbracketsExpected            int
	rbracesExpected              int
	context                      ParseContextFlags
	sourceSpanCache              map[string]AbsoluteSourceSpan
	index                        int
	input                        string
	parseSourceSpan              ParseSourceSpan
	absoluteOffset               int
	tokens                       []Token
	parseFlags                   ParseFlags
	errors                       *[]ParseError
	offset                       int
	supportsDirectPipeReferences bool
}

func newParseAST(
	input string,
	parseSourceSpan ParseSourceSpan,
	absoluteOffset int,
	tokens []Token,
	parseFlags ParseFlags,
	errors *[]ParseError,
	offset int,
	supportsDirectPipeReferences bool,
) *parseAST {
	return &parseAST{
		rparensExpected:              0,
		rbracketsExpected:            0,
		rbracesExpected:              0,
		context:                      ParseContextFlagsNone,
		sourceSpanCache:              make(map[string]AbsoluteSourceSpan),
		index:                        0,
		input:                        input,
		parseSourceSpan:              parseSourceSpan,
		absoluteOffset:               absoluteOffset,
		tokens:                       tokens,
		parseFlags:                   parseFlags,
		errors:                       errors,
		offset:                       offset,
		supportsDirectPipeReferences: supportsDirectPipeReferences,
	}
}

func (p *parseAST) peek(offset int) Token {
	i := p.index + offset
	if i < len(p.tokens) {
		return p.tokens[i]
	}
	return EOFToken
}

func (p *parseAST) next() Token {
	return p.peek(0)
}

func (p *parseAST) atEOF() bool {
	return p.index >= len(p.tokens)
}

func (p *parseAST) inputIndex() int {
	if p.atEOF() {
		return p.currentEndIndex()
	}
	return p.next().Index + p.offset
}

func (p *parseAST) newLiteralPrimitive(start int, value any) *LiteralPrimitive {
	return &LiteralPrimitive{
		ASTNodeBase: ASTNodeBase{
			SpanData:   p.span(start),
			SourceSpan: p.sourceSpan(start),
		},
		Value: value,
	}
}

func (p *parseAST) currentEndIndex() int {
	if p.index > 0 {
		curToken := p.peek(-1)
		return curToken.End + p.offset
	}
	if len(p.tokens) == 0 {
		return len(p.input) + p.offset
	}
	return p.next().Index + p.offset
}

func (p *parseAST) currentAbsoluteOffset() int {
	return p.absoluteOffset + p.inputIndex()
}

func (p *parseAST) span(start int, artificialEndIndex ...int) ParseSpan {
	endIndex := p.currentEndIndex()
	if len(artificialEndIndex) > 0 && artificialEndIndex[0] > p.currentEndIndex() {
		endIndex = artificialEndIndex[0]
	}
	if start > endIndex {
		tmp := endIndex
		endIndex = start
		start = tmp
	}
	return ParseSpan{Start: start, End: endIndex}
}

func (p *parseAST) sourceSpan(start int, artificialEndIndex ...int) AbsoluteSourceSpan {
	endIdx := -1
	if len(artificialEndIndex) > 0 {
		endIdx = artificialEndIndex[0]
	}
	serial := fmt.Sprintf("%d@%d:%d", start, p.inputIndex(), endIdx)
	if _, ok := p.sourceSpanCache[serial]; !ok {
		p.sourceSpanCache[serial] = p.span(start, artificialEndIndex...).ToAbsolute(p.absoluteOffset)
	}
	return p.sourceSpanCache[serial]
}

func (p *parseAST) advance() {
	p.index++
}

func (p *parseAST) withContext(context ParseContextFlags, cb func() AST) AST {
	p.context |= context
	ret := cb()
	p.context ^= context
	return ret
}

func (p *parseAST) consumeOptionalCharacter(code int) bool {
	if p.next().IsCharacter(code) {
		p.advance()
		return true
	}
	return false
}

func (p *parseAST) peekKeywordLet() bool {
	return p.next().IsKeywordLet()
}

func (p *parseAST) peekKeywordAs() bool {
	return p.next().IsKeywordAs()
}

func (p *parseAST) expectCharacter(code int) {
	if p.consumeOptionalCharacter(code) {
		return
	}
	p.error(fmt.Sprintf("Missing expected %c", code), p.index)
}

func (p *parseAST) consumeOptionalOperator(op string) bool {
	if p.next().IsOperator(op) {
		p.advance()
		return true
	}
	return false
}

func (p *parseAST) isAssignmentOperator(token Token) bool {
	return token.Type == TokenTypeOperator && IsAssignmentOperation(token.StrValue)
}

func (p *parseAST) expectOperator(operator string) {
	if p.consumeOptionalOperator(operator) {
		return
	}
	p.error(fmt.Sprintf("Missing expected operator %s", operator), p.index)
}

func (p *parseAST) prettyPrintToken(tok Token) string {
	if tok == EOFToken {
		return "end of input"
	}
	return fmt.Sprintf("token %s", *tok.ToString())
}

func (p *parseAST) expectIdentifierOrKeyword() *string {
	n := p.next()
	if !n.IsIdentifier() && !n.IsKeyword() {
		if n.IsPrivateIdentifier() {
			p.reportErrorForPrivateIdentifier(n, "expected identifier or keyword")
		} else {
			p.error(fmt.Sprintf("Unexpected %s, expected identifier or keyword", p.prettyPrintToken(n)), p.index)
		}
		return nil
	}
	p.advance()
	str := n.ToString()
	return str
}

func (p *parseAST) expectIdentifierOrKeywordOrString() string {
	n := p.next()
	if !n.IsIdentifier() && !n.IsKeyword() && !n.IsString() {
		if n.IsPrivateIdentifier() {
			p.reportErrorForPrivateIdentifier(n, "expected identifier, keyword or string")
		} else {
			p.error(fmt.Sprintf("Unexpected %s, expected identifier, keyword, or string", p.prettyPrintToken(n)), p.index)
		}
		return ""
	}
	p.advance()
	return *n.ToString()
}

func (p *parseAST) parseChain() AST {
	exprs := []AST{}
	start := p.inputIndex()
	for p.index < len(p.tokens) {
		expr := p.parsePipe()
		exprs = append(exprs, expr)

		if p.consumeOptionalCharacter(';') {
			if (p.parseFlags & ParseFlagsAction) == 0 {
				p.error("Binding expression cannot contain chained expression", p.index)
			}
			for p.consumeOptionalCharacter(';') {
			}
		} else if p.index < len(p.tokens) {
			errorIndex := p.index
			p.error(fmt.Sprintf("Unexpected token '%s'", *p.next().ToString()), p.index)
			if p.index == errorIndex {
				break
			}
		}
	}
	if len(exprs) == 0 {
		artificialStart := p.offset
		artificialEnd := p.offset + len(p.input)
		return NewEmptyExpr(p.span(artificialStart, artificialEnd), p.sourceSpan(artificialStart, artificialEnd))
	}
	if len(exprs) == 1 {
		return exprs[0]
	}
	return NewChain(p.span(start), p.sourceSpan(start), exprs)
}

func (p *parseAST) parsePipe() AST {
	start := p.inputIndex()
	result := p.parseExpression()
	if p.consumeOptionalOperator("|") {
		if (p.parseFlags & ParseFlagsAction) != 0 {
			p.error("Cannot have a pipe in an action expression", p.index)
		}

		for {
			nameStart := p.inputIndex()
			nameIdPtr := p.expectIdentifierOrKeyword()
			nameId := ""
			var nameSpan AbsoluteSourceSpan
			var fullSpanEnd int
			if nameIdPtr != nil {
				nameId = *nameIdPtr
				nameSpan = p.sourceSpan(nameStart)
				fullSpanEnd = -1
			} else {
				if p.next().Index != -1 {
					fullSpanEnd = p.next().Index
				} else {
					fullSpanEnd = len(p.input) + p.offset
				}
				nameSpan = p.sourceSpan(fullSpanEnd, fullSpanEnd)
			}

			args := []AST{}
			for p.consumeOptionalCharacter(':') {
				args = append(args, p.parseExpression())
			}

			var pipeType BindingPipeType
			if p.supportsDirectPipeReferences {
				if len(nameId) > 0 {
					charCode := nameId[0]
					if charCode == '_' || (charCode >= 'A' && charCode <= 'Z') {
						pipeType = BindingPipeTypeReferencedDirectly
					} else {
						pipeType = BindingPipeTypeReferencedByName
					}
				} else {
					pipeType = BindingPipeTypeReferencedByName
				}
			} else {
				pipeType = BindingPipeTypeReferencedByName
			}

			var srcSpan AbsoluteSourceSpan
			if fullSpanEnd != -1 {
				srcSpan = p.sourceSpan(start, fullSpanEnd)
			} else {
				srcSpan = p.sourceSpan(start)
			}

			result = NewBindingPipe(p.span(start), srcSpan, result, nameId, args, pipeType, nameSpan)

			if !p.consumeOptionalOperator("|") {
				break
			}
		}
	}
	return result
}

func (p *parseAST) parseExpression() AST {
	return p.parseConditional()
}

func (p *parseAST) parseConditional() AST {
	start := p.inputIndex()
	result := p.parseLogicalOr()

	if p.consumeOptionalOperator("?") {
		yes := p.parsePipe()
		var no AST
		if !p.consumeOptionalCharacter(':') {
			end := p.inputIndex()
			expression := p.input[start:end]
			p.error(fmt.Sprintf("Conditional expression %s requires all 3 expressions", expression), p.index)
			no = NewEmptyExpr(p.span(start), p.sourceSpan(start))
		} else {
			no = p.parsePipe()
		}
		return NewConditional(p.span(start), p.sourceSpan(start), result, yes, no)
	}
	return result
}

func (p *parseAST) parseLogicalOr() AST {
	start := p.inputIndex()
	result := p.parseLogicalAnd()
	for p.consumeOptionalOperator("||") {
		right := p.parseLogicalAnd()
		result = NewBinary(p.span(start), p.sourceSpan(start), "||", result, right)
	}
	return result
}

func (p *parseAST) parseLogicalAnd() AST {
	start := p.inputIndex()
	result := p.parseNullishCoalescing()
	for p.consumeOptionalOperator("&&") {
		right := p.parseNullishCoalescing()
		result = NewBinary(p.span(start), p.sourceSpan(start), "&&", result, right)
	}
	return result
}

func (p *parseAST) parseNullishCoalescing() AST {
	start := p.inputIndex()
	result := p.parseEquality()
	for p.consumeOptionalOperator("??") {
		right := p.parseEquality()
		result = NewBinary(p.span(start), p.sourceSpan(start), "??", result, right)
	}
	return result
}

func (p *parseAST) parseEquality() AST {
	start := p.inputIndex()
	result := p.parseRelational()
	for p.next().Type == TokenTypeOperator {
		operator := p.next().StrValue
		switch operator {
		case "==", "===", "!=", "!==":
			p.advance()
			right := p.parseRelational()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

func (p *parseAST) parseRelational() AST {
	start := p.inputIndex()
	result := p.parseAdditive()
	for p.next().Type == TokenTypeOperator || p.next().IsKeywordIn() || p.next().IsKeywordInstanceOf() {
		operator := p.next().StrValue
		switch operator {
		case "<", ">", "<=", ">=", "in", "instanceof":
			p.advance()
			right := p.parseAdditive()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

func (p *parseAST) parseAdditive() AST {
	start := p.inputIndex()
	result := p.parseMultiplicative()
	for p.next().Type == TokenTypeOperator {
		operator := p.next().StrValue
		switch operator {
		case "+", "-":
			p.advance()
			right := p.parseMultiplicative()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

func (p *parseAST) parseMultiplicative() AST {
	start := p.inputIndex()
	result := p.parseExponentiation()
	for p.next().Type == TokenTypeOperator {
		operator := p.next().StrValue
		switch operator {
		case "*", "%", "/":
			p.advance()
			right := p.parseExponentiation()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

func (p *parseAST) parseExponentiation() AST {
	start := p.inputIndex()
	result := p.parsePrefix()
	for p.next().Type == TokenTypeOperator && p.next().StrValue == "**" {
		if IsUnary(result) || IsPrefixNot(result) || IsTypeofExpression(result) || IsVoidExpression(result) {
			p.error("Unary operator used immediately before exponentiation expression. Parenthesis must be used to disambiguate operator precedence", p.index)
		}
		p.advance()
		right := p.parseExponentiation()
		result = NewBinary(p.span(start), p.sourceSpan(start), "**", result, right)
	}
	return result
}

func (p *parseAST) parsePrefix() AST {
	if p.next().Type == TokenTypeOperator {
		start := p.inputIndex()
		operator := p.next().StrValue
		var result AST
		switch operator {
		case "+":
			p.advance()
			result = p.parsePrefix()
			return CreateUnaryPlus(p.span(start), p.sourceSpan(start), result)
		case "-":
			p.advance()
			result = p.parsePrefix()
			return CreateUnaryMinus(p.span(start), p.sourceSpan(start), result)
		case "!":
			p.advance()
			result = p.parsePrefix()
			return NewPrefixNot(p.span(start), p.sourceSpan(start), result)
		}
	} else if p.next().IsKeywordTypeof() {
		start := p.inputIndex()
		p.advance()
		result := p.parsePrefix()
		return NewTypeofExpression(p.span(start), p.sourceSpan(start), result)
	} else if p.next().IsKeywordVoid() {
		start := p.inputIndex()
		p.advance()
		result := p.parsePrefix()
		return NewVoidExpression(p.span(start), p.sourceSpan(start), result)
	}
	return p.parseCallChain()
}

func (p *parseAST) parseCallChain() AST {
	start := p.inputIndex()
	result := p.parsePrimary()
	for {
		if p.consumeOptionalCharacter('.') {
			result = p.parseAccessMember(result, start, false)
		} else if p.consumeOptionalOperator("?.") {
			if p.consumeOptionalCharacter('(') {
				result = p.parseCall(result, start, true)
			} else {
				if p.consumeOptionalCharacter('[') {
					result = p.parseKeyedReadOrWrite(result, start, true)
				} else {
					result = p.parseAccessMember(result, start, true)
				}
			}
		} else if p.consumeOptionalCharacter('[') {
			result = p.parseKeyedReadOrWrite(result, start, false)
		} else if p.consumeOptionalCharacter('(') {
			result = p.parseCall(result, start, false)
		} else if p.consumeOptionalOperator("!") {
			result = NewNonNullAssert(p.span(start), p.sourceSpan(start), result)
		} else if p.next().IsTemplateLiteralEnd() {
			result = p.parseNoInterpolationTaggedTemplateLiteral(result, start)
		} else if p.next().IsTemplateLiteralPart() {
			result = p.parseTaggedTemplateLiteral(result, start)
		} else {
			return result
		}
	}
}

func (p *parseAST) parsePrimary() AST {
	start := p.inputIndex()
	if p.isArrowFunction() {
		return p.parseArrowFunction(start)
	} else if p.consumeOptionalCharacter('(') {
		p.rparensExpected++
		result := p.parsePipe()
		if !p.consumeOptionalCharacter(')') {
			p.error("Missing closing parentheses", p.index)
			p.consumeOptionalCharacter(')')
		}
		p.rparensExpected--
		return NewParenthesizedExpression(p.span(start), p.sourceSpan(start), result)
	} else if p.next().IsKeywordNull() {
		p.advance()
		return p.newLiteralPrimitive(start, nil)
	} else if p.next().IsKeywordUndefined() {
		p.advance()
		return p.newLiteralPrimitive(start, "undefined") // HACK for serializer
	} else if p.next().IsKeywordTrue() {
		p.advance()
		return p.newLiteralPrimitive(start, true)
	} else if p.next().IsKeywordFalse() {
		p.advance()
		return p.newLiteralPrimitive(start, false)
	} else if p.next().IsKeywordThis() {
		p.advance()
		return NewThisReceiver(p.span(start), p.sourceSpan(start))
	} else if p.consumeOptionalCharacter('[') {
		return p.parseLiteralArray(start)
	} else if p.next().IsCharacter('{') {
		return p.parseLiteralMap()
	} else if p.next().IsIdentifier() {
		return p.parseAccessMember(
			NewImplicitReceiver(p.span(start), p.sourceSpan(start)),
			start,
			false,
		)
	} else if p.next().IsNumber() {
		value := p.next().ToNumber()
		p.advance()
		return p.newLiteralPrimitive(start, value)
	} else if p.next().IsTemplateLiteralEnd() {
		return p.parseNoInterpolationTemplateLiteral()
	} else if p.next().IsTemplateLiteralPart() {
		return p.parseTemplateLiteral()
	} else if p.next().IsString() && p.next().Kind == StringTokenKindPlain {
		literalValue := *p.next().ToString()
		p.advance()
		return p.newLiteralPrimitive(start, literalValue)
	} else if p.next().IsPrivateIdentifier() {
		p.reportErrorForPrivateIdentifier(p.next(), "")
		return NewEmptyExpr(p.span(start), p.sourceSpan(start))
	} else if p.next().IsRegExpBody() {
		return p.parseRegularExpressionLiteral()
	} else if p.index >= len(p.tokens) {
		p.error(fmt.Sprintf("Unexpected end of expression: %s", p.input), p.index)
		return NewEmptyExpr(p.span(start), p.sourceSpan(start))
	} else {
		p.error(fmt.Sprintf("Unexpected token %s", *p.next().ToString()), p.index)
		return NewEmptyExpr(p.span(start), p.sourceSpan(start))
	}
}

func (p *parseAST) parseLiteralArray(arrayStart int) AST {
	p.rbracketsExpected++
	elements := []AST{}

	for {
		if p.next().IsOperator("...") {
			elements = append(elements, p.parseSpreadElement())
		} else if !p.next().IsCharacter(']') {
			elements = append(elements, p.parsePipe())
		} else {
			break
		}
		if !p.consumeOptionalCharacter(',') {
			break
		}
	}

	p.rbracketsExpected--
	p.expectCharacter(']')
	return NewLiteralArray(p.span(arrayStart), p.sourceSpan(arrayStart), elements)
}

func (p *parseAST) parseLiteralMap() AST {
	keys := []LiteralMapKey{}
	values := []AST{}
	start := p.inputIndex()
	p.expectCharacter('{')
	if !p.consumeOptionalCharacter('}') {
		p.rbracesExpected++
		for {
			keyStart := p.inputIndex()

			if p.next().IsOperator("...") {
				p.advance()
				keys = append(keys, NewLiteralMapSpreadKey(p.span(keyStart), p.sourceSpan(keyStart)))
				values = append(values, p.parsePipe())
				if !p.consumeOptionalCharacter(',') || p.next().IsCharacter('}') {
					break
				}
				continue
			}

			quoted := p.next().IsString()
			key := p.expectIdentifierOrKeywordOrString()
			keySpan := p.span(keyStart)
			keySourceSpan := p.sourceSpan(keyStart)

			literalMapKey := NewLiteralMapPropertyKey(key, quoted, keySpan, keySourceSpan)
			keys = append(keys, literalMapKey)

			if quoted {
				p.expectCharacter(':')
				values = append(values, p.parsePipe())
			} else if p.consumeOptionalCharacter(':') {
				values = append(values, p.parsePipe())
			} else {
				literalMapKey.SetIsShorthandInitialized(true)
				values = append(values, NewPropertyRead(
					keySpan,
					keySourceSpan,
					keySourceSpan,
					NewImplicitReceiver(keySpan, keySourceSpan),
					key,
				))
			}

			if !p.consumeOptionalCharacter(',') || p.next().IsCharacter('}') {
				break
			}
		}
		p.rbracesExpected--
		p.expectCharacter('}')
	}
	return NewLiteralMap(p.span(start), p.sourceSpan(start), keys, values)
}

func (p *parseAST) parseAccessMember(readReceiver AST, start int, isSafe bool) AST {
	nameStart := p.inputIndex()
	id := p.withContext(ParseContextFlagsWritable, func() AST {
		idStrPtr := p.expectIdentifierOrKeyword()
		idStr := ""
		if idStrPtr != nil {
			idStr = *idStrPtr
		}
		if len(idStr) == 0 {
			errIdx := p.index
			if errIdx > 0 {
				errIdx = p.index - 1
			}
			p.error("Expected identifier for property access", errIdx)
		}
		return &LiteralPrimitive{Value: idStr} // Just to pass string via AST, actually we want string
	})

	idStr := ""
	if lit, ok := id.(*LiteralPrimitive); ok {
		idStr = lit.Value.(string)
	}

	nameSpan := p.sourceSpan(nameStart)

	if isSafe {
		if p.isAssignmentOperator(p.next()) {
			p.advance()
			p.error("The '?.' operator cannot be used in the assignment", p.index)
			return NewEmptyExpr(p.span(start), p.sourceSpan(start))
		} else {
			return NewSafePropertyRead(
				p.span(start),
				p.sourceSpan(start),
				nameSpan,
				readReceiver,
				idStr,
			)
		}
	} else {
		if p.isAssignmentOperator(p.next()) {
			operation := p.next().StrValue

			if (p.parseFlags & ParseFlagsAction) == 0 {
				p.advance()
				p.error("Bindings cannot contain assignments", p.index)
				return NewEmptyExpr(p.span(start), p.sourceSpan(start))
			}
			receiver := NewPropertyRead(
				p.span(start),
				p.sourceSpan(start),
				nameSpan,
				readReceiver,
				idStr,
			)
			p.advance()
			value := p.parseConditional()
			return NewBinary(p.span(start), p.sourceSpan(start), operation, receiver, value)
		} else {
			return NewPropertyRead(
				p.span(start),
				p.sourceSpan(start),
				nameSpan,
				readReceiver,
				idStr,
			)
		}
	}
}

func (p *parseAST) parseCall(receiver AST, start int, isSafe bool) AST {
	argumentStart := p.inputIndex()
	p.rparensExpected++
	args := p.parseCallArguments()
	argumentSpan := p.span(argumentStart)
	p.expectCharacter(')')
	p.rparensExpected--
	span := p.span(start)
	sourceSpan := p.sourceSpan(start)
	if isSafe {
		return NewSafeCall(span, sourceSpan, receiver, args, argumentSpan)
	}
	return NewCall(span, sourceSpan, receiver, args, argumentSpan)
}

func (p *parseAST) parseCallArguments() []AST {
	if p.next().IsCharacter(')') {
		return []AST{}
	}

	positionals := []AST{}

	for {
		if p.next().IsOperator("...") {
			positionals = append(positionals, p.parseSpreadElement())
		} else {
			positionals = append(positionals, p.parsePipe())
		}
		if !p.consumeOptionalCharacter(',') {
			break
		}
	}

	return positionals
}

func (p *parseAST) parseSpreadElement() AST {
	if !p.next().IsOperator("...") {
		p.error("Spread element must start with '...' operator", p.index)
	}

	spreadStart := p.inputIndex()
	p.advance()
	expression := p.parsePipe()
	span := p.span(spreadStart)
	sourceSpan := p.sourceSpan(spreadStart)
	return NewSpreadElement(span, sourceSpan, expression)
}

func (p *parseAST) expectTemplateBindingKey() TemplateBindingIdentifier {
	result := ""
	operatorFound := false
	start := p.currentAbsoluteOffset()
	for {
		result += p.expectIdentifierOrKeywordOrString()
		operatorFound = p.consumeOptionalOperator("-")
		if operatorFound {
			result += "-"
		}
		if !operatorFound {
			break
		}
	}
	return TemplateBindingIdentifier{
		Source: result,
		Span:   ParseSpan{Start: start, End: start + len(result)},
	}
}

func (p *parseAST) parseTemplateBindings(templateKey TemplateBindingIdentifier) TemplateBindingParseResult {
	bindings := []TemplateBinding{}

	bindings = append(bindings, p.parseDirectiveKeywordBindings(templateKey)...)

	for p.index < len(p.tokens) {
		letBinding := p.parseLetBinding()
		if letBinding != nil {
			bindings = append(bindings, letBinding)
		} else {
			key := p.expectTemplateBindingKey()
			binding := p.parseAsBinding(key)
			if binding != nil {
				bindings = append(bindings, binding)
			} else {
				if len(key.Source) > 0 {
					key.Source = templateKey.Source + strings.ToUpper(string(key.Source[0])) + key.Source[1:]
				}
				bindings = append(bindings, p.parseDirectiveKeywordBindings(key)...)
			}
		}
		p.consumeStatementTerminator()
	}

	return TemplateBindingParseResult{
		TemplateBindings: bindings,
		Warnings:         []string{},
		Errors:           *p.errors,
	}
}

func (p *parseAST) parseKeyedReadOrWrite(receiver AST, start int, isSafe bool) AST {
	return p.withContext(ParseContextFlagsWritable, func() AST {
		p.rbracketsExpected++
		key := p.parsePipe()
		if IsEmptyExpr(key) {
			p.error("Key access cannot be empty", p.index)
		}
		p.rbracketsExpected--
		p.expectCharacter(']')
		if p.isAssignmentOperator(p.next()) {
			operation := p.next().StrValue

			if isSafe {
				p.advance()
				p.error("The '?.' operator cannot be used in the assignment", p.index)
			} else {
				binaryReceiver := NewKeyedRead(
					p.span(start),
					p.sourceSpan(start),
					receiver,
					key,
				)
				p.advance()
				value := p.parseConditional()
				return NewBinary(
					p.span(start),
					p.sourceSpan(start),
					operation,
					binaryReceiver,
					value,
				)
			}
		} else {
			if isSafe {
				return NewSafeKeyedRead(p.span(start), p.sourceSpan(start), receiver, key)
			}
			return NewKeyedRead(p.span(start), p.sourceSpan(start), receiver, key)
		}

		return NewEmptyExpr(p.span(start), p.sourceSpan(start))
	})
}

func (p *parseAST) parseDirectiveKeywordBindings(key TemplateBindingIdentifier) []TemplateBinding {
	bindings := []TemplateBinding{}
	p.consumeOptionalCharacter(':')
	value := p.getDirectiveBoundTarget()
	spanEnd := p.currentAbsoluteOffset()
	asBinding := p.parseAsBinding(key)
	if asBinding == nil {
		p.consumeStatementTerminator()
		spanEnd = p.currentAbsoluteOffset()
	}
	sourceSpan := NewAbsoluteSourceSpan(key.Span.Start, spanEnd)
	var valuePtr *ASTWithSource
	if value.Ast != nil {
		valuePtr = &value
	}
	bindings = append(bindings, NewExpressionBinding(sourceSpan, key, valuePtr))
	if asBinding != nil {
		bindings = append(bindings, asBinding)
	}
	return bindings
}

func (p *parseAST) getDirectiveBoundTarget() ASTWithSource {
	if p.next() == EOFToken || p.peekKeywordAs() || p.peekKeywordLet() {
		return ASTWithSource{}
	}
	startInputIdx := p.inputIndex()
	astNode := p.parsePipe()
	endInputIdx := p.currentEndIndex()
	value := ""
	if startInputIdx >= 0 && endInputIdx >= startInputIdx && endInputIdx <= len(p.input) {
		value = p.input[startInputIdx:endInputIdx]
	}
	return NewASTWithSource(
		astNode,
		value,
		getLocation(p.parseSourceSpan),
		p.absoluteOffset+startInputIdx,
		*p.errors,
	)
}

func (p *parseAST) parseAsBinding(value TemplateBindingIdentifier) TemplateBinding {
	if !p.peekKeywordAs() {
		return nil
	}
	p.advance()
	key := p.expectTemplateBindingKey()
	p.consumeStatementTerminator()
	sourceSpan := NewAbsoluteSourceSpan(value.Span.Start, p.currentAbsoluteOffset())
	return NewVariableBinding(sourceSpan, key, &value)
}

func (p *parseAST) parseLetBinding() TemplateBinding {
	if !p.peekKeywordLet() {
		return nil
	}
	spanStart := p.currentAbsoluteOffset()
	p.advance()
	key := p.expectTemplateBindingKey()
	var value *TemplateBindingIdentifier = nil
	if p.consumeOptionalOperator("=") {
		v := p.expectTemplateBindingKey()
		value = &v
	}
	p.consumeStatementTerminator()
	sourceSpan := NewAbsoluteSourceSpan(spanStart, p.currentAbsoluteOffset())
	return NewVariableBinding(sourceSpan, key, value)
}

func (p *parseAST) parseNoInterpolationTaggedTemplateLiteral(tag AST, start int) AST {
	template := p.parseNoInterpolationTemplateLiteral()
	return NewTaggedTemplateLiteral(ParseSpan{Start: start, End: p.index}, NewAbsoluteSourceSpan(start, p.index), tag, template)
}

func (p *parseAST) parseNoInterpolationTemplateLiteral() AST {
	text := p.next().StrValue
	start := p.inputIndex()
	p.advance()
	span := ParseSpan{Start: start, End: p.index}
	sourceSpan := NewAbsoluteSourceSpan(start, p.index)
	return NewTemplateLiteral(
		span,
		sourceSpan,
		[]*TemplateLiteralElement{NewTemplateLiteralElement(span, sourceSpan, text)},
		[]AST{},
	)
}

func (p *parseAST) parseTaggedTemplateLiteral(tag AST, start int) AST {
	template := p.parseTemplateLiteral()
	return NewTaggedTemplateLiteral(ParseSpan{Start: start, End: p.index}, NewAbsoluteSourceSpan(start, p.index), tag, template)
}

func (p *parseAST) parseTemplateLiteral() AST {
	elements := []*TemplateLiteralElement{}
	expressions := []AST{}
	start := p.inputIndex()

	for p.next() != EOFToken {
		token := p.next()

		if token.IsTemplateLiteralPart() || token.IsTemplateLiteralEnd() {
			partStart := p.inputIndex()
			p.advance()
			elements = append(elements, NewTemplateLiteralElement(
				p.span(partStart),
				p.sourceSpan(partStart),
				token.StrValue,
			))
			if token.IsTemplateLiteralEnd() {
				break
			}
		} else if token.IsTemplateLiteralInterpolationStart() {
			p.advance()
			p.rbracesExpected++
			expression := p.parsePipe()
			if IsEmptyExpr(expression) {
				p.error("Template literal interpolation cannot be empty", p.index)
			} else {
				expressions = append(expressions, expression)
			}
			p.rbracesExpected--
		} else {
			p.advance()
		}
	}

	return NewTemplateLiteral(p.span(start), p.sourceSpan(start), elements, expressions)
}

func (p *parseAST) parseRegularExpressionLiteral() AST {
	bodyToken := p.next()
	p.advance()

	if !bodyToken.IsRegExpBody() {
		return NewEmptyExpr(p.span(p.inputIndex()), p.sourceSpan(p.inputIndex()))
	}

	start := bodyToken.Index
	end := bodyToken.End
	flagsStr := ""

	if p.next().IsRegExpFlags() {
		flagsToken := p.next()
		p.advance()
		flagsStr = flagsToken.StrValue
		end = flagsToken.End
		seenFlags := make(map[byte]bool)

		for i := 0; i < len(flagsToken.StrValue); i++ {
			char := flagsToken.StrValue[i]
			charStr := string(char)

			if !SUPPORTED_REGEX_FLAGS[charStr] {
				supported := []string{}
				for f := range SUPPORTED_REGEX_FLAGS {
					supported = append(supported, fmt.Sprintf(`"%s"`, f))
				}
				p.error(
					fmt.Sprintf(`Unsupported regular expression flag "%c". The supported flags are: %s`, char, strings.Join(supported, ", ")),
					flagsToken.Index+i,
				)
			} else if seenFlags[char] {
				p.error(fmt.Sprintf(`Duplicate regular expression flag "%c"`, char), flagsToken.Index+i)
			} else {
				seenFlags[char] = true
			}
		}
	}

	return NewRegularExpressionLiteral(
		p.span(start, end),
		p.sourceSpan(start, end),
		bodyToken.StrValue,
		flagsStr,
	)
}

func (p *parseAST) parseArrowFunction(start int) AST {
	var params []ArrowFunctionParameter

	if p.next().IsIdentifier() {
		token := p.next()
		p.advance()
		params = []ArrowFunctionParameter{p.getArrowFunctionIdentifierArg(token)}
	} else if p.next().IsCharacter('(') {
		p.rparensExpected++
		p.advance()
		params = p.parseArrowFunctionParameters()
		p.rparensExpected--
	} else {
		params = []ArrowFunctionParameter{}
		p.error(fmt.Sprintf("Unexpected token %s", *p.next().ToString()), p.index)
	}

	p.expectOperator("=>")
	var body AST

	if p.next().IsCharacter('{') {
		p.error("Multi-line arrow functions are not supported. If you meant to return an object literal, wrap it with parentheses.", p.index)
		body = NewEmptyExpr(p.span(start), p.sourceSpan(start))
	} else {
		prevFlags := p.parseFlags
		p.parseFlags = ParseFlagsAction
		body = p.parseExpression()
		p.parseFlags = prevFlags
	}

	return NewArrowFunction(p.span(start), p.sourceSpan(start), params, body)
}

func (p *parseAST) parseArrowFunctionParameters() []ArrowFunctionParameter {
	params := []ArrowFunctionParameter{}

	if !p.consumeOptionalCharacter(')') {
		for p.next() != EOFToken {
			if p.next().IsIdentifier() {
				token := p.next()
				p.advance()
				params = append(params, p.getArrowFunctionIdentifierArg(token))

				if p.consumeOptionalCharacter(')') {
					break
				} else {
					p.expectCharacter(',')
				}
			} else {
				p.error(fmt.Sprintf("Unexpected token %s", *p.next().ToString()), p.index)
				break
			}
		}
	}

	return params
}

func (p *parseAST) getArrowFunctionIdentifierArg(token Token) ArrowFunctionParameter {
	return NewArrowFunctionIdentifierParameter(
		token.StrValue,
		p.span(token.Index),
		p.sourceSpan(token.Index),
	)
}

func (p *parseAST) isArrowFunction() bool {
	start := p.index
	tokens := p.tokens

	if start > len(tokens)-2 {
		return false
	}

	if tokens[start].IsIdentifier() && tokens[start+1].IsOperator("=>") {
		return true
	}

	if tokens[start].IsCharacter('(') {
		i := start + 1

		for ; i < len(tokens); i++ {
			if !tokens[i].IsIdentifier() && !tokens[i].IsCharacter(',') {
				break
			}
		}

		return i < len(tokens)-1 && tokens[i].IsCharacter(')') && tokens[i+1].IsOperator("=>")
	}

	return false
}

func (p *parseAST) consumeStatementTerminator() {
	if !p.consumeOptionalCharacter(';') {
		p.consumeOptionalCharacter(',')
	}
}

func (p *parseAST) error(message string, index int) {
	*p.errors = append(*p.errors, getParseError(
		message,
		p.input,
		p.getErrorLocationText(index),
		p.parseSourceSpan,
	))
	p.skip()
}

func (p *parseAST) getErrorLocationText(index int) string {
	if index < len(p.tokens) {
		return fmt.Sprintf("at column %d in", p.tokens[index].Index+1)
	}
	return "at the end of the expression"
}

func (p *parseAST) reportErrorForPrivateIdentifier(token Token, extraMessage string) {
	errorMessage := fmt.Sprintf("Private identifiers are not supported. Unexpected private identifier: %s", *token.ToString())
	if len(extraMessage) > 0 {
		errorMessage += ", " + extraMessage
	}
	p.error(errorMessage, p.index)
}

func (p *parseAST) skip() {
	n := p.next()
	for p.index < len(p.tokens) &&
		!n.IsCharacter(';') &&
		!n.IsOperator("|") &&
		(p.rparensExpected <= 0 || !n.IsCharacter(')')) &&
		(p.rbracesExpected <= 0 || !n.IsCharacter('}')) &&
		(p.rbracketsExpected <= 0 || !n.IsCharacter(']')) &&
		((p.context&ParseContextFlagsWritable) == 0 || !p.isAssignmentOperator(n)) {

		if p.next().IsError() {
			*p.errors = append(*p.errors, getParseError(
				p.next().StrValue,
				p.input,
				p.getErrorLocationText(p.next().Index),
				p.parseSourceSpan,
			))
		}
		p.advance()
		n = p.next()
	}
}

type SimpleExpressionChecker struct {
	RecursiveAstVisitor
	Errors []string
}

func NewSimpleExpressionChecker() *SimpleExpressionChecker {
	c := &SimpleExpressionChecker{Errors: []string{}}
	c.RecursiveAstVisitor.Impl = c
	return c
}

func (c *SimpleExpressionChecker) visit(ast AST) {
	if ast != nil {
		ast.Visit(c, nil)
	}
}

func (c *SimpleExpressionChecker) visitAll(asts []AST) {
	for _, a := range asts {
		c.visit(a)
	}
}

func (c *SimpleExpressionChecker) VisitImplicitReceiver(ast *ImplicitReceiver, context any) any {
	return nil
}
func (c *SimpleExpressionChecker) VisitThisReceiver(ast *ThisReceiver, context any) any {
	return nil
}
func (c *SimpleExpressionChecker) VisitLiteralPrimitive(ast *LiteralPrimitive, context any) any {
	return nil
}
func (c *SimpleExpressionChecker) VisitPropertyRead(ast *PropertyRead, context any) any {
	c.visit(ast.Receiver)
	return nil
}
func (c *SimpleExpressionChecker) VisitSafePropertyRead(ast *SafePropertyRead, context any) any {
	c.visit(ast.Receiver)
	return nil
}
func (c *SimpleExpressionChecker) VisitMethodCall(ast *MethodCall, context any) any {
	c.visit(ast.Receiver)
	c.visitAll(ast.Args)
	return nil
}
func (c *SimpleExpressionChecker) VisitSafeMethodCall(ast *SafeMethodCall, context any) any {
	c.visit(ast.Receiver)
	c.visitAll(ast.Args)
	return nil
}
func (c *SimpleExpressionChecker) VisitCall(ast *Call, context any) any {
	c.visit(ast.Receiver)
	c.visitAll(ast.Args)
	return nil
}
func (c *SimpleExpressionChecker) VisitSafeCall(ast *SafeCall, context any) any {
	c.visit(ast.Receiver)
	c.visitAll(ast.Args)
	return nil
}
func (c *SimpleExpressionChecker) VisitFunctionCall(ast *FunctionCall, context any) any {
	if ast.Target != nil {
		c.visit(ast.Target)
	}
	c.visitAll(ast.Args)
	return nil
}
func (c *SimpleExpressionChecker) VisitBinary(ast *Binary, context any) any {
	c.visit(ast.Left)
	c.visit(ast.Right)
	return nil
}
func (c *SimpleExpressionChecker) VisitConditional(ast *Conditional, context any) any {
	c.visit(ast.Condition)
	c.visit(ast.TrueExp)
	c.visit(ast.FalseExp)
	return nil
}
func (c *SimpleExpressionChecker) VisitKeyedRead(ast *KeyedRead, context any) any {
	c.visit(ast.Receiver)
	c.visit(ast.Key)
	return nil
}
func (c *SimpleExpressionChecker) VisitSafeKeyedRead(ast *SafeKeyedRead, context any) any {
	c.visit(ast.Receiver)
	c.visit(ast.Key)
	return nil
}
func (c *SimpleExpressionChecker) VisitKeyedWrite(ast *KeyedWrite, context any) any {
	c.visit(ast.Receiver)
	c.visit(ast.Key)
	c.visit(ast.Value)
	return nil
}
func (c *SimpleExpressionChecker) VisitPropertyWrite(ast *PropertyWrite, context any) any {
	c.visit(ast.Receiver)
	c.visit(ast.Value)
	return nil
}
func (c *SimpleExpressionChecker) VisitChain(ast *Chain, context any) any {
	c.visitAll(ast.Expressions)
	return nil
}
func (c *SimpleExpressionChecker) VisitPrefixNot(ast *PrefixNot, context any) any {
	c.visit(ast.Expression)
	return nil
}
func (c *SimpleExpressionChecker) VisitNonNullAssert(ast *NonNullAssert, context any) any {
	c.visit(ast.Expression)
	return nil
}
func (c *SimpleExpressionChecker) VisitLiteralArray(ast *LiteralArray, context any) any {
	c.visitAll(ast.Expressions)
	return nil
}
func (c *SimpleExpressionChecker) VisitLiteralMap(ast *LiteralMap, context any) any {
	c.visitAll(ast.Values)
	return nil
}
func (c *SimpleExpressionChecker) VisitInterpolation(ast *Interpolation, context any) any {
	c.visitAll(ast.Expressions)
	return nil
}
func (c *SimpleExpressionChecker) VisitUnary(ast *Unary, context any) any {
	c.visit(ast.Expr)
	return nil
}
func (c *SimpleExpressionChecker) VisitTypeofExpression(ast *TypeofExpression, context any) any {
	c.visit(ast.Expr)
	return nil
}
func (c *SimpleExpressionChecker) VisitVoidExpression(ast *VoidExpression, context any) any {
	c.visit(ast.Expr)
	return nil
}
func (c *SimpleExpressionChecker) VisitParenthesizedExpression(ast *ParenthesizedExpression, context any) any {
	c.visit(ast.Expression)
	return nil
}
func (c *SimpleExpressionChecker) VisitQuote(ast *Quote, context any) any { return nil }

// VisitPipe is the key method — report a "pipes" error
func (c *SimpleExpressionChecker) VisitPipe(ast *BindingPipe, context any) any {
	c.Errors = append(c.Errors, "pipes")
	c.visit(ast.Exp)
	c.visitAll(ast.Args)
	return nil
}

func IsAssignmentOperation(op string) bool {
	return op == "=" || op == "+=" || op == "-=" || op == "*=" || op == "**=" || op == "/=" || op == "%=" || op == "&=" || op == "|=" || op == "^=" || op == "<<=" || op == ">>=" || op == ">>>=" || op == "&&=" || op == "||=" || op == "??="
}

func getIndexMapForOriginalTemplate(interpolatedTokens []interface{}) map[int]int {
	offsetMap := make(map[int]int)
	consumedInOriginalTemplate := 0
	consumedInInput := 0
	tokenIndex := 0

	for tokenIndex < len(interpolatedTokens) {
		currentToken := interpolatedTokens[tokenIndex]

		// This uses type assertions based on the shape. We mock this loosely.
		// Assuming InterpolatedTextToken/InterpolatedAttributeToken interface
		if token, ok := currentToken.(interface {
			GetType() int
			GetParts() []string
		}); ok {
			if token.GetType() == 9 { // TokenTypeEncodedEntity is 9
				parts := token.GetParts()
				if len(parts) >= 2 {
					decoded := parts[0]
					encoded := parts[1]
					consumedInOriginalTemplate += len(encoded)
					consumedInInput += len(decoded)
				}
			} else {
				lengthOfParts := 0
				for _, part := range token.GetParts() {
					lengthOfParts += len(part)
				}
				consumedInInput += lengthOfParts
				consumedInOriginalTemplate += lengthOfParts
			}
		}
		offsetMap[consumedInInput] = consumedInOriginalTemplate
		tokenIndex++
	}
	return offsetMap
}
