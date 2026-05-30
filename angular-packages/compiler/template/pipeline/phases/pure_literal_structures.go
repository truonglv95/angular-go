package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

func GeneratePureLiteralStructures(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetUpdate().Ops {
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				if flags&ir.VisitorContextFlagInChildOperation != 0 {
					return expr
				}

				if arr, ok := expr.(*output.LiteralArrayExpr); ok {
					return transformLiteralArray(arr)
				} else if m, ok := expr.(*output.LiteralMapExpr); ok {
					return transformLiteralMap(m)
				}

				return expr
			}, ir.VisitorContextFlagNone)
		}
	}
}

func transformLiteralArray(expr *output.LiteralArrayExpr) output.Expression {
	var derivedEntries []output.Expression
	var nonConstantArgs []output.Expression
	for _, entry := range expr.Entries {
		if spread, ok := entry.(*output.SpreadElementExpr); ok {
			if spread.Expression.IsConstant() {
				derivedEntries = append(derivedEntries, spread)
			} else {
				idx := len(nonConstantArgs)
				nonConstantArgs = append(nonConstantArgs, spread.Expression)
				param := &ir.PureFunctionParameterExpr{Index: idx}
				param.Self = param
				newSpread := output.NewSpreadElementExpr(param, nil, nil)
				derivedEntries = append(derivedEntries, newSpread)
			}
			continue
		}

		if entry.IsConstant() {
			derivedEntries = append(derivedEntries, entry)
		} else {
			idx := len(nonConstantArgs)
			nonConstantArgs = append(nonConstantArgs, entry)
			param := &ir.PureFunctionParameterExpr{Index: idx}
			param.Self = param
			derivedEntries = append(derivedEntries, param)
		}
	}
	litArr := output.NewLiteralArrayExpr(derivedEntries, nil, nil, nil)
	pfe := &ir.PureFunctionExpr{
		Body: litArr,
		Args: nonConstantArgs,
	}
	pfe.Self = pfe
	return pfe
}

func transformLiteralMap(expr *output.LiteralMapExpr) output.Expression {
	var derivedEntries []output.LiteralMapEntry
	var nonConstantArgs []output.Expression
	for _, entry := range expr.Entries {
		if spread, ok := entry.(*output.LiteralMapSpreadAssignment); ok {
			if spread.Expression.IsConstant() {
				derivedEntries = append(derivedEntries, spread)
			} else {
				idx := len(nonConstantArgs)
				nonConstantArgs = append(nonConstantArgs, spread.Expression)
				param := &ir.PureFunctionParameterExpr{Index: idx}
				param.Self = param
				newSpread := output.NewLiteralMapSpreadAssignment(param)
				derivedEntries = append(derivedEntries, newSpread)
			}
			continue
		}

		if prop, ok := entry.(*output.LiteralMapPropertyAssignment); ok {
			if prop.Value.IsConstant() {
				derivedEntries = append(derivedEntries, prop)
			} else {
				idx := len(nonConstantArgs)
				nonConstantArgs = append(nonConstantArgs, prop.Value)
				param := &ir.PureFunctionParameterExpr{Index: idx}
				param.Self = param
				newProp := output.NewLiteralMapPropertyAssignment(prop.Key, param, prop.Quoted)
				derivedEntries = append(derivedEntries, newProp)
			}
		}
	}
	litMap := output.NewLiteralMapExpr(derivedEntries, nil, nil, nil)
	pfe := &ir.PureFunctionExpr{
		Body: litMap,
		Args: nonConstantArgs,
	}
	pfe.Self = pfe
	return pfe
}
