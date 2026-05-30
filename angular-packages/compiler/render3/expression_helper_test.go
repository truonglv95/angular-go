package render3

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

type HumanizedExpressionSource struct {
	AST  string
	Span expression_parser.AbsoluteSourceSpan
}

type innerExpressionVisitor struct {
	expression_parser.RecursiveAstVisitor
	humanizer *ExpressionSourceHumanizer
}

func newInnerExpressionVisitor(humanizer *ExpressionSourceHumanizer) *innerExpressionVisitor {
	iev := &innerExpressionVisitor{humanizer: humanizer}
	iev.RecursiveAstVisitor = expression_parser.RecursiveAstVisitor{Impl: iev}
	return iev
}

func (iev *innerExpressionVisitor) Visit(ast expression_parser.AST, context any) any {
	return iev.RecursiveAstVisitor.Visit(ast, context)
}

func (iev *innerExpressionVisitor) VisitImplicitReceiver(ast *expression_parser.ImplicitReceiver, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitImplicitReceiver(ast, context)
}
func (iev *innerExpressionVisitor) VisitPropertyRead(ast *expression_parser.PropertyRead, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitPropertyRead(ast, context)
}
func (iev *innerExpressionVisitor) VisitMethodCall(ast *expression_parser.MethodCall, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitMethodCall(ast, context)
}
func (iev *innerExpressionVisitor) VisitFunctionCall(ast *expression_parser.FunctionCall, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitFunctionCall(ast, context)
}
func (iev *innerExpressionVisitor) VisitLiteralPrimitive(ast *expression_parser.LiteralPrimitive, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitLiteralPrimitive(ast, context)
}
func (iev *innerExpressionVisitor) VisitBinary(ast *expression_parser.Binary, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitBinary(ast, context)
}
func (iev *innerExpressionVisitor) VisitInterpolation(ast *expression_parser.Interpolation, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitInterpolation(ast, context)
}
func (iev *innerExpressionVisitor) VisitKeyedRead(ast *expression_parser.KeyedRead, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitKeyedRead(ast, context)
}
func (iev *innerExpressionVisitor) VisitPipe(ast *expression_parser.BindingPipe, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitPipe(ast, context)
}
func (iev *innerExpressionVisitor) VisitLiteralArray(ast *expression_parser.LiteralArray, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitLiteralArray(ast, context)
}
func (iev *innerExpressionVisitor) VisitLiteralMap(ast *expression_parser.LiteralMap, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitLiteralMap(ast, context)
}
func (iev *innerExpressionVisitor) VisitConditional(ast *expression_parser.Conditional, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitConditional(ast, context)
}
func (iev *innerExpressionVisitor) VisitKeyedWrite(ast *expression_parser.KeyedWrite, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitKeyedWrite(ast, context)
}
func (iev *innerExpressionVisitor) VisitPropertyWrite(ast *expression_parser.PropertyWrite, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitPropertyWrite(ast, context)
}
func (iev *innerExpressionVisitor) VisitChain(ast *expression_parser.Chain, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitChain(ast, context)
}
func (iev *innerExpressionVisitor) VisitPrefixNot(ast *expression_parser.PrefixNot, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitPrefixNot(ast, context)
}
func (iev *innerExpressionVisitor) VisitNonNullAssert(ast *expression_parser.NonNullAssert, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitNonNullAssert(ast, context)
}
func (iev *innerExpressionVisitor) VisitSafePropertyRead(ast *expression_parser.SafePropertyRead, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitSafePropertyRead(ast, context)
}
func (iev *innerExpressionVisitor) VisitSafeMethodCall(ast *expression_parser.SafeMethodCall, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitSafeMethodCall(ast, context)
}
func (iev *innerExpressionVisitor) VisitSafeKeyedRead(ast *expression_parser.SafeKeyedRead, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitSafeKeyedRead(ast, context)
}
func (iev *innerExpressionVisitor) VisitQuote(ast *expression_parser.Quote, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitQuote(ast, context)
}
func (iev *innerExpressionVisitor) VisitCall(ast *expression_parser.Call, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitCall(ast, context)
}
func (iev *innerExpressionVisitor) VisitSafeCall(ast *expression_parser.SafeCall, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitSafeCall(ast, context)
}
func (iev *innerExpressionVisitor) VisitVoidExpression(ast *expression_parser.VoidExpression, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitVoidExpression(ast, context)
}
func (iev *innerExpressionVisitor) VisitTypeofExpression(ast *expression_parser.TypeofExpression, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitTypeofExpression(ast, context)
}
func (iev *innerExpressionVisitor) VisitParenthesizedExpression(ast *expression_parser.ParenthesizedExpression, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitParenthesizedExpression(ast, context)
}
func (iev *innerExpressionVisitor) VisitThisReceiver(ast *expression_parser.ThisReceiver, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitThisReceiver(ast, context)
}
func (iev *innerExpressionVisitor) VisitUnary(ast *expression_parser.Unary, context any) any {
	iev.humanizer.recordAst(ast)
	return iev.RecursiveAstVisitor.VisitUnary(ast, context)
}

