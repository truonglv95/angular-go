package ml_parser

import "github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"

type TokenType int

const (
	TokenTypeTagOpenStart TokenType = iota
	TokenTypeTagOpenEnd
	TokenTypeTagOpenEndVoid
	TokenTypeTagClose
	TokenTypeIncompleteTagOpen
	TokenTypeText
	TokenTypeEscapableRawText
	TokenTypeRawText
	TokenTypeInterpolation
	TokenTypeEncodedEntity
	TokenTypeCommentStart
	TokenTypeCommentEnd
	TokenTypeCdataStart
	TokenTypeCdataEnd
	TokenTypeAttrName
	TokenTypeAttrQuote
	TokenTypeAttrValueText
	TokenTypeAttrValueInterpolation
	TokenTypeDocType
	TokenTypeExpansionFormStart
	TokenTypeExpansionCaseValue
	TokenTypeExpansionCaseExpStart
	TokenTypeExpansionCaseExpEnd
	TokenTypeExpansionFormEnd
	TokenTypeBlockOpenStart
	TokenTypeBlockOpenEnd
	TokenTypeBlockClose
	TokenTypeBlockParameter
	TokenTypeIncompleteBlockOpen
	TokenTypeLetStart
	TokenTypeLetValue
	TokenTypeLetEnd
	TokenTypeIncompleteLet
	TokenTypeComponentOpenStart
	TokenTypeComponentOpenEnd
	TokenTypeComponentOpenEndVoid
	TokenTypeComponentClose
	TokenTypeIncompleteComponentOpen
	TokenTypeDirectiveName
	TokenTypeDirectiveOpen
	TokenTypeDirectiveClose
	TokenTypeEOF
)

type Token struct {
	Type       TokenType
	Parts      []string
	SourceSpan *parse_util.ParseSourceSpan
}

func (t Token) GetType() int {
	return int(t.Type)
}

func (t Token) GetParts() []string {
	return t.Parts
}

func (t Token) GetSpanOffsets() (int, int) {
	if t.SourceSpan != nil {
		return t.SourceSpan.Start.Offset, t.SourceSpan.End.Offset
	}
	return 0, 0
}
