package annotations

import (
	"fmt"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
)

type DirectiveImport struct {
	Name string
	Decl reflection.Declaration
}

type DirectiveAnalysis struct {
	Selector       string
	Template       string
	IsStandalone   bool
	Imports        []DirectiveImport
	Inputs         map[string]render3.R3InputMetadata
	Outputs        map[string]string
	ExportAs       []string
	Queries        []render3.R3QueryMetadata
	ViewQueries    []render3.R3QueryMetadata
	Host           render3.R3HostMetadata
	DecoratorNode  *ast.Node
	PropDecorators map[string][]*ast.Node
	Providers      output.Expression
	HostDirectives []metadata.HostDirectiveMeta
}

type DirectiveResolution struct {
	Dependencies []render3.R3TemplateDependency
}

type DirectiveDecoratorHandler struct {
	host         reflection.ReflectionHost
	metaRegistry *metadata.LocalMetadataRegistry
}

func NewDirectiveDecoratorHandler(host reflection.ReflectionHost, metaRegistry *metadata.LocalMetadataRegistry) *DirectiveDecoratorHandler {
	return &DirectiveDecoratorHandler{
		host:         host,
		metaRegistry: metaRegistry,
	}
}

func (h *DirectiveDecoratorHandler) Name() string {
	return "DirectiveDecoratorHandler"
}

