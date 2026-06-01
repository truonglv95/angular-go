package main

import (
	"flag"
	"os"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func main() {
	projectFlag := flag.String("p", ".", "Path to the project directory or tsconfig.json file")
	linkFlag := flag.String("link", "", "Path to a JavaScript file to link (AOT compile)")
	flag.Parse()

	if *linkFlag != "" {
		err := compiler_cli.LinkFile(*linkFlag)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		os.Exit(0)
	}

	config := compiler_cli.ReadConfiguration(*projectFlag)
	formatOpts := &diagnosticwriter.FormattingOptions{
		NewLine: "\n",
		Locale:  config.Locale,
		ComparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory:          ".",
			UseCaseSensitiveFileNames: true,
		},
	}

	if len(config.Errors) > 0 {
		diagnosticwriter.FormatDiagnosticsWithColorAndContext(os.Stderr, diagnosticwriter.FromASTDiagnostics(config.Errors), formatOpts)
		os.Exit(int(tsc.ExitStatusInvalidProject_OutputsSkipped))
	}

	result := compiler_cli.PerformCompilation(config)
	if len(result.Diagnostics) > 0 {
		diagnosticwriter.FormatDiagnosticsWithColorAndContext(os.Stderr, diagnosticwriter.FromASTDiagnostics(result.Diagnostics), formatOpts)
	}

	os.Exit(int(result.Status))
}
