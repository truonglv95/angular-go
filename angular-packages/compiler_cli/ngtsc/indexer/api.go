package indexer

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/internal/ast"
)

type IdentifierKind int

const (
	IdentifierKindProperty IdentifierKind = iota
	IdentifierKindMethod
	IdentifierKindElement
	IdentifierKindTemplate
	IdentifierKindAttribute
	IdentifierKindReference
	IdentifierKindVariable
	IdentifierKindLetDeclaration
	IdentifierKindComponent
	IdentifierKindDirective
)

type AbsoluteSourceSpan struct {
	Start int
	End   int
}

func NewAbsoluteSourceSpan(start, end int) AbsoluteSourceSpan {
	return AbsoluteSourceSpan{Start: start, End: end}
}

type DirectiveReference struct {
	Node     *ast.Node // DeclarationNode
	Selector string
}

type ReferenceTarget struct {
	Node      *TemplateIdentifier
	Directive *ast.Node
}

type BoundReferenceTarget interface {
	GetNode() any
	GetDirectiveNode() *ast.Node
}

type TemplateIdentifier struct {
	Name           string
	Span           AbsoluteSourceSpan
	Kind           IdentifierKind
	Target         *TemplateIdentifier
	Attributes     []TemplateIdentifier
	UsedDirectives []DirectiveReference
	RefTarget      *ReferenceTarget
}

type IndexedTemplate struct {
	Identifiers []TemplateIdentifier
	File        *parse_util.ParseSourceFile
}

type IndexedComponent struct {
	Name     string
	Selector *string
	File     *parse_util.ParseSourceFile
	Template IndexedTemplate
	Errors   []error
}

// AbstractBoundTemplate represents a bound template (used for querying template nodes).
type AbstractBoundTemplate interface {
	GetDirectivesOfNode(node any) []DirectiveReference
	GetReferenceTarget(node any) any // returns *Element, *Template, *Component, *Directive, or ReferenceTarget
	GetExpressionTarget(astNode any) any // returns *Reference, *Variable, *LetDeclaration
	GetUsedDirectives() []DirectiveReference
	GetTemplateAst() []any
}

// NodeAdapter provides helper methods to get class declaration info.
type NodeAdapter interface {
	GetName(node *ast.Node) string
	GetFileName(node *ast.Node) string
	GetContent(node *ast.Node) string
}
