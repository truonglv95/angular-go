// Package injectable_test ports annotations/test/injectable_spec.ts from ngtsc 1:1.
package injectable_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/annotations/injectable"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsctest"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopRegistry is a stub InjectableClassRegistry that does nothing.
type noopRegistry struct{}

func (r *noopRegistry) RegisterInjectable(node any, data interface{}) {}

// noopPerf is a stub PerfRecorder.
type noopPerf struct{}

func (p *noopPerf) EventCount(event string) {}

// testSetupResult holds the result of setupHandler.
type testSetupResult struct {
	handler    *injectable.InjectableDecoratorHandler
	testClass  *ast.Node
	provMember *reflection.ClassMember
	analysis   *injectable.InjectableHandlerData
}

// setupHandler mirrors the setupHandler function from injectable_spec.ts.
func setupHandler(t *testing.T, errorOnDuplicateProv bool) testSetupResult {
	t.Helper()
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded, skipping test requiring type checker")
	}

	result := ngtsctest.MakeProgram(t, []ngtsctest.ProgramFile{
		{
			Name:     "/node_modules/@angular/core/index.d.ts",
			Contents: `export const Injectable: any; export const ɵɵdefineInjectable: any`,
		},
		{
			Name: "/entry.ts",
			Contents: `
				import {Injectable, ɵɵdefineInjectable} from '@angular/core';
				export const TestClassToken = 'TestClassToken';
				@Injectable({providedIn: 'module'})
				export class TestClass {
					static ɵprov = ɵɵdefineInjectable({ factory: () => {}, token: TestClassToken, providedIn: "module" });
				}
			`,
		},
	})
	defer result.Release()

	sf := ngtsctest.RequireSourceFileByName(t, result, "entry.ts")
	testClassNode := ngtsctest.FindNamedClassDeclaration(sf, "TestClass")
	require.NotNil(t, testClassNode, "TestClass not found in entry.ts")

	host := reflection.NewTypeScriptReflectionHost(result.Checker)
	eval := partial_evaluator.NewPartialEvaluator(host, result.Checker, nil)

	h := injectable.NewInjectableDecoratorHandler(
		host,
		eval,
		false, // isCore
		false, // strictCtorDeps
		&noopRegistry{},
		&noopPerf{},
		true, // includeClassMetadata
		injectable.CompilationModeFull,
		errorOnDuplicateProv,
	)

	// Find the ɵprov member.
	members := host.GetMembersOfClass(testClassNode)
	var provMember *reflection.ClassMember
	for i := range members {
		if members[i].Name == "ɵprov" {
			m := members[i]
			provMember = &m
			break
		}
	}
	require.NotNil(t, provMember, "TestClass did not contain a ɵprov member")

	// Detect the Injectable decorator.
	decorators := host.GetDecoratorsOfDeclaration(testClassNode)
	detected := h.Detect(testClassNode, decorators)
	require.NotNil(t, detected, "Failed to recognize TestClass as Injectable")

	// Analyze.
	analysisOutput, err := h.Analyze(testClassNode, detected.Decorator)
	require.NoError(t, err, "Analyze should not return error")
	require.NotNil(t, analysisOutput, "Analyze should return AnalysisOutput")

	return testSetupResult{
		handler:    h,
		testClass:  testClassNode,
		provMember: provMember,
		analysis:   analysisOutput.Analysis,
	}
}

// TestInjectableDecoratorHandler_DuplicateProv_Error mirrors:
// it('should produce a diagnostic when injectable already has a static ɵprov property (with errorOnDuplicateProv true)')
func TestInjectableDecoratorHandler_DuplicateProv_Error(t *testing.T) {
	setup := setupHandler(t, true)
	_, err := setup.handler.CompileFull(setup.testClass, setup.analysis)
	require.Error(t, err, "CompileFull should have failed with duplicate ɵprov")
	fatalErr, ok := err.(*injectable.FatalDiagnosticError)
	require.True(t, ok, "Error should be a FatalDiagnosticError, got: %T", err)
	assert.Equal(t, injectable.ErrorCodeInjectableDuplicateProv, fatalErr.Code,
		"Should have INJECTABLE_DUPLICATE_PROV error code")
}

// TestInjectableDecoratorHandler_DuplicateProv_Skip mirrors:
// it('should not add new ɵprov property when injectable already has one (with errorOnDuplicateProv false)')
func TestInjectableDecoratorHandler_DuplicateProv_Skip(t *testing.T) {
	setup := setupHandler(t, false)
	results, err := setup.handler.CompileFull(setup.testClass, setup.analysis)
	require.NoError(t, err, "CompileFull should not return error when errorOnDuplicateProv=false")
	for _, r := range results {
		assert.NotEqual(t, "ɵprov", r.Name, "CompileFull should not add new ɵprov property")
	}
}
