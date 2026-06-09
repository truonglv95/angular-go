package integration_tests

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getDecorator(host reflection.ReflectionHost, classNode *ast.Node, name string) *reflection.Decorator {
	decorators := host.GetDecoratorsOfDeclaration(classNode)
	for _, dec := range decorators {
		if dec.Name == name {
			return &dec
		}
	}
	return nil
}

func TestIntegration_StandaloneBootstrap(t *testing.T) {
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
			`,
		},
		{
			Name: "/child.component.ts",
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'child-comp',
					standalone: true,
					template: '<p>Child Component</p>'
				})
				export class ChildComponent {}
			`,
		},
		{
			Name: "/my.directive.ts",
			Contents: `
				import {Directive} from '@angular/core';
				@Directive({
					selector: '[myDir]',
					standalone: true
				})
				export class MyDirective {}
			`,
		},
		{
			Name: "/my.pipe.ts",
			Contents: `
				import {Pipe} from '@angular/core';
				@Pipe({
					name: 'myPipe',
					standalone: true
				})
				export class MyPipe {}
			`,
		},
		{
			Name: "/app.component.ts",
			Contents: `
				import {Component} from '@angular/core';
				import {ChildComponent} from './child.component';
				import {MyDirective} from './my.directive';
				import {MyPipe} from './my.pipe';

				@Component({
					selector: 'app-root',
					standalone: true,
					imports: [ChildComponent, MyDirective, MyPipe],
					template: '<h1>App Root</h1><child-comp myDir></child-comp>{{ "test" | myPipe }}'
				})
				export class AppComponent {}
			`,
		},
	})
	defer result.Release()

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)

	compHandler := annotations.NewComponentDecoratorHandler(host, false, metaRegistry, scopeRegistry, nil, false)
	dirHandler := annotations.NewDirectiveDecoratorHandler(host, metaRegistry)
	pipeHandler := annotations.NewPipeDecoratorHandler(host, metaRegistry)

	// Retrieve source files & class declarations
	sfChild := ngtsctest.RequireSourceFile(t, result, "/child.component.ts")
	classChild := ngtsctest.FindNamedClassDeclaration(sfChild, "ChildComponent")
	require.NotNil(t, classChild)

	sfDir := ngtsctest.RequireSourceFile(t, result, "/my.directive.ts")
	classDir := ngtsctest.FindNamedClassDeclaration(sfDir, "MyDirective")
	require.NotNil(t, classDir)

	sfPipe := ngtsctest.RequireSourceFile(t, result, "/my.pipe.ts")
	classPipe := ngtsctest.FindNamedClassDeclaration(sfPipe, "MyPipe")
	require.NotNil(t, classPipe)

	sfApp := ngtsctest.RequireSourceFile(t, result, "/app.component.ts")
	classApp := ngtsctest.FindNamedClassDeclaration(sfApp, "AppComponent")
	require.NotNil(t, classApp)

	// Analyze & Register child standalone component
	childDec := getDecorator(host, classChild, "Component")
	require.NotNil(t, childDec)
	childVal, diags := compHandler.Analyze(classChild.AsClassDeclaration(), childDec)
	require.Empty(t, diags)
	childData := childVal.(*annotations.ComponentAnalysis)
	metaRegistry.RegisterDirective(classChild, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindComponent,
		Ref:         metadata.Reference{Node: classChild},
		Selector:    childData.Selector,
		Standalone:  childData.IsStandalone,
		IsComponent: true,
	})

	// Analyze & Register child standalone directive
	dirDec := getDecorator(host, classDir, "Directive")
	require.NotNil(t, dirDec)
	dirVal, diags := dirHandler.Analyze(classDir.AsClassDeclaration(), dirDec)
	require.Empty(t, diags)
	dirData := dirVal.(*annotations.DirectiveAnalysis)
	metaRegistry.RegisterDirective(classDir, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindDirective,
		Ref:         metadata.Reference{Node: classDir},
		Selector:    dirData.Selector,
		Standalone:  dirData.IsStandalone,
		IsComponent: false,
	})

	// Analyze & Register child standalone pipe
	pipeDec := getDecorator(host, classPipe, "Pipe")
	require.NotNil(t, pipeDec)
	pipeVal, diags := pipeHandler.Analyze(classPipe.AsClassDeclaration(), pipeDec)
	require.Empty(t, diags)
	pipeData := pipeVal.(*annotations.PipeAnalysis)
	metaRegistry.RegisterPipe(classPipe, &metadata.PipeMeta{
		Ref:        metadata.Reference{Node: classPipe},
		Name:       pipeData.Name,
		Pure:       pipeData.Pure,
		Standalone: pipeData.IsStandalone,
	})

	// Analyze & Register root standalone AppComponent
	appDec := getDecorator(host, classApp, "Component")
	require.NotNil(t, appDec)
	appVal, diags := compHandler.Analyze(classApp.AsClassDeclaration(), appDec)
	require.Empty(t, diags)
	appData := appVal.(*annotations.ComponentAnalysis)
	assert.True(t, appData.IsStandalone)

	// Map evaluated imports to references for metadata registry
	var importRefs []metadata.Reference
	for _, imp := range appData.Imports {
		if imp.Decl.Node != nil {
			importRefs = append(importRefs, metadata.Reference{Node: imp.Decl.Node})
		}
	}

	metaRegistry.RegisterDirective(classApp, &metadata.DirectiveMeta{
		Kind:        metadata.MetaKindComponent,
		Ref:         metadata.Reference{Node: classApp},
		Selector:    appData.Selector,
		Standalone:  appData.IsStandalone,
		IsComponent: true,
		Imports:     importRefs,
	})

	// Retrieve compilation scope for AppComponent
	compilationScope := scopeRegistry.GetCompilationScope(classApp)
	require.NotNil(t, compilationScope)

	// Verify dependencies are resolved correctly in the compilation scope
	var foundChild, foundDir, foundPipe bool
	for _, d := range compilationScope.Directives {
		if d.Selector == "child-comp" {
			foundChild = true
		}
		if d.Selector == "[myDir]" {
			foundDir = true
		}
	}
	for _, p := range compilationScope.Pipes {
		if p.Name == "myPipe" {
			foundPipe = true
		}
	}

	assert.True(t, foundChild, "ChildComponent should be resolved in standalone AppComponent scope")
	assert.True(t, foundDir, "MyDirective should be resolved in standalone AppComponent scope")
	assert.True(t, foundPipe, "MyPipe should be resolved in standalone AppComponent scope")
}

