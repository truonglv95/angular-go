package render3

import (
	"sort"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/stretchr/testify/assert"
)

type R3AstSourceSpans struct {
	Result [][]string
}

func humanizeSpan(span parse_util.ParseSourceSpan) string {
	if span.Start == nil || span.End == nil {
		return "<empty>"
	}
	return span.ToString()
}

func humanizePtrSpan(span *parse_util.ParseSourceSpan) string {
	if span == nil {
		return "<empty>"
	}
	return humanizeSpan(*span)
}

func (v *R3AstSourceSpans) visitNodes(nodes interface{}) {
	if nodes == nil {
		return
	}
	switch ns := nodes.(type) {
	case []Node:
		VisitAll(v, ns)
	case []*TextAttribute:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*BoundAttribute:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*BoundEvent:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*Directive:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*Reference:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*Variable:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*SwitchBlockCaseGroup:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*SwitchBlockCase:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	case []*IfBlockBranch:
		var list []Node
		for _, n := range ns {
			list = append(list, n)
		}
		VisitAll(v, list)
	}
}

func (v *R3AstSourceSpans) Visit(node Node) interface{} {
	return nil
}

func (v *R3AstSourceSpans) VisitElement(element *Element) interface{} {
	v.Result = append(v.Result, []string{
		"Element",
		humanizeSpan(element.SourceSpan),
		humanizeSpan(element.StartSourceSpan),
		humanizePtrSpan(element.EndSourceSpan),
	})
	v.visitNodes(element.Attributes)
	v.visitNodes(element.Inputs)
	v.visitNodes(element.Outputs)
	v.visitNodes(element.Directives)
	v.visitNodes(element.References)
	v.visitNodes(element.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitTemplate(template *Template) interface{} {
	v.Result = append(v.Result, []string{
		"Template",
		humanizeSpan(template.SourceSpan),
		humanizeSpan(template.StartSourceSpan),
		humanizePtrSpan(template.EndSourceSpan),
	})
	v.visitNodes(template.Attributes)
	v.visitNodes(template.Inputs)
	v.visitNodes(template.Outputs)
	v.visitNodes(template.Directives)
	v.visitNodes(template.TemplateAttrs)
	v.visitNodes(template.References)
	v.visitNodes(template.Variables)
	v.visitNodes(template.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitContent(content *Content) interface{} {
	v.Result = append(v.Result, []string{
		"Content",
		humanizeSpan(content.SourceSpan),
	})
	v.visitNodes(content.Attributes)
	v.visitNodes(content.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitVariableExpr(variable *Variable) interface{} {
	v.Result = append(v.Result, []string{
		"Variable",
		humanizeSpan(variable.SourceSpan),
		humanizeSpan(variable.KeySpan),
		humanizePtrSpan(variable.ValueSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitReference(reference *Reference) interface{} {
	v.Result = append(v.Result, []string{
		"Reference",
		humanizeSpan(reference.SourceSpan),
		humanizeSpan(reference.KeySpan),
		humanizePtrSpan(reference.ValueSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitTextAttribute(attribute *TextAttribute) interface{} {
	v.Result = append(v.Result, []string{
		"TextAttribute",
		humanizeSpan(attribute.SourceSpan),
		humanizePtrSpan(attribute.KeySpan),
		humanizePtrSpan(attribute.ValueSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitBoundAttribute(attribute *BoundAttribute) interface{} {
	v.Result = append(v.Result, []string{
		"BoundAttribute",
		humanizeSpan(attribute.SourceSpan),
		humanizeSpan(attribute.KeySpan),
		humanizePtrSpan(attribute.ValueSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitBoundEvent(event *BoundEvent) interface{} {
	v.Result = append(v.Result, []string{
		"BoundEvent",
		humanizeSpan(event.SourceSpan),
		humanizeSpan(event.KeySpan),
		humanizeSpan(event.HandlerSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitText(text *Text) interface{} {
	v.Result = append(v.Result, []string{
		"Text",
		humanizeSpan(text.SourceSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitBoundText(text *BoundText) interface{} {
	v.Result = append(v.Result, []string{
		"BoundText",
		humanizeSpan(text.SourceSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitIcu(icu *Icu) interface{} {
	v.Result = append(v.Result, []string{
		"Icu",
		humanizeSpan(icu.SourceSpan),
	})

	// Deterministic sorting of icu.Vars
	var varKeys []string
	for k := range icu.Vars {
		varKeys = append(varKeys, k)
	}
	sort.Strings(varKeys)
	for _, k := range varKeys {
		v.Result = append(v.Result, []string{
			"Icu:Var",
			humanizeSpan(icu.Vars[k].SourceSpan),
		})
	}

	// Deterministic sorting of icu.Placeholders
	var phKeys []string
	for k := range icu.Placeholders {
		phKeys = append(phKeys, k)
	}
	sort.Strings(phKeys)
	for _, k := range phKeys {
		v.Result = append(v.Result, []string{
			"Icu:Placeholder",
			humanizeSpan(icu.Placeholders[k].GetSourceSpan()),
		})
	}
	return nil
}

func (v *R3AstSourceSpans) VisitDeferredBlock(deferred *DeferredBlock) interface{} {
	v.Result = append(v.Result, []string{
		"DeferredBlock",
		humanizeSpan(deferred.SourceSpan),
		humanizeSpan(deferred.StartSourceSpan),
		humanizePtrSpan(deferred.EndSourceSpan),
	})
	deferred.VisitAll(v)
	return nil
}

func (v *R3AstSourceSpans) VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) interface{} {
	v.Result = append(v.Result, []string{
		"DeferredBlockPlaceholder",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
		humanizePtrSpan(block.EndSourceSpan),
	})
	v.visitNodes(block.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitDeferredBlockError(block *DeferredBlockError) interface{} {
	v.Result = append(v.Result, []string{
		"DeferredBlockError",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
		humanizePtrSpan(block.EndSourceSpan),
	})
	v.visitNodes(block.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitDeferredBlockLoading(block *DeferredBlockLoading) interface{} {
	v.Result = append(v.Result, []string{
		"DeferredBlockLoading",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
		humanizePtrSpan(block.EndSourceSpan),
	})
	v.visitNodes(block.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitDeferredTrigger(trigger DeferredTrigger) interface{} {
	var name string
	switch trigger.(type) {
	case *BoundDeferredTrigger:
		name = "BoundDeferredTrigger"
	case *ImmediateDeferredTrigger:
		name = "ImmediateDeferredTrigger"
	case *HoverDeferredTrigger:
		name = "HoverDeferredTrigger"
	case *IdleDeferredTrigger:
		name = "IdleDeferredTrigger"
	case *TimerDeferredTrigger:
		name = "TimerDeferredTrigger"
	case *InteractionDeferredTrigger:
		name = "InteractionDeferredTrigger"
	case *ViewportDeferredTrigger:
		name = "ViewportDeferredTrigger"
	case *NeverDeferredTrigger:
		name = "NeverDeferredTrigger"
	default:
		panic("Unknown trigger")
	}
	v.Result = append(v.Result, []string{
		name,
		humanizeSpan(trigger.GetSourceSpan()),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitSwitchBlock(block *SwitchBlock) interface{} {
	v.Result = append(v.Result, []string{
		"SwitchBlock",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
		humanizePtrSpan(block.EndSourceSpan),
	})
	v.visitNodes(block.Groups)
	if block.ExhaustiveCheck != nil {
		block.ExhaustiveCheck.Visit(v)
	}
	return nil
}

func (v *R3AstSourceSpans) VisitSwitchBlockCase(block *SwitchBlockCase) interface{} {
	v.Result = append(v.Result, []string{
		"SwitchBlockCase",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitSwitchBlockCaseGroup(block *SwitchBlockCaseGroup) interface{} {
	v.Result = append(v.Result, []string{
		"SwitchBlockCaseGroup",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
	})
	v.visitNodes(block.Cases)
	v.visitNodes(block.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitSwitchExhaustiveCheck(block *SwitchExhaustiveCheck) interface{} {
	v.Result = append(v.Result, []string{
		"SwitchExhaustiveCheck",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitForLoopBlock(block *ForLoopBlock) interface{} {
	v.Result = append(v.Result, []string{
		"ForLoopBlock",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
		humanizePtrSpan(block.EndSourceSpan),
	})
	if block.Item != nil {
		block.Item.Visit(v)
	}
	v.visitNodes(block.ContextVariables)
	v.visitNodes(block.Children)
	if block.Empty != nil {
		block.Empty.Visit(v)
	}
	return nil
}

func (v *R3AstSourceSpans) VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) interface{} {
	v.Result = append(v.Result, []string{
		"ForLoopBlockEmpty",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
	})
	v.visitNodes(block.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitIfBlock(block *IfBlock) interface{} {
	v.Result = append(v.Result, []string{
		"IfBlock",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
		humanizePtrSpan(block.EndSourceSpan),
	})
	v.visitNodes(block.Branches)
	return nil
}

func (v *R3AstSourceSpans) VisitIfBlockBranch(block *IfBlockBranch) interface{} {
	v.Result = append(v.Result, []string{
		"IfBlockBranch",
		humanizeSpan(block.SourceSpan),
		humanizeSpan(block.StartSourceSpan),
	})
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(v)
	}
	v.visitNodes(block.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitUnknownBlock(block *UnknownBlock) interface{} {
	v.Result = append(v.Result, []string{
		"UnknownBlock",
		humanizeSpan(block.SourceSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitLetDeclaration(decl *LetDeclaration) interface{} {
	v.Result = append(v.Result, []string{
		"LetDeclaration",
		humanizeSpan(decl.SourceSpan),
		humanizeSpan(decl.NameSpan),
		humanizeSpan(decl.ValueSpan),
	})
	return nil
}

func (v *R3AstSourceSpans) VisitComponent(component *Component) interface{} {
	v.Result = append(v.Result, []string{
		"Component",
		humanizeSpan(component.SourceSpan),
		humanizeSpan(component.StartSourceSpan),
		humanizePtrSpan(component.EndSourceSpan),
	})
	v.visitNodes(component.Attributes)
	v.visitNodes(component.Inputs)
	v.visitNodes(component.Outputs)
	v.visitNodes(component.Directives)
	v.visitNodes(component.References)
	v.visitNodes(component.Children)
	return nil
}

func (v *R3AstSourceSpans) VisitDirective(directive *Directive) interface{} {
	v.Result = append(v.Result, []string{
		"Directive",
		humanizeSpan(directive.SourceSpan),
		humanizeSpan(directive.StartSourceSpan),
		humanizePtrSpan(directive.EndSourceSpan),
	})
	v.visitNodes(directive.Attributes)
	v.visitNodes(directive.Inputs)
	v.visitNodes(directive.Outputs)
	v.visitNodes(directive.References)
	return nil
}

func expectFromHtml(html string, selectorlessEnabled bool) [][]string {
	result := parseR3(html, ParseR3Options{SelectorlessEnabled: selectorlessEnabled})
	return expectFromR3Nodes(result.Nodes)
}

func expectFromR3Nodes(nodes []Node) [][]string {
	humanizer := &R3AstSourceSpans{}
	VisitAll(humanizer, nodes)
	return humanizer.Result
}

type HasSourceSpan interface {
	GetSourceSpan() expression_parser.AbsoluteSourceSpan
}

func getASTSourceSpan(ast expression_parser.AST) expression_parser.AbsoluteSourceSpan {
	if hasSpan, ok := ast.(HasSourceSpan); ok {
		return hasSpan.GetSourceSpan()
	}
	panic("AST node does not implement GetSourceSpan()")
}

func TestR3AstSpans(t *testing.T) {
	t.Run("nodes without binding", func(t *testing.T) {
		t.Run("is correct for text nodes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Text", "a"},
			}, expectFromHtml("a", false))
		})

		t.Run("is correct for elements with attributes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div a=\"b\"></div>", "<div a=\"b\">", "</div>"},
				{"TextAttribute", "a=\"b\"", "a", "b"},
			}, expectFromHtml("<div a=\"b\"></div>", false))
		})

		t.Run("is correct for elements with attributes without value", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div a></div>", "<div a>", "</div>"},
				{"TextAttribute", "a", "a", "<empty>"},
			}, expectFromHtml("<div a></div>", false))
		})

		t.Run("is correct for self-closing elements with trailing whitespace", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<input />", "<input />", "<input />"},
				{"Element", "<span>\n</span>", "<span>", "</span>"},
			}, expectFromHtml("<input />\n  <span>\n</span>", false))
		})
	})

	t.Run("bound text nodes", func(t *testing.T) {
		t.Run("is correct for bound text nodes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"BoundText", "{{a}}"},
			}, expectFromHtml("{{a}}", false))
		})
	})

	t.Run("bound attributes", func(t *testing.T) {
		t.Run("is correct for bound properties", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div [someProp]=\"v\"></div>", "<div [someProp]=\"v\">", "</div>"},
				{"BoundAttribute", "[someProp]=\"v\"", "someProp", "v"},
			}, expectFromHtml("<div [someProp]=\"v\"></div>", false))
		})

		t.Run("is correct for bound properties without value", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div [someProp]></div>", "<div [someProp]>", "</div>"},
				{"BoundAttribute", "[someProp]", "someProp", "<empty>"},
			}, expectFromHtml("<div [someProp]></div>", false))
		})

		t.Run("is correct for bound properties via bind-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div bind-prop=\"v\"></div>", "<div bind-prop=\"v\">", "</div>"},
				{"BoundAttribute", "bind-prop=\"v\"", "prop", "v"},
			}, expectFromHtml("<div bind-prop=\"v\"></div>", false))
		})

		t.Run("is correct for bound properties via {{...}}", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div prop=\"{{v}}\"></div>", "<div prop=\"{{v}}\">", "</div>"},
				{"BoundAttribute", "prop=\"{{v}}\"", "prop", "{{v}}"},
			}, expectFromHtml("<div prop=\"{{v}}\"></div>", false))
		})

		t.Run("is correct for bound properties via data-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div data-prop=\"{{v}}\"></div>", "<div data-prop=\"{{v}}\">", "</div>"},
				{"BoundAttribute", "data-prop=\"{{v}}\"", "data-prop", "{{v}}"},
			}, expectFromHtml("<div data-prop=\"{{v}}\"></div>", false))
		})

		t.Run("is correct for bound properties via @", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div bind-@animation=\"v\"></div>", "<div bind-@animation=\"v\">", "</div>"},
				{"BoundAttribute", "bind-@animation=\"v\"", "animation", "v"},
			}, expectFromHtml("<div bind-@animation=\"v\"></div>", false))
		})

		t.Run("is correct for bound properties via animation-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div bind-animate-animationName=\"v\"></div>", "<div bind-animate-animationName=\"v\">", "</div>"},
				{"BoundAttribute", "bind-animate-animationName=\"v\"", "animationName", "v"},
			}, expectFromHtml("<div bind-animate-animationName=\"v\"></div>", false))
		})

		t.Run("is correct for bound properties via @ without value", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div @animation></div>", "<div @animation>", "</div>"},
				{"BoundAttribute", "@animation", "animation", "<empty>"},
			}, expectFromHtml("<div @animation></div>", false))
		})

		t.Run("should not throw off span of value in bound attribute when leading spaces are present", func(t *testing.T) {
			assertValueSpan := func(template string, start int, end int) {
				result := parseR3(template, ParseR3Options{})
				boundAttribute := result.Nodes[0].(*Element).Inputs[0]
				astWithSource := boundAttribute.Value.(*expression_parser.ASTWithSource)
				span := getASTSourceSpan(astWithSource.Ast)
				assert.Equal(t, start, span.Start)
				assert.Equal(t, end, span.End)
			}

			assertValueSpan("<a [b]=\"helloWorld\"></a>", 8, 18)
			assertValueSpan("<a [b]=\" helloWorld\"></a>", 9, 19)
			assertValueSpan("<a [b]=\"  helloWorld\"></a>", 10, 20)
			assertValueSpan("<a [b]=\"   helloWorld\"></a>", 11, 21)
			assertValueSpan("<a [b]=\"    helloWorld\"></a>", 12, 22)
			assertValueSpan("<a [b]=\"                                          helloWorld\"></a>", 50, 60)
		})

		t.Run("should not throw off span of value in template attribute when leading spaces are present", func(t *testing.T) {
			assertValueSpan := func(template string, start int, end int) {
				result := parseR3(template, ParseR3Options{})
				boundAttribute := result.Nodes[0].(*Template).TemplateAttrs[0].(*BoundAttribute)
				astWithSource := boundAttribute.Value.(*expression_parser.ASTWithSource)
				span := getASTSourceSpan(astWithSource.Ast)
				assert.Equal(t, start, span.Start)
				assert.Equal(t, end, span.End)
			}

			assertValueSpan("<ng-container *ngTemplateOutlet=\"helloWorld\"/>", 33, 43)
			assertValueSpan("<ng-container *ngTemplateOutlet=\" helloWorld\"/>", 34, 44)
			assertValueSpan("<ng-container *ngTemplateOutlet=\"  helloWorld\"/>", 35, 45)
			assertValueSpan("<ng-container *ngTemplateOutlet=\"   helloWorld\"/>", 36, 46)
			assertValueSpan("<ng-container *ngTemplateOutlet=\"    helloWorld\"/>", 37, 47)
			assertValueSpan("<ng-container *ngTemplateOutlet=\"                    helloWorld\"/>", 53, 63)
		})
	})

	t.Run("templates", func(t *testing.T) {
		t.Run("is correct for * directives", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<div *ngIf></div>", "<div *ngIf>", "</div>"},
				{"TextAttribute", "ngIf", "ngIf", "<empty>"},
				{"Element", "<div *ngIf></div>", "<div *ngIf>", "</div>"},
			}, expectFromHtml("<div *ngIf></div>", false))
		})

		t.Run("is correct for <ng-template>", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template></ng-template>", "<ng-template>", "</ng-template>"},
			}, expectFromHtml("<ng-template></ng-template>", false))
		})

		t.Run("is correct for reference via #...", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template #a></ng-template>", "<ng-template #a>", "</ng-template>"},
				{"Reference", "#a", "a", "<empty>"},
			}, expectFromHtml("<ng-template #a></ng-template>", false))
		})

		t.Run("is correct for reference with name", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template #a=\"b\"></ng-template>", "<ng-template #a=\"b\">", "</ng-template>"},
				{"Reference", "#a=\"b\"", "a", "b"},
			}, expectFromHtml("<ng-template #a=\"b\"></ng-template>", false))
		})

		t.Run("is correct for reference via ref-...", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template ref-a></ng-template>", "<ng-template ref-a>", "</ng-template>"},
				{"Reference", "ref-a", "a", "<empty>"},
			}, expectFromHtml("<ng-template ref-a></ng-template>", false))
		})

		t.Run("is correct for data-ref-... attribute", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template data-ref-a></ng-template>", "<ng-template data-ref-a>", "</ng-template>"},
				{"TextAttribute", "data-ref-a", "data-ref-a", "<empty>"},
			}, expectFromHtml("<ng-template data-ref-a></ng-template>", false))
		})

		t.Run("is correct for variables via let-...", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template let-a=\"b\"></ng-template>", "<ng-template let-a=\"b\">", "</ng-template>"},
				{"Variable", "let-a=\"b\"", "a", "b"},
			}, expectFromHtml("<ng-template let-a=\"b\"></ng-template>", false))
		})

		t.Run("is correct for data-let-... attribute", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template data-let-a=\"b\"></ng-template>", "<ng-template data-let-a=\"b\">", "</ng-template>"},
				{"TextAttribute", "data-let-a=\"b\"", "data-let-a", "b"},
			}, expectFromHtml("<ng-template data-let-a=\"b\"></ng-template>", false))
		})

		t.Run("is correct for attributes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template k1=\"v1\"></ng-template>", "<ng-template k1=\"v1\">", "</ng-template>"},
				{"TextAttribute", "k1=\"v1\"", "k1", "v1"},
			}, expectFromHtml("<ng-template k1=\"v1\"></ng-template>", false))
		})

		t.Run("is correct for bound attributes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<ng-template [k1]=\"v1\"></ng-template>", "<ng-template [k1]=\"v1\">", "</ng-template>"},
				{"BoundAttribute", "[k1]=\"v1\"", "k1", "v1"},
			}, expectFromHtml("<ng-template [k1]=\"v1\"></ng-template>", false))
		})
	})

	t.Run("inline templates", func(t *testing.T) {
		t.Run("is correct for attribute and bound attributes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<div *ngFor=\"let item of items\"></div>", "<div *ngFor=\"let item of items\">", "</div>"},
				{"TextAttribute", "ngFor", "ngFor", "<empty>"},
				{"BoundAttribute", "of items", "of", ""},
				{"Variable", "let item ", "item", "<empty>"},
				{"Element", "<div *ngFor=\"let item of items\"></div>", "<div *ngFor=\"let item of items\">", "</div>"},
			}, expectFromHtml("<div *ngFor=\"let item of items\"></div>", false))

			assert.Equal(t, [][]string{
				{"Template", "<div *ngFor=\"item of items\"></div>", "<div *ngFor=\"item of items\">", "</div>"},
				{"BoundAttribute", "ngFor=\"item ", "ngFor", ""},
				{"BoundAttribute", "of items", "of", ""},
				{"Element", "<div *ngFor=\"item of items\"></div>", "<div *ngFor=\"item of items\">", "</div>"},
			}, expectFromHtml("<div *ngFor=\"item of items\"></div>", false))

			assert.Equal(t, [][]string{
				{"Template", "<div *ngFor=\"let item of items; trackBy: trackByFn\"></div>", "<div *ngFor=\"let item of items; trackBy: trackByFn\">", "</div>"},
				{"TextAttribute", "ngFor", "ngFor", "<empty>"},
				{"BoundAttribute", "of items; ", "of", ""},
				{"BoundAttribute", "trackBy: trackByFn", "trackBy", ""},
				{"Variable", "let item ", "item", "<empty>"},
				{"Element", "<div *ngFor=\"let item of items; trackBy: trackByFn\"></div>", "<div *ngFor=\"let item of items; trackBy: trackByFn\">", "</div>"},
			}, expectFromHtml("<div *ngFor=\"let item of items; trackBy: trackByFn\"></div>", false))
		})

		t.Run("is correct for variables via let ...", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<div *ngIf=\"let a=b\"></div>", "<div *ngIf=\"let a=b\">", "</div>"},
				{"TextAttribute", "ngIf", "ngIf", "<empty>"},
				{"Variable", "let a=b", "a", "<empty>"},
				{"Element", "<div *ngIf=\"let a=b\"></div>", "<div *ngIf=\"let a=b\">", "</div>"},
			}, expectFromHtml("<div *ngIf=\"let a=b\"></div>", false))
		})

		t.Run("is correct for variables via as ...", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Template", "<div *ngIf=\"expr as local\"></div>", "<div *ngIf=\"expr as local\">", "</div>"},
				{"BoundAttribute", "ngIf=\"expr ", "ngIf", ""},
				{"Variable", "ngIf=\"expr as local", "local", "<empty>"},
				{"Element", "<div *ngIf=\"expr as local\"></div>", "<div *ngIf=\"expr as local\">", "</div>"},
			}, expectFromHtml("<div *ngIf=\"expr as local\"></div>", false))
		})
	})

	t.Run("events", func(t *testing.T) {
		t.Run("is correct for event names case sensitive", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div (someEvent)=\"v\"></div>", "<div (someEvent)=\"v\">", "</div>"},
				{"BoundEvent", "(someEvent)=\"v\"", "someEvent", "v"},
			}, expectFromHtml("<div (someEvent)=\"v\"></div>", false))
		})

		t.Run("is correct for bound events via on-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div on-event=\"v\"></div>", "<div on-event=\"v\">", "</div>"},
				{"BoundEvent", "on-event=\"v\"", "event", "v"},
			}, expectFromHtml("<div on-event=\"v\"></div>", false))
		})

		t.Run("is correct for text attribute via data-on-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div data-on-event=\"v\"></div>", "<div data-on-event=\"v\">", "</div>"},
				{"TextAttribute", "data-on-event=\"v\"", "data-on-event", "v"},
			}, expectFromHtml("<div data-on-event=\"v\"></div>", false))
		})

		t.Run("is correct for bound events and properties via [(...)]", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div [(prop)]=\"v\"></div>", "<div [(prop)]=\"v\">", "</div>"},
				{"BoundAttribute", "[(prop)]=\"v\"", "prop", "v"},
				{"BoundEvent", "[(prop)]=\"v\"", "prop", "v"},
			}, expectFromHtml("<div [(prop)]=\"v\"></div>", false))
		})

		t.Run("is correct for bound events and properties via bindon-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div bindon-prop=\"v\"></div>", "<div bindon-prop=\"v\">", "</div>"},
				{"BoundAttribute", "bindon-prop=\"v\"", "prop", "v"},
				{"BoundEvent", "bindon-prop=\"v\"", "prop", "v"},
			}, expectFromHtml("<div bindon-prop=\"v\"></div>", false))
		})

		t.Run("is correct for TextAttribute and properties via data-bindon-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div data-bindon-prop=\"v\"></div>", "<div data-bindon-prop=\"v\">", "</div>"},
				{"TextAttribute", "data-bindon-prop=\"v\"", "data-bindon-prop", "v"},
			}, expectFromHtml("<div data-bindon-prop=\"v\"></div>", false))
		})

		t.Run("is correct for bound events via @", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div (@name.done)=\"v\"></div>", "<div (@name.done)=\"v\">", "</div>"},
				{"BoundEvent", "(@name.done)=\"v\"", "name.done", "v"},
			}, expectFromHtml("<div (@name.done)=\"v\"></div>", false))
		})
	})

	t.Run("references", func(t *testing.T) {
		t.Run("is correct for references via #...", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div #a></div>", "<div #a>", "</div>"},
				{"Reference", "#a", "a", "<empty>"},
			}, expectFromHtml("<div #a></div>", false))
		})

		t.Run("is correct for references with name", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div #a=\"b\"></div>", "<div #a=\"b\">", "</div>"},
				{"Reference", "#a=\"b\"", "a", "b"},
			}, expectFromHtml("<div #a=\"b\"></div>", false))
		})

		t.Run("is correct for references via ref-", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div ref-a></div>", "<div ref-a>", "</div>"},
				{"Reference", "ref-a", "a", "<empty>"},
			}, expectFromHtml("<div ref-a></div>", false))
		})
	})

	t.Run("ICU expressions", func(t *testing.T) {
		t.Run("is correct for variables and placeholders", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<span i18n>{item.var, plural, other { {{item.placeholder}} items } }</span>", "<span i18n>", "</span>"},
				{"TextAttribute", "i18n", "i18n", "<empty>"},
				{"Icu", "{item.var, plural, other { {{item.placeholder}} items } }"},
				{"Icu:Var", "item.var"},
				{"Icu:Placeholder", "{{item.placeholder}}"},
			}, expectFromHtml("<span i18n>{item.var, plural, other { {{item.placeholder}} items } }</span>", false))
		})

		t.Run("is correct for nested ICUs", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<span i18n>{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }</span>", "<span i18n>", "</span>"},
				{"TextAttribute", "i18n", "i18n", "<empty>"},
				{"Icu", "{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }"},
				{"Icu:Var", "item.var"},
				{"Icu:Var", "nestedVar"},
				{"Icu:Placeholder", "{{item.placeholder}}"},
				{"Icu:Placeholder", "{{nestedPlaceholder}}"},
			}, expectFromHtml("<span i18n>{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }</span>", false))
		})
	})

	t.Run("deferred blocks", func(t *testing.T) {
		t.Run("is correct for deferred blocks", func(t *testing.T) {
			html := "@defer (when isVisible() && foo; on hover(button), timer(10s), idle, immediate, " +
				"interaction(button), viewport(container); prefetch on immediate; " +
				"prefetch when isDataLoaded(); hydrate on interaction; hydrate when isVisible(); hydrate on timer(1200)) {<calendar-cmp [date]=\"current\"/>}" +
				"@loading (minimum 1s; after 100ms) {Loading...}" +
				"@placeholder (minimum 500) {Placeholder content!}" +
				"@error {Loading failed :(}"

			assert.Equal(t, [][]string{
				{"DeferredBlock", html, "@defer (when isVisible() && foo; on hover(button), timer(10s), idle, immediate, interaction(button), viewport(container); prefetch on immediate; prefetch when isDataLoaded(); hydrate on interaction; hydrate when isVisible(); hydrate on timer(1200)) {", "}"},
				{"BoundDeferredTrigger", "hydrate when isVisible()"},
				{"TimerDeferredTrigger", "hydrate on timer(1200)"},
				{"InteractionDeferredTrigger", "hydrate on interaction"},
				{"BoundDeferredTrigger", "when isVisible() && foo"},
				{"IdleDeferredTrigger", "idle"},
				{"ImmediateDeferredTrigger", "immediate"},
				{"HoverDeferredTrigger", "on hover(button)"},
				{"TimerDeferredTrigger", "timer(10s)"},
				{"InteractionDeferredTrigger", "interaction(button)"},
				{"ViewportDeferredTrigger", "viewport(container)"},
				{"BoundDeferredTrigger", "prefetch when isDataLoaded()"},
				{"ImmediateDeferredTrigger", "prefetch on immediate"},
				{"Element", "<calendar-cmp [date]=\"current\"/>", "<calendar-cmp [date]=\"current\"/>", "<calendar-cmp [date]=\"current\"/>"},
				{"BoundAttribute", "[date]=\"current\"", "date", "current"},
				{"DeferredBlockPlaceholder", "@placeholder (minimum 500) {Placeholder content!}", "@placeholder (minimum 500) {", "}"},
				{"Text", "Placeholder content!"},
				{"DeferredBlockLoading", "@loading (minimum 1s; after 100ms) {Loading...}", "@loading (minimum 1s; after 100ms) {", "}"},
				{"Text", "Loading..."},
				{"DeferredBlockError", "@error {Loading failed :(}", "@error {", "}"},
				{"Text", "Loading failed :("},
			}, expectFromHtml(html, false))
		})
	})

	t.Run("switch blocks", func(t *testing.T) {
		t.Run("is correct for switch blocks", func(t *testing.T) {
			html := "@switch (cond.kind) {" +
				"@case (x()) {X case}" +
				"@case ('hello') {Y case}" +
				"@case (42) {Z case}" +
				"@default {No case matched}" +
				"}"

			assert.Equal(t, [][]string{
				{"SwitchBlock", html, "@switch (cond.kind) {", "}"},
				{"SwitchBlockCaseGroup", "@case (x()) {X case}", "@case (x()) {"},
				{"SwitchBlockCase", "@case (x()) {X case}", "@case (x()) {"},
				{"Text", "X case"},
				{"SwitchBlockCaseGroup", "@case ('hello') {Y case}", "@case ('hello') {"},
				{"SwitchBlockCase", "@case ('hello') {Y case}", "@case ('hello') {"},
				{"Text", "Y case"},
				{"SwitchBlockCaseGroup", "@case (42) {Z case}", "@case (42) {"},
				{"SwitchBlockCase", "@case (42) {Z case}", "@case (42) {"},
				{"Text", "Z case"},
				{"SwitchBlockCaseGroup", "@default {No case matched}", "@default {"},
				{"SwitchBlockCase", "@default {No case matched}", "@default {"},
				{"Text", "No case matched"},
			}, expectFromHtml(html, false))
		})

		t.Run("is correct for switch blocks with consecutive cases", func(t *testing.T) {
			html := "@switch (cond.kind) {" +
				"@case (x()) @case ('hello') {X case}" +
				"@default {No case matched}" +
				"}"

			assert.Equal(t, [][]string{
				{"SwitchBlock", html, "@switch (cond.kind) {", "}"},
				{"SwitchBlockCaseGroup", "@case (x()) @case ('hello') {X case}", "@case (x()) @case ('hello') {"},
				{"SwitchBlockCase", "@case (x()) ", "@case (x()) "},
				{"SwitchBlockCase", "@case ('hello') {X case}", "@case ('hello') {"},
				{"Text", "X case"},
				{"SwitchBlockCaseGroup", "@default {No case matched}", "@default {"},
				{"SwitchBlockCase", "@default {No case matched}", "@default {"},
				{"Text", "No case matched"},
			}, expectFromHtml(html, false))
		})

		t.Run("is correct for switch blocks with exhaustive checking", func(t *testing.T) {
			html := "@switch (cond.kind) {" + "@case (x()) {X case}" + "@default never;" + "}"

			assert.Equal(t, [][]string{
				{"SwitchBlock", html, "@switch (cond.kind) {", "}"},
				{"SwitchBlockCaseGroup", "@case (x()) {X case}", "@case (x()) {"},
				{"SwitchBlockCase", "@case (x()) {X case}", "@case (x()) {"},
				{"Text", "X case"},
				{"SwitchExhaustiveCheck", "@default never;", "@default never;"},
			}, expectFromHtml(html, false))
		})
	})

	t.Run("for loop blocks", func(t *testing.T) {
		t.Run("is correct for loop blocks", func(t *testing.T) {
			html := "@for (item of items.foo.bar; track item.id; let i = $index, _o_d_d_ = $odd) {<h1>{{ item }}</h1>}" +
				"@empty {There were no items in the list.}"

			assert.Equal(t, [][]string{
				{"ForLoopBlock", html, "@for (item of items.foo.bar; track item.id; let i = $index, _o_d_d_ = $odd) {", "}"},
				{"Variable", "item", "item", "<empty>"},
				{"Variable", "", "", "<empty>"},
				{"Variable", "", "", "<empty>"},
				{"Variable", "", "", "<empty>"},
				{"Variable", "", "", "<empty>"},
				{"Variable", "", "", "<empty>"},
				{"Variable", "", "", "<empty>"},
				{"Variable", "i = $index", "i = $index, _o_d_d_ = $odd", "$index"},
				{"Variable", "_o_d_d_ = $odd", "_o_d_d_", "$odd"},
				{"Element", "<h1>{{ item }}</h1>", "<h1>", "</h1>"},
				{"BoundText", "{{ item }}"},
				{"ForLoopBlockEmpty", "@empty {There were no items in the list.}", "@empty {"},
				{"Text", "There were no items in the list."},
			}, expectFromHtml(html, false))
		})
	})

	t.Run("if blocks", func(t *testing.T) {
		t.Run("is correct for if blocks", func(t *testing.T) {
			html := "@if (cond.expr; as foo) {Main case was true!}" +
				"@else if (other.expr) {Extra case was true!}" +
				"@else {False case!}"

			assert.Equal(t, [][]string{
				{"IfBlock", html, "@if (cond.expr; as foo) {", "}"},
				{"IfBlockBranch", "@if (cond.expr; as foo) {Main case was true!}", "@if (cond.expr; as foo) {"},
				{"Variable", "foo", "foo", "<empty>"},
				{"Text", "Main case was true!"},
				{"IfBlockBranch", "@else if (other.expr) {Extra case was true!}", "@else if (other.expr) {"},
				{"Text", "Extra case was true!"},
				{"IfBlockBranch", "@else {False case!}", "@else {"},
				{"Text", "False case!"},
			}, expectFromHtml(html, false))
		})
	})

	t.Run("@let declaration", func(t *testing.T) {
		t.Run("is correct for a let declaration", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"LetDeclaration", "@let foo = 123;", "@let foo", "123"},
			}, expectFromHtml("@let foo = 123;", false))
		})
	})

	t.Run("component tags", func(t *testing.T) {
		t.Run("is correct for a simple component", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Component", "<MyComp></MyComp>", "<MyComp>", "</MyComp>"},
			}, expectFromHtml("<MyComp></MyComp>", true))
		})

		t.Run("is correct for a self-closing component", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Component", "<MyComp/>", "<MyComp/>", "<MyComp/>"},
			}, expectFromHtml("<MyComp/>", true))
		})

		t.Run("is correct for a component with a tag name", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Component", "<MyComp:button></MyComp:button>", "<MyComp:button>", "</MyComp:button>"},
			}, expectFromHtml("<MyComp:button></MyComp:button>", true))
		})

		t.Run("is correct for a component with attributes and directives", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Component", "<MyComp before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\">Hello</MyComp>", "<MyComp before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\">", "</MyComp>"},
				{"TextAttribute", "before=\"foo\"", "before", "foo"},
				{"TextAttribute", "middle", "middle", "<empty>"},
				{"TextAttribute", "after=\"123\"", "after", "123"},
				{"Directive", "<empty>", "@Dir", "@Dir"},
				{"Directive", "<empty>", "@OtherDir", ")"},
				{"BoundAttribute", "[a]=\"a\"", "a", "a"},
				{"BoundEvent", "(b)=\"b()\"", "b", "b()"},
				{"Text", "Hello"},
			}, expectFromHtml("<MyComp before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\">Hello</MyComp>", true))
		})

		t.Run("is correct for a component nested inside other markup", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"IfBlock", "@if (expr) {<div>Hello: <MyComp><span><OtherComp></OtherComp></span></MyComp></div>}", "@if (expr) {", "}"},
				{"IfBlockBranch", "@if (expr) {<div>Hello: <MyComp><span><OtherComp></OtherComp></span></MyComp></div>}", "@if (expr) {"},
				{"Element", "<div>Hello: <MyComp><span><OtherComp></OtherComp></span></MyComp></div>", "<div>", "</div>"},
				{"Text", "Hello: "},
				{"Component", "<MyComp><span><OtherComp></OtherComp></span></MyComp>", "<MyComp>", "</MyComp>"},
				{"Element", "<span><OtherComp></OtherComp></span>", "<span>", "</span>"},
				{"Component", "<OtherComp></OtherComp>", "<OtherComp>", "</OtherComp>"},
			}, expectFromHtml("@if (expr) {<div>Hello: <MyComp><span><OtherComp></OtherComp></span></MyComp></div>}", true))
		})
	})

	t.Run("directives", func(t *testing.T) {
		t.Run("is correct for a directive with no attributes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div @Dir></div>", "<div @Dir>", "</div>"},
				{"Directive", "<empty>", "@Dir", "@Dir"},
			}, expectFromHtml("<div @Dir></div>", true))
		})

		t.Run("is correct for a directive with attributes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div @Dir(a=\"1\" [b]=\"two\" (c)=\"c()\")></div>", "<div @Dir(a=\"1\" [b]=\"two\" (c)=\"c()\")>", "</div>"},
				{"Directive", "<empty>", "@Dir", ")"},
				{"TextAttribute", "a=\"1\"", "a", "1"},
				{"BoundAttribute", "[b]=\"two\"", "b", "two"},
				{"BoundEvent", "(c)=\"c()\"", "c", "c()"},
			}, expectFromHtml("<div @Dir(a=\"1\" [b]=\"two\" (c)=\"c()\")></div>", true))
		})

		t.Run("is correct for directives mixed with other attributes", func(t *testing.T) {
			assert.Equal(t, [][]string{
				{"Element", "<div before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\"></div>", "<div before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\">", "</div>"},
				{"TextAttribute", "before=\"foo\"", "before", "foo"},
				{"TextAttribute", "middle", "middle", "<empty>"},
				{"TextAttribute", "after=\"123\"", "after", "123"},
				{"Directive", "<empty>", "@Dir", "@Dir"},
				{"Directive", "<empty>", "@OtherDir", ")"},
				{"BoundAttribute", "[a]=\"a\"", "a", "a"},
				{"BoundEvent", "(b)=\"b()\"", "b", "b()"},
			}, expectFromHtml("<div before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\"></div>", true))
		})
	})
}