func (h *DirectiveDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
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

func (h *DirectiveDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &DirectiveAnalysis{
		IsStandalone:   true,
		Inputs:         make(map[string]render3.R3InputMetadata),
		Outputs:        make(map[string]string),
		DecoratorNode:  decorator.Node,
		PropDecorators: make(map[string][]*ast.Node),
	}

	var allMembers []*ast.Node
	currentClass := node
	for {
		if currentClass.Members != nil {
			allMembers = append(allMembers, currentClass.Members.Nodes...)
		}
		if baseExpr := h.host.GetBaseClassExpression(currentClass.AsNode()); baseExpr != nil {
			if decl := h.host.GetDeclarationOfIdentifier(baseExpr); decl != nil && decl.Node != nil {
				if decl.Node.Kind == ast.KindClassDeclaration {
					baseClassNode := decl.Node.AsClassDeclaration()
					sf := ast.GetSourceFileOfNode(baseClassNode.AsNode())
					if sf != nil && strings.HasSuffix(sf.FileName(), ".d.ts") {
						// Base class is external, use MetadataReader
						if baseMeta := h.metaRegistry.GetDirectiveMetadata(baseClassNode.AsNode()); baseMeta != nil {
							for k, v := range baseMeta.Inputs {
								analysis.Inputs[k] = render3.R3InputMetadata{
									BindingPropertyName: v,
									ClassPropertyName:   k,
									Required:            false,
								}
							}
							for k, v := range baseMeta.Outputs {
								analysis.Outputs[k] = v
							}
						}
						break
					}
					currentClass = baseClassNode
					continue
				}
			}
		}
		break
	}

	if len(allMembers) > 0 {
		for _, member := range allMembers {
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
				propName := ""
				if nameNode != nil && nameNode.Kind == ast.KindIdentifier {
					propName = nameNode.AsIdentifier().Text
				}
				if member.Kind == ast.KindPropertyDeclaration && propName != "" {
					fmt.Printf("DEBUG: Iterating propName %s\n", propName)
					if initVal := member.Initializer(); initVal != nil {
						if meta, ok := parseSignalInput(initVal, propName); ok {
							analysis.Inputs[propName] = meta
						} else if meta, ok := parseModelInput(initVal, propName); ok {
							analysis.Inputs[propName] = meta
							analysis.Outputs[propName] = meta.BindingPropertyName + "Change"
						}
					}
				}
				if modifiers != nil {
					var propDecs []*ast.Node
					for _, mod := range modifiers.Nodes {
						if mod.Kind == ast.KindDecorator {
							propDecs = append(propDecs, mod)
							decExpr := mod.AsDecorator().Expression
							if decExpr.Kind == ast.KindCallExpression {
								callExpr := decExpr.AsCallExpression()
								if callExpr.Expression.Kind == ast.KindIdentifier {
									id := callExpr.Expression.AsIdentifier()
									if id.Text == "Input" {
										fmt.Printf("DEBUG: Found Input on %s\n", propName)
										alias := propName
										required := false
										if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
											arg := callExpr.Arguments.Nodes[0]
											if arg.Kind == ast.KindStringLiteral {
												alias = arg.AsStringLiteral().Text
											} else if arg.Kind == ast.KindObjectLiteralExpression {
												obj := arg.AsObjectLiteralExpression()
												if obj.Properties != nil {
													for _, prop := range obj.Properties.Nodes {
														if prop.Kind == ast.KindPropertyAssignment {
															pa := prop.AsPropertyAssignment()
															propNameText := ""
															if pa.Name() != nil {
																if pa.Name().Kind == ast.KindIdentifier {
																	propNameText = pa.Name().AsIdentifier().Text
																} else if pa.Name().Kind == ast.KindStringLiteral {
																	propNameText = pa.Name().AsStringLiteral().Text
																}
															}
															if propNameText == "alias" && pa.Initializer != nil && pa.Initializer.Kind == ast.KindStringLiteral {
																alias = pa.Initializer.AsStringLiteral().Text
															} else if propNameText == "required" && pa.Initializer != nil && pa.Initializer.Kind == ast.KindTrueKeyword {
																required = true
															}
														}
													}
												}
											}
										}
										analysis.Inputs[propName] = render3.R3InputMetadata{
											BindingPropertyName: alias,
											ClassPropertyName:   propName,
											Required:            required,
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

										var predicate interface{} = []string{""}
										if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
											predicate = parseQueryPredicate(callExpr.Arguments.Nodes[0])
										}
										isStatic := false
										if callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 1 && callExpr.Arguments.Nodes[1].Kind == ast.KindObjectLiteralExpression {
											for _, p := range callExpr.Arguments.Nodes[1].AsObjectLiteralExpression().Properties.Nodes {
												if p.Kind == ast.KindPropertyAssignment {
													pa := p.AsPropertyAssignment()
													if pa.Name() != nil && pa.Name().Kind == ast.KindIdentifier && pa.Name().AsIdentifier().Text == "static" {
														if pa.Initializer != nil && pa.Initializer.Kind == ast.KindTrueKeyword {
															isStatic = true
														}
													}
												}
											}
										}

										meta := render3.R3QueryMetadata{
											PropertyName:            propName,
											First:                   isFirst,
											Predicate:               predicate,
											Descendants:             isView || id.Text == "ContentChild",
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
					if len(propDecs) > 0 {
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
		case "exportAs":
			if assign.Initializer.Kind == ast.KindStringLiteral {
				val := assign.Initializer.AsStringLiteral().Text
				parts := strings.Split(val, ",")
				for _, part := range parts {
					analysis.ExportAs = append(analysis.ExportAs, strings.TrimSpace(part))
				}
			}
		case "standalone":
			if assign.Initializer.Kind == ast.KindTrueKeyword {
				analysis.IsStandalone = true
			} else if assign.Initializer.Kind == ast.KindFalseKeyword {
				analysis.IsStandalone = false
			}
		case "template":
			if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.Template = assign.Initializer.AsStringLiteral().Text
			} else if assign.Initializer.Kind == ast.KindNoSubstitutionTemplateLiteral {
				analysis.Template = assign.Initializer.AsNoSubstitutionTemplateLiteral().Text
			}
		case "inputs":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						if elem.Kind == ast.KindStringLiteral {
							strVal := elem.AsStringLiteral().Text
							parts := strings.Split(strVal, ":")
							classProp := strings.TrimSpace(parts[0])
							bindingProp := classProp
							if len(parts) > 1 {
								bindingProp = strings.TrimSpace(parts[1])
							}
							analysis.Inputs[classProp] = render3.R3InputMetadata{
								ClassPropertyName:   classProp,
								BindingPropertyName: bindingProp,
								Required:            false,
							}
						}
					}
				}
			}
		case "outputs":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						if elem.Kind == ast.KindStringLiteral {
							strVal := elem.AsStringLiteral().Text
							parts := strings.Split(strVal, ":")
							classProp := strings.TrimSpace(parts[0])
							bindingProp := classProp
							if len(parts) > 1 {
								bindingProp = strings.TrimSpace(parts[1])
							}
							analysis.Outputs[classProp] = bindingProp
						}
					}
				}
			}
		case "imports":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						if elem.Kind == ast.KindIdentifier {
							decl := h.host.GetDeclarationOfIdentifier(elem)
							if decl != nil && decl.Node != nil {
								analysis.Imports = append(analysis.Imports, DirectiveImport{
									Name: elem.AsIdentifier().Text,
									Decl: *decl,
								})
							}
						}
					}
				}
			}
		case "hostDirectives":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						hd := metadata.HostDirectiveMeta{}
						if elem.Kind == ast.KindIdentifier {
							decl := h.host.GetDeclarationOfIdentifier(elem)
							if decl != nil && decl.Node != nil {
								hd.Directive = metadata.Reference{Name: elem.AsIdentifier().Text, Node: decl.Node, OwningModule: decl.ViaModule}
							}
						} else if elem.Kind == ast.KindObjectLiteralExpression {
							obj := elem.AsObjectLiteralExpression()
							if obj.Properties != nil {
								for _, p := range obj.Properties.Nodes {
									if p.Kind == ast.KindPropertyAssignment {
										pa := p.AsPropertyAssignment()
										if pa.Name().Kind == ast.KindIdentifier {
											pName := pa.Name().AsIdentifier().Text
											if pName == "directive" && pa.Initializer.Kind == ast.KindIdentifier {
												decl := h.host.GetDeclarationOfIdentifier(pa.Initializer)
												if decl != nil && decl.Node != nil {
													hd.Directive = metadata.Reference{Name: pa.Initializer.AsIdentifier().Text, Node: decl.Node, OwningModule: decl.ViaModule}
												}
											} else if pName == "inputs" && pa.Initializer.Kind == ast.KindArrayLiteralExpression {
												hd.Inputs = make(map[string]string)
												for _, ip := range pa.Initializer.AsArrayLiteralExpression().Elements.Nodes {
													if ip.Kind == ast.KindStringLiteral {
														parts := strings.Split(ip.AsStringLiteral().Text, ":")
														internalName := strings.TrimSpace(parts[0])
														publicName := internalName
														if len(parts) > 1 {
															publicName = strings.TrimSpace(parts[1])
														}
														hd.Inputs[internalName] = publicName
													}
												}
											} else if pName == "outputs" && pa.Initializer.Kind == ast.KindArrayLiteralExpression {
												hd.Outputs = make(map[string]string)
												for _, op := range pa.Initializer.AsArrayLiteralExpression().Elements.Nodes {
													if op.Kind == ast.KindStringLiteral {
														parts := strings.Split(op.AsStringLiteral().Text, ":")
														internalName := strings.TrimSpace(parts[0])
														publicName := internalName
														if len(parts) > 1 {
															publicName = strings.TrimSpace(parts[1])
														}
														hd.Outputs[internalName] = publicName
													}
												}
											}
										}
									}
								}
							}
						}
						if hd.Directive.Node != nil {
							analysis.HostDirectives = append(analysis.HostDirectives, hd)
						}
					}
				}
			}
		case "providers":
			analysis.Providers = output.NewWrappedNodeExpr(assign.Initializer, nil, nil, nil)
		case "host":
			if assign.Initializer.Kind == ast.KindObjectLiteralExpression {
				hostMap := make(map[string]interface{})
				hostObj := assign.Initializer.AsObjectLiteralExpression()
				if hostObj.Properties != nil {
					for _, hostProp := range hostObj.Properties.Nodes {
						if hostProp.Kind == ast.KindPropertyAssignment {
							pa := hostProp.AsPropertyAssignment()

							keyName := ""
							if pa.Name().Kind == ast.KindIdentifier {
								keyName = pa.Name().AsIdentifier().Text
							} else if pa.Name().Kind == ast.KindStringLiteral {
								keyName = pa.Name().AsStringLiteral().Text
							} else if pa.Name().Kind == ast.KindNoSubstitutionTemplateLiteral {
								keyName = pa.Name().AsNoSubstitutionTemplateLiteral().Text
							}

							valStr := ""
							if pa.Initializer.Kind == ast.KindStringLiteral {
								valStr = pa.Initializer.AsStringLiteral().Text
							} else if pa.Initializer.Kind == ast.KindNoSubstitutionTemplateLiteral {
								valStr = pa.Initializer.AsNoSubstitutionTemplateLiteral().Text
							}

							if keyName != "" && valStr != "" {
								hostMap[keyName] = valStr
							}
						}
					}
				}
				parsedHost := render3.ParseHostBindings(hostMap)
				if parsedHost.SpecialAttributes.StyleAttr != nil {
					analysis.Host.SpecialAttributes.StyleAttr = parsedHost.SpecialAttributes.StyleAttr
				}
				if parsedHost.SpecialAttributes.ClassAttr != nil {
					analysis.Host.SpecialAttributes.ClassAttr = parsedHost.SpecialAttributes.ClassAttr
				}
				for k, v := range parsedHost.Properties {
					if analysis.Host.Properties == nil {
						analysis.Host.Properties = make(map[string]string)
					}
					analysis.Host.Properties[k] = v
				}
				for k, v := range parsedHost.Listeners {
					if analysis.Host.Listeners == nil {
						analysis.Host.Listeners = make(map[string]string)
					}
					analysis.Host.Listeners[k] = v
				}
				for k, v := range parsedHost.Attributes {
					if analysis.Host.Attributes == nil {
						analysis.Host.Attributes = make(map[string]render3.Expression)
					}
					analysis.Host.Attributes[k] = v
				}
			}
		}
	}

	sort.SliceStable(analysis.Queries, func(i, j int) bool {
		return analysis.Queries[i].First && !analysis.Queries[j].First
	})
	sort.SliceStable(analysis.ViewQueries, func(i, j int) bool {
		return analysis.ViewQueries[i].First && !analysis.ViewQueries[j].First
	})
	return analysis, nil
}

