package template_parser

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
)

// Port of packages/compiler/src/template_parser/template_preparser.ts

const ngContentSelectAttr = "select"
const linkElement = "link"
const linkStyleRelAttr = "rel"
const linkStyleHrefAttr = "href"
const linkStyleRelValue = "stylesheet"
const ngNonBindableAttr = "ngNonBindable"
const ngProjectAs = "ngProjectAs"

// styleElements is the set of style element names (matches STYLE_ELEMENTS in TS).
var styleElements = map[string]bool{
	":svg:style": true,
	"style":      true,
}

// scriptElements is the set of script element names (matches SCRIPT_ELEMENTS in TS).
var scriptElements = map[string]bool{
	":svg:script": true,
	"script":      true,
}

// PreparsedElementType mirrors the PreparsedElementType enum in template_preparser.ts.
type PreparsedElementType int

const (
	PreparsedElementTypeNgContent  PreparsedElementType = iota // NG_CONTENT
	PreparsedElementTypeStyle                                   // STYLE
	PreparsedElementTypeStylesheet                              // STYLESHEET
	PreparsedElementTypeScript                                  // SCRIPT
	PreparsedElementTypeOther                                   // OTHER
)

// PreparsedElement mirrors the PreparsedElement class in template_preparser.ts.
// Fields match the TypeScript constructor parameters in order:
//
//	type, selectAttr, hrefAttr, nonBindable, projectAs
type PreparsedElement struct {
	Type        PreparsedElementType
	SelectAttr  string  // always set; defaults to "*" when the select attr is absent
	HrefAttr    *string // null in TS when not present
	NonBindable bool
	ProjectAs   string
}

// PreparseElement is the Go port of preparseElement() in template_preparser.ts.
func PreparseElement(ast *ml_parser.Element) PreparsedElement {
	var selectAttr string // null in TS, empty string here until assigned
	var hrefAttr *string
	var relAttr *string
	nonBindable := false
	projectAs := ""

	for _, attr := range ast.Attrs {
		lcAttrName := strings.ToLower(attr.Name)
		if lcAttrName == ngContentSelectAttr {
			selectAttr = attr.Value
		} else if lcAttrName == linkStyleHrefAttr {
			v := attr.Value
			hrefAttr = &v
		} else if lcAttrName == linkStyleRelAttr {
			v := attr.Value
			relAttr = &v
		} else if attr.Name == ngNonBindableAttr {
			nonBindable = true
		} else if attr.Name == ngProjectAs {
			if len(attr.Value) > 0 {
				projectAs = attr.Value
			}
		}
	}

	// Normalize selector to '*' if empty (matches TS: selectAttr ||= '*')
	if selectAttr == "" {
		selectAttr = "*"
	}

	nodeName := strings.ToLower(ast.Name)
	elemType := PreparsedElementTypeOther

	if tags.IsNgContent(nodeName) {
		elemType = PreparsedElementTypeNgContent
	} else if styleElements[nodeName] {
		elemType = PreparsedElementTypeStyle
	} else if scriptElements[nodeName] {
		elemType = PreparsedElementTypeScript
	} else if nodeName == linkElement && relAttr != nil && *relAttr == linkStyleRelValue {
		elemType = PreparsedElementTypeStylesheet
	}

	return PreparsedElement{
		Type:        elemType,
		SelectAttr:  selectAttr,
		HrefAttr:    hrefAttr,
		NonBindable: nonBindable,
		ProjectAs:   projectAs,
	}
}
