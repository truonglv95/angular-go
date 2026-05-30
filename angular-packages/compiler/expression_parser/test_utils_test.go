package expression_parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type unparser struct {
	expression string
}

func (u *unparser) unparse(ast AST) string {
	u.expression = ""
	u.visit(ast)
	return u.expression
}

func (u *unparser) visit(ast AST) {
	if ast == nil {
		return
	}
	// Handle types not in the Visitor interface via type switch
	switch n := ast.(type) {
	case *TemplateLiteral:
		u.expression += "`"
		for i, elem := range n.Elements {
			u.expression += elem.Text
			if i < len(n.Expressions) {
				u.expression += "${"
				u.visit(n.Expressions[i])
				u.expression += "}"
			}
		}
		u.expression += "`"
	case *TaggedTemplateLiteral:
		u.visit(n.Tag)
		u.visit(n.Template)
	case *RegularExpressionLiteral:
		u.expression += "/" + n.Body + "/" + n.Flags
	case *SpreadElement:
		u.expression += "..."
		u.visit(n.Expression)
	case *ArrowFunction:
		if len(n.Params) == 1 {
			if p, ok := n.Params[0].(*ArrowFunctionIdentifierParameter); ok {
				u.expression += p.Name
			}
		} else {
			u.expression += "("
			for i, param := range n.Params {
				if i > 0 {
					u.expression += ", "
				}
				if p, ok := param.(*ArrowFunctionIdentifierParameter); ok {
					u.expression += p.Name
				}
			}
			u.expression += ")"
		}
		u.expression += " => "
		u.visit(n.Body)
	default:
		ast.Visit(u, nil)
	}
}

func (u *unparser) VisitImplicitReceiver(ast *ImplicitReceiver, context any) any { return nil }
func (u *unparser) VisitPropertyRead(ast *PropertyRead, context any) any {
	u.visit(ast.Receiver)
	if _, ok := ast.Receiver.(*ImplicitReceiver); ok {
		u.expression += ast.Name
	} else if _, ok := ast.Receiver.(*ThisReceiver); ok {
		u.expression += ast.Name
	} else {
		u.expression += "." + ast.Name
	}
	return nil
}

func (u *unparser) VisitMethodCall(ast *MethodCall, context any) any {
	u.visit(ast.Receiver)
	if _, ok := ast.Receiver.(*ImplicitReceiver); !ok {
		u.expression += "."
	}
	u.expression += ast.Name + "("
	for i, arg := range ast.Args {
		if i > 0 {
			u.expression += ", "
		}
		u.visit(arg)
	}
	u.expression += ")"
	return nil
}

func (u *unparser) VisitFunctionCall(ast *FunctionCall, context any) any {
	if ast.Target != nil {
		u.visit(ast.Target)
	}
	u.expression += "("
	for i, arg := range ast.Args {
		if i > 0 {
			u.expression += ", "
		}
		u.visit(arg)
	}
	u.expression += ")"
	return nil
}

func (u *unparser) VisitLiteralPrimitive(ast *LiteralPrimitive, context any) any {
	if ast.Value == nil {
		u.expression += "null"
	} else if str, ok := ast.Value.(string); ok {
		// Handle special keyword values stored as strings
		if str == "null" || str == "undefined" || str == "true" || str == "false" {
			u.expression += str
		} else {
			u.expression += "\"" + strings.ReplaceAll(str, "\"", "\\\"") + "\""
		}
	} else if b, ok := ast.Value.(bool); ok {
		if b {
			u.expression += "true"
		} else {
			u.expression += "false"
		}
	} else {
		u.expression += fmt.Sprintf("%v", ast.Value)
	}
	return nil
}

func (u *unparser) VisitBinary(ast *Binary, context any) any {
	u.visit(ast.Left)
	u.expression += " " + ast.Operation + " "
	u.visit(ast.Right)
	return nil
}

func (u *unparser) VisitInterpolation(ast *Interpolation, context any) any {
	for i, str := range ast.Strings {
		u.expression += fmt.Sprintf("%v", str)
		if i < len(ast.Expressions) {
			u.expression += "{{ "
			ast.Expressions[i].Visit(u, context)
			u.expression += " }}"
		}
	}
	return nil
}

func (u *unparser) VisitKeyedRead(ast *KeyedRead, context any) any {
	u.visit(ast.Receiver)
	u.expression += "["
	u.visit(ast.Key)
	u.expression += "]"
	return nil
}

