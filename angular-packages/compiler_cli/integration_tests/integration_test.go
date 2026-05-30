package integration_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/component"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/directive"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/ng_module"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/pipe"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
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
		// In actual compiler compilation, these values are resolved to class nodes.
		// For our integration test, the evaluator resolves the arrays of local identifiers
		// directly to their local class declarations.
		if declNode, ok := declVal.(*ast.Node); ok {
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
