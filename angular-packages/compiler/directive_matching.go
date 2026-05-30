package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

var selectorRegexp = regexp.MustCompile(
	`(\:not\()|` + // 1: ":not("
		`(([\.\#]?)[-\w]+)|` + // 2: "tag"; 3: ".","#";
		// "-" should appear first in the regexp below as FF31 parses "[.-\w]" as a range
		// 4: attribute; 5: attribute_string; 6: attribute_value
		`(?:\[([-\.\w*\\$]+)(?:=([\"']?)([^\]\"']*)[\"']?)?\])|` + // "[name]", "[name=value]", "[name="value"]", "[name='value']"
		`(\))|` + // 7: ")"
		`(\s*,\s*)`, // 8: ","
)

// selectorRegexpGroup indices (0-indexed matches for Go regexp, corresponding to TypeScript's const enum)
const (
	srAll            = 0 // The whole match
	srNot            = 1
	srTag            = 2
	srPrefix         = 3
	srAttribute      = 4
	srAttributeStr   = 5
	srAttributeValue = 6
	srNotEnd         = 7
	srSeparator      = 8
)

// CssSelector represents a CSS selector with an element name, CSS classes,
// attribute/value pairs, and :not() selectors.
type CssSelector struct {
	Element      *string
	ClassNames   []string
	Attrs        []string
	NotSelectors []*CssSelector
}

// NewCssSelector creates a new empty CssSelector.
func NewCssSelector() *CssSelector {
	return &CssSelector{
		ClassNames:   []string{},
		Attrs:        []string{},
		NotSelectors: []*CssSelector{},
	}
}

// CssSelectorParse parses a CSS selector string into a list of CssSelectors.
func CssSelectorParse(selector string) []*CssSelector {
	results := []*CssSelector{}

	addResult := func(res *[]*CssSelector, cssSel *CssSelector) {
		if len(cssSel.NotSelectors) > 0 && cssSel.Element == nil &&
			len(cssSel.ClassNames) == 0 && len(cssSel.Attrs) == 0 {
			star := "*"
			cssSel.Element = &star
		}
		*res = append(*res, cssSel)
	}

	cssSelector := NewCssSelector()
	current := cssSelector
	inNot := false

	allMatches := selectorRegexp.FindAllStringSubmatchIndex(selector, -1)
	for _, loc := range allMatches {
		// Build the submatch strings from loc indices
		submatches := make([]string, 9)
		for i := 0; i < 9; i++ {
			start := loc[i*2]
			end := loc[i*2+1]
			if start >= 0 && end >= 0 {
				submatches[i] = selector[start:end]
			}
		}

		if submatches[srNot] != "" {
			if inNot {
				panic("Nesting :not in a selector is not allowed")
			}
			inNot = true
			current = NewCssSelector()
			cssSelector.NotSelectors = append(cssSelector.NotSelectors, current)
		}

		tag := submatches[srTag]
		if tag != "" {
			prefix := submatches[srPrefix]
			if prefix == "#" {
				// #hash
				current.AddAttribute("id", tag[1:])
			} else if prefix == "." {
				// Class
				current.AddClassName(tag[1:])
			} else {
				// Element
				current.SetElement(tag)
			}
		}

		attribute := submatches[srAttribute]
		if attribute != "" {
			current.AddAttribute(
				current.UnescapeAttribute(attribute),
				submatches[srAttributeValue],
			)
		}

		if submatches[srNotEnd] != "" {
			inNot = false
			current = cssSelector
		}

		if submatches[srSeparator] != "" {
			if inNot {
				panic("Multiple selectors in :not are not supported")
			}
			addResult(&results, cssSelector)
			cssSelector = NewCssSelector()
			current = cssSelector
		}
	}
	addResult(&results, cssSelector)
	return results
}

// UnescapeAttribute unescapes \$ sequences from the CSS attribute selector.
func (c *CssSelector) UnescapeAttribute(attr string) string {
	var result strings.Builder
	escaping := false
	for _, ch := range attr {
		if ch == '\\' {
			escaping = true
			continue
		}
		if ch == '$' && !escaping {
			panic(fmt.Sprintf(
				`Error in attribute selector "%s". Unescaped "$" is not supported. Please escape with "\$".`,
				attr,
			))
		}
		escaping = false
		result.WriteRune(ch)
	}
	return result.String()
}

