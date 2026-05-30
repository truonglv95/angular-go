package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ApplyI18nExpressions adds apply operations after i18n expressions.
func ApplyI18nExpressions(job compilation.CompilationJob) {
	i18nContexts := map[ir.XrefId]ir.Op{}
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() == ir.OpKindI18nContext {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					i18nContexts[xg.GetXref()] = op
				}
			}
		}
	}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetUpdate().Elements() {
			if op.Kind() != ir.OpKindI18nExpression {
				continue
			}
			if i18nNeedsApplication(i18nContexts, op) {
				if exprOp, ok := op.(*ir.I18nExpressionOp); ok {
					applyOp := &ir.I18nApplyOp{
						Owner:  exprOp.I18nOwner,
						Handle: exprOp.Handle,
					}
					ir.InsertAfter(applyOp, op)
				}
			}
		}
	}
}

func i18nNeedsApplication(i18nContexts map[ir.XrefId]ir.Op, op ir.Op) bool {
	nextOp := op.Next()
	if nextOp == nil || nextOp.Kind() != ir.OpKindI18nExpression {
		return true
	}

	// i18nContexts is keyed by I18nContextOp xref
	opI18n, ok1 := op.(*ir.I18nExpressionOp)
	nextI18n, ok2 := nextOp.(*ir.I18nExpressionOp)
	if !ok1 || !ok2 {
		return true
	}

	ctxOp, ctxExists := i18nContexts[opI18n.Context]
	nextCtxOp, nextCtxExists := i18nContexts[nextI18n.Context]
	if !ctxExists || !nextCtxExists {
		return true
	}

	type i18nBlockGetter interface{ GetI18nBlock() ir.XrefId }
	if blockCtx, ok := ctxOp.(i18nBlockGetter); ok {
		if nextBlockCtx, ok2 := nextCtxOp.(i18nBlockGetter); ok2 {
			return blockCtx.GetI18nBlock() != nextBlockCtx.GetI18nBlock()
		}
	}

	return opI18n.I18nOwner != nextI18n.I18nOwner
}

// RemoveI18nContexts removes i18n context ops after they are no longer needed.
func RemoveI18nContexts(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindI18nContext:
				unit.GetCreate().Remove(op)
			case ir.OpKindI18nStart:
				if setter, ok := op.(interface{ SetContext(ir.XrefId) }); ok {
					setter.SetContext(0)
				}
			}
		}
	}
}

// RemoveUnusedI18nAttributesOps removes i18nAttributes ops that contain no dynamic content.
func RemoveUnusedI18nAttributesOps(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		ownersWithI18nExpressions := map[ir.XrefId]bool{}
		for _, op := range unit.GetUpdate().Elements() {
			if op.Kind() == ir.OpKindI18nExpression {
				if owner, ok := op.(interface{ GetI18nOwner() ir.XrefId }); ok {
					ownersWithI18nExpressions[owner.GetI18nOwner()] = true
				}
			}
		}
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() == ir.OpKindI18nAttributes {
				if xg, ok := op.(interface{ GetXref() ir.XrefId }); ok {
					if !ownersWithI18nExpressions[xg.GetXref()] {
						unit.GetCreate().Remove(op)
					}
				}
			}
		}
	}
}
