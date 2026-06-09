package indexer

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/internal/ast"
)

type astWithSource struct {
	source         *string
	absoluteOffset int
}

type templateVisitor struct {
	*compiler.CombinedRecursiveAstVisitor

	boundTemplate                AbstractBoundTemplate
	identifiers                  []TemplateIdentifier
	errors                       []error
	currentAstWithSource         *astWithSource
	targetIdentifierCache        map[any]TemplateIdentifier
	directiveHostIdentifierCache map[any]TemplateIdentifier
}

func (v *templateVisitor) addIdentifier(ident TemplateIdentifier) {
	for _, id := range v.identifiers {
		if id.Name == ident.Name && id.Span.Start == ident.Span.Start && id.Span.End == ident.Span.End && id.Kind == ident.Kind {
			return
		}
	}
	v.identifiers = append(v.identifiers, ident)
}

func (v *templateVisitor) visitExpr(expr expression_parser.AST) {
	if expr == nil {
		return
	}

	previous := v.currentAstWithSource
	var source *string
	absoluteOffset := -1
	if aws, ok := expr.(*expression_parser.ASTWithSource); ok {
		source = &aws.Source
		absoluteOffset = aws.GetSourceSpan().Start
		v.currentAstWithSource = &astWithSource{
			source:         source,
			absoluteOffset: absoluteOffset,
		}
		aws.Ast.Visit(v.RecursiveAstVisitor.Impl, nil)
	} else {
		expr.Visit(v.RecursiveAstVisitor.Impl, nil)
	}
	v.currentAstWithSource = previous
}

func (v *templateVisitor) directiveHostToIdentifier(node any) *TemplateIdentifier {
	if cached, ok := v.directiveHostIdentifierCache[node]; ok {
		return &cached
	}

	var name string
	var kind IdentifierKind
	var sourceSpan parse_util.ParseSourceSpan
	var attributes []*render3.TextAttribute

	switch n := node.(type) {
	case *render3.Template:
		if n.TagName != nil {
			name = *n.TagName
		} else {
			name = "ng-template"
		}
		kind = IdentifierKindTemplate
		sourceSpan = n.StartSourceSpan
		attributes = n.Attributes
	case *render3.Element:
		name = n.Name
		kind = IdentifierKindElement
		sourceSpan = n.StartSourceSpan
		attributes = n.Attributes
	case *render3.Component:
		name = n.FullName
		kind = IdentifierKindComponent
		sourceSpan = n.StartSourceSpan
		attributes = n.Attributes
	case *render3.Directive:
		name = n.Name
		kind = IdentifierKindDirective
		sourceSpan = n.StartSourceSpan
		attributes = n.Attributes
	default:
		return nil
	}

	switch node.(type) {
	case *render3.Template, *render3.Element:
		if strings.HasPrefix(name, ":") {
			parts := strings.Split(name, ":")
			name = parts[len(parts)-1]
		}
	}

	start := v.getStartLocation(name, sourceSpan)
	if start == nil {
		return nil
	}
	absoluteSpan := NewAbsoluteSourceSpan(*start, *start+len(name))

	var attrs []TemplateIdentifier
	for _, attr := range attributes {
		attrs = append(attrs, TemplateIdentifier{
			Name: attr.Name,
			Span: NewAbsoluteSourceSpan(attr.SourceSpan.Start.Offset, attr.SourceSpan.End.Offset),
			Kind: IdentifierKindAttribute,
		})
	}

	usedDirectives := v.boundTemplate.GetDirectivesOfNode(node)

	identifier := TemplateIdentifier{
		Name:           name,
		Span:           absoluteSpan,
		Kind:           kind,
		Attributes:     attrs,
		UsedDirectives: usedDirectives,
	}

	v.directiveHostIdentifierCache[node] = identifier
	return &identifier
}

