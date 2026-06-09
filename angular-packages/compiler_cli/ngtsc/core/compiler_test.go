package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createProgramWithOSFiles(t *testing.T, files []ngtsctest.ProgramFile) (*ngtsctest.ProgramResult, func()) {
	// Write all files to OS filesystem first
	for _, f := range files {
		dir := filepath.Dir(f.Name)
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(f.Name, []byte(f.Contents), 0644)
		require.NoError(t, err)
	}

	result := ngtsctest.MakeProgram(t, files)

	cleanup := func() {
		result.Release()
		// Clean up files written to OS filesystem
		for _, f := range files {
			_ = os.Remove(f.Name)
		}
	}

	return result, cleanup
}

func TestNgCompiler_GetDiagnosticsForFile(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	cmpTsPath := filepath.Join(tempDir, "cmp.ts")
	templateHtmlPath := filepath.Join(tempDir, "template.html")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: cmpTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'test-cmp',
					standalone: true,
					templateUrl: './template.html',
				})
				export class Cmp {}
			`,
		},
		{
			Name: templateHtmlPath,
			Contents: `{{does_not_exist.foo}}`,
		},
	}

	result, cleanup := createProgramWithOSFiles(t, files)
	defer cleanup()

	options := core.NgCompilerOptions{
		StrictTemplates: true,
	}
	compiler, err := core.NewNgCompiler(result.Program, options, nil)
	require.NoError(t, err)

	sf := result.GetSourceFileByName("cmp.ts")
	require.NotNil(t, sf)

	diags := compiler.GetDiagnosticsForFile(sf, nil)
	for _, d := range diags {
		msg := ""
		if len(d.MessageArgs()) > 0 {
			msg = d.MessageArgs()[0]
		}
		t.Logf("Diagnostic code: %d, message: %s", d.Code(), msg)
	}
	require.Len(t, diags, 1)
}

func TestNgCompiler_GetComponentsWithTemplateFile(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	cmpATsPath := filepath.Join(tempDir, "cmp-a.ts")
	cmpBTsPath := filepath.Join(tempDir, "cmp-b.ts")
	cmpCTsPath := filepath.Join(tempDir, "cmp-c.ts")
	templateHtmlPath := filepath.Join(tempDir, "template.html")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: cmpATsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'cmp-a',
					templateUrl: './template.html',
				})
				export class CmpA {}
			`,
		},
		{
			Name: cmpBTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'cmp-b',
					template: 'CmpB template inline',
				})
				export class CmpB {}
			`,
		},
		{
			Name: cmpCTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'cmp-c',
					templateUrl: './template.html',
				})
				export class CmpC {}
			`,
		},
		{
			Name: templateHtmlPath,
			Contents: `This is template`,
		},
	}

	result, cleanup := createProgramWithOSFiles(t, files)
	defer cleanup()

	options := core.NgCompilerOptions{}
	compiler, err := core.NewNgCompiler(result.Program, options, nil)
	require.NoError(t, err)

	compiler.AnalyzeSync()

	components := compiler.GetComponentsWithTemplateFile(templateHtmlPath)
	require.Len(t, components, 2)

	var names []string
	for _, c := range components {
		if c.Kind == ast.KindClassDeclaration {
			names = append(names, c.AsClassDeclaration().Name().AsIdentifier().Text)
		}
	}
	assert.Contains(t, names, "CmpA")
	assert.Contains(t, names, "CmpC")
}

func TestNgCompiler_GetComponentsWithStyleFile(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	cmpATsPath := filepath.Join(tempDir, "cmp-a.ts")
	cmpBTsPath := filepath.Join(tempDir, "cmp-b.ts")
	cmpCTsPath := filepath.Join(tempDir, "cmp-c.ts")
	stylePath := filepath.Join(tempDir, "style.css")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: cmpATsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'cmp-a',
					template: '',
					styleUrls: ['./style.css'],
				})
				export class CmpA {}
			`,
		},
		{
			Name: cmpBTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'cmp-b',
					template: '',
					styles: ['body { color: red; }'],
				})
				export class CmpB {}
			`,
		},
		{
			Name: cmpCTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'cmp-c',
					template: '',
					styleUrls: ['./style.css'],
				})
				export class CmpC {}
			`,
		},
		{
			Name: stylePath,
			Contents: `div {}`,
		},
	}

	result, cleanup := createProgramWithOSFiles(t, files)
	defer cleanup()

	options := core.NgCompilerOptions{}
	compiler, err := core.NewNgCompiler(result.Program, options, nil)
	require.NoError(t, err)

	compiler.AnalyzeSync()

	components := compiler.GetComponentsWithStyleFile(stylePath)
	require.Len(t, components, 2)

	var names []string
	for _, c := range components {
		if c.Kind == ast.KindClassDeclaration {
			names = append(names, c.AsClassDeclaration().Name().AsIdentifier().Text)
		}
	}
	assert.Contains(t, names, "CmpA")
	assert.Contains(t, names, "CmpC")
}

func TestNgCompiler_GetDirectiveResources(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	cmpATsPath := filepath.Join(tempDir, "cmp-a.ts")
	templateHtmlPath := filepath.Join(tempDir, "template.html")
	stylePath := filepath.Join(tempDir, "style.css")
	style2Path := filepath.Join(tempDir, "style2.css")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: cmpATsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'cmp-a',
					templateUrl: './template.html',
					styleUrls: ['./style.css', './style2.css'],
				})
				export class CmpA {}
			`,
		},
		{
			Name: templateHtmlPath,
			Contents: `This is the template`,
		},
		{
			Name: stylePath,
			Contents: `/* style 1 */`,
		},
		{
			Name: style2Path,
			Contents: `/* style 2 */`,
		},
	}

	result, cleanup := createProgramWithOSFiles(t, files)
	defer cleanup()

	options := core.NgCompilerOptions{}
	compiler, err := core.NewNgCompiler(result.Program, options, nil)
	require.NoError(t, err)

	compiler.AnalyzeSync()

	sf := result.GetSourceFileByName("cmp-a.ts")
	require.NotNil(t, sf)
	classDecl := ngtsctest.FindNamedClassDeclaration(sf, "CmpA")
	require.NotNil(t, classDecl)

	resources := compiler.GetDirectiveResources(classDecl.AsNode())
	require.NotNil(t, resources)
	assert.Equal(t, templateHtmlPath, resources.Template.Path)
	require.Len(t, resources.Styles, 2)

	var stylePaths []string
	for _, s := range resources.Styles {
		stylePaths = append(stylePaths, s.Path)
	}
	assert.Contains(t, stylePaths, stylePath)
	assert.Contains(t, stylePaths, style2Path)
}

