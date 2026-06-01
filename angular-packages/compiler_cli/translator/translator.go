package translator

import (
	"fmt"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/internal/ast"
	"reflect"
	"strings"
)

type RecordWrappedNodeFn func(node *output.WrappedNodeExpr)

type TranslatorOptions struct {
	DownlevelTaggedTemplates      bool
	DownlevelVariableDeclarations bool
	RecordWrappedNode             RecordWrappedNodeFn
	AnnotateForClosureCompiler    bool
	CoreImportExpression          *ast.Node
}

type Context struct {
	IsStatementMode bool
}

func (c Context) WithExpressionMode() Context {
	return Context{IsStatementMode: false}
}

func (c Context) WithStatementMode() Context {
	return Context{IsStatementMode: true}
}

type ExpressionTranslatorVisitor struct {
	factory                       *ast.NodeFactory
	imports                       *imports.ImportManager
	contextFile                   *ast.Node
	downlevelTaggedTemplates      bool
	downlevelVariableDeclarations bool
	recordWrappedNode             RecordWrappedNodeFn
	coreImportExpression          *ast.Node
}

func NewExpressionTranslatorVisitor(
	factory *ast.NodeFactory,
	importsManager *imports.ImportManager,
	contextFile *ast.Node,
	options TranslatorOptions,
) *ExpressionTranslatorVisitor {
	recordFn := options.RecordWrappedNode
	if recordFn == nil {
		recordFn = func(node *output.WrappedNodeExpr) {}
	}
	return &ExpressionTranslatorVisitor{
		factory:                       factory,
		imports:                       importsManager,
		contextFile:                   contextFile,
		downlevelTaggedTemplates:      options.DownlevelTaggedTemplates,
		downlevelVariableDeclarations: options.DownlevelVariableDeclarations,
		recordWrappedNode:             recordFn,
		coreImportExpression:          options.CoreImportExpression,
	}
}

func (v *ExpressionTranslatorVisitor) VisitDeclareVarStmt(stmt *output.DeclareVarStmt, context any) any {
	ctx := context.(Context)
	var flags ast.NodeFlags
	if !v.downlevelVariableDeclarations {
		if (stmt.Modifiers & output.StmtModifierFinal) != 0 {
			flags |= ast.NodeFlagsConst
		} else {
			flags |= ast.NodeFlagsLet
		}
	}

	name := v.factory.NewIdentifier(stmt.Name)

	var initializer *ast.Node
	if stmt.Value != nil {
		initializer = stmt.Value.VisitExpression(v, ctx.WithExpressionMode()).(*ast.Node)
	}

	decl := v.factory.NewVariableDeclaration(name, nil, nil, initializer)
	declList := v.factory.NewVariableDeclarationList(v.factory.NewNodeList([]*ast.Node{decl}), flags)
	return v.factory.NewVariableStatement(nil, declList)
}

func (v *ExpressionTranslatorVisitor) VisitDeclareFunctionStmt(stmt *output.DeclareFunctionStmt, context any) any {
	ctx := context.(Context)
	name := v.factory.NewIdentifier(stmt.Name)

	var params []*ast.Node
	for _, p := range stmt.Params {
		paramName := v.factory.NewIdentifier(p.Name)
		params = append(params, v.factory.NewParameterDeclaration(nil, nil, (*ast.BindingName)(paramName), nil, nil, nil))
	}

	var bodyStmts []*ast.Node
	for _, s := range stmt.Statements {
		bodyStmts = append(bodyStmts, s.VisitStatement(v, ctx.WithStatementMode()).(*ast.Node))
	}
	body := v.factory.NewBlock((*ast.StatementList)(v.factory.NewNodeList(bodyStmts)), false)

	return v.factory.NewFunctionDeclaration(nil, nil, name, nil, (*ast.ParameterList)(v.factory.NewNodeList(params)), nil, nil, (*ast.FunctionBody)(body))
}

func (v *ExpressionTranslatorVisitor) VisitExpressionStmt(stmt *output.ExpressionStatement, context any) any {
	ctx := context.(Context)
	expr := stmt.Expr.VisitExpression(v, ctx.WithStatementMode()).(*ast.Node)
	return v.factory.NewExpressionStatement(expr)
}

func (v *ExpressionTranslatorVisitor) VisitReturnStmt(stmt *output.ReturnStatement, context any) any {
	ctx := context.(Context)
	var value *ast.Node
	if stmt.Value != nil {
		value = stmt.Value.VisitExpression(v, ctx.WithExpressionMode()).(*ast.Node)
	}
	return v.factory.NewReturnStatement(value)
}

