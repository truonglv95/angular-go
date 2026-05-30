package expression_parser

import (
	"fmt"
	"strconv"
	"strings"
)

type TokenType int

const (
	TokenTypeCharacter TokenType = iota
	TokenTypeIdentifier
	TokenTypePrivateIdentifier
	TokenTypeKeyword
	TokenTypeString
	TokenTypeOperator
	TokenTypeNumber
	TokenTypeRegExpBody
	TokenTypeRegExpFlags
	TokenTypeError
)

type StringTokenKind int

const (
	StringTokenKindPlain StringTokenKind = iota
	StringTokenKindTemplateLiteralPart
	StringTokenKindTemplateLiteralEnd
)

var KEYWORDS = []string{
	"var", "let", "as", "null", "undefined", "true", "false",
	"if", "else", "this", "typeof", "void", "in", "instanceof",
}

type Token struct {
	Index    int
	End      int
	Type     TokenType
	NumValue float64
	StrValue string
	Kind     StringTokenKind // Only for String tokens
}

func (t Token) IsCharacter(code int) bool {
	return t.Type == TokenTypeCharacter && int(t.NumValue) == code
}

func (t Token) IsNumber() bool {
	return t.Type == TokenTypeNumber
}

func (t Token) IsString() bool {
	return t.Type == TokenTypeString
}

func (t Token) IsOperator(operator string) bool {
	return t.Type == TokenTypeOperator && t.StrValue == operator
}

func (t Token) IsIdentifier() bool {
	return t.Type == TokenTypeIdentifier
}

func (t Token) IsPrivateIdentifier() bool {
	return t.Type == TokenTypePrivateIdentifier
}

func (t Token) IsKeyword() bool {
	return t.Type == TokenTypeKeyword
}

func (t Token) IsKeywordLet() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "let"
}

func (t Token) IsKeywordAs() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "as"
}

func (t Token) IsKeywordNull() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "null"
}

func (t Token) IsKeywordUndefined() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "undefined"
}

func (t Token) IsKeywordTrue() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "true"
}

func (t Token) IsKeywordFalse() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "false"
}

func (t Token) IsKeywordThis() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "this"
}

func (t Token) IsKeywordTypeof() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "typeof"
}

func (t Token) IsKeywordVoid() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "void"
}

func (t Token) IsKeywordIn() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "in"
}

func (t Token) IsKeywordInstanceOf() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "instanceof"
}

func (t Token) IsError() bool {
	return t.Type == TokenTypeError
}

func (t Token) IsRegExpBody() bool {
	return t.Type == TokenTypeRegExpBody
}

func (t Token) IsRegExpFlags() bool {
	return t.Type == TokenTypeRegExpFlags
}

func (t Token) ToNumber() float64 {
	if t.Type == TokenTypeNumber {
		return t.NumValue
	}
	return -1
}

func (t Token) IsTemplateLiteralPart() bool {
	return t.IsString() && t.Kind == StringTokenKindTemplateLiteralPart
}

func (t Token) IsTemplateLiteralEnd() bool {
	return t.IsString() && t.Kind == StringTokenKindTemplateLiteralEnd
}

func (t Token) IsTemplateLiteralInterpolationStart() bool {
	return t.IsOperator("${")
}

func (t Token) ToString() *string {
	switch t.Type {
	case TokenTypeCharacter, TokenTypeIdentifier, TokenTypeKeyword, TokenTypeOperator,
		TokenTypePrivateIdentifier, TokenTypeString, TokenTypeError, TokenTypeRegExpBody, TokenTypeRegExpFlags:
		return &t.StrValue
	case TokenTypeNumber:
		str := fmt.Sprintf("%g", t.NumValue)
		return &str
	default:
		return nil
	}
}

func newStringToken(index int, end int, strValue string, kind StringTokenKind) Token {
	return Token{Index: index, End: end, Type: TokenTypeString, NumValue: 0, StrValue: strValue, Kind: kind}
}