func (h *DirectiveDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
	analysis := analysisData.(*DirectiveAnalysis)
	if h.metaRegistry != nil {
		var metaImports []metadata.Reference
		for _, imp := range analysis.Imports {
			metaImports = append(metaImports, metadata.Reference{Name: imp.Name, Node: imp.Decl.Node, OwningModule: imp.Decl.ViaModule})
		}
		metaInputs := make(map[string]string, len(analysis.Inputs))
		var reqInputs []string
		for prop, input := range analysis.Inputs {
			metaInputs[prop] = input.BindingPropertyName
			if input.Required {
				reqInputs = append(reqInputs, input.BindingPropertyName)
			}
		}
		h.metaRegistry.RegisterDirective(node.AsNode(), &metadata.DirectiveMeta{
			Name:        node.Name().AsIdentifier().Text,
			Selector:    analysis.Selector,
			Standalone:  analysis.IsStandalone,
			Imports:     metaImports,
			IsComponent: false,
			Inputs:      metaInputs,
			Outputs:     analysis.Outputs,
			ExportAs:    analysis.ExportAs,
			RequiredInputs: reqInputs,
			HostDirectives: analysis.HostDirectives,
			Ref: metadata.Reference{
				Name: node.Name().AsIdentifier().Text,
				Node: node.AsNode(),
			},
		})
	}
}

