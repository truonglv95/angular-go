package render3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecursiveVisitor(t *testing.T) {
	t.Run("should not mutate IfBlockBranch children when visiting", func(t *testing.T) {
		template := "@if (val; as alias) { <div></div> }"
		result := parseR3(template, ParseR3Options{})

		// Structure:
		// result.Nodes[0] is the IfBlock
		// IfBlock.Branches[0] is the IfBlockBranch
		assert.Len(t, result.Nodes, 1)
		ifBlock, ok := result.Nodes[0].(*IfBlock)
		assert.True(t, ok, "expected IfBlock")

		assert.Len(t, ifBlock.Branches, 1)
		branch := ifBlock.Branches[0]
		assert.NotNil(t, branch, "expected IfBlockBranch")

		// Verify initial state
		// Children should contain only the Element (<div></div>)
		assert.Len(t, branch.Children, 1)
		element, ok := branch.Children[0].(*Element)
		assert.True(t, ok, "expected Element")
		assert.Equal(t, "div", element.Name)

		// Alias should exist
		assert.NotNil(t, branch.ExpressionAlias)
		alias := branch.ExpressionAlias

		visitor := &RecursiveVisitor{}

		// Visit the branch
		visitor.VisitIfBlockBranch(branch)

		// Verify post-visit state
		// The children array should STILL only contain the Element.
		// It should NOT contain the alias variable.
		assert.Len(t, branch.Children, 1)
		assert.Equal(t, element, branch.Children[0])

		foundAlias := false
		for _, child := range branch.Children {
			if child == alias {
				foundAlias = true
				break
			}
		}
		assert.False(t, foundAlias, "children should not contain alias")
	})

	t.Run("should invoke overridden VisitDeferredBlock for nested blocks", func(t *testing.T) {
		template := "<div>@defer { <p>hello</p> }</div>"
		result := parseR3(template, ParseR3Options{})

		blockVisitor := &mockBlockVisitor{}
		blockVisitor.Impl = blockVisitor // Explicitly set delegate if supported
		VisitAll(blockVisitor, result.Nodes)

		assert.True(t, blockVisitor.hasBlocks, "expected hasBlocks to be true for nested block")
	})
}

type mockBlockVisitor struct {
	RecursiveVisitor
	hasBlocks bool
}

func (v *mockBlockVisitor) VisitDeferredBlock(deferred *DeferredBlock) interface{} {
	v.hasBlocks = true
	return nil
}

