package util

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type VisitListEntryResult struct {
	Node   *ast.Node
	Before []*ast.Node
	After  []*ast.Node
}

type Visitor interface {
	VisitClassDeclaration(node *ast.Node) VisitListEntryResult
	VisitOtherNode(node *ast.Node) *ast.Node
	Visit(node *ast.Node) *ast.Node
}

type BaseVisitor struct {
	before map[*ast.Node][]*ast.Node
	after  map[*ast.Node][]*ast.Node
}

func NewBaseVisitor() *BaseVisitor {
	return &BaseVisitor{
		before: make(map[*ast.Node][]*ast.Node),
		after:  make(map[*ast.Node][]*ast.Node),
	}
}

func (v *BaseVisitor) VisitListEntryNode(node *ast.Node, visitFn func(node *ast.Node) VisitListEntryResult) *ast.Node {
	result := visitFn(node)
	if len(result.Before) > 0 {
		v.before[result.Node] = append(v.before[result.Node], result.Before...)
	}
	if len(result.After) > 0 {
		v.after[result.Node] = append(v.after[result.Node], result.After...)
	}
	return result.Node
}

func (v *BaseVisitor) VisitOtherNode(node *ast.Node) *ast.Node {
	return node
}

func (v *BaseVisitor) MaybeProcessStatements(node *ast.Node) *ast.Node {
	// Logic to process statements if this node is a block or source file.
	// For this mock, we just return the node.
	return node
}

// GetBefore returns the "before" nodes recorded for the given node, if any.
func (v *BaseVisitor) GetBefore(node *ast.Node) []*ast.Node {
	return v.before[node]
}

// GetAfter returns the "after" nodes recorded for the given node, if any.
func (v *BaseVisitor) GetAfter(node *ast.Node) []*ast.Node {
	return v.after[node]
}
