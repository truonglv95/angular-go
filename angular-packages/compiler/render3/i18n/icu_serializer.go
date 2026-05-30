package i18n

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
)

type icuSerializerVisitor struct{}

func (v *icuSerializerVisitor) VisitText(text *i18n.Text, context any) any {
	return text.Value
}

func (v *icuSerializerVisitor) VisitContainer(container *i18n.Container, context any) any {
	var results []string
	for _, child := range container.Children {
		results = append(results, child.Visit(v, context).(string))
	}
	return strings.Join(results, "")
}

func (v *icuSerializerVisitor) VisitIcu(icu *i18n.Icu, context any) any {
	var strCases []string
	for _, k := range icu.CaseOrders {
		c := icu.Cases[k]
		strCases = append(strCases, fmt.Sprintf("%s {%v}", k, c.Visit(v, context)))
	}
	result := fmt.Sprintf("{%s, %s, %s}", icu.ExpressionPlaceholder, icu.Type, strings.Join(strCases, " "))
	return result
}

func (v *icuSerializerVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context any) any {
	if ph.IsVoid {
		return v.formatPh(ph.StartName)
	}
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, context).(string))
	}
	return fmt.Sprintf("%s%s%s", v.formatPh(ph.StartName), strings.Join(children, ""), v.formatPh(ph.CloseName))
}

func (v *icuSerializerVisitor) VisitPlaceholder(ph *i18n.Placeholder, context any) any {
	return v.formatPh(ph.Name)
}

func (v *icuSerializerVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context any) any {
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, context).(string))
	}
	return fmt.Sprintf("%s%s%s", v.formatPh(ph.StartName), strings.Join(children, ""), v.formatPh(ph.CloseName))
}

func (v *icuSerializerVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context any) any {
	return v.formatPh(ph.Name)
}

func (v *icuSerializerVisitor) formatPh(value string) string {
	return fmt.Sprintf("{%s}", FormatI18nPlaceholderName(value, false))
}

var serializer = &icuSerializerVisitor{}

func SerializeIcuNode(icu *i18n.Icu) string {
	return icu.Visit(serializer, nil).(string)
}
