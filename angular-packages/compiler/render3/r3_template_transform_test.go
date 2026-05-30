package render3

import (
	"fmt"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
	"github.com/stretchr/testify/assert"
)

type R3AstHumanizer struct {
	Result [][]any
}

func (h *R3AstHumanizer) Visit(node Node) interface{} {
	return nil
}

func (h *R3AstHumanizer) visitAllElements(nodes []Node) {
	for _, n := range nodes {
		if n != nil {
			n.Visit(h)
		}
	}
}

func (h *R3AstHumanizer) visitAllAttributes(nodes []*TextAttribute) {
	for _, n := range nodes {
		if n != nil {
			n.Visit(h)
		}
	}
}

func (h *R3AstHumanizer) visitAllBoundAttributes(nodes []*BoundAttribute) {
	for _, n := range nodes {
		if n != nil {
			n.Visit(h)
		}
	}
}

func (h *R3AstHumanizer) visitAllBoundEvents(nodes []*BoundEvent) {
	for _, n := range nodes {
		if n != nil {
			n.Visit(h)
		}
	}
}

func (h *R3AstHumanizer) visitAllDirectives(nodes []*Directive) {
	for _, n := range nodes {
		if n != nil {
			n.Visit(h)
		}
	}
}

func (h *R3AstHumanizer) visitAllReferences(nodes []*Reference) {
	for _, n := range nodes {
		if n != nil {
			n.Visit(h)
		}
	}
}

func (h *R3AstHumanizer) visitAllVariables(nodes []*Variable) {
	for _, n := range nodes {
		if n != nil {
			n.Visit(h)
		}
	}
}

func mapBindingTypeToTs(bt int) int {
	switch bt {
	case 0: // Property
		return 0
	case 1: // Attribute
		return 1
	case 2: // Class
		return 2
	case 3: // Style
		return 3
	case 4: // Animation
		return 6 // TS Animation
	case 5: // TwoWay
		return 5 // TS TwoWay
	case 6: // LegacyAnimation
		return 4 // TS LegacyAnimation
	default:
		return bt
	}
}

func mapParsedEventTypeToTs(et template_parser.ParsedEventType) int {
	switch et {
	case template_parser.ParsedEventTypeRegular:
		return 0
	case template_parser.ParsedEventTypeLegacyAnimation:
		return 1
	case template_parser.ParsedEventTypeTwoWay:
		return 2
	case template_parser.ParsedEventTypeAnimation:
		return 3
	default:
		return int(et)
	}
}

func (h *R3AstHumanizer) VisitElement(element *Element) interface{} {
	res := []any{"Element", element.Name}
	if element.IsSelfClosing {
		res = append(res, "#selfClosing")
	}
	h.Result = append(h.Result, res)
	h.visitAllAttributes(element.Attributes)
	h.visitAllBoundAttributes(element.Inputs)
	h.visitAllBoundEvents(element.Outputs)
	h.visitAllDirectives(element.Directives)
	h.visitAllReferences(element.References)
	h.visitAllElements(element.Children)
	return nil
}

func (h *R3AstHumanizer) VisitTemplate(template *Template) interface{} {
	res := []any{"Template"}
	if template.IsSelfClosing {
		res = append(res, "#selfClosing")
	}
	h.Result = append(h.Result, res)
	h.visitAllAttributes(template.Attributes)
	h.visitAllBoundAttributes(template.Inputs)
	h.visitAllBoundEvents(template.Outputs)
	h.visitAllDirectives(template.Directives)
	h.visitAllElements(template.TemplateAttrs)
	h.visitAllReferences(template.References)
	h.visitAllVariables(template.Variables)
	h.visitAllElements(template.Children)
	return nil
}

func (h *R3AstHumanizer) VisitContent(content *Content) interface{} {
	res := []any{"Content", content.Selector}
	if content.IsSelfClosing {
		res = append(res, "#selfClosing")
	}
	h.Result = append(h.Result, res)
	h.visitAllAttributes(content.Attributes)
	h.visitAllElements(content.Children)
	return nil
}

func (h *R3AstHumanizer) VisitVariableExpr(variable *Variable) interface{} {
	h.Result = append(h.Result, []any{"Variable", variable.Name, variable.Value})
	return nil
}

func (h *R3AstHumanizer) VisitReference(reference *Reference) interface{} {
	h.Result = append(h.Result, []any{"Reference", reference.Name, reference.Value})
	return nil
}

func (h *R3AstHumanizer) VisitTextAttribute(attribute *TextAttribute) interface{} {
	h.Result = append(h.Result, []any{"TextAttribute", attribute.Name, attribute.Value})
	return nil
}

func (h *R3AstHumanizer) VisitBoundAttribute(attribute *BoundAttribute) interface{} {
	var valStr string
	if attribute.Value != nil {
		valStr = expression_parser.Serialize(attribute.Value)
	}
	h.Result = append(h.Result, []any{"BoundAttribute", mapBindingTypeToTs(attribute.Type), attribute.Name, valStr})
	return nil
}

