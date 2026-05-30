package transform

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
)

// TransformSourceFile traverses the source file AST, detects Angular Ivy decorators,
// strips them, and injects compiled Ivy static properties (ɵcmp, ɵdir, etc.) into classes.
func TransformSourceFile(sf *ast.SourceFile, host reflection.ReflectionHost) {
	if sf == nil || sf.Statements == nil {
		return
	}

	factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
	var hasTransformedClass bool
	var newStatements []*ast.Node

	for _, node := range sf.Statements.Nodes {
		if node.Kind == ast.KindClassDeclaration {
			beforeNodes, metadataNodes, ok := transformClass(node, host, factory, sf)
			if ok {
				hasTransformedClass = true
				for _, beforeNode := range beforeNodes {
					if beforeNode != nil {
						fixupParentReferences(beforeNode, sf.AsNode())
						newStatements = append(newStatements, beforeNode)
					}
				}
				newStatements = append(newStatements, node)
				for _, metadataNode := range metadataNodes {
					if metadataNode != nil {
						fixupParentReferences(metadataNode, sf.AsNode())
						newStatements = append(newStatements, metadataNode)
					}
				}
			} else {
				newStatements = append(newStatements, node)
			}
		} else {
			newStatements = append(newStatements, node)
		}
	}
	sf.Statements.Nodes = newStatements

	if hasTransformedClass {
		// Prepend: import * as i0 from '@angular/core';
		i0Ident := factory.NewIdentifier("i0")
		namespaceImport := factory.NewNamespaceImport(i0Ident)
		importClause := factory.NewImportClause(
			ast.KindUnknown,
			nil,
			namespaceImport,
		)
		moduleSpecifier := factory.NewStringLiteral("@angular/core", 0)
		importDecl := factory.NewImportDeclaration(
			nil,
			importClause,
			moduleSpecifier,
			nil,
		)

		fixupParentReferences(importDecl, sf.AsNode())

		sf.Statements.Nodes = append([]*ast.Node{importDecl}, sf.Statements.Nodes...)
	}
}

func fixupParentReferences(node *ast.Node, parent *ast.Node) {
	if node == nil {
		return
	}
	node.Parent = parent
	node.ForEachChild(func(child *ast.Node) bool {
		fixupParentReferences(child, node)
		return false
	})
}

type templateState struct {
	className        string
	templateName     string
	templateText     string
	declsCount       int
	varsCount        int
	currentUpdateIdx int
	consts           *[]*ast.Node
	createStatements []*ast.Node
	updateStatements []*ast.Node
	subTemplates     []*ast.Node
	helpers          []*ast.Node
	replacements     map[string]string
	isSubTemplate    bool
	hasListener      bool
	dataIndex        *int
	sharedCtx        *string
	viewRefName      string
	preAllocViewSlot int
	ctxName          string
}

func (s *templateState) allocateDataSlot() int {
	*s.dataIndex++
	return *s.dataIndex
}

func ASTToString(ast expression_parser.AST) string {
	if ast == nil {
		return ""
	}
	switch n := ast.(type) {
	case *expression_parser.ASTWithSource:
		return n.Source
	case *expression_parser.ImplicitReceiver:
		return ""
	case *expression_parser.ThisReceiver:
		return "this"
	case *expression_parser.PropertyRead:
		recv := ASTToString(n.Receiver)
		if recv == "" {
			return n.Name
		}
		return recv + "." + n.Name
	case *expression_parser.SafePropertyRead:
		recv := ASTToString(n.Receiver)
		if recv == "" {
			return n.Name
		}
		return recv + "?." + n.Name
	case *expression_parser.KeyedRead:
		recv := ASTToString(n.Receiver)
		key := ASTToString(n.Key)
		return recv + "[" + key + "]"
	case *expression_parser.MethodCall:
		recv := ASTToString(n.Receiver)
		var args []string
		for _, arg := range n.Args {
			args = append(args, ASTToString(arg))
		}
		argsStr := strings.Join(args, ", ")
		if recv == "" {
			return n.Name + "(" + argsStr + ")"
		}
		return recv + "." + n.Name + "(" + argsStr + ")"
	case *expression_parser.SafeMethodCall:
		recv := ASTToString(n.Receiver)
		var args []string
		for _, arg := range n.Args {
			args = append(args, ASTToString(arg))
		}
		argsStr := strings.Join(args, ", ")
		if recv == "" {
			return n.Name + "(" + argsStr + ")"
		}
		return recv + "?." + n.Name + "(" + argsStr + ")"
	case *expression_parser.Binary:
		left := ASTToString(n.Left)
		right := ASTToString(n.Right)
		return left + " " + n.Operation + " " + right
	case *expression_parser.Conditional:
		cond := ASTToString(n.Condition)
		trueExp := ASTToString(n.TrueExp)
		falseExp := ASTToString(n.FalseExp)
		return cond + " ? " + trueExp + " : " + falseExp
	case *expression_parser.PrefixNot:
		expr := ASTToString(n.Expression)
		return "!" + expr
	case *expression_parser.LiteralPrimitive:
		switch v := n.Value.(type) {
		case string:
			return "'" + strings.ReplaceAll(v, "'", "\\'") + "'"
		default:
			return fmt.Sprintf("%v", v)
		}
	case *expression_parser.LiteralArray:
		var items []string
		for _, item := range n.Expressions {
			items = append(items, ASTToString(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case *expression_parser.LiteralMap:
		var entries []string
		for i, key := range n.Keys {
			val := ASTToString(n.Values[i])
			if propKey, ok := key.(*expression_parser.LiteralMapPropertyKey); ok {
				entries = append(entries, propKey.Key + ": " + val)
			} else {
				entries = append(entries, "... " + val)
			}
		}
		return "{" + strings.Join(entries, ", ") + "}"
	case *expression_parser.BindingPipe:
		exp := ASTToString(n.Exp)
		var args []string
		for _, arg := range n.Args {
			args = append(args, ASTToString(arg))
		}
		if len(args) > 0 {
			return exp + " | " + n.Name + ":" + strings.Join(args, ":")
		}
		return exp + " | " + n.Name
	case *expression_parser.EmptyExpr:
		return ""
	default:
		return ""
	}
}

func (s *templateState) getExpressionText(ast expression_parser.AST) string {
	if ast == nil {
		return ""
	}
	return ASTToString(ast)
}

func (s *templateState) translate(exprStr string, factory *ast.NodeFactory, ctxName string) *ast.Node {
	return translateExpression(exprStr, factory, s.replacements, ctxName)
}


func serializeConsts(nodes []*ast.Node) string {
	var parts []string
	for _, n := range nodes {
		if n.Kind == ast.KindStringLiteral {
			parts = append(parts, n.AsStringLiteral().Text)
		} else if n.Kind == ast.KindNumericLiteral {
			parts = append(parts, n.AsNumericLiteral().Text)
		}
	}
	return strings.Join(parts, "|")
}

func getConstsForElement(el *render3.Element, factory *ast.NodeFactory) *ast.Node {
	var nodes []*ast.Node
	var plainAttrs []string
	var classes []string

	for _, attr := range el.Attributes {
		if attr.Name == "class" {
			classes = append(classes, strings.Fields(attr.Value)...)
		} else if attr.Name != "style" {
			plainAttrs = append(plainAttrs, attr.Name, attr.Value)
		}
	}

	var bindings []string
	for _, out := range el.Outputs {
		bindings = append(bindings, out.Name)
	}
	for _, in := range el.Inputs {
		if in.Type == 0 || in.Type == 1 {
			if in.Name != "class" && in.Name != "style" && in.Name != "className" {
				bindings = append(bindings, in.Name)
			}
		}
	}

	for _, p := range plainAttrs {
		nodes = append(nodes, factory.NewStringLiteral(p, 0))
	}

	if len(classes) > 0 {
		nodes = append(nodes, factory.NewNumericLiteral("1", 0))
		for _, c := range classes {
			nodes = append(nodes, factory.NewStringLiteral(c, 0))
		}
	}

	if len(bindings) > 0 {
		nodes = append(nodes, factory.NewNumericLiteral("3", 0))
		for _, b := range bindings {
			nodes = append(nodes, factory.NewStringLiteral(b, 0))
		}
	}

	if len(nodes) == 0 {
		return nil
	}
	return factory.NewArrayLiteralExpression((*ast.ElementList)(factory.NewNodeList(nodes)), false)
}

func getContainerConstsForElement(el *render3.Element, factory *ast.NodeFactory) *ast.Node {
	var nodes []*ast.Node
	var plainAttrs []string
	var classes []string

	for _, attr := range el.Attributes {
		if attr.Name == "class" {
			classes = append(classes, strings.Fields(attr.Value)...)
		} else if attr.Name != "style" {
			plainAttrs = append(plainAttrs, attr.Name, attr.Value)
		}
	}

	var bindings []string
	for _, out := range el.Outputs {
		bindings = append(bindings, out.Name)
	}
	for _, in := range el.Inputs {
		if in.Name != "class" && in.Name != "style" && in.Name != "className" {
			bindings = append(bindings, in.Name)
		}
	}

	for _, p := range plainAttrs {
		nodes = append(nodes, factory.NewStringLiteral(p, 0))
	}

	if len(classes) > 0 {
		nodes = append(nodes, factory.NewNumericLiteral("1", 0))
		for _, c := range classes {
			nodes = append(nodes, factory.NewStringLiteral(c, 0))
		}
	}

	if len(bindings) > 0 {
		nodes = append(nodes, factory.NewNumericLiteral("3", 0))
		for _, b := range bindings {
			nodes = append(nodes, factory.NewStringLiteral(b, 0))
		}
	}

	if len(nodes) == 0 {
		return nil
	}
	return factory.NewArrayLiteralExpression((*ast.ElementList)(factory.NewNodeList(nodes)), false)
}

func (s *templateState) getOrAddContainerConst(el *render3.Element, factory *ast.NodeFactory) string {
	if s.consts == nil {
		return ""
	}
	constsNode := getContainerConstsForElement(el, factory)
	if constsNode == nil {
		return ""
	}
	if constsNode.Kind == ast.KindArrayLiteralExpression {
		arr := constsNode.AsArrayLiteralExpression()
		constStr := serializeConsts(arr.Elements.Nodes)
		for i, existingNode := range *s.consts {
			if existingNode.Kind == ast.KindArrayLiteralExpression {
				existingArr := existingNode.AsArrayLiteralExpression()
				if serializeConsts(existingArr.Elements.Nodes) == constStr {
					return fmt.Sprintf("%d", i)
				}
			}
		}
	}
	idx := len(*s.consts)
	*s.consts = append(*s.consts, constsNode)
	return fmt.Sprintf("%d", idx)
}

func (s *templateState) getOrAddConst(el *render3.Element, factory *ast.NodeFactory) string {
	if s.consts == nil {
		return ""
	}
	constsNode := getConstsForElement(el, factory)
	if constsNode == nil {
		return ""
	}
	if constsNode.Kind == ast.KindArrayLiteralExpression {
		arr := constsNode.AsArrayLiteralExpression()
		constStr := serializeConsts(arr.Elements.Nodes)
		for i, existingNode := range *s.consts {
			if existingNode.Kind == ast.KindArrayLiteralExpression {
				existingArr := existingNode.AsArrayLiteralExpression()
				if serializeConsts(existingArr.Elements.Nodes) == constStr {
					return fmt.Sprintf("%d", i)
				}
			}
		}
	}
	idx := len(*s.consts)
	*s.consts = append(*s.consts, constsNode)
	return fmt.Sprintf("%d", idx)
}

func (s *templateState) extractViewConsts(nodes []render3.Node, factory *ast.NodeFactory) {
	var subTemplates [][]render3.Node
	s.extractViewConstsLevel(nodes, &subTemplates, factory)
	
	for len(subTemplates) > 0 {
		var nextLevel [][]render3.Node
		for _, sub := range subTemplates {
			s.extractViewConstsLevel(sub, &nextLevel, factory)
		}
		subTemplates = nextLevel
	}
}

func (s *templateState) extractViewConstsLevel(nodes []render3.Node, subTemplates *[][]render3.Node, factory *ast.NodeFactory) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *render3.Element:
			s.getOrAddConst(n, factory)
			if len(n.Children) > 0 {
				s.extractViewConstsLevel(n.Children, subTemplates, factory)
			}
		case *render3.IfBlock:
			for _, branch := range n.Branches {
				if len(branch.Children) > 0 {
					if el, ok := branch.Children[0].(*render3.Element); ok {
						s.getOrAddContainerConst(el, factory)
					}
					*subTemplates = append(*subTemplates, branch.Children)
				}
			}
		case *render3.ForLoopBlock:
			if len(n.Children) > 0 {
				if el, ok := n.Children[0].(*render3.Element); ok {
					s.getOrAddContainerConst(el, factory)
				}
				*subTemplates = append(*subTemplates, n.Children)
			}
			if n.Empty != nil && len(n.Empty.Children) > 0 {
				if el, ok := n.Empty.Children[0].(*render3.Element); ok {
					s.getOrAddContainerConst(el, factory)
				}
				*subTemplates = append(*subTemplates, n.Empty.Children)
			}
		case *render3.SwitchBlock:
			for _, group := range n.Groups {
				if len(group.Children) > 0 {
					if el, ok := group.Children[0].(*render3.Element); ok {
						s.getOrAddContainerConst(el, factory)
					}
					*subTemplates = append(*subTemplates, group.Children)
				}
			}
		case *render3.Template:
			if len(n.Children) > 0 {
				*subTemplates = append(*subTemplates, n.Children)
			}
		}
	}
}

