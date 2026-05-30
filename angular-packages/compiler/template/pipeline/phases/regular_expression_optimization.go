package phases

import (
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

type RegularExpressionConstant struct{}

func (r *RegularExpressionConstant) KeyOf(expr output.Expression) string {
	return "regex:" + expr.(*output.RegularExpressionLiteralExpr).Body
}

func (r *RegularExpressionConstant) ToSharedConstantDeclaration(declName string, keyExpr output.Expression) output.Statement {
	var span output.ParseSourceSpan
	if keyExpr.GetSourceSpan() != nil {
		if s, ok := keyExpr.GetSourceSpan().(output.ParseSourceSpan); ok {
			span = s
		}
	}
	return output.NewDeclareVarStmt(declName, keyExpr, nil, output.StmtModifierFinal, span, nil)
}

// OptimizeRegularExpressions extracts stateless regular expression literals into shared constants.
func OptimizeRegularExpressions(job compilation.CompilationJob) {
	pool, ok := job.GetPool().(ConstantPool)
	if !ok {
		return
	}
	for _, unit := range job.GetUnits() {
		for _, op := range unit.Ops() {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				if regexExpr, ok := expr.(*output.RegularExpressionLiteralExpr); ok {
					if regexExpr.Flags == nil || !strings.Contains(*regexExpr.Flags, "g") {
						return pool.GetSharedConstant(&RegularExpressionConstant{}, regexExpr)
					}
				}
				return expr
			}, ir.VisitorContextFlagNone)
		}
	}
}
