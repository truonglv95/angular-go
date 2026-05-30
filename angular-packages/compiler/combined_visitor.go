package compiler

// Port of packages/compiler/src/combined_visitor.ts
//
// CombinedRecursiveAstVisitor traverses all template and expression AST nodes
// in a template. Useful for cases where every single node needs to be visited.
//
// In TypeScript this is:
//
//	class CombinedRecursiveAstVisitor extends RecursiveAstVisitor implements t.RecursiveVisitor
//
// In Go we embed both base visitor structs. Callers that want to override
// specific visit methods should embed *CombinedRecursiveAstVisitor and shadow
// the methods they care about, setting Impl on the embedded
// expression_parser.RecursiveAstVisitor and render3.RecursiveVisitor to themselves so
// recursive visits dispatch to the override.

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

// CombinedRecursiveAstVisitor combines:
//   - expression_parser.RecursiveAstVisitor  (visits expression/binding AST nodes)
//   - render3.RecursiveVisitor               (visits template AST nodes)
type CombinedRecursiveAstVisitor struct {
	expression_parser.RecursiveAstVisitor
	render3.RecursiveVisitor
}

func (v *CombinedRecursiveAstVisitor) r3Visitor() render3.Visitor {
	if v.RecursiveVisitor.Impl != nil {
		return v.RecursiveVisitor.Impl
	}
	return v
}

func (v *CombinedRecursiveAstVisitor) astVisitor() expression_parser.Visitor {
	if v.RecursiveAstVisitor.Impl != nil {
		return v.RecursiveAstVisitor.Impl
	}
	return v
}

// visit dispatches a node that is either an expression AST or a template AST node.
// Mirrors the TypeScript override of visit():
//
//	override visit(node: AST | t.Node): void {
//	  if (node instanceof ASTWithSource) { this.visit(node.ast); }
//	  else { node.visit(this); }
//	}
func (v *CombinedRecursiveAstVisitor) visitExpr(node expression_parser.AST) {
	if aws, ok := node.(*expression_parser.ASTWithSource); ok {
		v.visitExpr(aws.Ast)
		return
	}
	node.Visit(v.astVisitor(), nil)
}

// visitTemplateNode dispatches a render3 template AST node.
func (v *CombinedRecursiveAstVisitor) visitTemplateNode(node render3.Node) {
	node.Visit(v.r3Visitor())
}

// visitAllTemplateNodes visits every node in the slice.
// Mirrors: protected visitAllTemplateNodes(nodes: t.Node[]): void
func (v *CombinedRecursiveAstVisitor) visitAllTemplateNodes(nodes []render3.Node) {
	for _, node := range nodes {
		v.visitTemplateNode(node)
	}
}

// ---- render3.Visitor overrides ----
// Each method below matches the corresponding visitXxx() in the TypeScript class.