func hasListener(nodes []render3.Node) bool {
	for _, n := range nodes {
		if el, ok := n.(*render3.Element); ok {
			if len(el.Outputs) > 0 {
				return true
			}
			if hasListener(el.Children) {
				return true
			}
		} else if templ, ok := n.(*render3.Template); ok {
			if hasListener(templ.Children) {
				return true
			}
		}
	}
	return false
}

func (s *templateState) processNodes(nodes []render3.Node, factory *ast.NodeFactory) {
	if !s.isSubTemplate {
		s.extractViewConsts(nodes, factory)
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *render3.Element:
			elIdx := s.declsCount
			s.declsCount++

			var startArgs []*ast.Node
			startArgs = append(startArgs, factory.NewNumericLiteral(fmt.Sprintf("%d", elIdx), 0))
			startArgs = append(startArgs, factory.NewStringLiteral(n.Name, 0))

			constIdx := s.getOrAddConst(n, factory)
			if constIdx != "" {
				startArgs = append(startArgs, factory.NewNumericLiteral(constIdx, 0))
			}

			elStartCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵdomElementStart"),
					ast.NodeFlagsNone,
				),
				nil,
				nil,
				(*ast.ElementList)(factory.NewNodeList(startArgs)),
				ast.NodeFlagsNone,
			)
			s.createStatements = append(s.createStatements, factory.NewExpressionStatement(elStartCall))

			// Process event bindings (outputs)
			for _, output := range n.Outputs {
				ctxName := "ctx"
				if s.isSubTemplate {
					if !s.hasListener {
						s.hasListener = true
						if s.viewRefName == "" {
							var viewSlot int
							if s.preAllocViewSlot > 0 {
								viewSlot = s.preAllocViewSlot
								s.preAllocViewSlot = 0
							} else {
								viewSlot = s.allocateDataSlot()
							}
							s.viewRefName = fmt.Sprintf("_r%d", viewSlot)
							if s.sharedCtx != nil && *s.sharedCtx != "" {
								s.ctxName = *s.sharedCtx
							} else {
								ctxSlot := *s.dataIndex
								*s.dataIndex++
								s.ctxName = fmt.Sprintf("ctx_r%d", ctxSlot)
								if s.sharedCtx != nil {
									*s.sharedCtx = s.ctxName
								}
							}
						}
					}
					ctxName = s.ctxName
				}
				usedLocals := make(map[string]string)
				handlerExpr := translateExprAST(output.Handler, factory, s.replacements, ctxName, usedLocals)
				if handlerExpr == nil {
					handlerExpr = factory.NewCallExpression(factory.NewIdentifier("onClick"), nil, nil, (*ast.ElementList)(factory.NewNodeList([]*ast.Node{})), ast.NodeFlagsNone)
				}
				

				var listenerBody *ast.Node
				if s.isSubTemplate {
					var listenerBodyNodes []*ast.Node
					
					// Restorer
					if len(usedLocals) > 0 {
						restoreCall := factory.NewCallExpression(factory.NewPropertyAccessExpression(factory.NewIdentifier("i0"), nil, factory.NewIdentifier("ɵɵrestoreView"), ast.NodeFlagsNone), nil, nil, (*ast.ElementList)(factory.NewNodeList([]*ast.Node{factory.NewIdentifier(s.viewRefName)})), ast.NodeFlagsNone)
						restoredCtxDecl := factory.NewVariableStatement(nil, factory.NewVariableDeclarationList((*ast.VariableDeclarationNodeList)(factory.NewNodeList([]*ast.Node{
							factory.NewVariableDeclaration((*ast.BindingName)(factory.NewIdentifier("restoredCtx")), nil, nil, restoreCall),
						})), ast.NodeFlagsConst))
						listenerBodyNodes = append(listenerBodyNodes, restoredCtxDecl)
						
						for origName, renamed := range usedLocals {
							if origName == "$event" {
								continue
							}
							// Determine prop name based on original identifier name
							propName := origName
							if origName == s.replacements["$implicit_for_orig_name"] {
								propName = "$implicit"
							}
							
							localDecl := factory.NewVariableStatement(nil, factory.NewVariableDeclarationList((*ast.VariableDeclarationNodeList)(factory.NewNodeList([]*ast.Node{
								factory.NewVariableDeclaration((*ast.BindingName)(factory.NewIdentifier(renamed)), nil, nil, 
									factory.NewPropertyAccessExpression(factory.NewIdentifier("restoredCtx"), nil, factory.NewIdentifier(propName), ast.NodeFlagsNone),
								),
							})), ast.NodeFlagsConst))
							listenerBodyNodes = append(listenerBodyNodes, localDecl)
						}
					} else {
						restoreCall := factory.NewCallExpression(factory.NewPropertyAccessExpression(factory.NewIdentifier("i0"), nil, factory.NewIdentifier("ɵɵrestoreView"), ast.NodeFlagsNone), nil, nil, (*ast.ElementList)(factory.NewNodeList([]*ast.Node{factory.NewIdentifier(s.viewRefName)})), ast.NodeFlagsNone)
						listenerBodyNodes = append(listenerBodyNodes, factory.NewExpressionStatement(restoreCall))
					}
					
					nextCtxCall := factory.NewCallExpression(factory.NewPropertyAccessExpression(factory.NewIdentifier("i0"), nil, factory.NewIdentifier("ɵɵnextContext"), ast.NodeFlagsNone), nil, nil, (*ast.ElementList)(factory.NewNodeList([]*ast.Node{})), ast.NodeFlagsNone)
					ctxDecl := factory.NewVariableStatement(nil, factory.NewVariableDeclarationList((*ast.VariableDeclarationNodeList)(factory.NewNodeList([]*ast.Node{factory.NewVariableDeclaration((*ast.BindingName)(factory.NewIdentifier(s.ctxName)), nil, nil, nextCtxCall)})), ast.NodeFlagsConst))
					resetCall := factory.NewCallExpression(factory.NewPropertyAccessExpression(factory.NewIdentifier("i0"), nil, factory.NewIdentifier("ɵɵresetView"), ast.NodeFlagsNone), nil, nil, (*ast.ElementList)(factory.NewNodeList([]*ast.Node{handlerExpr})), ast.NodeFlagsNone)
					listenerReturn := factory.NewReturnStatement(resetCall)
					listenerBodyNodes = append(listenerBodyNodes, ctxDecl, listenerReturn)
					
					listenerBody = factory.NewBlock((*ast.StatementList)(factory.NewNodeList(listenerBodyNodes)), true)
				} else {
					listenerReturn := factory.NewReturnStatement(handlerExpr)
					listenerBody = factory.NewBlock((*ast.StatementList)(factory.NewNodeList([]*ast.Node{listenerReturn})), true)
				}
				tName := s.templateName
				if tName == "" {
					tName = s.className + "_Template"
				}
				var listenerParams []*ast.Node
				if _, hasEvent := usedLocals["$event"]; hasEvent {
					listenerParams = append(listenerParams, factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("$event"), nil, nil, nil))
				}
				
				listenerFn := factory.NewFunctionExpression(
					nil,
					nil,
					factory.NewIdentifier(fmt.Sprintf("%s_%s_%s_%d_listener", tName, n.Name, output.Name, elIdx)),
					nil,
					(*ast.ParameterList)(factory.NewNodeList(listenerParams)),
					nil,
					nil,
					(*ast.FunctionBody)(listenerBody),
				)
				listenerCall := factory.NewCallExpression(
					factory.NewPropertyAccessExpression(
						factory.NewIdentifier("i0"),
						nil,
						factory.NewIdentifier("ɵɵdomListener"),
						ast.NodeFlagsNone,
					),
					nil,
					nil,
					(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
						factory.NewStringLiteral(output.Name, 0),
						listenerFn,
					})),
					ast.NodeFlagsNone,
				)
				s.createStatements = append(s.createStatements, factory.NewExpressionStatement(listenerCall))
			}

			// Process inputs in update phase before processing children
			if len(n.Inputs) > 0 {
				diff := elIdx - s.currentUpdateIdx
				if diff > 0 {
					var advanceArgs []*ast.Node
					if diff == 1 {
						advanceArgs = []*ast.Node{}
					} else {
						advanceArgs = []*ast.Node{factory.NewNumericLiteral(fmt.Sprintf("%d", diff), 0)}
					}
					advanceCall := factory.NewCallExpression(
						factory.NewPropertyAccessExpression(
							factory.NewIdentifier("i0"),
							nil,
							factory.NewIdentifier("ɵɵadvance"),
							ast.NodeFlagsNone,
						),
						nil, nil,
						(*ast.ElementList)(factory.NewNodeList(advanceArgs)),
						ast.NodeFlagsNone,
					)
					s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(advanceCall))
				}
				s.currentUpdateIdx = elIdx
			}

			for _, input := range n.Inputs {
				if input.Type == 3 || input.Type == 2 || (input.Type == 0 && (input.Name == "class" || input.Name == "style")) {
					s.varsCount += 2
				} else {
					s.varsCount++
				}

				valExpr := translateExprAST(input.Value, factory, s.replacements, "ctx", nil)
				if valExpr == nil { valExpr = factory.NewPropertyAccessExpression(factory.NewIdentifier("ctx"), nil, factory.NewIdentifier("title"), ast.NodeFlagsNone) }

				var propCall *ast.Node
				if input.Name == "class" && input.Type == 0 {
					propCall = factory.NewCallExpression(
						factory.NewPropertyAccessExpression(
							factory.NewIdentifier("i0"),
							nil,
							factory.NewIdentifier("ɵɵclassMap"),
							ast.NodeFlagsNone,
						),
						nil, nil,
						(*ast.ElementList)(factory.NewNodeList([]*ast.Node{valExpr})),
						ast.NodeFlagsNone,
					)
				} else if input.Type == 2 { // Class binding (e.g. [class.featured])
					className := input.Name
					propCall = factory.NewCallExpression(
						factory.NewPropertyAccessExpression(
							factory.NewIdentifier("i0"),
							nil,
							factory.NewIdentifier("ɵɵclassProp"),
							ast.NodeFlagsNone,
						),
						nil, nil,
						(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
							factory.NewStringLiteral(className, 0),
							valExpr,
						})),
						ast.NodeFlagsNone,
					)
				} else if input.Type == 3 { // Style binding (e.g. [style.background])
					styleName := input.Name
					propCall = factory.NewCallExpression(
						factory.NewPropertyAccessExpression(
							factory.NewIdentifier("i0"),
							nil,
							factory.NewIdentifier("ɵɵstyleProp"),
							ast.NodeFlagsNone,
						),
						nil, nil,
						(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
							factory.NewStringLiteral(styleName, 0),
							valExpr,
						})),
						ast.NodeFlagsNone,
					)
				} else {
					propCall = factory.NewCallExpression(
						factory.NewPropertyAccessExpression(
							factory.NewIdentifier("i0"),
							nil,
							factory.NewIdentifier("ɵɵdomProperty"),
							ast.NodeFlagsNone,
						),
						nil, nil,
						(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
							factory.NewStringLiteral(input.Name, 0),
							valExpr,
						})),
						ast.NodeFlagsNone,
					)
				}
				s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(propCall))
			}

			// Process children
			s.processNodes(n.Children, factory)

			elEndCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵdomElementEnd"),
					ast.NodeFlagsNone,
				),
				nil,
				nil,
				(*ast.ElementList)(factory.NewNodeList(nil)),
				ast.NodeFlagsNone,
			)
			s.createStatements = append(s.createStatements, factory.NewExpressionStatement(elEndCall))

		case *render3.Text:
			txtIdx := s.declsCount
			s.declsCount++

			txtCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵtext"),
					ast.NodeFlagsNone,
				),
				nil,
				nil,
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
					factory.NewNumericLiteral(fmt.Sprintf("%d", txtIdx), 0),
					factory.NewStringLiteral(n.Value, 0),
				})),
				ast.NodeFlagsNone,
			)
			s.createStatements = append(s.createStatements, factory.NewExpressionStatement(txtCall))

		case *render3.BoundText:
			txtIdx := s.declsCount
			s.declsCount++

			var interpolation *expression_parser.Interpolation
			if aws, ok := n.Value.(*expression_parser.ASTWithSource); ok {
				if inter, ok := aws.Ast.(*expression_parser.Interpolation); ok {
					interpolation = inter
				}
			}

			if interpolation == nil {
				// Fallback to static text if no interpolation is found
				rawText := n.SourceSpan.ToString()
				txtCall := factory.NewCallExpression(
					factory.NewPropertyAccessExpression(
						factory.NewIdentifier("i0"),
						nil,
						factory.NewIdentifier("ɵɵtext"),
						ast.NodeFlagsNone,
					),
					nil, nil,
					(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
						factory.NewNumericLiteral(fmt.Sprintf("%d", txtIdx), 0),
						factory.NewStringLiteral(rawText, 0),
					})),
					ast.NodeFlagsNone,
				)
				s.createStatements = append(s.createStatements, factory.NewExpressionStatement(txtCall))
			} else {
				txtCall := factory.NewCallExpression(
					factory.NewPropertyAccessExpression(
						factory.NewIdentifier("i0"),
						nil,
						factory.NewIdentifier("ɵɵtext"),
						ast.NodeFlagsNone,
					),
					nil, nil,
					(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
						factory.NewNumericLiteral(fmt.Sprintf("%d", txtIdx), 0),
					})),
					ast.NodeFlagsNone,
				)
				s.createStatements = append(s.createStatements, factory.NewExpressionStatement(txtCall))

				expressions := interpolation.Expressions
				s.varsCount += len(expressions)
				diff := txtIdx - s.currentUpdateIdx
				if diff > 0 {
					var advanceArgs []*ast.Node
					if diff == 1 {
						advanceArgs = []*ast.Node{}
					} else {
						advanceArgs = []*ast.Node{factory.NewNumericLiteral(fmt.Sprintf("%d", diff), 0)}
					}
					advanceCall := factory.NewCallExpression(
						factory.NewPropertyAccessExpression(
							factory.NewIdentifier("i0"),
							nil,
							factory.NewIdentifier("ɵɵadvance"),
							ast.NodeFlagsNone,
						),
						nil, nil,
						(*ast.ElementList)(factory.NewNodeList(advanceArgs)),
						ast.NodeFlagsNone,
					)
					s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(advanceCall))
					s.currentUpdateIdx = txtIdx
				}

				var args []*ast.Node
				var funcName string

				if len(expressions) == 1 {
					str0 := interpolation.Strings[0].(string)
					str1 := interpolation.Strings[1].(string)
					valExpr := translateExprAST(expressions[0], factory, s.replacements, "ctx", nil)

					if str0 == "" && str1 == "" {
						funcName = "ɵɵtextInterpolate"
						args = append(args, valExpr)
					} else {
						funcName = "ɵɵtextInterpolate1"
						args = append(args, factory.NewStringLiteral(str0, 0))
						args = append(args, valExpr)
						args = append(args, factory.NewStringLiteral(str1, 0))
					}
				} else {
					funcName = fmt.Sprintf("ɵɵtextInterpolate%d", len(expressions))
					for i := 0; i < len(expressions); i++ {
						strVal := interpolation.Strings[i].(string)
						args = append(args, factory.NewStringLiteral(strVal, 0))

						valExpr := translateExprAST(expressions[i], factory, s.replacements, "ctx", nil)
						args = append(args, valExpr)
					}
					lastStrVal := interpolation.Strings[len(expressions)].(string)
					args = append(args, factory.NewStringLiteral(lastStrVal, 0))
				}

				interpolateCall := factory.NewCallExpression(
					factory.NewPropertyAccessExpression(
						factory.NewIdentifier("i0"),
						nil,
						factory.NewIdentifier(funcName),
						ast.NodeFlagsNone,
					),
					nil, nil,
					(*ast.ElementList)(factory.NewNodeList(args)),
					ast.NodeFlagsNone,
				)
				s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(interpolateCall))
			}

		case *render3.IfBlock:
			s.varsCount++
			startBranchIdx := s.declsCount
			var createCalls []*ast.Node
			for _, branch := range n.Branches {
				branchIdx := s.declsCount
				s.declsCount++

				var branchName string
				branchName = fmt.Sprintf("%s_Conditional_%d_Template", s.className, branchIdx)

				branchTag := "div"
				if len(branch.Children) > 0 {
					if el, ok := branch.Children[0].(*render3.Element); ok {
						branchTag = el.Name
					}
				}

				_, branchDecls, branchVars := s.compileSubTemplateWithReplacements(branchName, branch.Children, nil, 0, factory)

				argsList := []*ast.Node{
					factory.NewNumericLiteral(fmt.Sprintf("%d", branchIdx), 0),
					factory.NewIdentifier(branchName),
					factory.NewNumericLiteral(fmt.Sprintf("%d", branchDecls), 0),
					factory.NewNumericLiteral(fmt.Sprintf("%d", branchVars), 0),
					factory.NewStringLiteral(branchTag, 0),
				}
				if len(branch.Children) > 0 {
					if el, ok := branch.Children[0].(*render3.Element); ok {
						constIdx := s.getOrAddConst(el, factory)
						if constIdx != "" {
							argsList = append(argsList, factory.NewNumericLiteral(constIdx, 0))
						}
					}
				}
				createCall := factory.NewCallExpression(
					factory.NewIdentifier("i0"),
					nil,
					nil,
					(*ast.ElementList)(factory.NewNodeList(argsList)),
					ast.NodeFlagsNone,
				)
				createCalls = append(createCalls, createCall)
			}

			// Chain conditionalCreate calls
			var prevCall *ast.Node
			for i, call := range createCalls {
				if i == 0 {
					prevCall = call
					prevCall.AsCallExpression().Expression = factory.NewPropertyAccessExpression(
						factory.NewIdentifier("i0"),
						nil,
						factory.NewIdentifier("ɵɵconditionalCreate"),
						ast.NodeFlagsNone,
					)
				} else {
					prevCall = factory.NewCallExpression(
						prevCall,
						nil,
						nil,
						call.AsCallExpression().Arguments,
						ast.NodeFlagsNone,
					)
				}
			}
			s.createStatements = append(s.createStatements, factory.NewExpressionStatement(prevCall))

			// Update phase: dynamic conditional expression
			var condExpr *ast.Node
			for i := len(n.Branches) - 1; i >= 0; i-- {
				branch := n.Branches[i]
				branchIdx := startBranchIdx + i
				if branch.Expression == nil {
					condExpr = factory.NewNumericLiteral(fmt.Sprintf("%d", branchIdx), 0)
				} else {
					exprNode := translateExprAST(branch.Expression, factory, s.replacements, "ctx", nil)

					if condExpr == nil {
						condExpr = factory.NewNumericLiteral(fmt.Sprintf("%d", branchIdx), 0)
					} else {
						condExpr = factory.NewConditionalExpression(
							exprNode,
							factory.NewToken(ast.KindQuestionToken),
							factory.NewNumericLiteral(fmt.Sprintf("%d", branchIdx), 0),
							factory.NewToken(ast.KindColonToken),
							condExpr,
						)
					}
				}
			}

			diff := startBranchIdx - s.currentUpdateIdx
			s.currentUpdateIdx = startBranchIdx

			advanceCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵadvance"),
					ast.NodeFlagsNone,
				),
				nil,
				nil,
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
					factory.NewNumericLiteral(fmt.Sprintf("%d", diff), 0),
				})),
				ast.NodeFlagsNone,
			)
			s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(advanceCall))

			conditionalCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵconditional"),
					ast.NodeFlagsNone,
				),
				nil,
				nil,
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{condExpr})),
				ast.NodeFlagsNone,
			)
			s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(conditionalCall))

		case *render3.ForLoopBlock:
			s.varsCount++
			repeaterIdx := s.declsCount
			s.declsCount++

			itemTemplateIdx := s.declsCount
			s.declsCount++

			var emptyTemplateIdx int
			var emptyName string
			var emptyDecls, emptyVars int
			emptyTag := "div"
			if n.Empty != nil {
				emptyTemplateIdx = s.declsCount
				s.declsCount++
				emptyName = fmt.Sprintf("%s_ForEmpty_%d_Template", s.className, emptyTemplateIdx)
				if len(n.Empty.Children) > 0 {
					if el, ok := n.Empty.Children[0].(*render3.Element); ok {
						emptyTag = el.Name
					}
				}
				_, emptyDecls, emptyVars = s.compileSubTemplateWithReplacements(emptyName, n.Empty.Children, nil, 0, factory)
			}

			tag := "div"
			if len(n.Children) > 0 {
				if el, ok := n.Children[0].(*render3.Element); ok {
					tag = el.Name
				}
			}

			var viewSlot int
			if hasListener(n.Children) {
				viewSlot = s.allocateDataSlot()
			}
			
			itemName := fmt.Sprintf("%s_For_%d_Template", s.className, itemTemplateIdx)
			itemVarName := fmt.Sprintf("%s_r%d", n.Item.Name, s.allocateDataSlot())
			extra := map[string]string{
				n.Item.Name:     itemVarName,
				"$implicit_for": itemVarName,
				"$implicit_for_orig_name": n.Item.Name,
			}
			_, itemDecls, itemVars := s.compileSubTemplateWithReplacements(itemName, n.Children, extra, viewSlot, factory)

			trackExpr := translateExprAST(n.TrackBy, factory, map[string]string{n.Item.Name: "$item"}, "ctx", nil)

			trackArrowFn := factory.NewArrowFunction(
				nil, nil,
				(*ast.ParameterList)(factory.NewNodeList([]*ast.Node{
					factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("$index"), nil, nil, nil),
					factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("$item"), nil, nil, nil),
				})),
				nil, nil,
				factory.NewToken(ast.KindEqualsGreaterThanToken),
				trackExpr,
			)
			trackHelperName := fmt.Sprintf("_forTrack%d", repeaterIdx)
			trackHelper := factory.NewVariableStatement(
				nil,
				factory.NewVariableDeclarationList(
					(*ast.VariableDeclarationNodeList)(factory.NewNodeList([]*ast.Node{
						factory.NewVariableDeclaration(
							(*ast.BindingName)(factory.NewIdentifier(trackHelperName)),
							nil, nil,
							trackArrowFn,
						),
					})),
					ast.NodeFlagsNone,
				),
			)
			s.helpers = append(s.helpers, trackHelper)

			repeaterArgs := []*ast.Node{
				factory.NewNumericLiteral(fmt.Sprintf("%d", repeaterIdx), 0),
				factory.NewIdentifier(itemName),
				factory.NewNumericLiteral(fmt.Sprintf("%d", itemDecls), 0),
				factory.NewNumericLiteral(fmt.Sprintf("%d", itemVars), 0),
				factory.NewStringLiteral(tag, 0),
			}
			var constIdxStr string
			if len(n.Children) > 0 {
				if el, ok := n.Children[0].(*render3.Element); ok {
					constIdxStr = s.getOrAddContainerConst(el, factory)
				}
			}
			if constIdxStr != "" {
				repeaterArgs = append(repeaterArgs, factory.NewNumericLiteral(constIdxStr, 0))
			} else {
				repeaterArgs = append(repeaterArgs, factory.NewToken(ast.KindNullKeyword))
			}
			repeaterArgs = append(repeaterArgs, factory.NewIdentifier(trackHelperName))
			repeaterArgs = append(repeaterArgs, factory.NewToken(ast.KindFalseKeyword))
			if n.Empty != nil {
				repeaterArgs = append(repeaterArgs, factory.NewIdentifier(emptyName))
				repeaterArgs = append(repeaterArgs, factory.NewNumericLiteral(fmt.Sprintf("%d", emptyDecls), 0))
				repeaterArgs = append(repeaterArgs, factory.NewNumericLiteral(fmt.Sprintf("%d", emptyVars), 0))
				repeaterArgs = append(repeaterArgs, factory.NewStringLiteral(emptyTag, 0))
				
				var emptyConstIdxStr string
				if len(n.Empty.Children) > 0 {
					if el, ok := n.Empty.Children[0].(*render3.Element); ok {
						emptyConstIdxStr = s.getOrAddContainerConst(el, factory)
					}
				}
				if emptyConstIdxStr != "" {
					repeaterArgs = append(repeaterArgs, factory.NewNumericLiteral(emptyConstIdxStr, 0))
				}
			}
			repeaterCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵrepeaterCreate"),
					ast.NodeFlagsNone,
				),
				nil, nil,
				(*ast.ElementList)(factory.NewNodeList(repeaterArgs)),
				ast.NodeFlagsNone,
			)
			s.createStatements = append(s.createStatements, factory.NewExpressionStatement(repeaterCall))

			diff := repeaterIdx - s.currentUpdateIdx
			s.currentUpdateIdx = repeaterIdx

			advanceCallRepeater := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵadvance"),
					ast.NodeFlagsNone,
				),
				nil, nil,
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
					factory.NewNumericLiteral(fmt.Sprintf("%d", diff), 0),
				})),
				ast.NodeFlagsNone,
			)
			s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(advanceCallRepeater))

			sourceExpr := translateExprAST(&n.Expression, factory, s.replacements, "ctx", nil)

			updateCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵrepeater"),
					ast.NodeFlagsNone,
				),
				nil, nil,
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{sourceExpr})),
				ast.NodeFlagsNone,
			)
			s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(updateCall))

		case *render3.SwitchBlock:
			s.varsCount++
			startCaseIdx := s.declsCount
			var createCalls []*ast.Node
			for _, group := range n.Groups {
				for range group.Cases {
					caseIdx := s.declsCount
					s.declsCount++

					caseName := fmt.Sprintf("%s_Case_%d_Template", s.className, caseIdx)
					caseTag := "span"
					if len(group.Children) > 0 {
						if el, ok := group.Children[0].(*render3.Element); ok {
							caseTag = el.Name
						}
					}
					_, caseDecls, caseVars := s.compileSubTemplateWithReplacements(caseName, group.Children, nil, 0, factory)

					var caseArgs []*ast.Node
					caseArgs = append(caseArgs, factory.NewNumericLiteral(fmt.Sprintf("%d", caseIdx), 0))
					caseArgs = append(caseArgs, factory.NewIdentifier(caseName))
					caseArgs = append(caseArgs, factory.NewNumericLiteral(fmt.Sprintf("%d", caseDecls), 0))
					caseArgs = append(caseArgs, factory.NewNumericLiteral(fmt.Sprintf("%d", caseVars), 0))
					caseArgs = append(caseArgs, factory.NewStringLiteral(caseTag, 0))

					var constIdxStr string
					if len(group.Children) > 0 {
						if el, ok := group.Children[0].(*render3.Element); ok {
							constIdxStr = s.getOrAddContainerConst(el, factory)
						}
					}
					if constIdxStr != "" {
						caseArgs = append(caseArgs, factory.NewNumericLiteral(constIdxStr, 0))
					}

					createCall := factory.NewCallExpression(
						factory.NewIdentifier("i0"),
						nil, nil,
						(*ast.ElementList)(factory.NewNodeList(caseArgs)),
						ast.NodeFlagsNone,
					)
					createCalls = append(createCalls, createCall)
				}
			}

			// Chain conditionalCreate calls
			var prevCall *ast.Node
			for i, call := range createCalls {
				if i == 0 {
					prevCall = call
					prevCall.AsCallExpression().Expression = factory.NewPropertyAccessExpression(
						factory.NewIdentifier("i0"),
						nil,
						factory.NewIdentifier("ɵɵconditionalCreate"),
						ast.NodeFlagsNone,
					)
				} else {
					prevCall = factory.NewCallExpression(
						prevCall,
						nil, nil,
						call.AsCallExpression().Arguments,
						ast.NodeFlagsNone,
					)
				}
			}
			s.createStatements = append(s.createStatements, factory.NewExpressionStatement(prevCall))

			// Update phase: dynamic switch
			switchExpr := translateExprAST(n.Expression, factory, s.replacements, "ctx", nil)

			tmpVarName := fmt.Sprintf("tmp_%d_0", startCaseIdx)
			tmpVarIdent := factory.NewIdentifier(tmpVarName)

			s.updateStatements = append(s.updateStatements, factory.NewVariableStatement(
				nil,
				factory.NewVariableDeclarationList(
					(*ast.VariableDeclarationNodeList)(factory.NewNodeList([]*ast.Node{
						factory.NewVariableDeclaration(
							(*ast.BindingName)(factory.NewIdentifier(tmpVarName)),
							nil, nil, nil,
						),
					})),
					ast.NodeFlagsNone,
				),
			))

			diff := startCaseIdx - s.currentUpdateIdx
			s.currentUpdateIdx = startCaseIdx

			advanceCallSwitch := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵadvance"),
					ast.NodeFlagsNone,
				),
				nil, nil,
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
					factory.NewNumericLiteral(fmt.Sprintf("%d", diff), 0),
				})),
				ast.NodeFlagsNone,
			)
			s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(advanceCallSwitch))

			assignExpr := factory.NewBinaryExpression(
				nil,
				tmpVarIdent,
				nil,
				factory.NewToken(ast.KindEqualsToken),
				switchExpr,
			)
			parenAssignExpr := factory.NewParenthesizedExpression(assignExpr)

			var condExpr *ast.Node
			var defaultIdx int = -1
			var caseIdxs []int
			var caseExprs []expression_parser.AST

			currentIdx := startCaseIdx
			for _, group := range n.Groups {
				for _, c := range group.Cases {
					if c.Expression == nil {
						defaultIdx = currentIdx
					} else {
						caseIdxs = append(caseIdxs, currentIdx)
						caseExprs = append(caseExprs, c.Expression)
					}
					currentIdx++
				}
			}

			if defaultIdx == -1 {
				condExpr = factory.NewPrefixUnaryExpression(
					ast.KindMinusToken,
					factory.NewNumericLiteral("1", 0),
				)
			} else {
				condExpr = factory.NewNumericLiteral(fmt.Sprintf("%d", defaultIdx), 0)
			}

			for i := len(caseIdxs) - 1; i >= 0; i-- {
				cIdx := caseIdxs[i]
				cExprAST := caseExprs[i]
				cExpr := translateExprAST(cExprAST, factory, s.replacements, "ctx", nil)

				var leftSide *ast.Node
				if i == 0 {
					leftSide = factory.NewBinaryExpression(
						nil,
						parenAssignExpr,
						nil,
						factory.NewToken(ast.KindEqualsEqualsEqualsToken),
						cExpr,
					)
				} else {
					leftSide = factory.NewBinaryExpression(
						nil,
						tmpVarIdent,
						nil,
						factory.NewToken(ast.KindEqualsEqualsEqualsToken),
						cExpr,
					)
				}

				condExpr = factory.NewConditionalExpression(
					leftSide,
					factory.NewToken(ast.KindQuestionToken),
					factory.NewNumericLiteral(fmt.Sprintf("%d", cIdx), 0),
					factory.NewToken(ast.KindColonToken),
					condExpr,
				)
			}

			conditionalCall := factory.NewCallExpression(
				factory.NewPropertyAccessExpression(
					factory.NewIdentifier("i0"),
					nil,
					factory.NewIdentifier("ɵɵconditional"),
					ast.NodeFlagsNone,
				),
				nil, nil,
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{condExpr})),
				ast.NodeFlagsNone,
			)
			s.updateStatements = append(s.updateStatements, factory.NewExpressionStatement(conditionalCall))
		}
	}
}