func (u *unparser) VisitPipe(ast *BindingPipe, context any) any {
	u.expression += "("
	u.visit(ast.Exp)
	u.expression += " | " + ast.Name
	for _, arg := range ast.Args {
		u.expression += ":"
		u.visit(arg)
	}
	u.expression += ")"
	return nil
}

func (u *unparser) VisitLiteralArray(ast *LiteralArray, context any) any {
	u.expression += "["
	for i, expr := range ast.Expressions {
		if i > 0 {
			u.expression += ", "
		}
		u.visit(expr)
	}
	u.expression += "]"
	return nil
}

func (u *unparser) VisitLiteralMap(ast *LiteralMap, context any) any {
	u.expression += "{"
	for i, keyNode := range ast.Keys {
		if i > 0 {
			u.expression += ", "
		}
		if _, ok := keyNode.(*LiteralMapSpreadKey); ok {
			u.expression += "..."
		} else if propKey, ok := keyNode.(*LiteralMapPropertyKey); ok {
			if propKey.Quoted {
				u.expression += "\"" + propKey.Key + "\""
			} else {
				u.expression += propKey.Key
			}
			u.expression += ": "
		}
		u.visit(ast.Values[i])
	}
	u.expression += "}"
	return nil
}

func (u *unparser) VisitConditional(ast *Conditional, context any) any {
	u.visit(ast.Condition)
	u.expression += " ? "
	u.visit(ast.TrueExp)
	u.expression += " : "
	u.visit(ast.FalseExp)
	return nil
}

func (u *unparser) VisitKeyedWrite(ast *KeyedWrite, context any) any {
	u.visit(ast.Receiver)
	u.expression += "["
	u.visit(ast.Key)
	u.expression += "] = "
	u.visit(ast.Value)
	return nil
}

func (u *unparser) VisitPropertyWrite(ast *PropertyWrite, context any) any {
	u.visit(ast.Receiver)
	if _, ok := ast.Receiver.(*ImplicitReceiver); ok {
		u.expression += ast.Name
	} else if _, ok := ast.Receiver.(*ThisReceiver); ok {
		u.expression += ast.Name
	} else {
		u.expression += "." + ast.Name
	}
	u.expression += " = "
	u.visit(ast.Value)
	return nil
}

func (u *unparser) VisitChain(ast *Chain, context any) any {
	for i, expr := range ast.Expressions {
		u.visit(expr)
		if i == len(ast.Expressions)-1 {
			u.expression += ";"
		} else {
			u.expression += "; "
		}
	}
	return nil
}

func (u *unparser) VisitPrefixNot(ast *PrefixNot, context any) any {
	u.expression += "!"
	u.visit(ast.Expression)
	return nil
}

func (u *unparser) VisitNonNullAssert(ast *NonNullAssert, context any) any {
	u.visit(ast.Expression)
	u.expression += "!"
	return nil
}

func (u *unparser) VisitSafePropertyRead(ast *SafePropertyRead, context any) any {
	u.visit(ast.Receiver)
	u.expression += "?." + ast.Name
	return nil
}

func (u *unparser) VisitSafeMethodCall(ast *SafeMethodCall, context any) any {
	u.visit(ast.Receiver)
	u.expression += "?." + ast.Name + "("
	for i, arg := range ast.Args {
		if i > 0 {
			u.expression += ", "
		}
		u.visit(arg)
	}
	u.expression += ")"
	return nil
}

func (u *unparser) VisitSafeKeyedRead(ast *SafeKeyedRead, context any) any {
	u.visit(ast.Receiver)
	u.expression += "?.["
	u.visit(ast.Key)
	u.expression += "]"
	return nil
}

func (u *unparser) VisitQuote(ast *Quote, context any) any {
	u.expression += ast.Prefix + ":" + ast.UninterpretedExpression
	return nil
}

func (u *unparser) VisitUnary(ast *Unary, context any) any {
	u.expression += ast.Operator
	u.visit(ast.Expr)
	return nil
}

func (u *unparser) VisitCall(ast *Call, context any) any {
	u.visit(ast.Receiver)
	u.expression += "("
	for i, arg := range ast.Args {
		if i > 0 {
			u.expression += ", "
		}
		u.visit(arg)
	}
	u.expression += ")"
	return nil
}

