package component

import (
	"fmt"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// ComponentAnalysisData represents the analyzed metadata of an @Component decorator.
type ComponentAnalysisData struct {
	Selector     string
	Standalone   bool
	Template     string // For simplicity, we just store it as string for now
	TemplateUrl  string
	Styles       []string
	StyleUrls    []string
	HostBindings map[string]string
	Providers    partial_evaluator.ResolvedValue
	Animations   partial_evaluator.ResolvedValue
	Imports      []partial_evaluator.ResolvedValue
}

// ComponentDecoratorHandler processes @Component decorators.
type ComponentDecoratorHandler struct {
	host      reflection.ReflectionHost
	evaluator *partial_evaluator.PartialEvaluator
}

func NewComponentDecoratorHandler(host reflection.ReflectionHost, evaluator *partial_evaluator.PartialEvaluator) *ComponentDecoratorHandler {
	return &ComponentDecoratorHandler{
		host:      host,
		evaluator: evaluator,
	}
}

// Analyze analyzes a class node and its @Component decorator.
func (h *ComponentDecoratorHandler) Analyze(classNode *ast.Node) (*ComponentAnalysisData, error) {
	decorators := h.host.GetDecoratorsOfDeclaration(classNode)
	if len(decorators) == 0 {
		return nil, nil // Not a component
	}

	var compDecorator *reflection.Decorator
	for _, dec := range decorators {
		if dec.Name == "Component" {
			compDecorator = &dec
			break
		}
	}

	if compDecorator == nil {
		return nil, nil
	}

	// In Angular, @Component has exactly one argument which is an ObjectLiteral
	if len(compDecorator.Args) == 0 {
		return nil, nil
	}

	arg := compDecorator.Args[0]

	// Use Partial Evaluator to statically resolve the argument
	resolved := h.evaluator.Evaluate(arg, legacyAnimationTriggerResolver)

	resolvedMap, ok := resolved.(partial_evaluator.ResolvedValueMap)
	if !ok {
		return nil, nil // Not an object literal
	}

	data := &ComponentAnalysisData{}

	if selectorVal, ok := resolvedMap["selector"].(string); ok {
		data.Selector = selectorVal
	}

	if standaloneVal, ok := resolvedMap["standalone"].(bool); ok {
		data.Standalone = standaloneVal
	}

	if templateVal, ok := resolvedMap["template"].(string); ok {
		data.Template = templateVal
	}

	if templateUrlVal, ok := resolvedMap["templateUrl"].(string); ok {
		data.TemplateUrl = templateUrlVal
	}

	if stylesVal, ok := resolvedMap["styles"]; ok {
		flattenStyles(stylesVal, &data.Styles)
	}

	if styleUrlsArr, ok := resolvedMap["styleUrls"].(partial_evaluator.ResolvedValueArray); ok {
		for _, s := range styleUrlsArr {
			if strVal, ok := s.(string); ok {
				data.StyleUrls = append(data.StyleUrls, strVal)
			}
		}
	}

	if hostMap, ok := resolvedMap["host"].(partial_evaluator.ResolvedValueMap); ok {
		data.HostBindings = make(map[string]string)
		for k, v := range hostMap {
			if strVal, ok := v.(string); ok {
				data.HostBindings[k] = strVal
			}
		}
	}

	if providersVal, ok := resolvedMap["providers"]; ok {
		data.Providers = providersVal
	}

	if animationsVal, ok := resolvedMap["animations"]; ok {
		data.Animations = animationsVal
	}

	if importsVal, ok := resolvedMap["imports"]; ok {
		if !data.Standalone {
			return nil, fmt.Errorf("'imports' is only valid on a component that is standalone")
		}
		var imports []partial_evaluator.ResolvedValue
		if err := validateAndFlattenImports(importsVal, &imports); err != nil {
			return nil, err
		}
		data.Imports = imports
	}

	return data, nil
}
