package ir

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

// --------
// Traits from traits.ts
// --------

// ConsumesSlotTrait is implemented by create ops that consume a data slot.
// This is used by the slot allocation phase.
type ConsumesSlotTrait interface {
	// Xref returns the XrefId of the op.
	Xref() XrefId
	// Handle returns the SlotHandle that will be assigned a slot index.
	Handle() *SlotHandle
	// AddNumSlotsUsed adds to the number of slots used by this op.
	AddNumSlotsUsed(n int)
}

// ConsumesSlotOpTrait is implemented by ops that consume a slot and provide
// a GetXref() method, used in slot dependency assignment.
type ConsumesSlotOpTrait interface {
	GetXref() XrefId
}

// DependsOnSlotContextTrait is implemented by ops or expressions that depend on
// the runtime's implicit slot counter context.
type DependsOnSlotContextTrait interface {
	// Target returns the XrefId that the slot counter should point to.
	Target() XrefId
	// SourceSpan returns the source location associated with this expression.
	SourceSpan() *parse_util.ParseSourceSpan
}

// DependsOnSlotContextOpTrait is the op-level version of DependsOnSlotContextTrait,
// used in assign_i18n_slot_dependencies.go.
type DependsOnSlotContextOpTrait interface {
	GetTarget() XrefId
}

// DependsOnSlotContextExprTrait is the expression-level version of DependsOnSlotContextTrait,
// used in assign_i18n_slot_dependencies.go.
type DependsOnSlotContextExprTrait interface {
	GetTarget() XrefId
}

// ConsumesVarsTrait is implemented by ops or expressions that use vars.
type ConsumesVarsTrait interface {
	ConsumesVars() bool
}

// UsesVarOffsetTrait is implemented by ops/expressions with a var offset.
type UsesVarOffsetTrait interface {
	GetVarOffset() int
	SetVarOffset(offset int)
}

// --------
// Op-level Trait Interfaces
// --------

// XrefTrait is implemented by ops that have a cross-reference XrefId.
type XrefTrait interface {
	Xref() XrefId
}

// LocalRef represents a single local reference on an element.

// LocalRefsTrait is implemented by ops that have local references (e.g., ElementStartOp).
type LocalRefsTrait interface {
	LocalRefs() []LocalRef
	SetLocalRefsExpr(expr output.Expression)
}

// NonBindableTrait is implemented by ops that may be marked ngNonBindable.
type NonBindableTrait interface {
	NonBindable() bool
}

// BindingOpTrait is implemented by ops that represent data bindings.
type BindingOpTrait interface {
	GetBindingKind() BindingKind
	GetName() string
	SetName(name string)
	GetSourceSpan() *parse_util.ParseSourceSpan
}

// CreateOpTrait is the interface for create ops that wrap a child view
// (Template, ConditionalCreate, ConditionalBranchCreate).
type CreateOpTrait interface {
	Xref() XrefId
	Handle() *SlotHandle
	FunctionNameSuffix() string
	I18nPlaceholder() i18n.Node
}

// ListenerTrait is implemented by listener ops (Listener, TwoWayListener, Animation, AnimationListener).
type ListenerTrait interface {
	HandlerOps() *OpList
}

// ElementOpTrait is implemented by element ops (ElementStart, Element).
type ElementOpTrait interface {
	GetHandle() *SlotHandle
	GetStartSourceSpan() *parse_util.ParseSourceSpan
}

// ExpressionTrait is implemented by update ops that carry a single expression.
type ExpressionTrait interface {
	Expression() Expression
}

// TemplateKindTrait is implemented by ops with a template kind discriminant.
type TemplateKindTrait interface {
	TemplateKind() any
}

// OpWithKindTrait is implemented by ops that expose their kind.
type OpWithKindTrait interface {
	GetKind() OpKind
}

// --------
// Helper Functions
// --------

// IsElementOrContainerOp returns true if the op is an element or container op.
func IsElementOrContainerOp(op Op) bool {
	switch op.Kind() {
	case OpKindElement, OpKindElementStart, OpKindElementEnd,
		OpKindContainer, OpKindContainerStart, OpKindContainerEnd,
		OpKindTemplate, OpKindConditionalCreate, OpKindConditionalBranchCreate:
		return true
	}
	return false
}
