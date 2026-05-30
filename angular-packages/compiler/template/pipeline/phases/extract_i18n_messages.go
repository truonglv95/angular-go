package phases

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

const (
	i18nEscape        = "\uFFFD"
	i18nElementMarker  = "#"
	i18nTemplateMarker = "*"
	i18nCloseMarker    = "/"
	i18nContextMarker  = ":"
	i18nListStart      = "["
	i18nListEnd        = "]"
	i18nListDelimiter  = "|"
)

// ExtractI18nMessages formats param maps on extracted message ops into Expression maps.
func ExtractI18nMessages(job compilation.CompilationJob) {
	i18nMessagesByContext := map[ir.XrefId]*ir.I18nMessageOp{}
	i18nBlocks := map[ir.XrefId]*ir.I18nStartOp{}
	i18nContexts := map[ir.XrefId]*ir.I18nContextOp{}

	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindI18nContext:
				if ctxOp, ok := op.(*ir.I18nContextOp); ok {
					msgOp := createI18nMessage(job, ctxOp)
					unit.GetCreate().Push(msgOp)
					i18nMessagesByContext[ctxOp.Xref] = msgOp
					i18nContexts[ctxOp.Xref] = ctxOp
				}
			case ir.OpKindI18nStart:
				if i18nOp, ok := op.(*ir.I18nStartOp); ok {
					i18nBlocks[i18nOp.Xref] = i18nOp
				}
			}
		}
	}

	// Associate sub-messages for ICUs with root message.
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			switch op.Kind() {
			case ir.OpKindIcuStart:
				if icuOp, ok := op.(*ir.IcuStartOp); ok {
					ctxOp, ctxExists := i18nContexts[icuOp.Context]
					if !ctxExists {
						continue
					}
					i18nBlock := i18nBlocks[ctxOp.I18nBlock]
					if i18nBlock == nil {
						continue
					}
					msgOp, exists := i18nMessagesByContext[icuOp.Context]
					if exists {
						rootMsgOp, rootExists := i18nMessagesByContext[i18nBlock.Context]
						if rootExists {
							rootMsgOp.SubMessages = msgOp.Xref // XrefId of sub-message
						}
					}
				}
			case ir.OpKindIcuEnd:
				unit.GetCreate().Remove(op)
			}
		}
	}

	// Format i18n expression params.
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetUpdate().Elements() {
			if op.Kind() != ir.OpKindI18nExpression {
				continue
			}
			exprOp, ok := op.(*ir.I18nExpressionOp)
			if !ok {
				continue
			}
			msgOp, exists := i18nMessagesByContext[exprOp.Context]
			if !exists {
				continue
			}
			if exprOp.I18nPlaceholder == nil {
				continue
			}
			phName := *exprOp.I18nPlaceholder
			_ = msgOp
			_ = phName
		}
	}
}

// createI18nMessage creates an I18nMessageOp for the given context.
func createI18nMessage(_ compilation.CompilationJob, ctx *ir.I18nContextOp) *ir.I18nMessageOp {
	return &ir.I18nMessageOp{
		Xref:        ctx.Xref,
		I18nBlock:   ctx.I18nBlock,
		Message:     ctx.Message,
		NeedsPostprocessing: ctx.PostprocessingParams != "",
	}
}

// FormatI18nPlaceholderName formats a placeholder name with escape sequences.
func FormatI18nPlaceholderName(name string, useCamelCase bool) string {
	if !useCamelCase {
		return name
	}
	return strings.ToUpper(name)
}

// formatI18nParam formats an i18n param value.
func formatI18nParam(value interface{}) output.Expression {
	switch v := value.(type) {
	case string:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case []interface{}:
		parts := make([]string, len(v))
		for i, p := range v {
			parts[i] = fmt.Sprintf("%v", p)
		}
		return output.NewLiteralExpr(strings.Join(parts, i18nListDelimiter), nil, nil, nil)
	}
	return output.NULL_EXPR
}