func (h *R3AstHumanizer) VisitBoundEvent(event *BoundEvent) interface{} {
	var target interface{} = nil
	if event.Target != nil && *event.Target != "" {
		target = *event.Target
	}
	var handlerStr string
	if event.Handler != nil {
		handlerStr = expression_parser.Serialize(event.Handler)
	}
	h.Result = append(h.Result, []any{"BoundEvent", mapParsedEventTypeToTs(event.Type), event.Name, target, handlerStr})
	return nil
}

func (h *R3AstHumanizer) VisitText(text *Text) interface{} {
	h.Result = append(h.Result, []any{"Text", text.Value})
	return nil
}

func (h *R3AstHumanizer) VisitBoundText(text *BoundText) interface{} {
	var valStr string
	if text.Value != nil {
		valStr = expression_parser.Serialize(text.Value)
	}
	h.Result = append(h.Result, []any{"BoundText", valStr})
	return nil
}

func (h *R3AstHumanizer) VisitIcu(icu *Icu) interface{} {
	return nil
}

func (h *R3AstHumanizer) VisitDeferredBlock(deferred *DeferredBlock) interface{} {
	h.Result = append(h.Result, []any{"DeferredBlock"})
	deferred.VisitAll(h)
	return nil
}

func (h *R3AstHumanizer) VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) interface{} {
	result := []any{"DeferredBlockPlaceholder"}
	if block.MinimumTime != nil {
		result = append(result, fmt.Sprintf("minimum %gms", *block.MinimumTime))
	}
	h.Result = append(h.Result, result)
	h.visitAllElements(block.Children)
	return nil
}

func (h *R3AstHumanizer) VisitDeferredBlockLoading(block *DeferredBlockLoading) interface{} {
	result := []any{"DeferredBlockLoading"}
	if block.AfterTime != nil {
		result = append(result, fmt.Sprintf("after %gms", *block.AfterTime))
	}
	if block.MinimumTime != nil {
		result = append(result, fmt.Sprintf("minimum %gms", *block.MinimumTime))
	}
	h.Result = append(h.Result, result)
	h.visitAllElements(block.Children)
	return nil
}

func (h *R3AstHumanizer) VisitDeferredBlockError(block *DeferredBlockError) interface{} {
	h.Result = append(h.Result, []any{"DeferredBlockError"})
	h.visitAllElements(block.Children)
	return nil
}

func (h *R3AstHumanizer) VisitDeferredTrigger(trigger DeferredTrigger) interface{} {
	switch t := trigger.(type) {
	case *BoundDeferredTrigger:
		var valStr string
		if t.Value != nil {
			valStr = expression_parser.Serialize(t.Value)
		}
		h.Result = append(h.Result, []any{"BoundDeferredTrigger", valStr})
	case *ImmediateDeferredTrigger:
		h.Result = append(h.Result, []any{"ImmediateDeferredTrigger"})
	case *HoverDeferredTrigger:
		var ref interface{} = nil
		if t.Reference != nil {
			ref = *t.Reference
		}
		h.Result = append(h.Result, []any{"HoverDeferredTrigger", ref})
	case *IdleDeferredTrigger:
		if t.Timeout != nil {
			h.Result = append(h.Result, []any{"IdleDeferredTrigger", *t.Timeout})
		} else {
			h.Result = append(h.Result, []any{"IdleDeferredTrigger"})
		}
	case *TimerDeferredTrigger:
		h.Result = append(h.Result, []any{"TimerDeferredTrigger", t.Delay})
	case *InteractionDeferredTrigger:
		var ref interface{} = nil
		if t.Reference != nil {
			ref = *t.Reference
		}
		h.Result = append(h.Result, []any{"InteractionDeferredTrigger", ref})
	case *ViewportDeferredTrigger:
		var ref interface{} = nil
		if t.Reference != nil {
			ref = *t.Reference
		}
		h.Result = append(h.Result, []any{"ViewportDeferredTrigger", ref})
	case *NeverDeferredTrigger:
		h.Result = append(h.Result, []any{"NeverDeferredTrigger"})
	}
	return nil
}

func (h *R3AstHumanizer) VisitSwitchBlock(block *SwitchBlock) interface{} {
	var exprStr string
	if block.Expression != nil {
		exprStr = expression_parser.Serialize(block.Expression)
	}
	h.Result = append(h.Result, []any{"SwitchBlock", exprStr})
	for _, g := range block.Groups {
		if g != nil {
			g.Visit(h)
		}
	}
	if block.ExhaustiveCheck != nil {
		block.ExhaustiveCheck.Visit(h)
	}
	return nil
}

func (h *R3AstHumanizer) VisitSwitchBlockCase(block *SwitchBlockCase) interface{} {
	var expr interface{} = nil
	if block.Expression != nil {
		expr = expression_parser.Serialize(block.Expression)
	}
	h.Result = append(h.Result, []any{"SwitchBlockCase", expr})
	return nil
}