func (h *DirectiveDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	analysis := analysisData.(*DirectiveAnalysis)
	resolution := &DirectiveResolution{}

	for _, imp := range analysis.Imports {
		// In a real implementation we would match it against the template,
		// but here we just blindly add it to dependencies for demo purposes.
		resolution.Dependencies = append(resolution.Dependencies, render3.R3TemplateDependency{
			Kind: render3.R3TemplateDependencyKind_Directive,
			Type: output.NewReadVarExpr(imp.Name, nil, nil, nil),
		})
	}

	return resolution, nil
}

func (h *DirectiveDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	analysis := analysisData.(*DirectiveAnalysis)

	className := ""
	if node.Name() != nil && node.Name().Kind == ast.KindIdentifier {
		className = node.Name().AsIdentifier().Text
	}

	inputsOrder, outputsOrder := getPropDeclarationOrder(node)

	meta := render3.R3DirectiveMetadata{
		Name: className,
		Type: render3.R3Reference{
			Value: output.NewReadVarExpr(className, nil, nil, nil),
		},
		TypeArgumentCount: 0,
		TypeSourceSpan:    nil,
		Selector:          &analysis.Selector,
		Inputs:            analysis.Inputs,
		InputProperties:   inputsOrder,
		Outputs:           analysis.Outputs,
		OutputProperties:  outputsOrder,
		Host:              analysis.Host,
		Queries:           analysis.Queries,
		ViewQueries:       analysis.ViewQueries,
		IsStandalone:      analysis.IsStandalone,
		Providers:         analysis.Providers,
		HostDirectives:    render3HostDirectives(analysis.HostDirectives),
	}

	compiled := render3.CompileDirectiveFromMetadata(meta, pool, nil)

	visitor := translator.NewExpressionTranslatorVisitor(factory, importMgr, ast.GetSourceFileOfNode(node.AsNode()).AsNode(), translator.TranslatorOptions{})

	var initializerNode *ast.Node
	if compiled.Expression != nil {
		initializerNode = compiled.Expression.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	}

	factoryDeps := extractDependenciesDir(h.host, node.AsNode())

	facMeta := render3.R3ConstructorFactoryMetadata{
		Name: className,
		Type: render3.R3Reference{
			Value: output.NewReadVarExpr(className, nil, nil, nil),
		},
		Target: render3.FactoryTargetComponent,
		Deps:   factoryDeps,
	}
	facExpr := render3.CompileFactoryFunction(facMeta)

	var facInitializerNode *ast.Node
	if facExpr.Expression != nil {
		facInitializerNode = facExpr.Expression.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	}

	// Class Metadata
	var classMetadataNode *ast.Node
	var decArgs []output.Expression
	if analysis.DecoratorNode != nil {
		callExpr := analysis.DecoratorNode.AsDecorator().Expression.AsCallExpression()
		if callExpr != nil && callExpr.Arguments != nil && len(callExpr.Arguments.Nodes) > 0 {
			decArgs = append(decArgs, output.NewWrappedNodeExpr(callExpr.Arguments.Nodes[0], nil, nil, nil))
		}
	}

	coreModule := "@angular/core"
	directiveName := "Directive"
	decMapEntries := []output.LiteralMapEntry{
		output.NewLiteralMapPropertyAssignment("type", output.NewExternalExpr(output.ExternalReference{
			ModuleName: &coreModule,
			Name:       &directiveName,
		}, nil, nil, nil, nil), false),
	}
	if len(decArgs) > 0 {
		decMapEntries = append(decMapEntries, output.NewLiteralMapPropertyAssignment("args", output.NewLiteralArrayExpr(decArgs, nil, nil, nil), false))
	}

	propNamesMap := make(map[string]bool)
	for propName := range analysis.PropDecorators {
		propNamesMap[propName] = true
	}
	for propName, inputMeta := range analysis.Inputs {
		if inputMeta.IsSignal {
			propNamesMap[propName] = true
		}
	}

	var sortedPropNames []string
	for propName := range propNamesMap {
		sortedPropNames = append(sortedPropNames, propName)
	}
	if len(inputsOrder) > 0 {
		orderMap := make(map[string]int)
		for i, prop := range inputsOrder {
			orderMap[prop] = i
		}
		sort.Slice(sortedPropNames, func(i, j int) bool {
			idxI, okI := orderMap[sortedPropNames[i]]
			idxJ, okJ := orderMap[sortedPropNames[j]]
			if okI && okJ {
				return idxI < idxJ
			}
			if okI {
				return true
			}
			if okJ {
				return false
			}
			return sortedPropNames[i] < sortedPropNames[j]
		})
	} else {
		sort.Strings(sortedPropNames)
	}

	var propDecorators []output.LiteralMapEntry
	for _, propName := range sortedPropNames {
		var decExprs []output.Expression

		// 1. Process actual decorators
		if decNodes, exists := analysis.PropDecorators[propName]; exists {
			for _, decNode := range decNodes {
				callExpr := decNode.AsDecorator().Expression.AsCallExpression()
				if callExpr != nil && callExpr.Expression.Kind == ast.KindIdentifier {
					id := callExpr.Expression.AsIdentifier().Text
					var pDecArgs []output.Expression
					if callExpr.Arguments != nil {
						for _, arg := range callExpr.Arguments.Nodes {
							pDecArgs = append(pDecArgs, output.NewWrappedNodeExpr(arg, nil, nil, nil))
						}
					}
					pMap := []output.LiteralMapEntry{
						output.NewLiteralMapPropertyAssignment("type", output.NewExternalExpr(output.ExternalReference{
							ModuleName: &coreModule,
							Name:       &id,
						}, nil, nil, nil, nil), false),
					}
					if len(pDecArgs) > 0 {
						pMap = append(pMap, output.NewLiteralMapPropertyAssignment("args", output.NewLiteralArrayExpr(pDecArgs, nil, nil, nil), false))
					}
					decExprs = append(decExprs, output.NewLiteralMapExpr(pMap, nil, nil, nil))
				}
			}
		}

		// 2. Process virtual decorators for signal inputs/models
		if inputMeta, exists := analysis.Inputs[propName]; exists && inputMeta.IsSignal {
			inputDecName := "Input"
			inputArgsMap := []output.LiteralMapEntry{
				output.NewLiteralMapPropertyAssignment("isSignal", output.NewLiteralExpr(true, nil, nil, nil), false),
				output.NewLiteralMapPropertyAssignment("alias", output.NewLiteralExpr(inputMeta.BindingPropertyName, nil, nil, nil), false),
				output.NewLiteralMapPropertyAssignment("required", output.NewLiteralExpr(inputMeta.Required, nil, nil, nil), false),
			}
			inputDecExpr := output.NewLiteralMapExpr([]output.LiteralMapEntry{
				output.NewLiteralMapPropertyAssignment("type", output.NewExternalExpr(output.ExternalReference{
					ModuleName: &coreModule,
					Name:       &inputDecName,
				}, nil, nil, nil, nil), false),
				output.NewLiteralMapPropertyAssignment("args", output.NewLiteralArrayExpr([]output.Expression{
					output.NewLiteralMapExpr(inputArgsMap, nil, nil, nil),
				}, nil, nil, nil), false),
			}, nil, nil, nil)
			decExprs = append(decExprs, inputDecExpr)

			if outputName, ok := analysis.Outputs[propName]; ok {
				outputDecName := "Output"
				outputDecExpr := output.NewLiteralMapExpr([]output.LiteralMapEntry{
					output.NewLiteralMapPropertyAssignment("type", output.NewExternalExpr(output.ExternalReference{
						ModuleName: &coreModule,
						Name:       &outputDecName,
					}, nil, nil, nil, nil), false),
					output.NewLiteralMapPropertyAssignment("args", output.NewLiteralArrayExpr([]output.Expression{
						output.NewLiteralExpr(outputName, nil, nil, nil),
					}, nil, nil, nil), false),
				}, nil, nil, nil)
				decExprs = append(decExprs, outputDecExpr)
			}
		}

		if len(decExprs) > 0 {
			propDecorators = append(propDecorators, output.NewLiteralMapPropertyAssignment(propName, output.NewLiteralArrayExpr(decExprs, nil, nil, nil), false))
		}
	}

	var propDecExpr output.Expression
	if len(propDecorators) > 0 {
		propDecExpr = output.NewLiteralMapExpr(propDecorators, nil, nil, nil)
	}

	classMetaExpr := render3.CompileClassMetadata(render3.R3ClassMetadata{
		Type: output.NewReadVarExpr(className, nil, nil, nil),
		Decorators: output.NewLiteralArrayExpr([]output.Expression{
			output.NewLiteralMapExpr(decMapEntries, nil, nil, nil),
		}, nil, nil, nil),
		PropDecorators: propDecExpr,
	})

	if classMetaExpr != nil {
		stmt := classMetaExpr.ToStmt(nil)
		astStmt := stmt.VisitStatement(visitor, translator.Context{IsStatementMode: true})
		if astStmt != nil {
			if n, ok := astStmt.(*ast.Node); ok {
				classMetadataNode = n
			}
		}
	}

	// Strip Input and Output decorators from class members to avoid TS compiler panic
	if node.Members != nil {
		for _, member := range node.Members.Nodes {
			if member == nil {
				continue
			}
			if member.Kind == ast.KindPropertyDeclaration || member.Kind == ast.KindMethodDeclaration {
				var modifiers *ast.ModifierList
				if member.Kind == ast.KindPropertyDeclaration {
					modifiers = member.AsPropertyDeclaration().Modifiers()
				} else {
					modifiers = member.AsMethodDeclaration().Modifiers()
				}
				if modifiers != nil {
					var keptModifiers []*ast.Node
					for _, mod := range modifiers.Nodes {
						keep := true
						if mod.Kind == ast.KindDecorator {
							decExpr := mod.AsDecorator().Expression
							if decExpr.Kind == ast.KindCallExpression {
								callExpr := decExpr.AsCallExpression()
								if callExpr.Expression.Kind == ast.KindIdentifier {
									id := callExpr.Expression.AsIdentifier()
									if id.Text == "Input" || id.Text == "Output" || id.Text == "HostBinding" || id.Text == "HostListener" || id.Text == "ViewChild" || id.Text == "ContentChild" {
										keep = false
									}
								}
							}
						}
						if keep {
							keptModifiers = append(keptModifiers, mod)
						}
					}

					if len(keptModifiers) != len(modifiers.Nodes) {
						if len(keptModifiers) == 0 {
							if member.Kind == ast.KindPropertyDeclaration {
								member.AsPropertyDeclaration().AsMutable().SetModifiers(nil)
							} else {
								member.AsMethodDeclaration().AsMutable().SetModifiers(nil)
							}
						} else {
							modifiers.Nodes = keptModifiers
						}
					}
				}
			}
		}
	}

	var extraStatements []*ast.Node
	if classMetadataNode != nil {
		extraStatements = append(extraStatements, classMetadataNode)
	}


	return []transform.CompileResult{
		{
			PropertyName: "ɵfac",
			Initializer:  facInitializerNode,
		},
		{
			PropertyName: "ɵdir",
			Initializer:  initializerNode,
			Statements:   extraStatements,
		},
	}, nil
}

