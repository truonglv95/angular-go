package annotations

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3/partial"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	ngdiagnostics "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	ngscope "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
	core "github.com/microsoft/typescript-go/internal/core"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
)

type ComponentImport struct {
	Name       string
	ImportName string
	ImportPath string
	Decl       reflection.Declaration
}

type ComponentAnalysis struct {
	Selector            string
	Template            string
	TemplateUrl         string
	Styles              []string
	StyleUrls           []string
	IsStandalone        bool
	Imports             []ComponentImport
	Inputs              map[string]render3.R3InputMetadata
	Outputs             map[string]string
	ExportAs            []string
	Queries             []render3.R3QueryMetadata
	ViewQueries         []render3.R3QueryMetadata
	Host                render3.R3HostMetadata
	ParsedTemplate      *render3.ParsedTemplate
	Animations          output.Expression
	DecoratorNode       *ast.Node
	PropDecorators      map[string][]*ast.Node
	CompilationMode     string
	Encapsulation       int
	HostDirectives      []metadata.HostDirectiveMeta
	ChangeDetection     output.Expression
	PreserveWhitespaces *bool
	RawImports          output.Expression
	Providers           output.Expression
	ViewProviders       output.Expression
}

type ComponentResolution struct {
	Dependencies             []render3.R3TemplateDependency
	SemanticReferenceKeys    []string
	SemanticReferenceSymbols []semantic_graph.SemanticSymbol
}

func semanticKeyForRef(ref metadata.Reference) string {
	if ref.Node == nil || ref.Name == "" {
		return ""
	}
	sf := ast.GetSourceFileOfNode(ref.Node)
	if sf == nil {
		return ""
	}
	return semantic_graph.SymbolKey(sf.FileName(), ref.Name)
}

func semanticKeyForSymbol(symbol *semantic_graph.SemanticSymbol) string {
	if symbol == nil {
		return ""
	}
	return semantic_graph.SymbolKey(symbol.Path, symbol.Identifier)
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
	host              reflection.ReflectionHost
	isPartial         bool
	metaRegistry      *metadata.LocalMetadataRegistry
	scopeRegistry     *ngscope.LocalModuleScopeRegistry
	resourceRegistry  *metadata.ResourceRegistry
	enableHmr         bool
	styleIncludePaths []string
}