func (h *R3AstHumanizer) VisitSwitchBlockCaseGroup(block *SwitchBlockCaseGroup) interface{} {
	h.Result = append(h.Result, []any{"SwitchBlockCaseGroup"})
	for _, c := range block.Cases {
		if c != nil {
			c.Visit(h)
		}
	}
	h.visitAllElements(block.Children)
	return nil
}

func (h *R3AstHumanizer) VisitSwitchExhaustiveCheck(block *SwitchExhaustiveCheck) interface{} {
	h.Result = append(h.Result, []any{"SwitchExhaustiveCheck"})
	return nil
}

func (h *R3AstHumanizer) VisitForLoopBlock(block *ForLoopBlock) interface{} {
	var exprStr string
	if block.Expression.Ast != nil {
		exprStr = expression_parser.Serialize(block.Expression.Ast)
	}
	res := []any{"ForLoopBlock", exprStr}
	if block.TrackBy != nil && block.TrackBy.Ast != nil {
		res = append(res, expression_parser.Serialize(block.TrackBy.Ast))
	}
	h.Result = append(h.Result, res)
	if block.Item != nil {
		block.Item.Visit(h)
	}
	h.visitAllVariables(block.ContextVariables)
	h.visitAllElements(block.Children)
	if block.Empty != nil {
		block.Empty.Visit(h)
	}
	return nil
}

func (h *R3AstHumanizer) VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) interface{} {
	h.Result = append(h.Result, []any{"ForLoopBlockEmpty"})
	h.visitAllElements(block.Children)
	return nil
}

func (h *R3AstHumanizer) VisitIfBlock(block *IfBlock) interface{} {
	h.Result = append(h.Result, []any{"IfBlock"})
	for _, branch := range block.Branches {
		if branch != nil {
			branch.Visit(h)
		}
	}
	return nil
}

func (h *R3AstHumanizer) VisitIfBlockBranch(block *IfBlockBranch) interface{} {
	var expr interface{} = nil
	if block.Expression != nil {
		expr = expression_parser.Serialize(block.Expression)
	}
	h.Result = append(h.Result, []any{"IfBlockBranch", expr})
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(h)
	}
	h.visitAllElements(block.Children)
	return nil
}

func (h *R3AstHumanizer) VisitUnknownBlock(block *UnknownBlock) interface{} {
	h.Result = append(h.Result, []any{"UnknownBlock", block.Name})
	return nil
}

func (h *R3AstHumanizer) VisitLetDeclaration(decl *LetDeclaration) interface{} {
	var valStr string
	if decl.Value != nil {
		valStr = expression_parser.Serialize(decl.Value)
	}
	h.Result = append(h.Result, []any{"LetDeclaration", decl.Name, valStr})
	return nil
}

func (h *R3AstHumanizer) VisitComponent(component *Component) interface{} {
	var tagName string
	if component.TagName != nil {
		tagName = *component.TagName
	}
	res := []any{"Component", component.ComponentName, tagName, component.FullName}
	if component.IsSelfClosing {
		res = append(res, "#selfClosing")
	}
	h.Result = append(h.Result, res)
	h.visitAllAttributes(component.Attributes)
	h.visitAllBoundAttributes(component.Inputs)
	h.visitAllBoundEvents(component.Outputs)
	h.visitAllDirectives(component.Directives)
	h.visitAllReferences(component.References)
	h.visitAllElements(component.Children)
	return nil
}

func (h *R3AstHumanizer) VisitDirective(directive *Directive) interface{} {
	h.Result = append(h.Result, []any{"Directive", directive.Name})
	h.visitAllAttributes(directive.Attributes)
	h.visitAllBoundAttributes(directive.Inputs)
	h.visitAllBoundEvents(directive.Outputs)
	h.visitAllReferences(directive.References)
	return nil
}

func assertTransform(t *testing.T, html string, expected [][]any, options ...ParseR3Options) {
	t.Helper()
	opt := ParseR3Options{}
	if len(options) > 0 {
		opt = options[0]
	}
	res := parseR3(html, opt)
	humanizer := &R3AstHumanizer{}
	VisitAll(humanizer, res.Nodes)
	assert.Equal(t, expected, humanizer.Result)
}

