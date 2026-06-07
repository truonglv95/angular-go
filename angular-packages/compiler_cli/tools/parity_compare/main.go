package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/integration_tests/parity"
)

type status string

const (
	statusExact              status = "exact"
	statusNormalized         status = "normalized"
	statusSemantic           status = "semantic"
	statusMissingGoOutput    status = "missing-go-output"
	statusMissingNgtscOutput status = "missing-ngtsc-output"
	statusCompileFailed      status = "compile-failed"
)

type report struct {
	Baseline baselineReport `json:"baseline"`
	Kind     string         `json:"kind"`
	Fixture  string         `json:"fixture"`
	Status   status         `json:"status"`
	Files    []fileReport   `json:"files,omitempty"`
	Errors   []string       `json:"errors,omitempty"`
}

type baselineReport struct {
	Angular    string `json:"angular"`
	GoCompiler string `json:"goCompiler"`
	Ngc        string `json:"ngc"`
}

type fileReport struct {
	Path           string   `json:"path"`
	Status         status   `json:"status"`
	Normalizations []string `json:"normalizations,omitempty"`
}

type options struct {
	kind           string
	fixture        string
	all            bool
	outDir         string
	angularVersion string
	goCompiler     string
	ngcPath        string
	strict         bool
	failNormalized bool
	includeLegacy  bool
}

type commandResult struct {
	stdout string
	stderr string
	err    error
}

func main() {
	opts := parseFlags()
	repoRoot, err := findRepoRoot()
	if err != nil {
		fatal(err)
	}

	opts.goCompiler = resolveGoCompiler(repoRoot, opts.goCompiler)
	if opts.ngcPath == "" {
		opts.ngcPath = filepath.Join(repoRoot, "angular-packages", "new-demo-app", "node_modules", "@angular", "compiler-cli", "bundles", "src", "bin", "ngc.js")
	} else if !filepath.IsAbs(opts.ngcPath) {
		opts.ngcPath = filepath.Join(repoRoot, opts.ngcPath)
	}
	if !filepath.IsAbs(opts.outDir) {
		opts.outDir = filepath.Join(repoRoot, opts.outDir)
	}

	var reports []report
	switch opts.kind {
	case "render3":
		reports, err = runRender3(repoRoot, opts)
	case "linker":
		reports, err = runLinker(repoRoot, opts)
	default:
		err = fmt.Errorf("unsupported --kind %q; expected render3 or linker", opts.kind)
	}
	if err != nil {
		fatal(err)
	}

	printSummary(reports)
	if shouldFail(reports, opts) {
		os.Exit(1)
	}
}

func parseFlags() options {
	var opts options
	flag.StringVar(&opts.kind, "kind", "render3", "Comparison kind: render3 or linker")
	flag.StringVar(&opts.fixture, "fixture", "", "Fixture name to compare")
	flag.BoolVar(&opts.all, "all", false, "Compare all fixtures for the selected kind")
	flag.StringVar(&opts.outDir, "out", ".tmp/ngtsc-parity", "Directory for raw outputs, diffs, and reports")
	flag.StringVar(&opts.angularVersion, "angular-version", "21.2.15", "Angular compiler baseline version for reports")
	flag.StringVar(&opts.goCompiler, "go-compiler", "go-ngc", "Path to go-ngc binary")
	flag.StringVar(&opts.ngcPath, "ngc", "", "Path to Angular ngc.js")
	flag.BoolVar(&opts.strict, "strict", false, "Exit non-zero on semantic, missing-output, or compile failures")
	flag.BoolVar(&opts.failNormalized, "fail-normalized", false, "In strict mode, also fail when only normalized differences remain")
	flag.BoolVar(&opts.includeLegacy, "include-legacy", false, "Include non-parity render3 fixtures when using --all")
	flag.Parse()

	if !opts.all && opts.fixture == "" {
		fatal(errors.New("pass --fixture <name> or --all"))
	}
	return opts
}

