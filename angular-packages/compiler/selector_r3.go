package compiler

import "github.com/microsoft/typescript-go/angular-packages/compiler/core"

type R3CssSelector []any
type R3CssSelectorList []R3CssSelector

func parserSelectorToSimpleSelector(selector *CssSelector) R3CssSelector {
	var classes []any
	if len(selector.ClassNames) > 0 {
		classes = append(classes, core.SelectorFlagsClass)
		for _, c := range selector.ClassNames {
			classes = append(classes, c)
		}
	}

	var elementName any = ""
	if selector.Element != nil && *selector.Element != "*" {
		elementName = *selector.Element
	}

	var result R3CssSelector
	result = append(result, elementName)
	for _, a := range selector.Attrs {
		result = append(result, a)
	}
	result = append(result, classes...)
	return result
}

func parserSelectorToNegativeSelector(selector *CssSelector) R3CssSelector {
	var classes []any
	if len(selector.ClassNames) > 0 {
		classes = append(classes, core.SelectorFlagsClass)
		for _, c := range selector.ClassNames {
			classes = append(classes, c)
		}
	}

	if selector.Element != nil {
		var result R3CssSelector
		result = append(result, core.SelectorFlagsNot|core.SelectorFlagsElement, *selector.Element)
		for _, a := range selector.Attrs {
			result = append(result, a)
		}
		result = append(result, classes...)
		return result
	} else if len(selector.Attrs) > 0 {
		var result R3CssSelector
		result = append(result, core.SelectorFlagsNot|core.SelectorFlagsAttribute)
		for _, a := range selector.Attrs {
			result = append(result, a)
		}
		result = append(result, classes...)
		return result
	} else {
		if len(selector.ClassNames) > 0 {
			var result R3CssSelector
			result = append(result, core.SelectorFlagsNot|core.SelectorFlagsClass)
			for _, c := range selector.ClassNames {
				result = append(result, c)
			}
			return result
		} else {
			return R3CssSelector{}
		}
	}
}

func parserSelectorToR3Selector(selector *CssSelector) R3CssSelector {
	positive := parserSelectorToSimpleSelector(selector)

	var negative R3CssSelectorList
	if len(selector.NotSelectors) > 0 {
		for _, notSelector := range selector.NotSelectors {
			negative = append(negative, parserSelectorToNegativeSelector(notSelector))
		}
	}

	result := positive
	for _, n := range negative {
		result = append(result, n...)
	}
	return result
}

func ParseSelectorToR3Selector(selector *string) R3CssSelectorList {
	if selector != nil {
		parsed := CssSelectorParse(*selector)
		var result R3CssSelectorList
		for _, s := range parsed {
			result = append(result, parserSelectorToR3Selector(s))
		}
		return result
	}
	return R3CssSelectorList{}
}
