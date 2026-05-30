package compiler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var animationKeywords = map[string]bool{
	"inherit":           true,
	"initial":           true,
	"revert":            true,
	"unset":             true,
	"alternate":         true,
	"alternate-reverse": true,
	"normal":            true,
	"reverse":           true,
	"backwards":         true,
	"both":              true,
	"forwards":          true,
	"none":              true,
	"paused":            true,
	"running":           true,
	"ease":              true,
	"ease-in":           true,
	"ease-in-out":       true,
	"ease-out":          true,
	"linear":            true,
	"step-start":        true,
	"step-end":          true,
	"end":               true,
	"jump-both":         true,
	"jump-end":          true,
	"jump-none":         true,
	"jump-start":        true,
	"start":             true,
}

var scopedAtRuleIdentifiers = []string{
	"@media",
	"@supports",
	"@document",
	"@layer",
	"@container",
	"@scope",
	"@starting-style",
}

var (
	_newLinesRe                   = regexp.MustCompile(`\r?\n`)
	_commentRe                    = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	_commentWithHashRe            = regexp.MustCompile(`/\*\s*#\s*source(Mapping)?URL=`)
	_commentWithHashPlaceHolderRe = regexp.MustCompile(COMMENT_PLACEHOLDER)
	_shadowDeepSelectors          = regexp.MustCompile(`(?:>>>)|(?:/deep/)|(?:::ng-deep)`)

	_polyfillHost                        = "-shadowcsshost"
	_polyfillHostContext                 = "-shadowcsscontext"
	_polyfillHostNoCombinator            = _polyfillHost + "-no-combinator"
	_polyfillHostNoCombinatorRe          = regexp.MustCompile(`-shadowcsshost-no-combinator([^\s,]*)`)
	_polyfillHostRe                      = regexp.MustCompile(`(?i)-shadowcsshost`)
	_colonHostRe                         = regexp.MustCompile(`(?i):host`)
	_colonHostContextRe                  = regexp.MustCompile(`(?i):host-context(\(\s*[^)\s])`)
	_cssPrefixWithPseudoSelectorFunction = regexp.MustCompile(`(?i):(where|is)\(`)
	_keyframesSelectorRe                 = regexp.MustCompile(`^(@(?:-webkit-)?keyframes)(\s+)(.+)$`)
	_animationNameDeclRe                 = regexp.MustCompile(`(?m)(^|[;{]\s*|\s+)(animation-name\s*:\s*)([^;]+)(;)`)
	_animationDeclRe                     = regexp.MustCompile(`(?m)(^|[;{]\s*|\s+)(animation\s*:\s*)([^;]+)(;)`)
)

const COMMENT_PLACEHOLDER = "%COMMENT%"
const BLOCK_PLACEHOLDER = "%BLOCK%"
const COMMA_IN_PLACEHOLDER = "%COMMA_IN_PLACEHOLDER%"
const SEMI_IN_PLACEHOLDER = "%SEMI_IN_PLACEHOLDER%"
const COLON_IN_PLACEHOLDER = "%COLON_IN_PLACEHOLDER%"

type CssRule struct {
	Selector string
	Content  string
}

type ShadowCss struct {
	safeSelector         *SafeSelector
	shouldScopeIndicator bool
}

func NewShadowCss() *ShadowCss {
	return &ShadowCss{}
}

const _noParens = `[^)(]*`
const _level1Parens = `(?:\(` + _noParens + `\)|` + _noParens + `)+?`
const _level2Parens = `(?:\(` + _level1Parens + `\)|` + _noParens + `)+?`
const _parenSuffix = `(?:\((` + _level2Parens + `)\))`

var (
	nthRegex                     = regexp.MustCompile(`(:nth-[-\w]+)` + _parenSuffix)
	attrRegex                    = regexp.MustCompile(`(\[[^\]]*\])`)
	escapeRegex                  = regexp.MustCompile(`(\\.)`)
	restoreRegex                 = regexp.MustCompile(`__(?:ph|esc-ph)-(\d+)__`)
	_cssColonHostRe              = regexp.MustCompile(`(?i)` + _polyfillHost + _parenSuffix + `?([^,{]*)`)
	_hostContextPattern          = _polyfillHostContext + _parenSuffix + `([^{]*)`
	_cssColonHostContextReGlobal = regexp.MustCompile(`(?i)` + `(:(?:where|is)\()?` + _hostContextPattern)
)

type SafeSelector struct {
	placeholders []string
	index        int
	contentStr   string
}

