package render3

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/chars"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/microsoft/typescript-go/angular-packages/compiler/parse_util"
	"github.com/microsoft/typescript-go/angular-packages/compiler/template_parser"
)

var timePattern = regexp.MustCompile(`^\d+\.?\d*(ms|s)?$`)
var separatorPattern = regexp.MustCompile(`^\s$`)

var commaDelimitedSyntax = map[int]int{
	chars.LBRACE:   chars.RBRACE,   // Object literals
	chars.LBRACKET: chars.RBRACKET, // Array literals
	chars.LPAREN:   chars.RPAREN,   // Function calls
}

type onTriggerType string

const (
	onTriggerTypeIdle        onTriggerType = "idle"
	onTriggerTypeTimer       onTriggerType = "timer"
	onTriggerTypeInteraction onTriggerType = "interaction"
	onTriggerTypeImmediate   onTriggerType = "immediate"
	onTriggerTypeHover       onTriggerType = "hover"
	onTriggerTypeViewport    onTriggerType = "viewport"
	onTriggerTypeNever       onTriggerType = "never"
)

type referenceTriggerValidator func(triggerType onTriggerType, parameters []parsedParameter) error

type parsedParameter struct {
	expression string
	start      int
}

func ParseNeverTrigger(
	parameter *ml_parser.BlockParameter,
	triggers *DeferredBlockTriggers,
	errorsList *[]parse_util.ParseError,
) {
	expression := parameter.Expression
	sourceSpan := parameter.SourceSpan

	neverIndex := strings.Index(expression, "never")
	if neverIndex == -1 {
		*errorsList = append(*errorsList, *parse_util.NewParseError(sourceSpan, `Could not find "never" keyword in expression`, nil, nil))
		return
	}

	neverSourceSpan := parse_util.NewParseSourceSpan(
		sourceSpan.Start.MoveBy(neverIndex),
		sourceSpan.Start.MoveBy(neverIndex+len("never")),
		nil,
		nil,
	)
	prefetchSpan := getPrefetchSpan(expression, sourceSpan)
	hydrateSpan := getHydrateSpan(expression, sourceSpan)

	trackTrigger(
		"never",
		triggers,
		errorsList,
		&NeverDeferredTrigger{
			BaseDeferredTrigger: BaseDeferredTrigger{
				NameSpan:           neverSourceSpan,
				SourceSpan:         *sourceSpan,
				PrefetchSpan:       prefetchSpan,
				WhenOrOnSourceSpan: nil,
				HydrateSpan:        hydrateSpan,
			},
		},
	)
}

func ParseWhenTrigger(
	parameter *ml_parser.BlockParameter,
	bindingParser *template_parser.BindingParser,
	triggers *DeferredBlockTriggers,
	errorsList *[]parse_util.ParseError,
) {
	expression := parameter.Expression
	sourceSpan := parameter.SourceSpan

	whenIndex := strings.Index(expression, "when")
	if whenIndex == -1 {
		*errorsList = append(*errorsList, *parse_util.NewParseError(sourceSpan, `Could not find "when" keyword in expression`, nil, nil))
		return
	}

	whenSourceSpan := parse_util.NewParseSourceSpan(
		sourceSpan.Start.MoveBy(whenIndex),
		sourceSpan.Start.MoveBy(whenIndex+len("when")),
		nil,
		nil,
	)
	prefetchSpan := getPrefetchSpan(expression, sourceSpan)
	hydrateSpan := getHydrateSpan(expression, sourceSpan)

	start := GetTriggerParametersStart(expression, whenIndex+1)
	parsed := bindingParser.ParseBinding(
		expression[start:],
		false,
		expression_parser.ParseSourceSpan{},
		sourceSpan.Start.Offset+start,
	)

	trackTrigger(
		"when",
		triggers,
		errorsList,
		NewBoundDeferredTrigger(&parsed, *sourceSpan, prefetchSpan, *whenSourceSpan, hydrateSpan),
	)
}

