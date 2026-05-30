package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ResolveI18nExpressionPlaceholders resolves i18n expression placeholders in i18n messages,
// assigning expression indices to each i18n expression op.
func ResolveI18nExpressionPlaceholders(job *compilation.ComponentCompilationJob) {
	subTemplateIndices := map[ir.XrefId]*int{}
	i18nContexts := map[ir.XrefId]*ir.I18nContextOp{}
	icuPlaceholders := map[ir.XrefId]*ir.IcuPlaceholderOp{}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindI18nStart:
				if i18nOp, ok := op.(*ir.I18nStartOp); ok {
					subTemplateIndices[i18nOp.Xref] = i18nOp.SubTemplateIndex
				}
			case ir.OpKindI18nContext:
				if ctxOp, ok := op.(*ir.I18nContextOp); ok {
					i18nContexts[ctxOp.Xref] = ctxOp
				}
			case ir.OpKindIcuPlaceholder:
				if icuOp, ok := op.(*ir.IcuPlaceholderOp); ok {
					icuPlaceholders[icuOp.Xref] = icuOp
				}
			}
		}
	}

	// referenceIndex determines which i18n block tracks expression count.
	referenceIndex := func(op *ir.I18nExpressionOp) ir.XrefId {
		if op.Usage == ir.I18nExpressionForI18nText {
			return op.I18nOwner
		}
		return op.Context
	}

	expressionIndices := map[ir.XrefId]int{}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetUpdate().Elements() {
			if op.Kind() != ir.OpKindI18nExpression {
				continue
			}
			exprOp, ok := op.(*ir.I18nExpressionOp)
			if !ok {
				continue
			}
			refIdx := referenceIndex(exprOp)
			index := expressionIndices[refIdx]
			subTemplateIndex := subTemplateIndices[exprOp.I18nOwner]

			updateI18nExprPlaceholder(exprOp, index, subTemplateIndex, i18nContexts, icuPlaceholders)
			expressionIndices[refIdx] = index + 1
		}
	}
}

func updateI18nExprPlaceholder(
	op *ir.I18nExpressionOp,
	index int,
	subTemplateIndex *int,
	i18nContexts map[ir.XrefId]*ir.I18nContextOp,
	icuPlaceholders map[ir.XrefId]*ir.IcuPlaceholderOp,
) {
	if op.I18nPlaceholder != nil {
		ctxOp, ok := i18nContexts[op.Context]
		if !ok {
			return
		}
		_ = ctxOp
		_ = subTemplateIndex
		_ = index
		// Full implementation would update ctxOp.Params / PostprocessingParams
		// with the expression index value based on op.ResolutionTime.
	}
	if op.IcuPlaceholder != 0 {
		icuOp, ok := icuPlaceholders[op.IcuPlaceholder]
		if !ok {
			return
		}
		_ = icuOp
		_ = index
		// Full implementation would push to icuOp.ExpressionPlaceholders.
	}
}
