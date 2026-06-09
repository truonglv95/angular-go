package expression_parser

type Visitor interface {
	VisitImplicitReceiver(ast *ImplicitReceiver, context any) any
	VisitPropertyRead(ast *PropertyRead, context any) any
	VisitMethodCall(ast *MethodCall, context any) any
	VisitFunctionCall(ast *FunctionCall, context any) any
	VisitLiteralPrimitive(ast *LiteralPrimitive, context any) any
	VisitBinary(ast *Binary, context any) any
	VisitInterpolation(ast *Interpolation, context any) any
	VisitKeyedRead(ast *KeyedRead, context any) any
	VisitPipe(ast *BindingPipe, context any) any
	VisitLiteralArray(ast *LiteralArray, context any) any
	VisitLiteralMap(ast *LiteralMap, context any) any
	VisitConditional(ast *Conditional, context any) any
	VisitKeyedWrite(ast *KeyedWrite, context any) any
	VisitPropertyWrite(ast *PropertyWrite, context any) any
	VisitChain(ast *Chain, context any) any
	VisitPrefixNot(ast *PrefixNot, context any) any
	VisitNonNullAssert(ast *NonNullAssert, context any) any
	VisitSafePropertyRead(ast *SafePropertyRead, context any) any
	VisitSafeMethodCall(ast *SafeMethodCall, context any) any
	VisitSafeKeyedRead(ast *SafeKeyedRead, context any) any
	VisitQuote(ast *Quote, context any) any
	VisitCall(ast *Call, context any) any
	VisitSafeCall(ast *SafeCall, context any) any
	VisitVoidExpression(ast *VoidExpression, context any) any
	VisitTypeofExpression(ast *TypeofExpression, context any) any
	VisitParenthesizedExpression(ast *ParenthesizedExpression, context any) any
	VisitThisReceiver(ast *ThisReceiver, context any) any
	VisitUnary(ast *Unary, context any) any





}

// Ensure all new AST types are defined:

type ASTNodeBase struct {
	SpanData   ParseSpan
	SourceSpan AbsoluteSourceSpan
}

func (b ASTNodeBase) Span() ParseSpan { return b.SpanData }
func (b ASTNodeBase) GetSourceSpan() AbsoluteSourceSpan { return b.SourceSpan }


type Conditional struct {
	ASTNodeBase
	Condition AST
	TrueExp   AST
	FalseExp  AST
}

func (c *Conditional) Visit(visitor Visitor, context any) any {
	return visitor.VisitConditional(c, context)
}

type KeyedWrite struct {
	ASTNodeBase
	Receiver AST
	Key      AST
	Value    AST
}

func (k *KeyedWrite) Visit(visitor Visitor, context any) any {
	return visitor.VisitKeyedWrite(k, context)
}

type PropertyWrite struct {
	ASTNodeBase
	Receiver AST
	Name     string
	Value    AST
}

func (p *PropertyWrite) Visit(visitor Visitor, context any) any {
	return visitor.VisitPropertyWrite(p, context)
}

type Chain struct {
	ASTNodeBase
	Expressions []AST
}

func (c *Chain) Visit(visitor Visitor, context any) any { return visitor.VisitChain(c, context) }

type PrefixNot struct {
	ASTNodeBase
	Expression AST
}

func (p *PrefixNot) Visit(visitor Visitor, context any) any {
	return visitor.VisitPrefixNot(p, context)
}

type NonNullAssert struct {
	ASTNodeBase
	Expression AST
}

func (n *NonNullAssert) Visit(visitor Visitor, context any) any {
	return visitor.VisitNonNullAssert(n, context)
}

type SafePropertyRead struct {
	ASTNodeBase
	Receiver AST
	Name     string
}

func (s *SafePropertyRead) Visit(visitor Visitor, context any) any {
	return visitor.VisitSafePropertyRead(s, context)
}

type SafeMethodCall struct {
	ASTNodeBase
	Receiver AST
	Name     string
	Args     []AST
}

func (s *SafeMethodCall) Visit(visitor Visitor, context any) any {
	return visitor.VisitSafeMethodCall(s, context)
}

type SafeKeyedRead struct {
	ASTNodeBase
	Receiver AST
	Key      AST
}

func (s *SafeKeyedRead) Visit(visitor Visitor, context any) any {
	return visitor.VisitSafeKeyedRead(s, context)
}

type Quote struct {
	ASTNodeBase
	Prefix                  string
	UninterpretedExpression string
	Location                any
}

func (q *Quote) Visit(visitor Visitor, context any) any { return visitor.VisitQuote(q, context) }

