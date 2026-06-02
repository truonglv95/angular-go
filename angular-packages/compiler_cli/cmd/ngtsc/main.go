package main

import (
	"flag"
	"os"
	"runtime/pprof"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/perf"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func main() {
	projectFlag := flag.String("p", ".", "Path to the project directory or tsconfig.json file")
	linkFlag := flag.String("link", "", "Path to a JavaScript file to link (AOT compile)")
	cpuprofile := flag.String("cpuprofile", "", "Write cpu profile to file")
	compilationMode := flag.String("compilationMode", "global", "Compilation mode: 'global' or 'local'")
	discardOutput := flag.Bool("discard-output", false, "Discard all compiler output to measure CPU time without disk IO")
	flag.Parse()

	// Disable spelling suggestions for performance during builds
	var stopProfile func()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		stopProfile = func() {
			pprof.StopCPUProfile()
			f.Close()
		}
	}
	exitWith := func(code int) {
		if stopProfile != nil {
			stopProfile()
		}
		if err := perf.WriteConfiguredOutput(); err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
		}
		os.Exit(code)
	}

	if *linkFlag != "" {
		err := compiler_cli.LinkFile(*linkFlag)
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			exitWith(1)
		}
		exitWith(0)
	}

	config := compiler_cli.ReadConfiguration(*projectFlag)
	config.CompilationMode = *compilationMode
	config.DiscardOutput = *discardOutput
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
		exitWith(int(tsc.ExitStatusInvalidProject_OutputsSkipped))
	}

	result := compiler_cli.PerformCompilation(config)
	if len(result.Diagnostics) > 0 {
		diagnosticwriter.FormatDiagnosticsWithColorAndContext(os.Stderr, diagnosticwriter.FromASTDiagnostics(result.Diagnostics), formatOpts)
	}

	exitWith(int(result.Status))
}
