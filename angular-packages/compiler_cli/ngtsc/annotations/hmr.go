package annotations

import (
	"fmt"
	"sort"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/imports"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/translator"
	"github.com/microsoft/typescript-go/internal/ast"
)

type hmrDependencies struct {
	local    []render3.R3HmrLocalDependency
	external []render3.R3HmrNamespaceDependency
}

func extractHmrDependencies(
	node *ast.ClassDeclaration,
	definition render3.R3CompiledExpression,
	factory render3.R3CompiledExpression,
	classDebugInfo output.Expression,
) *hmrDependencies {
	className := ""
	if node.Name() != nil && node.Name().Kind == ast.KindIdentifier {
		className = node.Name().AsIdentifier().Text
	}

	visitor := &potentialTopLevelReadsVisitor{
		allReads:       make(map[string]bool),
		namespaceReads: make(map[string]bool),
	}

	if definition.Expression != nil {
		definition.Expression.VisitExpression(visitor, nil)
	}
	for _, stmt := range definition.Statements {
		stmt.VisitStatement(visitor, nil)
	}

	if factory.Expression != nil {
		factory.Expression.VisitExpression(visitor, nil)
	}
	for _, stmt := range factory.Statements {
		stmt.VisitStatement(visitor, nil)
	}

	if classDebugInfo != nil {
		classDebugInfo.VisitExpression(visitor, nil)
	}

	// HMR update modules re-evaluate pieces of decorator metadata and factory
	// expressions in isolation. Some of those references originate from the TS
	// decorator arguments rather than a translated output expression, for example
	// @Inject(SOME_TOKEN) or providers: [forwardRef(() => Service)].
	visitor.addDecoratorArgumentIdentifiers(node.AsNode())

	sourceFile := ast.GetSourceFileOfNode(node.AsNode())
	availableTopLevel := getTopLevelDeclarationNames(sourceFile)

	var local []render3.R3HmrLocalDependency
	seenLocals := make(map[string]bool)

	// Collect into a sorted list first so the order is stable across builds.
	localNames := make([]string, 0, len(visitor.allReads))

	for readName := range visitor.allReads {
		if readName != className && !seenLocals[readName] && availableTopLevel[readName] {
			localNames = append(localNames, readName)
			seenLocals[readName] = true
		}
	}
	sort.Strings(localNames)
	for _, readName := range localNames {
		var runtimeRep output.Expression
		modSpec, propName, _, isNamespace, found := findImportInfo(sourceFile, readName)
		if found {
			if isNamespace {
				runtimeRep = output.NewExternalExpr(output.ExternalReference{
					ModuleName: &modSpec,
					Name:       nil,
				}, nil, nil, nil, nil)
			} else {
				runtimeRep = output.NewExternalExpr(output.ExternalReference{
					ModuleName: &modSpec,
					Name:       &propName,
				}, nil, nil, nil, nil)
			}
		} else {
			runtimeRep = output.NewReadVarExpr(readName, nil, nil, nil)
		}

		local = append(local, render3.R3HmrLocalDependency{
			Name:                  readName,
			RuntimeRepresentation: runtimeRep,
		})
	}

	// B#8 FIX: Sort namespace module names before assigning ɵhmr0, ɵhmr1, ...
	// Go map iteration is random — without sorting the assigned names change each
	// compile causing inconsistent HMR update function signatures.
	namespaceList := make([]string, 0, len(visitor.namespaceReads))
	for modName := range visitor.namespaceReads {
		namespaceList = append(namespaceList, modName)
	}
	sort.Strings(namespaceList)

	var external []render3.R3HmrNamespaceDependency
	for i, modName := range namespaceList {
		external = append(external, render3.R3HmrNamespaceDependency{
			ModuleName:   modName,
			AssignedName: fmt.Sprintf("ɵhmr%d", i),
		})
	}

	return &hmrDependencies{
		local:    local,
		external: external,
	}
}

type potentialTopLevelReadsVisitor struct {
	output.RecursiveAstVisitor
	allReads       map[string]bool
	namespaceReads map[string]bool
}

