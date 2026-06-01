package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// AssignI18nSlotDependencies updates i18n expression ops to target the last slot in their
// owning i18n block and moves them after the last update instruction depending on that slot.
func AssignI18nSlotDependencies(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		type blockState struct {
			blockXref       ir.XrefId
			lastSlotConsumer ir.XrefId
		}

		// The first update op.
		var updateOp ir.Op
		if elems := unit.GetUpdate().Elements(); len(elems) > 0 {
			updateOp = elems[0]
		}

		var i18nExpressionsInProgress []*ir.I18nExpressionOp
		var state *blockState

		for _, createOp := range unit.GetCreate().Elements() {
			switch createOp.Kind() {
			case ir.OpKindI18nStart:
				if i18nOp, ok := createOp.(interface{ GetXref() ir.XrefId }); ok {
					state = &blockState{blockXref: i18nOp.GetXref(), lastSlotConsumer: i18nOp.GetXref()}
				}
			case ir.OpKindI18nEnd:
				for _, exprOp := range i18nExpressionsInProgress {
					if state != nil {
						exprOp.Target = state.lastSlotConsumer
					}
					if updateOp != nil {
						ir.InsertBefore(exprOp, updateOp)
					}
				}
				i18nExpressionsInProgress = i18nExpressionsInProgress[:0]
				state = nil
			}

			trait, hasSlot := createOp.(ir.ConsumesSlotTrait)
			if !hasSlot {
				continue
			}
			if state != nil {
				state.lastSlotConsumer = trait.GetXref()
			}

			for {
				if updateOp == nil || updateOp.Kind() == ir.OpKindListEnd {
					break
				}
				nextOp := updateOp.Next()

				if state != nil {
					if exprOp, ok := updateOp.(*ir.I18nExpressionOp); ok &&
						exprOp.Usage == ir.I18nExpressionForI18nText &&
						exprOp.I18nOwner == state.blockXref {
						unit.GetUpdate().Remove(updateOp)
						i18nExpressionsInProgress = append(i18nExpressionsInProgress, exprOp)
						updateOp = nextOp
						continue
					}
				}

				hasDifferentTarget := false
				if depOp, ok := updateOp.(ir.DependsOnSlotContextOpTrait); ok {
					if depOp.GetTarget() != trait.GetXref() {
						hasDifferentTarget = true
					}
				} else if updateOp.Kind() == ir.OpKindStatement || updateOp.Kind() == ir.OpKindVariable {
					ir.VisitExpressionsInOp(updateOp, func(expr ir.Expression) {
						if !hasDifferentTarget {
							if depExpr, ok2 := expr.(ir.DependsOnSlotContextOpTrait); ok2 {
								if depExpr.GetTarget() != trait.GetXref() {
									hasDifferentTarget = true
								}
							}
						}
					})
				}

				if hasDifferentTarget {
					break
				}
				updateOp = nextOp
			}
		}
	}
}

// ConvertI18nBindings replaces i18n binding ops in the update block with i18nExp instructions.
func ConvertI18nBindings(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		i18nAttributesByElem := map[ir.XrefId]*ir.I18nAttributesOp{}
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() == ir.OpKindI18nAttributes {
				if attrOp, ok := op.(*ir.I18nAttributesOp); ok {
					i18nAttributesByElem[attrOp.Target] = attrOp
				}
			}
		}

		for _, op := range unit.GetUpdate().Elements() {
			switch op.Kind() {
			case ir.OpKindProperty, ir.OpKindAttribute:
				type bindingOp interface {
					GetI18nContext() ir.XrefId
					GetExpression() ir.Expression
					GetTarget() ir.XrefId
					GetName() string
					GetSourceSpan() interface{}
				}
				bop, ok := op.(bindingOp)
				if !ok {
					continue
				}
				if bop.GetI18nContext() == 0 {
					continue
				}
				// Interpolation embeds output.Expression, so can't directly type-assert ir.Expression to *ir.Interpolation.
				// Use interface trick: cast expression raw to check it's actually a *ir.Interpolation.
				var interp *ir.Interpolation
				if raw := bop.GetExpression(); raw != nil {
					if ip, ok2 := any(raw).(*ir.Interpolation); ok2 {
						interp = ip
					}
				}
				if interp == nil {
					continue
				}
				i18nAttr, exists := i18nAttributesByElem[bop.GetTarget()]
				if !exists {
					panic("AssertionError: An i18n attribute binding instruction requires the owning element to have an I18nAttributes create instruction")
				}

				var newOps []ir.Op
				for i, expr := range interp.Expressions {
					if len(interp.I18nPlaceholders) != len(interp.Expressions) {
						panic("AssertionError: mismatched expressions and placeholders in i18n attribute binding")
					}
					ph := interp.I18nPlaceholders[i]
					exprOp := &ir.I18nExpressionOp{
						Context:         bop.GetI18nContext(),
						Target:          i18nAttr.Target,
						I18nOwner:       i18nAttr.Target,
						Handle:          nil, // resolved by slot_allocation
						Expression:      expr,
						I18nPlaceholder: &ph,
						ResolutionTime:  ir.I18nParamResolutionTimeCreation,
						Usage:           ir.I18nExpressionForI18nAttribute,
						Name:            bop.GetName(),
					}
					newOps = append(newOps, exprOp)
				}
				for _, newOp := range newOps {
					ir.InsertBefore(newOp, op)
				}
				unit.GetUpdate().Remove(op)
			}
		}
	}
}
