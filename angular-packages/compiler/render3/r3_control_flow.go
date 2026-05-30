package render3

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

var (
	forLoopExpressionPattern                 = regexp.MustCompile(`^\s*([0-9A-Za-z_$]*)\s+of\s+([\S\s]*)`)
	forLoopTrackPattern                      = regexp.MustCompile(`^track\s+([\S\s]*)`)
	conditionalAliasPattern                  = regexp.MustCompile(`^(as\s+)(.*)`)
	elseIfPattern                            = regexp.MustCompile(`^else[^\S\r\n]+if`)
	forLoopLetPattern                        = regexp.MustCompile(`^let\s+([\S\s]*)`)
	identifierPattern                        = regexp.MustCompile(`(?i)^[$A-Z_][0-9A-Z_$]*$`)
	charactersInSurroundingWhitespacePattern = regexp.MustCompile(`(\s*)(\S+)(\s*)`)
)

var allowedForLoopLetVariables = map[string]bool{
	"$index": true,
	"$first": true,
	"$last":  true,
	"$even":  true,
	"$odd":   true,
	"$count": true,
}

func IsConnectedForLoopBlock(name string) bool {
	return name == "empty"
}

func IsConnectedIfLoopBlock(name string) bool {
	return name == "else" || elseIfPattern.MatchString(name)
}

func CreateIfBlock(
	ast *ml_parser.Block,
	connectedBlocks []*ml_parser.Block,
	visitor ml_parser.Visitor,
	bindingParser *template_parser.BindingParser,
) (*IfBlock, []parse_util.ParseError) {
	var parseErrors []parse_util.ParseError
	validateIfConnectedBlocks(connectedBlocks, &parseErrors)

	var branches []*IfBlockBranch
	mainParams := parseConditionalBlockParameters(ast, &parseErrors, bindingParser)
	if mainParams != nil {
		children := htmlVisitAll(visitor, ast.Children, ast.Children)
		
		nameSpan := parse_util.ParseSourceSpan{}
		if ast.StartSourceSpan != nil {
			nameSpan = *ast.StartSourceSpan
		}

		branches = append(branches, &IfBlockBranch{
			BlockNode: BlockNode{
				NameSpan:        nameSpan,
				SourceSpan:      *ast.SourceSpan,
				StartSourceSpan: *ast.StartSourceSpan,
				EndSourceSpan:   ast.EndSourceSpan,
			},
			Expression:      mainParams.expression,
			Children:        children,
			ExpressionAlias: mainParams.expressionAlias,
		})
	}

	for _, block := range connectedBlocks {
		if elseIfPattern.MatchString(block.Name) {
			params := parseConditionalBlockParameters(block, &parseErrors, bindingParser)
			if params != nil {
				children := htmlVisitAll(visitor, block.Children, block.Children)
				branches = append(branches, &IfBlockBranch{
					BlockNode: BlockNode{
						NameSpan:        *block.StartSourceSpan,
						SourceSpan:      *block.SourceSpan,
						StartSourceSpan: *block.StartSourceSpan,
						EndSourceSpan:   block.EndSourceSpan,
					},
					Expression:      params.expression,
					Children:        children,
					ExpressionAlias: params.expressionAlias,
				})
			}
		} else if block.Name == "else" {
			children := htmlVisitAll(visitor, block.Children, block.Children)
			branches = append(branches, &IfBlockBranch{
				BlockNode: BlockNode{
					NameSpan:        *block.StartSourceSpan,
					SourceSpan:      *block.SourceSpan,
					StartSourceSpan: *block.StartSourceSpan,
					EndSourceSpan:   block.EndSourceSpan,
				},
				Expression:      nil,
				Children:        children,
				ExpressionAlias: nil,
			})
		}
	}

	var startSourceSpan parse_util.ParseSourceSpan
	if len(branches) > 0 {
		startSourceSpan = branches[0].StartSourceSpan
	} else if ast.StartSourceSpan != nil {
		startSourceSpan = *ast.StartSourceSpan
	}

	var endSourceSpan *parse_util.ParseSourceSpan
	if len(branches) > 0 {
		endSourceSpan = branches[len(branches)-1].EndSourceSpan
	} else {
		endSourceSpan = ast.EndSourceSpan
	}

	wholeSourceSpan := *ast.SourceSpan
	if len(branches) > 0 {
		lastBranch := branches[len(branches)-1]
		wholeSourceSpan = *parse_util.NewParseSourceSpan(
			startSourceSpan.Start,
			lastBranch.SourceSpan.End,
			nil,
			nil,
		)
	}

	nameSpan := parse_util.ParseSourceSpan{}
	if ast.StartSourceSpan != nil {
		nameSpan = *ast.StartSourceSpan
	}

	node := &IfBlock{
		BlockNode: BlockNode{
			NameSpan:        nameSpan,
			SourceSpan:      wholeSourceSpan,
			StartSourceSpan: startSourceSpan,
			EndSourceSpan:   endSourceSpan,
		},
		Branches: branches,
	}

	return node, parseErrors
}