func NewSafeSelector(selector string) *SafeSelector {
	s := &SafeSelector{}

	selector = s.escapeRegexMatches(selector, attrRegex)

	selector = escapeRegex.ReplaceAllStringFunc(selector, func(match string) string {
		// match is \\., meaning we keep the match as is in placeholder
		replaceBy := fmt.Sprintf("__esc-ph-%d__", s.index)
		s.placeholders = append(s.placeholders, match)
		s.index++
		return replaceBy
	})

	selector = nthRegex.ReplaceAllStringFunc(selector, func(match string) string {
		matches := nthRegex.FindStringSubmatch(match)
		pseudo := matches[1]
		exp := matches[2]
		replaceBy := fmt.Sprintf("__ph-%d__", s.index)
		s.placeholders = append(s.placeholders, fmt.Sprintf("(%s)", exp))
		s.index++
		return pseudo + replaceBy
	})
	s.contentStr = selector
	return s
}

func (s *SafeSelector) restore(content string) string {
	return restoreRegex.ReplaceAllStringFunc(content, func(match string) string {
		matches := restoreRegex.FindStringSubmatch(match)
		idx, _ := strconv.Atoi(matches[1])
		if idx < 0 || idx >= len(s.placeholders) {
			return match
		}
		return s.placeholders[idx]
	})
}

func (s *SafeSelector) content() string {
	return s.contentStr
}

func (s *SafeSelector) escapeRegexMatches(content string, pattern *regexp.Regexp) string {
	return pattern.ReplaceAllStringFunc(content, func(match string) string {
		replaceBy := fmt.Sprintf("__ph-%d__", s.index)
		s.placeholders = append(s.placeholders, match)
		s.index++
		return replaceBy
	})
}

func _combineHostContextSelectors(contextSelectors []string, otherSelectors string, pseudoPrefix string) string {
	hostMarker := _polyfillHostNoCombinator
	otherSelectorsHasHost := _polyfillHostRe.MatchString(otherSelectors)

	if len(contextSelectors) == 0 {
		return hostMarker + otherSelectors
	}

	combined := []string{contextSelectors[len(contextSelectors)-1]}
	contextSelectors = contextSelectors[:len(contextSelectors)-1]

	for len(contextSelectors) > 0 {
		length := len(combined)
		contextSelector := contextSelectors[len(contextSelectors)-1]
		contextSelectors = contextSelectors[:len(contextSelectors)-1]

		// we need to grow combined array to length * 3
		newCombined := make([]string, length*3)
		for i := 0; i < length; i++ {
			previousSelectors := combined[i]
			newCombined[length*2+i] = previousSelectors + " " + contextSelector
			newCombined[length+i] = contextSelector + " " + previousSelectors
			newCombined[i] = contextSelector + previousSelectors
		}
		combined = newCombined
	}

	var result []string
	for _, comb := range combined {
		if otherSelectorsHasHost {
			result = append(result, pseudoPrefix+comb+otherSelectors)
		} else {
			result = append(result, pseudoPrefix+comb+hostMarker+otherSelectors)
			result = append(result, pseudoPrefix+comb+" "+hostMarker+otherSelectors)
		}
	}
	return strings.Join(result, ",")
}

func repeatGroups(groups [][]string, multiples int) [][]string {
	length := len(groups)
	for i := 1; i < multiples; i++ {
		for j := 0; j < length; j++ {
			clone := make([]string, len(groups[j]))
			copy(clone, groups[j])
			groups = append(groups, clone)
		}
	}
	return groups
}

func (s *ShadowCss) ShimCssText(cssText string, selector string, hostSelector string) string {
	var comments []string
	cssText = _commentRe.ReplaceAllStringFunc(cssText, func(m string) string {
		if _commentWithHashRe.MatchString(m) {
			comments = append(comments, m)
		} else {
			newLinesMatches := _newLinesRe.FindAllString(m, -1)
			comments = append(comments, strings.Join(newLinesMatches, ""))
		}
		return COMMENT_PLACEHOLDER
	})

	scopedCssText := s._scopeCssText(cssText, selector, hostSelector)

	commentIdx := 0
	return _commentWithHashPlaceHolderRe.ReplaceAllStringFunc(scopedCssText, func(m string) string {
		c := comments[commentIdx]
		commentIdx++
		return c
	})
}

