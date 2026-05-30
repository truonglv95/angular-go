package render3

import (
	"fmt"
	"regexp"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

var (
	prefetchWhenPattern     = regexp.MustCompile(`^prefetch\s+when\s`)
	prefetchOnPattern       = regexp.MustCompile(`^prefetch\s+on\s`)
	hydrateWhenPattern      = regexp.MustCompile(`^hydrate\s+when\s`)
	hydrateOnPattern        = regexp.MustCompile(`^hydrate\s+on\s`)
	hydrateNeverPattern     = regexp.MustCompile(`^hydrate\s+never(\s*)$`)
	minimumParameterPattern = regexp.MustCompile(`^minimum\s`)
	afterParameterPattern   = regexp.MustCompile(`^after\s`)
	whenParameterPattern    = regexp.MustCompile(`^when\s`)
	onParameterPattern      = regexp.MustCompile(`^on\s`)
)

func IsConnectedDeferLoopBlock(name string) bool {
	return name == "placeholder" || name == "loading" || name == "error"
}

func htmlVisitAll(visitor ml_parser.Visitor, nodes []ml_parser.Node, context any) []Node {
	var result []Node
	for _, node := range nodes {
		res := node.Visit(visitor, context)
		if res != nil {
			if n, ok := res.(Node); ok {
				result = append(result, n)
			} else if ns, ok := res.([]Node); ok {
				result = append(result, ns...)
			}
		}
	}
	return result
}

func CreateDeferredBlock(ast *ml_parser.Block, connectedBlocks []*ml_parser.Block, visitor ml_parser.Visitor, bindingParser *template_parser.BindingParser) (*DeferredBlock, []parse_util.ParseError) {
	errors := []parse_util.ParseError{}
	placeholder, loading, errorBlock := parseConnectedBlocks(connectedBlocks, &errors, visitor)

	triggers, prefetchTriggers, hydrateTriggers := parsePrimaryTriggers(ast, bindingParser, &errors, placeholder)

	lastEndSourceSpan := ast.EndSourceSpan
	endOfLastSourceSpan := ast.SourceSpan.End

	if len(connectedBlocks) > 0 {
		lastConnectedBlock := connectedBlocks[len(connectedBlocks)-1]
		lastEndSourceSpan = lastConnectedBlock.EndSourceSpan
		endOfLastSourceSpan = lastConnectedBlock.SourceSpan.End
	}

	sourceSpanWithConnectedBlocks := parse_util.NewParseSourceSpan(
		ast.SourceSpan.Start,
		endOfLastSourceSpan,
		nil,
		nil,
	)

	var i18nMeta i18n.I18nMeta
	if ast.I18n != nil {
		// Just cast safely or ignore if it cannot be cast directly to struct
		// In Go it might be a struct or interface depending on i18nMeta. Here we assume we can just cast or ignore it.
	}

	startSourceSpan := *ast.StartSourceSpan
	if startSourceSpan.Start == nil {
		startSourceSpan = *ast.SourceSpan
	}

	node := &DeferredBlock{
		BlockNode: BlockNode{
			NameSpan:        startSourceSpan,
			SourceSpan:      *sourceSpanWithConnectedBlocks,
			StartSourceSpan: startSourceSpan,
			EndSourceSpan:   lastEndSourceSpan,
		},
		Children:         htmlVisitAll(visitor, ast.Children, ast.Children),
		Triggers:         triggers,
		PrefetchTriggers: prefetchTriggers,
		HydrateTriggers:  hydrateTriggers,
		Placeholder:      placeholder,
		Loading:          loading,
		Error:            errorBlock,
		MainBlockSpan:    *ast.SourceSpan,
		I18n:             i18nMeta,
	}

	return node, errors
}

func parseConnectedBlocks(
	connectedBlocks []*ml_parser.Block,
	errors *[]parse_util.ParseError,
	visitor ml_parser.Visitor,
) (*DeferredBlockPlaceholder, *DeferredBlockLoading, *DeferredBlockError) {
	var placeholder *DeferredBlockPlaceholder
	var loading *DeferredBlockLoading
	var errorBlock *DeferredBlockError

	for _, block := range connectedBlocks {
		if !IsConnectedDeferLoopBlock(block.Name) {
			*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, fmt.Sprintf("Unrecognized block \"@%s\"", block.Name), nil, nil))
			break
		}

		switch block.Name {
		case "placeholder":
			if placeholder != nil {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@defer block can only have one @placeholder block", nil, nil))
			} else {
				p, err := parsePlaceholderBlock(block, visitor)
				if err != nil {
					*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, err.Error(), nil, nil))
				} else {
					placeholder = p
				}
			}
		case "loading":
			if loading != nil {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@defer block can only have one @loading block", nil, nil))
			} else {
				l, err := parseLoadingBlock(block, visitor)
				if err != nil {
					*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, err.Error(), nil, nil))
				} else {
					loading = l
				}
			}
		case "error":
			if errorBlock != nil {
				*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, "@defer block can only have one @error block", nil, nil))
			} else {
				e, err := parseErrorBlock(block, visitor)
				if err != nil {
					*errors = append(*errors, *parse_util.NewParseError(block.StartSourceSpan, err.Error(), nil, nil))
				} else {
					errorBlock = e
				}
			}
		}
	}

	return placeholder, loading, errorBlock
}

