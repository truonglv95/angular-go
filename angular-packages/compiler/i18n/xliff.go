package i18n

import (
	"fmt"
	"strings"

	xmlser "github.com/microsoft/typescript-go/angular-packages/compiler/i18n/serializers"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

const xliffVersion = "1.2"
const xliffXMLNS = "urn:oasis:names:tc:xliff:document:1.2"
const xliffDefaultSourceLang = "en"
const xliffPlaceholderTag = "x"
const xliffMarkerTag = "mrk"

const xliffFileTag = "file"
const xliffSourceTag = "source"
const xliffSegSourceTag = "seg-source"
const xliffAltTransTag = "alt-trans"
const xliffTargetTag = "target"
const xliffUnitTag = "trans-unit"
const xliffContextGroupTag = "context-group"
const xliffContextTag = "context"

type Xliff struct{}

func (x *Xliff) Write(messages []Message, locale *string) string {
	visitor := &xliffWriteVisitor{}
	transUnits := make([]xmlser.Node, 0, len(messages)*2)

	for _, message := range messages {
		contextTags := make([]xmlser.Node, 0, len(message.Sources)*2)
		for _, source := range message.Sources {
			contextGroupTag := xmlser.NewTag(xliffContextGroupTag, map[string]string{"purpose": "location"}, []xmlser.Node{
				xmlser.NewCR(10),
				xmlser.NewTag(xliffContextTag, map[string]string{"context-type": "sourcefile"}, []xmlser.Node{xmlser.NewText(source.FilePath)}),
				xmlser.NewCR(10),
				xmlser.NewTag(xliffContextTag, map[string]string{"context-type": "linenumber"}, []xmlser.Node{xmlser.NewText(fmt.Sprintf("%d", source.StartLine))}),
				xmlser.NewCR(8),
			})
			contextTags = append(contextTags, xmlser.NewCR(8), contextGroupTag)
		}

		transUnit := xmlser.NewTag(xliffUnitTag, map[string]string{"id": message.Id, "datatype": "html"}, nil)
		transUnit.Children = append(transUnit.Children,
			xmlser.NewCR(8),
			xmlser.NewTag(xliffSourceTag, map[string]string{}, visitor.serialize(message.Nodes)),
		)
		transUnit.Children = append(transUnit.Children, contextTags...)

		if message.Description != "" {
			transUnit.Children = append(transUnit.Children,
				xmlser.NewCR(8),
				xmlser.NewTag("note", map[string]string{"priority": "1", "from": "description"}, []xmlser.Node{xmlser.NewText(message.Description)}),
			)
		}
		if message.Meaning != "" {
			transUnit.Children = append(transUnit.Children,
				xmlser.NewCR(8),
				xmlser.NewTag("note", map[string]string{"priority": "1", "from": "meaning"}, []xmlser.Node{xmlser.NewText(message.Meaning)}),
			)
		}

		transUnit.Children = append(transUnit.Children, xmlser.NewCR(6))
		transUnits = append(transUnits, xmlser.NewCR(6), transUnit)
	}

	sourceLang := xliffDefaultSourceLang
	if locale != nil {
		sourceLang = *locale
	}
	body := xmlser.NewTag("body", map[string]string{}, append(transUnits, xmlser.NewCR(4)))
	file := xmlser.NewTag(xliffFileTag, map[string]string{
		"source-language": sourceLang,
		"datatype":        "plaintext",
		"original":        "ng2.template",
	}, []xmlser.Node{xmlser.NewCR(4), body, xmlser.NewCR(2)})
	root := xmlser.NewTag("xliff", map[string]string{"version": xliffVersion, "xmlns": xliffXMLNS}, []xmlser.Node{
		xmlser.NewCR(2),
		file,
		xmlser.NewCR(0),
	})

	return serializeI18nXML([]xmlser.Node{
		xmlser.NewDeclaration(map[string]string{"version": "1.0", "encoding": "UTF-8"}),
		xmlser.NewCR(0),
		root,
		xmlser.NewCR(0),
	})
}

func (x *Xliff) Load(content string, url string) LoadResult {
	parser := &xliffParser{}
	locale, msgIdToHTML, errors := parser.parse(content, url)

	i18nNodesByMsgId := make(map[string][]Node, len(msgIdToHTML))
	converter := &xliffXMLToI18n{}
	for msgID, html := range msgIdToHTML {
		i18nNodes, errList := converter.convert(html, url)
		errors = append(errors, errList...)
		i18nNodesByMsgId[msgID] = i18nNodes
	}

	if len(errors) > 0 {
		panic("xliff parse errors:\n" + joinParseErrors(errors))
	}
	return LoadResult{Locale: locale, I18nNodesByMsgId: NewEagerTranslationStore(i18nNodesByMsgId)}
}

func (x *Xliff) Digest(message *Message) string {
	return Digest(message)
}

func (x *Xliff) CreateNameMapper(message *Message) PlaceholderMapper { return nil }

func NewXliff() Serializer { return &Xliff{} }

type xliffWriteVisitor struct{}

func (v *xliffWriteVisitor) VisitText(text *Text, context any) any {
	return []xmlser.Node{xmlser.NewText(text.Value)}
}

func (v *xliffWriteVisitor) VisitContainer(container *Container, context any) any {
	var nodes []xmlser.Node
	for _, node := range container.Children {
		nodes = append(nodes, node.Visit(v, nil).([]xmlser.Node)...)
	}
	return nodes
}

func (v *xliffWriteVisitor) VisitIcu(icu *Icu, context any) any {
	nodes := []xmlser.Node{xmlser.NewText(fmt.Sprintf("{%s, %s, ", icu.ExpressionPlaceholder, icu.Type))}
	for _, c := range icu.CaseOrders {
		nodes = append(nodes, xmlser.NewText(fmt.Sprintf("%s {", c)))
		nodes = append(nodes, icu.Cases[c].Visit(v, nil).([]xmlser.Node)...)
		nodes = append(nodes, xmlser.NewText("} "))
	}
	nodes = append(nodes, xmlser.NewText("}"))
	return nodes
}

func (v *xliffWriteVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	ctype := getXliffCtypeForTag(ph.Tag)
	if ph.IsVoid {
		return []xmlser.Node{xmlser.NewTag(xliffPlaceholderTag, map[string]string{
			"id":         ph.StartName,
			"ctype":      ctype,
			"equiv-text": fmt.Sprintf("<%s/>", ph.Tag),
		}, nil)}
	}

	start := xmlser.NewTag(xliffPlaceholderTag, map[string]string{
		"id":         ph.StartName,
		"ctype":      ctype,
		"equiv-text": fmt.Sprintf("<%s>", ph.Tag),
	}, nil)
	end := xmlser.NewTag(xliffPlaceholderTag, map[string]string{
		"id":         ph.CloseName,
		"ctype":      ctype,
		"equiv-text": fmt.Sprintf("</%s>", ph.Tag),
	}, nil)

	nodes := []xmlser.Node{start}
	nodes = append(nodes, v.serialize(ph.Children)...)
	nodes = append(nodes, end)
	return nodes
}

func (v *xliffWriteVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	return []xmlser.Node{xmlser.NewTag(xliffPlaceholderTag, map[string]string{
		"id":         ph.Name,
		"equiv-text": "{{" + ph.Value + "}}",
	}, nil)}
}

func (v *xliffWriteVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	ctype := "x-" + strings.ToLower(nonAlphaNumRE.ReplaceAllString(ph.Name, "-"))
	start := xmlser.NewTag(xliffPlaceholderTag, map[string]string{
		"id":         ph.StartName,
		"ctype":      ctype,
		"equiv-text": "@" + ph.Name,
	}, nil)
	end := xmlser.NewTag(xliffPlaceholderTag, map[string]string{
		"id":         ph.CloseName,
		"ctype":      ctype,
		"equiv-text": "}",
	}, nil)

	nodes := []xmlser.Node{start}
	nodes = append(nodes, v.serialize(ph.Children)...)
	nodes = append(nodes, end)
	return nodes
}

func (v *xliffWriteVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	parts := make([]string, 0, len(ph.Value.CaseOrders))
	for _, value := range ph.Value.CaseOrders {
		parts = append(parts, value+" {...}")
	}
	return []xmlser.Node{xmlser.NewTag(xliffPlaceholderTag, map[string]string{
		"id":         ph.Name,
		"equiv-text": fmt.Sprintf("{%s, %s, %s}", ph.Value.Expression, ph.Value.Type, strings.Join(parts, " ")),
	}, nil)}
}

func (v *xliffWriteVisitor) serialize(nodes []Node) []xmlser.Node {
	var out []xmlser.Node
	for _, node := range nodes {
		out = append(out, node.Visit(v, nil).([]xmlser.Node)...)
	}
	return out
}

type xliffParser struct {
	unitMLString *string
	errors       []*parse_util.ParseError
	msgIDToHTML  map[string]string
	locale       *string
}

func (p *xliffParser) parse(xliff string, url string) (*string, map[string]string, []*parse_util.ParseError) {
	p.unitMLString = nil
	p.msgIDToHTML = map[string]string{}

	xml := ml_parser.NewXmlParser().Parse(xliff, url, nil)
	p.errors = errorsFromML(xml.Errors)
	ml_parser.VisitAll(p, xml.RootNodes, nil)
	return p.locale, p.msgIDToHTML, p.errors
}

func (p *xliffParser) VisitElement(element *ml_parser.Element, context any) any {
	switch element.Name {
	case xliffUnitTag:
		p.unitMLString = nil
		idAttr := findMLAttr(element.Attrs, "id")
		if idAttr == nil {
			p.addError(element, `<`+xliffUnitTag+`> misses the "id" attribute`)
		} else if _, ok := p.msgIDToHTML[idAttr.Value]; ok {
			p.addError(element, "Duplicated translations for msg "+idAttr.Value)
		} else {
			ml_parser.VisitAll(p, element.Children, nil)
			if p.unitMLString != nil {
				p.msgIDToHTML[idAttr.Value] = *p.unitMLString
			} else {
				p.addError(element, "Message "+idAttr.Value+" misses a translation")
			}
		}
	case xliffSourceTag, xliffSegSourceTag, xliffAltTransTag:
	case xliffTargetTag:
		inner := innerText(element)
		p.unitMLString = &inner
	case xliffFileTag:
		if localeAttr := findMLAttr(element.Attrs, "target-language"); localeAttr != nil {
			p.locale = &localeAttr.Value
		}
		ml_parser.VisitAll(p, element.Children, nil)
	default:
		ml_parser.VisitAll(p, element.Children, nil)
	}
	return nil
}

func (p *xliffParser) VisitAttribute(attribute *ml_parser.Attribute, context any) any { return nil }
func (p *xliffParser) VisitText(text *ml_parser.Text, context any) any                { return nil }
func (p *xliffParser) VisitComment(comment *ml_parser.Comment, context any) any       { return nil }
func (p *xliffParser) VisitExpansion(expansion *ml_parser.Expansion, context any) any { return nil }
func (p *xliffParser) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	return nil
}
func (p *xliffParser) VisitBlock(block *ml_parser.Block, context any) any { return nil }
func (p *xliffParser) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}
func (p *xliffParser) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	return nil
}
func (p *xliffParser) VisitComponent(component *ml_parser.Component, context any) any { return nil }
func (p *xliffParser) VisitDirective(directive *ml_parser.Directive, context any) any { return nil }

