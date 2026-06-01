package phases

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/compilation"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template/pipeline/ir"
)

// ResolveDollarEvent transforms $event variables inside listeners into lexical reads.
func ResolveDollarEvent(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		transformDollarEventInList(unit.GetCreate())
		transformDollarEventInList(unit.GetUpdate())
	}
}

func transformDollarEventInList(ops *ir.OpList) {
	for _, op := range ops.Elements() {
		switch op.Kind() {
		case ir.OpKindListener, ir.OpKindTwoWayListener, ir.OpKindAnimationListener:
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				if lex, ok := expr.(*ir.LexicalReadExpr); ok && lex.Name == "$event" {
					if op.Kind() == ir.OpKindListener || op.Kind() == ir.OpKindAnimationListener {
						if setter, ok2 := op.(interface{ SetConsumesDollarEvent(bool) }); ok2 {
							setter.SetConsumesDollarEvent(true)
						}
					}
					return output.NewReadVarExpr(lex.Name, nil, nil, nil)
				}
				return expr
			}, ir.VisitorContextFlagInChildOperation)
		}
	}
}

// RemoveIllegalLetReferences detects illegal @let forward references and replaces them with undefined.
func RemoveIllegalLetReferences(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for i, op := range unit.GetUpdate().Elements() {
			varOp, ok := op.(*ir.VariableOp)
			if !ok {
				continue
			}
			if varOp.Variable.Kind != ir.SemanticVariableKindIdentifier {
				continue
			}
			if _, ok2 := varOp.Initializer.(*ir.StoreLetExpr); !ok2 {
				continue
			}
			name := varOp.Variable.Identifier
			// Walk backwards through ops before this one.
			for j := i - 1; j >= 0; j-- {
				current := unit.GetUpdate().Elements()[j]
				ir.TransformExpressionsInOp(current, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
					if lex, ok3 := expr.(*ir.LexicalReadExpr); ok3 && lex.Name == name {
						return output.NewLiteralExpr(nil, nil, nil, nil) // undefined
					}
					return expr
				}, ir.VisitorContextFlagNone)
			}
		}
	}
}

// TransformTwoWayBindingSet transforms TwoWayBindingSet expressions into twoWayBindingSet calls.
func TransformTwoWayBindingSet(job compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for _, op := range unit.GetCreate().Elements() {
			if op.Kind() != ir.OpKindTwoWayListener {
				continue
			}
			ir.TransformExpressionsInOp(op, func(expr output.Expression, flags ir.VisitorContextFlag) output.Expression {
				twbs, ok := expr.(*ir.TwoWayBindingSetExpr)
				if !ok {
					return expr
				}
				target := twbs.TargetExpr
				value := twbs.Value

				// If the target is an assignment (BinaryOperatorAssign), extract its LHS as the read expression.
				if binOp, ok := target.(*output.BinaryOperatorExpr); ok && binOp.Operator == output.BinaryOperatorAssign {
					target = binOp.Lhs
				}

				switch t := target.(type) {
				case *output.ReadPropExpr:
					// twoWayBindingSet(target, value) || (target = value)
					setExpr := t.Set(value)
					// Represent the twoWayBindingSet call as a new TwoWayBindingSetExpr stub for now.
					// The reify phase will emit the actual instruction call.
					twoWayCall := ir.NewTwoWayBindingSetExpr(t, value)
					return output.NewBinaryOperatorExpr(output.BinaryOperatorOr, twoWayCall, setExpr, nil, nil, nil)
				case *output.ReadKeyExpr:
					setExpr := t.Set(value)
					twoWayCall := ir.NewTwoWayBindingSetExpr(t, value)
					return output.NewBinaryOperatorExpr(output.BinaryOperatorOr, twoWayCall, setExpr, nil, nil, nil)
				case *ir.ReadVariableExpr:
					return ir.NewTwoWayBindingSetExpr(t, value)
				default:
					panic("Unsupported expression in two-way action binding.")
				}
			}, ir.VisitorContextFlagInChildOperation)
		}
	}
}
