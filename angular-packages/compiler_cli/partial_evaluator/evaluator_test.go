package partial_evaluator_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// evaluate is a helper that creates an in-memory program, finds the initializer
// of `target$`, and evaluates it with the PartialEvaluator.
// It mirrors the TypeScript `evaluate` helper from utils.ts:
//
//	`${code}; const target$ = ${expr};`
func evaluate(t *testing.T, code string, expr string) partial_evaluator.ResolvedValue {
	t.Helper()
	src := code + "; const target$ = " + expr + ";"
	opts := ast.SourceFileParseOptions{
		FileName: "/entry.ts",
	}
	sf := parser.ParseSourceFile(opts, src, core.ScriptKindTS)
	require.NotNil(t, sf, "failed to parse source file")

	// Find the initializer of target$
	var initNode *ast.Node
	var walk func(n *ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsVariableDeclaration(n) {
			vd := n.AsVariableDeclaration()
			name := vd.Name()
			if name != nil && ast.IsIdentifier(name) && name.AsIdentifier().Text == "target$" {
				if vd.Initializer != nil {
					initNode = vd.Initializer
					return true
				}
			}
		}
		return n.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)
	require.NotNil(t, initNode, "could not find initializer of target$ in:\n%s", src)

	host := reflection.NewTypeScriptReflectionHost(nil)
	ev := partial_evaluator.NewPartialEvaluator(host, nil, nil)
	return ev.Evaluate(initNode, nil)
}

// -----------------------------------------------------------------------
// Tests ported from evaluator_spec.ts
// -----------------------------------------------------------------------

func TestEvaluator_ForwardRef(t *testing.T) {
	// Test forwardRef with arrow function
	assert.Equal(t, float64(42), evaluate(t, "const val = 42; const ref = forwardRef(() => val);", "ref"))
	
	// Test forwardRef with function expression
	assert.Equal(t, float64(42), evaluate(t, "const val = 42; const ref = forwardRef(function() { return val; });", "ref"))
}

func TestEvaluator_Addition(t *testing.T) {
	result := evaluate(t, "const x = 1 + 2;", "x")
	assert.Equal(t, float64(3), result)
}

func TestEvaluator_AndOperations(t *testing.T) {
	assert.Equal(t, "world", evaluate(t, "const a = 'hello', b = 'world';", "a && b"))
	assert.Equal(t, false, evaluate(t, "const a = false, b = 'world';", "a && b"))
	assert.Equal(t, float64(0), evaluate(t, "const a = 'hello', b = 0;", "a && b"))
}

func TestEvaluator_OrOperations(t *testing.T) {
	assert.Equal(t, "hello", evaluate(t, "const a = 'hello', b = 'world';", "a || b"))
	assert.Equal(t, "world", evaluate(t, "const a = false, b = 'world';", "a || b"))
	assert.Equal(t, "hello", evaluate(t, "const a = 'hello', b = 0;", "a || b"))
}

func TestEvaluator_ArithmeticOperators(t *testing.T) {
	assert.Equal(t, float64(9), evaluate(t, "const a = 6, b = 3;", "a + b"))
	assert.Equal(t, float64(3), evaluate(t, "const a = 6, b = 3;", "a - b"))
	assert.Equal(t, float64(18), evaluate(t, "const a = 6, b = 3;", "a * b"))
	assert.Equal(t, float64(2), evaluate(t, "const a = 6, b = 3;", "a / b"))
	assert.Equal(t, float64(0), evaluate(t, "const a = 6, b = 3;", "a % b"))
	assert.Equal(t, float64(2), evaluate(t, "const a = 6, b = 3;", "a & b"))
	assert.Equal(t, float64(7), evaluate(t, "const a = 6, b = 3;", "a | b"))
	assert.Equal(t, float64(5), evaluate(t, "const a = 6, b = 3;", "a ^ b"))
	assert.Equal(t, float64(216), evaluate(t, "const a = 6, b = 3;", "a ** b"))
	assert.Equal(t, float64(48), evaluate(t, "const a = 6, b = 3;", "a << b"))
	assert.Equal(t, float64(-2), evaluate(t, "const a = -6, b = 2;", "a >> b"))
	assert.Equal(t, float64(1073741822), evaluate(t, "const a = -6, b = 2;", "a >>> b"))
}

