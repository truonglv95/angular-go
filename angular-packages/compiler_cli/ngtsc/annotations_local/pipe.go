package annotations_local

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

type PipeLocalDecoratorHandler struct {
	host         reflection.ReflectionHost
	metaRegistry *metadata.LocalMetadataRegistry
}

func NewPipeLocalDecoratorHandler(host reflection.ReflectionHost, metaRegistry *metadata.LocalMetadataRegistry) *PipeLocalDecoratorHandler {
	return &PipeLocalDecoratorHandler{
		host:         host,
		metaRegistry: metaRegistry,
	}
}

func (h *PipeLocalDecoratorHandler) Name() string {
	return "PipeLocalDecoratorHandler"
}

func (h *PipeLocalDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	if decorators == nil {
		return nil
	}
	for _, dec := range decorators {
		if dec.Name == "Pipe" {
			d := dec
			return &d
		}
	}
	return nil
}

func (h *PipeLocalDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &annotations.PipeAnalysis{
		Pure:          true,
		IsStandalone:  true,
		DecoratorNode: decorator.Node,
	}

	if len(decorator.Args) == 0 {
		return analysis, nil
	}

	arg := decorator.Args[0]
	if arg.Kind != ast.KindObjectLiteralExpression {
		return analysis, nil
	}

	obj := arg.AsObjectLiteralExpression()
	if obj.Properties == nil {
		return analysis, nil
	}

	for _, prop := range obj.Properties.Nodes {
		if prop.Kind != ast.KindPropertyAssignment {
			continue
		}

		assign := prop.AsPropertyAssignment()
		name := ""
		if assign.Name().Kind == ast.KindIdentifier {
			name = assign.Name().AsIdentifier().Text
		}

		switch name {
		case "name":
			if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.Name = assign.Initializer.AsStringLiteral().Text
			}
		case "pure":
			if assign.Initializer.Kind == ast.KindTrueKeyword {
				analysis.Pure = true
			} else if assign.Initializer.Kind == ast.KindFalseKeyword {
				analysis.Pure = false
			}
		case "standalone":
			if assign.Initializer.Kind == ast.KindTrueKeyword {
				analysis.IsStandalone = true
			} else if assign.Initializer.Kind == ast.KindFalseKeyword {
				analysis.IsStandalone = false
			}
		case "imports":
			// Deferred to Resolve
		}
	}

	return analysis, nil
}

func (h *PipeLocalDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	localAnalysis := analysisData.(*annotations.PipeAnalysis)

	if localAnalysis.DecoratorNode != nil {
		decExpr := localAnalysis.DecoratorNode.AsDecorator().Expression
		if decExpr.Kind == ast.KindCallExpression {
			callExpr := decExpr.AsCallExpression()
			if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
				arg := callExpr.Arguments.Nodes[0]
				if arg.Kind == ast.KindObjectLiteralExpression {
					obj := arg.AsObjectLiteralExpression()
					for _, prop := range obj.Properties.Nodes {
						if prop.Kind == ast.KindPropertyAssignment {
							pa := prop.AsPropertyAssignment()
							name := ""
							if pa.Name().Kind == ast.KindIdentifier {
								name = pa.Name().AsIdentifier().Text
							}
							if name == "imports" && pa.Initializer.Kind == ast.KindArrayLiteralExpression {
								arr := pa.Initializer.AsArrayLiteralExpression()
								for _, elem := range arr.Elements.Nodes {
									if elem.Kind == ast.KindIdentifier {
										decl := h.host.GetDeclarationOfIdentifier(elem)
										if decl != nil && decl.Node != nil {
											localAnalysis.Imports = append(localAnalysis.Imports, annotations.PipeImport{
												Name: elem.AsIdentifier().Text,
												Decl: *decl,
											})
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	globalHandler := annotations.NewPipeDecoratorHandler(h.host, h.metaRegistry)
	return globalHandler.Resolve(node, localAnalysis)
}

func (h *PipeLocalDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	globalHandler := annotations.NewPipeDecoratorHandler(h.host, h.metaRegistry)
	return globalHandler.CompileFull(node, analysisData, resolutionData, pool, importMgr, factory)
}

func (h *PipeLocalDecoratorHandler) GetSemanticSymbol(node *ast.ClassDeclaration, analysis any) *semantic_graph.SemanticSymbol {
	globalHandler := annotations.NewPipeDecoratorHandler(h.host, h.metaRegistry)
	return globalHandler.GetSemanticSymbol(node, analysis)
}


func (h *PipeLocalDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
	analysis := analysisData.(*annotations.PipeAnalysis)
	globalHandler := annotations.NewPipeDecoratorHandler(h.host, h.metaRegistry)
	globalHandler.Register(node, analysis)
}
