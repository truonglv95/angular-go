package compiler

import "github.com/microsoft/typescript-go/angular-packages/compiler/render3"

func init() {
	render3.ParseSelectorToR3Selector = func(selector *string) []any {
		if selector == nil {
			return nil
		}
		res := ParseSelectorToR3Selector(selector)
		var anyRes []any
		for _, r := range res {
			anyRes = append(anyRes, r)
		}
		return anyRes
	}
}