func extractDependenciesDir(refHost reflection.ReflectionHost, classDecl *ast.Node) interface{} {
	ctorParams := refHost.GetConstructorParameters(classDecl)
	if ctorParams == nil {
		if refHost.HasBaseClass(classDecl) {
			return nil
		}
		return []render3.R3DependencyMetadata{}
		return nil
	}

	deps := make([]render3.R3DependencyMetadata, len(ctorParams))
	for i, param := range ctorParams {
		dep := render3.R3DependencyMetadata{}

		for _, dec := range param.Decorators {
			switch dec.Name {
			case "Optional":
				dep.Optional = true
			case "Self":
				dep.Self = true
			case "SkipSelf":
				dep.SkipSelf = true
			case "Host":
				dep.Host = true
			case "Inject":
				if len(dec.Args) > 0 {
					dep.Token = extractTokenFromNodeDir(dec.Args[0])
				}
			case "Attribute":
				if len(dec.Args) > 0 {
					dep.AttributeNameType = extractTokenFromNodeDir(dec.Args[0])
				}
			}
		}

		if dep.Token == nil {
			switch typeRef := param.TypeValueReference.(type) {
			case *reflection.ImportedTypeValueReference:
				moduleName := typeRef.ModuleName
				importedName := typeRef.ImportedName
				dep.Token = output.NewExternalExpr(output.ExternalReference{
					ModuleName: &moduleName,
					Name:       &importedName,
				}, nil, nil, nil, nil)
			case *reflection.LocalTypeValueReference:
				if ast.IsIdentifier(typeRef.Expression) {
					identName := typeRef.Expression.AsIdentifier().Text
					dep.Token = output.NewReadVarExpr(identName, nil, nil, nil)
				} else {
					dep.Token = output.NewLiteralExpr("LOCAL_UNKNOWN", nil, nil, nil)
				}
			case *reflection.UnavailableTypeValueReference:
				dep.Token = output.NewLiteralExpr("INVALID_TOKEN", nil, nil, nil)
			default:
				dep.Token = output.NewLiteralExpr(nil, nil, nil, nil)
			}
		}

		deps[i] = dep
	}
	return deps
}

