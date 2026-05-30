package expression_parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

type trackingVisitor struct {
	expression_parser.RecursiveAstVisitor
	path []expression_parser.AST
}

func newTrackingVisitor() *trackingVisitor {
	v := &trackingVisitor{
		path: []expression_parser.AST{},
	}
	v.RecursiveAstVisitor.Impl = v
	return v
}

func (v *trackingVisitor) visit(ast expression_parser.AST, context any) {
    if ast != nil {
		v.path = append(v.path, ast)
	}
}

func (v *trackingVisitor) VisitImplicitReceiver(ast *expression_parser.ImplicitReceiver, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitImplicitReceiver(ast, context) 
}
func (v *trackingVisitor) VisitPropertyRead(ast *expression_parser.PropertyRead, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitPropertyRead(ast, context) 
}
func (v *trackingVisitor) VisitMethodCall(ast *expression_parser.MethodCall, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitMethodCall(ast, context) 
}
func (v *trackingVisitor) VisitCall(ast *expression_parser.Call, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitCall(ast, context) 
}
func (v *trackingVisitor) VisitFunctionCall(ast *expression_parser.FunctionCall, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitFunctionCall(ast, context) 
}
func (v *trackingVisitor) VisitLiteralPrimitive(ast *expression_parser.LiteralPrimitive, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitLiteralPrimitive(ast, context) 
}
func (v *trackingVisitor) VisitBinary(ast *expression_parser.Binary, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitBinary(ast, context) 
}
func (v *trackingVisitor) VisitInterpolation(ast *expression_parser.Interpolation, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitInterpolation(ast, context) 
}
func (v *trackingVisitor) VisitKeyedRead(ast *expression_parser.KeyedRead, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitKeyedRead(ast, context) 
}
func (v *trackingVisitor) VisitPipe(ast *expression_parser.BindingPipe, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitPipe(ast, context) 
}
func (v *trackingVisitor) VisitLiteralArray(ast *expression_parser.LiteralArray, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitLiteralArray(ast, context) 
}
func (v *trackingVisitor) VisitLiteralMap(ast *expression_parser.LiteralMap, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitLiteralMap(ast, context) 
}
func (v *trackingVisitor) VisitConditional(ast *expression_parser.Conditional, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitConditional(ast, context) 
}
func (v *trackingVisitor) VisitKeyedWrite(ast *expression_parser.KeyedWrite, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitKeyedWrite(ast, context) 
}
func (v *trackingVisitor) VisitPropertyWrite(ast *expression_parser.PropertyWrite, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitPropertyWrite(ast, context) 
}
func (v *trackingVisitor) VisitChain(ast *expression_parser.Chain, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitChain(ast, context) 
}
func (v *trackingVisitor) VisitPrefixNot(ast *expression_parser.PrefixNot, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitPrefixNot(ast, context) 
}
func (v *trackingVisitor) VisitNonNullAssert(ast *expression_parser.NonNullAssert, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitNonNullAssert(ast, context) 
}
func (v *trackingVisitor) VisitSafePropertyRead(ast *expression_parser.SafePropertyRead, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitSafePropertyRead(ast, context) 
}
func (v *trackingVisitor) VisitSafeMethodCall(ast *expression_parser.SafeMethodCall, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitSafeMethodCall(ast, context) 
}
func (v *trackingVisitor) VisitSafeKeyedRead(ast *expression_parser.SafeKeyedRead, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitSafeKeyedRead(ast, context) 
}
func (v *trackingVisitor) VisitQuote(ast *expression_parser.Quote, context any) any { 
    v.visit(ast, context)
    return v.RecursiveAstVisitor.VisitQuote(ast, context) 
}

func TestRecursiveAstVisitor(t *testing.T) {
	t.Run("should visit every node", func(t *testing.T) {
		parser := expression_parser.NewParser(expression_parser.Lexer{}, false)
		ast := parser.ParseBinding("x.y()", expression_parser.ParseSourceSpan{}, 0 /* absoluteOffset */)
		
		visitor := newTrackingVisitor()
		visitor.Visit(ast.Ast, nil)
		
		assert.Equal(t, 4, len(visitor.path))
		
		call, ok := visitor.path[0].(*expression_parser.Call)
		assert.True(t, ok, "expect call")
		
		yRead, ok := visitor.path[1].(*expression_parser.PropertyRead)
		assert.True(t, ok, "expect property read")
		
		xRead, ok := visitor.path[2].(*expression_parser.PropertyRead)
		assert.True(t, ok, "expect property read")
		
		_, ok = visitor.path[3].(*expression_parser.ImplicitReceiver)
		assert.True(t, ok, "expect implicit receiver")
		
		assert.Equal(t, "x", xRead.Name)
		assert.Equal(t, "y", yRead.Name)
		assert.Equal(t, []expression_parser.AST{}, call.Args)
	})
}
