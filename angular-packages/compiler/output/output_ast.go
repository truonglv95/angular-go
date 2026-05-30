package output

import (
	"reflect"
	"strings"
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type ParseSourceSpan interface{ GetStart() ParseSourceLocation }
type Message any
type I18nMeta any

type SourceSpanAdapter struct {
	Span *parse_util.ParseSourceSpan
}

func NewSourceSpanAdapter(span *parse_util.ParseSourceSpan) ParseSourceSpan {
	if span == nil {
		return nil
	}
	return &SourceSpanAdapter{Span: span}
}

func (a *SourceSpanAdapter) GetStart() ParseSourceLocation {
	if a.Span == nil || a.Span.Start == nil {
		return nil
	}
	return &LocationAdapter{Location: a.Span.Start}
}

type LocationAdapter struct {
	Location *parse_util.ParseLocation
}

func (l *LocationAdapter) GetFile() ParseSourceFile {
	if l.Location == nil || l.Location.File == nil {
		return nil
	}
	return &FileAdapter{File: l.Location.File}
}

func (l *LocationAdapter) GetLine() int {
	if l.Location == nil {
		return 0
	}
	return l.Location.Line
}

func (l *LocationAdapter) GetCol() int {
	if l.Location == nil {
		return 0
	}
	return l.Location.Col
}

type FileAdapter struct {
	File *parse_util.ParseSourceFile
}

func (f *FileAdapter) GetUrl() string {
	if f.File == nil {
		return ""
	}
	return f.File.Url
}

func (f *FileAdapter) GetContent() string {
	if f.File == nil {
		return ""
	}
	return f.File.Content
}

type TypeModifier int

const (
	TypeModifierNone  TypeModifier = 0
	TypeModifierConst TypeModifier = 1 << 0
)

type Type interface {
	VisitType(visitor TypeVisitor, context any) any
	HasModifier(modifier TypeModifier) bool
	GetModifiers() TypeModifier
}

type BaseType struct {
	Modifiers TypeModifier
}

func (t *BaseType) HasModifier(modifier TypeModifier) bool {
	return (t.Modifiers & modifier) != 0
}
func (t *BaseType) GetModifiers() TypeModifier { return t.Modifiers }

type BuiltinTypeName int

const (
	BuiltinTypeNameDynamic BuiltinTypeName = iota
	BuiltinTypeNameBool
	BuiltinTypeNameString
	BuiltinTypeNameInt
	BuiltinTypeNameNumber
	BuiltinTypeNameFunction
	BuiltinTypeNameInferred
	BuiltinTypeNameNone
)

type BuiltinType struct {
	BaseType
	Name BuiltinTypeName
}

func NewBuiltinType(name BuiltinTypeName, modifiers ...TypeModifier) *BuiltinType {
	mod := TypeModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return &BuiltinType{BaseType: BaseType{Modifiers: mod}, Name: name}
}
func (t *BuiltinType) VisitType(visitor TypeVisitor, context any) any {
	return visitor.VisitBuiltinType(t, context)
}

type ExpressionType struct {
	BaseType
	Value      Expression
	TypeParams []Type
}

func NewExpressionType(value Expression, modifiers ...TypeModifier) *ExpressionType {
	mod := TypeModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return &ExpressionType{BaseType: BaseType{Modifiers: mod}, Value: value}
}
func NewExpressionTypeWithParams(value Expression, typeParams []Type, modifiers ...TypeModifier) *ExpressionType {
	mod := TypeModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return &ExpressionType{BaseType: BaseType{Modifiers: mod}, Value: value, TypeParams: typeParams}
}
func (t *ExpressionType) VisitType(visitor TypeVisitor, context any) any {
	return visitor.VisitExpressionType(t, context)
}

type ArrayType struct {
	BaseType
	Of Type
}

func NewArrayType(of Type, modifiers ...TypeModifier) *ArrayType {
	mod := TypeModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return &ArrayType{BaseType: BaseType{Modifiers: mod}, Of: of}
}
func (t *ArrayType) VisitType(visitor TypeVisitor, context any) any {
	return visitor.VisitArrayType(t, context)
}

type MapType struct {
	BaseType
	ValueType Type
}

func NewMapType(valueType Type, modifiers ...TypeModifier) *MapType {
	mod := TypeModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return &MapType{BaseType: BaseType{Modifiers: mod}, ValueType: valueType}
}
func (t *MapType) VisitType(visitor TypeVisitor, context any) any {
	return visitor.VisitMapType(t, context)
}

type TransplantedType struct {
	BaseType
	Type any
}

func NewTransplantedType(type_ any, modifiers ...TypeModifier) *TransplantedType {
	mod := TypeModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return &TransplantedType{BaseType: BaseType{Modifiers: mod}, Type: type_}
}
func (t *TransplantedType) VisitType(visitor TypeVisitor, context any) any {
	return visitor.VisitTransplantedType(t, context)
}

var (
	DYNAMIC_TYPE  = NewBuiltinType(BuiltinTypeNameDynamic)
	INFERRED_TYPE = NewBuiltinType(BuiltinTypeNameInferred)
	BOOL_TYPE     = NewBuiltinType(BuiltinTypeNameBool)
	INT_TYPE      = NewBuiltinType(BuiltinTypeNameInt)
	NUMBER_TYPE   = NewBuiltinType(BuiltinTypeNameNumber)
	STRING_TYPE   = NewBuiltinType(BuiltinTypeNameString)
	FUNCTION_TYPE = NewBuiltinType(BuiltinTypeNameFunction)
	NONE_TYPE     = NewBuiltinType(BuiltinTypeNameNone)
)

type TypeVisitor interface {
	VisitBuiltinType(t *BuiltinType, context any) any
	VisitExpressionType(t *ExpressionType, context any) any
	VisitArrayType(t *ArrayType, context any) any
	VisitMapType(t *MapType, context any) any
	VisitTransplantedType(t *TransplantedType, context any) any
}

type UnaryOperator int

const (
	UnaryOperatorMinus UnaryOperator = iota
	UnaryOperatorPlus
)

type BinaryOperator int

const (
	BinaryOperatorEquals BinaryOperator = iota
	BinaryOperatorNotEquals
	BinaryOperatorAssign
	BinaryOperatorIdentical
	BinaryOperatorNotIdentical
	BinaryOperatorMinus
	BinaryOperatorPlus
	BinaryOperatorDivide
	BinaryOperatorMultiply
	BinaryOperatorModulo
	BinaryOperatorAnd
	BinaryOperatorOr
	BinaryOperatorBitwiseOr
	BinaryOperatorBitwiseAnd
	BinaryOperatorLower
	BinaryOperatorLowerEquals
	BinaryOperatorBigger
	BinaryOperatorBiggerEquals
	BinaryOperatorNullishCoalesce
	BinaryOperatorExponentiation
	BinaryOperatorIn
	BinaryOperatorInstanceOf
	BinaryOperatorAdditionAssignment
	BinaryOperatorSubtractionAssignment
	BinaryOperatorMultiplicationAssignment
	BinaryOperatorDivisionAssignment
	BinaryOperatorRemainderAssignment
	BinaryOperatorExponentiationAssignment
	BinaryOperatorAndAssignment
	BinaryOperatorOrAssignment
	BinaryOperatorNullishCoalesceAssignment
)

func NullSafeIsEquivalent(base Expression, other Expression) bool {
	if base == nil || other == nil {
		return base == other
	}
	return base.IsEquivalent(other)
}
func AreAllEquivalent(base []Expression, other []Expression) bool {
	if len(base) != len(other) {
		return false
	}
	for i := range base {
		if !base[i].IsEquivalent(other[i]) {
			return false
		}
	}
	return true
}
func AreAllEquivalentStatements(base []Statement, other []Statement) bool {
	if len(base) != len(other) {
		return false
	}
	for i := range base {
		if !base[i].IsEquivalent(other[i]) {
			return false
		}
	}
	return true
}
func AreAllEquivalentFnParams(base []*FnParam, other []*FnParam) bool {
	if len(base) != len(other) {
		return false
	}
	for i := range base {
		if !base[i].IsEquivalent(other[i]) {
			return false
		}
	}
	return true
}
func AreAllEquivalentTemplateElements(base []*TemplateLiteralElementExpr, other []*TemplateLiteralElementExpr) bool {
	if len(base) != len(other) {
		return false
	}
	for i := range base {
		if base[i].Text != other[i].Text {
			return false
		}
	}
	return true
}
func AreAllEquivalentLiteralMapEntries(base []LiteralMapEntry, other []LiteralMapEntry) bool {
	if len(base) != len(other) {
		return false
	}
	for i := range base {
		if !base[i].IsEquivalent(other[i]) {
			return false
		}
	}
	return true
}

type LeadingComment struct {
	Text            string
	Multiline       bool
	TrailingNewline bool
}

func (c *LeadingComment) ToString() string {
	if c.Multiline {
		return " " + c.Text + " "
	}
	return c.Text
}

type JSDocTag struct {
	TagName string
	Text    string
}

type JSDocComment struct {
	LeadingComment
	Tags []JSDocTag
}

func NewJSDocComment(tags []JSDocTag) *JSDocComment {
	return &JSDocComment{
		LeadingComment: LeadingComment{Text: "", Multiline: true, TrailingNewline: true},
		Tags:           tags,
	}
}

type Expression interface {
	VisitExpression(visitor ExpressionVisitor, context any) any
	IsEquivalent(e Expression) bool
	IsConstant() bool
	Clone() Expression

	Prop(name string, sourceSpan ParseSourceSpan) *ReadPropExpr
	Key(index Expression, type_ Type, sourceSpan ParseSourceSpan) *ReadKeyExpr
	CallFn(params []Expression, sourceSpan ParseSourceSpan, pure bool, leadingComments []LeadingComment) *InvokeFunctionExpr
	Instantiate(params []Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *InstantiateExpr
	Conditional(trueCase Expression, falseCase Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ConditionalExpr
	Equals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	NotEquals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Identical(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	NotIdentical(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Minus(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Plus(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Divide(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Multiply(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Modulo(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Power(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	And(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	BitwiseOr(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	BitwiseAnd(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Or(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Lower(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	LowerEquals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	Bigger(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	BiggerEquals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr
	IsBlank(sourceSpan ParseSourceSpan) Expression
	NullishCoalesce(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr

	ToStmt(leadingComments []LeadingComment) Statement

	GetType() Type
	GetSourceSpan() interface{}
	GetLeadingComments() []LeadingComment
}

type BaseExpression struct {
	Self            Expression
	Type            Type
	SourceSpan      ParseSourceSpan
	LeadingComments []LeadingComment
}

func (e *BaseExpression) GetType() Type                        { return e.Type }
func (e *BaseExpression) GetSourceSpan() interface{}           { return e.SourceSpan }
func (e *BaseExpression) GetLeadingComments() []LeadingComment { return e.LeadingComments }

func (e *BaseExpression) Prop(name string, sourceSpan ParseSourceSpan) *ReadPropExpr {
	return NewReadPropExpr(e.Self, name, nil, sourceSpan, nil, false)
}
func (e *BaseExpression) Key(index Expression, type_ Type, sourceSpan ParseSourceSpan) *ReadKeyExpr {
	return NewReadKeyExpr(e.Self, index, type_, sourceSpan, nil, false)
}
func (e *BaseExpression) CallFn(params []Expression, sourceSpan ParseSourceSpan, pure bool, leadingComments []LeadingComment) *InvokeFunctionExpr {
	return NewInvokeFunctionExpr(e.Self, params, nil, sourceSpan, pure, leadingComments, false)
}
func (e *BaseExpression) Instantiate(params []Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *InstantiateExpr {
	return NewInstantiateExpr(e.Self, params, type_, sourceSpan, leadingComments)
}
func (e *BaseExpression) Conditional(trueCase Expression, falseCase Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ConditionalExpr {
	return NewConditionalExpr(e.Self, trueCase, falseCase, nil, sourceSpan, leadingComments)
}
func (e *BaseExpression) BinaryOp(op BinaryOperator, rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return NewBinaryOperatorExpr(op, e.Self, rhs, nil, sourceSpan, nil)
}
func (e *BaseExpression) Equals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorEquals, rhs, sourceSpan)
}
func (e *BaseExpression) NotEquals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorNotEquals, rhs, sourceSpan)
}
func (e *BaseExpression) Identical(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorIdentical, rhs, sourceSpan)
}
func (e *BaseExpression) NotIdentical(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorNotIdentical, rhs, sourceSpan)
}
func (e *BaseExpression) Minus(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorMinus, rhs, sourceSpan)
}
func (e *BaseExpression) Plus(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorPlus, rhs, sourceSpan)
}
func (e *BaseExpression) Divide(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorDivide, rhs, sourceSpan)
}
func (e *BaseExpression) Multiply(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorMultiply, rhs, sourceSpan)
}
func (e *BaseExpression) Modulo(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorModulo, rhs, sourceSpan)
}
func (e *BaseExpression) Power(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorExponentiation, rhs, sourceSpan)
}
func (e *BaseExpression) And(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorAnd, rhs, sourceSpan)
}
func (e *BaseExpression) BitwiseOr(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorBitwiseOr, rhs, sourceSpan)
}
func (e *BaseExpression) BitwiseAnd(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorBitwiseAnd, rhs, sourceSpan)
}
func (e *BaseExpression) Or(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorOr, rhs, sourceSpan)
}
func (e *BaseExpression) Lower(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorLower, rhs, sourceSpan)
}
func (e *BaseExpression) LowerEquals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorLowerEquals, rhs, sourceSpan)
}
func (e *BaseExpression) Bigger(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorBigger, rhs, sourceSpan)
}
func (e *BaseExpression) BiggerEquals(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorBiggerEquals, rhs, sourceSpan)
}
func (e *BaseExpression) IsBlank(sourceSpan ParseSourceSpan) Expression {
	return e.Equals(TYPED_NULL_EXPR, sourceSpan)
}
func (e *BaseExpression) NullishCoalesce(rhs Expression, sourceSpan ParseSourceSpan) *BinaryOperatorExpr {
	return e.BinaryOp(BinaryOperatorNullishCoalesce, rhs, sourceSpan)
}
func (e *BaseExpression) ToStmt(leadingComments []LeadingComment) Statement {
	return NewExpressionStatement(e.Self, nil, leadingComments)
}

