package typecheck

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/program_driver"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeChecker(t *testing.T) {
	t.Run("should batch diagnostic operations when requested in WholeProgram mode", func(t *testing.T) {
		file1 := "/file1.ts"
		file2 := "/file2.ts"
		targets := []TypeCheckingTarget{
			{FileName: file1, Templates: map[string]string{"Cmp1": "<div></div>"}},
			{FileName: file2, Templates: map[string]string{"Cmp2": "<span></span>"}},
		}
		ttc, program, programStrategy := Setup(t, targets, nil)

		sf1 := program.GetSourceFile(file1)
		require.NotNil(t, sf1)
		ttc.GetDiagnosticsForFile(sf1, OptimizeFor_WholeProgram)
		ttcProgram1 := programStrategy.GetProgram()

		sf2 := program.GetSourceFile(file2)
		require.NotNil(t, sf2)
		ttc.GetDiagnosticsForFile(sf2, OptimizeFor_WholeProgram)
		ttcProgram2 := programStrategy.GetProgram()

		assert.True(t, ttcProgram1 == ttcProgram2)
	})

	t.Run("should not batch diagnostic operations when requested in SingleFile mode", func(t *testing.T) {
		file1 := "/file1.ts"
		file2 := "/file2.ts"
		targets := []TypeCheckingTarget{
			{FileName: file1, Templates: map[string]string{"Cmp1": "<div></div>"}},
			{FileName: file2, Templates: map[string]string{"Cmp2": "<span></span>"}},
		}
		ttc, program, programStrategy := Setup(t, targets, nil)

		sf1 := program.GetSourceFile(file1)
		require.NotNil(t, sf1)
		ttc.GetDiagnosticsForFile(sf1, OptimizeFor_SingleFile)
		ttcProgram1 := programStrategy.GetProgram().(*compiler.Program)

		// ttcProgram1 should not contain a type check block for Cmp2.
		ttcSf2Before := ttcProgram1.GetSourceFile("/file2.ngtypecheck.ts")
		require.NotNil(t, ttcSf2Before)
		assert.NotContains(t, ttcSf2Before.Text(), "Cmp2")

		sf2 := program.GetSourceFile(file2)
		require.NotNil(t, sf2)
		ttc.GetDiagnosticsForFile(sf2, OptimizeFor_SingleFile)
		ttcProgram2 := programStrategy.GetProgram().(*compiler.Program)

		// ttcProgram2 should now contain a type check block for Cmp2.
		ttcSf2After := ttcProgram2.GetSourceFile("/file2.ngtypecheck.ts")
		require.NotNil(t, ttcSf2After)
		assert.Contains(t, ttcSf2After.Text(), "Cmp2")

		assert.True(t, ttcProgram1 != ttcProgram2)
	})

	t.Run("should allow access to the type-check block of a component", func(t *testing.T) {
		file1 := "/file1.ts"
		file2 := "/file2.ts"
		targets := []TypeCheckingTarget{
			{FileName: file1, Templates: map[string]string{"Cmp1": "<div>{{value}}</div>"}},
			{FileName: file2, Templates: map[string]string{"Cmp2": "<span></span>"}},
		}
		ttc, program, _ := Setup(t, targets, nil)

		sf1 := program.GetSourceFile(file1)
		require.NotNil(t, sf1)
		cmp1 := ttc.getClassDecl(sf1, "Cmp1")
		require.NotNil(t, cmp1)

		block := ttc.GetTypeCheckBlock(cmp1)
		require.NotNil(t, block)
		blockText := ast.GetSourceFileOfNode(block).Text()[block.Pos():block.End()]
		assert.Contains(t, blockText, "Cmp1")
		assert.Contains(t, blockText, "value")
	})

	t.Run("should clear old inlines when necessary", func(t *testing.T) {
		file1 := "/file1.ts"
		file2 := "/file2.ts"
		dirFile := "/dir.ts"
		dirDeclaration := TestDeclaration{
			Name:      "TestDir",
			Selector:  "[dir]",
			File:      dirFile,
			Type:      "directive",
			IsGeneric: true,
		}
		targets := []TypeCheckingTarget{
			{
				FileName:     file1,
				Templates:    map[string]string{"CmpA": "<div dir></div>"},
				Declarations: []TestDeclaration{dirDeclaration},
			},
			{
				FileName:     file2,
				Templates:    map[string]string{"CmpB": "<div dir></div>"},
				Declarations: []TestDeclaration{dirDeclaration},
			},
			{
				FileName: dirFile,
				Source: `
					interface NotExported {}
					export abstract class TestDir<T extends NotExported> {}
				`,
				Templates: map[string]string{},
			},
		}
		ttc, program, programStrategy := Setup(t, targets, nil)

		sf1 := program.GetSourceFile(file1)
		require.NotNil(t, sf1)
		cmpA := ttc.getClassDecl(sf1, "CmpA")
		require.NotNil(t, cmpA)

		sf2 := program.GetSourceFile(file2)
		require.NotNil(t, sf2)
		cmpB := ttc.getClassDecl(sf2, "CmpB")
		require.NotNil(t, cmpB)

		// Prime the TemplateTypeChecker by asking for a TCB from file1.
		assert.NotNil(t, ttc.GetTypeCheckBlock(cmpA))

		// Next, ask for a TCB from file2. This operation should clear data on TCBs generated for file1.
		assert.NotNil(t, ttc.GetTypeCheckBlock(cmpB))

		// This can be detected by asking for a TCB again from file1. Since no data should be
		// available for file1, this should cause another type-checking program step.
		prevTtcProgram := programStrategy.GetProgram()
		assert.NotNil(t, ttc.GetTypeCheckBlock(cmpA))
		assert.True(t, prevTtcProgram != programStrategy.GetProgram())
	})

	t.Run("when inlining is unsupported", func(t *testing.T) {
		t.Run("should not produce errors for components that do not require inlining", func(t *testing.T) {
			fileName := "/main.ts"
			dirFile := "/dir.ts"
			targets := []TypeCheckingTarget{
				{
					FileName: fileName,
					Source:   `export class Cmp {}`,
					Templates: map[string]string{"Cmp": "<div dir></div>"},
					Declarations: []TestDeclaration{
						{
							Name:     "TestDir",
							Selector: "[dir]",
							File:     dirFile,
							Type:     "directive",
						},
					},
				},
				{
					FileName:  dirFile,
					Source:    `export class TestDir {}`,
					Templates: map[string]string{},
				},
			}
			ttc, program, _ := Setup(t, targets, map[string]any{"inlining": false})

			sf := program.GetSourceFile(fileName)
			require.NotNil(t, sf)
			diags := ttc.GetDiagnosticsForFile(sf, OptimizeFor_WholeProgram)
			assert.Empty(t, diags)
		})

		t.Run("should produce errors for components that require TCB inlining", func(t *testing.T) {
			fileName := "/main.ts"
			targets := []TypeCheckingTarget{
				{
					FileName: fileName,
					Source: `
						abstract class Cmp {
							value: string = 'hello';
						}
					`,
					Templates: map[string]string{"Cmp": "<div></div>"},
				},
			}
			ttc, program, _ := Setup(t, targets, map[string]any{"inlining": false})

			sf := program.GetSourceFile(fileName)
			require.NotNil(t, sf)
			diags := ttc.GetDiagnosticsForFile(sf, OptimizeFor_WholeProgram)
			require.Len(t, diags, 1)
			assert.Equal(t, diagnostics.NgErrorCode(diagnostics.ErrorCode_INLINE_TCB_REQUIRED), int(diags[0].Code()))
		})

		t.Run("should generate TCB in shim file when inlining is unsupported but required", func(t *testing.T) {
			fileName := "/main.ts"
			targets := []TypeCheckingTarget{
				{
					FileName:  fileName,
					Source:    `abstract class Cmp {} // not exported, so requires inline`,
					Templates: map[string]string{"Cmp": "<div></div>"},
				},
			}
			ttc, program, _ := Setup(t, targets, map[string]any{
				"inliningMode": program_driver.InliningMode_CopySourceToTcb,
			})

			sf := program.GetSourceFile(fileName)
			require.NotNil(t, sf)
			diags := ttc.GetDiagnosticsForFile(sf, OptimizeFor_WholeProgram)
			assert.Empty(t, diags)

			cmp := ttc.getClassDecl(sf, "Cmp")
			require.NotNil(t, cmp)
			block := ttc.GetTypeCheckBlock(cmp)
			require.NotNil(t, block)
			assert.Contains(t, ast.GetSourceFileOfNode(block).Text()[block.Pos():block.End()], "Cmp")
		})

		t.Run("should filter out diagnostics from copied source in CopySourceToTcb mode", func(t *testing.T) {
			fileName := "/main.ts"
			targets := []TypeCheckingTarget{
				{
					FileName: fileName,
					Source: `
						abstract class Cmp {
							value: string = 1; // Semantic error
						}
					`,
					Templates: map[string]string{"Cmp": "<div></div>"},
				},
			}
			ttc, program, _ := Setup(t, targets, map[string]any{
				"inliningMode": program_driver.InliningMode_CopySourceToTcb,
			})

			sf := program.GetSourceFile(fileName)
			require.NotNil(t, sf)
			diags := ttc.GetDiagnosticsForFile(sf, OptimizeFor_WholeProgram)
			assert.Empty(t, diags)
		})

		t.Run("should not filter out template errors in CopySourceToTcb mode", func(t *testing.T) {
			fileName := "/main.ts"
			targets := []TypeCheckingTarget{
				{
					FileName: fileName,
					Source: `
						export class Cmp {
							value: string = 'hello';
						}
					`,
					Templates: map[string]string{"Cmp": "<div>{{nonExisting}}</div>"},
				},
			}
			ttc, program, _ := Setup(t, targets, map[string]any{
				"inliningMode": program_driver.InliningMode_CopySourceToTcb,
			})

			sf := program.GetSourceFile(fileName)
			require.NotNil(t, sf)
			diags := ttc.GetDiagnosticsForFile(sf, OptimizeFor_WholeProgram)
			assert.NotEmpty(t, diags)
		})
	})

	t.Run("getTemplateOfComponent()", func(t *testing.T) {
		t.Run("should provide access to a component's real template", func(t *testing.T) {
			fileName := "/main.ts"
			targets := []TypeCheckingTarget{
				{
					FileName:  fileName,
					Templates: map[string]string{"Cmp": "<div>Template</div>"},
				},
			}
			ttc, program, _ := Setup(t, targets, nil)

			sf := program.GetSourceFile(fileName)
			require.NotNil(t, sf)
			cmp := ttc.getClassDecl(sf, "Cmp")
			require.NotNil(t, cmp)

			nodes := ttc.GetTemplate(cmp)
			require.NotEmpty(t, nodes)
			assert.Equal(t, "div", nodes[0].(*render3.Element).Name)
		})
	})
}
