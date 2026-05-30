package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CreateI18nContexts creates one helper context op per i18n block (including generate descending blocks).
//
// Also, if an ICU exists inside an i18n block that also contains other localizable content (such as
// string), create an additional helper context op for the ICU.
//
// These context ops are later used for generating i18n messages. (Although we generate at least one
// context op per nested view, we will collect them up the tree later, to generate a top-level
// message.)
func CreateI18nContexts(job compilation.CompilationJob) {
	// 1. Create i18n context ops for i18n attrs.
	attrContextByMessage := make(map[*i18n.Message]ir.XrefId)
	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			var msg *i18n.Message
			switch o := op.(type) {
			case *ir.BindingOp:
				if o.I18nMessage != nil {
					msg, _ = o.I18nMessage.(*i18n.Message)
				}
			case *ir.PropertyOp:
				if o.I18nMessage != nil {
					msg, _ = o.I18nMessage.(*i18n.Message)
				}
			case *ir.AttributeOp:
				if o.I18nMessage != nil {
					msg, _ = o.I18nMessage.(*i18n.Message)
				}
			case *ir.ExtractedAttributeOp:
				if o.I18nMessage != nil {
					msg, _ = o.I18nMessage.(*i18n.Message)
				}
			}

			if msg == nil {
				continue
			}

			if _, exists := attrContextByMessage[msg]; !exists {
				i18nContext := &ir.I18nContextOp{
					ContextKind: ir.I18nContextKindAttr,
					Xref:        job.AllocateXrefId(),
					I18nBlock:   0,
					Message:     msg,
					SourceSpan:  nil,
				}
				unit.GetCreate().Push(i18nContext)
				attrContextByMessage[msg] = i18nContext.Xref
			}

			xref := attrContextByMessage[msg]
			switch o := op.(type) {
			case *ir.BindingOp:
				o.I18nContext = xref
			case *ir.PropertyOp:
				o.I18nContext = xref
			case *ir.AttributeOp:
				o.I18nContext = xref
			case *ir.ExtractedAttributeOp:
				o.I18nContext = xref
			}
		}
	}

	// 2. Create i18n context ops for root i18n blocks.
	blockContextByI18nBlock := make(map[ir.XrefId]*ir.I18nContextOp)
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Ops {
			if op.Kind() == ir.OpKindI18nStart {
				i18nStart := op.(*ir.I18nStartOp)
				if i18nStart.Xref == i18nStart.Root {
					contextOp := &ir.I18nContextOp{
						ContextKind: ir.I18nContextKindRootI18n,
						Xref:        job.AllocateXrefId(),
						I18nBlock:   i18nStart.Xref,
						Message:     i18nStart.Message,
						SourceSpan:  nil,
					}
					unit.GetCreate().Push(contextOp)
					i18nStart.Context = contextOp.Xref
					blockContextByI18nBlock[i18nStart.Xref] = contextOp
				}
			}
		}
	}

	// 3. Assign i18n contexts for child i18n blocks.
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Ops {
			if op.Kind() == ir.OpKindI18nStart {
				i18nStart := op.(*ir.I18nStartOp)
				if i18nStart.Xref != i18nStart.Root {
					rootContext, ok := blockContextByI18nBlock[i18nStart.Root]
					if !ok {
						panic("AssertionError: Root i18n block i18n context should have been created.")
					}
					i18nStart.Context = rootContext.Xref
					blockContextByI18nBlock[i18nStart.Xref] = rootContext
				}
			}
		}
	}

	// 4. Create or assign i18n contexts for ICUs.
	var currentI18nOp *ir.I18nStartOp = nil
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Ops {
			switch o := op.(type) {
			case *ir.I18nStartOp:
				currentI18nOp = o
			case *ir.I18nEndOp:
				currentI18nOp = nil
			case *ir.IcuStartOp:
				if currentI18nOp == nil {
					panic("AssertionError: Unexpected ICU outside of an i18n block.")
				}

				icuMsg, ok1 := o.Message.(*i18n.Message)
				parentMsg, ok2 := currentI18nOp.Message.(*i18n.Message)

				if ok1 && ok2 && icuMsg.Id != parentMsg.Id {
					// Sub-message ICU gets its own context
					contextOp := &ir.I18nContextOp{
						ContextKind: ir.I18nContextKindIcu,
						Xref:        job.AllocateXrefId(),
						I18nBlock:   currentI18nOp.Root,
						Message:     o.Message,
						SourceSpan:  nil,
					}
					unit.GetCreate().Push(contextOp)
					o.Context = contextOp.Xref
				} else {
					// Convert parent's context kind to ICU
					o.Context = currentI18nOp.Context
					if parentCtx, found := blockContextByI18nBlock[currentI18nOp.Xref]; found {
						parentCtx.ContextKind = ir.I18nContextKindIcu
					}
				}
			}
		}
	}
}
