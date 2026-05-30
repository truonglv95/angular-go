package ml_parser

import (
	"fmt"
	"testing"
)

func TestDebugEscape(t *testing.T) {
	text := "<t\\x64></t\\x64>"
	result := Tokenize(text, "someUrl", getHtmlTagDef, &TokenizeOptions{EscapedString: true})
	fmt.Printf("DEBUG ERRORS: %#v\n", result.Errors)
	for _, e := range result.Errors {
		fmt.Printf("Error: %s at %v\n", e.Msg, e.Span)
	}
}