type ReadVarExpr struct {
	BaseExpression
	Name string
}

func NewReadVarExpr(name string, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ReadVarExpr {
	e := &ReadVarExpr{Name: name}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *ReadVarExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*ReadVarExpr)
	return ok && e.Name == o.Name
}
func (e *ReadVarExpr) IsConstant() bool { return false }
func (e *ReadVarExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitReadVarExpr(e, context)
}
func (e *ReadVarExpr) Clone() Expression { return NewReadVarExpr(e.Name, e.Type, e.SourceSpan, nil) }
func (e *ReadVarExpr) Set(value Expression) *BinaryOperatorExpr {
	return NewBinaryOperatorExpr(BinaryOperatorAssign, e, value, nil, e.SourceSpan, nil)
}

type TypeofExpr struct {
	BaseExpression
	Expr Expression
}

func NewTypeofExpr(expr Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *TypeofExpr {
	e := &TypeofExpr{Expr: expr}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *TypeofExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*TypeofExpr)
	return ok && e.Expr.IsEquivalent(o.Expr)
}
func (e *TypeofExpr) IsConstant() bool { return e.Expr.IsConstant() }
func (e *TypeofExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitTypeofExpr(e, context)
}
func (e *TypeofExpr) Clone() Expression { return NewTypeofExpr(e.Expr.Clone(), nil, nil, nil) }