func (s *templateState) compileSubTemplate(name string, nodes []render3.Node, factory *ast.NodeFactory) string {
	name, _, _ = s.compileSubTemplateWithReplacements(name, nodes, nil, 0, factory)
	return name
}

func (s *templateState) compileSubTemplateWithReplacements(
	name string,
	nodes []render3.Node,
	extraReplacements map[string]string,
	preAllocViewSlot int,
	factory *ast.NodeFactory,
) (string, int, int) {
	merged := make(map[string]string)
	for k, v := range s.replacements {
		merged[k] = v
	}
	for k, v := range extraReplacements {
		merged[k] = v
	}

	subState := &templateState{
		className:    s.className,
		templateName: name,
		templateText: s.templateText,
		replacements: merged,
		consts:       s.consts,
		isSubTemplate: true,
		dataIndex:    s.dataIndex,
		sharedCtx:    s.sharedCtx,
		preAllocViewSlot: preAllocViewSlot,
	}
	if len(nodes) > 0 {
		subState.processNodes(nodes, factory)
	}

	if subState.hasListener {
		getCurrentViewCall := factory.NewCallExpression(factory.NewPropertyAccessExpression(factory.NewIdentifier("i0"), nil, factory.NewIdentifier("ɵɵgetCurrentView"), ast.NodeFlagsNone), nil, nil, (*ast.ElementList)(factory.NewNodeList([]*ast.Node{})), ast.NodeFlagsNone)
		r1Decl := factory.NewVariableStatement(nil, factory.NewVariableDeclarationList((*ast.VariableDeclarationNodeList)(factory.NewNodeList([]*ast.Node{factory.NewVariableDeclaration((*ast.BindingName)(factory.NewIdentifier(subState.viewRefName)), nil, nil, getCurrentViewCall)})), ast.NodeFlagsConst))
		subState.createStatements = append([]*ast.Node{r1Decl}, subState.createStatements...)
	}

	if itemName, ok := extraReplacements["$implicit_for"]; ok {
		itemVarDecl := factory.NewVariableStatement(
			nil,
			factory.NewVariableDeclarationList(
				(*ast.VariableDeclarationNodeList)(factory.NewNodeList([]*ast.Node{
					factory.NewVariableDeclaration(
						(*ast.BindingName)(factory.NewIdentifier(itemName)),
						nil, nil,
						factory.NewPropertyAccessExpression(factory.NewIdentifier("ctx"), nil, factory.NewIdentifier("$implicit"), ast.NodeFlagsNone),
					),
				})),
				ast.NodeFlagsNone,
			),
		)
		subState.updateStatements = append([]*ast.Node{itemVarDecl}, subState.updateStatements...)
	}

	rfParam := factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("rf"), nil, nil, nil)
	ctxParam := factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("ctx"), nil, nil, nil)
	paramList := (*ast.ParameterList)(factory.NewNodeList([]*ast.Node{rfParam, ctxParam}))

	var bodyStmts []*ast.Node

	ifBlock1 := factory.NewIfStatement(
		factory.NewBinaryExpression(
			nil,
			factory.NewIdentifier("rf"),
			nil,
			factory.NewToken(ast.KindAmpersandToken),
			factory.NewNumericLiteral("1", 0),
		),
		factory.NewBlock((*ast.StatementList)(factory.NewNodeList(chainIvyCalls(subState.createStatements, factory))), true),
		nil,
	)
	bodyStmts = append(bodyStmts, ifBlock1)

	if len(subState.updateStatements) > 0 {
		ifBlock2 := factory.NewIfStatement(
			factory.NewBinaryExpression(
				nil,
				factory.NewIdentifier("rf"),
				nil,
				factory.NewToken(ast.KindAmpersandToken),
				factory.NewNumericLiteral("2", 0),
			),
			factory.NewBlock((*ast.StatementList)(factory.NewNodeList(chainIvyCalls(subState.updateStatements, factory))), true),
			nil,
		)
		bodyStmts = append(bodyStmts, ifBlock2)
	}

	fnBody := factory.NewBlock((*ast.StatementList)(factory.NewNodeList(bodyStmts)), true)
	fnExpr := factory.NewFunctionDeclaration(
		nil,
		nil,
		factory.NewIdentifier(name),
		nil,
		paramList,
		nil,
		nil,
		fnBody,
	)

	s.subTemplates = append(s.subTemplates, fnExpr)
	return name, subState.declsCount, subState.varsCount
}

