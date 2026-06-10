package annotations

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	ngdiagnostics "github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/diagnostics"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/incremental/semantic_graph"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
)

type NgModuleAnalysis struct {
	Declarations     []render3.R3Reference
	Imports          []render3.R3Reference
	Exports          []render3.R3Reference
	Bootstrap        []render3.R3Reference
	Schemas          []render3.R3Reference
	DeclarationNodes []*ast.Node
	ImportNodes      []*ast.Node
	ExportNodes      []*ast.Node
	BootstrapNodes   []*ast.Node
	SchemaNodes      []*ast.Node
	ProvidersExpr    output.Expression
	ImportsExpr      output.Expression
	DecoratorNode    *ast.Node
}

type NgModuleResolution struct {
}

type NgModuleDecoratorHandler struct {
	host          reflection.ReflectionHost
	metaRegistry  *metadata.LocalMetadataRegistry
	scopeRegistry *scope.LocalModuleScopeRegistry
}

func NewNgModuleDecoratorHandler(host reflection.ReflectionHost, metaRegistry *metadata.LocalMetadataRegistry, scopeRegistry *scope.LocalModuleScopeRegistry) *NgModuleDecoratorHandler {
	return &NgModuleDecoratorHandler{
		host:          host,
		metaRegistry:  metaRegistry,
		scopeRegistry: scopeRegistry,
	}
}

func (h *NgModuleDecoratorHandler) Name() string {
	return "NgModuleDecoratorHandler"
}

func (h *NgModuleDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
	if decorators == nil {
		return nil
	}
	for _, dec := range decorators {
		if dec.Name == "NgModule" {
			d := dec
			return &d
		}
	}
	return nil
}

func extractR3References(expr *ast.Expression) []render3.R3Reference {
	var refs []render3.R3Reference
	if expr.Kind == ast.KindArrayLiteralExpression {
		arr := expr.AsArrayLiteralExpression()
		if arr.Elements != nil {
			for _, elem := range arr.Elements.Nodes {
				if elem.Kind == ast.KindIdentifier {
					ident := elem.AsIdentifier().Text
					refs = append(refs, render3.R3Reference{
						Value: output.NewReadVarExpr(ident, nil, nil, nil),
					})
				} else {
					refs = append(refs, render3.R3Reference{
						Value: output.NewWrappedNodeExpr(elem, nil, nil, nil),
					})
				}
			}
		}
	}
	return refs
}

func extractASTNodes(expr *ast.Expression) []*ast.Node {
	var nodes []*ast.Node
	if expr.Kind == ast.KindArrayLiteralExpression {
		arr := expr.AsArrayLiteralExpression()
		if arr.Elements != nil {
			for _, elem := range arr.Elements.Nodes {
				nodes = append(nodes, elem)
			}
		}
	}
	return nodes
}

func topLevelClassByName(sourceFile *ast.SourceFile) map[string]*ast.Node {
	classes := make(map[string]*ast.Node)
	if sourceFile == nil || sourceFile.Statements == nil {
		return classes
	}
	for _, stmt := range sourceFile.Statements.Nodes {
		if stmt.Kind != ast.KindClassDeclaration {
			continue
		}
		classDecl := stmt.AsClassDeclaration()
		if classDecl.Name() == nil {
			continue
		}
		classes[classDecl.Name().AsIdentifier().Text] = stmt
	}
	return classes
}

func referenceFromR3(ref render3.R3Reference, classes map[string]*ast.Node) metadata.Reference {
	name := ""
	if rv, ok := ref.Value.(*output.ReadVarExpr); ok {
		name = rv.Name
	}
	return metadata.Reference{Name: name, Node: classes[name]}
}

func (h *NgModuleDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &NgModuleAnalysis{
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

		initExpr := assign.Initializer

		switch name {
		case "declarations":
			analysis.Declarations = extractR3References(initExpr)
			analysis.DeclarationNodes = extractASTNodes(initExpr)
		case "imports":
			analysis.Imports = extractR3References(initExpr)
			analysis.ImportNodes = extractASTNodes(initExpr)
			analysis.ImportsExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		case "exports":
			analysis.Exports = extractR3References(initExpr)
			analysis.ExportNodes = extractASTNodes(initExpr)
		case "bootstrap":
			analysis.Bootstrap = extractR3References(initExpr)
			analysis.BootstrapNodes = extractASTNodes(initExpr)
		case "schemas":
			analysis.Schemas = extractR3References(initExpr)
			analysis.SchemaNodes = extractASTNodes(initExpr)
		case "providers":
			analysis.ProvidersExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		}
	}

	return analysis, nil
}