type VoidExpr struct {
	BaseExpression
	Expr Expression
}

func NewVoidExpr(expr Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *VoidExpr {
	e := &VoidExpr{Expr: expr}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *VoidExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*VoidExpr)
	return ok && e.Expr.IsEquivalent(o.Expr)
}
func (e *VoidExpr) IsConstant() bool { return e.Expr.IsConstant() }
func (e *VoidExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitVoidExpr(e, context)
}
func (e *VoidExpr) Clone() Expression { return NewVoidExpr(e.Expr.Clone(), nil, nil, nil) }

type WrappedNodeExpr struct {
	BaseExpression
	Node any
}

func NewWrappedNodeExpr(node any, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *WrappedNodeExpr {
	e := &WrappedNodeExpr{Node: node}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *WrappedNodeExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*WrappedNodeExpr)
	return ok && e.Node == o.Node
}
func (e *WrappedNodeExpr) IsConstant() bool { return false }
func (e *WrappedNodeExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitWrappedNodeExpr(e, context)
}
func (e *WrappedNodeExpr) Clone() Expression {
	return NewWrappedNodeExpr(e.Node, e.Type, e.SourceSpan, nil)
}

type InvokeFunctionExpr struct {
	BaseExpression
	Fn         Expression
	Args       []Expression
	Pure       bool
	IsOptional bool
}

func NewInvokeFunctionExpr(fn Expression, args []Expression, type_ Type, sourceSpan ParseSourceSpan, pure bool, leadingComments []LeadingComment, isOptional bool) *InvokeFunctionExpr {
	e := &InvokeFunctionExpr{Fn: fn, Args: args, Pure: pure, IsOptional: isOptional}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *InvokeFunctionExpr) Receiver() Expression { return e.Fn }
func (e *InvokeFunctionExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*InvokeFunctionExpr)
	return ok && e.Fn.IsEquivalent(o.Fn) && AreAllEquivalent(e.Args, o.Args) && e.Pure == o.Pure
}
func (e *InvokeFunctionExpr) IsConstant() bool { return false }
func (e *InvokeFunctionExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitInvokeFunctionExpr(e, context)
}
func (e *InvokeFunctionExpr) Clone() Expression {
	args := make([]Expression, len(e.Args))
	for i, a := range e.Args {
		args[i] = a.Clone()
	}
	return NewInvokeFunctionExpr(e.Fn.Clone(), args, e.Type, e.SourceSpan, e.Pure, nil, e.IsOptional)
}

type TemplateLiteralExpr struct {
	BaseExpression
	Elements    []*TemplateLiteralElementExpr
	Expressions []Expression
}

func NewTemplateLiteralExpr(elements []*TemplateLiteralElementExpr, expressions []Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *TemplateLiteralExpr {
	e := &TemplateLiteralExpr{Elements: elements, Expressions: expressions}
	e.Self = e
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *TemplateLiteralExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*TemplateLiteralExpr)
	return ok && AreAllEquivalentTemplateElements(e.Elements, o.Elements) && AreAllEquivalent(e.Expressions, o.Expressions)
}
func (e *TemplateLiteralExpr) IsConstant() bool { return false }
func (e *TemplateLiteralExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitTemplateLiteralExpr(e, context)
}
func (e *TemplateLiteralExpr) Clone() Expression {
	els := make([]*TemplateLiteralElementExpr, len(e.Elements))
	for i, el := range e.Elements {
		els[i] = el.Clone().(*TemplateLiteralElementExpr)
	}
	exps := make([]Expression, len(e.Expressions))
	for i, exp := range e.Expressions {
		exps[i] = exp.Clone()
	}
	return NewTemplateLiteralExpr(els, exps, nil, nil)
}

type TaggedTemplateLiteralExpr struct {
	BaseExpression
	Tag      Expression
	Template *TemplateLiteralExpr
}

func NewTaggedTemplateLiteralExpr(tag Expression, template *TemplateLiteralExpr, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *TaggedTemplateLiteralExpr {
	e := &TaggedTemplateLiteralExpr{Tag: tag, Template: template}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *TaggedTemplateLiteralExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*TaggedTemplateLiteralExpr)
	return ok && e.Tag.IsEquivalent(o.Tag) && e.Template.IsEquivalent(o.Template)
}
func (e *TaggedTemplateLiteralExpr) IsConstant() bool { return false }
func (e *TaggedTemplateLiteralExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitTaggedTemplateLiteralExpr(e, context)
}
func (e *TaggedTemplateLiteralExpr) Clone() Expression {
	return NewTaggedTemplateLiteralExpr(e.Tag.Clone(), e.Template.Clone().(*TemplateLiteralExpr), e.Type, e.SourceSpan, nil)
}

type InstantiateExpr struct {
	BaseExpression
	ClassExpr Expression
	Args      []Expression
}

func NewInstantiateExpr(classExpr Expression, args []Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *InstantiateExpr {
	e := &InstantiateExpr{ClassExpr: classExpr, Args: args}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *InstantiateExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*InstantiateExpr)
	return ok && e.ClassExpr.IsEquivalent(o.ClassExpr) && AreAllEquivalent(e.Args, o.Args)
}
func (e *InstantiateExpr) IsConstant() bool { return false }
func (e *InstantiateExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitInstantiateExpr(e, context)
}
func (e *InstantiateExpr) Clone() Expression {
	args := make([]Expression, len(e.Args))
	for i, a := range e.Args {
		args[i] = a.Clone()
	}
	return NewInstantiateExpr(e.ClassExpr.Clone(), args, e.Type, e.SourceSpan, nil)
}

type RegularExpressionLiteralExpr struct {
	BaseExpression
	Body  string
	Flags *string
}

func NewRegularExpressionLiteralExpr(body string, flags *string, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *RegularExpressionLiteralExpr {
	e := &RegularExpressionLiteralExpr{Body: body, Flags: flags}
	e.Self = e
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *RegularExpressionLiteralExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*RegularExpressionLiteralExpr)
	if !ok {
		return false
	}
	if e.Body != o.Body {
		return false
	}
	if e.Flags == nil && o.Flags == nil {
		return true
	}
	if e.Flags != nil && o.Flags != nil && *e.Flags == *o.Flags {
		return true
	}
	return false
}
func (e *RegularExpressionLiteralExpr) IsConstant() bool { return true }
func (e *RegularExpressionLiteralExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitRegularExpressionLiteral(e, context)
}
func (e *RegularExpressionLiteralExpr) Clone() Expression {
	return NewRegularExpressionLiteralExpr(e.Body, e.Flags, e.SourceSpan, nil)
}

type LiteralExpr struct {
	BaseExpression
	Value any
}

func NewLiteralExpr(value any, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *LiteralExpr {
	e := &LiteralExpr{Value: value}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *LiteralExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*LiteralExpr)
	return ok && e.Value == o.Value
}
func (e *LiteralExpr) IsConstant() bool { return true }
func (e *LiteralExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitLiteralExpr(e, context)
}
func (e *LiteralExpr) Clone() Expression { return NewLiteralExpr(e.Value, e.Type, e.SourceSpan, nil) }

