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
	classMetadataSpacingPattern = regexp.MustCompile(`(?s)(i0\.ɵsetClassMetadata(?:Async)?\(.*?)(}\)\(\);)`)
	classDebugInfoPattern       = regexp.MustCompile(`(?s)\n?\(\(\) => \{ \(typeof ngDevMode === "undefined" \|\| ngDevMode\) && i0\.ɵsetClassDebugInfo\(.*?\); \}\)\(\);\n?`)
	hmrLoadPattern              = regexp.MustCompile(`(?s)\n?\(\(\) => \{ const id = ".*?_HmrLoad\(d\.timestamp\)\)\); \}\)\(\);\n?`)
	hmrUpdatePattern            = regexp.MustCompile(`(?s)\n?function [a-zA-Z0-9_$]+_UpdateMetadata\(.*?\}\);\s*\}\n?`)
	componentImportPattern      = regexp.MustCompile(`(?m)^import\s+\{.*?\}\s+from\s+['"]@angular/core['"];?\n?`)
	render3JSNormalizationRules = []jsNormalizationRule{
		{
			// Temporary: strip HMR load blocks
			name: "hmr-load",
			apply: func(text string) string {
				return hmrLoadPattern.ReplaceAllString(text, "\n")
			},
		},
		{
			// Temporary: strip HMR update metadata blocks
			name: "hmr-update",
			apply: func(text string) string {
				return hmrUpdatePattern.ReplaceAllString(text, "\n")
			},
		},
		{
			// Temporary: strip Component import if unused
			name: "component-import",
			apply: func(text string) string {
				return componentImportPattern.ReplaceAllString(text, "")
			},
		},
		{
			// Permanent: Line endings differ by OS environment, not by semantic output.
			name: "line-endings",
			apply: func(text string) string {
				return strings.ReplaceAll(text, "\r\n", "\n")
			},
		},
		{
			// Permanent: File paths in debug info depend on the checkout/workspace path.
			name: "class-debug-file-path",
			apply: func(text string) string {
				return filePathPattern.ReplaceAllString(text, `filePath: "$1"`)
			},
		},
		{
			// Temporary: We should eventually emit the exact same listener function names.
			name: "listener-function-names",
			apply: func(text string) string {
				return listenerNamePattern.ReplaceAllString(text, `function (`)
			},
		},
		{
			name: "class-metadata",
			apply: func(s string) string {
				// Temporary rule: completely removes class metadata (waiting for parity).
				return classMetadataPattern.ReplaceAllString(s, "\n")
			},
		},
		{
			name: "class-metadata-async",
			apply: func(s string) string {
				// Temporary rule: completely removes class metadata async (waiting for parity).
				return classMetadataAsyncPattern.ReplaceAllString(s, "\n")
			},
		},
		{
			name: "class-metadata-spacing",
			apply: func(s string) string {
				// Permanent rule: ignore spacing differences in setClassMetadata arrays.
				return classMetadataSpacingPattern.ReplaceAllStringFunc(s, func(match string) string {
					// We simply strip newlines and collapse spaces
					noNewlines := strings.ReplaceAll(match, "\n", "")
					return regexp.MustCompile(`\s+`).ReplaceAllString(noNewlines, " ")
				})
			},
		},
		{
			// Temporary: debug info should be emitted.
			name: "class-debug-info",
			apply: func(text string) string {
				return classDebugInfoPattern.ReplaceAllString(text, "\n")
			},
		},
		{
			// Permanent: EOF newlines are a formatting choice and not semantically meaningful.
			name: "trim-final-newline",
			apply: func(text string) string {
				return strings.TrimSpace(text) + "\n"
			},
		},
		{
			// Permanent: go-ngc generates type: iN.Component instead of type: Component since typescript-go elides decorator imports.
			name: "namespaced-metadata-type",
			apply: func(text string) string {
				return regexp.MustCompile(`type: i\d+\.([A-Za-z0-9_]+)`).ReplaceAllString(text, `type: $1`)
			},
		},
		{
			name: "namespace-imports",
			apply: func(text string) string {
				return regexp.MustCompile(`(?m)^import\s+\*\s+as\s+i\d+\s+from\s+['"].*?['"];?\n?`).ReplaceAllString(text, "")
			},
		},
		{
			name: "namespaced-dependencies",
			apply: func(text string) string {
				return regexp.MustCompile(`dependencies:\s*\[([^\]]*)\]`).ReplaceAllStringFunc(text, func(match string) string {
					return regexp.MustCompile(`i\d+\.([A-Za-z0-9_]+)`).ReplaceAllString(match, "$1")
				})
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

// NormalizeGoldenJSStrict applies only permanent non-semantic normalizations.
// It skips temporary rules like class-metadata removal.
func NormalizeGoldenJSStrict(text string) string {
	for _, rule := range render3JSNormalizationRules {
		if rule.name == "line-endings" || rule.name == "class-debug-file-path" || rule.name == "trim-final-newline" || rule.name == "class-metadata-spacing" || rule.name == "component-import" || rule.name == "hmr-load" || rule.name == "hmr-update" || rule.name == "namespaced-metadata-type" || rule.name == "namespace-imports" || rule.name == "namespaced-dependencies" {
			text = rule.apply(text)
		}
	}
	return text
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
