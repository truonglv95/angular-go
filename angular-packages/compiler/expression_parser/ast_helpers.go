package expression_parser

type Unary struct {
	ASTNodeBase
	Operator string
	Expr     AST
}

func (u *Unary) Visit(v Visitor, c any) any { return v.VisitUnary(u, c) }

type TypeofExpression struct {
	ASTNodeBase
	Expr AST
}

func (t *TypeofExpression) Visit(v Visitor, c any) any { return v.VisitTypeofExpression(t, c) }

type VoidExpression struct {
	ASTNodeBase
	Expr AST
}

func IsUnary(ast AST) bool {
	_, ok := ast.(*Unary)
	return ok
}

func IsPrefixNot(ast AST) bool {
	_, ok := ast.(*PrefixNot)
	return ok
}

func IsTypeofExpression(ast AST) bool {
	_, ok := ast.(*TypeofExpression)
	return ok
}

func IsVoidExpression(ast AST) bool {
	_, ok := ast.(*VoidExpression)
	return ok
}

func CreateUnaryPlus(span ParseSpan, sourceSpan AbsoluteSourceSpan, expr AST) *Unary {
	return &Unary{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Operator: "+", Expr: expr}
}

func CreateUnaryMinus(span ParseSpan, sourceSpan AbsoluteSourceSpan, expr AST) *Unary {
	return &Unary{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Operator: "-", Expr: expr}
}

func NewPrefixNot(span ParseSpan, sourceSpan AbsoluteSourceSpan, expr AST) *PrefixNot {
	return &PrefixNot{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expression: expr}
}

func NewTypeofExpression(span ParseSpan, sourceSpan AbsoluteSourceSpan, expr AST) *TypeofExpression {
	return &TypeofExpression{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expr: expr}
}

type ParenthesizedExpression struct {
	ASTNodeBase
	Expression AST
}

func (n *ParenthesizedExpression) Visit(v Visitor, c any) any { return v.VisitParenthesizedExpression(n, c) }
func NewParenthesizedExpression(span ParseSpan, sourceSpan AbsoluteSourceSpan, expr AST) *ParenthesizedExpression {
	return &ParenthesizedExpression{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expression: expr}
}



func NewSafePropertyRead(span ParseSpan, sourceSpan AbsoluteSourceSpan, nameSpan AbsoluteSourceSpan, receiver AST, name string) *SafePropertyRead { return &SafePropertyRead{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Receiver: receiver, Name: name} }

func NewSafeKeyedRead(span ParseSpan, sourceSpan AbsoluteSourceSpan, receiver AST, key AST) *SafeKeyedRead { return &SafeKeyedRead{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Receiver: receiver, Key: key} }



func NewNonNullAssert(span ParseSpan, sourceSpan AbsoluteSourceSpan, expr AST) *NonNullAssert { return &NonNullAssert{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expression: expr} }



type ArrowFunction struct {
	ASTNodeBase
	Params []ArrowFunctionParameter
	Body   AST
}

func (n *ArrowFunction) Visit(v Visitor, c any) any { return nil }
func NewArrowFunction(span ParseSpan, sourceSpan AbsoluteSourceSpan, params []ArrowFunctionParameter, body AST) *ArrowFunction {
	return &ArrowFunction{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Params: params, Body: body}
}

type SafeCall struct {
	ASTNodeBase
	Receiver AST
	Args     []AST
}

func (n *SafeCall) Visit(v Visitor, c any) any { return v.VisitSafeCall(n, c) }
func NewSafeCall(span ParseSpan, sourceSpan AbsoluteSourceSpan, receiver AST, args []AST, argumentSpan ParseSpan) *SafeCall { return &SafeCall{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Receiver: receiver, Args: args} }

type SpreadElement struct {
	ASTNodeBase
	Expression AST
}

func (n *SpreadElement) Visit(v Visitor, c any) any { return nil }
func NewSpreadElement(span ParseSpan, sourceSpan AbsoluteSourceSpan, expression AST) *SpreadElement {
	return &SpreadElement{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expression: expression}
}



type ThisReceiver struct {
	ASTNodeBase
}

func (n *ThisReceiver) Visit(v Visitor, c any) any { return v.VisitThisReceiver(n, c) }
func NewThisReceiver(span ParseSpan, sourceSpan AbsoluteSourceSpan) *ThisReceiver { return &ThisReceiver{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}} }



type Call struct {
	ASTNodeBase
	Receiver AST
	Args     []AST
}

