package ml_parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/microsoft/typescript-go/angular-packages/compiler/chars"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
)

type TokenizeResult struct {
	Tokens                      []Token
	Errors                      []*parse_util.ParseError
	NonNormalizedIcuExpressions []Token
}

type LexerRange struct {
	StartPos  int
	StartLine int
	StartCol  int
	EndPos    int
}

type TokenizeOptions struct {
	TokenizeExpansionForms         bool
	Range                          *LexerRange
	EscapedString                  bool
	I18nNormalizeLineEndingsInICUs bool
	LeadingTriviaChars             []string
	PreserveLineEndings            bool
	TokenizeBlocks                 *bool
	TokenizeLet                    *bool
	SelectorlessEnabled            bool
}

func Tokenize(source string, url string, getTagDefinition func(string) tags.TagDefinition, options *TokenizeOptions) *TokenizeResult {
	if options == nil {
		options = &TokenizeOptions{}
	}
	file := parse_util.NewParseSourceFile(source, url)
	tokenizer := newTokenizer(file, getTagDefinition, options)
	tokenizer.tokenize()
	return &TokenizeResult{
		Tokens:                      mergeTextTokens(tokenizer.tokens),
		Errors:                      tokenizer.errors,
		NonNormalizedIcuExpressions: tokenizer.nonNormalizedIcuExpressions,
	}
}

var crOrCrlfRegexp = regexp.MustCompile(`\r\n?`)

func unexpectedCharacterErrorMsg(charCode int) string {
	char := "EOF"
	if charCode != chars.EOF {
		char = string(rune(charCode))
	}
	return fmt.Sprintf("Unexpected character \"%s\"", char)
}

func unknownEntityErrorMsg(entitySrc string) string {
	return fmt.Sprintf("Unknown entity \"%s\" - use the \"&#<decimal>;\" or  \"&#x<hex>;\" syntax", entitySrc)
}

func unparsableEntityErrorMsg(entityType string, entityStr string) string {
	return fmt.Sprintf("Unable to parse entity \"%s\" - %s character reference entities must end with \";\"", entityStr, entityType)
}

var supportedBlocks = []string{
	"@if", "@else", "@for", "@switch", "@case", "@default",
	"@empty", "@defer", "@placeholder", "@loading", "@error",
}

const interpolationStart = "{{"
const interpolationEnd = "}}"

type tokenizer struct {
	cursor                         characterCursor
	tokenizeIcu                    bool
	leadingTriviaCodePoints        []int
	currentTokenStart              characterCursor
	currentTokenType               *TokenType
	expansionCaseStack             []TokenType
	openDirectiveCount             int
	inInterpolation                bool
	preserveLineEndings            bool
	i18nNormalizeLineEndingsInICUs bool
	tokenizeBlocks                 bool
	tokenizeLet                    bool
	selectorlessEnabled            bool
	tokens                         []Token
	errors                         []*parse_util.ParseError
	nonNormalizedIcuExpressions    []Token
	getTagDefinition               func(string) tags.TagDefinition
}

func newTokenizer(file *parse_util.ParseSourceFile, getTagDefinition func(string) tags.TagDefinition, options *TokenizeOptions) *tokenizer {
	var leadingTriviaCodePoints []int
	for _, c := range options.LeadingTriviaChars {
		if len(c) > 0 {
			leadingTriviaCodePoints = append(leadingTriviaCodePoints, int([]rune(c)[0]))
		}
	}

	rng := options.Range
	if rng == nil {
		rng = &LexerRange{
			EndPos:    len(file.Content),
			StartPos:  0,
			StartLine: 0,
			StartCol:  0,
		}
	}

	var cursor characterCursor
	if options.EscapedString {
		cursor = newEscapedCharacterCursor(file, rng)
	} else {
		cursor = newPlainCharacterCursor(file, rng)
	}

	t := &tokenizer{
		tokenizeIcu:                    options.TokenizeExpansionForms,
		leadingTriviaCodePoints:        leadingTriviaCodePoints,
		preserveLineEndings:            options.PreserveLineEndings,
		i18nNormalizeLineEndingsInICUs: options.I18nNormalizeLineEndingsInICUs,
		tokenizeBlocks:                 options.TokenizeBlocks == nil || *options.TokenizeBlocks,
		tokenizeLet:                    options.TokenizeLet == nil || *options.TokenizeLet,
		selectorlessEnabled:            options.SelectorlessEnabled,
		tokens:                         []Token{},
		errors:                         []*parse_util.ParseError{},
		nonNormalizedIcuExpressions:    []Token{},
		getTagDefinition:               getTagDefinition,
		cursor:                         cursor,
	}

	// Default true overrides
	// In Go, bool defaults to false, but ts defaults to true for tokenizeBlocks and Let
	// Since we don't have tri-state bool here, we assume true by default unless explicitly false?
	// The caller will provide options.

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.handleError(r)
				t.cursor = newPlainCharacterCursor(file, rng)
				t.cursor.(*plainCharacterCursor).state.peek = chars.EOF
			}
		}()
		t.cursor.init()
	}()

	return t
}

func (t *tokenizer) processCarriageReturns(content string) string {
	if t.preserveLineEndings {
		return content
	}
	return crOrCrlfRegexp.ReplaceAllString(content, "\n")
}

func (t *tokenizer) tokenize() {
	for t.cursor.peek() != chars.EOF {
		start := t.cursor.clone()
		// Try/catch logic is handled via panic/recover internally if needed, but in Go we just check errors.
		// To simulate try/catch in TS tokenizer:
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.handleError(r)
				}
			}()

			if t.attemptCharCode(chars.LT) {
				if t.attemptCharCode(chars.QUESTION) {
					t.consumeProcessingInstruction(start)
				} else if t.attemptCharCode(chars.BANG) {
					if t.attemptCharCode(chars.LBRACKET) {
						t.consumeCdata(start)
					} else if t.attemptCharCode(chars.MINUS) {
						t.consumeComment(start)
					} else {
						t.consumeDocType(start)
					}
				} else if t.attemptCharCode(chars.SLASH) {
					t.consumeTagClose(start)
				} else {
					t.consumeTagOpen(start)
				}
			} else if t.tokenizeLet && t.cursor.peek() == chars.AT && !t.inInterpolation && t.isLetStart() {
				t.consumeLetDeclaration(start)
			} else if t.tokenizeBlocks && t.isBlockStart() {
				t.consumeBlockStart(start)
			} else if t.tokenizeBlocks && !t.inInterpolation && !t.isInExpansionCase() && !t.isInExpansionForm() && t.attemptCharCode(chars.RBRACE) {
				t.consumeBlockEnd(start)
			} else if !(t.tokenizeIcu && t.tokenizeExpansionForm()) {
				t.consumeWithInterpolation(
					TokenTypeText,
					TokenTypeInterpolation,
					t.isTextEnd,
					t.isTagStart,
				)
			}
		}()
	}
	t.beginToken(TokenTypeEOF, t.cursor.clone())
	t.endToken([]string{}, nil)
}

func (t *tokenizer) getBlockName() string {
	spacesInNameAllowed := false
	nameCursor := t.cursor.clone()

	t.attemptCharCodeUntilFn(func(code int) bool {
		if chars.IsWhitespace(code) {
			return !spacesInNameAllowed
		}
		if isBlockNameChar(code) {
			spacesInNameAllowed = true
			return false
		}
		return true
	})
	return strings.TrimSpace(t.cursor.getChars(nameCursor))
}

