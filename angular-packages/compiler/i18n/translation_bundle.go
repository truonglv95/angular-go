package i18n

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type Console interface {
	Warn(msg string)
}

type TranslationBundle struct {
	i18nToHtml       *I18nToHtmlVisitor
	i18nNodesByMsgId TranslationStore
	digest           func(m *Message) string
}

func NewTranslationBundle(
	i18nNodesByMsgId TranslationStore,
	locale *string,
	digest func(m *Message) string,
	mapperFactory func(m *Message) PlaceholderMapper,
	missingTranslationStrategy core.MissingTranslationStrategy,
	console any,
) *TranslationBundle {
	if i18nNodesByMsgId == nil {
		i18nNodesByMsgId = NewEagerTranslationStore(map[string][]Node{})
	}
	return &TranslationBundle{
		i18nToHtml: NewI18nToHtmlVisitor(
			i18nNodesByMsgId,
			locale,
			digest,
			mapperFactory,
			missingTranslationStrategy,
			console,
		),
		i18nNodesByMsgId: i18nNodesByMsgId,
		digest:           digest,
	}
}

func (t *TranslationBundle) Digest(message *Message) string {
	return t.digest(message)
}

func TranslationBundleLoad(
	content string,
	url string,
	serializer Serializer,
	missingTranslationStrategy core.MissingTranslationStrategy,
	console any,
) *TranslationBundle {
	res := serializer.Load(content, url)
	digestFn := func(m *Message) string {
		return serializer.Digest(m)
	}
	mapperFactory := func(m *Message) PlaceholderMapper {
		return serializer.CreateNameMapper(m)
	}
	return NewTranslationBundle(
		res.I18nNodesByMsgId,
		res.Locale,
		digestFn,
		mapperFactory,
		missingTranslationStrategy,
		console,
	)
}

func (t *TranslationBundle) Get(srcMsg *Message) []ml_parser.Node {
	htmlRes := t.i18nToHtml.Convert(srcMsg)
	if len(htmlRes.Errors) > 0 {
		var errMsgs []string
		for _, err := range htmlRes.Errors {
			errMsgs = append(errMsgs, err.Error())
		}
		panic(strings.Join(errMsgs, "\n"))
	}
	return htmlRes.RootNodes
}

func (t *TranslationBundle) Has(srcMsg *Message) bool {
	id := t.digest(srcMsg)
	return t.i18nNodesByMsgId.Has(id)
}

type I18nToHtmlVisitor struct {
	i18nNodesByMsgId           TranslationStore
	locale                     *string
	digest                     func(m *Message) string
	mapperFactory              func(m *Message) PlaceholderMapper
	missingTranslationStrategy core.MissingTranslationStrategy
	console                    any

	srcMsg       *Message
	errors       []*parse_util.ParseError
	contextStack []i18nToHtmlContext
	mapper       func(name string) string
}

type i18nToHtmlContext struct {
	msg    *Message
	mapper func(name string) string
}

func NewI18nToHtmlVisitor(
	i18nNodesByMsgId TranslationStore,
	locale *string,
	digest func(m *Message) string,
	mapperFactory func(m *Message) PlaceholderMapper,
	missingTranslationStrategy core.MissingTranslationStrategy,
	console any,
) *I18nToHtmlVisitor {
	return &I18nToHtmlVisitor{
		i18nNodesByMsgId:           i18nNodesByMsgId,
		locale:                     locale,
		digest:                     digest,
		mapperFactory:              mapperFactory,
		missingTranslationStrategy: missingTranslationStrategy,
		console:                    console,
	}
}

func (v *I18nToHtmlVisitor) Convert(srcMsg *Message) ml_parser.ParseTreeResult {
	v.contextStack = []i18nToHtmlContext{}
	v.errors = []*parse_util.ParseError{}

	text := v.convertToText(srcMsg)

	url := ""
	if len(srcMsg.Nodes) > 0 && srcMsg.Nodes[0].GetSourceSpan() != nil {
		url = srcMsg.Nodes[0].GetSourceSpan().Start.File.Url
	}

	htmlParser := ml_parser.NewHtmlParser()
	htmlRes := htmlParser.Parse(text, url, &ml_parser.TokenizeOptions{TokenizeExpansionForms: true})

	var parseErrors []error
	for _, err := range v.errors {
		parseErrors = append(parseErrors, err)
	}
	for _, err := range htmlRes.Errors {
		parseErrors = append(parseErrors, err)
	}

	return ml_parser.ParseTreeResult{
		RootNodes: htmlRes.RootNodes,
		Errors:    parseErrors,
	}
}

func (v *I18nToHtmlVisitor) VisitText(text *Text, context any) any {
	return EscapeXml(text.Value)
}

func (v *I18nToHtmlVisitor) VisitContainer(container *Container, context any) any {
	var parts []string
	for _, child := range container.Children {
		parts = append(parts, child.Visit(v, context).(string))
	}
	return strings.Join(parts, "")
}