func ParseOnTrigger(
	parameter *ml_parser.BlockParameter,
	bindingParser *template_parser.BindingParser,
	triggers *DeferredBlockTriggers,
	errorsList *[]parse_util.ParseError,
	placeholder *DeferredBlockPlaceholder,
) {
	expression := parameter.Expression
	sourceSpan := parameter.SourceSpan

	onIndex := strings.Index(expression, "on")
	if onIndex == -1 {
		*errorsList = append(*errorsList, *parse_util.NewParseError(sourceSpan, `Could not find "on" keyword in expression`, nil, nil))
		return
	}

	onSourceSpan := parse_util.NewParseSourceSpan(
		sourceSpan.Start.MoveBy(onIndex),
		sourceSpan.Start.MoveBy(onIndex+len("on")),
		nil,
		nil,
	)
	prefetchSpan := getPrefetchSpan(expression, sourceSpan)
	hydrateSpan := getHydrateSpan(expression, sourceSpan)

	start := GetTriggerParametersStart(expression, onIndex+1)
	isHydrationTrigger := strings.HasPrefix(expression, "hydrate")

	validator := validatePlainReferenceBasedTrigger
	if isHydrationTrigger {
		validator = validateHydrateReferenceBasedTrigger
	}

	parser := newOnTriggerParser(
		expression,
		bindingParser,
		start,
		sourceSpan,
		triggers,
		errorsList,
		validator,
		isHydrationTrigger,
		prefetchSpan,
		onSourceSpan,
		hydrateSpan,
	)
	parser.parse()
}

func getPrefetchSpan(expression string, sourceSpan *parse_util.ParseSourceSpan) *parse_util.ParseSourceSpan {
	if !strings.HasPrefix(expression, "prefetch") {
		return nil
	}
	return parse_util.NewParseSourceSpan(sourceSpan.Start, sourceSpan.Start.MoveBy(len("prefetch")), nil, nil)
}

func getHydrateSpan(expression string, sourceSpan *parse_util.ParseSourceSpan) *parse_util.ParseSourceSpan {
	if !strings.HasPrefix(expression, "hydrate") {
		return nil
	}
	return parse_util.NewParseSourceSpan(sourceSpan.Start, sourceSpan.Start.MoveBy(len("hydrate")), nil, nil)
}

type onTriggerParser struct {
	index              int
	tokens             []expression_parser.Token
	expression         string
	bindingParser      *template_parser.BindingParser
	start              int
	span               *parse_util.ParseSourceSpan
	triggers           *DeferredBlockTriggers
	errorsList         *[]parse_util.ParseError
	validator          referenceTriggerValidator
	isHydrationTrigger bool
	prefetchSpan       *parse_util.ParseSourceSpan
	onSourceSpan       *parse_util.ParseSourceSpan
	hydrateSpan        *parse_util.ParseSourceSpan
}

func newOnTriggerParser(
	expression string,
	bindingParser *template_parser.BindingParser,
	start int,
	span *parse_util.ParseSourceSpan,
	triggers *DeferredBlockTriggers,
	errorsList *[]parse_util.ParseError,
	validator referenceTriggerValidator,
	isHydrationTrigger bool,
	prefetchSpan *parse_util.ParseSourceSpan,
	onSourceSpan *parse_util.ParseSourceSpan,
	hydrateSpan *parse_util.ParseSourceSpan,
) *onTriggerParser {
	lexer := &expression_parser.Lexer{}
	tokens := lexer.Tokenize(expression[start:])
	return &onTriggerParser{
		index:              0,
		tokens:             tokens,
		expression:         expression,
		bindingParser:      bindingParser,
		start:              start,
		span:               span,
		triggers:           triggers,
		errorsList:         errorsList,
		validator:          validator,
		isHydrationTrigger: isHydrationTrigger,
		prefetchSpan:       prefetchSpan,
		onSourceSpan:       onSourceSpan,
		hydrateSpan:        hydrateSpan,
	}
}

func (p *onTriggerParser) parse() {
	for len(p.tokens) > 0 && p.index < len(p.tokens) {
		token := p.token()

		if !token.IsIdentifier() {
			p.unexpectedToken(token)
			break
		}

		if p.isFollowedByOrLast(chars.COMMA) {
			p.consumeTrigger(token, []parsedParameter{})
			p.advance()
		} else if p.isFollowedByOrLast(chars.LPAREN) {
			p.advance()
			prevErrors := len(*p.errorsList)
			parameters := p.consumeParameters()
			if len(*p.errorsList) != prevErrors {
				break
			}
			p.consumeTrigger(token, parameters)
			p.advance()
		} else if p.index < len(p.tokens)-1 {
			p.unexpectedToken(p.tokens[p.index+1])
		}

		p.advance()
	}
}

func (p *onTriggerParser) advance() {
	p.index++
}

