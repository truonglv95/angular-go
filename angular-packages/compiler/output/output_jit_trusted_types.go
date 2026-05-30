package output

import "strings"

func NewTrustedFunctionForJIT(args ...string) func(any) any {
	// In Go, evaluating JavaScript strings as functions dynamically
	// doesn't directly apply like in JS (e.g. `new Function`).
	// For compilation, JIT in Go might just panic or return a stub
	// depending on how it's used. We'll return a stub function since Go can't `eval`.
	return func(any) any {
		panic(strings.Join(args, ", "))
	}
}
