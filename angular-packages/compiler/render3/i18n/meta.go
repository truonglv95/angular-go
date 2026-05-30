package i18n

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	o "github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type I18nMeta struct {
	Id          string
	CustomId    string
	LegacyIds   []string
	Description string
	Meaning     string
}

// I18n separators for metadata
const I18N_MEANING_SEPARATOR = "|"
const I18N_ID_SEPARATOR = "@@"

// ParseI18nMeta parses i18n metas like:
// - "@@id"
// - "description[@@id]"
// - "meaning|description[@@id]"
// and returns an object with parsed output.
func ParseI18nMeta(meta string) I18nMeta {
	meta = strings.TrimSpace(meta)
	var customId, meaning, description string

	if meta != "" {
		idIndex := strings.Index(meta, I18N_ID_SEPARATOR)
		descIndex := strings.Index(meta, I18N_MEANING_SEPARATOR)

		var meaningAndDesc string
		if idIndex > -1 {
			meaningAndDesc = meta[:idIndex]
			customId = meta[idIndex+2:]
		} else {
			meaningAndDesc = meta
			customId = ""
		}

		if descIndex > -1 {
			meaning = meaningAndDesc[:descIndex]
			description = meaningAndDesc[descIndex+1:]
		} else {
			meaning = ""
			description = meaningAndDesc
		}
	}

	return I18nMeta{
		CustomId:    customId,
		Meaning:     meaning,
		Description: description,
	}
}

// I18nMetaToJSDoc converts i18n meta information for a message to a JsDoc statement
func I18nMetaToJSDoc(meta I18nMeta) []o.JSDocTag {
	var tags []o.JSDocTag
	if meta.Description != "" {
		tags = append(tags, o.JSDocTag{TagName: "desc", Text: meta.Description})
	} else {
		tags = append(tags, o.JSDocTag{TagName: "suppress", Text: "{msgDescriptions}"})
	}
	if meta.Meaning != "" {
		tags = append(tags, o.JSDocTag{TagName: "meaning", Text: meta.Meaning})
	}
	return tags
}

func HasI18nMeta(node ml_parser.Node) bool {
	return HasI18nAttrs(node)
}

// TODO: I18nMetaVisitor is quite large and depends on MLParser HTML whitespace visitor,
// digests, etc. which are not yet fully ported or imported here.
// I will stub the I18nMetaVisitor struct but wait for its dependencies to be fully ported.
type I18nMetaVisitor struct {
	HasI18nMeta                     bool
	KeepI18nAttrs                   bool
	EnableI18nLegacyMessageIdFormat bool
	PreserveSignificantWhitespace   bool
	RetainEmptyTokens               bool
	errors                          []*parse_util.ParseError
}

func NewI18nMetaVisitor(keepI18nAttrs bool, enableI18nLegacyMessageIdFormat bool, preserveSignificantWhitespace bool, retainEmptyTokens bool) *I18nMetaVisitor {
	return &I18nMetaVisitor{
		KeepI18nAttrs:                   keepI18nAttrs,
		EnableI18nLegacyMessageIdFormat: enableI18nLegacyMessageIdFormat,
		PreserveSignificantWhitespace:   preserveSignificantWhitespace,
		RetainEmptyTokens:               retainEmptyTokens,
	}
}

func (v *I18nMetaVisitor) VisitElement(element *ml_parser.Element, context any) any {
	panic("I18nMetaVisitor.VisitElement unimplemented")
}
