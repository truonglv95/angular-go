package annotations

import (
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

type PipeImport struct {
	Name string
	Decl reflection.Declaration
}

type PipeAnalysis struct {
	Name          string
	Pure          bool
	IsStandalone  bool
	DecoratorNode *ast.Node
	Imports       []PipeImport
}

type PipeResolution struct {
}

type PipeDecoratorHandler struct {
	host         reflection.ReflectionHost
	metaRegistry *metadata.LocalMetadataRegistry
}

func NewPipeDecoratorHandler(host reflection.ReflectionHost, metaRegistry *metadata.LocalMetadataRegistry) *PipeDecoratorHandler {
	return &PipeDecoratorHandler{
		host:         host,
		metaRegistry: metaRegistry,
	}
}

func (h *PipeDecoratorHandler) Name() string {
	return "PipeDecoratorHandler"
}

func (h *PipeDecoratorHandler) Detect(node *ast.ClassDeclaration, decorators []reflection.Decorator) *reflection.Decorator {
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

func (h *PipeDecoratorHandler) Analyze(node *ast.ClassDeclaration, decorator *reflection.Decorator) (any, []ast.Diagnostic) {
	analysis := &PipeAnalysis{
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
			} else if assign.Initializer.Kind == ast.KindNoSubstitutionTemplateLiteral {
				analysis.Name = assign.Initializer.AsNoSubstitutionTemplateLiteral().Text
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
		}
	}

	return analysis, nil
}

func (h *PipeDecoratorHandler) Register(node *ast.ClassDeclaration, analysisData any) {
	analysis := analysisData.(*PipeAnalysis)
	if h.metaRegistry != nil {
		h.metaRegistry.RegisterPipe(node.AsNode(), &metadata.PipeMeta{
			Ref: metadata.Reference{
				Name: node.Name().AsIdentifier().Text,
				Node: node.AsNode(),
			},
			Name:       analysis.Name, // use analysis.Name which is the pipe name string
			Pure:       analysis.Pure,
			Standalone: analysis.IsStandalone,
		})
	}
}

func (h *PipeDecoratorHandler) Resolve(node *ast.ClassDeclaration, analysisData any) (any, []ast.Diagnostic) {
	return &PipeResolution{}, nil
}

func (h *PipeDecoratorHandler) CompileFull(node *ast.ClassDeclaration, analysisData any, resolutionData any, pool *compiler.ConstantPool, importMgr *imports.ImportManager, factory *ast.NodeFactory) ([]transform.CompileResult, []ast.Diagnostic) {
	analysis := analysisData.(*PipeAnalysis)

	className := ""
	if node.Name() != nil && node.Name().Kind == ast.KindIdentifier {
		className = node.Name().AsIdentifier().Text
	}

	meta := render3.R3PipeMetadata{
		Name: className,
		Type: render3.R3Reference{
			Value: output.NewReadVarExpr(className, nil, nil, nil),
		},
		TypeArgumentCount: 0,
		PipeName:          &analysis.Name,
		Pure:              analysis.Pure,
		IsStandalone:      analysis.IsStandalone,
	}

	compiled := render3.CompilePipeFromMetadata(meta)

	visitor := translator.NewExpressionTranslatorVisitor(factory, importMgr, ast.GetSourceFileOfNode(node.AsNode()).AsNode(), translator.TranslatorOptions{})

	var initializerNode *ast.Node
	if compiled.Expression != nil {
		initializerNode = compiled.Expression.VisitExpression(visitor, translator.Context{IsStatementMode: false}).(*ast.Node)
	}

	factoryDeps := extractDependenciesPipe(h.host, node.AsNode())

	facMeta := render3.R3ConstructorFactoryMetadata{
		Name: className,
		Type: render3.R3Reference{
			Value: output.NewReadVarExpr(className, nil, nil, nil),
		},
		Target: render3.FactoryTargetPipe,
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
	pipeName := "Pipe"
	decMapEntries := []output.LiteralMapEntry{
		output.NewLiteralMapPropertyAssignment("type", output.NewExternalExpr(output.ExternalReference{
			ModuleName: &coreModule,
			Name:       &pipeName,
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

	// Class Debug Info
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
			PropertyName: "ɵpipe",
			Initializer:  initializerNode,
			Statements:   extraStatements,
		},
	}, nil
}

func extractDependenciesPipe(refHost reflection.ReflectionHost, classDecl *ast.Node) interface{} {
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
					dep.Token = extractTokenFromNodePipe(dec.Args[0])
				}
			case "Attribute":
				if len(dec.Args) > 0 {
					dep.AttributeNameType = extractTokenFromNodePipe(dec.Args[0])
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

func extractTokenFromNodePipe(node *ast.Node) output.Expression {
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
		recv := extractTokenFromNodePipe(pa.Expression)
		if pa.Name().Kind == ast.KindIdentifier {
			return output.NewReadPropExpr(recv, pa.Name().AsIdentifier().Text, nil, nil, nil, false)
		}
	}
	return output.NewLiteralExpr("UNKNOWN_TOKEN", nil, nil, nil)
}

func (h *PipeDecoratorHandler) GetSemanticSymbol(node *ast.ClassDeclaration, analysis any) *semantic_graph.SemanticSymbol {
	a, ok := analysis.(*PipeAnalysis)
	if !ok || a == nil {
		return nil
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
		Kind:           "pipe",
		PipeName:       a.Name,
		Pure:           a.Pure,
		Standalone:     a.IsStandalone,
		Imports:        imps,
		TypeParameters: tps,
	}
}