func TestEvaluator_ComparisonOperators(t *testing.T) {
	assert.Equal(t, true, evaluate(t, "const a = 2, b = 3;", "a < b"))
	assert.Equal(t, false, evaluate(t, "const a = 3, b = 3;", "a < b"))

	assert.Equal(t, true, evaluate(t, "const a = 3, b = 3;", "a <= b"))
	assert.Equal(t, false, evaluate(t, "const a = 4, b = 3;", "a <= b"))

	assert.Equal(t, true, evaluate(t, "const a = 4, b = 3;", "a > b"))
	assert.Equal(t, false, evaluate(t, "const a = 3, b = 3;", "a > b"))

	assert.Equal(t, true, evaluate(t, "const a = 3, b = 3;", "a >= b"))
	assert.Equal(t, false, evaluate(t, "const a = 2, b = 3;", "a >= b"))

	assert.Equal(t, true, evaluate(t, "const a: any = 3, b = \"3\";", "a == b"))
	assert.Equal(t, false, evaluate(t, "const a: any = 2, b = \"3\";", "a == b"))

	assert.Equal(t, true, evaluate(t, "const a: any = 2, b = \"3\";", "a != b"))
	assert.Equal(t, false, evaluate(t, "const a: any = 3, b = \"3\";", "a != b"))

	assert.Equal(t, true, evaluate(t, "const a: any = 3, b = 3;", "a === b"))
	assert.Equal(t, false, evaluate(t, "const a: any = 3, b = \"3\";", "a === b"))

	assert.Equal(t, true, evaluate(t, "const a: any = 3, b = \"3\";", "a !== b"))
	assert.Equal(t, false, evaluate(t, "const a: any = 3, b = 3;", "a !== b"))
}

func TestEvaluator_Parentheticals(t *testing.T) {
	assert.Equal(t, float64(21), evaluate(t, "const a = 3, b = 4;", "a * (a + b)"))
}

func TestEvaluator_ArrayAccess(t *testing.T) {
	assert.Equal(t, float64(3), evaluate(t, "const a = [1, 2, 3];", "a[1] + a[0]"))
}

func TestEvaluator_ArrayAccessOutOfBounds(t *testing.T) {
	assert.Nil(t, evaluate(t, "const a = [1, 2, 3];", "a[-1]"))
	assert.Nil(t, evaluate(t, "const a = [1, 2, 3];", "a[3]"))
}

func TestEvaluator_ArrayLengthProperty(t *testing.T) {
	result := evaluate(t, "const a = [1, 2, 3];", "a['length'] + 1")
	assert.Equal(t, float64(4), result)
}

func TestEvaluator_ArraySlice(t *testing.T) {
	result := evaluate(t, "const a = [1, 2, 3];", "a['slice']()")
	arr, ok := result.(partial_evaluator.ResolvedValueArray)
	require.True(t, ok, "expected ResolvedValueArray, got %T", result)
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3)}, arr)
}

func TestEvaluator_ArrayConcat(t *testing.T) {
	result := evaluate(t, "const a = [1, 2], b = [3, 4];", "a['concat'](b)")
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3), float64(4)}, result)

	result2 := evaluate(t, "const a = [1, 2], b = 3;", "a['concat'](b)")
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3)}, result2)

	result3 := evaluate(t, "const a = [1, 2], b = 3, c = [4, 5];", "a['concat'](b, c)")
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3), float64(4), float64(5)}, result3)

	result4 := evaluate(t, "const a = [1, 2], b = [3, 4]", "a['concat'](...b)")
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3), float64(4)}, result4)
}