func (h *NgModuleDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
	analysis := analysisData.(*NgModuleAnalysis)

	if h.metaRegistry != nil {
		classes := topLevelClassByName(ast.GetSourceFileOfNode(node.AsNode()))

		resolveRefNode := func(elem *ast.Node) metadata.Reference {
			if elem == nil {
				return metadata.Reference{}
			}
			if elem.Kind == ast.KindIdentifier {
				decl := h.host.GetDeclarationOfIdentifier(elem)
				if decl != nil && decl.Node != nil {
					return metadata.Reference{
						Name:         elem.AsIdentifier().Text,
						Node:         decl.Node,
						OwningModule: decl.ViaModule,
					}
				}
				// Fallback to same-file class decl
				name := elem.AsIdentifier().Text
				if cNode, ok := classes[name]; ok {
					return metadata.Reference{Name: name, Node: cNode}
				}
			}
			return metadata.Reference{}
		}

		var decls, imps, exps, schemas []metadata.Reference
		for _, d := range analysis.DeclarationNodes {
			decls = append(decls, resolveRefNode(d))
		}
		for _, i := range analysis.ImportNodes {
			imps = append(imps, resolveRefNode(i))
		}
		for _, e := range analysis.ExportNodes {
			exps = append(exps, resolveRefNode(e))
		}
		for _, s := range analysis.SchemaNodes {
			schemas = append(schemas, resolveRefNode(s))
		}

		h.metaRegistry.RegisterNgModule(node.AsNode(), &metadata.NgModuleMeta{
			Name: node.Name().AsIdentifier().Text,
			Ref: metadata.Reference{
				Name: node.Name().AsIdentifier().Text,
				Node: node.AsNode(),
			},
			Declarations: decls,
			Imports:      imps,
			Exports:      exps,
			Schemas:      schemas,
		})
	}

	// Register components declared in this module
	if h.scopeRegistry != nil {
		classes := topLevelClassByName(ast.GetSourceFileOfNode(node.AsNode()))

		resolveRefNode := func(elem *ast.Node) metadata.Reference {
			if elem == nil {
				return metadata.Reference{}
			}
			if elem.Kind == ast.KindIdentifier {
				decl := h.host.GetDeclarationOfIdentifier(elem)
				if decl != nil && decl.Node != nil {
					return metadata.Reference{
						Name:         elem.AsIdentifier().Text,
						Node:         decl.Node,
						OwningModule: decl.ViaModule,
					}
				}
				// Fallback to same-file class decl
				name := elem.AsIdentifier().Text
				if cNode, ok := classes[name]; ok {
					return metadata.Reference{Name: name, Node: cNode}
				}
			}
			return metadata.Reference{}
		}

		for _, decl := range analysis.DeclarationNodes {
			ref := resolveRefNode(decl)
			if ref.Node != nil {
				h.scopeRegistry.RegisterComponentDeclaration(ref.Node, node.AsNode())
			}
		}
	}
}

