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
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindExtractedAttribute {
				continue
			}
			extAttr, ok := op.(*ir.ExtractedAttributeOp)
			if !ok {
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
			unit.GetCreate().Remove(op)
		}
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
				if !exists || !ea.hasEntries() {
					continue
				}
				arrExpr := buildAttrArrayExpr(ea)
				constIdx := j.AddConst(arrExpr, nil)
				// Store const index into op's Attributes field (type any).
				if setter, ok2 := op.(interface{ SetAttrConstIndex(ir.ConstIndex) }); ok2 {
					setter.SetAttrConstIndex(constIdx)
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
	for _, a := range ea.attrs {
		entries = append(entries, output.NewLiteralExpr(a.name, nil, nil, nil))
		if a.trustedValueFn != nil {
			entries = append(entries, a.trustedValueFn)
		} else if a.value != nil {
			entries = append(entries, a.value)
		} else {
			entries = append(entries, output.NULL_EXPR)
		}
	}
	return output.NewLiteralArrayExpr(entries, nil, nil, nil)
}