func (t *tokenizer) consumeBlockStart(start characterCursor) {
	t.requireCharCode(chars.AT)
	t.beginToken(TokenTypeBlockOpenStart, start)
	t.endToken([]string{t.getBlockName()}, nil)
	startTokenIdx := len(t.tokens) - 1

	if t.cursor.peek() == chars.LPAREN {
		t.cursor.advance()
		t.consumeBlockParameters()
		t.attemptCharCodeUntilFn(isNotWhitespace)

		if t.attemptCharCode(chars.RPAREN) {
			t.attemptCharCodeUntilFn(isNotWhitespace)
		} else {
			t.tokens[startTokenIdx].Type = TokenTypeIncompleteBlockOpen
			return
		}
	}

	if t.tokens[startTokenIdx].Parts[0] == "default never" && t.attemptCharCode(chars.SEMICOLON) {
		t.beginToken(TokenTypeBlockOpenEnd, t.cursor.clone())
		t.endToken([]string{}, nil)
		t.beginToken(TokenTypeBlockClose, t.cursor.clone())
		t.endToken([]string{}, nil)
		return
	}

	if t.attemptCharCode(chars.LBRACE) {
		t.beginToken(TokenTypeBlockOpenEnd, t.cursor.clone())
		t.endToken([]string{}, nil)
	} else if t.isBlockStart() && (t.tokens[startTokenIdx].Parts[0] == "case" || t.tokens[startTokenIdx].Parts[0] == "default") {
		t.beginToken(TokenTypeBlockOpenEnd, t.cursor.clone())
		t.endToken([]string{}, nil)
		t.beginToken(TokenTypeBlockClose, t.cursor.clone())
		t.endToken([]string{}, nil)
	} else {
		t.tokens[startTokenIdx].Type = TokenTypeIncompleteBlockOpen
	}
}

func (t *tokenizer) consumeBlockEnd(start characterCursor) {
	t.beginToken(TokenTypeBlockClose, start)
	t.endToken([]string{}, nil)
}

func (t *tokenizer) consumeBlockParameters() {
	t.attemptCharCodeUntilFn(isBlockParameterChar)

	for t.cursor.peek() != chars.RPAREN && t.cursor.peek() != chars.EOF {
		t.beginToken(TokenTypeBlockParameter, t.cursor.clone())
		start := t.cursor.clone()
		var inQuote *int = nil
		openParens := 0

		for (t.cursor.peek() != chars.SEMICOLON && t.cursor.peek() != chars.EOF) || inQuote != nil {
			char := t.cursor.peek()

			if char == chars.EOF {
				panic(t.createError(unexpectedCharacterErrorMsg(chars.EOF), t.cursor.getSpan(t.cursor.clone(), nil)))
			}

			if char == chars.BACKSLASH {
				t.cursor.advance()
			} else if inQuote != nil && char == *inQuote {
				inQuote = nil
			} else if inQuote == nil && chars.IsQuote(char) {
				inQuote = &char
			} else if char == chars.LPAREN && inQuote == nil {
				openParens++
			} else if char == chars.RPAREN && inQuote == nil {
				if openParens == 0 {
					break
				} else if openParens > 0 {
					openParens--
				}
			}
			t.cursor.advance()
		}

		t.endToken([]string{t.cursor.getChars(start)}, nil)
		t.attemptCharCodeUntilFn(isBlockParameterChar)
	}
}

func (t *tokenizer) consumeLetDeclaration(start characterCursor) {
	t.requireStr("@let")
	t.beginToken(TokenTypeLetStart, start)

	if chars.IsWhitespace(t.cursor.peek()) {
		t.attemptCharCodeUntilFn(isNotWhitespace)
	} else {
		tokenIdx := len(t.tokens)
		t.endToken([]string{t.cursor.getChars(start)}, nil)
		t.tokens[tokenIdx].Type = TokenTypeIncompleteLet
		return
	}

	startTokenIdx := len(t.tokens)
	t.endToken([]string{t.getLetDeclarationName()}, nil)

	t.attemptCharCodeUntilFn(isNotWhitespace)

	if !t.attemptCharCode(chars.EQ) {
		t.tokens[startTokenIdx].Type = TokenTypeIncompleteLet
		return
	}

	t.attemptCharCodeUntilFn(func(code int) bool {
		return isNotWhitespace(code) && !chars.IsNewLine(code)
	})
	t.consumeLetDeclarationValue()

	endChar := t.cursor.peek()
	if endChar == chars.SEMICOLON {
		t.beginToken(TokenTypeLetEnd, t.cursor.clone())
		t.cursor.advance()
		t.endToken([]string{}, nil)
	} else {
		t.tokens[startTokenIdx].Type = TokenTypeIncompleteLet
		t.tokens[startTokenIdx].SourceSpan = t.cursor.getSpan(start, nil)
	}
}

func (t *tokenizer) getLetDeclarationName() string {
	nameCursor := t.cursor.clone()
	allowDigit := false

	t.attemptCharCodeUntilFn(func(code int) bool {
		if chars.IsAsciiLetter(code) || code == chars.Dollar || code == chars.Underscore || (allowDigit && chars.IsDigit(code)) {
			allowDigit = true
			return false
		}
		return true
	})

	return strings.TrimSpace(t.cursor.getChars(nameCursor))
}

func (t *tokenizer) consumeLetDeclarationValue() {
	start := t.cursor.clone()
	t.beginToken(TokenTypeLetValue, start)

	for t.cursor.peek() != chars.EOF {
		char := t.cursor.peek()

		if char == chars.SEMICOLON {
			break
		}

		if chars.IsQuote(char) {
			t.cursor.advance()
			t.attemptCharCodeUntilFn(func(inner int) bool {
				if inner == chars.EOF {
					panic(t.createError(unexpectedCharacterErrorMsg(chars.EOF), t.cursor.getSpan(t.cursor.clone(), nil)))
				}
				if inner == chars.BACKSLASH {
					t.cursor.advance()
					return false
				}
				return inner == char
			})
		}
		if t.cursor.peek() != chars.EOF {
			t.cursor.advance()
		}
	}

	t.endToken([]string{t.cursor.getChars(start)}, nil)
}

func (t *tokenizer) tokenizeExpansionForm() bool {
	if t.isExpansionFormStart() {
		t.consumeExpansionFormStart()
		return true
	}

	if isExpansionCaseStart(t.cursor.peek()) && t.isInExpansionForm() {
		t.consumeExpansionCaseStart()
		return true
	}

	if t.cursor.peek() == chars.RBRACE {
		if t.isInExpansionCase() {
			t.consumeExpansionCaseEnd()
			return true
		}

		if t.isInExpansionForm() {
			t.consumeExpansionFormEnd()
			return true
		}
	}
	return false
}

func (t *tokenizer) beginToken(tokenType TokenType, start characterCursor) {
	t.currentTokenStart = start
	t.currentTokenType = &tokenType
}

func (t *tokenizer) endToken(parts []string, end characterCursor) *Token {
	if t.currentTokenStart == nil {
		panic(parse_util.NewParseError(t.cursor.getSpan(end, nil), "Programming error - attempted to end a token when there was no start to the token", nil, nil))
	}
	if t.currentTokenType == nil {
		panic(parse_util.NewParseError(t.cursor.getSpan(t.currentTokenStart, nil), "Programming error - attempted to end a token which has no token type", nil, nil))
	}

	actualEnd := end
	if actualEnd == nil {
		actualEnd = t.cursor
	}

	token := Token{
		Type:       *t.currentTokenType,
		Parts:      parts,
		SourceSpan: actualEnd.getSpan(t.currentTokenStart, t.leadingTriviaCodePoints),
	}
	t.tokens = append(t.tokens, token)
	t.currentTokenStart = nil
	t.currentTokenType = nil
	return &t.tokens[len(t.tokens)-1]
}