func CreateForLoop(
	ast *ml_parser.Block,
	connectedBlocks []*ml_parser.Block,
	visitor ml_parser.Visitor,
	bindingParser *template_parser.BindingParser,
) (*ForLoopBlock, []parse_util.ParseError) {
	var parseErrors []parse_util.ParseError
	params := parseForLoopParameters(ast, &parseErrors, bindingParser)
	var node *ForLoopBlock
	var empty *ForLoopBlockEmpty

	for _, block := range connectedBlocks {
		if block.Name == "empty" {
			if empty != nil {
				parseErrors = append(parseErrors, *parse_util.NewParseError(block.SourceSpan, "@for loop can only have one @empty block", nil, nil))
			} else if len(block.Parameters) > 0 {
				parseErrors = append(parseErrors, *parse_util.NewParseError(block.SourceSpan, "@empty block cannot have parameters", nil, nil))
			} else {
				empty = &ForLoopBlockEmpty{
					BlockNode: BlockNode{
						NameSpan:        *block.StartSourceSpan,
						SourceSpan:      *block.SourceSpan,
						StartSourceSpan: *block.StartSourceSpan,
						EndSourceSpan:   block.EndSourceSpan,
					},
					Children: htmlVisitAll(visitor, block.Children, block.Children),
				}
			}
		} else {
			parseErrors = append(parseErrors, *parse_util.NewParseError(block.SourceSpan, fmt.Sprintf("Unrecognized @for loop block \"%s\"", block.Name), nil, nil))
		}
	}

	if params != nil {
		var endSpan *parse_util.ParseSourceSpan
		if empty != nil {
			endSpan = empty.EndSourceSpan
		} else {
			endSpan = ast.EndSourceSpan
		}

		var sourceSpanEnd *parse_util.ParseLocation
		if endSpan != nil {
			sourceSpanEnd = endSpan.End
		} else {
			sourceSpanEnd = ast.SourceSpan.End
		}

		sourceSpan := parse_util.NewParseSourceSpan(
			ast.SourceSpan.Start,
			sourceSpanEnd,
			nil,
			nil,
		)

		var trackExpression expression_parser.ASTWithSource
		var trackKeywordSpan *parse_util.ParseSourceSpan

		if params.trackBy == nil {
			parseErrors = append(parseErrors, *parse_util.NewParseError(ast.StartSourceSpan, "@for loop must have a \"track\" expression", nil, nil))
		} else {
			trackExpression = params.trackBy.expression
			trackKeywordSpan = params.trackBy.keywordSpan
			validateTrackByExpression(params.trackBy.expression, params.trackBy.keywordSpan, &parseErrors)
		}

		node = &ForLoopBlock{
			BlockNode: BlockNode{
				NameSpan:        *ast.StartSourceSpan,
				SourceSpan:      *sourceSpan,
				StartSourceSpan: *ast.StartSourceSpan,
				EndSourceSpan:   endSpan,
			},
			Item:             params.itemName,
			Expression:       params.expression,
			TrackBy:          &trackExpression,
			TrackKeywordSpan: trackKeywordSpan,
			ContextVariables: params.context,
			Children:         htmlVisitAll(visitor, ast.Children, ast.Children),
			Empty:            empty,
			MainBlockSpan:    *ast.SourceSpan,
		}
	}

	return node, parseErrors
}