func (h *NgModuleDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	analysis := analysisData.(*NgModuleAnalysis)
	var diagnostics []ast.Diagnostic

	// 1. Validate declarations
	seenDecls := make(map[*ast.Node]bool)
	classes := topLevelClassByName(ast.GetSourceFileOfNode(node.AsNode()))

	resolveRefNode := func(elem *ast.Node) metadata.Reference {
		if elem == nil {
			return metadata.Reference{}
		}
		if elem.Kind == ast.KindIdentifier {
			decl := h.host.GetDeclarationOfIdentifier(elem)
			if decl != nil && decl.Node != nil {
				return metadata.Reference{
					Name:         elem.AsIdentifier().Text,
					Node:         decl.Node,
					OwningModule: decl.ViaModule,
				}
			}
			name := elem.AsIdentifier().Text
			if cNode, ok := classes[name]; ok {
				return metadata.Reference{Name: name, Node: cNode}
			}
		}
		return metadata.Reference{}
	}

	for _, declExpr := range analysis.DeclarationNodes {
		ref := resolveRefNode(declExpr)
		if ref.Node == nil {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_NGMODULE_INVALID_DECLARATION,
				declExpr,
				fmt.Sprintf("Declaration is not a valid class/type."),
				nil,
				tsdiagnostics.CategoryError,
			))
			continue
		}

		if seenDecls[ref.Node] {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_NGMODULE_INVALID_DECLARATION,
				declExpr,
				fmt.Sprintf("Type %s is declared multiple times in this NgModule.", ref.Name),
				nil,
				tsdiagnostics.CategoryError,
			))
		}
		seenDecls[ref.Node] = true

		dirMeta := h.metaRegistry.GetDirectiveMetadata(ref.Node)
		pipeMeta := h.metaRegistry.GetPipeMetadata(ref.Node)
		isStandalone := false
		if dirMeta != nil && dirMeta.Standalone {
			isStandalone = true
		} else if pipeMeta != nil && pipeMeta.Standalone {
			isStandalone = true
		}
		if isStandalone {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_NGMODULE_DECLARATION_IS_STANDALONE,
				declExpr,
				fmt.Sprintf("The component/directive/pipe '%s' is standalone and cannot be declared in an NgModule.", ref.Name),
				nil,
				tsdiagnostics.CategoryError,
			))
		}

		if h.scopeRegistry != nil {
			if modules := h.scopeRegistry.GetComponentModules(ref.Node); len(modules) > 1 {
				var moduleNames []string
				for _, m := range modules {
					if name := getClassName(m); name != "" {
						moduleNames = append(moduleNames, name)
					}
				}
				diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
					ngdiagnostics.ErrorCode_NGMODULE_DECLARATION_NOT_UNIQUE,
					declExpr,
					fmt.Sprintf("The component/directive/pipe '%s' is declared in multiple NgModules: %s", ref.Name, strings.Join(moduleNames, ", ")),
					nil,
					tsdiagnostics.CategoryError,
				))
			}
		}
	}

	var reader metadata.MetadataReader = h.metaRegistry
	if h.scopeRegistry != nil {
		reader = h.scopeRegistry.MetadataReader()
	}

	// 2. Validate imports
	for _, impExpr := range analysis.ImportNodes {
		ref := resolveRefNode(impExpr)
		if ref.Node == nil {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_NGMODULE_INVALID_IMPORT,
				impExpr,
				fmt.Sprintf("Import is not a valid class/type."),
				nil,
				tsdiagnostics.CategoryError,
			))
			continue
		}

		isNgModule := reader.GetNgModuleMetadata(ref.Node) != nil
		dirMeta := reader.GetDirectiveMetadata(ref.Node)
		pipeMeta := reader.GetPipeMetadata(ref.Node)

		isStandaloneDirectiveOrPipe := false
		if dirMeta != nil && dirMeta.Standalone {
			isStandaloneDirectiveOrPipe = true
		} else if pipeMeta != nil && pipeMeta.Standalone {
			isStandaloneDirectiveOrPipe = true
		}

		if !isNgModule && !isStandaloneDirectiveOrPipe {
			diagnostics = append(diagnostics, *ngdiagnostics.MakeDiagnostic(
				ngdiagnostics.ErrorCode_NGMODULE_INVALID_IMPORT,
				impExpr,
				fmt.Sprintf("The imported class '%s' is not standalone and cannot be imported directly. It must be declared in an NgModule, and that NgModule must be imported instead.", ref.Name),
				nil,
				tsdiagnostics.CategoryError,
			))
		}
	}

	return &NgModuleResolution{}, diagnostics
}

func getClassName(node *ast.Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == ast.KindClassDeclaration {
		decl := node.AsClassDeclaration()
		if decl.Name() != nil && decl.Name().Kind == ast.KindIdentifier {
			return decl.Name().AsIdentifier().Text
		}
	}
	return ""
}

