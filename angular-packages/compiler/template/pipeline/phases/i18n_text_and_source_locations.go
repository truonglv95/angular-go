package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ConvertI18nText removes text nodes within i18n blocks and converts interpolations to i18n expressions.
func ConvertI18nText(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		var currentI18n *ir.I18nStartOp
		var currentIcu *ir.IcuStartOp
		textNodeI18nBlocks := map[ir.XrefId]*ir.I18nStartOp{}
		textNodeIcus := map[ir.XrefId]*ir.IcuStartOp{}
		icuPlaceholderByText := map[ir.XrefId]*ir.IcuPlaceholderOp{}

		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindI18nStart:
				if i18nOp, ok := op.(*ir.I18nStartOp); ok {
					currentI18n = i18nOp
				}
			case ir.OpKindI18nEnd:
				currentI18n = nil
			case ir.OpKindIcuStart:
				if icuOp, ok := op.(*ir.IcuStartOp); ok {
					currentIcu = icuOp
				}
			case ir.OpKindIcuEnd:
				currentIcu = nil
			case ir.OpKindText:
				if currentI18n == nil {
					continue
				}
				textOp, ok := op.(*ir.TextOp)
				if !ok {
					continue
				}
				textNodeI18nBlocks[textOp.Xref] = currentI18n
				textNodeIcus[textOp.Xref] = currentIcu
				if textOp.IcuPlaceholder != nil {
					icuPhOp := &ir.IcuPlaceholderOp{
						Xref:    job.AllocateXrefId(),
						Name:    *textOp.IcuPlaceholder,
						Strings: textOp.InitialValue, // string type
					}
					// Replace the text op with the icu placeholder op.
					ir.InsertBefore(icuPhOp, op)
					unit.GetCreate().Remove(op)
					icuPlaceholderByText[textOp.Xref] = icuPhOp
				} else {
					unit.GetCreate().Remove(op)
				}
			}
		}

		for _, op := range unit.GetUpdate().Elements() {
			if op.Kind() != ir.OpKindInterpolateText {
				continue
			}
			itOp, ok := op.(*ir.InterpolateTextOp)
			if !ok {
				continue
			}
			if _, inI18n := textNodeI18nBlocks[itOp.Target]; !inI18n {
				continue
			}
			i18nOp := textNodeI18nBlocks[itOp.Target]
			icuOp := textNodeIcus[itOp.Target]
			icuPlaceholder := icuPlaceholderByText[itOp.Target]

			var contextId ir.XrefId
			if icuOp != nil {
				contextId = icuOp.Context
			} else {
				contextId = i18nOp.Context
			}
			resolutionTime := ir.I18nParamResolutionTimeCreation
			if icuOp != nil {
				resolutionTime = ir.I18nParamResolutionTimePostproccessing
			}

			// Interpolation is `any` — type-assert to *ir.Interpolation.
			interp, _ := itOp.Interpolation.(*ir.Interpolation)
			if interp == nil {
				continue
			}

			var newOps []ir.Op
			for i, expr := range interp.Expressions {
				var ph *string
				if i < len(interp.I18nPlaceholders) {
					p := interp.I18nPlaceholders[i]
					ph = &p
				}
				var icuXrefVal ir.XrefId
				if icuPlaceholder != nil {
					icuXrefVal = icuPlaceholder.Xref
				}
				exprOp := &ir.I18nExpressionOp{
					Context:         contextId,
					Target:          i18nOp.Xref,
					I18nOwner:       i18nOp.Xref,
					Expression:      expr.(output.Expression),
					IcuPlaceholder:  icuXrefVal,
					I18nPlaceholder: ph,
					ResolutionTime:  resolutionTime,
					Usage:           ir.I18nExpressionForI18nText,
					Name:            "",
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

// AttachSourceLocations attaches source location information to element ops for debug mode.
func AttachSourceLocations(job *compilation.ComponentCompilationJob) {
	if !job.EnableDebugLocations || job.RelativeTemplatePath == nil {
		return
	}
	for _, unit := range job.GetUnits() {
		var locations []ir.ElementSourceLocation
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindElementStart && op.Kind() != ir.OpKindElement {
				continue
			}
			if slotOp, ok := op.(ir.ConsumesSlotTrait); ok {
				handle := slotOp.Handle()
				locations = append(locations, ir.ElementSourceLocation{
					TargetSlot: handle,
				})
			}
		}
		if len(locations) > 0 {
			srcLocOp := &ir.SourceLocationOp{
				TemplatePath: *job.RelativeTemplatePath,
				Locations:    locations,
			}
			unit.GetCreate().Push(srcLocOp)
		}
	}
}
