package schema

import "strings"

var TRUSTED_TYPES_SINKS = map[string]bool{
	"iframe|srcdoc":   true,
	"*|innerhtml":     true,
	"*|outerhtml":     true,
	"embed|src":       true,
	"iframe|src":      true,
	"object|codebase": true,
	"object|data":     true,
}

func IsTrustedTypesSink(tagName, propName string) bool {
	tagName = strings.ToLower(tagName)
	propName = strings.ToLower(propName)

	return TRUSTED_TYPES_SINKS[tagName+"|"+propName] || TRUSTED_TYPES_SINKS["*|"+propName]
}