func (v *ExpressionTranslatorVisitor) VisitIfStmt(stmt *output.IfStmt, context any) any {
	ctx := context.(Context)
	cond := stmt.Condition.VisitExpression(v, ctx).(*ast.Node)
	var thenStmts []*ast.Node
	for _, s := range stmt.TrueCase {
		thenStmts = append(thenStmts, s.VisitStatement(v, ctx.WithStatementMode()).(*ast.Node))
	}
	thenBlock := v.factory.NewBlock(v.factory.NewNodeList(thenStmts), false)

	var elseStmt *ast.Node
	if len(stmt.FalseCase) > 0 {
		var elseStmts []*ast.Node
		for _, s := range stmt.FalseCase {
			elseStmts = append(elseStmts, s.VisitStatement(v, ctx.WithStatementMode()).(*ast.Node))
		}
		elseStmt = v.factory.NewBlock(v.factory.NewNodeList(elseStmts), false).AsNode()
	}

	return v.factory.NewIfStatement((*ast.Expression)(cond), thenBlock, (*ast.Statement)(elseStmt))
}

func (v *ExpressionTranslatorVisitor) VisitReadVarExpr(astNode *output.ReadVarExpr, context any) any {
	if strings.HasPrefix(astNode.Name, "ɵɵ") {
		if v.coreImportExpression != nil {
			return v.factory.NewPropertyAccessExpression(
				(*ast.Expression)(v.coreImportExpression),
				nil,
				(*ast.MemberName)(v.factory.NewIdentifier(astNode.Name)),
				0,
			)
		}
		req := imports.ImportRequest{
			ExportModuleSpecifier: "@angular/core",
			ExportSymbolName:      &astNode.Name,
			RequestedFile:         v.contextFile,
			AsTypeReference:       false,
		}
		return v.imports.AddImport(req)
	}
	return v.factory.NewIdentifier(astNode.Name)
}

func (v *ExpressionTranslatorVisitor) VisitInvokeFunctionExpr(astNode *output.InvokeFunctionExpr, context any) any {
	ctx := context.(Context)
	fn := astNode.Fn.VisitExpression(v, ctx).(*ast.Node)
	var args []*ast.Node
	for _, arg := range astNode.Args {
		args = append(args, arg.VisitExpression(v, ctx).(*ast.Node))
	}
	var qDot *ast.Node
	if astNode.IsOptional {
		qDot = v.factory.NewToken(ast.KindQuestionDotToken)
	}
	return v.factory.NewCallExpression((*ast.Expression)(fn), (*ast.QuestionDotToken)(qDot), nil, v.factory.NewNodeList(args), 0)
}

func (v *ExpressionTranslatorVisitor) VisitTaggedTemplateLiteralExpr(astNode *output.TaggedTemplateLiteralExpr, context any) any {
	ctx := context.(Context)
	tag := astNode.Tag.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewTaggedTemplateExpression(tag, nil, nil, nil, 0)
}

func (v *ExpressionTranslatorVisitor) VisitTemplateLiteralExpr(astNode *output.TemplateLiteralExpr, context any) any {
	return v.factory.NewNoSubstitutionTemplateLiteral("", 0)
}

func (v *ExpressionTranslatorVisitor) VisitTemplateLiteralElementExpr(astNode *output.TemplateLiteralElementExpr, context any) any {
	return nil
}

func (v *ExpressionTranslatorVisitor) VisitInstantiateExpr(astNode *output.InstantiateExpr, context any) any {
	ctx := context.(Context)
	classExpr := astNode.ClassExpr.VisitExpression(v, ctx).(*ast.Node)
	var args []*ast.Node
	for _, arg := range astNode.Args {
		args = append(args, arg.VisitExpression(v, ctx).(*ast.Node))
	}
	return v.factory.NewNewExpression((*ast.Expression)(classExpr), nil, v.factory.NewNodeList(args))
}

func (v *ExpressionTranslatorVisitor) VisitLiteralExpr(astNode *output.LiteralExpr, context any) any {
	if astNode.Value == nil {
		return v.factory.NewToken(ast.KindNullKeyword)
	}
	switch val := astNode.Value.(type) {
	case string:
		return v.factory.NewStringLiteral(val, 0)
	case int:
		// Go allows creating a NumericLiteral directly with a string
		return v.factory.NewNumericLiteral(fmt.Sprintf("%d", val), 0)
	case float64:
		return v.factory.NewNumericLiteral(fmt.Sprintf("%f", val), 0)
	case bool:
		if val {
			return v.factory.NewToken(ast.KindTrueKeyword)
		}
		return v.factory.NewToken(ast.KindFalseKeyword)
	default:
		rv := reflect.ValueOf(astNode.Value)
		if rv.Kind() == reflect.Ptr && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Int || rv.Kind() == reflect.Int32 || rv.Kind() == reflect.Int64 {
			return v.factory.NewNumericLiteral(fmt.Sprintf("%d", rv.Int()), 0)
		}
		return v.factory.NewToken(ast.KindNullKeyword)
	}
}

