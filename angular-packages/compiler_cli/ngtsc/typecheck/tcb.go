package typecheck

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

// GenerateTcb converts a component template AST into a Type Check Block (TCB) TypeScript string
func GenerateTcb(
	className string,
	parsedTemplate *render3.ParsedTemplate,
	getDirectives func(node render3.Node) []string,
	pipes map[string]string,
) (string, map[int]parse_util.ParseSourceSpan) {
	var sb strings.Builder
	lineSpans := make(map[int]parse_util.ParseSourceSpan)

	currentLine := 1
	writeLine := func(text string, span parse_util.ParseSourceSpan) {
		sb.WriteString(text)
		sb.WriteString("\n")
		if span.Start != nil {
			lineSpans[currentLine] = span
		}
		currentLine++
	}

	// 1. Declare the TCB function
	writeLine(fmt.Sprintf("function _tcb_%s(this: %s) {", className, className), parse_util.ParseSourceSpan{})

	// 2. Declare used pipes
	for pipeName, pipeClass := range pipes {
		writeLine(fmt.Sprintf("  var _pipe_%s: %s = null!;", pipeName, pipeClass), parse_util.ParseSourceSpan{})
	}

	var walkNodes func(nodes []render3.Node, indent string)
	var walkNode func(node render3.Node, indent string)

	elId := 0
	dirId := 0

	walkNode = func(node render3.Node, indent string) {
		if node == nil {
			return
		}
		switch n := node.(type) {
		case *render3.Element:
			elId++
			myElId := elId
			tagType := getElementType(n.Name)

			// Declare element
			writeLine(fmt.Sprintf("%s  var _el%d: %s = null!;", indent, myElId, tagType), n.SourceSpan)

			// Declare directives matched
			dirs := getDirectives(n)
			var matchedDirs []int
			for _, dirClass := range dirs {
				dirId++
				matchedDirs = append(matchedDirs, dirId)
				writeLine(fmt.Sprintf("%s  var _dir%d: %s = null!;", indent, dirId, dirClass), n.SourceSpan)
			}

			// Typecheck local references
			for _, ref := range n.References {
				targetVar := fmt.Sprintf("_el%d", myElId)
				// If ref matches a directive's exportAs, use that directive
				if ref.Value != "" && len(matchedDirs) > 0 {
					targetVar = fmt.Sprintf("_dir%d", matchedDirs[0])
				}
				writeLine(fmt.Sprintf("%s  var %s = %s;", indent, ref.Name, targetVar), ref.SourceSpan)
			}

			// Typecheck inputs
			for _, prop := range n.Inputs {
				if prop.Type == 4 || prop.Type == 6 {
					continue
				}
				exprStr := astToString(prop.Value, "this")
				if exprStr != "" {
					directiveMatched := false
					for _, dId := range matchedDirs {
						writeLine(fmt.Sprintf("%s  _dir%d.%s = (%s);", indent, dId, prop.Name, exprStr), prop.SourceSpan)
						directiveMatched = true
					}
					if !directiveMatched {
						writeLine(fmt.Sprintf("%s  _el%d.%s = (%s);", indent, myElId, prop.Name, exprStr), prop.SourceSpan)
					}
				}
			}

			// Typecheck outputs/events
			for _, event := range n.Outputs {
				handlerStr := astToString(event.Handler, "this")
				if handlerStr != "" {
					eventType := getEventType(event.Name)
					replacedHandler := strings.ReplaceAll(handlerStr, "$event", "_event")
					writeLine(fmt.Sprintf("%s  ((_event: %s) => { %s; })(null!);", indent, eventType, replacedHandler), event.SourceSpan)
				}
			}

			walkNodes(n.Children, indent)

		case *render3.Template:
			elId++
			myElId := elId

			writeLine(fmt.Sprintf("%s  var _el%d: any = null!;", indent, myElId), n.SourceSpan)

			dirs := getDirectives(n)
			var matchedDirs []int
			for _, dirClass := range dirs {
				dirId++
				matchedDirs = append(matchedDirs, dirId)
				writeLine(fmt.Sprintf("%s  var _dir%d: %s = null!;", indent, dirId, dirClass), n.SourceSpan)
			}

			for _, prop := range n.Inputs {
				exprStr := astToString(prop.Value, "this")
				if exprStr != "" {
					for _, dId := range matchedDirs {
						writeLine(fmt.Sprintf("%s  _dir%d.%s = (%s);", indent, dId, prop.Name, exprStr), prop.SourceSpan)
					}
				}
			}

			walkNodes(n.Children, indent)

		case *render3.BoundText:
			exprStr := astToString(n.Value, "this")
			if exprStr != "" {
				writeLine(fmt.Sprintf("%s  var _text = (%s);", indent, exprStr), n.SourceSpan)
			}

		case *render3.IfBlock:
			for _, branch := range n.Branches {
				exprStr := astToString(branch.Expression, "this")
				if exprStr != "" {
					writeLine(fmt.Sprintf("%s  if (%s) {", indent, exprStr), branch.SourceSpan)
				} else {
					writeLine(fmt.Sprintf("%s  {", indent), branch.SourceSpan)
				}
				walkNodes(branch.Children, indent+"  ")
				writeLine(fmt.Sprintf("%s  }", indent), branch.SourceSpan)
			}

		case *render3.ForLoopBlock:
			exprStr := astToString(&n.Expression, "this")
			itemName := "item"
			if n.Item != nil {
				itemName = n.Item.Name
			}
			writeLine(fmt.Sprintf("%s  for (let %s of (%s)) {", indent, itemName, exprStr), n.SourceSpan)
			walkNodes(n.Children, indent+"  ")
			writeLine(fmt.Sprintf("%s  }", indent), n.SourceSpan)
			if n.Empty != nil {
				writeLine(fmt.Sprintf("%s  {", indent), n.Empty.SourceSpan)
				walkNodes(n.Empty.Children, indent+"  ")
				writeLine(fmt.Sprintf("%s  }", indent), n.Empty.SourceSpan)
			}

		case *render3.SwitchBlock:
			exprStr := astToString(n.Expression, "this")
			writeLine(fmt.Sprintf("%s  switch (%s) {", indent, exprStr), n.SourceSpan)
			for _, group := range n.Groups {
				for _, c := range group.Cases {
					if c.Expression != nil {
						exprStr = astToString(c.Expression, "this")
						writeLine(fmt.Sprintf("%s    case %s:", indent, exprStr), c.SourceSpan)
					} else {
						writeLine(fmt.Sprintf("%s    default:", indent), c.SourceSpan)
					}
				}
				walkNodes(group.Children, indent+"    ")
			}
			writeLine(fmt.Sprintf("%s  }", indent), n.SourceSpan)

		case *render3.DeferredBlock:
			writeLine(fmt.Sprintf("%s  {", indent), n.SourceSpan)
			walkNodes(n.Children, indent+"  ")
			if n.Placeholder != nil {
				walkNodes(n.Placeholder.Children, indent+"  ")
			}
			if n.Loading != nil {
				walkNodes(n.Loading.Children, indent+"  ")
			}
			if n.Error != nil {
				walkNodes(n.Error.Children, indent+"  ")
			}
			writeLine(fmt.Sprintf("%s  }", indent), n.SourceSpan)
		}
	}

	walkNodes = func(nodes []render3.Node, indent string) {
		for _, node := range nodes {
			walkNode(node, indent)
		}
	}

	walkNodes(parsedTemplate.Nodes, "  ")

	writeLine("}", parse_util.ParseSourceSpan{})

	return sb.String(), lineSpans
}