func TestTemplateTransform(t *testing.T) {
	t.Run("ParseSpan on nodes toString", func(t *testing.T) {
		res := parseR3("<div></div>", ParseR3Options{})
		span := res.Nodes[0].GetSourceSpan()
		assert.Equal(t, "<div></div>", (&span).ToString())
	})

	t.Run("Nodes without binding", func(t *testing.T) {
		t.Run("should parse incomplete tags terminated by EOF", func(t *testing.T) {
			assertTransform(t, "<a", [][]any{
				{"Element", "a"},
			}, ParseR3Options{IgnoreError: true})
		})

		t.Run("should parse incomplete tags terminated by another tag", func(t *testing.T) {
			assertTransform(t, "<a <span></span>", [][]any{
				{"Element", "a"},
				{"Element", "span"},
			}, ParseR3Options{IgnoreError: true})
		})

		t.Run("should parse text nodes", func(t *testing.T) {
			assertTransform(t, "a", [][]any{
				{"BoundText", ""},
			})
		})

		t.Run("should parse elements with attributes", func(t *testing.T) {
			assertTransform(t, "<div a=b></div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "a", "b"},
			})
		})

		t.Run("should parse ngContent", func(t *testing.T) {
			assertTransform(t, "<ng-content select=\"a\"></ng-content>", [][]any{
				{"Content", ""},
				{"TextAttribute", "select", "a"},
			})
		})

		t.Run("should parse ngContent when it contains WS only", func(t *testing.T) {
			assertTransform(t, "<ng-content select=\"a\">    \n   </ng-content>", [][]any{
				{"Content", ""},
				{"TextAttribute", "select", "a"},
			})
		})

		t.Run("should parse ngContent regardless the namespace", func(t *testing.T) {
			assertTransform(t, "<svg><ng-content select=\"a\"></ng-content></svg>", [][]any{
				{"Element", ":svg:svg"},
				{"Content", ""},
				{"TextAttribute", "select", "a"},
			})
		})

		t.Run("should indicate whether an element is void", func(t *testing.T) {
			res := parseR3("<input><div></div>", ParseR3Options{})
			assert.Len(t, res.Nodes, 2)
			inputNode := res.Nodes[0].(*Element)
			divNode := res.Nodes[1].(*Element)
			assert.Equal(t, "input", inputNode.Name)
			assert.True(t, inputNode.IsVoid)
			assert.Equal(t, "div", divNode.Name)
			assert.False(t, divNode.IsVoid)
		})
	})

	t.Run("Bound text nodes", func(t *testing.T) {
		assertTransform(t, "{{a}}", [][]any{
			{"BoundText", "a"},
		})
	})

	t.Run("Bound attributes", func(t *testing.T) {
		t.Run("should parse mixed case bound properties", func(t *testing.T) {
			assertTransform(t, "<div [someProp]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 0, "someProp", "v"},
			})
		})

		t.Run("should parse bound properties via bind-", func(t *testing.T) {
			assertTransform(t, "<div bind-prop=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 0, "prop", "v"},
			})
		})

		t.Run("should report missing property names in bind- syntax", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div bind-></div>", ParseR3Options{})
			})
		})

		t.Run("should parse bound properties via {{...}}", func(t *testing.T) {
			assertTransform(t, "<div prop=\"{{v}}\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 0, "prop", "v"},
			})
		})

		t.Run("should parse dash case bound properties", func(t *testing.T) {
			assertTransform(t, "<div [some-prop]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 0, "some-prop", "v"},
			})
		})

		t.Run("should parse dotted name bound properties", func(t *testing.T) {
			assertTransform(t, "<div [d.ot]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 0, "d.ot", "v"},
			})
		})

		t.Run("should not normalize property names via the element schema", func(t *testing.T) {
			assertTransform(t, "<div [mappedAttr]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 0, "mappedAttr", "v"},
			})
		})

		t.Run("should parse mixed case bound attributes", func(t *testing.T) {
			assertTransform(t, "<div [attr.someAttr]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 1, "someAttr", "v"},
			})
		})

		t.Run("should parse and dash case bound classes", func(t *testing.T) {
			assertTransform(t, "<div [class.some-class]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 2, "some-class", "v"},
			})
		})

		t.Run("should parse mixed case bound classes", func(t *testing.T) {
			assertTransform(t, "<div [class.someClass]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 2, "someClass", "v"},
			})
		})

		t.Run("should parse mixed case bound styles", func(t *testing.T) {
			assertTransform(t, "<div [style.someStyle]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 3, "someStyle", "v"},
			})
		})

		t.Run("should parse class bindings with various characters", func(t *testing.T) {
			assertTransform(t, "<foo [class.text-primary/80]=\"expr\" [class.data-active:text-green-300/80]=\"expr2\" [class.data-[size='large']:p-8]=\"expr3\" some-attr/>", [][]any{
				{"Element", "foo", "#selfClosing"},
				{"TextAttribute", "some-attr", ""},
				{"BoundAttribute", 2, "text-primary/80", "expr"},
				{"BoundAttribute", 2, "data-active:text-green-300/80", "expr2"},
				{"BoundAttribute", 2, "data-[size='large']:p-8", "expr3"},
			})
		})
	})

	t.Run("animation bindings", func(t *testing.T) {
		t.Run("should support animate.enter", func(t *testing.T) {
			assertTransform(t, "<div animate.enter=\"foo\"></div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "animate.enter", "foo"},
			})

			assertTransform(t, "<div [animate.enter]=\"['foo', 'bar']\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 6, "animate.enter", `['foo', 'bar']`},
			})

			assertTransform(t, "<div (animate.enter)=\"animateFn($event)\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 3, "animate.enter", nil, "animateFn($event)"},
			})
		})

		t.Run("should support animate.leave", func(t *testing.T) {
			assertTransform(t, "<div animate.leave=\"foo\"></div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "animate.leave", "foo"},
			})

			assertTransform(t, "<div [animate.leave]=\"['foo', 'bar']\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 6, "animate.leave", `['foo', 'bar']`},
			})

			assertTransform(t, "<div (animate.leave)=\"animateFn($event)\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 3, "animate.leave", nil, "animateFn($event)"},
			})

			assertTransform(t, "<div (animateXYZ)=\"animateFn()\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 0, "animateXYZ", nil, "animateFn()"},
			})
		})
	})

	t.Run("templates", func(t *testing.T) {
		t.Run("should support * directives", func(t *testing.T) {
			assertTransform(t, "<div *ngIf></div>", [][]any{
				{"Template"},
				{"TextAttribute", "ngIf", ""},
				{"Element", "div"},
			})
		})

		t.Run("should support <ng-template>", func(t *testing.T) {
			assertTransform(t, "<ng-template></ng-template>", [][]any{
				{"Template"},
			})
		})

		t.Run("should support <ng-template> regardless the namespace", func(t *testing.T) {
			assertTransform(t, "<svg><ng-template></ng-template></svg>", [][]any{
				{"Element", ":svg:svg"},
				{"Template"},
			})
		})

		t.Run("should support <ng-template> with structural directive", func(t *testing.T) {
			assertTransform(t, "<ng-template *ngIf=\"true\"></ng-template>", [][]any{
				{"Template"},
				{"BoundAttribute", 0, "ngIf", "true"},
				{"Template"},
			})
		})

		t.Run("should support reference via #...", func(t *testing.T) {
			assertTransform(t, "<ng-template #a></ng-template>", [][]any{
				{"Template"},
				{"Reference", "a", ""},
			})
		})

		t.Run("should support reference via ref-...", func(t *testing.T) {
			assertTransform(t, "<ng-template ref-a></ng-template>", [][]any{
				{"Template"},
				{"Reference", "a", ""},
			})
		})

		t.Run("should report an error if a reference is used multiple times on the same template", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<ng-template #a #a></ng-template>", ParseR3Options{})
			})
		})

		t.Run("should parse variables via let-...", func(t *testing.T) {
			assertTransform(t, "<ng-template let-a=\"b\"></ng-template>", [][]any{
				{"Template"},
				{"Variable", "a", "b"},
			})
		})

		t.Run("should parse attributes", func(t *testing.T) {
			assertTransform(t, "<ng-template k1=\"v1\" k2=\"v2\"></ng-template>", [][]any{
				{"Template"},
				{"TextAttribute", "k1", "v1"},
				{"TextAttribute", "k2", "v2"},
			})
		})

		t.Run("should parse bound attributes", func(t *testing.T) {
			assertTransform(t, "<ng-template [k1]=\"v1\" [k2]=\"v2\"></ng-template>", [][]any{
				{"Template"},
				{"BoundAttribute", 0, "k1", "v1"},
				{"BoundAttribute", 0, "k2", "v2"},
			})
		})
	})

	t.Run("inline templates", func(t *testing.T) {
		t.Run("should support attribute and bound attributes", func(t *testing.T) {
			assertTransform(t, "<div *ngFor=\"let item of items\"></div>", [][]any{
				{"Template"},
				{"TextAttribute", "ngFor", ""},
				{"BoundAttribute", 0, "ngForOf", "items"},
				{"Variable", "item", "$implicit"},
				{"Element", "div"},
			})

			assertTransform(t, "<div *ngFor=\"item of items\"></div>", [][]any{
				{"Template"},
				{"BoundAttribute", 0, "ngFor", "item"},
				{"BoundAttribute", 0, "ngForOf", "items"},
				{"Element", "div"},
			})
		})

		t.Run("should parse variables via let ...", func(t *testing.T) {
			assertTransform(t, "<div *ngIf=\"let a=b\"></div>", [][]any{
				{"Template"},
				{"TextAttribute", "ngIf", ""},
				{"Variable", "a", "b"},
				{"Element", "div"},
			})
		})

		t.Run("should parse variables via as ...", func(t *testing.T) {
			assertTransform(t, "<div *ngIf=\"expr as local\"></div>", [][]any{
				{"Template"},
				{"BoundAttribute", 0, "ngIf", "expr"},
				{"Variable", "local", "ngIf"},
				{"Element", "div"},
			})
		})
	})

	t.Run("events", func(t *testing.T) {
		t.Run("should parse bound events with a target", func(t *testing.T) {
			assertTransform(t, "<div (window:event)=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 0, "event", "window", "v"},
			})
		})

		t.Run("should parse event names case sensitive", func(t *testing.T) {
			assertTransform(t, "<div (some-event)=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 0, "some-event", nil, "v"},
			})

			assertTransform(t, "<div (someEvent)=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 0, "someEvent", nil, "v"},
			})
		})

		t.Run("should parse bound events via on-", func(t *testing.T) {
			assertTransform(t, "<div on-event=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 0, "event", nil, "v"},
			})
		})

		t.Run("should report missing event names in on- syntax", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div on-></div>", ParseR3Options{})
			})
		})

		t.Run("should parse bound events and properties via [(...)]", func(t *testing.T) {
			assertTransform(t, "<div [(prop)]=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 5, "prop", "v"},
				{"BoundEvent", 2, "propChange", nil, "v"},
			})
		})

		t.Run("should parse $any in a two-way binding", func(t *testing.T) {
			assertTransform(t, "<div [(prop)]=\"$any(v)\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 5, "prop", "$any(v)"},
				{"BoundEvent", 2, "propChange", nil, "$any(v)"},
			})
		})

		t.Run("should parse bound events and properties via bindon-", func(t *testing.T) {
			assertTransform(t, "<div bindon-prop=\"v\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 5, "prop", "v"},
				{"BoundEvent", 2, "propChange", nil, "v"},
			})
		})

		t.Run("should parse bound events and properties via [(...)] with non-null operator", func(t *testing.T) {
			assertTransform(t, "<div [(prop)]=\"v!\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 5, "prop", "v!"},
				{"BoundEvent", 2, "propChange", nil, "v!"},
			})
		})

		t.Run("should parse property reads bound via [(...)]", func(t *testing.T) {
			assertTransform(t, "<div [(prop)]=\"a.b.c\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 5, "prop", "a.b.c"},
				{"BoundEvent", 2, "propChange", nil, "a.b.c"},
			})
		})

		t.Run("should parse keyed reads bound via [(...)]", func(t *testing.T) {
			assertTransform(t, "<div [(prop)]=\"a['b']['c']\"></div>", [][]any{
				{"Element", "div"},
				{"BoundAttribute", 5, "prop", `a['b']['c']`},
				{"BoundEvent", 2, "propChange", nil, `a['b']['c']`},
			})
		})

		t.Run("should report assignments in two-way bindings", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div [(prop)]=\"v = 1\"></div>", ParseR3Options{})
			})
		})

		t.Run("should report pipes in two-way bindings", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div [(prop)]=\"v | pipe\"></div>", ParseR3Options{})
			})
		})

		t.Run("should report unsupported expressions in two-way bindings", func(t *testing.T) {
			unsupported := []string{
				"v + 1", "foo.bar?.baz", "foo.bar?.['baz']", "foo?.bar.baz[0]",
				"foo?.bar.baz[0].boo[0]", "foo?.bar.baz()", "(foo?.bar).baz", "true",
				"123", "a.b()", "v()", "[1, 2, 3]", "{a: 1, b: 2, c: 3}", "v === 1",
				"a || b", "a && b", "a ?? b", "!a", "!!a", "a ? b : c", "$any(a || b)",
				"this.$any(a)", "$any(a, b)",
			}
			for _, expr := range unsupported {
				assert.Panics(t, func() {
					parseR3(fmt.Sprintf("<div [(prop)]=\"%s\"></div>", expr), ParseR3Options{})
				}, "expected unsupported expression error for: %s", expr)
			}
		})

		t.Run("should report missing property names in bindon- syntax", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div bindon-></div>", ParseR3Options{})
			})
		})

		t.Run("should report an error on empty expression", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div (event)=\"\">", ParseR3Options{})
			})
			assert.Panics(t, func() {
				parseR3("<div (event)=\"   \">", ParseR3Options{})
			})
		})

		t.Run("should parse bound animation events when event name is empty", func(t *testing.T) {
			assertTransform(t, "<div (@)=\"onAnimationEvent($event)\"></div>", [][]any{
				{"Element", "div"},
				{"BoundEvent", 1, "", nil, "onAnimationEvent($event)"},
			}, ParseR3Options{IgnoreError: true})

			assert.Panics(t, func() {
				parseR3("<div (@)></div>", ParseR3Options{})
			})
		})

		t.Run("should report invalid phase value of animation event", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div (@event.invalidPhase)></div>", ParseR3Options{})
			})
			assert.Panics(t, func() {
				parseR3("<div (@event.)></div>", ParseR3Options{})
			})
			assert.Panics(t, func() {
				parseR3("<div (@event)></div>", ParseR3Options{})
			})
		})
	})

	t.Run("variables", func(t *testing.T) {
		t.Run("should report variables not on template elements", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div let-a-name=\"b\"></div>", ParseR3Options{})
			})
		})

		t.Run("should report missing variable names", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<ng-template let-><ng-template>", ParseR3Options{})
			})
		})
	})

	t.Run("references", func(t *testing.T) {
		t.Run("should parse references via #...", func(t *testing.T) {
			assertTransform(t, "<div #a></div>", [][]any{
				{"Element", "div"},
				{"Reference", "a", ""},
			})
		})

		t.Run("should parse references via ref-", func(t *testing.T) {
			assertTransform(t, "<div ref-a></div>", [][]any{
				{"Element", "div"},
				{"Reference", "a", ""},
			})
		})

		t.Run("should parse camel case references", func(t *testing.T) {
			assertTransform(t, "<div #someA></div>", [][]any{
				{"Element", "div"},
				{"Reference", "someA", ""},
			})
		})

		t.Run("should report invalid reference names", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div #a-b></div>", ParseR3Options{})
			})
		})

		t.Run("should report missing reference names", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div #></div>", ParseR3Options{})
			})
		})

		t.Run("should report an error if a reference is used multiple times on the same element", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div #a #a></div>", ParseR3Options{})
			})
		})
	})

	t.Run("literal attribute", func(t *testing.T) {
		t.Run("should report missing animation trigger in @ syntax", func(t *testing.T) {
			assert.Panics(t, func() {
				parseR3("<div @></div>", ParseR3Options{})
			})
		})
	})

	t.Run("ng-content", func(t *testing.T) {
		t.Run("should parse ngContent without selector", func(t *testing.T) {
			assertTransform(t, "<ng-content></ng-content>", [][]any{
				{"Content", ""},
			})
		})

		t.Run("should parse ngContent with a specific selector", func(t *testing.T) {
			assertTransform(t, "<ng-content select=\"tag[attribute]\"></ng-content>", [][]any{
				{"Content", ""},
				{"TextAttribute", "select", "tag[attribute]"},
			})
		})

		t.Run("should parse ngContent with a selector", func(t *testing.T) {
			assertTransform(t, "<ng-content select=\"a\"></ng-content><ng-content></ng-content><ng-content select=\"b\"></ng-content>", [][]any{
				{"Content", ""},
				{"TextAttribute", "select", "a"},
				{"Content", ""},
				{"Content", ""},
				{"TextAttribute", "select", "b"},
			})
		})

		t.Run("should parse ngProjectAs as an attribute", func(t *testing.T) {
			assertTransform(t, "<ng-content ngProjectAs=\"a\"></ng-content>", [][]any{
				{"Content", ""},
				{"TextAttribute", "ngProjectAs", "a"},
			})
		})

		t.Run("should parse ngContent with children", func(t *testing.T) {
			assertTransform(t, "<ng-content><section>Root <div>Parent <span>Child</span></div></section></ng-content>", [][]any{
				{"Content", ""},
				{"Element", "section"},
				{"BoundText", ""},
				{"Element", "div"},
				{"BoundText", ""},
				{"Element", "span"},
				{"BoundText", ""},
			})
		})
	})

	t.Run("parser errors", func(t *testing.T) {
		t.Run("should only report errors on the node on which the error occurred", func(t *testing.T) {
			res := parseR3(`
        <input (input)="foo(12#3)">
        <button (click)="bar()"></button>
        <span (mousedown)="baz()"></span>
      `, ParseR3Options{IgnoreError: true})
			assert.Len(t, res.Errors, 3)
			assert.Contains(t, res.Errors[0].Msg, "Parser Error: Missing expected )")
			assert.Contains(t, res.Errors[1].Msg, "Invalid character [#]")
			assert.Contains(t, res.Errors[2].Msg, "Unexpected token ')'")
		})

		t.Run("should report parsing errors on specific interpolated expressions", func(t *testing.T) {
			res := parseR3(`
          bunch of text bunch of text bunch of text bunch of text bunch of text bunch of text
          bunch of text bunch of text bunch of text bunch of text
 
          {{foo[0}} bunch of text bunch of text bunch of text bunch of text {{.bar}}
 
          bunch of text
          bunch of text
          bunch of text
          bunch of text
          bunch of text {{one + #two + baz}}
        `, ParseR3Options{IgnoreError: true})
			assert.Len(t, res.Errors, 3)
			assert.Equal(t, "{{foo[0}}", res.Errors[0].Span.ToString())
			assert.Equal(t, "{{.bar}}", res.Errors[1].Span.ToString())
			assert.Equal(t, "{{one + #two + baz}}", res.Errors[2].Span.ToString())
		})
	})

	t.Run("Ignored elements", func(t *testing.T) {
		t.Run("should ignore <script> elements", func(t *testing.T) {
			assertTransform(t, "<script></script>a", [][]any{
				{"BoundText", ""},
			})
		})

		t.Run("should ignore <style> elements", func(t *testing.T) {
			assertTransform(t, "<style></style>a", [][]any{
				{"BoundText", ""},
			})
		})
	})

	t.Run("link rel=stylesheet", func(t *testing.T) {
		t.Run("should keep link elements if they have an absolute url", func(t *testing.T) {
			assertTransform(t, "<link rel=\"stylesheet\" href=\"http://someurl\">", nil)
			assertTransform(t, "<link REL=\"stylesheet\" href=\"http://someurl\">", nil)
		})

		t.Run("should keep link elements if they have no uri", func(t *testing.T) {
			assertTransform(t, "<link rel=\"stylesheet\">", [][]any{
				{"Element", "link"},
				{"TextAttribute", "rel", "stylesheet"},
			})
			assertTransform(t, "<link REL=\"stylesheet\">", [][]any{
				{"Element", "link"},
				{"TextAttribute", "REL", "stylesheet"},
			})
		})

		t.Run("should ignore link elements if they have a relative uri", func(t *testing.T) {
			assertTransform(t, "<link rel=\"stylesheet\" href=\"./other.css\">", nil)
			assertTransform(t, "<link REL=\"stylesheet\" HREF=\"./other.css\">", nil)
		})
	})

	t.Run("ngNonBindable", func(t *testing.T) {
		t.Run("should ignore bindings on children of elements with ngNonBindable", func(t *testing.T) {
			assertTransform(t, "<div ngNonBindable>{{b}}</div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "ngNonBindable", ""},
				{"Text", "{{b}}"},
			})
		})

		t.Run("should keep nested children of elements with ngNonBindable", func(t *testing.T) {
			assertTransform(t, "<div ngNonBindable><span>{{b}}</span></div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "ngNonBindable", ""},
				{"Element", "span"},
				{"Text", "{{b}}"},
			})
		})

		t.Run("should ignore <script> elements inside of elements with ngNonBindable", func(t *testing.T) {
			assertTransform(t, "<div ngNonBindable><script></script>a</div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "ngNonBindable", ""},
				{"Text", "a"},
			})
		})

		t.Run("should ignore <style> elements inside of elements with ngNonBindable", func(t *testing.T) {
			assertTransform(t, "<div ngNonBindable><style></style>a</div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "ngNonBindable", ""},
				{"Text", "a"},
			})
		})

		t.Run("should ignore <link rel=stylesheet> elements inside of elements with ngNonBindable", func(t *testing.T) {
			assertTransform(t, "<div ngNonBindable><link rel=\"stylesheet\">a</div>", [][]any{
				{"Element", "div"},
				{"TextAttribute", "ngNonBindable", ""},
				{"Text", "a"},
			})
		})
	})

	t.Run("deferred blocks", func(t *testing.T) {
		t.Run("should parse a simple deferred block", func(t *testing.T) {
			assertTransform(t, "@defer{hello}", [][]any{
				{"DeferredBlock"},
				{"BoundText", ""},
			})
		})

		t.Run("should parse a deferred block with triggers", func(t *testing.T) {
			assertTransform(t, "@defer (when isVisible() && loaded){hello}", [][]any{
				{"DeferredBlock"},
				{"BoundDeferredTrigger", "isVisible() && loaded"},
				{"BoundText", ""},
			})
		})
	})

	t.Run("switch blocks", func(t *testing.T) {
		t.Run("should parse a switch block", func(t *testing.T) {
			assertTransform(t, `
          @switch (cond.kind) {
            @case (x()) { X case }
            @case ('hello') {<button>Y case</button>}
            @case (42) { Z case }
            @default { No case matched }
          }
        `, [][]any{
				{"SwitchBlock", "cond.kind"},
				{"SwitchBlockCaseGroup"},
				{"SwitchBlockCase", "x()"},
				{"BoundText", ""},
				{"SwitchBlockCaseGroup"},
				{"SwitchBlockCase", "'hello'"},
				{"Element", "button"},
				{"BoundText", ""},
				{"SwitchBlockCaseGroup"},
				{"SwitchBlockCase", "42"},
				{"BoundText", ""},
				{"SwitchBlockCaseGroup"},
				{"SwitchBlockCase", nil},
				{"BoundText", ""},
			})
		})

		t.Run("should parse a switch block with a default never case", func(t *testing.T) {
			assertTransform(t, `
          @switch (cond.kind) {
            @default never;
          }
        `, [][]any{
				{"SwitchBlock", "cond.kind"},
				{"SwitchExhaustiveCheck"},
			})
		})
	})

	t.Run("for loop blocks", func(t *testing.T) {
		t.Run("should parse a for loop block", func(t *testing.T) {
			assertTransform(t, `
        @for (item of items.foo.bar; track item.id) {
          {{ item }}
        } @empty {
          There were no items in the list.
        }
      `, [][]any{
				{"ForLoopBlock", "items.foo.bar", "item.id"},
				{"Variable", "item", "$implicit"},
				{"Variable", "$index", "$index"},
				{"Variable", "$first", "$first"},
				{"Variable", "$last", "$last"},
				{"Variable", "$even", "$even"},
				{"Variable", "$odd", "$odd"},
				{"Variable", "$count", "$count"},
				{"BoundText", " item "},
				{"ForLoopBlockEmpty"},
				{"BoundText", ""},
			})
		})
	})

	t.Run("if blocks", func(t *testing.T) {
		t.Run("should parse a simple if block", func(t *testing.T) {
			assertTransform(t, `
        @if (expr) {
          hello
        }
      `, [][]any{
				{"IfBlock"},
				{"IfBlockBranch", "expr"},
				{"BoundText", ""},
			})
		})
	})

	t.Run("@let declarations", func(t *testing.T) {
		t.Run("should parse a let declaration", func(t *testing.T) {
			assertTransform(t, "@let a = 1;", [][]any{
				{"LetDeclaration", "a", "1"},
			})
		})
	})
}