type AST interface {
	Visit(visitor Visitor, context any) any
	Span() ParseSpan
}

type ImplicitReceiver struct {
	ASTNodeBase
}

func (i *ImplicitReceiver) Visit(visitor Visitor, context any) any {
	return visitor.VisitImplicitReceiver(i, context)
}

type PropertyRead struct {
	ASTNodeBase
	Receiver AST
	Name     string
	NameSpan AbsoluteSourceSpan
}


func (p *PropertyRead) Visit(visitor Visitor, context any) any {
	return visitor.VisitPropertyRead(p, context)
}

type MethodCall struct {
	ASTNodeBase
	Receiver AST
	Name     string
	Args     []AST
}

func (m *MethodCall) Visit(visitor Visitor, context any) any {
	return visitor.VisitMethodCall(m, context)
}

type FunctionCall struct {
	ASTNodeBase
	Target AST
	Args   []AST
}

func (f *FunctionCall) Visit(visitor Visitor, context any) any {
	return visitor.VisitFunctionCall(f, context)
}

type LiteralPrimitive struct {
	ASTNodeBase
	Value any
}

func (l *LiteralPrimitive) Visit(visitor Visitor, context any) any {
	return visitor.VisitLiteralPrimitive(l, context)
}

type Binary struct {
	ASTNodeBase
	Operation string
	Left      AST
	Right     AST
}

func (b *Binary) Visit(visitor Visitor, context any) any {
	return visitor.VisitBinary(b, context)
}

type Interpolation struct {
	ASTNodeBase
	Strings     []any
	Expressions []AST
}

func (i *Interpolation) Visit(visitor Visitor, context any) any {
	return visitor.VisitInterpolation(i, context)
}

type KeyedRead struct {
	ASTNodeBase
	Receiver AST
	Key      AST
}

func (k *KeyedRead) Visit(visitor Visitor, context any) any {
	return visitor.VisitKeyedRead(k, context)
}

type BindingPipe struct {
	ASTNodeBase
	Exp      AST
	Name     string
	Args     []AST
	PipeType BindingPipeType
}

func (b *BindingPipe) Visit(visitor Visitor, context any) any {
	return visitor.VisitPipe(b, context)
}

type LiteralArray struct {
	ASTNodeBase
	Expressions []AST
}

func (l *LiteralArray) Visit(visitor Visitor, context any) any {
	return visitor.VisitLiteralArray(l, context)
}

type LiteralMapKey interface {
	isLiteralMapKey()
}

type LiteralMapPropertyKey struct {
	Key      string
	Quoted   bool
	Span     ParseSpan
	SourceSpan AbsoluteSourceSpan
	IsShorthandInitialized bool
}

func (k *LiteralMapPropertyKey) isLiteralMapKey() {}

func (k *LiteralMapPropertyKey) SetIsShorthandInitialized(v bool) { k.IsShorthandInitialized = v }

func NewLiteralMapPropertyKey(key string, quoted bool, span ParseSpan, sourceSpan AbsoluteSourceSpan) *LiteralMapPropertyKey {
	return &LiteralMapPropertyKey{Key: key, Quoted: quoted, Span: span, SourceSpan: sourceSpan}
}

type LiteralMapSpreadKey struct {
	Span       ParseSpan
	SourceSpan AbsoluteSourceSpan
}

func (k *LiteralMapSpreadKey) isLiteralMapKey() {}

func NewLiteralMapSpreadKey(span ParseSpan, sourceSpan AbsoluteSourceSpan) *LiteralMapSpreadKey {
	return &LiteralMapSpreadKey{Span: span, SourceSpan: sourceSpan}
}

type LiteralMap struct {
	ASTNodeBase
	Keys   []LiteralMapKey
	Values []AST
}

func (l *LiteralMap) Visit(visitor Visitor, context any) any {
	return visitor.VisitLiteralMap(l, context)
}

// Additional Types for Parity with parser.ts

type AbsoluteSourceSpan struct {
	Start int
	End   int
}

type ParseSourceSpan struct {
	Start     int
	End       int
	FullStart int // Pre-trivia start offset (matches TS sourceSpan.fullStart.offset)
}

type ParseError struct {
	Span    ParseSourceSpan
	Message string
}

func (pe *ParseError) Error() string {
	return pe.Message
}

type ASTWithSource struct {
	Ast            AST
	Source         string
	Location       string
	AbsoluteOffset int
	Errors         []ParseError
}

type ParseSpan struct {
	Start int
	End   int
}

