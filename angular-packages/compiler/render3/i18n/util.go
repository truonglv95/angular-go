package i18n

import (
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	o "github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// Name of the i18n attributes
const I18N_ATTR = "i18n"
const I18N_ATTR_PREFIX = "i18n-"

// Prefix of var expressions used in ICUs
const I18N_ICU_VAR_PREFIX = "VAR_"

func IsI18nAttribute(name string) bool {
	return name == I18N_ATTR || strings.HasPrefix(name, I18N_ATTR_PREFIX)
}

func HasI18nAttrs(node ml_parser.Node) bool {
	switch n := node.(type) {
	case *ml_parser.Element:
		for _, attr := range n.Attrs {
			if IsI18nAttribute(attr.Name) {
				return true
			}
		}
	}
	return false
}

func IcuFromI18nMessage(message *i18n.Message) *i18n.IcuPlaceholder {
	if len(message.Nodes) > 0 {
		if icu, ok := message.Nodes[0].(*i18n.IcuPlaceholder); ok {
			return icu
		}
	}
	return nil
}

func PlaceholdersToParams(placeholders map[string][]string) map[string]o.Expression {
	params := make(map[string]o.Expression)
	for key, values := range placeholders {
		if len(values) > 1 {
			params[key] = o.NewLiteralExpr("["+strings.Join(values, "|")+"]", nil, nil, nil)
		} else if len(values) == 1 {
			params[key] = o.NewLiteralExpr(values[0], nil, nil, nil)
		}
	}
	return params
}

// FormatI18nPlaceholderNamesInMap format the placeholder names in a map of placeholders to expressions.
func FormatI18nPlaceholderNamesInMap(params map[string]o.Expression, useCamelCase bool) map[string]o.Expression {
	_params := make(map[string]o.Expression)
	if params != nil && len(params) > 0 {
		for key, val := range params {
			_params[FormatI18nPlaceholderName(key, useCamelCase)] = val
		}
	}
	return _params
}

func toPublicName(internalName string) string {
	return regexp.MustCompile(`[^A-Z0-9_]`).ReplaceAllString(strings.ToUpper(internalName), "_")
}

// FormatI18nPlaceholderName converts internal placeholder names to public-facing format
func FormatI18nPlaceholderName(name string, useCamelCase bool) string {
	publicName := toPublicName(name)
	if !useCamelCase {
		return publicName
	}
	chunks := strings.Split(publicName, "_")
	if len(chunks) == 1 {
		return strings.ToLower(name)
	}
	var postfix *string
	// eject last element if it's a number
	lastChunk := chunks[len(chunks)-1]
	if regexp.MustCompile(`^\d+$`).MatchString(lastChunk) {
		postfix = &lastChunk
		chunks = chunks[:len(chunks)-1]
	}
	raw := strings.ToLower(chunks[0])
	chunks = chunks[1:]
	if len(chunks) > 0 {
		for _, c := range chunks {
			if len(c) > 0 {
				raw += strings.ToUpper(c[0:1]) + strings.ToLower(c[1:])
			}
		}
	}
	if postfix != nil {
		return raw + "_" + *postfix
	}
	return raw
}
