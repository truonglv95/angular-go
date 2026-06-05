package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
)

type RewriteResult struct {
	SourceFile    *ast.SourceFile
	Changed       bool
	Errors        []error
	ImportManager *imports.ImportManager
}

// LinkSourceFile rewrites partial declaration calls in a TypeScript source file
// using typescript-go AST transforms. It does not perform text replacement.
func LinkSourceFile(sourceFile *ast.SourceFile, constantPool render3.ConstantPool) RewriteResult {
	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	importManager := imports.NewImportManager(&imports.PresetImportManagerForceNamespaceImports, factory)
	environment := NewLinkerEnvironment(NewTypeScriptAstHost())
	fileLinker := NewFileLinker(environment, sourceFile.FileName(), sourceFile.Text())

	rewriter := &sourceFileRewriter{
		sourceFile:    sourceFile,
		factory:       factory,
		importManager: importManager,
		fileLinker:    fileLinker,
		constantPool:  constantPool,
	}

	var visitor *ast.NodeVisitor
	visitor = ast.NewNodeVisitor(func(node *ast.Node) *ast.Node {
		if node == nil {
			return nil
		}
		if ast.IsCallExpression(node) {
			if replacement := rewriter.tryLinkCall(node); replacement != nil {
				rewriter.changed = true
				return replacement
			}
		}
		return visitor.VisitEachChild(node)
	}, factory, ast.NodeVisitorHooks{})

	// We will create our own constant pool internally if the caller passes nil,
	// but the caller of LinkSourceFile doesn't have access to compiler.ConstantPool right now,
	// because it's in the compiler package. So we'll use a type assertion.
	pool, ok := constantPool.(interface {
		GetStatements() []output.Statement
	})

	var getPoolStatements func() []output.Statement
	if ok {
		getPoolStatements = pool.GetStatements
	}

	linked := visitor.VisitSourceFile(sourceFile)

	if rewriter.changed && getPoolStatements != nil {
		poolStatements := getPoolStatements()
		if len(poolStatements) > 0 {
			var generatedNodes []*ast.Node
			t := translator.NewExpressionTranslatorVisitor(
				factory,
				importManager,
				sourceFile.AsNode(),
				translator.TranslatorOptions{CoreImportExpression: rewriter.coreImport},
			)
			for _, stmt := range poolStatements {
				if astStmt := stmt.VisitStatement(t, translator.Context{IsStatementMode: true}); astStmt != nil {
					if n, ok := astStmt.(*ast.Node); ok {
						generatedNodes = append(generatedNodes, n)
					}
				}
			}

			if len(generatedNodes) > 0 {
				insertAt := 0
				for insertAt < len(linked.Statements.Nodes) && ast.IsImportDeclaration(linked.Statements.Nodes[insertAt]) {
					insertAt++
				}
				var finalNodes []*ast.Node
				finalNodes = append(finalNodes, linked.Statements.Nodes[:insertAt]...)
				finalNodes = append(finalNodes, generatedNodes...)
				finalNodes = append(finalNodes, linked.Statements.Nodes[insertAt:]...)
				linked.Statements.Nodes = finalNodes
				ast.SetParentInChildren(linked.AsNode())
			}
		}
	}

	return RewriteResult{
		SourceFile:    linked,
		Changed:       rewriter.changed,
		Errors:        rewriter.errors,
		ImportManager: importManager,
	}
}

type sourceFileRewriter struct {
	sourceFile    *ast.SourceFile
	factory       *ast.NodeFactory
	importManager *imports.ImportManager
	fileLinker    *FileLinker
	constantPool  render3.ConstantPool
	coreImport    *ast.Node
	changed       bool
	errors        []error
}

func (r *sourceFileRewriter) tryLinkCall(node *ast.Node) *ast.Node {
	call := node.AsCallExpression()
	calleeName := partialDeclarationCalleeName(call.Expression)
	if calleeName == "" || !r.fileLinker.IsPartialDeclaration(calleeName) {
		return nil
	}
	var args []*ast.Node
	if call.Arguments != nil {
		args = call.Arguments.Nodes
	}
	coreImport, err := partialDeclarationNgImport(args)
	if err != nil {
		r.errors = append(r.errors, err)
		return nil
	}
	r.coreImport = coreImport
	linked, err := r.fileLinker.LinkPartialDeclaration(calleeName, args, r.constantPool)
	if err != nil {
		r.errors = append(r.errors, err)
		return nil
	}
	return r.translateExpression(linked.Expression)
}

func (r *sourceFileRewriter) translateExpression(expr render3.Expression) *ast.Node {
	outputExpr, ok := expr.(output.Expression)
	if !ok {
		return nil
	}
	visitor := translator.NewExpressionTranslatorVisitor(
		r.factory,
		r.importManager,
		r.sourceFile.AsNode(),
		translator.TranslatorOptions{CoreImportExpression: r.coreImport},
	)
	translated := outputExpr.VisitExpression(visitor, translator.Context{IsStatementMode: false})
	if node, ok := translated.(*ast.Node); ok {
		return node
	}
	return nil
}

func partialDeclarationNgImport(args []*ast.Node) (*ast.Node, error) {
	if len(args) != 1 {
		return nil, nil
	}
	host := NewTypeScriptAstHost()
	metaObj, err := ParseAstObject(args[0], host)
	if err != nil {
		return nil, err
	}
	if !metaObj.Has("ngImport") {
		return nil, nil
	}
	return metaObj.GetNode("ngImport")
}

func partialDeclarationCalleeName(callee *ast.Node) string {
	if callee == nil {
		return ""
	}
	if ast.IsIdentifier(callee) {
		return callee.Text()
	}
	if ast.IsPropertyAccessExpression(callee) {
		name := callee.AsPropertyAccessExpression().Name()
		if name != nil {
			return name.Text()
		}
	}
	return ""
}
