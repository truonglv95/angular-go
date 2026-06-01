package ir

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

type irExpressionBase struct {
	output.BaseExpression
}

func (e *irExpressionBase) VisitExpression(v output.ExpressionVisitor, context any) any {
	return nil
}

func (e *irExpressionBase) IsEquivalent(other output.Expression) bool {
	return false
}

func (e *irExpressionBase) IsConstant() bool {
	return false
}

func (e *irExpressionBase) Clone() output.Expression {
	return nil
}

// Expression is the union type of all Angular-specific IR expression types.
// Each concrete expression type implements this interface, as well as output.Expression.
//
// This corresponds to the TypeScript Expression union type in expression.ts.
type Expression interface {
	output.Expression
	// ExprKind returns the ExpressionKind discriminant.
	ExprKind() ExpressionKind
	// TransformInternalExpressions applies transform to any sub-expressions.
	TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag)
}

// ExpressionTransform is a function that transforms an output.Expression into another.
// This corresponds to the TypeScript ExpressionTransform type.
type ExpressionTransform func(expr output.Expression, flags VisitorContextFlag) output.Expression

// VisitorContextFlag is a set of flags that control expression visitor behavior.
type VisitorContextFlag int

const (
	VisitorContextFlagNone               VisitorContextFlag = 0b0000
	VisitorContextFlagInChildOperation   VisitorContextFlag = 0b0001
	VisitorContextFlagInArrowFunction    VisitorContextFlag = 0b0010
	VisitorContextFlagInSafeNavMigration VisitorContextFlag = 0b0100
)

// --------
// LexicalReadExpr
// --------

// LexicalReadExpr represents a lexical read of a variable name.
type LexicalReadExpr struct {
	irExpressionBase
	Name string
}

func (e *LexicalReadExpr) ExprKind() ExpressionKind { return ExpressionKindLexicalRead }
func (e *LexicalReadExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *LexicalReadExpr) Clone() output.Expression {
	res := &LexicalReadExpr{
		Name: e.Name,
	}
	res.Self = res
	return res
}

// NewLexicalReadExpr creates a LexicalReadExpr.
func NewLexicalReadExpr(name string) *LexicalReadExpr {
	e := &LexicalReadExpr{
		Name: name,
	}
	e.Self = e
	return e
}

// --------
// ReferenceExpr
// --------

// ReferenceExpr retrieves the value of a local reference.
type ReferenceExpr struct {
	irExpressionBase
	Target     XrefId
	TargetSlot *SlotHandle
	Offset     int
}

