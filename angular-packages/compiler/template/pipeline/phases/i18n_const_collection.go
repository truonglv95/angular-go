package phases

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CollectI18nConsts collects i18n message ops into the const array and generates
// the translation call expressions.
func CollectI18nConsts(job *compilation.ComponentCompilationJob) {
	i18nBlocks := map[ir.XrefId]ir.Op{}
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if i18nStart, ok := op.(*ir.I18nStartOp); ok {
				i18nBlocks[i18nStart.Xref] = op
			} else if i18nOp, ok := op.(*ir.I18nOp); ok {
				i18nBlocks[i18nOp.Xref] = op
			}
		}
	}

	for _, unit := range job.GetUnits() {
		var newCreateOps []ir.Op
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindI18nMessage {
				newCreateOps = append(newCreateOps, op)
				continue
			}
			msgOp, ok := op.(*ir.I18nMessageOp)
			if !ok {
				newCreateOps = append(newCreateOps, op)
				continue
			}
			
			// Generate a const expression for this i18n message.
			msgExpr := buildI18nMessageExpr(job, msgOp)
			constIdx := job.AddConst(msgExpr, nil)
			fmt.Printf("[go-ngc-debug] ADDING CONST %v: %v\n", constIdx, msgOp)

			blockOp := i18nBlocks[msgOp.I18nBlock]
			if startOp, ok := blockOp.(*ir.I18nStartOp); ok {
				startOp.MessageIndex = constIdx
			} else if i18nOp, ok := blockOp.(*ir.I18nOp); ok {
				i18nOp.MessageIndex = constIdx
			}
		}
		unit.GetCreate().Ops = newCreateOps
	}
}

// buildI18nMessageExpr creates an output expression for an i18n message.
func buildI18nMessageExpr(_ *compilation.ComponentCompilationJob, msgOp *ir.I18nMessageOp) output.Expression {
	// Build params map from the message's params.
	params := buildI18nParamsMap(msgOp.Params)
	_ = params

	// Generate $localize template literal
	if msg, ok := msgOp.Message.(*i18n.Message); ok {
		messageParts := []output.LiteralPiece{{Text: msg.MessageString}}
		return output.NewLocalizedString(msg, messageParts, nil, nil, nil, nil)
	}
	messageParts := []output.LiteralPiece{{Text: fmt.Sprintf("%v", msgOp.Xref)}}
	return output.NewLocalizedString(msgOp.Xref, messageParts, nil, nil, nil, nil)
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