func getElementType(tag string) string {
	switch tag {
	case "div":
		return "HTMLDivElement"
	case "span":
		return "HTMLSpanElement"
	case "input":
		return "HTMLInputElement"
	case "form":
		return "HTMLFormElement"
	case "button":
		return "HTMLButtonElement"
	case "p":
		return "HTMLParagraphElement"
	case "a":
		return "HTMLAnchorElement"
	case "img":
		return "HTMLImageElement"
	case "select":
		return "HTMLSelectElement"
	case "option":
		return "HTMLOptionElement"
	default:
		return "HTMLElement"
	}
}

func getEventType(eventName string) string {
	switch eventName {
	case "click", "mousedown", "mouseup", "dblclick":
		return "MouseEvent"
	case "keydown", "keyup", "keypress":
		return "KeyboardEvent"
	case "focus", "blur":
		return "FocusEvent"
	case "input", "change", "submit":
		return "Event"
	default:
		return "any"
	}
}

func astToString(node expression_parser.AST, context string) string {
	if node == nil {
		return ""
	}
	switch n := node.(type) {
	case *expression_parser.ImplicitReceiver:
		return context
	case *expression_parser.PropertyRead:
		receiver := astToString(n.Receiver, context)
		if receiver == "" || receiver == "this" {
			return "this." + n.Name
		}
		return receiver + "." + n.Name
	case *expression_parser.SafePropertyRead:
		receiver := astToString(n.Receiver, context)
		if receiver == "" || receiver == "this" {
			receiver = "this"
		}
		return fmt.Sprintf("(%s == null ? null : %s.%s)", receiver, receiver, n.Name)
	case *expression_parser.PropertyWrite:
		receiver := astToString(n.Receiver, context)
		if receiver == "" || receiver == "this" {
			receiver = "this"
		}
		return fmt.Sprintf("%s.%s = (%s)", receiver, n.Name, astToString(n.Value, context))
	case *expression_parser.MethodCall:
		receiver := astToString(n.Receiver, context)
		if receiver == "" || receiver == "this" {
			receiver = "this"
		}
		args := []string{}
		for _, arg := range n.Args {
			args = append(args, astToString(arg, context))
		}
		return fmt.Sprintf("%s.%s(%s)", receiver, n.Name, strings.Join(args, ", "))
	case *expression_parser.SafeMethodCall:
		receiver := astToString(n.Receiver, context)
		if receiver == "" || receiver == "this" {
			receiver = "this"
		}
		args := []string{}
		for _, arg := range n.Args {
			args = append(args, astToString(arg, context))
		}
		return fmt.Sprintf("(%s == null ? null : %s.%s(%s))", receiver, receiver, n.Name, strings.Join(args, ", "))
	case *expression_parser.LiteralPrimitive:
		switch val := n.Value.(type) {
		case string:
			return fmt.Sprintf("%q", val)
		default:
			return fmt.Sprintf("%v", val)
		}
	case *expression_parser.LiteralArray:
		elems := []string{}
		for _, elem := range n.Expressions {
			elems = append(elems, astToString(elem, context))
		}
		return "[" + strings.Join(elems, ", ") + "]"
	case *expression_parser.LiteralMap:
		keys := []string{}
		for i, key := range n.Keys {
			if propKey, ok := key.(*expression_parser.LiteralMapPropertyKey); ok {
				keys = append(keys, fmt.Sprintf("%s: %s", propKey.Key, astToString(n.Values[i], context)))
			} else {
				keys = append(keys, fmt.Sprintf("...%s", astToString(n.Values[i], context)))
			}
		}
		return "{" + strings.Join(keys, ", ") + "}"
	case *expression_parser.Binary:
		return fmt.Sprintf("(%s %s %s)", astToString(n.Left, context), n.Operation, astToString(n.Right, context))
	case *expression_parser.PrefixNot:
		return fmt.Sprintf("(!%s)", astToString(n.Expression, context))
	case *expression_parser.Conditional:
		return fmt.Sprintf("(%s ? %s : %s)", astToString(n.Condition, context), astToString(n.TrueExp, context), astToString(n.FalseExp, context))
	case *expression_parser.NonNullAssert:
		return fmt.Sprintf("(%s!)", astToString(n.Expression, context))
	case *expression_parser.BindingPipe:
		return fmt.Sprintf("_pipe_%s.transform(%s)", n.Name, astToString(n.Exp, context))
	case *expression_parser.Interpolation:
		exprs := []string{}
		for _, expr := range n.Expressions {
			exprs = append(exprs, astToString(expr, context))
		}
		return strings.Join(exprs, " + ")
	case *expression_parser.ASTWithSource:
		return astToString(n.Ast, context)
	}
	return ""
}
