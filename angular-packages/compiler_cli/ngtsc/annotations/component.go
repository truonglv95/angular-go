package annotations

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	ngdiagnostics "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3/partial"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
)

type ComponentImport struct {
	Name string
	Decl reflection.Declaration
}

type ComponentAnalysis struct {
	Selector string
	Template string
	TemplateUrl string
	Styles []string
	StyleUrls []string
	IsStandalone bool
	Imports []ComponentImport
	Inputs map[string]render3.R3InputMetadata
	Outputs map[string]string
	Queries []render3.R3QueryMetadata
	ViewQueries []render3.R3QueryMetadata
	Host render3.R3HostMetadata
	DecoratorNode *ast.Node
	PropDecorators map[string][]*ast.Node
}

type ComponentResolution struct {
	Dependencies []render3.R3TemplateDependency
}

func (a *ComponentAnalysis) ResourceDependencies(sourceFile *ast.SourceFile) []string {
	if a == nil || sourceFile == nil {
		return nil
	}
	baseDir := filepath.Dir(sourceFile.FileName())
	var deps []string
	if a.TemplateUrl != "" {
		deps = append(deps, filepath.Clean(filepath.Join(baseDir, a.TemplateUrl)))
	}
	for _, styleUrl := range a.StyleUrls {
		if styleUrl != "" {
			deps = append(deps, filepath.Clean(filepath.Join(baseDir, styleUrl)))
		}
	}
	return deps
}

type ComponentDecoratorHandler struct {
	host          reflection.ReflectionHost
	isPartial     bool
	metaRegistry  *metadata.LocalMetadataRegistry
	scopeRegistry *scope.LocalModuleScopeRegistry
}

func NewComponentDecoratorHandler(host reflection.ReflectionHost, isPartial bool, metaRegistry *metadata.LocalMetadataRegistry, scopeRegistry *scope.LocalModuleScopeRegistry) *ComponentDecoratorHandler {
	return &ComponentDecoratorHandler{
		host:          host,
		isPartial:     isPartial,
		metaRegistry:  metaRegistry,
		scopeRegistry: scopeRegistry,
	}
}

func (h *ComponentDecoratorHandler) Name() string {
	return "ComponentDecoratorHandler"
}

func (h *ComponentDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	if decorators == nil {
		return nil
	}
	for _, dec := range decorators {
		if dec.Name == "Component" {
			d := dec
			return &d
		}
	}
	return nil
}

func (h *ComponentDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &ComponentAnalysis{
		IsStandalone: true,
		Inputs: make(map[string]render3.R3InputMetadata),
		Outputs: make(map[string]string),
		Host: render3.R3HostMetadata{},
		DecoratorNode: decorator.Node,
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
												PropertyName: propName,
												First: isFirst,
												Predicate: []string{predicate},
												Descendants: !isView,
												Static: isStatic,
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
		case "template":
			if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.Template = assign.Initializer.AsStringLiteral().Text
			} else if assign.Initializer.Kind == ast.KindNoSubstitutionTemplateLiteral {
				analysis.Template = assign.Initializer.AsNoSubstitutionTemplateLiteral().Text
			}
		case "templateUrl":
			if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.TemplateUrl = assign.Initializer.AsStringLiteral().Text
			}
		case "styles":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						if elem.Kind == ast.KindStringLiteral {
							analysis.Styles = append(analysis.Styles, elem.AsStringLiteral().Text)
						} else if elem.Kind == ast.KindNoSubstitutionTemplateLiteral {
							analysis.Styles = append(analysis.Styles, elem.AsNoSubstitutionTemplateLiteral().Text)
						}
					}
				}
			} else if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.Styles = append(analysis.Styles, assign.Initializer.AsStringLiteral().Text)
			}
		case "styleUrls":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						if elem.Kind == ast.KindStringLiteral {
							analysis.StyleUrls = append(analysis.StyleUrls, elem.AsStringLiteral().Text)
						}
					}
				}
			}
		case "styleUrl":
			if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.StyleUrls = append(analysis.StyleUrls, assign.Initializer.AsStringLiteral().Text)
			}
		case "imports":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						if elem.Kind == ast.KindIdentifier {
							decl := h.host.GetDeclarationOfIdentifier(elem)
							fmt.Printf("ComponentDecoratorHandler.Analyze -> checking import %s, decl=%v\n", elem.AsIdentifier().Text, decl)
							if decl != nil && decl.Node != nil {
								fmt.Printf("decl.Node.Kind = %d\n", decl.Node.Kind)
								analysis.Imports = append(analysis.Imports, ComponentImport{
									Name: elem.AsIdentifier().Text,
									Decl: *decl,
								})
							}
						}
					}
				}
			}
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
	
	if h.metaRegistry != nil {
		var metaImports []metadata.Reference
		for _, imp := range analysis.Imports {
			metaImports = append(metaImports, metadata.Reference{Name: imp.Name, Node: imp.Decl.Node, OwningModule: imp.Decl.ViaModule})
		}
		h.metaRegistry.RegisterDirective(node.AsNode(), &metadata.DirectiveMeta{
			Name:       node.Name().AsIdentifier().Text,
			Selector:   analysis.Selector,
			Standalone: analysis.IsStandalone,
			Imports:    metaImports,
			IsComponent: true,
		})
	}
	return analysis, nil
}

