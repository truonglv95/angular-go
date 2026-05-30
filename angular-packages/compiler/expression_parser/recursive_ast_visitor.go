package expression_parser

type RecursiveAstVisitor struct {
	Impl Visitor // To allow overriding
}

func (v *RecursiveAstVisitor) Visit(ast AST, context any) any {
	if ast != nil {
		if v.Impl != nil {
			return ast.Visit(v.Impl, context)
		}
		return ast.Visit(v, context)
	}
	return nil
}

func (v *RecursiveAstVisitor) VisitAll(asts []AST, context any) {
	for _, ast := range asts {
		v.Visit(ast, context)
	}
}

func (v *RecursiveAstVisitor) VisitImplicitReceiver(ast *ImplicitReceiver, context any) any {
	return nil
}

func (v *RecursiveAstVisitor) VisitPropertyRead(ast *PropertyRead, context any) any {
	v.Visit(ast.Receiver, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitMethodCall(ast *MethodCall, context any) any {
	v.Visit(ast.Receiver, context)
	v.VisitAll(ast.Args, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitFunctionCall(ast *FunctionCall, context any) any {
	if ast.Target != nil {
		v.Visit(ast.Target, context)
	}
	v.VisitAll(ast.Args, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitLiteralPrimitive(ast *LiteralPrimitive, context any) any {
	return nil
}

func (v *RecursiveAstVisitor) VisitBinary(ast *Binary, context any) any {
	v.Visit(ast.Left, context)
	v.Visit(ast.Right, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitInterpolation(ast *Interpolation, context any) any {
	v.VisitAll(ast.Expressions, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitKeyedRead(ast *KeyedRead, context any) any {
	v.Visit(ast.Receiver, context)
	v.Visit(ast.Key, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitPipe(ast *BindingPipe, context any) any {
	v.Visit(ast.Exp, context)
	v.VisitAll(ast.Args, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitLiteralArray(ast *LiteralArray, context any) any {
	v.VisitAll(ast.Expressions, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitLiteralMap(ast *LiteralMap, context any) any {
	v.VisitAll(ast.Values, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitConditional(ast *Conditional, context any) any {
	v.Visit(ast.Condition, context)
	v.Visit(ast.TrueExp, context)
	v.Visit(ast.FalseExp, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitKeyedWrite(ast *KeyedWrite, context any) any {
	v.Visit(ast.Receiver, context)
	v.Visit(ast.Key, context)
	v.Visit(ast.Value, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitPropertyWrite(ast *PropertyWrite, context any) any {
	v.Visit(ast.Receiver, context)
	v.Visit(ast.Value, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitChain(ast *Chain, context any) any {
	v.VisitAll(ast.Expressions, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitPrefixNot(ast *PrefixNot, context any) any {
	v.Visit(ast.Expression, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitNonNullAssert(ast *NonNullAssert, context any) any {
	v.Visit(ast.Expression, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitSafePropertyRead(ast *SafePropertyRead, context any) any {
	v.Visit(ast.Receiver, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitSafeMethodCall(ast *SafeMethodCall, context any) any {
	v.Visit(ast.Receiver, context)
	v.VisitAll(ast.Args, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitSafeKeyedRead(ast *SafeKeyedRead, context any) any {
	v.Visit(ast.Receiver, context)
	v.Visit(ast.Key, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitQuote(ast *Quote, context any) any {
	return nil
}

func (v *RecursiveAstVisitor) VisitCall(ast *Call, context any) any {
	v.Visit(ast.Receiver, context)
	v.VisitAll(ast.Args, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitSafeCall(ast *SafeCall, context any) any {
	v.Visit(ast.Receiver, context)
	v.VisitAll(ast.Args, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitVoidExpression(ast *VoidExpression, context any) any {
	v.Visit(ast.Expr, context)
	return nil
}
func (v *RecursiveAstVisitor) VisitTypeofExpression(ast *TypeofExpression, context any) any {
	v.Visit(ast.Expr, context)
	return nil
}
func (v *RecursiveAstVisitor) VisitParenthesizedExpression(ast *ParenthesizedExpression, context any) any {
	v.Visit(ast.Expression, context)
	return nil
}

func (v *RecursiveAstVisitor) VisitThisReceiver(ast *ThisReceiver, context any) any { return nil }

func (v *RecursiveAstVisitor) VisitUnary(ast *Unary, context any) any { 
    v.Visit(ast.Expr, context)
    return nil 
}