func TestEvaluator_Negation(t *testing.T) {
	assert.Equal(t, false, evaluate(t, "const x = 3;", "!x"))
	assert.Equal(t, true, evaluate(t, "const x = 3;", "!!x"))
}

func TestEvaluator_BooleanLiterals(t *testing.T) {
	assert.Equal(t, true, evaluate(t, "const a = true;", "a"))
	assert.Equal(t, false, evaluate(t, "const a = false;", "a"))
}

func TestEvaluator_Undefined(t *testing.T) {
	assert.Nil(t, evaluate(t, "const a = undefined;", "a"))
}

func TestEvaluator_Null(t *testing.T) {
	assert.Nil(t, evaluate(t, "const a = null;", "a"))
}

func TestEvaluator_NegativeNumbers(t *testing.T) {
	assert.Equal(t, float64(-123), evaluate(t, "const a = -123;", "a"))
}

func TestEvaluator_Array(t *testing.T) {
	result := evaluate(t, "const x = 'test'; const y = [1, x, 2];", "y")
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(1), "test", float64(2)}, result)
}

func TestEvaluator_ArraySpread(t *testing.T) {
	result := evaluate(t, "const a = [1, 2]; const b = [4, 5]; const c = [...a, 3, ...b];", "c")
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3), float64(4), float64(5)}, result)
}

func TestEvaluator_Conditionals(t *testing.T) {
	result := evaluate(t, "const x = false; const y = x ? 'true' : 'false';", "y")
	assert.Equal(t, "false", result)
}

func TestEvaluator_MapAccess(t *testing.T) {
	result := evaluate(t, "const obj = {a: \"test\"};", "obj.a")
	assert.Equal(t, "test", result)
}

func TestEvaluator_FunctionCall(t *testing.T) {
	result := evaluate(t, `function foo(bar) { return bar; }`, `foo("test")`)
	assert.Equal(t, "test", result)
}

func TestEvaluator_FunctionCallDefaultValue(t *testing.T) {
	assert.Equal(t, float64(1), evaluate(t, `function foo(bar = 1) { return bar; }`, "foo()"))
	assert.Equal(t, float64(2), evaluate(t, `function foo(bar = 1) { return bar; }`, "foo(2)"))
	// foo(2) where c has default a (param): c = a in fn = 2
	assert.Equal(t, float64(2), evaluate(t, `function foo(a, c = a) { return c; }; const a = 1;`, "foo(2)"))
}

func TestEvaluator_FunctionCallSpread(t *testing.T) {
	result := evaluate(t, `function foo(a, ...b) { return [a, b]; }`, "foo(1, ...[2, 3])")
	arr, ok := result.(partial_evaluator.ResolvedValueArray)
	require.True(t, ok, "expected array, got %T", result)
	require.Len(t, arr, 2)
	assert.Equal(t, float64(1), arr[0])
	innerArr, ok2 := arr[1].(partial_evaluator.ResolvedValueArray)
	require.True(t, ok2, "expected inner array")
	assert.Equal(t, partial_evaluator.ResolvedValueArray{float64(2), float64(3)}, innerArr)
}

func TestEvaluator_DestructuringArray(t *testing.T) {
	code := `const [a, b, c, d] = [0, 1, 2, 3]; const e = c;`
	assert.Equal(t, float64(0), evaluate(t, code, "a"))
	assert.Equal(t, float64(1), evaluate(t, code, "b"))
	assert.Equal(t, float64(2), evaluate(t, code, "c"))
	assert.Equal(t, float64(3), evaluate(t, code, "d"))
	assert.Equal(t, float64(2), evaluate(t, code, "e"))
}

