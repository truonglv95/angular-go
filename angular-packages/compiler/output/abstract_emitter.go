package output

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	SingleQuoteEscapeStringRe = regexp.MustCompile(`'|\\|\n|\r`)
	LegalIdentifierRe         = regexp.MustCompile(`^(?i)[$A-Z_][0-9A-Z_$]*$`)
	IndentWith                = "  "
)

type EmittedLine struct {
	PartsLength int
	Parts       []string
	SrcSpans    []interface{}
	Indent      int
}

func NewEmittedLine(indent int) *EmittedLine {
	return &EmittedLine{
		Indent:   indent,
		Parts:    []string{},
		SrcSpans: []interface{}{},
	}
}

var BinaryOperators = map[BinaryOperator]string{
	BinaryOperatorAnd:                       "&&",
	BinaryOperatorBigger:                    ">",
	BinaryOperatorBiggerEquals:              ">=",
	BinaryOperatorBitwiseOr:                 "|",
	BinaryOperatorBitwiseAnd:                "&",
	BinaryOperatorDivide:                    "/",
	BinaryOperatorAssign:                    "=",
	BinaryOperatorEquals:                    "==",
	BinaryOperatorIdentical:                 "===",
	BinaryOperatorLower:                     "<",
	BinaryOperatorLowerEquals:               "<=",
	BinaryOperatorMinus:                     "-",
	BinaryOperatorModulo:                    "%",
	BinaryOperatorExponentiation:            "**",
	BinaryOperatorMultiply:                  "*",
	BinaryOperatorNotEquals:                 "!=",
	BinaryOperatorNotIdentical:              "!==",
	BinaryOperatorNullishCoalesce:           "??",
	BinaryOperatorOr:                        "||",
	BinaryOperatorPlus:                      "+",
	BinaryOperatorIn:                        "in",
	BinaryOperatorInstanceOf:                "instanceof",
	BinaryOperatorAdditionAssignment:        "+=",
	BinaryOperatorSubtractionAssignment:     "-=",
	BinaryOperatorMultiplicationAssignment:  "*=",
	BinaryOperatorDivisionAssignment:        "/=",
	BinaryOperatorRemainderAssignment:       "%=",
	BinaryOperatorExponentiationAssignment:  "**=",
	BinaryOperatorAndAssignment:             "&&=",
	BinaryOperatorOrAssignment:              "||=",
	BinaryOperatorNullishCoalesceAssignment: "??=",
}

type NodeWithSourceSpan interface {
	GetSourceSpan() interface{}
}

type EmitterVisitorContext struct {
	indent int
	lines  []*EmittedLine
}

func NewEmitterVisitorContext(indent int) *EmitterVisitorContext {
	return &EmitterVisitorContext{
		indent: indent,
		lines:  []*EmittedLine{NewEmittedLine(indent)},
	}
}

func EmitterVisitorContextCreateRoot() *EmitterVisitorContext {
	return NewEmitterVisitorContext(0)
}

func (c *EmitterVisitorContext) currentLine() *EmittedLine {
	return c.lines[len(c.lines)-1]
}

func (c *EmitterVisitorContext) Println(from NodeWithSourceSpan, lastPart string) {
	c.Print(from, lastPart, true)
}

func (c *EmitterVisitorContext) LineIsEmpty() bool {
	return len(c.currentLine().Parts) == 0
}

func (c *EmitterVisitorContext) LineLength() int {
	return c.currentLine().Indent*len(IndentWith) + c.currentLine().PartsLength
}

func (c *EmitterVisitorContext) Print(from NodeWithSourceSpan, part string, newLine bool) {
	if len(part) > 0 {
		c.currentLine().Parts = append(c.currentLine().Parts, part)
		c.currentLine().PartsLength += len(part)
		var span interface{}
		if from != nil && from.GetSourceSpan() != nil {
			span = from.GetSourceSpan()
		}
		c.currentLine().SrcSpans = append(c.currentLine().SrcSpans, span)
	}
	if newLine {
		c.lines = append(c.lines, NewEmittedLine(c.indent))
	}
}

