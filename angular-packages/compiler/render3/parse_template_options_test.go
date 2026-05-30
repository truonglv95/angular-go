package render3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectCommentNodes(t *testing.T) {
	t.Run("should include an array of HTML comment nodes on the returned R3 AST", func(t *testing.T) {
		html := `
      <!-- eslint-disable-next-line -->
      <div *ngFor="let item of items">
        {{item.name}}
      </div>

      <div>
        <p>
          <!-- some nested comment -->
          <span>Text</span>
        </p>
      </div>
    `

		templateNoCommentsOption := parseR3(html, ParseR3Options{CollectCommentNodes: false})
		assert.Nil(t, templateNoCommentsOption.CommentNodes)

		templateCommentsOptionEnabled := parseR3(html, ParseR3Options{CollectCommentNodes: true})
		assert.Len(t, templateCommentsOptionEnabled.CommentNodes, 2)

		comment1 := templateCommentsOptionEnabled.CommentNodes[0]
		assert.Equal(t, "eslint-disable-next-line", comment1.Value)
		assert.Equal(t, "<!-- eslint-disable-next-line -->", comment1.SourceSpan.ToString())

		comment2 := templateCommentsOptionEnabled.CommentNodes[1]
		assert.Equal(t, "some nested comment", comment2.Value)
		assert.Equal(t, "<!-- some nested comment -->", comment2.SourceSpan.ToString())
	})
}
