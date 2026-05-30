package i18n

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	o "github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
)

func CreateLocalizeStatements(
	variable *o.ReadVarExpr,
	message *i18n.Message,
	params map[string]o.Expression,
) []o.Statement {
	messageParts, placeHolders := SerializeI18nMessageForLocalize(message)

	var expressions []o.Expression
	for _, ph := range placeHolders {
		expressions = append(expressions, params[ph.Text])
	}

	var sourceSpan *parse_util.ParseSourceSpan
	if message != nil {
		sourceSpan = getSourceSpan(message)
	}

	localizedString := o.NewLocalizedString(
		message,
		messageParts,
		placeHolders,
		expressions,
		o.NewSourceSpanAdapter(sourceSpan),
		nil,
	)
	variableInitialization := variable.Set(localizedString)
	return []o.Statement{
		o.NewExpressionStatement(variableInitialization, nil, []o.LeadingComment{
			o.LeadingComment{Text: "@suppress {missingRequire}", Multiline: true, TrailingNewline: true}, // tsIgnoreComment equivalent
		}),
	}
}

type localizeSerializerVisitor struct {
	placeholderToMessage map[string]*i18n.Message
	pieces               []any
}

func (v *localizeSerializerVisitor) VisitText(text *i18n.Text, context any) any {
	if len(v.pieces) > 0 {
		lastIdx := len(v.pieces) - 1
		if lp, ok := v.pieces[lastIdx].(*o.LiteralPiece); ok {
			lp.Text += text.Value
			return nil
		}
	}
	var sourceSpan *parse_util.ParseSourceSpan
	if text.SourceSpan != nil {
		sourceSpan = parse_util.NewParseSourceSpan(
			text.SourceSpan.FullStart,
			text.SourceSpan.End,
			text.SourceSpan.FullStart,
			text.SourceSpan.Details,
		)
	}
	v.pieces = append(v.pieces, &o.LiteralPiece{Text: text.Value, SourceSpan: o.NewSourceSpanAdapter(sourceSpan)})
	return nil
}

func (v *localizeSerializerVisitor) VisitContainer(container *i18n.Container, context any) any {
	for _, child := range container.Children {
		child.Visit(v, context)
	}
	return nil
}

func (v *localizeSerializerVisitor) VisitIcu(icu *i18n.Icu, context any) any {
	v.pieces = append(v.pieces, &o.LiteralPiece{Text: SerializeIcuNode(icu), SourceSpan: o.NewSourceSpanAdapter(icu.SourceSpan)})
	return nil
}

func (v *localizeSerializerVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context any) any {
	var startSpan *parse_util.ParseSourceSpan
	if ph.StartSourceSpan != nil {
		startSpan = ph.StartSourceSpan
	} else {
		startSpan = ph.SourceSpan
	}
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.StartName, startSpan, nil))

	if !ph.IsVoid {
		for _, child := range ph.Children {
			child.Visit(v, context)
		}
		var endSpan *parse_util.ParseSourceSpan
		if ph.EndSourceSpan != nil {
			endSpan = ph.EndSourceSpan
		} else {
			endSpan = ph.SourceSpan
		}
		v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.CloseName, endSpan, nil))
	}
	return nil
}

func (v *localizeSerializerVisitor) VisitPlaceholder(ph *i18n.Placeholder, context any) any {
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.Name, ph.SourceSpan, nil))
	return nil
}

func (v *localizeSerializerVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context any) any {
	var startSpan *parse_util.ParseSourceSpan
	if ph.StartSourceSpan != nil {
		startSpan = ph.StartSourceSpan
	} else {
		startSpan = ph.SourceSpan
	}
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.StartName, startSpan, nil))

	for _, child := range ph.Children {
		child.Visit(v, context)
	}

	var endSpan *parse_util.ParseSourceSpan
	if ph.EndSourceSpan != nil {
		endSpan = ph.EndSourceSpan
	} else {
		endSpan = ph.SourceSpan
	}
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.CloseName, endSpan, nil))
	return nil
}