func (p *onTriggerParser) isFollowedByOrLast(char int) bool {
	if p.index == len(p.tokens)-1 {
		return true
	}
	return p.tokens[p.index+1].IsCharacter(char)
}

func (p *onTriggerParser) token() expression_parser.Token {
	idx := p.index
	if idx >= len(p.tokens) {
		idx = len(p.tokens) - 1
	}
	return p.tokens[idx]
}

func (p *onTriggerParser) consumeTrigger(identifier expression_parser.Token, parameters []parsedParameter) {
	triggerNameStartSpan := p.span.Start.MoveBy(p.start + identifier.Index - p.tokens[0].Index)
	nameSpan := parse_util.NewParseSourceSpan(
		triggerNameStartSpan,
		triggerNameStartSpan.MoveBy(len(identifier.StrValue)),
		nil,
		nil,
	)
	endSpan := triggerNameStartSpan.MoveBy(p.token().End - identifier.Index)

	isFirstTrigger := identifier.Index == 0
	var onSourceSpan, prefetchSourceSpan, hydrateSourceSpan *parse_util.ParseSourceSpan
	if isFirstTrigger {
		onSourceSpan = p.onSourceSpan
		prefetchSourceSpan = p.prefetchSpan
		hydrateSourceSpan = p.hydrateSpan
	}

	startLoc := triggerNameStartSpan
	if isFirstTrigger {
		startLoc = p.span.Start
	}
	sourceSpan := parse_util.NewParseSourceSpan(startLoc, endSpan, nil, nil)

	var err error
	switch onTriggerType(identifier.StrValue) {
	case onTriggerTypeIdle:
		trigger, e := createIdleTrigger(parameters, nameSpan, sourceSpan, prefetchSourceSpan, onSourceSpan, hydrateSourceSpan)
		err = e
		if err == nil {
			p.trackTrigger("idle", trigger)
		}
	case onTriggerTypeTimer:
		trigger, e := createTimerTrigger(parameters, nameSpan, sourceSpan, p.prefetchSpan, p.onSourceSpan, p.hydrateSpan)
		err = e
		if err == nil {
			p.trackTrigger("timer", trigger)
		}
	case onTriggerTypeInteraction:
		trigger, e := createInteractionTrigger(parameters, nameSpan, sourceSpan, p.prefetchSpan, p.onSourceSpan, p.hydrateSpan, p.validator)
		err = e
		if err == nil {
			p.trackTrigger("interaction", trigger)
		}
	case onTriggerTypeImmediate:
		trigger, e := createImmediateTrigger(parameters, nameSpan, sourceSpan, p.prefetchSpan, p.onSourceSpan, p.hydrateSpan)
		err = e
		if err == nil {
			p.trackTrigger("immediate", trigger)
		}
	case onTriggerTypeHover:
		trigger, e := createHoverTrigger(parameters, nameSpan, sourceSpan, p.prefetchSpan, p.onSourceSpan, p.hydrateSpan, p.validator)
		err = e
		if err == nil {
			p.trackTrigger("hover", trigger)
		}
	case onTriggerTypeViewport:
		trigger, e := createViewportTrigger(p.start, p.isHydrationTrigger, p.bindingParser, parameters, nameSpan, sourceSpan, p.prefetchSpan, p.onSourceSpan, p.hydrateSpan, p.validator)
		err = e
		if err == nil {
			p.trackTrigger("viewport", trigger)
		}
	default:
		err = fmt.Errorf("Unrecognized trigger type \"%s\"", identifier.StrValue)
	}

	if err != nil {
		p.error(identifier, err.Error())
	}
}