func CreateSwitchBlock(
	ast *ml_parser.Block,
	visitor ml_parser.Visitor,
	bindingParser *template_parser.BindingParser,
) (*SwitchBlock, []parse_util.ParseError) {
	var parseErrors []parse_util.ParseError
	validateSwitchBlock(ast, &parseErrors)

	var primaryExpression expression_parser.ASTWithSource
	if len(ast.Parameters) > 0 {
		primaryExpression = parseBlockParameterToBinding(ast.Parameters[0], bindingParser, "")
	} else {
		primaryExpression = bindingParser.ParseBinding("", false, toExpressionSourceSpan(ast.SourceSpan), 0)
	}

	var groups []*SwitchBlockCaseGroup
	var unknownBlocks []*UnknownBlock
	var collectedCases []*SwitchBlockCase
	var firstCaseStart *parse_util.ParseSourceSpan
	var exhaustiveCheck *SwitchExhaustiveCheck

	for _, node := range ast.Children {
		block, ok := node.(*ml_parser.Block)
		if !ok {
			continue
		}

		if (block.Name != "case" || len(block.Parameters) == 0) &&
			block.Name != "default" &&
			block.Name != "default never" {
			unknownBlocks = append(unknownBlocks, &UnknownBlock{
				Name:       block.Name,
				SourceSpan: *block.SourceSpan,
				NameSpan:   *block.SourceSpan,
			})
			continue
		}

		if exhaustiveCheck != nil {
			parseErrors = append(parseErrors, *parse_util.NewParseError(
				block.SourceSpan,
				"@default block with \"never\" parameter must be the last case in a switch",
				nil, nil,
			))
		}

		isCase := block.Name == "case"
		var expression expression_parser.AST

		if isCase {
			expression = parseBlockParameterToBinding(block.Parameters[0], bindingParser, "").Ast
		} else if block.Name == "default never" {
			if len(block.Parameters) > 0 {
				expression = parseBlockParameterToBinding(block.Parameters[0], bindingParser, "").Ast
			}

			if len(block.Children) > 0 || (block.EndSourceSpan != nil && block.EndSourceSpan.Start.Offset != block.EndSourceSpan.End.Offset) {
				parseErrors = append(parseErrors, *parse_util.NewParseError(
					block.SourceSpan,
					"@default block with \"never\" parameter cannot have a body",
					nil, nil,
				))
			}

			if len(collectedCases) > 0 {
				parseErrors = append(parseErrors, *parse_util.NewParseError(
					block.SourceSpan,
					"A @case block with no body cannot be followed by a @default block with \"never\" parameter",
					nil, nil,
				))
			}

			exhaustiveCheck = &SwitchExhaustiveCheck{
				BlockNode: BlockNode{
					NameSpan:        *block.StartSourceSpan,
					SourceSpan:      *block.SourceSpan,
					StartSourceSpan: *block.StartSourceSpan,
					EndSourceSpan:   block.EndSourceSpan,
				},
				Expression: expression,
			}
			continue
		}

		switchCase := &SwitchBlockCase{
			BlockNode: BlockNode{
				NameSpan:        *block.StartSourceSpan,
				SourceSpan:      *block.SourceSpan,
				StartSourceSpan: *block.StartSourceSpan,
				EndSourceSpan:   block.EndSourceSpan,
			},
			Expression: expression,
		}
		collectedCases = append(collectedCases, switchCase)

		caseWithoutBody := len(block.Children) == 0 &&
			block.EndSourceSpan != nil &&
			block.EndSourceSpan.Start.Offset == block.EndSourceSpan.End.Offset

		if caseWithoutBody {
			if firstCaseStart == nil {
				firstCaseStart = block.SourceSpan
			}
			continue
		}

		sourceSpan := *block.SourceSpan
		startSourceSpan := *block.StartSourceSpan
		if firstCaseStart != nil {
			sourceSpan = *parse_util.NewParseSourceSpan(firstCaseStart.Start, block.SourceSpan.End, nil, nil)
			startSourceSpan = *parse_util.NewParseSourceSpan(firstCaseStart.Start, block.StartSourceSpan.End, nil, nil)
			firstCaseStart = nil
		}

		group := &SwitchBlockCaseGroup{
			BlockNode: BlockNode{
				NameSpan:        *block.StartSourceSpan,
				SourceSpan:      sourceSpan,
				StartSourceSpan: startSourceSpan,
				EndSourceSpan:   block.EndSourceSpan,
			},
			Cases:    collectedCases,
			Children: htmlVisitAll(visitor, block.Children, block.Children),
		}
		groups = append(groups, group)
		collectedCases = nil
	}

	nameSpan := parse_util.ParseSourceSpan{}
	if ast.StartSourceSpan != nil {
		nameSpan = *ast.StartSourceSpan
	}

	node := &SwitchBlock{
		BlockNode: BlockNode{
			NameSpan:        nameSpan,
			SourceSpan:      *ast.SourceSpan,
			StartSourceSpan: nameSpan,
			EndSourceSpan:   ast.EndSourceSpan,
		},
		Expression:      primaryExpression.Ast,
		Groups:          groups,
		UnknownBlocks:   unknownBlocks,
		ExhaustiveCheck: exhaustiveCheck,
	}

	return node, parseErrors
}

