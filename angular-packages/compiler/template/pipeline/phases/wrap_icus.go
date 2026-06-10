package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// WrapI18nIcus wraps ICUs that do not already belong to an i18n block in a new i18n block.
func WrapI18nIcus(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		var currentI18nOp *ir.I18nStartOp = nil
		var addedI18nId *ir.XrefId = nil
		var newOps []ir.Op

		for _, op := range unit.GetCreate().Ops {
			switch o := op.(type) {
			case *ir.I18nStartOp:
				currentI18nOp = o
				newOps = append(newOps, op)
			case *ir.I18nEndOp:
				currentI18nOp = nil
				newOps = append(newOps, op)
			case *ir.IcuStartOp:
				if currentI18nOp == nil {
					id := job.AllocateXrefId()
					addedI18nId = &id
					i18nStart := &ir.I18nStartOp{
						I18nOpBase: ir.I18nOpBase{
							Xref:         id,
							Message:      o.Message,
							Root:         id,
							SourceSpan:   nil,
							TargetSlot:   &ir.SlotHandle{},
							NumSlotsUsed: 1,
						},
					}
					newOps = append(newOps, i18nStart)
				}
				newOps = append(newOps, op)
			case *ir.IcuEndOp:
				newOps = append(newOps, op)
				if addedI18nId != nil {
					i18nEnd := &ir.I18nEndOp{
						Xref:       *addedI18nId,
						SourceSpan: nil,
					}
					newOps = append(newOps, i18nEnd)
					addedI18nId = nil
				}
			default:
				newOps = append(newOps, op)
			}
		}
		unit.GetCreate().Ops = newOps
	}
}
