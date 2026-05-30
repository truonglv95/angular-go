package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

const svgNamespace = "http://www.w3.org/2000/svg"
const mathMLNamespace = "http://www.w3.org/1998/Math/MathML"

// ResolveI18nAttrSanitizers wraps static i18n extracted attributes in their sanitizers/validators.
func ResolveI18nAttrSanitizers(job compilation.CompilationJob) {
	tagNamesByElement := map[ir.XrefId]string{}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			if op.Kind() != ir.OpKindElementStart && op.Kind() != ir.OpKindTemplate {
				continue
			}
			type taggedOp interface {
				GetTag() *string
				GetXref() ir.XrefId
				GetNamespace() ir.Namespace
			}
			if tagged, ok := op.(taggedOp); ok {
				tag := ""
				if t := tagged.GetTag(); t != nil {
					tag = *t
				}
				switch tagged.GetNamespace() {
				case ir.NamespaceSVG:
					tag = ":" + svgNamespace + ":" + tag
				case ir.NamespaceMath:
					tag = ":" + mathMLNamespace + ":" + tag
				}
				tagNamesByElement[tagged.GetXref()] = tag
			}
		}
	}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindExtractedAttribute {
				continue
			}
			extAttr, ok := op.(*ir.ExtractedAttributeOp)
			if !ok || extAttr.I18nContext == 0 || extAttr.Expression == nil {
				continue
			}
			tagName := tagNamesByElement[extAttr.Target]
			sc := getOnlySecurityContextValue(extAttr.SecurityContext)
			var sanitized output.Expression
			switch sc {
			case core.SecurityContextHTML:
				sanitized = callSanitizerFn("ɵɵsanitizeHtml", extAttr.Expression)
			case core.SecurityContextStyle:
				sanitized = callSanitizerFn("ɵɵsanitizeStyle", extAttr.Expression)
			case core.SecurityContextScript:
				sanitized = callSanitizerFn("ɵɵsanitizeScript", extAttr.Expression)
			case core.SecurityContextURL:
				sanitized = callSanitizerFn("ɵɵsanitizeUrl", extAttr.Expression)
			case core.SecurityContextResourceURL:
				sanitized = callSanitizerFn("ɵɵsanitizeResourceUrl", extAttr.Expression)
			case core.SecurityContextAttributeNoBinding:
				sanitized = callValidateFn("ɵɵvalidateAttribute", extAttr.Expression, tagName, extAttr.Name)
			default:
				continue
			}
			extAttr.Expression = sanitized
		}
	}
}

func callSanitizerFn(fnName string, arg output.Expression) output.Expression {
	fn := output.NewReadVarExpr(fnName, nil, nil, nil)
	return output.NewInvokeFunctionExpr(fn, []output.Expression{arg}, nil, nil, false, nil, false)
}

func callValidateFn(fnName string, arg output.Expression, tagName, attrName string) output.Expression {
	fn := output.NewReadVarExpr(fnName, nil, nil, nil)
	args := []output.Expression{
		arg,
		output.NewLiteralExpr(tagName, nil, nil, nil),
		output.NewLiteralExpr(attrName, nil, nil, nil),
	}
	return output.NewInvokeFunctionExpr(fn, args, nil, nil, false, nil, false)
}
