package pipe

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// PipeAnalysisData represents the analyzed metadata of a @Pipe decorator.
type PipeAnalysisData struct {
	Name       string
	Pure       bool
	Standalone bool
}

// PipeDecoratorHandler processes @Pipe decorators.
type PipeDecoratorHandler struct {
	host      reflection.ReflectionHost
	evaluator *partial_evaluator.PartialEvaluator
}

func NewPipeDecoratorHandler(host reflection.ReflectionHost, evaluator *partial_evaluator.PartialEvaluator) *PipeDecoratorHandler {
	return &PipeDecoratorHandler{
		host:      host,
		evaluator: evaluator,
	}
}

// Analyze analyzes a class node and its @Pipe decorator.
func (h *PipeDecoratorHandler) Analyze(classNode *ast.Node) (*PipeAnalysisData, error) {
	decorators := h.host.GetDecoratorsOfDeclaration(classNode)
	if len(decorators) == 0 {
		return nil, nil // Not a pipe
	}

	var pipeDecorator *reflection.Decorator
	for _, dec := range decorators {
		if dec.Name == "Pipe" {
			pipeDecorator = &dec
			break
		}
	}

	if pipeDecorator == nil {
		return nil, nil
	}

	// @Pipe has exactly one argument which is an ObjectLiteral
	if len(pipeDecorator.Args) == 0 {
		return nil, nil
	}

	arg := pipeDecorator.Args[0]
	resolved := h.evaluator.Evaluate(arg, nil)

	resolvedMap, ok := resolved.(partial_evaluator.ResolvedValueMap)
	if !ok {
		return nil, nil // Not an object literal
	}

	data := &PipeAnalysisData{
		Pure: true, // Default in Angular is true
	}

	if nameVal, ok := resolvedMap["name"].(string); ok {
		data.Name = nameVal
	}

	if pureVal, ok := resolvedMap["pure"].(bool); ok {
		data.Pure = pureVal
	}

	if standaloneVal, ok := resolvedMap["standalone"].(bool); ok {
		data.Standalone = standaloneVal
	}

	return data, nil
}
