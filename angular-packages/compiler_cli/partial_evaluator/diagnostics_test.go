package partial_evaluator_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyKnownFn struct{}

func (d dummyKnownFn) Evaluate(node *ast.CallExpression, args partial_evaluator.ResolvedValueArray) partial_evaluator.ResolvedValue {
	return nil
}

func TestDescribeResolvedType_Primitives(t *testing.T) {
	assert.Equal(t, "number", partial_evaluator.DescribeResolvedType(float64(0), 1))
	assert.Equal(t, "boolean", partial_evaluator.DescribeResolvedType(true, 1))
	assert.Equal(t, "boolean", partial_evaluator.DescribeResolvedType(false, 1))
	assert.Equal(t, "null", partial_evaluator.DescribeResolvedType(nil, 1))
	assert.Equal(t, "undefined", partial_evaluator.DescribeResolvedType(partial_evaluator.Undefined, 1))
	assert.Equal(t, "string", partial_evaluator.DescribeResolvedType("text", 1))
}

func TestDescribeResolvedType_Objects(t *testing.T) {
	assert.Equal(t, "{}", partial_evaluator.DescribeResolvedType(partial_evaluator.ResolvedValueMap{}, 1))

	m1 := partial_evaluator.ResolvedValueMap{
		"a": float64(0),
		"b": true,
	}
	assert.Equal(t, "{ a: number; b: boolean }", partial_evaluator.DescribeResolvedType(m1, 1))

	m2 := partial_evaluator.ResolvedValueMap{
		"a": partial_evaluator.ResolvedValueMap{},
	}
	assert.Equal(t, "{ a: object }", partial_evaluator.DescribeResolvedType(m2, 1))

	m3 := partial_evaluator.ResolvedValueMap{
		"a": partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3)},
	}
	assert.Equal(t, "{ a: Array }", partial_evaluator.DescribeResolvedType(m3, 1))
}

func TestDescribeResolvedType_Arrays(t *testing.T) {
	assert.Equal(t, "[]", partial_evaluator.DescribeResolvedType(partial_evaluator.ResolvedValueArray{}, 1))

	arr1 := partial_evaluator.ResolvedValueArray{float64(1), float64(2), float64(3)}
	assert.Equal(t, "[number, number, number]", partial_evaluator.DescribeResolvedType(arr1, 1))

	arr2 := partial_evaluator.ResolvedValueArray{
		partial_evaluator.ResolvedValueArray{float64(1), float64(2)},
		partial_evaluator.ResolvedValueArray{float64(3), float64(4)},
	}
	assert.Equal(t, "[Array, Array]", partial_evaluator.DescribeResolvedType(arr2, 1))

	arr3 := partial_evaluator.ResolvedValueArray{
		partial_evaluator.ResolvedValueMap{"a": float64(0)},
	}
	assert.Equal(t, "[object]", partial_evaluator.DescribeResolvedType(arr3, 1))
}

func TestDescribeResolvedType_References(t *testing.T) {
	src := `function test() {}`
	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/file.ts"}, src, core.ScriptKindTS)
	require.NotNil(t, sf)

	var namedFn *ast.Node
	var anonymousFn *ast.Node

	var walk func(n *ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionDeclaration(n) {
			fd := n.AsFunctionDeclaration()
			if fd.Name() != nil && fd.Name().AsIdentifier().Text == "test" {
				namedFn = n
			} else {
				anonymousFn = n
			}
		}
		return n.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)

	if namedFn != nil {
		ref := imports.NewReference(namedFn, nil)
		assert.Equal(t, "test", partial_evaluator.DescribeResolvedType(ref, 1))
	}

	src2 := `function () {}`
	sf2 := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/file2.ts"}, src2, core.ScriptKindTS)
	require.NotNil(t, sf2)
	sf2.AsNode().ForEachChild(walk)

	if anonymousFn != nil {
		ref := imports.NewReference(anonymousFn, nil)
		assert.Equal(t, "(anonymous)", partial_evaluator.DescribeResolvedType(ref, 1))
	}
}