func (v *potentialTopLevelReadsVisitor) VisitExternalExpr(astExpr *output.ExternalExpr, context any) any {
	if astExpr.Value.ModuleName != nil {
		v.namespaceReads[*astExpr.Value.ModuleName] = true
	}
	return v.RecursiveAstVisitor.VisitExternalExpr(astExpr, context)
}

func (v *potentialTopLevelReadsVisitor) VisitInvokeFunctionExpr(astExpr *output.InvokeFunctionExpr, context any) any {
	astExpr.Fn.VisitExpression(v, context)
	for _, arg := range astExpr.Args {
		arg.VisitExpression(v, context)
	}
	return v.RecursiveAstVisitor.VisitInvokeFunctionExpr(astExpr, context)
}

func (v *potentialTopLevelReadsVisitor) VisitLiteralMapExpr(astExpr *output.LiteralMapExpr, context any) any {
	for _, entry := range astExpr.Entries {
		if prop, ok := entry.(*output.LiteralMapPropertyAssignment); ok {
			prop.Value.VisitExpression(v, context)
		} else if spread, ok := entry.(*output.LiteralMapSpreadAssignment); ok {
			spread.Expression.VisitExpression(v, context)
		}
	}
	return v.RecursiveAstVisitor.VisitLiteralMapExpr(astExpr, context)
}

func (v *potentialTopLevelReadsVisitor) VisitFunctionExpr(astExpr *output.FunctionExpr, context any) any {
	for _, stmt := range astExpr.Statements {
		stmt.VisitStatement(v, context)
	}
	return v.RecursiveAstVisitor.VisitFunctionExpr(astExpr, context)
}

func (v *potentialTopLevelReadsVisitor) VisitReturnStmt(stmt *output.ReturnStatement, context any) any {
	if stmt.Value != nil {
		stmt.Value.VisitExpression(v, context)
	}
	return v.RecursiveAstVisitor.VisitReturnStmt(stmt, context)
}

func (v *potentialTopLevelReadsVisitor) VisitLiteralArrayExpr(astExpr *output.LiteralArrayExpr, context any) any {
	for _, entry := range astExpr.Entries {
		entry.VisitExpression(v, context)
	}
	return v.RecursiveAstVisitor.VisitLiteralArrayExpr(astExpr, context)
}

func (v *potentialTopLevelReadsVisitor) VisitReadVarExpr(astExpr *output.ReadVarExpr, context any) any {
	v.allReads[astExpr.Name] = true
	return v.RecursiveAstVisitor.VisitReadVarExpr(astExpr, context)
}

func (v *potentialTopLevelReadsVisitor) VisitWrappedNodeExpr(astExpr *output.WrappedNodeExpr, context any) any {
	v.addAllTopLevelIdentifiers(astExpr.Node.(*ast.Node))
	return v.RecursiveAstVisitor.VisitWrappedNodeExpr(astExpr, context)
}

func (v *potentialTopLevelReadsVisitor) addDecoratorArgumentIdentifiers(node *ast.Node) {
	if node == nil {
		return
	}
	if node.Kind == ast.KindDecorator {
		expr := node.AsDecorator().Expression
		if expr != nil && expr.Kind == ast.KindCallExpression {
			call := expr.AsCallExpression()
			if call.Arguments != nil {
				for _, arg := range call.Arguments.Nodes {
					v.addAllTopLevelIdentifiers(arg)
				}
			}
		}
		return
	}
	node.ForEachChild(func(child *ast.Node) bool {
		v.addDecoratorArgumentIdentifiers(child)
		return false
	})
}

func (v *potentialTopLevelReadsVisitor) addAllTopLevelIdentifiers(node *ast.Node) {
	if node == nil {
		return
	}
	if node.Kind == ast.KindIdentifier && v.isTopLevelIdentifierReference(node) {
		v.allReads[node.AsIdentifier().Text] = true
	} else {
		node.ForEachChild(func(child *ast.Node) bool {
			v.addAllTopLevelIdentifiers(child)
			return false
		})
	}
}