func TestIntegration_LazyLoading(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test that requires type checker")
	}

	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/node_modules/@angular/core/index.d.ts",
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: "/app.routes.ts",
			Contents: `
				export const routes = [
					{
						path: 'lazy',
						loadComponent: () => import('./lazy.component').then(m => m.LazyComponent)
					}
				];
			`,
		},
		{
			Name: "/lazy.component.ts",
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'lazy-comp',
					template: 'Lazy Loaded!'
				})
				export class LazyComponent {}
			`,
		},
	})
	defer result.Release()

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	evaluator := partial_evaluator.NewPartialEvaluator(host, result.Checker, nil)

	sfRoutes := ngtsctest.RequireSourceFile(t, result, "/app.routes.ts")
	varDecl := ngtsctest.FindVariableDeclaration(sfRoutes, "routes")
	require.NotNil(t, varDecl)

	// Evaluate the routes array which contains a dynamic import arrow function.
	// We want to ensure that evaluation does not crash and handles dynamic import expression gracefully.
	resolved := evaluator.Evaluate(varDecl.Initializer(), nil)
	require.NotNil(t, resolved)

	// It should resolve to an array of objects
	arr, ok := resolved.(partial_evaluator.ResolvedValueArray)
	require.True(t, ok)
	require.Equal(t, 1, len(arr))

	routeMap, ok := arr[0].(partial_evaluator.ResolvedValueMap)
	require.True(t, ok)
	require.Equal(t, "lazy", routeMap["path"])

	// loadComponent is an arrow function call containing a dynamic import
	// Our partial evaluator represents it as a dynamic value or similar.
	_, isDynamic := routeMap["loadComponent"].(*partial_evaluator.DynamicValue)
	assert.True(t, isDynamic, "loadComponent should be resolved as a DynamicValue since it involves a dynamic import call")
}

func TestIntegration_DeferBlocks(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test that requires type checker")
	}

	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/node_modules/@angular/core/index.d.ts",
			Contents: `
				export const Component: any;
			`,
		},
		{
			Name: "/app.component.ts",
			Contents: `
				import {Component} from '@angular/core';
				@Component({
					selector: 'app-root',
					standalone: true,
					template: ` + "`" + `
						<h1>Main Content</h1>
						@defer (on timer(100ms)) {
							<p>Deferred block content</p>
						} @placeholder {
							<p>Placeholder</p>
						}
					` + "`" + `
				})
				export class AppComponent {}
			`,
		},
	})
	defer result.Release()

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)
	compHandler := annotations.NewComponentDecoratorHandler(host, false, metaRegistry, scopeRegistry, nil, false)

	sfApp := ngtsctest.RequireSourceFile(t, result, "/app.component.ts")
	classApp := ngtsctest.FindNamedClassDeclaration(sfApp, "AppComponent")
	require.NotNil(t, classApp)

	// Core check: Verify the ComponentHandler can successfully analyze a component using @defer syntax
	// without failing or crashing.
	appDec := getDecorator(host, classApp, "Component")
	require.NotNil(t, appDec)
	analysisVal, diags := compHandler.Analyze(classApp.AsClassDeclaration(), appDec)
	require.Empty(t, diags)
	
	data := analysisVal.(*annotations.ComponentAnalysis)
	require.NotNil(t, data)
	assert.Contains(t, data.Template, "@defer")
}

func TestIntegration_SignalInputs(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping integration test that requires type checker")
	}

	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name: "/node_modules/@angular/core/index.d.ts",
			Contents: `
				export const Component: any;
				export const input: any;
			`,
		},
		{
			Name: "/app.component.ts",
			Contents: `
				import {Component, input} from '@angular/core';
				@Component({
					selector: 'app-root',
					template: '<div>Signal Inputs</div>'
				})
				export class AppComponent {
					myInput = input('defaultVal');
					requiredInput = input.required<string>();
					aliasedInput = input('defaultVal', { alias: 'publicName' });
					aliasedRequiredInput = input.required<string>({ alias: 'publicRequiredName' });
				}
			`,
		},
	})
	defer result.Release()

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	metaRegistry := metadata.NewLocalMetadataRegistry()
	scopeRegistry := scope.NewLocalModuleScopeRegistry(metaRegistry)
	compHandler := annotations.NewComponentDecoratorHandler(host, false, metaRegistry, scopeRegistry, nil, false)

	sfApp := ngtsctest.RequireSourceFile(t, result, "/app.component.ts")
	classApp := ngtsctest.FindNamedClassDeclaration(sfApp, "AppComponent")
	require.NotNil(t, classApp)

	// Analyze the component to trigger parsing of signal inputs
	appDec := getDecorator(host, classApp, "Component")
	require.NotNil(t, appDec)
	analysisVal, diags := compHandler.Analyze(classApp.AsClassDeclaration(), appDec)
	require.Empty(t, diags)

	data := analysisVal.(*annotations.ComponentAnalysis)
	require.NotNil(t, data)

	// Verify the signal inputs metadata were collected correctly
	require.Equal(t, 4, len(data.Inputs))

	// 1. myInput = input('defaultVal');
	myInput, ok := data.Inputs["myInput"]
	require.True(t, ok)
	assert.Equal(t, "myInput", myInput.ClassPropertyName)
	assert.Equal(t, "myInput", myInput.BindingPropertyName)
	assert.True(t, myInput.IsSignal)
	assert.False(t, myInput.Required)

	// 2. requiredInput = input.required<string>();
	requiredInput, ok := data.Inputs["requiredInput"]
	require.True(t, ok)
	assert.Equal(t, "requiredInput", requiredInput.ClassPropertyName)
	assert.Equal(t, "requiredInput", requiredInput.BindingPropertyName)
	assert.True(t, requiredInput.IsSignal)
	assert.True(t, requiredInput.Required)

	// 3. aliasedInput = input('defaultVal', { alias: 'publicName' });
	aliasedInput, ok := data.Inputs["aliasedInput"]
	require.True(t, ok)
	assert.Equal(t, "aliasedInput", aliasedInput.ClassPropertyName)
	assert.Equal(t, "publicName", aliasedInput.BindingPropertyName)
	assert.True(t, aliasedInput.IsSignal)
	assert.False(t, aliasedInput.Required)

	// 4. aliasedRequiredInput = input.required<string>({ alias: 'publicRequiredName' });
	aliasedRequiredInput, ok := data.Inputs["aliasedRequiredInput"]
	require.True(t, ok)
	assert.Equal(t, "aliasedRequiredInput", aliasedRequiredInput.ClassPropertyName)
	assert.Equal(t, "publicRequiredName", aliasedRequiredInput.BindingPropertyName)
	assert.True(t, aliasedRequiredInput.IsSignal)
	assert.True(t, aliasedRequiredInput.Required)
}