type TemplateLiteralElementExpr struct {
	BaseExpression
	Text    string
	RawText string
}

func NewTemplateLiteralElementExpr(text string, sourceSpan ParseSourceSpan, rawText *string, leadingComments []LeadingComment) *TemplateLiteralElementExpr {
	rt := ""
	if rawText != nil {
		rt = *rawText
	} else {
		rt = text
	} // Escape logic omitted for simplicity
	e := &TemplateLiteralElementExpr{Text: text, RawText: rt}
	e.Self = e
	e.Type = STRING_TYPE
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *TemplateLiteralElementExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitTemplateLiteralElementExpr(e, context)
}
func (e *TemplateLiteralElementExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*TemplateLiteralElementExpr)
	return ok && e.Text == o.Text && e.RawText == o.RawText
}
func (e *TemplateLiteralElementExpr) IsConstant() bool { return true }
func (e *TemplateLiteralElementExpr) Clone() Expression {
	rt := e.RawText
	return NewTemplateLiteralElementExpr(e.Text, e.SourceSpan, &rt, nil)
}

type LiteralPiece struct {
	Text       string
	SourceSpan ParseSourceSpan
}
type PlaceholderPiece struct {
	Text              string
	SourceSpan        ParseSourceSpan
	AssociatedMessage Message
}
type CookedRawString struct {
	Cooked string
	Raw    string
	Range  ParseSourceSpan
}

type LocalizedString struct {
	BaseExpression
	MetaBlock        I18nMeta
	MessageParts     []LiteralPiece
	PlaceHolderNames []PlaceholderPiece
	Expressions      []Expression
}

func NewLocalizedString(metaBlock I18nMeta, messageParts []LiteralPiece, placeHolderNames []PlaceholderPiece, expressions []Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *LocalizedString {
	e := &LocalizedString{MetaBlock: metaBlock, MessageParts: messageParts, PlaceHolderNames: placeHolderNames, Expressions: expressions}
	e.Self = e
	e.Type = STRING_TYPE
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *LocalizedString) IsEquivalent(other Expression) bool { return false }
func (e *LocalizedString) IsConstant() bool                   { return false }
func (e *LocalizedString) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitLocalizedString(e, context)
}
func (e *LocalizedString) Clone() Expression {
	exps := make([]Expression, len(e.Expressions))
	for i, exp := range e.Expressions {
		exps[i] = exp.Clone()
	}
	return NewLocalizedString(e.MetaBlock, e.MessageParts, e.PlaceHolderNames, exps, e.SourceSpan, nil)
}
const MEANING_SEPARATOR = "|"
const ID_SEPARATOR = "@@"
const LEGACY_ID_INDICATOR = "␟"

func escapeSlashes(str string) string {
	return strings.ReplaceAll(str, "\\", "\\\\")
}

func escapeStartingColon(str string) string {
	if strings.HasPrefix(str, ":") {
		return "\\:" + str[1:]
	}
	return str
}

func escapeColons(str string) string {
	return strings.ReplaceAll(str, ":", "\\:")
}

func escapeForTemplateLiteral(str string) string {
	str = strings.ReplaceAll(str, "`", "\\`")
	str = strings.ReplaceAll(str, "${", "$\\{")
	return str
}

func createCookedRawString(metaBlock string, messagePart string, span ParseSourceSpan) CookedRawString {
	if metaBlock == "" {
		return CookedRawString{
			Cooked: messagePart,
			Raw:    escapeForTemplateLiteral(escapeStartingColon(escapeSlashes(messagePart))),
			Range:  span,
		}
	}
	return CookedRawString{
		Cooked: ":" + metaBlock + ":" + messagePart,
		Raw:    escapeForTemplateLiteral(":" + escapeColons(escapeSlashes(metaBlock)) + ":" + escapeSlashes(messagePart)),
		Range:  span,
	}
}

func getI18nMetaInfo(meta any) (description string, meaning string, customId string, legacyIds []string) {
	if meta == nil {
		return "", "", "", nil
	}
	val := reflect.ValueOf(meta)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		if val.Kind() == reflect.Map {
			descVal := val.MapIndex(reflect.ValueOf("description"))
			if descVal.IsValid() && !descVal.IsNil() {
				description = descVal.Elem().String()
			}
			meaningVal := val.MapIndex(reflect.ValueOf("meaning"))
			if meaningVal.IsValid() && !meaningVal.IsNil() {
				meaning = meaningVal.Elem().String()
			}
			customIdVal := val.MapIndex(reflect.ValueOf("customId"))
			if customIdVal.IsValid() && !customIdVal.IsNil() {
				customId = customIdVal.Elem().String()
			}
			legacyIdsVal := val.MapIndex(reflect.ValueOf("legacyIds"))
			if legacyIdsVal.IsValid() && !legacyIdsVal.IsNil() {
				if ids, ok := legacyIdsVal.Interface().([]string); ok {
					legacyIds = ids
				}
			}
			return description, meaning, customId, legacyIds
		}
		return "", "", "", nil
	}

	descField := val.FieldByName("Description")
	if !descField.IsValid() {
		descField = val.FieldByName("description")
	}
	meaningField := val.FieldByName("Meaning")
	if !meaningField.IsValid() {
		meaningField = val.FieldByName("meaning")
	}
	customIdField := val.FieldByName("CustomId")
	if !customIdField.IsValid() {
		customIdField = val.FieldByName("customId")
	}
	legacyIdsField := val.FieldByName("LegacyIds")
	if !legacyIdsField.IsValid() {
		legacyIdsField = val.FieldByName("legacyIds")
	}

	if descField.IsValid() {
		description = descField.String()
	}
	if meaningField.IsValid() {
		meaning = meaningField.String()
	}
	if customIdField.IsValid() {
		customId = customIdField.String()
	}
	if legacyIdsField.IsValid() {
		if ids, ok := legacyIdsField.Interface().([]string); ok {
			legacyIds = ids
		}
	}
	return description, meaning, customId, legacyIds
}

func getAssociatedMsgInfo(msg any) (legacyIds []string, messageString string, meaning string, hasMsg bool) {
	if msg == nil {
		return nil, "", "", false
	}
	val := reflect.ValueOf(msg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		if val.Kind() == reflect.Map {
			legacyIdsVal := val.MapIndex(reflect.ValueOf("legacyIds"))
			if legacyIdsVal.IsValid() && !legacyIdsVal.IsNil() {
				if ids, ok := legacyIdsVal.Interface().([]string); ok {
					legacyIds = ids
				}
			}
			msgStrVal := val.MapIndex(reflect.ValueOf("messageString"))
			if msgStrVal.IsValid() && !msgStrVal.IsNil() {
				messageString = msgStrVal.Elem().String()
			}
			meaningVal := val.MapIndex(reflect.ValueOf("meaning"))
			if meaningVal.IsValid() && !meaningVal.IsNil() {
				meaning = meaningVal.Elem().String()
			}
			return legacyIds, messageString, meaning, true
		}
		return nil, "", "", false
	}

	legacyIdsField := val.FieldByName("LegacyIds")
	if !legacyIdsField.IsValid() {
		legacyIdsField = val.FieldByName("legacyIds")
	}
	msgStrField := val.FieldByName("MessageString")
	if !msgStrField.IsValid() {
		msgStrField = val.FieldByName("messageString")
	}
	meaningField := val.FieldByName("Meaning")
	if !meaningField.IsValid() {
		meaningField = val.FieldByName("meaning")
	}

	if legacyIdsField.IsValid() && msgStrField.IsValid() && meaningField.IsValid() {
		if ids, ok := legacyIdsField.Interface().([]string); ok {
			legacyIds = ids
		}
		messageString = msgStrField.String()
		meaning = meaningField.String()
		return legacyIds, messageString, meaning, true
	}
	return nil, "", "", false
}

