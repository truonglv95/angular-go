package ml_parser

import "github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"

// Visitor interface for traversing the ML AST.
type Visitor interface {
	VisitElement(element *Element, context any) any
	VisitAttribute(attribute *Attribute, context any) any
	VisitText(text *Text, context any) any
	VisitComment(comment *Comment, context any) any
	VisitExpansion(expansion *Expansion, context any) any
	VisitExpansionCase(expansionCase *ExpansionCase, context any) any
	VisitBlock(block *Block, context any) any
	VisitBlockParameter(blockParameter *BlockParameter, context any) any
}

type GenericVisitor interface {
	Visit(node Node, context any) any
}

type DirectiveVisitor interface {
	VisitDirective(directive *Directive, context any) any
}

type LetDeclarationVisitor interface {
	VisitLetDeclaration(decl *LetDeclaration, context any) any
}

// Node is the base interface for all ML AST nodes.
type Node interface {
	Visit(visitor Visitor, context any) any
	GetSourceSpan() *parse_util.ParseSourceSpan
}

// I18nMeta represents i18n metadata
type I18nMeta = any

// BaseNode implementation struct that can be embedded.
type BaseNode struct {
	SourceSpan *parse_util.ParseSourceSpan
	I18n       I18nMeta
}

func (b *BaseNode) GetSourceSpan() *parse_util.ParseSourceSpan {
	return b.SourceSpan
}

type Text struct {
	BaseNode
	Value  string
	Tokens []Token
}

func (t *Text) Visit(visitor Visitor, context any) any {
	return visitor.VisitText(t, context)
}

type Expansion struct {
	BaseNode
	SwitchValue string
	Type        string
	Cases       []*ExpansionCase
}

func (e *Expansion) Visit(visitor Visitor, context any) any {
	return visitor.VisitExpansion(e, context)
}

type ExpansionCase struct {
	BaseNode
	Value      string
	Expression []Node
}

func (e *ExpansionCase) Visit(visitor Visitor, context any) any {
	return visitor.VisitExpansionCase(e, context)
}

type Attribute struct {
	BaseNode
	Name        string
	Value       string
	KeySpan     *parse_util.ParseSourceSpan
	ValueSpan   *parse_util.ParseSourceSpan
	ValueTokens []Token
}

func (a *Attribute) Visit(visitor Visitor, context any) any {
	return visitor.VisitAttribute(a, context)
}

type Element struct {
	BaseNode
	Name            string
	Attrs           []*Attribute
	Directives      []*Directive
	Children        []Node
	EndSourceSpan   *parse_util.ParseSourceSpan
	StartSourceSpan *parse_util.ParseSourceSpan
	IsVoid          bool
	IsSelfClosing   bool
}

func (e *Element) GetDirectives() []*Directive {
	return e.Directives
}

func (e *Element) Visit(visitor Visitor, context any) any {
	return visitor.VisitElement(e, context)
}

type Comment struct {
	BaseNode
	Value string
}

func (c *Comment) Visit(visitor Visitor, context any) any {
	return visitor.VisitComment(c, context)
}

type Block struct {
	BaseNode
	Name            string
	Parameters      []*BlockParameter
	Children        []Node
	EndSourceSpan   *parse_util.ParseSourceSpan
	StartSourceSpan *parse_util.ParseSourceSpan
}

func (b *Block) Visit(visitor Visitor, context any) any {
	return visitor.VisitBlock(b, context)
}

type BlockParameter struct {
	BaseNode
	Expression string
}

func (bp *BlockParameter) Visit(visitor Visitor, context any) any {
	return visitor.VisitBlockParameter(bp, context)
}

type Directive struct {
	Name            string
	Attrs           []*Attribute
	SourceSpan      parse_util.ParseSourceSpan
	StartSourceSpan *parse_util.ParseSourceSpan
	EndSourceSpan   *parse_util.ParseSourceSpan
}

func (d *Directive) Visit(visitor Visitor, context any) any {
	if dv, ok := visitor.(DirectiveVisitor); ok {
		return dv.VisitDirective(d, context)
	}
	return nil
}

func (d *Directive) GetSourceSpan() *parse_util.ParseSourceSpan {
	return &d.SourceSpan
}

type LetDeclaration struct {
	Name       string
	Value      string
	SourceSpan parse_util.ParseSourceSpan
	NameSpan   *parse_util.ParseSourceSpan
	ValueSpan  *parse_util.ParseSourceSpan
}

func (l *LetDeclaration) Visit(visitor Visitor, context any) any {
	if lv, ok := visitor.(LetDeclarationVisitor); ok {
		return lv.VisitLetDeclaration(l, context)
	}
	return nil
}

func (l *LetDeclaration) GetSourceSpan() *parse_util.ParseSourceSpan {
	return &l.SourceSpan
}

func (l *LetDeclaration) GetName() string {
	return l.Name
}

type Component struct {
	Element
	ComponentName string
	TagName       string
	FullName      string
}

func (e *Element) getChildren() []Node                        { return e.Children }
func (e *Element) setChildren(c []Node)                       { e.Children = c }
func (e *Element) getName() string                            { return e.Name }
func (e *Element) getSourceSpan() *parse_util.ParseSourceSpan { return e.SourceSpan }
func (e *Element) setEndSourceSpan(span *parse_util.ParseSourceSpan) {
	e.EndSourceSpan = span
	if span != nil {
		e.SourceSpan.End = span.End
	}
}
func (e *Element) isBlock() bool     { return false }
func (e *Element) isElement() bool   { return true }
func (e *Element) isComponent() bool { return false }

func (b *Block) getChildren() []Node                        { return b.Children }
func (b *Block) setChildren(c []Node)                       { b.Children = c }
func (b *Block) getName() string                            { return b.Name }
func (b *Block) getSourceSpan() *parse_util.ParseSourceSpan { return b.SourceSpan }
func (b *Block) setEndSourceSpan(span *parse_util.ParseSourceSpan) {
	b.EndSourceSpan = span
	if span != nil {
		b.SourceSpan.End = span.End
	}
}
func (b *Block) isBlock() bool     { return true }
func (b *Block) isElement() bool   { return false }
func (b *Block) isComponent() bool { return false }

func (c *Component) isComponent() bool { return true }
func (c *Component) getName() string   { return c.FullName }

type ComponentVisitor interface {
	VisitComponent(component *Component, context any) any
}

func (c *Component) Visit(visitor Visitor, context any) any {
	if cv, ok := visitor.(ComponentVisitor); ok {
		return cv.VisitComponent(c, context)
	}
	return visitor.VisitElement(&c.Element, context)
}

func VisitAll(visitor Visitor, nodes []Node, context any) []any {
	result := make([]any, 0, len(nodes))
	for _, node := range nodes {
		var nodeResult any
		if genericVisitor, ok := visitor.(GenericVisitor); ok {
			nodeResult = genericVisitor.Visit(node, context)
		}
		if nodeResult == nil {
			nodeResult = node.Visit(visitor, context)
		}
		if nodeResult != nil {
			result = append(result, nodeResult)
		}
	}
	return result
}
