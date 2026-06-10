package typecheck

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type DirectiveInfo struct {
	ClassName    string
	OwningModule string
}

// GenerateTcb converts a component template AST into a Type Check Block (TCB) TypeScript string
func GenerateTcb(
	className string,
	parsedTemplate *render3.ParsedTemplate,
	getDirectives func(node render3.Node) []DirectiveInfo,
	getBindingConsumer func(node render3.Node, binding any) (string, string, bool),
	pipes map[string]string,
) (string, map[int]parse_util.ParseSourceSpan) {
	return GenerateTcbWithOptions(className, parsedTemplate, getDirectives, getBindingConsumer, pipes, false, 0)
}

// GenerateTcbWithOptions converts a component template AST into a TCB string with advanced option support.
func GenerateTcbWithOptions(
	className string,
	parsedTemplate *render3.ParsedTemplate,
	getDirectives func(node render3.Node) []DirectiveInfo,
	getBindingConsumer func(node render3.Node, binding any) (string, string, bool),
	pipes map[string]string,
	emitSpans bool,
	absoluteOffset int,
) (string, map[int]parse_util.ParseSourceSpan) {
	var sb strings.Builder
	lineSpans := make(map[int]parse_util.ParseSourceSpan)

	sourceTemplate := ""
	if len(parsedTemplate.Nodes) > 0 {
		span := parsedTemplate.Nodes[0].GetSourceSpan()
		if span.Start != nil && span.Start.File != nil {
			sourceTemplate = span.Start.File.Content
		}
	}

	currentLine := 1
	writeLine := func(text string, span parse_util.ParseSourceSpan) {
		sb.WriteString(text)
		sb.WriteString("\n")
		if span.Start != nil {
			lineSpans[currentLine] = span
		}
		currentLine++
	}

	// 1. Declare the TCB function and assignment helper
	writeLine(fmt.Sprintf("function _tcb_%s(this: %s) {", className, className), parse_util.ParseSourceSpan{})
	writeLine("  function _assign<T>(target: T | { set(val: T): any }, value: T) {}", parse_util.ParseSourceSpan{})

	// 2. Declare used pipes
	pipeVars := make(map[string]string)
	if emitSpans {
		usedPipeNames := []string{}
		pipeSeen := make(map[string]bool)
		findPipes(parsedTemplate.Nodes, pipeSeen, &usedPipeNames)
		for i, pipeName := range usedPipeNames {
			pipeVar := fmt.Sprintf("_pipe%d", i+1)
			pipeVars[pipeName] = pipeVar
			pipeClass := pipes[pipeName]
			writeLine(fmt.Sprintf("  var %s = null! as %s;", pipeVar, pipeClass), parse_util.ParseSourceSpan{})
		}
	} else {
		for pipeName, pipeClass := range pipes {
			pipeVar := fmt.Sprintf("_pipe_%s", pipeName)
			pipeVars[pipeName] = pipeVar
			writeLine(fmt.Sprintf("  var %s: %s = null!;", pipeVar, pipeClass), parse_util.ParseSourceSpan{})
		}
	}

	var walkNodes func(nodes []render3.Node, indent string, scopeVars map[string]bool)
	var walkNode func(node render3.Node, indent string, scopeVars map[string]bool)

	elId := 0
	dirId := 0

	cloneScopeVars := func(vars map[string]bool) map[string]bool {
		res := make(map[string]bool, len(vars))
		for k, v := range vars {
			res[k] = v
		}
		return res
	}

	walkNode = func(node render3.Node, indent string, scopeVars map[string]bool) {
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
			dirClassToId := make(map[string]int)
			for _, dirInfo := range dirs {
				dirId++
				matchedDirs = append(matchedDirs, dirId)
				dirClassToId[dirInfo.ClassName] = dirId
				typeRef := dirInfo.ClassName
				if dirInfo.OwningModule != "" {
					typeRef = fmt.Sprintf("import('%s').%s", dirInfo.OwningModule, dirInfo.ClassName)
				}
				writeLine(fmt.Sprintf("%s  var _dir%d: %s = null!;", indent, dirId, typeRef), n.SourceSpan)
			}

			newScopeVars := cloneScopeVars(scopeVars)
			for _, ref := range n.References {
				newScopeVars[ref.Name] = true
			}

			// Typecheck local references
			for _, ref := range n.References {
				targetVar := fmt.Sprintf("_el%d", myElId)
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
				exprStr := astToStringWithOptions(prop.Value, "this", newScopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
				if exprStr != "" {
					if consumerClass, classPropName, hasConsumer := getBindingConsumer(n, prop); hasConsumer {
						if dId, ok := dirClassToId[consumerClass]; ok {
							writeLine(fmt.Sprintf("%s  _assign(_dir%d[%q], (%s));", indent, dId, classPropName, exprStr), prop.SourceSpan)
						}
					} else {
						if prop.Type != 1 && prop.Type != 2 && prop.Type != 3 && prop.Name != "style" && prop.Name != "class" {
							writeLine(fmt.Sprintf("%s  _assign(_el%d[%q], (%s));", indent, myElId, prop.Name, exprStr), prop.SourceSpan)
						} else {
							// For attributes, class, and style, just typecheck the expression without assigning to the DOM element
							writeLine(fmt.Sprintf("%s  (%s);", indent, exprStr), prop.SourceSpan)
						}
					}
				}
			}

			// Typecheck outputs/events
			for _, event := range n.Outputs {
				eventScopeVars := cloneScopeVars(newScopeVars)
				eventScopeVars["$event"] = true
				eventScopeVars["_event"] = true
				handlerStr := astToStringWithOptions(event.Handler, "this", eventScopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
				if handlerStr != "" {
					eventType := getEventType(event.Name)
					replacedHandler := strings.ReplaceAll(handlerStr, "$event", "_event")
					writeLine(fmt.Sprintf("%s  ((_event: %s) => { %s; })(null!);", indent, eventType, replacedHandler), event.SourceSpan)
				}
			}

			walkNodes(n.Children, indent, newScopeVars)

		case *render3.Template:
			elId++
			myElId := elId

			writeLine(fmt.Sprintf("%s  var _el%d: any = null!;", indent, myElId), n.SourceSpan)

			dirs := getDirectives(n)
			var matchedDirs []int
			dirClassToId := make(map[string]int)
			for _, dirInfo := range dirs {
				dirId++
				matchedDirs = append(matchedDirs, dirId)
				dirClassToId[dirInfo.ClassName] = dirId
				typeRef := dirInfo.ClassName
				if dirInfo.OwningModule != "" {
					typeRef = fmt.Sprintf("import('%s').%s", dirInfo.OwningModule, dirInfo.ClassName)
				}
				writeLine(fmt.Sprintf("%s  var _dir%d: %s = null!;", indent, dirId, typeRef), n.SourceSpan)
			}

			newScopeVars := cloneScopeVars(scopeVars)
			for _, ref := range n.References {
				newScopeVars[ref.Name] = true
			}
			for _, v := range n.Variables {
				newScopeVars[v.Name] = true
			}

			// Declare template variables
			for _, v := range n.Variables {
				writeLine(fmt.Sprintf("%s  var %s: any = null!;", indent, v.Name), v.SourceSpan)
			}

			// Typecheck local references
			for _, ref := range n.References {
				targetVar := fmt.Sprintf("_el%d", myElId)
				if ref.Value != "" && len(matchedDirs) > 0 {
					targetVar = fmt.Sprintf("_dir%d", matchedDirs[0])
				}
				writeLine(fmt.Sprintf("%s  var %s = %s;", indent, ref.Name, targetVar), ref.SourceSpan)
			}

			for _, prop := range n.Inputs {
				exprStr := astToStringWithOptions(prop.Value, "this", newScopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
				if exprStr != "" {
					if consumerClass, classPropName, hasConsumer := getBindingConsumer(n, prop); hasConsumer {
						if dId, ok := dirClassToId[consumerClass]; ok {
							writeLine(fmt.Sprintf("%s  _assign(_dir%d[%q], (%s));", indent, dId, classPropName, exprStr), prop.SourceSpan)
						}
					}
				}
			}
			for _, attr := range n.TemplateAttrs {
				if prop, ok := attr.(*render3.BoundAttribute); ok {
					exprStr := astToStringWithOptions(prop.Value, "this", newScopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
					if exprStr != "" {
						if consumerClass, classPropName, hasConsumer := getBindingConsumer(n, prop); hasConsumer {
							if dId, ok := dirClassToId[consumerClass]; ok {
								writeLine(fmt.Sprintf("%s  _assign(_dir%d[%q], (%s));", indent, dId, classPropName, exprStr), prop.SourceSpan)
							}
						}
					}
				}
			}

			walkNodes(n.Children, indent, newScopeVars)

		case *render3.BoundText:
			exprStr := astToStringWithOptions(n.Value, "this", scopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
			if exprStr != "" {
				writeLine(fmt.Sprintf("%s  (%s);", indent, exprStr), n.SourceSpan)
			}

		case *render3.IfBlock:
			for _, branch := range n.Branches {
				exprStr := astToStringWithOptions(branch.Expression, "this", scopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
				if exprStr != "" {
					writeLine(fmt.Sprintf("%s  if (%s) {", indent, exprStr), branch.SourceSpan)
				} else {
					writeLine(fmt.Sprintf("%s  {", indent), branch.SourceSpan)
				}
				walkNodes(branch.Children, indent+"  ", scopeVars)
				writeLine(fmt.Sprintf("%s  }", indent), branch.SourceSpan)
			}

		case *render3.ForLoopBlock:
			exprStr := astToStringWithOptions(&n.Expression, "this", scopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
			itemName := "item"
			if n.Item != nil {
				itemName = n.Item.Name
			}
			newScopeVars := cloneScopeVars(scopeVars)
			newScopeVars[itemName] = true

			writeLine(fmt.Sprintf("%s  for (let %s of (%s)) {", indent, itemName, exprStr), n.SourceSpan)
			walkNodes(n.Children, indent+"  ", newScopeVars)
			writeLine(fmt.Sprintf("%s  }", indent), n.SourceSpan)
			if n.Empty != nil {
				writeLine(fmt.Sprintf("%s  {", indent), n.Empty.SourceSpan)
				walkNodes(n.Empty.Children, indent+"  ", scopeVars)
				writeLine(fmt.Sprintf("%s  }", indent), n.Empty.SourceSpan)
			}

		case *render3.SwitchBlock:
			exprStr := astToStringWithOptions(n.Expression, "this", scopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
			writeLine(fmt.Sprintf("%s  switch (%s) {", indent, exprStr), n.SourceSpan)
			for _, group := range n.Groups {
				for _, c := range group.Cases {
					if c.Expression != nil {
						exprStr = astToStringWithOptions(c.Expression, "this", scopeVars, absoluteOffset, emitSpans, pipeVars, sourceTemplate)
						writeLine(fmt.Sprintf("%s    case %s:", indent, exprStr), c.SourceSpan)
					} else {
						writeLine(fmt.Sprintf("%s    default:", indent), c.SourceSpan)
					}
				}
				walkNodes(group.Children, indent+"    ", scopeVars)
			}
			writeLine(fmt.Sprintf("%s  }", indent), n.SourceSpan)

		case *render3.DeferredBlock:
			writeLine(fmt.Sprintf("%s  {", indent), n.SourceSpan)
			walkNodes(n.Children, indent+"  ", scopeVars)
			if n.Placeholder != nil {
				walkNodes(n.Placeholder.Children, indent+"  ", scopeVars)
			}
			if n.Loading != nil {
				walkNodes(n.Loading.Children, indent+"  ", scopeVars)
			}
			if n.Error != nil {
				walkNodes(n.Error.Children, indent+"  ", scopeVars)
			}
			writeLine(fmt.Sprintf("%s  }", indent), n.SourceSpan)
		}
	}

	walkNodes = func(nodes []render3.Node, indent string, scopeVars map[string]bool) {
		for _, node := range nodes {
			if el, ok := node.(*render3.Element); ok {
				for _, ref := range el.References {
					scopeVars[ref.Name] = true
				}
			}
			if tmpl, ok := node.(*render3.Template); ok {
				for _, ref := range tmpl.References {
					scopeVars[ref.Name] = true
				}
			}
			walkNode(node, indent, scopeVars)
		}
	}

	initialScopeVars := make(map[string]bool)
	walkNodes(parsedTemplate.Nodes, "  ", initialScopeVars)

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

func astToString(node expression_parser.AST, context string, scopeVars map[string]bool) string {
	return astToStringWithOptions(node, context, scopeVars, 0, false, nil, "")
}

func getNameSpan(source string, start int, name string) expression_parser.ParseSpan {
	if source == "" {
		return expression_parser.ParseSpan{Start: start, End: start + len(name)}
	}
	idx := strings.Index(source[start:], name)
	if idx != -1 {
		return expression_parser.ParseSpan{
			Start: start + idx,
			End:   start + idx + len(name),
		}
	}
	return expression_parser.ParseSpan{Start: start, End: start + len(name)}
}

func astToStringWithOptions(
	node expression_parser.AST,
	context string,
	scopeVars map[string]bool,
	absoluteOffset int,
	emitSpans bool,
	pipeVars map[string]string,
	sourceTemplate string,
) string {
	if node == nil {
		return ""
	}

	comment := func(span expression_parser.ParseSpan) string {
		if !emitSpans {
			return ""
		}
		return fmt.Sprintf(" /*%d,%d*/", absoluteOffset+span.Start, absoluteOffset+span.End)
	}

	wrap := func(expr string, span expression_parser.ParseSpan) string {
		if !emitSpans {
			return expr
		}
		return fmt.Sprintf("(%s)%s", expr, comment(span))
	}

	parenthesize := func(expr string) string {
		if !emitSpans {
			return expr
		}
		return fmt.Sprintf("(%s)", expr)
	}

	var translate func(n expression_parser.AST, isSubExpr bool) string
	translate = func(n expression_parser.AST, isSubExpr bool) string {
		if n == nil {
			return ""
		}
		switch astNode := n.(type) {
		case *expression_parser.ImplicitReceiver:
			return context

		case *expression_parser.PropertyRead:
			receiver := translate(astNode.Receiver, true)
			if receiver == "" || receiver == "this" {
				if scopeVars[astNode.Name] {
					res := astNode.Name
					if emitSpans {
						res += comment(astNode.Span())
					}
					return res
				}
				receiver = "this"
			}
			receiverWrapped := receiver
			propAccess := receiverWrapped + "." + astNode.Name
			nameSpan := getNameSpan(sourceTemplate, absoluteOffset+astNode.Span().Start, astNode.Name)
			if emitSpans {
				propAccess += comment(nameSpan)
			}
			res := wrap(propAccess, astNode.Span())
			if isSubExpr {
				res = parenthesize(res)
			}
			return res

		case *expression_parser.SafePropertyRead:
			receiver := translate(astNode.Receiver, true)
			nameSpan := getNameSpan(sourceTemplate, absoluteOffset+astNode.Span().Start, astNode.Name)
			res := fmt.Sprintf("%s?.%s", receiver, astNode.Name)
			if emitSpans {
				res += comment(nameSpan)
				res += comment(astNode.Span())
			}
			return parenthesize(res)

		case *expression_parser.PropertyWrite:
			receiver := translate(astNode.Receiver, true)
			if receiver == "" || receiver == "this" {
				if scopeVars[astNode.Name] {
					expr := fmt.Sprintf("%s = %s", astNode.Name, translate(astNode.Value, true))
					return wrap(expr, astNode.Span())
				}
				receiver = "this"
			}
			receiverWrapped := receiver
			valStr := translate(astNode.Value, true)
			expr := fmt.Sprintf("%s.%s = %s", receiverWrapped, astNode.Name, valStr)
			return wrap(expr, astNode.Span())

		case *expression_parser.MethodCall:
			if astNode.Name == "$any" && len(astNode.Args) == 1 {
				argStr := translate(astNode.Args[0], true)
				res := fmt.Sprintf("%s as any", argStr)
				return wrap(res, astNode.Span())
			}

			var receiver string
			var methodStr string
			if propRead, ok := astNode.Receiver.(*expression_parser.PropertyRead); ok {
				receiver = translate(propRead.Receiver, true)
				if receiver == "" || receiver == "this" {
					receiver = "this"
				}
				receiverWrapped := parenthesize(receiver)
				nameSpan := getNameSpan(sourceTemplate, absoluteOffset+propRead.Span().Start, propRead.Name)
				methodStr = fmt.Sprintf("%s.%s", receiverWrapped, propRead.Name)
				if emitSpans {
					methodStr += comment(nameSpan)
				}
			} else {
				receiver = translate(astNode.Receiver, true)
				methodStr = receiver
			}

			var args []string
			for _, arg := range astNode.Args {
				args = append(args, translate(arg, true))
			}
			res := fmt.Sprintf("%s(%s)", methodStr, strings.Join(args, ", "))
			if emitSpans {
				res += comment(astNode.Span())
			}
			return res

		case *expression_parser.SafeMethodCall:
			receiver := translate(astNode.Receiver, true)
			receiverWrapped := parenthesize(receiver)
			nameSpan := getNameSpan(sourceTemplate, absoluteOffset+astNode.Span().Start, astNode.Name)
			methodStr := fmt.Sprintf("%s?.%s", receiverWrapped, astNode.Name)
			if emitSpans {
				methodStr += comment(nameSpan)
				methodStr += comment(expression_parser.ParseSpan{Start: astNode.Span().Start, End: nameSpan.End}) // Property access comment
			}

			var args []string
			for _, arg := range astNode.Args {
				args = append(args, translate(arg, true))
			}
			res := fmt.Sprintf("(0 as any ? %s!(%s) : undefined)", methodStr, strings.Join(args, ", "))
			if emitSpans {
				res += comment(astNode.Span())
			}
			return parenthesize(res)

		case *expression_parser.FunctionCall:
			target := translate(astNode.Target, true)
			var args []string
			for _, arg := range astNode.Args {
				args = append(args, translate(arg, true))
			}
			res := fmt.Sprintf("%s(%s)", target, strings.Join(args, ", "))
			if emitSpans {
				res += comment(astNode.Span())
			}
			return res

		case *expression_parser.Call:
			if propRead, ok := astNode.Receiver.(*expression_parser.PropertyRead); ok && propRead.Name == "$any" && len(astNode.Args) == 1 {
				argStr := translate(astNode.Args[0], false)
				res := fmt.Sprintf("%s as any", argStr)
				return wrap(res, astNode.Span())
			}
			if safePropRead, ok := astNode.Receiver.(*expression_parser.SafePropertyRead); ok {
				receiver := translate(safePropRead.Receiver, true)
				nameSpan := getNameSpan(sourceTemplate, absoluteOffset+safePropRead.Span().Start, safePropRead.Name)
				methodStr := fmt.Sprintf("%s?.%s", receiver, safePropRead.Name)
				if emitSpans {
					methodStr += comment(nameSpan)
					methodStr += comment(expression_parser.ParseSpan{Start: safePropRead.Span().Start, End: nameSpan.End})
				}
				var args []string
				for _, arg := range astNode.Args {
					args = append(args, translate(arg, false))
				}
				res := fmt.Sprintf("(0 as any ? %s!(%s) : undefined)", methodStr, strings.Join(args, ", "))
				if emitSpans {
					res += comment(astNode.Span())
				}
				return parenthesize(res)
			}
			if propRead, ok := astNode.Receiver.(*expression_parser.PropertyRead); ok {
				receiver := translate(propRead.Receiver, true)
				if receiver == "" || receiver == "this" {
					if scopeVars[propRead.Name] {
						methodStr := propRead.Name
						if emitSpans {
							methodStr += comment(propRead.Span())
						}
						var args []string
						for _, arg := range astNode.Args {
							args = append(args, translate(arg, false))
						}
						res := fmt.Sprintf("%s(%s)", methodStr, strings.Join(args, ", "))
						if emitSpans {
							res += comment(astNode.Span())
						}
						return res
					}
					receiver = "this"
				}
				receiverWrapped := receiver
				nameSpan := getNameSpan(sourceTemplate, absoluteOffset+propRead.Span().Start, propRead.Name)
				methodStr := fmt.Sprintf("%s.%s", receiverWrapped, propRead.Name)
				if emitSpans {
					methodStr += comment(nameSpan)
				}
				var args []string
				for _, arg := range astNode.Args {
					args = append(args, translate(arg, false))
				}
				res := fmt.Sprintf("%s(%s)", methodStr, strings.Join(args, ", "))
				if emitSpans {
					res += comment(astNode.Span())
				}
				return res
			} else {
				target := translate(astNode.Receiver, true)
				var args []string
				for _, arg := range astNode.Args {
					args = append(args, translate(arg, false))
				}
				res := fmt.Sprintf("%s(%s)", target, strings.Join(args, ", "))
				if emitSpans {
					res += comment(astNode.Span())
				}
				return res
			}

		case *expression_parser.SafeCall:
			if safePropRead, ok := astNode.Receiver.(*expression_parser.SafePropertyRead); ok {
				receiver := translate(safePropRead.Receiver, true)
				receiverWrapped := receiver
				nameSpan := getNameSpan(sourceTemplate, absoluteOffset+safePropRead.Span().Start, safePropRead.Name)
				methodStr := fmt.Sprintf("%s?.%s", receiverWrapped, safePropRead.Name)
				if emitSpans {
					methodStr += comment(nameSpan)
					methodStr += comment(expression_parser.ParseSpan{Start: safePropRead.Span().Start, End: nameSpan.End})
				}
				var args []string
				for _, arg := range astNode.Args {
					args = append(args, translate(arg, false))
				}
				res := fmt.Sprintf("(0 as any ? %s!(%s) : undefined)", methodStr, strings.Join(args, ", "))
				if emitSpans {
					res += comment(astNode.Span())
				}
				return parenthesize(res)
			} else {
				target := translate(astNode.Receiver, true)
				var args []string
				for _, arg := range astNode.Args {
					args = append(args, translate(arg, false))
				}
				res := fmt.Sprintf("(0 as any ? %s!(%s) : undefined)", target, strings.Join(args, ", "))
				if emitSpans {
					res += comment(astNode.Span())
				}
				return parenthesize(res)
			}

		case *expression_parser.KeyedRead:
			receiver := translate(astNode.Receiver, true)
			receiverWrapped := receiver
			keyStr := translate(astNode.Key, true)
			res := fmt.Sprintf("%s[%s]", receiverWrapped, keyStr)
			if emitSpans {
				res += comment(astNode.Span())
			}
			return res

		case *expression_parser.KeyedWrite:
			receiver := translate(astNode.Receiver, true)
			receiverWrapped := receiver
			keyStr := translate(astNode.Key, true)
			valStr := translate(astNode.Value, true)
			readSpan := expression_parser.ParseSpan{
				Start: astNode.Receiver.Span().Start,
				End:   astNode.Key.Span().End + 1,
			}
			readExpr := fmt.Sprintf("%s[%s]", receiverWrapped, keyStr)
			readWrapped := wrap(readExpr, readSpan)
			expr := fmt.Sprintf("%s = %s", readWrapped, valStr)
			return wrap(expr, astNode.Span())

		case *expression_parser.SafeKeyedRead:
			receiver := translate(astNode.Receiver, true)
			keyStr := translate(astNode.Key, true)
			res := fmt.Sprintf("%s?.[%s]", receiver, keyStr)
			if emitSpans {
				res += comment(astNode.Span())
			}
			return parenthesize(res)

		case *expression_parser.LiteralPrimitive:
			var valStr string
			switch val := astNode.Value.(type) {
			case string:
				valStr = fmt.Sprintf("%q", val)
			case nil:
				valStr = "null"
			default:
				valStr = fmt.Sprintf("%v", val)
			}
			res := valStr
			if emitSpans {
				res += comment(astNode.Span())
			}
			return res

		case *expression_parser.LiteralArray:
			var elems []string
			for _, elem := range astNode.Expressions {
				elems = append(elems, translate(elem, true))
			}
			res := "[" + strings.Join(elems, ", ") + "]"
			return wrap(res, astNode.Span())

		case *expression_parser.LiteralMap:
			var props []string
			for i, key := range astNode.Keys {
				value := translate(astNode.Values[i], true)
				if propKey, ok := key.(*expression_parser.LiteralMapPropertyKey); ok {
					keyStr := fmt.Sprintf("%q", propKey.Key)
					if emitSpans {
						keyStr += comment(propKey.Span)
					}
					props = append(props, fmt.Sprintf("%s: %s", keyStr, value))
				}
			}
			res := "{ " + strings.Join(props, ", ") + " }"
			resWrapped := wrap(res, astNode.Span())
			return parenthesize(resWrapped)

		case *expression_parser.Binary:
			lhs := parenthesize(translate(astNode.Left, true))
			rhs := parenthesize(translate(astNode.Right, true))
			res := fmt.Sprintf("%s %s %s", lhs, astNode.Operation, rhs)
			if emitSpans {
				res += comment(astNode.Span())
			}
			return res

		case *expression_parser.Unary:
			expr := translate(astNode.Expr, false)
			res := astNode.Operator + expr
			wrapped := wrap(res, astNode.Span())
			if isSubExpr {
				wrapped = parenthesize(wrapped)
			}
			return wrapped

		case *expression_parser.PrefixNot:
			expr := parenthesize(translate(astNode.Expression, true))
			res := "!" + expr
			return wrap(res, astNode.Span())

		case *expression_parser.Conditional:
			cond := parenthesize(translate(astNode.Condition, true))
			trueVal := translate(astNode.TrueExp, true)
			falseVal := parenthesize(translate(astNode.FalseExp, true))
			res := fmt.Sprintf("%s ? %s : %s", cond, trueVal, falseVal)
			return wrap(res, astNode.Span())

		case *expression_parser.NonNullAssert:
			expr := parenthesize(translate(astNode.Expression, true))
			res := expr + "!"
			if emitSpans {
				res += comment(astNode.Span())
			}
			return res

		case *expression_parser.BindingPipe:
			pipeVar := "_pipe_" + astNode.Name
			if v, ok := pipeVars[astNode.Name]; ok {
				pipeVar = v
			}
			nameSpan := getNameSpan(sourceTemplate, absoluteOffset+astNode.Span().Start, astNode.Name)
			pipeExpr := fmt.Sprintf("%s.transform", pipeVar)
			if emitSpans {
				pipeExpr += comment(nameSpan)
			}
			var args []string
			args = append(args, translate(astNode.Exp, false))
			for _, arg := range astNode.Args {
				args = append(args, translate(arg, false))
			}
			res := fmt.Sprintf("%s(%s)", pipeExpr, strings.Join(args, ", "))
			if emitSpans {
				res += comment(astNode.Span())
			}
			return parenthesize(res)

		case *expression_parser.Interpolation:
			var exprs []string
			for _, expr := range astNode.Expressions {
				exprs = append(exprs, parenthesize(translate(expr, true)))
			}
			return "\"\" + " + strings.Join(exprs, " + ")

		case *expression_parser.Chain:
			var exprs []string
			for _, expr := range astNode.Expressions {
				exprs = append(exprs, translate(expr, true))
			}
			res := strings.Join(exprs, ", ")
			return wrap(res, astNode.Span())

		case *expression_parser.ASTWithSource:
			return translate(astNode.Ast, isSubExpr)
		}
		return ""
	}

	return translate(node, false)
}

func findPipes(nodes []render3.Node, seen map[string]bool, names *[]string) {
	for _, node := range nodes {
		findPipe(node, seen, names)
	}
}

func findPipe(node render3.Node, seen map[string]bool, names *[]string) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *render3.Element:
		for _, prop := range n.Inputs {
			findPipesInAst(prop.Value, seen, names)
		}
		for _, event := range n.Outputs {
			findPipesInAst(event.Handler, seen, names)
		}
		findPipes(n.Children, seen, names)
	case *render3.Template:
		for _, prop := range n.Inputs {
			findPipesInAst(prop.Value, seen, names)
		}
		for _, attr := range n.TemplateAttrs {
			if prop, ok := attr.(*render3.BoundAttribute); ok {
				findPipesInAst(prop.Value, seen, names)
			}
		}
		findPipes(n.Children, seen, names)
	case *render3.BoundText:
		findPipesInAst(n.Value, seen, names)
	case *render3.IfBlock:
		for _, branch := range n.Branches {
			findPipesInAst(branch.Expression, seen, names)
			findPipes(branch.Children, seen, names)
		}
	case *render3.ForLoopBlock:
		findPipesInAst(&n.Expression, seen, names)
		findPipes(n.Children, seen, names)
		if n.Empty != nil {
			findPipes(n.Empty.Children, seen, names)
		}
	case *render3.SwitchBlock:
		findPipesInAst(n.Expression, seen, names)
		for _, group := range n.Groups {
			for _, c := range group.Cases {
				findPipesInAst(c.Expression, seen, names)
			}
			findPipes(group.Children, seen, names)
		}
	case *render3.DeferredBlock:
		findPipes(n.Children, seen, names)
		if n.Placeholder != nil {
			findPipes(n.Placeholder.Children, seen, names)
		}
		if n.Loading != nil {
			findPipes(n.Loading.Children, seen, names)
		}
		if n.Error != nil {
			findPipes(n.Error.Children, seen, names)
		}
	}
}

func findPipesInAst(ast expression_parser.AST, seen map[string]bool, names *[]string) {
	if ast == nil {
		return
	}
	switch n := ast.(type) {
	case *expression_parser.BindingPipe:
		if !seen[n.Name] {
			seen[n.Name] = true
			*names = append(*names, n.Name)
		}
		findPipesInAst(n.Exp, seen, names)
		for _, arg := range n.Args {
			findPipesInAst(arg, seen, names)
		}
	case *expression_parser.PropertyRead:
		findPipesInAst(n.Receiver, seen, names)
	case *expression_parser.SafePropertyRead:
		findPipesInAst(n.Receiver, seen, names)
	case *expression_parser.PropertyWrite:
		findPipesInAst(n.Receiver, seen, names)
		findPipesInAst(n.Value, seen, names)
	case *expression_parser.MethodCall:
		findPipesInAst(n.Receiver, seen, names)
		for _, arg := range n.Args {
			findPipesInAst(arg, seen, names)
		}
	case *expression_parser.SafeMethodCall:
		findPipesInAst(n.Receiver, seen, names)
		for _, arg := range n.Args {
			findPipesInAst(arg, seen, names)
		}
	case *expression_parser.FunctionCall:
		findPipesInAst(n.Target, seen, names)
		for _, arg := range n.Args {
			findPipesInAst(arg, seen, names)
		}
	case *expression_parser.Binary:
		findPipesInAst(n.Left, seen, names)
		findPipesInAst(n.Right, seen, names)
	case *expression_parser.PrefixNot:
		findPipesInAst(n.Expression, seen, names)
	case *expression_parser.NonNullAssert:
		findPipesInAst(n.Expression, seen, names)
	case *expression_parser.Conditional:
		findPipesInAst(n.Condition, seen, names)
		findPipesInAst(n.TrueExp, seen, names)
		findPipesInAst(n.FalseExp, seen, names)
	case *expression_parser.LiteralArray:
		for _, expr := range n.Expressions {
			findPipesInAst(expr, seen, names)
		}
	case *expression_parser.LiteralMap:
		for _, val := range n.Values {
			findPipesInAst(val, seen, names)
		}
	case *expression_parser.Interpolation:
		for _, expr := range n.Expressions {
			findPipesInAst(expr, seen, names)
		}
	case *expression_parser.ASTWithSource:
		findPipesInAst(n.Ast, seen, names)
	}
}
