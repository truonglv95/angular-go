package chars

const EOF = 0
const BSPACE = 8
const TAB = 9
const NewLine = 10
const LF = 10
const VTAB = 11
const FF = 12
const CR = 13
const Return = 13
const SPACE = 32
const BANG = 33
const DQ = 34
const HASH = 35
const Dollar = 36
const PERCENT = 37
const AMPERSAND = 38
const SQ = 39
const LPAREN = 40
const RPAREN = 41
const STAR = 42
const PLUS = 43
const COMMA = 44
const MINUS = 45
const PERIOD = 46
const SLASH = 47
const COLON = 58
const SEMICOLON = 59
const LT = 60
const EQ = 61
const GT = 62
const QUESTION = 63

const Num0 = 48
const Num7 = 55
const Num9 = 57

const Letter_A = 65
const Letter_E = 69
const Letter_F = 70
const Letter_X = 88
const Letter_Z = 90

const LBRACKET = 91
const BACKSLASH = 92
const RBRACKET = 93
const CARET = 94
const Underscore = 95

const Letter_a = 97
const Letter_b = 98
const Letter_e = 101
const Letter_f = 102
const Letter_n = 110
const Letter_r = 114
const Letter_t = 116
const Letter_u = 117
const Letter_v = 118
const Letter_x = 120
const Letter_z = 122

const LBRACE = 123
const BAR = 124
const RBRACE = 125
const NBSP = 160

const PIPE = 124
const TILDA = 126
const AT = 64

const BT = 96
const BTAB = 98

func IsWhitespace(code int) bool {
	return (code >= TAB && code <= SPACE) || code == NBSP
}

func IsDigit(code int) bool {
	return Num0 <= code && code <= Num9
}

func IsAsciiLetter(code int) bool {
	return (code >= Letter_a && code <= Letter_z) || (code >= Letter_A && code <= Letter_Z)
}

func IsAsciiHexDigit(code int) bool {
	return (code >= Letter_a && code <= Letter_f) || (code >= Letter_A && code <= Letter_F) || IsDigit(code)
}

func IsNewLine(code int) bool {
	return code == LF || code == CR
}

func IsOctalDigit(code int) bool {
	return Num0 <= code && code <= Num7
}

func IsQuote(code int) bool {
	return code == SQ || code == DQ || code == BT
}