func (e *LocalizedString) SerializeI18nHead() CookedRawString {
	description, meaning, customId, legacyIds := getI18nMetaInfo(e.MetaBlock)
	metaBlock := description
	if meaning != "" {
		metaBlock = meaning + MEANING_SEPARATOR + metaBlock
	}
	if customId != "" {
		metaBlock = metaBlock + ID_SEPARATOR + customId
	}
	for _, legacyId := range legacyIds {
		metaBlock = metaBlock + LEGACY_ID_INDICATOR + legacyId
	}

	messagePartText := ""
	if len(e.MessageParts) > 0 {
		messagePartText = e.MessageParts[0].Text
	}

	return createCookedRawString(
		metaBlock,
		messagePartText,
		e.GetMessagePartSourceSpan(0),
	)
}

func (e *LocalizedString) GetMessagePartSourceSpan(i int) ParseSourceSpan {
	if i >= 0 && i < len(e.MessageParts) && e.MessageParts[i].SourceSpan != nil {
		return e.MessageParts[i].SourceSpan
	}
	return e.SourceSpan
}

func (e *LocalizedString) GetPlaceholderSourceSpan(i int) ParseSourceSpan {
	if i >= 0 && i < len(e.PlaceHolderNames) && e.PlaceHolderNames[i].SourceSpan != nil {
		return e.PlaceHolderNames[i].SourceSpan
	}
	if i >= 0 && i < len(e.Expressions) && e.Expressions[i].GetSourceSpan() != nil {
		if span, ok := e.Expressions[i].GetSourceSpan().(ParseSourceSpan); ok {
			return span
		}
	}
	return e.SourceSpan
}

func (e *LocalizedString) SerializeI18nTemplatePart(partIndex int) CookedRawString {
	placeholder := e.PlaceHolderNames[partIndex-1]
	messagePart := e.MessageParts[partIndex]
	metaBlock := placeholder.Text
	
	if legacyIds, messageString, meaning, hasMsg := getAssociatedMsgInfo(placeholder.AssociatedMessage); hasMsg {
		if len(legacyIds) == 0 {
			metaBlock += ID_SEPARATOR + i18n.ComputeMsgId(messageString, meaning)
		}
	}
	
	return createCookedRawString(
		metaBlock,
		messagePart.Text,
		e.GetMessagePartSourceSpan(partIndex),
	)
}

type ExternalReference struct {
	ModuleName *string
	Name       *string
}
type ExternalExpr struct {
	BaseExpression
	Value      ExternalReference
	TypeParams []Type
}

func NewExternalExpr(value ExternalReference, type_ Type, typeParams []Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ExternalExpr {
	e := &ExternalExpr{Value: value, TypeParams: typeParams}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *ExternalExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*ExternalExpr)
	return ok && e.Value.Name == o.Value.Name && e.Value.ModuleName == o.Value.ModuleName
}
func (e *ExternalExpr) IsConstant() bool { return false }
func (e *ExternalExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitExternalExpr(e, context)
}
func (e *ExternalExpr) Clone() Expression {
	return NewExternalExpr(e.Value, e.Type, e.TypeParams, e.SourceSpan, nil)
}

type ConditionalExpr struct {
	BaseExpression
	Condition Expression
	TrueCase  Expression
	FalseCase Expression
}

func NewConditionalExpr(condition Expression, trueCase Expression, falseCase Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ConditionalExpr {
	e := &ConditionalExpr{Condition: condition, TrueCase: trueCase, FalseCase: falseCase}
	e.Self = e
	if type_ != nil {
		e.Type = type_
	} else {
		e.Type = trueCase.GetType()
	}
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *ConditionalExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*ConditionalExpr)
	return ok && e.Condition.IsEquivalent(o.Condition) && e.TrueCase.IsEquivalent(o.TrueCase) && NullSafeIsEquivalent(e.FalseCase, o.FalseCase)
}
func (e *ConditionalExpr) IsConstant() bool { return false }
func (e *ConditionalExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitConditionalExpr(e, context)
}
func (e *ConditionalExpr) Clone() Expression {
	var f Expression
	if e.FalseCase != nil {
		f = e.FalseCase.Clone()
	}
	return NewConditionalExpr(e.Condition.Clone(), e.TrueCase.Clone(), f, e.Type, e.SourceSpan, nil)
}

type DynamicImportExpr struct {
	BaseExpression
	Url        any // string | Expression
	UrlComment *string
}

func NewDynamicImportExpr(url any, sourceSpan ParseSourceSpan, urlComment *string, leadingComments []LeadingComment) *DynamicImportExpr {
	e := &DynamicImportExpr{Url: url, UrlComment: urlComment}
	e.Self = e
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *DynamicImportExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*DynamicImportExpr)
	if !ok {
		return false
	}
	if e.UrlComment != o.UrlComment {
		return false
	}
	if urlStr, ok1 := e.Url.(string); ok1 {
		if oUrlStr, ok2 := o.Url.(string); ok2 {
			return urlStr == oUrlStr
		}
		return false
	}
	if urlExpr, ok1 := e.Url.(Expression); ok1 {
		if oUrlExpr, ok2 := o.Url.(Expression); ok2 {
			return urlExpr.IsEquivalent(oUrlExpr)
		}
		return false
	}
	return false
}
func (e *DynamicImportExpr) IsConstant() bool { return false }
func (e *DynamicImportExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitDynamicImportExpr(e, context)
}
func (e *DynamicImportExpr) Clone() Expression {
	var u any
	if us, ok := e.Url.(string); ok {
		u = us
	} else {
		u = e.Url.(Expression).Clone()
	}
	return NewDynamicImportExpr(u, e.SourceSpan, e.UrlComment, nil)
}

type NotExpr struct {
	BaseExpression
	Condition Expression
}

func NewNotExpr(condition Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *NotExpr {
	e := &NotExpr{Condition: condition}
	e.Self = e
	e.Type = BOOL_TYPE
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *NotExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*NotExpr)
	return ok && e.Condition.IsEquivalent(o.Condition)
}
func (e *NotExpr) IsConstant() bool { return false }
func (e *NotExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitNotExpr(e, context)
}
func (e *NotExpr) Clone() Expression { return NewNotExpr(e.Condition.Clone(), e.SourceSpan, nil) }

type FnParam struct {
	Name string
	Type Type
}

func NewFnParam(name string, type_ Type) *FnParam   { return &FnParam{Name: name, Type: type_} }
func (p *FnParam) IsEquivalent(param *FnParam) bool { return p.Name == param.Name }
func (p *FnParam) Clone() *FnParam                  { return NewFnParam(p.Name, p.Type) }

type FunctionExpr struct {
	BaseExpression
	Params     []*FnParam
	Statements []Statement
	Name       *string
}

func NewFunctionExpr(params []*FnParam, statements []Statement, type_ Type, sourceSpan ParseSourceSpan, name *string, leadingComments []LeadingComment) *FunctionExpr {
	e := &FunctionExpr{Params: params, Statements: statements, Name: name}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *FunctionExpr) IsEquivalent(other Expression) bool {
	if o, ok := other.(*FunctionExpr); ok {
		return AreAllEquivalentFnParams(e.Params, o.Params) && AreAllEquivalentStatements(e.Statements, o.Statements)
	}
	// Note: TS says e instanceof DeclareFunctionStmt too
	return false
}
func (e *FunctionExpr) IsConstant() bool { return false }
func (e *FunctionExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitFunctionExpr(e, context)
}
func (e *FunctionExpr) ToDeclStmt(name string, modifiers ...StmtModifier) *DeclareFunctionStmt {
	mod := StmtModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return NewDeclareFunctionStmt(name, e.Params, e.Statements, e.Type, mod, e.SourceSpan, nil)
}
func (e *FunctionExpr) Clone() Expression {
	params := make([]*FnParam, len(e.Params))
	for i, p := range e.Params {
		params[i] = p.Clone()
	}
	return NewFunctionExpr(params, e.Statements, e.Type, e.SourceSpan, e.Name, nil)
}