// EscapeAttribute escapes $ sequences in the CSS attribute selector.
func (c *CssSelector) EscapeAttribute(attr string) string {
	attr = strings.ReplaceAll(attr, `\`, `\\`)
	attr = strings.ReplaceAll(attr, `$`, `\$`)
	return attr
}

// IsElementSelector returns true if this selector is a pure element selector
// (no classes, attrs, or :not selectors).
func (c *CssSelector) IsElementSelector() bool {
	return c.HasElementSelector() &&
		len(c.ClassNames) == 0 &&
		len(c.Attrs) == 0 &&
		len(c.NotSelectors) == 0
}

// HasElementSelector returns true if an element name is set.
func (c *CssSelector) HasElementSelector() bool {
	return c.Element != nil
}

// SetElement sets the element name. Pass nil or empty string to unset.
func (c *CssSelector) SetElement(element string) {
	c.Element = &element
}

// SetElementNil sets the element to nil.
func (c *CssSelector) SetElementNil() {
	c.Element = nil
}

// GetAttrs returns the effective attrs array, prepending "class" and class names if present.
func (c *CssSelector) GetAttrs() []string {
	var result []string
	if len(c.ClassNames) > 0 {
		result = append(result, "class", strings.Join(c.ClassNames, " "))
	}
	return append(result, c.Attrs...)
}

// AddAttribute adds an attribute name/value pair to this selector.
func (c *CssSelector) AddAttribute(name string, value string) {
	lowValue := ""
	if value != "" {
		lowValue = strings.ToLower(value)
	}
	c.Attrs = append(c.Attrs, name, lowValue)
}

// AddClassName adds a CSS class name to this selector.
func (c *CssSelector) AddClassName(name string) {
	c.ClassNames = append(c.ClassNames, strings.ToLower(name))
}

// String returns the string representation of the selector.
func (c *CssSelector) String() string {
	var res strings.Builder
	if c.Element != nil {
		res.WriteString(*c.Element)
	}
	for _, klass := range c.ClassNames {
		res.WriteString(".")
		res.WriteString(klass)
	}
	for i := 0; i < len(c.Attrs); i += 2 {
		name := c.EscapeAttribute(c.Attrs[i])
		value := c.Attrs[i+1]
		res.WriteString("[")
		res.WriteString(name)
		if value != "" {
			res.WriteString("=")
			res.WriteString(value)
		}
		res.WriteString("]")
	}
	for _, notSelector := range c.NotSelectors {
		res.WriteString(":not(")
		res.WriteString(notSelector.String())
		res.WriteString(")")
	}
	return res.String()
}

// SelectorMatcher reads a list of CssSelectors and allows calculating which ones
// are contained in a given CssSelector.
type SelectorMatcher[T any] struct {
	elementMap          map[string][]*SelectorContext[T]
	elementPartialMap   map[string]*SelectorMatcher[T]
	classMap            map[string][]*SelectorContext[T]
	classPartialMap     map[string]*SelectorMatcher[T]
	attrValueMap        map[string]map[string][]*SelectorContext[T]
	attrValuePartialMap map[string]map[string]*SelectorMatcher[T]
	listContexts        []*SelectorListContext
}

// NewSelectorMatcher creates a new SelectorMatcher.
func NewSelectorMatcher[T any]() *SelectorMatcher[T] {
	return &SelectorMatcher[T]{
		elementMap:          make(map[string][]*SelectorContext[T]),
		elementPartialMap:   make(map[string]*SelectorMatcher[T]),
		classMap:            make(map[string][]*SelectorContext[T]),
		classPartialMap:     make(map[string]*SelectorMatcher[T]),
		attrValueMap:        make(map[string]map[string][]*SelectorContext[T]),
		attrValuePartialMap: make(map[string]map[string]*SelectorMatcher[T]),
		listContexts:        []*SelectorListContext{},
	}
}

// CreateNotMatcher creates a SelectorMatcher for :not() selectors.
func CreateNotMatcher(notSelectors []*CssSelector) *SelectorMatcher[*struct{}] {
	notMatcher := NewSelectorMatcher[*struct{}]()
	notMatcher.AddSelectables(notSelectors, nil)
	return notMatcher
}

// AddSelectables adds CSS selectors with an associated callback context.
func (m *SelectorMatcher[T]) AddSelectables(cssSelectors []*CssSelector, callbackCtxt T) {
	var listContext *SelectorListContext
	if len(cssSelectors) > 1 {
		listContext = &SelectorListContext{Selectors: cssSelectors}
		m.listContexts = append(m.listContexts, listContext)
	}
	for _, sel := range cssSelectors {
		m.addSelectable(sel, callbackCtxt, listContext)
	}
}

func (m *SelectorMatcher[T]) addSelectable(
	cssSelector *CssSelector,
	callbackCtxt T,
	listContext *SelectorListContext,
) {
	matcher := m
	element := cssSelector.Element
	classNames := cssSelector.ClassNames
	attrs := cssSelector.Attrs
	selectable := &SelectorContext[T]{
		Selector:     cssSelector,
		CbContext:    callbackCtxt,
		ListContext:  listContext,
		notSelectors: cssSelector.NotSelectors,
	}

	if element != nil {
		isTerminal := len(attrs) == 0 && len(classNames) == 0
		if isTerminal {
			addTerminal(matcher.elementMap, *element, selectable)
		} else {
			matcher = addPartial[T](matcher.elementPartialMap, *element)
		}
	}

	if len(classNames) > 0 {
		for i, className := range classNames {
			isTerminal := len(attrs) == 0 && i == len(classNames)-1
			if isTerminal {
				addTerminal(matcher.classMap, className, selectable)
			} else {
				matcher = addPartial[T](matcher.classPartialMap, className)
			}
		}
	}

	for i := 0; i < len(attrs); i += 2 {
		isTerminal := i == len(attrs)-2
		name := attrs[i]
		value := attrs[i+1]
		if isTerminal {
			terminalMap := matcher.attrValueMap
			if _, exists := terminalMap[name]; !exists {
				terminalMap[name] = make(map[string][]*SelectorContext[T])
			}
			addTerminal(terminalMap[name], value, selectable)
		} else {
			partialMap := matcher.attrValuePartialMap
			if _, exists := partialMap[name]; !exists {
				partialMap[name] = make(map[string]*SelectorMatcher[T])
			}
			matcher = addPartial[T](partialMap[name], value)
		}
	}
}

func addTerminal[T any](
	m map[string][]*SelectorContext[T],
	name string,
	selectable *SelectorContext[T],
) {
	m[name] = append(m[name], selectable)
}

func addPartial[T any](
	m map[string]*SelectorMatcher[T],
	name string,
) *SelectorMatcher[T] {
	if _, exists := m[name]; !exists {
		m[name] = NewSelectorMatcher[T]()
	}
	return m[name]
}

// Match finds objects added via AddSelectables whose CSS selector is contained in the given
// CSS selector. Returns true if a match was found.
func (m *SelectorMatcher[T]) Match(
	cssSelector *CssSelector,
	matchedCallback func(c *CssSelector, a T),
) bool {
	result := false
	element := ""
	if cssSelector.Element != nil {
		element = *cssSelector.Element
	}
	classNames := cssSelector.ClassNames
	attrs := cssSelector.Attrs

	for _, ctx := range m.listContexts {
		ctx.AlreadyMatched = false
	}

	result = m.matchTerminal(m.elementMap, element, cssSelector, matchedCallback) || result
	result = m.matchPartial(m.elementPartialMap, element, cssSelector, matchedCallback) || result

	for _, className := range classNames {
		result = m.matchTerminal(m.classMap, className, cssSelector, matchedCallback) || result
		result = m.matchPartial(m.classPartialMap, className, cssSelector, matchedCallback) || result
	}

	for i := 0; i < len(attrs); i += 2 {
		name := attrs[i]
		value := attrs[i+1]

		terminalValuesMap := m.attrValueMap[name]
		if value != "" {
			result = m.matchTerminal(terminalValuesMap, "", cssSelector, matchedCallback) || result
		}
		result = m.matchTerminal(terminalValuesMap, value, cssSelector, matchedCallback) || result

		partialValuesMap := m.attrValuePartialMap[name]
		if value != "" {
			result = m.matchPartial(partialValuesMap, "", cssSelector, matchedCallback) || result
		}
		result = m.matchPartial(partialValuesMap, value, cssSelector, matchedCallback) || result
	}
	return result
}

// MatchTerminal is exported for internal use; matches against terminal entries.
func (m *SelectorMatcher[T]) MatchTerminal(
	nameMap map[string][]*SelectorContext[T],
	name string,
	cssSelector *CssSelector,
	matchedCallback func(c *CssSelector, a T),
) bool {
	return m.matchTerminal(nameMap, name, cssSelector, matchedCallback)
}

func (m *SelectorMatcher[T]) matchTerminal(
	nameMap map[string][]*SelectorContext[T],
	name string,
	cssSelector *CssSelector,
	matchedCallback func(c *CssSelector, a T),
) bool {
	if nameMap == nil {
		return false
	}

	selectables := nameMap[name]
	starSelectables := nameMap["*"]
	if starSelectables != nil {
		selectables = append(selectables, starSelectables...)
	}
	if len(selectables) == 0 {
		return false
	}

	result := false
	for _, selectable := range selectables {
		result = selectable.Finalize(cssSelector, matchedCallback) || result
	}
	return result
}

// MatchPartial is exported for internal use; matches against partial entries.
func (m *SelectorMatcher[T]) MatchPartial(
	nameMap map[string]*SelectorMatcher[T],
	name string,
	cssSelector *CssSelector,
	matchedCallback func(c *CssSelector, a T),
) bool {
	return m.matchPartial(nameMap, name, cssSelector, matchedCallback)
}

func (m *SelectorMatcher[T]) matchPartial(
	nameMap map[string]*SelectorMatcher[T],
	name string,
	cssSelector *CssSelector,
	matchedCallback func(c *CssSelector, a T),
) bool {
	if nameMap == nil {
		return false
	}
	nestedSelector := nameMap[name]
	if nestedSelector == nil {
		return false
	}
	return nestedSelector.Match(cssSelector, matchedCallback)
}

// SelectorListContext holds context for a list of selectors that are matched together.
type SelectorListContext struct {
	Selectors      []*CssSelector
	AlreadyMatched bool
}

// SelectorContext stores context to pass back selector and context when a selector is matched.
type SelectorContext[T any] struct {
	Selector     *CssSelector
	CbContext    T
	ListContext  *SelectorListContext
	notSelectors []*CssSelector
}

// Finalize checks the not-selectors and calls the callback if appropriate.
// Returns true if the selector matches.
func (c *SelectorContext[T]) Finalize(
	cssSelector *CssSelector,
	callback func(c *CssSelector, a T),
) bool {
	result := true
	if len(c.notSelectors) > 0 && (c.ListContext == nil || !c.ListContext.AlreadyMatched) {
		notMatcher := CreateNotMatcher(c.notSelectors)
		matched := notMatcher.Match(cssSelector, nil)
		result = !matched
	}
	if result && callback != nil && (c.ListContext == nil || !c.ListContext.AlreadyMatched) {
		if c.ListContext != nil {
			c.ListContext.AlreadyMatched = true
		}
		callback(c.Selector, c.CbContext)
	}
	return result
}

// SelectorlessMatcher matches by name against a registry, without CSS selector parsing.
type SelectorlessMatcher[T any] struct {
	registry map[string][]T
}

// NewSelectorlessMatcher creates a new SelectorlessMatcher with the given registry.
func NewSelectorlessMatcher[T any](registry map[string][]T) *SelectorlessMatcher[T] {
	return &SelectorlessMatcher[T]{registry: registry}
}

// MatchByName returns all values registered under the given name.
func (m *SelectorlessMatcher[T]) MatchByName(name string) []T {
	if v, ok := m.registry[name]; ok {
		return v
	}
	return []T{}
}
