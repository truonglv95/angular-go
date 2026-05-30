package i18n

import (
	"fmt"
	"strings"

	xmlser "github.com/microsoft/typescript-go/angular-packages/compiler/i18n/serializers"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

const xliff2Version = "2.0"
const xliff2XMLNS = "urn:oasis:names:tc:xliff:document:2.0"
const xliff2DefaultSourceLang = "en"
const xliff2PlaceholderTag = "ph"
const xliff2PlaceholderSpanningTag = "pc"
const xliff2MarkerTag = "mrk"

const xliff2XLIFFTag = "xliff"
const xliff2SourceTag = "source"
const xliff2TargetTag = "target"
const xliff2UnitTag = "unit"

type Xliff2 struct{}

func (x *Xliff2) Write(messages []Message, locale *string) string {
	visitor := &xliff2WriteVisitor{}
	units := make([]xmlser.Node, 0, len(messages))

	for _, message := range messages {
		visitor.nextPlaceholderID = 0
		unit := xmlser.NewTag(xliff2UnitTag, map[string]string{"id": message.Id}, nil)
		notes := xmlser.NewTag("notes", map[string]string{}, nil)

		if message.Description != "" {
			notes.Children = append(notes.Children,
				xmlser.NewCR(8),
				xmlser.NewTag("note", map[string]string{"category": "description"}, []xmlser.Node{xmlser.NewText(message.Description)}),
			)
		}
		if message.Meaning != "" {
			notes.Children = append(notes.Children,
				xmlser.NewCR(8),
				xmlser.NewTag("note", map[string]string{"category": "meaning"}, []xmlser.Node{xmlser.NewText(message.Meaning)}),
			)
		}
		for _, source := range message.Sources {
			lineText := fmt.Sprintf("%s:%d", source.FilePath, source.StartLine)
			if source.EndLine != source.StartLine {
				lineText = fmt.Sprintf("%s,%d", lineText, source.EndLine)
			}
			notes.Children = append(notes.Children,
				xmlser.NewCR(8),
				xmlser.NewTag("note", map[string]string{"category": "location"}, []xmlser.Node{xmlser.NewText(lineText)}),
			)
		}
		notes.Children = append(notes.Children, xmlser.NewCR(6))

		segment := xmlser.NewTag("segment", map[string]string{}, []xmlser.Node{
			xmlser.NewCR(8),
			xmlser.NewTag(xliff2SourceTag, map[string]string{}, visitor.serialize(message.Nodes)),
			xmlser.NewCR(6),
		})

		unit.Children = append(unit.Children,
			xmlser.NewCR(6),
			notes,
			xmlser.NewCR(6),
			segment,
			xmlser.NewCR(4),
		)
		units = append(units, xmlser.NewCR(4), unit)
	}

	sourceLang := xliff2DefaultSourceLang
	if locale != nil {
		sourceLang = *locale
	}
	file := xmlser.NewTag("file", map[string]string{"original": "ng.template", "id": "ngi18n"}, append(units, xmlser.NewCR(2)))
	root := xmlser.NewTag(xliff2XLIFFTag, map[string]string{
		"version": xliff2Version,
		"xmlns":   xliff2XMLNS,
		"srcLang": sourceLang,
	}, []xmlser.Node{xmlser.NewCR(2), file, xmlser.NewCR(0)})

	return serializeI18nXML([]xmlser.Node{
		xmlser.NewDeclaration(map[string]string{"version": "1.0", "encoding": "UTF-8"}),
		xmlser.NewCR(0),
		root,
		xmlser.NewCR(0),
	})
}

func (x *Xliff2) Load(content string, url string) LoadResult {
	parser := &xliff2Parser{}
	locale, msgIdToHTML, errors := parser.parse(content, url)

	i18nNodesByMsgId := make(map[string][]Node, len(msgIdToHTML))
	converter := &xliff2XMLToI18n{}
	for msgID, html := range msgIdToHTML {
		i18nNodes, errList := converter.convert(html, url)
		errors = append(errors, errList...)
		i18nNodesByMsgId[msgID] = i18nNodes
	}
	if len(errors) > 0 {
		panic("xliff2 parse errors:\n" + joinParseErrors(errors))
	}
	return LoadResult{Locale: locale, I18nNodesByMsgId: NewEagerTranslationStore(i18nNodesByMsgId)}
}

func (x *Xliff2) Digest(message *Message) string {
	return DecimalDigest(message)
}

func (x *Xliff2) CreateNameMapper(message *Message) PlaceholderMapper { return nil }

func NewXliff2() Serializer { return &Xliff2{} }

type xliff2WriteVisitor struct {
	nextPlaceholderID int
}

func (v *xliff2WriteVisitor) VisitText(text *Text, context any) any {
	return []xmlser.Node{xmlser.NewText(text.Value)}
}

func (v *xliff2WriteVisitor) VisitContainer(container *Container, context any) any {
	var nodes []xmlser.Node
	for _, node := range container.Children {
		nodes = append(nodes, node.Visit(v, nil).([]xmlser.Node)...)
	}
	return nodes
}

func (v *xliff2WriteVisitor) VisitIcu(icu *Icu, context any) any {
	nodes := []xmlser.Node{xmlser.NewText(fmt.Sprintf("{%s, %s, ", icu.ExpressionPlaceholder, icu.Type))}
	for _, c := range icu.CaseOrders {
		nodes = append(nodes, xmlser.NewText(fmt.Sprintf("%s {", c)))
		nodes = append(nodes, icu.Cases[c].Visit(v, nil).([]xmlser.Node)...)
		nodes = append(nodes, xmlser.NewText("} "))
	}
	nodes = append(nodes, xmlser.NewText("}"))
	return nodes
}

func (v *xliff2WriteVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	typ := getXliff2TypeForTag(ph.Tag)
	if ph.IsVoid {
		tag := xmlser.NewTag(xliff2PlaceholderTag, map[string]string{
			"id":    fmt.Sprintf("%d", v.nextPlaceholderID),
			"equiv": ph.StartName,
			"type":  typ,
			"disp":  fmt.Sprintf("<%s/>", ph.Tag),
		}, nil)
		v.nextPlaceholderID++
		return []xmlser.Node{tag}
	}

	tag := xmlser.NewTag(xliff2PlaceholderSpanningTag, map[string]string{
		"id":         fmt.Sprintf("%d", v.nextPlaceholderID),
		"equivStart": ph.StartName,
		"equivEnd":   ph.CloseName,
		"type":       typ,
		"dispStart":  fmt.Sprintf("<%s>", ph.Tag),
		"dispEnd":    fmt.Sprintf("</%s>", ph.Tag),
	}, nil)
	v.nextPlaceholderID++
	nodes := v.serialize(ph.Children)
	if len(nodes) > 0 {
		tag.Children = append(tag.Children, nodes...)
	} else {
		tag.Children = append(tag.Children, xmlser.NewText(""))
	}
	return []xmlser.Node{tag}
}

func (v *xliff2WriteVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	tag := xmlser.NewTag(xliff2PlaceholderTag, map[string]string{
		"id":    fmt.Sprintf("%d", v.nextPlaceholderID),
		"equiv": ph.Name,
		"disp":  "{{" + ph.Value + "}}",
	}, nil)
	v.nextPlaceholderID++
	return []xmlser.Node{tag}
}

func (v *xliff2WriteVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	tag := xmlser.NewTag(xliff2PlaceholderSpanningTag, map[string]string{
		"id":         fmt.Sprintf("%d", v.nextPlaceholderID),
		"equivStart": ph.StartName,
		"equivEnd":   ph.CloseName,
		"type":       "other",
		"dispStart":  "@" + ph.Name,
		"dispEnd":    "}",
	}, nil)
	v.nextPlaceholderID++
	nodes := v.serialize(ph.Children)
	if len(nodes) > 0 {
		tag.Children = append(tag.Children, nodes...)
	} else {
		tag.Children = append(tag.Children, xmlser.NewText(""))
	}
	return []xmlser.Node{tag}
}

func (v *xliff2WriteVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	parts := make([]string, 0, len(ph.Value.CaseOrders))
	for _, value := range ph.Value.CaseOrders {
		parts = append(parts, value+" {...}")
	}
	tag := xmlser.NewTag(xliff2PlaceholderTag, map[string]string{
		"id":    fmt.Sprintf("%d", v.nextPlaceholderID),
		"equiv": ph.Name,
		"disp":  fmt.Sprintf("{%s, %s, %s}", ph.Value.Expression, ph.Value.Type, strings.Join(parts, " ")),
	}, nil)
	v.nextPlaceholderID++
	return []xmlser.Node{tag}
}

func (v *xliff2WriteVisitor) serialize(nodes []Node) []xmlser.Node {
	var out []xmlser.Node
	for _, node := range nodes {
		out = append(out, node.Visit(v, nil).([]xmlser.Node)...)
	}
	return out
}

type xliff2Parser struct {
	unitMLString *string
	errors       []*parse_util.ParseError
	msgIDToHTML  map[string]string
	locale       *string
}

func (p *xliff2Parser) parse(xliff string, url string) (*string, map[string]string, []*parse_util.ParseError) {
	p.unitMLString = nil
	p.msgIDToHTML = map[string]string{}

	xml := ml_parser.NewXmlParser().Parse(xliff, url, nil)
	p.errors = errorsFromML(xml.Errors)
	ml_parser.VisitAll(p, xml.RootNodes, nil)
	return p.locale, p.msgIDToHTML, p.errors
}

func (p *xliff2Parser) VisitElement(element *ml_parser.Element, context any) any {
	switch element.Name {
	case xliff2UnitTag:
		p.unitMLString = nil
		idAttr := findMLAttr(element.Attrs, "id")
		if idAttr == nil {
			p.addError(element, `<`+xliff2UnitTag+`> misses the "id" attribute`)
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
	case xliff2SourceTag:
	case xliff2TargetTag:
		inner := innerText(element)
		p.unitMLString = &inner
	case xliff2XLIFFTag:
		if localeAttr := findMLAttr(element.Attrs, "trgLang"); localeAttr != nil {
			p.locale = &localeAttr.Value
		}
		versionAttr := findMLAttr(element.Attrs, "version")
		if versionAttr != nil {
			if versionAttr.Value != "2.0" {
				p.addError(element, "The XLIFF file version "+versionAttr.Value+" is not compatible with XLIFF 2.0 serializer")
			} else {
				ml_parser.VisitAll(p, element.Children, nil)
			}
		}
	default:
		ml_parser.VisitAll(p, element.Children, nil)
	}
	return nil
}

func (p *xliff2Parser) VisitAttribute(attribute *ml_parser.Attribute, context any) any { return nil }
func (p *xliff2Parser) VisitText(text *ml_parser.Text, context any) any                { return nil }
func (p *xliff2Parser) VisitComment(comment *ml_parser.Comment, context any) any       { return nil }
func (p *xliff2Parser) VisitExpansion(expansion *ml_parser.Expansion, context any) any { return nil }
func (p *xliff2Parser) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	return nil
}
func (p *xliff2Parser) VisitBlock(block *ml_parser.Block, context any) any { return nil }
func (p *xliff2Parser) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}
func (p *xliff2Parser) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	return nil
}
func (p *xliff2Parser) VisitComponent(component *ml_parser.Component, context any) any { return nil }
func (p *xliff2Parser) VisitDirective(directive *ml_parser.Directive, context any) any { return nil }

