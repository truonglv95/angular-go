package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// CreatePipes generates pipe creation instructions based on the pipe bindings found in the update block.
func CreatePipes(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		processPipeBindingsInView(unit)
	}
}

func processPipeBindingsInView(unit compilation.CompilationUnit) {
	for _, updateOp := range unit.GetUpdate().Ops {
		ir.TransformExpressionsInOp(updateOp, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
			binding, ok := expr.(*ir.PipeBindingExpr)
			if !ok {
				return expr
			}

			if flags&ir.VisitorContextFlagInChildOperation != 0 {
				panic("AssertionError: pipe bindings should not appear in child expressions")
			}

			targetXref, ok := getTargetXref(updateOp)
			if !ok {
				panic("AssertionError: expected slot handle to be assigned for pipe creation")
			}

			addPipeToCreationBlock(unit, targetXref, binding)
			return binding
		}, ir.VisitorContextFlagNone)
	}
}

func addPipeToCreationBlock(
	unit compilation.CompilationUnit,
	afterTargetXref ir.XrefId,
	binding *ir.PipeBindingExpr,
) {
	createOps := unit.GetCreate()
	ops := createOps.Ops

	for i, op := range ops {
		trait, ok := op.(ir.ConsumesSlotOpTrait)
		if !ok {
			continue
		}

		if trait.GetXref() != afterTargetXref {
			continue
		}

		// We've found a tentative insertion point; skip past any subsequent Pipe operations.
		insertIdx := i + 1
		for insertIdx < len(ops) && ops[insertIdx].Kind() == ir.OpKindPipe {
			insertIdx++
		}

		pipeOp := &ir.PipeOp{
			Xref: binding.Target,
			Name: binding.PipeName,
		}

		// Insert pipeOp at insertIdx
		if insertIdx >= len(ops) {
			createOps.Ops = append(createOps.Ops, pipeOp)
		} else {
			createOps.Ops = append(createOps.Ops[:insertIdx], append([]ir.Op{pipeOp}, createOps.Ops[insertIdx:]...)...)
		}
		return
	}

	panic("AssertionError: unable to find insertion point for pipe " + binding.PipeName)
}

func getTargetXref(op ir.Op) (ir.XrefId, bool) {
	if trait, ok := op.(ir.DependsOnSlotContextOpTrait); ok {
		return trait.GetTarget(), true
	}
	switch o := op.(type) {
	case *ir.BindingOp:
		return o.Target, true
	case *ir.AttributeOp:
		return o.Target, true
	case *ir.InterpolateTextOp:
		return o.Target, true
	case *ir.StoreLetOp:
		return o.Target, true
	case *ir.PropertyOp:
		return o.Target, true
	case *ir.TwoWayPropertyOp:
		return o.Target, true
	case *ir.StylePropOp:
		return o.Target, true
	case *ir.ClassPropOp:
		return o.Target, true
	case *ir.StyleMapOp:
		return o.Target, true
	case *ir.ClassMapOp:
		return o.Target, true
	case *ir.ConditionalOp:
		return o.Target, true
	case *ir.RepeaterOp:
		return o.Target, true
	case *ir.DeferWhenOp:
		return o.Target, true
	case *ir.I18nExpressionOp:
		return o.Target, true
	case *ir.ControlOp:
		return o.Target, true
	}
	return 0, false
}
