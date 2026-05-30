package expression_parser

import (
	"fmt"
	"strings"
)

// Serialize serializes the given AST into a normalized string format.
func Serialize(expression AST) string {
	visitor := &SerializeExpressionVisitor{}
	res := expression.Visit(visitor, nil)
	if str, ok := res.(string); ok {
		return str
	}
	return ""
}

type SerializeExpressionVisitor struct{}

func (v *SerializeExpressionVisitor) VisitUnary(ast *Unary, context any) any {
	return fmt.Sprintf("%s%v", ast.Operator, ast.Expr.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitBinary(ast *Binary, context any) any {
	return fmt.Sprintf("%v %s %v", ast.Left.Visit(v, context), ast.Operation, ast.Right.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitChain(ast *Chain, context any) any {
	parts := make([]string, len(ast.Expressions))
	for i, expr := range ast.Expressions {
		parts[i] = fmt.Sprintf("%v", expr.Visit(v, context))
	}
	return strings.Join(parts, "; ")
}

func (v *SerializeExpressionVisitor) VisitConditional(ast *Conditional, context any) any {
	return fmt.Sprintf("%v ? %v : %v", ast.Condition.Visit(v, context), ast.TrueExp.Visit(v, context), ast.FalseExp.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitThisReceiver(ast *ThisReceiver, context any) any {
	return "this"
}

func (v *SerializeExpressionVisitor) VisitImplicitReceiver(ast *ImplicitReceiver, context any) any {
	return ""
}

func (v *SerializeExpressionVisitor) VisitInterpolation(ast *Interpolation, context any) any {
	var result strings.Builder
	maxLen := len(ast.Strings)
	if len(ast.Expressions) > maxLen {
		maxLen = len(ast.Expressions)
	}
	for i := 0; i < maxLen; i++ {
		if i < len(ast.Strings) {
			result.WriteString(fmt.Sprintf("%v", ast.Strings[i]))
		}
		if i < len(ast.Expressions) {
			result.WriteString(fmt.Sprintf("%v", ast.Expressions[i].Visit(v, context)))
		}
	}
	return result.String()
}

func (v *SerializeExpressionVisitor) VisitKeyedRead(ast *KeyedRead, context any) any {
	return fmt.Sprintf("%v[%v]", ast.Receiver.Visit(v, context), ast.Key.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitKeyedWrite(ast *KeyedWrite, context any) any {
	return fmt.Sprintf("%v[%v] = %v", ast.Receiver.Visit(v, context), ast.Key.Visit(v, context), ast.Value.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitLiteralArray(ast *LiteralArray, context any) any {
	parts := make([]string, len(ast.Expressions))
	for i, expr := range ast.Expressions {
		parts[i] = fmt.Sprintf("%v", expr.Visit(v, context))
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func (v *SerializeExpressionVisitor) VisitLiteralMap(ast *LiteralMap, context any) any {
	parts := make([]string, 0)
	for i, key := range ast.Keys {
		if i >= len(ast.Values) {
			break
		}
		
		keyStr := ""
		if spreadKey, ok := key.(*LiteralMapSpreadKey); ok {
			_ = spreadKey
			keyStr = "..."
		} else if propKey, ok := key.(*LiteralMapPropertyKey); ok {
			if propKey.Quoted {
				keyStr = fmt.Sprintf("'%s'", propKey.Key)
			} else {
				keyStr = propKey.Key
			}
		}
		
		valStr := fmt.Sprintf("%v", ast.Values[i].Visit(v, context))
		parts = append(parts, fmt.Sprintf("%s: %s", keyStr, valStr))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

func (v *SerializeExpressionVisitor) VisitLiteralPrimitive(ast *LiteralPrimitive, context any) any {
	if ast.Value == nil {
		return "null"
	}
	switch val := ast.Value.(type) {
	case float64:
		return fmt.Sprintf("%v", val)
	case int:
		return fmt.Sprintf("%v", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case string:
		if val == "undefined" { return "undefined" }
		escaped := strings.ReplaceAll(val, "'", "\\'")
		return fmt.Sprintf("'%s'", escaped)
	default:
		// Try to handle undefined if possible, but go doesn't have it natively.
		return fmt.Sprintf("%v", ast.Value)
	}
}

func (v *SerializeExpressionVisitor) VisitPipe(ast *BindingPipe, context any) any {
	return fmt.Sprintf("%v | %s", ast.Exp.Visit(v, context), ast.Name)
}

func (v *SerializeExpressionVisitor) VisitPrefixNot(ast *PrefixNot, context any) any {
	return fmt.Sprintf("!%v", ast.Expression.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitNonNullAssert(ast *NonNullAssert, context any) any {
	return fmt.Sprintf("%v!", ast.Expression.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitPropertyRead(ast *PropertyRead, context any) any {
	_, isImplicit := ast.Receiver.(*ImplicitReceiver)
	_, isThis := ast.Receiver.(*ThisReceiver)
	if isImplicit || isThis {
		return ast.Name
	}
	return fmt.Sprintf("%v.%s", ast.Receiver.Visit(v, context), ast.Name)
}

func (v *SerializeExpressionVisitor) VisitPropertyWrite(ast *PropertyWrite, context any) any {
	_, isImplicit := ast.Receiver.(*ImplicitReceiver)
	_, isThis := ast.Receiver.(*ThisReceiver)
	
	receiverStr := ""
	if isImplicit || isThis {
		receiverStr = ast.Name
	} else {
		receiverStr = fmt.Sprintf("%v.%s", ast.Receiver.Visit(v, context), ast.Name)
	}
	return fmt.Sprintf("%s = %v", receiverStr, ast.Value.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitSafePropertyRead(ast *SafePropertyRead, context any) any {
	return fmt.Sprintf("%v?.%s", ast.Receiver.Visit(v, context), ast.Name)
}

func (v *SerializeExpressionVisitor) VisitSafeKeyedRead(ast *SafeKeyedRead, context any) any {
	return fmt.Sprintf("%v?.[%v]", ast.Receiver.Visit(v, context), ast.Key.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitCall(ast *Call, context any) any {
	args := make([]string, len(ast.Args))
	for i, arg := range ast.Args {
		if _, isEmpty := arg.(*EmptyExpr); isEmpty {
			args[i] = ""
		} else {
			args[i] = fmt.Sprintf("%v", arg.Visit(v, context))
		}
	}
	return fmt.Sprintf("%v(%s)", ast.Receiver.Visit(v, context), strings.Join(args, ", "))
}

func (v *SerializeExpressionVisitor) VisitMethodCall(ast *MethodCall, context any) any {
	args := make([]string, len(ast.Args))
	for i, arg := range ast.Args {
		args[i] = fmt.Sprintf("%v", arg.Visit(v, context))
	}
	
	_, isImplicit := ast.Receiver.(*ImplicitReceiver)
	if isImplicit {
		return fmt.Sprintf("%s(%s)", ast.Name, strings.Join(args, ", "))
	}
	return fmt.Sprintf("%v.%s(%s)", ast.Receiver.Visit(v, context), ast.Name, strings.Join(args, ", "))
}

func (v *SerializeExpressionVisitor) VisitFunctionCall(ast *FunctionCall, context any) any {
	args := make([]string, len(ast.Args))
	for i, arg := range ast.Args {
		args[i] = fmt.Sprintf("%v", arg.Visit(v, context))
	}
	return fmt.Sprintf("%v(%s)", ast.Target.Visit(v, context), strings.Join(args, ", "))
}

func (v *SerializeExpressionVisitor) VisitSafeCall(ast *SafeCall, context any) any {
	args := make([]string, len(ast.Args))
	for i, arg := range ast.Args {
		if _, isEmpty := arg.(*EmptyExpr); isEmpty {
			args[i] = ""
		} else {
			args[i] = fmt.Sprintf("%v", arg.Visit(v, context))
		}
	}
	return fmt.Sprintf("%v?.(%s)", ast.Receiver.Visit(v, context), strings.Join(args, ", "))
}

func (v *SerializeExpressionVisitor) VisitSafeMethodCall(ast *SafeMethodCall, context any) any {
	args := make([]string, len(ast.Args))
	for i, arg := range ast.Args {
		args[i] = fmt.Sprintf("%v", arg.Visit(v, context))
	}
	return fmt.Sprintf("%v?.%s(%s)", ast.Receiver.Visit(v, context), ast.Name, strings.Join(args, ", "))
}

func (v *SerializeExpressionVisitor) VisitTypeofExpression(ast *TypeofExpression, context any) any {
	return fmt.Sprintf("typeof %v", ast.Expr.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitVoidExpression(ast *VoidExpression, context any) any {
	return fmt.Sprintf("void %v", ast.Expr.Visit(v, context))
}

func (v *SerializeExpressionVisitor) VisitQuote(ast *Quote, context any) any {
	return fmt.Sprintf("%s:%s", ast.Prefix, ast.UninterpretedExpression)
}

func (v *SerializeExpressionVisitor) VisitParenthesizedExpression(ast *ParenthesizedExpression, context any) any {
	// Paren doesn't have expression field in Go yet? Wait! In ast_helpers.go it is empty struct.
	// We'll just return "" if no expression. But I'll modify ast_helpers.go to give it an expression.
	return ""
}
