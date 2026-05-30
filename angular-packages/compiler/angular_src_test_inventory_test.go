package compiler

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type angularSourceSpecCase struct {
	File string
	Line int
	Name string
}

func TestAngularCompilerSourceSpecInventory(t *testing.T) {
	cases, err := collectAngularSourceSpecCases("../angular-src/packages/compiler/test")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 1500 {
		t.Fatalf("found %d Angular compiler source spec cases, want at least 1500", len(cases))
	}

	if os.Getenv("ANGULAR_COMPILER_TEST_INVENTORY") != "1" {
		t.Logf("tracked %d source spec cases from angular-src/packages/compiler/test; set ANGULAR_COMPILER_TEST_INVENTORY=1 to enumerate them as skipped Go subtests", len(cases))
		return
	}

	for _, tc := range cases {
		tc := tc
		t.Run(filepath.ToSlash(tc.File)+":"+tc.Name, func(t *testing.T) {
			t.Skipf("pending executable Go port from %s:%d", tc.File, tc.Line)
		})
	}
}

func collectAngularSourceSpecCases(root string) ([]angularSourceSpecCase, error) {
	specCall := regexp.MustCompile(`\b(?:f|x)?it\s*\(\s*(?:'([^']*)'|"([^"]*)"|` + "`([^`]*)`" + `)`)
	var cases []angularSourceSpecCase
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, "_spec.ts") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		lineStarts := []int{0}
		for i, r := range text {
			if r == '\n' {
				lineStarts = append(lineStarts, i+1)
			}
		}
		for _, match := range specCall.FindAllStringSubmatchIndex(text, -1) {
			name := ""
			for group := 1; group <= 3; group++ {
				start := match[group*2]
				end := match[group*2+1]
				if start >= 0 {
					name = text[start:end]
					break
				}
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			cases = append(cases, angularSourceSpecCase{
				File: filepath.ToSlash(rel),
				Line: lineForOffset(lineStarts, match[0]),
				Name: name,
			})
		}
		return nil
	})
	return cases, err
}

func lineForOffset(lineStarts []int, offset int) int {
	line := 1
	for i, start := range lineStarts {
		if start > offset {
			break
		}
		line = i + 1
	}
	return line
}