type conditionalParams struct {
	expression      expression_parser.AST
	expressionAlias *Variable
}

func parseConditionalBlockParameters(
	block *ml_parser.Block,
	errors *[]parse_util.ParseError,
	bindingParser *template_parser.BindingParser,
) *conditionalParams {
	if len(block.Parameters) == 0 {
		*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "Conditional block does not have an expression", nil, nil))
		return nil
	}

	expression := parseBlockParameterToBinding(block.Parameters[0], bindingParser, "")
	var expressionAlias *Variable

	for i := 1; i < len(block.Parameters); i++ {
		param := block.Parameters[i]
		aliasMatch := conditionalAliasPattern.FindStringSubmatch(param.Expression)

		if aliasMatch == nil {
			*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, fmt.Sprintf("Unrecognized conditional parameter \"%s\"", param.Expression), nil, nil))
		} else if block.Name != "if" && !elseIfPattern.MatchString(block.Name) {
			*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, "\"as\" expression is only allowed on `@if` and `@else if` blocks", nil, nil))
		} else if expressionAlias != nil {
			*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, "Conditional can only have one \"as\" expression", nil, nil))
		} else {
			name := strings.TrimSpace(aliasMatch[2])

			if identifierPattern.MatchString(name) {
				variableStart := param.SourceSpan.Start.MoveBy(len(aliasMatch[1]))
				variableSpan := parse_util.NewParseSourceSpan(variableStart, variableStart.MoveBy(len(name)), nil, nil)
				expressionAlias = &Variable{
					Name:       name,
					Value:      name,
					SourceSpan: *variableSpan,
					KeySpan:    *variableSpan,
				}
			} else {
				*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, "\"as\" expression must be a valid JavaScript identifier", nil, nil))
			}
		}
	}

	return &conditionalParams{
		expression:      expression.Ast,
		expressionAlias: expressionAlias,
	}
}

func validateIfConnectedBlocks(connectedBlocks []*ml_parser.Block, errors *[]parse_util.ParseError) {
	hasElse := false
	for i, block := range connectedBlocks {
		if block.Name == "else" {
			if hasElse {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "Conditional can only have one @else block", nil, nil))
			} else if len(connectedBlocks) > 1 && i < len(connectedBlocks)-1 {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@else block must be last inside the conditional", nil, nil))
			} else if len(block.Parameters) > 0 {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@else block cannot have parameters", nil, nil))
			}
			hasElse = true
		} else if !elseIfPattern.MatchString(block.Name) {
			*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, fmt.Sprintf("Unrecognized conditional block @%s", block.Name), nil, nil))
		}
	}
}

