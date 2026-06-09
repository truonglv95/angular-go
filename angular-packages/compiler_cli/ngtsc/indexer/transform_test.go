package indexer

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func populateContext(
	context *IndexingContext,
	component *ast.Node,
	selector string,
	template string,
	boundTemplate AbstractBoundTemplate,
	isInline bool,
) {
	sf := findSourceFileNode(component)
	context.AddComponent(&ComponentInfo{
		Declaration:   component,
		Selector:      &selector,
		BoundTemplate: boundTemplate,
		TemplateMeta: ComponentTemplateMeta{
			IsInline: isInline,
			File:     parse_util.NewParseSourceFile(template, sf.FileName()),
		},
	})
}

func TestGenerateAnalysis(t *testing.T) {
	t.Run("should emit component and template analysis information", func(t *testing.T) {
		context := NewIndexingContext()
		decl, release := getComponentDeclaration(t, "class C {}", "C")
		defer release()

		template := "<div>{{foo}}</div>"
		boundTemplate := getBoundTemplate(t, template, nil, nil)
		populateContext(context, decl, "c-selector", template, boundTemplate, false)

		analysis := GenerateAnalysis(context, nodeAdapter{})
		assert.Len(t, analysis, 1)

		info, ok := analysis[decl]
		require.True(t, ok)

		sf := findSourceFileNode(decl)
		assert.Equal(t, "C", info.Name)
		assert.Equal(t, "c-selector", *info.Selector)
		assert.Equal(t, "class C {}", info.File.Content)
		assert.Equal(t, sf.FileName(), info.File.Url)

		expectedIdentifiers, expectedErrors := getTemplateIdentifiers(getBoundTemplate(t, "<div>{{foo}}</div>", nil, nil))
		assert.Equal(t, expectedErrors, info.Errors)
		assert.Equal(t, expectedIdentifiers, info.Template.Identifiers)
		assert.Equal(t, "<div>{{foo}}</div>", info.Template.File.Content)
		assert.Equal(t, sf.FileName(), info.Template.File.Url)
	})

	t.Run("should give inline templates the component source file", func(t *testing.T) {
		context := NewIndexingContext()
		decl, release := getComponentDeclaration(t, "class C {}", "C")
		defer release()

		template := "<div>{{foo}}</div>"
		boundTemplate := getBoundTemplate(t, template, nil, nil)
		populateContext(context, decl, "c-selector", template, boundTemplate, true)

		analysis := GenerateAnalysis(context, nodeAdapter{})
		assert.Len(t, analysis, 1)

		info, ok := analysis[decl]
		require.True(t, ok)

		sf := findSourceFileNode(decl)
		assert.Equal(t, "class C {}", info.Template.File.Content)
		assert.Equal(t, sf.FileName(), info.Template.File.Url)
	})

	t.Run("should give external templates their own source file", func(t *testing.T) {
		context := NewIndexingContext()
		decl, release := getComponentDeclaration(t, "class C {}", "C")
		defer release()

		template := "<div>{{foo}}</div>"
		boundTemplate := getBoundTemplate(t, template, nil, nil)
		populateContext(context, decl, "c-selector", template, boundTemplate, false)

		analysis := GenerateAnalysis(context, nodeAdapter{})
		assert.Len(t, analysis, 1)

		info, ok := analysis[decl]
		require.True(t, ok)

		sf := findSourceFileNode(decl)
		assert.Equal(t, "<div>{{foo}}</div>", info.Template.File.Content)
		assert.Equal(t, sf.FileName(), info.Template.File.Url)
	})
}