func newCharacterToken(index int, end int, code int) Token {
	return Token{Index: index, End: end, Type: TokenTypeCharacter, NumValue: float64(code), StrValue: string(rune(code))}
}

func newIdentifierToken(index int, end int, text string) Token {
	return Token{Index: index, End: end, Type: TokenTypeIdentifier, NumValue: 0, StrValue: text}
}

func newPrivateIdentifierToken(index int, end int, text string) Token {
	return Token{Index: index, End: end, Type: TokenTypePrivateIdentifier, NumValue: 0, StrValue: text}
}

func newKeywordToken(index int, end int, text string) Token {
	return Token{Index: index, End: end, Type: TokenTypeKeyword, NumValue: 0, StrValue: text}
}

func newOperatorToken(index int, end int, text string) Token {
	return Token{Index: index, End: end, Type: TokenTypeOperator, NumValue: 0, StrValue: text}
}

func newNumberToken(index int, end int, n float64) Token {
	return Token{Index: index, End: end, Type: TokenTypeNumber, NumValue: n, StrValue: ""}
}

func newErrorToken(index int, end int, message string) Token {
	return Token{Index: index, End: end, Type: TokenTypeError, NumValue: 0, StrValue: message}
}

func newRegExpBodyToken(index int, end int, text string) Token {
	return Token{Index: index, End: end, Type: TokenTypeRegExpBody, NumValue: 0, StrValue: text}
}

func newRegExpFlagsToken(index int, end int, text string) Token {
	return Token{Index: index, End: end, Type: TokenTypeRegExpFlags, NumValue: 0, StrValue: text}
}

var EOF = Token{Index: -1, End: -1, Type: TokenTypeCharacter, NumValue: 0, StrValue: ""}

type Lexer struct{}

func (l *Lexer) Tokenize(text string) []Token {
	scanner := newScanner(text)
	return scanner.scan()
}

type scanner struct {
	input      string
	length     int
	peek       int
	index      int
	braceStack []string
	tokens     []Token
}

func newScanner(input string) *scanner {
	s := &scanner{
		input:      input,
		length:     len(input),
		index:      -1,
		braceStack: []string{},
		tokens:     []Token{},
	}
	s.advance()
	return s
}

func (s *scanner) scan() []Token {
	token := s.scanToken()
	for token != nil {
		s.tokens = append(s.tokens, *token)
		token = s.scanToken()
	}
	return s.tokens
}

func (s *scanner) advance() {
	s.index++
	if s.index >= s.length {
		s.peek = 0 // $EOF
	} else {
		s.peek = int(s.input[s.index])
	}
}