func (v *ExpressionTranslatorVisitor) VisitLocalizedString(astNode *output.LocalizedString, context any) any {
	return v.factory.NewIdentifier("$localize")
}

func (v *ExpressionTranslatorVisitor) VisitExternalExpr(astNode *output.ExternalExpr, context any) any {
	if astNode.Value.ModuleName != nil && *astNode.Value.ModuleName == "@angular/core" && v.coreImportExpression != nil {
		if astNode.Value.Name == nil {
			return v.coreImportExpression
		}
		return v.factory.NewPropertyAccessExpression(
			(*ast.Expression)(v.coreImportExpression),
			nil,
			(*ast.MemberName)(v.factory.NewIdentifier(*astNode.Value.Name)),
			0,
		)
	}
	req := imports.ImportRequest{
		ExportModuleSpecifier: *astNode.Value.ModuleName,
		ExportSymbolName:      astNode.Value.Name,
		RequestedFile:         v.contextFile,
		AsTypeReference:       false,
	}
	return v.imports.AddImport(req)
}

func (v *ExpressionTranslatorVisitor) VisitConditionalExpr(astNode *output.ConditionalExpr, context any) any {
	ctx := context.(Context)
	cond := astNode.Condition.VisitExpression(v, ctx).(*ast.Node)
	trueExpr := astNode.TrueCase.VisitExpression(v, ctx).(*ast.Node)
	var falseExpr *ast.Node
	if astNode.FalseCase != nil {
		falseExpr = astNode.FalseCase.VisitExpression(v, ctx).(*ast.Node)
	}
	// Syntax: ConditionalExpression(condition, questionToken, whenTrue, colonToken, whenFalse)
	questionToken := v.factory.NewToken(ast.KindQuestionToken)
	colonToken := v.factory.NewToken(ast.KindColonToken)
	return v.factory.NewConditionalExpression(cond, questionToken, trueExpr, colonToken, falseExpr)
}

func (v *ExpressionTranslatorVisitor) VisitDynamicImportExpr(astNode *output.DynamicImportExpr, context any) any {
	return v.factory.NewCallExpression(v.factory.NewIdentifier("import"), nil, nil, nil, 0)
}

func (v *ExpressionTranslatorVisitor) VisitNotExpr(astNode *output.NotExpr, context any) any {
	ctx := context.(Context)
	cond := astNode.Condition.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewPrefixUnaryExpression(ast.KindExclamationToken, cond)
}

func (v *ExpressionTranslatorVisitor) VisitFunctionExpr(astNode *output.FunctionExpr, context any) any {
	var name *ast.Node
	if astNode.Name != nil {
		name = v.factory.NewIdentifier(*astNode.Name)
	}

	ctx := context.(Context)
	var params []*ast.Node
	for _, p := range astNode.Params {
		paramName := v.factory.NewIdentifier(p.Name)
		param := v.factory.NewParameterDeclaration(nil, nil, (*ast.BindingName)(paramName), nil, nil, nil)
		params = append(params, param)
	}

	var statements []*ast.Node
	for _, s := range astNode.Statements {
		stmt := s.VisitStatement(v, ctx.WithStatementMode()).(*ast.Node)
		statements = append(statements, stmt)
	}

	body := v.factory.NewBlock(v.factory.NewNodeList(statements), false)
	return v.factory.NewFunctionExpression(nil, nil, (*ast.IdentifierNode)(name), nil, (*ast.ParameterList)(v.factory.NewNodeList(params)), nil, nil, (*ast.FunctionBody)(body))
}

func (v *ExpressionTranslatorVisitor) VisitUnaryOperatorExpr(astNode *output.UnaryOperatorExpr, context any) any {
	ctx := context.(Context)
	expr := astNode.Expr.VisitExpression(v, ctx).(*ast.Node)
	var kind ast.Kind
	switch astNode.Operator {
	case output.UnaryOperatorMinus:
		kind = ast.KindMinusToken
	case output.UnaryOperatorPlus:
		kind = ast.KindPlusToken
	}
	return v.factory.NewPrefixUnaryExpression(kind, expr)
}