func (s ParseSpan) ToAbsolute(absoluteOffset int) AbsoluteSourceSpan {
	return AbsoluteSourceSpan{Start: absoluteOffset + s.Start, End: absoluteOffset + s.End}
}

func NewParseError(span ParseSourceSpan, message string) ParseError {
	return ParseError{Span: span, Message: message}
}

func NewASTWithSource(ast AST, source string, location string, absoluteOffset int, errors []ParseError) ASTWithSource {
	return ASTWithSource{Ast: ast, Source: source, Location: location, AbsoluteOffset: absoluteOffset, Errors: errors}
}

func NewAbsoluteSourceSpan(start int, end int) AbsoluteSourceSpan {
	return AbsoluteSourceSpan{Start: start, End: end}
}

type ArrowFunctionParameter interface {
	isArrowFunctionParameter()
}

type ArrowFunctionIdentifierParameter struct {
	Name       string
	Span       ParseSpan
	SourceSpan AbsoluteSourceSpan
}

func (a *ArrowFunctionIdentifierParameter) isArrowFunctionParameter() {}

func NewArrowFunctionIdentifierParameter(name string, span ParseSpan, sourceSpan AbsoluteSourceSpan) *ArrowFunctionIdentifierParameter {
	return &ArrowFunctionIdentifierParameter{Name: name, Span: span, SourceSpan: sourceSpan}
}

type EmptyExpr struct {
	ASTNodeBase
}

func (e *EmptyExpr) Visit(visitor Visitor, context any) any { return nil }

func NewEmptyExpr(span ParseSpan, sourceSpan AbsoluteSourceSpan) *EmptyExpr {
	return &EmptyExpr{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}}
}

func NewBindingPipe(span ParseSpan, sourceSpan AbsoluteSourceSpan, exp AST, name string, args []AST, pipeType BindingPipeType, nameSpan AbsoluteSourceSpan) *BindingPipe {
	return &BindingPipe{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Exp: exp, Name: name, Args: args, PipeType: pipeType}
}

func NewConditional(span ParseSpan, sourceSpan AbsoluteSourceSpan, condition AST, trueExp AST, falseExp AST) *Conditional {
	return &Conditional{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Condition: condition, TrueExp: trueExp, FalseExp: falseExp}
}

func NewBinary(span ParseSpan, sourceSpan AbsoluteSourceSpan, operation string, left AST, right AST) *Binary {
	return &Binary{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Operation: operation, Left: left, Right: right}
}

func NewChain(span ParseSpan, sourceSpan AbsoluteSourceSpan, expressions []AST) *Chain {
	return &Chain{ASTNodeBase: ASTNodeBase{SpanData: span, SourceSpan: sourceSpan}, Expressions: expressions}
}

type BindingPipeType int

const (
	BindingPipeTypeReferencedByName BindingPipeType = iota
	BindingPipeTypeReferencedDirectly
)

type TemplateBindingIdentifier struct {
	Source string
	Span   ParseSpan
}

type TemplateBinding interface {
	isTemplateBinding()
}

type VariableBinding struct {
	SourceSpan AbsoluteSourceSpan
	Key        TemplateBindingIdentifier
	Value      *TemplateBindingIdentifier
}

func (v *VariableBinding) isTemplateBinding() {}

func NewVariableBinding(sourceSpan AbsoluteSourceSpan, key TemplateBindingIdentifier, value *TemplateBindingIdentifier) *VariableBinding {
	return &VariableBinding{SourceSpan: sourceSpan, Key: key, Value: value}
}

type ExpressionBinding struct {
	SourceSpan AbsoluteSourceSpan
	Key        TemplateBindingIdentifier
	Value      *ASTWithSource
}

func (e *ExpressionBinding) isTemplateBinding() {}

func NewExpressionBinding(sourceSpan AbsoluteSourceSpan, key TemplateBindingIdentifier, value *ASTWithSource) *ExpressionBinding {
	return &ExpressionBinding{SourceSpan: sourceSpan, Key: key, Value: value}
}

type TemplateBindingParseResult struct {
	TemplateBindings []TemplateBinding
	Warnings         []string
	Errors           []ParseError
}
func IsEmptyExpr(ast AST) bool { _, ok := ast.(*EmptyExpr); return ok }

func (a *ASTWithSource) Visit(visitor Visitor, context any) any {
	if a.Ast != nil {
		return a.Ast.Visit(visitor, context)
	}
	return nil
}
type BindingType int
const (
	BindingType_Property BindingType = iota
	BindingType_Attribute
	BindingType_Class
	BindingType_Style
	BindingType_Animation
)