func (t *tokenizer) createError(msg string, span *parse_util.ParseSourceSpan) *parse_util.ParseError {
	if t.isInExpansionForm() {
		msg += ` (Do you have an unescaped "{" in your template? Use "{{ '{' }}") to escape it.)`
	}
	err := parse_util.NewParseError(span, msg, nil, nil)
	t.currentTokenStart = nil
	t.currentTokenType = nil
	return err
}

func (t *tokenizer) handleError(e any) {
	if cursorErr, ok := e.(*cursorError); ok {
		e = t.createError(cursorErr.msg, t.cursor.getSpan(cursorErr.cursor, nil))
	}

	if parseErr, ok := e.(*parse_util.ParseError); ok {
		t.errors = append(t.errors, parseErr)
	} else {
		panic(e) // if it's not our known errors, re-panic
	}
}

func (t *tokenizer) attemptCharCode(charCode int) bool {
	if t.cursor.peek() == charCode {
		t.cursor.advance()
		return true
	}
	return false
}

func (t *tokenizer) attemptCharCodeCaseInsensitive(charCode int) bool {
	if compareCharCodeCaseInsensitive(t.cursor.peek(), charCode) {
		t.cursor.advance()
		return true
	}
	return false
}

func (t *tokenizer) requireCharCode(charCode int) {
	location := t.cursor.clone()
	if !t.attemptCharCode(charCode) {
		panic(t.createError(unexpectedCharacterErrorMsg(t.cursor.peek()), t.cursor.getSpan(location, nil)))
	}
}

func (t *tokenizer) attemptStr(charsStr string) bool {
	length := len(charsStr)
	if t.cursor.charsLeft() < length {
		return false
	}
	initialPosition := t.cursor.clone()
	for i := 0; i < length; i++ {
		if !t.attemptCharCode(int(charsStr[i])) {
			t.cursor = initialPosition
			return false
		}
	}
	return true
}

func (t *tokenizer) attemptStrCaseInsensitive(charsStr string) bool {
	for i := 0; i < len(charsStr); i++ {
		if !t.attemptCharCodeCaseInsensitive(int(charsStr[i])) {
			return false
		}
	}
	return true
}

func (t *tokenizer) requireStr(charsStr string) {
	location := t.cursor.clone()
	if !t.attemptStr(charsStr) {
		panic(t.createError(unexpectedCharacterErrorMsg(t.cursor.peek()), t.cursor.getSpan(location, nil)))
	}
}

func (t *tokenizer) attemptCharCodeUntilFn(predicate func(int) bool) {
	for !predicate(t.cursor.peek()) {
		t.cursor.advance()
	}
}

func (t *tokenizer) requireCharCodeUntilFn(predicate func(int) bool, length int) {
	start := t.cursor.clone()
	t.attemptCharCodeUntilFn(predicate)
	if t.cursor.diff(start) < length {
		panic(t.createError(unexpectedCharacterErrorMsg(t.cursor.peek()), t.cursor.getSpan(start, nil)))
	}
}

func (t *tokenizer) attemptUntilChar(char int) {
	for t.cursor.peek() != char && t.cursor.peek() != chars.EOF {
		t.cursor.advance()
	}
}

func (t *tokenizer) readChar() string {
	char := string(rune(t.cursor.peek()))
	t.cursor.advance()
	return char
}

func (t *tokenizer) peekStr(charsStr string) bool {
	length := len(charsStr)
	if t.cursor.charsLeft() < length {
		return false
	}
	cursor := t.cursor.clone()
	for i := 0; i < length; i++ {
		if cursor.peek() != int(charsStr[i]) {
			return false
		}
		cursor.advance()
	}
	return true
}

func (t *tokenizer) isBlockStart() bool {
	if t.cursor.peek() != chars.AT {
		return false
	}
	for _, blockName := range supportedBlocks {
		if t.peekStr(blockName) {
			return true
		}
	}
	return false
}

func (t *tokenizer) isLetStart() bool {
	return t.cursor.peek() == chars.AT && t.peekStr("@let")
}

func (t *tokenizer) consumeEntity(textTokenType TokenType) {
	t.beginToken(TokenTypeEncodedEntity, t.cursor.clone())
	start := t.cursor.clone()
	t.cursor.advance()

	if t.attemptCharCode(chars.HASH) {
		isHex := t.attemptCharCode(chars.Letter_x) || t.attemptCharCode(chars.Letter_X)
		codeStart := t.cursor.clone()
		t.attemptCharCodeUntilFn(isDigitEntityEnd)
		if t.cursor.peek() != chars.SEMICOLON {
			t.cursor.advance()
			entityType := "decimal"
			if isHex {
				entityType = "hexadecimal"
			}
			panic(t.createError(unparsableEntityErrorMsg(entityType, t.cursor.getChars(start)), t.cursor.getSpan(nil, nil)))
		}
		strNum := t.cursor.getChars(codeStart)
		t.cursor.advance()

		base := 10
		if isHex {
			base = 16
		}
		charCode, err := strconv.ParseInt(strNum, base, 32)
		if err != nil {
			panic(t.createError(unknownEntityErrorMsg(t.cursor.getChars(start)), t.cursor.getSpan(nil, nil)))
		}
		t.endToken([]string{string(rune(charCode)), t.cursor.getChars(start)}, nil)
	} else {
		nameStart := t.cursor.clone()
		t.attemptCharCodeUntilFn(isNamedEntityEnd)
		if t.cursor.peek() != chars.SEMICOLON {
			t.beginToken(textTokenType, start)
			t.cursor = nameStart
			t.endToken([]string{"&"}, nil)
		} else {
			name := t.cursor.getChars(nameStart)
			t.cursor.advance()
			char, ok := NAMED_ENTITIES[name]
			if !ok {
				panic(t.createError(unknownEntityErrorMsg(name), t.cursor.getSpan(start, nil)))
			}
			t.endToken([]string{char, fmt.Sprintf("&%s;", name)}, nil)
		}
	}
}

func (t *tokenizer) consumeRawText(consumeEntities bool, endMarkerPredicate func() bool) {
	tokenType := TokenTypeRawText
	if consumeEntities {
		tokenType = TokenTypeEscapableRawText
	}
	t.beginToken(tokenType, t.cursor.clone())
	parts := []string{}

	for {
		tagCloseStart := t.cursor.clone()
		foundEndMarker := endMarkerPredicate()
		t.cursor = tagCloseStart
		if foundEndMarker {
			break
		}
		if t.cursor.peek() == chars.EOF {
			break
		}
		if consumeEntities && t.cursor.peek() == chars.AMPERSAND {
			t.endToken([]string{t.processCarriageReturns(strings.Join(parts, ""))}, nil)
			parts = []string{}
			t.consumeEntity(TokenTypeEscapableRawText)
			t.beginToken(TokenTypeEscapableRawText, t.cursor.clone())
		} else {
			parts = append(parts, t.readChar())
		}
	}
	t.endToken([]string{t.processCarriageReturns(strings.Join(parts, ""))}, nil)
}

func (t *tokenizer) consumeComment(start characterCursor) {
	t.beginToken(TokenTypeCommentStart, start)
	t.requireCharCode(chars.MINUS)
	t.endToken([]string{}, nil)
	t.consumeRawText(false, func() bool { return t.attemptStr("-->") })
	t.beginToken(TokenTypeCommentEnd, t.cursor.clone())
	t.requireStr("-->")
	t.endToken([]string{}, nil)
}