func TestDescribeResolvedType_EnumValues(t *testing.T) {
	src := `enum MyEnum { member = 1 }`
	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/file.ts"}, src, core.ScriptKindTS)
	require.NotNil(t, sf)

	var enumDecl *ast.Node
	var walk func(n *ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsEnumDeclaration(n) {
			enumDecl = n
			return true
		}
		return n.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)
	require.NotNil(t, enumDecl)

	enumVal := &partial_evaluator.EnumValue{
		EnumRef:  enumDecl,
		Name:     "member",
		Resolved: float64(1),
	}
	assert.Equal(t, "MyEnum", partial_evaluator.DescribeResolvedType(enumVal, 1))
}

func TestDescribeResolvedType_DynamicValues(t *testing.T) {
	src := `{}`
	sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/file.ts"}, src, core.ScriptKindTS)
	require.NotNil(t, sf)
	dv := &partial_evaluator.DynamicValue{Node: sf.AsNode(), Reason: "unsupported syntax"}
	assert.Equal(t, "(not statically analyzable)", partial_evaluator.DescribeResolvedType(dv, 1))
}

func TestDescribeResolvedType_KnownFunctions(t *testing.T) {
	fn := dummyKnownFn{}
	assert.Equal(t, "Function", partial_evaluator.DescribeResolvedType(fn, 1))
}

func TestDescribeResolvedType_ExternalModules(t *testing.T) {
	mod := &partial_evaluator.ResolvedModule{}
	assert.Equal(t, "(module)", partial_evaluator.DescribeResolvedType(mod, 1))
}

// ─── traceDynamicValue ────────────────────────────────────────────────────────

func getSourceCode(diag *ast.Diagnostic) string {
	if diag == nil || diag.File() == nil {
		return ""
	}
	text := diag.File().Text()
	pos := diag.Pos()
	end := diag.End()
	if pos < 0 || end > len(text) || pos > end {
		return ""
	}
	pos = scanner.SkipTrivia(text, pos)
	if pos < 0 || pos > end {
		return ""
	}
	return text[pos:end]
}

func diagnosticMessageText(d *ast.Diagnostic) string {
	if d == nil {
		return ""
	}
	if len(d.MessageArgs()) > 0 && d.MessageKey() == "" {
		return d.MessageArgs()[0]
	}
	return d.String()
}

func assertDocumentExternalReference(t *testing.T, diag *ast.Diagnostic) {
	t.Helper()
	assert.Equal(t, "A value for 'document' cannot be determined statically, as it is an external declaration.", diagnosticMessageText(diag))
	filename := diag.File().FileName()
	assert.True(t, filename == "/lib.d.ts" || strings.HasSuffix(filename, "lib.dom.d.ts"), "Expected /lib.d.ts or lib.dom.d.ts, got: %s", filename)
	src := getSourceCode(diag)
	assert.True(t, src == "document: any" || src == "document: Document", "Expected document: any or document: Document, got: %s", src)
}

func traceExpression(t *testing.T, code string, expr string) []*ast.Diagnostic {
	t.Helper()
	programResult := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{Name: "/entry.ts", Contents: code + "; const target$ = " + expr + ";"},
		{Name: "/lib.d.ts", Contents: "declare const document: any;"},
	})
	defer programResult.Release()

	sf := ngtsctest.RequireSourceFile(t, programResult, "/entry.ts")
	decl := ngtsctest.RequireDeclaration(t, sf, "target$", ast.IsVariableDeclaration)
	vd := decl.AsVariableDeclaration()
	require.NotNil(t, vd.Initializer)

	host := reflection.NewTypeScriptReflectionHost(programResult.Checker)
	evaluator := partial_evaluator.NewPartialEvaluator(host, programResult.Checker, nil)
	resolved := evaluator.Evaluate(vd.Initializer, nil)

	dv, ok := resolved.(*partial_evaluator.DynamicValue)
	require.True(t, ok, "Expected DynamicValue, got %T (%v)", resolved, resolved)

	return partial_evaluator.TraceDynamicValue(vd.Initializer, dv)
}

