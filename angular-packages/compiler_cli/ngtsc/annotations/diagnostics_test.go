package annotations_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	ngtscdiag "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getSourceCode(diag *ast.Diagnostic) string {
	if diag == nil || diag.File() == nil {
		return ""
	}
	text := diag.File().Text()
	start := int(diag.Pos())
	length := int(diag.Len())
	if start < 0 || start+length > len(text) {
		return ""
	}
	return text[start : start+length]
}

func createError(t *testing.T, code string, expr string, messageText string, supportingFiles []ngtsctest.ProgramFile) (*ngtscdiag.FatalDiagnosticError, *ngtsctest.ProgramResult) {
	files := []ngtsctest.ProgramFile{
		{
			Name:     "/entry.ts",
			Contents: code + "; const target$ = " + expr,
		},
	}
	files = append(files, supportingFiles...)

	res := ngtsctest.MakeProgram(t, files)

	sf := ngtsctest.RequireSourceFile(t, res, "/entry.ts")
	decl := ngtsctest.RequireDeclaration(t, sf, "target$", ast.IsVariableDeclaration)
	valueExpr := decl.AsVariableDeclaration().Initializer
	require.NotNil(t, valueExpr)

	reflectionHost := reflection.NewTypeScriptReflectionHost(res.Checker)
	evaluator := partial_evaluator.NewPartialEvaluator(reflectionHost, res.Checker, nil)

	value := evaluator.Evaluate(valueExpr, nil)
	errorVal := annotations.CreateValueHasWrongTypeError(valueExpr, value, messageText)
	return errorVal, res
}

func TestAnnotationDiagnostics(t *testing.T) {
	t.Run("createValueError()", func(t *testing.T) {
		t.Run("should include a trace for dynamic values", func(t *testing.T) {
			err, program := createError(t, "", "nonexistent", "Error message", nil)
			defer program.Release()

			entrySf := ngtsctest.RequireSourceFile(t, program, "/entry.ts")

			diag := err.ToDiagnostic()
			require.NotNil(t, diag)

			chain := diag.MessageChain()
			require.Len(t, chain, 1)
			assert.Equal(t, "Value could not be determined statically.", chain[0].MessageArgs()[0])

			related := diag.RelatedInformation()
			require.Len(t, related, 1)
			assert.Equal(t, "Unknown reference.", related[0].MessageArgs()[0])
			assert.Equal(t, entrySf.FileName(), related[0].File().FileName())
			assert.Equal(t, "nonexistent", getSourceCode(related[0]))
		})

		t.Run("should include a pointer for a reference to a named declaration", func(t *testing.T) {
			err, program := createError(t, `import {Foo} from './foo';`, "Foo", "Error message", []ngtsctest.ProgramFile{
				{Name: "/foo.ts", Contents: "export class Foo {}"},
			})
			defer program.Release()

			fooSf := ngtsctest.RequireSourceFile(t, program, "/foo.ts")

			diag := err.ToDiagnostic()
			require.NotNil(t, diag)

			chain := diag.MessageChain()
			require.Len(t, chain, 1)
			assert.Equal(t, "Value is a reference to 'Foo'.", chain[0].MessageArgs()[0])

			related := diag.RelatedInformation()
			require.Len(t, related, 1)
			assert.Equal(t, "Reference is declared here.", related[0].MessageArgs()[0])
			assert.Equal(t, fooSf.FileName(), related[0].File().FileName())
			assert.Equal(t, "Foo", getSourceCode(related[0]))
		})

		t.Run("should include a pointer for a reference to an anonymous declaration", func(t *testing.T) {
			err, program := createError(t, `import Foo from './foo';`, "Foo", "Error message", []ngtsctest.ProgramFile{
				{Name: "/foo.ts", Contents: "export default class {}"},
			})
			defer program.Release()

			fooSf := ngtsctest.RequireSourceFile(t, program, "/foo.ts")

			diag := err.ToDiagnostic()
			require.NotNil(t, diag)

			chain := diag.MessageChain()
			require.Len(t, chain, 1)
			assert.Equal(t, "Value is a reference to an anonymous declaration.", chain[0].MessageArgs()[0])

			related := diag.RelatedInformation()
			require.Len(t, related, 1)
			assert.Equal(t, "Reference is declared here.", related[0].MessageArgs()[0])
			assert.Equal(t, fooSf.FileName(), related[0].File().FileName())
			assert.Equal(t, "export default class {}", getSourceCode(related[0]))
		})

		t.Run("should include a representation of the value's type", func(t *testing.T) {
			err, program := createError(t, "", "{a: 2}", "Error message", nil)
			defer program.Release()

			diag := err.ToDiagnostic()
			require.NotNil(t, diag)

			chain := diag.MessageChain()
			require.Len(t, chain, 1)
			assert.Equal(t, "Value is of type '{ a: number }'.", chain[0].MessageArgs()[0])

			related := diag.RelatedInformation()
			assert.Empty(t, related)
		})
	})
}