func (p *onTriggerParser) consumeParameters() []parsedParameter {
	parameters := []parsedParameter{}

	if !p.token().IsCharacter(chars.LPAREN) {
		p.unexpectedToken(p.token())
		return parameters
	}

	p.advance()

	commaDelimStack := []int{}
	tokens := []expression_parser.Token{}

	for p.index < len(p.tokens) {
		token := p.token()

		if token.IsCharacter(chars.RPAREN) && len(commaDelimStack) == 0 {
			if len(tokens) > 0 {
				parameters = append(parameters, parsedParameter{expression: p.tokenRangeText(tokens), start: tokens[0].Index})
			}
			break
		}

		if token.Type == expression_parser.TokenTypeCharacter {
			if endChar, ok := commaDelimitedSyntax[int(token.NumValue)]; ok {
				commaDelimStack = append(commaDelimStack, endChar)
			}
		}

		if len(commaDelimStack) > 0 && token.IsCharacter(commaDelimStack[len(commaDelimStack)-1]) {
			commaDelimStack = commaDelimStack[:len(commaDelimStack)-1]
		}

		if len(commaDelimStack) == 0 && token.IsCharacter(chars.COMMA) && len(tokens) > 0 {
			parameters = append(parameters, parsedParameter{expression: p.tokenRangeText(tokens), start: tokens[0].Index})
			p.advance()
			tokens = []expression_parser.Token{}
			continue
		}

		tokens = append(tokens, token)
		p.advance()
	}

	if !p.token().IsCharacter(chars.RPAREN) || len(commaDelimStack) > 0 {
		p.error(p.token(), "Unexpected end of expression")
	}

	if p.index < len(p.tokens)-1 && !p.tokens[p.index+1].IsCharacter(chars.COMMA) {
		p.unexpectedToken(p.tokens[p.index+1])
	}

	return parameters
}

func (p *onTriggerParser) tokenRangeText(tokens []expression_parser.Token) string {
	if len(tokens) == 0 {
		return ""
	}
	return p.expression[p.start+tokens[0].Index : p.start+tokens[len(tokens)-1].End]
}

func (p *onTriggerParser) trackTrigger(name string, trigger DeferredTrigger) {
	trackTrigger(name, p.triggers, p.errorsList, trigger)
}

func (p *onTriggerParser) error(token expression_parser.Token, message string) {
	newStart := p.span.Start.MoveBy(p.start + token.Index)
	newEnd := newStart.MoveBy(token.End - token.Index)
	*p.errorsList = append(*p.errorsList, *parse_util.NewParseError(parse_util.NewParseSourceSpan(newStart, newEnd, nil, nil), message, nil, nil))
}

func (p *onTriggerParser) unexpectedToken(token expression_parser.Token) {
	str := ""
	if token.ToString() != nil {
		str = *token.ToString()
	}
	p.error(token, fmt.Sprintf("Unexpected token \"%s\"", str))
}

func trackTrigger(
	name string,
	allTriggers *DeferredBlockTriggers,
	errorsList *[]parse_util.ParseError,
	trigger DeferredTrigger,
) {
	hasDuplicate := false
	switch name {
	case "when":
		if allTriggers.When != nil {
			hasDuplicate = true
		} else {
			allTriggers.When = trigger.(*BoundDeferredTrigger)
		}
	case "idle":
		if allTriggers.Idle != nil {
			hasDuplicate = true
		} else {
			allTriggers.Idle = trigger.(*IdleDeferredTrigger)
		}
	case "immediate":
		if allTriggers.Immediate != nil {
			hasDuplicate = true
		} else {
			allTriggers.Immediate = trigger.(*ImmediateDeferredTrigger)
		}
	case "hover":
		if allTriggers.Hover != nil {
			hasDuplicate = true
		} else {
			allTriggers.Hover = trigger.(*HoverDeferredTrigger)
		}
	case "timer":
		if allTriggers.Timer != nil {
			hasDuplicate = true
		} else {
			allTriggers.Timer = trigger.(*TimerDeferredTrigger)
		}
	case "interaction":
		if allTriggers.Interaction != nil {
			hasDuplicate = true
		} else {
			allTriggers.Interaction = trigger.(*InteractionDeferredTrigger)
		}
	case "viewport":
		if allTriggers.Viewport != nil {
			hasDuplicate = true
		} else {
			allTriggers.Viewport = trigger.(*ViewportDeferredTrigger)
		}
	case "never":
		if allTriggers.Never != nil {
			hasDuplicate = true
		} else {
			allTriggers.Never = trigger.(*NeverDeferredTrigger)
		}
	}

	if hasDuplicate {
		span := trigger.GetSourceSpan()
		*errorsList = append(*errorsList, *parse_util.NewParseError(&span, fmt.Sprintf("Duplicate \"%s\" trigger is not allowed", name), nil, nil))
	}
}

func createIdleTrigger(
	parameters []parsedParameter,
	nameSpan *parse_util.ParseSourceSpan,
	sourceSpan *parse_util.ParseSourceSpan,
	prefetchSpan *parse_util.ParseSourceSpan,
	onSourceSpan *parse_util.ParseSourceSpan,
	hydrateSpan *parse_util.ParseSourceSpan,
) (*IdleDeferredTrigger, error) {
	if len(parameters) > 1 {
		return nil, fmt.Errorf("\"%s\" trigger can only have zero or one parameters", onTriggerTypeIdle)
	}

	var timeout *float64
	if len(parameters) > 0 {
		timeout = ParseDeferredTime(parameters[0].expression)
		if timeout == nil {
			return nil, fmt.Errorf("Could not parse time value of trigger \"%s\"", onTriggerTypeIdle)
		}
	}

	return NewIdleDeferredTrigger(*nameSpan, *sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan, timeout), nil
}