func (v *templateVisitor) targetToIdentifier(node any) *TemplateIdentifier {
	if cached, ok := v.targetIdentifierCache[node]; ok {
		return &cached
	}

	var name string
	var sourceSpan parse_util.ParseSourceSpan

	switch n := node.(type) {
	case *render3.Reference:
		name = n.Name
		sourceSpan = n.SourceSpan
	case *render3.Variable:
		name = n.Name
		sourceSpan = n.SourceSpan
	case *render3.LetDeclaration:
		name = n.Name
		sourceSpan = n.SourceSpan
	default:
		return nil
	}

	start := v.getStartLocation(name, sourceSpan)
	if start == nil {
		return nil
	}

	span := NewAbsoluteSourceSpan(*start, *start+len(name))
	var identifier TemplateIdentifier

	switch n := node.(type) {
	case *render3.Reference:
		refTarget := v.boundTemplate.GetReferenceTarget(n)
		var refTargetStruct *ReferenceTarget
		if refTarget != nil {
			var targetNode *TemplateIdentifier
			var directive *ast.Node

			switch rt := refTarget.(type) {
			case *render3.Element, *render3.Template, *render3.Component, *render3.Directive:
				targetNode = v.directiveHostToIdentifier(rt)
			case BoundReferenceTarget:
				targetNode = v.directiveHostToIdentifier(rt.GetNode())
				directive = rt.GetDirectiveNode()
			case *ReferenceTarget:
				targetNode = rt.Node
				directive = rt.Directive
			case ReferenceTarget:
				targetNode = rt.Node
				directive = rt.Directive
			}

			if targetNode != nil {
				refTargetStruct = &ReferenceTarget{
					Node:      targetNode,
					Directive: directive,
				}
			}
		}

		identifier = TemplateIdentifier{
			Name:      name,
			Span:      span,
			Kind:      IdentifierKindReference,
			RefTarget: refTargetStruct,
		}
	case *render3.Variable:
		identifier = TemplateIdentifier{
			Name: name,
			Span: span,
			Kind: IdentifierKindVariable,
		}
	case *render3.LetDeclaration:
		identifier = TemplateIdentifier{
			Name: name,
			Span: span,
			Kind: IdentifierKindLetDeclaration,
		}
	}

	v.targetIdentifierCache[node] = identifier
	return &identifier
}

func (v *templateVisitor) getStartLocation(name string, context parse_util.ParseSourceSpan) *int {
	localStr := context.ToString()
	idx := strings.Index(localStr, name)
	if idx == -1 {
		v.errors = append(v.errors, fmt.Errorf("Impossible state: %q not found in %q", name, localStr))
		return nil
	}
	res := context.Start.Offset + idx
	return &res
}

func (v *templateVisitor) visitIdentifier(astNode *expression_parser.PropertyRead, kind IdentifierKind) {
	fmt.Printf("DEBUG: visitIdentifier Name=%s, currentAstWithSource=%+v\n", astNode.Name, v.currentAstWithSource)
	if v.currentAstWithSource == nil || v.currentAstWithSource.source == nil {
		fmt.Printf("DEBUG: early return nil source\n")
		return
	}
	if v.currentAstWithSource.source != nil {
		fmt.Printf("DEBUG: source=%q, offset=%d, NameSpan=%+v\n", *v.currentAstWithSource.source, v.currentAstWithSource.absoluteOffset, astNode.NameSpan)
	}

	switch astNode.Receiver.(type) {
	case *expression_parser.ImplicitReceiver, *expression_parser.ThisReceiver:
		// allowed
	default:
		fmt.Printf("DEBUG: early return receiver type %T\n", astNode.Receiver)
		return
	}

	expressionStr := *v.currentAstWithSource.source
	absoluteOffset := v.currentAstWithSource.absoluteOffset

	identifierStart := astNode.NameSpan.Start - absoluteOffset

	if identifierStart < 0 || identifierStart >= len(expressionStr) || !strings.HasPrefix(expressionStr[identifierStart:], astNode.Name) {
		v.errors = append(v.errors, fmt.Errorf("Impossible state: %q not found in %q at location %d", astNode.Name, expressionStr, identifierStart))
		return
	}

	absoluteStart := absoluteOffset + identifierStart
	span := NewAbsoluteSourceSpan(absoluteStart, absoluteStart+len(astNode.Name))

	targetAst := v.boundTemplate.GetExpressionTarget(astNode)
	var target *TemplateIdentifier
	if targetAst != nil {
		target = v.targetToIdentifier(targetAst)
	}

	identifier := TemplateIdentifier{
		Name:   astNode.Name,
		Span:   span,
		Kind:   kind,
		Target: target,
	}

	v.addIdentifier(identifier)
}

func (v *templateVisitor) VisitElement(element *render3.Element) interface{} {
	elementIdentifier := v.directiveHostToIdentifier(element)
	if elementIdentifier != nil {
		v.addIdentifier(*elementIdentifier)
	}
	v.CombinedRecursiveAstVisitor.VisitElement(element)
	return nil
}

func (v *templateVisitor) VisitTemplate(template *render3.Template) interface{} {
	templateIdentifier := v.directiveHostToIdentifier(template)
	if templateIdentifier != nil {
		v.addIdentifier(*templateIdentifier)
	}
	v.CombinedRecursiveAstVisitor.VisitTemplate(template)
	return nil
}