func transformClass(classNode *ast.Node, host reflection.ReflectionHost, factory *ast.NodeFactory, sf *ast.SourceFile) ([]*ast.Node, []*ast.Node, bool) {
	decorators := host.GetDecoratorsOfDeclaration(classNode)
	if len(decorators) == 0 {
		return nil, nil, false
	}

	var hasAngularDecorator bool
	var ivyPropName string
	var ivyFuncName string
	var props []*ast.Node
	var state *templateState

	className := classNode.AsClassDeclaration().Name().AsIdentifier().Text
	classNameIdent := factory.NewIdentifier(className)

	for _, dec := range decorators {
		switch dec.Name {
		case "Component":
			hasAngularDecorator = true
			ivyPropName = "ɵcmp"
			ivyFuncName = "ɵɵdefineComponent"

			// type: ClassName
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("type"),
				nil,
				nil,
				classNameIdent,
			))

			// selectors: [['app-root']]
			selectorStr := factory.NewStringLiteral(getSelector(&dec), 0)
			innerArray := factory.NewArrayLiteralExpression(
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{selectorStr})),
				false,
			)
			outerArray := factory.NewArrayLiteralExpression(
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{innerArray})),
				false,
			)
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("selectors"),
				nil,
				nil,
				outerArray,
			))

			// Extract template property from component decorator arguments
			var templateText string

			if len(dec.Args) > 0 {
				objLiteral := dec.Args[0].AsObjectLiteralExpression()
				if objLiteral != nil {
					for _, prop := range objLiteral.Properties.Nodes {
						if prop.Kind == ast.KindPropertyAssignment {
							pa := prop.AsPropertyAssignment()
							propName := pa.Name().AsIdentifier().Text
							switch propName {
							case "template":
								if pa.Initializer.Kind == ast.KindStringLiteral {
									strLit := pa.Initializer.AsStringLiteral()
									if strLit != nil {
										templateText = strLit.Text
									}
								} else if pa.Initializer.Kind == ast.KindNoSubstitutionTemplateLiteral {
									templateLit := pa.Initializer.AsNoSubstitutionTemplateLiteral()
									if templateLit != nil {
										templateText = templateLit.Text
									}
								}
							}
						}
					}
				}
			}

			// Compile the HTML template using render3.ParseTemplate
			parsedTemplate := render3.ParseTemplate(templateText, "", nil)
			constsPool := make([]*ast.Node, 0)
			var dataIndex int = 0
			var sharedCtx string = ""
			state = &templateState{
				className:    className,
				templateName: className + "_Template",
				templateText: templateText,
				consts:       &constsPool,
				dataIndex:    &dataIndex,
				sharedCtx:    &sharedCtx,
			}
			state.processNodes(parsedTemplate.Nodes, factory)

			// Construct rf & ctx parameters
			rfParam := factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("rf"), nil, nil, nil)
			ctxParam := factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("ctx"), nil, nil, nil)
			paramList := (*ast.ParameterList)(factory.NewNodeList([]*ast.Node{rfParam, ctxParam}))

			var bodyStmts []*ast.Node

			// Construct the if (rf & 1) statement
			if1Block := factory.NewIfStatement(
				factory.NewBinaryExpression(
					nil,
					factory.NewIdentifier("rf"),
					nil,
					factory.NewToken(ast.KindAmpersandToken),
					factory.NewNumericLiteral("1", 0),
				),
				factory.NewBlock((*ast.StatementList)(factory.NewNodeList(chainIvyCalls(state.createStatements, factory))), true),
				nil,
			)
			bodyStmts = append(bodyStmts, if1Block)

			// Construct the if (rf & 2) statement
			if len(state.updateStatements) > 0 {
				if2Block := factory.NewIfStatement(
					factory.NewBinaryExpression(
						nil,
						factory.NewIdentifier("rf"),
						nil,
						factory.NewToken(ast.KindAmpersandToken),
						factory.NewNumericLiteral("2", 0),
					),
					factory.NewBlock((*ast.StatementList)(factory.NewNodeList(chainIvyCalls(state.updateStatements, factory))), true),
					nil,
				)
				bodyStmts = append(bodyStmts, if2Block)
			}

			templateBody := factory.NewBlock((*ast.StatementList)(factory.NewNodeList(bodyStmts)), true)

			// Construct the Template function expression
			templateFn := factory.NewFunctionExpression(
				nil,
				nil,
				factory.NewIdentifier(className+"_Template"),
				nil,
				paramList,
				nil,
				nil,
				templateBody,
			)

			// decls: declsCount
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("decls"),
				nil,
				nil,
				factory.NewNumericLiteral(fmt.Sprintf("%d", state.declsCount), 0),
			))

			// vars: varsCount
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("vars"),
				nil,
				nil,
				factory.NewNumericLiteral(fmt.Sprintf("%d", state.varsCount), 0),
			))

			if state != nil && state.consts != nil && len(*state.consts) > 0 {
				props = append(props, factory.NewPropertyAssignment(
					nil,
					factory.NewIdentifier("consts"),
					nil,
					nil,
					factory.NewArrayLiteralExpression(
						(*ast.ElementList)(factory.NewNodeList(*state.consts)),
						false,
					),
				))
			}

			// template: templateFn
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("template"),
				nil,
				nil,
				templateFn,
			))

			// encapsulation: 2
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("encapsulation"),
				nil,
				nil,
				factory.NewNumericLiteral("2", 0),
			))

		case "Directive":
			hasAngularDecorator = true
			ivyPropName = "ɵdir"
			ivyFuncName = "ɵɵdefineDirective"

			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("type"),
				nil,
				nil,
				classNameIdent,
			))

			selectorStr := factory.NewStringLiteral(getSelector(&dec), 0)
			innerArray := factory.NewArrayLiteralExpression(
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{selectorStr})),
				false,
			)
			outerArray := factory.NewArrayLiteralExpression(
				(*ast.ElementList)(factory.NewNodeList([]*ast.Node{innerArray})),
				false,
			)
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("selectors"),
				nil,
				nil,
				outerArray,
			))

		case "NgModule":
			hasAngularDecorator = true
			ivyPropName = "ɵmod"
			ivyFuncName = "ɵɵdefineNgModule"

			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("type"),
				nil,
				nil,
				classNameIdent,
			))

		case "Pipe":
			hasAngularDecorator = true
			ivyPropName = "ɵpipe"
			ivyFuncName = "ɵɵdefinePipe"

			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("name"),
				nil,
				nil,
				factory.NewStringLiteral(getPipeName(&dec), 0),
			))
			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("type"),
				nil,
				nil,
				classNameIdent,
			))

		case "Injectable":
			hasAngularDecorator = true
			ivyPropName = "ɵprov"
			ivyFuncName = "ɵɵdefineInjectable"

			props = append(props, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("token"),
				nil,
				nil,
				classNameIdent,
			))
		}
	}

	if !hasAngularDecorator {
		return nil, nil, false
	}

	// 1. Strip the decorators from the class modifiers
	if classNode.Modifiers() != nil {
		var cleanNodes []*ast.Node
		for _, mod := range classNode.Modifiers().Nodes {
			if mod.Kind != ast.KindDecorator {
				cleanNodes = append(cleanNodes, mod)
			}
		}
		if len(cleanNodes) == 0 {
			classNode.AsMutable().SetModifiers(nil)
		} else {
			classNode.Modifiers().Nodes = cleanNodes
		}
	}

	// 2. Build the compiled static property node (static ɵcmp = i0.ɵɵdefineComponent({...}))
	staticToken := factory.NewToken(ast.KindStaticKeyword)
	modifiersList := factory.NewModifierList([]*ast.Node{staticToken})

	// i0.ɵɵdefineComponent
	i0IdentNode := factory.NewIdentifier("i0")
	defineCompIdentNode := factory.NewIdentifier(ivyFuncName)
	propertyAccess := factory.NewPropertyAccessExpression(
		i0IdentNode,
		nil,
		defineCompIdentNode,
		ast.NodeFlagsNone,
	)

	// ObjectLiteralExpression: { type: ClassName, selectors: ... }
	objLiteral := factory.NewObjectLiteralExpression(
		factory.NewNodeList(props),
		false,
	)

	// CallExpression: i0.ɵɵdefineComponent({ type: ClassName, selectors: ... })
	callExpr := factory.NewCallExpression(
		propertyAccess,
		nil,
		nil,
		(*ast.ElementList)(factory.NewNodeList([]*ast.Node{objLiteral})),
		ast.NodeFlagsNone,
	)

	propNameNode := factory.NewIdentifier(ivyPropName)

	propertyNode := factory.NewPropertyDeclaration(
		modifiersList,
		propNameNode,
		nil,
		nil,
		callExpr,
	)

	fixupParentReferences(propertyNode, classNode)

	// 3. Construct and prepend facProp to class members, then append propertyNode
	paramNode := factory.NewParameterDeclaration(nil, nil, factory.NewIdentifier("__ngFactoryType__"), nil, nil, nil)
	left := factory.NewIdentifier("__ngFactoryType__")
	right := classNameIdent
	orExpr := factory.NewBinaryExpression(nil, left, nil, factory.NewToken(ast.KindBarBarToken), right)
	parenOrExpr := factory.NewParenthesizedExpression(orExpr)
	newExpr := factory.NewNewExpression(parenOrExpr, nil, (*ast.ElementList)(factory.NewNodeList(nil)))
	returnStmt := factory.NewReturnStatement(newExpr)
	fnBody := factory.NewBlock((*ast.StatementList)(factory.NewNodeList([]*ast.Node{returnStmt})), true)
	factoryFn := factory.NewFunctionExpression(nil, nil, factory.NewIdentifier(className+"_Factory"), nil, (*ast.ParameterList)(factory.NewNodeList([]*ast.Node{paramNode})), nil, nil, fnBody)
	facStaticToken := factory.NewToken(ast.KindStaticKeyword)
	facModifiersList := factory.NewModifierList([]*ast.Node{facStaticToken})
	facProp := factory.NewPropertyDeclaration(facModifiersList, factory.NewIdentifier("ɵfac"), nil, nil, factoryFn)

	fixupParentReferences(facProp, classNode)

	classDecl := classNode.AsClassDeclaration()
	classDecl.Members.Nodes = append(classDecl.Members.Nodes, facProp, propertyNode)

	// consts assignment removed from here

	var metadataNodes []*ast.Node
	metadataNode := buildClassMetadataIIFE(className, decorators, factory)
	metadataNodes = append(metadataNodes, metadataNode)

	filePath := getBaseName(sf.FileName())
	lineNumber := getLineNumber(sf.Text(), classNode.AsClassDeclaration().Name().Pos())
	debugInfoNode := buildClassDebugInfoIIFE(className, filePath, lineNumber, factory)
	metadataNodes = append(metadataNodes, debugInfoNode)

	var beforeNodes []*ast.Node
	beforeNodes = append(beforeNodes, state.helpers...)
	beforeNodes = append(beforeNodes, state.subTemplates...)

	return beforeNodes, metadataNodes, true
}

