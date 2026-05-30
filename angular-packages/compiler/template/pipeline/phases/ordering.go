package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

type Rule struct {
	Test      func(ir.Op) bool
	Transform func([]ir.Op) []ir.Op
}

func kindTest(kind ir.OpKind) func(ir.Op) bool {
	return func(op ir.Op) bool {
		return op.Kind() == kind
	}
}

func kindWithInterpolationTest(kind ir.OpKind, interpolation bool) func(ir.Op) bool {
	return func(op ir.Op) bool {
		if op.Kind() != kind {
			return false
		}
		var expr output.Expression
		switch o := op.(type) {
		case *ir.AttributeOp:
			expr = o.Expression
		case *ir.PropertyOp:
			expr = o.Expression
		case *ir.DomPropertyOp:
			if e, ok := o.Expression.(output.Expression); ok {
				expr = e
			} else if interp, ok := o.Expression.(*ir.Interpolation); ok {
				expr = interp
			}
		}
		if expr == nil {
			return false
		}
		_, isInterpolation := expr.(*ir.Interpolation)
		return interpolation == isInterpolation
	}
}

func basicListenerKindTest(op ir.Op) bool {
	switch o := op.(type) {
	case *ir.ListenerOp:
		return !(o.HostListener && o.IsLegacyAnimationListener)
	case *ir.TwoWayListenerOp:
		return true
	case *ir.AnimationOp:
		return true
	case *ir.AnimationListenerOp:
		return true
	}
	return false
}

func nonInterpolationPropertyKindTest(op ir.Op) bool {
	if op.Kind() != ir.OpKindProperty && op.Kind() != ir.OpKindTwoWayProperty {
		return false
	}
	var isInterp bool
	switch o := op.(type) {
	case *ir.PropertyOp:
		_, isInterp = o.Expression.(*ir.Interpolation)
	case *ir.TwoWayPropertyOp:
		_, isInterp = o.Expression.(*ir.Interpolation)
	}
	return !isInterp
}

var CREATE_ORDERING = []Rule{
	{
		Test: func(op ir.Op) bool {
			if l, ok := op.(*ir.ListenerOp); ok {
				return l.HostListener && l.IsLegacyAnimationListener
			}
			return false
		},
	},
	{Test: basicListenerKindTest},
}

var UPDATE_ORDERING = []Rule{
	{Test: kindTest(ir.OpKindStyleMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindClassMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindStyleProp)},
	{Test: kindTest(ir.OpKindClassProp)},
	{Test: kindWithInterpolationTest(ir.OpKindAttribute, true)},
	{Test: kindWithInterpolationTest(ir.OpKindProperty, true)},
	{Test: nonInterpolationPropertyKindTest},
	{Test: kindWithInterpolationTest(ir.OpKindAttribute, false)},
	{Test: kindTest(ir.OpKindControl)},
}

var UPDATE_HOST_ORDERING = []Rule{
	{Test: kindWithInterpolationTest(ir.OpKindDomProperty, true)},
	{Test: kindWithInterpolationTest(ir.OpKindDomProperty, false)},
	{Test: kindTest(ir.OpKindAttribute)},
	{Test: kindTest(ir.OpKindStyleMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindClassMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindStyleProp)},
	{Test: kindTest(ir.OpKindClassProp)},
}

var handledOpKinds = map[ir.OpKind]bool{
	ir.OpKindListener:          true,
	ir.OpKindTwoWayListener:    true,
	ir.OpKindAnimationListener: true,
	ir.OpKindStyleMap:          true,
	ir.OpKindClassMap:          true,
	ir.OpKindStyleProp:         true,
	ir.OpKindClassProp:         true,
	ir.OpKindProperty:          true,
	ir.OpKindTwoWayProperty:    true,
	ir.OpKindDomProperty:       true,
	ir.OpKindAttribute:         true,
	ir.OpKindAnimation:         true,
	ir.OpKindControl:           true,
}

func OrderOps(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		orderWithin(unit.GetCreate(), CREATE_ORDERING)

		var ordering []Rule
		if unit.GetJob().GetKind() == compilation.CompilationJobKind_Host {
			ordering = UPDATE_HOST_ORDERING
		} else {
			ordering = UPDATE_ORDERING
		}
		orderWithin(unit.GetUpdate(), ordering)
	}
}

func orderWithin(opList *ir.OpList, ordering []Rule) {
	newOps := make([]ir.Op, 0, len(opList.Ops))
	var opsToOrder []ir.Op
	firstTargetInGroup := ir.XrefId(-1)

	for _, op := range opList.Ops {
		currentTarget := ir.XrefId(-1)
		if trait, ok := op.(ir.DependsOnSlotContextOpTrait); ok {
			currentTarget = trait.GetTarget()
		}

		diffTarget := false
		if firstTargetInGroup != ir.XrefId(-1) && currentTarget != ir.XrefId(-1) && firstTargetInGroup != currentTarget {
			diffTarget = true
		}

		isHandled := handledOpKinds[op.Kind()]
		if !isHandled || diffTarget {
			if len(opsToOrder) > 0 {
				newOps = append(newOps, reorder(opsToOrder, ordering)...)
				opsToOrder = nil
			}
			firstTargetInGroup = ir.XrefId(-1)
		}

		if isHandled {
			opsToOrder = append(opsToOrder, op)
			if currentTarget != ir.XrefId(-1) {
				firstTargetInGroup = currentTarget
			}
		} else {
			newOps = append(newOps, op)
		}
	}

	if len(opsToOrder) > 0 {
		newOps = append(newOps, reorder(opsToOrder, ordering)...)
	}
	opList.Ops = newOps
}

func reorder(ops []ir.Op, ordering []Rule) []ir.Op {
	groups := make([][]ir.Op, len(ordering)+1)
	for _, op := range ops {
		groupIndex := len(ordering)
		for idx, rule := range ordering {
			if rule.Test(op) {
				groupIndex = idx
				break
			}
		}
		groups[groupIndex] = append(groups[groupIndex], op)
	}

	var result []ir.Op
	for idx, rule := range ordering {
		group := groups[idx]
		if rule.Transform != nil {
			result = append(result, rule.Transform(group)...)
		} else {
			result = append(result, group...)
		}
	}
	result = append(result, groups[len(ordering)]...)
	return result
}

func keepLast(ops []ir.Op) []ir.Op {
	if len(ops) == 0 {
		return ops
	}
	return ops[len(ops)-1:]
}
