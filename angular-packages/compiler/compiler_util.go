package compiler

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var dashCaseRegexp = regexp.MustCompile(`-+([a-z0-9])`)

// DashCaseToCamelCase converts a dash-case string to camelCase.
func DashCaseToCamelCase(input string) string {
	return dashCaseRegexp.ReplaceAllStringFunc(input, func(m string) string {
		// Find the last character (the captured group [a-z0-9])
		sub := dashCaseRegexp.FindStringSubmatch(m)
		if len(sub) > 1 {
			return strings.ToUpper(sub[1])
		}
		return m
	})
}

// SplitAtColon splits a string at the first colon, returning defaultValues if no colon found.
func SplitAtColon(input string, defaultValues []string) []string {
	return splitAt(input, ":", defaultValues)
}

// SplitAtPeriod splits a string at the first period, returning defaultValues if no period found.
func SplitAtPeriod(input string, defaultValues []string) []string {
	return splitAt(input, ".", defaultValues)
}

func splitAt(input string, character string, defaultValues []string) []string {
	idx := strings.Index(input, character)
	if idx == -1 {
		return defaultValues
	}
	return []string{
		strings.TrimSpace(input[:idx]),
		strings.TrimSpace(input[idx+len(character):]),
	}
}

// NoUndefined returns val if it is not nil; otherwise returns nil.
// In TypeScript, noUndefined converts undefined → null. In Go, *T nil represents both.
func NoUndefined[T any](val *T) *T {
	return val
}

// InternalError panics with an internal error message.
func InternalError(msg string) {
	panic(fmt.Sprintf("Internal Error: %s", msg))
}

// EscapeRegExp escapes characters that have special meaning in regular expressions.
func EscapeRegExp(s string) string {
	special := `.*+?^=!:${}()|[]/\`
	var b strings.Builder
	for _, c := range s {
		if strings.ContainsRune(special, c) {
			b.WriteRune('\\')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// Byte is an alias for byte representing a UTF-8 byte.
type Byte = byte

// Utf8Encode encodes a string to a slice of UTF-8 bytes including surrogate pair handling
// equivalent to the TypeScript utf8Encode function.
func Utf8Encode(str string) []Byte {
	if !utf8.ValidString(str) {
		return []byte(str)
	}
	// In Go, strings are already UTF-8. However, to replicate the TypeScript behavior
	// (which operates on UTF-16 code units and handles surrogate pairs), we need to
	// convert code points to UTF-8 bytes manually.
	var encoded []Byte
	runes := []rune(str)
	for i := 0; i < len(runes); i++ {
		codePoint := runes[i]

		if codePoint <= 0x7f {
			encoded = append(encoded, byte(codePoint))
		} else if codePoint <= 0x7ff {
			encoded = append(encoded,
				byte(((codePoint>>6)&0x1f)|0xc0),
				byte((codePoint&0x3f)|0x80),
			)
		} else if codePoint <= 0xffff {
			encoded = append(encoded,
				byte((codePoint>>12)|0xe0),
				byte(((codePoint>>6)&0x3f)|0x80),
				byte((codePoint&0x3f)|0x80),
			)
		} else if codePoint <= 0x1fffff {
			encoded = append(encoded,
				byte(((codePoint>>18)&0x07)|0xf0),
				byte(((codePoint>>12)&0x3f)|0x80),
				byte(((codePoint>>6)&0x3f)|0x80),
				byte((codePoint&0x3f)|0x80),
			)
		}
	}
	_ = utf8.RuneError // ensure utf8 import is used
	return encoded
}

// Stringify returns a string representation of the given value.
func Stringify(token any) string {
	if token == nil {
		return "null"
	}
	switch v := token.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, len(v))
		for i, t := range v {
			parts[i] = Stringify(t)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		return "object"
	default:
		result := fmt.Sprintf("%v", token)
		// Find and return only up to newline
		if idx := strings.Index(result, "\n"); idx >= 0 {
			return result[:idx]
		}
		return result
	}
}

// Version holds a parsed semantic version string.
type Version struct {
	Full  string
	Major string
	Minor string
	Patch string
}

// NewVersion parses a version string in "major.minor.patch" format.
func NewVersion(full string) *Version {
	splits := strings.SplitN(full, ".", 3)
	major, minor, patch := "", "", ""
	if len(splits) >= 1 {
		major = splits[0]
	}
	if len(splits) >= 2 {
		minor = splits[1]
	}
	if len(splits) >= 3 {
		patch = splits[2]
	}
	return &Version{
		Full:  full,
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

// Console is an interface matching the TypeScript Console interface.
type Console interface {
	Log(message string)
	Warn(message string)
}

var v1To18Regexp = regexp.MustCompile(`^([1-9]|1[0-8])\.`)

// GetJitStandaloneDefaultForVersion returns whether the standalone default is true for the given version.
// - Version starting with "0." → true (always latest, default is true)
// - Version v1-v18 → false
// - All other versions (v19+) → true
func GetJitStandaloneDefaultForVersion(version string) bool {
	if strings.HasPrefix(version, "0.") {
		return true
	}
	if v1To18Regexp.MatchString(version) {
		return false
	}
	return true
}
