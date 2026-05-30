package main

import (
	"flag"
	"os"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
)

func main() {
	projectFlag := flag.String("p", ".", "Path to the project directory or tsconfig.json file")
	flag.Parse()

	config := compiler_cli.ReadConfiguration(*projectFlag)
	if len(config.Errors) > 0 {
		for _, err := range config.Errors {
			diagnosticwriter.WriteFlattenedASTDiagnosticMessage(os.Stderr, err, "\n", config.Locale)
		}
		os.Exit(int(tsc.ExitStatusInvalidProject_OutputsSkipped))
	}

	result := compiler_cli.PerformCompilation(config)
	if len(result.Diagnostics) > 0 {
		for _, diag := range result.Diagnostics {
			diagnosticwriter.WriteFlattenedASTDiagnosticMessage(os.Stderr, diag, "\n", config.Locale)
		}
	}

	os.Exit(int(result.Status))
}