func (p *xliffParser) addError(node ml_parser.Node, message string) {
	p.errors = append(p.errors, parse_util.NewParseError(node.GetSourceSpan(), message, nil, nil))
}

type xliffXMLToI18n struct {
	errors []*parse_util.ParseError
}

func (c *xliffXMLToI18n) convert(message string, url string) ([]Node, []*parse_util.ParseError) {
	xmlIcu := ml_parser.NewXmlParser().Parse(message, url, &ml_parser.TokenizeOptions{TokenizeExpansionForms: true})
	c.errors = errorsFromML(xmlIcu.Errors)
	if len(c.errors) > 0 || len(xmlIcu.RootNodes) == 0 {
		return []Node{}, c.errors
	}
	var out []Node
	for _, visited := range ml_parser.VisitAll(c, xmlIcu.RootNodes, nil) {
		out = appendNodes(out, visited)
	}
	return out, c.errors
}

func (c *xliffXMLToI18n) VisitText(text *ml_parser.Text, context any) any {
	return NewText(text.Value, text.SourceSpan)
}

func (c *xliffXMLToI18n) VisitElement(el *ml_parser.Element, context any) any {
	if el.Name == xliffPlaceholderTag {
		nameAttr := findMLAttr(el.Attrs, "id")
		if nameAttr != nil {
			return NewPlaceholder("", nameAttr.Value, el.SourceSpan)
		}
		c.addError(el, `<`+xliffPlaceholderTag+`> misses the "id" attribute`)
		return nil
	}
	if el.Name == xliffMarkerTag {
		var out []Node
		for _, visited := range ml_parser.VisitAll(c, el.Children, nil) {
			out = appendNodes(out, visited)
		}
		return out
	}
	c.addError(el, "Unexpected tag")
	return nil
}