func (t *tokenizer) consumeCdata(start characterCursor) {
	t.beginToken(TokenTypeCdataStart, start)
	t.requireStr("CDATA[")
	t.endToken([]string{}, nil)
	t.consumeRawText(false, func() bool { return t.attemptStr("]]>") })
	t.beginToken(TokenTypeCdataEnd, t.cursor.clone())
	t.requireStr("]]>")
	t.endToken([]string{}, nil)
}

func (t *tokenizer) consumeDocType(start characterCursor) {
	t.beginToken(TokenTypeDocType, start)
	contentStart := t.cursor.clone()
	quote := chars.EOF
	internalSubsetDepth := 0
	for {
		code := t.cursor.peek()
		if code == chars.EOF {
			panic(t.createError(unexpectedCharacterErrorMsg(chars.EOF), t.cursor.getSpan(contentStart, nil)))
		}
		if quote != chars.EOF {
			t.cursor.advance()
			if code == quote {
				quote = chars.EOF
			}
			continue
		}
		if code == chars.SQ || code == chars.DQ {
			quote = code
			t.cursor.advance()
			continue
		}
		if code == chars.LBRACKET {
			internalSubsetDepth++
			t.cursor.advance()
			continue
		}
		if code == chars.RBRACKET && internalSubsetDepth > 0 {
			internalSubsetDepth--
			t.cursor.advance()
			continue
		}
		if code == chars.GT && internalSubsetDepth == 0 {
			break
		}
		t.cursor.advance()
	}
	content := t.cursor.getChars(contentStart)
	t.cursor.advance()
	t.endToken([]string{content}, nil)
}

func (t *tokenizer) consumeProcessingInstruction(start characterCursor) {
	contentStart := t.cursor.clone()
	for {
		if t.cursor.peek() == chars.EOF {
			panic(t.createError(unexpectedCharacterErrorMsg(chars.EOF), t.cursor.getSpan(contentStart, nil)))
		}
		if t.attemptStr("?>") {
			return
		}
		t.cursor.advance()
	}
}

func (t *tokenizer) consumePrefixAndName(endPredicate func(int) bool) []string {
	nameOrPrefixStart := t.cursor.clone()
	prefix := ""
	for t.cursor.peek() != chars.COLON && !isPrefixEnd(t.cursor.peek()) {
		t.cursor.advance()
	}
	var nameStart characterCursor
	if t.cursor.peek() == chars.COLON {
		prefix = t.cursor.getChars(nameOrPrefixStart)
		t.cursor.advance()
		nameStart = t.cursor.clone()
	} else {
		nameStart = nameOrPrefixStart
	}
	lenRequired := 1
	if prefix == "" {
		lenRequired = 0
	}
	t.requireCharCodeUntilFn(endPredicate, lenRequired)
	name := t.cursor.getChars(nameStart)
	return []string{prefix, name}
}

func (t *tokenizer) consumeSingleLineComment() {
	t.attemptCharCodeUntilFn(func(code int) bool {
		return chars.IsNewLine(code) || code == chars.EOF
	})
	t.attemptCharCodeUntilFn(isNotWhitespace)
}

func (t *tokenizer) consumeMultiLineComment() {
	t.attemptCharCodeUntilFn(func(code int) bool {
		if code == chars.EOF {
			return true
		}
		if code == chars.STAR {
			next := t.cursor.clone()
			next.advance()
			return next.peek() == chars.SLASH
		}
		return false
	})
	if t.attemptStr("*/") {
		t.attemptCharCodeUntilFn(isNotWhitespace)
	}
}

func (t *tokenizer) consumeTagOpen(start characterCursor) {
	var tagName string
	var prefix string
	var closingTagName string
	openTokenIdx := -1 // index into t.tokens, -1 means not set

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*parse_util.ParseError); ok {
				if openTokenIdx >= 0 {
					if t.tokens[openTokenIdx].Type == TokenTypeComponentOpenStart {
						t.tokens[openTokenIdx].Type = TokenTypeIncompleteComponentOpen
					} else {
						t.tokens[openTokenIdx].Type = TokenTypeIncompleteTagOpen
					}
				} else {
					t.beginToken(TokenTypeText, start)
					t.endToken([]string{"<"}, nil)
				}
				panic(r)
			}
			panic(r)
		}
	}()

	if t.selectorlessEnabled && isSelectorlessNameStart(t.cursor.peek()) {
		t.consumeComponentOpenStart(start)
		openTokenIdx = len(t.tokens) - 1
		closingTagName = t.tokens[openTokenIdx].Parts[0]
		prefix = t.tokens[openTokenIdx].Parts[1]
		tagName = t.tokens[openTokenIdx].Parts[2]
		if prefix != "" {
			closingTagName += ":" + prefix
		}
		if tagName != "" {
			closingTagName += ":" + tagName
		}
		t.attemptCharCodeUntilFn(isNotWhitespace)
	} else {
		if !chars.IsAsciiLetter(t.cursor.peek()) {
			panic(t.createError(unexpectedCharacterErrorMsg(t.cursor.peek()), t.cursor.getSpan(start, nil)))
		}

		t.consumeTagOpenStart(start)
		openTokenIdx = len(t.tokens) - 1
		prefix = t.tokens[openTokenIdx].Parts[0]
		tagName = t.tokens[openTokenIdx].Parts[1]
		closingTagName = tagName
		t.attemptCharCodeUntilFn(isNotWhitespace)
	}

	for {
		if t.attemptStr("//") {
			t.consumeSingleLineComment()
			continue
		}

		if t.attemptStr("/*") {
			t.consumeMultiLineComment()
			continue
		}

		if isAttributeTerminator(t.cursor.peek()) {
			break
		}

		if t.selectorlessEnabled && t.cursor.peek() == chars.AT {
			s := t.cursor.clone()
			nameStart := s.clone()
			nameStart.advance()

			if isSelectorlessNameStart(nameStart.peek()) {
				t.consumeDirective(s, nameStart)
			}
		} else {
			t.consumeAttribute()
		}
	}

	if t.tokens[openTokenIdx].Type == TokenTypeComponentOpenStart {
		t.consumeComponentOpenEnd()
	} else {
		t.consumeTagOpenEnd()
	}

	contentTokenType := t.getTagDefinition(tagName).GetContentType(&prefix)

	openType := t.tokens[openTokenIdx].Type
	openParts := t.tokens[openTokenIdx].Parts
	if contentTokenType == tags.TagContentTypeRawText {
		t.consumeRawTextWithTagClose(openType, openParts, closingTagName, false)
	} else if contentTokenType == tags.TagContentTypeEscapableRawText {
		t.consumeRawTextWithTagClose(openType, openParts, closingTagName, true)
	}
}

func (t *tokenizer) consumeRawTextWithTagClose(openTokenType TokenType, openTokenParts []string, tagName string, consumeEntities bool) {
	t.consumeRawText(consumeEntities, func() bool {
		if !t.attemptCharCode(chars.LT) {
			return false
		}
		if !t.attemptCharCode(chars.SLASH) {
			return false
		}
		t.attemptCharCodeUntilFn(isNotWhitespace)
		if !t.attemptStrCaseInsensitive(tagName) {
			return false
		}
		t.attemptCharCodeUntilFn(isNotWhitespace)
		return t.attemptCharCode(chars.GT)
	})

	closeType := TokenTypeTagClose
	if openTokenType == TokenTypeComponentOpenStart {
		closeType = TokenTypeComponentClose
	}
	t.beginToken(closeType, t.cursor.clone())
	t.requireCharCodeUntilFn(func(code int) bool { return code == chars.GT }, 3)
	t.cursor.advance()
	t.endToken(openTokenParts, nil)
}

