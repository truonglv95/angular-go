package annotations_local

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/annotations"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

type DirectiveLocalDecoratorHandler struct {
	host         reflection.ReflectionHost
	metaRegistry *metadata.LocalMetadataRegistry
}

func NewDirectiveLocalDecoratorHandler(host reflection.ReflectionHost, metaRegistry *metadata.LocalMetadataRegistry) *DirectiveLocalDecoratorHandler {
	return &DirectiveLocalDecoratorHandler{
		host:         host,
		metaRegistry: metaRegistry,
	}
}

func (h *DirectiveLocalDecoratorHandler) Name() string {
	return "DirectiveLocalDecoratorHandler"
}

func (h *DirectiveLocalDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	if decorators == nil {
		return nil
	}
	for _, dec := range decorators {
		if dec.Name == "Directive" {
			d := dec
			return &d
		}
	}
	return nil
}

func (h *DirectiveLocalDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &annotations.DirectiveAnalysis{
		IsStandalone:   true,
		Inputs:         make(map[string]render3.R3InputMetadata),
		Outputs:        make(map[string]string),
		Host:           render3.R3HostMetadata{},
		DecoratorNode:  decorator.Node,
		PropDecorators: make(map[string][]*ast.Node),
	}

	if node.Members != nil {
		for _, member := range node.Members.Nodes {
			if member.Kind == ast.KindPropertyDeclaration || member.Kind == ast.KindMethodDeclaration {
				var modifiers *ast.ModifierList
				var nameNode *ast.Node
				if member.Kind == ast.KindPropertyDeclaration {
					modifiers = member.AsPropertyDeclaration().Modifiers()
					nameNode = member.AsPropertyDeclaration().Name()
				} else {
					modifiers = member.AsMethodDeclaration().Modifiers()
					nameNode = member.AsMethodDeclaration().Name()
				}
				if modifiers != nil {
					propName := ""
					if nameNode != nil && nameNode.Kind == ast.KindIdentifier {
						propName = nameNode.AsIdentifier().Text
					}
					var propDecs []*ast.Node
					for _, mod := range modifiers.Nodes {
						if mod.Kind == ast.KindDecorator {
							propDecs = append(propDecs, mod)
							decExpr := mod.AsDecorator().Expression
							if decExpr.Kind == ast.KindCallExpression {
								callExpr := decExpr.AsCallExpression()
								if callExpr.Expression.Kind == ast.KindIdentifier {
									id := callExpr.Expression.AsIdentifier()
									if propName != "" {
										if id.Text == "Input" {
											alias := propName
											if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
												if callExpr.Arguments.Nodes[0].Kind == ast.KindStringLiteral {
													alias = callExpr.Arguments.Nodes[0].AsStringLiteral().Text
												}
											}
											analysis.Inputs[propName] = render3.R3InputMetadata{
												ClassPropertyName:   propName,
												BindingPropertyName: alias,
											}
										} else if id.Text == "Output" {
											alias := propName
											if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
												if callExpr.Arguments.Nodes[0].Kind == ast.KindStringLiteral {
													alias = callExpr.Arguments.Nodes[0].AsStringLiteral().Text
												}
											}
											analysis.Outputs[propName] = alias
										} else if id.Text == "HostBinding" {
											alias := propName
											if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
												if callExpr.Arguments.Nodes[0].Kind == ast.KindStringLiteral {
													alias = callExpr.Arguments.Nodes[0].AsStringLiteral().Text
												}
											}
											if analysis.Host.Properties == nil {
												analysis.Host.Properties = make(map[string]string)
											}
											analysis.Host.Properties[alias] = propName
										} else if id.Text == "HostListener" {
											eventName := propName
											if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
												if callExpr.Arguments.Nodes[0].Kind == ast.KindStringLiteral {
													eventName = callExpr.Arguments.Nodes[0].AsStringLiteral().Text
												}
											}
											if analysis.Host.Listeners == nil {
												analysis.Host.Listeners = make(map[string]string)
											}
											argsStr := ""
											if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 1 {
												if callExpr.Arguments.Nodes[1].Kind == ast.KindArrayLiteralExpression {
													argsStr = "$event"
												}
											}
											analysis.Host.Listeners[eventName] = propName + "(" + argsStr + ")"
										} else if id.Text == "ViewChild" || id.Text == "ViewChildren" || id.Text == "ContentChild" || id.Text == "ContentChildren" {
											isView := id.Text == "ViewChild" || id.Text == "ViewChildren"
											isFirst := id.Text == "ViewChild" || id.Text == "ContentChild"

											predicate := ""
											if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
												if callExpr.Arguments.Nodes[0].Kind == ast.KindStringLiteral {
													predicate = callExpr.Arguments.Nodes[0].AsStringLiteral().Text
												}
											}

											isStatic := false
											if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 1 {
												if callExpr.Arguments.Nodes[1].Kind == ast.KindObjectLiteralExpression {
													obj := callExpr.Arguments.Nodes[1].AsObjectLiteralExpression()
													for _, prop := range obj.Properties.Nodes {
														if prop.Kind == ast.KindPropertyAssignment {
															pa := prop.AsPropertyAssignment()
															if pa.Name() != nil && pa.Name().Kind == ast.KindIdentifier && pa.Name().AsIdentifier().Text == "static" {
																if pa.Initializer != nil && pa.Initializer.Kind == ast.KindTrueKeyword {
																	isStatic = true
																}
															}
														}
													}
												}
											}

											meta := render3.R3QueryMetadata{
												PropertyName:            propName,
												First:                   isFirst,
												Predicate:               []string{predicate},
												Descendants:             isView || id.Text == "ContentChildren",
												Static:                  isStatic,
												EmitDistinctChangesOnly: true,
											}

											if isView {
												analysis.ViewQueries = append(analysis.ViewQueries, meta)
											} else {
												analysis.Queries = append(analysis.Queries, meta)
											}
										}
									}
								}
							}
						}
					}
					if len(propDecs) > 0 && propName != "" {
						analysis.PropDecorators[propName] = propDecs
					}
				}
			}
		}
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
		case "selector":
			if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.Selector = assign.Initializer.AsStringLiteral().Text
			}
		case "standalone":
			if assign.Initializer.Kind == ast.KindTrueKeyword {
				analysis.IsStandalone = true
			} else if assign.Initializer.Kind == ast.KindFalseKeyword {
				analysis.IsStandalone = false
			}
		}
	}

	return analysis, nil
}

func (h *DirectiveLocalDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	localAnalysis := analysisData.(*annotations.DirectiveAnalysis)

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
											localAnalysis.Imports = append(localAnalysis.Imports, annotations.DirectiveImport{
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

	globalHandler := annotations.NewDirectiveDecoratorHandler(h.host, h.metaRegistry, h.metaRegistry)
	return globalHandler.Resolve(node, localAnalysis)
}

func (h *DirectiveLocalDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	globalHandler := annotations.NewDirectiveDecoratorHandler(h.host, h.metaRegistry, h.metaRegistry)
	return globalHandler.CompileFull(node, analysisData, resolutionData, pool, importMgr, factory)
}

func (h *DirectiveLocalDecoratorHandler) GetSemanticSymbol(node *ast.ClassDeclaration, analysis any) *semantic_graph.SemanticSymbol {
	globalHandler := annotations.NewDirectiveDecoratorHandler(h.host, h.metaRegistry, h.metaRegistry)
	return globalHandler.GetSemanticSymbol(node, analysis)
}


func (h *DirectiveLocalDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
}