func (c *EmitterVisitorContext) RemoveEmptyLastLine() {
	if c.LineIsEmpty() {
		c.lines = c.lines[:len(c.lines)-1]
	}
}

func (c *EmitterVisitorContext) IncIndent() {
	c.indent++
	if c.LineIsEmpty() {
		c.currentLine().Indent = c.indent
	}
}

func (c *EmitterVisitorContext) DecIndent() {
	c.indent--
	if c.LineIsEmpty() {
		c.currentLine().Indent = c.indent
	}
}

func (c *EmitterVisitorContext) SourceLines() []*EmittedLine {
	if len(c.lines) > 0 && len(c.lines[len(c.lines)-1].Parts) == 0 {
		return c.lines[:len(c.lines)-1]
	}
	return c.lines
}

func (c *EmitterVisitorContext) ToSource() string {
	var parts []string
	for _, l := range c.SourceLines() {
		if len(l.Parts) > 0 {
			parts = append(parts, strings.Repeat(IndentWith, l.Indent)+strings.Join(l.Parts, ""))
		} else {
			parts = append(parts, "")
		}
	}
	return strings.Join(parts, "\n")
}

func (c *EmitterVisitorContext) ToSourceMapGenerator(genFilePath string, startsAtLine int) *SourceMapGenerator {
	mapGen := NewSourceMapGenerator(&genFilePath)

	firstOffsetMapped := false
	space := " "
	zero := 0
	mapFirstOffsetIfNeeded := func() {
		if !firstOffsetMapped {
			mapGen.AddSource(genFilePath, &space).AddMapping(0, &genFilePath, &zero, &zero)
			firstOffsetMapped = true
		}
	}

	for i := 0; i < startsAtLine; i++ {
		mapGen.AddLine()
		mapFirstOffsetIfNeeded()
	}

	for lineIdx, line := range c.SourceLines() {
		mapGen.AddLine()

		spans := line.SrcSpans
		parts := line.Parts
		col0 := line.Indent * len(IndentWith)
		spanIdx := 0

		// skip leading parts without source spans
		for spanIdx < len(spans) && spans[spanIdx] == nil {
			col0 += len(parts[spanIdx])
			spanIdx++
		}

		if spanIdx < len(spans) && lineIdx == 0 && col0 == 0 {
			firstOffsetMapped = true
		} else {
			mapFirstOffsetIfNeeded()
		}

		for spanIdx < len(spans) {
			span := spans[spanIdx].(ParseSourceSpan)
			source := span.GetStart().GetFile()
			sourceLine := span.GetStart().GetLine()
			sourceCol := span.GetStart().GetCol()
			
			content := source.GetContent()
			url := source.GetUrl()
			mapGen.AddSource(url, &content).
				AddMapping(col0, &url, &sourceLine, &sourceCol)

			col0 += len(parts[spanIdx])
			spanIdx++

			// assign parts without span or the same span to the previous segment
			for spanIdx < len(spans) && (spans[spanIdx] == span || spans[spanIdx] == nil) {
				col0 += len(parts[spanIdx])
				spanIdx++
			}
		}
	}

	return mapGen
}

func (c *EmitterVisitorContext) SpanOf(line, column int) interface{} {
	if line < 0 || line >= len(c.lines) {
		return nil
	}
	emittedLine := c.lines[line]
	if emittedLine != nil {
		columnsLeft := column - len(strings.Repeat(IndentWith, emittedLine.Indent))
		for partIndex, part := range emittedLine.Parts {
			if len(part) > columnsLeft {
				return emittedLine.SrcSpans[partIndex]
			}
			columnsLeft -= len(part)
		}
	}
	return nil
}

type AbstractEmitterVisitor struct {
	PrintLeadingComments bool
	PrintTypes           bool
	LastIfCondition      Expression
	Visitor              Visitor
}

func (v *AbstractEmitterVisitor) VisitExpressionStmt(stmt *ExpressionStatement, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(stmt, ctx)
	stmt.Expr.VisitExpression(v.Visitor, ctx)
	ctx.Println(stmt, ";")
	return nil
}