func buildClassMetadataIIFE(className string, decorators []reflection.Decorator, factory *ast.NodeFactory) *ast.Node {
	classNameIdent := factory.NewIdentifier(className)

	var decObjects []*ast.Node
	for _, dec := range decorators {
		var decProps []*ast.Node

		// type: DecoratorName
		decProps = append(decProps, factory.NewPropertyAssignment(
			nil,
			factory.NewIdentifier("type"),
			nil,
			nil,
			factory.NewIdentifier(dec.Name),
		))

		// args: [...]
		if len(dec.Args) > 0 {
			var argNodes []*ast.Node
			for _, arg := range dec.Args {
				argNodes = append(argNodes, arg)
			}
			argsArray := factory.NewArrayLiteralExpression(
				(*ast.ElementList)(factory.NewNodeList(argNodes)),
				false,
			)
			decProps = append(decProps, factory.NewPropertyAssignment(
				nil,
				factory.NewIdentifier("args"),
				nil,
				nil,
				argsArray,
			))
		}

		decObj := factory.NewObjectLiteralExpression(
			factory.NewNodeList(decProps),
			false,
		)
		decObjects = append(decObjects, decObj)
	}

	decoratorsArray := factory.NewArrayLiteralExpression(
		(*ast.ElementList)(factory.NewNodeList(decObjects)),
		false,
	)

	// Call: i0.ɵsetClassMetadata(AppComponent, [{ type: Component, args: [...] }], null, null)
	setClassMetadataCall := factory.NewCallExpression(
		factory.NewPropertyAccessExpression(
			factory.NewIdentifier("i0"),
			nil,
			factory.NewIdentifier("ɵsetClassMetadata"),
			ast.NodeFlagsNone,
		),
		nil,
		nil,
		(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
			classNameIdent,
			decoratorsArray,
			factory.NewToken(ast.KindNullKeyword),
			factory.NewToken(ast.KindNullKeyword),
		})),
		ast.NodeFlagsNone,
	)

	// Guard: (typeof ngDevMode === "undefined" || ngDevMode) && i0.ɵsetClassMetadata(...)
	leftSide := factory.NewBinaryExpression(
		nil,
		factory.NewTypeOfExpression(factory.NewIdentifier("ngDevMode")),
		nil,
		factory.NewToken(ast.KindEqualsEqualsEqualsToken),
		factory.NewStringLiteral("undefined", 0),
	)
	rightSide := factory.NewIdentifier("ngDevMode")
	orExpr := factory.NewBinaryExpression(
		nil,
		leftSide,
		nil,
		factory.NewToken(ast.KindBarBarToken),
		rightSide,
	)
	parenExpr := factory.NewParenthesizedExpression(orExpr)

	guardedCall := factory.NewBinaryExpression(
		nil,
		parenExpr,
		nil,
		factory.NewToken(ast.KindAmpersandAmpersandToken),
		setClassMetadataCall,
	)

	// Wrap in IIFE with Arrow Function: (() => { guardedCall; })()
	fnBody := factory.NewBlock(
		(*ast.StatementList)(factory.NewNodeList([]*ast.Node{
			factory.NewExpressionStatement(guardedCall),
		})),
		true,
	)
	arrowFn := factory.NewArrowFunction(
		nil,
		nil,
		(*ast.ParameterList)(factory.NewNodeList(nil)),
		nil,
		nil,
		factory.NewToken(ast.KindEqualsGreaterThanToken),
		fnBody,
	)
	parenFnExpr := factory.NewParenthesizedExpression(arrowFn)
	iifeCall := factory.NewCallExpression(
		parenFnExpr,
		nil,
		nil,
		(*ast.ElementList)(factory.NewNodeList(nil)),
		ast.NodeFlagsNone,
	)
	iifeStmt := factory.NewExpressionStatement(iifeCall)

	return iifeStmt
}

