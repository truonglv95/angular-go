package parity

import (
	"regexp"
	"strings"
)

type NormalizedJS struct {
	Text  string
	Rules []string
}

type jsNormalizationRule struct {
	name  string
	apply func(string) string
}

var (
	filePathPattern             = regexp.MustCompile(`filePath: "[^"]*(app/[^"]+)"`)
	listenerNamePattern         = regexp.MustCompile(`function [A-Za-z0-9_$]+_listener\(`)
	classMetadataPattern        = regexp.MustCompile(`(?s)\n?\(\(\) => \{ \(typeof ngDevMode === "undefined" \|\| ngDevMode\) && i0\.ɵsetClassMetadata\(.*?\); \}\)\(\);\n?`)
	classMetadataAsyncPattern   = regexp.MustCompile(`(?s)\n?\(\(\) => \{ \(typeof ngDevMode === "undefined" \|\| ngDevMode\) && i0\.ɵsetClassMetadataAsync\(.*?\); \}\)\(\);\n?`)
	classDebugInfoPattern       = regexp.MustCompile(`(?s)\n?\(\(\) => \{ \(typeof ngDevMode === "undefined" \|\| ngDevMode\) && i0\.ɵsetClassDebugInfo\(.*?\); \}\)\(\);\n?`)
	render3JSNormalizationRules = []jsNormalizationRule{
		{
			name: "line-endings",
			apply: func(text string) string {
				return strings.ReplaceAll(text, "\r\n", "\n")
			},
		},
		{
			name: "class-debug-file-path",
			apply: func(text string) string {
				return filePathPattern.ReplaceAllString(text, `filePath: "$1"`)
			},
		},
		{
			name: "listener-function-names",
			apply: func(text string) string {
				return listenerNamePattern.ReplaceAllString(text, `function (`)
			},
		},
		{
			name: "class-metadata",
			apply: func(text string) string {
				return classMetadataPattern.ReplaceAllString(text, "\n")
			},
		},
		{
			name: "class-metadata-async",
			apply: func(text string) string {
				return classMetadataAsyncPattern.ReplaceAllString(text, "\n")
			},
		},
		{
			name: "class-debug-info",
			apply: func(text string) string {
				return classDebugInfoPattern.ReplaceAllString(text, "\n")
			},
		},
		{
			name: "trim-final-newline",
			apply: func(text string) string {
				return strings.TrimSpace(text) + "\n"
			},
		},
	}
)

// NormalizeGoldenJS applies only accepted non-semantic normalizations used by
// the render3 parity tests and comparison runner. Raw output must still be kept
// by callers for inspection.
func NormalizeGoldenJS(text string) string {
	return NormalizeGoldenJSWithRules(text).Text
}

// NormalizeGoldenJSWithRules returns normalized JS plus the accepted rules that
// changed the input. Use this when producing parity reports.
func NormalizeGoldenJSWithRules(text string) NormalizedJS {
	rules := make([]string, 0, len(render3JSNormalizationRules))
	for _, rule := range render3JSNormalizationRules {
		next := rule.apply(text)
		if next != text {
			rules = append(rules, rule.name)
			text = next
		}
	}
	return NormalizedJS{Text: text, Rules: rules}
}

func MergeRuleNames(left []string, right []string) []string {
	seen := map[string]bool{}
	merged := make([]string, 0, len(left)+len(right))
	for _, rules := range [][]string{left, right} {
		for _, rule := range rules {
			if seen[rule] {
				continue
			}
			seen[rule] = true
			merged = append(merged, rule)
		}
	}
	return merged
}