func extractTokenFromNodeDir(node *ast.Node) output.Expression {
	if node == nil {
		return output.NewLiteralExpr(nil, nil, nil, nil)
	}
	if ast.IsStringLiteral(node) {
		return output.NewLiteralExpr(node.AsStringLiteral().Text, nil, nil, nil)
	}
	if ast.IsIdentifier(node) {
		return output.NewReadVarExpr(node.AsIdentifier().Text, nil, nil, nil)
	}
	if ast.IsPropertyAccessExpression(node) {
		pa := node.AsPropertyAccessExpression()
		recv := extractTokenFromNodeDir(pa.Expression)
		if pa.Name().Kind == ast.KindIdentifier {
			return output.NewReadPropExpr(recv, pa.Name().AsIdentifier().Text, nil, nil, nil, false)
		}
	}
	// Fallback for complex expressions
	return output.NewLiteralExpr("UNKNOWN_TOKEN", nil, nil, nil)
}

func (h *DirectiveDecoratorHandler) GetSemanticSymbol(node *ast.ClassDeclaration, analysis any) *semantic_graph.SemanticSymbol {
	a, ok := analysis.(*DirectiveAnalysis)
	if !ok || a == nil {
		return nil
	}

	inputs := make(map[string]string)
	for prop, meta := range a.Inputs {
		inputs[prop] = meta.BindingPropertyName
	}

	var imps []string
	for _, imp := range a.Imports {
		imps = append(imps, imp.Name)
	}

	var tps []string
	for _, tp := range semantic_graph.ExtractSemanticTypeParameters(node.AsNode()) {
		tps = append(tps, tp.GetName())
	}

	return &semantic_graph.SemanticSymbol{
		Path:           ast.GetSourceFileOfNode(node.AsNode()).FileName(),
		Identifier:     node.Name().AsIdentifier().Text,
		Kind:           "directive",
		Selector:       a.Selector,
		Inputs:         inputs,
		Outputs:        a.Outputs,
		ExportAs:       a.ExportAs,
		Standalone:     a.IsStandalone,
		Imports:        imps,
		TypeParameters: tps,
	}
}