func (v *localizeSerializerVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context any) any {
	associatedMessage := v.placeholderToMessage[ph.Name]
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.Name, ph.SourceSpan, associatedMessage))
	return nil
}

func (v *localizeSerializerVisitor) createPlaceholderPiece(name string, sourceSpan *parse_util.ParseSourceSpan, associatedMessage *i18n.Message) *o.PlaceholderPiece {
	var assoc o.Message = nil
	if associatedMessage != nil {
		assoc = associatedMessage
	}
	return &o.PlaceholderPiece{
		Text:              FormatI18nPlaceholderName(name, false),
		SourceSpan:        o.NewSourceSpanAdapter(sourceSpan),
		AssociatedMessage: assoc,
	}
}

func SerializeI18nMessageForLocalize(message *i18n.Message) ([]o.LiteralPiece, []o.PlaceholderPiece) {
	var pieces []any
	serializerVisitor := &localizeSerializerVisitor{
		placeholderToMessage: message.PlaceholderToMessage,
		pieces:               pieces,
	}
	for _, node := range message.Nodes {
		node.Visit(serializerVisitor, nil)
	}
	return processMessagePieces(serializerVisitor.pieces)
}

func getSourceSpan(message *i18n.Message) *parse_util.ParseSourceSpan {
	if len(message.Nodes) == 0 {
		return nil
	}
	startNode := message.Nodes[0]
	endNode := message.Nodes[len(message.Nodes)-1]
	startSpan := startNode.GetSourceSpan()
	endSpan := endNode.GetSourceSpan()
	if startSpan == nil || endSpan == nil {
		return nil
	}
	return parse_util.NewParseSourceSpan(
		startSpan.FullStart,
		endSpan.End,
		startSpan.FullStart,
		startSpan.Details,
	)
}

func processMessagePieces(pieces []any) ([]o.LiteralPiece, []o.PlaceholderPiece) {
	var messageParts []o.LiteralPiece
	var placeHolders []o.PlaceholderPiece

	if len(pieces) == 0 {
		return messageParts, placeHolders
	}

	if ph, isPh := pieces[0].(*o.PlaceholderPiece); isPh {
		var startLoc *parse_util.ParseLocation
		if ph.SourceSpan != nil {
			if adapter, ok := ph.SourceSpan.(*o.SourceSpanAdapter); ok && adapter.Span != nil {
				startLoc = adapter.Span.Start
			}
		}
		messageParts = append(messageParts, createEmptyMessagePart(startLoc))
	}

	for i := 0; i < len(pieces); i++ {
		part := pieces[i]
		if lp, isLp := part.(*o.LiteralPiece); isLp {
			messageParts = append(messageParts, *lp)
		} else if ph, isPh := part.(*o.PlaceholderPiece); isPh {
			placeHolders = append(placeHolders, *ph)
			if i > 0 {
				if prevPh, prevIsPh := pieces[i-1].(*o.PlaceholderPiece); prevIsPh {
					var endLoc *parse_util.ParseLocation
					if prevPh.SourceSpan != nil {
						if adapter, ok := prevPh.SourceSpan.(*o.SourceSpanAdapter); ok && adapter.Span != nil {
							endLoc = adapter.Span.End
						}
					}
					messageParts = append(messageParts, createEmptyMessagePart(endLoc))
				}
			}
		}
	}

	if ph, isPh := pieces[len(pieces)-1].(*o.PlaceholderPiece); isPh {
		var endLoc *parse_util.ParseLocation
		if ph.SourceSpan != nil {
			if adapter, ok := ph.SourceSpan.(*o.SourceSpanAdapter); ok && adapter.Span != nil {
				endLoc = adapter.Span.End
			}
		}
		messageParts = append(messageParts, createEmptyMessagePart(endLoc))
	}

	return messageParts, placeHolders
}

func createEmptyMessagePart(location *parse_util.ParseLocation) o.LiteralPiece {
	var span *parse_util.ParseSourceSpan
	if location != nil {
		span = parse_util.NewParseSourceSpan(location, location, nil, nil)
	}
	return o.LiteralPiece{Text: "", SourceSpan: o.NewSourceSpanAdapter(span)}
}
