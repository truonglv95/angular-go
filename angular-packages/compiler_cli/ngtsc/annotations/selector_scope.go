package annotations

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type selectorScopeVisitor struct {
	matcher *render3.SelectorMatcher[*render3.R3TemplateDependency]
	matched map[*render3.R3TemplateDependency]bool
}

func newSelectorScopeVisitor(matcher *render3.SelectorMatcher[*render3.R3TemplateDependency]) *selectorScopeVisitor {
	return &selectorScopeVisitor{
		matcher: matcher,
		matched: make(map[*render3.R3TemplateDependency]bool),
	}
}

func (v *selectorScopeVisitor) Visit(node render3.Node) interface{} {
	return node.Visit(v)
}

func (v *selectorScopeVisitor) matchElement(name string, attrs []render3.Node) {
	selector := render3.NewCssSelector()
	selector.Element = &name
	for _, attrNode := range attrs {
		if textAttr, ok := attrNode.(*render3.TextAttribute); ok {
			if textAttr.Name == "class" {
				// parse classes
			} else {
				selector.Attrs = append(selector.Attrs, textAttr.Name, textAttr.Value)
			}
		} else if boundAttr, ok := attrNode.(*render3.BoundAttribute); ok {
			selector.Attrs = append(selector.Attrs, boundAttr.Name, "")
		} else if boundEvent, ok := attrNode.(*render3.BoundEvent); ok {
			selector.Attrs = append(selector.Attrs, boundEvent.Name, "")
		}
	}
	
	v.matcher.Match(selector, func(c *render3.CssSelector, dep *render3.R3TemplateDependency) {
		v.matched[dep] = true
	})
}

func (v *selectorScopeVisitor) VisitElement(element *render3.Element) interface{} {
	var attrs []render3.Node
	for _, attr := range element.Attributes {
		attrs = append(attrs, attr)
	}
	for _, attr := range element.Inputs {
		attrs = append(attrs, attr)
	}
	for _, attr := range element.Outputs {
		attrs = append(attrs, attr)
	}
	v.matchElement(element.Name, attrs)
	for _, child := range element.Children {
		child.Visit(v)
	}
	return nil
}

func (v *selectorScopeVisitor) VisitTemplate(template *render3.Template) interface{} {
	var attrs []render3.Node
	for _, attr := range template.TemplateAttrs {
		attrs = append(attrs, attr)
	}
	for _, attr := range template.Attributes {
		attrs = append(attrs, attr)
	}
	for _, attr := range template.Inputs {
		attrs = append(attrs, attr)
	}
	for _, attr := range template.Outputs {
		attrs = append(attrs, attr)
	}
	name := ""
	if template.TagName != nil {
		name = *template.TagName
	}
	v.matchElement(name, attrs)
	for _, child := range template.Children {
		child.Visit(v)
	}
	return nil
}

func (v *selectorScopeVisitor) VisitContent(content *render3.Content) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitVariableExpr(variable *render3.Variable) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitComponent(component *render3.Component) interface{} {
	name := ""
	if component.TagName != nil {
		name = *component.TagName
	}
	v.matchElement(name, nil)
	for _, child := range component.Children {
		child.Visit(v)
	}
	return nil
}
func (v *selectorScopeVisitor) VisitDirective(directive *render3.Directive) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitSwitchExhaustiveCheck(block *render3.SwitchExhaustiveCheck) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	for _, child := range block.Children {
		child.Visit(v)
	}
	return nil
}
func (v *selectorScopeVisitor) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	for _, child := range block.Children {
		child.Visit(v)
	}
	return nil
}
func (v *selectorScopeVisitor) VisitIfBlock(block *render3.IfBlock) interface{} {
	for _, branch := range block.Branches {
		branch.Visit(v)
	}
	return nil
}
func (v *selectorScopeVisitor) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	for _, child := range block.Children {
		child.Visit(v)
	}
	return nil
}
func (v *selectorScopeVisitor) VisitUnknownBlock(block *render3.UnknownBlock) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitReference(reference *render3.Reference) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitTextAttribute(attribute *render3.TextAttribute) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitBoundAttribute(attribute *render3.BoundAttribute) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitBoundEvent(attribute *render3.BoundEvent) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitText(text *render3.Text) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitBoundText(text *render3.BoundText) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitIcu(icu *render3.Icu) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitDeferredTrigger(trigger render3.DeferredTrigger) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	return nil
}
func (v *selectorScopeVisitor) VisitSwitchBlockCaseGroup(block *render3.SwitchBlockCaseGroup) interface{} {
	return nil
}