func (h *NgModuleDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	analysis := analysisData.(*NgModuleAnalysis)

	className := ""
	if node.Name() != nil && node.Name().Kind == ast.KindIdentifier {
		className = node.Name().AsIdentifier().Text
	}

	visitor := translator.NewExpressionTranslatorVisitor(factory, importMgr, ast.GetSourceFileOfNode(node.AsNode()).AsNode(), translator.TranslatorOptions{})

	// 1. Compile ɵmod (NgModule)
	ngModuleMeta := &render3.R3NgModuleMetadataGlobal{
		R3NgModuleMetadataCommon: render3.R3NgModuleMetadataCommon{
			Kind: render3.R3NgModuleMetadataKindGlobal,
			Type: render3.R3Reference{
				Value: output.NewReadVarExpr(className, nil, nil, nil),
			},
			SelectorScopeMode: render3.R3SelectorScopeModeInline,
			Schemas:           analysis.Schemas,
		},
		Declarations: analysis.Declarations,
		Imports:      analysis.Imports,
		Exports:      analysis.Exports,
		Bootstrap:    analysis.Bootstrap,
	}

	compiledMod := render3.CompileNgModule(ngModuleMeta)
	var modInitializerNode *ast.Node
	if compiledMod.Expression != nil {
		modInitializerNode = compiledMod.Expression.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	}

	// 2. Compile ɵinj (Injector)
	var injectorImports []output.Expression
	if analysis.ImportsExpr != nil {
		injectorImports = append(injectorImports, analysis.ImportsExpr)
	}
	injMeta := render3.R3InjectorMetadata{
		Name: className,
		Type: render3.R3Reference{
			Value: output.NewReadVarExpr(className, nil, nil, nil),
		},
		Providers: analysis.ProvidersExpr,
		Imports:   injectorImports,
	}

	compiledInj := render3.CompileInjector(injMeta)
	var injInitializerNode *ast.Node
	if compiledInj.Expression != nil {
		injInitializerNode = compiledInj.Expression.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	}

	// 3. Compile ɵfac (Factory)
	factoryDeps := extractDependenciesNg(h.host, node.AsNode())
	facMeta := render3.R3ConstructorFactoryMetadata{
		Name: className,
		Type: render3.R3Reference{
			Value: output.NewReadVarExpr(className, nil, nil, nil),
		},
		Target: render3.FactoryTargetNgModule,
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
	ngModuleName := "NgModule"
	decMapEntries := []output.LiteralMapEntry{
		output.NewLiteralMapPropertyAssignment("type", output.NewExternalExpr(output.ExternalReference{
			ModuleName: &coreModule,
			Name:       &ngModuleName,
		}, nil, nil, nil, nil), false),
	}
	if len(decArgs) > 0 {
		decMapEntries = append(decMapEntries, output.NewLiteralMapPropertyAssignment("args", output.NewLiteralArrayExpr(decArgs, nil, nil, nil), false))
	}

	classMetaExpr := render3.CompileClassMetadata(render3.R3ClassMetadata{
		Type: output.NewReadVarExpr(className, nil, nil, nil),
		Decorators: output.NewLiteralArrayExpr([]output.Expression{
			output.NewLiteralMapExpr(decMapEntries, nil, nil, nil),
		}, nil, nil, nil),
		PropDecorators: nil,
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
			PropertyName: "ɵmod",
			Initializer:  modInitializerNode,
			Statements:   extraStatements,
		},
		{
			PropertyName: "ɵinj",
			Initializer:  injInitializerNode,
		},
	}, nil
}

func (h *NgModuleDecoratorHandler) GetSemanticSymbol(node *ast.ClassDeclaration, analysis any) *semantic_graph.SemanticSymbol {
	a, ok := analysis.(*NgModuleAnalysis)
	if !ok || a == nil {
		return nil
	}
	names := func(refs []render3.R3Reference) []string {
		var res []string
		for _, ref := range refs {
			if rv, ok := ref.Value.(*output.ReadVarExpr); ok && rv.Name != "" {
				res = append(res, rv.Name)
			}
		}
		return res
	}
	return &semantic_graph.SemanticSymbol{
		Path:         ast.GetSourceFileOfNode(node.AsNode()).FileName(),
		Identifier:   node.Name().AsIdentifier().Text,
		Kind:         "ngmodule",
		Declarations: names(a.Declarations),
		Imports:      names(a.Imports),
		Exports:      names(a.Exports),
	}
}

func extractDependenciesNg(refHost reflection.ReflectionHost, classDecl *ast.Node) interface{} {
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
					dep.Token = extractTokenFromNodeNg(dec.Args[0])
				}
			case "Attribute":
				if len(dec.Args) > 0 {
					dep.AttributeNameType = extractTokenFromNodeNg(dec.Args[0])
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

func extractTokenFromNodeNg(node *ast.Node) output.Expression {
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
		recv := extractTokenFromNodeNg(pa.Expression)
		if pa.Name().Kind == ast.KindIdentifier {
			return output.NewReadPropExpr(recv, pa.Name().AsIdentifier().Text, nil, nil, nil, false)
		}
	}
	return output.NewLiteralExpr("UNKNOWN_TOKEN", nil, nil, nil)
}