func (s *ShadowCss) _splitOnTopLevelCommas(text string, returnOnClosingParen bool) []string {
	var result []string
	length := len(text)
	parens := 0
	prev := 0

	for i := 0; i < length; i++ {
		charCode := text[i]

		if charCode == '(' {
			parens++
		} else if charCode == ')' {
			parens--
			if parens < 0 && returnOnClosingParen {
				result = append(result, text[prev:i])
				return result
			}
		} else if charCode == ',' && parens == 0 {
			result = append(result, text[prev:i])
			prev = i + 1
		}
	}

	result = append(result, text[prev:])
	return result
}

func (s *ShadowCss) _scopeCssText(cssText string, scopeSelector string, hostSelector string) string {
	cssText = s._insertPolyfillHostInCssText(cssText)
	cssText = s._convertColonHost(cssText)
	cssText = s._convertColonHostContext(cssText)
	if scopeSelector != "" {
		cssText = s._scopeKeyframesRelatedCss(cssText, scopeSelector)
		cssText = s._scopeSelectors(cssText, scopeSelector, hostSelector)
	}
	return strings.TrimSpace(cssText)
}

func (s *ShadowCss) _insertPolyfillHostInCssText(selector string) string {
	selector = _colonHostContextRe.ReplaceAllString(selector, _polyfillHostContext+"$1")
	selector = regexp.MustCompile(`(?i):host-context`).ReplaceAllString(selector, "__host_context_ph__")
	selector = _colonHostRe.ReplaceAllString(selector, _polyfillHost)
	selector = strings.ReplaceAll(selector, "__host_context_ph__", ":host-context")
	return selector
}

func (s *ShadowCss) _convertColonHost(cssText string) string {
	return _cssColonHostRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := _cssColonHostRe.FindStringSubmatch(match)
		hostSelectors := matches[1]
		otherSelectors := matches[2]
		if hostSelectors != "" {
			parts := s._splitOnTopLevelCommas(hostSelectors, true)
			if len(parts) > 1 {
				return ":host(" + hostSelectors + ")" + otherSelectors
			}
			trimmedHostSelector := strings.TrimSpace(parts[0])
			if trimmedHostSelector != "" {
				return _polyfillHostNoCombinator + strings.ReplaceAll(trimmedHostSelector, _polyfillHost, "") + otherSelectors
			}
		}
		return _polyfillHostNoCombinator + otherSelectors
	})
}

func (s *ShadowCss) _convertColonHostContext(cssText string) string {
	var results []string
	for _, part := range s._splitOnTopLevelCommas(cssText, false) {
		results = append(results, s._convertColonHostContextInSelectorPart(part))
	}
	return strings.Join(results, ",")
}

func (s *ShadowCss) _convertColonHostContextInSelectorPart(cssText string) string {
	return _cssColonHostContextReGlobal.ReplaceAllStringFunc(cssText, func(selectorText string) string {
		matches := _cssColonHostContextReGlobal.FindStringSubmatch(selectorText)
		pseudoPrefix := matches[1]

		contextSelectorGroups := [][]string{{}}

		startIndex := strings.Index(selectorText, _polyfillHostContext)
		for startIndex != -1 {
			afterPrefix := selectorText[startIndex+len(_polyfillHostContext):]

			if len(afterPrefix) == 0 || afterPrefix[0] != '(' {
				selectorText = afterPrefix
				startIndex = strings.Index(selectorText, _polyfillHostContext)
				continue
			}

			var newContextSelectors []string
			endIndex := 0
			for _, selector := range s._splitOnTopLevelCommas(afterPrefix[1:], true) {
				endIndex = endIndex + len(selector) + 1
				trimmed := strings.TrimSpace(selector)
				if trimmed != "" {
					newContextSelectors = append(newContextSelectors, trimmed)
				}
			}

			contextSelectorGroupsLength := len(contextSelectorGroups)
			contextSelectorGroups = repeatGroups(contextSelectorGroups, len(newContextSelectors))
			for i := 0; i < len(newContextSelectors); i++ {
				for j := 0; j < contextSelectorGroupsLength; j++ {
					contextSelectorGroups[j+i*contextSelectorGroupsLength] = append(contextSelectorGroups[j+i*contextSelectorGroupsLength], newContextSelectors[i])
				}
			}

			selectorText = afterPrefix[endIndex+1:]
			startIndex = strings.Index(selectorText, _polyfillHostContext)
		}

		var resultParts []string
		for _, contextSelectors := range contextSelectorGroups {
			resultParts = append(resultParts, _combineHostContextSelectors(contextSelectors, selectorText, pseudoPrefix))
		}
		return strings.Join(resultParts, ", ")
	})
}

