package translator

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/internal/ast"
)

type RecordWrappedNodeFn func(node *output.WrappedNodeExpr)

type TranslatorOptions struct {
	DownlevelTaggedTemplates      bool
	DownlevelVariableDeclarations bool
	RecordWrappedNode             RecordWrappedNodeFn
	AnnotateForClosureCompiler    bool
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
	name := v.factory.NewIdentifier(stmt.Name)
	return v.factory.NewFunctionDeclaration(nil, nil, name, nil, nil, nil, nil, nil)
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
	return v.factory.NewIfStatement(cond, nil, nil)
}

func (v *ExpressionTranslatorVisitor) VisitReadVarExpr(astNode *output.ReadVarExpr, context any) any {
	return v.factory.NewIdentifier(astNode.Name)
}

func (v *ExpressionTranslatorVisitor) VisitInvokeFunctionExpr(astNode *output.InvokeFunctionExpr, context any) any {
	ctx := context.(Context)
	fn := astNode.Fn.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewCallExpression(fn, nil, nil, nil, 0)
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
	return v.factory.NewNewExpression(classExpr, nil, nil)
}

func (v *ExpressionTranslatorVisitor) VisitLiteralExpr(astNode *output.LiteralExpr, context any) any {
	return v.factory.NewStringLiteral("TODO", 0)
}

func (v *ExpressionTranslatorVisitor) VisitLocalizedString(astNode *output.LocalizedString, context any) any {
	return v.factory.NewIdentifier("$localize")
}

func (v *ExpressionTranslatorVisitor) VisitExternalExpr(astNode *output.ExternalExpr, context any) any {
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
	return v.factory.NewConditionalExpression(cond, nil, trueExpr, nil, falseExpr)
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
	return v.factory.NewFunctionExpression(nil, nil, nil, nil, nil, nil, nil, nil)
}

func (v *ExpressionTranslatorVisitor) VisitUnaryOperatorExpr(astNode *output.UnaryOperatorExpr, context any) any {
	ctx := context.(Context)
	expr := astNode.Expr.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewPrefixUnaryExpression(ast.KindPlusToken, expr)
}

func (v *ExpressionTranslatorVisitor) VisitBinaryOperatorExpr(astNode *output.BinaryOperatorExpr, context any) any {
	ctx := context.(Context)
	lhs := astNode.Lhs.VisitExpression(v, ctx).(*ast.Node)
	rhs := astNode.Rhs.VisitExpression(v, ctx).(*ast.Node)
	return v.factory.NewBinaryExpression(nil, lhs, nil, nil, rhs)
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
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitCommaExpr(astNode *output.CommaExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitWrappedNodeExpr(astNode *output.WrappedNodeExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitTypeofExpr(astNode *output.TypeofExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitVoidExpr(astNode *output.VoidExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitParenthesizedExpr(astNode *output.ParenthesizedExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitReadPropExpr(astNode *output.ReadPropExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitReadKeyExpr(astNode *output.ReadKeyExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitLiteralArrayExpr(astNode *output.LiteralArrayExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitLiteralMapExpr(astNode *output.LiteralMapExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitSpreadElementExpr(astNode *output.SpreadElementExpr, context any) any {
	return nil
}
func (v *ExpressionTranslatorVisitor) VisitRegularExpressionLiteral(astNode *output.RegularExpressionLiteralExpr, context any) any {
	return nil
}