func (v *ExpressionTranslatorVisitor) VisitBinaryOperatorExpr(astNode *output.BinaryOperatorExpr, context any) any {
	ctx := context.(Context)
	lhs := astNode.Lhs.VisitExpression(v, ctx).(*ast.Node)
	rhs := astNode.Rhs.VisitExpression(v, ctx).(*ast.Node)
	var kind ast.Kind
	switch astNode.Operator {
	case output.BinaryOperatorAssign:
		kind = ast.KindEqualsToken
	case output.BinaryOperatorEquals:
		kind = ast.KindEqualsEqualsEqualsToken
	case output.BinaryOperatorNotEquals:
		kind = ast.KindExclamationEqualsEqualsToken
	case output.BinaryOperatorIdentical:
		kind = ast.KindEqualsEqualsEqualsToken
	case output.BinaryOperatorNotIdentical:
		kind = ast.KindExclamationEqualsEqualsToken
	case output.BinaryOperatorMinus:
		kind = ast.KindMinusToken
	case output.BinaryOperatorPlus:
		kind = ast.KindPlusToken
	case output.BinaryOperatorDivide:
		kind = ast.KindSlashToken
	case output.BinaryOperatorMultiply:
		kind = ast.KindAsteriskToken
	case output.BinaryOperatorModulo:
		kind = ast.KindPercentToken
	case output.BinaryOperatorAnd:
		kind = ast.KindAmpersandAmpersandToken
	case output.BinaryOperatorOr:
		kind = ast.KindBarBarToken
	case output.BinaryOperatorNullishCoalesce:
		kind = ast.KindQuestionQuestionToken
	case output.BinaryOperatorLower:
		kind = ast.KindLessThanToken
	case output.BinaryOperatorLowerEquals:
		kind = ast.KindLessThanEqualsToken
	case output.BinaryOperatorBigger:
		kind = ast.KindGreaterThanToken
	case output.BinaryOperatorBiggerEquals:
		kind = ast.KindGreaterThanEqualsToken
	case output.BinaryOperatorBitwiseAnd:
		kind = ast.KindAmpersandToken
	case output.BinaryOperatorBitwiseOr:
		kind = ast.KindBarToken
	default:
		kind = ast.KindEqualsEqualsEqualsToken
	}
	opToken := v.factory.NewToken(kind)
	return v.factory.NewBinaryExpression(nil, (*ast.Expression)(lhs), nil, (*ast.BinaryOperatorToken)(opToken), (*ast.Expression)(rhs))
}

