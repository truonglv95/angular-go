package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func GenerateLocalLetReferences(job *compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		opsCopy := make([]ir.Op, len(unit.GetUpdate().Ops))
		copy(opsCopy, unit.GetUpdate().Ops)

		for _, op := range opsCopy {
			if storeLetOp, ok := op.(*ir.StoreLetOp); ok {
				variable := &ir.SemanticVariable{
					Kind:       ir.SemanticVariableKindIdentifier,
					Identifier: storeLetOp.DeclaredName,
					Local:      true,
				}

				newVarOp := &ir.VariableOp{
					Xref:        job.AllocateXrefId(),
					Variable:    variable,
					Initializer: ir.NewStoreLetExpr(storeLetOp.Target, storeLetOp.Value, storeLetOp.SourceSpan),
					Flags:       ir.VariableFlagsNone,
				}

				OpListReplace(unit.GetUpdate(), storeLetOp, newVarOp)
			}
		}
	}
}
