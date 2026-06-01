package annotations

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3/partial"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	ngdiagnostics "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
)

type ComponentImport struct {
	Name       string
	ImportName string
	ImportPath string
	Decl       reflection.Declaration
}

type ComponentAnalysis struct {
	Selector       string
	Template       string
	TemplateUrl    string
	Styles         []string
	StyleUrls      []string
	IsStandalone   bool
	Imports        []ComponentImport
	Inputs         map[string]render3.R3InputMetadata
	Outputs        map[string]string
	Queries        []render3.R3QueryMetadata
	ViewQueries    []render3.R3QueryMetadata
	Host           render3.R3HostMetadata
	Animations     output.Expression
	DecoratorNode  *ast.Node
	PropDecorators map[string][]*ast.Node
}

type ComponentResolution struct {
	Dependencies []render3.R3TemplateDependency
}

type directiveMetaAdapter struct {
	meta *metadata.DirectiveMeta
}

func (d directiveMetaAdapter) GetName() string { return d.meta.Name }
func (d directiveMetaAdapter) GetRefKey() string {
	if d.meta.Ref.OwningModule != "" {
		return d.meta.Ref.OwningModule + "#" + d.meta.Ref.Name
	}
	return d.meta.Ref.Name
}
func (d directiveMetaAdapter) GetSelector() *string {
	if d.meta.Selector == "" {
		return nil
	}
	return &d.meta.Selector
}
func (d directiveMetaAdapter) IsComponent() bool { return d.meta.IsComponent }
func (d directiveMetaAdapter) GetInputs() any {
	obj := make(map[string]interface{}, len(d.meta.Inputs))
	for className, bindingName := range d.meta.Inputs {
		obj[className] = bindingName
	}
	return render3.ClassPropertyMappingFromMappedObject(obj)
}
func (d directiveMetaAdapter) GetOutputs() any {
	obj := make(map[string]interface{}, len(d.meta.Outputs))
	for className, bindingName := range d.meta.Outputs {
		obj[className] = bindingName
	}
	return render3.ClassPropertyMappingFromMappedObject(obj)
}
func (d directiveMetaAdapter) GetExportAs() []string { return d.meta.ExportAs }
func (d directiveMetaAdapter) IsStructural() bool    { return strings.HasPrefix(d.meta.Selector, "[") }
func (d directiveMetaAdapter) GetNgContentSelectors() []string {
	return nil
}
func (d directiveMetaAdapter) GetPreserveWhitespaces() bool { return false }
func (d directiveMetaAdapter) GetAnimationTriggerNames() *render3.LegacyAnimationTriggerNames {
	return nil
}
func (d directiveMetaAdapter) GetMatchSource() render3.MatchSource {
	return render3.MatchSourceSelector
}

type pipeUsageCollector struct {
	expression_parser.RecursiveAstVisitor
	used map[string]bool
}

func newPipeUsageCollector() *pipeUsageCollector {
	collector := &pipeUsageCollector{used: make(map[string]bool)}
	collector.RecursiveAstVisitor.Impl = collector
	return collector
}

func (c *pipeUsageCollector) VisitPipe(ast *expression_parser.BindingPipe, context any) any {
	c.used[ast.Name] = true
	return c.RecursiveAstVisitor.VisitPipe(ast, context)
}

func collectUsedPipes(nodes []render3.Node) map[string]bool {
	collector := newPipeUsageCollector()
	var walk func([]render3.Node)
	visitAttr := func(attr *render3.BoundAttribute) {
		if attr != nil {
			collector.Visit(attr.Value, nil)
		}
	}
	walk = func(nodes []render3.Node) {
		for _, node := range nodes {
			switch n := node.(type) {
			case *render3.BoundText:
				collector.Visit(n.Value, nil)
			case *render3.Element:
				for _, input := range n.Inputs {
					visitAttr(input)
				}
				walk(n.Children)
			case *render3.Template:
				for _, attr := range n.TemplateAttrs {
					if bound, ok := attr.(*render3.BoundAttribute); ok {
						visitAttr(bound)
					}
				}
				for _, input := range n.Inputs {
					visitAttr(input)
				}
				walk(n.Children)
			case *render3.IfBlock:
				for _, branch := range n.Branches {
					collector.Visit(branch.Expression, nil)
					walk(branch.Children)
				}
			case *render3.ForLoopBlock:
				collector.Visit(n.Expression.Ast, nil)
				if n.TrackBy != nil {
					collector.Visit(n.TrackBy.Ast, nil)
				}
				walk(n.Children)
				if n.Empty != nil {
					walk(n.Empty.Children)
				}
			case *render3.DeferredBlock:
				walk(n.Children)
				if n.Placeholder != nil {
					walk(n.Placeholder.Children)
				}
				if n.Loading != nil {
					walk(n.Loading.Children)
				}
				if n.Error != nil {
					walk(n.Error.Children)
				}
			}
		}
	}
	walk(nodes)
	return collector.used
}