func (h *ComponentDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	analysis := analysisData.(*ComponentAnalysis)
	resolution := &ComponentResolution{}
	
	if h.scopeRegistry != nil {
		scope := h.scopeRegistry.GetCompilationScope(node.AsNode())
		if scope != nil {
			// 1. Read and parse template to AST
			sourceFile := ast.GetSourceFileOfNode(node.AsNode())
			baseDir := ""
			if sourceFile != nil {
				baseDir = filepath.Dir(sourceFile.FileName())
			}
			
			templateStr := analysis.Template
			if analysis.TemplateUrl != "" {
				templatePath := filepath.Join(baseDir, analysis.TemplateUrl)
				if content, err := ioutil.ReadFile(templatePath); err == nil {
					templateStr = string(content)
				}
			}
			_ = render3.ParseTemplate(templateStr, "", nil)

// Deduplicate dependencies
			seenDeps := make(map[string]bool)
			addDep := func(dep render3.R3TemplateDependency) {
				key := ""
				if expr, ok := dep.Type.(*output.ExternalExpr); ok {
					key = *expr.Value.ModuleName + "#" + *expr.Value.Name
				} else if expr, ok := dep.Type.(*output.ReadVarExpr); ok {
					key = expr.Name
				}
				if !seenDeps[key] {
					seenDeps[key] = true
					resolution.Dependencies = append(resolution.Dependencies, dep)
				}
			}

			// Add original imports verbatim
			for _, imp := range analysis.Imports {
				dep := render3.R3TemplateDependency{
					Kind: render3.R3TemplateDependencyKind_NgModule,
				}
				if imp.Decl.ViaModule != "" {
					moduleName := imp.Decl.ViaModule
					name := imp.Name
					dep.Type = output.NewExternalExpr(output.ExternalReference{
						ModuleName: &moduleName,
						Name:       &name,
					}, nil, nil, nil, nil)
				} else {
					dep.Type = output.NewReadVarExpr(imp.Name, nil, nil, nil)
				}
				addDep(dep)
			}

			// Add all directives in scope
			for i := range scope.Directives {
				dir := &scope.Directives[i]
				dep := render3.R3TemplateDependency{
					Kind: render3.R3TemplateDependencyKind_Directive,
				}
				if dir.Ref.OwningModule != "" {
					moduleName := dir.Ref.OwningModule
					name := dir.Ref.Name
					dep.Type = output.NewExternalExpr(output.ExternalReference{
						ModuleName: &moduleName,
						Name:       &name,
					}, nil, nil, nil, nil)
				} else {
					dep.Type = output.NewReadVarExpr(dir.Ref.Name, nil, nil, nil)
				}
				addDep(dep)
			}

			// Add all pipes in scope
			for _, pipe := range scope.Pipes {
				dep := render3.R3TemplateDependency{
					Kind: render3.R3TemplateDependencyKind_Pipe,
				}
				if pipe.Ref.OwningModule != "" {
					moduleName := pipe.Ref.OwningModule
					name := pipe.Ref.Name
					dep.Type = output.NewExternalExpr(output.ExternalReference{
						ModuleName: &moduleName,
						Name:       &name,
					}, nil, nil, nil, nil)
				} else {
					dep.Type = output.NewReadVarExpr(pipe.Name, nil, nil, nil)
				}
				addDep(dep)
			}


		}
	}
	
	return resolution, nil
}

