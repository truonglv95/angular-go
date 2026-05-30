package serializers

import (
	"sort"
	"strings"
)

type Visitor interface {
	VisitTag(tag *Tag) any
	VisitText(text *Text) any
	VisitDeclaration(decl *Declaration) any
	VisitDoctype(doctype *Doctype) any
}

type Node interface {
	Visit(visitor Visitor) any
}

type visitor struct{}

func (v *visitor) VisitTag(tag *Tag) any {
	strAttrs := serializeAttributes(tag.Name, tag.Attrs)
	if len(tag.Children) == 0 {
		return "<" + tag.Name + strAttrs + "/>"
	}

	children := make([]string, len(tag.Children))
	for i, child := range tag.Children {
		children[i] = child.Visit(v).(string)
	}
	return "<" + tag.Name + strAttrs + ">" + strings.Join(children, "") + "</" + tag.Name + ">"
}

func (v *visitor) VisitText(text *Text) any {
	return text.Value
}

func (v *visitor) VisitDeclaration(decl *Declaration) any {
	return "<?xml" + serializeAttributes("?xml", decl.Attrs) + " ?>"
}

func (v *visitor) VisitDoctype(doctype *Doctype) any {
	return "<!DOCTYPE " + doctype.RootTag + " [\n" + doctype.Dtd + "\n]>"
}

func serializeAttributes(tagName string, attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for name := range attrs {
		keys = append(keys, name)
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, okI := attrOrder(tagName, keys[i])
		pj, okJ := attrOrder(tagName, keys[j])
		switch {
		case okI && okJ:
			if pi != pj {
				return pi < pj
			}
		case okI:
			return true
		case okJ:
			return false
		}
		return keys[i] < keys[j]
	})

	parts := make([]string, 0, len(attrs))
	for _, name := range keys {
		value := attrs[name]
		parts = append(parts, name+`="`+value+`"`)
	}
	return " " + strings.Join(parts, " ")
}

var tagAttrOrders = map[string]map[string]int{
	"?xml": {
		"version":  0,
		"encoding": 1,
	},
	"xliff": {
		"version": 0,
		"xmlns":   1,
		"srcLang": 2,
		"trgLang": 3,
	},
	"file": {
		"source-language": 0,
		"target-language": 1,
		"datatype":        2,
		"original":        3,
		"id":              4,
	},
	"trans-unit": {
		"id":       0,
		"datatype": 1,
	},
	"unit": {
		"id": 0,
	},
	"context-group": {
		"purpose": 0,
	},
	"context": {
		"context-type": 0,
	},
	"note": {
		"priority":  0,
		"from":      1,
		"category":  2,
		"id":        3,
		"appliesTo": 4,
		"ref":       5,
	},
	"msg": {
		"id":        0,
		"seq":       1,
		"name":      2,
		"desc":      3,
		"meaning":   4,
		"obsolete":  5,
		"xml:space": 6,
		"is_hidden": 7,
	},
	"ph": {
		"id":    0,
		"equiv": 1,
		"type":  2,
		"disp":  3,
		"name":  4,
	},
	"pc": {
		"id":         0,
		"equivStart": 1,
		"equivEnd":   2,
		"type":       3,
		"dispStart":  4,
		"dispEnd":    5,
	},
	"x": {
		"id":         0,
		"ctype":      1,
		"equiv-text": 2,
	},
}

func attrOrder(tagName, attr string) (int, bool) {
	if order, ok := tagAttrOrders[tagName]; ok {
		if priority, ok := order[attr]; ok {
			return priority, true
		}
	}
	return 0, false
}

var defaultVisitor Visitor = &visitor{}

func Serialize(nodes []Node) string {
	parts := make([]string, len(nodes))
	for i, node := range nodes {
		parts[i] = node.Visit(defaultVisitor).(string)
	}
	return strings.Join(parts, "")
}

type Declaration struct {
	Attrs map[string]string
}

func NewDeclaration(unescapedAttrs map[string]string) *Declaration {
	attrs := make(map[string]string, len(unescapedAttrs))
	for k, v := range unescapedAttrs {
		attrs[k] = EscapeXml(v)
	}
	return &Declaration{Attrs: attrs}
}

func (d *Declaration) Visit(visitor Visitor) any {
	return visitor.VisitDeclaration(d)
}

type Doctype struct {
	RootTag string
	Dtd     string
}

func NewDoctype(rootTag, dtd string) *Doctype {
	return &Doctype{RootTag: rootTag, Dtd: dtd}
}

func (d *Doctype) Visit(visitor Visitor) any {
	return visitor.VisitDoctype(d)
}

type Tag struct {
	Name     string
	Attrs    map[string]string
	Children []Node
}

func NewTag(name string, unescapedAttrs map[string]string, children []Node) *Tag {
	attrs := make(map[string]string, len(unescapedAttrs))
	for k, v := range unescapedAttrs {
		attrs[k] = EscapeXml(v)
	}
	return &Tag{Name: name, Attrs: attrs, Children: children}
}

func (t *Tag) Visit(visitor Visitor) any {
	return visitor.VisitTag(t)
}

type Text struct {
	Value string
}

func NewText(unescapedValue string) *Text {
	return &Text{Value: EscapeXml(unescapedValue)}
}

func (t *Text) Visit(visitor Visitor) any {
	return visitor.VisitText(t)
}

type CR struct {
	*Text
}

func NewCR(ws int) *CR {
	return &CR{Text: NewText("\n" + strings.Repeat(" ", ws))}
}

var escapedChars = []struct {
	old string
	new string
}{
	{"&", "&amp;"},
	{`"`, "&quot;"},
	{"'", "&apos;"},
	{"<", "&lt;"},
	{">", "&gt;"},
}

func EscapeXml(text string) string {
	for _, entry := range escapedChars {
		text = strings.ReplaceAll(text, entry.old, entry.new)
	}
	return text
}