func TestNgCompiler_GetDirectiveResources_InvalidUrls(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	cmpATsPath := filepath.Join(tempDir, "cmp-a.ts")
	stylePath := filepath.Join(tempDir, "style.css")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: cmpATsPath,
			Contents: `
				import {Component} from '@angular/core';
				const STYLES = ['./style.css'];
				@Component({
					selector: 'cmp-a',
					template: '',
					styleUrls: STYLES,
				})
				export class CmpA {}
			`,
		},
		{
			Name: stylePath,
			Contents: `/* style 1 */`,
		},
	}

	result, cleanup := createProgramWithOSFiles(t, files)
	defer cleanup()

	options := core.NgCompilerOptions{}
	compiler, err := core.NewNgCompiler(result.Program, options, nil)
	require.NoError(t, err)

	compiler.AnalyzeSync()

	sf := result.GetSourceFileByName("cmp-a.ts")
	require.NotNil(t, sf)
	classDecl := ngtsctest.FindNamedClassDeclaration(sf, "CmpA")
	require.NotNil(t, classDecl)

	resources := compiler.GetDirectiveResources(classDecl.AsNode())
	require.NotNil(t, resources)
	assert.Len(t, resources.Styles, 0)
}

func TestNgCompiler_GetResourceDependencies(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	cmpTsPath := filepath.Join(tempDir, "cmp.ts")
	templateHtmlPath := filepath.Join(tempDir, "template.html")
	stylePath := filepath.Join(tempDir, "style.css")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: cmpTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'test-cmp',
					templateUrl: './template.html',
					styleUrls: ['./style.css'],
				})
				export class Cmp {}
			`,
		},
		{
			Name: templateHtmlPath,
			Contents: `<h1>Resource</h1>`,
		},
		{
			Name: stylePath,
			Contents: `h1 {}`,
		},
	}

	result, cleanup := createProgramWithOSFiles(t, files)
	defer cleanup()

	options := core.NgCompilerOptions{
		StrictTemplates: true,
	}
	compiler, err := core.NewNgCompiler(result.Program, options, nil)
	require.NoError(t, err)

	sf := result.GetSourceFileByName("cmp.ts")
	require.NotNil(t, sf)

	deps := compiler.GetResourceDependencies(sf)
	require.Len(t, deps, 2)
	assert.Contains(t, deps, templateHtmlPath)
	assert.Contains(t, deps, stylePath)
}

func TestNgCompiler_ResourceOnlyChanges(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	cmpTsPath := filepath.Join(tempDir, "cmp.ts")
	templateHtmlPath := filepath.Join(tempDir, "template.html")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: cmpTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'test-cmp',
					standalone: true,
					templateUrl: './template.html',
				})
				export class Cmp {}
			`,
		},
		{
			Name: templateHtmlPath,
			Contents: `<h1>Resource</h1>`,
		},
	}

	result, cleanup := createProgramWithOSFiles(t, files)
	defer cleanup()

	// 1. Initial compile
	optionsA := core.NgCompilerOptions{
		StrictTemplates: true,
	}
	compilerA, err := core.NewNgCompiler(result.Program, optionsA, nil)
	require.NoError(t, err)

	sf := result.GetSourceFileByName("cmp.ts")
	diagsA := compilerA.GetDiagnosticsForFile(sf, nil)
	assert.Len(t, diagsA, 0)

	// 2. Change resource file to introduce error
	err = result.Program.Host().FS().WriteFile(templateHtmlPath, "{{invalid_var.foo}}")
	require.NoError(t, err)
	err = os.WriteFile(templateHtmlPath, []byte("{{invalid_var.foo}}"), 0644)
	require.NoError(t, err)

	// 3. Incremental compile with resource change
	optionsB := core.NgCompilerOptions{
		StrictTemplates: true,
		InvalidatedFiles: map[string]bool{
			templateHtmlPath: true,
		},
	}
	compilerB, err := core.NewNgCompiler(result.Program, optionsB, compilerA)
	require.NoError(t, err)

	diagsB := compilerB.GetDiagnosticsForFile(sf, nil)
	assert.Len(t, diagsB, 1)
	assert.Contains(t, diagsB[0].MessageArgs()[0], "invalid_var")
}