func buildClassDebugInfoIIFE(className string, filePath string, lineNumber int, factory *ast.NodeFactory) *ast.Node {
	props := []*ast.Node{
		factory.NewPropertyAssignment(nil, factory.NewIdentifier("className"), nil, nil, factory.NewStringLiteral(className, 0)),
		factory.NewPropertyAssignment(nil, factory.NewIdentifier("filePath"), nil, nil, factory.NewStringLiteral(filePath, 0)),
		factory.NewPropertyAssignment(nil, factory.NewIdentifier("lineNumber"), nil, nil, factory.NewNumericLiteral(fmt.Sprintf("%d", lineNumber), 0)),
	}
	objLiteral := factory.NewObjectLiteralExpression(factory.NewNodeList(props), false)

	setClassDebugInfoCall := factory.NewCallExpression(
		factory.NewPropertyAccessExpression(
			factory.NewIdentifier("i0"),
			nil,
			factory.NewIdentifier("ɵsetClassDebugInfo"),
			ast.NodeFlagsNone,
		),
		nil,
		nil,
		(*ast.ElementList)(factory.NewNodeList([]*ast.Node{
			factory.NewIdentifier(className),
			objLiteral,
		})),
		ast.NodeFlagsNone,
	)

	leftSide := factory.NewBinaryExpression(
		nil,
		factory.NewTypeOfExpression(factory.NewIdentifier("ngDevMode")),
		nil,
		factory.NewToken(ast.KindEqualsEqualsEqualsToken),
		factory.NewStringLiteral("undefined", 0),
	)
	rightSide := factory.NewIdentifier("ngDevMode")
	orExpr := factory.NewBinaryExpression(
		nil,
		leftSide,
		nil,
		factory.NewToken(ast.KindBarBarToken),
		rightSide,
	)
	parenExpr := factory.NewParenthesizedExpression(orExpr)

	guardedCall := factory.NewBinaryExpression(
		nil,
		parenExpr,
		nil,
		factory.NewToken(ast.KindAmpersandAmpersandToken),
		setClassDebugInfoCall,
	)

	fnBody := factory.NewBlock(
		(*ast.StatementList)(factory.NewNodeList([]*ast.Node{
			factory.NewExpressionStatement(guardedCall),
		})),
		true,
	)
	arrowFn := factory.NewArrowFunction(
		nil,
		nil,
		(*ast.ParameterList)(factory.NewNodeList(nil)),
		nil,
		nil,
		factory.NewToken(ast.KindEqualsGreaterThanToken),
		fnBody,
	)
	parenFnExpr := factory.NewParenthesizedExpression(arrowFn)
	iifeCall := factory.NewCallExpression(
		parenFnExpr,
		nil,
		nil,
		(*ast.ElementList)(factory.NewNodeList(nil)),
		ast.NodeFlagsNone,
	)
	iifeStmt := factory.NewExpressionStatement(iifeCall)

	return iifeStmt
}