func createTimerTrigger(
	parameters []parsedParameter,
	nameSpan *parse_util.ParseSourceSpan,
	sourceSpan *parse_util.ParseSourceSpan,
	prefetchSpan *parse_util.ParseSourceSpan,
	onSourceSpan *parse_util.ParseSourceSpan,
	hydrateSpan *parse_util.ParseSourceSpan,
) (*TimerDeferredTrigger, error) {
	if len(parameters) != 1 {
		return nil, fmt.Errorf("\"%s\" trigger must have exactly one parameter", onTriggerTypeTimer)
	}

	delay := ParseDeferredTime(parameters[0].expression)
	if delay == nil {
		return nil, fmt.Errorf("Could not parse time value of trigger \"%s\"", onTriggerTypeTimer)
	}

	return NewTimerDeferredTrigger(*delay, *nameSpan, *sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

func createImmediateTrigger(
	parameters []parsedParameter,
	nameSpan *parse_util.ParseSourceSpan,
	sourceSpan *parse_util.ParseSourceSpan,
	prefetchSpan *parse_util.ParseSourceSpan,
	onSourceSpan *parse_util.ParseSourceSpan,
	hydrateSpan *parse_util.ParseSourceSpan,
) (*ImmediateDeferredTrigger, error) {
	if len(parameters) > 0 {
		return nil, fmt.Errorf("\"%s\" trigger cannot have parameters", onTriggerTypeImmediate)
	}

	return &ImmediateDeferredTrigger{
		BaseDeferredTrigger: BaseDeferredTrigger{
			NameSpan:           nameSpan,
			SourceSpan:         *sourceSpan,
			PrefetchSpan:       prefetchSpan,
			WhenOrOnSourceSpan: onSourceSpan,
			HydrateSpan:        hydrateSpan,
		},
	}, nil
}

func createHoverTrigger(
	parameters []parsedParameter,
	nameSpan *parse_util.ParseSourceSpan,
	sourceSpan *parse_util.ParseSourceSpan,
	prefetchSpan *parse_util.ParseSourceSpan,
	onSourceSpan *parse_util.ParseSourceSpan,
	hydrateSpan *parse_util.ParseSourceSpan,
	validator referenceTriggerValidator,
) (*HoverDeferredTrigger, error) {
	if err := validator(onTriggerTypeHover, parameters); err != nil {
		return nil, err
	}
	var ref *string
	if len(parameters) > 0 {
		ref = &parameters[0].expression
	}
	return NewHoverDeferredTrigger(ref, *nameSpan, *sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

func createInteractionTrigger(
	parameters []parsedParameter,
	nameSpan *parse_util.ParseSourceSpan,
	sourceSpan *parse_util.ParseSourceSpan,
	prefetchSpan *parse_util.ParseSourceSpan,
	onSourceSpan *parse_util.ParseSourceSpan,
	hydrateSpan *parse_util.ParseSourceSpan,
	validator referenceTriggerValidator,
) (*InteractionDeferredTrigger, error) {
	if err := validator(onTriggerTypeInteraction, parameters); err != nil {
		return nil, err
	}
	var ref *string
	if len(parameters) > 0 {
		ref = &parameters[0].expression
	}
	return NewInteractionDeferredTrigger(ref, *nameSpan, *sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

func createViewportTrigger(
	start int,
	isHydrationTrigger bool,
	bindingParser *template_parser.BindingParser,
	parameters []parsedParameter,
	nameSpan *parse_util.ParseSourceSpan,
	sourceSpan *parse_util.ParseSourceSpan,
	prefetchSpan *parse_util.ParseSourceSpan,
	onSourceSpan *parse_util.ParseSourceSpan,
	hydrateSpan *parse_util.ParseSourceSpan,
	validator referenceTriggerValidator,
) (*ViewportDeferredTrigger, error) {
	if err := validator(onTriggerTypeViewport, parameters); err != nil {
		return nil, err
	}

	var reference *string

	if len(parameters) == 0 {
		reference = nil
	} else if !strings.HasPrefix(parameters[0].expression, "{") {
		reference = &parameters[0].expression
	} else {
		parsed := bindingParser.ParseBinding(
			parameters[0].expression,
			false,
			expression_parser.ParseSourceSpan{},
			sourceSpan.Start.Offset+start+parameters[0].start,
		)
		if parsed.Ast == nil {
			return nil, fmt.Errorf("Options parameter of the \"viewport\" trigger must be an object literal")
		}

		if litMap, ok := parsed.Ast.(*expression_parser.LiteralMap); ok {
			triggerIndex := -1
			for i, key := range litMap.Keys {
				if spreadKey, isSpread := key.(*expression_parser.LiteralMapSpreadKey); isSpread {
					_ = spreadKey
					return nil, fmt.Errorf("Spread operator are not allowed in this context")
				} else if propKey, isProp := key.(*expression_parser.LiteralMapPropertyKey); isProp {
					if propKey.Key == "root" {
						return nil, fmt.Errorf("The \"root\" option is not supported in the options parameter of the \"viewport\" trigger")
					}
					if propKey.Key == "trigger" {
						triggerIndex = i
					}
				}
			}

			if triggerIndex == -1 {
				reference = nil
			} else {
				val := litMap.Values[triggerIndex]
				if propRead, isPropRead := val.(*expression_parser.PropertyRead); isPropRead {
					if _, isImplicit := propRead.Receiver.(*expression_parser.ImplicitReceiver); isImplicit {
						reference = &propRead.Name
					} else {
						return nil, fmt.Errorf("\"trigger\" option of the \"viewport\" trigger must be an identifier")
					}
				} else {
					return nil, fmt.Errorf("\"trigger\" option of the \"viewport\" trigger must be an identifier")
				}
			}

			for i, val := range litMap.Values {
				if i != triggerIndex && !isLiteralValue(val) {
					return nil, fmt.Errorf("Options of the \"viewport\" trigger must be an object literal containing only literal values, but a non-literal was found")
				}
			}
		} else {
			return nil, fmt.Errorf("Options parameter of the \"viewport\" trigger must be an object literal")
		}
	}

	if isHydrationTrigger && reference != nil {
		return nil, fmt.Errorf("\"viewport\" hydration trigger cannot have a \"trigger\"")
	}

	return NewViewportDeferredTrigger(reference, *nameSpan, *sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

func isLiteralValue(ast expression_parser.AST) bool {
	switch ast.(type) {
	case *expression_parser.ASTWithSource:
		return isLiteralValue(ast.(*expression_parser.ASTWithSource).Ast)
	case *expression_parser.LiteralPrimitive:
		return true
	case *expression_parser.LiteralArray:
		return true
	case *expression_parser.LiteralMap:
		return true
	default:
		return false
	}
}

func validatePlainReferenceBasedTrigger(triggerType onTriggerType, parameters []parsedParameter) error {
	if len(parameters) > 1 {
		return fmt.Errorf("\"%s\" trigger can only have zero or one parameters", triggerType)
	}
	return nil
}

func validateHydrateReferenceBasedTrigger(triggerType onTriggerType, parameters []parsedParameter) error {
	if triggerType == onTriggerTypeViewport {
		if len(parameters) > 1 {
			return fmt.Errorf("Hydration trigger \"%s\" cannot have more than one parameter", triggerType)
		}
		return nil
	}
	if len(parameters) > 0 {
		return fmt.Errorf("Hydration trigger \"%s\" cannot have parameters", triggerType)
	}
	return nil
}

func GetTriggerParametersStart(value string, startPosition int) int {
	hasFoundSeparator := false
	for i := startPosition; i < len(value); i++ {
		if separatorPattern.MatchString(string(value[i])) {
			hasFoundSeparator = true
		} else if hasFoundSeparator {
			return i
		}
	}
	return -1
}

func ParseDeferredTime(value string) *float64 {
	if !timePattern.MatchString(value) {
		return nil
	}

	var time float64
	var err error
	if strings.HasSuffix(value, "ms") {
		time, err = strconv.ParseFloat(value[:len(value)-2], 64)
	} else if strings.HasSuffix(value, "s") {
		time, err = strconv.ParseFloat(value[:len(value)-1], 64)
		time *= 1000
	} else {
		time, err = strconv.ParseFloat(value, 64)
	}
	if err != nil {
		return nil
	}
	return &time
}