func stripOptionalParentheses(param *ml_parser.BlockParameter, errors *[]parse_util.ParseError) string {
	expression := param.Expression
	openParens := 0
	start := 0
	end := len(expression) - 1

	for i := 0; i < len(expression); i++ {
		char := expression[i]
		if char == '(' {
			start = i + 1
			openParens++
		} else if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			continue
		} else {
			break
		}
	}

	if openParens == 0 {
		return expression
	}

	for i := len(expression) - 1; i > -1; i-- {
		char := expression[i]
		if char == ')' {
			end = i
			openParens--
			if openParens == 0 {
				break
			}
		} else if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			continue
		} else {
			break
		}
	}

	if openParens != 0 {
		*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, "Unclosed parentheses in expression", nil, nil))
	}

	return expression[start:end]
}

type forLoopParams struct {
	itemName   *Variable
	trackBy    *trackByInfo
	expression expression_parser.ASTWithSource
	context    []*Variable
}

type trackByInfo struct {
	expression  expression_parser.ASTWithSource
	keywordSpan *parse_util.ParseSourceSpan
}

func parseForLoopParameters(
	block *ml_parser.Block,
	errors *[]parse_util.ParseError,
	bindingParser *template_parser.BindingParser,
) *forLoopParams {
	if len(block.Parameters) == 0 {
		*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@for loop does not have an expression", nil, nil))
		return nil
	}

	expressionParam := block.Parameters[0]
	stripped := stripOptionalParentheses(expressionParam, errors)
	match := forLoopExpressionPattern.FindStringSubmatch(stripped)

	if match == nil || len(strings.TrimSpace(match[2])) == 0 {
		*errors = append(*errors, *parse_util.NewParseError(expressionParam.SourceSpan, "Cannot parse expression. @for loop expression must match the pattern \"<identifier> of <expression>\"", nil, nil))
		return nil
	}

	itemName := match[1]
	rawExpression := match[2]

	if allowedForLoopLetVariables[itemName] {
		*errors = append(*errors, *parse_util.NewParseError(expressionParam.SourceSpan, "@for loop item name cannot be one of $index, $first, $last, $even, $odd, $count.", nil, nil))
	}

	variableName := strings.Split(expressionParam.Expression, " ")[0]
	variableSpan := parse_util.NewParseSourceSpan(
		expressionParam.SourceSpan.Start,
		expressionParam.SourceSpan.Start.MoveBy(len(variableName)),
		nil,
		nil,
	)

	var context []*Variable
	for varName := range allowedForLoopLetVariables {
		emptySpan := parse_util.NewParseSourceSpan(block.StartSourceSpan.End, block.StartSourceSpan.End, nil, nil)
		context = append(context, &Variable{
			Name:       varName,
			Value:      varName,
			SourceSpan: *emptySpan,
			KeySpan:    *emptySpan,
		})
	}

	orderMap := map[string]int{
		"$index": 0,
		"$first": 1,
		"$last":  2,
		"$even":  3,
		"$odd":   4,
		"$count": 5,
	}
	sort.Slice(context, func(i, j int) bool {
		return orderMap[context[i].Name] < orderMap[context[j].Name]
	})

	result := &forLoopParams{
		itemName:   &Variable{Name: itemName, Value: "$implicit", SourceSpan: *variableSpan, KeySpan: *variableSpan},
		expression: parseBlockParameterToBinding(expressionParam, bindingParser, rawExpression),
		context:    context,
	}

	for i := 1; i < len(block.Parameters); i++ {
		param := block.Parameters[i]
		letMatch := forLoopLetPattern.FindStringSubmatch(param.Expression)

		if letMatch != nil {
			variablesSpan := parse_util.NewParseSourceSpan(
				param.SourceSpan.Start.MoveBy(len(letMatch[0]) - len(letMatch[1])),
				param.SourceSpan.End,
				nil,
				nil,
			)
			parseLetParameter(
				param.SourceSpan,
				letMatch[1],
				variablesSpan,
				itemName,
				&result.context,
				errors,
			)
			continue
		}

		trackMatch := forLoopTrackPattern.FindStringSubmatch(param.Expression)

		if trackMatch != nil {
			if result.trackBy != nil {
				*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, "@for loop can only have one \"track\" expression", nil, nil))
			} else {
				expression := parseBlockParameterToBinding(param, bindingParser, trackMatch[1])
				if _, ok := expression.Ast.(*expression_parser.EmptyExpr); ok {
					*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@for loop must have a \"track\" expression", nil, nil))
				}
				keywordSpan := parse_util.NewParseSourceSpan(
					param.SourceSpan.Start,
					param.SourceSpan.Start.MoveBy(len("track")),
					nil,
					nil,
				)
				result.trackBy = &trackByInfo{expression: expression, keywordSpan: keywordSpan}
			}
			continue
		}

		*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, fmt.Sprintf("Unrecognized @for loop parameter \"%s\"", param.Expression), nil, nil))
	}

	return result
}

