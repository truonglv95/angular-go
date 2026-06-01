package annotations

import (
	"path/filepath"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/scope"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/transform"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
)

type NgModuleAnalysis struct {
	Declarations  []render3.R3Reference
	Imports       []render3.R3Reference
	Exports       []render3.R3Reference
	Bootstrap     []render3.R3Reference
	ProvidersExpr output.Expression
	ImportsExpr   output.Expression
	DecoratorNode *ast.Node
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
		case "imports":
			analysis.Imports = extractR3References(initExpr)
			analysis.ImportsExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		case "exports":
			analysis.Exports = extractR3References(initExpr)
		case "bootstrap":
			analysis.Bootstrap = extractR3References(initExpr)
		case "providers":
			analysis.ProvidersExpr = output.NewWrappedNodeExpr(initExpr, nil, nil, nil)
		}
	}

	// Register module metadata
	if h.metaRegistry != nil {
		var decls, imps, exps []metadata.Reference
		for _, d := range analysis.Declarations {
			name := ""
			if rv, ok := d.Value.(*output.ReadVarExpr); ok { name = rv.Name }
			decls = append(decls, metadata.Reference{Name: name}) // Note: Needs .Node for scope registry to work properly! But we don't have it here yet.
		}
		for _, i := range analysis.Imports {
			name := ""
			if rv, ok := i.Value.(*output.ReadVarExpr); ok { name = rv.Name }
			imps = append(imps, metadata.Reference{Name: name})
		}
		for _, e := range analysis.Exports {
			name := ""
			if rv, ok := e.Value.(*output.ReadVarExpr); ok { name = rv.Name }
			exps = append(exps, metadata.Reference{Name: name})
		}
		h.metaRegistry.RegisterNgModule(node.AsNode(), &metadata.NgModuleMeta{
			Name:         node.Name().AsIdentifier().Text,
			Declarations: decls,
			Imports:      imps,
			Exports:      exps,
		})
	}
	
	// Register components declared in this module
	if h.scopeRegistry != nil {
		// TODO: Register components declared in this module when symbol resolution is available
		// for _, decl := range analysis.Declarations {
		//	if decl.Node != nil {
		//		h.scopeRegistry.RegisterComponentDeclaration(decl.Node, node.AsNode())
		//	}
		// }
	}

	return analysis, nil
}

func (h *NgModuleDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	return &NgModuleResolution{}, nil
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
			Kind:              render3.R3NgModuleMetadataKindGlobal,
			Type: render3.R3Reference{
				Value: output.NewReadVarExpr(className, nil, nil, nil),
			},
			SelectorScopeMode: render3.R3SelectorScopeModeInline,
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

	decMapEntries := []output.LiteralMapEntry{
		output.NewLiteralMapPropertyAssignment("type", output.NewReadVarExpr("NgModule", nil, nil, nil), false),
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

	// Class Debug Info
	var classDebugInfoNode *ast.Node
	filePath := ""
	lineNum := 0
	if node.AsNode().Parent != nil && node.AsNode().Parent.Kind == ast.KindSourceFile {
		sf := node.AsNode().Parent.AsSourceFile()
		rawPath := sf.FileName()

		if rel, err := filepath.Rel(".", rawPath); err == nil && !strings.HasPrefix(rel, "..") {
			filePath = rel
		} else {
			if idx := strings.Index(rawPath, "src/"); idx != -1 {
				filePath = rawPath[idx:]
			} else {
				filePath = rawPath
			}
		}

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

func extractDependenciesNg(refHost reflection.ReflectionHost, classDecl *ast.Node) interface{} {
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