func (v *AbstractEmitterVisitor) VisitReturnStmt(stmt *ReturnStatement, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(stmt, ctx)
	ctx.Print(stmt, "return ", false)
	stmt.Value.VisitExpression(v.Visitor, ctx)
	ctx.Println(stmt, ";")
	return nil
}

func (v *AbstractEmitterVisitor) VisitIfStmt(stmt *IfStmt, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(stmt, ctx)
	ctx.Print(stmt, "if (", false)
	v.LastIfCondition = stmt.Condition
	stmt.Condition.VisitExpression(v.Visitor, ctx)
	v.LastIfCondition = nil
	ctx.Print(stmt, ") {", false)

	hasElseCase := stmt.FalseCase != nil && len(stmt.FalseCase) > 0
	if len(stmt.TrueCase) <= 1 && !hasElseCase {
		ctx.Print(stmt, " ", false)
		v.VisitAllStatements(stmt.TrueCase, ctx)
		ctx.RemoveEmptyLastLine()
		ctx.Print(stmt, " ", false)
	} else {
		ctx.Println(nil, "")
		ctx.IncIndent()
		v.VisitAllStatements(stmt.TrueCase, ctx)
		ctx.DecIndent()
		if hasElseCase {
			ctx.Println(stmt, "} else {")
			ctx.IncIndent()
			v.VisitAllStatements(stmt.FalseCase, ctx)
			ctx.DecIndent()
		}
	}
	ctx.Println(stmt, "}")
	return nil
}

func (v *AbstractEmitterVisitor) VisitDeclareVarStmt(stmt *DeclareVarStmt, context any) any {
	ctx := context.(*EmitterVisitorContext)
	varKind := "let"
	if stmt.HasModifier(StmtModifierFinal) {
		varKind = "const"
	}
	v.printLeadingComments(stmt, ctx)
	ctx.Print(stmt, fmt.Sprintf("%s %s", varKind, stmt.Name), false)
	if stmt.Type != nil {
		stmt.Type.VisitType(v.Visitor, ctx)
	}
	if stmt.Value != nil {
		ctx.Print(stmt, " = ", false)
		stmt.Value.VisitExpression(v.Visitor, ctx)
	}
	ctx.Println(stmt, ";")
	return nil
}

