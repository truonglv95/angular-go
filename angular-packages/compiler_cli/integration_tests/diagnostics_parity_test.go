package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/core"
	ngdiagnostics "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/bundled"
	internalcompiler "github.com/microsoft/typescript-go/internal/compiler"
	internalcore "github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/vfs/vfstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ComponentTemplateDiagnostics(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test that requires type checker")
	}

	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/node_modules/@angular/core/index.d.ts",
			Contents: `
				export const Component: any;
				export const Directive: any;
				export const Pipe: any;
				export const NgModule: any;
				export const Input: any;
				export const Output: any;
			`,
		},
		{
			Name: "/app.component.ts",
			Contents: `
				import {Component, Input, Output} from '@angular/core';

				@Component({
					selector: 'app-valid',
					standalone: true,
					template: '<div>Hello</div>'
				})
				export class ValidComponent {}

				@Component({
					selector: 'app-invalid-elements-properties',
					standalone: true,
					template: '<unknown-el [invalidProp]="123">Hello</unknown-el>'
				})
				export class InvalidElementsPropertiesComponent {}

				@Component({
					selector: 'app-missing-pipe',
					standalone: true,
					template: '<p>{{ 123 | unknownPipe }}</p>'
				})
				export class MissingPipeComponent {}

				@Component({
					selector: 'app-invalid-banana',
					standalone: true,
					template: '<div ([banana])="val"></div>'
				})
				export class InvalidBananaComponent {}

				@Component({
					selector: 'app-duplicate-bindings',
					standalone: true,
					template: '<div></div>',
					inputs: ['dup'],
				})
				export class DuplicateBindingsComponent {
					@Input('dup') myInput: any;
					@Output('dup') myOutput: any;
				}

				@Component({
					selector: 'app-missing-required-input',
					standalone: true,
					imports: [ChildComponent],
					template: '<child-cmp></child-cmp>'
				})
				export class MissingRequiredInputComponent {}

				@Component({
					selector: 'child-cmp',
					standalone: true,
					template: '<div></div>'
				})
				export class ChildComponent {
					@Input({required: true}) reqVal: any;
				}
			`,
		},
	})
	defer result.Release()

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	compHandler := annotations.NewComponentDecoratorHandler(host, false, metaRegistry, scopeRegistry, false)

	sfComponent := ngtsctest.RequireSourceFile(t, result, "/app.component.ts")

	// Helper to analyze & resolve a component
	runResolve := func(className string) []ngdiagnostics.ErrorCode {
		classDecl := ngtsctest.FindNamedClassDeclaration(sfComponent, className)
		require.NotNil(t, classDecl)
		dec := compHandler.Detect(classDecl.AsClassDeclaration(), host.GetDecoratorsOfDeclaration(classDecl))
		require.NotNil(t, dec)
		analysis, diags := compHandler.Analyze(classDecl.AsClassDeclaration(), dec)
		require.Empty(t, diags)

		if className == "ChildComponent" {
			// Register child component metadata
			metaRegistry.RegisterDirective(classDecl.AsNode(), &metadata.DirectiveMeta{
				Name:        className,
				Selector:    "child-cmp",
				Standalone:  true,
				IsComponent: true,
				Inputs:      map[string]string{"reqVal": "reqVal"},
				RequiredInputs: []string{"reqVal"},
				Ref: metadata.Reference{
					Name: className,
					Node: classDecl.AsNode(),
				},
			})
		} else if className == "MissingRequiredInputComponent" {
			// Pre-register child component so that it resolves in parent template binder
			childDecl := ngtsctest.FindNamedClassDeclaration(sfComponent, "ChildComponent")
			require.NotNil(t, childDecl)
			metaRegistry.RegisterDirective(childDecl.AsNode(), &metadata.DirectiveMeta{
				Name:        "ChildComponent",
				Selector:    "child-cmp",
				Standalone:  true,
				IsComponent: true,
				Inputs:      map[string]string{"reqVal": "reqVal"},
				RequiredInputs: []string{"reqVal"},
				Ref: metadata.Reference{
					Name: "ChildComponent",
					Node: childDecl.AsNode(),
				},
			})
		}

		_, resolveDiags := compHandler.Resolve(classDecl.AsClassDeclaration(), analysis)
		var fatalDiags []ngdiagnostics.ErrorCode
		for _, d := range resolveDiags {
			codeVal := d.Code()
			// Reverse NgErrorCode mapping:
			// NgErrorCode does: val, _ := strconv.Atoi("-99" + strconv.Itoa(int(code)))
			// which is -990000 - code. For example, 8001 becomes -998001.
			// Let's decode it:
			origCode := ngdiagnostics.ErrorCode(0)
			if codeVal < 0 {
				origCode = ngdiagnostics.ErrorCode(-codeVal - 990000)
			}
			fatalDiags = append(fatalDiags, origCode)
			msgStr := ""
			if len(d.MessageArgs()) > 0 {
				msgStr = d.MessageArgs()[0]
			}
			t.Logf("Component %s got diagnostic: Code: %d (Orig: %d), Message: %s", className, codeVal, origCode, msgStr)
		}
		return fatalDiags
	}

	// 1. Valid Component should have no resolve diagnostics
	assert.Empty(t, runResolve("ValidComponent"))

	// 2. Invalid elements and properties
	diags := runResolve("InvalidElementsPropertiesComponent")
	assert.NotEmpty(t, diags)
	assert.Contains(t, diags, ngdiagnostics.ErrorCode_SCHEMA_INVALID_ELEMENT)
	assert.Contains(t, diags, ngdiagnostics.ErrorCode_SCHEMA_INVALID_ATTRIBUTE)

	// 3. Missing Pipe
	diags = runResolve("MissingPipeComponent")
	assert.NotEmpty(t, diags)
	assert.Contains(t, diags, ngdiagnostics.ErrorCode_MISSING_PIPE)

	// 4. Invalid banana in box
	diags = runResolve("InvalidBananaComponent")
	assert.NotEmpty(t, diags)
	assert.Contains(t, diags, ngdiagnostics.ErrorCode_INVALID_BANANA_IN_BOX)

	// 5. Duplicate bindings
	diags = runResolve("DuplicateBindingsComponent")
	assert.NotEmpty(t, diags)
	assert.Contains(t, diags, ngdiagnostics.ErrorCode_DUPLICATE_BINDING_NAME)

	// 6. Missing required inputs
	diags = runResolve("MissingRequiredInputComponent")
	assert.NotEmpty(t, diags)
	assert.Contains(t, diags, ngdiagnostics.ErrorCode_MISSING_REQUIRED_INPUTS)
}

