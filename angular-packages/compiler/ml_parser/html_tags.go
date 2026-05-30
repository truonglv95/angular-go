package ml_parser

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/schema"
	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
)

type HtmlTagContentType interface{}

type HtmlTagContentTypeDefault struct {
	Default  tags.TagContentType
	Override map[string]tags.TagContentType
}

type HtmlTagDefinition struct {
	closedByChildren            map[string]bool
	contentType                 interface{} // tags.TagContentType or HtmlTagContentTypeDefault
	closedByParent              bool
	implicitNamespacePrefix     *string
	isVoid                      bool
	ignoreFirstLf               bool
	canSelfClose                bool
	preventNamespaceInheritance bool
}

type HtmlTagDefinitionOptions struct {
	ClosedByChildren            []string
	ClosedByParent              bool
	ImplicitNamespacePrefix     *string
	ContentType                 interface{} // tags.TagContentType or HtmlTagContentTypeDefault
	IsVoid                      bool
	IgnoreFirstLf               bool
	PreventNamespaceInheritance bool
	CanSelfClose                *bool
}

func NewHtmlTagDefinition(opts HtmlTagDefinitionOptions) *HtmlTagDefinition {
	closedByChildren := make(map[string]bool)
	if len(opts.ClosedByChildren) > 0 {
		for _, tagName := range opts.ClosedByChildren {
			closedByChildren[tagName] = true
		}
	}

	contentType := opts.ContentType
	if contentType == nil {
		contentType = tags.TagContentTypeParsableData
	}

	canSelfClose := opts.IsVoid
	if opts.CanSelfClose != nil {
		canSelfClose = *opts.CanSelfClose
	}

	return &HtmlTagDefinition{
		closedByChildren:            closedByChildren,
		isVoid:                      opts.IsVoid,
		closedByParent:              opts.ClosedByParent || opts.IsVoid,
		implicitNamespacePrefix:     opts.ImplicitNamespacePrefix,
		contentType:                 contentType,
		ignoreFirstLf:               opts.IgnoreFirstLf,
		preventNamespaceInheritance: opts.PreventNamespaceInheritance,
		canSelfClose:                canSelfClose,
	}
}

func (t *HtmlTagDefinition) ClosedByParent() bool {
	return t.closedByParent
}

func (t *HtmlTagDefinition) ImplicitNamespacePrefix() *string {
	return t.implicitNamespacePrefix
}

func (t *HtmlTagDefinition) IsVoid() bool {
	return t.isVoid
}

func (t *HtmlTagDefinition) IgnoreFirstLf() bool {
	return t.ignoreFirstLf
}

func (t *HtmlTagDefinition) CanSelfClose() bool {
	return t.canSelfClose
}

func (t *HtmlTagDefinition) PreventNamespaceInheritance() bool {
	return t.preventNamespaceInheritance
}

func (t *HtmlTagDefinition) IsClosedByChild(name string) bool {
	return t.isVoid || t.closedByChildren[strings.ToLower(name)]
}

func (t *HtmlTagDefinition) GetContentType(prefix *string) tags.TagContentType {
	if c, ok := t.contentType.(HtmlTagContentTypeDefault); ok {
		if prefix != nil {
			if overrideType, exists := c.Override[*prefix]; exists {
				return overrideType
			}
		}
		return c.Default
	}
	return t.contentType.(tags.TagContentType)
}

var defaultTagDefinition *HtmlTagDefinition
var tagDefinitions map[string]*HtmlTagDefinition