func (v *AbstractEmitterVisitor) VisitInvokeFunctionExpr(expr *InvokeFunctionExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(expr, ctx)
	shouldParenthesize := v.ShouldParenthesize(expr.Fn, expr)
	if shouldParenthesize {
		ctx.Print(expr.Fn, "(", false)
	}
	expr.Fn.VisitExpression(v.Visitor, ctx)
	if shouldParenthesize {
		ctx.Print(expr.Fn, ")", false)
	}
	if expr.IsOptional {
		ctx.Print(expr, "?.(", false)
	} else {
		ctx.Print(expr, "(", false)
	}
	v.VisitAllExpressions(expr.Args, ctx, ",")
	ctx.Print(expr, ")", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitTaggedTemplateLiteralExpr(expr *TaggedTemplateLiteralExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(expr, ctx)
	expr.Tag.VisitExpression(v.Visitor, ctx)
	expr.Template.VisitExpression(v.Visitor, ctx)
	return nil
}

func (v *AbstractEmitterVisitor) VisitTemplateLiteralExpr(expr *TemplateLiteralExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(expr, ctx)
	ctx.Print(expr, "`", false)
	for i := 0; i < len(expr.Elements); i++ {
		expr.Elements[i].VisitExpression(v.Visitor, ctx)
		if i < len(expr.Expressions) {
			expression := expr.Expressions[i]
			ctx.Print(expression, "${", false)
			expression.VisitExpression(v.Visitor, ctx)
			ctx.Print(expression, "}", false)
		}
	}
	ctx.Print(expr, "`", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitTemplateLiteralElementExpr(expr *TemplateLiteralElementExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(expr, ctx)
	ctx.Print(expr, expr.RawText, false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitTypeofExpr(expr *TypeofExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(expr, ctx)
	ctx.Print(expr, "typeof ", false)
	expr.Expr.VisitExpression(v.Visitor, ctx)
	return nil
}

func (v *AbstractEmitterVisitor) VisitVoidExpr(expr *VoidExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(expr, ctx)
	ctx.Print(expr, "void ", false)
	expr.Expr.VisitExpression(v.Visitor, ctx)
	return nil
}

func (v *AbstractEmitterVisitor) VisitReadVarExpr(ast *ReadVarExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, ast.Name, false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitInstantiateExpr(ast *InstantiateExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "new ", false)
	ast.ClassExpr.VisitExpression(v.Visitor, ctx)
	ctx.Print(ast, "(", false)
	v.VisitAllExpressions(ast.Args, ctx, ",")
	ctx.Print(ast, ")", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitLiteralExpr(ast *LiteralExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	if valStr, ok := ast.Value.(string); ok {
		ctx.Print(ast, EscapeIdentifier(valStr, true), false)
	} else {
		ctx.Print(ast, fmt.Sprintf("%v", ast.Value), false)
	}
	return nil
}

func (v *AbstractEmitterVisitor) VisitRegularExpressionLiteral(ast *RegularExpressionLiteralExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	flags := ""
	if ast.Flags != nil {
		flags = *ast.Flags
	}
	ctx.Print(ast, fmt.Sprintf("/%s/%s", ast.Body, flags), false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitLocalizedString(ast *LocalizedString, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	head := ast.SerializeI18nHead()
	ctx.Print(ast, "$localize `"+head.Raw, false)
	for i := 1; i < len(ast.MessageParts); i++ {
		ctx.Print(ast, "${", false)
		ast.Expressions[i-1].VisitExpression(v.Visitor, ctx)
		ctx.Print(ast, fmt.Sprintf("}%s", ast.SerializeI18nTemplatePart(i).Raw), false)
	}
	ctx.Print(ast, "`", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitConditionalExpr(ast *ConditionalExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "(", false)
	ast.Condition.VisitExpression(v.Visitor, ctx)
	ctx.Print(ast, " ? ", false)
	ast.TrueCase.VisitExpression(v.Visitor, ctx)
	ctx.Print(ast, " : ", false)
	if ast.FalseCase != nil {
		ast.FalseCase.VisitExpression(v.Visitor, ctx)
	}
	ctx.Print(ast, ")", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitDynamicImportExpr(ast *DynamicImportExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "import(", false)
	if urlStr, ok := ast.Url.(string); ok {
		ctx.Print(ast, EscapeIdentifier(urlStr, true), false)
	} else if urlExpr, ok := ast.Url.(Expression); ok {
		urlExpr.VisitExpression(v.Visitor, ctx)
	}
	ctx.Print(ast, ")", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitNotExpr(ast *NotExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "!", false)
	ast.Condition.VisitExpression(v.Visitor, ctx)
	return nil
}

func (v *AbstractEmitterVisitor) VisitFunctionExpr(ast *FunctionExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	namePart := ""
	if ast.Name != nil {
		namePart = " " + *ast.Name
	}
	ctx.Print(ast, fmt.Sprintf("function%s(", namePart), false)
	v.VisitParams(ast.Params, ctx)
	ctx.Print(ast, ")", false)
	if ast.Type != nil {
		ast.Type.VisitType(v.Visitor, ctx)
	}
	ctx.Print(ast, " {", false)
	ctx.Println(ast, "")
	ctx.IncIndent()
	v.VisitAllStatements(ast.Statements, ctx)
	ctx.DecIndent()
	ctx.Println(ast, "}")
	return nil
}

func (v *AbstractEmitterVisitor) VisitArrowFunctionExpr(ast *ArrowFunctionExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "(", false)
	v.VisitParams(ast.Params, ctx)
	ctx.Print(ast, ")", false)
	if ast.Type != nil {
		ast.Type.VisitType(v.Visitor, ctx)
	}
	ctx.Print(ast, " => ", false)

	if bodyStmts, ok := ast.Body.([]Statement); ok {
		ctx.Print(ast, "{", false)
		ctx.Println(ast, "")
		ctx.IncIndent()
		v.VisitAllStatements(bodyStmts, ctx)
		ctx.DecIndent()
		ctx.Println(ast, "}")
	} else if bodyExpr, ok := ast.Body.(Expression); ok {
		shouldParenthesize := v.ShouldParenthesize(bodyExpr, ast)
		if shouldParenthesize {
			ctx.Print(ast, "(", false)
		}
		bodyExpr.VisitExpression(v.Visitor, ctx)
		if shouldParenthesize {
			ctx.Print(ast, ")", false)
		}
	}
	return nil
}

func (v *AbstractEmitterVisitor) VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(stmt, ctx)
	ctx.Print(stmt, fmt.Sprintf("function %s(", stmt.Name), false)
	v.VisitParams(stmt.Params, ctx)
	ctx.Print(stmt, ")", false)
	if stmt.Type != nil {
		stmt.Type.VisitType(v.Visitor, ctx)
	}
	ctx.Print(stmt, " {", false)
	ctx.Println(stmt, "")
	ctx.IncIndent()
	v.VisitAllStatements(stmt.Statements, ctx)
	ctx.DecIndent()
	ctx.Println(stmt, "}")
	return nil
}

func (v *AbstractEmitterVisitor) VisitUnaryOperatorExpr(ast *UnaryOperatorExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	var opStr string
	switch ast.Operator {
	case UnaryOperatorPlus:
		opStr = "+"
	case UnaryOperatorMinus:
		opStr = "-"
	default:
		panic(fmt.Sprintf("Unknown operator %v", ast.Operator))
	}
	parens := ast != v.LastIfCondition
	if parens {
		ctx.Print(ast, "(", false)
	}
	ctx.Print(ast, opStr, false)
	ast.Expr.VisitExpression(v.Visitor, ctx)
	if parens {
		ctx.Print(ast, ")", false)
	}
	return nil
}

func (v *AbstractEmitterVisitor) VisitBinaryOperatorExpr(ast *BinaryOperatorExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	operator, ok := BinaryOperators[ast.Operator]
	if !ok {
		panic(fmt.Sprintf("Unknown operator %v", ast.Operator))
	}
	parens := ast != v.LastIfCondition
	if parens {
		ctx.Print(ast, "(", false)
	}
	ast.Lhs.VisitExpression(v.Visitor, ctx)
	ctx.Print(ast, fmt.Sprintf(" %s ", operator), false)
	ast.Rhs.VisitExpression(v.Visitor, ctx)
	if parens {
		ctx.Print(ast, ")", false)
	}
	return nil
}

func (v *AbstractEmitterVisitor) VisitReadPropExpr(ast *ReadPropExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ast.Receiver.VisitExpression(v.Visitor, ctx)
	if ast.IsOptional {
		ctx.Print(ast, "?.", false)
	} else {
		ctx.Print(ast, ".", false)
	}
	ctx.Print(ast, ast.Name, false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitReadKeyExpr(ast *ReadKeyExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ast.Receiver.VisitExpression(v.Visitor, ctx)
	if ast.IsOptional {
		ctx.Print(ast, "?.[", false)
	} else {
		ctx.Print(ast, "[", false)
	}
	ast.Index.VisitExpression(v.Visitor, ctx)
	ctx.Print(ast, "]", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitLiteralArrayExpr(ast *LiteralArrayExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "[", false)
	v.VisitAllExpressions(ast.Entries, ctx, ", ")
	ctx.Print(ast, "]", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitLiteralMapExpr(ast *LiteralMapExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "{", false)

	incrementedIndent := false
	for i, entry := range ast.Entries {
		if i > 0 {
			if ctx.LineLength() > 80 {
				ctx.Print(nil, ", ", true)
				if !incrementedIndent {
					ctx.IncIndent()
					ctx.IncIndent()
					incrementedIndent = true
				}
			} else {
				ctx.Print(nil, ", ", false)
			}
		}

		if spread, ok := entry.(*LiteralMapSpreadAssignment); ok {
			ctx.Print(ast, "...", false)
			spread.Expression.VisitExpression(v.Visitor, ctx)
		} else if prop, ok := entry.(*LiteralMapPropertyAssignment); ok {
			key := prop.Key
			if prop.Quoted {
				ctx.Print(ast, EscapeIdentifier(key, true)+": ", false)
			} else {
				ctx.Print(ast, EscapeIdentifier(key, false)+": ", false)
			}
			prop.Value.VisitExpression(v.Visitor, ctx)
		}
	}
	if incrementedIndent {
		ctx.DecIndent()
		ctx.DecIndent()
	}

	ctx.Print(ast, "}", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitCommaExpr(ast *CommaExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "(", false)
	v.VisitAllExpressions(ast.Parts, ctx, ", ")
	ctx.Print(ast, ")", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitParenthesizedExpr(ast *ParenthesizedExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ast.Expr.VisitExpression(v.Visitor, ctx)
	return nil
}

func (v *AbstractEmitterVisitor) VisitSpreadElementExpr(ast *SpreadElementExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	v.printLeadingComments(ast, ctx)
	ctx.Print(ast, "...", false)
	ast.Expression.VisitExpression(v.Visitor, ctx)
	return nil
}

func (v *AbstractEmitterVisitor) VisitBuiltinType(t *BuiltinType, context any) any {
	ctx := context.(*EmitterVisitorContext)
	if !v.PrintTypes {
		return nil
	}
	switch t.Name {
	case BuiltinTypeNameBool:
		ctx.Print(nil, ": boolean", false)
	case BuiltinTypeNameDynamic:
		ctx.Print(nil, ": any", false)
	case BuiltinTypeNameInt, BuiltinTypeNameNumber:
		ctx.Print(nil, ": number", false)
	case BuiltinTypeNameString:
		ctx.Print(nil, ": string", false)
	case BuiltinTypeNameNone:
		ctx.Print(nil, ": void", false)
	case BuiltinTypeNameInferred:
		// Emits nothing
	case BuiltinTypeNameFunction:
		ctx.Print(nil, ": Function", false)
	default:
		ctx.Print(nil, ": any", false)
	}
	return nil
}

func (v *AbstractEmitterVisitor) VisitExpressionType(t *ExpressionType, context any) any {
	ctx := context.(*EmitterVisitorContext)
	if !v.PrintTypes {
		return nil
	}
	ctx.Print(nil, ": ", false)
	t.Value.VisitExpression(v.Visitor, ctx)

	if len(t.TypeParams) > 0 {
		ctx.Print(nil, "<", false)

		incrementedIndent := false
		for i, param := range t.TypeParams {
			if i > 0 {
				if ctx.LineLength() > 80 {
					ctx.Print(nil, ",", true)
					if !incrementedIndent {
						ctx.IncIndent()
						ctx.IncIndent()
						incrementedIndent = true
					}
				} else {
					ctx.Print(nil, ",", false)
				}
			}
			param.VisitType(v.Visitor, ctx)
		}
		if incrementedIndent {
			ctx.DecIndent()
			ctx.DecIndent()
		}

		ctx.Print(nil, ">", false)
	}
	return nil
}

func (v *AbstractEmitterVisitor) VisitArrayType(t *ArrayType, context any) any {
	ctx := context.(*EmitterVisitorContext)
	if !v.PrintTypes {
		return nil
	}
	ctx.Print(nil, ": ", false)
	t.Of.VisitType(v.Visitor, ctx)
	ctx.Print(nil, "[]", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitMapType(t *MapType, context any) any {
	ctx := context.(*EmitterVisitorContext)
	if !v.PrintTypes {
		return nil
	}
	ctx.Print(nil, ": { [key: string]: ", false)
	if t.ValueType != nil {
		t.ValueType.VisitType(v.Visitor, ctx)
	} else {
		ctx.Print(nil, "any", false)
	}
	ctx.Print(nil, "}", false)
	return nil
}

func (v *AbstractEmitterVisitor) VisitTransplantedType(t *TransplantedType, context any) any {
	panic("TransplantedType nodes are not supported")
}

func (v *AbstractEmitterVisitor) VisitAllExpressions(expressions []Expression, ctx *EmitterVisitorContext, separator string) {
	incrementedIndent := false
	for i, expr := range expressions {
		if i > 0 {
			if ctx.LineLength() > 80 {
				ctx.Print(nil, separator, true)
				if !incrementedIndent {
					ctx.IncIndent()
					ctx.IncIndent()
					incrementedIndent = true
				}
			} else {
				ctx.Print(nil, separator, false)
			}
		}
		expr.VisitExpression(v.Visitor, ctx)
	}
	if incrementedIndent {
		ctx.DecIndent()
		ctx.DecIndent()
	}
}

func (v *AbstractEmitterVisitor) VisitAllStatements(statements []Statement, ctx *EmitterVisitorContext) {
	for _, stmt := range statements {
		stmt.VisitStatement(v.Visitor, ctx)
	}
}

func (v *AbstractEmitterVisitor) VisitParams(params []*FnParam, ctx *EmitterVisitorContext) {
	incrementedIndent := false
	for i, param := range params {
		if i > 0 {
			if ctx.LineLength() > 80 {
				ctx.Print(nil, ", ", true)
				if !incrementedIndent {
					ctx.IncIndent()
					ctx.IncIndent()
					incrementedIndent = true
				}
			} else {
				ctx.Print(nil, ", ", false)
			}
		}
		ctx.Print(nil, param.Name, false)
		if param.Type != nil {
			param.Type.VisitType(v.Visitor, ctx)
		}
	}
	if incrementedIndent {
		ctx.DecIndent()
		ctx.DecIndent()
	}
}

func (v *AbstractEmitterVisitor) ShouldParenthesize(expression Expression, containingExpression Expression) bool {
	switch expression.(type) {
	case *ArrowFunctionExpr, *FunctionExpr:
		_, isInvoke := containingExpression.(*InvokeFunctionExpr)
		return isInvoke
	case *LiteralMapExpr:
		_, isArrow := containingExpression.(*ArrowFunctionExpr)
		return isArrow
	default:
		return false
	}
}

func (v *AbstractEmitterVisitor) printLeadingComments(node interface{}, ctx *EmitterVisitorContext) {
	if !v.PrintLeadingComments {
		return
	}
	var comments []LeadingComment
	var nws NodeWithSourceSpan

	switch n := node.(type) {
	case Expression:
		if hasComments, ok := n.(interface{ GetLeadingComments() []LeadingComment }); ok {
			comments = hasComments.GetLeadingComments()
		}
		if n_span, ok := n.(NodeWithSourceSpan); ok {
			nws = n_span
		}
	case Statement:
		if hasComments, ok := n.(interface{ GetLeadingComments() []LeadingComment }); ok {
			comments = hasComments.GetLeadingComments()
		}
		if n_span, ok := n.(NodeWithSourceSpan); ok {
			nws = n_span
		}
	}

	if comments == nil {
		return
	}

	for _, comment := range comments {
		if comment.Multiline {
			ctx.Print(nws, "/* "+comment.Text+" */", comment.TrailingNewline)
		} else {
			lines := strings.Split(comment.Text, "\n")
			for _, line := range lines {
				ctx.Println(nws, "// "+line)
			}
		}
	}
}

var SINGLE_QUOTE_ESCAPE_STRING_RE = regexp.MustCompile(`'|\\|\n|\r`)
var LEGAL_IDENTIFIER_RE = regexp.MustCompile(`(?i)^[$A-Z_][0-9A-Z_$]*$`)

func EscapeIdentifier(input string, alwaysQuote bool) string {
	body := SINGLE_QUOTE_ESCAPE_STRING_RE.ReplaceAllStringFunc(input, func(match string) string {
		if match == "\n" {
			return "\\n"
		} else if match == "\r" {
			return "\\r"
		} else {
			return "\\" + match
		}
	})

	requiresQuotes := alwaysQuote || !LEGAL_IDENTIFIER_RE.MatchString(body)
	if requiresQuotes {
		return "'" + body + "'"
	}
	return body
}

func (v *AbstractEmitterVisitor) VisitExternalExpr(ast *ExternalExpr, context any) any {
	panic("VisitExternalExpr not implemented")
}
