package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func GenerateConditionalExpressions(job *compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			if op.Kind() != ir.OpKindConditional {
				continue
			}

			condOp, ok := op.(*ir.ConditionalOp)
			if !ok {
				continue
			}

			var conditions []*ir.ConditionalCaseExpr
			if condOp.Conditions != nil {
				if conds, ok := condOp.Conditions.([]*ir.ConditionalCaseExpr); ok {
					conditions = conds
				}
			}

			var test output.Expression

			defaultCaseIdx := -1
			for idx, cond := range conditions {
				if cond.Expr == nil {
					defaultCaseIdx = idx
					break
				}
			}

			if defaultCaseIdx >= 0 {
				defaultCase := conditions[defaultCaseIdx]
				conditions = append(conditions[:defaultCaseIdx], conditions[defaultCaseIdx+1:]...)
				slotLit := &ir.SlotLiteralExpr{SlotHandle: defaultCase.TargetSlot}
				slotLit.Self = slotLit
				test = slotLit
			} else {
				test = output.NewLiteralExpr(-1, nil, nil, nil)
			}

			var tmp *ir.AssignTemporaryExpr
			if condOp.Test != nil {
				tmp = ir.NewAssignTemporaryExpr(job.AllocateXrefId())
				tmp.Expr = condOp.Test
			}

			var caseExpressionTemporaryXref *ir.XrefId

			for i := len(conditions) - 1; i >= 0; i-- {
				conditionalCase := conditions[i]
				if conditionalCase.Expr == nil {
					continue
				}

				if tmp != nil {
					var useTmp output.Expression
					if i == 0 {
						useTmp = tmp
					} else {
						useTmp = ir.NewReadTemporaryExpr(tmp.Xref)
					}
					conditionalCase.Expr = output.NewBinaryOperatorExpr(
						output.BinaryOperatorIdentical,
						useTmp,
						conditionalCase.Expr,
						nil, nil, nil,
					)
				} else if conditionalCase.Alias != nil {
					if caseExpressionTemporaryXref == nil {
						id := job.AllocateXrefId()
						caseExpressionTemporaryXref = &id
					}
					assignTmp := ir.NewAssignTemporaryExpr(*caseExpressionTemporaryXref)
					assignTmp.Expr = conditionalCase.Expr
					conditionalCase.Expr = assignTmp
					condOp.ContextValue = ir.NewReadTemporaryExpr(*caseExpressionTemporaryXref)
				}

				slotLit := &ir.SlotLiteralExpr{SlotHandle: conditionalCase.TargetSlot}
				slotLit.Self = slotLit
				test = output.NewConditionalExpr(
					conditionalCase.Expr,
					slotLit,
					test,
					nil, nil, nil,
				)
			}

			condOp.Processed = test
			condOp.Conditions = []*ir.ConditionalCaseExpr{}
		}
	}
}
