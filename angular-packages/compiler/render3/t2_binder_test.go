package render3

import (
	"fmt"
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

type mockPropertyMapping struct {
	byBindingName map[string]bool
}

func (m *mockPropertyMapping) HasBindingPropertyName(name string) bool {
	if m == nil || m.byBindingName == nil {
		return false
	}
	return m.byBindingName[name]
}

func makeMockPropertyMapping(obj map[string]interface{}) *mockPropertyMapping {
	byBindingName := make(map[string]bool)
	for _, val := range obj {
		if s, ok := val.(string); ok {
			byBindingName[s] = true
		}
	}
	return &mockPropertyMapping{byBindingName: byBindingName}
}

func boolPtr(b bool) *bool {
	return &b
}



var keyCounter = 0

type mockDirectiveMeta struct {
	name                  string
	refKey                string
	selector              *string
	isComponent           bool
	inputs                any
	outputs               any
	exportAs              []string
	isStructural          bool
	ngContentSelectors    []string
	preserveWhitespaces   bool
	animationTriggerNames *LegacyAnimationTriggerNames
	matchSource           MatchSource
}

func (m *mockDirectiveMeta) GetName() string                                        { return m.name }
func (m *mockDirectiveMeta) GetRefKey() string                                      { return m.refKey }
func (m *mockDirectiveMeta) GetSelector() *string                                   { return m.selector }
func (m *mockDirectiveMeta) IsComponent() bool                                      { return m.isComponent }
func (m *mockDirectiveMeta) GetInputs() any                                         { return m.inputs }
func (m *mockDirectiveMeta) GetOutputs() any                                        { return m.outputs }
func (m *mockDirectiveMeta) GetExportAs() []string                                  { return m.exportAs }
func (m *mockDirectiveMeta) IsStructural() bool                                     { return m.isStructural }
func (m *mockDirectiveMeta) GetNgContentSelectors() []string                        { return m.ngContentSelectors }
func (m *mockDirectiveMeta) GetPreserveWhitespaces() bool                           { return m.preserveWhitespaces }
func (m *mockDirectiveMeta) GetAnimationTriggerNames() *LegacyAnimationTriggerNames { return m.animationTriggerNames }
func (m *mockDirectiveMeta) GetMatchSource() MatchSource                            { return m.matchSource }

func (m *mockDirectiveMeta) WithInputsAndOutputs(inputs any, outputs any) DirectiveMeta {
	mCopy := *m
	mCopy.inputs = inputs
	mCopy.outputs = outputs
	return &mCopy
}

func (m *mockDirectiveMeta) WithMatchSource(ms MatchSource) DirectiveMeta {
	mCopy := *m
	mCopy.matchSource = ms
	return &mCopy
}

func makeDirectiveMeta(config struct {
	name         string
	selector     string
	inputs       map[string]string
	outputs      map[string]string
	exportAs     []string
	isComponent  bool
	isStructural bool
	matchSource  MatchSource
}) DirectiveMeta {
	var selector *string
	if config.selector != "" {
		s := config.selector
		selector = &s
	}
	refKey := fmt.Sprintf("%s#%d", config.name, keyCounter)
	keyCounter++

	inputsObj := make(map[string]interface{})
	for k, v := range config.inputs {
		inputsObj[k] = v
	}
	outputsObj := make(map[string]interface{})
	for k, v := range config.outputs {
		outputsObj[k] = v
	}

	return &mockDirectiveMeta{
		name:         config.name,
		refKey:       refKey,
		selector:     selector,
		isComponent:  config.isComponent,
		isStructural: config.isStructural,
		exportAs:     config.exportAs,
		inputs:       ClassPropertyMappingFromMappedObject(inputsObj),
		outputs:      ClassPropertyMappingFromMappedObject(outputsObj),
		matchSource:  config.matchSource,
	}
}

func makeSelectorMatcher() *SelectorMatcher[[]DirectiveMeta] {
	matcher := NewSelectorMatcher[[]DirectiveMeta]()
	matcher.AddSelectables(CssSelectorParse("[ngFor][ngForOf]"), []DirectiveMeta{
		makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:         "NgFor",
			selector:     "[ngFor][ngForOf]",
			isStructural: true,
			inputs:       map[string]string{"ngForOf": "ngForOf"},
		}),
	})
	matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
		makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "Dir",
			selector: "[dir]",
			exportAs: []string{"dir"},
		}),
	})
	matcher.AddSelectables(CssSelectorParse("[hasOutput]"), []DirectiveMeta{
		makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "HasOutput",
			selector: "[hasOutput]",
			outputs:  map[string]string{"outputBinding": "outputBinding"},
		}),
	})
	matcher.AddSelectables(CssSelectorParse("[hasInput]"), []DirectiveMeta{
		makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "HasInput",
			selector: "[hasInput]",
			inputs:   map[string]string{"inputBinding": "inputBinding"},
		}),
	})
	matcher.AddSelectables(CssSelectorParse("[sameSelectorAsInput]"), []DirectiveMeta{
		makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "SameSelectorAsInput",
			selector: "[sameSelectorAsInput]",
			inputs:   map[string]string{"sameSelectorAsInput": "sameSelectorAsInput"},
		}),
	})
	matcher.AddSelectables(CssSelectorParse("comp"), []DirectiveMeta{
		makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "Comp",
			selector:    "comp",
			isComponent: true,
		}),
	})

	simpleDirs := []string{"a", "b", "c", "d", "e", "f"}
	deferDirs := []string{"loading", "error", "placeholder"}
	for _, dir := range append(simpleDirs, deferDirs...) {
		name := "Dir" + strings.ToUpper(dir[:1]) + strings.ToLower(dir[1:])
		matcher.AddSelectables(CssSelectorParse("["+dir+"]"), []DirectiveMeta{
			makeDirectiveMeta(struct {
				name         string
				selector     string
				inputs       map[string]string
				outputs      map[string]string
				exportAs     []string
				isComponent  bool
				isStructural bool
				matchSource  MatchSource
			}{
				name:         name,
				selector:     "[" + dir + "]",
				isStructural: true,
			}),
		})
	}
	return matcher
}

func makeSelectorlessMatcher(directives []any) *SelectorlessMatcher[DirectiveMeta] {
	registry := make(map[string][]DirectiveMeta)
	for _, d := range directives {
		switch val := d.(type) {
		case DirectiveMeta:
			registry[val.GetName()] = []DirectiveMeta{val}
		case struct {
			root                 DirectiveMeta
			additionalDirectives []DirectiveMeta
		}:
			registry[val.root.GetName()] = append([]DirectiveMeta{val.root}, val.additionalDirectives...)
		}
	}
	return NewSelectorlessMatcher(registry)
}