func TestEvaluator_DestructuringObject(t *testing.T) {
	code := `const {a, b, c, d} = {a: 0, b: 1, c: 2, d: 3}; const e = c;`
	assert.Equal(t, float64(0), evaluate(t, code, "a"))
	assert.Equal(t, float64(1), evaluate(t, code, "b"))
	assert.Equal(t, float64(2), evaluate(t, code, "c"))
	assert.Equal(t, float64(3), evaluate(t, code, "d"))
	assert.Equal(t, float64(2), evaluate(t, code, "e"))
}

func TestEvaluator_DestructuringObjectWithAlias(t *testing.T) {
	result := evaluate(t, `const {a: value} = {a: 5}; const e = value;`, "e")
	assert.Equal(t, float64(5), result)
}

func TestEvaluator_NestedDestructuringObject(t *testing.T) {
	result := evaluate(t, `const {a: {b: {c}}} = {a: {b: {c: 0}}};`, "c")
	assert.Equal(t, float64(0), result)
}

func TestEvaluator_NestedDestructuringArray(t *testing.T) {
	result := evaluate(t, `const [[[a]]] = [[[1]]];`, "a")
	assert.Equal(t, float64(1), result)
}

func TestEvaluator_NestedDestructuringMixed(t *testing.T) {
	result := evaluate(t, `const {a: {b: [[c]]}} = {a: {b: [[1337]]}};`, "c")
	assert.Equal(t, float64(1337), result)
}

func TestEvaluator_UnknownBinaryOperator(t *testing.T) {
	// "location" in window - 'in' operator is unsupported
	result := evaluate(t, `declare const window: any;`, `"location" in window`)
	dv, ok := result.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "expected DynamicValue, got %T (%v)", result, result)
	assert.True(t, dv.IsFromUnsupportedSyntax())
}

func TestEvaluator_UnknownUnaryOperator(t *testing.T) {
	// ++index is unsupported
	result := evaluate(t, `let index = 0;`, "++index")
	dv, ok := result.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "expected DynamicValue, got %T (%v)", result, result)
	assert.True(t, dv.IsFromUnsupportedSyntax())
}

func TestEvaluator_InvalidArrayAccess(t *testing.T) {
	// a[index] where index is 1.5 (not integer)
	result := evaluate(t, `const a = []; const index = 1.5;`, "a[index]")
	dv, ok := result.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "expected DynamicValue, got %T (%v)", result, result)
	assert.True(t, dv.IsFromInvalidExpressionType())
	assert.Equal(t, float64(1.5), dv.Reason)
}

func TestEvaluator_BinaryOnNonLiterals(t *testing.T) {
	// a + b where both are [] (arrays - not numbers or strings)
	result := evaluate(t, `const a: any = []; const b: any = [];`, "a + b")
	dv, ok := result.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "expected DynamicValue, got %T (%v)", result, result)
	reason, ok2 := dv.Reason.(*partial_evaluator.DynamicValue)
	require.True(t, ok2, "expected reason to be DynamicValue, got %T", dv.Reason)
	assert.True(t, reason.IsFromInvalidExpressionType())
	// reason.Reason should be the left operand value (empty array)
	_, isArr := reason.Reason.(partial_evaluator.ResolvedValueArray)
	assert.True(t, isArr, "expected reason.Reason to be array, got %T", reason.Reason)
}

func TestEvaluator_InvalidSpreadInArrayLiteral(t *testing.T) {
	// [1, ...a] where a is true (not array)
	result := evaluate(t, `const a: any = true;`, "[1, ...a]")
	arr, ok := result.(partial_evaluator.ResolvedValueArray)
	require.True(t, ok, "expected array, got %T", result)
	assert.Equal(t, float64(1), arr[0])
	dv, ok2 := arr[1].(*partial_evaluator.DynamicValue)
	require.True(t, ok2, "expected DynamicValue at index 1, got %T", arr[1])
	assert.True(t, dv.IsFromInvalidExpressionType())
	assert.Equal(t, true, dv.Reason)
}

