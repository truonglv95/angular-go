package phases

import (
	"errors"
	"strconv"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func GenerateArrowFunctions(job compilation.CompilationJob) {
	nextFnId := 0
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Ops {
			if repeater, ok := op.(*ir.RepeaterCreateOp); ok && repeater.TrackByOps != nil {
				if cj, ok := job.(*compilation.ComponentCompilationJob); ok {
					params := []*output.FnParam{
						{Name: "$index", Type: nil},
						{Name: "$item", Type: nil},
					}
					trackByOps, _ := repeater.TrackByOps.(*ir.OpList)
					var stmts []output.Statement
					for _, trackOp := range trackByOps.Ops {
						if stmtOp, ok := trackOp.(*ir.StatementOp); ok {
							stmts = append(stmts, stmtOp.Statement)
						}
					}
					
					var body any = stmts
					if len(stmts) == 1 {
						if retStmt, ok := stmts[0].(*output.ReturnStatement); ok {
							body = retStmt.Value
						}
					}
					
					trackByFn := output.NewArrowFunctionExpr(params, body, nil, nil, nil)
					name := "_forTrack" + strconv.Itoa(nextFnId)
					nextFnId++
					
					if pool, ok := cj.Pool.(interface {
						GetSharedFunctionReference(fn output.Expression, name string, unique bool) output.Expression
					}); ok {
						repeater.TrackByFn = pool.GetSharedFunctionReference(trackByFn, name, false)
					}
				}
			}

			if op.Kind() != ir.OpKindAnimation &&
				op.Kind() != ir.OpKindAnimationListener &&
				op.Kind() != ir.OpKindListener &&
				op.Kind() != ir.OpKindTwoWayListener {
				addArrowFunctions(unit, op)
			}
		}

		for _, op := range unit.GetUpdate().Ops {
			addArrowFunctions(unit, op)
		}
	}
}

func addArrowFunctions(unit compilation.CompilationUnit, op ir.Op) {
	ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
		arrowFn, ok := expr.(*output.ArrowFunctionExpr)
		if !ok {
			return expr
		}

		if flags&ir.VisitorContextFlagInChildOperation != 0 {
			return expr
		}

		if _, ok := arrowFn.Body.([]output.Statement); ok {
			panic(errors.New("AssertionError: unexpected multi-line arrow function"))
		}

		params := make([]output.FnParam, len(arrowFn.Params))
		for i, p := range arrowFn.Params {
			params[i] = *p
		}

		var body output.Expression
		if b, ok := arrowFn.Body.(output.Expression); ok {
			body = b
		}

		irArrowFn := ir.NewArrowFunctionExpr(params, body, ir.NewOpList())

		switch u := unit.(type) {
		case *compilation.ViewCompilationUnit:
			u.Functions = append(u.Functions, irArrowFn)
		case *compilation.HostBindingCompilationUnit:
			u.Functions = append(u.Functions, irArrowFn)
		}

		return irArrowFn
	}, ir.VisitorContextFlagNone)
}
