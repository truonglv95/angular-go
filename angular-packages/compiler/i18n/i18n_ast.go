package i18n

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type MessagePlaceholder struct {
	Text       string
	SourceSpan *parse_util.ParseSourceSpan
}

type Message struct {
	Sources              []*MessageSpan
	Id                   string
	LegacyIds            []string
	MessageString        string
	Nodes                []Node
	Placeholders         map[string]*MessagePlaceholder
	PlaceholderToMessage map[string]*Message
	Meaning              string
	Description          string
	CustomId             string
}

func NewMessage(nodes []Node, placeholders map[string]*MessagePlaceholder, placeholderToMessage map[string]*Message, meaning string, description string, customId string) *Message {
	id := customId
	messageString := serializeMessage(nodes)
	var sources []*MessageSpan
	if len(nodes) > 0 {
		sources = []*MessageSpan{
			{
				FilePath:  nodes[0].GetSourceSpan().Start.File.Url,
				StartLine: nodes[0].GetSourceSpan().Start.Line + 1,
				StartCol:  nodes[0].GetSourceSpan().Start.Col + 1,
				EndLine:   nodes[len(nodes)-1].GetSourceSpan().End.Line + 1,
				EndCol:    nodes[0].GetSourceSpan().Start.Col + 1,
			},
		}
	} else {
		sources = []*MessageSpan{}
	}

	return &Message{
		Sources:              sources,
		Id:                   id,
		LegacyIds:            []string{},
		MessageString:        messageString,
		Nodes:                nodes,
		Placeholders:         placeholders,
		PlaceholderToMessage: placeholderToMessage,
		Meaning:              meaning,
		Description:          description,
		CustomId:             customId,
	}
}