func (h *ComponentDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	analysis := analysisData.(*ComponentAnalysis)
	resolution := resolutionData.(*ComponentResolution)
	var diagnostics []ast.Diagnostic
	
	// RESOURCE LOADER: Resolve template and styles from disk
	sourceFile := ast.GetSourceFileOfNode(node.AsNode())
	baseDir := ""
	if sourceFile != nil {
		baseDir = filepath.Dir(sourceFile.FileName())
	}
	
	if analysis.TemplateUrl != "" {
		templatePath := filepath.Join(baseDir, analysis.TemplateUrl)
		if content, err := ioutil.ReadFile(templatePath); err == nil {
			analysis.Template = string(content)
		} else {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_COMPONENT_RESOURCE_NOT_FOUND,
				node.AsNode(),
				fmt.Sprintf("Failed to load templateUrl %q.", templatePath),
				nil,
				tsdiagnostics.CategoryError,
			))
		}
	}
		
	for _, styleUrl := range analysis.StyleUrls {
		stylePath := filepath.Join(baseDir, styleUrl)
		if content, err := ioutil.ReadFile(stylePath); err == nil {
			analysis.Styles = append(analysis.Styles, string(content))
		} else {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_COMPONENT_RESOURCE_NOT_FOUND,
				node.AsNode(),
				fmt.Sprintf("Failed to load styleUrl %q.", stylePath),
				nil,
				tsdiagnostics.CategoryError,
			))
		}
	}

	parsedTemplate := render3.ParseTemplate(analysis.Template, "", nil)

	className := ""
	if node.Name() != nil && node.Name().Kind == ast.KindIdentifier {
		className = node.Name().AsIdentifier().Text
	}

	if len(parsedTemplate.Styles) > 0 {
		analysis.Styles = append(analysis.Styles, parsedTemplate.Styles...)
	}

	meta := render3.R3ComponentMetadata[render3.R3TemplateDependency]{
		R3DirectiveMetadata: render3.R3DirectiveMetadata{
			Name: className,
			Type: render3.R3Reference{
				Value: output.NewReadVarExpr(className, nil, nil, nil),
			},
			Selector:     &analysis.Selector,
			IsStandalone: analysis.IsStandalone,
			Inputs:       analysis.Inputs,
			Outputs:      analysis.Outputs,
			Host:         analysis.Host,
			Queries:      analysis.Queries,
			ViewQueries:  analysis.ViewQueries,
		},
		Template: render3.Template{
			Children: parsedTemplate.Nodes,
		},
		Declarations: resolution.Dependencies,
		Styles:       analysis.Styles,
		HasDirectiveDependencies: len(resolution.Dependencies) > 0,
	}
		var compiled render3.R3CompiledExpression
		if h.isPartial {
			// Convert Dependencies to R3TemplateDependencyMetadata
			var metaDeps []render3.R3TemplateDependencyMetadata
			for _, dep := range resolution.Dependencies {
				metaDeps = append(metaDeps, render3.R3DirectiveDependencyMetadata{
					Selector: "unknown",
					R3TemplateDependency: dep,
					IsComponent: false,
				})
			}
			metaPartial := render3.R3ComponentMetadata[render3.R3TemplateDependencyMetadata]{
				R3DirectiveMetadata: meta.R3DirectiveMetadata,
				Template: meta.Template,
				Declarations: metaDeps,
				Styles: meta.Styles,
			}
			compiled = partial.CompileDeclareComponentFromMetadata(metaPartial, parsedTemplate, partial.DeclareComponentTemplateInfo{
				Content: analysis.Template,
				IsInline: analysis.TemplateUrl == "",
			})
		} else {
			compiled = render3.CompileComponentFromMetadata(meta, pool, nil)
		}

		visitor := translator.NewExpressionTranslatorVisitor(factory, importMgr, ast.GetSourceFileOfNode(node.AsNode()).AsNode(), translator.TranslatorOptions{})
	
	var initializerNode *ast.Node
	if compiled.Expression != nil {
		initializerNode = compiled.Expression.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	}

	factoryDeps := extractDependencies(h.host, node.AsNode())

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
	
	decMapEntries := []output.LiteralMapEntry{
		output.NewLiteralMapPropertyAssignment("type", output.NewReadVarExpr("Component", nil, nil, nil), false),
	}
	if len(decArgs) > 0 {
		decMapEntries = append(decMapEntries, output.NewLiteralMapPropertyAssignment("args", output.NewLiteralArrayExpr(decArgs, nil, nil, nil), false))
	}

	var propDecorators []output.LiteralMapEntry
	for propName, decNodes := range analysis.PropDecorators {
		var decExprs []output.Expression
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
					output.NewLiteralMapPropertyAssignment("type", output.NewReadVarExpr(id, nil, nil, nil), false),
				}
				if len(pDecArgs) > 0 {
					pMap = append(pMap, output.NewLiteralMapPropertyAssignment("args", output.NewLiteralArrayExpr(pDecArgs, nil, nil, nil), false))
				}
				decExprs = append(decExprs, output.NewLiteralMapExpr(pMap, nil, nil, nil))
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

	// Class Debug Info
	var classDebugInfoNode *ast.Node
	
	// get file path and line number
	filePath := ""
	lineNum := 0
	if node.AsNode().Parent != nil && node.AsNode().Parent.Kind == ast.KindSourceFile {
		sf := node.AsNode().Parent.AsSourceFile()
		rawPath := sf.FileName()
		
		// Attempt to make path relative
		if rel, err := filepath.Rel(".", rawPath); err == nil && !strings.HasPrefix(rel, "..") {
			filePath = rel
		} else {
			// fallback: find "src/" and cut from there
			if idx := strings.Index(rawPath, "src/"); idx != -1 {
				filePath = rawPath[idx:]
			} else {
				filePath = rawPath
			}
		}
		
		// Compute line number (0-based internally, we output what ngc does)
		// Usually ngc outputs the line where the class is declared, wait. If we just count \n up to node.Pos()
		var targetPos int
		if node.Name() != nil {
			targetPos = node.Name().Pos()
		} else {
			targetPos = node.AsNode().Pos()
		}
		if targetPos > 0 {
			text := sf.Text()
			if targetPos <= len(text) {
				lineNum = strings.Count(text[:targetPos], "\n")
			}
		}
	}
	
	classDebugExpr := render3.CompileClassDebugInfo(render3.R3ClassDebugInfo{
		Type: output.NewReadVarExpr(className, nil, nil, nil),
		ClassName: output.NewLiteralExpr(className, nil, nil, nil),
		FilePath: output.NewLiteralExpr(filePath, nil, nil, nil),
		LineNumber: output.NewLiteralExpr(lineNum, nil, nil, nil),
	})
	
	if classDebugExpr != nil {
		stmt := classDebugExpr.ToStmt(nil)
		astStmt := stmt.VisitStatement(visitor, translator.Context{IsStatementMode: true})
		if astStmt != nil {
			if n, ok := astStmt.(*ast.Node); ok {
				classDebugInfoNode = n
			}
		}
	}

	// Strip Input and Output decorators from class members to avoid TS compiler panic
	if node.Members != nil {
		for _, member := range node.Members.Nodes {
				if member.Kind == ast.KindPropertyDeclaration || member.Kind == ast.KindMethodDeclaration {
					var modifiers *ast.ModifierList
					if member.Kind == ast.KindPropertyDeclaration {
						modifiers = member.AsPropertyDeclaration().Modifiers()
					} else {
						modifiers = member.AsMethodDeclaration().Modifiers()
					}
					if modifiers != nil {
						var cleanMods []*ast.Node
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
								cleanMods = append(cleanMods, mod)
							}
						}
						if len(cleanMods) == 0 {
							if member.Kind == ast.KindPropertyDeclaration {
								member.AsPropertyDeclaration().AsMutable().SetModifiers(nil)
							} else {
								member.AsMethodDeclaration().AsMutable().SetModifiers(nil)
							}
						} else {
							modifiers.Nodes = cleanMods
						}
					}
				}
		}
	}

	var extraStatements []*ast.Node
	if classMetadataNode != nil {
		extraStatements = append(extraStatements, classMetadataNode)
	}
	if classDebugInfoNode != nil {
		extraStatements = append(extraStatements, classDebugInfoNode)
	}

		return []transform.CompileResult{
			{
				PropertyName: "ɵfac",
				Initializer:  facInitializerNode,
		},
		{
			PropertyName: "ɵcmp",
			Initializer:  initializerNode,
				Statements:   extraStatements,
			},
		}, diagnostics
	}

func extractDependencies(refHost reflection.ReflectionHost, classDecl *ast.Node) interface{} {
	ctorParams := refHost.GetConstructorParameters(classDecl)
	if ctorParams == nil {
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
					dep.Token = extractTokenFromNode(dec.Args[0])
				}
			case "Attribute":
				if len(dec.Args) > 0 {
					dep.AttributeNameType = extractTokenFromNode(dec.Args[0])
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

func extractTokenFromNode(node *ast.Node) output.Expression {
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
		recv := extractTokenFromNode(pa.Expression)
		if pa.Name().Kind == ast.KindIdentifier {
			return output.NewReadPropExpr(recv, pa.Name().AsIdentifier().Text, nil, nil, nil, false)
		}
	}
	// Fallback for complex expressions
	return output.NewLiteralExpr("UNKNOWN_TOKEN", nil, nil, nil)
}
