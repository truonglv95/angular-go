package indexer

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bind(template string, enableSelectorless ...bool) AbstractBoundTemplate {
	opts := &render3.ParseTemplateOptions{
		PreserveWhitespaces: boolPtr(true),
	}
	if len(enableSelectorless) > 0 {
		opts.EnableSelectorless = &enableSelectorless[0]
	}
	return getBoundTemplate(nil, template, opts, nil)
}

func findIdent(idents []TemplateIdentifier, name string, kind IdentifierKind, start, end int) *TemplateIdentifier {
	for _, id := range idents {
		if id.Name == name && id.Kind == kind && id.Span.Start == start && id.Span.End == end {
			return &id
		}
	}
	return nil
}

func TestGetTemplateIdentifiers(t *testing.T) {
	t.Run("should handle svg elements", func(t *testing.T) {
		template := "<svg></svg>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "svg", IdentifierKindElement, 1, 4)
		require.NotNil(t, id)
		assert.Empty(t, id.UsedDirectives)
		assert.Empty(t, id.Attributes)
	})

	t.Run("should handle svg elements on templates", func(t *testing.T) {
		template := "<svg *ngIf=\"true\"></svg>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id := findIdent(idents, "svg", IdentifierKindTemplate, 1, 4)
		require.NotNil(t, id)
	})

	t.Run("should handle comments in interpolations", func(t *testing.T) {
		template := "{{foo // comment}}"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "foo", IdentifierKindProperty, 2, 5)
		require.NotNil(t, id)
		assert.Nil(t, id.Target)
	})

	t.Run("should handle whitespace and comments in interpolations", func(t *testing.T) {
		template := "{{   foo // comment   }}"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "foo", IdentifierKindProperty, 5, 8)
		require.NotNil(t, id)
		assert.Nil(t, id.Target)
	})

	t.Run("works when structural directives are on templates", func(t *testing.T) {
		template := "<ng-template *ngIf=\"true\">"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id := findIdent(idents, "ng-template", IdentifierKindTemplate, 1, 12)
		require.NotNil(t, id)
	})

	t.Run("should generate nothing in empty template", func(t *testing.T) {
		template := ""
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Empty(t, idents)
	})

	t.Run("should ignore comments", func(t *testing.T) {
		template := "<!-- {{comment}} -->"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Empty(t, idents)
	})

	t.Run("should handle arbitrary whitespace", func(t *testing.T) {
		template := "\n\n   {{foo}}"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "foo", IdentifierKindProperty, 7, 10)
		require.NotNil(t, id)
		assert.Nil(t, id.Target)
	})

	t.Run("should resist collisions", func(t *testing.T) {
		template := "<div [bar]=\"bar ? bar : bar\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id1 := findIdent(idents, "bar", IdentifierKindProperty, 12, 15)
		id2 := findIdent(idents, "bar", IdentifierKindProperty, 18, 21)
		id3 := findIdent(idents, "bar", IdentifierKindProperty, 24, 27)

		assert.NotNil(t, id1)
		assert.NotNil(t, id2)
		assert.NotNil(t, id3)
	})

	t.Run("should discover component properties", func(t *testing.T) {
		template := "{{foo}}"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "foo", IdentifierKindProperty, 2, 5)
		require.NotNil(t, id)
		assert.Nil(t, id.Target)
	})

	t.Run("should discover component properties read using 'this' as a receiver", func(t *testing.T) {
		template := "{{this.foo}}"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "foo", IdentifierKindProperty, 7, 10)
		require.NotNil(t, id)
		assert.Nil(t, id.Target)
	})

	t.Run("should discover nested properties", func(t *testing.T) {
		template := "<div><span>{{foo}}</span></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id := findIdent(idents, "foo", IdentifierKindProperty, 13, 16)
		require.NotNil(t, id)
		assert.Nil(t, id.Target)
	})

	t.Run("should ignore identifiers that are not implicitly received by the template", func(t *testing.T) {
		template := "{{foo.bar.baz}}"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "foo", IdentifierKindProperty, 2, 5)
		require.NotNil(t, id)
	})

	t.Run("should discover properties in bound attributes", func(t *testing.T) {
		template := "<div [bar]=\"bar\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id := findIdent(idents, "bar", IdentifierKindProperty, 12, 15)
		require.NotNil(t, id)
		assert.Nil(t, id.Target)
	})

	t.Run("should handle bound attributes with no value", func(t *testing.T) {
		template := "<div [bar]></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id := findIdent(idents, "div", IdentifierKindElement, 1, 4)
		require.NotNil(t, id)
	})

	t.Run("should discover variables in bound attributes", func(t *testing.T) {
		template := "<div #div [value]=\"div.innerText\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		elemRef := findIdent(idents, "div", IdentifierKindElement, 1, 4)
		require.NotNil(t, elemRef)

		refId := findIdent(idents, "div", IdentifierKindReference, 6, 9)
		require.NotNil(t, refId)
		require.NotNil(t, refId.RefTarget)
		assert.Equal(t, elemRef.Name, refId.RefTarget.Node.Name)

		propId := findIdent(idents, "div", IdentifierKindProperty, 19, 22)
		require.NotNil(t, propId)
		require.NotNil(t, propId.Target)
		assert.Equal(t, refId.Name, propId.Target.Name)
	})

	t.Run("should discover properties in template expressions", func(t *testing.T) {
		template := "<div [bar]=\"bar ? bar1 : bar2\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id1 := findIdent(idents, "bar", IdentifierKindProperty, 12, 15)
		id2 := findIdent(idents, "bar1", IdentifierKindProperty, 18, 22)
		id3 := findIdent(idents, "bar2", IdentifierKindProperty, 25, 29)

		assert.NotNil(t, id1)
		assert.NotNil(t, id2)
		assert.NotNil(t, id3)
	})

	t.Run("should discover properties in structural directive", func(t *testing.T) {
		template := "<div *ngFor=\"let foo of foos\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id := findIdent(idents, "foos", IdentifierKindProperty, 24, 28)
		require.NotNil(t, id)
	})

	t.Run("should discover property writes in bound events", func(t *testing.T) {
		template := "<div (click)=\"foo=bar\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id1 := findIdent(idents, "foo", IdentifierKindProperty, 14, 17)
		id2 := findIdent(idents, "bar", IdentifierKindProperty, 18, 21)

		assert.NotNil(t, id1)
		assert.NotNil(t, id2)
	})

	t.Run("should discover component method calls", func(t *testing.T) {
		template := "{{foo()}}"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)
		assert.Len(t, idents, 1)

		id := findIdent(idents, "foo", IdentifierKindProperty, 2, 5)
		require.NotNil(t, id)
	})

	t.Run("should discover references", func(t *testing.T) {
		template := "<div #foo>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		elemRef := findIdent(idents, "div", IdentifierKindElement, 1, 4)
		require.NotNil(t, elemRef)

		refId := findIdent(idents, "foo", IdentifierKindReference, 6, 9)
		require.NotNil(t, refId)
		require.NotNil(t, refId.RefTarget)
		assert.Equal(t, elemRef.Name, refId.RefTarget.Node.Name)
	})

	t.Run("should discover references to variables", func(t *testing.T) {
		template := "<div *ngFor=\"let foo of foos; let i = index\">{{foo + i}}</div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		fooVar := findIdent(idents, "foo", IdentifierKindVariable, 17, 20)
		require.NotNil(t, fooVar)

		iVar := findIdent(idents, "i", IdentifierKindVariable, 34, 35)
		require.NotNil(t, iVar)

		fooProp := findIdent(idents, "foo", IdentifierKindProperty, 47, 50)
		require.NotNil(t, fooProp)
		require.NotNil(t, fooProp.Target)
		assert.Equal(t, fooVar.Name, fooProp.Target.Name)

		iProp := findIdent(idents, "i", IdentifierKindProperty, 53, 54)
		require.NotNil(t, iProp)
		require.NotNil(t, iProp.Target)
		assert.Equal(t, iVar.Name, iProp.Target.Name)
	})

	t.Run("should discover references to let declaration", func(t *testing.T) {
		template := "@let foo = 123; <div [someInput]=\"foo\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		letDecl := findIdent(idents, "foo", IdentifierKindLetDeclaration, 5, 8)
		require.NotNil(t, letDecl)

		fooProp := findIdent(idents, "foo", IdentifierKindProperty, 34, 37)
		require.NotNil(t, fooProp)
		require.NotNil(t, fooProp.Target)
		assert.Equal(t, letDecl.Name, fooProp.Target.Name)
	})

	t.Run("should generate information about attributes", func(t *testing.T) {
		template := "<div attrA attrB=\"val\"></div>"
		idents, errs := getTemplateIdentifiers(bind(template))
		assert.Empty(t, errs)

		id := findIdent(idents, "div", IdentifierKindElement, 1, 4)
		require.NotNil(t, id)
		assert.Len(t, id.Attributes, 2)

		var attrNames []string
		for _, attr := range id.Attributes {
			attrNames = append(attrNames, attr.Name)
		}
		assert.Contains(t, attrNames, "attrA")
		assert.Contains(t, attrNames, "attrB")
	})

	t.Run("should generate information about used directives", func(t *testing.T) {
		declA, releaseA := getComponentDeclaration(t, "class A {}", "A")
		defer releaseA()
		declB, releaseB := getComponentDeclaration(t, "class B {}", "B")
		defer releaseB()

		template := "<a-selector b-selector></a-selector>"
		boundTemplate := getBoundTemplate(t, template, nil, []bindComponent{
			{selector: "a-selector", declaration: declA},
			{selector: "[b-selector]", declaration: declB},
		})

		idents, errs := getTemplateIdentifiers(boundTemplate)
		assert.Empty(t, errs)

		id := findIdent(idents, "a-selector", IdentifierKindElement, 1, 11)
		require.NotNil(t, id)

		require.Len(t, id.UsedDirectives, 2)
		var selectors []string
		for _, dir := range id.UsedDirectives {
			selectors = append(selectors, dir.Selector)
		}
		assert.Contains(t, selectors, "a-selector")
		assert.Contains(t, selectors, "[b-selector]")
	})

	t.Run("should generate information directive targets", func(t *testing.T) {
		declB, releaseB := getComponentDeclaration(t, "class B {}", "B")
		defer releaseB()

		template := "<div #foo b-selector>"
		boundTemplate := getBoundTemplate(t, template, nil, []bindComponent{
			{selector: "[b-selector]", declaration: declB},
		})

		idents, errs := getTemplateIdentifiers(boundTemplate)
		assert.Empty(t, errs)

		fooRef := findIdent(idents, "foo", IdentifierKindReference, 6, 9)
		require.NotNil(t, fooRef)
		require.NotNil(t, fooRef.RefTarget)
		assert.Equal(t, "div", fooRef.RefTarget.Node.Name)
		assert.Equal(t, declB, fooRef.RefTarget.Directive)
	})

	t.Run("should generate information about selectorless component nodes", func(t *testing.T) {
		compDecl, releaseComp := getComponentDeclaration(t, "class Comp {}", "Comp")
		defer releaseComp()
		fooDecl, releaseFoo := getComponentDeclaration(t, "class Foo {}", "Foo")
		defer releaseFoo()
		barDecl, releaseBar := getComponentDeclaration(t, "class Bar {}", "Bar")
		defer releaseBar()

		template := "<Comp @Foo @Bar([input]=\"value\")/>"
		boundTemplate := getBoundTemplate(t, template, &render3.ParseTemplateOptions{EnableSelectorless: boolPtr(true)}, []bindComponent{
			{selector: "", declaration: compDecl},
			{selector: "", declaration: fooDecl},
			{selector: "", declaration: barDecl},
		})

		idents, errs := getTemplateIdentifiers(boundTemplate)
		assert.Empty(t, errs)

		compRef := findIdent(idents, "Comp", IdentifierKindComponent, 1, 5)
		require.NotNil(t, compRef)
		assert.Len(t, compRef.UsedDirectives, 1)

		fooRef := findIdent(idents, "Foo", IdentifierKindDirective, 7, 10)
		require.NotNil(t, fooRef)

		barRef := findIdent(idents, "Bar", IdentifierKindDirective, 12, 15)
		require.NotNil(t, barRef)

		valRef := findIdent(idents, "value", IdentifierKindProperty, 25, 30)
		require.NotNil(t, valRef)
	})
}

func boolPtr(b bool) *bool {
	return &b
}