type ExpressionSourceHumanizer struct {
	exprVisitor *innerExpressionVisitor
	result      []HumanizedExpressionSource
}

func NewExpressionSourceHumanizer() *ExpressionSourceHumanizer {
	h := &ExpressionSourceHumanizer{}
	h.exprVisitor = newInnerExpressionVisitor(h)
	return h
}

func (h *ExpressionSourceHumanizer) recordAst(ast expression_parser.AST) {
	unparsed := Unparse(ast)
	var sourceSpan expression_parser.AbsoluteSourceSpan
	if has, ok := ast.(interface{ GetSourceSpan() expression_parser.AbsoluteSourceSpan }); ok {
		sourceSpan = has.GetSourceSpan()
	}
	h.result = append(h.result, HumanizedExpressionSource{
		AST:  unparsed,
		Span: sourceSpan,
	})
}

func (h *ExpressionSourceHumanizer) Visit(node Node) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) visitExpression(ast expression_parser.AST) {
	if ast == nil {
		return
	}
	if astWithSource, ok := ast.(*expression_parser.ASTWithSource); ok {
		h.recordAst(astWithSource)
		h.exprVisitor.Visit(astWithSource.Ast, nil)
	} else {
		h.exprVisitor.Visit(ast, nil)
	}
}

func (h *ExpressionSourceHumanizer) VisitElement(element *Element) interface{} {
	for _, attr := range element.Attributes {
		attr.Visit(h)
	}
	for _, input := range element.Inputs {
		input.Visit(h)
	}
	for _, output := range element.Outputs {
		output.Visit(h)
	}
	for _, dir := range element.Directives {
		dir.Visit(h)
	}
	for _, ref := range element.References {
		ref.Visit(h)
	}
	for _, child := range element.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitTemplate(template *Template) interface{} {
	for _, attr := range template.Attributes {
		attr.Visit(h)
	}
	for _, input := range template.Inputs {
		input.Visit(h)
	}
	for _, output := range template.Outputs {
		output.Visit(h)
	}
	for _, dir := range template.Directives {
		dir.Visit(h)
	}
	for _, attr := range template.TemplateAttrs {
		attr.Visit(h)
	}
	for _, ref := range template.References {
		ref.Visit(h)
	}
	for _, variable := range template.Variables {
		variable.Visit(h)
	}
	for _, child := range template.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitBoundAttribute(attribute *BoundAttribute) interface{} {
	h.visitExpression(attribute.Value)
	return nil
}

func (h *ExpressionSourceHumanizer) VisitBoundEvent(event *BoundEvent) interface{} {
	h.visitExpression(event.Handler)
	return nil
}

func (h *ExpressionSourceHumanizer) VisitBoundText(text *BoundText) interface{} {
	h.visitExpression(text.Value)
	return nil
}

func (h *ExpressionSourceHumanizer) VisitIcu(icu *Icu) interface{} {
	for _, variable := range icu.Vars {
		variable.Visit(h)
	}
	for _, ph := range icu.Placeholders {
		ph.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitSwitchBlock(block *SwitchBlock) interface{} {
	h.visitExpression(block.Expression)
	for _, group := range block.Groups {
		group.Visit(h)
	}
	if block.ExhaustiveCheck != nil {
		block.ExhaustiveCheck.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitSwitchBlockCase(block *SwitchBlockCase) interface{} {
	if block.Expression != nil {
		h.visitExpression(block.Expression)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitSwitchBlockCaseGroup(block *SwitchBlockCaseGroup) interface{} {
	for _, c := range block.Cases {
		c.Visit(h)
	}
	for _, child := range block.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitSwitchExhaustiveCheck(block *SwitchExhaustiveCheck) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) VisitForLoopBlock(block *ForLoopBlock) interface{} {
	block.Item.Visit(h)
	for _, cv := range block.ContextVariables {
		cv.Visit(h)
	}
	h.visitExpression(&block.Expression)
	for _, child := range block.Children {
		child.Visit(h)
	}
	if block.Empty != nil {
		block.Empty.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) interface{} {
	for _, child := range block.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitIfBlock(block *IfBlock) interface{} {
	for _, branch := range block.Branches {
		branch.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitIfBlockBranch(block *IfBlockBranch) interface{} {
	if block.Expression != nil {
		h.visitExpression(block.Expression)
	}
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(h)
	}
	for _, child := range block.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitLetDeclaration(decl *LetDeclaration) interface{} {
	h.visitExpression(decl.Value)
	return nil
}

func (h *ExpressionSourceHumanizer) VisitDeferredBlock(deferred *DeferredBlock) interface{} {
	deferred.VisitAll(h)
	return nil
}

func (h *ExpressionSourceHumanizer) VisitDeferredTrigger(trigger DeferredTrigger) interface{} {
	if bound, ok := trigger.(*BoundDeferredTrigger); ok {
		h.visitExpression(bound.Value)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) interface{} {
	for _, child := range block.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitDeferredBlockLoading(block *DeferredBlockLoading) interface{} {
	for _, child := range block.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitDeferredBlockError(block *DeferredBlockError) interface{} {
	for _, child := range block.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitComment(comment *Comment) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) VisitText(text *Text) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) VisitTextAttribute(attribute *TextAttribute) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) VisitReference(reference *Reference) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) VisitVariableExpr(variable *Variable) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) VisitContent(content *Content) interface{} {
	for _, child := range content.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitUnknownBlock(block *UnknownBlock) interface{} {
	return nil
}

func (h *ExpressionSourceHumanizer) VisitComponent(component *Component) interface{} {
	for _, attr := range component.Attributes {
		attr.Visit(h)
	}
	for _, input := range component.Inputs {
		input.Visit(h)
	}
	for _, output := range component.Outputs {
		output.Visit(h)
	}
	for _, dir := range component.Directives {
		dir.Visit(h)
	}
	for _, ref := range component.References {
		ref.Visit(h)
	}
	for _, child := range component.Children {
		child.Visit(h)
	}
	return nil
}

func (h *ExpressionSourceHumanizer) VisitDirective(directive *Directive) interface{} {
	for _, attr := range directive.Attributes {
		attr.Visit(h)
	}
	for _, input := range directive.Inputs {
		input.Visit(h)
	}
	for _, output := range directive.Outputs {
		output.Visit(h)
	}
	for _, ref := range directive.References {
		ref.Visit(h)
	}
	return nil
}

func humanizeExpressionSource(templateAsts []Node) []HumanizedExpressionSource {
	humanizer := NewExpressionSourceHumanizer()
	for _, node := range templateAsts {
		node.Visit(humanizer)
	}
	return humanizer.result
}

type Unparser struct {
	result strings.Builder
}

func Unparse(ast expression_parser.AST) string {
	if ast == nil {
		return ""
	}
	u := &Unparser{}
	u.visit(ast)
	return u.result.String()
}

func (u *Unparser) visit(ast expression_parser.AST) {
	if ast == nil {
		return
	}
	ast.Visit(u, nil)
}

func (u *Unparser) VisitImplicitReceiver(ast *expression_parser.ImplicitReceiver, context any) any {
	return nil
}

func (u *Unparser) VisitThisReceiver(ast *expression_parser.ThisReceiver, context any) any {
	return nil
}

func (u *Unparser) VisitPropertyRead(ast *expression_parser.PropertyRead, context any) any {
	u.visit(ast.Receiver)
	_, isImplicit := ast.Receiver.(*expression_parser.ImplicitReceiver)
	_, isThis := ast.Receiver.(*expression_parser.ThisReceiver)
	if isImplicit || isThis {
		u.result.WriteString(ast.Name)
	} else {
		u.result.WriteString(".")
		u.result.WriteString(ast.Name)
	}
	return nil
}

func (u *Unparser) VisitUnary(ast *expression_parser.Unary, context any) any {
	u.result.WriteString(ast.Operator)
	u.visit(ast.Expr)
	return nil
}

func (u *Unparser) VisitBinary(ast *expression_parser.Binary, context any) any {
	u.visit(ast.Left)
	u.result.WriteString(" ")
	u.result.WriteString(ast.Operation)
	u.result.WriteString(" ")
	u.visit(ast.Right)
	return nil
}

func (u *Unparser) VisitChain(ast *expression_parser.Chain, context any) any {
	for i, expr := range ast.Expressions {
		u.visit(expr)
		if i == len(ast.Expressions)-1 {
			u.result.WriteString(";")
		} else {
			u.result.WriteString("; ")
		}
	}
	return nil
}

func (u *Unparser) VisitConditional(ast *expression_parser.Conditional, context any) any {
	u.visit(ast.Condition)
	u.result.WriteString(" ? ")
	u.visit(ast.TrueExp)
	u.result.WriteString(" : ")
	u.visit(ast.FalseExp)
	return nil
}

func (u *Unparser) VisitPipe(ast *expression_parser.BindingPipe, context any) any {
	u.result.WriteString("(")
	u.visit(ast.Exp)
	u.result.WriteString(" | ")
	u.result.WriteString(ast.Name)
	for _, arg := range ast.Args {
		u.result.WriteString(":")
		u.visit(arg)
	}
	u.result.WriteString(")")
	return nil
}

func (u *Unparser) VisitCall(ast *expression_parser.Call, context any) any {
	u.visit(ast.Receiver)
	u.result.WriteString("(")
	for i, arg := range ast.Args {
		if i > 0 {
			u.result.WriteString(", ")
		}
		u.visit(arg)
	}
	u.result.WriteString(")")
	return nil
}

func (u *Unparser) VisitSafeCall(ast *expression_parser.SafeCall, context any) any {
	u.visit(ast.Receiver)
	u.result.WriteString("?.(")
	for i, arg := range ast.Args {
		if i > 0 {
			u.result.WriteString(", ")
		}
		u.visit(arg)
	}
	u.result.WriteString(")")
	return nil
}

func (u *Unparser) VisitInterpolation(ast *expression_parser.Interpolation, context any) any {
	for i := 0; i < len(ast.Strings); i++ {
		u.result.WriteString(fmt.Sprintf("%v", ast.Strings[i]))
		if i < len(ast.Expressions) {
			u.result.WriteString("{{ ")
			u.visit(ast.Expressions[i])
			u.result.WriteString(" }}")
		}
	}
	return nil
}

func (u *Unparser) VisitKeyedRead(ast *expression_parser.KeyedRead, context any) any {
	u.visit(ast.Receiver)
	u.result.WriteString("[")
	u.visit(ast.Key)
	u.result.WriteString("]")
	return nil
}

func (u *Unparser) VisitKeyedWrite(ast *expression_parser.KeyedWrite, context any) any {
	u.visit(ast.Receiver)
	u.result.WriteString("[")
	u.visit(ast.Key)
	u.result.WriteString("] = ")
	u.visit(ast.Value)
	return nil
}

func (u *Unparser) VisitLiteralArray(ast *expression_parser.LiteralArray, context any) any {
	u.result.WriteString("[")
	for i, expr := range ast.Expressions {
		if i > 0 {
			u.result.WriteString(", ")
		}
		u.visit(expr)
	}
	u.result.WriteString("]")
	return nil
}

func (u *Unparser) VisitLiteralMap(ast *expression_parser.LiteralMap, context any) any {
	u.result.WriteString("{")
	for i, key := range ast.Keys {
		if i > 0 {
			u.result.WriteString(", ")
		}
		if spreadKey, ok := key.(*expression_parser.LiteralMapSpreadKey); ok {
			_ = spreadKey
			u.result.WriteString("...")
		} else if propKey, ok := key.(*expression_parser.LiteralMapPropertyKey); ok {
			if propKey.Quoted {
				u.result.WriteString(fmt.Sprintf("\"%s\"", propKey.Key))
			} else {
				u.result.WriteString(propKey.Key)
			}
			u.result.WriteString(": ")
		}
		u.visit(ast.Values[i])
	}
	u.result.WriteString("}")
	return nil
}

func (u *Unparser) VisitLiteralPrimitive(ast *expression_parser.LiteralPrimitive, context any) any {
	if ast.Value == nil {
		u.result.WriteString("null")
		return nil
	}
	switch val := ast.Value.(type) {
	case string:
		u.result.WriteString(fmt.Sprintf("\"%s\"", val))
	case bool:
		u.result.WriteString(fmt.Sprintf("%t", val))
	default:
		u.result.WriteString(fmt.Sprintf("%v", val))
	}
	return nil
}

func (u *Unparser) VisitPrefixNot(ast *expression_parser.PrefixNot, context any) any {
	u.result.WriteString("!")
	u.visit(ast.Expression)
	return nil
}

func (u *Unparser) VisitTypeofExpression(ast *expression_parser.TypeofExpression, context any) any {
	u.result.WriteString("typeof ")
	u.visit(ast.Expr)
	return nil
}

func (u *Unparser) VisitVoidExpression(ast *expression_parser.VoidExpression, context any) any {
	u.result.WriteString("void ")
	u.visit(ast.Expr)
	return nil
}

func (u *Unparser) VisitNonNullAssert(ast *expression_parser.NonNullAssert, context any) any {
	u.visit(ast.Expression)
	u.result.WriteString("!")
	return nil
}

func (u *Unparser) VisitSafePropertyRead(ast *expression_parser.SafePropertyRead, context any) any {
	u.visit(ast.Receiver)
	u.result.WriteString("?.")
	u.result.WriteString(ast.Name)
	return nil
}

func (u *Unparser) VisitSafeKeyedRead(ast *expression_parser.SafeKeyedRead, context any) any {
	u.visit(ast.Receiver)
	u.result.WriteString("?.[")
	u.visit(ast.Key)
	u.result.WriteString("]")
	return nil
}

func (u *Unparser) VisitTemplateLiteral(ast *expression_parser.TemplateLiteral, context any) any {
	u.result.WriteString("`")
	for i, elem := range ast.Elements {
		u.visit(elem)
		if i < len(ast.Expressions) {
			u.result.WriteString("${")
			u.visit(ast.Expressions[i])
			u.result.WriteString("}")
		}
	}
	u.result.WriteString("`")
	return nil
}

func (u *Unparser) VisitTemplateLiteralElement(ast *expression_parser.TemplateLiteralElement, context any) any {
	u.result.WriteString(ast.Text)
	return nil
}

func (u *Unparser) VisitTaggedTemplateLiteral(ast *expression_parser.TaggedTemplateLiteral, context any) any {
	u.visit(ast.Tag)
	u.visit(ast.Template)
	return nil
}

func (u *Unparser) VisitParenthesizedExpression(ast *expression_parser.ParenthesizedExpression, context any) any {
	u.result.WriteString("(")
	u.visit(ast.Expression)
	u.result.WriteString(")")
	return nil
}

func (u *Unparser) VisitRegularExpressionLiteral(ast *expression_parser.RegularExpressionLiteral, context any) any {
	u.result.WriteString("/")
	u.result.WriteString(ast.Body)
	u.result.WriteString("/")
	u.result.WriteString(ast.Flags)
	return nil
}

func (u *Unparser) VisitSpreadElement(ast *expression_parser.SpreadElement, context any) any {
	u.result.WriteString("...")
	u.visit(ast.Expression)
	return nil
}

func (u *Unparser) VisitArrowFunction(ast *expression_parser.ArrowFunction, context any) any {
	if len(ast.Params) == 1 {
		if id, ok := ast.Params[0].(*expression_parser.ArrowFunctionIdentifierParameter); ok {
			u.result.WriteString(id.Name)
		}
	} else {
		u.result.WriteString("(")
		for i, p := range ast.Params {
			if i > 0 {
				u.result.WriteString(", ")
			}
			if id, ok := p.(*expression_parser.ArrowFunctionIdentifierParameter); ok {
				u.result.WriteString(id.Name)
			}
		}
		u.result.WriteString(")")
	}
	u.result.WriteString(" => ")
	u.visit(ast.Body)
	return nil
}

func (u *Unparser) VisitPropertyWrite(ast *expression_parser.PropertyWrite, context any) any {
	u.visit(ast.Receiver)
	_, isImplicit := ast.Receiver.(*expression_parser.ImplicitReceiver)
	_, isThis := ast.Receiver.(*expression_parser.ThisReceiver)
	if isImplicit || isThis {
		u.result.WriteString(ast.Name)
	} else {
		u.result.WriteString(".")
		u.result.WriteString(ast.Name)
	}
	u.result.WriteString(" = ")
	u.visit(ast.Value)
	return nil
}

func (u *Unparser) VisitSafeMethodCall(ast *expression_parser.SafeMethodCall, context any) any {
	u.visit(ast.Receiver)
	u.result.WriteString("?.")
	u.result.WriteString(ast.Name)
	u.result.WriteString("(")
	for i, arg := range ast.Args {
		if i > 0 {
			u.result.WriteString(", ")
		}
		u.visit(arg)
	}
	u.result.WriteString(")")
	return nil
}

func (u *Unparser) VisitMethodCall(ast *expression_parser.MethodCall, context any) any {
	u.visit(ast.Receiver)
	_, isImplicit := ast.Receiver.(*expression_parser.ImplicitReceiver)
	_, isThis := ast.Receiver.(*expression_parser.ThisReceiver)
	if isImplicit || isThis {
		u.result.WriteString(ast.Name)
	} else {
		u.result.WriteString(".")
		u.result.WriteString(ast.Name)
	}
	u.result.WriteString("(")
	for i, arg := range ast.Args {
		if i > 0 {
			u.result.WriteString(", ")
		}
		u.visit(arg)
	}
	u.result.WriteString(")")
	return nil
}

func (u *Unparser) VisitFunctionCall(ast *expression_parser.FunctionCall, context any) any {
	u.visit(ast.Target)
	u.result.WriteString("(")
	for i, arg := range ast.Args {
		if i > 0 {
			u.result.WriteString(", ")
		}
		u.visit(arg)
	}
	u.result.WriteString(")")
	return nil
}

func (u *Unparser) VisitQuote(ast *expression_parser.Quote, context any) any {
	u.result.WriteString(ast.Prefix)
	u.result.WriteString(":")
	u.result.WriteString(ast.UninterpretedExpression)
	return nil
}