func (s *scanner) scanToken() *Token {
	input := s.input
	length := s.length
	peek := s.peek
	index := s.index

	// Skip whitespace.
	for peek <= ' ' {
		index++
		if index >= length {
			peek = 0 // $EOF
			break
		} else {
			peek = int(input[index])
		}
	}

	s.peek = peek
	s.index = index

	if index >= length {
		return nil
	}

	// Handle identifiers and numbers.
	if isIdentifierStart(peek) {
		t := s.scanIdentifier()
		return &t
	}

	if isDigit(peek) {
		t := s.scanNumber(index)
		return &t
	}

	start := index
	switch peek {
	case '.':
		s.advance()
		if isDigit(s.peek) {
			t := s.scanNumber(start)
			return &t
		}
		if s.peek != '.' {
			t := newCharacterToken(start, s.index, '.')
			return &t
		}
		s.advance()
		if s.peek == '.' {
			s.advance()
			t := newOperatorToken(start, s.index, "...")
			return &t
		}
		t := s.error(fmt.Sprintf("Unexpected character [%c]", peek), 0)
		return &t
	case '(', ')', '[', ']', ',', ':', ';':
		t := s.scanCharacter(start, peek)
		return &t
	case '{':
		t := s.scanOpenBrace(start, peek)
		return &t
	case '}':
		t := s.scanCloseBrace(start, peek)
		return &t
	case '\'', '"':
		t := s.scanString()
		return &t
	case '`':
		s.advance()
		t := s.scanTemplateLiteralPart(start)
		return &t
	case '#':
		t := s.scanPrivateIdentifier()
		return &t
	case '+':
		t := s.scanComplexOperator(start, "+", '=', "=", 0, "")
		return &t
	case '-':
		t := s.scanComplexOperator(start, "-", '=', "=", 0, "")
		return &t
	case '/':
		if s.isStartOfRegex() {
			t := s.scanRegex(index)
			return &t
		}
		t := s.scanComplexOperator(start, "/", '=', "=", 0, "")
		return &t
	case '%':
		t := s.scanComplexOperator(start, "%", '=', "=", 0, "")
		return &t
	case '^':
		t := s.scanOperator(start, "^")
		return &t
	case '*':
		t := s.scanStar(start)
		return &t
	case '?':
		t := s.scanQuestion(start)
		return &t
	case '<', '>':
		t := s.scanComplexOperator(start, string(rune(peek)), '=', "=", 0, "")
		return &t
	case '!':
		t := s.scanComplexOperator(start, "!", '=', "=", '=', "=")
		return &t
	case '=':
		t := s.scanEquals(start)
		return &t
	case '&':
		t := s.scanComplexOperator(start, "&", '&', "&", '=', "=")
		return &t
	case '|':
		t := s.scanComplexOperator(start, "|", '|', "|", '=', "=")
		return &t
	case 160: // $NBSP
		for isWhitespace(s.peek) {
			s.advance()
		}
		return s.scanToken()
	}

	s.advance()
	t := s.error(fmt.Sprintf("Unexpected character [%c]", peek), 0)
	return &t
}

func (s *scanner) scanCharacter(start int, code int) Token {
	s.advance()
	return newCharacterToken(start, s.index, code)
}

func (s *scanner) scanOperator(start int, str string) Token {
	s.advance()
	return newOperatorToken(start, s.index, str)
}

func (s *scanner) scanOpenBrace(start int, code int) Token {
	s.braceStack = append(s.braceStack, "expression")
	s.advance()
	return newCharacterToken(start, s.index, code)
}

func (s *scanner) scanCloseBrace(start int, code int) Token {
	s.advance()

	var currentBrace string
	if len(s.braceStack) > 0 {
		currentBrace = s.braceStack[len(s.braceStack)-1]
		s.braceStack = s.braceStack[:len(s.braceStack)-1]
	}

	if currentBrace == "interpolation" {
		s.tokens = append(s.tokens, newCharacterToken(start, s.index, '}'))
		return s.scanTemplateLiteralPart(s.index)
	}

	return newCharacterToken(start, s.index, code)
}

func (s *scanner) scanComplexOperator(start int, one string, twoCode int, two string, threeCode int, three string) Token {
	s.advance()
	str := one
	if s.peek == twoCode {
		s.advance()
		str += two
	}
	if threeCode != 0 && s.peek == threeCode {
		s.advance()
		str += three
	}
	return newOperatorToken(start, s.index, str)
}

func (s *scanner) scanEquals(start int) Token {
	s.advance()
	str := "="
	if s.peek == '=' {
		s.advance()
		str += "="
	} else if s.peek == '>' {
		s.advance()
		str += ">"
		return newOperatorToken(start, s.index, str)
	}
	if s.peek == '=' {
		s.advance()
		str += "="
	}
	return newOperatorToken(start, s.index, str)
}

func (s *scanner) scanIdentifier() Token {
	start := s.index
	s.advance()
	for isIdentifierPart(s.peek) {
		s.advance()
	}
	str := s.input[start:s.index]
	for _, kw := range KEYWORDS {
		if kw == str {
			return newKeywordToken(start, s.index, str)
		}
	}
	return newIdentifierToken(start, s.index, str)
}

func (s *scanner) scanPrivateIdentifier() Token {
	start := s.index
	s.advance()
	if !isIdentifierStart(s.peek) {
		return s.error("Invalid character [#]", -1)
	}
	for isIdentifierPart(s.peek) {
		s.advance()
	}
	identifierName := s.input[start:s.index]
	return newPrivateIdentifierToken(start, s.index, identifierName)
}

