package output

type RecursiveAstVisitor struct{}

func NewRecursiveAstVisitor() *RecursiveAstVisitor {
	return &RecursiveAstVisitor{}
}

func (v *RecursiveAstVisitor) VisitType(ast Type, context any) any {
	return ast
}

func (v *RecursiveAstVisitor) VisitExpression(ast Expression, context any) any {
	if ast.GetType() != nil {
		ast.GetType().VisitType(v, context)
	}
	return ast
}

func (v *RecursiveAstVisitor) VisitBuiltinType(type_ *BuiltinType, context any) any {
	return v.VisitType(type_, context)
}

func (v *RecursiveAstVisitor) VisitExpressionType(type_ *ExpressionType, context any) any {
	type_.Value.VisitExpression(v, context)
	if type_.TypeParams != nil {
		for _, param := range type_.TypeParams {
			param.VisitType(v, context)
		}
	}
	return v.VisitType(type_, context)
}

func (v *RecursiveAstVisitor) VisitArrayType(type_ *ArrayType, context any) any {
	return v.VisitType(type_, context)
}

func (v *RecursiveAstVisitor) VisitMapType(type_ *MapType, context any) any {
	return v.VisitType(type_, context)
}

func (v *RecursiveAstVisitor) VisitTransplantedType(type_ *TransplantedType, context any) any {
	return type_
}

func (v *RecursiveAstVisitor) VisitWrappedNodeExpr(ast *WrappedNodeExpr, context any) any {
	return ast
}

func (v *RecursiveAstVisitor) VisitReadVarExpr(ast *ReadVarExpr, context any) any {
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitDynamicImportExpr(ast *DynamicImportExpr, context any) any {
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitInvokeFunctionExpr(ast *InvokeFunctionExpr, context any) any {
	ast.Fn.VisitExpression(v, context)
	v.VisitAllExpressions(ast.Args, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitTaggedTemplateLiteralExpr(ast *TaggedTemplateLiteralExpr, context any) any {
	ast.Tag.VisitExpression(v, context)
	ast.Template.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitInstantiateExpr(ast *InstantiateExpr, context any) any {
	ast.ClassExpr.VisitExpression(v, context)
	v.VisitAllExpressions(ast.Args, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitLiteralExpr(ast *LiteralExpr, context any) any {
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitRegularExpressionLiteral(ast *RegularExpressionLiteralExpr, context any) any {
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitLocalizedString(ast *LocalizedString, context any) any {
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitExternalExpr(ast *ExternalExpr, context any) any {
	if ast.TypeParams != nil {
		for _, type_ := range ast.TypeParams {
			type_.VisitType(v, context)
		}
	}
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitConditionalExpr(ast *ConditionalExpr, context any) any {
	ast.Condition.VisitExpression(v, context)
	ast.TrueCase.VisitExpression(v, context)
	if ast.FalseCase != nil {
		ast.FalseCase.VisitExpression(v, context)
	}
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitNotExpr(ast *NotExpr, context any) any {
	ast.Condition.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitFunctionExpr(ast *FunctionExpr, context any) any {
	v.VisitAllStatements(ast.Statements, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitArrowFunctionExpr(ast *ArrowFunctionExpr, context any) any {
	if stmts, ok := ast.Body.([]Statement); ok {
		v.VisitAllStatements(stmts, context)
	} else if expr, ok := ast.Body.(Expression); ok {
		expr.VisitExpression(v, context)
	}
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitUnaryOperatorExpr(ast *UnaryOperatorExpr, context any) any {
	ast.Expr.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitTypeofExpr(ast *TypeofExpr, context any) any {
	ast.Expr.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitVoidExpr(ast *VoidExpr, context any) any {
	ast.Expr.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitBinaryOperatorExpr(ast *BinaryOperatorExpr, context any) any {
	ast.Lhs.VisitExpression(v, context)
	ast.Rhs.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitReadPropExpr(ast *ReadPropExpr, context any) any {
	ast.Receiver.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitReadKeyExpr(ast *ReadKeyExpr, context any) any {
	ast.Receiver.VisitExpression(v, context)
	ast.Index.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitLiteralArrayExpr(ast *LiteralArrayExpr, context any) any {
	v.VisitAllExpressions(ast.Entries, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitLiteralMapExpr(ast *LiteralMapExpr, context any) any {
	for _, entry := range ast.Entries {
		if spread, ok := entry.(*LiteralMapSpreadAssignment); ok {
			spread.Expression.VisitExpression(v, context)
		} else if prop, ok := entry.(*LiteralMapPropertyAssignment); ok {
			prop.Value.VisitExpression(v, context)
		}
	}
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitCommaExpr(ast *CommaExpr, context any) any {
	v.VisitAllExpressions(ast.Parts, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitTemplateLiteralExpr(ast *TemplateLiteralExpr, context any) any {
	for _, element := range ast.Elements {
		element.VisitExpression(v, context)
	}
	v.VisitAllExpressions(ast.Expressions, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitTemplateLiteralElementExpr(ast *TemplateLiteralElementExpr, context any) any {
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitParenthesizedExpr(ast *ParenthesizedExpr, context any) any {
	ast.Expr.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitSpreadElementExpr(ast *SpreadElementExpr, context any) any {
	ast.Expression.VisitExpression(v, context)
	return v.VisitExpression(ast, context)
}

func (v *RecursiveAstVisitor) VisitAllExpressions(exprs []Expression, context any) {
	for _, expr := range exprs {
		expr.VisitExpression(v, context)
	}
}

func (v *RecursiveAstVisitor) VisitDeclareVarStmt(stmt *DeclareVarStmt, context any) any {
	if stmt.Value != nil {
		stmt.Value.VisitExpression(v, context)
	}
	if stmt.Type != nil {
		stmt.Type.VisitType(v, context)
	}
	return stmt
}

func (v *RecursiveAstVisitor) VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context any) any {
	v.VisitAllStatements(stmt.Statements, context)
	if stmt.Type != nil {
		stmt.Type.VisitType(v, context)
	}
	return stmt
}

func (v *RecursiveAstVisitor) VisitExpressionStmt(stmt *ExpressionStatement, context any) any {
	stmt.Expr.VisitExpression(v, context)
	return stmt
}

func (v *RecursiveAstVisitor) VisitReturnStmt(stmt *ReturnStatement, context any) any {
	if stmt.Value != nil {
		stmt.Value.VisitExpression(v, context)
	}
	return stmt
}

func (v *RecursiveAstVisitor) VisitIfStmt(stmt *IfStmt, context any) any {
	stmt.Condition.VisitExpression(v, context)
	v.VisitAllStatements(stmt.TrueCase, context)
	if stmt.FalseCase != nil {
		v.VisitAllStatements(stmt.FalseCase, context)
	}
	return stmt
}

func (v *RecursiveAstVisitor) VisitAllStatements(stmts []Statement, context any) {
	for _, stmt := range stmts {
		stmt.VisitStatement(v, context)
	}
}