func (v *CombinedRecursiveAstVisitor) VisitElement(element *render3.Element) interface{} {
	vis := v.r3Visitor()
	for _, n := range element.Attributes {
		n.Visit(vis)
	}
	for _, n := range element.Inputs {
		n.Visit(vis)
	}
	for _, n := range element.Outputs {
		n.Visit(vis)
	}
	for _, n := range element.Directives {
		n.Visit(vis)
	}
	for _, n := range element.References {
		n.Visit(vis)
	}
	for _, n := range element.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitTemplate(template *render3.Template) interface{} {
	vis := v.r3Visitor()
	for _, n := range template.Attributes {
		n.Visit(vis)
	}
	for _, n := range template.Inputs {
		n.Visit(vis)
	}
	for _, n := range template.Outputs {
		n.Visit(vis)
	}
	for _, n := range template.Directives {
		n.Visit(vis)
	}
	for _, n := range template.TemplateAttrs {
		n.Visit(vis)
	}
	for _, n := range template.Variables {
		n.Visit(vis)
	}
	for _, n := range template.References {
		n.Visit(vis)
	}
	for _, n := range template.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitContent(content *render3.Content) interface{} {
	vis := v.r3Visitor()
	for _, n := range content.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitBoundAttribute(attribute *render3.BoundAttribute) interface{} {
	if attribute.Value != nil {
		v.visitExpr(attribute.Value)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitBoundEvent(event *render3.BoundEvent) interface{} {
	if event.Handler != nil {
		v.visitExpr(event.Handler)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitBoundText(text *render3.BoundText) interface{} {
	if text.Value != nil {
		v.visitExpr(text.Value)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitIcu(icu *render3.Icu) interface{} {
	vis := v.r3Visitor()
	for _, bt := range icu.Vars {
		bt.Visit(vis)
	}
	for _, node := range icu.Placeholders {
		node.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	deferred.VisitAll(v.r3Visitor())
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitDeferredTrigger(trigger render3.DeferredTrigger) interface{} {
	if bt, ok := trigger.(*render3.BoundDeferredTrigger); ok {
		v.visitExpr(bt.Value)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	vis := v.r3Visitor()
	for _, n := range block.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	vis := v.r3Visitor()
	for _, n := range block.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	vis := v.r3Visitor()
	for _, n := range block.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	v.visitExpr(block.Expression)
	vis := v.r3Visitor()
	for _, g := range block.Groups {
		g.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	if block.Expression != nil {
		v.visitExpr(block.Expression)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitSwitchBlockCaseGroup(block *render3.SwitchBlockCaseGroup) interface{} {
	vis := v.r3Visitor()
	for _, c := range block.Cases {
		c.Visit(vis)
	}
	for _, n := range block.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitSwitchExhaustiveCheck(_ *render3.SwitchExhaustiveCheck) interface{} {
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	vis := v.r3Visitor()
	if block.Item != nil {
		block.Item.Visit(vis)
	}
	for _, cv := range block.ContextVariables {
		cv.Visit(vis)
	}
	v.visitExpr(&block.Expression)
	for _, n := range block.Children {
		n.Visit(vis)
	}
	if block.Empty != nil {
		block.Empty.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	vis := v.r3Visitor()
	for _, n := range block.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitIfBlock(block *render3.IfBlock) interface{} {
	vis := v.r3Visitor()
	for _, branch := range block.Branches {
		branch.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	vis := v.r3Visitor()
	if block.Expression != nil {
		v.visitExpr(block.Expression)
	}
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(vis)
	}
	for _, n := range block.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	if decl.Value != nil {
		v.visitExpr(decl.Value)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitComponent(component *render3.Component) interface{} {
	vis := v.r3Visitor()
	for _, n := range component.Attributes {
		n.Visit(vis)
	}
	for _, n := range component.Inputs {
		n.Visit(vis)
	}
	for _, n := range component.Outputs {
		n.Visit(vis)
	}
	for _, n := range component.Directives {
		n.Visit(vis)
	}
	for _, n := range component.References {
		n.Visit(vis)
	}
	for _, n := range component.Children {
		n.Visit(vis)
	}
	return nil
}

func (v *CombinedRecursiveAstVisitor) VisitDirective(directive *render3.Directive) interface{} {
	vis := v.r3Visitor()
	for _, n := range directive.Attributes {
		n.Visit(vis)
	}
	for _, n := range directive.Inputs {
		n.Visit(vis)
	}
	for _, n := range directive.Outputs {
		n.Visit(vis)
	}
	for _, n := range directive.References {
		n.Visit(vis)
	}
	return nil
}

// No-op leaf visitors (matching TS: visitVariable, visitReference, visitTextAttribute, visitText, visitUnknownBlock)
func (v *CombinedRecursiveAstVisitor) VisitVariableExpr(_ *render3.Variable) interface{}        { return nil }
func (v *CombinedRecursiveAstVisitor) VisitReference(_ *render3.Reference) interface{}           { return nil }
func (v *CombinedRecursiveAstVisitor) VisitTextAttribute(_ *render3.TextAttribute) interface{}   { return nil }
func (v *CombinedRecursiveAstVisitor) VisitText(_ *render3.Text) interface{}                     { return nil }
func (v *CombinedRecursiveAstVisitor) VisitUnknownBlock(_ *render3.UnknownBlock) interface{}     { return nil }

// Visit satisfies the render3.Visitor interface (dispatches to the node).
func (v *CombinedRecursiveAstVisitor) Visit(node render3.Node) interface{} {
	return node.Visit(v)
}