func NewComponentDecoratorHandler(host reflection.ReflectionHost, isPartial bool, metaRegistry *metadata.LocalMetadataRegistry, scopeRegistry *ngscope.LocalModuleScopeRegistry, resourceRegistry *metadata.ResourceRegistry, enableHmr bool, styleIncludePaths []string) *ComponentDecoratorHandler {
	return &ComponentDecoratorHandler{
		host:              host,
		isPartial:         isPartial,
		metaRegistry:      metaRegistry,
		scopeRegistry:     scopeRegistry,
		resourceRegistry:  resourceRegistry,
		enableHmr:         enableHmr,
		styleIncludePaths: styleIncludePaths,
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
		IsStandalone:    true,
		Inputs:          make(map[string]render3.R3InputMetadata),
		Outputs:         make(map[string]string),
		Host:            render3.R3HostMetadata{},
		DecoratorNode:   decorator.Node,
		PropDecorators:  make(map[string][]*ast.Node),
		CompilationMode: "global",
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
				propName := ""
				if nameNode != nil && nameNode.Kind == ast.KindIdentifier {
					propName = nameNode.AsIdentifier().Text
				}
				if member.Kind == ast.KindPropertyDeclaration && propName != "" {
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
									if propName != "" {
										if id.Text == "Input" {
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
												ClassPropertyName:   propName,
												BindingPropertyName: alias,
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
			fmt.Fprintf(os.Stderr, ">>> [DEBUG] Found styleUrl: kind %v\n", assign.Initializer.Kind)
			if assign.Initializer.Kind == ast.KindStringLiteral {
				analysis.StyleUrls = append(analysis.StyleUrls, assign.Initializer.AsStringLiteral().Text)
			} else if assign.Initializer.Kind == ast.KindNoSubstitutionTemplateLiteral {
				analysis.StyleUrls = append(analysis.StyleUrls, assign.Initializer.AsNoSubstitutionTemplateLiteral().Text)
			}
		case "animations":
			analysis.Animations = output.NewWrappedNodeExpr(assign.Initializer, nil, nil, nil)
		case "encapsulation":
			analysis.Encapsulation = parseEncapsulation(assign.Initializer)
		case "changeDetection":
			analysis.ChangeDetection = parseChangeDetection(assign.Initializer)
		case "providers":
			analysis.Providers = output.NewWrappedNodeExpr(assign.Initializer, nil, nil, nil)
		case "viewProviders":
			analysis.ViewProviders = output.NewWrappedNodeExpr(assign.Initializer, nil, nil, nil)
		case "preserveWhitespaces":
			var val bool
			if assign.Initializer.Kind == ast.KindTrueKeyword {
				val = true
				analysis.PreserveWhitespaces = &val
			} else if assign.Initializer.Kind == ast.KindFalseKeyword {
				val = false
				analysis.PreserveWhitespaces = &val
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

func (h *ComponentDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
	analysis := analysisData.(*ComponentAnalysis)
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
			Name:           node.Name().AsIdentifier().Text,
			Selector:       analysis.Selector,
			Standalone:     analysis.IsStandalone,
			Imports:        metaImports,
			IsComponent:    true,
			Inputs:         metaInputs,
			Outputs:        analysis.Outputs,
			ExportAs:       analysis.ExportAs,
			RequiredInputs: reqInputs,
			HostDirectives: analysis.HostDirectives,
			Ref: metadata.Reference{
				Name: node.Name().AsIdentifier().Text,
				Node: node.AsNode(),
			},
		})
	}

	if h.resourceRegistry != nil {
		sourceFile := ast.GetSourceFileOfNode(node.AsNode())
		baseDir := filepath.Dir(sourceFile.FileName())

		var templateResource *metadata.Resource
		if analysis.TemplateUrl != "" {
			templateResource = &metadata.Resource{
				Path: filepath.Clean(filepath.Join(baseDir, analysis.TemplateUrl)),
				Node: node.AsNode(),
			}
		} else {
			templateResource = &metadata.Resource{
				Path: "",
				Node: node.AsNode(),
			}
		}

		var styleResources []*metadata.Resource
		for _, styleUrl := range analysis.StyleUrls {
			if styleUrl != "" {
				styleResources = append(styleResources, &metadata.Resource{
					Path: filepath.Clean(filepath.Join(baseDir, styleUrl)),
					Node: node.AsNode(),
				})
			}
		}
		for _, style := range analysis.Styles {
			if style != "" {
				styleResources = append(styleResources, &metadata.Resource{
					Path: "",
					Node: node.AsNode(),
				})
			}
		}

		h.resourceRegistry.RegisterResources(metadata.DirectiveResources{
			Template: templateResource,
			Styles:   styleResources,
		}, node.AsNode())
	}
}

func (h *ComponentDecoratorHandler) getMetadataReader() metadata.MetadataReader {
	if h.scopeRegistry != nil {
		return h.scopeRegistry.MetadataReader()
	}
	return h.metaRegistry
}

func (h *ComponentDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	reader := h.getMetadataReader()
	analysis := analysisData.(*ComponentAnalysis)
	resolution := &ComponentResolution{}
	var scope *ngscope.CompilationScope
	var bound render3.BoundTarget[directiveMetaAdapter]

	if h.scopeRegistry != nil {
		scope = h.scopeRegistry.GetCompilationScope(node.AsNode())
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
				if !strings.HasPrefix(templatePath, "http") {
					if content, err := ioutil.ReadFile(templatePath); err == nil {
						templateStr = string(content)
					}
				}
			}
			var parseOpts *render3.ParseTemplateOptions
			if analysis.PreserveWhitespaces != nil {
				parseOpts = &render3.ParseTemplateOptions{
					PreserveWhitespaces: analysis.PreserveWhitespaces,
				}
			}
			templateUrlStr := ""
			if analysis.TemplateUrl != "" {
				templateUrlStr = filepath.Join(baseDir, analysis.TemplateUrl)
			}
			parsedTemplate := render3.ParseTemplate(templateStr, templateUrlStr, parseOpts)
			analysis.ParsedTemplate = &parsedTemplate

			// Deduplicate dependencies
			seenDeps := make(map[string]bool)
			seenSemanticRefs := make(map[string]bool)
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
			addSemanticRef := func(key string) {
				if key == "" || seenSemanticRefs[key] {
					return
				}
				seenSemanticRefs[key] = true
				resolution.SemanticReferenceKeys = append(resolution.SemanticReferenceKeys, key)
			}
			addSemanticRefForRef := func(ref metadata.Reference) {
				if h.scopeRegistry != nil {
					if semanticReader, ok := any(h.scopeRegistry).(ngscope.SemanticScopeReader); ok {
						if sym := semanticReader.GetSemanticSymbol(ref.Node); sym != nil {
							key := semanticKeyForSymbol(sym)
							if key != "" {
								addSemanticRef(key)
								resolution.SemanticReferenceSymbols = append(resolution.SemanticReferenceSymbols, *sym)
								return
							}
						}
					}
				}
				addSemanticRef(semanticKeyForRef(ref))
			}

			// Add imported NgModules verbatim. Standalone directives and pipes are added only
			// when the template binder proves that the template actually consumes them.
			for _, imp := range analysis.Imports {
				if reader != nil {
					if reader.GetDirectiveMetadata(imp.Decl.Node) != nil || reader.GetPipeMetadata(imp.Decl.Node) != nil {
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
				if h.metaRegistry != nil && imp.Decl.Node != nil {
					addSemanticRefForRef(metadata.Reference{Name: imp.Name, Node: imp.Decl.Node, OwningModule: imp.Decl.ViaModule})
				}
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
			bound = binder.Bind(render3.Target[directiveMetaAdapter]{Template: parsedTemplate.Nodes})
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
							addSemanticRefForRef(matchedDir.meta.Ref)
							addDep(render3.R3TemplateDependency{
								Kind: render3.R3TemplateDependencyKind_Directive,
								Type: output.NewReadVarExpr(imp.Name, nil, nil, nil),
							})
							break
						}
					}
					continue
				}
				if dirMeta := reader.GetDirectiveMetadata(imp.Decl.Node); dirMeta != nil {
					for _, matchedDir := range matchedDirectives {
						if matchedDir.meta == dirMeta || (matchedDir.meta.Selector != "" && matchedDir.meta.Selector == dirMeta.Selector) || directImportMatchesDirective(imp, matchedDir.meta) {
							importedDirectiveNodes[imp.Decl.Node] = true
							importedDirectiveNodes[matchedDir.meta.Ref.Node] = true
							addSemanticRefForRef(matchedDir.meta.Ref)
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
							addSemanticRefForRef(matchedDir.meta.Ref)
							addDep(render3.R3TemplateDependency{
								Kind: render3.R3TemplateDependencyKind_Directive,
								Type: output.NewReadVarExpr(imp.Name, nil, nil, nil),
							})
							break
						}
					}
				}
				if pipeMeta := reader.GetPipeMetadata(imp.Decl.Node); pipeMeta != nil && usedPipes[pipeMeta.Name] {
					importedPipeNodes[imp.Decl.Node] = true
					addSemanticRefForRef(pipeMeta.Ref)
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
				addSemanticRefForRef(dir.Ref)
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
				addSemanticRefForRef(pipe.Ref)
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

	var diagnostics []ast.Diagnostic

	var decorator *reflection.Decorator
	for _, dec := range h.host.GetDecoratorsOfDeclaration(node.AsNode()) {
		if dec.Name == "Component" {
			d := dec
			decorator = &d
			break
		}
	}

	// 1. Standalone / Non-standalone import checks
	if analysis.IsStandalone {
		for _, imp := range analysis.Imports {
			isNgModule := reader.GetNgModuleMetadata(imp.Decl.Node) != nil
			dirMeta := reader.GetDirectiveMetadata(imp.Decl.Node)
			pipeMeta := reader.GetPipeMetadata(imp.Decl.Node)

			isStandaloneDirectiveOrPipe := false
			if dirMeta != nil && dirMeta.Standalone {
				isStandaloneDirectiveOrPipe = true
			} else if pipeMeta != nil && pipeMeta.Standalone {
				isStandaloneDirectiveOrPipe = true
			}

			if !isNgModule && !isStandaloneDirectiveOrPipe {
				var importNode *ast.Node = analysis.DecoratorNode
				if decorator != nil {
					importsArrExpr := findComponentPropertyNode(decorator, "imports")
					if importsArrExpr != nil && importsArrExpr.Kind == ast.KindArrayLiteralExpression {
						for _, elem := range importsArrExpr.AsArrayLiteralExpression().Elements.Nodes {
							if elem.Kind == ast.KindIdentifier && elem.AsIdentifier().Text == imp.Name {
								importNode = elem
								break
							}
						}
					}
				}

				if dirMeta == nil && pipeMeta == nil && !isNgModule {
					diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
						ngdiagnostics.ErrorCode_COMPONENT_UNKNOWN_IMPORT,
						importNode,
						fmt.Sprintf("The class '%s' is not an NgModule, Component, Directive, or Pipe, and cannot be imported.", imp.Name),
						nil,
						tsdiagnostics.CategoryError,
					))
				} else {
					diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
						ngdiagnostics.ErrorCode_COMPONENT_IMPORT_NOT_STANDALONE,
						importNode,
						fmt.Sprintf("The imported class '%s' is not standalone and cannot be imported directly. It must be declared in an NgModule, and that NgModule must be imported instead.", imp.Name),
						nil,
						tsdiagnostics.CategoryError,
					))
				}
			}
		}
	} else {
		if decorator != nil {
			importsExpr := findComponentPropertyNode(decorator, "imports")
			if importsExpr != nil {
				diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
					ngdiagnostics.ErrorCode_COMPONENT_NOT_STANDALONE,
					importsExpr,
					fmt.Sprintf("The component %s is not standalone, but has an imports list. Only standalone components can specify imports.", node.Name().AsIdentifier().Text),
					nil,
					tsdiagnostics.CategoryError,
				))
			}
		}
	}

	// 2. Duplicate inputs/outputs checks
	seenInputs := make(map[string]string)
	for propName, input := range analysis.Inputs {
		if prevProp, exists := seenInputs[input.BindingPropertyName]; exists {
			var propNode *ast.Node = node.AsNode()
			if node.Members != nil {
				for _, member := range node.Members.Nodes {
					if member.Kind == ast.KindPropertyDeclaration {
						pd := member.AsPropertyDeclaration()
						if pd.Name() != nil && pd.Name().Kind == ast.KindIdentifier && pd.Name().AsIdentifier().Text == propName {
							propNode = member
							break
						}
					}
				}
			}
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_DUPLICATE_BINDING_NAME,
				propNode,
				fmt.Sprintf("Duplicate input binding name '%s' (bound to '%s' and '%s').", input.BindingPropertyName, prevProp, propName),
				nil,
				tsdiagnostics.CategoryError,
			))
		}
		seenInputs[input.BindingPropertyName] = propName
	}

	seenOutputs := make(map[string]string)
	for propName, bindingName := range analysis.Outputs {
		if prevProp, exists := seenOutputs[bindingName]; exists {
			var propNode *ast.Node = node.AsNode()
			if node.Members != nil {
				for _, member := range node.Members.Nodes {
					if member.Kind == ast.KindPropertyDeclaration {
						pd := member.AsPropertyDeclaration()
						if pd.Name() != nil && pd.Name().Kind == ast.KindIdentifier && pd.Name().AsIdentifier().Text == propName {
							propNode = member
							break
						}
					}
				}
			}
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_DUPLICATE_BINDING_NAME,
				propNode,
				fmt.Sprintf("Duplicate output binding name '%s' (bound to '%s' and '%s').", bindingName, prevProp, propName),
				nil,
				tsdiagnostics.CategoryError,
			))
		}
		seenOutputs[bindingName] = propName
	}

	// 3. Template validations (Syntax & Semantic Checks)
	parsedTemplate := analysis.ParsedTemplate
	if parsedTemplate == nil {
		var parseOpts *render3.ParseTemplateOptions
		if analysis.PreserveWhitespaces != nil {
			parseOpts = &render3.ParseTemplateOptions{
				PreserveWhitespaces: analysis.PreserveWhitespaces,
			}
		}
		pt := render3.ParseTemplate(analysis.Template, "", parseOpts)
		parsedTemplate = &pt
	}

	if parsedTemplate != nil {
		// Syntax parser errors
		for _, parseErr := range parsedTemplate.Errors {
			templateNode := findComponentPropertyNode(decorator, "template")
			if templateNode == nil {
				templateNode = findComponentPropertyNode(decorator, "templateUrl")
			}
			var diagNode *ast.Node
			if templateNode != nil {
				cloned := *templateNode
				if findComponentPropertyNode(decorator, "template") != nil && parseErr.Span != nil {
					pos := templateNode.Pos() + 1 + parseErr.Span.Start.Offset
					end := templateNode.Pos() + 1 + parseErr.Span.End.Offset
					cloned.Loc = core.NewTextRange(pos, end)
				}
				diagNode = &cloned
			} else {
				diagNode = node.AsNode()
			}
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_TEMPLATE_PARSE_ERROR,
				diagNode,
				parseErr.Msg,
				nil,
				tsdiagnostics.CategoryError,
			))
		}

		// Out-of-band checks
		if scope != nil {
			walker := &templateDiagnosticWalker{
				diagnostics: diagnostics,
				h:           h,
				node:        node,
				decorator:   decorator,
				analysis:    analysis,
				scope:       scope,
				bound:       bound,
			}

			// Missing Pipe Check
			usedPipes := collectUsedPipes(parsedTemplate.Nodes)
			for pipeName := range usedPipes {
				found := false
				for _, pipe := range scope.Pipes {
					if pipe.Name == pipeName {
						found = true
						break
					}
				}
				if !found {
					var commonPipes = map[string][2]string{
						"async":      {"AsyncPipe", "@angular/common"},
						"uppercase":  {"UpperCasePipe", "@angular/common"},
						"lowercase":  {"LowerCasePipe", "@angular/common"},
						"json":       {"JsonPipe", "@angular/common"},
						"slice":      {"SlicePipe", "@angular/common"},
						"number":     {"DecimalPipe", "@angular/common"},
						"percent":    {"PercentPipe", "@angular/common"},
						"titlecase":  {"TitleCasePipe", "@angular/common"},
						"currency":   {"CurrencyPipe", "@angular/common"},
						"date":       {"DatePipe", "@angular/common"},
						"i18nPlural": {"I18nPluralPipe", "@angular/common"},
						"i18nSelect": {"I18nSelectPipe", "@angular/common"},
						"keyvalue":   {"KeyValuePipe", "@angular/common"},
					}
					msg := fmt.Sprintf("No pipe found with name '%s'.", pipeName)
					if sugg, exists := commonPipes[pipeName]; exists {
						className := sugg[0]
						importPath := sugg[1]
						if analysis.IsStandalone {
							msg += fmt.Sprintf("\nTo fix this, import the \"%s\" class from \"%s\" and add it to the \"imports\" array of the component.", className, importPath)
						} else {
							msg += fmt.Sprintf("\nTo fix this, import the \"%s\" class from \"%s\" and add it to the \"imports\" array of the module declaring the component.", className, importPath)
						}
					}
					var pipeSpan parse_util.ParseSourceSpan
					for _, nodeVal := range parsedTemplate.Nodes {
						if span, ok := findPipeSpan(nodeVal, pipeName); ok {
							pipeSpan = span
							break
						}
					}
					if (pipeSpan.Start == nil || pipeSpan.Start.File == nil) && len(parsedTemplate.Nodes) > 0 {
						pipeSpan = parsedTemplate.Nodes[0].GetSourceSpan()
					}
					walker.makeTemplateDiagnostic(pipeSpan, ngdiagnostics.ErrorCode_MISSING_PIPE, msg)
				}
			}

			walker.walkNodes(parsedTemplate.Nodes)
			diagnostics = walker.diagnostics
		}
	}

	return resolution, diagnostics
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
		if content, err := readAndCompileStyle(stylePath, h.styleIncludePaths); err == nil {
			analysis.Styles = append(analysis.Styles, content)
		} else {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_COMPONENT_RESOURCE_NOT_FOUND,
				node.AsNode(),
				fmt.Sprintf("Failed to load styleUrl %q: %s.", stylePath, err.Error()),
				nil,
				tsdiagnostics.CategoryError,
			))
		}
	}

	var parsedTemplate render3.ParsedTemplate
	if analysis.ParsedTemplate != nil {
		parsedTemplate = *analysis.ParsedTemplate
	} else {
		var parseOpts *render3.ParseTemplateOptions
		if analysis.PreserveWhitespaces != nil {
			parseOpts = &render3.ParseTemplateOptions{
				PreserveWhitespaces: analysis.PreserveWhitespaces,
			}
		}
		templateUrlStr := ""
		if analysis.TemplateUrl != "" {
			templateUrlStr = filepath.Join(baseDir, analysis.TemplateUrl)
		}
		parsedTemplate = render3.ParseTemplate(analysis.Template, templateUrlStr, parseOpts)
	}

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

	inputsOrder, outputsOrder := getPropDeclarationOrder(node)

	meta := render3.R3ComponentMetadata[render3.R3TemplateDependency]{
		RawImports: analysis.RawImports,
		R3DirectiveMetadata: render3.R3DirectiveMetadata{
			Name: className,
			Type: render3.R3Reference{
				Value: output.NewReadVarExpr(className, nil, nil, nil),
			},
			Selector:         &analysis.Selector,
			IsStandalone:     analysis.IsStandalone,
			Inputs:           analysis.Inputs,
			InputProperties:  inputsOrder,
			Outputs:          analysis.Outputs,
			OutputProperties: outputsOrder,
			Host:             analysis.Host,
			Queries:          analysis.Queries,
			ViewQueries:      analysis.ViewQueries,
			Providers:        analysis.Providers,
			HostDirectives:   render3HostDirectives(analysis.HostDirectives),
		},
		Template: render3.Template{
			Children: parsedTemplate.Nodes,
		},
		Declarations:             eagerDeps,
		Styles:                   analysis.Styles,
		Animations:               analysis.Animations,
		Encapsulation:            analysis.Encapsulation,
		ChangeDetection:          analysis.ChangeDetection,
		ViewProviders:            analysis.ViewProviders,
		HasDirectiveDependencies: hasDirectiveDeps,
		Defer: render3.R3ComponentDeferMetadata{
			Mode:   render3.DeferBlockDepsEmitMode_PerBlock,
			Blocks: deferBlocks,
		},
	}
	if analysis.CompilationMode == "local" {
		meta.DeclarationListEmitMode = render3.DeclarationListEmitMode_RuntimeResolved
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
			Encapsulation:       meta.Encapsulation,
			ChangeDetection:     meta.ChangeDetection,
			ViewProviders:       meta.ViewProviders,
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
			if analysis.TemplateUrl != "" || len(analysis.StyleUrls) > 0 {
				decArgs = append(decArgs, componentDecoratorMetadataExpr(analysis))
			} else {
				decArgs = append(decArgs, output.NewWrappedNodeExpr(callExpr.Arguments.Nodes[0], nil, nil, nil))
			}
		}
	}

	coreModule := "@angular/core"
	componentName := "Component"
	decMapEntries := []output.LiteralMapEntry{
		output.NewLiteralMapPropertyAssignment("type", output.NewExternalExpr(output.ExternalReference{
			ModuleName: &coreModule,
			Name:       &componentName,
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
		if rel, err := func() (string, error) { cwd, _ := os.Getwd(); return filepath.Rel(cwd, rawPath) }(); err == nil && !strings.HasPrefix(rel, "..") {
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

	var hmrUpdateDecl *HmrUpdateDeclaration
	if !h.isPartial && h.enableHmr {
		var classDebugExprArg output.Expression
		if classDebugExpr != nil {
			classDebugExprArg = classDebugExpr
		}
		hmrDeps := extractHmrDependencies(node, compiled, facExpr, classDebugExprArg)
		if hmrDeps != nil {
			hmrMeta := render3.R3HmrMetadata{
				Type:                  output.NewWrappedNodeExpr(node.Name(), nil, nil, nil),
				ClassName:             className,
				FilePath:              filePath,
				LocalDependencies:     hmrDeps.local,
				NamespaceDependencies: hmrDeps.external,
			}

			hmrInitExpr := render3.CompileHmrInitializer(hmrMeta)
			if hmrInitExpr != nil {
				hmrInitStmt := hmrInitExpr.ToStmt(nil)
				astStmt := hmrInitStmt.VisitStatement(visitor, translator.Context{IsStatementMode: true})
				if astStmt != nil {
					if n, ok := astStmt.(*ast.Node); ok {
						extraStatements = append(extraStatements, n)
					} else if arr, ok := astStmt.([]*ast.Node); ok {
						extraStatements = append(extraStatements, arr...)
					}
					// Unknown AST stmt types are silently skipped —
					// stdout is the JSON-RPC stream in daemon mode, so no debug prints here.
				}
			}

			constantStatements := pool.Statements

			hmrUpdateDecl = getHmrUpdateDeclaration(
				facExpr,
				compiled,
				constantStatements,
				hmrMeta,
				node,
				importMgr,
				factory,
			)
		}
	}

	var hmrUpdateNodes []*ast.Node
	var hmrImports []*ast.Node
	if hmrUpdateDecl != nil {
		hmrUpdateNodes = hmrUpdateDecl.Nodes
		hmrImports = hmrUpdateDecl.Imports
	}

	return []transform.CompileResult{
		{
			PropertyName: "ɵfac",
			Initializer:  facInitializerNode,
		},
		{
			PropertyName:   "ɵcmp",
			Initializer:    initializerNode,
			Statements:     extraStatements,
			HmrUpdateNodes: hmrUpdateNodes,
			HmrImports:     hmrImports,
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

type styleCacheEntry struct {
	css   string
	mtime time.Time
	size  int64
}

var (
	styleCache   = make(map[string]styleCacheEntry)
	styleCacheMu sync.RWMutex
)

func readAndCompileStyle(stylePath string, includePaths []string) (string, error) {
	fmt.Fprintf(os.Stderr, ">>> [DEBUG] readAndCompileStyle called for: %s\n", stylePath)
	ext := filepath.Ext(stylePath)
	isSass := ext == ".scss" || ext == ".sass"
	isLess := ext == ".less"

	stat, statErr := os.Stat(stylePath)
	if statErr != nil {
		fmt.Fprintf(os.Stderr, ">>> [DEBUG] stat error: %v\n", statErr)
		return "", statErr
	}
	mtime := stat.ModTime()
	size := stat.Size()

	if isSass || isLess {
		styleCacheMu.RLock()
		entry, exists := styleCache[stylePath]
		styleCacheMu.RUnlock()
		if exists && entry.mtime.Equal(mtime) && entry.size == size {
			return entry.css, nil
		}
	}

	var content []byte
	var err error

	if isSass {
		args := []string{}
		for _, p := range includePaths {
			args = append(args, "--load-path=" + p)
		}
		args = append(args, stylePath)
		cmd := exec.Command("sass", args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err = cmd.Run(); err == nil {
			content = stdout.Bytes()
			fmt.Fprintf(os.Stderr, ">>> [DEBUG] sass success. Output len: %d\n", len(content))
		} else {
			fmt.Fprintf(os.Stderr, ">>> [DEBUG] sass error: %v, stderr: %s\n", err, stderr.String())
			if strings.Contains(err.Error(), "executable file not found") || strings.Contains(err.Error(), "command not found") {
				npxArgs := append([]string{"sass"}, args...)
				cmdNpx := exec.Command("npx", npxArgs...)
				stdout.Reset()
				stderr.Reset()
				cmdNpx.Stdout = &stdout
				cmdNpx.Stderr = &stderr
				if err = cmdNpx.Run(); err == nil {
					content = stdout.Bytes()
					fmt.Fprintf(os.Stderr, ">>> [DEBUG] npx sass success. Output len: %d\n", len(content))
				} else {
					fmt.Fprintf(os.Stderr, ">>> [DEBUG] npx sass error: %v, stderr: %s\n", err, stderr.String())
					return "", fmt.Errorf("sass compilation failed:\n%s", stderr.String())
				}
			} else {
				return "", fmt.Errorf("sass compilation failed:\n%s", stderr.String())
			}
		}
	} else if isLess {
		cmd := exec.Command("lessc", stylePath)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err = cmd.Run(); err == nil {
			content = stdout.Bytes()
		} else {
			content, err = ioutil.ReadFile(stylePath)
		}
	} else {
		content, err = ioutil.ReadFile(stylePath)
	}

	if err != nil {
		return "", err
	}

	css := string(content)

	if isSass || isLess {
		styleCacheMu.Lock()
		styleCache[stylePath] = styleCacheEntry{
			css:   css,
			mtime: mtime,
			size:  size,
		}
		styleCacheMu.Unlock()
	}

	return css, nil
}

func (h *ComponentDecoratorHandler) GetSemanticSymbol(node *ast.ClassDeclaration, analysis any) *semantic_graph.SemanticSymbol {
	a, ok := analysis.(*ComponentAnalysis)
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
		Kind:           "component",
		Selector:       a.Selector,
		Inputs:         inputs,
		Outputs:        a.Outputs,
		ExportAs:       a.ExportAs,
		Standalone:     a.IsStandalone,
		Imports:        imps,
		TypeParameters: tps,
	}
}

func (h *ComponentDecoratorHandler) GetSemanticReferenceKeys(node *ast.ClassDeclaration, analysis any, resolution any) []string {
	r, ok := resolution.(*ComponentResolution)
	if !ok || r == nil {
		return nil
	}
	return r.SemanticReferenceKeys
}

func (h *ComponentDecoratorHandler) GetSemanticReferenceSymbols(node *ast.ClassDeclaration, analysis any, resolution any) []semantic_graph.SemanticSymbol {
	r, ok := resolution.(*ComponentResolution)
	if !ok || r == nil {
		return nil
	}
	return r.SemanticReferenceSymbols
}

func parseEncapsulation(node *ast.Node) int {
	if node == nil {
		return 0
	}
	if node.Kind == ast.KindNumericLiteral {
		t := node.AsNumericLiteral().Text
		if t == "2" {
			return 2
		} else if t == "3" {
			return 3
		} else if t == "4" {
			return 4
		} else if t == "0" {
			return 0
		}
	}
	if node.Kind == ast.KindPropertyAccessExpression {
		pa := node.AsPropertyAccessExpression()
		nameNode := pa.Name()
		if nameNode != nil && nameNode.Kind == ast.KindIdentifier {
			name := nameNode.AsIdentifier().Text
			switch name {
			case "Emulated":
				return 0
			case "None":
				return 2
			case "ShadowDom":
				return 3
			case "ExperimentalIsolatedShadowDom":
				return 4
			}
		}
	}
	return 0 // default
}

func parseChangeDetection(node *ast.Node) output.Expression {
	if node == nil {
		return nil
	}
	if node.Kind == ast.KindNumericLiteral {
		t := node.AsNumericLiteral().Text
		if t == "0" {
			return output.NewLiteralExpr(0, nil, nil, nil)
		} else if t == "1" {
			return output.NewLiteralExpr(1, nil, nil, nil)
		}
	}
	if node.Kind == ast.KindPropertyAccessExpression {
		pa := node.AsPropertyAccessExpression()
		nameNode := pa.Name()
		if nameNode != nil && nameNode.Kind == ast.KindIdentifier {
			name := nameNode.AsIdentifier().Text
			switch name {
			case "OnPush":
				return output.NewLiteralExpr(0, nil, nil, nil)
			case "Default":
				return output.NewLiteralExpr(1, nil, nil, nil)
			}
		}
	}
	return output.NewWrappedNodeExpr(node, nil, nil, nil)
}

func findComponentPropertyNode(decorator *reflection.Decorator, propName string) *ast.Node {
	if len(decorator.Args) == 0 {
		return nil
	}
	arg := decorator.Args[0]
	if arg.Kind != ast.KindObjectLiteralExpression {
		return nil
	}
	obj := arg.AsObjectLiteralExpression()
	if obj.Properties == nil {
		return nil
	}
	for _, prop := range obj.Properties.Nodes {
		if prop.Kind != ast.KindPropertyAssignment {
			continue
		}
		assign := prop.AsPropertyAssignment()
		if assign.Name().Kind == ast.KindIdentifier {
			if assign.Name().AsIdentifier().Text == propName {
				return assign.Initializer
			}
		}
	}
	return nil
}

func componentDecoratorMetadataExpr(analysis *ComponentAnalysis) output.Expression {
	entries := make([]output.LiteralMapEntry, 0)
	add := func(name string, expr output.Expression) {
		if expr != nil {
			entries = append(entries, output.NewLiteralMapPropertyAssignment(name, expr, false))
		}
	}
	addString := func(name string, value string) {
		if value != "" {
			add(name, output.NewLiteralExpr(value, nil, nil, nil))
		}
	}

	addString("selector", analysis.Selector)
	if analysis.RawImports != nil {
		add("imports", analysis.RawImports)
	} else if node := findComponentPropertyNodeFromAnalysis(analysis, "imports"); node != nil {
		add("imports", output.NewWrappedNodeExpr(node, nil, nil, nil))
	}
	if !analysis.IsStandalone {
		add("standalone", output.NewLiteralExpr(false, nil, nil, nil))
	}
	if len(analysis.ExportAs) > 0 {
		add("exportAs", output.NewLiteralExpr(strings.Join(analysis.ExportAs, ","), nil, nil, nil))
	}
	if analysis.Providers != nil {
		add("providers", analysis.Providers)
	}
	if analysis.ViewProviders != nil {
		add("viewProviders", analysis.ViewProviders)
	}
	if analysis.Animations != nil {
		add("animations", analysis.Animations)
	}
	if analysis.ChangeDetection != nil {
		add("changeDetection", analysis.ChangeDetection)
	}
	if analysis.Encapsulation != 0 {
		add("encapsulation", output.NewLiteralExpr(analysis.Encapsulation, nil, nil, nil))
	}
	if analysis.PreserveWhitespaces != nil {
		add("preserveWhitespaces", output.NewLiteralExpr(*analysis.PreserveWhitespaces, nil, nil, nil))
	}

	// Match Angular's JIT metadata shape after external resources are resolved:
	// templateUrl/styleUrl(s) are replaced with the loaded template/styles.
	add("template", output.NewLiteralExpr(analysis.Template, nil, nil, nil))
	if len(analysis.Styles) > 0 {
		styleExprs := make([]output.Expression, 0, len(analysis.Styles))
		for _, style := range analysis.Styles {
			styleExprs = append(styleExprs, output.NewLiteralExpr(style, nil, nil, nil))
		}
		add("styles", output.NewLiteralArrayExpr(styleExprs, nil, nil, nil))
	}

	return output.NewLiteralMapExpr(entries, nil, nil, nil)
}

func findComponentPropertyNodeFromAnalysis(analysis *ComponentAnalysis, propName string) *ast.Node {
	if analysis.DecoratorNode == nil {
		return nil
	}
	callExpr := analysis.DecoratorNode.AsDecorator().Expression.AsCallExpression()
	if callExpr == nil || callExpr.Arguments == nil || len(callExpr.Arguments.Nodes) == 0 {
		return nil
	}
	arg := callExpr.Arguments.Nodes[0]
	if arg.Kind != ast.KindObjectLiteralExpression {
		return nil
	}
	obj := arg.AsObjectLiteralExpression()
	if obj.Properties == nil {
		return nil
	}
	for _, prop := range obj.Properties.Nodes {
		if prop.Kind != ast.KindPropertyAssignment {
			continue
		}
		assign := prop.AsPropertyAssignment()
		if assign.Name().Kind == ast.KindIdentifier && assign.Name().AsIdentifier().Text == propName {
			return assign.Initializer
		}
	}
	return nil
}

type templateDiagnosticWalker struct {
	diagnostics []ast.Diagnostic
	h           *ComponentDecoratorHandler
	node        *ast.ClassDeclaration
	decorator   *reflection.Decorator
	analysis    *ComponentAnalysis
	scope       *ngscope.CompilationScope
	bound       render3.BoundTarget[directiveMetaAdapter]
}

func (w *templateDiagnosticWalker) walkNodes(nodes []render3.Node) {
	for _, node := range nodes {
		w.walkNode(node)
	}
}

func (w *templateDiagnosticWalker) walkNode(node render3.Node) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *render3.Element:
		w.checkElement(n)
		w.walkNodes(n.Children)
	case *render3.Template:
		w.checkTemplate(n)
		w.walkNodes(n.Children)
	}
}

func (w *templateDiagnosticWalker) makeTemplateDiagnostic(span parse_util.ParseSourceSpan, code ngdiagnostics.ErrorCode, msg string) {
	templateNode := findComponentPropertyNode(w.decorator, "template")
	if templateNode == nil {
		templateNode = findComponentPropertyNode(w.decorator, "templateUrl")
	}
	var diagNode *ast.Node
	if templateNode != nil {
		cloned := *templateNode
		if findComponentPropertyNode(w.decorator, "template") != nil && span.Start != nil && span.Start.File != nil {
			pos := templateNode.Pos() + 1 + span.Start.Offset
			end := templateNode.Pos() + 1 + span.End.Offset
			cloned.Loc = core.NewTextRange(pos, end)
		}
		diagNode = &cloned
	} else {
		diagNode = w.node.AsNode()
	}
	w.diagnostics = append(w.diagnostics, *ngdiagnostics.MakeDiagnostic(
		code,
		diagNode,
		msg,
		nil,
		tsdiagnostics.CategoryError,
	))
}

func (w *templateDiagnosticWalker) checkElement(n *render3.Element) {
	directives := w.bound.GetDirectivesOfNode(n)

	// 1. Unknown Element Check
	isWebComponent := strings.Contains(n.Name, "-")
	isStandardHtml := render3.ElementRegistry.HasElement(n.Name, nil)

	if len(directives) == 0 && !isStandardHtml && n.Name != "ng-container" && n.Name != "ng-content" && n.Name != "ng-template" {
		hostIsStandalone := w.analysis.IsStandalone
		schemasText := "'@Component.schemas'"
		if !hostIsStandalone {
			schemasText = "'@NgModule.schemas'"
		}
		importsExplanation := "included in the '@Component.imports' of this component"
		if !hostIsStandalone {
			importsExplanation = "part of this module"
		}
		msg := fmt.Sprintf("'%s' is not a known element:\n", n.Name)
		msg += fmt.Sprintf("1. If '%s' is an Angular component, then verify that it is %s.\n", n.Name, importsExplanation)
		if isWebComponent {
			msg += fmt.Sprintf("2. If '%s' is a Web Component then add 'CUSTOM_ELEMENTS_SCHEMA' to the %s of this component to suppress this message.", n.Name, schemasText)
		} else {
			msg += fmt.Sprintf("2. To allow any element add 'NO_ERRORS_SCHEMA' to the %s of this component.", schemasText)
		}
		w.makeTemplateDiagnostic(n.GetSourceSpan(), ngdiagnostics.ErrorCode_SCHEMA_INVALID_ELEMENT, msg)
	}

	// 2. Unknown Property / Attribute Binding Check
	for _, prop := range n.Inputs {
		if prop.Type == 4 || prop.Type == 6 {
			continue
		}
		matched := false
		for _, dir := range directives {
			for _, bindingName := range dir.meta.Inputs {
				if bindingName == prop.Name {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			continue
		}

		report := render3.ElementRegistry.ValidateProperty(prop.Name)
		if report.Error {
			w.makeTemplateDiagnostic(prop.SourceSpan, ngdiagnostics.ErrorCode_SCHEMA_INVALID_ATTRIBUTE, *report.Msg)
			continue
		}

		if !render3.ElementRegistry.HasProperty(n.Name, prop.Name, nil) && n.Name != "ng-container" && n.Name != "ng-template" {
			hostIsStandalone := w.analysis.IsStandalone
			decorator := "@Component"
			if !hostIsStandalone {
				decorator = "@NgModule"
			}
			schemas := fmt.Sprintf("'%s.schemas'", decorator)
			errorMsg := fmt.Sprintf("Can't bind to '%s' since it isn't a known property of '%s'.", prop.Name, n.Name)
			if strings.HasPrefix(n.Name, "ng-") {
				errorMsg += fmt.Sprintf("\n1. If '%s' is an Angular directive, then add 'CommonModule' to the '%s.imports' of this component.", prop.Name, decorator)
				errorMsg += fmt.Sprintf("\n2. To allow any property add 'NO_ERRORS_SCHEMA' to the %s of this component.", schemas)
			} else if strings.Contains(n.Name, "-") {
				importsExplanation := "included in the '@Component.imports' of this component"
				if !hostIsStandalone {
					importsExplanation = "part of this module"
				}
				errorMsg += fmt.Sprintf("\n1. If '%s' is an Angular component and it has '%s' input, then verify that it is %s.", n.Name, prop.Name, importsExplanation)
				errorMsg += fmt.Sprintf("\n2. If '%s' is a Web Component then add 'CUSTOM_ELEMENTS_SCHEMA' to the %s of this component to suppress this message.", n.Name, schemas)
				errorMsg += fmt.Sprintf("\n3. To allow any property add 'NO_ERRORS_SCHEMA' to the %s of this component.", schemas)
			}
			w.makeTemplateDiagnostic(prop.SourceSpan, ngdiagnostics.ErrorCode_SCHEMA_INVALID_ATTRIBUTE, errorMsg)
		}
	}

	// 3. Required Inputs Missing Check
	for _, dir := range directives {
		var missingInputs []string
		for _, reqInput := range dir.meta.RequiredInputs {
			bound := false
			for _, input := range n.Inputs {
				if input.Name == reqInput {
					bound = true
					break
				}
			}
			if bound {
				continue
			}
			for _, attr := range n.Attributes {
				if attr.Name == reqInput {
					bound = true
					break
				}
			}
			if bound {
				continue
			}
			missingInputs = append(missingInputs, reqInput)
		}

		if len(missingInputs) > 0 {
			var quoted []string
			for _, mi := range missingInputs {
				quoted = append(quoted, fmt.Sprintf("'%s'", mi))
			}
			pluralStr := ""
			if len(missingInputs) > 1 {
				pluralStr = "s"
			}
			dirKind := "directive"
			if dir.meta.IsComponent {
				dirKind = "component"
			}
			msg := fmt.Sprintf("Required input%s %s from %s %s must be specified.",
				pluralStr,
				strings.Join(quoted, ", "),
				dirKind,
				dir.meta.Name,
			)
			w.makeTemplateDiagnostic(n.GetSourceSpan(), ngdiagnostics.ErrorCode_MISSING_REQUIRED_INPUTS, msg)
		}
	}

	// 4. Missing Reference Target Check
	for _, ref := range n.References {
		if ref.Value != "" {
			found := false
			for _, dir := range directives {
				for _, exportAs := range dir.meta.ExportAs {
					if exportAs == ref.Value {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				msg := fmt.Sprintf("No directive found with exportAs '%s'.", ref.Value)
				w.makeTemplateDiagnostic(ref.SourceSpan, ngdiagnostics.ErrorCode_MISSING_REFERENCE_TARGET, msg)
			}
		}
	}

	// 5. Invalid Banana in Box Check
	for _, event := range n.Outputs {
		if strings.HasPrefix(event.Name, "[") && strings.HasSuffix(event.Name, "]") {
			boundSyntax := event.SourceSpan.ToString()
			expectedBoundSyntax := strings.ReplaceAll(boundSyntax, "("+event.Name+")", "[("+event.Name[1:len(event.Name)-1]+")]")
			msg := fmt.Sprintf("In the two-way binding syntax the parentheses should be inside the brackets, ex. '%s'", expectedBoundSyntax)
			w.makeTemplateDiagnostic(event.SourceSpan, ngdiagnostics.ErrorCode_INVALID_BANANA_IN_BOX, msg)
		}
	}
}

func (w *templateDiagnosticWalker) checkTemplate(n *render3.Template) {
	directives := w.bound.GetDirectivesOfNode(n)

	// 1. Unknown Property / Attribute Binding Check on ng-template
	for _, prop := range n.Inputs {
		matched := false
		for _, dir := range directives {
			for _, bindingName := range dir.meta.Inputs {
				if bindingName == prop.Name {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			continue
		}

		report := render3.ElementRegistry.ValidateProperty(prop.Name)
		if report.Error {
			w.makeTemplateDiagnostic(prop.SourceSpan, ngdiagnostics.ErrorCode_SCHEMA_INVALID_ATTRIBUTE, *report.Msg)
			continue
		}

		if !render3.ElementRegistry.HasProperty("ng-template", prop.Name, nil) {
			hostIsStandalone := w.analysis.IsStandalone
			decorator := "@Component"
			if !hostIsStandalone {
				decorator = "@NgModule"
			}
			schemas := fmt.Sprintf("'%s.schemas'", decorator)
			errorMsg := fmt.Sprintf("Can't bind to '%s' since it isn't a known property of 'ng-template'.", prop.Name)
			errorMsg += fmt.Sprintf("\n1. If '%s' is an Angular directive, then add 'CommonModule' to the '%s.imports' of this component.", prop.Name, decorator)
			errorMsg += fmt.Sprintf("\n2. To allow any property add 'NO_ERRORS_SCHEMA' to the %s of this component.", schemas)
			w.makeTemplateDiagnostic(prop.SourceSpan, ngdiagnostics.ErrorCode_SCHEMA_INVALID_ATTRIBUTE, errorMsg)
		}
	}

	// 2. Required Inputs Missing Check
	for _, dir := range directives {
		var missingInputs []string
		for _, reqInput := range dir.meta.RequiredInputs {
			bound := false
			for _, input := range n.Inputs {
				if input.Name == reqInput {
					bound = true
					break
				}
			}
			if bound {
				continue
			}
			for _, attr := range n.Attributes {
				if attr.Name == reqInput {
					bound = true
					break
				}
			}
			if bound {
				continue
			}
			missingInputs = append(missingInputs, reqInput)
		}

		if len(missingInputs) > 0 {
			var quoted []string
			for _, mi := range missingInputs {
				quoted = append(quoted, fmt.Sprintf("'%s'", mi))
			}
			pluralStr := ""
			if len(missingInputs) > 1 {
				pluralStr = "s"
			}
			dirKind := "directive"
			if dir.meta.IsComponent {
				dirKind = "component"
			}
			msg := fmt.Sprintf("Required input%s %s from %s %s must be specified.",
				pluralStr,
				strings.Join(quoted, ", "),
				dirKind,
				dir.meta.Name,
			)
			w.makeTemplateDiagnostic(n.GetSourceSpan(), ngdiagnostics.ErrorCode_MISSING_REQUIRED_INPUTS, msg)
		}
	}

	// 3. Missing Reference Target Check
	for _, ref := range n.References {
		if ref.Value != "" {
			found := false
			for _, dir := range directives {
				for _, exportAs := range dir.meta.ExportAs {
					if exportAs == ref.Value {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				msg := fmt.Sprintf("No directive found with exportAs '%s'.", ref.Value)
				w.makeTemplateDiagnostic(ref.SourceSpan, ngdiagnostics.ErrorCode_MISSING_REFERENCE_TARGET, msg)
			}
		}
	}

	// 4. Invalid Banana in Box Check
	for _, event := range n.Outputs {
		if strings.HasPrefix(event.Name, "[") && strings.HasSuffix(event.Name, "]") {
			boundSyntax := event.SourceSpan.ToString()
			expectedBoundSyntax := strings.ReplaceAll(boundSyntax, "("+event.Name+")", "[("+event.Name[1:len(event.Name)-1]+")]")
			msg := fmt.Sprintf("In the two-way binding syntax the parentheses should be inside the brackets, ex. '%s'", expectedBoundSyntax)
			w.makeTemplateDiagnostic(event.SourceSpan, ngdiagnostics.ErrorCode_INVALID_BANANA_IN_BOX, msg)
		}
	}
}

type pipeSpanCollector struct {
	expression_parser.RecursiveAstVisitor
	targetName string
	foundNode  *expression_parser.BindingPipe
}

func (c *pipeSpanCollector) VisitPipe(ast *expression_parser.BindingPipe, context any) any {
	if ast.Name == c.targetName {
		c.foundNode = ast
		return nil
	}
	return c.RecursiveAstVisitor.VisitPipe(ast, context)
}

func findPipeSpan(node render3.Node, pipeName string) (parse_util.ParseSourceSpan, bool) {
	collector := &pipeSpanCollector{targetName: pipeName}
	switch n := node.(type) {
	case *render3.BoundText:
		collector.Visit(n.Value, nil)
		if collector.foundNode != nil {
			return n.GetSourceSpan(), true
		}
	case *render3.Element:
		for _, input := range n.Inputs {
			if input != nil {
				collector.Visit(input.Value, nil)
				if collector.foundNode != nil {
					return input.GetSourceSpan(), true
				}
			}
		}
		for _, child := range n.Children {
			if span, ok := findPipeSpan(child, pipeName); ok {
				return span, true
			}
		}
	case *render3.Template:
		for _, attr := range n.TemplateAttrs {
			if bound, ok := attr.(*render3.BoundAttribute); ok && bound != nil {
				collector.Visit(bound.Value, nil)
				if collector.foundNode != nil {
					return bound.GetSourceSpan(), true
				}
			}
		}
		for _, input := range n.Inputs {
			if input != nil {
				collector.Visit(input.Value, nil)
				if collector.foundNode != nil {
					return input.GetSourceSpan(), true
				}
			}
		}
		for _, child := range n.Children {
			if span, ok := findPipeSpan(child, pipeName); ok {
				return span, true
			}
		}
	}
	return parse_util.ParseSourceSpan{}, false
}

func parseQueryPredicate(argNode *ast.Node) interface{} {
	if argNode == nil {
		return []string{""}
	}
	if argNode.Kind == ast.KindStringLiteral {
		return []string{argNode.AsStringLiteral().Text}
	}
	if argNode.Kind == ast.KindIdentifier {
		return render3.MaybeForwardRefExpression{
			Expression: output.NewReadVarExpr(argNode.AsIdentifier().Text, nil, nil, nil),
			ForwardRef: render3.ForwardRefHandlingNone,
		}
	}
	if argNode.Kind == ast.KindCallExpression {
		call := argNode.AsCallExpression()
		if call.Expression.Kind == ast.KindIdentifier && call.Expression.AsIdentifier().Text == "forwardRef" {
			if call.Arguments != nil && len(call.Arguments.Nodes) > 0 {
				fnArg := call.Arguments.Nodes[0]
				var returnExpr *ast.Node
				if fnArg.Kind == ast.KindArrowFunction {
					arrow := fnArg.AsArrowFunction()
					if arrow.Body != nil {
						if arrow.Body.Kind == ast.KindIdentifier {
							returnExpr = arrow.Body
						} else if arrow.Body.Kind == ast.KindBlock {
							block := arrow.Body.AsBlock()
							if block.Statements != nil && len(block.Statements.Nodes) > 0 {
								stmt := block.Statements.Nodes[0]
								if stmt.Kind == ast.KindReturnStatement {
									returnExpr = stmt.AsReturnStatement().Expression
								}
							}
						}
					}
				} else if fnArg.Kind == ast.KindFunctionExpression {
					fn := fnArg.AsFunctionExpression()
					if fn.Body != nil && fn.Body.Kind == ast.KindBlock {
						block := fn.Body.AsBlock()
						if block.Statements != nil && len(block.Statements.Nodes) > 0 {
							stmt := block.Statements.Nodes[0]
							if stmt.Kind == ast.KindReturnStatement {
								returnExpr = stmt.AsReturnStatement().Expression
							}
						}
					}
				}

				if returnExpr != nil && returnExpr.Kind == ast.KindIdentifier {
					return render3.MaybeForwardRefExpression{
						Expression: output.NewReadVarExpr(returnExpr.AsIdentifier().Text, nil, nil, nil),
						ForwardRef: render3.ForwardRefHandlingWrapped,
					}
				}
			}
		}
	}
	return []string{""}
}
