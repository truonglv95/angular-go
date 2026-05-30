package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ExtractAttributes extracts attributes, properties, style and class props into extracted attribute ops.
func ExtractAttributes(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		elements := createOpXrefMap(unit)

		toInsertBefore := make(map[ir.Op][]ir.Op)
		var newUpdateOps []ir.Op
		var hostCreateOps []ir.Op

		// Pass 1: Scan and process update operations
		for _, op := range unit.GetUpdate().Ops {
			switch op.Kind() {
			case ir.OpKindAttribute:
				attrOp := op.(*ir.AttributeOp)
				if _, isInterpolation := attrOp.Expression.(*ir.Interpolation); isInterpolation {
					newUpdateOps = append(newUpdateOps, op)
					continue
				}

				if attrOp.IsTextAttribute {
					var bindingKind ir.BindingKind
					if attrOp.IsStructuralTemplateAttribute {
						bindingKind = ir.BindingKindTemplate
					} else {
						bindingKind = ir.BindingKindAttribute
					}

					extractedAttrOp := &ir.ExtractedAttributeOp{
						Target:          attrOp.Target,
						BindingKind:     bindingKind,
						Namespace:       attrOp.Namespace,
						Name:            attrOp.Name,
						Expression:      attrOp.Expression,
						I18nContext:     attrOp.I18nContext,
						I18nMessage:     attrOp.I18nMessage,
						SecurityContext: attrOp.SecurityContext,
					}

					if job.GetKind() == compilation.CompilationJobKind_Host {
						hostCreateOps = append(hostCreateOps, extractedAttrOp)
					} else {
						targetEl := lookupElement(elements, attrOp.Target)
						if targetEl != nil {
							toInsertBefore[targetEl] = append(toInsertBefore[targetEl], extractedAttrOp)
						}
					}
					// Filtered out (removed) from update ops
					continue
				} else {
					newUpdateOps = append(newUpdateOps, op)
				}

			case ir.OpKindProperty:
				propOp := op.(*ir.PropertyOp)
				bKind, _ := propOp.BindingKind.(ir.BindingKind)
				if bKind != ir.BindingKindLegacyAnimation && bKind != ir.BindingKindAnimation {
					var bindingKind ir.BindingKind
					if propOp.I18nMessage != nil && propOp.TemplateKind == nil {
						bindingKind = ir.BindingKindI18n
					} else if propOp.IsStructuralTemplateAttribute {
						bindingKind = ir.BindingKindTemplate
					} else {
						bindingKind = ir.BindingKindProperty
					}

					extractedAttrOp := &ir.ExtractedAttributeOp{
						Target:          propOp.Target,
						BindingKind:     bindingKind,
						Namespace:       nil,
						Name:            propOp.Name,
						Expression:      nil,
						I18nContext:     0,
						I18nMessage:     nil,
						SecurityContext: propOp.SecurityContext,
					}

					targetEl := lookupElement(elements, propOp.Target)
					if targetEl != nil {
						toInsertBefore[targetEl] = append(toInsertBefore[targetEl], extractedAttrOp)
					}
				}
				newUpdateOps = append(newUpdateOps, op)

			case ir.OpKindTwoWayProperty:
				twoWayOp := op.(*ir.TwoWayPropertyOp)
				extractedAttrOp := &ir.ExtractedAttributeOp{
					Target:          twoWayOp.Target,
					BindingKind:     ir.BindingKindTwoWayProperty,
					Namespace:       nil,
					Name:            twoWayOp.Name,
					Expression:      nil,
					I18nContext:     0,
					I18nMessage:     nil,
					SecurityContext: twoWayOp.SecurityContext,
				}

				targetEl := lookupElement(elements, twoWayOp.Target)
				if targetEl != nil {
					toInsertBefore[targetEl] = append(toInsertBefore[targetEl], extractedAttrOp)
				}
				newUpdateOps = append(newUpdateOps, op)

			case ir.OpKindStyleProp, ir.OpKindClassProp:
				var name string
				var target ir.XrefId
				var expr ir.Expression
				isStyleOrClassProp := false

				if styleProp, ok := op.(*ir.StylePropOp); ok {
					name = styleProp.Name
					target = styleProp.Target
					expr, _ = styleProp.Expression.(ir.Expression)
					isStyleOrClassProp = true
				} else if classProp, ok := op.(*ir.ClassPropOp); ok {
					name = classProp.Name
					target = classProp.Target
					expr, _ = classProp.Expression.(ir.Expression)
					isStyleOrClassProp = true
				}

				if isStyleOrClassProp && expr != nil {
					if _, isEmpty := expr.(*ir.EmptyExpr); isEmpty {
						extractedAttrOp := &ir.ExtractedAttributeOp{
							Target:          target,
							BindingKind:     ir.BindingKindProperty,
							Namespace:       nil,
							Name:            name,
							Expression:      nil,
							I18nContext:     0,
							I18nMessage:     nil,
							SecurityContext: core.SecurityContextStyle,
						}

						targetEl := lookupElement(elements, target)
						if targetEl != nil {
							toInsertBefore[targetEl] = append(toInsertBefore[targetEl], extractedAttrOp)
						}
					}
				}
				newUpdateOps = append(newUpdateOps, op)

			default:
				newUpdateOps = append(newUpdateOps, op)
			}
		}

		// Pass 2: Scan and process create operations (for listener extraction)
		for _, op := range unit.GetCreate().Ops {
			if op.Kind() == ir.OpKindListener {
				listenerOp := op.(*ir.ListenerOp)
				if !listenerOp.IsLegacyAnimationListener {
					extractedAttrOp := &ir.ExtractedAttributeOp{
						Target:          listenerOp.Target,
						BindingKind:     ir.BindingKindProperty,
						Namespace:       nil,
						Name:            listenerOp.Name,
						Expression:      nil,
						I18nContext:     0,
						I18nMessage:     nil,
						SecurityContext: core.SecurityContextNone,
					}

					if job.GetKind() != compilation.CompilationJobKind_Host {
						targetEl := lookupElement(elements, listenerOp.Target)
						if targetEl != nil {
							toInsertBefore[targetEl] = append(toInsertBefore[targetEl], extractedAttrOp)
						}
					}
				}
			} else if op.Kind() == ir.OpKindTwoWayListener {
				twoWayListenerOp := op.(*ir.TwoWayListenerOp)
				extractedAttrOp := &ir.ExtractedAttributeOp{
					Target:          twoWayListenerOp.Target,
					BindingKind:     ir.BindingKindProperty,
					Namespace:       nil,
					Name:            twoWayListenerOp.Name,
					Expression:      nil,
					I18nContext:     0,
					I18nMessage:     nil,
					SecurityContext: core.SecurityContextNone,
				}

				if job.GetKind() != compilation.CompilationJobKind_Host {
					targetEl := lookupElement(elements, twoWayListenerOp.Target)
					if targetEl != nil {
						toInsertBefore[targetEl] = append(toInsertBefore[targetEl], extractedAttrOp)
					}
				}
			}
		}

		// Pass 3: Rebuild create ops out-of-place
		var newCreateOps []ir.Op
		for _, createOp := range unit.GetCreate().Ops {
			if list, ok := toInsertBefore[createOp]; ok {
				newCreateOps = append(newCreateOps, list...)
			}
			newCreateOps = append(newCreateOps, createOp)
		}
		if job.GetKind() == compilation.CompilationJobKind_Host {
			newCreateOps = append(newCreateOps, hostCreateOps...)
		}

		unit.GetCreate().Ops = newCreateOps
		unit.GetUpdate().Ops = newUpdateOps
	}
}

func lookupElement(elements map[ir.XrefId]ir.Op, target ir.XrefId) ir.Op {
	el := elements[target]
	if el == nil {
		panic("All attributes should have an element-like target.")
	}
	return el
}
