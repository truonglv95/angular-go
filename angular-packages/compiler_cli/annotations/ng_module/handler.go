package ng_module

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// NgModuleAnalysisData represents the analyzed metadata of an @NgModule decorator.
type NgModuleAnalysisData struct {
	Declarations []partial_evaluator.ResolvedValue
	Imports      []partial_evaluator.ResolvedValue
	Exports      []partial_evaluator.ResolvedValue
	Providers    partial_evaluator.ResolvedValue
	Bootstrap    []partial_evaluator.ResolvedValue
}

// NgModuleDecoratorHandler processes @NgModule decorators.
type NgModuleDecoratorHandler struct {
	host      reflection.ReflectionHost
	evaluator *partial_evaluator.PartialEvaluator
}

func NewNgModuleDecoratorHandler(host reflection.ReflectionHost, evaluator *partial_evaluator.PartialEvaluator) *NgModuleDecoratorHandler {
	return &NgModuleDecoratorHandler{
		host:      host,
		evaluator: evaluator,
	}
}

// Analyze analyzes a class node and its @NgModule decorator.
func (h *NgModuleDecoratorHandler) Analyze(classNode *ast.Node) (*NgModuleAnalysisData, error) {
	decorators := h.host.GetDecoratorsOfDeclaration(classNode)
	if len(decorators) == 0 {
		return nil, nil // Not a module
	}

	var ngModuleDecorator *reflection.Decorator
	for _, dec := range decorators {
		if dec.Name == "NgModule" {
			ngModuleDecorator = &dec
			break
		}
	}

	if ngModuleDecorator == nil {
		return nil, nil
	}

	// @NgModule has exactly one argument which is an ObjectLiteral
	if len(ngModuleDecorator.Args) == 0 {
		return &NgModuleAnalysisData{}, nil
	}

	arg := ngModuleDecorator.Args[0]
	resolved := h.evaluator.Evaluate(arg, nil)

	resolvedMap, ok := resolved.(partial_evaluator.ResolvedValueMap)
	if !ok {
		return &NgModuleAnalysisData{}, nil // Not an object literal
	}

	data := &NgModuleAnalysisData{}

	if declsVal, ok := resolvedMap["declarations"].(partial_evaluator.ResolvedValueArray); ok {
		data.Declarations = append(data.Declarations, declsVal...)
	}

	if importsVal, ok := resolvedMap["imports"].(partial_evaluator.ResolvedValueArray); ok {
		data.Imports = append(data.Imports, importsVal...)
	}

	if exportsVal, ok := resolvedMap["exports"].(partial_evaluator.ResolvedValueArray); ok {
		data.Exports = append(data.Exports, exportsVal...)
	}

	if bootstrapVal, ok := resolvedMap["bootstrap"].(partial_evaluator.ResolvedValueArray); ok {
		data.Bootstrap = append(data.Bootstrap, bootstrapVal...)
	}

	if providersVal, ok := resolvedMap["providers"]; ok {
		data.Providers = providersVal
	}

	return data, nil
}
