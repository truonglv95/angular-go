package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// elementAttrs groups extracted attributes by binding kind.
type elementAttrs struct {
	attrs []extractedAttr
}

type extractedAttr struct {
	bindingKind    ir.BindingKind
	name           string
	value          output.Expression
	namespace      string
	trustedValueFn output.Expression
}

func (ea *elementAttrs) add(kind ir.BindingKind, name string, expr output.Expression, ns string, trusted output.Expression) {
	ea.attrs = append(ea.attrs, extractedAttr{
		bindingKind:    kind,
		name:           name,
		value:          expr,
		namespace:      ns,
		trustedValueFn: trusted,
	})
}

func (ea *elementAttrs) hasEntries() bool {
	return len(ea.attrs) > 0
}

// CollectElementConsts converts extracted attributes into const array expressions.
func CollectElementConsts(job compilation.CompilationJob) {
	allElementAttributes := map[ir.XrefId]*elementAttrs{}

	// Collect all extracted attributes.
	for _, unit := range job.GetUnits() {
		var newCreateOps []ir.Op
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindExtractedAttribute {
				newCreateOps = append(newCreateOps, op)
				continue
			}
			extAttr, ok := op.(*ir.ExtractedAttributeOp)
			if !ok {
				newCreateOps = append(newCreateOps, op)
				continue
			}
			ea := allElementAttributes[extAttr.Target]
			if ea == nil {
				ea = &elementAttrs{}
				allElementAttributes[extAttr.Target] = ea
			}
			bk := ir.BindingKindAttribute
			if bkv, ok2 := extAttr.BindingKind.(ir.BindingKind); ok2 {
				bk = bkv
			}
			ns := ""
			if extAttr.Namespace != nil {
				ns = *extAttr.Namespace
			}
			ea.add(bk, extAttr.Name, extAttr.Expression, ns, extAttr.TrustedValueFn)
		}
		unit.GetCreate().Ops = newCreateOps
	}

	// Serialize into const array depending on job type.
	switch j := job.(type) {
	case *compilation.ComponentCompilationJob:
		for _, unit := range j.GetUnits() {
			for _, op := range unit.GetCreate().Elements() {
				if !ir.IsElementOrContainerOp(op) && op.Kind() != ir.OpKindProjection {
					continue
				}
				xg, ok := op.(interface{ GetXref() ir.XrefId })
				if !ok {
					continue
				}
				xref := xg.GetXref()
				ea, exists := allElementAttributes[xref]
				if exists && ea.hasEntries() {
					arrExpr := buildAttrArrayExpr(ea)
					constIdx := j.AddConst(arrExpr, nil)
					if setter, ok2 := op.(interface{ SetAttributes(any) }); ok2 {
						setter.SetAttributes(constIdx)
					}
				}
				if repeater, ok2 := op.(*ir.RepeaterCreateOp); ok2 && repeater.EmptyView != 0 {
					if emptyAttrs, exists := allElementAttributes[repeater.EmptyView]; exists && emptyAttrs.hasEntries() {
						repeater.EmptyAttributes = j.AddConst(buildAttrArrayExpr(emptyAttrs), nil)
					}
				}
			}
		}
	case *compilation.HostBindingCompilationJob:
		for xref, ea := range allElementAttributes {
			root, ok := j.GetRoot().(*compilation.HostBindingCompilationUnit)
			if !ok {
				continue
			}
			if xref != root.GetXref() {
				panic("An attribute would be const collected into the host binding's template function, but is not associated with the root xref.")
			}
			if ea.hasEntries() {
				root.Attributes = buildAttrArrayExpr(ea)
			}
		}
	}
}

// buildAttrArrayExpr serializes element attributes into an output.LiteralArrayExpr.
func buildAttrArrayExpr(ea *elementAttrs) *output.LiteralArrayExpr {
	var entries []output.Expression

	var attributes []extractedAttr
	var classes []extractedAttr
	var styles []extractedAttr
	var bindings []extractedAttr
	var templates []extractedAttr
	var i18ns []extractedAttr

	for _, a := range ea.attrs {
		switch a.bindingKind {
		case ir.BindingKindAttribute:
			attributes = append(attributes, a)
		case ir.BindingKindClassName:
			classes = append(classes, a)
		case ir.BindingKindStyleProperty:
			styles = append(styles, a)
		case ir.BindingKindProperty, ir.BindingKindTwoWayProperty:
			bindings = append(bindings, a)
		case ir.BindingKindTemplate:
			templates = append(templates, a)
		case ir.BindingKindI18n:
			i18ns = append(i18ns, a)
		}
	}

	for _, a := range attributes {
		entries = append(entries, output.NewLiteralExpr(a.name, nil, nil, nil))
		if a.trustedValueFn != nil {
			entries = append(entries, a.trustedValueFn)
		} else if a.value != nil {
			entries = append(entries, a.value)
		}
	}

	if len(classes) > 0 {
		entries = append(entries, output.NewLiteralExpr(1, nil, nil, nil))
		for _, a := range classes {
			entries = append(entries, output.NewLiteralExpr(a.name, nil, nil, nil))
		}
	}

	if len(styles) > 0 {
		entries = append(entries, output.NewLiteralExpr(2, nil, nil, nil))
		for _, a := range styles {
			entries = append(entries, output.NewLiteralExpr(a.name, nil, nil, nil))
			if a.trustedValueFn != nil {
				entries = append(entries, a.trustedValueFn)
			} else if a.value != nil {
				entries = append(entries, a.value)
			} else {
				entries = append(entries, output.NULL_EXPR)
			}
		}
	}

	if len(bindings) > 0 {
		entries = append(entries, output.NewLiteralExpr(3, nil, nil, nil))
		for _, a := range bindings {
			entries = append(entries, output.NewLiteralExpr(a.name, nil, nil, nil))
		}
	}

	if len(templates) > 0 {
		entries = append(entries, output.NewLiteralExpr(4, nil, nil, nil))
		for _, a := range templates {
			entries = append(entries, output.NewLiteralExpr(a.name, nil, nil, nil))
			if a.trustedValueFn != nil {
				entries = append(entries, a.trustedValueFn)
			} else if a.value != nil {
				if lit, ok := a.value.(*output.LiteralExpr); ok && lit.Value == "" {
					continue
				}
				entries = append(entries, a.value)
			}
		}
	}

	if len(i18ns) > 0 {
		entries = append(entries, output.NewLiteralExpr(6, nil, nil, nil))
		for _, a := range i18ns {
			entries = append(entries, output.NewLiteralExpr(a.name, nil, nil, nil))
		}
	}

	return output.NewLiteralArrayExpr(entries, nil, nil, nil)
}