type pipeChecker struct {
	expression_parser.RecursiveAstVisitor
	hasPipe bool
}

func (c *pipeChecker) VisitPipe(ast *expression_parser.BindingPipe, context any) any {
	c.hasPipe = true
	return c.RecursiveAstVisitor.VisitPipe(ast, context)
}

func validateTrackByExpression(
	expression expression_parser.ASTWithSource,
	parseSourceSpan *parse_util.ParseSourceSpan,
	errors *[]parse_util.ParseError,
) {
	checker := &pipeChecker{}
	checker.Impl = checker
	checker.Visit(expression.Ast, nil)
	if checker.hasPipe {
		*errors = append(*errors, *parse_util.NewParseError(parseSourceSpan, "Cannot use pipes in track expressions", nil, nil))
	}
}

func parseLetParameter(
	sourceSpan *parse_util.ParseSourceSpan,
	expression string,
	span *parse_util.ParseSourceSpan,
	loopItemName string,
	context *[]*Variable,
	errors *[]parse_util.ParseError,
) {
	parts := strings.Split(expression, ",")
	startSpan := span.Start

	for _, part := range parts {
		expressionParts := strings.Split(part, "=")
		name := ""
		variableName := ""
		if len(expressionParts) == 2 {
			name = strings.TrimSpace(expressionParts[0])
			variableName = strings.TrimSpace(expressionParts[1])
		}

		if len(name) == 0 || len(variableName) == 0 {
			*errors = append(*errors, *parse_util.NewParseError(sourceSpan, `Invalid @for loop "let" parameter. Parameter should match the pattern "<name> = <variable name>"`, nil, nil))
		} else if !allowedForLoopLetVariables[variableName] {
			var allowed []string
			for k := range allowedForLoopLetVariables {
				allowed = append(allowed, k)
			}
			sort.Strings(allowed)
			*errors = append(*errors, *parse_util.NewParseError(sourceSpan, fmt.Sprintf(`Unknown "let" parameter variable "%s". The allowed variables are: %s`, variableName, strings.Join(allowed, ", ")), nil, nil))
		} else if name == loopItemName {
			*errors = append(*errors, *parse_util.NewParseError(sourceSpan, fmt.Sprintf(`Invalid @for loop "let" parameter. Variable cannot be called "%s"`, loopItemName), nil, nil))
		} else {
			hasDuplicate := false
			for _, v := range *context {
				if v.Name == name {
					hasDuplicate = true
					break
				}
			}
			if hasDuplicate {
				*errors = append(*errors, *parse_util.NewParseError(sourceSpan, fmt.Sprintf(`Duplicate "let" parameter variable "%s"`, variableName), nil, nil))
			} else {
				var keyLeadingWhitespace, keyName string
				matchKey := charactersInSurroundingWhitespacePattern.FindStringSubmatch(expressionParts[0])
				if len(matchKey) == 4 {
					keyLeadingWhitespace = matchKey[1]
					keyName = matchKey[2]
				}

				keySpan := span
				if keyLeadingWhitespace != "" && len(expressionParts) == 2 {
					keySpan = parse_util.NewParseSourceSpan(
						startSpan.MoveBy(len(keyLeadingWhitespace)),
						startSpan.MoveBy(len(keyLeadingWhitespace)+len(keyName)),
						nil,
						nil,
					)
				}

				var valueSpan *parse_util.ParseSourceSpan
				if len(expressionParts) == 2 {
					var valueLeadingWhitespace, implicit string
					matchValue := charactersInSurroundingWhitespacePattern.FindStringSubmatch(expressionParts[1])
					if len(matchValue) == 4 {
						valueLeadingWhitespace = matchValue[1]
						implicit = matchValue[2]
					}
					if valueLeadingWhitespace != "" {
						valueSpan = parse_util.NewParseSourceSpan(
							startSpan.MoveBy(len(expressionParts[0])+1+len(valueLeadingWhitespace)),
							startSpan.MoveBy(len(expressionParts[0])+1+len(valueLeadingWhitespace)+len(implicit)),
							nil,
							nil,
						)
					}
				}

				var sourceSpanEnd *parse_util.ParseLocation
				if valueSpan != nil {
					sourceSpanEnd = valueSpan.End
				} else {
					sourceSpanEnd = keySpan.End
				}

				combinedSpan := parse_util.NewParseSourceSpan(keySpan.Start, sourceSpanEnd, nil, nil)
				*context = append(*context, &Variable{
					Name:       name,
					Value:      variableName,
					SourceSpan: *combinedSpan,
					KeySpan:    *keySpan,
					ValueSpan:  valueSpan,
				})
			}
		}

		startSpan = startSpan.MoveBy(len(part) + 1)
	}
}

