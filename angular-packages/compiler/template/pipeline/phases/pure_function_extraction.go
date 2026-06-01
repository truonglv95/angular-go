package phases

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ExtractPureFunctions extracts pure function bodies into shared constants via the constant pool.
func ExtractPureFunctions(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			ir.VisitExpressionsInOp(op, func(expr ir.Expression) {
				pf, ok := expr.(*ir.PureFunctionExpr)
				if !ok || pf.Body == nil {
					return
				}
				if pool, ok2 := job.GetPool().(interface {
					GetSharedConstant(def compiler.SharedConstantDefinition, expr output.Expression) output.Expression
				}); ok2 {
					constDef := &pureFunctionConstant{numArgs: len(pf.Args)}
					pf.Fn = pool.GetSharedConstant(constDef, pf.Body)
					pf.Body = nil
				}
			})
		}
	}
}

type pureFunctionConstant struct {
	numArgs int
}

func (c *pureFunctionConstant) KeyOf(expr output.Expression) string {
	if pfp, ok := expr.(*ir.PureFunctionParameterExpr); ok {
		return fmt.Sprintf("param(%d)", pfp.Index)
	}
	return fmt.Sprintf("%T:%v", expr, expr)
}

func (c *pureFunctionConstant) ToSharedConstantDeclaration(declName string, keyExpr output.Expression) output.Statement {
	fnParams := make([]*output.FnParam, c.numArgs)
	for i := 0; i < c.numArgs; i++ {
		fnParams[i] = &output.FnParam{Name: fmt.Sprintf("a%d", i), Type: output.DYNAMIC_TYPE}
	}

	returnExpr := ir.TransformExpressionsInExpression(keyExpr, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
		if pfp, ok := expr.(*ir.PureFunctionParameterExpr); ok {
			return output.NewReadVarExpr(fmt.Sprintf("a%d", pfp.Index), nil, nil, nil)
		}
		return expr
	}, ir.VisitorContextFlagNone)

	arrowFn := output.NewArrowFunctionExpr(fnParams, returnExpr, nil, nil, nil)
	return output.NewDeclareVarStmt(declName, arrowFn, nil, output.StmtModifierFinal, nil, nil)
}