func getLineNumber(text string, pos int) int {
	line := 1
	for i := 0; i < pos && i < len(text); i++ {
		if text[i] == '\n' {
			line++
		}
	}
	return line
}

func getBaseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

func getSelector(dec *reflection.Decorator) string {
	if len(dec.Args) > 0 {
		// Attempt to extract selector property from ObjectLiteral
		objLiteral := dec.Args[0].AsObjectLiteralExpression()
		if objLiteral != nil {
			for _, prop := range objLiteral.Properties.Nodes {
				if prop.Kind == ast.KindPropertyAssignment {
					pa := prop.AsPropertyAssignment()
					if pa.Name().AsIdentifier().Text == "selector" {
						strLit := pa.Initializer.AsStringLiteral()
						if strLit != nil {
							return strLit.Text
						}
					}
				}
			}
		}
	}
	return "app-selector"
}

func getPipeName(dec *reflection.Decorator) string {
	if len(dec.Args) > 0 {
		objLiteral := dec.Args[0].AsObjectLiteralExpression()
		if objLiteral != nil {
			for _, prop := range objLiteral.Properties.Nodes {
				if prop.Kind == ast.KindPropertyAssignment {
					pa := prop.AsPropertyAssignment()
					if pa.Name().AsIdentifier().Text == "name" {
						strLit := pa.Initializer.AsStringLiteral()
						if strLit != nil {
							return strLit.Text
						}
					}
				}
			}
		}
	}
	return "customPipe"
}

func translateExpression(exprStr string, factory *ast.NodeFactory, replacements map[string]string, ctxName string) *ast.Node {
	exprStr = strings.TrimSpace(exprStr)
	if exprStr == "" {
		return factory.NewIdentifier(ctxName)
	}
	opts := ast.SourceFileParseOptions{FileName: "/expr.ts"}
	sf := parser.ParseSourceFile(opts, "const _ = " + exprStr, core.ScriptKindTS)
	if sf == nil || len(sf.Statements.Nodes) == 0 {
		return factory.NewIdentifier(ctxName)
	}
	varStmt := sf.Statements.Nodes[0].AsVariableStatement()
	if varStmt == nil || varStmt.DeclarationList == nil {
		return factory.NewIdentifier(ctxName)
	}
	declList := varStmt.DeclarationList.AsVariableDeclarationList()
	if declList == nil || declList.Declarations == nil || len(declList.Declarations.Nodes) == 0 {
		return factory.NewIdentifier(ctxName)
	}
	varInit := declList.Declarations.Nodes[0].AsVariableDeclaration().Initializer
	return rewriteExpressionWithCtx(varInit, factory, replacements, ctxName)
}

func rewriteExpressionWithCtx(node *ast.Node, factory *ast.NodeFactory, replacements map[string]string, ctxName string) *ast.Node {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case ast.KindIdentifier:
		name := node.AsIdentifier().Text
		if repl, ok := replacements[name]; ok {
			return factory.NewIdentifier(repl)
		}
		if name == "$index" || name == "$item" || strings.HasPrefix(name, "tmp_") || strings.HasPrefix(name, "item_r") {
			return factory.NewIdentifier(name)
		}
		if name == "true" || name == "false" || name == "null" || name == "undefined" || name == "$event" {
			return factory.NewIdentifier(name)
		}
		return factory.NewPropertyAccessExpression(
			factory.NewIdentifier(ctxName),
			nil,
			factory.NewIdentifier(name),
			ast.NodeFlagsNone,
		)
	case ast.KindCallExpression:
		call := node.AsCallExpression()
		expr := rewriteExpressionWithCtx(call.Expression, factory, replacements, ctxName)
		var args []*ast.Node
		if call.Arguments != nil {
			for _, arg := range call.Arguments.Nodes {
				args = append(args, rewriteExpressionWithCtx(arg, factory, replacements, ctxName))
			}
		}
		return factory.NewCallExpression(
			expr,
			nil,
			nil,
			(*ast.ElementList)(factory.NewNodeList(args)),
			ast.NodeFlagsNone,
		)
	case ast.KindPrefixUnaryExpression:
		pe := node.AsPrefixUnaryExpression()
		operand := rewriteExpressionWithCtx(pe.Operand, factory, replacements, ctxName)
		return factory.NewPrefixUnaryExpression(pe.Operator, operand)
	case ast.KindPropertyAccessExpression:
		pa := node.AsPropertyAccessExpression()
		expr := rewriteExpressionWithCtx(pa.Expression, factory, replacements, ctxName)
		return factory.NewPropertyAccessExpression(
			expr,
			nil,
			(*ast.MemberName)((*ast.Node)(factory.NewIdentifier(pa.Name().AsIdentifier().Text))),
			ast.NodeFlagsNone,
		)
	case ast.KindBinaryExpression:
		bin := node.AsBinaryExpression()
		left := rewriteExpressionWithCtx(bin.Left, factory, replacements, ctxName)
		right := rewriteExpressionWithCtx(bin.Right, factory, replacements, ctxName)
		return factory.NewBinaryExpression(
			nil,
			left,
			nil,
			bin.OperatorToken,
			right,
		)
	case ast.KindConditionalExpression:
		cond := node.AsConditionalExpression()
		condition := rewriteExpressionWithCtx(cond.Condition, factory, replacements, ctxName)
		whenTrue := rewriteExpressionWithCtx(cond.WhenTrue, factory, replacements, ctxName)
		whenFalse := rewriteExpressionWithCtx(cond.WhenFalse, factory, replacements, ctxName)
		return factory.NewConditionalExpression(
			condition,
			factory.NewToken(ast.KindQuestionToken),
			whenTrue,
			factory.NewToken(ast.KindColonToken),
			whenFalse,
		)
	case ast.KindParenthesizedExpression:
		pe := node.AsParenthesizedExpression()
		return factory.NewParenthesizedExpression(rewriteExpressionWithCtx(pe.Expression, factory, replacements, ctxName))
	default:
		// Clone literal values directly
		if node.Kind == ast.KindStringLiteral {
			return factory.NewStringLiteral(node.AsStringLiteral().Text, 0)
		}
		if node.Kind == ast.KindNumericLiteral {
			return factory.NewNumericLiteral(node.AsNumericLiteral().Text, 0)
		}
		return node
	}
}

func getExpressionString(span parse_util.ParseSourceSpan, sourceText string) string {
	start := span.Start.Offset
	end := span.End.Offset
	if start < 0 || end > len(sourceText) || start >= end {
		return ""
	}
	s := sourceText[start:end]
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		s = s[2 : len(s)-2]
	}
	return strings.TrimSpace(s)
}

func getASTSource(ast expression_parser.AST) string {
	if ast == nil {
		return ""
	}
	if aws, ok := ast.(*expression_parser.ASTWithSource); ok {
		return aws.Source
	}
	return ""
}




	
	
	