func findExpression(nodes []Node, source string) expression_parser.AST {
	var found expression_parser.AST
	var walk func(n any)
	walk = func(n any) {
		if found != nil {
			return
		}
		switch val := n.(type) {
		case []Node:
			for _, child := range val {
				walk(child)
			}
		case *Element:
			for _, child := range val.Children {
				walk(child)
			}
			for _, input := range val.Inputs {
				if strings.Contains(input.SourceSpan.ToString(), source) || strings.Contains(fmt.Sprintf("%v", input.Value), source) {
					found = input.Value
					return
				}
				walk(input.Value)
			}
			for _, output := range val.Outputs {
				if strings.Contains(output.SourceSpan.ToString(), source) || strings.Contains(fmt.Sprintf("%v", output.Handler), source) {
					found = output.Handler
					return
				}
				walk(output.Handler)
			}
		case *Template:
			for _, child := range val.Children {
				walk(child)
			}
			for _, input := range val.Inputs {
				if strings.Contains(input.SourceSpan.ToString(), source) || strings.Contains(fmt.Sprintf("%v", input.Value), source) {
					found = input.Value
					return
				}
				walk(input.Value)
			}
			for _, output := range val.Outputs {
				if strings.Contains(output.SourceSpan.ToString(), source) || strings.Contains(fmt.Sprintf("%v", output.Handler), source) {
					found = output.Handler
					return
				}
				walk(output.Handler)
			}
		case *BoundText:
			if strings.Contains(val.SourceSpan.ToString(), source) || strings.Contains(fmt.Sprintf("%v", val.Value), source) {
				found = val.Value
				return
			}
		case *DeferredBlock:
			walk(val.Children)
			if val.Placeholder != nil {
				walk(val.Placeholder)
			}
			if val.Loading != nil {
				walk(val.Loading)
			}
			if val.Error != nil {
				walk(val.Error)
			}
		case *DeferredBlockPlaceholder:
			walk(val.Children)
		case *DeferredBlockLoading:
			walk(val.Children)
		case *DeferredBlockError:
			walk(val.Children)
		case *IfBlock:
			for _, b := range val.Branches {
				walk(b)
			}
		case *IfBlockBranch:
			walk(val.Children)
		case *ForLoopBlock:
			walk(val.Children)
			if val.Empty != nil {
				walk(val.Empty)
			}
		case *ForLoopBlockEmpty:
			walk(val.Children)
		case *SwitchBlock:
			for _, g := range val.Groups {
				walk(g)
			}
		case *SwitchBlockCaseGroup:
			walk(val.Children)
		case *Content:
			walk(val.Children)
		}
	}
	walk(nodes)
	return found
}

func TestFindMatchingDirectivesAndPipes(t *testing.T) {
	t.Run("should match directives and detect pipes in eager and deferrable parts of a template", func(t *testing.T) {
		template := `
			<div [title]="abc | uppercase"></div>
			@defer {
				<my-defer-cmp [label]="abc | lowercase" />
			} @placeholder {}
		`
		directiveSelectors := []string{"[title]", "my-defer-cmp", "not-matching"}
		res := FindMatchingDirectivesAndPipes(template, directiveSelectors)

		assert.Equal(t, []string{"[title]"}, res.Directives.Regular)
		assert.Equal(t, []string{"my-defer-cmp"}, res.Directives.DeferCandidates)
		assert.Equal(t, []string{"uppercase"}, res.Pipes.Regular)
		assert.Equal(t, []string{"lowercase"}, res.Pipes.DeferCandidates)
	})

	t.Run("should return empty directive list if no selectors are provided", func(t *testing.T) {
		template := `
			<div [title]="abc | uppercase"></div>
			@defer {
				<my-defer-cmp [label]="abc | lowercase" />
			} @placeholder {}
		`
		var directiveSelectors []string
		res := FindMatchingDirectivesAndPipes(template, directiveSelectors)

		assert.Empty(t, res.Directives.Regular)
		assert.Empty(t, res.Directives.DeferCandidates)
		assert.Equal(t, []string{"uppercase"}, res.Pipes.Regular)
		assert.Equal(t, []string{"lowercase"}, res.Pipes.DeferCandidates)
	})

	t.Run("should return a directive and a pipe only once (either as a regular or deferrable)", func(t *testing.T) {
		template := `
			<my-defer-cmp [label]="abc | lowercase" [title]="abc | uppercase" />
			@defer {
				<my-defer-cmp [label]="abc | lowercase" [title]="abc | uppercase" />
			} @placeholder {}
		`
		directiveSelectors := []string{"[title]", "my-defer-cmp", "not-matching"}
		res := FindMatchingDirectivesAndPipes(template, directiveSelectors)

		assert.ElementsMatch(t, []string{"my-defer-cmp", "[title]"}, res.Directives.Regular)
		assert.Empty(t, res.Directives.DeferCandidates)
		assert.ElementsMatch(t, []string{"lowercase", "uppercase"}, res.Pipes.Regular)
		assert.Empty(t, res.Pipes.DeferCandidates)
	})

	t.Run("should handle directives on elements with local refs", func(t *testing.T) {
		template := `
			<input [(ngModel)]="name" #ctrl="ngModel" required />
			@defer {
				<my-defer-cmp [label]="abc | lowercase" [title]="abc | uppercase" />
				<input [(ngModel)]="name" #ctrl="ngModel" required />
			} @placeholder {}
		`
		directiveSelectors := []string{
			"[ngModel]:not([formControlName]):not([formControl])",
			"[title]",
			"my-defer-cmp",
			"not-matching",
		}
		res := FindMatchingDirectivesAndPipes(template, directiveSelectors)

		assert.Equal(t, []string{"[ngModel]:not([formControlName]):not([formControl])"}, res.Directives.Regular)
		assert.ElementsMatch(t, []string{"my-defer-cmp", "[title]"}, res.Directives.DeferCandidates)
		assert.Empty(t, res.Pipes.Regular)
		assert.ElementsMatch(t, []string{"lowercase", "uppercase"}, res.Pipes.DeferCandidates)
	})
}

