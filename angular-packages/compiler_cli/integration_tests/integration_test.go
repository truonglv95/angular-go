package integration_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/component"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/directive"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/ng_module"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/pipe"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	ngdiagnostics "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_AngularAppCompilation(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test that requires type checker")
	}

	// 1. Define a miniature real Angular application structure as a fixture.
	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/node_modules/@angular/core/index.d.ts",
			Contents: `
				export const Component: any;
				export const Directive: any;
				export const Pipe: any;
				export const NgModule: any;
			`,
		},
		{
			Name: "/highlight.directive.ts",
			Contents: `
				import {Directive} from '@angular/core';
				@Directive({
					selector: '[appHighlight]',
					standalone: false
				})
				export class HighlightDirective {}
			`,
		},
		{
			Name: "/truncate.pipe.ts",
			Contents: `
				import {Pipe} from '@angular/core';
				@Pipe({
					name: 'truncate',
					standalone: false
				})
				export class TruncatePipe {}
			`,
		},
		{
			Name: "/app.component.ts",
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'app-root',
					standalone: false,
					template: '<h1>Hello Angular Go!</h1>'
				})
				export class AppComponent {}
			`,
		},
		{
			Name: "/app.module.ts",
			Contents: `
				import {NgModule} from '@angular/core';
				import {AppComponent} from './app.component';
				import {HighlightDirective} from './highlight.directive';
				import {TruncatePipe} from './truncate.pipe';

				@NgModule({
					declarations: [AppComponent, HighlightDirective, TruncatePipe],
					bootstrap: [AppComponent]
				})
				export class AppModule {}
			`,
		},
	})
	defer result.Release()

	// 2. Set up reflection host, evaluator, registries, and scope builder.
	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	evaluator := partial_evaluator.NewPartialEvaluator(host, result.Checker, nil)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	// 3. Initialize individual decorators handlers.
	compHandler := component.NewComponentDecoratorHandler(host, evaluator)
	dirHandler := directive.NewDirectiveDecoratorHandler(host, evaluator)
	pipeHandler := pipe.NewPipeDecoratorHandler(host, evaluator)
	moduleHandler := ng_module.NewNgModuleDecoratorHandler(host, evaluator)

	// 4. Retrieve source files and find class declarations.
	sfHighlight := ngtsctest.RequireSourceFile(t, result, "/highlight.directive.ts")
	classHighlight := ngtsctest.FindNamedClassDeclaration(sfHighlight, "HighlightDirective")
	require.NotNil(t, classHighlight)

	sfTruncate := ngtsctest.RequireSourceFile(t, result, "/truncate.pipe.ts")
	classTruncate := ngtsctest.FindNamedClassDeclaration(sfTruncate, "TruncatePipe")
	require.NotNil(t, classTruncate)

	sfComponent := ngtsctest.RequireSourceFile(t, result, "/app.component.ts")
	classComponent := ngtsctest.FindNamedClassDeclaration(sfComponent, "AppComponent")
	require.NotNil(t, classComponent)

	sfModule := ngtsctest.RequireSourceFile(t, result, "/app.module.ts")
	classModule := ngtsctest.FindNamedClassDeclaration(sfModule, "AppModule")
	require.NotNil(t, classModule)

	// 5. Analyze HighlightDirective and register metadata.
	dirData, err := dirHandler.Analyze(classHighlight)
	require.NoError(t, err)
	require.NotNil(t, dirData)
	metaRegistry.RegisterDirective(classHighlight, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindDirective,
		Ref:         metadata.Reference{Node: classHighlight},
		Selector:    dirData.Selector,
		Standalone:  dirData.Standalone,
		IsComponent: false,
	})

	// 6. Analyze TruncatePipe and register metadata.
	pipeData, err := pipeHandler.Analyze(classTruncate)
	require.NoError(t, err)
	require.NotNil(t, pipeData)
	metaRegistry.RegisterPipe(classTruncate, &metadata.PipeMeta{
		Ref:        metadata.Reference{Node: classTruncate},
		Name:       pipeData.Name,
		Pure:       pipeData.Pure,
		Standalone: pipeData.Standalone,
	})

	// 7. Analyze AppComponent and register metadata.
	compData, err := compHandler.Analyze(classComponent)
	require.NoError(t, err)
	require.NotNil(t, compData)
	metaRegistry.RegisterDirective(classComponent, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindComponent,
		Ref:         metadata.Reference{Node: classComponent},
		Selector:    compData.Selector,
		Standalone:  compData.Standalone,
		IsComponent: true,
	})

	// 8. Analyze AppModule and register metadata.
	moduleData, err := moduleHandler.Analyze(classModule)
	require.NoError(t, err)
	require.NotNil(t, moduleData)

	// Build NgModuleMeta by mapping resolved declarations to References
	var declRefs []metadata.Reference
	for _, declVal := range moduleData.Declarations {
		if ref, ok := declVal.(*imports.Reference); ok {
			declRefs = append(declRefs, metadata.Reference{Node: ref.Node})
		} else if declNode, ok := declVal.(*ast.Node); ok {
			declRefs = append(declRefs, metadata.Reference{Node: declNode})
		}
	}

	metaRegistry.RegisterNgModule(classModule, &metadata.NgModuleMeta{
		Ref:          metadata.Reference{Node: classModule},
		Declarations: declRefs,
	})

	// 9. Register component declaration in scope registry and verify visible scope.
	scopeRegistry.RegisterComponentDeclaration(classComponent, classModule)
	compilationScope := scopeRegistry.GetCompilationScope(classComponent)
	require.NotNil(t, compilationScope)

	// Since AppModule declared both AppComponent, HighlightDirective, and TruncatePipe,
	// the compilation scope of AppComponent must contain:
	// - HighlightDirective (visible directive!)
	// - TruncatePipe (visible pipe!)
	// - AppComponent itself (visible directive/component!)

	var foundHighlight, foundTruncate, foundComponent bool
	for _, d := range compilationScope.Directives {
		if d.Selector == "[appHighlight]" {
			foundHighlight = true
		}
		if d.Selector == "app-root" {
			foundComponent = true
		}
	}
	for _, p := range compilationScope.Pipes {
		if p.Name == "truncate" {
			foundTruncate = true
		}
	}

	assert.True(t, foundHighlight, "HighlightDirective should be visible in compilation scope")
	assert.True(t, foundComponent, "AppComponent should be visible in compilation scope")
	assert.True(t, foundTruncate, "TruncatePipe should be visible in compilation scope")
}

func TestIntegration_NgModuleDiagnostics(t *testing.T) {
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
			`,
		},
		{
			Name: "/app.component.ts",
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'app-root',
					standalone: true,
					template: '<h1>Hello Standalone!</h1>'
				})
				export class StandaloneComponent {}

				@Component({
					selector: 'app-non-standalone',
					standalone: false,
					template: '<h1>Hello Non-Standalone!</h1>'
				})
				export class NonStandaloneComponent {}
			`,
		},
		{
			Name: "/app.module.ts",
			Contents: `
				import {NgModule} from '@angular/core';
				import {StandaloneComponent, NonStandaloneComponent} from './app.component';

				@NgModule({
					declarations: [StandaloneComponent, NonStandaloneComponent, NonStandaloneComponent],
					imports: [NonStandaloneComponent]
				})
				export class AppModule {}
			`,
		},
	})
	defer result.Release()

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	compHandler := annotations.NewComponentDecoratorHandler(host, false, metaRegistry, scopeRegistry, nil, false, nil)
	moduleHandler := annotations.NewNgModuleDecoratorHandler(host, metaRegistry, scopeRegistry)

	sfComponent := ngtsctest.RequireSourceFile(t, result, "/app.component.ts")
	classStandalone := ngtsctest.FindNamedClassDeclaration(sfComponent, "StandaloneComponent")
	classNonStandalone := ngtsctest.FindNamedClassDeclaration(sfComponent, "NonStandaloneComponent")

	sfModule := ngtsctest.RequireSourceFile(t, result, "/app.module.ts")
	classModule := ngtsctest.FindNamedClassDeclaration(sfModule, "AppModule")

	// 1. Analyze components
	standaloneDec := compHandler.Detect(classStandalone.AsClassDeclaration(), host.GetDecoratorsOfDeclaration(classStandalone))
	_, diags := compHandler.Analyze(classStandalone.AsClassDeclaration(), standaloneDec)
	require.Empty(t, diags)

	nonStandaloneDec := compHandler.Detect(classNonStandalone.AsClassDeclaration(), host.GetDecoratorsOfDeclaration(classNonStandalone))
	_, diags = compHandler.Analyze(classNonStandalone.AsClassDeclaration(), nonStandaloneDec)
	require.Empty(t, diags)

	// 2. Analyze NgModule
	moduleDec := moduleHandler.Detect(classModule.AsClassDeclaration(), host.GetDecoratorsOfDeclaration(classModule))
	analysis, diags := moduleHandler.Analyze(classModule.AsClassDeclaration(), moduleDec)
	require.Empty(t, diags)

	// 3. Resolve NgModule
	_, diags = moduleHandler.Resolve(classModule.AsClassDeclaration(), analysis)

	// Expect diagnostics errors:
	// - StandaloneComponent is standalone but declared in NgModule (6008)
	// - NonStandaloneComponent is declared multiple times in this NgModule (6001)
	// - NonStandaloneComponent is imported but not standalone/NgModule (6002)
	assert.NotEmpty(t, diags)

	found6008 := false
	found6001 := false
	found6002 := false
	for _, diag := range diags {
		if diag.Code() == int32(ngdiagnostics.NgErrorCode(ngdiagnostics.ErrorCode_NGMODULE_DECLARATION_IS_STANDALONE)) {
			found6008 = true
		}
		if diag.Code() == int32(ngdiagnostics.NgErrorCode(ngdiagnostics.ErrorCode_NGMODULE_INVALID_DECLARATION)) {
			found6001 = true
		}
		if diag.Code() == int32(ngdiagnostics.NgErrorCode(ngdiagnostics.ErrorCode_NGMODULE_INVALID_IMPORT)) {
			found6002 = true
		}
	}
	assert.True(t, found6008, "Should report ErrorCode_NGMODULE_DECLARATION_IS_STANDALONE (6008)")
	assert.True(t, found6001, "Should report ErrorCode_NGMODULE_INVALID_DECLARATION (6001) for duplicate declarations")
	assert.True(t, found6002, "Should report ErrorCode_NGMODULE_INVALID_IMPORT (6002) for non-standalone imported directive")
}
