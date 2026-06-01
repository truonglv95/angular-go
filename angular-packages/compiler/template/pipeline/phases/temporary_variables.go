package phases

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// GenerateTemporaryVariables generates declarations for temporary variables used in IR expressions.
func GenerateTemporaryVariables(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		createStmts := generateTemporaries(unit.GetCreate())
		unit.GetCreate().Prepend(createStmts)
		updateStmts := generateTemporaries(unit.GetUpdate())
		unit.GetUpdate().Prepend(updateStmts)

		for _, fnExpr := range unit.GetFunctions() {
			if arrowFn, ok := fnExpr.(*ir.ArrowFunctionExpr); ok {
				stmts := generateTemporaries(arrowFn.Ops)
				arrowFn.Ops.Prepend(stmts)
			}
		}
	}
}

func generateTemporaries(ops *ir.OpList) []ir.Op {
	opCount := 0
	var generatedStatements []ir.Op

	for _, op := range ops.Elements() {
		// Identify the final time each temp var is read.
		finalReads := map[ir.XrefId]*ir.ReadTemporaryExpr{}
		ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
			if rt, ok := expr.(*ir.ReadTemporaryExpr); ok {
				finalReads[rt.Xref] = rt
			}
		})

		// Name temp vars, reusing names after the final read.
		count := 0
		assigned := map[ir.XrefId]bool{}
		defs := map[ir.XrefId]string{}

		ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
			switch e := expr.(type) {
			case *ir.AssignTemporaryExpr:
				if !assigned[e.Xref] {
					assigned[e.Xref] = true
					defs[e.Xref] = fmt.Sprintf("tmp_%d_%d", opCount, count)
					count++
				}
				if name, ok := defs[e.Xref]; ok {
					e.Name = &name
				}
			case *ir.ReadTemporaryExpr:
				if finalReads[e.Xref] == e {
					count--
				}
				if name, ok := defs[e.Xref]; ok {
					e.Name = &name
				}
			}
		})

		// Generate DeclareVarStmt for each unique temp var name.
		seen := map[string]bool{}
		for _, name := range defs {
			if !seen[name] {
				seen[name] = true
				declStmt := output.NewDeclareVarStmt(name, nil, nil, output.StmtModifierNone, nil, nil)
				stmtOp := &ir.StatementOp{Statement: declStmt}
				generatedStatements = append(generatedStatements, stmtOp)
			}
		}
		if countsForTemporaryName(op) {
			opCount++
		}

		// Handle nested ops (listeners, track).
		switch op.Kind() {
		case ir.OpKindListener, ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindTwoWayListener:
			if lOp, ok := op.(ir.ListenerTrait); ok {
				nested := generateTemporaries(lOp.GetHandlerOps())
				lOp.GetHandlerOps().Prepend(nested)
			}
		case ir.OpKindRepeaterCreate:
			if r, ok := op.(*ir.RepeaterCreateOp); ok {
				if opList, ok2 := r.TrackByOps.(*ir.OpList); ok2 {
					nested := generateTemporaries(opList)
					opList.Prepend(nested)
				}
			}
		}
	}

	return generatedStatements
}

func countsForTemporaryName(op ir.Op) bool {
	switch op.Kind() {
	case ir.OpKindAdvance:
		return false
	default:
		return true
	}
}
