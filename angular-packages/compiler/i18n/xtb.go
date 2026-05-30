package i18n

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

const xtbTranslationsTag = "translationbundle"
const xtbTranslationTag = "translation"
const xtbPlaceholderTag = "ph"

type Xtb struct{}

func (x *Xtb) Write(messages []Message, locale *string) string {
	_ = messages
	_ = locale
	panic("Unsupported")
}

func (x *Xtb) Load(content string, url string) LoadResult {
	parser := &xtbParser{}
	locale, msgIdToHTML, errors := parser.parse(content, url)

	loaders := make(map[string]func() ([]Node, error), len(msgIdToHTML))
	ids := make([]string, 0, len(msgIdToHTML))
	for msgID, html := range msgIdToHTML {
		ids = append(ids, msgID)
		msgHTML := html
		loaders[msgID] = func() ([]Node, error) {
			converter := &xtbXMLToI18n{}
			i18nNodes, errList := converter.convert(msgHTML, url)
			if len(errList) > 0 {
				return nil, fmt.Errorf("xtb parse errors:\n%s", joinParseErrors(errList))
			}
			return i18nNodes, nil
		}
	}

	if len(errors) > 0 {
		panic("xtb parse errors:\n" + joinParseErrors(errors))
	}
	return LoadResult{Locale: locale, I18nNodesByMsgId: NewLazyTranslationStore(ids, loaders)}
}

func (x *Xtb) Digest(message *Message) string {
	return (&Xmb{}).Digest(message)
}

func (x *Xtb) CreateNameMapper(message *Message) PlaceholderMapper {
	return NewSimplePlaceholderMapper(message, xmbToPublicName)
}

func NewXtb() Serializer { return &Xtb{} }

type xtbParser struct {
	bundleDepth int
	errors      []*parse_util.ParseError
	msgIDToHTML map[string]string
	locale      *string
}

func (p *xtbParser) parse(xtb string, url string) (*string, map[string]string, []*parse_util.ParseError) {
	p.bundleDepth = 0
	p.msgIDToHTML = map[string]string{}

	xml := ml_parser.NewXmlParser().Parse(xtb, url, nil)
	p.errors = errorsFromML(xml.Errors)
	ml_parser.VisitAll(p, xml.RootNodes, nil)
	return p.locale, p.msgIDToHTML, p.errors
}

func (p *xtbParser) VisitElement(element *ml_parser.Element, context any) any {
	switch element.Name {
	case xtbTranslationsTag:
		p.bundleDepth++
		if p.bundleDepth > 1 {
			p.addError(element, `<`+xtbTranslationsTag+`> elements can not be nested`)
		}
		if langAttr := findMLAttr(element.Attrs, "lang"); langAttr != nil {
			p.locale = &langAttr.Value
		}
		ml_parser.VisitAll(p, element.Children, nil)
		p.bundleDepth--
	case xtbTranslationTag:
		idAttr := findMLAttr(element.Attrs, "id")
		if idAttr == nil {
			p.addError(element, `<`+xtbTranslationTag+`> misses the "id" attribute`)
		} else if _, ok := p.msgIDToHTML[idAttr.Value]; ok {
			p.addError(element, "Duplicated translations for msg "+idAttr.Value)
		} else {
			p.msgIDToHTML[idAttr.Value] = innerText(element)
		}
	default:
		p.addError(element, "Unexpected tag")
	}
	return nil
}

func (p *xtbParser) VisitAttribute(attribute *ml_parser.Attribute, context any) any { return nil }
func (p *xtbParser) VisitText(text *ml_parser.Text, context any) any                { return nil }
func (p *xtbParser) VisitComment(comment *ml_parser.Comment, context any) any       { return nil }
func (p *xtbParser) VisitExpansion(expansion *ml_parser.Expansion, context any) any { return nil }
func (p *xtbParser) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context any) any {
	return nil
}
func (p *xtbParser) VisitBlock(block *ml_parser.Block, context any) any { return nil }
func (p *xtbParser) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}
func (p *xtbParser) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any { return nil }
func (p *xtbParser) VisitComponent(component *ml_parser.Component, context any) any {
	p.addError(component, "Unexpected node")
	return nil
}
func (p *xtbParser) VisitDirective(directive *ml_parser.Directive, context any) any {
	p.addError(directive, "Unexpected node")
	return nil
}

func (p *xtbParser) addError(node ml_parser.Node, message string) {
	p.errors = append(p.errors, parse_util.NewParseError(node.GetSourceSpan(), message, nil, nil))
}

type xtbXMLToI18n struct {
	errors []*parse_util.ParseError
}

func (c *xtbXMLToI18n) convert(message string, url string) ([]Node, []*parse_util.ParseError) {
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

func (c *xtbXMLToI18n) VisitText(text *ml_parser.Text, context any) any {
	return NewText(text.Value, text.SourceSpan)
}

func (c *xtbXMLToI18n) VisitExpansion(icu *ml_parser.Expansion, context any) any {
	caseMap := make(map[string]Node)
	caseOrders := make([]string, 0, len(icu.Cases))
	for _, visited := range ml_parser.VisitAll(c, expansionCasesToNodes(icu.Cases), nil) {
		caseValue := visited.(xtbCaseValue)
		caseMap[caseValue.value] = NewContainer(caseValue.nodes, icu.SourceSpan)
		caseOrders = append(caseOrders, caseValue.value)
	}
	return NewIcu(icu.SwitchValue, icu.Type, caseMap, caseOrders, icu.SourceSpan, "")
}

func (c *xtbXMLToI18n) VisitExpansionCase(icuCase *ml_parser.ExpansionCase, context any) any {
	var nodes []Node
	for _, visited := range ml_parser.VisitAll(c, icuCase.Expression, nil) {
		nodes = appendNodes(nodes, visited)
	}
	return xtbCaseValue{value: icuCase.Value, nodes: nodes}
}

func (c *xtbXMLToI18n) VisitElement(el *ml_parser.Element, context any) any {
	if el.Name == xtbPlaceholderTag {
		nameAttr := findMLAttr(el.Attrs, "name")
		if nameAttr != nil {
			return NewPlaceholder("", nameAttr.Value, el.SourceSpan)
		}
		c.addError(el, `<`+xtbPlaceholderTag+`> misses the "name" attribute`)
	} else {
		c.addError(el, "Unexpected tag")
	}
	return nil
}

func (c *xtbXMLToI18n) VisitComment(comment *ml_parser.Comment, context any) any       { return nil }
func (c *xtbXMLToI18n) VisitAttribute(attribute *ml_parser.Attribute, context any) any { return nil }
func (c *xtbXMLToI18n) VisitBlock(block *ml_parser.Block, context any) any             { return nil }
func (c *xtbXMLToI18n) VisitBlockParameter(parameter *ml_parser.BlockParameter, context any) any {
	return nil
}
func (c *xtbXMLToI18n) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context any) any {
	return nil
}
func (c *xtbXMLToI18n) VisitComponent(component *ml_parser.Component, context any) any {
	c.addError(component, "Unexpected node")
	return nil
}
func (c *xtbXMLToI18n) VisitDirective(directive *ml_parser.Directive, context any) any {
	c.addError(directive, "Unexpected node")
	return nil
}

func (c *xtbXMLToI18n) addError(node ml_parser.Node, message string) {
	c.errors = append(c.errors, parse_util.NewParseError(node.GetSourceSpan(), message, nil, nil))
}

type xtbCaseValue struct {
	value string
	nodes []Node
}
