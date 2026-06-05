package integration_tests

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
)

func TestMemoryOutputMatchesDiskOutput(t *testing.T) {
	fixtureRoot := filepath.Join("testdata", "render3_ir", "parity_ng_if")

	projectRoot := copyFixtureProject(t, filepath.Join(fixtureRoot, "project"))
	if err := os.RemoveAll(filepath.Join(projectRoot, "out")); err != nil {
		t.Fatalf("clean memory out dir: %v", err)
	}
	memoryConfig := compiler_cli.ReadConfiguration(filepath.Join(projectRoot, "tsconfig.json"))
	memoryConfig.Write = false
	memoryResult := compiler_cli.PerformCompilation(memoryConfig)
	if memoryResult.Status != tsc.ExitStatusSuccess || len(memoryResult.Diagnostics) > 0 {
		t.Fatalf("memory compilation failed: status=%v diagnostics=%s", memoryResult.Status, diagnosticsText(memoryResult.Diagnostics))
	}

	memoryByRelPath := make(map[string]string)
	for _, output := range memoryResult.Outputs {
		rel, err := filepath.Rel(projectRoot, output.Path)
		if err != nil {
			t.Fatalf("rel memory output path %s: %v", output.Path, err)
		}
		memoryByRelPath[filepath.ToSlash(rel)] = output.Text
	}

	const relJs = "out/app/app.component.js"
	memoryText, ok := memoryByRelPath[relJs]
	if !ok {
		t.Fatalf("memory output missing %s; got outputs %v", relJs, sortedKeys(memoryByRelPath))
	}
	if _, err := os.Stat(filepath.Join(projectRoot, filepath.FromSlash(relJs))); err == nil {
		t.Fatalf("memory compilation unexpectedly wrote %s to disk", relJs)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat memory disk output %s: %v", relJs, err)
	}

	diskConfig := compiler_cli.ReadConfiguration(filepath.Join(projectRoot, "tsconfig.json"))
	diskConfig.Write = true
	diskResult := compiler_cli.PerformCompilation(diskConfig)
	if diskResult.Status != tsc.ExitStatusSuccess || len(diskResult.Diagnostics) > 0 {
		t.Fatalf("disk compilation failed: status=%v diagnostics=%s", diskResult.Status, diagnosticsText(diskResult.Diagnostics))
	}

	diskText, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(relJs)))
	if err != nil {
		t.Fatalf("read disk output %s: %v", relJs, err)
	}
	if memoryText != string(diskText) {
		t.Fatalf("memory output mismatch for %s\nMEMORY:\n%s\nDISK:\n%s", relJs, memoryText, string(diskText))
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