func parsePlaceholderBlock(ast *ml_parser.Block, visitor ml_parser.Visitor) (*DeferredBlockPlaceholder, error) {
	var minimumTime *float64

	for _, param := range ast.Parameters {
		if minimumParameterPattern.MatchString(param.Expression) {
			if minimumTime != nil {
				return nil, fmt.Errorf("@placeholder block can only have one \"minimum\" parameter")
			}

			parsedTime := ParseDeferredTime(param.Expression[GetTriggerParametersStart(param.Expression, 0):])

			if parsedTime == nil {
				return nil, fmt.Errorf("Could not parse time value of parameter \"minimum\"")
			}

			minimumTime = parsedTime
		} else {
			return nil, fmt.Errorf("Unrecognized parameter in @placeholder block: \"%s\"", param.Expression)
		}
	}

	var i18nMeta i18n.I18nMeta

	return &DeferredBlockPlaceholder{
		BlockNode: BlockNode{
			NameSpan:        *ast.StartSourceSpan,
			SourceSpan:      *ast.SourceSpan,
			StartSourceSpan: *ast.StartSourceSpan,
			EndSourceSpan:   ast.EndSourceSpan,
		},
		Children:    htmlVisitAll(visitor, ast.Children, ast.Children),
		MinimumTime: minimumTime,
		I18n:        i18nMeta,
	}, nil
}

func parseLoadingBlock(ast *ml_parser.Block, visitor ml_parser.Visitor) (*DeferredBlockLoading, error) {
	var afterTime *float64
	var minimumTime *float64

	for _, param := range ast.Parameters {
		if afterParameterPattern.MatchString(param.Expression) {
			if afterTime != nil {
				return nil, fmt.Errorf("@loading block can only have one \"after\" parameter")
			}

			parsedTime := ParseDeferredTime(param.Expression[GetTriggerParametersStart(param.Expression, 0):])

			if parsedTime == nil {
				return nil, fmt.Errorf("Could not parse time value of parameter \"after\"")
			}

			afterTime = parsedTime
		} else if minimumParameterPattern.MatchString(param.Expression) {
			if minimumTime != nil {
				return nil, fmt.Errorf("@loading block can only have one \"minimum\" parameter")
			}

			parsedTime := ParseDeferredTime(param.Expression[GetTriggerParametersStart(param.Expression, 0):])

			if parsedTime == nil {
				return nil, fmt.Errorf("Could not parse time value of parameter \"minimum\"")
			}

			minimumTime = parsedTime
		} else {
			return nil, fmt.Errorf("Unrecognized parameter in @loading block: \"%s\"", param.Expression)
		}
	}

	var i18nMeta i18n.I18nMeta

	return &DeferredBlockLoading{
		BlockNode: BlockNode{
			NameSpan:        *ast.StartSourceSpan,
			SourceSpan:      *ast.SourceSpan,
			StartSourceSpan: *ast.StartSourceSpan,
			EndSourceSpan:   ast.EndSourceSpan,
		},
		Children:    htmlVisitAll(visitor, ast.Children, ast.Children),
		AfterTime:   afterTime,
		MinimumTime: minimumTime,
		I18n:        i18nMeta,
	}, nil
}

