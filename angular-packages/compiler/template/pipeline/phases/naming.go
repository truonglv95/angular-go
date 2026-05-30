package phases

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// NameFunctionsAndVariables generates names for functions and variables across all views.
// This includes propagating those names into any ReadVariableExprs of those variables.
func NameFunctionsAndVariables(job compilation.CompilationJob) {
	addNamesToView(job.GetRoot(), job.GetComponentName(), &nameState{index: 0})
}

type nameState struct {
	index int
}

// unitFnName safely retrieves the unit's fn name or empty string.
func unitFnName(unit compilation.CompilationUnit) string {
	if unit.GetFnName() != nil {
		return *unit.GetFnName()
	}
	return ""
}

// lookupView returns the ViewCompilationUnit for the given XrefId using the job's Views map.
func lookupView(job compilation.CompilationJob, xref ir.XrefId) compilation.CompilationUnit {
	if cj, ok := job.(*compilation.ComponentCompilationJob); ok {
		if v, exists := cj.Views[xref]; exists {
			return v
		}
	}
	return nil
}

func addNamesToView(unit compilation.CompilationUnit, baseName string, state *nameState) {
	if unit.GetFnName() == nil {
		var name string
		if pool, ok := unit.GetJob().GetPool().(ConstantPool); ok {
			name = pool.UniqueName(sanitizeIdentifier(baseName + "_" + unit.GetJob().GetFnSuffix()))
		} else {
			name = sanitizeIdentifier(baseName + "_" + unit.GetJob().GetFnSuffix())
		}
		unit.SetFnName(&name)
	}

	// Keep track of the names we assign to variables in the view.
	varNames := map[ir.XrefId]string{}

	for _, op := range unit.Ops() {
		switch op.Kind() {
		case ir.OpKindProperty, ir.OpKindDomProperty:
			if prop, ok := op.(interface {
				GetBindingKind() ir.BindingKind
				SetName(string)
				GetName() string
			}); ok {
				if prop.GetBindingKind() == ir.BindingKindLegacyAnimation {
					prop.SetName("@" + prop.GetName())
				}
			}
		case ir.OpKindAnimation:
			if anim, ok := op.(interface {
				GetHandlerFnName() *string
				SetHandlerFnName(string)
				GetName() string
			}); ok {
				if anim.GetHandlerFnName() == nil {
					animKind := strings.ReplaceAll(anim.GetName(), ".", "")
					fnName := sanitizeIdentifier(unitFnName(unit) + "_" + animKind + "_cb")
					anim.SetHandlerFnName(fnName)
				}
			}
		case ir.OpKindAnimationListener:
			if listener, ok := op.(interface {
				GetHandlerFnName() *string
				SetHandlerFnName(string)
				GetHostListener() bool
				GetName() string
				GetTag() *string
				GetTargetSlot() ir.SlotHandle
			}); ok {
				if listener.GetHandlerFnName() != nil {
					break
				}
				if !listener.GetHostListener() && listener.GetTargetSlot().Slot == nil {
					panic("Expected a slot to be assigned")
				}
				animKind := strings.ReplaceAll(listener.GetName(), ".", "")
				var fnName string
				if listener.GetHostListener() {
					fnName = baseName + "_" + animKind + "_HostBindingHandler"
				} else {
					tag := ""
					if listener.GetTag() != nil {
						tag = strings.ReplaceAll(*listener.GetTag(), "-", "_")
					}
					slot := *listener.GetTargetSlot().Slot
					fnName = fmt.Sprintf("%s_%s_%s_%d_listener", unitFnName(unit), tag, animKind, slot)
				}
				fnName = sanitizeIdentifier(fnName)
				listener.SetHandlerFnName(fnName)
			}
		case ir.OpKindListener:
			if listener, ok := op.(interface {
				GetHandlerFnName() *string
				SetHandlerFnName(string)
				GetHostListener() bool
				GetName() string
				SetName(string)
				GetIsLegacyAnimationListener() bool
				GetLegacyAnimationPhase() string
				GetTag() *string
				GetTargetSlot() ir.SlotHandle
			}); ok {
				if listener.GetHandlerFnName() != nil {
					break
				}
				if !listener.GetHostListener() && listener.GetTargetSlot().Slot == nil {
					panic("Expected a slot to be assigned")
				}
				animation := ""
				if listener.GetIsLegacyAnimationListener() {
					listener.SetName("@" + listener.GetName() + "." + listener.GetLegacyAnimationPhase())
					animation = "animation"
				}
				var fnName string
				if listener.GetHostListener() {
					fnName = baseName + "_" + animation + listener.GetName() + "_HostBindingHandler"
				} else {
					tag := ""
					if listener.GetTag() != nil {
						tag = strings.ReplaceAll(*listener.GetTag(), "-", "_")
					}
					slot := *listener.GetTargetSlot().Slot
					fnName = fmt.Sprintf("%s_%s_%s%s_%d_listener", unitFnName(unit), tag, animation, listener.GetName(), slot)
				}
				fnName = sanitizeIdentifier(fnName)
				listener.SetHandlerFnName(fnName)
			}
		case ir.OpKindTwoWayListener:
			if listener, ok := op.(interface {
				GetHandlerFnName() *string
				SetHandlerFnName(string)
				GetTag() *string
				GetName() string
				GetTargetSlot() ir.SlotHandle
			}); ok {
				if listener.GetHandlerFnName() != nil {
					break
				}
				if listener.GetTargetSlot().Slot == nil {
					panic("Expected a slot to be assigned")
				}
				tag := ""
				if listener.GetTag() != nil {
					tag = strings.ReplaceAll(*listener.GetTag(), "-", "_")
				}
				slot := *listener.GetTargetSlot().Slot
				fnName := sanitizeIdentifier(fmt.Sprintf("%s_%s_%s_%d_listener", unitFnName(unit), tag, listener.GetName(), slot))
				listener.SetHandlerFnName(fnName)
			}
		case ir.OpKindVariable:
			if v, ok := op.(*ir.VariableOp); ok {
				varNames[v.Xref] = getVariableName(v.Variable, state)
			}
		case ir.OpKindRepeaterCreate:
			if vcu, ok := unit.(*compilation.ViewCompilationUnit); ok {
				if repeater, ok := op.(interface {
					GetHandle() ir.SlotHandle
					GetEmptyView() ir.XrefId
					GetXref() ir.XrefId
					GetFunctionNameSuffix() string
				}); ok {
					if repeater.GetHandle().Slot == nil {
						panic("Expected slot to be assigned")
					}
					slot := *repeater.GetHandle().Slot
					emptyViewId := repeater.GetEmptyView()
					if emptyViewId != 0 {
						if emptyView, exists := vcu.Job.Views[emptyViewId]; exists {
							addNamesToView(emptyView,
								fmt.Sprintf("%s_%sEmpty_%d", baseName, repeater.GetFunctionNameSuffix(), slot+2),
								state)
						}
					}
					if childView, exists := vcu.Job.Views[repeater.GetXref()]; exists {
						addNamesToView(childView,
							fmt.Sprintf("%s_%s_%d", baseName, repeater.GetFunctionNameSuffix(), slot+1),
							state)
					}
				}
			}
		case ir.OpKindProjection:
			if vcu, ok := unit.(*compilation.ViewCompilationUnit); ok {
				if proj, ok := op.(interface {
					GetHandle() ir.SlotHandle
					GetFallbackView() ir.XrefId
				}); ok {
					if proj.GetHandle().Slot == nil {
						panic("Expected slot to be assigned")
					}
					slot := *proj.GetHandle().Slot
					fallbackViewId := proj.GetFallbackView()
					if fallbackViewId != 0 {
						if fallbackView, exists := vcu.Job.Views[fallbackViewId]; exists {
							addNamesToView(fallbackView, fmt.Sprintf("%s_ProjectionFallback_%d", baseName, slot), state)
						}
					}
				}
			}
		case ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate, ir.OpKindTemplate:
			if vcu, ok := unit.(*compilation.ViewCompilationUnit); ok {
				if tmpl, ok := op.(interface {
					GetXref() ir.XrefId
					GetHandle() ir.SlotHandle
					GetFunctionNameSuffix() string
				}); ok {
					if childView, exists := vcu.Job.Views[tmpl.GetXref()]; exists {
						if tmpl.GetHandle().Slot == nil {
							panic("Expected slot to be assigned")
						}
						slot := *tmpl.GetHandle().Slot
						suffix := ""
						if tmpl.GetFunctionNameSuffix() != "" {
							suffix = "_" + tmpl.GetFunctionNameSuffix()
						}
						addNamesToView(childView, fmt.Sprintf("%s%s_%d", baseName, suffix, slot), state)
					}
				}
			}
		case ir.OpKindStyleProp:
			if sp, ok := op.(interface {
				GetName() string
				SetName(string)
			}); ok {
				sp.SetName(stripImportant(normalizeStylePropName(sp.GetName())))
			}
		case ir.OpKindClassProp:
			if cp, ok := op.(interface {
				GetName() string
				SetName(string)
			}); ok {
				cp.SetName(stripImportant(cp.GetName()))
			}
		}
	}

	// Propagate variable names into ReadVariableExpr expressions.
	for _, op := range unit.Ops() {
		ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
			if rv, ok := expr.(*ir.ReadVariableExpr); ok && rv.Name == nil {
				name, found := varNames[rv.Xref]
				if !found {
					panic(fmt.Sprintf("Variable %v not yet named", rv.Xref))
				}
				rv.Name = &name
			}
		})
	}
}

func getVariableName(variable *ir.SemanticVariable, state *nameState) string {
	if variable.Name == nil {
		var name string
		switch variable.Kind {
		case ir.SemanticVariableKindContext:
			name = fmt.Sprintf("ctx_r%d", state.index)
			state.index++
		case ir.SemanticVariableKindIdentifier:
			compatPrefix := ""
			if variable.Identifier == "ctx" {
				compatPrefix = "i"
			}
			state.index++
			name = fmt.Sprintf("%s_%sr%d", variable.Identifier, compatPrefix, state.index)
		default:
			state.index++
			name = fmt.Sprintf("_r%d", state.index)
		}
		variable.Name = &name
	}
	return *variable.Name
}

func normalizeStylePropName(name string) string {
	if strings.HasPrefix(name, "--") {
		return name
	}
	return hyphenate(name)
}

func stripImportant(name string) string {
	idx := strings.Index(name, "!important")
	if idx > -1 {
		return name[:idx]
	}
	return name
}