func GetHtmlTagDefinition(tagName string) *HtmlTagDefinition {
	if tagDefinitions == nil {
		canSelfCloseDefault := true
		defaultTagDefinition = NewHtmlTagDefinition(HtmlTagDefinitionOptions{CanSelfClose: &canSelfCloseDefault})
		tagDefinitions = make(map[string]*HtmlTagDefinition)

		svgPrefix := "svg"
		mathPrefix := "math"

		defs := map[string]*HtmlTagDefinition{
			"base":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"meta":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"area":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"embed":  NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"link":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"img":    NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"input":  NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"param":  NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"hr":     NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"br":     NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"source": NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"track":  NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"wbr":    NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"p": NewHtmlTagDefinition(HtmlTagDefinitionOptions{
				ClosedByChildren: []string{
					"address", "article", "aside", "blockquote", "div", "dl", "fieldset",
					"footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup",
					"hr", "main", "nav", "ol", "p", "pre", "section", "table", "ul",
				},
				ClosedByParent: true,
			}),
			"thead": NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"tbody", "tfoot"}}),
			"tbody": NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"tbody", "tfoot"}, ClosedByParent: true}),
			"tfoot": NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"tbody"}, ClosedByParent: true}),
			"tr":    NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"tr"}, ClosedByParent: true}),
			"td":    NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"td", "th"}, ClosedByParent: true}),
			"th":    NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"td", "th"}, ClosedByParent: true}),
			"col":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true}),
			"svg":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{ImplicitNamespacePrefix: &svgPrefix}),
			"foreignObject": NewHtmlTagDefinition(HtmlTagDefinitionOptions{
				ImplicitNamespacePrefix:     &svgPrefix,
				PreventNamespaceInheritance: true,
			}),
			"math":     NewHtmlTagDefinition(HtmlTagDefinitionOptions{ImplicitNamespacePrefix: &mathPrefix}),
			"li":       NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"li"}, ClosedByParent: true}),
			"dt":       NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"dt", "dd"}}),
			"dd":       NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"dt", "dd"}, ClosedByParent: true}),
			"rb":       NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"rb", "rt", "rtc", "rp"}, ClosedByParent: true}),
			"rt":       NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"rb", "rt", "rtc", "rp"}, ClosedByParent: true}),
			"rtc":      NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"rb", "rtc", "rp"}, ClosedByParent: true}),
			"rp":       NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"rb", "rt", "rtc", "rp"}, ClosedByParent: true}),
			"optgroup": NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"optgroup"}, ClosedByParent: true}),
			"option":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{ClosedByChildren: []string{"option", "optgroup"}, ClosedByParent: true}),
			"pre":      NewHtmlTagDefinition(HtmlTagDefinitionOptions{IgnoreFirstLf: true}),
			"listing":  NewHtmlTagDefinition(HtmlTagDefinitionOptions{IgnoreFirstLf: true}),
			"style":    NewHtmlTagDefinition(HtmlTagDefinitionOptions{ContentType: tags.TagContentTypeRawText}),
			"script":   NewHtmlTagDefinition(HtmlTagDefinitionOptions{ContentType: tags.TagContentTypeRawText}),
			"title": NewHtmlTagDefinition(HtmlTagDefinitionOptions{
				ContentType: HtmlTagContentTypeDefault{
					Default: tags.TagContentTypeEscapableRawText,
					Override: map[string]tags.TagContentType{
						"svg": tags.TagContentTypeParsableData,
					},
				},
			}),
			"textarea": NewHtmlTagDefinition(HtmlTagDefinitionOptions{
				ContentType:   tags.TagContentTypeEscapableRawText,
				IgnoreFirstLf: true,
			}),
		}

		for k, v := range defs {
			tagDefinitions[k] = v
		}

		registry := schema.NewDomElementSchemaRegistry()
		canSelfCloseFalse := false
		for _, knownTagName := range registry.AllKnownElementNames() {
			if _, exists := tagDefinitions[knownTagName]; !exists {
				knownTagNameCopy := knownTagName
				if tags.GetNsPrefix(&knownTagNameCopy) == nil {
					tagDefinitions[knownTagName] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{CanSelfClose: &canSelfCloseFalse})
				}
			}
		}
	}

	if def, exists := tagDefinitions[tagName]; exists {
		return def
	}
	if def, exists := tagDefinitions[strings.ToLower(tagName)]; exists {
		return def
	}
	return defaultTagDefinition
}