func (n *Call) Visit(v Visitor, c any) any { return v.VisitCall(n, c) }
func NewCall(span ParseSpan, sourceSpan AbsoluteSourceSpan, receiver AST, args []AST, argumentSpan ParseSpan) *Call { return &Call{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Receiver: receiver, Args: args} }



type TaggedTemplateLiteral struct {
	ASTNodeBase
	Tag      AST
	Template AST
}

func (n *TaggedTemplateLiteral) Visit(v Visitor, c any) any { return nil }
func NewTaggedTemplateLiteral(span ParseSpan, sourceSpan AbsoluteSourceSpan, tag AST, template AST) *TaggedTemplateLiteral {
	return &TaggedTemplateLiteral{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Tag: tag, Template: template}
}

type RegularExpressionLiteral struct {
	ASTNodeBase
	Body  string
	Flags string
}

func (n *RegularExpressionLiteral) Visit(v Visitor, c any) any { return nil }
func NewRegularExpressionLiteral(span ParseSpan, sourceSpan AbsoluteSourceSpan, body string, flags string) *RegularExpressionLiteral {
	return &RegularExpressionLiteral{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Body: body, Flags: flags}
}

func NewPropertyRead(span ParseSpan, sourceSpan AbsoluteSourceSpan, nameSpan AbsoluteSourceSpan, receiver AST, name string) *PropertyRead {
	return &PropertyRead{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Receiver: receiver, Name: name, NameSpan: nameSpan}
}

func NewTemplateBindingParseResult(args ...any) *TemplateBindingParseResult {
	return &TemplateBindingParseResult{}
}

func NewLiteralMap(span ParseSpan, sourceSpan AbsoluteSourceSpan, keys []LiteralMapKey, values []AST) *LiteralMap { return &LiteralMap{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Keys: keys, Values: values} }

func NewKeyedRead(span ParseSpan, sourceSpan AbsoluteSourceSpan, receiver AST, key AST) *KeyedRead { return &KeyedRead{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Receiver: receiver, Key: key} }

func NewImplicitReceiver(span ParseSpan, sourceSpan AbsoluteSourceSpan) *ImplicitReceiver { return &ImplicitReceiver{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}} }

func NewLiteralArray(span ParseSpan, sourceSpan AbsoluteSourceSpan, expressions []AST) *LiteralArray { return &LiteralArray{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expressions: expressions} }

type TemplateLiteralElement struct {
	ASTNodeBase
	Text string
}

func (n *TemplateLiteralElement) Visit(v Visitor, c any) any { return nil }
func NewTemplateLiteralElement(span ParseSpan, sourceSpan AbsoluteSourceSpan, text string) *TemplateLiteralElement {
	return &TemplateLiteralElement{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Text: text}
}

type TemplateLiteral struct {
	ASTNodeBase
	Elements    []*TemplateLiteralElement
	Expressions []AST
}

func (n *TemplateLiteral) Visit(v Visitor, c any) any { return nil }
func NewTemplateLiteral(span ParseSpan, sourceSpan AbsoluteSourceSpan, elements []*TemplateLiteralElement, expressions []AST) *TemplateLiteral {
	return &TemplateLiteral{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Elements: elements, Expressions: expressions}
}

func NewVoidExpression(span ParseSpan, sourceSpan AbsoluteSourceSpan, expr AST) *VoidExpression { return &VoidExpression{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expr: expr} }

func (v *VoidExpression) Visit(vis Visitor, c any) any { return vis.VisitVoidExpression(v, c) }



func (n *AbsoluteSourceSpan) Span() ParseSpan { return ParseSpan{} }















func (n *TemplateBindingParseResult) Span() ParseSpan { return ParseSpan{} }









func (n *ParseSourceSpan) Span() ParseSpan { return ParseSpan{} }

















func (n *PropertyRead) Span() ParseSpan { return n.SpanData }









































func (n *ParseSpan) Span() ParseSpan { return ParseSpan{} }













type LiteralPrimitiveIntf interface {
	isLiteralPrimitive()
}

func (a *ASTWithSource) Span() ParseSpan {
	length := 0
	if a.Source != "" {
		length = len(a.Source)
	}
	return ParseSpan{Start: 0, End: length}
}

func (a *ASTWithSource) GetSourceSpan() AbsoluteSourceSpan {
	length := 0
	if a.Source != "" {
		length = len(a.Source)
	}
	return AbsoluteSourceSpan{Start: a.AbsoluteOffset, End: a.AbsoluteOffset + length}
}