func getPropDeclarationOrder(node *ast.ClassDeclaration) ([]string, []string) {
	var inputs []string
	var outputs []string
	if node.Members != nil {
		for _, member := range node.Members.Nodes {
			if member.Kind == ast.KindPropertyDeclaration || member.Kind == ast.KindMethodDeclaration {
				propName := ""
				var nameNode *ast.Node
				if member.Kind == ast.KindPropertyDeclaration {
					nameNode = member.AsPropertyDeclaration().Name()
				} else {
					nameNode = member.AsMethodDeclaration().Name()
				}
				if nameNode != nil && nameNode.Kind == ast.KindIdentifier {
					propName = nameNode.AsIdentifier().Text
				}
				if propName != "" {
					inputs = append(inputs, propName)
					outputs = append(outputs, propName)
				}
			}
		}
	}
	return inputs, outputs
}

func render3HostDirectives(hds []metadata.HostDirectiveMeta) []render3.R3HostDirectiveMetadata {
	var result []render3.R3HostDirectiveMetadata
	for _, hd := range hds {
		r3hd := render3.R3HostDirectiveMetadata{
			Directive: render3.R3Reference{
				Value: output.NewReadVarExpr(hd.Directive.Name, nil, nil, nil),
				Type:  output.NewReadVarExpr(hd.Directive.Name, nil, nil, nil),
			},
			IsForwardReference: hd.IsForwardReference,
			Inputs:             hd.Inputs,
			Outputs:            hd.Outputs,
		}
		result = append(result, r3hd)
	}
	return result
}