func (u *unparser) VisitSafeCall(ast *SafeCall, context any) any {
	u.visit(ast.Receiver)
	u.expression += "?.("
	for i, arg := range ast.Args {
		if i > 0 {
			u.expression += ", "
		}
		u.visit(arg)
	}
	u.expression += ")"
	return nil
}
func (u *unparser) VisitThisReceiver(ast *ThisReceiver, context any) any { return nil }
func (u *unparser) VisitParenthesizedExpression(ast *ParenthesizedExpression, context any) any {
	u.expression += "("
	if ast.Expression != nil {
		u.visit(ast.Expression)
	}
	u.expression += ")"
	return nil
}

func (u *unparser) VisitTypeofExpression(ast *TypeofExpression, context any) any {
	u.expression += "typeof "
	u.visit(ast.Expr)
	return nil
}

func (u *unparser) VisitVoidExpression(ast *VoidExpression, context any) any {
	u.expression += "void "
	u.visit(ast.Expr)
	return nil
}

func unparse(ast AST) string {
	u := &unparser{}
	return u.unparse(ast)
}

func parseAction(t *testing.T, exp string) ASTWithSource {
	parser := NewParser(Lexer{}, true)
	return parser.ParseAction(exp, ParseSourceSpan{}, 0)
}

func parseBinding(t *testing.T, exp string, supportsDirectPipeReferences ...bool) ASTWithSource {
	supports := true
	if len(supportsDirectPipeReferences) > 0 {
		supports = supportsDirectPipeReferences[0]
	}
	parser := NewParser(Lexer{}, supports)
	return parser.ParseBinding(exp, ParseSourceSpan{}, 0)
}

func parseInterpolation(t *testing.T, exp string) ASTWithSource {
	parser := NewParser(Lexer{}, true)
	return parser.ParseInterpolation(exp, ParseSourceSpan{}, 0, nil)
}

func parseTemplateBindings(t *testing.T, attr string) TemplateBindingParseResult {
	t.Helper()
	// attr format: *key="value" — strip the leading *
	attr = strings.TrimPrefix(attr, "*")
	// split on first '=' to get key and value
	key := attr
	value := ""
	if idx := strings.Index(attr, "="); idx != -1 {
		key = attr[:idx]
		// strip surrounding quotes from value
		value = strings.Trim(attr[idx+1:], `"'`)
	}
	parser := NewParser(Lexer{}, true)
	return parser.ParseTemplateBindings(key, value, ParseSourceSpan{}, 0, 0)
}

func checkAction(t *testing.T, exp string, expected ...string) {
	t.Helper()
	ast := parseAction(t, exp)
	expToMatch := exp
	if len(expected) > 0 {
		expToMatch = expected[0]
	}
	assert.Equal(t, expToMatch, unparse(ast.Ast))
}

func checkBinding(t *testing.T, exp string, expected ...string) {
	t.Helper()
	ast := parseBinding(t, exp)
	expToMatch := exp
	if len(expected) > 0 {
		expToMatch = expected[0]
	}
	assert.Equal(t, expToMatch, unparse(ast.Ast))
}

func checkInterpolation(t *testing.T, exp string, expected ...string) {
	t.Helper()
	ast := parseInterpolation(t, exp)
	expToMatch := exp
	if len(expected) > 0 {
		expToMatch = expected[0]
	}
	assert.Equal(t, expToMatch, unparse(ast.Ast))
}

func expectError(t *testing.T, errs []ParseError, message string, errorCount ...int) {
	t.Helper()
	if len(errorCount) > 0 {
		assert.Equal(t, errorCount[0], len(errs))
	} else {
		assert.Greater(t, len(errs), 0)
	}

	for _, err := range errs {
		if strings.Contains(err.Message, message) {
			return
		}
	}

	errMsgs := []string{}
	for _, err := range errs {
		errMsgs = append(errMsgs, err.Message)
	}
	assert.Fail(t, fmt.Sprintf("Expected an error containing %q to be reported, but got the errors:\n%s", message, strings.Join(errMsgs, "\n")))
}

func expectActionError(t *testing.T, text string, message string, errorCount ...int) {
	t.Helper()
	ast := parseAction(t, text)
	expectError(t, ast.Errors, message, errorCount...)
}

func expectBindingError(t *testing.T, text string, message string, errorCount ...int) {
	t.Helper()
	ast := parseBinding(t, text)
	expectError(t, ast.Errors, message, errorCount...)
}

func checkActionWithError(t *testing.T, text string, expected string, errorMsg string) {
	t.Helper()
	checkAction(t, text, expected)
	expectActionError(t, text, errorMsg)
}

func mapErrors(errs []ParseError) []string {
	res := []string{}
	for _, e := range errs {
		res = append(res, e.Message)
	}
	return res
}