func TestEvaluator_InvalidSpreadInObjectLiteral(t *testing.T) {
	// {b: true, ...a} where a is true (not map)
	result := evaluate(t, `const a: any = true;`, "{b: true, ...a}")
	dv, ok := result.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "expected DynamicValue, got %T (%v)", result, result)
	assert.True(t, dv.IsFromDynamicInput())
	reason, ok2 := dv.Reason.(*partial_evaluator.DynamicValue)
	require.True(t, ok2, "expected reason to be DynamicValue, got %T", dv.Reason)
	assert.True(t, reason.IsFromInvalidExpressionType())
	assert.Equal(t, true, reason.Reason)
}

func TestEvaluator_MapSpread(t *testing.T) {
	result := evaluate(t, "const a = {a: 1}; const b = {b: 2, c: 1}; const c = {...a, ...b, c: 3};", "c")
	m, ok := result.(partial_evaluator.ResolvedValueMap)
	require.True(t, ok, "expected ResolvedValueMap, got %T", result)
	assert.Equal(t, float64(1), m["a"])
	assert.Equal(t, float64(2), m["b"])
	assert.Equal(t, float64(3), m["c"])
}

func TestEvaluator_TemplateExpressions(t *testing.T) {
	result := evaluate(t, "const a = 2, b = 4;", "`1${a}3${b}5`")
	assert.Equal(t, "12345", result)
}

func TestEvaluator_StringConcat(t *testing.T) {
	assert.Equal(t, "1234", evaluate(t, "const a = '12', b = '34';", "a['concat'](b)"))
	assert.Equal(t, "123", evaluate(t, "const a = '12', b = '3';", "a['concat'](b)"))
	assert.Equal(t, "12345", evaluate(t, "const a = '12', b = '3', c = '45';", "a['concat'](b,c)"))

	// concat with mixed types: numbers, booleans, null
	result := evaluate(t, `const a = '1', b = 2, c = '3', d = true, e = null;`, "a['concat'](b,c,d,e)")
	assert.Equal(t, "123truenull", result)
}

func TestEvaluator_IndirectedObjectFunctionCall(t *testing.T) {
	t.Skip("requires function value resolution through object property - complex case")
}

func TestEvaluator_ComplexFunction(t *testing.T) {
	// Function with more than one statement should return DynamicValue from complex function call
	result := evaluate(t, `function foo(bar) { const b = bar; return b; }`, `foo("test")`)
	dv, ok := result.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "expected DynamicValue, got %T (%v)", result, result)
	assert.True(t, dv.IsFromComplexFunctionCall())
}

func TestEvaluator_ShorthandProperties(t *testing.T) {
	// const prop = 42; const target$ = {prop}; → evaluating {prop} → prop resolves to 42
	result := evaluate(t, "const prop = 42;", "{prop}")
	m, ok := result.(partial_evaluator.ResolvedValueMap)
	require.True(t, ok, "expected ResolvedValueMap, got %T", result)
	assert.Equal(t, float64(42), m["prop"])
}

func TestEvaluator_InvalidElementAccess(t *testing.T) {
	// a[index] where a = {} (map) and index = true (bool, not string)
	result := evaluate(t, `const a = {}; const index: any = true;`, "a[index]")
	dv, ok := result.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "expected DynamicValue, got %T (%v)", result, result)
	assert.True(t, dv.IsFromInvalidExpressionType())
	assert.Equal(t, true, dv.Reason)
}

func TestEvaluator_TemplateExpressionsWithEnum(t *testing.T) {
	t.Skip("enum resolution requires type checker support")
}

func TestEvaluator_StringConcatWithEnum(t *testing.T) {
	t.Skip("enum resolution requires type checker support")
}

func TestEvaluator_MapAccessUndefined(t *testing.T) {
	t.Skip("requires type checker to resolve 'obj.bar' for 'declare const obj: any' as undefined")
}