func TestIntegration_ComponentTemplateTypeChecking(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test that requires type checker")
	}

	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/node_modules/@angular/core/index.d.ts",
			Contents: `
				export const Component: any;
				export const Input: any;
			`,
		},
		{
			Name: "/app.component.ts",
			Contents: `
				import {Component, Input} from '@angular/core';

				@Component({
					selector: 'child-cmp',
					standalone: true,
					template: '<div></div>'
				})
				export class ChildComponent {
					@Input() inputStr: string = '';
				}

				@Component({
					selector: 'app-tcb-test',
					standalone: true,
					imports: [ChildComponent],
					template: '<div>{{ age }}</div><child-cmp [inputStr]="age"></child-cmp><p>{{ nonExistentProp }}</p>'
				})
				export class TcbTestComponent {
					age: number = 10;
				}
			`,
		},
	})
	defer result.Release()

	options := core.NgCompilerOptions{
		CompilationMode: "global",
		StrictTemplates:  true,
	}
	compiler, err := core.NewNgCompiler(result.Program, options, nil)
	require.NoError(t, err)

	// Analyze
	diags := compiler.AnalyzeSync()
	require.Empty(t, diags)

	// Resolve (which executes the TCB type checker!)
	resolveDiags := compiler.Resolve()
	assert.NotEmpty(t, resolveDiags)

	hasInputTypeMismatch := false
	hasNonExistentProp := false

	for _, d := range resolveDiags {
		msg := d.String()
		if strings.Contains(msg, "Type 'number' is not assignable to type 'string'") {
			hasInputTypeMismatch = true
		}
		if strings.Contains(msg, "Property 'nonExistentProp' does not exist on type 'TcbTestComponent'") {
			hasNonExistentProp = true
		}
		t.Logf("TCB type checking diagnostic: %s", msg)
	}

	assert.True(t, hasInputTypeMismatch, "Expected to catch input type mismatch")
	assert.True(t, hasNonExistentProp, "Expected to catch non-existent property access")
}

