package typecheck

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/file_system"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/program_driver"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/shims"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateTypeCheckingProgram(t *testing.T) {
	t.Run("should not be created if no components need to be checked", func(t *testing.T) {
		fileName := "/main.ts"
		targets := []TypeCheckingTarget{
			{
				FileName:  fileName,
				Templates: map[string]string{},
				Source:    `export class NotACmp {}`,
			},
		}
		ttc, program, programStrategy := Setup(t, targets, nil)

		sf := program.GetSourceFile(fileName)
		require.NotNil(t, sf)

		ttc.GetDiagnosticsForFile(sf, OptimizeFor_WholeProgram)

		assert.True(t, programStrategy.GetProgram() == program)
	})

	t.Run("should have complete reuse if no structural changes are made to shims", func(t *testing.T) {
		program, host, _, typecheckPath := makeSingleFileProgramWithTypecheckShim(t)
		programStrategy := program_driver.NewTsCreateProgramDriver(program, host, nil, []string{"ngtypecheck"})

		// Update /main.ngtypecheck.ts without changing its shape.
		updates := map[file_system.AbsoluteFsPath]program_driver.FileUpdate{
			file_system.AbsoluteFsPath(typecheckPath): {
				NewText: "export const VERSION = 2;",
			},
		}
		programStrategy.UpdateFiles(updates, program_driver.UpdateMode_Complete)

		assert.True(t, programStrategy.LastUpdateReused)
	})

	t.Run("should have complete reuse if no structural changes are made to input files", func(t *testing.T) {
		program, host, mainPath, _ := makeSingleFileProgramWithTypecheckShim(t)
		programStrategy := program_driver.NewTsCreateProgramDriver(program, host, nil, []string{"ngtypecheck"})

		// Update /main.ts without changing its shape.
		updates := map[file_system.AbsoluteFsPath]program_driver.FileUpdate{
			file_system.AbsoluteFsPath(mainPath): {
				NewText: "export const STILL_NOT_A_COMPONENT = true;",
			},
		}
		programStrategy.UpdateFiles(updates, program_driver.UpdateMode_Complete)

		assert.True(t, programStrategy.LastUpdateReused)
	})
}

func makeSingleFileProgramWithTypecheckShim(t *testing.T) (any, any, string, string) {
	mainPath := "/main.ts"
	typecheckPath := "/main.ngtypecheck.ts"

	files := []ngtsctest.ProgramFile{
		{
			Name:     mainPath,
			Contents: "export const NOT_A_COMPONENT = true;",
		},
		{
			Name:     typecheckPath,
			Contents: "export const VERSION = 1;",
		},
	}
	progResult := ngtsctest.MakeProgram(t, files)
	program := progResult.Program
	host := program.Host()

	sf := program.GetSourceFile(mainPath)
	typecheckSf := program.GetSourceFile(typecheckPath)

	tagger := shims.NewShimReferenceTagger([]string{"ngtypecheck"})
	tagger.Tag(sf)
	tagger.Finalize()

	shims.SetIsShim(typecheckSf, true)
	shims.SetIsFileShimSourceFile(typecheckSf, true)

	return program, host, mainPath, typecheckPath
}