func (v *I18nToHtmlVisitor) VisitIcu(icu *Icu, context any) any {
	var cases []string
	for _, k := range icu.CaseOrders {
		cases = append(cases, fmt.Sprintf("%s {%s}", k, icu.Cases[k].Visit(v, context).(string)))
	}

	exp := icu.Expression
	if ph, ok := v.srcMsg.Placeholders[icu.Expression]; ok {
		exp = ph.Text
	}

	return fmt.Sprintf("{%s, %s, %s}", exp, icu.Type, strings.Join(cases, " "))
}

func (v *I18nToHtmlVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	phName := v.mapper(ph.Name)
	if phContent, ok := v.srcMsg.Placeholders[phName]; ok {
		return phContent.Text
	}

	if refMsg, ok := v.srcMsg.PlaceholderToMessage[phName]; ok {
		return v.convertToText(refMsg)
	}

	v.addError(ph, fmt.Sprintf("Unknown placeholder %q", ph.Name))
	return ""
}

func (v *I18nToHtmlVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	tag := ph.Tag
	var attrs []string
	var keys []string
	for k := range ph.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		attrs = append(attrs, fmt.Sprintf("%s=%q", k, ph.Attrs[k]))
	}
	attrsStr := ""
	if len(attrs) > 0 {
		attrsStr = " " + strings.Join(attrs, " ")
	}

	if ph.IsVoid {
		return fmt.Sprintf("<%s%s/>", tag, attrsStr)
	}

	var children []string
	for _, c := range ph.Children {
		children = append(children, c.Visit(v, context).(string))
	}
	return fmt.Sprintf("<%s%s>%s</%s>", tag, attrsStr, strings.Join(children, ""), tag)
}

func (v *I18nToHtmlVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	return v.convertToText(v.srcMsg.PlaceholderToMessage[ph.Name])
}

func (v *I18nToHtmlVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	params := ""
	if len(ph.Parameters) > 0 {
		params = fmt.Sprintf(" (%s)", strings.Join(ph.Parameters, "; "))
	}
	var children []string
	for _, c := range ph.Children {
		children = append(children, c.Visit(v, context).(string))
	}
	return fmt.Sprintf("@%s%s {%s}", ph.Name, params, strings.Join(children, ""))
}

func (v *I18nToHtmlVisitor) convertToText(srcMsg *Message) string {
	id := v.digest(srcMsg)
	var mapper PlaceholderMapper
	if v.mapperFactory != nil {
		mapper = v.mapperFactory(srcMsg)
	}
	var nodes []Node

	v.contextStack = append(v.contextStack, i18nToHtmlContext{msg: v.srcMsg, mapper: v.mapper})
	v.srcMsg = srcMsg

	if translatedNodes, ok := v.i18nNodesByMsgId.Get(id); ok {
		nodes = translatedNodes
		v.mapper = func(name string) string {
			if mapper != nil {
				if internalNamePtr := mapper.ToInternalName(name); internalNamePtr != nil {
					return *internalNamePtr
				}
			}
			return name
		}
	} else {
		if v.missingTranslationStrategy == core.MissingTranslationStrategyError {
			ctx := ""
			if v.locale != nil {
				ctx = fmt.Sprintf(" for locale %q", *v.locale)
			}
			node := srcMsg.Nodes[0]
			v.addError(node, fmt.Sprintf("Missing translation for message %q%s", id, ctx))
		} else if v.console != nil && v.missingTranslationStrategy == core.MissingTranslationStrategyWarning {
			ctx := ""
			if v.locale != nil {
				ctx = fmt.Sprintf(" for locale %q", *v.locale)
			}
			msgStr := fmt.Sprintf("Missing translation for message %q%s", id, ctx)
			if c, ok := v.console.(Console); ok {
				c.Warn(msgStr)
			} else if c, ok := v.console.(interface{ Warn(string) }); ok {
				c.Warn(msgStr)
			}
		}
		nodes = srcMsg.Nodes
		v.mapper = func(name string) string {
			return name
		}
	}

	var parts []string
	for _, node := range nodes {
		parts = append(parts, node.Visit(v, nil).(string))
	}
	text := strings.Join(parts, "")

	context := v.contextStack[len(v.contextStack)-1]
	v.contextStack = v.contextStack[:len(v.contextStack)-1]
	v.srcMsg = context.msg
	v.mapper = context.mapper

	return text
}

func (v *I18nToHtmlVisitor) addError(node Node, msg string) {
	v.errors = append(v.errors, parse_util.NewParseError(node.GetSourceSpan(), msg, nil, nil))
}

var attrEscapeRegexp = regexp.MustCompile(`[<&"]`)

func EscapeXml(val string) string {
	val = strings.ReplaceAll(val, "&", "&amp;")
	val = strings.ReplaceAll(val, "<", "&lt;")
	val = strings.ReplaceAll(val, ">", "&gt;")
	val = strings.ReplaceAll(val, "\"", "&quot;")
	val = strings.ReplaceAll(val, "'", "&apos;")
	return val
}
