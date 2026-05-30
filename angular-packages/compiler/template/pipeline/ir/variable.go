package ir

import "github.com/microsoft/typescript-go/angular-packages/compiler/output"

// CTX_REF is a marker for a reference to the entire context object.
const CTX_REF = "CTX_REF_MARKER"

// SemanticVariable is the base struct representing all kinds of semantic variables.
// It acts as the union type for ContextVariable, IdentifierVariable, SavedViewVariable,
// and AliasVariable. The Kind field discriminates between the different cases.
//
// In TypeScript, these are distinct interface types; in Go they are represented as a
// single "fat struct" to preserve structural compatibility across the codebase.
type SemanticVariable struct {
	irExpressionBase
	Kind SemanticVariableKind
	// Name assigned to this variable in generated code, or nil if not yet assigned.
	Name *string
	// View is set for SemanticVariableKindContext and SemanticVariableKindSavedView.
	View XrefId
	// Identifier is set for SemanticVariableKindIdentifier and SemanticVariableKindAlias.
	Identifier string
	// Local indicates whether an IdentifierVariable was declared locally within the same render3.
	Local bool
	// Expression is set for SemanticVariableKindAlias; it is the expression that will be inlined.
	Expression output.Expression
}

func (v *SemanticVariable) ExprKind() ExpressionKind {
	return 0
}

func (v *SemanticVariable) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	if v.Expression != nil {
		v.Expression = TransformExpressionsInExpression(v.Expression, transform, flags)
	}
}

func (v *SemanticVariable) Clone() output.Expression {
	cloned := &SemanticVariable{
		Kind:       v.Kind,
		Name:       v.Name,
		View:       v.View,
		Identifier: v.Identifier,
		Local:      v.Local,
	}
	cloned.Self = cloned
	if v.Expression != nil {
		cloned.Expression = v.Expression.Clone()
	}
	return cloned
}

// IdentifierVariable is a type alias for SemanticVariable, representing
// variables of kind SemanticVariableKindIdentifier.
//
// Using a type alias preserves assignability with *SemanticVariable throughout
// the codebase, matching TypeScript structural typing semantics.
type IdentifierVariable = SemanticVariable

// AliasVariable is a type alias for SemanticVariable, representing
// variables of kind SemanticVariableKindAlias.
//
// AliasVariable has the additional fields Identifier and Expression populated.
type AliasVariable = SemanticVariable

type VariableOp struct {
	OpBase
	Xref        XrefId
	Variable    *SemanticVariable
	Initializer output.Expression
	Flags       VariableFlags
}

func (o *VariableOp) Kind() OpKind    { return OpKindVariable }
func (o *VariableOp) GetKind() OpKind { return o.Kind() }

type CreateVariableOp struct {
	OpBase
	Xref        XrefId
	Variable    *SemanticVariable
	Initializer output.Expression
	Flags       VariableFlags
}

func (o *CreateVariableOp) Kind() OpKind    { return OpKindVariable }
func (o *CreateVariableOp) GetKind() OpKind { return o.Kind() }
