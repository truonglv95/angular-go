package phases

import (
	"reflect"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

type ConstantPoolExt interface {
	ConstantPool
	GetConstLiteral(literal output.Expression, share bool) output.Expression
}

func GenerateProjectionDefs(job *compilation.ComponentCompilationJob) {
	share := true

	var selectors []*string
	projectionSlotIndex := 0
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Ops {
			if projOp, ok := op.(*ir.ProjectionOp); ok {
				selectors = append(selectors, projOp.Selector)
				idx := projectionSlotIndex
				projOp.ProjectionSlotIndex = &idx
				projectionSlotIndex++
			}
		}
	}

	if len(selectors) > 0 {
		var defExpr output.Expression
		if len(selectors) > 1 || (selectors[0] != nil && *selectors[0] != "*") {
			var def []any
			for _, s := range selectors {
				if s != nil && *s == "*" {
					def = append(def, "*")
				} else {
					if compilation.ParseSelectorToR3Selector != nil {
						def = append(def, compilation.ParseSelectorToR3Selector(s))
					} else {
						def = append(def, []any{[]any{""}})
					}
				}
			}
			if pool, ok := job.GetPool().(ConstantPoolExt); ok {
				defExpr = pool.GetConstLiteral(literalOrArrayLiteral(def), share)
			}
		}

		if pool, ok := job.GetPool().(ConstantPoolExt); ok {
			var selStr []any
			for _, s := range selectors {
				if s == nil {
					selStr = append(selStr, "")
				} else {
					selStr = append(selStr, *s)
				}
			}
			job.ContentSelectors = pool.GetConstLiteral(literalOrArrayLiteral(selStr), share)
		}

		defOp := &ir.ProjectionDefOp{
			Def: defExpr,
		}
		job.Root.GetCreate().Prepend([]ir.Op{defOp})
	}
}

func literalOrArrayLiteral(val any) output.Expression {
	if val == nil {
		return output.NewLiteralExpr(nil, nil, nil, nil)
	}

	switch v := val.(type) {
	case string:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case int:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case float64:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case bool:
		return output.NewLiteralExpr(v, nil, nil, nil)
	case output.Expression:
		return v
	}

	rt := reflect.TypeOf(val)
	if rt != nil && rt.Kind() == reflect.Slice {
		rv := reflect.ValueOf(val)
		var entries []output.Expression
		for i := 0; i < rv.Len(); i++ {
			entries = append(entries, literalOrArrayLiteral(rv.Index(i).Interface()))
		}
		return output.NewLiteralArrayExpr(entries, nil, nil, nil)
	}

	return output.NewLiteralExpr(val, nil, nil, nil)
}