func (t *tokenizer) consumeTagOpenStart(start characterCursor) *Token {
	t.beginToken(TokenTypeTagOpenStart, start)
	parts := t.consumePrefixAndName(isNameEnd)
	return t.endToken(parts, nil)
}

func (t *tokenizer) consumeComponentOpenStart(start characterCursor) *Token {
	t.beginToken(TokenTypeComponentOpenStart, start)
	parts := t.consumeComponentName()
	return t.endToken(parts, nil)
}

func (t *tokenizer) consumeComponentName() []string {
	nameStart := t.cursor.clone()
	for isSelectorlessNameChar(t.cursor.peek()) {
		t.cursor.advance()
	}
	name := t.cursor.getChars(nameStart)
	prefix := ""
	tagName := ""
	if t.cursor.peek() == chars.COLON {
		t.cursor.advance()
		parts := t.consumePrefixAndName(isNameEnd)
		prefix = parts[0]
		tagName = parts[1]
	}
	return []string{name, prefix, tagName}
}

func (t *tokenizer) consumeAttribute() {
	t.consumeAttributeName()
	t.attemptCharCodeUntilFn(isNotWhitespace)
	if t.attemptCharCode(chars.EQ) {
		t.attemptCharCodeUntilFn(isNotWhitespace)
		t.consumeAttributeValue()
	}
	t.attemptCharCodeUntilFn(isNotWhitespace)
}

func (t *tokenizer) consumeAttributeName() {
	attrNameStart := t.cursor.peek()
	if attrNameStart == chars.SQ || attrNameStart == chars.DQ {
		panic(t.createError(unexpectedCharacterErrorMsg(attrNameStart), t.cursor.getSpan(nil, nil)))
	}
	t.beginToken(TokenTypeAttrName, t.cursor.clone())
	var nameEndPredicate func(int) bool

	if t.openDirectiveCount > 0 {
		openParens := 0
		nameEndPredicate = func(code int) bool {
			if t.openDirectiveCount > 0 {
				if code == chars.LPAREN {
					openParens++
				} else if code == chars.RPAREN {
					if openParens == 0 {
						return true
					}
					openParens--
				}
			}
			return isNameEnd(code)
		}
	} else if attrNameStart == chars.LBRACKET {
		openBrackets := 0
		nameEndPredicate = func(code int) bool {
			if code == chars.LBRACKET {
				openBrackets++
			} else if code == chars.RBRACKET {
				openBrackets--
			}
			if openBrackets <= 0 {
				return isNameEnd(code)
			}
			return chars.IsNewLine(code)
		}
	} else {
		nameEndPredicate = isNameEnd
	}

	prefixAndName := t.consumePrefixAndName(nameEndPredicate)
	t.endToken(prefixAndName, nil)
}

func (t *tokenizer) consumeAttributeValue() {
	if t.cursor.peek() == chars.SQ || t.cursor.peek() == chars.DQ {
		quoteChar := t.cursor.peek()
		t.consumeQuote(quoteChar)
		endPredicate := func() bool { return t.cursor.peek() == quoteChar }
		t.consumeWithInterpolation(
			TokenTypeAttrValueText,
			TokenTypeAttrValueInterpolation,
			endPredicate,
			endPredicate,
		)
		t.consumeQuote(quoteChar)
	} else {
		endPredicate := func() bool { return isNameEnd(t.cursor.peek()) }
		t.consumeWithInterpolation(
			TokenTypeAttrValueText,
			TokenTypeAttrValueInterpolation,
			endPredicate,
			endPredicate,
		)
	}
}

func (t *tokenizer) consumeQuote(quoteChar int) {
	t.beginToken(TokenTypeAttrQuote, t.cursor.clone())
	t.requireCharCode(quoteChar)
	t.endToken([]string{string(rune(quoteChar))}, nil)
}

func (t *tokenizer) consumeTagOpenEnd() {
	tokenType := TokenTypeTagOpenEnd
	if t.attemptCharCode(chars.SLASH) {
		tokenType = TokenTypeTagOpenEndVoid
	}
	t.beginToken(tokenType, t.cursor.clone())
	t.requireCharCode(chars.GT)
	t.endToken([]string{}, nil)
}

func (t *tokenizer) consumeComponentOpenEnd() {
	tokenType := TokenTypeComponentOpenEnd
	if t.attemptCharCode(chars.SLASH) {
		tokenType = TokenTypeComponentOpenEndVoid
	}
	t.beginToken(tokenType, t.cursor.clone())
	t.requireCharCode(chars.GT)
	t.endToken([]string{}, nil)
}

func (t *tokenizer) consumeTagClose(start characterCursor) {
	if t.selectorlessEnabled {
		clone := start.clone()
		for clone.peek() != chars.GT && !isSelectorlessNameStart(clone.peek()) {
			clone.advance()
		}
		if isSelectorlessNameStart(clone.peek()) {
			t.beginToken(TokenTypeComponentClose, start)
			parts := t.consumeComponentName()
			t.attemptCharCodeUntilFn(isNotWhitespace)
			t.requireCharCode(chars.GT)
			t.endToken(parts, nil)
			return
		}
	}

	t.beginToken(TokenTypeTagClose, start)
	t.attemptCharCodeUntilFn(isNotWhitespace)
	prefixAndName := t.consumePrefixAndName(isNameEnd)
	t.attemptCharCodeUntilFn(isNotWhitespace)
	t.requireCharCode(chars.GT)
	t.endToken(prefixAndName, nil)
}

func (t *tokenizer) consumeExpansionFormStart() {
	t.beginToken(TokenTypeExpansionFormStart, t.cursor.clone())
	t.requireCharCode(chars.LBRACE)
	t.endToken([]string{}, nil)

	t.expansionCaseStack = append(t.expansionCaseStack, TokenTypeExpansionFormStart)

	t.beginToken(TokenTypeRawText, t.cursor.clone())
	condition := t.readUntil(chars.COMMA)
	normalizedCondition := t.processCarriageReturns(condition)
	if t.i18nNormalizeLineEndingsInICUs {
		t.endToken([]string{normalizedCondition}, nil)
	} else {
		conditionToken := t.endToken([]string{condition}, nil)
		if normalizedCondition != condition {
			t.nonNormalizedIcuExpressions = append(t.nonNormalizedIcuExpressions, *conditionToken)
		}
	}
	t.requireCharCode(chars.COMMA)
	t.attemptCharCodeUntilFn(isNotWhitespace)

	t.beginToken(TokenTypeRawText, t.cursor.clone())
	tokenType := t.readUntil(chars.COMMA)
	t.endToken([]string{tokenType}, nil)
	t.requireCharCode(chars.COMMA)
	t.attemptCharCodeUntilFn(isNotWhitespace)
}

func (t *tokenizer) consumeExpansionCaseStart() {
	t.beginToken(TokenTypeExpansionCaseValue, t.cursor.clone())
	value := strings.TrimSpace(t.readUntil(chars.LBRACE))
	t.endToken([]string{value}, nil)
	t.attemptCharCodeUntilFn(isNotWhitespace)

	t.beginToken(TokenTypeExpansionCaseExpStart, t.cursor.clone())
	t.requireCharCode(chars.LBRACE)
	t.endToken([]string{}, nil)
	t.attemptCharCodeUntilFn(isNotWhitespace)

	t.expansionCaseStack = append(t.expansionCaseStack, TokenTypeExpansionCaseExpStart)
}

