package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ResolveSanitizers resolves sanitization functions for ops that need them.
func ResolveSanitizers(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// For Host jobs, skip the trusted value step.
		if job.GetKind() != compilation.CompilationJobKind_Host {
			for _, op := range unit.GetCreate().Elements() {
				if op.Kind() != ir.OpKindExtractedAttribute {
					continue
				}
				extAttr, ok := op.(*ir.ExtractedAttributeOp)
				if !ok {
					continue
				}
				sc := getOnlySecurityContextValue(extAttr.SecurityContext)
				if fn := trustedValueFnForContext(sc); fn != nil {
					extAttr.TrustedValueFn = fn
				}
			}
		}

		for _, op := range unit.GetUpdate().Elements() {
			switch op.Kind() {
			case ir.OpKindProperty, ir.OpKindAttribute, ir.OpKindDomProperty:
				type sanitizerSetter interface {
					GetSecurityContext() any
					SetSanitizer(output.Expression)
				}
				if setter, ok := op.(sanitizerSetter); ok {
					sc := setter.GetSecurityContext()
					var sanitizerFn output.Expression
					scSlice, isSlice := sc.([]core.SecurityContext)
					if isSlice && len(scSlice) == 2 &&
						containsSecurityContext(scSlice, core.SecurityContextURL) &&
						containsSecurityContext(scSlice, core.SecurityContextResourceURL) {
						sanitizerFn = output.NewReadVarExpr("ɵɵsanitizeUrlOrResourceUrl", nil, nil, nil)
					} else {
						sanitizerFn = sanitizerFnForContext(getOnlySecurityContextValue(sc))
					}
					setter.SetSanitizer(sanitizerFn)
				}
			}
		}
	}
}

func getOnlySecurityContextValue(sc any) core.SecurityContext {
	switch v := sc.(type) {
	case core.SecurityContext:
		return v
	case []core.SecurityContext:
		if len(v) > 1 {
			panic("AssertionError: Ambiguous security context")
		}
		if len(v) == 0 {
			return core.SecurityContextNone
		}
		return v[0]
	}
	return core.SecurityContextNone
}

func containsSecurityContext(slice []core.SecurityContext, target core.SecurityContext) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

func sanitizerFnForContext(sc core.SecurityContext) output.Expression {
	switch sc {
	case core.SecurityContextHTML:
		return output.NewReadVarExpr("ɵɵsanitizeHtml", nil, nil, nil)
	case core.SecurityContextResourceURL:
		return output.NewReadVarExpr("ɵɵsanitizeResourceUrl", nil, nil, nil)
	case core.SecurityContextScript:
		return output.NewReadVarExpr("ɵɵsanitizeScript", nil, nil, nil)
	case core.SecurityContextStyle:
		return output.NewReadVarExpr("ɵɵsanitizeStyle", nil, nil, nil)
	case core.SecurityContextURL:
		return output.NewReadVarExpr("ɵɵsanitizeUrl", nil, nil, nil)
	case core.SecurityContextAttributeNoBinding:
		return output.NewReadVarExpr("ɵɵvalidateAttribute", nil, nil, nil)
	}
	return nil
}

func trustedValueFnForContext(sc core.SecurityContext) output.Expression {
	switch sc {
	case core.SecurityContextHTML:
		return output.NewReadVarExpr("ɵɵtrustConstantHtml", nil, nil, nil)
	case core.SecurityContextResourceURL:
		return output.NewReadVarExpr("ɵɵtrustConstantResourceUrl", nil, nil, nil)
	}
	return nil
}
