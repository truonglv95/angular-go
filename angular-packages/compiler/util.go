package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

var _anonymousTypeIndex = 0

func IdentifierName(compileIdentifier *CompileIdentifierMetadata) *string {
	if compileIdentifier == nil || compileIdentifier.Reference == nil {
		return nil
	}
	ref := compileIdentifier.Reference

	if refMap, ok := ref.(map[string]any); ok {
		if val, ok := refMap["__anonymousType"]; ok {
			strVal := val.(string)
			return &strVal
		}
		if _, ok := refMap["__forward_ref__"]; ok {
			strVal := "__forward_ref__"
			return &strVal
		}
		identifier := fmt.Sprintf("%v", ref)
		if strings.Contains(identifier, "(") {
			identifier = fmt.Sprintf("anonymous_%d", _anonymousTypeIndex)
			_anonymousTypeIndex++
			refMap["__anonymousType"] = identifier
		} else {
			identifier = SanitizeIdentifier(identifier)
		}
		return &identifier
	}

	identifier := fmt.Sprintf("%v", ref)
	identifier = SanitizeIdentifier(identifier)
	return &identifier
}

type CompileIdentifierMetadata struct {
	Reference any
}

var nonWordRegexp = regexp.MustCompile(`\W`)

func SanitizeIdentifier(name string) string {
	return nonWordRegexp.ReplaceAllString(name, "_")
}
