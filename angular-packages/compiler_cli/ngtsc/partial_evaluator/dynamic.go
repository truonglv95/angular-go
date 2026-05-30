package partial_evaluator

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

type DynamicValueReason int

const (
	DYNAMIC_INPUT DynamicValueReason = iota
	DYNAMIC_STRING
	EXTERNAL_REFERENCE
	UNSUPPORTED_SYNTAX
	UNKNOWN_IDENTIFIER
	INVALID_EXPRESSION_TYPE
	COMPLEX_FUNCTION_CALL
	DYNAMIC_TYPE
	SYNTHETIC_INPUT
	UNKNOWN
)

type DynamicValue struct {
}

func (recv *DynamicValue) FromDynamicInput(node ast.Node, input DynamicValue) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromDynamicString(node ast.Node) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromExternalany(node ast.Node, ref *any) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromUnsupportedSyntax(node ast.Node) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromUnknownIdentifier(node ast.Identifier) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromInvalidExpressionType(node ast.Node, value any) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromComplexFunctionCall(node ast.Node, fn any) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromDynamicType(node ast.Node) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromSyntheticInput(node ast.Node, value SyntheticValue) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) FromUnknown(node ast.Node) DynamicValue {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromDynamicInput(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromDynamicString(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromExternalany(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromUnsupportedSyntax(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromUnknownIdentifier(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromInvalidExpressionType(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromComplexFunctionCall(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromDynamicType(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) IsFromUnknown(this DynamicValue) any {
	// TODO: stub
	panic("unimplemented")
}

func (recv *DynamicValue) Accept(visitor DynamicValueVisitor) any {
	// TODO: stub
	panic("unimplemented")
}

type DynamicValueVisitor interface {
	VisitDynamicInput(value DynamicValue) any
	VisitDynamicString(value DynamicValue) any
	VisitExternalany(value DynamicValue) any
	VisitUnsupportedSyntax(value DynamicValue) any
	VisitUnknownIdentifier(value DynamicValue) any
	VisitInvalidExpressionType(value DynamicValue) any
	VisitComplexFunctionCall(value DynamicValue) any
	VisitDynamicType(value DynamicValue) any
	VisitSyntheticInput(value DynamicValue) any
	VisitUnknown(value DynamicValue) any
}
