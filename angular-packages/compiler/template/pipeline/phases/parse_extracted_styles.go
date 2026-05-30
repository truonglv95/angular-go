package phases

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

const (
	charOpenParen   = 40
	charCloseParen  = 41
	charColon       = 58
	charSemicolon   = 59
	charBackSlash   = 92
	charQuoteNone   = 0
	charQuoteDouble = 34
	charQuoteSingle = 39
)

// ParseExtractedStyles parses inline style and class attribute string constants into individual ExtractedAttributeOps.
func ParseExtractedStyles(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		elements := make(map[ir.XrefId]ir.Op)
		for _, createOp := range unit.GetCreate().Ops {
			if ir.IsElementOrContainerOp(createOp) {
				if xref, ok := getXrefId(createOp); ok {
					elements[xref] = createOp
				}
			}
		}

		var newCreateOps []ir.Op
		for _, op := range unit.GetCreate().Ops {
			if op.Kind() == ir.OpKindExtractedAttribute {
				attrOp := op.(*ir.ExtractedAttributeOp)
				bKind, _ := attrOp.BindingKind.(ir.BindingKind)

				if bKind == ir.BindingKindAttribute && attrOp.Expression != nil && isStringLiteral(attrOp.Expression) {
					target := elements[attrOp.Target]

					if target != nil {
						if target.Kind() == ir.OpKindTemplate ||
							target.Kind() == ir.OpKindConditionalCreate ||
							target.Kind() == ir.OpKindConditionalBranchCreate {
							if k, ok := getTemplateKind(target); ok && k == ir.TemplateKindStructural {
								newCreateOps = append(newCreateOps, op)
								continue
							}
						}
					}

					valStr := attrOp.Expression.(*output.LiteralExpr).Value.(string)

					if attrOp.Name == "style" {
						parsedStyles := parseStyles(valStr)
						for i := 0; i < len(parsedStyles)-1; i += 2 {
							styleAttr := &ir.ExtractedAttributeOp{
								Target:          attrOp.Target,
								BindingKind:     ir.BindingKindStyleProperty,
								Namespace:       nil,
								Name:            parsedStyles[i],
								Expression:      output.NewLiteralExpr(parsedStyles[i+1], nil, nil, nil),
								SecurityContext: core.SecurityContextStyle,
							}
							newCreateOps = append(newCreateOps, styleAttr)
						}
						// Filter out the original "style" attribute op
						continue
					} else if attrOp.Name == "class" {
						parsedClasses := strings.Fields(valStr)
						for _, cls := range parsedClasses {
							classAttr := &ir.ExtractedAttributeOp{
								Target:          attrOp.Target,
								BindingKind:     ir.BindingKindClassName,
								Namespace:       nil,
								Name:            cls,
								Expression:      nil,
								SecurityContext: core.SecurityContextNone,
							}
							newCreateOps = append(newCreateOps, classAttr)
						}
						// Filter out the original "class" attribute op
						continue
					}
				}
			}
			newCreateOps = append(newCreateOps, op)
		}
		unit.GetCreate().Ops = newCreateOps
	}
}

func parseStyles(value string) []string {
	styles := []string{}
	i := 0
	parenDepth := 0
	quote := charQuoteNone
	valueStart := 0
	propStart := 0
	var currentProp *string = nil

	runes := []rune(value)
	length := len(runes)

	for i < length {
		token := runes[i]
		i++
		switch token {
		case charOpenParen:
			parenDepth++
		case charCloseParen:
			parenDepth--
		case charQuoteSingle:
			if quote == charQuoteNone {
				quote = charQuoteSingle
			} else if quote == charQuoteSingle {
				var escapeCheckRune rune = 0
				if i-2 >= 0 {
					escapeCheckRune = runes[i-2]
				}
				if escapeCheckRune != charBackSlash {
					quote = charQuoteNone
				}
			}
		case charQuoteDouble:
			if quote == charQuoteNone {
				quote = charQuoteDouble
			} else if quote == charQuoteDouble {
				var escapeCheckRune rune = 0
				if i-2 >= 0 {
					escapeCheckRune = runes[i-2]
				}
				if escapeCheckRune != charBackSlash {
					quote = charQuoteNone
				}
			}
		case charColon:
			if currentProp == nil && parenDepth == 0 && quote == charQuoteNone {
				propStr := hyphenate(strings.TrimSpace(string(runes[propStart : i-1])))
				currentProp = &propStr
				valueStart = i
			}
		case charSemicolon:
			if currentProp != nil && valueStart > 0 && parenDepth == 0 && quote == charQuoteNone {
				styleVal := strings.TrimSpace(string(runes[valueStart : i-1]))
				styles = append(styles, *currentProp, styleVal)
				propStart = i
				valueStart = 0
				currentProp = nil
			}
		}
	}

	if currentProp != nil && valueStart > 0 {
		styleVal := strings.TrimSpace(string(runes[valueStart:]))
		styles = append(styles, *currentProp, styleVal)
	}

	return styles
}

func isStringLiteral(expr output.Expression) bool {
	lit, ok := expr.(*output.LiteralExpr)
	if !ok {
		return false
	}
	_, isStr := lit.Value.(string)
	return isStr
}

func getTemplateKind(op ir.Op) (ir.TemplateKind, bool) {
	switch o := op.(type) {
	case *ir.TemplateOp:
		if k, ok := o.TemplateKind.(ir.TemplateKind); ok {
			return k, true
		}
	case *ir.ConditionalCreateOp:
		if k, ok := o.TemplateKind.(ir.TemplateKind); ok {
			return k, true
		}
	case *ir.ConditionalBranchCreateOp:
		if k, ok := o.TemplateKind.(ir.TemplateKind); ok {
			return k, true
		}
	}
	return 0, false
}
