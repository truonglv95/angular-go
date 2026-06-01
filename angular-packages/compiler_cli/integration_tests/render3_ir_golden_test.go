package integration_tests

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/integration_tests/parity"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
)

func TestRender3IRGolden(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test that requires type checker")
	}

	tests := []struct {
		name  string
		files []string
	}{
		{
			name: "basic_projection",
			files: []string{
				"out/card.component.js",
				"out/app.component.js",
			},
		},
		{
			name: "host_directive",
			files: []string{
				"out/track.directive.js",
				"out/host.component.js",
			},
		},
		{
			name: "pipe_interpolation",
			files: []string{
				"out/shout.pipe.js",
				"out/pipe.component.js",
			},
		},
		{
			name:  "parity_ng_if",
			files: []string{"out/app/app.component.js"},
		},
		{
			name:  "parity_ng_for",
			files: []string{"out/app/app.component.js"},
		},
		{
			name:  "parity_ng_template_outlet",
			files: []string{"out/app/app.component.js"},
		},
		{
			name:  "parity_template_refs",
			files: []string{"out/app/app.component.js"},
		},
		{
			name: "parity_nested_templates",
			files: []string{
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_template_outlet_structural_refs",
			files: []string{
				"out/app/app.component.js",
			},
		},
		{
			name:  "parity_nested_views",
			files: []string{"out/app/app.component.js"},
		},
		{
			name:  "parity_listeners",
			files: []string{"out/app/app.component.js"},
		},
		{
			name: "parity_forms",
			files: []string{
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_forms_form_control",
			files: []string{
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_forms_form_control_name",
			files: []string{
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_forms_form_field",
			files: []string{
				"out/app/app.component.js",
			},
		},
		{
			name:  "parity_pipes",
			files: []string{"out/app/app.component.js"},
		},
		{
			name:  "parity_animations",
			files: []string{"out/app/app.component.js"},
		},
		{
			name: "parity_defer",
			files: []string{
				"out/app/heavy.component.js",
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_defer_triggers",
			files: []string{
				"out/app/heavy.component.js",
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_defer_blocks",
			files: []string{
				"out/app/heavy.component.js",
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_defer_nested",
			files: []string{
				"out/app/heavy.component.js",
				"out/app/app.component.js",
			},
		},
		{
			name: "parity_defer_alias_barrel",
			files: []string{
				"out/app/deferred.js",
				"out/app/heavy.component.js",
				"out/app/app.component.js",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixtureRoot := filepath.Join("testdata", "render3_ir", tt.name)
			projectRoot := copyFixtureProject(t, filepath.Join(fixtureRoot, "project"))

			config := compiler_cli.ReadConfiguration(filepath.Join(projectRoot, "tsconfig.json"))
			if len(config.Errors) > 0 {
				t.Fatalf("config diagnostics:\n%s", diagnosticsText(config.Errors))
			}

			result := compiler_cli.PerformCompilation(config)
			if result.Status != tsc.ExitStatusSuccess {
				t.Fatalf("compilation status = %v, diagnostics:\n%s", result.Status, diagnosticsText(result.Diagnostics))
			}
			if len(result.Diagnostics) > 0 {
				t.Fatalf("unexpected diagnostics:\n%s", diagnosticsText(result.Diagnostics))
			}

			for _, file := range tt.files {
				assertGoldenFile(t, projectRoot, fixtureRoot, file)
			}
		})
	}
}

func copyFixtureProject(t *testing.T, src string) string {
	t.Helper()

	dst := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "node_modules" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			// Skip symlinks
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture project: %v", err)
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	nodeModules := filepath.Join(repoRoot, "angular-packages", "new-demo-app", "node_modules")
	if st, err := os.Stat(nodeModules); err == nil && st.IsDir() {
		if err := os.Symlink(nodeModules, filepath.Join(dst, "node_modules")); err != nil {
			t.Fatalf("link fixture node_modules: %v", err)
		}
	}

	return dst
}

func assertGoldenFile(t *testing.T, projectRoot string, fixtureRoot string, rel string) {
	t.Helper()

	actualPath := filepath.Join(projectRoot, filepath.FromSlash(rel))
	expectedPath := filepath.Join(fixtureRoot, "golden", filepath.FromSlash(rel))

	actual, err := os.ReadFile(actualPath)
	if err != nil {
		t.Fatalf("read actual %s: %v", actualPath, err)
	}
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", expectedPath, err)
	}

	actualText := parity.NormalizeGoldenJS(string(actual))
	expectedText := parity.NormalizeGoldenJS(string(expected))
	if actualText != expectedText {
		t.Fatalf("%s golden mismatch\nactual:\n%s\nexpected:\n%s", rel, actualText, expectedText)
	}
}

func diagnosticsText(diags any) string {
	return fmt.Sprintf("%v", diags)
}
