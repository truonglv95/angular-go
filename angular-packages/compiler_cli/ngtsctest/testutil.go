// Package ngtsctest provides shared test utilities for ngtsc Go tests,
// mirroring the makeProgram/getDeclaration test helpers from the TypeScript ngtsc test suite.
package ngtsctest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/vfs/vfstest"
)

// ProgramFile represents an in-memory source file for program creation.
type ProgramFile struct {
	Name     string
	Contents string
}

// ProgramResult holds the program and its associated type checker.
type ProgramResult struct {
	Program *compiler.Program
	Checker *checker.Checker
	release func()
}

// Release releases the type checker back to the pool.
func (r *ProgramResult) Release() {
	if r.release != nil {
		r.release()
	}
}

// GetSourceFile returns the source file with the given name, or nil.
func (r *ProgramResult) GetSourceFile(name string) *ast.SourceFile {
	return r.Program.GetSourceFile(name)
}

// MakeProgram creates an in-memory TypeScript program from the given files.
// The entry point is the first file. All files are added to a virtual filesystem.
func MakeProgram(t *testing.T, files []ProgramFile) *ProgramResult {
	t.Helper()
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test that requires type checker")
	}

	mapFS := make(fstest.MapFS)
	for _, f := range files {
		mapFS[f.Name] = &fstest.MapFile{Data: []byte(f.Contents)}
	}

	fs := vfstest.FromMap[any](nil, false /*useCaseSensitiveFileNames*/)
	fs = bundled.WrapFS(fs)
	for name, content := range mapFS {
		_ = fs.WriteFile(name, string(content.Data))
	}

	opts := &core.CompilerOptions{
		Target:                 core.ScriptTargetES2015,
		Module:                 core.ModuleKindCommonJS,
		ExperimentalDecorators: core.TSTrue,
	}

	// Use ALL non-lib files as root names (mirrors ngtsc makeProgram which includes all TestFiles).
	// This ensures files with external dependencies are included in the program.
	var rootNames []string
	for _, f := range files {
		rootNames = append(rootNames, f.Name)
	}

	program := compiler.NewProgram(compiler.ProgramOptions{
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &core.ParsedOptions{
				FileNames:       rootNames,
				CompilerOptions: opts,
			},
		},
		Host: compiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil),
	})

	chk, release := program.GetTypeChecker(context.Background())
	return &ProgramResult{
		Program: program,
		Checker: chk,
		release: release,
	}
}

// GetSourceFileByName returns the source file whose filename ends with the given suffix.
// This handles path normalization differences across platforms.
func (r *ProgramResult) GetSourceFileByName(nameSuffix string) *ast.SourceFile {
	for _, sf := range r.Program.SourceFiles() {
		name := sf.FileName()
		if strings.HasSuffix(name, nameSuffix) || name == nameSuffix {
			return sf
		}
	}
	return nil
}

// RequireSourceFileByName returns the source file by name suffix or fails the test.
func RequireSourceFileByName(t *testing.T, program *ProgramResult, nameSuffix string) *ast.SourceFile {
	t.Helper()
	sf := program.GetSourceFileByName(nameSuffix)
	if sf == nil {
		var names []string
		for _, f := range program.Program.SourceFiles() {
			names = append(names, f.FileName())
		}
		t.Fatalf("Source file %q not found in program. Available: %v", nameSuffix, names)
	}
	return sf
}

// GetDeclaration finds a declaration with the given name in the source file and
// applies the given predicate to match the right kind. Returns nil if not found.
func GetDeclaration(sf *ast.SourceFile, name string, pred func(*ast.Node) bool) *ast.Node {
	if sf == nil {
		return nil
	}
	var result *ast.Node
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		if pred(node) {
			n := node.Name()
			if n != nil && ast.IsIdentifier(n) && n.AsIdentifier().Text == name {
				result = node
				return true
			}
		}
		return node.ForEachChild(walk)
	}
	sf.AsNode().ForEachChild(walk)
	return result
}

// FindNamedClassDeclaration finds a class declaration with the given name in a source file.
func FindNamedClassDeclaration(sf *ast.SourceFile, name string) *ast.Node {
	return GetDeclaration(sf, name, ast.IsClassDeclaration)
}

// FindVariableDeclaration finds a variable declaration with the given name.
func FindVariableDeclaration(sf *ast.SourceFile, name string) *ast.Node {
	return GetDeclaration(sf, name, ast.IsVariableDeclaration)
}

// FindFunctionDeclaration finds a function declaration with the given name.
func FindFunctionDeclaration(sf *ast.SourceFile, name string) *ast.Node {
	return GetDeclaration(sf, name, ast.IsFunctionDeclaration)
}

// RequireSourceFile returns the source file or fails the test.
// It matches by name suffix for cross-platform path compatibility.
func RequireSourceFile(t *testing.T, program *ProgramResult, name string) *ast.SourceFile {
	t.Helper()
	sf := program.GetSourceFileByName(name)
	if sf == nil {
		var names []string
		for _, f := range program.Program.SourceFiles() {
			names = append(names, f.FileName())
		}
		t.Fatalf("Source file %q not found in program. Available: %v", name, names)
	}
	return sf
}

// RequireDeclaration finds a declaration and fails the test if not found.
func RequireDeclaration(t *testing.T, sf *ast.SourceFile, name string, pred func(*ast.Node) bool) *ast.Node {
	t.Helper()
	decl := GetDeclaration(sf, name, pred)
	if decl == nil {
		t.Fatalf("Declaration %q not found in %s", name, sf.FileName())
	}
	return decl
}

// ArgExpressionToString converts an identifier or property access expression to a string.
// Matches the TypeScript argExpressionToString helper in the spec tests.
// Also handles QualifiedName nodes that appear in type positions (e.g. i1.Bar).
func ArgExpressionToString(node *ast.Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("node is nil")
	}
	if ast.IsIdentifier(node) {
		return node.AsIdentifier().Text, nil
	}
	if ast.IsPropertyAccessExpression(node) {
		pae := node.AsPropertyAccessExpression()
		left, err := ArgExpressionToString(pae.Expression)
		if err != nil {
			return "", err
		}
		return left + "." + pae.Name().AsIdentifier().Text, nil
	}
	// Handle QualifiedName nodes that appear in type positions (e.g. namespace.Type).
	if ast.IsQualifiedName(node) {
		qn := node.AsQualifiedName()
		left, err := ArgExpressionToString(qn.Left)
		if err != nil {
			return "", err
		}
		return left + "." + qn.Right.Text(), nil
	}
	return "", fmt.Errorf("unexpected node kind: %v", node.Kind)
}