func (v *ExpressionTranslatorVisitor) VisitBuiltinType(t *output.BuiltinType, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitExpressionType(t *output.ExpressionType, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitArrayType(t *output.ArrayType, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitMapType(t *output.MapType, context any) any { return nil }
func (v *ExpressionTranslatorVisitor) VisitTransplantedType(t *output.TransplantedType, context any) any {
	return nil
}

func (v *ExpressionTranslatorVisitor) VisitArrowFunctionExpr(astNode *output.ArrowFunctionExpr, context any) any {
	ctx := context.(Context)
	var params []*ast.Node
	for _, p := range astNode.Params {
		paramName := v.factory.NewIdentifier(p.Name)
		param := v.factory.NewParameterDeclaration(nil, nil, (*ast.BindingName)(paramName), nil, nil, nil)
		params = append(params, param)
	}

	var body *ast.Node
	if exprBody, ok := astNode.Body.(output.Expression); ok {
		body = exprBody.VisitExpression(v, ctx).(*ast.Node)
	} else if stmtBody, ok := astNode.Body.([]output.Statement); ok {
		var statements []*ast.Node
		for _, s := range stmtBody {
			stmt := s.VisitStatement(v, ctx.WithStatementMode()).(*ast.Node)
			statements = append(statements, stmt)
		}
		body = v.factory.NewBlock(v.factory.NewNodeList(statements), false).AsNode()
	}
	equalsGreaterThanToken := v.factory.NewToken(ast.KindEqualsGreaterThanToken)
	return v.factory.NewArrowFunction(nil, nil, (*ast.ParameterList)(v.factory.NewNodeList(params)), nil, nil, (*ast.EqualsGreaterThanToken)(equalsGreaterThanToken), (*ast.ConciseBody)(body))
}

func (v *ExpressionTranslatorVisitor) VisitCommaExpr(astNode *output.CommaExpr, context any) any {
	ctx := context.(Context)
	if len(astNode.Parts) == 0 {
		return v.factory.NewOmittedExpression()
	}

	var result *ast.Node = astNode.Parts[0].VisitExpression(v, ctx).(*ast.Node)
	for i := 1; i < len(astNode.Parts); i++ {
		next := astNode.Parts[i].VisitExpression(v, ctx).(*ast.Node)
		commaToken := v.factory.NewToken(ast.KindCommaToken)
		result = v.factory.NewBinaryExpression(nil, (*ast.Expression)(result), nil, (*ast.BinaryOperatorToken)(commaToken), (*ast.Expression)(next))
	}
	return result
}

func (v *ExpressionTranslatorVisitor) VisitWrappedNodeExpr(astNode *output.WrappedNodeExpr, context any) any {
	return astNode.Node
}

func (v *ExpressionTranslatorVisitor) VisitTypeofExpr(astNode *output.TypeofExpr, context any) any {
	ctx := context.(Context)
	expr := astNode.Expr.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewTypeOfExpression(expr)
}

func (v *ExpressionTranslatorVisitor) VisitVoidExpr(astNode *output.VoidExpr, context any) any {
	ctx := context.(Context)
	expr := astNode.Expr.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewVoidExpression(expr)
}

func (v *ExpressionTranslatorVisitor) VisitParenthesizedExpr(astNode *output.ParenthesizedExpr, context any) any {
	ctx := context.(Context)
	expr := astNode.Expr.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewParenthesizedExpression(expr)
}

func (v *ExpressionTranslatorVisitor) VisitReadPropExpr(astNode *output.ReadPropExpr, context any) any {
	ctx := context.(Context)
	receiver := astNode.Receiver.VisitExpression(v, ctx).(*ast.Node)
	name := v.factory.NewIdentifier(astNode.Name)
	var qDot *ast.Node
	if astNode.IsOptional {
		qDot = v.factory.NewToken(ast.KindQuestionDotToken)
	}
	return v.factory.NewPropertyAccessExpression((*ast.Expression)(receiver), (*ast.QuestionDotToken)(qDot), (*ast.MemberName)(name), 0)
}

func (v *ExpressionTranslatorVisitor) VisitReadKeyExpr(astNode *output.ReadKeyExpr, context any) any {
	ctx := context.(Context)
	receiver := astNode.Receiver.VisitExpression(v, ctx).(*ast.Node)
	index := astNode.Index.VisitExpression(v, ctx).(*ast.Node)
	var qDot *ast.Node
	if astNode.IsOptional {
		qDot = v.factory.NewToken(ast.KindQuestionDotToken)
	}
	return v.factory.NewElementAccessExpression((*ast.Expression)(receiver), (*ast.QuestionDotToken)(qDot), (*ast.Expression)(index), 0)
}

func (v *ExpressionTranslatorVisitor) VisitLiteralArrayExpr(astNode *output.LiteralArrayExpr, context any) any {
	ctx := context.(Context)
	var elements []*ast.Node
	for _, e := range astNode.Entries {
		elements = append(elements, e.VisitExpression(v, ctx).(*ast.Node))
	}
	return v.factory.NewArrayLiteralExpression(v.factory.NewNodeList(elements), false)
}

func (v *ExpressionTranslatorVisitor) VisitLiteralMapExpr(astNode *output.LiteralMapExpr, context any) any {
	ctx := context.(Context)
	var properties []*ast.Node
	for _, entry := range astNode.Entries {
		switch e := entry.(type) {
		case *output.LiteralMapPropertyAssignment:
			var name *ast.Node
			if e.Quoted {
				name = v.factory.NewStringLiteral(e.Key, 0)
			} else {
				name = v.factory.NewIdentifier(e.Key)
			}
			val := e.Value.VisitExpression(v, ctx).(*ast.Node)
			prop := v.factory.NewPropertyAssignment(nil, (*ast.PropertyName)(name), nil, nil, (*ast.Expression)(val))
			properties = append(properties, prop.AsNode())
		case *output.LiteralMapSpreadAssignment:
			val := e.Expression.VisitExpression(v, ctx).(*ast.Node)
			prop := v.factory.NewSpreadAssignment(val)
			properties = append(properties, prop.AsNode())
		}
	}
	return v.factory.NewObjectLiteralExpression(v.factory.NewNodeList(properties), false)
}

func (v *ExpressionTranslatorVisitor) VisitSpreadElementExpr(astNode *output.SpreadElementExpr, context any) any {
	ctx := context.(Context)
	expr := astNode.Expression.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewSpreadElement(expr)
}

func (v *ExpressionTranslatorVisitor) VisitRegularExpressionLiteral(astNode *output.RegularExpressionLiteralExpr, context any) any {
	text := "/" + astNode.Body + "/"
	if astNode.Flags != nil {
		text += *astNode.Flags
	}
	return v.factory.NewRegularExpressionLiteral(text, 0)
}
