package compiler

// Character code constants.
const (
	CharEOF       = 0
	CharBSPACE    = 8
	CharTAB       = 9
	CharLF        = 10
	CharVTAB      = 11
	CharFF        = 12
	CharCR        = 13
	CharSPACE     = 32
	CharBANG      = 33
	CharDQ        = 34
	CharHASH      = 35
	CharDollar    = 36
	CharPERCENT   = 37
	CharAMPERSAND = 38
	CharSQ        = 39
	CharLPAREN    = 40
	CharRPAREN    = 41
	CharSTAR      = 42
	CharPLUS      = 43
	CharCOMMA     = 44
	CharMINUS     = 45
	CharPERIOD    = 46
	CharSLASH     = 47
	CharCOLON     = 58
	CharSEMICOLON = 59
	CharLT        = 60
	CharEQ        = 61
	CharGT        = 62
	CharQUESTION  = 63

	CharNum0 = 48
	CharNum7 = 55
	CharNum9 = 57

	CharLetterA = 65
	CharLetterE = 69
	CharLetterF = 70
	CharLetterX = 88
	CharLetterZ = 90

	CharLBRACKET   = 91
	CharBACKSLASH  = 92
	CharRBRACKET   = 93
	CharCARET      = 94
	CharUnderscore = 95

	CharLettera  = 97
	CharLetterb  = 98
	CharLetterE2 = 101
	CharLetterf  = 102
	CharLettern  = 110
	CharLetterr  = 114
	CharLettert  = 116
	CharLetteru  = 117
	CharLetterv  = 118
	CharLetterx  = 120
	CharLetterz  = 122

	CharLBRACE = 123
	CharBAR    = 124
	CharRBRACE = 125
	CharNBSP   = 160

	CharPIPE  = 124
	CharTILDA = 126
	CharAT    = 64

	CharBT = 96
)

// IsWhitespace returns true if the given character code is a whitespace character.
func IsWhitespace(code int) bool {
	return (code >= CharTAB && code <= CharSPACE) || code == CharNBSP
}

// IsDigit returns true if the given character code is a digit.
func IsDigit(code int) bool {
	return CharNum0 <= code && code <= CharNum9
}

// IsAsciiLetter returns true if the given character code is an ASCII letter.
func IsAsciiLetter(code int) bool {
	return (code >= CharLettera && code <= CharLetterz) || (code >= CharLetterA && code <= CharLetterZ)
}

// IsAsciiHexDigit returns true if the given character code is an ASCII hex digit.
func IsAsciiHexDigit(code int) bool {
	return (code >= CharLettera && code <= CharLetterf) || (code >= CharLetterA && code <= CharLetterF) || IsDigit(code)
}

// IsNewLine returns true if the given character code is a newline character.
func IsNewLine(code int) bool {
	return code == CharLF || code == CharCR
}

// IsOctalDigit returns true if the given character code is an octal digit.
func IsOctalDigit(code int) bool {
	return CharNum0 <= code && code <= CharNum7
}

// IsQuote returns true if the given character code is a quote character.
func IsQuote(code int) bool {
	return code == CharSQ || code == CharDQ || code == CharBT
}