func translateExprAST(node expression_parser.AST, factory *ast.NodeFactory, replacements map[string]string, ctxName string, usedLocals map[string]string) *ast.Node {
	if node == nil {
		return factory.NewIdentifier(ctxName)
	}
	switch n := node.(type) {
	case *expression_parser.ASTWithSource:
		return translateExprAST(n.Ast, factory, replacements, ctxName, usedLocals)
	case *expression_parser.ImplicitReceiver:
		return factory.NewIdentifier(ctxName)
	case *expression_parser.Chain:
		if len(n.Expressions) > 0 {
			return translateExprAST(n.Expressions[0], factory, replacements, ctxName, usedLocals)
		}
		return factory.NewIdentifier(ctxName)
	case *expression_parser.ThisReceiver:
		return factory.NewIdentifier("this")
	case *expression_parser.PropertyRead:
		recv := translateExprAST(n.Receiver, factory, replacements, ctxName, usedLocals)
		name := n.Name
		
		// If Receiver is ImplicitReceiver, we might be reading a local variable like `proj` or `$implicit`
		_, isImplicit := n.Receiver.(*expression_parser.ImplicitReceiver)
		if isImplicit {
			if repl, ok := replacements[name]; ok {
				if usedLocals != nil {
					usedLocals[name] = repl
				}
				return factory.NewIdentifier(repl)
			}
			if name == "$event" || name == "$index" || name == "$item" {
				if usedLocals != nil {
					usedLocals[name] = name
				}
				return factory.NewIdentifier(name)
			}
			// Fallback to ctx.name
			return factory.NewPropertyAccessExpression(
				factory.NewIdentifier(ctxName),
				nil,
				factory.NewIdentifier(name),
				ast.NodeFlagsNone,
			)
		}
		
		return factory.NewPropertyAccessExpression(
			recv,
			nil,
			factory.NewIdentifier(name),
			ast.NodeFlagsNone,
		)
	case *expression_parser.SafePropertyRead:
		recv := translateExprAST(n.Receiver, factory, replacements, ctxName, usedLocals)
		return factory.NewPropertyAccessExpression(
			recv,
			factory.NewToken(ast.KindQuestionDotToken),
			factory.NewIdentifier(n.Name),
			ast.NodeFlagsNone,
		)
	case *expression_parser.KeyedRead:
		recv := translateExprAST(n.Receiver, factory, replacements, ctxName, usedLocals)
		key := translateExprAST(n.Key, factory, replacements, ctxName, usedLocals)
		return factory.NewElementAccessExpression(
			recv,
			nil,
			key,
			ast.NodeFlagsNone,
		)
	case *expression_parser.MethodCall:
		recv := translateExprAST(n.Receiver, factory, replacements, ctxName, usedLocals)
		var args []*ast.Node
		for _, arg := range n.Args {
			args = append(args, translateExprAST(arg, factory, replacements, ctxName, usedLocals))
		}
		
		var expr *ast.Node
		_, isImplicit := n.Receiver.(*expression_parser.ImplicitReceiver)
		if isImplicit {
			expr = factory.NewPropertyAccessExpression(
				factory.NewIdentifier(ctxName),
				nil,
				factory.NewIdentifier(n.Name),
				ast.NodeFlagsNone,
			)
		} else {
			expr = factory.NewPropertyAccessExpression(
				recv,
				nil,
				factory.NewIdentifier(n.Name),
				ast.NodeFlagsNone,
			)
		}
		
		return factory.NewCallExpression(
			expr,
			nil,
			nil,
			(*ast.ElementList)(factory.NewNodeList(args)),
			ast.NodeFlagsNone,
		)
	case *expression_parser.Call:
		recv := translateExprAST(n.Receiver, factory, replacements, ctxName, usedLocals)
		var args []*ast.Node
		for _, arg := range n.Args {
			args = append(args, translateExprAST(arg, factory, replacements, ctxName, usedLocals))
		}
		return factory.NewCallExpression(
			recv,
			nil,
			nil,
			(*ast.ElementList)(factory.NewNodeList(args)),
			ast.NodeFlagsNone,
		)
	case *expression_parser.SafeCall:
		recv := translateExprAST(n.Receiver, factory, replacements, ctxName, usedLocals)
		var args []*ast.Node
		for _, arg := range n.Args {
			args = append(args, translateExprAST(arg, factory, replacements, ctxName, usedLocals))
		}
		return factory.NewCallExpression(
			recv,
			factory.NewToken(ast.KindQuestionDotToken),
			nil,
			(*ast.ElementList)(factory.NewNodeList(args)),
			ast.NodeFlagsNone,
		)
	case *expression_parser.SafeMethodCall:
		recv := translateExprAST(n.Receiver, factory, replacements, ctxName, usedLocals)
		var args []*ast.Node
		for _, arg := range n.Args {
			args = append(args, translateExprAST(arg, factory, replacements, ctxName, usedLocals))
		}
		
		var expr *ast.Node
		_, isImplicit := n.Receiver.(*expression_parser.ImplicitReceiver)
		if isImplicit {
			expr = factory.NewPropertyAccessExpression(
				factory.NewIdentifier(ctxName),
				nil,
				factory.NewIdentifier(n.Name),
				ast.NodeFlagsNone,
			)
		} else {
			expr = factory.NewPropertyAccessExpression(
				recv,
				factory.NewToken(ast.KindQuestionDotToken),
				factory.NewIdentifier(n.Name),
				ast.NodeFlagsNone,
			)
		}
		
		return factory.NewCallExpression(
			expr,
			nil,
			nil,
			(*ast.ElementList)(factory.NewNodeList(args)),
			ast.NodeFlagsNone,
		)
	case *expression_parser.Binary:
		left := translateExprAST(n.Left, factory, replacements, ctxName, usedLocals)
		right := translateExprAST(n.Right, factory, replacements, ctxName, usedLocals)
		
		var opToken ast.Kind
		switch n.Operation {
		case "+": opToken = ast.KindPlusToken
		case "-": opToken = ast.KindMinusToken
		case "*": opToken = ast.KindAsteriskToken
		case "/": opToken = ast.KindSlashToken
		case "==": opToken = ast.KindEqualsEqualsToken
		case "===": opToken = ast.KindEqualsEqualsEqualsToken
		case "!=": opToken = ast.KindExclamationEqualsToken
		case "!==": opToken = ast.KindExclamationEqualsEqualsToken
		case "<": opToken = ast.KindLessThanToken
		case "<=": opToken = ast.KindLessThanEqualsToken
		case ">": opToken = ast.KindGreaterThanToken
		case ">=": opToken = ast.KindGreaterThanEqualsToken
		case "&&": opToken = ast.KindAmpersandAmpersandToken
		case "||": opToken = ast.KindBarBarToken
		case "??": opToken = ast.KindQuestionQuestionToken
		default: opToken = ast.KindUnknown
		}
		
		return factory.NewBinaryExpression(nil, left, nil, factory.NewToken(opToken), right)
	case *expression_parser.Conditional:
		cond := translateExprAST(n.Condition, factory, replacements, ctxName, usedLocals)
		whenTrue := translateExprAST(n.TrueExp, factory, replacements, ctxName, usedLocals)
		whenFalse := translateExprAST(n.FalseExp, factory, replacements, ctxName, usedLocals)
		return factory.NewConditionalExpression(
			cond,
			factory.NewToken(ast.KindQuestionToken),
			whenTrue,
			factory.NewToken(ast.KindColonToken),
			whenFalse,
		)
	case *expression_parser.PrefixNot:
		expr := translateExprAST(n.Expression, factory, replacements, ctxName, usedLocals)
		return factory.NewPrefixUnaryExpression(ast.KindExclamationToken, expr)
	case *expression_parser.LiteralPrimitive:
		switch v := n.Value.(type) {
		case string:
			return factory.NewStringLiteral(v, 0)
		case float64, float32, int, int32, int64:
			return factory.NewNumericLiteral(fmt.Sprintf("%v", v), 0)
		case bool:
			if v {
				return factory.NewToken(ast.KindTrueKeyword)
			}
			return factory.NewToken(ast.KindFalseKeyword)
		default:
			if v == nil {
				return factory.NewToken(ast.KindNullKeyword)
			}
			return factory.NewIdentifier("undefined")
		}
	case *expression_parser.LiteralArray:
		var items []*ast.Node
		for _, item := range n.Expressions {
			items = append(items, translateExprAST(item, factory, replacements, ctxName, usedLocals))
		}
		return factory.NewArrayLiteralExpression((*ast.ElementList)(factory.NewNodeList(items)), false)
	case *expression_parser.LiteralMap:
		// Not heavily used in simple templates, stubbed as property access for now or object literal
		return factory.NewIdentifier("LiteralMapNotSupported")
	case *expression_parser.EmptyExpr:
		return factory.NewIdentifier(ctxName)
	default:
		return factory.NewIdentifier(ctxName)
	}
}

func getIvyInstructionName(expr *ast.Node) string {
	if expr == nil {
		return ""
	}
	if expr.Kind == ast.KindCallExpression {
		call := expr.AsCallExpression()
		return getIvyInstructionName(call.Expression)
	}
	if expr.Kind == ast.KindPropertyAccessExpression {
		propAccess := expr.AsPropertyAccessExpression()
		if propAccess.Expression.Kind == ast.KindIdentifier && propAccess.Expression.AsIdentifier().Text == "i0" {
			if propAccess.Name().Kind == ast.KindIdentifier {
				return propAccess.Name().AsIdentifier().Text
			}
		}
	}
	return ""
}

func isChainableIvyInstruction(name string) bool {
	switch name {
	case "ɵɵdomElementStart", "ɵɵdomElementEnd", "ɵɵelementStart", "ɵɵelementEnd", "ɵɵelement", "ɵɵtext", "ɵɵlistener", "ɵɵconditionalCreate", "ɵɵrepeaterCreate", "ɵɵtextInterpolate", "ɵɵtextInterpolate1", "ɵɵtextInterpolate2", "ɵɵtextInterpolate3", "ɵɵtextInterpolate4", "ɵɵtextInterpolate5", "ɵɵtextInterpolate6", "ɵɵtextInterpolate7", "ɵɵtextInterpolate8", "ɵɵtextInterpolateV":
		return true
	}
	return false
}

func chainIvyCalls(statements []*ast.Node, factory *ast.NodeFactory) []*ast.Node {
	if len(statements) <= 1 {
		return statements
	}
	var optimized []*ast.Node

	for _, stmt := range statements {
		if stmt.Kind != ast.KindExpressionStatement {
			optimized = append(optimized, stmt)
			continue
		}
		exprStmt := stmt.AsExpressionStatement()
		if exprStmt.Expression.Kind != ast.KindCallExpression {
			optimized = append(optimized, stmt)
			continue
		}

		call := exprStmt.Expression.AsCallExpression()
		instrName := getIvyInstructionName(exprStmt.Expression)

		if instrName != "" && isChainableIvyInstruction(instrName) {
			if len(optimized) > 0 {
				prevStmt := optimized[len(optimized)-1]
				if prevStmt.Kind == ast.KindExpressionStatement {
					prevExprStmt := prevStmt.AsExpressionStatement()
					if prevExprStmt.Expression.Kind == ast.KindCallExpression {
						prevInstrName := getIvyInstructionName(prevExprStmt.Expression)
						// fmt.Printf("Comparing %s and %s\n", prevInstrName, instrName)
						if prevInstrName == instrName {
							newCall := factory.NewCallExpression(
								prevExprStmt.Expression,
								nil,
								nil,
								call.Arguments,
								ast.NodeFlagsNone,
							)
							var newExpr *ast.Node = newCall
							prevExprStmt.Expression = newExpr
							continue
						}
					}
				}
			}
		}

		optimized = append(optimized, stmt)
	}

	return optimized
}