func (v *potentialTopLevelReadsVisitor) isTopLevelIdentifierReference(identifier *ast.Node) bool {
	var node *ast.Node = identifier
	var parent *ast.Node = node.Parent

	if parent == nil {
		return false
	}

	if parent.Kind == ast.KindParenthesizedExpression && parent.AsParenthesizedExpression().Expression == node {
		for parent != nil && parent.Kind == ast.KindParenthesizedExpression {
			node = parent
			parent = parent.Parent
		}
	}

	if parent.Kind == ast.KindSourceFile {
		return true
	}

	if parent.Kind == ast.KindCallExpression {
		callExpr := parent.AsCallExpression()
		if callExpr.Expression == node {
			return true
		}
		if callExpr.Arguments != nil {
			for _, arg := range callExpr.Arguments.Nodes {
				if arg == node {
					return true
				}
			}
		}
		return false
	}

	if parent.Kind == ast.KindNewExpression {
		newExpr := parent.AsNewExpression()
		if newExpr.Expression == node {
			return true
		}
		if newExpr.Arguments != nil {
			for _, arg := range newExpr.Arguments.Nodes {
				if arg == node {
					return true
				}
			}
		}
		return false
	}

	if parent.Kind == ast.KindPropertyAccessExpression {
		return parent.AsPropertyAccessExpression().Expression == node
	}

	if parent.Kind == ast.KindElementAccessExpression {
		return parent.AsElementAccessExpression().Expression == node
	}

	if parent.Kind == ast.KindTemplateSpan {
		return parent.AsTemplateSpan().Expression == node
	}

	if parent.Kind == ast.KindConditionalExpression {
		cond := parent.AsConditionalExpression()
		return cond.Condition == node || cond.WhenTrue == node || cond.WhenFalse == node
	}

	if parent.Kind == ast.KindBinaryExpression {
		bin := parent.AsBinaryExpression()
		return bin.Left == node || bin.Right == node
	}

	if parent.Kind == ast.KindPrefixUnaryExpression {
		return parent.AsPrefixUnaryExpression().Operand == node
	}

	if parent.Kind == ast.KindPostfixUnaryExpression {
		return parent.AsPostfixUnaryExpression().Operand == node
	}

	if parent.Kind == ast.KindExpressionStatement {
		return parent.AsExpressionStatement().Expression == node
	}

	if parent.Kind == ast.KindReturnStatement {
		return parent.AsReturnStatement().Expression == node
	}

	if parent.Kind == ast.KindArrowFunction {
		return parent.AsArrowFunction().Body == node
	}

	if parent.Kind == ast.KindVariableDeclaration {
		return parent.AsVariableDeclaration().Initializer == node
	}

	if parent.Kind == ast.KindPropertyDeclaration {
		return parent.AsPropertyDeclaration().Initializer == node
	}

	if parent.Kind == ast.KindPropertyAssignment {
		return parent.AsPropertyAssignment().Initializer == node
	}

	if parent.Kind == ast.KindShorthandPropertyAssignment {
		return true
	}

	if parent.Kind == ast.KindArrayLiteralExpression {
		return true
	}

	if parent.Kind == ast.KindSpreadElement {
		return true
	}

	if parent.Kind == ast.KindSpreadAssignment {
		return true
	}

	if parent.Kind == ast.KindAsExpression {
		return parent.AsAsExpression().Expression == node
	}

	if parent.Kind == ast.KindTypeAssertionExpression {
		return parent.AsTypeAssertion().Expression == node
	}

	if parent.Kind == ast.KindNonNullExpression {
		return parent.AsNonNullExpression().Expression == node
	}

	if parent.Kind == ast.KindAwaitExpression {
		return parent.AsAwaitExpression().Expression == node
	}

	if parent.Kind == ast.KindYieldExpression {
		return parent.AsYieldExpression().Expression == node
	}

	if parent.Kind == ast.KindTaggedTemplateExpression {
		return parent.AsTaggedTemplateExpression().Tag == node
	}

	return false
}

