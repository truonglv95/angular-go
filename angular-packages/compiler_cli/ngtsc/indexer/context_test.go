package indexer

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/stretchr/testify/assert"
)

func TestComponentAnalysisContext(t *testing.T) {
	t.Run("should store and return information about components", func(t *testing.T) {
		declaration, release := getComponentDeclaration(t, "class C {};", "C")
		defer release()

		boundTemplate := getBoundTemplate(t, "<div></div>", nil, nil)
		context := NewIndexingContext()

		sf := findSourceFileNode(declaration)
		fileName := sf.FileName()

		info := &ComponentInfo{
			Declaration:   declaration,
			Selector:      strPtr("c-selector"),
			BoundTemplate: boundTemplate,
			TemplateMeta: ComponentTemplateMeta{
				IsInline: false,
				File:     parse_util.NewParseSourceFile("<div></div>", fileName),
			},
		}

		context.AddComponent(info)

		assert.Len(t, context.Components, 1)
		assert.Equal(t, declaration, context.Components[0].Declaration)
		assert.Equal(t, "c-selector", *context.Components[0].Selector)
		assert.Equal(t, boundTemplate, context.Components[0].BoundTemplate)
		assert.Equal(t, false, context.Components[0].TemplateMeta.IsInline)
		assert.Equal(t, "<div></div>", context.Components[0].TemplateMeta.File.Content)
		assert.Equal(t, fileName, context.Components[0].TemplateMeta.File.Url)
	})
}

func strPtr(s string) *string {
	return &s
}