func (c *xliffXMLToI18n) VisitExpansion(icu *ml_parser.Expansion, context any) any {
	caseMap := make(map[string]Node)
	caseOrders := make([]string, 0, len(icu.Cases))
	for _, visited := range ml_parser.VisitAll(c, expansionCasesToNodes(icu.Cases), nil) {
		caseValue := visited.(xliffCaseValue)
		caseMap[caseValue.value] = NewContainer(caseValue.nodes, icu.SourceSpan)
		caseOrders = append(caseOrders, caseValue.value)
	}
	return NewIcu(icu.SwitchValue, icu.Type, caseMap, caseOrders, icu.SourceSpan, "")
}

func (c *xliffXMLToI18n) VisitExpansionCase(icuCase *ml_parser.ExpansionCase, context any) any {
	var nodes []Node
	for _, visited := range ml_parser.VisitAll(c, icuCase.Expression, nil) {
		nodes = appendNodes(nodes, visited)
	}
	return xliffCaseValue{value: icuCase.Value, nodes: nodes}
}

func (c *xliffXMLToI18n) VisitComment(comment *ml_parser.Comment, context any) any       { return nil }
func (c *xliffXMLToI18n) VisitAttribute(attribute *ml_parser.Attribute, context any) any { return nil }
func (c *xliffXMLToI18n) VisitBlock(block *ml_parser.Block, context any) any             { return nil }
func (c *xliffXMLToI18n) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}
func (c *xliffXMLToI18n) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	return nil
}
func (c *xliffXMLToI18n) VisitComponent(component *ml_parser.Component, context any) any {
	c.addError(component, "Unexpected node")
	return nil
}
func (c *xliffXMLToI18n) VisitDirective(directive *ml_parser.Directive, context any) any {
	c.addError(directive, "Unexpected node")
	return nil
}

func (c *xliffXMLToI18n) addError(node ml_parser.Node, message string) {
	c.errors = append(c.errors, parse_util.NewParseError(node.GetSourceSpan(), message, nil, nil))
}

type xliffCaseValue struct {
	value string
	nodes []Node
}