func runRender3(repoRoot string, opts options) ([]report, error) {
	fixtures, err := selectedRender3Fixtures(repoRoot, opts)
	if err != nil {
		return nil, err
	}

	reports := make([]report, 0, len(fixtures))
	for _, fixture := range fixtures {
		rep, err := compareRender3Fixture(repoRoot, opts, fixture)
		if err != nil {
			return reports, err
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

func selectedRender3Fixtures(repoRoot string, opts options) ([]string, error) {
	if !opts.all {
		return []string{opts.fixture}, nil
	}

	root := filepath.Join(repoRoot, "angular-packages", "compiler_cli", "integration_tests", "testdata", "render3_ir")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var fixtures []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !opts.includeLegacy && !strings.HasPrefix(entry.Name(), "parity_") {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "project", "tsconfig.json")); err == nil {
			fixtures = append(fixtures, entry.Name())
		}
	}
	sort.Strings(fixtures)
	return fixtures, nil
}

func compareRender3Fixture(repoRoot string, opts options, fixture string) (report, error) {
	baseOut := filepath.Join(opts.outDir, "render3", fixture)
	_ = os.RemoveAll(baseOut)
	if err := os.MkdirAll(baseOut, 0o755); err != nil {
		return report{}, err
	}

	rep := newReport(opts, "render3", fixture)
	fixtureProject := filepath.Join(repoRoot, "angular-packages", "compiler_cli", "integration_tests", "testdata", "render3_ir", fixture, "project")
	if _, err := os.Stat(filepath.Join(fixtureProject, "tsconfig.json")); err != nil {
		rep.Status = statusCompileFailed
		rep.Errors = append(rep.Errors, fmt.Sprintf("fixture project not found: %v", err))
		_ = writeReport(baseOut, rep)
		return rep, nil
	}

	goProject := filepath.Join(baseOut, "work", "go-project")
	ngtscProject := filepath.Join(baseOut, "work", "ngtsc-project")
	if err := copyProject(fixtureProject, goProject, repoRoot); err != nil {
		return report{}, err
	}
	if err := copyProject(fixtureProject, ngtscProject, repoRoot); err != nil {
		return report{}, err
	}

	goResult := runCommand(goProject, opts.goCompiler, "-p", filepath.Join(goProject, "tsconfig.json"))
	writeCommandOutput(filepath.Join(baseOut, "go.compile.stdout.txt"), goResult.stdout)
	writeCommandOutput(filepath.Join(baseOut, "go.compile.stderr.txt"), goResult.stderr)

	ngcResult := runCommand(ngtscProject, "node", opts.ngcPath, "-p", filepath.Join(ngtscProject, "tsconfig.json"))
	writeCommandOutput(filepath.Join(baseOut, "ngtsc.compile.stdout.txt"), ngcResult.stdout)
	writeCommandOutput(filepath.Join(baseOut, "ngtsc.compile.stderr.txt"), ngcResult.stderr)

	if goResult.err != nil || ngcResult.err != nil {
		rep.Status = statusCompileFailed
		if goResult.err != nil {
			rep.Errors = append(rep.Errors, fmt.Sprintf("go compile failed: %v", goResult.err))
		}
		if ngcResult.err != nil {
			rep.Errors = append(rep.Errors, fmt.Sprintf("ngtsc compile failed: %v", ngcResult.err))
		}
		_ = copyDir(filepath.Join(goProject, "out"), filepath.Join(baseOut, "go"))
		_ = copyDir(filepath.Join(ngtscProject, "out"), filepath.Join(baseOut, "ngtsc"))
		_ = writeReport(baseOut, rep)
		return rep, nil
	}

	goOut := filepath.Join(goProject, "out")
	ngtscOut := filepath.Join(ngtscProject, "out")
	if err := copyDir(goOut, filepath.Join(baseOut, "go")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return report{}, err
	}
	if err := copyDir(ngtscOut, filepath.Join(baseOut, "ngtsc")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return report{}, err
	}

	goFiles, err := collectEmitFiles(goOut)
	if err != nil {
		return report{}, err
	}
	ngtscFiles, err := collectEmitFiles(ngtscOut)
	if err != nil {
		return report{}, err
	}

	paths := unionKeys(goFiles, ngtscFiles)
	var rawDiff bytes.Buffer
	var normalizedDiff bytes.Buffer

	for _, rel := range paths {
		fr := fileReport{Path: filepath.ToSlash(rel)}
		goPath, hasGo := goFiles[rel]
		ngtscPath, hasNgtsc := ngtscFiles[rel]

		switch {
		case !hasGo:
			fr.Status = statusMissingGoOutput
		case !hasNgtsc:
			fr.Status = statusMissingNgtscOutput
		default:
			goText, err := os.ReadFile(goPath)
			if err != nil {
				return report{}, err
			}
			ngtscText, err := os.ReadFile(ngtscPath)
			if err != nil {
				return report{}, err
			}

			if string(goText) == string(ngtscText) {
				fr.Status = statusExact
			} else {
				var goNormalized parity.NormalizedJS
				var ngtscNormalized parity.NormalizedJS
				if strings.HasSuffix(rel, ".d.ts") {
					goNormText := strings.ReplaceAll(string(goText), "\r\n", "\n")
					ngtscNormText := strings.ReplaceAll(string(ngtscText), "\r\n", "\n")
					goNormalized = parity.NormalizedJS{Text: goNormText, Rules: []string{"line-endings"}}
					ngtscNormalized = parity.NormalizedJS{Text: ngtscNormText, Rules: []string{"line-endings"}}
				} else {
					goNormalized = parity.NormalizeGoldenJSWithRules(string(goText))
					ngtscNormalized = parity.NormalizeGoldenJSWithRules(string(ngtscText))
				}
				fr.Normalizations = parity.MergeRuleNames(goNormalized.Rules, ngtscNormalized.Rules)
				writeText(filepath.Join(baseOut, "normalized", "go", rel), goNormalized.Text)
				writeText(filepath.Join(baseOut, "normalized", "ngtsc", rel), ngtscNormalized.Text)
				rawDiff.WriteString(renderDiff(rel, string(ngtscText), string(goText)))
				if goNormalized.Text == ngtscNormalized.Text {
					fr.Status = statusNormalized
				} else {
					fr.Status = statusSemantic
					normalizedDiff.WriteString(renderDiff(rel, ngtscNormalized.Text, goNormalized.Text))
				}
			}
		}
		rep.Files = append(rep.Files, fr)
		rep.Status = worseStatus(rep.Status, fr.Status)
	}

	if rep.Status == "" {
		rep.Status = statusExact
	}
	writeCommandOutput(filepath.Join(baseOut, "diff.raw.patch"), rawDiff.String())
	writeCommandOutput(filepath.Join(baseOut, "diff.normalized.patch"), normalizedDiff.String())
	if err := writeReport(baseOut, rep); err != nil {
		return report{}, err
	}
	return rep, nil
}

func runLinker(repoRoot string, opts options) ([]report, error) {
	files, err := selectedLinkerFiles(repoRoot, opts)
	if err != nil {
		return nil, err
	}

	reports := make([]report, 0, len(files))
	for _, input := range files {
		fixture := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
		if opts.fixture != "" && !opts.all {
			fixture = opts.fixture
		}
		rep, err := compareLinkerFile(repoRoot, opts, fixture, input)
		if err != nil {
			return reports, err
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

func selectedLinkerFiles(repoRoot string, opts options) ([]string, error) {
	if !opts.all {
		if opts.fixture == "" {
			return nil, errors.New("linker mode requires --fixture <path-or-name> or --all")
		}
		if filepath.IsAbs(opts.fixture) {
			return []string{opts.fixture}, nil
		}
		repoRelative := filepath.Join(repoRoot, opts.fixture)
		if _, err := os.Stat(repoRelative); err == nil {
			return []string{repoRelative}, nil
		}
		return []string{filepath.Join(repoRoot, "angular-packages", "new-demo-app", "node_modules", opts.fixture)}, nil
	}

	nodeModules := filepath.Join(repoRoot, "angular-packages", "new-demo-app", "node_modules")
	relFiles := []string{
		"@angular/common/fesm2022/common.mjs",
		"@angular/forms/fesm2022/forms.mjs",
		"@angular/router/fesm2022/router.mjs",
		"primeng/fesm2022/primeng-button.mjs",
		"primeng/fesm2022/primeng-table.mjs",
		"primeng/fesm2022/primeng-select.mjs",
		"primeng/fesm2022/primeng-menu.mjs",
		"primeng/fesm2022/primeng-datepicker.mjs",
	}
	files := make([]string, 0, len(relFiles))
	for _, rel := range relFiles {
		files = append(files, filepath.Join(nodeModules, rel))
	}
	return files, nil
}

func compareLinkerFile(repoRoot string, opts options, fixture string, input string) (report, error) {
	safeFixture := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "@", "").Replace(fixture)
	baseOut := filepath.Join(opts.outDir, "linker", safeFixture)
	_ = os.RemoveAll(baseOut)
	if err := os.MkdirAll(baseOut, 0o755); err != nil {
		return report{}, err
	}

	rep := newReport(opts, "linker", fixture)
	goOut := filepath.Join(baseOut, "go.mjs")
	ngtscOut := filepath.Join(baseOut, "ngtsc.mjs")
	goResult := runCommand(repoRoot, opts.goCompiler, "--link", input)
	writeCommandOutput(filepath.Join(baseOut, "go.link.stdout.txt"), goResult.stdout)
	writeCommandOutput(filepath.Join(baseOut, "go.link.stderr.txt"), goResult.stderr)
	if goResult.err != nil {
		rep.Status = statusCompileFailed
		rep.Errors = append(rep.Errors, fmt.Sprintf("go linker failed: %v", goResult.err))
		_ = writeReport(baseOut, rep)
		return rep, nil
	}
	if err := os.WriteFile(goOut, []byte(goResult.stdout), 0o644); err != nil {
		return report{}, err
	}

	compareScript := filepath.Join(repoRoot, "angular-packages", "compiler_cli", "integration_tests", "compare_linker_output.mjs")
	compareResult := runCommand(filepath.Dir(compareScript), "node", compareScript, input, goOut, ngtscOut)
	writeCommandOutput(filepath.Join(baseOut, "compare.stdout.txt"), compareResult.stdout)
	writeCommandOutput(filepath.Join(baseOut, "compare.stderr.txt"), compareResult.stderr)

	if compareResult.err != nil {
		rep.Status = statusSemantic
		rep.Errors = append(rep.Errors, fmt.Sprintf("linker compare failed: %v", compareResult.err))
	} else {
		goText, _ := os.ReadFile(goOut)
		ngtscText, _ := os.ReadFile(ngtscOut)
		if len(ngtscText) > 0 && string(goText) == string(ngtscText) {
			rep.Status = statusExact
		} else {
			rep.Status = statusNormalized
		}
	}

	rep.Files = []fileReport{{Path: filepath.ToSlash(input), Status: rep.Status}}
	_ = writeReport(baseOut, rep)
	return rep, nil
}

func newReport(opts options, kind string, fixture string) report {
	return report{
		Baseline: baselineReport{
			Angular:    opts.angularVersion,
			GoCompiler: opts.goCompiler,
			Ngc:        opts.ngcPath,
		},
		Kind:    kind,
		Fixture: fixture,
		Status:  statusExact,
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", errors.New("could not find repo root containing go.mod")
		}
		wd = parent
	}
}

func resolveGoCompiler(repoRoot string, compiler string) string {
	if compiler == "go-ngc" {
		local := filepath.Join(repoRoot, "go-ngc")
		if _, err := os.Stat(local); err == nil {
			return local
		}
	}
	if filepath.IsAbs(compiler) {
		return compiler
	}
	if strings.Contains(compiler, string(filepath.Separator)) || strings.Contains(compiler, "/") {
		return filepath.Join(repoRoot, compiler)
	}
	return compiler
}

func copyProject(src string, dst string, repoRoot string) error {
	if err := copyDir(src, dst); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(dst, "out")); err != nil {
		return err
	}
	nodeModules := filepath.Join(repoRoot, "angular-packages", "new-demo-app", "node_modules")
	if st, err := os.Stat(nodeModules); err == nil && st.IsDir() {
		linkPath := filepath.Join(dst, "node_modules")
		_ = os.Remove(linkPath)
		if err := os.Symlink(nodeModules, linkPath); err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func copyDir(src string, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func runCommand(cwd string, name string, args ...string) commandResult {
	cmd := exec.Command(name, args...)
	cmd.Dir = cwd
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func collectEmitFiles(root string) (map[string]string, error) {
	files := map[string]string{}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".d.ts") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[rel] = path
		return nil
	})
	return files, err
}

func unionKeys(left map[string]string, right map[string]string) []string {
	seen := map[string]bool{}
	for key := range left {
		seen[key] = true
	}
	for key := range right {
		seen[key] = true
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func worseStatus(current status, next status) status {
	if rankStatus(next) > rankStatus(current) {
		return next
	}
	return current
}

func rankStatus(s status) int {
	switch s {
	case statusCompileFailed:
		return 6
	case statusMissingGoOutput, statusMissingNgtscOutput:
		return 5
	case statusSemantic:
		return 4
	case statusNormalized:
		return 3
	case statusExact:
		return 1
	default:
		return 0
	}
}

func renderDiff(rel string, expected string, actual string) string {
	if expected == actual {
		return ""
	}
	return fmt.Sprintf("--- ngtsc/%s\n+++ go/%s\n@@\n%s\n", filepath.ToSlash(rel), filepath.ToSlash(rel), lineDiff(expected, actual))
}

func lineDiff(expected string, actual string) string {
	expectedLines := strings.Split(strings.TrimSuffix(expected, "\n"), "\n")
	actualLines := strings.Split(strings.TrimSuffix(actual, "\n"), "\n")
	max := len(expectedLines)
	if len(actualLines) > max {
		max = len(actualLines)
	}
	var b strings.Builder
	for i := 0; i < max; i++ {
		var exp, act string
		if i < len(expectedLines) {
			exp = expectedLines[i]
		}
		if i < len(actualLines) {
			act = actualLines[i]
		}
		if exp == act {
			continue
		}
		if i < len(expectedLines) {
			fmt.Fprintf(&b, "-%04d %s\n", i+1, exp)
		}
		if i < len(actualLines) {
			fmt.Fprintf(&b, "+%04d %s\n", i+1, act)
		}
	}
	return b.String()
}

func writeText(path string, text string) {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(text), 0o644)
}

func writeCommandOutput(path string, text string) {
	if text == "" {
		return
	}
	writeText(path, text)
}

func writeReport(baseOut string, rep report) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(baseOut, "report.json"), append(data, '\n'), 0o644)
}

func printSummary(reports []report) {
	counts := map[status]int{}
	ruleCounts := map[string]int{}
	for _, rep := range reports {
		counts[rep.Status]++
		fmt.Printf("%s/%s: %s\n", rep.Kind, rep.Fixture, rep.Status)
		for _, msg := range rep.Errors {
			fmt.Printf("  %s\n", msg)
		}
		for _, file := range rep.Files {
			for _, rule := range file.Normalizations {
				ruleCounts[rule]++
			}
		}
	}
	fmt.Printf("summary: exact=%d normalized=%d semantic=%d missing-go=%d missing-ngtsc=%d compile-failed=%d generated-at=%s\n",
		counts[statusExact],
		counts[statusNormalized],
		counts[statusSemantic],
		counts[statusMissingGoOutput],
		counts[statusMissingNgtscOutput],
		counts[statusCompileFailed],
		time.Now().Format(time.RFC3339),
	)
	if len(ruleCounts) > 0 {
		rules := make([]string, 0, len(ruleCounts))
		for rule := range ruleCounts {
			rules = append(rules, rule)
		}
		sort.Strings(rules)
		fmt.Print("normalization-rules:")
		for _, rule := range rules {
			fmt.Printf(" %s=%d", rule, ruleCounts[rule])
		}
		fmt.Println()
	}
}

func shouldFail(reports []report, opts options) bool {
	if !opts.strict {
		return false
	}
	for _, rep := range reports {
		switch rep.Status {
		case statusCompileFailed, statusMissingGoOutput, statusMissingNgtscOutput, statusSemantic:
			return true
		case statusNormalized:
			if opts.failNormalized {
				return true
			}
		}
	}
	return false
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