func TestTraceDynamicValue(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping traceDynamicValue tests on Windows")
	}

	t.Run("should not include the origin node if points to a different dynamic node", func(t *testing.T) {
		trace := traceExpression(t, "const value = nonexistent;", "value")
		require.Len(t, trace, 1)
		assert.Equal(t, "Unknown reference.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "nonexistent", getSourceCode(trace[0]))
	})

	t.Run("should include the origin node if it is dynamic by itself", func(t *testing.T) {
		trace := traceExpression(t, "", "nonexistent;")
		require.Len(t, trace, 1)
		assert.Equal(t, "Unknown reference.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "nonexistent", getSourceCode(trace[0]))
	})

	t.Run("should include a trace for a dynamic subexpression in the origin expression", func(t *testing.T) {
		trace := traceExpression(t, "const value = nonexistent;", "value.property")
		require.Len(t, trace, 2)
		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "value", getSourceCode(trace[0]))

		assert.Equal(t, "Unknown reference.", diagnosticMessageText(trace[1]))
		assert.Equal(t, "/entry.ts", trace[1].File().FileName())
		assert.Equal(t, "nonexistent", getSourceCode(trace[1]))
	})

	t.Run("should reduce the granularity to a single entry per statement", func(t *testing.T) {
		trace := traceExpression(t, `
			const firstChild = document.body.childNodes[0];
			const child = firstChild.firstChild;
		`, "child !== undefined")

		require.Len(t, trace, 4)
		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "child", getSourceCode(trace[0]))

		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[1]))
		assert.Equal(t, "/entry.ts", trace[1].File().FileName())
		assert.Equal(t, "firstChild", getSourceCode(trace[1]))

		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[2]))
		assert.Equal(t, "/entry.ts", trace[2].File().FileName())
		assert.Equal(t, "document.body", getSourceCode(trace[2]))

		assertDocumentExternalReference(t, trace[3])
	})

	t.Run("should trace dynamic strings", func(t *testing.T) {
		trace := traceExpression(t, "", "`${document}`")
		require.Len(t, trace, 1)
		assert.Equal(t, "A string value could not be determined statically.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "document", getSourceCode(trace[0]))
	})

	t.Run("should trace invalid expression types", func(t *testing.T) {
		trace := traceExpression(t, "", "true()")
		require.Len(t, trace, 1)
		assert.Equal(t, "Unable to evaluate an invalid expression.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "true", getSourceCode(trace[0]))
	})

	t.Run("should trace unknown syntax", func(t *testing.T) {
		trace := traceExpression(t, "", "new String('test')")
		require.Len(t, trace, 1)
		assert.Equal(t, "This syntax is not supported.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "new String('test')", getSourceCode(trace[0]))
	})

	t.Run("should trace complex function invocations", func(t *testing.T) {
		trace := traceExpression(t, `
			function complex() {
				console.log('test');
				return true;
			}
		`, "complex()")

		require.Len(t, trace, 2)
		assert.Equal(t, "Unable to evaluate function call of complex function. A function must have exactly one return statement.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "complex()", getSourceCode(trace[0]))

		assert.Equal(t, "Function is declared here.", diagnosticMessageText(trace[1]))
		assert.Equal(t, "/entry.ts", trace[1].File().FileName())
		assert.Contains(t, getSourceCode(trace[1]), "console.log('test')")
	})

	t.Run("should trace object destructuring of external reference", func(t *testing.T) {
		trace := traceExpression(t, "const {body: {firstChild}} = document;", "firstChild")
		require.Len(t, trace, 2)
		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "body: {firstChild}", getSourceCode(trace[0]))

		assertDocumentExternalReference(t, trace[1])
	})

	t.Run("should trace deep object destructuring of external reference", func(t *testing.T) {
		trace := traceExpression(t, "const {doc: {body: {firstChild}}} = {doc: document};", "firstChild")
		require.Len(t, trace, 2)
		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "body: {firstChild}", getSourceCode(trace[0]))

		assertDocumentExternalReference(t, trace[1])
	})

	t.Run("should trace array destructuring of dynamic value", func(t *testing.T) {
		trace := traceExpression(t, "const [firstChild] = document.body.childNodes;", "firstChild")
		require.Len(t, trace, 3)
		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[0]))
		assert.Equal(t, "/entry.ts", trace[0].File().FileName())
		assert.Equal(t, "firstChild", getSourceCode(trace[0]))

		assert.Equal(t, "Unable to evaluate this expression statically.", diagnosticMessageText(trace[1]))
		assert.Equal(t, "/entry.ts", trace[1].File().FileName())
		assert.Equal(t, "document.body", getSourceCode(trace[1]))

		assertDocumentExternalReference(t, trace[2])
	})
}
