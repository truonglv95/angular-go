package integration_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/linker"
	ngcompiler "github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/printer"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func TestLinkerGolden(t *testing.T) {
	nodeModules := "../../new-demo-app/node_modules"
	filesToTest := []string{
		"@angular/common/fesm2022/common.mjs",
		"@angular/forms/fesm2022/forms.mjs",
		"@angular/router/fesm2022/router.mjs",
		"@angular/animations/fesm2022/animations.mjs",
		"@angular/animations/fesm2022/browser.mjs",
		"primeng/fesm2022/primeng-button.mjs",
		"primeng/fesm2022/primeng-table.mjs",
		"primeng/fesm2022/primeng-select.mjs",
		"primeng/fesm2022/primeng-menu.mjs",
		"primeng/fesm2022/primeng-datepicker.mjs",
	}

	for _, file := range filesToTest {
		t.Run(file, func(t *testing.T) {
			fileToTest := filepath.Join(nodeModules, file)
			
			sourceBytes, err := os.ReadFile(fileToTest)
			if err != nil {
				t.Fatalf("failed to read %s: %v", fileToTest, err)
			}
			source := string(sourceBytes)
			
			absFileName, _ := filepath.Abs(fileToTest)
			absFileName = tspath.NormalizePath(absFileName)
			sf := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: absFileName}, source, core.ScriptKindJS)
			result := linker.LinkSourceFile(sf, ngcompiler.NewConstantPool(false))
			
			p := printer.NewPrinter(printer.PrinterOptions{NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, nil)
			goOutput := p.EmitSourceFile(result.SourceFile)
			
			tmpGoOut := filepath.Join(t.TempDir(), "go_out.mjs")
			if err := os.WriteFile(tmpGoOut, []byte(goOutput), 0644); err != nil {
				t.Fatalf("failed to write %s: %v", tmpGoOut, err)
			}
			
			cmd := exec.Command("node", "compare_linker_output.mjs", absFileName, tmpGoOut)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compare failed: %v\nOutput: %s", err, string(out))
			}
		})
	}
}