func (v *templateVisitor) VisitReference(reference *render3.Reference) interface{} {
	referenceIdentifier := v.targetToIdentifier(reference)
	if referenceIdentifier != nil {
		v.addIdentifier(*referenceIdentifier)
	}
	v.CombinedRecursiveAstVisitor.VisitReference(reference)
	return nil
}

func (v *templateVisitor) VisitVariableExpr(variable *render3.Variable) interface{} {
	variableIdentifier := v.targetToIdentifier(variable)
	if variableIdentifier != nil {
		v.addIdentifier(*variableIdentifier)
	}
	v.CombinedRecursiveAstVisitor.VisitVariableExpr(variable)
	return nil
}

func (v *templateVisitor) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	identifier := v.targetToIdentifier(decl)
	if identifier != nil {
		v.addIdentifier(*identifier)
	}
	if decl.Value != nil {
		v.visitExpr(decl.Value)
	}
	return nil
}

func (v *templateVisitor) VisitComponent(component *render3.Component) interface{} {
	identifier := v.directiveHostToIdentifier(component)
	if identifier != nil {
		v.addIdentifier(*identifier)
	}
	v.CombinedRecursiveAstVisitor.VisitComponent(component)
	return nil
}

func (v *templateVisitor) VisitDirective(directive *render3.Directive) interface{} {
	identifier := v.directiveHostToIdentifier(directive)
	if identifier != nil {
		v.addIdentifier(*identifier)
	}
	v.CombinedRecursiveAstVisitor.VisitDirective(directive)
	return nil
}

func (v *templateVisitor) VisitPropertyRead(ast *expression_parser.PropertyRead, context any) any {
	v.visitIdentifier(ast, IdentifierKindProperty)
	v.CombinedRecursiveAstVisitor.RecursiveAstVisitor.VisitPropertyRead(ast, context)
	return nil
}

func (v *templateVisitor) VisitBoundAttribute(attribute *render3.BoundAttribute) interface{} {
	v.visitExpr(attribute.Value)
	return nil
}

func (v *templateVisitor) VisitBoundEvent(event *render3.BoundEvent) interface{} {
	v.visitExpr(event.Handler)
	return nil
}

func (v *templateVisitor) VisitBoundText(text *render3.BoundText) interface{} {
	v.visitExpr(text.Value)
	return nil
}

func (v *templateVisitor) VisitDeferredTrigger(trigger render3.DeferredTrigger) interface{} {
	if bound, ok := trigger.(*render3.BoundDeferredTrigger); ok {
		v.visitExpr(bound.Value)
	}
	return nil
}

func (v *templateVisitor) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	v.visitExpr(block.Expression)
	for _, g := range block.Groups {
		g.Visit(v)
	}
	return nil
}

func (v *templateVisitor) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	if block.Expression != nil {
		v.visitExpr(block.Expression)
	}
	return nil
}

func (v *templateVisitor) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	if block.Item != nil {
		block.Item.Visit(v)
	}
	for _, cv := range block.ContextVariables {
		cv.Visit(v)
	}
	v.visitExpr(&block.Expression)
	if block.TrackBy != nil {
		v.visitExpr(block.TrackBy)
	}
	for _, n := range block.Children {
		n.Visit(v)
	}
	if block.Empty != nil {
		block.Empty.Visit(v)
	}
	return nil
}

func (v *templateVisitor) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	if block.Expression != nil {
		v.visitExpr(block.Expression)
	}
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(v)
	}
	for _, n := range block.Children {
		n.Visit(v)
	}
	return nil
}

func getTemplateIdentifiers(boundTemplate AbstractBoundTemplate) ([]TemplateIdentifier, []error) {
	visitor := &templateVisitor{
		boundTemplate:                boundTemplate,
		identifiers:                  []TemplateIdentifier{},
		errors:                       []error{},
		targetIdentifierCache:        make(map[any]TemplateIdentifier),
		directiveHostIdentifierCache: make(map[any]TemplateIdentifier),
	}

	combined := &compiler.CombinedRecursiveAstVisitor{}
	combined.RecursiveAstVisitor.Impl = visitor
	combined.RecursiveVisitor.Impl = visitor
	visitor.CombinedRecursiveAstVisitor = combined

	templateAst := boundTemplate.GetTemplateAst()
	if templateAst != nil {
		for _, node := range templateAst {
			if r3Node, ok := node.(render3.Node); ok {
				r3Node.Visit(visitor)
			}
		}
	}

	return visitor.identifiers, visitor.errors
}
