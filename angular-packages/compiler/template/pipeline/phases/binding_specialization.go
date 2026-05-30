package phases

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/tags"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// SpecializeBindings specializes other bindings into attributes, animation values, properties, DOM attributes, or two-way bindings.
func SpecializeBindings(job compilation.CompilationJob) {
	elements := make(map[ir.XrefId]ir.Op)
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if ir.IsElementOrContainerOp(op) {
				if xrefTrait, ok := op.(ir.XrefTrait); ok {
					elements[xrefTrait.Xref()] = op
				}
			}
		}
	}

	for _, unit := range job.GetUnits() {
		var newOps []ir.Op
		for _, op := range unit.GetUpdate().Elements() {
			bindingOp, ok := op.(*ir.BindingOp)
			if !ok {
				newOps = append(newOps, op)
				continue
			}

			bindingKind, ok := bindingOp.BindingKind.(ir.BindingKind)
			if !ok {
				newOps = append(newOps, op)
				continue
			}

			keep := true
			var finalOp ir.Op = op

			switch bindingKind {
			case ir.BindingKindAttribute:
				if bindingOp.Name == "ngNonBindable" {
					keep = false
					if target, ok := elements[bindingOp.Target]; ok {
						setNonBindable(target, true)
					}
				} else if strings.HasPrefix(bindingOp.Name, "animate.") {
					animKind := ir.AnimationKindENTER
					if bindingOp.Name != "animate.enter" {
						animKind = ir.AnimationKindLEAVE
					}
					finalOp = &ir.AnimationBindingOp{
						Name:                 bindingOp.Name,
						Target:               bindingOp.Target,
						AnimationKind:        animKind,
						Expression:           bindingOp.Expression,
						SecurityContext:      bindingOp.SecurityContext,
						SourceSpan:           bindingOp.SourceSpan,
						AnimationBindingKind: ir.AnimationBindingKindSTRING,
					}
				} else {
					ns, name, _ := tags.SplitNsName(bindingOp.Name, false)
					var nsStr *string
					if ns != nil {
						nsStr = ns
					}
					finalOp = &ir.AttributeOp{
						Target:                        bindingOp.Target,
						Namespace:                     nsStr,
						Name:                          name,
						Expression:                    bindingOp.Expression,
						SecurityContext:               bindingOp.SecurityContext,
						IsTextAttribute:               bindingOp.IsTextAttribute,
						IsStructuralTemplateAttribute: bindingOp.IsStructuralTemplateAttribute,
						TemplateKind:                  bindingOp.TemplateKind,
						I18nMessage:                   bindingOp.I18nMessage,
						SourceSpan:                    bindingOp.SourceSpan,
					}
				}

			case ir.BindingKindAnimation:
				animKind := ir.AnimationKindENTER
				if bindingOp.Name != "animate.enter" {
					animKind = ir.AnimationKindLEAVE
				}
				finalOp = &ir.AnimationBindingOp{
					Name:                 bindingOp.Name,
					Target:               bindingOp.Target,
					AnimationKind:        animKind,
					Expression:           bindingOp.Expression,
					SecurityContext:      bindingOp.SecurityContext,
					SourceSpan:           bindingOp.SourceSpan,
					AnimationBindingKind: ir.AnimationBindingKindVALUE,
				}

			case ir.BindingKindProperty, ir.BindingKindLegacyAnimation:
				if job.GetMode() == compilation.TemplateCompilationMode_DomOnly && isAriaAttribute(bindingOp.Name) {
					finalOp = &ir.AttributeOp{
						Target:                        bindingOp.Target,
						Namespace:                     nil,
						Name:                          bindingOp.Name,
						Expression:                    bindingOp.Expression,
						SecurityContext:               bindingOp.SecurityContext,
						IsTextAttribute:               false,
						IsStructuralTemplateAttribute: bindingOp.IsStructuralTemplateAttribute,
						TemplateKind:                  bindingOp.TemplateKind,
						I18nMessage:                   bindingOp.I18nMessage,
						SourceSpan:                    bindingOp.SourceSpan,
					}
				} else if job.GetKind() == compilation.CompilationJobKind_Host {
					var i18nContext *ir.XrefId
					if bindingOp.I18nContext != 0 {
						i18nContext = &bindingOp.I18nContext
					}
					finalOp = ir.CreateDomPropertyOp(
						bindingOp.Name,
						bindingOp.Expression,
						bindingKind,
						i18nContext,
						bindingOp.SecurityContext,
						bindingOp.SourceSpan,
					)
				} else {
					finalOp = &ir.PropertyOp{
						Target:                        bindingOp.Target,
						Name:                          bindingOp.Name,
						Expression:                    bindingOp.Expression,
						BindingKind:                   bindingKind,
						SecurityContext:               bindingOp.SecurityContext,
						IsStructuralTemplateAttribute: bindingOp.IsStructuralTemplateAttribute,
						TemplateKind:                  bindingOp.TemplateKind,
						I18nContext:                   bindingOp.I18nContext,
						I18nMessage:                   bindingOp.I18nMessage,
						SourceSpan:                    bindingOp.SourceSpan,
					}
				}

			case ir.BindingKindTwoWayProperty:
				finalOp = &ir.TwoWayPropertyOp{
					Target:                        bindingOp.Target,
					Name:                          bindingOp.Name,
					Expression:                    bindingOp.Expression,
					SecurityContext:               bindingOp.SecurityContext,
					IsStructuralTemplateAttribute: bindingOp.IsStructuralTemplateAttribute,
					TemplateKind:                  bindingOp.TemplateKind,
					I18nContext:                   bindingOp.I18nContext,
					I18nMessage:                   bindingOp.I18nMessage,
					SourceSpan:                    bindingOp.SourceSpan,
				}

			case ir.BindingKindI18n, ir.BindingKindClassName, ir.BindingKindStyleProperty:
				panic(fmt.Sprintf("Unhandled binding of kind %v", bindingKind))
			}

			if keep {
				newOps = append(newOps, finalOp)
			}
		}
		unit.GetUpdate().Ops = newOps
	}
}
