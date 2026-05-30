package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CollectI18nConsts collects i18n message ops into the const array and generates
// the translation call expressions.
func CollectI18nConsts(job *compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindI18nMessage {
				continue
			}
			msgOp, ok := op.(*ir.I18nMessageOp)
			if !ok {
				continue
			}
			// Generate a const expression for this i18n message.
			msgExpr := buildI18nMessageExpr(job, msgOp)
			// Store result in Message field (overwrite with expression representation).
			msgOp.Message = msgExpr
		}
	}
}

// buildI18nMessageExpr creates an output expression for an i18n message.
func buildI18nMessageExpr(_ *compilation.ComponentCompilationJob, msgOp *ir.I18nMessageOp) output.Expression {
	// Build params map from the message's params.
	params := buildI18nParamsMap(msgOp.Params)
	_ = params

	// For now, return a stub - the full implementation requires i18n message serialization.
	// In the complete implementation, this generates either:
	// - $localize template literal (for non-closure mode)
	// - goog.getMsg() calls (for closure mode)
	return output.NewLiteralExpr(msgOp.Xref, nil, nil, nil)
}

// buildI18nParamsMap converts i18n params into output expressions.
func buildI18nParamsMap(params interface{}) map[string]output.Expression {
	result := map[string]output.Expression{}
	if params == nil {
		return result
	}
	if paramsMap, ok := params.(map[string]interface{}); ok {
		for k, v := range paramsMap {
			if expr, ok2 := v.(output.Expression); ok2 {
				result[k] = expr
			} else {
				result[k] = output.NewLiteralExpr(v, nil, nil, nil)
			}
		}
	}
	return result
}