func directImportMatchesDirective(imp ComponentImport, dir *metadata.DirectiveMeta) bool {
	if dir == nil {
		return false
	}
	if imp.Decl.ViaModule != "" && dir.Ref.OwningModule != "" && imp.Decl.ViaModule != dir.Ref.OwningModule {
		return false
	}
	return imp.Name == dir.Ref.Name || imp.Name+"Of" == dir.Ref.Name
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
												Descendants:             isView || id.Text == "ContentChildren", // ViewChild/Children and ContentChildren are descendants: true by default
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
		case "animations":
			analysis.Animations = output.NewWrappedNodeExpr(assign.Initializer, nil, nil, nil)
		case "imports":
			if assign.Initializer.Kind == ast.KindArrayLiteralExpression {
				arr := assign.Initializer.AsArrayLiteralExpression()
				if arr.Elements != nil {
					for _, elem := range arr.Elements.Nodes {
						if elem.Kind == ast.KindIdentifier {
							decl := h.host.GetDeclarationOfIdentifier(elem)
							if decl != nil && decl.Node != nil {
								importInfo := h.host.GetImportOfIdentifier(elem)
								importName := ""
								importPath := ""
								if importInfo != nil {
									importName = importInfo.Name
									importPath = importInfo.From
								}
								analysis.Imports = append(analysis.Imports, ComponentImport{
									Name:       elem.AsIdentifier().Text,
									ImportName: importName,
									ImportPath: importPath,
									Decl:       *decl,
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
			Name:        node.Name().AsIdentifier().Text,
			Selector:    analysis.Selector,
			Standalone:  analysis.IsStandalone,
			Imports:     metaImports,
			IsComponent: true,
			Ref: metadata.Reference{
				Name: node.Name().AsIdentifier().Text,
				Node: node.AsNode(),
			},
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
			parsedTemplate := render3.ParseTemplate(templateStr, "", nil)

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

			// Add imported NgModules verbatim. Standalone directives and pipes are added only
			// when the template binder proves that the template actually consumes them.
			for _, imp := range analysis.Imports {
				if h.metaRegistry != nil {
					if h.metaRegistry.GetDirectiveMetadata(imp.Decl.Node) != nil || h.metaRegistry.GetPipeMetadata(imp.Decl.Node) != nil {
						continue
					}
				}
				if !strings.HasSuffix(imp.Name, "Module") {
					continue
				}
				dep := render3.R3TemplateDependency{
					Kind: render3.R3TemplateDependencyKind_NgModule,
				}
				dep.Type = output.NewReadVarExpr(imp.Name, nil, nil, nil)
				addDep(dep)
			}

			matcher := render3.NewSelectorMatcher[[]directiveMetaAdapter]()
			for i := range scope.Directives {
				dir := &scope.Directives[i]
				if dir.Selector == "" {
					continue
				}
				matcher.AddSelectables(render3.CssSelectorParse(dir.Selector), []directiveMetaAdapter{{meta: dir}})
			}
			binder := render3.NewR3TargetBinder[directiveMetaAdapter](matcher, nil)
			bound := binder.Bind(render3.Target[directiveMetaAdapter]{Template: parsedTemplate.Nodes})
			usedPipes := collectUsedPipes(parsedTemplate.Nodes)

			importedDirectiveNodes := make(map[*ast.Node]bool)
			importedPipeNodes := make(map[*ast.Node]bool)
			matchedDirectives := bound.GetEagerlyUsedDirectives()

			// Sort matchedDirectives by their index in scope.Directives to match ngc order
			sort.SliceStable(matchedDirectives, func(i, j int) bool {
				idxI, idxJ := -1, -1
				for k := range scope.Directives {
					if scope.Directives[k].Ref.Node == matchedDirectives[i].meta.Ref.Node {
						idxI = k
					}
					if scope.Directives[k].Ref.Node == matchedDirectives[j].meta.Ref.Node {
						idxJ = k
					}
				}
				return idxI < idxJ
			})

			for _, imp := range analysis.Imports {
				if h.metaRegistry == nil || imp.Decl.Node == nil {
					for _, matchedDir := range matchedDirectives {
						if directImportMatchesDirective(imp, matchedDir.meta) {
							importedDirectiveNodes[matchedDir.meta.Ref.Node] = true
							addDep(render3.R3TemplateDependency{
								Kind: render3.R3TemplateDependencyKind_Directive,
								Type: output.NewReadVarExpr(imp.Name, nil, nil, nil),
							})
							break
						}
					}
					continue
				}
				if dirMeta := h.metaRegistry.GetDirectiveMetadata(imp.Decl.Node); dirMeta != nil {
					for _, matchedDir := range matchedDirectives {
						if matchedDir.meta == dirMeta || (matchedDir.meta.Selector != "" && matchedDir.meta.Selector == dirMeta.Selector) || directImportMatchesDirective(imp, matchedDir.meta) {
							importedDirectiveNodes[imp.Decl.Node] = true
							importedDirectiveNodes[matchedDir.meta.Ref.Node] = true
							addDep(render3.R3TemplateDependency{
								Kind: render3.R3TemplateDependencyKind_Directive,
								Type: output.NewReadVarExpr(imp.Name, nil, nil, nil),
							})
							break
						}
					}
				} else {
					for _, matchedDir := range matchedDirectives {
						if directImportMatchesDirective(imp, matchedDir.meta) {
							importedDirectiveNodes[matchedDir.meta.Ref.Node] = true
							addDep(render3.R3TemplateDependency{
								Kind: render3.R3TemplateDependencyKind_Directive,
								Type: output.NewReadVarExpr(imp.Name, nil, nil, nil),
							})
							break
						}
					}
				}
				if pipeMeta := h.metaRegistry.GetPipeMetadata(imp.Decl.Node); pipeMeta != nil && usedPipes[pipeMeta.Name] {
					importedPipeNodes[imp.Decl.Node] = true
					addDep(render3.R3TemplateDependency{
						Kind: render3.R3TemplateDependencyKind_Pipe,
						Type: output.NewReadVarExpr(imp.Name, nil, nil, nil),
					})
				}
			}

			// Add directives matched by the template and not already represented by a
			// directly imported standalone symbol.
			for _, matchedDir := range matchedDirectives {
				dir := matchedDir.meta
				if importedDirectiveNodes[dir.Ref.Node] {
					continue
				}
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

			// Add pipes used by the template.
			for _, pipe := range scope.Pipes {
				if !usedPipes[pipe.Name] {
					continue
				}
				if importedPipeNodes[pipe.Ref.Node] {
					continue
				}
				if pipe.Ref.Name == "" {
					continue
				}
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
					dep.Type = output.NewReadVarExpr(pipe.Ref.Name, nil, nil, nil)
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

	compScope := h.scopeRegistry.GetCompilationScope(node.AsNode())
	deferBlocks, deferComponentDeps := extractDeferBlocks(parsedTemplate.Nodes, compScope, node.AsNode(), analysis.Imports)
	if importMgr != nil {
		sourceFile := ast.GetSourceFileOfNode(node.AsNode())
		if sourceFile != nil {
			for _, deferDep := range deferComponentDeps {
				removeName := deferDep.LocalName
				if removeName == "" {
					removeName = deferDep.SymbolName
				}
				if removeName != "" && deferDep.ImportPath != "" {
					importMgr.RemoveImport(sourceFile.AsNode(), removeName, deferDep.ImportPath)
				}
			}
		}
	}

	var eagerDeps []render3.R3TemplateDependency
	for _, dep := range resolution.Dependencies {
		isDeferred := false
		var depName string
		if ext, ok := dep.Type.(*output.ExternalExpr); ok {
			depName = *ext.Value.Name
		} else if read, ok := dep.Type.(*output.ReadVarExpr); ok {
			depName = read.Name
		}

		for _, deferDep := range deferComponentDeps {
			localName := deferDep.LocalName
			if localName == "" {
				localName = deferDep.SymbolName
			}
			if deferDep.SymbolName == depName || localName == depName {
				isDeferred = true
				break
			}
		}

		if !isDeferred {
			eagerDeps = append(eagerDeps, dep)
		} else {
			if depName != "" {
				// We need to find the module specifier for this dependency
				// In this compiler, we just assume the dependency is in the deferComponentDeps
				// and we use its ImportPath.
				var importPath string
				for _, deferDep := range deferComponentDeps {
					localName := deferDep.LocalName
					if localName == "" {
						localName = deferDep.SymbolName
					}
					if (deferDep.SymbolName == depName || localName == depName) && deferDep.ImportPath != "" {
						importPath = deferDep.ImportPath
						break
					}
				}
				if importPath != "" && importMgr != nil {
					sourceFile := ast.GetSourceFileOfNode(node.AsNode())
					if sourceFile != nil {
						importMgr.RemoveImport(sourceFile.AsNode(), depName, importPath)
					}
				}
			}
		}
	}

	hasDirectiveDeps := len(deferComponentDeps) > 0
	if !hasDirectiveDeps {
		for _, dep := range resolution.Dependencies {
			if dep.Kind == render3.R3TemplateDependencyKind_Directive {
				hasDirectiveDeps = true
				break
			}
		}
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
		Declarations:             eagerDeps,
		Styles:                   analysis.Styles,
		Animations:               analysis.Animations,
		HasDirectiveDependencies: hasDirectiveDeps,
		Defer: render3.R3ComponentDeferMetadata{
			Mode:   render3.DeferBlockDepsEmitMode_PerBlock,
			Blocks: deferBlocks,
		},
	}
	var compiled render3.R3CompiledExpression
	if h.isPartial {
		// Convert Dependencies to R3TemplateDependencyMetadata
		var metaDeps []render3.R3TemplateDependencyMetadata
		for _, dep := range resolution.Dependencies {
			metaDeps = append(metaDeps, render3.R3DirectiveDependencyMetadata{
				Selector:             "unknown",
				R3TemplateDependency: dep,
				IsComponent:          false,
			})
		}
		metaPartial := render3.R3ComponentMetadata[render3.R3TemplateDependencyMetadata]{
			R3DirectiveMetadata: meta.R3DirectiveMetadata,
			Template:            meta.Template,
			Declarations:        metaDeps,
			Styles:              meta.Styles,
		}
		compiled = partial.CompileDeclareComponentFromMetadata(metaPartial, parsedTemplate, partial.DeclareComponentTemplateInfo{
			Content:  analysis.Template,
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

	classMetaExpr := render3.CompileComponentClassMetadata(render3.R3ClassMetadata{
		Type: output.NewReadVarExpr(className, nil, nil, nil),
		Decorators: output.NewLiteralArrayExpr([]output.Expression{
			output.NewLiteralMapExpr(decMapEntries, nil, nil, nil),
		}, nil, nil, nil),
		PropDecorators: propDecExpr,
	}, deferComponentDeps)

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
				lineNum = strings.Count(text[:targetPos], "\n") + 1
			}
		}
	}

	classDebugExpr := render3.CompileClassDebugInfo(render3.R3ClassDebugInfo{
		Type:       output.NewReadVarExpr(className, nil, nil, nil),
		ClassName:  output.NewLiteralExpr(className, nil, nil, nil),
		FilePath:   output.NewLiteralExpr(filePath, nil, nil, nil),
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
		if refHost.HasBaseClass(classDecl) {
			return nil
		}
		return []render3.R3DependencyMetadata{}
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
