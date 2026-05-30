package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ConvertAnimations converts animations bindings into separate animation ops.
func ConvertAnimations(job compilation.CompilationJob) {
	elements := make(map[ir.XrefId]ir.Op)
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Ops {
			if ir.IsElementOrContainerOp(op) {
				if xref, ok := getXrefId(op); ok {
					elements[xref] = op
				}
			}
		}
	}

	for _, unit := range job.GetUnits() {
		var newUpdateOps []ir.Op
		var hostAnimations []ir.Op
		createInsertions := make(map[ir.Op][]ir.Op)

		for _, op := range unit.GetUpdate().Ops {
			if op.Kind() == ir.OpKindAnimationBinding {
				animBindOp := op.(*ir.AnimationBindingOp)
				createAnimationOp := getAnimationOp(animBindOp)

				if job.GetKind() == compilation.CompilationJobKind_Host {
					hostAnimations = append(hostAnimations, createAnimationOp)
				} else {
					targetEl := elements[animBindOp.Target]
					if targetEl != nil {
						createInsertions[targetEl] = append(createInsertions[targetEl], createAnimationOp)
					}
				}
				// Filter out the original update op
				continue
			}
			newUpdateOps = append(newUpdateOps, op)
		}

		// Rebuild unit.GetCreate().Ops out-of-place
		var newCreateOps []ir.Op
		for _, createOp := range unit.GetCreate().Ops {
			newCreateOps = append(newCreateOps, createOp)
			if list, ok := createInsertions[createOp]; ok {
				newCreateOps = append(newCreateOps, list...)
			}
		}
		if job.GetKind() == compilation.CompilationJobKind_Host {
			newCreateOps = append(newCreateOps, hostAnimations...)
		}

		unit.GetCreate().Ops = newCreateOps
		unit.GetUpdate().Ops = newUpdateOps
	}
}

func getAnimationOp(op *ir.AnimationBindingOp) ir.Op {
	var animKind ir.AnimationKind
	if op.Name == "animate.enter" {
		animKind = ir.AnimationKindENTER
	} else {
		animKind = ir.AnimationKindLEAVE
	}

	if op.AnimationBindingKind == ir.AnimationBindingKindSTRING {
		return &ir.AnimationStringOp{
			Name:            op.Name,
			Target:          op.Target,
			AnimationKind:   animKind,
			Expression:      op.Expression,
			SecurityContext: op.SecurityContext,
			SourceSpan:      op.SourceSpan,
		}
	} else {
		handlerOps := ir.NewOpList()
		returnStatement := output.NewReturnStatement(op.Expression, nil, nil)
		stmtOp := &ir.StatementOp{
			Statement: returnStatement,
		}
		handlerOps.Push(stmtOp)

		return &ir.AnimationOp{
			Name:            op.Name,
			Target:          op.Target,
			AnimationKind:   animKind,
			HandlerOps:      handlerOps,
			SecurityContext: op.SecurityContext,
			SourceSpan:      op.SourceSpan,
		}
	}
}