func (s *scanner) scanNumber(start int) Token {
	simple := s.index == start
	hasSeparators := false
	s.advance() // Skip initial digit.
	for {
		if isDigit(s.peek) {
			// Do nothing.
		} else if s.peek == '_' {
			if s.index-1 < 0 || s.index+1 >= s.length || !isDigit(int(s.input[s.index-1])) || !isDigit(int(s.input[s.index+1])) {
				return s.error("Invalid numeric separator", 0)
			}
			hasSeparators = true
		} else if s.peek == '.' {
			simple = false
		} else if isExponentStart(s.peek) {
			s.advance()
			if isExponentSign(s.peek) {
				s.advance()
			}
			if !isDigit(s.peek) {
				return s.error("Invalid exponent", -1)
			}
			simple = false
		} else {
			break
		}
		s.advance()
	}

	str := s.input[start:s.index]
	if hasSeparators {
		str = strings.ReplaceAll(str, "_", "")
	}
	var value float64
	if simple {
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			panic("Invalid integer literal when parsing " + str)
		}
		value = float64(val)
	} else {
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			panic("Invalid float literal when parsing " + str)
		}
		value = val
	}
	return newNumberToken(start, s.index, value)
}

func (s *scanner) scanString() Token {
	start := s.index
	quote := s.peek
	s.advance() // Skip initial quote.

	buffer := ""
	marker := s.index
	input := s.input

	for s.peek != quote {
		if s.peek == '\\' {
			result, t := s.scanStringBackslash(buffer, marker)
			if t != nil {
				return *t // Error
			}
			buffer = result
			marker = s.index
		} else if s.peek == 0 { // $EOF
			return s.error("Unterminated quote", 0)
		} else {
			s.advance()
		}
	}

	last := input[marker:s.index]
	s.advance() // Skip terminating quote.

	return newStringToken(start, s.index, buffer+last, StringTokenKindPlain)
}

func (s *scanner) scanQuestion(start int) Token {
	s.advance()
	operator := "?"
	if s.peek == '?' {
		operator += "?"
		s.advance()
		if s.peek == '=' {
			operator += "="
			s.advance()
		}
	} else if s.peek == '.' {
		operator += "."
		s.advance()
	}
	return newOperatorToken(start, s.index, operator)
}

func (s *scanner) scanTemplateLiteralPart(start int) Token {
	buffer := ""
	marker := s.index

	for s.peek != '`' {
		if s.peek == '\\' {
			result, t := s.scanStringBackslash(buffer, marker)
			if t != nil {
				return *t // Error
			}
			buffer = result
			marker = s.index
		} else if s.peek == '$' {
			dollar := s.index
			s.advance()
			if s.peek == '{' {
				s.braceStack = append(s.braceStack, "interpolation")
				s.tokens = append(s.tokens, newStringToken(
					start,
					dollar,
					buffer+s.input[marker:dollar],
					StringTokenKindTemplateLiteralPart,
				))
				s.advance()
				return newOperatorToken(dollar, s.index, s.input[dollar:s.index])
			}
		} else if s.peek == 0 { // $EOF
			return s.error("Unterminated template literal", 0)
		} else {
			s.advance()
		}
	}

	last := s.input[marker:s.index]
	s.advance()
	return newStringToken(start, s.index, buffer+last, StringTokenKindTemplateLiteralEnd)
}

func (s *scanner) error(message string, offset int) Token {
	position := s.index + offset
	return newErrorToken(
		position,
		s.index,
		fmt.Sprintf("Lexer Error: %s at column %d in expression [%s]", message, position, s.input),
	)
}