type ArrowFunctionExpr struct {
	BaseExpression
	Params []*FnParam
	Body   any // Expression | []Statement
}

func NewArrowFunctionExpr(params []*FnParam, body any, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ArrowFunctionExpr {
	e := &ArrowFunctionExpr{Params: params, Body: body}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *ArrowFunctionExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*ArrowFunctionExpr)
	if !ok {
		return false
	}
	if !AreAllEquivalentFnParams(e.Params, o.Params) {
		return false
	}
	if eb, ok1 := e.Body.(Expression); ok1 {
		if ob, ok2 := o.Body.(Expression); ok2 {
			return eb.IsEquivalent(ob)
		}
		return false
	}
	if eb, ok1 := e.Body.([]Statement); ok1 {
		if ob, ok2 := o.Body.([]Statement); ok2 {
			return AreAllEquivalentStatements(eb, ob)
		}
		return false
	}
	return false
}
func (e *ArrowFunctionExpr) IsConstant() bool { return false }
func (e *ArrowFunctionExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitArrowFunctionExpr(e, context)
}
func (e *ArrowFunctionExpr) Clone() Expression {
	params := make([]*FnParam, len(e.Params))
	for i, p := range e.Params {
		params[i] = p.Clone()
	}
	var b any
	if be, ok := e.Body.(Expression); ok {
		b = be.Clone()
	} else {
		b = e.Body
	}
	return NewArrowFunctionExpr(params, b, e.Type, e.SourceSpan, nil)
}
func (e *ArrowFunctionExpr) ToDeclStmt(name string, modifiers ...StmtModifier) *DeclareVarStmt {
	mod := StmtModifierNone
	if len(modifiers) > 0 {
		mod = modifiers[0]
	}
	return NewDeclareVarStmt(name, e, INFERRED_TYPE, mod, e.SourceSpan, nil)
}

type UnaryOperatorExpr struct {
	BaseExpression
	Operator UnaryOperator
	Expr     Expression
	Parens   bool
}

func NewUnaryOperatorExpr(operator UnaryOperator, expr Expression, type_ Type, sourceSpan ParseSourceSpan, parens bool, leadingComments []LeadingComment) *UnaryOperatorExpr {
	e := &UnaryOperatorExpr{Operator: operator, Expr: expr, Parens: parens}
	e.Self = e
	if type_ != nil {
		e.Type = type_
	} else {
		e.Type = NUMBER_TYPE
	}
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *UnaryOperatorExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*UnaryOperatorExpr)
	return ok && e.Operator == o.Operator && e.Expr.IsEquivalent(o.Expr)
}
func (e *UnaryOperatorExpr) IsConstant() bool { return false }
func (e *UnaryOperatorExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitUnaryOperatorExpr(e, context)
}
func (e *UnaryOperatorExpr) Clone() Expression {
	return NewUnaryOperatorExpr(e.Operator, e.Expr.Clone(), e.Type, e.SourceSpan, e.Parens, nil)
}

type ParenthesizedExpr struct {
	BaseExpression
	Expr Expression
}

func NewParenthesizedExpr(expr Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ParenthesizedExpr {
	e := &ParenthesizedExpr{Expr: expr}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *ParenthesizedExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitParenthesizedExpr(e, context)
}
func (e *ParenthesizedExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*ParenthesizedExpr)
	return ok && e.Expr.IsEquivalent(o.Expr)
}
func (e *ParenthesizedExpr) IsConstant() bool { return e.Expr.IsConstant() }
func (e *ParenthesizedExpr) Clone() Expression {
	return NewParenthesizedExpr(e.Expr.Clone(), nil, nil, nil)
}

type BinaryOperatorExpr struct {
	BaseExpression
	Operator BinaryOperator
	Lhs      Expression
	Rhs      Expression
}

func NewBinaryOperatorExpr(operator BinaryOperator, lhs Expression, rhs Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *BinaryOperatorExpr {
	e := &BinaryOperatorExpr{Operator: operator, Lhs: lhs, Rhs: rhs}
	e.Self = e
	if type_ != nil {
		e.Type = type_
	} else {
		e.Type = lhs.GetType()
	}
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *BinaryOperatorExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*BinaryOperatorExpr)
	return ok && e.Operator == o.Operator && e.Lhs.IsEquivalent(o.Lhs) && e.Rhs.IsEquivalent(o.Rhs)
}
func (e *BinaryOperatorExpr) IsConstant() bool { return false }
func (e *BinaryOperatorExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitBinaryOperatorExpr(e, context)
}
func (e *BinaryOperatorExpr) Clone() Expression {
	return NewBinaryOperatorExpr(e.Operator, e.Lhs.Clone(), e.Rhs.Clone(), e.Type, e.SourceSpan, nil)
}
func (e *BinaryOperatorExpr) IsAssignment() bool {
	op := e.Operator
	return op == BinaryOperatorAssign ||
		op == BinaryOperatorAdditionAssignment ||
		op == BinaryOperatorSubtractionAssignment ||
		op == BinaryOperatorMultiplicationAssignment ||
		op == BinaryOperatorDivisionAssignment ||
		op == BinaryOperatorRemainderAssignment ||
		op == BinaryOperatorExponentiationAssignment ||
		op == BinaryOperatorAndAssignment ||
		op == BinaryOperatorOrAssignment ||
		op == BinaryOperatorNullishCoalesceAssignment
}

type ReadPropExpr struct {
	BaseExpression
	Receiver   Expression
	Name       string
	IsOptional bool
}

func NewReadPropExpr(receiver Expression, name string, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment, isOptional bool) *ReadPropExpr {
	e := &ReadPropExpr{Receiver: receiver, Name: name, IsOptional: isOptional}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *ReadPropExpr) Index() string { return e.Name }
func (e *ReadPropExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*ReadPropExpr)
	return ok && e.Receiver.IsEquivalent(o.Receiver) && e.Name == o.Name && e.IsOptional == o.IsOptional
}
func (e *ReadPropExpr) IsConstant() bool { return false }
func (e *ReadPropExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitReadPropExpr(e, context)
}
func (e *ReadPropExpr) Set(value Expression) *BinaryOperatorExpr {
	return NewBinaryOperatorExpr(BinaryOperatorAssign, e.Receiver.Prop(e.Name, nil), value, nil, e.SourceSpan, nil)
}
func (e *ReadPropExpr) Clone() Expression {
	return NewReadPropExpr(e.Receiver.Clone(), e.Name, e.Type, e.SourceSpan, nil, e.IsOptional)
}

type ReadKeyExpr struct {
	BaseExpression
	Receiver   Expression
	Index      Expression
	IsOptional bool
}