func (s *ShadowCss) _scopeKeyframesRelatedCss(cssText string, scopeSelector string) string {
	localKeyframes := map[string]bool{}
	scopedKeyframesCssText := processRules(cssText, func(rule CssRule) CssRule {
		prefix, whitespace, name, ok := parseKeyframesSelector(rule.Selector)
		if !ok {
			return rule
		}
		unquotedName, quote := unquoteCssIdentifier(strings.TrimSpace(name))
		localKeyframes[canonicalCssIdentifier(unquotedName)] = true
		rule.Selector = prefix + whitespace + quote + scopeSelector + "_" + unquotedName + quote
		return rule
	})

	return processRules(scopedKeyframesCssText, func(rule CssRule) CssRule {
		if _, _, _, ok := parseKeyframesSelector(rule.Selector); ok {
			return rule
		}
		rule.Content = scopeAnimationDeclarations(rule.Content, scopeSelector, localKeyframes)
		return rule
	})
}

func parseKeyframesSelector(selector string) (prefix string, whitespace string, name string, ok bool) {
	matches := _keyframesSelectorRe.FindStringSubmatch(selector)
	if matches == nil {
		return "", "", "", false
	}
	return matches[1], matches[2], matches[3], true
}

func unquoteCssIdentifier(value string) (string, string) {
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return value[1 : len(value)-1], string(first)
		}
	}
	return value, ""
}