func TestEvaluator_StaticPropertyOnClass(t *testing.T) {
	t.Skip("static class property resolution requires type checker")
}

func TestEvaluator_StaticPropertyCallOnClass(t *testing.T) {
	t.Skip("static class method resolution requires type checker")
}

func TestEvaluator_IndirectedStaticPropertyCall(t *testing.T) {
	t.Skip("static class method indirection requires type checker")
}

func TestEvaluator_EnumResolution(t *testing.T) {
	t.Skip("enum resolution requires type checker support")
}

func TestEvaluator_ReadFileFromImport(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_Imports(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_AbsoluteImports(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_DefaultExport(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_NamedExports(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_ChainOfReExports(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_ModuleSpread(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_CircularImports(t *testing.T) {
	t.Skip("requires cross-file module resolution not yet implemented")
}

func TestEvaluator_DeclarationsOfPrimitiveConstantTypes(t *testing.T) {
	t.Skip("requires type checker to read type annotation literals")
}

func TestEvaluator_DeclarationsOfTuples(t *testing.T) {
	t.Skip("requires type checker to read tuple type annotations")
}

func TestEvaluator_AccessFromExternalVariable(t *testing.T) {
	t.Skip("requires type checker for external reference resolution")
}

func TestEvaluator_VariableDeclarationResolution(t *testing.T) {
	t.Skip("requires type checker for declaration file resolution")
}

func TestEvaluator_DynamicObjectValues(t *testing.T) {
	t.Skip("requires type checker for fn() call resolution")
}

func TestEvaluator_FFRResolution(t *testing.T) {
	t.Skip("requires foreign function resolver - not yet implemented")
}

func TestEvaluator_NoSubstitutionTemplateLiteral(t *testing.T) {
	result := evaluate(t, "", "`hello world`")
	assert.Equal(t, "hello world", result)
}

func TestEvaluator_StringLiteral(t *testing.T) {
	result := evaluate(t, "", `"hello"`)
	assert.Equal(t, "hello", result)
}

func TestEvaluator_NumericLiteral(t *testing.T) {
	result := evaluate(t, "", "42")
	assert.Equal(t, float64(42), result)
}

func TestEvaluator_ObjectLiteralBasic(t *testing.T) {
	result := evaluate(t, "", `{a: 1, b: "test", c: true}`)
	m, ok := result.(partial_evaluator.ResolvedValueMap)
	require.True(t, ok)
	assert.Equal(t, float64(1), m["a"])
	assert.Equal(t, "test", m["b"])
	assert.Equal(t, true, m["c"])
}

func TestEvaluator_PrefixMinus(t *testing.T) {
	result := evaluate(t, "", "-5")
	assert.Equal(t, float64(-5), result)
}

func TestEvaluator_PrefixPlus(t *testing.T) {
	result := evaluate(t, "const x = 3;", "+x")
	assert.Equal(t, float64(3), result)
}

func TestEvaluator_BitwiseNOT(t *testing.T) {
	result := evaluate(t, "const x = 5;", "~x")
	assert.Equal(t, float64(-6), result)
}

func TestEvaluator_VariableResolution(t *testing.T) {
	result := evaluate(t, "const x = 42;", "x")
	assert.Equal(t, float64(42), result)
}

func TestEvaluator_ChainedVariables(t *testing.T) {
	result := evaluate(t, "const x = 10; const y = x + 5;", "y")
	assert.Equal(t, float64(15), result)
}

func TestEvaluator_StringInterpolation(t *testing.T) {
	result := evaluate(t, `const name = "World";`, "`Hello, ${name}!`")
	assert.Equal(t, "Hello, World!", result)
}

func TestEvaluator_ConditionalWithIdentifiers(t *testing.T) {
	result := evaluate(t, "const a = true; const b = 1; const c = 2;", "a ? b : c")
	assert.Equal(t, float64(1), result)
}

func TestEvaluator_NestedConditional(t *testing.T) {
	result := evaluate(t, "const x = 5;", "x > 3 ? 'big' : 'small'")
	assert.Equal(t, "big", result)
}

func TestEvaluator_PropertyAccessOnObject(t *testing.T) {
	result := evaluate(t, "const obj = {x: 10, y: 20};", "obj.x + obj.y")
	assert.Equal(t, float64(30), result)
}

func TestEvaluator_ArrayIndexWithVariable(t *testing.T) {
	result := evaluate(t, "const arr = [10, 20, 30]; const i = 2;", "arr[i]")
	assert.Equal(t, float64(30), result)
}

func TestEvaluator_StringConcatPlus(t *testing.T) {
	result := evaluate(t, `const a = "hello"; const b = " world";`, "a + b")
	assert.Equal(t, "hello world", result)
}

func TestEvaluator_NumberStringConcat(t *testing.T) {
	result := evaluate(t, `const a = 42; const b = " items";`, "a + b")
	assert.Equal(t, "42 items", result)
}

func TestEvaluator_DeepObjectNesting(t *testing.T) {
	result := evaluate(t, "const obj = {a: {b: {c: 42}}};", "obj.a.b.c")
	assert.Equal(t, float64(42), result)
}

func TestEvaluator_ShortCircuitAnd(t *testing.T) {
	// falsy left side - should not evaluate right side
	result := evaluate(t, "const a = 0;", "a && undefined_var")
	assert.Equal(t, float64(0), result)
}

func TestEvaluator_ShortCircuitOr(t *testing.T) {
	// truthy left side - should not evaluate right side
	result := evaluate(t, `const a = "hello";`, "a || undefined_var")
	assert.Equal(t, "hello", result)
}

func TestEvaluator_AsExpression(t *testing.T) {
	// TypeScript `as` expression should be transparent
	result := evaluate(t, "const x = 42;", "x as number")
	assert.Equal(t, float64(42), result)
}

// TestPartialEvaluator_EvaluateObjectLiteralWithScope tests object literals evaluated with a scope.
func TestPartialEvaluator_EvaluateObjectLiteralScope(t *testing.T) {
	sourceText := `
const config = { selector: 'my-app', standalone: true, items: ['a', 'b'] };
`
	opts := ast.SourceFileParseOptions{
		FileName: "/test.ts",
	}

	sf := parser.ParseSourceFile(opts, sourceText, core.ScriptKindTS)
	require.NotNil(t, sf, "Failed to parse source file")

	var objNode *ast.Node
	for _, stmt := range sf.Statements.Nodes {
		stmt.ForEachChild(func(child *ast.Node) bool {
			child.ForEachChild(func(subChild *ast.Node) bool {
				if ast.IsObjectLiteralExpression(subChild) {
					objNode = subChild
				}
				subChild.ForEachChild(func(grandChild *ast.Node) bool {
					if ast.IsObjectLiteralExpression(grandChild) {
						objNode = grandChild
					}
					return false
				})
				return false
			})
			return false
		})
	}

	require.NotNil(t, objNode, "Could not find object literal in AST")

	host := reflection.NewTypeScriptReflectionHost(nil)
	evaluatorInst := partial_evaluator.NewPartialEvaluator(host, nil, nil)

	result := evaluatorInst.Evaluate(objNode, nil)
	require.NotNil(t, result)

	resolvedMap, ok := result.(partial_evaluator.ResolvedValueMap)
	require.True(t, ok, "Expected ResolvedValueMap, got %T", result)

	assert.Equal(t, "my-app", resolvedMap["selector"])
	assert.Equal(t, true, resolvedMap["standalone"])
	items, ok := resolvedMap["items"].(partial_evaluator.ResolvedValueArray)
	require.True(t, ok)
	assert.Equal(t, "a", items[0])
	assert.Equal(t, "b", items[1])
}
