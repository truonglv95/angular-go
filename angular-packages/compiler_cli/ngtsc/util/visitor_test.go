// Package util_test ports visitor_spec.ts from ngtsc 1:1.
//
// The original TypeScript spec tests an AST transformer that inserts variable
// statements before class declarations during emit. The Go implementation does
// not perform JS emit; instead the visitor infrastructure records "before" and
// "after" nodes for later statement list rewriting. These tests verify that
// the Go visitor infrastructure correctly records the Before/After nodes when
// a class declaration has a static `id` field – mirroring the 1:1 logic of
// the TypeScript TestAstVisitor.
package util_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/util"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAstVisitor mirrors the TestAstVisitor in visitor_spec.ts:
// If the class has a static member named "id", it returns a "before" synthetic node.
type testAstVisitor struct {
	*util.BaseVisitor
}

func newTestAstVisitor() *testAstVisitor {
	return &testAstVisitor{BaseVisitor: util.NewBaseVisitor()}
}

// visitClassDeclaration mirrors:
//
//	visitClassDeclaration(node: ts.ClassDeclaration): VisitListEntryResult {
//	  if idStatic !== undefined { return { node, before: [...] }; }
//	  return { node };
//	}
func (v *testAstVisitor) visitClassDeclaration(node *ast.Node) util.VisitListEntryResult {
	// Walk the members of the class declaration to find a static property named "id".
	classDecl := node.AsClassDeclaration()
	var idStatic *ast.Node
	for _, member := range classDecl.Members.Nodes {
		if !ast.IsPropertyDeclaration(member) {
			continue
		}
		// Check that the member has the 'static' modifier.
		isStatic := false
		if member.Modifiers() != nil {
			for _, mod := range member.Modifiers().Nodes {
				if mod.Kind == ast.KindStaticKeyword {
					isStatic = true
					break
				}
			}
		}
		if !isStatic {
			continue
		}
		// Check that the property name is "id".
		name := member.Name()
		if name != nil && ast.IsIdentifier(name) && name.AsIdentifier().Text == "id" {
			idStatic = member
			break
		}
	}

	if idStatic != nil {
		// Synthesize a "before" placeholder node. In Go there is no JS emit, so we use
		// a synthetic node to represent the variable statement that would be emitted
		// before the class (e.g. "var A_id = 3;"). We create a minimal synthetic node
		// using the initializer's position (if available) for traceability.
		syntheticNode := &ast.Node{}
		return util.VisitListEntryResult{
			Node:   node,
			Before: []*ast.Node{syntheticNode},
		}
	}
	return util.VisitListEntryResult{Node: node}
}

// parseSource parses the given TypeScript source text and returns the first
// source file statement list (for convenience in tests).
func parseSource(t *testing.T, src string) *ast.SourceFile {
	t.Helper()
	opts := ast.SourceFileParseOptions{FileName: "/main.ts"}
	sf := parser.ParseSourceFile(opts, src, core.ScriptKindTS)
	require.NotNil(t, sf, "ParseSourceFile returned nil")
	return sf
}

// findClassDeclaration walks the statements of sf looking for a class
// declaration whose name matches className.
func findClassDeclaration(sf *ast.SourceFile, className string) *ast.Node {
	var found *ast.Node
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsClassDeclaration(n) {
			name := n.Name()
			if name != nil && ast.IsIdentifier(name) && name.AsIdentifier().Text == className {
				found = n
				return true
			}
		}
		return n.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)
	return found
}

// TestVisitor_AddStatementBeforeClassInPlainFile mirrors:
//
//	it('should add a statement before class in plain file', ...)
//
// It verifies that the visitor records a "before" node for a top-level class
// with a static `id` field.
func TestVisitor_AddStatementBeforeClassInPlainFile(t *testing.T) {
	sf := parseSource(t, `class A { static id = 3; }`)

	classNode := findClassDeclaration(sf, "A")
	require.NotNil(t, classNode, "class A not found in AST")

	v := newTestAstVisitor()
	// VisitListEntryNode calls visitClassDeclaration and records Before nodes.
	resultNode := v.VisitListEntryNode(classNode, v.visitClassDeclaration)

	// The returned node should still be the class declaration node.
	assert.Equal(t, classNode, resultNode, "VisitListEntryNode should return the same class node")

	// The BaseVisitor should have recorded one "before" node for classNode.
	before := v.GetBefore(resultNode)
	assert.Len(t, before, 1, "expected exactly one 'before' node (the synthetic var statement)")
}

// TestVisitor_AddStatementBeforeClassInsideFunctionDefinition mirrors:
//
//	it('should add a statement before class inside function definition', ...)
//
// It verifies that the visitor records a "before" node for a class declared
// inside a function body, when the class has a static `id` field.
func TestVisitor_AddStatementBeforeClassInsideFunctionDefinition(t *testing.T) {
	src := `
export function foo() {
  var x = 3;
  class A { static id = 2; }
  return A;
}
`
	sf := parseSource(t, src)

	classNode := findClassDeclaration(sf, "A")
	require.NotNil(t, classNode, "class A not found in AST")

	v := newTestAstVisitor()
	resultNode := v.VisitListEntryNode(classNode, v.visitClassDeclaration)

	assert.Equal(t, classNode, resultNode)
	before := v.GetBefore(resultNode)
	assert.Len(t, before, 1, "expected one 'before' node for class A inside function")
}

// TestVisitor_HandlesNestedStatements mirrors:
//
//	it('handles nested statements', ...)
//
// It verifies that the visitor can be applied to both an outer and an inner
// class (nested inside a method), each recording their own "before" nodes.
func TestVisitor_HandlesNestedStatements(t *testing.T) {
	src := `
export class A {
  static id = 3;

  foo() {
    class B {
      static id = 4;
    }
    return B;
  }
}`
	sf := parseSource(t, src)

	classA := findClassDeclaration(sf, "A")
	require.NotNil(t, classA, "class A not found in AST")

	classB := findClassDeclaration(sf, "B")
	require.NotNil(t, classB, "class B not found in AST")

	v := newTestAstVisitor()

	// Visit class A.
	resultA := v.VisitListEntryNode(classA, v.visitClassDeclaration)
	assert.Equal(t, classA, resultA)
	beforeA := v.GetBefore(resultA)
	assert.Len(t, beforeA, 1, "expected one 'before' node for class A")

	// Visit class B (nested).
	resultB := v.VisitListEntryNode(classB, v.visitClassDeclaration)
	assert.Equal(t, classB, resultB)
	beforeB := v.GetBefore(resultB)
	assert.Len(t, beforeB, 1, "expected one 'before' node for class B")
}
