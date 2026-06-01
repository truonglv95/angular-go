package compiler_cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/linker"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/printer"
)

func LinkFile(filePath string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not read file %s: %v", filePath, err)
	}

	content := string(contentBytes)
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		absFilePath = filePath
	}

	output, changed, err := LinkJavaScriptText(absFilePath, content, core.ScriptKindTS)
	if err != nil {
		return err
	}
	if !changed {
		// Output original code if no changes were made
		fmt.Print(content)
		return nil
	}

	fmt.Print(output)

	return nil
}

func LinkJavaScriptText(fileName string, text string, scriptKind core.ScriptKind) (string, bool, error) {
	if !linker.NeedsLinking(fileName, text) {
		return text, false, nil
	}

	opts := ast.SourceFileParseOptions{
		FileName: fileName,
	}
	sourceFile := parser.ParseSourceFile(opts, text, scriptKind)
	if sourceFile == nil {
		return text, false, nil
	}

	constantPool := compiler.NewConstantPool(false)
	result := linker.LinkSourceFile(sourceFile, constantPool)
	if len(result.Errors) > 0 {
		return text, false, fmt.Errorf("linking failed in %s: %v", fileName, result.Errors[0])
	}
	if !result.Changed {
		return text, false, nil
	}

	p := printer.NewPrinter(printer.PrinterOptions{
		NewLine: core.NewLineKindLF,
	}, printer.PrintHandlers{}, nil)

	return p.EmitSourceFile(result.SourceFile), true, nil
}