type MessageSpan struct {
	FilePath  string
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

type Node interface {
	GetSourceSpan() *parse_util.ParseSourceSpan
	Visit(visitor Visitor, context any) any
}

type Text struct {
	Value      string
	SourceSpan *parse_util.ParseSourceSpan
}

func NewText(value string, sourceSpan *parse_util.ParseSourceSpan) *Text {
	return &Text{Value: value, SourceSpan: sourceSpan}
}
func (t *Text) GetSourceSpan() *parse_util.ParseSourceSpan { return t.SourceSpan }
func (t *Text) Visit(visitor Visitor, context any) any {
	return visitor.VisitText(t, context)
}

type Container struct {
	Children   []Node
	SourceSpan *parse_util.ParseSourceSpan
}

func NewContainer(children []Node, sourceSpan *parse_util.ParseSourceSpan) *Container {
	return &Container{Children: children, SourceSpan: sourceSpan}
}
func (c *Container) GetSourceSpan() *parse_util.ParseSourceSpan { return c.SourceSpan }
func (c *Container) Visit(visitor Visitor, context any) any {
	return visitor.VisitContainer(c, context)
}

type Icu struct {
	Expression            string
	Type                  string
	Cases                 map[string]Node
	CaseOrders            []string
	SourceSpan            *parse_util.ParseSourceSpan
	ExpressionPlaceholder string
}

func NewIcu(expression string, typ string, cases map[string]Node, caseOrders []string, sourceSpan *parse_util.ParseSourceSpan, expressionPlaceholder string) *Icu {
	return &Icu{Expression: expression, Type: typ, Cases: cases, CaseOrders: caseOrders, SourceSpan: sourceSpan, ExpressionPlaceholder: expressionPlaceholder}
}
func (i *Icu) GetSourceSpan() *parse_util.ParseSourceSpan { return i.SourceSpan }
func (i *Icu) Visit(visitor Visitor, context any) any {
	return visitor.VisitIcu(i, context)
}

type TagPlaceholder struct {
	Tag             string
	Attrs           map[string]string
	StartName       string
	CloseName       string
	Children        []Node
	IsVoid          bool
	SourceSpan      *parse_util.ParseSourceSpan
	StartSourceSpan *parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
}

func NewTagPlaceholder(tag string, attrs map[string]string, startName string, closeName string, children []Node, isVoid bool, sourceSpan *parse_util.ParseSourceSpan, startSourceSpan *parse_util.ParseSourceSpan, endSourceSpan *parse_util.ParseSourceSpan) *TagPlaceholder {
	return &TagPlaceholder{Tag: tag, Attrs: attrs, StartName: startName, CloseName: closeName, Children: children, IsVoid: isVoid, SourceSpan: sourceSpan, StartSourceSpan: startSourceSpan, EndSourceSpan: endSourceSpan}
}
func (p *TagPlaceholder) GetSourceSpan() *parse_util.ParseSourceSpan { return p.SourceSpan }
func (p *TagPlaceholder) Visit(visitor Visitor, context any) any {
	return visitor.VisitTagPlaceholder(p, context)
}

type Placeholder struct {
	Value      string
	Name       string
	SourceSpan *parse_util.ParseSourceSpan
}

func NewPlaceholder(value string, name string, sourceSpan *parse_util.ParseSourceSpan) *Placeholder {
	return &Placeholder{Value: value, Name: name, SourceSpan: sourceSpan}
}
func (p *Placeholder) GetSourceSpan() *parse_util.ParseSourceSpan { return p.SourceSpan }
func (p *Placeholder) Visit(visitor Visitor, context any) any {
	return visitor.VisitPlaceholder(p, context)
}

type IcuPlaceholder struct {
	PreviousMessage *Message
	Value           *Icu
	Name            string
	SourceSpan      *parse_util.ParseSourceSpan
}

func NewIcuPlaceholder(value *Icu, name string, sourceSpan *parse_util.ParseSourceSpan) *IcuPlaceholder {
	return &IcuPlaceholder{Value: value, Name: name, SourceSpan: sourceSpan}
}
func (p *IcuPlaceholder) GetSourceSpan() *parse_util.ParseSourceSpan { return p.SourceSpan }
func (p *IcuPlaceholder) Visit(visitor Visitor, context any) any {
	return visitor.VisitIcuPlaceholder(p, context)
}

type BlockPlaceholder struct {
	Name            string
	Parameters      []string
	StartName       string
	CloseName       string
	Children        []Node
	SourceSpan      *parse_util.ParseSourceSpan
	StartSourceSpan *parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
}

func NewBlockPlaceholder(name string, parameters []string, startName string, closeName string, children []Node, sourceSpan *parse_util.ParseSourceSpan, startSourceSpan *parse_util.ParseSourceSpan, endSourceSpan *parse_util.ParseSourceSpan) *BlockPlaceholder {
	return &BlockPlaceholder{Name: name, Parameters: parameters, StartName: startName, CloseName: closeName, Children: children, SourceSpan: sourceSpan, StartSourceSpan: startSourceSpan, EndSourceSpan: endSourceSpan}
}
func (p *BlockPlaceholder) GetSourceSpan() *parse_util.ParseSourceSpan { return p.SourceSpan }
func (p *BlockPlaceholder) Visit(visitor Visitor, context any) any {
	return visitor.VisitBlockPlaceholder(p, context)
}

type I18nMeta interface{}

type Visitor interface {
	VisitText(text *Text, context any) any
	VisitContainer(container *Container, context any) any
	VisitIcu(icu *Icu, context any) any
	VisitTagPlaceholder(ph *TagPlaceholder, context any) any
	VisitPlaceholder(ph *Placeholder, context any) any
	VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any
	VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any
}

type CloneVisitor struct{}

func (v *CloneVisitor) VisitText(text *Text, context any) any {
	return NewText(text.Value, text.SourceSpan)
}

func (v *CloneVisitor) VisitContainer(container *Container, context any) any {
	children := make([]Node, len(container.Children))
	for i, n := range container.Children {
		children[i] = n.Visit(v, context).(Node)
	}
	return NewContainer(children, container.SourceSpan)
}

func (v *CloneVisitor) VisitIcu(icu *Icu, context any) any {
	cases := make(map[string]Node)
	for key, val := range icu.Cases {
		cases[key] = val.Visit(v, context).(Node)
	}
	orders := make([]string, len(icu.CaseOrders))
	copy(orders, icu.CaseOrders)
	return NewIcu(icu.Expression, icu.Type, cases, orders, icu.SourceSpan, icu.ExpressionPlaceholder)
}

func (v *CloneVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	children := make([]Node, len(ph.Children))
	for i, n := range ph.Children {
		children[i] = n.Visit(v, context).(Node)
	}
	return NewTagPlaceholder(
		ph.Tag,
		ph.Attrs,
		ph.StartName,
		ph.CloseName,
		children,
		ph.IsVoid,
		ph.SourceSpan,
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

func (v *CloneVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	return NewPlaceholder(ph.Value, ph.Name, ph.SourceSpan)
}

func (v *CloneVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	return NewIcuPlaceholder(ph.Value, ph.Name, ph.SourceSpan)
}

func (v *CloneVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	children := make([]Node, len(ph.Children))
	for i, n := range ph.Children {
		children[i] = n.Visit(v, context).(Node)
	}
	return NewBlockPlaceholder(
		ph.Name,
		ph.Parameters,
		ph.StartName,
		ph.CloseName,
		children,
		ph.SourceSpan,
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

type RecurseVisitor struct{}

func (v *RecurseVisitor) VisitText(text *Text, context any) any { return nil }

func (v *RecurseVisitor) VisitContainer(container *Container, context any) any {
	for _, child := range container.Children {
		child.Visit(v, nil)
	}
	return nil
}

func (v *RecurseVisitor) VisitIcu(icu *Icu, context any) any {
	for _, k := range icu.CaseOrders {
		icu.Cases[k].Visit(v, nil)
	}
	return nil
}

func (v *RecurseVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	return nil
}

func (v *RecurseVisitor) VisitPlaceholder(ph *Placeholder, context any) any { return nil }

func (v *RecurseVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any { return nil }

func (v *RecurseVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	return nil
}

func serializeMessage(messageNodes []Node) string {
	visitor := &LocalizeMessageStringVisitor{}
	var strBuilder strings.Builder
	for _, n := range messageNodes {
		strBuilder.WriteString(n.Visit(visitor, nil).(string))
	}
	return strBuilder.String()
}

type LocalizeMessageStringVisitor struct{}

func (v *LocalizeMessageStringVisitor) VisitText(text *Text, context any) any {
	return text.Value
}

func (v *LocalizeMessageStringVisitor) VisitContainer(container *Container, context any) any {
	var strBuilder strings.Builder
	for _, child := range container.Children {
		strBuilder.WriteString(child.Visit(v, nil).(string))
	}
	return strBuilder.String()
}

func (v *LocalizeMessageStringVisitor) VisitIcu(icu *Icu, context any) any {
	var strCases []string
	for _, k := range icu.CaseOrders {
		strCases = append(strCases, fmt.Sprintf("%s {%s}", k, icu.Cases[k].Visit(v, nil).(string)))
	}
	return fmt.Sprintf("{%s, %s, %s}", icu.ExpressionPlaceholder, icu.Type, strings.Join(strCases, " "))
}

func (v *LocalizeMessageStringVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	var strBuilder strings.Builder
	for _, child := range ph.Children {
		strBuilder.WriteString(child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("{$%s}%s{$%s}", ph.StartName, strBuilder.String(), ph.CloseName)
}

func (v *LocalizeMessageStringVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	return fmt.Sprintf("{$%s}", ph.Name)
}

func (v *LocalizeMessageStringVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	return fmt.Sprintf("{$%s}", ph.Name)
}

func (v *LocalizeMessageStringVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	var strBuilder strings.Builder
	for _, child := range ph.Children {
		strBuilder.WriteString(child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("{$%s}%s{$%s}", ph.StartName, strBuilder.String(), ph.CloseName)
}
