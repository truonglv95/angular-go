package ir

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

// Logical operation representing a binding to a native DOM property.
type DomPropertyOp struct {
	OpBase

	Name            string
	Expression      any // output.Expression | *Interpolation
	BindingKind     BindingKind
	I18nContext     *XrefId
	SecurityContext any // compiler.SecurityContext | []compiler.SecurityContext
	Sanitizer       output.Expression
	SourceSpan      *parse_util.ParseSourceSpan
}

func (o *DomPropertyOp) Kind() OpKind {
	return OpKindDomProperty
}

func (o *DomPropertyOp) ConsumesVars() bool {
	return true
}

func CreateDomPropertyOp(
	name string,
	expression any,
	bindingKind BindingKind,
	i18nContext *XrefId,
	securityContext any,
	sourceSpan *parse_util.ParseSourceSpan,
) *DomPropertyOp {
	return &DomPropertyOp{
		Name:            name,
		Expression:      expression,
		BindingKind:     bindingKind,
		I18nContext:     i18nContext,
		SecurityContext: securityContext,
		Sanitizer:       nil,
		SourceSpan:      sourceSpan,
	}
}