func NewReadKeyExpr(receiver Expression, index Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment, isOptional bool) *ReadKeyExpr {
	e := &ReadKeyExpr{Receiver: receiver, Index: index, IsOptional: isOptional}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *ReadKeyExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*ReadKeyExpr)
	return ok && e.Receiver.IsEquivalent(o.Receiver) && e.Index.IsEquivalent(o.Index) && e.IsOptional == o.IsOptional
}
func (e *ReadKeyExpr) IsConstant() bool { return false }
func (e *ReadKeyExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitReadKeyExpr(e, context)
}
func (e *ReadKeyExpr) Set(value Expression) *BinaryOperatorExpr {
	return NewBinaryOperatorExpr(BinaryOperatorAssign, e.Receiver.Key(e.Index, nil, nil), value, nil, e.SourceSpan, nil)
}
func (e *ReadKeyExpr) Clone() Expression {
	return NewReadKeyExpr(e.Receiver.Clone(), e.Index.Clone(), e.Type, e.SourceSpan, nil, e.IsOptional)
}

type LiteralArrayExpr struct {
	BaseExpression
	Entries []Expression
}

func NewLiteralArrayExpr(entries []Expression, type_ Type, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *LiteralArrayExpr {
	e := &LiteralArrayExpr{Entries: entries}
	e.Self = e
	e.Type = type_
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *LiteralArrayExpr) IsConstant() bool {
	for _, entry := range e.Entries {
		if !entry.IsConstant() {
			return false
		}
	}
	return true
}
func (e *LiteralArrayExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*LiteralArrayExpr)
	return ok && AreAllEquivalent(e.Entries, o.Entries)
}
func (e *LiteralArrayExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitLiteralArrayExpr(e, context)
}
func (e *LiteralArrayExpr) Clone() Expression {
	entries := make([]Expression, len(e.Entries))
	for i, ent := range e.Entries {
		entries[i] = ent.Clone()
	}
	return NewLiteralArrayExpr(entries, e.Type, e.SourceSpan, nil)
}

type LiteralMapEntry interface {
	IsEquivalent(e LiteralMapEntry) bool
	Clone() LiteralMapEntry
	IsConstant() bool
}

type LiteralMapPropertyAssignment struct {
	Key    string
	Value  Expression
	Quoted bool
}

func NewLiteralMapPropertyAssignment(key string, value Expression, quoted bool) *LiteralMapPropertyAssignment {
	return &LiteralMapPropertyAssignment{Key: key, Value: value, Quoted: quoted}
}
func (e *LiteralMapPropertyAssignment) IsEquivalent(o LiteralMapEntry) bool {
	other, ok := o.(*LiteralMapPropertyAssignment)
	return ok && e.Key == other.Key && e.Value.IsEquivalent(other.Value)
}
func (e *LiteralMapPropertyAssignment) Clone() LiteralMapEntry {
	return NewLiteralMapPropertyAssignment(e.Key, e.Value.Clone(), e.Quoted)
}
func (e *LiteralMapPropertyAssignment) IsConstant() bool { return e.Value.IsConstant() }

type LiteralMapSpreadAssignment struct {
	Expression Expression
}

func NewLiteralMapSpreadAssignment(expression Expression) *LiteralMapSpreadAssignment {
	return &LiteralMapSpreadAssignment{Expression: expression}
}
func (e *LiteralMapSpreadAssignment) IsEquivalent(o LiteralMapEntry) bool {
	other, ok := o.(*LiteralMapSpreadAssignment)
	return ok && e.Expression.IsEquivalent(other.Expression)
}
func (e *LiteralMapSpreadAssignment) Clone() LiteralMapEntry {
	return NewLiteralMapSpreadAssignment(e.Expression.Clone())
}
func (e *LiteralMapSpreadAssignment) IsConstant() bool { return e.Expression.IsConstant() }

type LiteralMapExpr struct {
	BaseExpression
	Entries   []LiteralMapEntry
	ValueType Type
}

func NewLiteralMapExpr(entries []LiteralMapEntry, type_ *MapType, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *LiteralMapExpr {
	e := &LiteralMapExpr{Entries: entries}
	e.Self = e
	if type_ != nil {
		e.Type = type_
		e.ValueType = type_.ValueType
	}
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *LiteralMapExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*LiteralMapExpr)
	return ok && AreAllEquivalentLiteralMapEntries(e.Entries, o.Entries)
}
func (e *LiteralMapExpr) IsConstant() bool {
	for _, entry := range e.Entries {
		if !entry.IsConstant() {
			return false
		}
	}
	return true
}
func (e *LiteralMapExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitLiteralMapExpr(e, context)
}
func (e *LiteralMapExpr) Clone() Expression {
	entries := make([]LiteralMapEntry, len(e.Entries))
	for i, ent := range e.Entries {
		entries[i] = ent.Clone()
	}
	var t *MapType
	if e.Type != nil {
		t = e.Type.(*MapType)
	}
	return NewLiteralMapExpr(entries, t, e.SourceSpan, nil)
}

type CommaExpr struct {
	BaseExpression
	Parts []Expression
}

func NewCommaExpr(parts []Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *CommaExpr {
	e := &CommaExpr{Parts: parts}
	e.Self = e
	if len(parts) > 0 {
		e.Type = parts[len(parts)-1].GetType()
	}
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *CommaExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*CommaExpr)
	return ok && AreAllEquivalent(e.Parts, o.Parts)
}
func (e *CommaExpr) IsConstant() bool { return false }
func (e *CommaExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitCommaExpr(e, context)
}
func (e *CommaExpr) Clone() Expression {
	parts := make([]Expression, len(e.Parts))
	for i, p := range e.Parts {
		parts[i] = p.Clone()
	}
	return NewCommaExpr(parts, e.SourceSpan, nil)
}

type SpreadElementExpr struct {
	BaseExpression
	Expression Expression
}

