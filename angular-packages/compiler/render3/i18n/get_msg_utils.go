package i18n

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	o "github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

func createMapEntries(m map[string]o.Expression) []o.LiteralMapEntry {
	var res []o.LiteralMapEntry
	for k, v := range m {
		res = append(res, o.NewLiteralMapPropertyAssignment(k, v, true))
	}
	return res
}

func CreateGoogGetMsgStatements(
	variable *o.ReadVarExpr,
	message *i18n.Message,
	closureVar *o.ReadVarExpr,
	placeholderValues map[string]o.Expression,
) []o.Statement {
	messageString := SerializeI18nMessageForGetMsg(message)
	var args []o.Expression
	args = append(args, o.NewLiteralExpr(messageString, nil, nil, nil))

	if len(placeholderValues) > 0 {
		args = append(args, o.NewLiteralMapExpr(
			createMapEntries(FormatI18nPlaceholderNamesInMap(placeholderValues, true)),
			nil, nil, nil,
		))

		// Create original_code map
		entries := make(map[string]o.Expression)
		for param := range placeholderValues {
			var valueExpr o.Expression
			if ph, ok := message.Placeholders[param]; ok {
				valueExpr = o.NewLiteralExpr("...", nil, nil, nil) // placeholder sourceSpan
				_ = ph
			} else {
				// Must be ICU expression
				if icuMsg, ok := message.PlaceholderToMessage[param]; ok {
					var spans []string
					for _, node := range icuMsg.Nodes {
						_ = node
						spans = append(spans, "...")
					}
					valueExpr = o.NewLiteralExpr(strings.Join(spans, ""), nil, nil, nil)
				} else {
					valueExpr = o.NewLiteralExpr("", nil, nil, nil)
				}
			}
			entries[FormatI18nPlaceholderName(param, false)] = valueExpr
		}

		args = append(args, o.NewLiteralMapExpr(
			[]o.LiteralMapEntry{o.NewLiteralMapPropertyAssignment("original_code", o.NewLiteralMapExpr(createMapEntries(entries), nil, nil, nil), false)},
			nil, nil, nil,
		))
	}

	googGetMsgStmt := o.NewDeclareVarStmt(
		closureVar.Name,
		o.NewReadVarExpr("goog", nil, nil, nil).Prop("getMsg", nil).CallFn(args, nil, false, nil),
		o.INFERRED_TYPE,
		o.StmtModifierFinal,
		nil,
		nil,
	)

	i18nAssignmentStmt := o.NewExpressionStatement(variable.Set(closureVar), nil, nil)
	return []o.Statement{googGetMsgStmt, i18nAssignmentStmt}
}

type getMsgSerializerVisitor struct{}

func (v *getMsgSerializerVisitor) formatPh(value string) string {
	return "{$" + FormatI18nPlaceholderName(value, true) + "}"
}

func (v *getMsgSerializerVisitor) VisitText(text *i18n.Text, context any) any {
	return text.Value
}

func (v *getMsgSerializerVisitor) VisitContainer(container *i18n.Container, context any) any {
	var results []string
	for _, child := range container.Children {
		if res := child.Visit(v, context); res != nil {
			results = append(results, res.(string))
		}
	}
	return strings.Join(results, "")
}

func (v *getMsgSerializerVisitor) VisitIcu(icu *i18n.Icu, context any) any {
	return SerializeIcuNode(icu)
}

func (v *getMsgSerializerVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context any) any {
	if ph.IsVoid {
		return v.formatPh(ph.StartName)
	}
	var children []string
	for _, child := range ph.Children {
		if res := child.Visit(v, context); res != nil {
			children = append(children, res.(string))
		}
	}
	return v.formatPh(ph.StartName) + strings.Join(children, "") + v.formatPh(ph.CloseName)
}

func (v *getMsgSerializerVisitor) VisitPlaceholder(ph *i18n.Placeholder, context any) any {
	return v.formatPh(ph.Name)
}

func (v *getMsgSerializerVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context any) any {
	var children []string
	for _, child := range ph.Children {
		if res := child.Visit(v, context); res != nil {
			children = append(children, res.(string))
		}
	}
	return v.formatPh(ph.StartName) + strings.Join(children, "") + v.formatPh(ph.CloseName)
}

func (v *getMsgSerializerVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context any) any {
	return v.formatPh(ph.Name)
}

func SerializeI18nMessageForGetMsg(message *i18n.Message) string {
	var results []string
	serializerVisitor := &getMsgSerializerVisitor{}
	for _, node := range message.Nodes {
		if res := node.Visit(serializerVisitor, nil); res != nil {
			results = append(results, res.(string))
		}
	}
	return strings.Join(results, "")
}