func TestIntegration_IncrementalCompilationParity(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	appComponentTsPath := filepath.Join(tempDir, "app.component.ts")
	appComponentHtmlPath := filepath.Join(tempDir, "app.component.html")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
				export const Directive: any;
				export const Input: any;
				export const Pipe: any;
				export const NgModule: any;
			`,
		},
		{
			Name: appComponentTsPath,
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'app-root',
					standalone: true,
					templateUrl: './app.component.html'
				})
				export class AppComponent {
					title = 'angular-go';
				}
			`,
		},
		{
			Name: appComponentHtmlPath,
			Contents: `<h1>{{ title }}</h1>`,
		},
	}

	fs := vfstest.FromMap[any](nil, false)
	fs = bundled.WrapFS(fs)

	writeFile := func(path string, contents string) {
		_ = fs.WriteFile(path, contents)
		dir := filepath.Dir(path)
		_ = os.MkdirAll(dir, 0755)
		_ = os.WriteFile(path, []byte(contents), 0644)
	}

	for _, f := range files {
		writeFile(f.Name, f.Contents)
	}

	createProgram := func() (*internalcompiler.Program, func()) {
		opts := &internalcore.CompilerOptions{
			Target:                 internalcore.ScriptTargetES2015,
			Module:                 internalcore.ModuleKindCommonJS,
			ExperimentalDecorators: internalcore.TSTrue,
		}
		var rootNames []string
		for _, f := range files {
			if !strings.HasSuffix(f.Name, ".html") {
				rootNames = append(rootNames, f.Name)
			}
		}
		progOpts := internalcompiler.ProgramOptions{
			Config: &tsoptions.ParsedCommandLine{
				ParsedConfig: &internalcore.ParsedOptions{
					FileNames:       rootNames,
					CompilerOptions: opts,
				},
			},
			Host: internalcompiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil),
		}
		program := internalcompiler.NewProgram(progOpts)
		chk, release := program.GetTypeChecker(context.Background())
		_ = chk
		return program, release
	}

	// --- FIRST COMPILATION (Fresh) ---
	prog1, release1 := createProgram()
	defer release1()

	opts1 := core.NgCompilerOptions{
		CompilationMode: "global",
		StrictTemplates:  true,
	}
	comp1, err := core.NewNgCompiler(prog1, opts1, nil)
	require.NoError(t, err)

	diags1 := comp1.AnalyzeSync()
	require.Empty(t, diags1)

	resolveDiags1 := comp1.Resolve()
	assert.Empty(t, resolveDiags1)

	// --- SECOND COMPILATION (Incremental: Unchanged build) ---
	prog2, release2 := createProgram()
	defer release2()

	opts2 := core.NgCompilerOptions{
		CompilationMode: "global",
		StrictTemplates:  true,
	}
	comp2, err := core.NewNgCompiler(prog2, opts2, comp1)
	require.NoError(t, err)

	diags2 := comp2.AnalyzeSync()
	require.Empty(t, diags2)

	resolveDiags2 := comp2.Resolve()
	assert.Empty(t, resolveDiags2)

	// --- THIRD COMPILATION (Incremental: Edit template to introduce error) ---
	writeFile(appComponentHtmlPath, `<h1>{{ invalidProp }}</h1>`)

	prog3, release3 := createProgram()
	defer release3()

	opts3 := core.NgCompilerOptions{
		CompilationMode:  "global",
		StrictTemplates:   true,
		InvalidatedFiles: map[string]bool{appComponentHtmlPath: true},
	}
	comp3, err := core.NewNgCompiler(prog3, opts3, comp2)
	require.NoError(t, err)

	diags3 := comp3.AnalyzeSync()
	require.Empty(t, diags3)

	resolveDiags3 := comp3.Resolve()
	assert.NotEmpty(t, resolveDiags3)
	foundErr := false
	for _, d := range resolveDiags3 {
		if strings.Contains(d.String(), "Property 'invalidProp' does not exist on type 'AppComponent'") {
			foundErr = true
		}
	}
	assert.True(t, foundErr, "Expected template type checking diagnostic for 'invalidProp'")

	// --- FOURTH COMPILATION (Incremental: Fix template error) ---
	writeFile(appComponentHtmlPath, `<h1>{{ title }}</h1>`)

	prog4, release4 := createProgram()
	defer release4()

	opts4 := core.NgCompilerOptions{
		CompilationMode:  "global",
		StrictTemplates:   true,
		InvalidatedFiles: map[string]bool{appComponentHtmlPath: true},
	}
	comp4, err := core.NewNgCompiler(prog4, opts4, comp3)
	require.NoError(t, err)

	diags4 := comp4.AnalyzeSync()
	require.Empty(t, diags4)

	resolveDiags4 := comp4.Resolve()
	assert.Empty(t, resolveDiags4)
}