func getTopLevelDeclarationNames(sourceFile *ast.SourceFile) map[string]bool {
	results := make(map[string]bool)
	if sourceFile == nil || sourceFile.Statements == nil {
		return results
	}

	for _, node := range sourceFile.Statements.Nodes {
		if node.Kind == ast.KindClassDeclaration {
			if node.AsClassDeclaration().Name() != nil {
				results[node.AsClassDeclaration().Name().AsIdentifier().Text] = true
			}
			continue
		}
		if node.Kind == ast.KindFunctionDeclaration {
			if node.AsFunctionDeclaration().Name() != nil {
				results[node.AsFunctionDeclaration().Name().AsIdentifier().Text] = true
			}
			continue
		}
		if node.Kind == ast.KindEnumDeclaration {
			if node.AsEnumDeclaration().Name() != nil {
				results[node.AsEnumDeclaration().Name().AsIdentifier().Text] = true
			}
			continue
		}
		if node.Kind == ast.KindVariableStatement {
			varList := node.AsVariableStatement().DeclarationList
			if varList != nil {
				for _, decl := range varList.AsVariableDeclarationList().Declarations.Nodes {
					trackBindingName(decl.AsVariableDeclaration().Name(), results)
				}
			}
			continue
		}
		if node.Kind == ast.KindImportDeclaration {
			importClause := node.AsImportDeclaration().ImportClause
			if importClause != nil && importClause.AsImportClause() != nil {
				clause := importClause.AsImportClause()
				if clause.PhaseModifier == ast.KindTypeKeyword {
					continue
				}
				if clause.Name() != nil {
					results[clause.Name().AsIdentifier().Text] = true
				}
				if clause.NamedBindings != nil {
					if clause.NamedBindings.Kind == ast.KindNamespaceImport {
						results[clause.NamedBindings.AsNamespaceImport().Name().AsIdentifier().Text] = true
					} else if clause.NamedBindings.Kind == ast.KindNamedImports {
						for _, el := range clause.NamedBindings.AsNamedImports().Elements.Nodes {
							if !el.AsImportSpecifier().IsTypeOnly {
								results[el.AsImportSpecifier().Name().AsIdentifier().Text] = true
							}
						}
					}
				}
			}
			continue
		}
	}
	return results
}

func trackBindingName(node *ast.Node, results map[string]bool) {
	if node == nil {
		return
	}
	if node.Kind == ast.KindIdentifier {
		results[node.AsIdentifier().Text] = true
	} else if node.Kind == ast.KindObjectBindingPattern {
		for _, el := range node.AsBindingPattern().Elements.Nodes {
			trackBindingName(el.AsBindingElement().Name(), results)
		}
	} else if node.Kind == ast.KindArrayBindingPattern {
		for _, el := range node.AsBindingPattern().Elements.Nodes {
			if el.Kind != ast.KindOmittedExpression {
				trackBindingName(el.AsBindingElement().Name(), results)
			}
		}
	}
}

type hmrModuleImportRewriter struct {
	lookup map[string]string
}

func (r *hmrModuleImportRewriter) RewriteSymbol(symbol string, specifier string) string {
	return symbol
}

func (r *hmrModuleImportRewriter) RewriteSpecifier(specifier string, inContextOfFile string) string {
	return specifier
}

func (r *hmrModuleImportRewriter) RewriteNamespaceImportIdentifier(specifier string, moduleName string) string {
	if val, ok := r.lookup[moduleName]; ok {
		return val
	}
	return specifier
}

// HmrUpdateDeclaration bundles the generated AST nodes for the HMR virtual
// module together with the namespace imports that the function body references.
// Both must be present for the printed virtual source file to be valid JS.
type HmrUpdateDeclaration struct {
	// Nodes contains the export-default function declaration.
	Nodes []*ast.Node
	// Imports contains the namespace import declarations (e.g. import * as i0 from '@angular/core')
	// produced by the HMR-specific ImportManager. These must be prepended to Nodes
	// so the function body can reference i0, i1, etc.
	Imports []*ast.Node
}