func canonicalCssIdentifier(value string) string {
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\'`, `'`)
	return value
}

func scopeAnimationDeclarations(content string, scopeSelector string, localKeyframes map[string]bool) string {
	content = _animationNameDeclRe.ReplaceAllStringFunc(content, func(match string) string {
		matches := _animationNameDeclRe.FindStringSubmatch(match)
		return matches[1] + matches[2] + scopeAnimationNameList(matches[3], scopeSelector, localKeyframes) + matches[4]
	})
	content = _animationDeclRe.ReplaceAllStringFunc(content, func(match string) string {
		matches := _animationDeclRe.FindStringSubmatch(match)
		return matches[1] + matches[2] + scopeAnimationValue(matches[3], scopeSelector, localKeyframes) + matches[4]
	})
	return content
}

func scopeAnimationNameList(value string, scopeSelector string, localKeyframes map[string]bool) string {
	parts := splitCssCommaList(value)
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		name, quote := unquoteCssIdentifier(trimmed)
		if localKeyframes[canonicalCssIdentifier(name)] {
			leading := part[:len(part)-len(strings.TrimLeft(part, " \t\r\n"))]
			trailing := part[len(strings.TrimRight(part, " \t\r\n")):]
			parts[i] = leading + quote + scopeSelector + "_" + name + quote + trailing
		}
	}
	return strings.Join(parts, ",")
}

func splitCssCommaList(value string) []string {
	var parts []string
	start := 0
	quote := byte(0)
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if quote != 0 {
			if ch == '\\' {
				i++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == ',' {
			parts = append(parts, value[start:i])
			start = i + 1
		}
	}
	parts = append(parts, value[start:])
	return parts
}

func scopeAnimationValue(value string, scopeSelector string, localKeyframes map[string]bool) string {
	var out strings.Builder
	for i := 0; i < len(value); {
		ch := value[i]
		if isCssWhitespace(ch) || ch == ',' {
			out.WriteByte(ch)
			i++
			continue
		}
		if ch == '\'' || ch == '"' {
			start := i
			quote := ch
			i++
			for i < len(value) {
				if value[i] == '\\' {
					i += 2
					continue
				}
				if value[i] == quote {
					i++
					break
				}
				i++
			}
			token := value[start:i]
			name, quoteStr := unquoteCssIdentifier(token)
			if localKeyframes[canonicalCssIdentifier(name)] {
				out.WriteString(quoteStr + scopeSelector + "_" + name + quoteStr)
			} else {
				out.WriteString(token)
			}
			continue
		}
		start := i
		for i < len(value) && !isCssWhitespace(value[i]) && value[i] != ',' {
			i++
		}
		token := value[start:i]
		if localKeyframes[canonicalCssIdentifier(token)] && !animationKeywords[token] {
			out.WriteString(scopeSelector + "_" + token)
		} else {
			out.WriteString(token)
		}
	}
	return out.String()
}

func (s *ShadowCss) _scopeSelectors(cssText string, scopeSelector string, hostSelector string) string {
	return processRules(cssText, func(rule CssRule) CssRule {
		selector := rule.Selector
		content := rule.Content
		if selector != "" && selector[0] != '@' {
			selector = s._scopeSelector(selector, scopeSelector, hostSelector)
		} else if startsWithAny(selector, scopedAtRuleIdentifiers) {
			content = s._scopeSelectors(content, scopeSelector, hostSelector)
		} else if strings.HasPrefix(selector, "@font-face") || strings.HasPrefix(selector, "@page") {
			content = s._stripScopingSelectors(content)
		}
		return CssRule{Selector: selector, Content: content}
	})
}

func startsWithAny(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func (s *ShadowCss) _stripScopingSelectors(cssText string) string {
	return processRules(cssText, func(rule CssRule) CssRule {
		selector := _shadowDeepSelectors.ReplaceAllString(rule.Selector, " ")
		selector = _polyfillHostNoCombinatorRe.ReplaceAllString(selector, " ")
		return CssRule{Selector: selector, Content: rule.Content}
	})
}

func (s *ShadowCss) _scopeSelector(selector string, scopeSelector string, hostSelector string) string {
	parts, delimiters := splitSelectorByTopLevelCommas(selector)
	for i, part := range parts {
		deepParts := _shadowDeepSelectors.Split(part, -1)
		if len(deepParts) == 0 {
			continue
		}
		deepParts[0] = s._applySelectorScope(deepParts[0], scopeSelector, hostSelector)
		parts[i] = strings.Join(deepParts, " ")
	}
	var out strings.Builder
	for i, part := range parts {
		if i > 0 {
			out.WriteString(delimiters[i-1])
		}
		out.WriteString(part)
	}
	return out.String()
}

func splitSelectorByTopLevelCommas(selector string) ([]string, []string) {
	var parts []string
	var delimiters []string
	parens := 0
	prev := 0
	for i := 0; i < len(selector); i++ {
		switch selector[i] {
		case '(':
			parens++
		case ')':
			if parens > 0 {
				parens--
			}
		case ',':
			if parens == 0 {
				parts = append(parts, strings.TrimSpace(selector[prev:i]))
				j := i + 1
				for j < len(selector) && isCssWhitespace(selector[j]) {
					j++
				}
				whitespace := selector[i+1 : j]
				if strings.ContainsAny(whitespace, "\n\r") {
					delimiters = append(delimiters, ", "+whitespace)
				} else {
					delimiters = append(delimiters, ", ")
				}
				prev = j
				i = j - 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(selector[prev:]))
	return parts, delimiters
}

func (s *ShadowCss) _applySelectorScope(selector string, scopeSelector string, hostSelector string) string {
	safeSelector := NewSafeSelector(selector)
	selector = safeSelector.content()

	attrName := "[" + scopeSelector + "]"
	var scoped strings.Builder
	hasHost := strings.Contains(selector, _polyfillHostNoCombinator)
	seenHost := !hasHost
	for _, token := range splitSelectorParts(selector) {
		if token.IsSeparator {
			scoped.WriteString(token.Value)
			continue
		}
		part := token.Value
		partHasHost := strings.Contains(part, _polyfillHostNoCombinator)
		if !seenHost && !partHasHost {
			scoped.WriteString(part)
			continue
		}
		scoped.WriteString(s._scopeSimpleSelectorPart(part, attrName, scopeSelector, hostSelector))
		if partHasHost {
			seenHost = true
		}
	}

	return safeSelector.restore(scoped.String())
}

type selectorToken struct {
	Value       string
	IsSeparator bool
}

func splitSelectorParts(selector string) []selectorToken {
	var tokens []selectorToken
	start := 0
	parenDepth := 0
	for i := 0; i < len(selector); i++ {
		ch := selector[i]
		if ch == '\\' {
			i++
			continue
		}
		switch ch {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case ' ', '\n', '\t', '\r', '>', '+', '~':
			if parenDepth != 0 {
				continue
			}
			if start < i {
				part := selector[start:i]
				if isEscapedHexWhitespace(part, selector, i) {
					continue
				}
				tokens = append(tokens, selectorToken{Value: selector[start:i]})
			}
			sepStart := i
			if ch == '>' || ch == '+' || ch == '~' {
				i++
				for i < len(selector) && isCssWhitespace(selector[i]) {
					i++
				}
				tokens = append(tokens, selectorToken{Value: " " + string(ch) + " ", IsSeparator: true})
				i--
				start = i + 1
				continue
			} else {
				for i < len(selector) && isCssWhitespace(selector[i]) {
					i++
				}
			}
			separator := selector[sepStart:i]
			if strings.Contains(selector[start:sepStart], "__esc-ph-") {
				separator = "   "
			}
			tokens = append(tokens, selectorToken{Value: separator, IsSeparator: true})
			i--
			start = i + 1
		}
	}
	if start < len(selector) {
		tokens = append(tokens, selectorToken{Value: selector[start:]})
	}
	return tokens
}

func isEscapedHexWhitespace(part string, selector string, index int) bool {
	if !strings.Contains(part, "__esc-ph-") {
		return false
	}
	next := index + 1
	if next >= len(selector) {
		return false
	}
	ch := selector[next]
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isCssWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\t' || ch == '\r' || ch == '\f'
}

func (s *ShadowCss) _scopeSimpleSelectorPart(part string, attrName string, scopeSelector string, hostSelector string) string {
	if strings.TrimSpace(part) == "" {
		return part
	}

	if (strings.HasPrefix(part, ":where(") || strings.HasPrefix(part, ":is(")) &&
		isStandalonePseudoFunctionChain(part) &&
		strings.Contains(part, _polyfillHostNoCombinator) {
		return s._scopePseudoFunctionChainWithHost(part, scopeSelector, hostSelector)
	}

	if strings.Contains(part, "(") && strings.Contains(part, _polyfillHostNoCombinator) {
		if scoped, ok := s._scopeHostInPseudoFunction(part, attrName, scopeSelector, hostSelector); ok {
			return scoped
		}
	}

	if strings.Contains(part, _polyfillHostNoCombinator) || _polyfillHostRe.MatchString(part) {
		return s._applySimpleSelectorScope(part, hostSelector)
	}

	if strings.HasPrefix(part, ":where(") || strings.HasPrefix(part, ":is(") {
		if !isStandalonePseudoFunctionChain(part) {
			return attrName + part
		}
		return s._scopePseudoFunctionChain(part, strings.Trim(attrName, "[]"), hostSelector)
	}

	p := _polyfillHostRe.ReplaceAllString(part, "")
	if p == "" {
		return part
	}
	pseudoIndex := firstUnescapedPseudoIndex(p)
	if pseudoIndex == -1 {
		return p + attrName
	}
	return p[:pseudoIndex] + attrName + p[pseudoIndex:]
}

func (s *ShadowCss) _scopeHostInPseudoFunction(part string, attrName string, scopeSelector string, hostSelector string) (string, bool) {
	open := strings.Index(part, "(")
	close := matchingParenIndex(part, open)
	if open == -1 || close <= open {
		return "", false
	}
	pseudoStart := strings.LastIndex(part[:open], ":")
	if pseudoStart == -1 {
		return "", false
	}
	before := part[:pseudoStart]
	pseudoPrefix := part[pseudoStart : open+1]
	inner := part[open+1 : close]
	if !strings.Contains(inner, _polyfillHostNoCombinator) {
		return "", false
	}
	innerHadLeadingWhitespace := len(inner) > 0 && isCssWhitespace(inner[0])
	after := part[close+1:]
	scopedBefore := before
	if strings.Contains(scopedBefore, _polyfillHostNoCombinator) || _polyfillHostRe.MatchString(scopedBefore) {
		scopedBefore = s._applySimpleSelectorScope(scopedBefore, hostSelector)
	} else if strings.HasPrefix(scopedBefore, ":") {
		scopedBefore = attrName + scopedBefore
	} else if scopedBefore != "" {
		scopedBefore = scopedBefore + attrName
	} else if after != "" && (strings.HasPrefix(pseudoPrefix, ":where(") || strings.HasPrefix(pseudoPrefix, ":is(")) {
		scopedBefore = attrName
	} else if strings.HasPrefix(pseudoPrefix, ":has(") {
		scopedBefore = attrName
	}
	scopeNonHostParts := (strings.HasPrefix(pseudoPrefix, ":where(") || strings.HasPrefix(pseudoPrefix, ":is(")) && !(after != "" && before == "") && !strings.HasPrefix(before, ":")
	scopedInner := s._scopeHostOnlyInSelector(inner, scopeSelector, hostSelector, scopeNonHostParts)
	if innerHadLeadingWhitespace && scopedInner != "" && !isCssWhitespace(scopedInner[0]) {
		scopedInner = " " + scopedInner
	}
	return scopedBefore + pseudoPrefix + scopedInner + ")" + after, true
}

func matchingParenIndex(value string, open int) int {
	if open < 0 || open >= len(value) || value[open] != '(' {
		return -1
	}
	depth := 1
	for i := open + 1; i < len(value); i++ {
		if value[i] == '(' {
			depth++
		} else if value[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isStandalonePseudoFunctionChain(part string) bool {
	i := 0
	for i < len(part) {
		if !(strings.HasPrefix(part[i:], ":where(") || strings.HasPrefix(part[i:], ":is(")) {
			return false
		}
		open := strings.IndexByte(part[i:], '(')
		if open == -1 {
			return false
		}
		open += i
		depth := 1
		j := open + 1
		for j < len(part) && depth > 0 {
			if part[j] == '(' {
				depth++
			} else if part[j] == ')' {
				depth--
			}
			j++
		}
		if depth != 0 {
			return false
		}
		i = j
	}
	return true
}

func (s *ShadowCss) _scopeHostOnlyInSelector(selector string, scopeSelector string, hostSelector string, scopeNonHostParts bool) string {
	parts, delimiters := splitSelectorByTopLevelCommas(selector)
	for i, part := range parts {
		if !strings.Contains(part, _polyfillHostNoCombinator) && !_polyfillHostRe.MatchString(part) {
			if scopeNonHostParts {
				parts[i] = s._scopeSelector(part, scopeSelector, hostSelector)
			}
			continue
		}
		tokens := splitSelectorParts(part)
		var out strings.Builder
		for _, token := range tokens {
			if token.IsSeparator {
				out.WriteString(token.Value)
				continue
			}
			if strings.Contains(token.Value, _polyfillHostNoCombinator) || _polyfillHostRe.MatchString(token.Value) {
				out.WriteString(s._applySimpleSelectorScope(token.Value, hostSelector))
			} else if strings.HasPrefix(token.Value, ":where(") || strings.HasPrefix(token.Value, ":is(") {
				if isStandalonePseudoFunctionChain(token.Value) {
					out.WriteString(s._scopePseudoFunctionChain(token.Value, scopeSelector, hostSelector))
				} else {
					out.WriteString("[" + scopeSelector + "]" + token.Value)
				}
			} else {
				out.WriteString(token.Value)
			}
		}
		parts[i] = out.String()
	}
	var out strings.Builder
	for i, part := range parts {
		if i > 0 {
			out.WriteString(delimiters[i-1])
		}
		out.WriteString(part)
	}
	return out.String()
}

func (s *ShadowCss) _scopePseudoFunction(part string, scopeSelector string, hostSelector string) string {
	open := strings.Index(part, "(")
	if open == -1 || !strings.HasSuffix(part, ")") {
		return part
	}
	prefix := part[:open+1]
	inner := part[open+1 : len(part)-1]
	return prefix + s._scopeSelector(inner, scopeSelector, hostSelector) + ")"
}

func (s *ShadowCss) _scopePseudoFunctionChain(part string, scopeSelector string, hostSelector string) string {
	var out strings.Builder
	for i := 0; i < len(part); {
		open := strings.IndexByte(part[i:], '(')
		if open == -1 {
			out.WriteString(part[i:])
			break
		}
		open += i
		prefix := part[i : open+1]
		depth := 1
		j := open + 1
		for j < len(part) && depth > 0 {
			if part[j] == '(' {
				depth++
			} else if part[j] == ')' {
				depth--
			}
			j++
		}
		if depth != 0 {
			out.WriteString(part[i:])
			break
		}
		inner := part[open+1 : j-1]
		out.WriteString(prefix)
		out.WriteString(s._scopeSelector(inner, scopeSelector, hostSelector))
		out.WriteString(")")
		i = j
	}
	return out.String()
}

func (s *ShadowCss) _scopePseudoFunctionChainWithHost(part string, scopeSelector string, hostSelector string) string {
	var out strings.Builder
	seenHost := false
	for i := 0; i < len(part); {
		open := strings.IndexByte(part[i:], '(')
		if open == -1 {
			out.WriteString(part[i:])
			break
		}
		open += i
		prefix := part[i : open+1]
		depth := 1
		j := open + 1
		for j < len(part) && depth > 0 {
			if part[j] == '(' {
				depth++
			} else if part[j] == ')' {
				depth--
			}
			j++
		}
		if depth != 0 {
			out.WriteString(part[i:])
			break
		}
		inner := part[open+1 : j-1]
		out.WriteString(prefix)
		if strings.Contains(inner, _polyfillHostNoCombinator) || _polyfillHostRe.MatchString(inner) {
			out.WriteString(s._scopeHostOnlyInSelector(inner, scopeSelector, hostSelector, true))
			seenHost = true
		} else if seenHost {
			out.WriteString(s._scopeSelector(inner, scopeSelector, hostSelector))
		} else {
			out.WriteString(inner)
		}
		out.WriteString(")")
		i = j
	}
	return out.String()
}

func (s *ShadowCss) _applySimpleSelectorScope(selector string, hostSelector string) string {
	replaceBy := "[" + hostSelector + "]"
	result := selector
	for _polyfillHostNoCombinatorRe.MatchString(result) {
		result = _polyfillHostNoCombinatorRe.ReplaceAllStringFunc(result, func(match string) string {
			submatches := _polyfillHostNoCombinatorRe.FindStringSubmatch(match)
			suffix := ""
			if len(submatches) > 1 {
				suffix = submatches[1]
			}
			colonIndex := firstUnescapedPseudoIndex(suffix)
			if colonIndex == -1 {
				return suffix + replaceBy
			}
			return suffix[:colonIndex] + replaceBy + suffix[colonIndex:]
		})
	}
	return _polyfillHostRe.ReplaceAllString(result, replaceBy)
}

func firstUnescapedPseudoIndex(selector string) int {
	for i := 0; i < len(selector); i++ {
		if selector[i] == ':' {
			if i > 0 && selector[i-1] == '\\' {
				continue
			}
			return i
		}
	}
	return -1
}

func processRules(input string, ruleCallback func(CssRule) CssRule) string {
	var out strings.Builder
	for i := 0; i < len(input); {
		for i < len(input) && isCssWhitespace(input[i]) {
			out.WriteByte(input[i])
			i++
		}
		if i >= len(input) {
			break
		}

		delimiter, hasBody := findNextRuleDelimiter(input, i)
		if delimiter == -1 {
			out.WriteString(input[i:])
			break
		}

		selectorRaw := input[i:delimiter]
		selectorPrefix, selectorRaw := splitLeadingCommentPlaceholders(selectorRaw)
		selector := strings.TrimSpace(selectorRaw)
		if !hasBody {
			rule := ruleCallback(CssRule{Selector: selector, Content: ""})
			out.WriteString(selectorPrefix)
			out.WriteString(rule.Selector)
			out.WriteString(";")
			i = delimiter + 1
			continue
		}

		close := findMatchingBrace(input, delimiter)
		if close == -1 {
			out.WriteString(input[i:])
			break
		}

		rule := ruleCallback(CssRule{
			Selector: selector,
			Content:  input[delimiter+1 : close],
		})
		separator := trailingCssWhitespace(selectorRaw)
		out.WriteString(selectorPrefix)
		out.WriteString(rule.Selector)
		out.WriteString(separator)
		out.WriteString("{")
		out.WriteString(rule.Content)
		out.WriteString("}")
		i = close + 1
	}
	return out.String()
}

func splitLeadingCommentPlaceholders(selectorRaw string) (string, string) {
	prefixEnd := 0
	for {
		for prefixEnd < len(selectorRaw) && isCssWhitespace(selectorRaw[prefixEnd]) {
			prefixEnd++
		}
		if !strings.HasPrefix(selectorRaw[prefixEnd:], COMMENT_PLACEHOLDER) {
			break
		}
		prefixEnd += len(COMMENT_PLACEHOLDER)
		for prefixEnd < len(selectorRaw) && isCssWhitespace(selectorRaw[prefixEnd]) {
			prefixEnd++
		}
	}
	return selectorRaw[:prefixEnd], selectorRaw[prefixEnd:]
}

func trailingCssWhitespace(value string) string {
	i := len(value)
	for i > 0 && isCssWhitespace(value[i-1]) {
		i--
	}
	return value[i:]
}

func findNextRuleDelimiter(input string, start int) (int, bool) {
	quote := byte(0)
	for i := start; i < len(input); i++ {
		ch := input[i]
		if quote != 0 {
			if ch == '\\' {
				i++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == ';' {
			return i, false
		}
		if ch == '{' {
			return i, true
		}
	}
	return -1, false
}

func findMatchingBrace(input string, open int) int {
	depth := 0
	quote := byte(0)
	for i := open; i < len(input); i++ {
		ch := input[i]
		if quote != 0 {
			if ch == '\\' {
				i++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