func (s *scanner) scanStringBackslash(buffer string, marker int) (string, *Token) {
	buffer += s.input[marker:s.index]
	var unescapedCode int
	s.advance()
	if s.peek == 'u' {
		// 4 character hex code for unicode character.
		if s.index+5 > s.length {
			t := s.error("Invalid unicode escape", 0)
			return "", &t
		}
		hex := s.input[s.index+1 : s.index+5]
		val, err := strconv.ParseInt(hex, 16, 64)
		if err == nil {
			unescapedCode = int(val)
		} else {
			t := s.error(fmt.Sprintf("Invalid unicode escape [\\u%s]", hex), 0)
			return "", &t
		}
		for i := 0; i < 5; i++ {
			s.advance()
		}
	} else {
		unescapedCode = unescape(s.peek)
		s.advance()
	}
	buffer += string(rune(unescapedCode))
	return buffer, nil
}

func (s *scanner) scanStar(start int) Token {
	s.advance()
	operator := "*"

	if s.peek == '*' {
		operator += "*"
		s.advance()

		if s.peek == '=' {
			operator += "="
			s.advance()
		}
	} else if s.peek == '=' {
		operator += "="
		s.advance()
	}

	return newOperatorToken(start, s.index, operator)
}

func (s *scanner) isStartOfRegex() bool {
	if len(s.tokens) == 0 {
		return true
	}

	prevToken := s.tokens[len(s.tokens)-1]

	if prevToken.IsOperator("!") {
		var beforePrevToken *Token
		if len(s.tokens) > 1 {
			beforePrevToken = &s.tokens[len(s.tokens)-2]
		}
		isNegation := beforePrevToken == nil ||
			(beforePrevToken.Type != TokenTypeIdentifier &&
				!beforePrevToken.IsCharacter(')') &&
				!beforePrevToken.IsCharacter(']'))

		return isNegation
	}

	return prevToken.Type == TokenTypeOperator ||
		prevToken.IsCharacter('(') ||
		prevToken.IsCharacter('[') ||
		prevToken.IsCharacter(',') ||
		prevToken.IsCharacter(':')
}

func (s *scanner) scanRegex(tokenStart int) Token {
	s.advance()
	textStart := s.index
	inEscape := false
	inCharacterClass := false

	for {
		peek := s.peek

		if peek == 0 { // $EOF
			return s.error("Unterminated regular expression", 0)
		}

		if inEscape {
			inEscape = false
		} else if peek == '\\' {
			inEscape = true
		} else if peek == '[' {
			inCharacterClass = true
		} else if peek == ']' {
			inCharacterClass = false
		} else if peek == '/' && !inCharacterClass {
			break
		}
		s.advance()
	}

	value := s.input[textStart:s.index]
	s.advance()
	bodyToken := newRegExpBodyToken(tokenStart, s.index, value)
	flagsToken := s.scanRegexFlags(s.index)

	if flagsToken != nil {
		s.tokens = append(s.tokens, bodyToken)
		return *flagsToken
	}

	return bodyToken
}

func (s *scanner) scanRegexFlags(start int) *Token {
	if !isAsciiLetter(s.peek) {
		return nil
	}

	for isAsciiLetter(s.peek) {
		s.advance()
	}

	t := newRegExpFlagsToken(start, s.index, s.input[start:s.index])
	return &t
}

func isIdentifierStart(code int) bool {
	return ('a' <= code && code <= 'z') ||
		('A' <= code && code <= 'Z') ||
		code == '_' ||
		code == '$'
}

func isIdentifierPart(code int) bool {
	return isAsciiLetter(code) || isDigit(code) || code == '_' || code == '$'
}

func isExponentStart(code int) bool {
	return code == 'e' || code == 'E'
}

func isExponentSign(code int) bool {
	return code == '-' || code == '+'
}

func unescape(code int) int {
	switch code {
	case 'n':
		return '\n'
	case 'f':
		return '\f'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'v':
		return '\v'
	default:
		return code
	}
}

func isWhitespace(code int) bool {
	return (code >= '\t' && code <= ' ') || code == 160
}

func isDigit(code int) bool {
	return '0' <= code && code <= '9'
}

func isAsciiLetter(code int) bool {
	return (code >= 'a' && code <= 'z') || (code >= 'A' && code <= 'Z')
}
