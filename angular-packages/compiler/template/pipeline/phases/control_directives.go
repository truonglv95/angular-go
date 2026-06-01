package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// SpecializeControlProperties handles the control directive properties.
func SpecializeControlProperties(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		processView(unit)
	}
}

func processView(view compilation.CompilationUnit) {
	createInsertions := make(map[ir.Op][]ir.Op)
	updateInsertions := make(map[ir.Op][]ir.Op)

	for _, op := range view.GetUpdate().Ops {
		if op.Kind() != ir.OpKindProperty &&
			op.Kind() != ir.OpKindTwoWayProperty &&
			op.Kind() != ir.OpKindAttribute {
			continue
		}

		name, ok := getOpName(op)
		if !ok {
			continue
		}

		if isEligibleControlProperty(name, op.Kind()) {
			target, ok := getTargetId(op)
			if !ok {
				continue
			}

			targetCreateOp := findCreateInstruction(view, target)
			if targetCreateOp == nil {
				continue
			}

			span := getSourceSpan(op)
			controlCreateOp := &ir.ControlCreateOp{
				SourceSpan: span,
			}
			controlOp := &ir.ControlOp{
				Target:     target,
				SourceSpan: span,
			}

			createInsertions[targetCreateOp] = append(createInsertions[targetCreateOp], controlCreateOp)
			updateInsertions[op] = append(updateInsertions[op], controlOp)
		}
	}

	// 1. Rebuild unit.GetCreate().Ops out-of-place
	if len(createInsertions) > 0 {
		var newCreateOps []ir.Op
		for _, createOp := range view.GetCreate().Ops {
			newCreateOps = append(newCreateOps, createOp)
			if list, ok := createInsertions[createOp]; ok {
				newCreateOps = append(newCreateOps, list...)
			}
		}
		view.GetCreate().Ops = newCreateOps
	}

	// 2. Rebuild unit.GetUpdate().Ops out-of-place
	if len(updateInsertions) > 0 {
		var newUpdateOps []ir.Op
		for _, updateOp := range view.GetUpdate().Ops {
			newUpdateOps = append(newUpdateOps, updateOp)
			if list, ok := updateInsertions[updateOp]; ok {
				newUpdateOps = append(newUpdateOps, list...)
			}
		}
		view.GetUpdate().Ops = newUpdateOps
	}
}

func isRelevantCreateOp(kind ir.OpKind) bool {
	switch kind {
	case ir.OpKindContainer, ir.OpKindContainerStart, ir.OpKindContainerEnd,
		ir.OpKindElement, ir.OpKindElementStart, ir.OpKindElementEnd,
		ir.OpKindTemplate:
		return true
	}
	return false
}

func findCreateInstruction(view compilation.CompilationUnit, target ir.XrefId) ir.Op {
	var lastFoundOp ir.Op
	for _, createOp := range view.GetCreate().Ops {
		if !isRelevantCreateOp(createOp.Kind()) {
			continue
		}
		xref, ok := getXrefId(createOp)
		if !ok || xref != target {
			continue
		}
		lastFoundOp = createOp
	}
	return lastFoundOp
}

func getXrefId(op ir.Op) (ir.XrefId, bool) {
	switch o := op.(type) {
	case *ir.ElementStartOp:
		return o.Xref, true
	case *ir.ElementOp:
		return o.Xref, true
	case *ir.TemplateOp:
		return o.Xref, true
	case *ir.ConditionalCreateOp:
		return o.Xref, true
	case *ir.ConditionalBranchCreateOp:
		return o.Xref, true
	case *ir.ContainerStartOp:
		return o.Xref, true
	case *ir.ContainerOp:
		return o.Xref, true
	case *ir.RepeaterCreateOp:
		return o.Xref, true
	case *ir.ElementEndOp:
		return o.Xref, true
	case *ir.ContainerEndOp:
		return o.Xref, true
	}
	return 0, false
}

func getTargetId(op ir.Op) (ir.XrefId, bool) {
	switch o := op.(type) {
	case *ir.PropertyOp:
		return o.Target, true
	case *ir.TwoWayPropertyOp:
		return o.Target, true
	case *ir.AttributeOp:
		return o.Target, true
	}
	return 0, false
}

func getOpName(op ir.Op) (string, bool) {
	switch o := op.(type) {
	case *ir.PropertyOp:
		return o.Name, true
	case *ir.TwoWayPropertyOp:
		return o.Name, true
	case *ir.AttributeOp:
		return o.Name, true
	}
	return "", false
}

func getSourceSpan(op ir.Op) *parse_util.ParseSourceSpan {
	switch o := op.(type) {
	case *ir.PropertyOp:
		return o.SourceSpan
	case *ir.TwoWayPropertyOp:
		return o.SourceSpan
	case *ir.AttributeOp:
		return o.SourceSpan
	}
	return nil
}

func isEligibleControlProperty(name string, kind ir.OpKind) bool {
	switch name {
	case "formField":
		return kind == ir.OpKindProperty
	}
	return false
}
