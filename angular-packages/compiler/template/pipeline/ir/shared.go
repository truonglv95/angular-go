package ir

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// StatementOp is a genuine implementation of an update statement operation.
type StatementOp struct {
	OpBase
	Statement output.Statement
}

func (o *StatementOp) Kind() OpKind {
	return OpKindStatement
}

func (o *StatementOp) GetKind() OpKind {
	return o.Kind()
}

// CreateStatementOp is a genuine implementation of a create statement operation.
type CreateStatementOp struct {
	OpBase
	Statement output.Statement
}

func (o *CreateStatementOp) Kind() OpKind {
	return OpKindStatement
}

func (o *CreateStatementOp) GetKind() OpKind {
	return o.Kind()
}

func NewStatementOp(xref XrefId, statement any, name *string) any {
	stmt, _ := statement.(output.Statement)
	return &StatementOp{
		Statement: stmt,
	}
}

func NewCreateStatementOp(xref XrefId, statement any, name *string) any {
	stmt, _ := statement.(output.Statement)
	return &CreateStatementOp{
		Statement: stmt,
	}
}

func NewVariableOp(xref XrefId, variable SemanticVariable, initializer any, flags VariableFlags) any {
	var initExpr output.Expression
	if initializer != nil {
		initExpr, _ = initializer.(output.Expression)
	}
	return &VariableOp{
		Xref:        xref,
		Variable:    &variable,
		Initializer: initExpr,
		Flags:       flags,
	}
}

func NewCreateVariableOp(xref XrefId, variable SemanticVariable, initializer any, flags VariableFlags) any {
	var initExpr output.Expression
	if initializer != nil {
		initExpr, _ = initializer.(output.Expression)
	}
	return &CreateVariableOp{
		Xref:        xref,
		Variable:    &variable,
		Initializer: initExpr,
		Flags:       flags,
	}
}
