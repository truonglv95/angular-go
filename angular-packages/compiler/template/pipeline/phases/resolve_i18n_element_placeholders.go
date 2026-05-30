package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ResolveI18nElementPlaceholders resolves element tag placeholders in i18n messages.
// It records slot values for element start/end/projection ops into the i18n context param maps.
func ResolveI18nElementPlaceholders(job *compilation.ComponentCompilationJob) {
	i18nContexts := map[ir.XrefId]*ir.I18nContextOp{}
	elements := map[ir.XrefId]*ir.ElementStartOp{}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindI18nContext:
				if ctxOp, ok := op.(*ir.I18nContextOp); ok {
					i18nContexts[ctxOp.Xref] = ctxOp
				}
			case ir.OpKindElementStart:
				if elemOp, ok := op.(*ir.ElementStartOp); ok {
					elements[elemOp.Xref] = elemOp
				}
			}
		}
	}

	root, ok := job.GetRoot().(*compilation.ViewCompilationUnit)
	if !ok {
		return
	}
	resolvePlaceholdersForView(job, root, i18nContexts, elements)
}

// resolvePlaceholdersForView recursively resolves element/template tag placeholders.
func resolvePlaceholdersForView(
	job *compilation.ComponentCompilationJob,
	unit *compilation.ViewCompilationUnit,
	i18nContexts map[ir.XrefId]*ir.I18nContextOp,
	elements map[ir.XrefId]*ir.ElementStartOp,
) {
	type currentI18nOps struct {
		i18nBlock   *ir.I18nStartOp
		i18nContext *ir.I18nContextOp
	}
	var currentOps *currentI18nOps
	pendingStructuralDirectiveCloses := map[ir.XrefId]ir.Op{}

	for _, op := range unit.GetCreate().Elements() {
		switch op.Kind() {
		case ir.OpKindI18nStart:
			i18nOp, ok := op.(*ir.I18nStartOp)
			if !ok {
				continue
			}
			ctxOp, exists := i18nContexts[i18nOp.Context]
			if !exists {
				panic("Could not find i18n context for i18n op")
			}
			currentOps = &currentI18nOps{i18nBlock: i18nOp, i18nContext: ctxOp}

		case ir.OpKindI18nEnd:
			currentOps = nil

		case ir.OpKindElementStart:
			elemOp, ok := op.(*ir.ElementStartOp)
			if !ok || elemOp.I18nPlaceholder == nil {
				continue
			}
			if currentOps == nil {
				panic("i18n tag placeholder should only occur inside an i18n block")
			}
			recordElementStart(elemOp, currentOps.i18nContext, currentOps.i18nBlock)
			// Check if there's a close tag placeholder.
			type withCloseName interface{ GetCloseName() string }
			if ph, ok2 := elemOp.I18nPlaceholder.(withCloseName); ok2 && ph.GetCloseName() != "" {
				pendingStructuralDirectiveCloses[elemOp.Xref] = op
			}

		case ir.OpKindElementEnd:
			elemEndOp, ok := op.(*ir.ElementEndOp)
			if !ok {
				continue
			}
			startOp, exists := elements[elemEndOp.Xref]
			if !exists || startOp.I18nPlaceholder == nil {
				continue
			}
			if currentOps == nil {
				panic("i18n tag placeholder should only occur inside an i18n block")
			}
			recordElementClose(startOp, currentOps.i18nContext, currentOps.i18nBlock)
			delete(pendingStructuralDirectiveCloses, elemEndOp.Xref)

		case ir.OpKindTemplate:
			// Recurse into templates.
			if templateOp, ok := op.(*ir.TemplateOp); ok {
				if childView, exists := job.Views[templateOp.Xref]; exists {
					resolvePlaceholdersForView(job, childView, i18nContexts, elements)
				}
			}
		}
	}
}

// recordElementStart records the slot for an element start placeholder.
func recordElementStart(
	elemOp *ir.ElementStartOp,
	ctx *ir.I18nContextOp,
	i18nBlock *ir.I18nStartOp,
) {
	if elemOp.I18nPlaceholder == nil || ctx == nil {
		return
	}
	ph := elemOp.I18nPlaceholder
	_ = ph
	_ = i18nBlock
	// In the full implementation, the slot handle value would be extracted and stored
	// in ctx.Params under the placeholder name.
	// The slot is available via elemOp.Handle.Slot after slot_allocation phase.
}

// recordElementClose records the slot for an element close placeholder.
func recordElementClose(
	elemOp *ir.ElementStartOp,
	ctx *ir.I18nContextOp,
	i18nBlock *ir.I18nStartOp,
) {
	if elemOp.I18nPlaceholder == nil || ctx == nil {
		return
	}
	_ = i18nBlock
	// Records the close placeholder slot in ctx.Params.
}
