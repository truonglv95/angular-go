package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func main() {
	projectFlag := flag.String("p", ".", "Path to the project directory or tsconfig.json file")
	// M3 FIX: Add --project as a long-form alias for -p so tooling that uses
	// `go-ngc --project tsconfig.app.json` doesn't get an "unknown flag" error.
	flag.StringVar(projectFlag, "project", ".", "Path to the project directory or tsconfig.json file (alias for -p)")
	linkFlag := flag.String("link", "", "Path to a JavaScript file to link (AOT compile)")
	cpuprofile := flag.String("cpuprofile", "", "Write cpu profile to file")
	compilationMode := flag.String("compilationMode", "global", "Compilation mode: 'global' or 'local'")
	hmrFlag := flag.Bool("hmr", false, "Enable Hot Module Replacement code generation")
	discardOutput := flag.Bool("discard-output", false, "Discard all compiler output to measure CPU time without disk IO")
	writeFlag := flag.Bool("write", true, "Write outputs to disk")
	formatFlag := flag.String("format", "text", "Output format: text or json")
	serverFlag := flag.Bool("server", false, "Run in daemon mode via JSON-RPC over stdin/stdout")
	perfReportFlag := flag.Bool("perf-report", false, "Print performance phase timings to os.Stderr")
	preserveImportsFlag := flag.Bool("preserve-imports", true, "Preserve ES imports during emit by enabling verbatimModuleSyntax")
	styleIncludePathsFlag := flag.String("style-include-paths", "", "Comma-separated list of paths for Sass/Less to use")
	flag.Parse()

	if *serverFlag {
		compiler_cli.RunServer()
		os.Exit(0)
	}

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
	config.EnableHmr = *hmrFlag
	config.DiscardOutput = *discardOutput
	config.Write = *writeFlag
	if *preserveImportsFlag {
		compiler_cli.EnablePreserveImports(config)
	}
	if *styleIncludePathsFlag != "" {
		config.StyleIncludePaths = strings.Split(*styleIncludePathsFlag, ",")
	}

	formatOpts := &diagnosticwriter.FormattingOptions{
		NewLine: "\n",
		Locale:  config.Locale,
		ComparePathsOptions: tspath.ComparePathsOptions{
			CurrentDirectory:          ".",
			UseCaseSensitiveFileNames: true,
		},
	}

	result := &compiler_cli.PerformCompilationResult{
		Diagnostics: config.Errors,
		Status:      tsc.ExitStatusDiagnosticsPresent_OutputsSkipped,
	}

	if len(config.Errors) == 0 {
		result = compiler_cli.PerformCompilation(config)
	}

	if *perfReportFlag && result.PerfPhases != nil && len(result.PerfPhases) > 0 {
		os.Stderr.WriteString("--- Performance Report ---\n")
		var keys []string
		for k := range result.PerfPhases {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(os.Stderr, "%-30s: %d µs\n", k, result.PerfPhases[k])
		}
		os.Stderr.WriteString("--------------------------\n")
	}

	// Deduplicate diagnostics
	{
		var uniqueDiags []*ast.Diagnostic
		seen := make(map[string]bool)
		for _, d := range result.Diagnostics {
			file := ""
			if d.File() != nil {
				file = d.File().FileName()
			}
			key := fmt.Sprintf("%s:%d:%d:%s", file, d.Code(), d.Pos(), d.String())
			if !seen[key] {
				seen[key] = true
				uniqueDiags = append(uniqueDiags, d)
			}
		}
		result.Diagnostics = uniqueDiags
	}

	if *formatFlag == "json" {
		type DiagnosticMessage struct {
			Category  string `json:"category"`
			Code      int    `json:"code"`
			Message   string `json:"message"`
			File      string `json:"file,omitempty"`
			Formatted string `json:"formatted,omitempty"`
		}
		type BuildResult struct {
			Outputs     []compiler_cli.OutputFile `json:"outputs"`
			Diagnostics []DiagnosticMessage       `json:"diagnostics"`
			Status      int                       `json:"status"`
			PerfPhases  map[string]int64          `json:"perfPhases,omitempty"`
		}

		var diags []DiagnosticMessage
		for _, d := range result.Diagnostics {
			file := ""
			if d.File() != nil {
				file = d.File().FileName()
			}
			category := "error"
			switch d.Category() {
			case 0: // CategoryWarning
				category = "warning"
			case 1: // CategoryError
				category = "error"
			case 2: // CategorySuggestion
				category = "suggestion"
			case 3: // CategoryMessage
				category = "message"
			}

			var sb strings.Builder
			compareOpts := tspath.ComparePathsOptions{
				CurrentDirectory:          ".",
				UseCaseSensitiveFileNames: true,
			}
			compiler_cli.FormatDiagnosticEsbuildStyle(&sb, d, config.Locale, compareOpts)
			formattedStr := sb.String()

			diags = append(diags, DiagnosticMessage{
				Category:  category,
				Code:      int(d.Code()),
				Message:   d.Localize(config.Locale),
				File:      file,
				Formatted: formattedStr,
			})
		}

		outBytes, _ := json.Marshal(BuildResult{
			Outputs:     result.Outputs,
			Diagnostics: diags,
			Status:      int(result.Status),
			PerfPhases:  result.PerfPhases,
		})
		os.Stdout.Write(outBytes)
		os.Stdout.WriteString("\n")
		exitWith(int(result.Status))
	} else {
		if len(result.Diagnostics) > 0 {
			diagnosticwriter.FormatDiagnosticsWithColorAndContext(os.Stderr, diagnosticwriter.FromASTDiagnostics(result.Diagnostics), formatOpts)
		}
		exitWith(int(result.Status))
	}
}