func TestT2Binding(t *testing.T) {
	t.Run("should bind a simple template", func(t *testing.T) {
		template := ParseTemplate("<div *ngFor=\"let item of items\">{{item.name}}</div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		itemBinding := findExpression(template.Nodes, "{{item.name}}")
		assert.NotNil(t, itemBinding)

		// Get implicitly read property target 'item'
		var implicitRead expression_parser.AST
		astNode := itemBinding
		if aws, ok := itemBinding.(*expression_parser.ASTWithSource); ok {
			astNode = aws.Ast
		}
		if interp, ok := astNode.(*expression_parser.Interpolation); ok {
			if pr, ok := interp.Expressions[0].(*expression_parser.PropertyRead); ok {
				implicitRead = pr.Receiver
			}
		}
		assert.NotNil(t, implicitRead)

		target := res.GetExpressionTarget(implicitRead)
		assert.NotNil(t, target)
		assert.Equal(t, "$implicit", target.(*Variable).Value)

		itemTemplate := res.GetDefinitionNodeOfSymbol(target)
		assert.NotNil(t, itemTemplate)
		assert.Equal(t, 1, res.GetNestingLevel(itemTemplate))
	})

	t.Run("should match directives when binding a simple template", func(t *testing.T) {
		template := ParseTemplate("<div *ngFor=\"let item of items\">{{item.name}}</div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		tmpl := template.Nodes[0].(*Template)
		directives := res.GetDirectivesOfNode(tmpl)
		assert.Len(t, directives, 1)
		assert.Equal(t, "NgFor", directives[0].GetName())
	})

	t.Run("should match directives on namespaced elements", func(t *testing.T) {
		template := ParseTemplate("<svg><text dir>SVG</text></svg>", "", nil)
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		matcher.AddSelectables(CssSelectorParse("text[dir]"), []DirectiveMeta{
			makeDirectiveMeta(struct {
				name         string
				selector     string
				inputs       map[string]string
				outputs      map[string]string
				exportAs     []string
				isComponent  bool
				isStructural bool
				matchSource  MatchSource
			}{
				name:     "Dir",
				selector: "text[dir]",
			}),
		})
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		svgNode := template.Nodes[0].(*Element)
		textNode := svgNode.Children[0].(*Element)
		directives := res.GetDirectivesOfNode(textNode)
		assert.Len(t, directives, 1)
		assert.Equal(t, "Dir", directives[0].GetName())
	})

	t.Run("should not match directives intended for an element on a microsyntax template", func(t *testing.T) {
		template := ParseTemplate("<div *ngFor=\"let item of items\" dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		tmpl := template.Nodes[0].(*Template)
		tmplDirectives := res.GetDirectivesOfNode(tmpl)
		assert.Len(t, tmplDirectives, 1)
		assert.Equal(t, "NgFor", tmplDirectives[0].GetName())

		elDirectives := res.GetDirectivesOfNode(tmpl.Children[0].(*Element))
		assert.Len(t, elDirectives, 1)
		assert.Equal(t, "Dir", elDirectives[0].GetName())
	})

	t.Run("should get @let declarations when resolving entities at the root", func(t *testing.T) {
		template := ParseTemplate(`
			@let one = 1;
			@let two = 2;
			@let sum = one + two;
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		entities := res.GetEntitiesInScope(nil)
		var names []string
		for _, e := range entities {
			names = append(names, e.GetName())
		}
		assert.ElementsMatch(t, []string{"one", "two", "sum"}, names)
	})

	t.Run("should scope @let declarations to their current view", func(t *testing.T) {
		template := ParseTemplate(`
			@let one = 1;
			@if (true) {
				@let two = 2;
			}
			@if (true) {
				@let three = 3;
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		rootEntities := res.GetEntitiesInScope(nil)
		firstBranchEntities := res.GetEntitiesInScope(template.Nodes[1].(*IfBlock).Branches[0])
		secondBranchEntities := res.GetEntitiesInScope(template.Nodes[2].(*IfBlock).Branches[0])

		var rootNames, firstNames, secondNames []string
		for _, e := range rootEntities {
			rootNames = append(rootNames, e.GetName())
		}
		for _, e := range firstBranchEntities {
			firstNames = append(firstNames, e.GetName())
		}
		for _, e := range secondBranchEntities {
			secondNames = append(secondNames, e.GetName())
		}

		assert.Equal(t, []string{"one"}, rootNames)
		assert.ElementsMatch(t, []string{"one", "two"}, firstNames)
		assert.ElementsMatch(t, []string{"one", "three"}, secondNames)
	})

	t.Run("should resolve expressions to an @let declaration", func(t *testing.T) {
		template := ParseTemplate(`
			@let value = 1;
			{{value}}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		boundText := template.Nodes[1].(*BoundText)
		var propRead expression_parser.AST
		if interp, ok := boundText.Value.(*expression_parser.ASTWithSource).Ast.(*expression_parser.Interpolation); ok {
			propRead = interp.Expressions[0]
		}
		assert.NotNil(t, propRead)

		target := res.GetExpressionTarget(propRead)
		assert.NotNil(t, target)
		assert.IsType(t, &LetDeclaration{}, target)
		assert.Equal(t, "value", target.(*LetDeclaration).Name)
	})

	t.Run("should not resolve a `this` access to a template reference", func(t *testing.T) {
		template := ParseTemplate(`
			<input #value>
			{{this.value}}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		boundText := template.Nodes[1].(*BoundText)
		var propRead expression_parser.AST
		if interp, ok := boundText.Value.(*expression_parser.ASTWithSource).Ast.(*expression_parser.Interpolation); ok {
			propRead = interp.Expressions[0]
		}
		assert.NotNil(t, propRead)

		target := res.GetExpressionTarget(propRead)
		assert.Nil(t, target)
	})

	t.Run("should not resolve a `this` access to a template variable", func(t *testing.T) {
		template := ParseTemplate(`<ng-template let-value>{{this.value}}</ng-template>`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		tmpl := template.Nodes[0].(*Template)
		boundText := tmpl.Children[0].(*BoundText)
		var propRead expression_parser.AST
		if interp, ok := boundText.Value.(*expression_parser.ASTWithSource).Ast.(*expression_parser.Interpolation); ok {
			propRead = interp.Expressions[0]
		}
		assert.NotNil(t, propRead)

		target := res.GetExpressionTarget(propRead)
		assert.Nil(t, target)
	})

	t.Run("should not resolve a `this` access to a `@let` declaration", func(t *testing.T) {
		template := ParseTemplate(`
			@let value = 1;
			{{this.value}}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		boundText := template.Nodes[1].(*BoundText)
		var propRead expression_parser.AST
		if interp, ok := boundText.Value.(*expression_parser.ASTWithSource).Ast.(*expression_parser.Interpolation); ok {
			propRead = interp.Expressions[0]
		}
		assert.NotNil(t, propRead)

		target := res.GetExpressionTarget(propRead)
		assert.Nil(t, target)
	})

	t.Run("should resolve the definition node of let declarations", func(t *testing.T) {
		template := ParseTemplate(`
			@if (true) {
				@let one = 1;
			}
			@if (true) {
				@let two = 2;
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		firstBranch := template.Nodes[0].(*IfBlock).Branches[0]
		firstLet := firstBranch.Children[0].(*LetDeclaration)
		secondBranch := template.Nodes[1].(*IfBlock).Branches[0]
		secondLet := secondBranch.Children[0].(*LetDeclaration)

		assert.Equal(t, firstBranch, res.GetDefinitionNodeOfSymbol(firstLet))
		assert.Equal(t, secondBranch, res.GetDefinitionNodeOfSymbol(secondLet))
	})

	t.Run("should resolve an element reference without a directive matcher", func(t *testing.T) {
		template := ParseTemplate("<div #foo></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](nil, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		node := template.Nodes[0].(*Element)
		reference := node.References[0]
		result := res.GetReferenceTarget(reference)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Element)
		assert.Equal(t, "div", result.Element.Name)
	})

	t.Run("matching inputs to consuming directives: should work for bound attributes", func(t *testing.T) {
		template := ParseTemplate("<div hasInput [inputBinding]=\"myValue\"></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Element)
		attr := el.Inputs[0]
		consumer := res.GetConsumerOfBinding(attr).(DirectiveMeta)
		assert.Equal(t, "HasInput", consumer.GetName())
	})

	t.Run("matching inputs to consuming directives: should work for text attributes on elements", func(t *testing.T) {
		template := ParseTemplate("<div hasInput inputBinding=\"text\"></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Element)
		attr := el.Attributes[1]
		consumer := res.GetConsumerOfBinding(attr).(DirectiveMeta)
		assert.Equal(t, "HasInput", consumer.GetName())
	})

	t.Run("matching inputs to consuming directives: should work for text attributes on templates", func(t *testing.T) {
		template := ParseTemplate("<ng-template hasInput inputBinding=\"text\"></ng-template>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Template)
		attr := el.Attributes[1]
		consumer := res.GetConsumerOfBinding(attr).(DirectiveMeta)
		assert.Equal(t, "HasInput", consumer.GetName())
	})

	t.Run("matching inputs to consuming directives: should not match directives on attribute bindings with the same name as an input", func(t *testing.T) {
		template := ParseTemplate("<ng-template [attr.sameSelectorAsInput]=\"123\"></ng-template>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Template)
		input := el.Inputs[0]
		consumer := res.GetConsumerOfBinding(input)
		assert.Equal(t, el, consumer)
	})

	t.Run("matching inputs to consuming directives: should bind to the encompassing node when no directive input is matched", func(t *testing.T) {
		template := ParseTemplate("<span dir></span>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Element)
		attr := el.Attributes[0]
		consumer := res.GetConsumerOfBinding(attr)
		assert.Equal(t, el, consumer)
	})

	t.Run("matching outputs to consuming directives: should work for bound events", func(t *testing.T) {
		template := ParseTemplate("<div hasOutput (outputBinding)=\"myHandler($event)\"></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Element)
		attr := el.Outputs[0]
		consumer := res.GetConsumerOfBinding(attr).(DirectiveMeta)
		assert.Equal(t, "HasOutput", consumer.GetName())
	})

	t.Run("matching outputs to consuming directives: should bind to the encompassing node when no directive output is matched", func(t *testing.T) {
		template := ParseTemplate("<span dir (fakeOutput)=\"myHandler($event)\"></span>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Element)
		attr := el.Outputs[0]
		consumer := res.GetConsumerOfBinding(attr)
		assert.Equal(t, el, consumer)
	})

	t.Run("extracting defer blocks info: should extract top-level defer blocks", func(t *testing.T) {
		template := ParseTemplate(`
			@defer {<cmp-a />}
			@defer {<cmp-b />}
			<cmp-c />
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		deferBlocks := bound.GetDeferBlocks()
		assert.Len(t, deferBlocks, 2)
	})

	t.Run("extracting defer blocks info: should extract nested defer blocks and associated pipes", func(t *testing.T) {
		template := ParseTemplate(`
			@defer {
				{{ name | pipeA }}
				@defer {
					{{ name | pipeB }}
				}
			} @loading {
				@defer {
					{{ name | pipeC }}
				}
				{{ name | loading }}
			} @placeholder {
				@defer {
					{{ name | pipeD }}
				}
				{{ name | placeholder }}
			} @error {
				@defer {
					{{ name | pipeE }}
				}
				{{ name | error }}
			}
			{{ name | pipeF }}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		deferBlocks := bound.GetDeferBlocks()

		assert.Len(t, deferBlocks, 5)
		assert.ElementsMatch(t, []string{"placeholder", "loading", "error", "pipeF"}, bound.GetEagerlyUsedPipes())
		assert.ElementsMatch(t, []string{
			"pipeA", "pipeB", "pipeD", "placeholder", "pipeC", "loading", "pipeE", "error", "pipeF",
		}, bound.GetUsedPipes())
	})

	t.Run("extracting defer blocks info: should identify pipes used after a nested defer block as being lazy", func(t *testing.T) {
		template := ParseTemplate(`
			@defer {
				{{ name | pipeA }}
				@defer {
					{{ name | pipeB }}
				}
				{{ name | pipeC }}
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		assert.ElementsMatch(t, []string{"pipeA", "pipeB", "pipeC"}, bound.GetUsedPipes())
		assert.Empty(t, bound.GetEagerlyUsedPipes())
	})

	t.Run("extracting defer blocks info: should extract nested defer blocks and associated directives", func(t *testing.T) {
		template := ParseTemplate(`
			@defer {
				<img *a />
				@defer {
					<img *b />
				}
			} @loading {
				@defer {
					<img *c />
				}
				<img *loading />
			} @placeholder {
				@defer {
					<img *d />
				}
				<img *placeholder />
			} @error {
				@defer {
					<img *e />
				}
				<img *error />
			}
			<img *f />
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		deferBlocks := bound.GetDeferBlocks()

		assert.Len(t, deferBlocks, 5)

		eagerDirs := bound.GetEagerlyUsedDirectives()
		var eagerNames []string
		for _, d := range eagerDirs {
			eagerNames = append(eagerNames, d.GetName())
		}
		assert.ElementsMatch(t, []string{"DirPlaceholder", "DirLoading", "DirError", "DirF"}, eagerNames)

		allDirs := bound.GetUsedDirectives()
		var allNames []string
		for _, d := range allDirs {
			allNames = append(allNames, d.GetName())
		}
		assert.ElementsMatch(t, []string{
			"DirA", "DirB", "DirD", "DirPlaceholder", "DirC", "DirLoading", "DirE", "DirError", "DirF",
		}, allNames)
	})

	t.Run("extracting defer blocks info: should identify directives used after a nested defer block as being lazy", func(t *testing.T) {
		template := ParseTemplate(`
			@defer {
				<img *a />
				@defer {<img *b />}
				<img *c />
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		var allNames, eagerNames []string
		for _, d := range bound.GetUsedDirectives() {
			allNames = append(allNames, d.GetName())
		}
		for _, d := range bound.GetEagerlyUsedDirectives() {
			eagerNames = append(eagerNames, d.GetName())
		}

		assert.ElementsMatch(t, []string{"DirA", "DirB", "DirC"}, allNames)
		assert.Empty(t, eagerNames)
	})

	t.Run("extracting defer blocks info: should identify a trigger element that is a parent of the deferred block", func(t *testing.T) {
		template := ParseTemplate(`
			<div #trigger>
				@defer (on viewport(trigger)) {}
			</div>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "div", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should identify a trigger element outside of the deferred block", func(t *testing.T) {
		template := ParseTemplate(`
			<div>
				@defer (on viewport(trigger)) {}
			</div>
			<div>
				<div>
					<button #trigger></button>
				</div>
			</div>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "button", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should identify a trigger element in a parent embedded view", func(t *testing.T) {
		template := ParseTemplate(`
			<div *ngFor="let item of items">
				<button #trigger></button>
				<div *ngFor="let child of item.children">
					<div *ngFor="let grandchild of child.children">
						@defer (on viewport(trigger)) {}
					</div>
				</div>
			</div>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "button", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should identify a trigger element inside the placeholder", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport(trigger)) {
				main
			} @placeholder {
				<button #trigger></button>
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "button", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should not identify a trigger inside the main content block", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport(trigger)) {<button #trigger></button>}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("extracting defer blocks info: should identify a trigger element on a component", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport(trigger)) {}
			<comp #trigger/>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "comp", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should identify a trigger element on a directive", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport(trigger)) {}
			<button dir #trigger="dir"></button>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "button", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should identify an implicit trigger inside the placeholder block", func(t *testing.T) {
		template := ParseTemplate(`
			<div #trigger>
				@defer (on viewport) {} @placeholder {<button></button>}
			</div>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "button", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should identify an implicit trigger inside the placeholder block with comments", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport) {
				main
			} @placeholder {
				<!-- before -->
				<button #trigger></button>
				<!-- after -->
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.NotNil(t, triggerEl)
		assert.Equal(t, "button", triggerEl.Name)
	})

	t.Run("extracting defer blocks info: should not identify an implicit trigger if the placeholder has multiple root nodes", func(t *testing.T) {
		template := ParseTemplate(`
			<div #trigger>
				@defer (on viewport) {} @placeholder {<button></button><div></div>}
			</div>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("extracting defer blocks info: should not identify an implicit trigger if there is no placeholder", func(t *testing.T) {
		template := ParseTemplate(`
			<div #trigger>
				@defer (on viewport) {}
				<button></button>
			</div>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("extracting defer blocks info: should not identify an implicit trigger if the placeholder has a single root text node", func(t *testing.T) {
		template := ParseTemplate(`
			<div #trigger>
				@defer (on viewport) {} @placeholder {hello}
			</div>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("extracting defer blocks info: should not identify a trigger inside a sibling embedded view", func(t *testing.T) {
		template := ParseTemplate(`
			<div *ngIf="cond">
				<button #trigger></button>
			</div>
			@defer (on viewport(trigger)) {}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("extracting defer blocks info: should not identify a trigger element in an embedded view inside the placeholder", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport(trigger)) {
				main
			} @placeholder {
				<div *ngIf="cond"><button #trigger></button></div>
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("extracting defer blocks info: should not identify a trigger element inside a deferred block within the placeholder", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport(trigger)) {
				main
			} @placeholder {
				@defer {
					<button #trigger></button>
				}
			}
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("extracting defer blocks info: should not identify a trigger element on a template", func(t *testing.T) {
		template := ParseTemplate(`
			@defer (on viewport(trigger)) {}
			<ng-template #trigger></ng-template>
		`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		bound := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		block := bound.GetDeferBlocks()[0]
		triggerEl := bound.GetDeferredTriggerTarget(block, block.Triggers.Viewport)
		assert.Nil(t, triggerEl)
	})

	t.Run("used pipes: should record pipes used in interpolations", func(t *testing.T) {
		template := ParseTemplate("{{value|date}}", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		assert.Equal(t, []string{"date"}, res.GetUsedPipes())
	})

	t.Run("used pipes: should record pipes used in bound attributes", func(t *testing.T) {
		template := ParseTemplate("<person [age]=\"age|number\"></person>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		assert.Equal(t, []string{"number"}, res.GetUsedPipes())
	})

	t.Run("used pipes: should record pipes used in bound template attributes", func(t *testing.T) {
		template := ParseTemplate("<ng-template [ngIf]=\"obs|async\"></ng-template>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		assert.Equal(t, []string{"async"}, res.GetUsedPipes())
	})

	t.Run("used pipes: should record pipes used in ICUs", func(t *testing.T) {
		template := ParseTemplate(`<span i18n>{count|number, plural, =1 { {{value|date}} } }</span>`, "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		assert.ElementsMatch(t, []string{"number", "date"}, res.GetUsedPipes())
	})

	t.Run("selectorless: should resolve directives applied on a component node", func(t *testing.T) {
		template := ParseTemplate("<MyComp @Dir @OtherDir/>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				struct {
					root                 DirectiveMeta
					additionalDirectives []DirectiveMeta
				}{
					root: makeDirectiveMeta(struct {
						name         string
						selector     string
						inputs       map[string]string
						outputs      map[string]string
						exportAs     []string
						isComponent  bool
						isStructural bool
						matchSource  MatchSource
					}{
						name:        "MyComp",
						isComponent: true,
					}),
					additionalDirectives: []DirectiveMeta{
						makeDirectiveMeta(struct {
							name         string
							selector     string
							inputs       map[string]string
							outputs      map[string]string
							exportAs     []string
							isComponent  bool
							isStructural bool
							matchSource  MatchSource
						}{
							name: "MyHostDir",
						}),
					},
				},
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "Dir",
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "OtherDir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		node := template.Nodes[0].(*Component)
		
		var names []string
		for _, d := range res.GetDirectivesOfNode(node) {
			names = append(names, d.GetName())
		}
		assert.ElementsMatch(t, []string{"MyComp", "MyHostDir"}, names)
	})

	t.Run("selectorless: should resolve directives applied on a directive node", func(t *testing.T) {
		template := ParseTemplate("<MyComp @Dir @OtherDir/>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		t.Logf("Node: %#v", template.Nodes[0])
		if comp, ok := template.Nodes[0].(*Component); ok {
			t.Logf("Comp Directives len: %d", len(comp.Directives))
			for i, d := range comp.Directives {
				t.Logf("  Dir %d: %#v", i, d)
			}
		}
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:        "MyComp",
					isComponent: true,
				}),
				struct {
					root                 DirectiveMeta
					additionalDirectives []DirectiveMeta
				}{
					root: makeDirectiveMeta(struct {
						name         string
						selector     string
						inputs       map[string]string
						outputs      map[string]string
						exportAs     []string
						isComponent  bool
						isStructural bool
						matchSource  MatchSource
					}{
						name: "Dir",
					}),
					additionalDirectives: []DirectiveMeta{
						makeDirectiveMeta(struct {
							name         string
							selector     string
							inputs       map[string]string
							outputs      map[string]string
							exportAs     []string
							isComponent  bool
							isStructural bool
							matchSource  MatchSource
						}{
							name: "HostDir",
						}),
					},
				},
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "OtherDir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		dirs := template.Nodes[0].(*Component).Directives

		var names1, names2 []string
		for _, d := range res.GetDirectivesOfNode(dirs[0]) {
			names1 = append(names1, d.GetName())
		}
		for _, d := range res.GetDirectivesOfNode(dirs[1]) {
			names2 = append(names2, d.GetName())
		}

		assert.ElementsMatch(t, []string{"Dir", "HostDir"}, names1)
		assert.Equal(t, []string{"OtherDir"}, names2)
	})

	t.Run("selectorless: should not apply selectorless directives on an element node", func(t *testing.T) {
		template := ParseTemplate("<div @Dir @OtherDir></div>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "Dir",
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "OtherDir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		node := template.Nodes[0].(*Element)
		assert.Nil(t, res.GetDirectivesOfNode(node))
	})

	t.Run("selectorless: should resolve a reference on a component node to the component", func(t *testing.T) {
		template := ParseTemplate("<MyComp #foo/>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:        "MyComp",
					isComponent: true,
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		node := template.Nodes[0].(*Component)
		reference := node.References[0]
		result := res.GetReferenceTarget(reference)

		assert.NotNil(t, result)
		assert.Equal(t, node, result.Node)
		assert.Equal(t, "MyComp", result.Element.Name)
	})

	t.Run("selectorless: should resolve a reference on a directive node to the component", func(t *testing.T) {
		template := ParseTemplate("<div @Dir(#foo)></div>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "Dir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		node := template.Nodes[0].(*Element)
		t.Logf("Node directives len: %d", len(node.Directives))
		if len(node.Directives) > 0 {
			t.Logf("Directive 0: %#v", node.Directives[0])
			t.Logf("Directive 0 References len: %d", len(node.Directives[0].References))
			for i, r := range node.Directives[0].References {
				t.Logf("  Ref %d: %#v", i, r)
			}
		}
		
		// Access the underlying references map using reflection or type assertion if possible, or just print target
		directive := node.Directives[0]
		reference := directive.References[0]
		result := res.GetReferenceTarget(reference)

		assert.NotNil(t, result)
		assert.Equal(t, directive, result.Node)
		assert.Equal(t, "Dir", result.Directive.GetName())
	})

	t.Run("selectorless: should resolve a reference on an element when using a selectorless matcher", func(t *testing.T) {
		template := ParseTemplate("<div #foo></div>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorlessMatcher([]any{}), nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		node := template.Nodes[0].(*Element)
		reference := node.References[0]
		result := res.GetReferenceTarget(reference)

		assert.NotNil(t, result)
		assert.NotNil(t, result.Element)
		assert.Equal(t, "div", result.Element.Name)
	})

	t.Run("selectorless: should get consumer of component bindings", func(t *testing.T) {
		template := ParseTemplate(
			"<MyComp [input]=\"value\" static=\"value\" (output)=\"doStuff()\" [doesNotExist]=\"value\" [attr.input]=\"value\"/>",
			"",
			&ParseTemplateOptions{EnableSelectorless: boolPtr(true)},
		)
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:        "MyComp",
					isComponent: true,
					inputs:      map[string]string{"input": "input", "static": "static"},
					outputs:     map[string]string{"output": "output"},
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		node := template.Nodes[0].(*Component)
		
		input := node.Inputs[0]
		staticAttr := node.Attributes[0]
		output := node.Outputs[0]
		doesNotExist := node.Inputs[1]
		attrBinding := node.Inputs[2]

		assert.Equal(t, "MyComp", res.GetConsumerOfBinding(input).(DirectiveMeta).GetName())
		assert.Equal(t, "MyComp", res.GetConsumerOfBinding(staticAttr).(DirectiveMeta).GetName())
		assert.Equal(t, "MyComp", res.GetConsumerOfBinding(output).(DirectiveMeta).GetName())
		assert.Nil(t, res.GetConsumerOfBinding(doesNotExist))
		assert.Nil(t, res.GetConsumerOfBinding(attrBinding))
	})

	t.Run("selectorless: should get consumer of directive bindings", func(t *testing.T) {
		template := ParseTemplate(
			"<div @Dir([input]=\"value\" static=\"value\" (output)=\"doStuff()\" [doesNotExist]=\"value\")></div>",
			"",
			&ParseTemplateOptions{EnableSelectorless: boolPtr(true)},
		)
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:    "Dir",
					inputs:  map[string]string{"input": "input", "static": "static"},
					outputs: map[string]string{"output": "output"},
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		node := template.Nodes[0].(*Element)
		directive := node.Directives[0]
		input := directive.Inputs[0]
		staticAttr := directive.Attributes[0]
		output := directive.Outputs[0]
		doesNotExist := directive.Inputs[1]

		assert.Equal(t, "Dir", res.GetConsumerOfBinding(input).(DirectiveMeta).GetName())
		assert.Equal(t, "Dir", res.GetConsumerOfBinding(staticAttr).(DirectiveMeta).GetName())
		assert.Equal(t, "Dir", res.GetConsumerOfBinding(output).(DirectiveMeta).GetName())
		assert.Nil(t, res.GetConsumerOfBinding(doesNotExist))
	})

	t.Run("selectorless: should get eagerly-used selectorless directives", func(t *testing.T) {
		template := ParseTemplate("<MyComp @Dir @OtherDir/>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:        "MyComp",
					isComponent: true,
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "Dir",
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "OtherDir",
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "UnusedDir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		var usedNames, eagerNames []string
		for _, d := range res.GetUsedDirectives() {
			usedNames = append(usedNames, d.GetName())
		}
		for _, d := range res.GetEagerlyUsedDirectives() {
			eagerNames = append(eagerNames, d.GetName())
		}

		assert.ElementsMatch(t, []string{"MyComp", "Dir", "OtherDir"}, usedNames)
		assert.ElementsMatch(t, []string{"MyComp", "Dir", "OtherDir"}, eagerNames)
	})

	t.Run("selectorless: should get deferred selectorless directives", func(t *testing.T) {
		template := ParseTemplate("@defer {<MyComp @Dir @OtherDir/>}", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:        "MyComp",
					isComponent: true,
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "Dir",
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "OtherDir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		var usedNames, eagerNames []string
		for _, d := range res.GetUsedDirectives() {
			usedNames = append(usedNames, d.GetName())
		}
		for _, d := range res.GetEagerlyUsedDirectives() {
			eagerNames = append(eagerNames, d.GetName())
		}

		assert.ElementsMatch(t, []string{"MyComp", "Dir", "OtherDir"}, usedNames)
		assert.Empty(t, eagerNames)
	})

	t.Run("selectorless: should get selectorless directives nested in other code", func(t *testing.T) {
		template := ParseTemplate(`
			<section>
				@if (someCond) {
					<MyComp>
						<div>
							<h1>
								<span @Dir></span>
							</h1>
						</div>
					</MyComp>
				}
			</section>
		`, "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:        "MyComp",
					isComponent: true,
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "Dir",
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "UnusedDir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		var usedNames, eagerNames []string
		for _, d := range res.GetUsedDirectives() {
			usedNames = append(usedNames, d.GetName())
		}
		for _, d := range res.GetEagerlyUsedDirectives() {
			eagerNames = append(eagerNames, d.GetName())
		}

		assert.ElementsMatch(t, []string{"MyComp", "Dir"}, usedNames)
		assert.ElementsMatch(t, []string{"MyComp", "Dir"}, eagerNames)
	})

	t.Run("selectorless: should check whether a referenced directive exists", func(t *testing.T) {
		template := ParseTemplate("<MyComp @MissingDir/><MissingComp @Dir/>", "", &ParseTemplateOptions{EnableSelectorless: boolPtr(true)})
		binder := NewR3TargetBinder[DirectiveMeta](
			makeSelectorlessMatcher([]any{
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name:        "MyComp",
					isComponent: true,
				}),
				makeDirectiveMeta(struct {
					name         string
					selector     string
					inputs       map[string]string
					outputs      map[string]string
					exportAs     []string
					isComponent  bool
					isStructural bool
					matchSource  MatchSource
				}{
					name: "Dir",
				}),
			}),
			nil,
		)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		assert.True(t, res.ReferencedDirectiveExists("MyComp"))
		assert.True(t, res.ReferencedDirectiveExists("Dir"))
		assert.False(t, res.ReferencedDirectiveExists("MissingDir"))
		assert.False(t, res.ReferencedDirectiveExists("MissingComp"))
	})

	t.Run("directive de-duplication: should give precedence to the template-matched directive over a host-directive-based match", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			selector:    "[dir]",
			matchSource: MatchSourceHostDirective,
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			hostDir.(*mockDirectiveMeta).WithMatchSource(MatchSourceSelector),
		})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			makeDirectiveMeta(struct {
				name         string
				selector     string
				inputs       map[string]string
				outputs      map[string]string
				exportAs     []string
				isComponent  bool
				isStructural bool
				matchSource  MatchSource
			}{
				name:     "Dir",
				selector: "[dir]",
			}),
			hostDir,
		})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)
		directives := res.GetDirectivesOfNode(element)

		var formatted []string
		for _, d := range directives {
			kind := "Selector"
			if d.GetMatchSource() == MatchSourceHostDirective {
				kind = "HostDirective"
			}
			formatted = append(formatted, fmt.Sprintf("%s:%s", d.GetName(), kind))
		}
		assert.ElementsMatch(t, []string{"HostDir:Selector", "Dir:Selector"}, formatted)
	})

	t.Run("directive de-duplication: should de-duplicate directives that match multiple times as host directives", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			matchSource: MatchSourceHostDirective,
		})
		oneDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "OneDir",
			selector: "[dir]",
		})
		twoDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "TwoDir",
			selector: "[dir]",
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{oneDir, hostDir})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{twoDir, hostDir})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)
		directives := res.GetDirectivesOfNode(element)

		var formatted []string
		for _, d := range directives {
			kind := "Selector"
			if d.GetMatchSource() == MatchSourceHostDirective {
				kind = "HostDirective"
			}
			formatted = append(formatted, fmt.Sprintf("%s:%s", d.GetName(), kind))
		}
		assert.ElementsMatch(t, []string{"OneDir:Selector", "HostDir:HostDirective", "TwoDir:Selector"}, formatted)
	})

	t.Run("selectorless: should match foreign components by tag name", func(t *testing.T) {
		template := ParseTemplate("<FancyButton></FancyButton>", "", nil)
		registry := make(map[string][]ForeignComponentMeta)
		registry["FancyButton"] = []ForeignComponentMeta{{Name: "FancyButton", Ref: ForeignComponentMetaRef{Key: "FancyButtonKey"}}}
		foreignMatcher := NewSelectorlessMatcher(registry)

		binder := NewR3TargetBinder[DirectiveMeta](NewSelectorMatcher[[]DirectiveMeta](), foreignMatcher)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})

		el := template.Nodes[0].(*Element)
		foreignComp := res.GetForeignComponent(el)
		assert.NotNil(t, foreignComp)
		assert.Equal(t, "FancyButton", foreignComp.Name)
	})

	t.Run("selectorless: should throw an error when tag matches both directive and foreign component", func(t *testing.T) {
		template := ParseTemplate("<comp></comp>", "", nil)
		registry := make(map[string][]ForeignComponentMeta)
		registry["comp"] = []ForeignComponentMeta{{Name: "comp", Ref: ForeignComponentMetaRef{Key: "compKey"}}}
		foreignMatcher := NewSelectorlessMatcher(registry)

		binder := NewR3TargetBinder[DirectiveMeta](makeSelectorMatcher(), foreignMatcher)

		assert.Panics(t, func() {
			binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		})
	})

	t.Run("directive de-duplication: should merge the inputs of duplicated host directives", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			matchSource: MatchSourceHostDirective,
		})
		oneDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "OneDir",
			selector: "[dir]",
		})
		twoDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "TwoDir",
			selector: "[dir]",
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			oneDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "one"}),
				EmptyClassPropertyMapping(),
			),
		})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			twoDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"two": "twoAlias"}),
				EmptyClassPropertyMapping(),
			),
		})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)

		var mergedHost DirectiveMeta
		for _, d := range res.GetDirectivesOfNode(element) {
			if d.GetName() == "HostDir" {
				mergedHost = d
				break
			}
		}

		assert.NotNil(t, mergedHost)
		assert.Equal(t, MatchSourceHostDirective, mergedHost.GetMatchSource())
		inputsMap := mergedHost.GetInputs().(*ClassPropertyMappingGeneric).ToDirectMappedObject()
		assert.Equal(t, map[string]string{"one": "one", "two": "twoAlias"}, inputsMap)
		assert.Nil(t, res.GetConflictingHostDirectiveBindings(element))
	})

	t.Run("directive de-duplication: should merge the outputs of duplicated host directives", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			matchSource: MatchSourceHostDirective,
		})
		oneDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "OneDir",
			selector: "[dir]",
		})
		twoDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "TwoDir",
			selector: "[dir]",
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			oneDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				EmptyClassPropertyMapping(),
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "one"}),
			),
		})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			twoDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				EmptyClassPropertyMapping(),
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"two": "twoAlias"}),
			),
		})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)

		var mergedHost DirectiveMeta
		for _, d := range res.GetDirectivesOfNode(element) {
			if d.GetName() == "HostDir" {
				mergedHost = d
				break
			}
		}

		assert.NotNil(t, mergedHost)
		assert.Equal(t, MatchSourceHostDirective, mergedHost.GetMatchSource())
		outputsMap := mergedHost.GetOutputs().(*ClassPropertyMappingGeneric).ToDirectMappedObject()
		assert.Equal(t, map[string]string{"one": "one", "two": "twoAlias"}, outputsMap)
		assert.Nil(t, res.GetConflictingHostDirectiveBindings(element))
	})

	t.Run("directive de-duplication: should capture conflicting input bindings in host directives", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			matchSource: MatchSourceHostDirective,
		})
		oneDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "OneDir",
			selector: "[dir]",
		})
		twoDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "TwoDir",
			selector: "[dir]",
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			oneDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "one"}),
				EmptyClassPropertyMapping(),
			),
		})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			twoDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "oneAlias"}),
				EmptyClassPropertyMapping(),
			),
		})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)

		var mergedHost DirectiveMeta
		for _, d := range res.GetDirectivesOfNode(element) {
			if d.GetName() == "HostDir" {
				mergedHost = d
				break
			}
		}

		assert.NotNil(t, mergedHost)
		assert.Equal(t, MatchSourceHostDirective, mergedHost.GetMatchSource())
		inputsMap := mergedHost.GetInputs().(*ClassPropertyMappingGeneric).ToDirectMappedObject()
		assert.Equal(t, map[string]string{"one": "one"}, inputsMap)

		conflicts := res.GetConflictingHostDirectiveBindings(element)
		assert.Len(t, conflicts, 1)
		conflict := conflicts[0]
		assert.Equal(t, "input", conflict.Kind)
		assert.Equal(t, "one", conflict.ClassPropertyName)
		assert.Equal(t, map[string]bool{"one": true, "oneAlias": true}, conflict.ConflictingAliases)
	})

	t.Run("directive de-duplication: should not capture conflicting input bindings if they are equivalent", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			matchSource: MatchSourceHostDirective,
		})
		oneDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "OneDir",
			selector: "[dir]",
		})
		twoDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "TwoDir",
			selector: "[dir]",
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			oneDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "oneAlias"}),
				EmptyClassPropertyMapping(),
			),
		})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			twoDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "oneAlias"}),
				EmptyClassPropertyMapping(),
			),
		})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)

		var mergedHost DirectiveMeta
		for _, d := range res.GetDirectivesOfNode(element) {
			if d.GetName() == "HostDir" {
				mergedHost = d
				break
			}
		}

		assert.NotNil(t, mergedHost)
		assert.Equal(t, MatchSourceHostDirective, mergedHost.GetMatchSource())
		inputsMap := mergedHost.GetInputs().(*ClassPropertyMappingGeneric).ToDirectMappedObject()
		assert.Equal(t, map[string]string{"one": "oneAlias"}, inputsMap)
		assert.Nil(t, res.GetConflictingHostDirectiveBindings(element))
	})

	t.Run("directive de-duplication: should capture conflicting output bindings in host directives", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			matchSource: MatchSourceHostDirective,
		})
		oneDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "OneDir",
			selector: "[dir]",
		})
		twoDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "TwoDir",
			selector: "[dir]",
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			oneDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				EmptyClassPropertyMapping(),
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "one"}),
			),
		})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			twoDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				EmptyClassPropertyMapping(),
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "oneAlias"}),
			),
		})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)

		var mergedHost DirectiveMeta
		for _, d := range res.GetDirectivesOfNode(element) {
			if d.GetName() == "HostDir" {
				mergedHost = d
				break
			}
		}

		assert.NotNil(t, mergedHost)
		assert.Equal(t, MatchSourceHostDirective, mergedHost.GetMatchSource())
		outputsMap := mergedHost.GetOutputs().(*ClassPropertyMappingGeneric).ToDirectMappedObject()
		assert.Equal(t, map[string]string{"one": "one"}, outputsMap)

		conflicts := res.GetConflictingHostDirectiveBindings(element)
		assert.Len(t, conflicts, 1)
		conflict := conflicts[0]
		assert.Equal(t, "output", conflict.Kind)
		assert.Equal(t, "one", conflict.ClassPropertyName)
		assert.Equal(t, map[string]bool{"one": true, "oneAlias": true}, conflict.ConflictingAliases)
	})

	t.Run("directive de-duplication: should not capture conflicting output bindings if they are equivalent", func(t *testing.T) {
		matcher := NewSelectorMatcher[[]DirectiveMeta]()
		hostDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:        "HostDir",
			matchSource: MatchSourceHostDirective,
		})
		oneDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "OneDir",
			selector: "[dir]",
		})
		twoDir := makeDirectiveMeta(struct {
			name         string
			selector     string
			inputs       map[string]string
			outputs      map[string]string
			exportAs     []string
			isComponent  bool
			isStructural bool
			matchSource  MatchSource
		}{
			name:     "TwoDir",
			selector: "[dir]",
		})

		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			oneDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				EmptyClassPropertyMapping(),
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "oneAlias"}),
			),
		})
		matcher.AddSelectables(CssSelectorParse("[dir]"), []DirectiveMeta{
			twoDir,
			hostDir.(*mockDirectiveMeta).WithInputsAndOutputs(
				EmptyClassPropertyMapping(),
				ClassPropertyMappingFromMappedObject(map[string]interface{}{"one": "oneAlias"}),
			),
		})

		template := ParseTemplate("<div dir></div>", "", nil)
		binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
		res := binder.Bind(Target[DirectiveMeta]{Template: template.Nodes})
		element := template.Nodes[0].(*Element)

		var mergedHost DirectiveMeta
		for _, d := range res.GetDirectivesOfNode(element) {
			if d.GetName() == "HostDir" {
				mergedHost = d
				break
			}
		}

		assert.NotNil(t, mergedHost)
		assert.Equal(t, MatchSourceHostDirective, mergedHost.GetMatchSource())
		outputsMap := mergedHost.GetOutputs().(*ClassPropertyMappingGeneric).ToDirectMappedObject()
		assert.Equal(t, map[string]string{"one": "oneAlias"}, outputsMap)
		assert.Nil(t, res.GetConflictingHostDirectiveBindings(element))
	})
}
