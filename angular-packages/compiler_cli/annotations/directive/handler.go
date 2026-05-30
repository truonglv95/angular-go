package directive

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// DirectiveAnalysisData represents the analyzed metadata of a @Directive decorator.
type DirectiveAnalysisData struct {
	Selector     string
	Standalone   bool
	Inputs       map[string]string
	Outputs      map[string]string
	HostBindings map[string]string
	Providers    partial_evaluator.ResolvedValue
	ExportAs     []string
}

// DirectiveDecoratorHandler processes @Directive decorators.
type DirectiveDecoratorHandler struct {
	host      reflection.ReflectionHost
	evaluator *partial_evaluator.PartialEvaluator
}

func NewDirectiveDecoratorHandler(host reflection.ReflectionHost, evaluator *partial_evaluator.PartialEvaluator) *DirectiveDecoratorHandler {
	return &DirectiveDecoratorHandler{
		host:      host,
		evaluator: evaluator,
	}
}

// Analyze analyzes a class node and its @Directive decorator.
func (h *DirectiveDecoratorHandler) Analyze(classNode *ast.Node) (*DirectiveAnalysisData, error) {
	decorators := h.host.GetDecoratorsOfDeclaration(classNode)
	if len(decorators) == 0 {
		return nil, nil // Not a directive
	}

	var dirDecorator *reflection.Decorator
	for _, dec := range decorators {
		if dec.Name == "Directive" {
			dirDecorator = &dec
			break
		}
	}

	if dirDecorator == nil {
		return nil, nil
	}

	// @Directive has exactly one argument which is an ObjectLiteral
	if len(dirDecorator.Args) == 0 {
		return nil, nil
	}

	arg := dirDecorator.Args[0]
	resolved := h.evaluator.Evaluate(arg, nil)

	resolvedMap, ok := resolved.(partial_evaluator.ResolvedValueMap)
	if !ok {
		return nil, nil // Not an object literal
	}

	data := &DirectiveAnalysisData{
		Inputs:       make(map[string]string),
		Outputs:      make(map[string]string),
		HostBindings: make(map[string]string),
	}

	if selectorVal, ok := resolvedMap["selector"].(string); ok {
		data.Selector = selectorVal
	}

	if standaloneVal, ok := resolvedMap["standalone"].(bool); ok {
		data.Standalone = standaloneVal
	}

	if inputsMap, ok := resolvedMap["inputs"].(partial_evaluator.ResolvedValueArray); ok {
		// Angular directive inputs can be an array of strings e.g. inputs: ['propName']
		for _, in := range inputsMap {
			if strVal, ok := in.(string); ok {
				data.Inputs[strVal] = strVal
			}
		}
	}

	if outputsMap, ok := resolvedMap["outputs"].(partial_evaluator.ResolvedValueArray); ok {
		for _, out := range outputsMap {
			if strVal, ok := out.(string); ok {
				data.Outputs[strVal] = strVal
			}
		}
	}

	if hostMap, ok := resolvedMap["host"].(partial_evaluator.ResolvedValueMap); ok {
		for k, v := range hostMap {
			if strVal, ok := v.(string); ok {
				data.HostBindings[k] = strVal
			}
		}
	}

	if providersVal, ok := resolvedMap["providers"]; ok {
		data.Providers = providersVal
	}

	if exportAsVal, ok := resolvedMap["exportAs"].(string); ok {
		data.ExportAs = append(data.ExportAs, exportAsVal)
	}

	return data, nil
}