func (p *xliff2Parser) addError(node ml_parser.Node, message string) {
	p.errors = append(p.errors, parse_util.NewParseError(node.GetSourceSpan(), message, nil, nil))
}

type xliff2XMLToI18n struct {
	errors []*parse_util.ParseError
}

func (c *xliff2XMLToI18n) convert(message string, url string) ([]Node, []*parse_util.ParseError) {
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

func (c *xliff2XMLToI18n) VisitText(text *ml_parser.Text, context any) any {
	return NewText(text.Value, text.SourceSpan)
}

func (c *xliff2XMLToI18n) VisitElement(el *ml_parser.Element, context any) any {
	switch el.Name {
	case xliff2PlaceholderTag:
		nameAttr := findMLAttr(el.Attrs, "equiv")
		if nameAttr != nil {
			return []Node{NewPlaceholder("", nameAttr.Value, el.SourceSpan)}
		}
		c.addError(el, `<`+xliff2PlaceholderTag+`> misses the "equiv" attribute`)
	case xliff2PlaceholderSpanningTag:
		startAttr := findMLAttr(el.Attrs, "equivStart")
		endAttr := findMLAttr(el.Attrs, "equivEnd")
		if startAttr == nil {
			c.addError(el, `<`+xliff2PlaceholderTag+`> misses the "equivStart" attribute`)
		} else if endAttr == nil {
			c.addError(el, `<`+xliff2PlaceholderTag+`> misses the "equivEnd" attribute`)
		} else {
			nodes := []Node{NewPlaceholder("", startAttr.Value, el.SourceSpan)}
			for _, child := range el.Children {
				nodes = appendNodes(nodes, child.Visit(c, nil))
			}
			nodes = append(nodes, NewPlaceholder("", endAttr.Value, el.SourceSpan))
			return nodes
		}
	case xliff2MarkerTag:
		var out []Node
		for _, visited := range ml_parser.VisitAll(c, el.Children, nil) {
			out = appendNodes(out, visited)
		}
		return out
	default:
		c.addError(el, "Unexpected tag")
	}
	return nil
}

func (c *xliff2XMLToI18n) VisitExpansion(icu *ml_parser.Expansion, context any) any {
	caseMap := make(map[string]Node)
	caseOrders := make([]string, 0, len(icu.Cases))
	for _, visited := range ml_parser.VisitAll(c, expansionCasesToNodes(icu.Cases), nil) {
		caseValue := visited.(xliffCaseValue)
		caseMap[caseValue.value] = NewContainer(caseValue.nodes, icu.SourceSpan)
		caseOrders = append(caseOrders, caseValue.value)
	}
	return NewIcu(icu.SwitchValue, icu.Type, caseMap, caseOrders, icu.SourceSpan, "")
}

func (c *xliff2XMLToI18n) VisitExpansionCase(icuCase *ml_parser.ExpansionCase, context any) any {
	var nodes []Node
	for _, visited := range ml_parser.VisitAll(c, icuCase.Expression, nil) {
		nodes = appendNodes(nodes, visited)
	}
	return xliffCaseValue{value: icuCase.Value, nodes: nodes}
}

func (c *xliff2XMLToI18n) VisitComment(comment *ml_parser.Comment, context any) any       { return nil }
func (c *xliff2XMLToI18n) VisitAttribute(attribute *ml_parser.Attribute, context any) any { return nil }
func (c *xliff2XMLToI18n) VisitBlock(block *ml_parser.Block, context any) any             { return nil }
func (c *xliff2XMLToI18n) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}
func (c *xliff2XMLToI18n) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	return nil
}
func (c *xliff2XMLToI18n) VisitComponent(component *ml_parser.Component, context any) any {
	c.addError(component, "Unexpected node")
	return nil
}
func (c *xliff2XMLToI18n) VisitDirective(directive *ml_parser.Directive, context any) any {
	c.addError(directive, "Unexpected node")
	return nil
}

func (c *xliff2XMLToI18n) addError(node ml_parser.Node, message string) {
	c.errors = append(c.errors, parse_util.NewParseError(node.GetSourceSpan(), message, nil, nil))
}