func (t *tokenizer) consumeExpansionCaseEnd() {
	t.beginToken(TokenTypeExpansionCaseExpEnd, t.cursor.clone())
	t.requireCharCode(chars.RBRACE)
	t.endToken([]string{}, nil)
	t.attemptCharCodeUntilFn(isNotWhitespace)

	t.expansionCaseStack = t.expansionCaseStack[:len(t.expansionCaseStack)-1]
}

func (t *tokenizer) consumeExpansionFormEnd() {
	t.beginToken(TokenTypeExpansionFormEnd, t.cursor.clone())
	t.requireCharCode(chars.RBRACE)
	t.endToken([]string{}, nil)

	t.expansionCaseStack = t.expansionCaseStack[:len(t.expansionCaseStack)-1]
}

func (t *tokenizer) consumeWithInterpolation(
	textTokenType TokenType,
	interpolationTokenType TokenType,
	endPredicate func() bool,
	endInterpolation func() bool,
) {
	t.beginToken(textTokenType, t.cursor.clone())
	parts := []string{}

	for !endPredicate() && t.cursor.peek() != chars.EOF {
		current := t.cursor.clone()
		if t.attemptStr(interpolationStart) {
			t.endToken([]string{t.processCarriageReturns(strings.Join(parts, ""))}, current)
			parts = []string{}
			t.consumeInterpolation(interpolationTokenType, current, endInterpolation)
			t.beginToken(textTokenType, t.cursor.clone())
		} else if t.cursor.peek() == chars.AMPERSAND {
			t.endToken([]string{t.processCarriageReturns(strings.Join(parts, ""))}, nil)
			parts = []string{}
			t.consumeEntity(textTokenType)
			t.beginToken(textTokenType, t.cursor.clone())
		} else {
			parts = append(parts, t.readChar())
		}
	}

	t.inInterpolation = false
	t.endToken([]string{t.processCarriageReturns(strings.Join(parts, ""))}, nil)
}

func (t *tokenizer) consumeInterpolation(
	interpolationTokenType TokenType,
	interpStart characterCursor,
	prematureEndPredicate func() bool,
) {
	parts := []string{}
	t.beginToken(interpolationTokenType, interpStart)
	parts = append(parts, interpolationStart)

	expressionStart := t.cursor.clone()
	var inQuote *int = nil
	inComment := false

	for t.cursor.peek() != chars.EOF {
		current := t.cursor.clone()

		if (inQuote == nil || *inQuote == t.cursor.peek()) && prematureEndPredicate != nil && prematureEndPredicate() {
			break
		}

		if t.isTagStart() {
			t.cursor = current
			parts = append(parts, t.getProcessedChars(expressionStart, current))
			t.endToken(parts, nil)
			return
		}

		if inQuote == nil {
			if t.attemptStr(interpolationEnd) {
				parts = append(parts, t.getProcessedChars(expressionStart, current))
				parts = append(parts, interpolationEnd)
				t.endToken(parts, nil)
				return
			} else if t.attemptStr("//") {
				inComment = true
			}
		}

		char := t.cursor.peek()
		t.cursor.advance()
		if char == chars.BACKSLASH {
			t.cursor.advance()
		} else if inQuote != nil && char == *inQuote {
			inQuote = nil
		} else if !inComment && inQuote == nil && chars.IsQuote(char) {
			inQuote = &char
		}
	}

	parts = append(parts, t.getProcessedChars(expressionStart, t.cursor))
	t.endToken(parts, nil)
}

func (t *tokenizer) consumeDirective(start characterCursor, nameStart characterCursor) {
	t.requireCharCode(chars.AT)

	for isSelectorlessNameChar(t.cursor.peek()) {
		t.cursor.advance()
	}

	t.beginToken(TokenTypeDirectiveName, start)
	name := t.cursor.getChars(nameStart)
	t.endToken([]string{name}, nil)
	t.attemptCharCodeUntilFn(isNotWhitespace)

	if t.cursor.peek() != chars.LPAREN {
		return
	}

	t.openDirectiveCount++
	t.beginToken(TokenTypeDirectiveOpen, t.cursor.clone())
	t.cursor.advance()
	t.endToken([]string{}, nil)
	t.attemptCharCodeUntilFn(isNotWhitespace)

	for !isAttributeTerminator(t.cursor.peek()) && t.cursor.peek() != chars.RPAREN {
		t.consumeAttribute()
	}

	t.attemptCharCodeUntilFn(isNotWhitespace)
	t.openDirectiveCount--

	if t.cursor.peek() != chars.RPAREN {
		if t.cursor.peek() == chars.GT || t.cursor.peek() == chars.SLASH {
			return
		}
		panic(t.createError(unexpectedCharacterErrorMsg(t.cursor.peek()), t.cursor.getSpan(start, nil)))
	}

	t.beginToken(TokenTypeDirectiveClose, t.cursor.clone())
	t.cursor.advance()
	t.endToken([]string{}, nil)
	t.attemptCharCodeUntilFn(isNotWhitespace)
}

func (t *tokenizer) getProcessedChars(start characterCursor, end characterCursor) string {
	return t.processCarriageReturns(end.getChars(start))
}

func (t *tokenizer) isTextEnd() bool {
	if t.isTagStart() || t.cursor.peek() == chars.EOF {
		return true
	}

	if t.tokenizeIcu && !t.inInterpolation {
		if t.isExpansionFormStart() {
			return true
		}
		if t.cursor.peek() == chars.RBRACE && t.isInExpansionCase() {
			return true
		}
	}

	if t.tokenizeBlocks && !t.inInterpolation && !t.isInExpansion() && (t.isBlockStart() || t.isLetStart() || t.cursor.peek() == chars.RBRACE) {
		return true
	}

	return false
}

func (t *tokenizer) isTagStart() bool {
	if t.cursor.peek() == chars.LT {
		tmp := t.cursor.clone()
		tmp.advance()
		code := tmp.peek()
		if (chars.Letter_a <= code && code <= chars.Letter_z) || (chars.Letter_A <= code && code <= chars.Letter_Z) || code == chars.SLASH || code == chars.BANG {
			return true
		}
	}
	return false
}

func (t *tokenizer) readUntil(char int) string {
	start := t.cursor.clone()
	t.attemptUntilChar(char)
	return t.cursor.getChars(start)
}

func (t *tokenizer) isInExpansion() bool {
	return t.isInExpansionCase() || t.isInExpansionForm()
}

func (t *tokenizer) isInExpansionCase() bool {
	return len(t.expansionCaseStack) > 0 && t.expansionCaseStack[len(t.expansionCaseStack)-1] == TokenTypeExpansionCaseExpStart
}

func (t *tokenizer) isInExpansionForm() bool {
	return len(t.expansionCaseStack) > 0 && t.expansionCaseStack[len(t.expansionCaseStack)-1] == TokenTypeExpansionFormStart
}

func (t *tokenizer) isExpansionFormStart() bool {
	if t.cursor.peek() != chars.LBRACE {
		return false
	}
	start := t.cursor.clone()
	isInterpol := t.attemptStr(interpolationStart)
	t.cursor = start
	return !isInterpol
}

func isNotWhitespace(code int) bool {
	return !chars.IsWhitespace(code) || code == chars.EOF
}

func isNameEnd(code int) bool {
	return chars.IsWhitespace(code) || code == chars.GT || code == chars.LT || code == chars.SLASH || code == chars.SQ || code == chars.DQ || code == chars.EQ || code == chars.EOF
}

