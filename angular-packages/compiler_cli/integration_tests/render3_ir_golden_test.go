package integration_tests

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli"
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
				"out/app/card.component.js",
				"out/app/app.component.js",
			},
		},
		{
			name: "host_directive",
			files: []string{
				"out/app/track.directive.js",
				"out/app/host.component.js",
			},
		},
		{
			name: "pipe_interpolation",
			files: []string{
				"out/app/shout.pipe.js",
				"out/app/pipe.component.js",
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
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
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

	actualText := normalizeGoldenJS(string(actual))
	expectedText := normalizeGoldenJS(string(expected))
	if actualText != expectedText {
		t.Fatalf("%s golden mismatch\nactual:\n%s\nexpected:\n%s", rel, actualText, expectedText)
	}
}

func normalizeGoldenJS(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = regexp.MustCompile(`filePath: "[^"]*/(app/[^"]+)"`).ReplaceAllString(text, `filePath: "<PROJECT>/$1"`)
	return strings.TrimSpace(text) + "\n"
}

func diagnosticsText(diags any) string {
	return fmt.Sprintf("%v", diags)
}
