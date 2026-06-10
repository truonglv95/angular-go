package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// PropagateI18nBlocks propagates i18n blocks down through child templates that act as placeholders in the root i18n
// message. Specifically, perform an in-order traversal of all the views, and add i18nStart/i18nEnd
// op pairs into descending views. Also, assign an increasing sub-template index to each
// descending view.
func PropagateI18nBlocks(job compilation.CompilationJob) {
	cJob, ok := job.(*compilation.ComponentCompilationJob)
	if !ok {
		return
	}
	propagateI18nBlocksToTemplates(cJob, cJob.Root, 0)
}

func propagateI18nBlocksToTemplates(
	cJob *compilation.ComponentCompilationJob,
	unit *compilation.ViewCompilationUnit,
	subTemplateIndex int,
) int {
	var i18nBlock *ir.I18nStartOp = nil
	for _, op := range unit.GetCreate().Ops {
		switch o := op.(type) {
		case *ir.I18nStartOp:
			if subTemplateIndex == 0 {
				o.SubTemplateIndex = nil
			} else {
				idx := subTemplateIndex
				o.SubTemplateIndex = &idx
			}
			i18nBlock = o
		case *ir.I18nEndOp:
			if i18nBlock != nil && i18nBlock.SubTemplateIndex == nil {
				subTemplateIndex = 0
			}
			i18nBlock = nil
		case *ir.ConditionalCreateOp:
			subTemplateIndex = propagateI18nBlocksForView(cJob, cJob.Views[o.Xref], i18nBlock, o.I18nPlaceholder, subTemplateIndex)
		case *ir.ConditionalBranchCreateOp:
			subTemplateIndex = propagateI18nBlocksForView(cJob, cJob.Views[o.Xref], i18nBlock, o.I18nPlaceholder, subTemplateIndex)
		case *ir.TemplateOp:
			subTemplateIndex = propagateI18nBlocksForView(cJob, cJob.Views[o.Xref], i18nBlock, o.I18nPlaceholder, subTemplateIndex)
		case *ir.RepeaterCreateOp:
			subTemplateIndex = propagateI18nBlocksForView(cJob, cJob.Views[o.Xref], i18nBlock, o.I18nPlaceholder, subTemplateIndex)
			if o.EmptyView != 0 && cJob.Views[o.EmptyView] != nil {
				subTemplateIndex = propagateI18nBlocksForView(cJob, cJob.Views[o.EmptyView], i18nBlock, o.EmptyI18nPlaceholder, subTemplateIndex)
			}
		case *ir.ProjectionOp:
			if o.FallbackView != 0 && cJob.Views[o.FallbackView] != nil {
				subTemplateIndex = propagateI18nBlocksForView(cJob, cJob.Views[o.FallbackView], i18nBlock, o.FallbackViewI18nPlaceholder, subTemplateIndex)
			}
		}
	}
	return subTemplateIndex
}

func propagateI18nBlocksForView(
	cJob *compilation.ComponentCompilationJob,
	view *compilation.ViewCompilationUnit,
	i18nBlock *ir.I18nStartOp,
	i18nPlaceholder any,
	subTemplateIndex int,
) int {
	if view == nil {
		return subTemplateIndex
	}
	if i18nPlaceholder != nil {
		if i18nBlock == nil {
			panic("Expected template with i18n placeholder to be in an i18n block.")
		}
		subTemplateIndex++
		wrapTemplateWithI18n(cJob, view, i18nBlock)
	}
	return propagateI18nBlocksToTemplates(cJob, view, subTemplateIndex)
}

func wrapTemplateWithI18n(
	cJob *compilation.ComponentCompilationJob,
	unit *compilation.ViewCompilationUnit,
	parentI18n *ir.I18nStartOp,
) {
	createOps := unit.GetCreate()
	if len(createOps.Ops) == 0 || createOps.Ops[0].Kind() != ir.OpKindI18nStart {
		id := cJob.AllocateXrefId()
		i18nStart := &ir.I18nStartOp{
			I18nOpBase: ir.I18nOpBase{
				Xref:         id,
				Message:      parentI18n.Message,
				Root:         parentI18n.Root,
				SourceSpan:   nil,
				TargetSlot:   &ir.SlotHandle{},
				NumSlotsUsed: 1,
			},
		}
		i18nEnd := &ir.I18nEndOp{
			Xref:       id,
			SourceSpan: nil,
		}
		createOps.Ops = append([]ir.Op{i18nStart}, append(createOps.Ops, i18nEnd)...)
	}
}
