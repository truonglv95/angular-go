package emit

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func init() {
	render3.EmitTemplateFn = EmitTemplateFn
	render3.EmitHostBindingFunction = EmitHostBindingFunction
}

func EmitTemplateFn(job *compilation.ComponentCompilationJob, pool render3.ConstantPool) output.Expression {
	rootFn := emitView(job.Root)

	var childFns []output.Statement
	for _, unit := range childUnitsInEmitOrder(job) {
		childFnExpr := emitView(unit).(*output.FunctionExpr)
		stmt := output.NewDeclareFunctionStmt(*childFnExpr.Name, childFnExpr.Params, childFnExpr.Statements, nil, output.StmtModifierNone, nil, nil)
		childFns = append(childFns, stmt)
	}

	if poolExt, ok := pool.(interface{ AddStatement(stmt output.Statement) }); ok {
		for _, childFn := range childFns {
			poolExt.AddStatement(childFn)
		}
	}

	return rootFn
}

func EmitHostBindingFunction(job *compilation.HostBindingCompilationJob) *output.FunctionExpr {
	if len(job.Root.GetCreate().Elements()) == 0 && len(job.Root.GetUpdate().Elements()) == 0 {
		return nil
	}
	expr := emitView(job.Root)
	if fn, ok := expr.(*output.FunctionExpr); ok {
		return fn
	}
	panic("emitView returned non-function")
}

func childUnitsInEmitOrder(job *compilation.ComponentCompilationJob) []compilation.CompilationUnit {
	children := map[ir.XrefId][]*compilation.ViewCompilationUnit{}
	for _, unit := range job.Units {
		if unit == job.Root || unit.Parent == nil {
			continue
		}
		parent := *unit.Parent
		children[parent] = append(children[parent], unit)
	}

	var units []compilation.CompilationUnit
	var visit func(unit *compilation.ViewCompilationUnit)
	visit = func(unit *compilation.ViewCompilationUnit) {
		for _, child := range children[unit.Xref] {
			visit(child)
			if child != job.Root {
				units = append(units, child)
			}
		}
	}
	visit(job.Root)
	return units
}

func emitView(view compilation.CompilationUnit) output.Expression {
	fnName := view.GetFnName()
	if fnName == nil {
		panic(fmt.Sprintf("AssertionError: view %v is unnamed", view.GetXref()))
	}

	var createStatements []output.Statement
	for _, op := range view.GetCreate().Elements() {
		if stmtOp, ok := op.(*ir.StatementOp); ok {
			createStatements = append(createStatements, stmtOp.Statement)
		} else if op.Kind() != ir.OpKindStatement {
			panic(fmt.Sprintf("AssertionError: expected all create ops to have been compiled, but got %d (type %T)", op.Kind(), op))
		}
	}
	createStatements = balanceDomElementEnds(createStatements)

	var updateStatements []output.Statement
	for _, op := range view.GetUpdate().Elements() {
		if stmtOp, ok := op.(*ir.StatementOp); ok {
			updateStatements = append(updateStatements, stmtOp.Statement)
		} else if op.Kind() != ir.OpKindStatement {
			panic(fmt.Sprintf("AssertionError: expected all update ops to have been compiled, but got %d (type %T)", op.Kind(), op))
		}
	}

	createCond := maybeGenerateRfBlock(1, createStatements)
	updateCond := maybeGenerateRfBlock(2, updateStatements)

	var stmts []output.Statement
	stmts = append(stmts, createCond...)
	stmts = append(stmts, updateCond...)

	rfParam := output.NewFnParam("rf", output.NUMBER_TYPE)
	ctxParam := output.NewFnParam("ctx", output.DYNAMIC_TYPE)

	nameStr := *fnName
	return output.NewFunctionExpr([]*output.FnParam{rfParam, ctxParam}, stmts, nil, nil, &nameStr, nil)
}

func balanceDomElementEnds(statements []output.Statement) []output.Statement {
	starts := 0
	ends := 0
	for _, stmt := range statements {
		exprStmt, ok := stmt.(*output.ExpressionStatement)
		if !ok {
			continue
		}
		countDomInstructions(exprStmt.Expr, &starts, &ends)
	}
	for starts > ends {
		call := output.NewInvokeFunctionExpr(output.NewReadVarExpr("ɵɵdomElementEnd", nil, nil, nil), nil, nil, nil, false, nil, false)
		statements = append(statements, output.NewExpressionStatement(call, nil, nil))
		ends++
	}
	return statements
}

func countDomInstructions(expr output.Expression, starts *int, ends *int) {
	invoke, ok := expr.(*output.InvokeFunctionExpr)
	if !ok {
		return
	}
	if read, ok := invoke.Fn.(*output.ReadVarExpr); ok {
		countDomInstructionName(read.Name, starts, ends)
		return
	}
	if _, ok := invoke.Fn.(*output.InvokeFunctionExpr); ok {
		name := domInstructionName(invoke.Fn)
		countDomInstructions(invoke.Fn, starts, ends)
		countDomInstructionName(name, starts, ends)
	}
}

func domInstructionName(expr output.Expression) string {
	invoke, ok := expr.(*output.InvokeFunctionExpr)
	if !ok {
		return ""
	}
	if read, ok := invoke.Fn.(*output.ReadVarExpr); ok {
		return read.Name
	}
	return domInstructionName(invoke.Fn)
}

func countDomInstructionName(name string, starts *int, ends *int) {
	switch name {
	case "ɵɵdomElementStart":
		(*starts)++
	case "ɵɵdomElementEnd":
		(*ends)++
	}
}

func maybeGenerateRfBlock(flag int, statements []output.Statement) []output.Statement {
	if len(statements) == 0 {
		return nil
	}
	cond := output.NewBinaryOperatorExpr(output.BinaryOperatorBitwiseAnd, output.NewReadVarExpr("rf", nil, nil, nil), output.NewLiteralExpr(flag, nil, nil, nil), nil, nil, nil)
	return []output.Statement{output.NewIfStmt(cond, statements, nil, nil, nil)}
}