func (e *ReferenceExpr) ExprKind() ExpressionKind                                                 { return ExpressionKindReference }
func (e *ReferenceExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {}
func (e *ReferenceExpr) Clone() output.Expression {
	res := &ReferenceExpr{
		Target:     e.Target,
		TargetSlot: e.TargetSlot,
		Offset:     e.Offset,
	}
	res.Self = res
	return res
}

// NewReferenceExpr creates a ReferenceExpr.
func NewReferenceExpr(target XrefId, targetSlot *SlotHandle, offset int) *ReferenceExpr {
	e := &ReferenceExpr{
		Target:     target,
		TargetSlot: targetSlot,
		Offset:     offset,
	}
	e.Self = e
	return e
}

// --------
// StoreLetExpr
// --------

// StoreLetExpr stores a value into a @let declaration variable.
type StoreLetExpr struct {
	irExpressionBase
	Target     XrefId
	Value      output.Expression
	SourceSpan *parse_util.ParseSourceSpan
}

func (e *StoreLetExpr) ExprKind() ExpressionKind { return ExpressionKindStoreLet }
func (e *StoreLetExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Value = TransformExpressionsInExpression(e.Value, transform, flags)
}
func (e *StoreLetExpr) Clone() output.Expression {
	res := &StoreLetExpr{
		Target:     e.Target,
		Value:      e.Value.Clone(),
		SourceSpan: e.SourceSpan,
	}
	res.Self = res
	return res
}
func (e *StoreLetExpr) GetTarget() XrefId { return e.Target }

// NewStoreLetExpr creates a StoreLetExpr.
func NewStoreLetExpr(target XrefId, value output.Expression, sourceSpan *parse_util.ParseSourceSpan) *StoreLetExpr {
	e := &StoreLetExpr{
		Target:     target,
		Value:      value,
		SourceSpan: sourceSpan,
	}
	e.Self = e
	return e
}

// --------
// ContextLetReferenceExpr
// --------

// ContextLetReferenceExpr reads a @let declaration from the current context.
type ContextLetReferenceExpr struct {
	irExpressionBase
	Target     XrefId
	TargetSlot *SlotHandle
}

func (e *ContextLetReferenceExpr) ExprKind() ExpressionKind {
	return ExpressionKindContextLetReference
}
func (e *ContextLetReferenceExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *ContextLetReferenceExpr) Clone() output.Expression {
	res := &ContextLetReferenceExpr{
		Target:     e.Target,
		TargetSlot: e.TargetSlot,
	}
	res.Self = res
	return res
}

// NewContextLetReferenceExpr creates a ContextLetReferenceExpr.
func NewContextLetReferenceExpr(target XrefId, targetSlot *SlotHandle) *ContextLetReferenceExpr {
	e := &ContextLetReferenceExpr{
		Target:     target,
		TargetSlot: targetSlot,
	}
	e.Self = e
	return e
}

// --------
// ContextExpr
// --------

// ContextExpr is a reference to the current view context (the `ctx` variable).
type ContextExpr struct {
	irExpressionBase
	View XrefId
}

func (e *ContextExpr) ExprKind() ExpressionKind                                                 { return ExpressionKindContext }
func (e *ContextExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {}
func (e *ContextExpr) Clone() output.Expression {
	res := &ContextExpr{
		View: e.View,
	}
	res.Self = res
	return res
}

// NewContextExpr creates a ContextExpr.
func NewContextExpr(view XrefId) *ContextExpr {
	e := &ContextExpr{
		View: view,
	}
	e.Self = e
	return e
}

// --------
// TrackContextExpr
// --------

// TrackContextExpr is a reference to the current view context inside a track function.
type TrackContextExpr struct {
	irExpressionBase
	View XrefId
}

func (e *TrackContextExpr) ExprKind() ExpressionKind { return ExpressionKindTrackContext }
func (e *TrackContextExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *TrackContextExpr) Clone() output.Expression {
	res := &TrackContextExpr{
		View: e.View,
	}
	res.Self = res
	return res
}

// NewTrackContextExpr creates a TrackContextExpr.
func NewTrackContextExpr(view XrefId) *TrackContextExpr {
	e := &TrackContextExpr{
		View: view,
	}
	e.Self = e
	return e
}

// --------
// NextContextExpr
// --------

// NextContextExpr navigates to the next view context in the hierarchy.
type NextContextExpr struct {
	irExpressionBase
	Steps int
}

func (e *NextContextExpr) ExprKind() ExpressionKind { return ExpressionKindNextContext }
func (e *NextContextExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *NextContextExpr) Clone() output.Expression {
	res := &NextContextExpr{
		Steps: e.Steps,
	}
	res.Self = res
	return res
}

// NewNextContextExpr creates a NextContextExpr with steps = 1.
func NewNextContextExpr() *NextContextExpr {
	e := &NextContextExpr{
		Steps: 1,
	}
	e.Self = e
	return e
}

// --------
// GetCurrentViewExpr
// --------

// GetCurrentViewExpr snapshots the current view context.
type GetCurrentViewExpr struct {
	irExpressionBase
}

func (e *GetCurrentViewExpr) ExprKind() ExpressionKind { return ExpressionKindGetCurrentView }
func (e *GetCurrentViewExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *GetCurrentViewExpr) Clone() output.Expression {
	res := &GetCurrentViewExpr{}
	res.Self = res
	return res
}

// NewGetCurrentViewExpr creates a GetCurrentViewExpr.
func NewGetCurrentViewExpr() *GetCurrentViewExpr {
	e := &GetCurrentViewExpr{}
	e.Self = e
	return e
}

// --------
// RestoreViewExpr
// --------

// RestoreViewExpr restores a snapshotted view context.
// View may be either an XrefId (as int) or an output.Expression.
type RestoreViewExpr struct {
	irExpressionBase
	// ViewXref holds the XrefId if the view is resolved to a number.
	ViewXref *XrefId
	// ViewExpr holds the expression if the view hasn't been resolved yet.
	ViewExpr output.Expression
}

func (e *RestoreViewExpr) ExprKind() ExpressionKind { return ExpressionKindRestoreView }
func (e *RestoreViewExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	if e.ViewExpr != nil {
		e.ViewExpr = TransformExpressionsInExpression(e.ViewExpr, transform, flags)
	}
}
func (e *RestoreViewExpr) Clone() output.Expression {
	c := &RestoreViewExpr{}
	c.Self = c
	if e.ViewXref != nil {
		xref := *e.ViewXref
		c.ViewXref = &xref
	} else if e.ViewExpr != nil {
		c.ViewExpr = e.ViewExpr.Clone()
	}
	return c
}

// NewRestoreViewExpr creates a RestoreViewExpr with an XrefId render3.
func NewRestoreViewExpr(view XrefId) *RestoreViewExpr {
	e := &RestoreViewExpr{
		ViewXref: &view,
	}
	e.Self = e
	return e
}

// NewRestoreViewExprFromExpr creates a RestoreViewExpr with an expression render3.
func NewRestoreViewExprFromExpr(view output.Expression) *RestoreViewExpr {
	e := &RestoreViewExpr{
		ViewExpr: view,
	}
	e.Self = e
	return e
}

// --------
// ResetViewExpr
// --------

// ResetViewExpr resets the current view context after a RestoreView.
type ResetViewExpr struct {
	irExpressionBase
	Expr output.Expression
}

func (e *ResetViewExpr) ExprKind() ExpressionKind { return ExpressionKindResetView }
func (e *ResetViewExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
}
func (e *ResetViewExpr) Clone() output.Expression {
	res := &ResetViewExpr{
		Expr: e.Expr.Clone(),
	}
	res.Self = res
	return res
}

// NewResetViewExpr creates a ResetViewExpr.
func NewResetViewExpr(expr output.Expression) *ResetViewExpr {
	e := &ResetViewExpr{
		Expr: expr,
	}
	e.Self = e
	return e
}

// --------
// TwoWayBindingSetExpr
// --------

// TwoWayBindingSetExpr sets a two-way binding value.
type TwoWayBindingSetExpr struct {
	irExpressionBase
	TargetExpr output.Expression
	Value      output.Expression
}

func (e *TwoWayBindingSetExpr) ExprKind() ExpressionKind { return ExpressionKindTwoWayBindingSet }
func (e *TwoWayBindingSetExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.TargetExpr = TransformExpressionsInExpression(e.TargetExpr, transform, flags)
	e.Value = TransformExpressionsInExpression(e.Value, transform, flags)
}
func (e *TwoWayBindingSetExpr) Clone() output.Expression {
	res := &TwoWayBindingSetExpr{
		TargetExpr: e.TargetExpr.Clone(),
		Value:      e.Value.Clone(),
	}
	res.Self = res
	return res
}

// NewTwoWayBindingSetExpr creates a TwoWayBindingSetExpr.
func NewTwoWayBindingSetExpr(target output.Expression, value output.Expression) *TwoWayBindingSetExpr {
	e := &TwoWayBindingSetExpr{
		TargetExpr: target,
		Value:      value,
	}
	e.Self = e
	return e
}

// --------
// ReadVariableExpr
// --------

// ReadVariableExpr reads a variable declared as a VariableOp by XrefId.
type ReadVariableExpr struct {
	irExpressionBase
	Xref XrefId
	Name *string
}

func (e *ReadVariableExpr) ExprKind() ExpressionKind { return ExpressionKindReadVariable }
func (e *ReadVariableExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *ReadVariableExpr) Clone() output.Expression {
	c := &ReadVariableExpr{Xref: e.Xref}
	c.Name = e.Name
	c.Self = c
	return c
}

// NewReadVariableExpr creates a ReadVariableExpr.
func NewReadVariableExpr(xref XrefId) *ReadVariableExpr {
	e := &ReadVariableExpr{
		Xref: xref,
	}
	e.Self = e
	return e
}

// --------
// PureFunctionExpr
// --------

// PureFunctionExpr memoizes a pure computation.
type PureFunctionExpr struct {
	irExpressionBase
	VarOffset *int
	Body      output.Expression
	Args      []output.Expression
	Fn        output.Expression
}

func (e *PureFunctionExpr) ExprKind() ExpressionKind { return ExpressionKindPureFunctionExpr }
func (e *PureFunctionExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	if e.Body != nil {
		e.Body = TransformExpressionsInExpression(e.Body, transform, flags|VisitorContextFlagInChildOperation)
	} else if e.Fn != nil {
		e.Fn = TransformExpressionsInExpression(e.Fn, transform, flags)
	}
	for i := range e.Args {
		e.Args[i] = TransformExpressionsInExpression(e.Args[i], transform, flags)
	}
}
func (e *PureFunctionExpr) Clone() output.Expression {
	var args []output.Expression
	if e.Args != nil {
		args = make([]output.Expression, len(e.Args))
		for i, a := range e.Args {
			args[i] = a.Clone()
		}
	}
	c := &PureFunctionExpr{
		Args:      args,
		VarOffset: e.VarOffset,
	}
	c.Self = c
	if e.Body != nil {
		c.Body = e.Body.Clone()
	}
	if e.Fn != nil {
		c.Fn = e.Fn.Clone()
	}
	return c
}
func (e *PureFunctionExpr) GetVarOffset() int {
	if e.VarOffset == nil {
		return -1
	}
	return *e.VarOffset
}
func (e *PureFunctionExpr) SetVarOffset(n int) { e.VarOffset = &n }
func (e *PureFunctionExpr) ConsumesVars() bool { return true }

// --------
// PureFunctionParameterExpr
// --------

// PureFunctionParameterExpr is a placeholder for a pure function parameter.
type PureFunctionParameterExpr struct {
	irExpressionBase
	Index int
}

func (e *PureFunctionParameterExpr) ExprKind() ExpressionKind {
	return ExpressionKindPureFunctionParameterExpr
}
func (e *PureFunctionParameterExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *PureFunctionParameterExpr) Clone() output.Expression {
	res := &PureFunctionParameterExpr{
		Index: e.Index,
	}
	res.Self = res
	return res
}

// --------
// PipeBindingExpr
// --------

// PipeBindingExpr calls a pipe with a fixed number of arguments.
type PipeBindingExpr struct {
	irExpressionBase
	TargetXref XrefId
	TargetSlot *SlotHandle
	PipeName   string
	Args       []output.Expression
	VarOffset  *int
}

func (e *PipeBindingExpr) Target() XrefId { return e.TargetXref } // Named Target() to match trait
func (e *PipeBindingExpr) SourceSpan() *parse_util.ParseSourceSpan { return nil }

func (e *PipeBindingExpr) ExprKind() ExpressionKind { return ExpressionKindPipeBinding }
func (e *PipeBindingExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	for i := range e.Args {
		e.Args[i] = TransformExpressionsInExpression(e.Args[i], transform, flags)
	}
}
func (e *PipeBindingExpr) Clone() output.Expression {
	var args []output.Expression
	if e.Args != nil {
		args = make([]output.Expression, len(e.Args))
		for i, a := range e.Args {
			args[i] = a.Clone()
		}
	}
	res := &PipeBindingExpr{
		TargetXref: e.TargetXref,
		TargetSlot: e.TargetSlot,
		PipeName:   e.PipeName,
		Args:       args,
		VarOffset:  e.VarOffset,
	}
	res.Self = res
	return res
}
func (e *PipeBindingExpr) ConsumesVars() bool { return true }
func (e *PipeBindingExpr) GetVarOffset() int {
	if e.VarOffset == nil {
		return -1
	}
	return *e.VarOffset
}
func (e *PipeBindingExpr) SetVarOffset(n int) { e.VarOffset = &n }

// NewPipeBindingExpr creates a PipeBindingExpr.
func NewPipeBindingExpr(target XrefId, targetSlot *SlotHandle, name string, args []output.Expression) *PipeBindingExpr {
	e := &PipeBindingExpr{
		TargetXref: target,
		TargetSlot: targetSlot,
		PipeName:   name,
		Args:       args,
	}
	e.Self = e
	return e
}

// --------
// PipeBindingVariadicExpr
// --------

// PipeBindingVariadicExpr calls a pipe with a variadic number of arguments.
type PipeBindingVariadicExpr struct {
	irExpressionBase
	TargetXref XrefId
	TargetSlot *SlotHandle
	PipeName   string
	Args       output.Expression
	NumArgs    int
	VarOffset  *int
}

func (e *PipeBindingVariadicExpr) Target() XrefId { return e.TargetXref }
func (e *PipeBindingVariadicExpr) SourceSpan() *parse_util.ParseSourceSpan { return nil }

func (e *PipeBindingVariadicExpr) ExprKind() ExpressionKind { return ExpressionKindPipeBindingVariadic }
func (e *PipeBindingVariadicExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Args = TransformExpressionsInExpression(e.Args, transform, flags)
}
func (e *PipeBindingVariadicExpr) Clone() output.Expression {
	res := &PipeBindingVariadicExpr{
		TargetXref: e.TargetXref,
		TargetSlot: e.TargetSlot,
		PipeName:   e.PipeName,
		Args:       e.Args.Clone(),
		NumArgs:    e.NumArgs,
		VarOffset:  e.VarOffset,
	}
	res.Self = res
	return res
}
func (e *PipeBindingVariadicExpr) ConsumesVars() bool { return true }
func (e *PipeBindingVariadicExpr) GetVarOffset() int {
	if e.VarOffset == nil {
		return -1
	}
	return *e.VarOffset
}
func (e *PipeBindingVariadicExpr) SetVarOffset(n int) { e.VarOffset = &n }

// NewPipeBindingVariadicExpr creates a PipeBindingVariadicExpr.
func NewPipeBindingVariadicExpr(target XrefId, targetSlot *SlotHandle, name string, args output.Expression, numArgs int) *PipeBindingVariadicExpr {
	e := &PipeBindingVariadicExpr{
		TargetXref: target,
		TargetSlot: targetSlot,
		PipeName:   name,
		Args:       args,
		NumArgs:    numArgs,
	}
	e.Self = e
	return e
}

// --------
// SafePropertyReadExpr
// --------

// SafePropertyReadExpr reads a property safely (null-checking the receiver).
type SafePropertyReadExpr struct {
	irExpressionBase
	Receiver output.Expression
	PropName string
}

func (e *SafePropertyReadExpr) ExprKind() ExpressionKind { return ExpressionKindSafePropertyRead }
func (e *SafePropertyReadExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Receiver = TransformExpressionsInExpression(e.Receiver, transform, flags)
}
func (e *SafePropertyReadExpr) Clone() output.Expression {
	res := &SafePropertyReadExpr{
		Receiver: e.Receiver.Clone(),
		PropName: e.PropName,
	}
	res.Self = res
	return res
}

// NewSafePropertyReadExpr creates a SafePropertyReadExpr.
func NewSafePropertyReadExpr(receiver output.Expression, name string) *SafePropertyReadExpr {
	e := &SafePropertyReadExpr{
		Receiver: receiver,
		PropName: name,
	}
	e.Self = e
	return e
}

// --------
// SafeKeyedReadExpr
// --------

// SafeKeyedReadExpr reads a key safely (null-checking the receiver).
type SafeKeyedReadExpr struct {
	irExpressionBase
	Receiver   output.Expression
	Index      output.Expression
	SourceSpan *parse_util.ParseSourceSpan
}

func (e *SafeKeyedReadExpr) ExprKind() ExpressionKind { return ExpressionKindSafeKeyedRead }
func (e *SafeKeyedReadExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Receiver = TransformExpressionsInExpression(e.Receiver, transform, flags)
	e.Index = TransformExpressionsInExpression(e.Index, transform, flags)
}
func (e *SafeKeyedReadExpr) Clone() output.Expression {
	res := &SafeKeyedReadExpr{
		Receiver:   e.Receiver.Clone(),
		Index:      e.Index.Clone(),
		SourceSpan: e.SourceSpan,
	}
	res.Self = res
	return res
}

// NewSafeKeyedReadExpr creates a SafeKeyedReadExpr.
func NewSafeKeyedReadExpr(receiver output.Expression, index output.Expression, sourceSpan *parse_util.ParseSourceSpan) *SafeKeyedReadExpr {
	e := &SafeKeyedReadExpr{
		Receiver:   receiver,
		Index:      index,
		SourceSpan: sourceSpan,
	}
	e.Self = e
	return e
}

// --------
// SafeNavigationMigrationExpr
// --------

// SafeNavigationMigrationExpr wraps an expression for legacy null-safe navigation.
type SafeNavigationMigrationExpr struct {
	irExpressionBase
	Expr output.Expression
}

func (e *SafeNavigationMigrationExpr) ExprKind() ExpressionKind {
	return ExpressionKindSafeNavigationMigration
}
func (e *SafeNavigationMigrationExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags|VisitorContextFlagInSafeNavMigration)
}
func (e *SafeNavigationMigrationExpr) Clone() output.Expression {
	res := &SafeNavigationMigrationExpr{
		Expr: e.Expr.Clone(),
	}
	res.Self = res
	return res
}

// --------
// SafeTernaryExpr
// --------

// SafeTernaryExpr is a ternary used in safe navigation expansion.
type SafeTernaryExpr struct {
	irExpressionBase
	Guard output.Expression
	Expr  output.Expression
}

func (e *SafeTernaryExpr) ExprKind() ExpressionKind { return ExpressionKindSafeTernaryExpr }
func (e *SafeTernaryExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Guard = TransformExpressionsInExpression(e.Guard, transform, flags)
	e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
}
func (e *SafeTernaryExpr) Clone() output.Expression {
	res := &SafeTernaryExpr{
		Guard: e.Guard.Clone(),
		Expr:  e.Expr.Clone(),
	}
	res.Self = res
	return res
}

// --------
// EmptyExpr
// --------

// EmptyExpr represents an empty/absent expression.
type EmptyExpr struct {
	irExpressionBase
	SourceSpan *parse_util.ParseSourceSpan
}

func (e *EmptyExpr) ExprKind() ExpressionKind                                                 { return ExpressionKindEmptyExpr }
func (e *EmptyExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {}
func (e *EmptyExpr) Clone() output.Expression {
	res := &EmptyExpr{
		SourceSpan: e.SourceSpan,
	}
	res.Self = res
	return res
}

// NewEmptyExpr creates an EmptyExpr.
func NewEmptyExpr(sourceSpan *parse_util.ParseSourceSpan) *EmptyExpr {
	e := &EmptyExpr{
		SourceSpan: sourceSpan,
	}
	e.Self = e
	return e
}

// --------
// AssignTemporaryExpr
// --------

// AssignTemporaryExpr assigns the result of expr to a named temporary variable.
type AssignTemporaryExpr struct {
	irExpressionBase
	Expr output.Expression
	Xref XrefId
	Name *string
}

func (e *AssignTemporaryExpr) ExprKind() ExpressionKind { return ExpressionKindAssignTemporaryExpr }
func (e *AssignTemporaryExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
}
func (e *AssignTemporaryExpr) Clone() output.Expression {
	res := &AssignTemporaryExpr{
		Xref: e.Xref,
	}
	res.Name = e.Name
	if e.Expr != nil {
		res.Expr = e.Expr.Clone()
	}
	res.Self = res
	return res
}

// NewAssignTemporaryExpr creates an AssignTemporaryExpr.
func NewAssignTemporaryExpr(xref XrefId) *AssignTemporaryExpr {
	e := &AssignTemporaryExpr{
		Xref: xref,
	}
	e.Self = e
	return e
}

// --------
// ReadTemporaryExpr
// --------

// ReadTemporaryExpr reads the value of a previously assigned temporary variable.
type ReadTemporaryExpr struct {
	irExpressionBase
	Xref XrefId
	Name *string
}

func (e *ReadTemporaryExpr) ExprKind() ExpressionKind { return ExpressionKindReadTemporaryExpr }
func (e *ReadTemporaryExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {
}
func (e *ReadTemporaryExpr) Clone() output.Expression {
	res := &ReadTemporaryExpr{
		Xref: e.Xref,
	}
	res.Name = e.Name
	res.Self = res
	return res
}

// NewReadTemporaryExpr creates a ReadTemporaryExpr.
func NewReadTemporaryExpr(xref XrefId) *ReadTemporaryExpr {
	e := &ReadTemporaryExpr{
		Xref: xref,
	}
	e.Self = e
	return e
}

// --------
// SlotLiteralExpr
// --------

// SlotLiteralExpr is a literal reference to a slot handle.
type SlotLiteralExpr struct {
	irExpressionBase
	SlotHandle *SlotHandle
}

func (e *SlotLiteralExpr) ExprKind() ExpressionKind                                                 { return ExpressionKindSlotLiteralExpr }
func (e *SlotLiteralExpr) TransformInternalExpressions(_ ExpressionTransform, _ VisitorContextFlag) {}
func (e *SlotLiteralExpr) Clone() output.Expression {
	res := &SlotLiteralExpr{
		SlotHandle: e.SlotHandle,
	}
	res.Self = res
	return res
}

// --------
// ConditionalCaseExpr
// --------

// ConditionalCaseExpr is one branch of a conditional expression.
type ConditionalCaseExpr struct {
	irExpressionBase
	Expr       output.Expression // nil for else case
	Target     XrefId
	TargetSlot *SlotHandle
	Alias      any // *render3.Variable | nil
}

func (e *ConditionalCaseExpr) ExprKind() ExpressionKind { return ExpressionKindConditionalCase }
func (e *ConditionalCaseExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	if e.Expr != nil {
		e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
	}
}
func (e *ConditionalCaseExpr) Clone() output.Expression {
	res := &ConditionalCaseExpr{
		Target:     e.Target,
		TargetSlot: e.TargetSlot,
		Alias:      e.Alias,
	}
	res.Self = res
	if e.Expr != nil {
		res.Expr = e.Expr.Clone()
	}
	return res
}

// NewConditionalCaseExpr creates a ConditionalCaseExpr.
func NewConditionalCaseExpr(expr output.Expression, target XrefId, targetSlot *SlotHandle, alias any) *ConditionalCaseExpr {
	e := &ConditionalCaseExpr{
		Expr:       expr,
		Target:     target,
		TargetSlot: targetSlot,
		Alias:      alias,
	}
	e.Self = e
	return e
}

// --------
// ConstCollectedExpr
// --------

// ConstCollectedExpr wraps an expression that has been collected into the consts array.
type ConstCollectedExpr struct {
	irExpressionBase
	Expr output.Expression
}

func (e *ConstCollectedExpr) ExprKind() ExpressionKind { return ExpressionKindConstCollected }
func (e *ConstCollectedExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	e.Expr = transform(e.Expr, flags)
}
func (e *ConstCollectedExpr) Clone() output.Expression {
	res := &ConstCollectedExpr{
		Expr: e.Expr,
	}
	res.Self = res
	return res
}

// --------
// ArrowFunctionExpr
// --------

// ArrowFunctionExpr represents an arrow function in the IR update phase.
type ArrowFunctionExpr struct {
	irExpressionBase
	Parameters      []output.FnParam
	Body            output.Expression
	Ops             *OpList
	VarOffset       *int
	CurrentViewName string
}

func (e *ArrowFunctionExpr) ExprKind() ExpressionKind { return ExpressionKindArrowFunction }
func (e *ArrowFunctionExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	innerFlags := flags | VisitorContextFlagInChildOperation | VisitorContextFlagInArrowFunction
	for _, op := range e.Ops.Ops {
		TransformExpressionsInOp(op, transform, innerFlags)
	}
}
func (e *ArrowFunctionExpr) Clone() output.Expression {
	var params []output.FnParam
	if e.Parameters != nil {
		params = make([]output.FnParam, len(e.Parameters))
		copy(params, e.Parameters)
	}
	var clonedOps *OpList
	if e.Ops != nil {
		clonedOps = &OpList{
			DebugListId: NewOpListId(),
			HeadNode:    e.Ops.HeadNode,
			TailNode:    e.Ops.TailNode,
		}
		if e.Ops.Ops != nil {
			clonedOps.Ops = make([]Op, len(e.Ops.Ops))
			copy(clonedOps.Ops, e.Ops.Ops)
		}
	}
	var varOffset *int
	if e.VarOffset != nil {
		val := *e.VarOffset
		varOffset = &val
	}
	res := &ArrowFunctionExpr{
		Parameters:      params,
		Ops:             clonedOps,
		VarOffset:       varOffset,
		CurrentViewName: e.CurrentViewName,
	}
	if e.Body != nil {
		res.Body = e.Body.Clone()
	}
	res.Self = res
	return res
}
func (e *ArrowFunctionExpr) ConsumesVars() bool { return true }
func (e *ArrowFunctionExpr) GetVarOffset() int {
	if e.VarOffset == nil {
		return -1
	}
	return *e.VarOffset
}
func (e *ArrowFunctionExpr) SetVarOffset(n int) { e.VarOffset = &n }

// NewArrowFunctionExpr creates an ArrowFunctionExpr.
func NewArrowFunctionExpr(params []output.FnParam, body output.Expression, ops *OpList) *ArrowFunctionExpr {
	e := &ArrowFunctionExpr{
		Parameters: params,
		Body:       body,
		Ops:        ops,
	}
	e.Self = e
	return e
}

// --------
// IsIrExpression
// --------

// IsIrExpression checks whether a given output.Expression is an IR expression.
func IsIrExpression(expr output.Expression) bool {
	_, ok := expr.(Expression)
	return ok
}

// --------
// TransformExpressionsInExpression
// --------

// TransformExpressionsInExpression applies a transform to all sub-expressions of expr.
// This is a 1:1 port of the TypeScript function of the same name.
func TransformExpressionsInExpression(expr output.Expression, transform ExpressionTransform, flags VisitorContextFlag) output.Expression {
	if irExpr, ok := expr.(Expression); ok {
		irExpr.TransformInternalExpressions(transform, flags)
	} else if interp, ok := expr.(*Interpolation); ok {
		for i, e := range interp.Expressions {
			interp.Expressions[i] = TransformExpressionsInExpression(e, transform, flags)
		}
	} else {
		// Traverse standard output.Expression nodes.
		switch e := expr.(type) {
		case *output.InvokeFunctionExpr:
			e.Fn = TransformExpressionsInExpression(e.Fn, transform, flags)
			for i, arg := range e.Args {
				e.Args[i] = TransformExpressionsInExpression(arg, transform, flags)
			}
		case *output.ReadPropExpr:
			e.Receiver = TransformExpressionsInExpression(e.Receiver, transform, flags)
		case *output.ReadKeyExpr:
			e.Receiver = TransformExpressionsInExpression(e.Receiver, transform, flags)
			e.Index = TransformExpressionsInExpression(e.Index, transform, flags)
		case *output.BinaryOperatorExpr:
			e.Lhs = TransformExpressionsInExpression(e.Lhs, transform, flags)
			e.Rhs = TransformExpressionsInExpression(e.Rhs, transform, flags)
		case *output.UnaryOperatorExpr:
			e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
		case *output.ParenthesizedExpr:
			e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
		case *output.ConditionalExpr:
			e.Condition = TransformExpressionsInExpression(e.Condition, transform, flags)
			e.TrueCase = TransformExpressionsInExpression(e.TrueCase, transform, flags)
			if e.FalseCase != nil {
				e.FalseCase = TransformExpressionsInExpression(e.FalseCase, transform, flags)
			}
		case *output.NotExpr:
			e.Condition = TransformExpressionsInExpression(e.Condition, transform, flags)
		case *output.SpreadElementExpr:
			e.Expression = TransformExpressionsInExpression(e.Expression, transform, flags)
		case *output.TypeofExpr:
			e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
		case *output.VoidExpr:
			e.Expr = TransformExpressionsInExpression(e.Expr, transform, flags)
		case *output.LiteralArrayExpr:
			for i, entry := range e.Entries {
				e.Entries[i] = TransformExpressionsInExpression(entry, transform, flags)
			}
		case *output.LiteralMapExpr:
			for _, entry := range e.Entries {
				switch assign := entry.(type) {
				case *output.LiteralMapPropertyAssignment:
					assign.Value = TransformExpressionsInExpression(assign.Value, transform, flags)
				case *output.LiteralMapSpreadAssignment:
					assign.Expression = TransformExpressionsInExpression(assign.Expression, transform, flags)
				}
			}
		case *output.CommaExpr:
			for i, part := range e.Parts {
				e.Parts[i] = TransformExpressionsInExpression(part, transform, flags)
			}
		case *output.InstantiateExpr:
			e.ClassExpr = TransformExpressionsInExpression(e.ClassExpr, transform, flags)
			for i, arg := range e.Args {
				e.Args[i] = TransformExpressionsInExpression(arg, transform, flags)
			}
		case *output.TaggedTemplateLiteralExpr:
			e.Tag = TransformExpressionsInExpression(e.Tag, transform, flags)
		case *output.FunctionExpr:
			for _, stmt := range e.Statements {
				TransformExpressionsInStatement(stmt, transform, flags)
			}
		case *output.ArrowFunctionExpr:
			if stmts, ok := e.Body.([]output.Statement); ok {
				for _, stmt := range stmts {
					TransformExpressionsInStatement(stmt, transform, flags)
				}
			} else if expr, ok := e.Body.(output.Expression); ok {
				e.Body = TransformExpressionsInExpression(expr, transform, flags)
			}
		}
	}
	return transform(expr, flags)
}