func parseErrorBlock(ast *ml_parser.Block, visitor ml_parser.Visitor) (*DeferredBlockError, error) {
	if len(ast.Parameters) > 0 {
		return nil, fmt.Errorf("@error block cannot have parameters")
	}

	var i18nMeta i18n.I18nMeta

	return &DeferredBlockError{
		BlockNode: BlockNode{
			NameSpan:        *ast.StartSourceSpan,
			SourceSpan:      *ast.SourceSpan,
			StartSourceSpan: *ast.StartSourceSpan,
			EndSourceSpan:   ast.EndSourceSpan,
		},
		Children: htmlVisitAll(visitor, ast.Children, ast.Children),
		I18n:     i18nMeta,
	}, nil
}

func parsePrimaryTriggers(
	ast *ml_parser.Block,
	bindingParser *template_parser.BindingParser,
	errors *[]parse_util.ParseError,
	placeholder *DeferredBlockPlaceholder,
) (DeferredBlockTriggers, DeferredBlockTriggers, DeferredBlockTriggers) {
	triggers := DeferredBlockTriggers{}
	prefetchTriggers := DeferredBlockTriggers{}
	hydrateTriggers := DeferredBlockTriggers{}

	for _, param := range ast.Parameters {
		if whenParameterPattern.MatchString(param.Expression) {
			ParseWhenTrigger(param, bindingParser, &triggers, errors)
		} else if onParameterPattern.MatchString(param.Expression) {
			ParseOnTrigger(param, bindingParser, &triggers, errors, placeholder)
		} else if prefetchWhenPattern.MatchString(param.Expression) {
			ParseWhenTrigger(param, bindingParser, &prefetchTriggers, errors)
		} else if prefetchOnPattern.MatchString(param.Expression) {
			ParseOnTrigger(param, bindingParser, &prefetchTriggers, errors, placeholder)
		} else if hydrateWhenPattern.MatchString(param.Expression) {
			ParseWhenTrigger(param, bindingParser, &hydrateTriggers, errors)
		} else if hydrateOnPattern.MatchString(param.Expression) {
			ParseOnTrigger(param, bindingParser, &hydrateTriggers, errors, placeholder)
		} else if hydrateNeverPattern.MatchString(param.Expression) {
			ParseNeverTrigger(param, &hydrateTriggers, errors)
		} else {
			*errors = append(*errors, *parse_util.NewParseError(param.SourceSpan, "Unrecognized trigger", nil, nil))
		}
	}

	if hydrateTriggers.Never != nil && countHydrateTriggers(hydrateTriggers) > 1 {
		*errors = append(*errors, *parse_util.NewParseError(ast.StartSourceSpan, "Cannot specify additional `hydrate` triggers if `hydrate never` is present", nil, nil))
	}

	return triggers, prefetchTriggers, hydrateTriggers
}

func countHydrateTriggers(triggers DeferredBlockTriggers) int {
	count := 0
	if triggers.When != nil {
		count++
	}
	if triggers.Idle != nil {
		count++
	}
	if triggers.Immediate != nil {
		count++
	}
	if triggers.Hover != nil {
		count++
	}
	if triggers.Timer != nil {
		count++
	}
	if triggers.Interaction != nil {
		count++
	}
	if triggers.Viewport != nil {
		count++
	}
	if triggers.Never != nil {
		count++
	}
	return count
}