func validateSwitchBlock(ast *ml_parser.Block, errors *[]parse_util.ParseError) {
	hasDefault := false

	if len(ast.Parameters) != 1 {
		*errors = append(*errors, *parse_util.NewParseError(ast.StartSourceSpan, "@switch block must have exactly one parameter", nil, nil))
		return
	}

	for _, node := range ast.Children {
		if _, isComment := node.(*ml_parser.Comment); isComment {
			continue
		}
		if text, isText := node.(*ml_parser.Text); isText && len(strings.TrimSpace(text.Value)) == 0 {
			continue
		}

		block, isBlock := node.(*ml_parser.Block)
		if !isBlock || (block.Name != "case" && block.Name != "default" && block.Name != "default never") {
			*errors = append(*errors, *parse_util.NewParseError(node.GetSourceSpan(), "@switch block can only contain @case and @default blocks", nil, nil))
			continue
		}

		if block.Name == "default never" {
			if hasDefault {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@switch block can only have one @default block", nil, nil))
			}
			hasDefault = true
		} else if block.Name == "default" {
			if hasDefault {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@switch block can only have one @default block", nil, nil))
			} else if len(block.Parameters) > 0 {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@default block cannot have parameters", nil, nil))
			}
			hasDefault = true
		} else if block.Name == "case" && len(block.Parameters) != 1 {
			*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@case block must have exactly one parameter", nil, nil))
		}
	}
}

func parseBlockParameterToBinding(
	ast *ml_parser.BlockParameter,
	bindingParser *template_parser.BindingParser,
	part string,
) expression_parser.ASTWithSource {
	start := 0
	end := len(ast.Expression)

	if part != "" {
		start = strings.LastIndex(ast.Expression, part)
		if start < 0 {
			start = 0
		}
		end = start + len(part)
	}

	exprSlice := ast.Expression[start:end]
	exprSpan := toExpressionSourceSpan(ast.SourceSpan)
	
	absOffset := 0
	if ast.SourceSpan != nil && ast.SourceSpan.Start != nil {
		absOffset = ast.SourceSpan.Start.Offset
	}

	return bindingParser.ParseBinding(
		exprSlice,
		false,
		exprSpan,
		absOffset + start,
	)
}

func toExpressionSourceSpan(span *parse_util.ParseSourceSpan) expression_parser.ParseSourceSpan {
	if span == nil || span.Start == nil || span.End == nil {
		return expression_parser.ParseSourceSpan{}
	}
	return expression_parser.ParseSourceSpan{
		Start: span.Start.Offset,
		End:   span.End.Offset,
	}
}