func getHmrUpdateDeclaration(
	factory render3.R3CompiledExpression,
	compiled render3.R3CompiledExpression,
	constantStatements []output.Statement,
	meta render3.R3HmrMetadata,
	node *ast.ClassDeclaration,
	importMgr *imports.ImportManager,
	astFactory *ast.NodeFactory,
) *HmrUpdateDeclaration {
	namespaceSpecifiers := make(map[string]string)
	for _, current := range meta.NamespaceDependencies {
		namespaceSpecifiers[current.ModuleName] = current.AssignedName
	}

	importRewriter := &hmrModuleImportRewriter{lookup: namespaceSpecifiers}
	hmrImportMgr := imports.NewImportManager(&imports.ImportManagerConfig{
		ForceGenerateNamespacesForNewImports: true,
		Rewriter:                             importRewriter,
	}, astFactory)

	definitions := []render3.R3HmrDefinitionField{
		{
			Name:        "ɵfac",
			Initializer: factory.Expression,
			Statements:  factory.Statements,
		},
		{
			Name:        "ɵcmp",
			Initializer: compiled.Expression,
			Statements:  compiled.Statements,
		},
	}

	callback := render3.CompileHmrUpdateCallback(definitions, constantStatements, meta)

	sourceFile := ast.GetSourceFileOfNode(node.AsNode())
	visitor := translator.NewExpressionTranslatorVisitor(astFactory, hmrImportMgr, sourceFile.AsNode(), translator.TranslatorOptions{})

	callbackStmt := callback.VisitStatement(visitor, translator.Context{IsStatementMode: true})

	var resultStmts []*ast.Node
	if callbackStmt != nil {
		if n, ok := callbackStmt.(*ast.Node); ok {
			if n.Kind == ast.KindFunctionDeclaration {
				fnDecl := n.AsFunctionDeclaration()
				var newModifiers []*ast.Node
				newModifiers = append(newModifiers, astFactory.NewModifier(ast.KindExportKeyword))
				newModifiers = append(newModifiers, astFactory.NewModifier(ast.KindDefaultKeyword))
				fnDecl.AsMutable().SetModifiers(astFactory.NewModifierList(newModifiers))
				resultStmts = append(resultStmts, n)
			} else {
				resultStmts = append(resultStmts, n)
			}
		}
	}

	if len(resultStmts) == 0 {
		return nil
	}

	// Collect the namespace imports that the function body references.
	// These come from hmrImportMgr (not the main-file importMgr), so they
	// must be captured here before the manager goes out of scope.
	hmrImports := hmrImportMgr.GetAllImports(sourceFile.AsNode())

	return &HmrUpdateDeclaration{
		Nodes:   resultStmts,
		Imports: hmrImports,
	}
}

func getModuleExportNameText(node *ast.Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == ast.KindIdentifier {
		return node.AsIdentifier().Text
	}
	if node.Kind == ast.KindStringLiteral {
		return node.AsStringLiteral().Text
	}
	return ""
}

func findImportInfo(sourceFile *ast.SourceFile, symbol string) (moduleSpecifier string, symbolName string, isDefault bool, isNamespace bool, found bool) {
	if sourceFile == nil || sourceFile.Statements == nil {
		return "", "", false, false, false
	}
	for _, node := range sourceFile.Statements.Nodes {
		if node.Kind == ast.KindImportDeclaration {
			importDecl := node.AsImportDeclaration()
			if importDecl.ModuleSpecifier == nil || importDecl.ModuleSpecifier.Kind != ast.KindStringLiteral {
				continue
			}
			modSpec := importDecl.ModuleSpecifier.AsStringLiteral().Text
			importClause := importDecl.ImportClause
			if importClause == nil || importClause.AsImportClause() == nil {
				continue
			}
			clause := importClause.AsImportClause()
			if clause.PhaseModifier == ast.KindTypeKeyword {
				continue
			}
			if clause.Name() != nil && clause.Name().AsIdentifier().Text == symbol {
				return modSpec, "default", true, false, true
			}
			if clause.NamedBindings != nil {
				if clause.NamedBindings.Kind == ast.KindNamespaceImport {
					ns := clause.NamedBindings.AsNamespaceImport()
					if ns.Name() != nil && ns.Name().AsIdentifier().Text == symbol {
						return modSpec, "", false, true, true
					}
				} else if clause.NamedBindings.Kind == ast.KindNamedImports {
					for _, el := range clause.NamedBindings.AsNamedImports().Elements.Nodes {
						spec := el.AsImportSpecifier()
						if spec.IsTypeOnly {
							continue
						}
						name := spec.Name().AsIdentifier().Text
						if name == symbol {
							if spec.PropertyName != nil {
								propText := getModuleExportNameText((*ast.Node)(spec.PropertyName))
								if propText != "" {
									return modSpec, propText, propText == "default", false, true
								}
							}
							return modSpec, name, name == "default", false, true
						}
					}
				}
			}
		}
	}
	return "", "", false, false, false
}