func NewSpreadElementExpr(expression Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *SpreadElementExpr {
	e := &SpreadElementExpr{Expression: expression}
	e.Self = e
	e.SourceSpan = sourceSpan
	e.LeadingComments = leadingComments
	return e
}
func (e *SpreadElementExpr) IsEquivalent(other Expression) bool {
	o, ok := other.(*SpreadElementExpr)
	return ok && e.Expression.IsEquivalent(o.Expression)
}
func (e *SpreadElementExpr) IsConstant() bool { return e.Expression.IsConstant() }
func (e *SpreadElementExpr) VisitExpression(v ExpressionVisitor, context any) any {
	return v.VisitSpreadElementExpr(e, context)
}
func (e *SpreadElementExpr) Clone() Expression {
	return NewSpreadElementExpr(e.Expression.Clone(), e.SourceSpan, nil)
}

type ExpressionVisitor interface {
	VisitReadVarExpr(ast *ReadVarExpr, context any) any
	VisitInvokeFunctionExpr(ast *InvokeFunctionExpr, context any) any
	VisitTaggedTemplateLiteralExpr(ast *TaggedTemplateLiteralExpr, context any) any
	VisitTemplateLiteralExpr(ast *TemplateLiteralExpr, context any) any
	VisitTemplateLiteralElementExpr(ast *TemplateLiteralElementExpr, context any) any
	VisitInstantiateExpr(ast *InstantiateExpr, context any) any
	VisitLiteralExpr(ast *LiteralExpr, context any) any
	VisitLocalizedString(ast *LocalizedString, context any) any
	VisitExternalExpr(ast *ExternalExpr, context any) any
	VisitConditionalExpr(ast *ConditionalExpr, context any) any
	VisitDynamicImportExpr(ast *DynamicImportExpr, context any) any
	VisitNotExpr(ast *NotExpr, context any) any
	VisitFunctionExpr(ast *FunctionExpr, context any) any
	VisitUnaryOperatorExpr(ast *UnaryOperatorExpr, context any) any
	VisitBinaryOperatorExpr(ast *BinaryOperatorExpr, context any) any
	VisitReadPropExpr(ast *ReadPropExpr, context any) any
	VisitReadKeyExpr(ast *ReadKeyExpr, context any) any
	VisitLiteralArrayExpr(ast *LiteralArrayExpr, context any) any
	VisitLiteralMapExpr(ast *LiteralMapExpr, context any) any
	VisitCommaExpr(ast *CommaExpr, context any) any
	VisitWrappedNodeExpr(ast *WrappedNodeExpr, context any) any
	VisitTypeofExpr(ast *TypeofExpr, context any) any
	VisitVoidExpr(ast *VoidExpr, context any) any
	VisitArrowFunctionExpr(ast *ArrowFunctionExpr, context any) any
	VisitParenthesizedExpr(ast *ParenthesizedExpr, context any) any
	VisitRegularExpressionLiteral(ast *RegularExpressionLiteralExpr, context any) any
	VisitSpreadElementExpr(ast *SpreadElementExpr, context any) any
}

var (
	NULL_EXPR       = NewLiteralExpr(nil, nil, nil, nil)
	TYPED_NULL_EXPR = NewLiteralExpr(nil, INFERRED_TYPE, nil, nil)
)

// // Statements
type StmtModifier int

const (
	StmtModifierNone     StmtModifier = 0
	StmtModifierFinal    StmtModifier = 1 << 0
	StmtModifierPrivate  StmtModifier = 1 << 1
	StmtModifierExported StmtModifier = 1 << 2
	StmtModifierStatic   StmtModifier = 1 << 3
)

type Statement interface {
	IsEquivalent(stmt Statement) bool
	VisitStatement(visitor StatementVisitor, context any) any
	HasModifier(modifier StmtModifier) bool
	AddLeadingComment(leadingComment LeadingComment)
	GetModifiers() StmtModifier
	GetSourceSpan() interface{}
	GetLeadingComments() []LeadingComment
}

type BaseStatement struct {
	Modifiers       StmtModifier
	SourceSpan      ParseSourceSpan
	LeadingComments []LeadingComment
}

func (s *BaseStatement) HasModifier(modifier StmtModifier) bool { return (s.Modifiers & modifier) != 0 }
func (s *BaseStatement) AddLeadingComment(leadingComment LeadingComment) {
	s.LeadingComments = append(s.LeadingComments, leadingComment)
}
func (s *BaseStatement) GetModifiers() StmtModifier           { return s.Modifiers }
func (s *BaseStatement) GetSourceSpan() interface{}           { return s.SourceSpan }
func (s *BaseStatement) GetLeadingComments() []LeadingComment { return s.LeadingComments }

type DeclareVarStmt struct {
	BaseStatement
	Name  string
	Value Expression
	Type  Type
}

func NewDeclareVarStmt(name string, value Expression, type_ Type, modifiers StmtModifier, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *DeclareVarStmt {
	s := &DeclareVarStmt{Name: name, Value: value}
	s.Modifiers = modifiers
	s.SourceSpan = sourceSpan
	s.LeadingComments = leadingComments
	if type_ != nil {
		s.Type = type_
	} else if value != nil {
		s.Type = value.GetType()
	}
	return s
}
func (s *DeclareVarStmt) IsEquivalent(stmt Statement) bool {
	o, ok := stmt.(*DeclareVarStmt)
	if !ok || s.Name != o.Name {
		return false
	}
	if s.Value != nil {
		return o.Value != nil && s.Value.IsEquivalent(o.Value)
	}
	return o.Value == nil
}
func (s *DeclareVarStmt) VisitStatement(v StatementVisitor, context any) any {
	return v.VisitDeclareVarStmt(s, context)
}

type DeclareFunctionStmt struct {
	BaseStatement
	Name       string
	Params     []*FnParam
	Statements []Statement
	Type       Type
}

func NewDeclareFunctionStmt(name string, params []*FnParam, statements []Statement, type_ Type, modifiers StmtModifier, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *DeclareFunctionStmt {
	s := &DeclareFunctionStmt{Name: name, Params: params, Statements: statements, Type: type_}
	s.Modifiers = modifiers
	s.SourceSpan = sourceSpan
	s.LeadingComments = leadingComments
	return s
}
func (s *DeclareFunctionStmt) IsEquivalent(stmt Statement) bool {
	o, ok := stmt.(*DeclareFunctionStmt)
	return ok && AreAllEquivalentFnParams(s.Params, o.Params) && AreAllEquivalentStatements(s.Statements, o.Statements)
}
func (s *DeclareFunctionStmt) VisitStatement(v StatementVisitor, context any) any {
	return v.VisitDeclareFunctionStmt(s, context)
}

type ExpressionStatement struct {
	BaseStatement
	Expr Expression
}

func NewExpressionStatement(expr Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ExpressionStatement {
	s := &ExpressionStatement{Expr: expr}
	s.Modifiers = StmtModifierNone
	s.SourceSpan = sourceSpan
	s.LeadingComments = leadingComments
	return s
}
func (s *ExpressionStatement) IsEquivalent(stmt Statement) bool {
	o, ok := stmt.(*ExpressionStatement)
	return ok && s.Expr.IsEquivalent(o.Expr)
}
func (s *ExpressionStatement) VisitStatement(v StatementVisitor, context any) any {
	return v.VisitExpressionStmt(s, context)
}

type ReturnStatement struct {
	BaseStatement
	Value Expression
}

func NewReturnStatement(value Expression, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *ReturnStatement {
	s := &ReturnStatement{Value: value}
	s.Modifiers = StmtModifierNone
	s.SourceSpan = sourceSpan
	s.LeadingComments = leadingComments
	return s
}
func (s *ReturnStatement) IsEquivalent(stmt Statement) bool {
	o, ok := stmt.(*ReturnStatement)
	return ok && s.Value.IsEquivalent(o.Value)
}
func (s *ReturnStatement) VisitStatement(v StatementVisitor, context any) any {
	return v.VisitReturnStmt(s, context)
}

type IfStmt struct {
	BaseStatement
	Condition Expression
	TrueCase  []Statement
	FalseCase []Statement
}

func NewIfStmt(condition Expression, trueCase []Statement, falseCase []Statement, sourceSpan ParseSourceSpan, leadingComments []LeadingComment) *IfStmt {
	if falseCase == nil {
		falseCase = []Statement{}
	}
	s := &IfStmt{Condition: condition, TrueCase: trueCase, FalseCase: falseCase}
	s.Modifiers = StmtModifierNone
	s.SourceSpan = sourceSpan
	s.LeadingComments = leadingComments
	return s
}
func (s *IfStmt) IsEquivalent(stmt Statement) bool {
	o, ok := stmt.(*IfStmt)
	return ok && s.Condition.IsEquivalent(o.Condition) && AreAllEquivalentStatements(s.TrueCase, o.TrueCase) && AreAllEquivalentStatements(s.FalseCase, o.FalseCase)
}
func (s *IfStmt) VisitStatement(v StatementVisitor, context any) any {
	return v.VisitIfStmt(s, context)
}

type StatementVisitor interface {
	VisitDeclareVarStmt(stmt *DeclareVarStmt, context any) any
	VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context any) any
	VisitExpressionStmt(stmt *ExpressionStatement, context any) any
	VisitReturnStmt(stmt *ReturnStatement, context any) any
	VisitIfStmt(stmt *IfStmt, context any) any
}

// Utilities (variable, importExpr, literalArr, etc.) omitted since not all are strictly needed, but let's implement a few basic ones if required.
// We are focused on structural parity of the AST nodes and visitors.

type Visitor interface {
	StatementVisitor
	ExpressionVisitor
	TypeVisitor
}


type ParseSourceLocation interface {
	GetFile() ParseSourceFile
	GetLine() int
	GetCol() int
}

type ParseSourceFile interface {
	GetUrl() string
	GetContent() string
}