func isPrefixEnd(code int) bool {
	return (code < chars.Letter_a || chars.Letter_z < code) && (code < chars.Letter_A || chars.Letter_Z < code) && (code < chars.Num0 || code > chars.Num9)
}

func isDigitEntityEnd(code int) bool {
	return code == chars.SEMICOLON || code == chars.EOF || !chars.IsAsciiHexDigit(code)
}

func isNamedEntityEnd(code int) bool {
	return code == chars.SEMICOLON || code == chars.EOF || !(chars.IsAsciiLetter(code) || chars.IsDigit(code))
}

func isExpansionCaseStart(peek int) bool {
	return peek != chars.RBRACE
}

func compareCharCodeCaseInsensitive(code1 int, code2 int) bool {
	return toUpperCaseCharCode(code1) == toUpperCaseCharCode(code2)
}

func toUpperCaseCharCode(code int) int {
	if code >= chars.Letter_a && code <= chars.Letter_z {
		return code - chars.Letter_a + chars.Letter_A
	}
	return code
}

func isBlockNameChar(code int) bool {
	return chars.IsAsciiLetter(code) || chars.IsDigit(code) || code == chars.Underscore
}

func isBlockParameterChar(code int) bool {
	return code != chars.SEMICOLON && isNotWhitespace(code)
}

func isSelectorlessNameStart(code int) bool {
	return code == chars.Underscore || (code >= chars.Letter_A && code <= chars.Letter_Z)
}

func isSelectorlessNameChar(code int) bool {
	return chars.IsAsciiLetter(code) || chars.IsDigit(code) || code == chars.Underscore
}

func isAttributeTerminator(code int) bool {
	return code == chars.SLASH || code == chars.GT || code == chars.LT || code == chars.EOF
}

func mergeTextTokens(srcTokens []Token) []Token {
	dstTokens := []Token{}
	var lastDstToken *Token = nil
	for i := 0; i < len(srcTokens); i++ {
		token := srcTokens[i]
		if (lastDstToken != nil && lastDstToken.Type == TokenTypeText && token.Type == TokenTypeText) ||
			(lastDstToken != nil && lastDstToken.Type == TokenTypeAttrValueText && token.Type == TokenTypeAttrValueText) {
			lastDstToken.Parts[0] += token.Parts[0]
			lastDstToken.SourceSpan.End = token.SourceSpan.End
		} else {
			dstTokens = append(dstTokens, token)
			lastDstToken = &dstTokens[len(dstTokens)-1]
		}
	}
	return dstTokens
}

type characterCursor interface {
	init()
	peek() int
	advance()
	getSpan(start characterCursor, leadingTriviaCodePoints []int) *parse_util.ParseSourceSpan
	getChars(start characterCursor) string
	charsLeft() int
	diff(other characterCursor) int
	clone() characterCursor
	getOffset() int
	getLine() int
	getColumn() int
}

type cursorState struct {
	peek   int
	offset int
	line   int
	column int
}

type cursorError struct {
	msg    string
	cursor characterCursor
}

func (e *cursorError) Error() string {
	return e.msg
}

type plainCharacterCursor struct {
	state cursorState
	file  *parse_util.ParseSourceFile
	input string
	end   int
}

func newPlainCharacterCursor(file *parse_util.ParseSourceFile, rng *LexerRange) *plainCharacterCursor {
	return &plainCharacterCursor{
		file:  file,
		input: file.Content,
		end:   rng.EndPos,
		state: cursorState{
			peek:   -1,
			offset: rng.StartPos,
			line:   rng.StartLine,
			column: rng.StartCol,
		},
	}
}

func (c *plainCharacterCursor) clone() characterCursor {
	return &plainCharacterCursor{
		file:  c.file,
		input: c.input,
		end:   c.end,
		state: cursorState{
			peek:   c.state.peek,
			offset: c.state.offset,
			line:   c.state.line,
			column: c.state.column,
		},
	}
}

func (c *plainCharacterCursor) init() {
	c.updatePeek(c.state.offset)
}

func (c *plainCharacterCursor) peek() int {
	return c.state.peek
}

func (c *plainCharacterCursor) charsLeft() int {
	return c.end - c.state.offset
}

func (c *plainCharacterCursor) diff(other characterCursor) int {
	return c.state.offset - other.getOffset()
}

func (c *plainCharacterCursor) getOffset() int {
	return c.state.offset
}
func (c *plainCharacterCursor) getLine() int {
	return c.state.line
}
func (c *plainCharacterCursor) getColumn() int {
	return c.state.column
}

func (c *plainCharacterCursor) advance() {
	c.advanceState(&c.state)
}

func (c *plainCharacterCursor) advanceState(state *cursorState) {
	if state.offset >= c.end {
		panic(parse_util.NewParseError(
			parse_util.NewParseSourceSpan(
				parse_util.NewParseLocation(c.file, state.offset, state.line, state.column),
				parse_util.NewParseLocation(c.file, state.offset, state.line, state.column),
				parse_util.NewParseLocation(c.file, state.offset, state.line, state.column),
				nil,
			),
			"Unexpected character \"EOF\"",
			nil,
			nil,
		))
	}
	currentChar := c.charAt(state.offset)
	width := c.charWidthAt(state.offset)
	if currentChar == chars.NewLine {
		state.line++
		state.column = 0
	} else if !chars.IsNewLine(currentChar) {
		state.column++
	}
	state.offset += width
	if state.offset >= c.end {
		state.peek = chars.EOF
	} else {
		state.peek = c.charAt(state.offset)
	}
}

func (c *plainCharacterCursor) updatePeek(offset int) {
	if offset >= c.end {
		c.state.peek = chars.EOF
	} else {
		c.state.peek = c.charAt(offset)
	}
}

func (c *plainCharacterCursor) charAt(pos int) int {
	r, _ := utf8.DecodeRuneInString(c.input[pos:])
	return int(r)
}

func (c *plainCharacterCursor) charWidthAt(pos int) int {
	_, width := utf8.DecodeRuneInString(c.input[pos:])
	if width <= 0 {
		return 1
	}
	return width
}

func (c *plainCharacterCursor) getSpan(start characterCursor, leadingTriviaCodePoints []int) *parse_util.ParseSourceSpan {
	var startOffset, startLine, startCol int
	if start != nil {
		startOffset = start.getOffset()
		startLine = start.getLine()
		startCol = start.getColumn()
	} else {
		startOffset = c.getOffset()
		startLine = c.getLine()
		startCol = c.getColumn()
	}
	fullStart := startOffset
	fullStartLine := startLine
	fullStartCol := startCol
	if leadingTriviaCodePoints != nil {
		for startOffset < c.end && startOffset < c.state.offset {
			if contains(leadingTriviaCodePoints, c.charAt(startOffset)) {
				ch := c.charAt(startOffset)
				startOffset++
				if ch == chars.NewLine {
					startLine++
					startCol = 0
				} else {
					startCol++
				}
			} else {
				break
			}
		}
	}

	startLoc := parse_util.NewParseLocation(c.file, startOffset, startLine, startCol)
	endLoc := parse_util.NewParseLocation(c.file, c.state.offset, c.state.line, c.state.column)
	fullStartLoc := parse_util.NewParseLocation(c.file, fullStart, fullStartLine, fullStartCol)
	return parse_util.NewParseSourceSpan(startLoc, endLoc, fullStartLoc, nil)
}

