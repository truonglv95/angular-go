package output

import (
	"fmt"
	"strings"
)

const makeTemplateObjectPolyfill = "(this&&this.__makeTemplateObject||function(e,t){return Object.defineProperty?Object.defineProperty(e,\"raw\",{value:t}):e.raw=t,e})"

type AbstractJsEmitterVisitor struct {
	AbstractEmitterVisitor
}

func (v *AbstractJsEmitterVisitor) Init() {
	v.PrintLeadingComments = false
	v.PrintTypes = false
}

func (v *AbstractJsEmitterVisitor) VisitWrappedNodeExpr(ast *WrappedNodeExpr, context any) any {
	panic("Cannot emit a WrappedNodeExpr in Javascript.")
}

func (v *AbstractJsEmitterVisitor) VisitDeclareVarStmt(stmt *DeclareVarStmt, context any) any {
	ctx := context.(*EmitterVisitorContext)
	ctx.Print(stmt, fmt.Sprintf("var %s", stmt.Name), false)
	if stmt.Value != nil {
		ctx.Print(stmt, " = ", false)
		stmt.Value.VisitExpression(v, ctx)
	}
	ctx.Println(stmt, ";")
	return nil
}

func (v *AbstractJsEmitterVisitor) VisitTaggedTemplateLiteralExpr(ast *TaggedTemplateLiteralExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	elements := ast.Template.Elements
	ast.Tag.VisitExpression(v, ctx)
	ctx.Print(ast, fmt.Sprintf("(%s(", makeTemplateObjectPolyfill), false)

	var cooked []string
	var raw []string
	for _, part := range elements {
		cooked = append(cooked, EscapeIdentifier(part.Text, false))
		raw = append(raw, EscapeIdentifier(part.RawText, false))
	}
	ctx.Print(ast, fmt.Sprintf("[%s], ", strings.Join(cooked, ", ")), false)
	ctx.Print(ast, fmt.Sprintf("[%s])", strings.Join(raw, ", ")), false)

	for _, expression := range ast.Template.Expressions {
		ctx.Print(ast, ", ", false)
		expression.VisitExpression(v, ctx)
	}
	ctx.Print(ast, ")", false)
	return nil
}

func (v *AbstractJsEmitterVisitor) VisitTemplateLiteralExpr(expr *TemplateLiteralExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	ctx.Print(expr, "`", false)
	for i := 0; i < len(expr.Elements); i++ {
		expr.Elements[i].VisitExpression(v, ctx)
		if i < len(expr.Expressions) {
			expression := expr.Expressions[i]
			ctx.Print(expression, "${", false)
			expression.VisitExpression(v, ctx)
			ctx.Print(expression, "}", false)
		}
	}
	ctx.Print(expr, "`", false)
	return nil
}

func (v *AbstractJsEmitterVisitor) VisitTemplateLiteralElementExpr(expr *TemplateLiteralElementExpr, context any) any {
	ctx := context.(*EmitterVisitorContext)
	ctx.Print(expr, expr.RawText, false)
	return nil
}

type I18nTemplatePart interface{}

func (v *AbstractJsEmitterVisitor) VisitLocalizedString(ast *LocalizedString, context any) any {
	ctx := context.(*EmitterVisitorContext)
	ctx.Print(ast, fmt.Sprintf("$localize(%s(", makeTemplateObjectPolyfill), false)
	parts := []any{ast.SerializeI18nHead()}
	for i := 1; i < len(ast.MessageParts); i++ {
		parts = append(parts, ast.SerializeI18nTemplatePart(i))
	}

	var cooked []string
	var raw []string
	for _, partRaw := range parts {
		part := partRaw.(map[string]string)
		cooked = append(cooked, EscapeIdentifier(part["Cooked"], false))
		raw = append(raw, EscapeIdentifier(part["Raw"], false))
	}
	ctx.Print(ast, fmt.Sprintf("[%s], ", strings.Join(cooked, ", ")), false)
	ctx.Print(ast, fmt.Sprintf("[%s])", strings.Join(raw, ", ")), false)

	for _, expression := range ast.Expressions {
		ctx.Print(ast, ", ", false)
		expression.VisitExpression(v, ctx)
	}
	ctx.Print(ast, ")", false)
	return nil
}