func TestIntegration_IncrementalSemanticInvalidation(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test")
	}

	tempDir := t.TempDir()
	coreDtsPath := filepath.Join(tempDir, "node_modules/@angular/core/index.d.ts")
	dirTsPath := filepath.Join(tempDir, "dir.ts")
	appComponentTsPath := filepath.Join(tempDir, "app.component.ts")

	files := []ngtsctest.ProgramFile{
		{
			Name: coreDtsPath,
			Contents: `
				export const Component: any;
				export const Directive: any;
				export const Input: any;
				export const Pipe: any;
				export const NgModule: any;
			`,
		},
		{
			Name: dirTsPath,
			Contents: `
				import {Directive, Input} from '@angular/core';
				@Directive({
					selector: '[appDir]',
					standalone: true
				})
				export class MyDirective {
					@Input() dirInput: string = '';
				}
			`,
		},
		{
			Name: appComponentTsPath,
			Contents: `
				import {Component} from '@angular/core';
				import {MyDirective} from './dir';
				@Component({
					selector: 'app-root',
					standalone: true,
					imports: [MyDirective],
					template: '<div appDir [dirInput]="title"></div>'
				})
				export class AppComponent {
					title = 'angular-go';
				}
			`,
		},
	}

	fs := vfstest.FromMap[any](nil, false)
	fs = bundled.WrapFS(fs)

	writeFile := func(path string, contents string) {
		_ = fs.WriteFile(path, contents)
		dir := filepath.Dir(path)
		_ = os.MkdirAll(dir, 0755)
		_ = os.WriteFile(path, []byte(contents), 0644)
	}

	for _, f := range files {
		writeFile(f.Name, f.Contents)
	}

	createProgram := func() (*internalcompiler.Program, func()) {
		opts := &internalcore.CompilerOptions{
			Target:                 internalcore.ScriptTargetES2015,
			Module:                 internalcore.ModuleKindCommonJS,
			ExperimentalDecorators: internalcore.TSTrue,
		}
		var rootNames []string
		for _, f := range files {
			rootNames = append(rootNames, f.Name)
		}
		progOpts := internalcompiler.ProgramOptions{
			Config: &tsoptions.ParsedCommandLine{
				ParsedConfig: &internalcore.ParsedOptions{
					FileNames:       rootNames,
					CompilerOptions: opts,
				},
			},
			Host: internalcompiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil),
		}
		program := internalcompiler.NewProgram(progOpts)
		chk, release := program.GetTypeChecker(context.Background())
		_ = chk
		return program, release
	}

	// --- 1. Fresh Compilation ---
	prog1, release1 := createProgram()
	defer release1()

	opts1 := core.NgCompilerOptions{
		CompilationMode: "global",
		StrictTemplates:  true,
	}
	comp1, err := core.NewNgCompiler(prog1, opts1, nil)
	require.NoError(t, err)
	require.Empty(t, comp1.AnalyzeSync())
	require.Empty(t, comp1.Resolve())

	// --- 2. Change Directive input name (dirInput -> dirInputNew) ---
	writeFile(dirTsPath, `
		import {Directive, Input} from '@angular/core';
		@Directive({
			selector: '[appDir]',
			standalone: true
		})
		export class MyDirective {
			@Input() dirInputNew: string = '';
		}
	`)

	prog2, release2 := createProgram()
	defer release2()

	opts2 := core.NgCompilerOptions{
		CompilationMode: "global",
		StrictTemplates:  true,
	}
	comp2, err := core.NewNgCompiler(prog2, opts2, comp1)
	require.NoError(t, err)
	require.Empty(t, comp2.AnalyzeSync())
	
	resolveDiags2 := comp2.Resolve()
	assert.NotEmpty(t, resolveDiags2, "Expected semantic invalidation to catch the binding error")
}