func (c *plainCharacterCursor) getChars(start characterCursor) string {
	return c.input[start.getOffset():c.state.offset]
}

func (c *escapedCharacterCursor) diff(other characterCursor) int {
	return c.state.offset - other.getOffset()
}

func (c *escapedCharacterCursor) getOffset() int {
	return c.state.offset
}
func (c *escapedCharacterCursor) getLine() int {
	return c.state.line
}
func (c *escapedCharacterCursor) getColumn() int {
	return c.state.column
}

func (c *escapedCharacterCursor) peek() int {
	return c.state.peek
}

func (c *escapedCharacterCursor) getSpan(start characterCursor, leadingTriviaCodePoints []int) *parse_util.ParseSourceSpan {
	var startOffset, startLine, startCol int
	if start != nil {
		startOffset = start.getOffset()
		startLine = start.getLine()
		startCol = start.getColumn()
	} else {
		startOffset = c.getOffset()
		startLine = c.getLine()
		startCol = c.getColumn()
	}
	fullStart := startOffset
	fullStartLine := startLine
	fullStartCol := startCol
	if leadingTriviaCodePoints != nil {
		for startOffset < c.end && startOffset < c.getOffset() {
			if contains(leadingTriviaCodePoints, c.charAt(startOffset)) {
				ch := c.charAt(startOffset)
				startOffset++
				if ch == chars.NewLine {
					startLine++
					startCol = 0
				} else {
					startCol++
				}
			} else {
				break
			}
		}
	}

	startLoc := parse_util.NewParseLocation(c.file, startOffset, startLine, startCol)
	endLoc := parse_util.NewParseLocation(c.file, c.getOffset(), c.getLine(), c.getColumn())
	fullStartLoc := parse_util.NewParseLocation(c.file, fullStart, fullStartLine, fullStartCol)
	return parse_util.NewParseSourceSpan(startLoc, endLoc, fullStartLoc, nil)
}

func contains(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

type escapedCharacterCursor struct {
	plainCharacterCursor
	internalState cursorState
}

func newEscapedCharacterCursor(file *parse_util.ParseSourceFile, rng *LexerRange) *escapedCharacterCursor {
	return &escapedCharacterCursor{
		plainCharacterCursor: *newPlainCharacterCursor(file, rng),
		internalState: cursorState{
			peek:   -1,
			offset: rng.StartPos,
			line:   rng.StartLine,
			column: rng.StartCol,
		},
	}
}

func (c *escapedCharacterCursor) clone() characterCursor {
	return &escapedCharacterCursor{
		plainCharacterCursor: *c.plainCharacterCursor.clone().(*plainCharacterCursor),
		internalState: cursorState{
			peek:   c.internalState.peek,
			offset: c.internalState.offset,
			line:   c.internalState.line,
			column: c.internalState.column,
		},
	}
}

func (c *escapedCharacterCursor) advance() {
	c.state = c.internalState
	c.plainCharacterCursor.advance()
	c.internalState = c.state
	c.processEscapeSequence()
}

func (c *escapedCharacterCursor) init() {
	c.plainCharacterCursor.init()
	c.internalState = c.state
	c.processEscapeSequence()
}

func (c *escapedCharacterCursor) processEscapeSequence() {
	peek := func() int { return c.internalState.peek }

	if peek() == chars.BACKSLASH {
		c.plainCharacterCursor.advanceState(&c.internalState)
		peek2 := peek()
		if peek2 == chars.Letter_n {
			c.state.peek = chars.NewLine
		} else if peek2 == chars.Letter_r {
			c.state.peek = chars.Return
		} else if peek2 == chars.Letter_v {
			c.state.peek = chars.VTAB
		} else if peek2 == chars.Letter_t {
			c.state.peek = chars.TAB
		} else if peek2 == chars.Letter_b {
			c.state.peek = chars.BSPACE
		} else if peek2 == chars.Letter_f {
			c.state.peek = chars.FF
		} else if peek2 == chars.Letter_u {
			c.plainCharacterCursor.advanceState(&c.internalState)
			if c.internalState.peek == chars.LBRACE {
				c.plainCharacterCursor.advanceState(&c.internalState)
				c.state.peek = c.readHexVariable()
			} else {
				c.state.peek = c.readHex(4)
			}
		} else if peek2 == chars.Letter_x {
			c.plainCharacterCursor.advanceState(&c.internalState)
			hexCode := c.readHex(2)
			c.state.peek = hexCode
		} else if peek2 >= '0' && peek2 <= '7' {
			c.state.peek = c.readOctal()
		} else if chars.IsNewLine(peek2) {
			c.plainCharacterCursor.advanceState(&c.internalState)
			c.state = c.internalState
		} else if peek2 == chars.EOF {
			panic(&cursorError{msg: "Unexpected character \"EOF\"", cursor: c.clone()})
		} else {
			c.state.peek = peek2
		}
	}
}

func (c *escapedCharacterCursor) getChars(start characterCursor) string {
	cursor := start.clone()
	var charsStr strings.Builder
	for cursor.(*escapedCharacterCursor).internalState.offset < c.internalState.offset {
		charsStr.WriteRune(rune(cursor.peek()))
		cursor.advance()
	}
	return charsStr.String()
}

func (c *escapedCharacterCursor) readHex(length int) int {
	start := c.clone().(*escapedCharacterCursor)
	var hexStr strings.Builder
	for i := 0; i < length; i++ {
		if c.internalState.peek == chars.EOF {
			c.state = c.internalState
			panic(&cursorError{msg: "Unexpected character \"EOF\"", cursor: c.clone()})
		}
		hexStr.WriteRune(rune(c.internalState.peek))
		if i < length-1 {
			c.plainCharacterCursor.advanceState(&c.internalState)
		}
	}
	hexCode, err := strconv.ParseInt(hexStr.String(), 16, 32)
	if err != nil {
		start.state = start.internalState
		panic(&cursorError{msg: "Invalid hexadecimal escape sequence", cursor: start})
	}
	return int(hexCode)
}

func (c *escapedCharacterCursor) readHexVariable() int {
	start := c.clone().(*escapedCharacterCursor)
	var hexStr strings.Builder
	for c.internalState.peek != chars.RBRACE {
		if c.internalState.peek == chars.EOF {
			c.state = c.internalState
			panic(&cursorError{msg: "Unexpected character \"EOF\"", cursor: c.clone()})
		}
		hexStr.WriteRune(rune(c.internalState.peek))
		c.plainCharacterCursor.advanceState(&c.internalState)
	}
	// We intentionally leave internalState at RBRACE so advance() will consume it
	hexCode, err := strconv.ParseInt(hexStr.String(), 16, 32)
	if err != nil {
		start.state = start.internalState
		panic(&cursorError{msg: "Invalid hexadecimal escape sequence", cursor: start})
	}
	return int(hexCode)
}

func (c *escapedCharacterCursor) readOctal() int {
	var octalStr strings.Builder
	octalStr.WriteRune(rune(c.internalState.peek))

	for i := 1; i < 3; i++ {
		nextOffset := c.internalState.offset + 1
		var nextChar int
		if nextOffset >= c.end {
			nextChar = chars.EOF
		} else {
			nextChar = c.plainCharacterCursor.charAt(nextOffset)
		}

		if nextChar >= '0' && nextChar <= '7' {
			c.plainCharacterCursor.advanceState(&c.internalState)
			octalStr.WriteRune(rune(c.internalState.peek))
		} else {
			break
		}
	}

	octalCode, _ := strconv.ParseInt(octalStr.String(), 8, 32)
	return int(octalCode)
}
