package render3

import (
	"fmt"
	"reflect"
	"strings"
	"github.com/microsoft/typescript-go/angular-packages/compiler/expression_parser"
)

func diff(fullList []string, itemsToExclude []string) []string {
	exclude := make(map[string]bool)
	for _, item := range itemsToExclude {
		exclude[item] = true
	}
	var res []string
	for _, item := range fullList {
		if !exclude[item] {
			res = append(res, item)
		}
	}
	return res
}

type BindingsMap[DirectiveT any] map[any]any
type ReferenceMap[DirectiveT any] map[*Reference]any
type MatchedDirectives[DirectiveT any] map[DirectiveOwner][]DirectiveT
type ScopedNodeEntities map[ScopedNode]map[TemplateEntity]bool
type DeferBlockScope struct {
	Block *DeferredBlock
	Scope *Scope
}
type DeferBlockScopes []DeferBlockScope

type DirectivesResult struct {
	Regular         []string
	DeferCandidates []string
}

type PipesResult struct {
	Regular         []string
	DeferCandidates []string
}

type DirectivesPipesResult struct {
	Directives DirectivesResult
	Pipes      PipesResult
}

type FakePropertyMapping struct{}

func (f *FakePropertyMapping) HasBindingPropertyName(name string) bool { return false }
func (f *FakePropertyMapping) ToDirectMappedObject() map[string]string { return nil }
func (f *FakePropertyMapping) Iterator() []any                         { return nil }

type FakeDirective struct {
	Name        string
	Selector    string
	MatchSource MatchSource
	Inputs      any
	Outputs     any
	ExportAs    []string
}

func (f *FakeDirective) GetName() string                                        { return f.Name }
func (f *FakeDirective) GetRefKey() string                                      { return f.Name }
func (f *FakeDirective) GetSelector() *string                                   { return &f.Selector }
func (f *FakeDirective) IsComponent() bool                                      { return false }
func (f *FakeDirective) GetInputs() any                                         { return f.Inputs }
func (f *FakeDirective) GetOutputs() any                                        { return f.Outputs }
func (f *FakeDirective) GetExportAs() []string                                  { return f.ExportAs }
func (f *FakeDirective) IsStructural() bool                                     { return false }
func (f *FakeDirective) GetNgContentSelectors() []string                        { return nil }
func (f *FakeDirective) GetPreserveWhitespaces() bool                           { return false }
func (f *FakeDirective) GetAnimationTriggerNames() *LegacyAnimationTriggerNames { return nil }
func (f *FakeDirective) GetMatchSource() MatchSource                            { return f.MatchSource }

func FindMatchingDirectivesAndPipes(templateStr string, directiveSelectors []string) DirectivesPipesResult {
	matcher := NewSelectorMatcher[[]DirectiveMeta]()
	for _, selector := range directiveSelectors {
		fakeDirective := &FakeDirective{
			Name:        selector,
			Selector:    selector,
			MatchSource: MatchSourceSelector,
			Inputs:      &FakePropertyMapping{},
			Outputs:     &FakePropertyMapping{},
		}
		matcher.AddSelectables(CssSelectorParse(selector), []DirectiveMeta{fakeDirective})
	}
	parsedTemplate := ParseTemplate(templateStr, "", nil)
	binder := NewR3TargetBinder[DirectiveMeta](matcher, nil)
	bound := binder.Bind(Target[DirectiveMeta]{Template: parsedTemplate.Nodes})

	var eagerDirectiveSelectors []string
	for _, dir := range bound.GetEagerlyUsedDirectives() {
		if sel := dir.GetSelector(); sel != nil {
			eagerDirectiveSelectors = append(eagerDirectiveSelectors, *sel)
		}
	}
	var allMatchedDirectiveSelectors []string
	for _, dir := range bound.GetUsedDirectives() {
		if sel := dir.GetSelector(); sel != nil {
			allMatchedDirectiveSelectors = append(allMatchedDirectiveSelectors, *sel)
		}
	}
	eagerPipes := bound.GetEagerlyUsedPipes()

	var res DirectivesPipesResult
	res.Directives.Regular = eagerDirectiveSelectors
	res.Directives.DeferCandidates = diff(allMatchedDirectiveSelectors, eagerDirectiveSelectors)
	res.Pipes.Regular = eagerPipes
	res.Pipes.DeferCandidates = diff(bound.GetUsedPipes(), eagerPipes)
	return res
}

type R3TargetBinder[DirectiveT DirectiveMeta] struct {
	DirectiveMatcher        any
	ForeignComponentMatcher any
}

func NewR3TargetBinder[DirectiveT DirectiveMeta](directiveMatcher any, foreignComponentMatcher any) *R3TargetBinder[DirectiveT] {
	return &R3TargetBinder[DirectiveT]{
		DirectiveMatcher:        directiveMatcher,
		ForeignComponentMatcher: foreignComponentMatcher,
	}
}

func (b *R3TargetBinder[DirectiveT]) Bind(target Target[DirectiveT]) BoundTarget[DirectiveT] {
	if target.Template == nil && target.Host == nil {
		panic("Empty bound targets are not supported")
	}

	directives := make(MatchedDirectives[DirectiveT])
	foreignComponents := make(map[*Element]*ForeignComponentMeta)
	var eagerDirectives []DirectiveT
	missingDirectives := make(map[string]bool)
	bindings := make(BindingsMap[DirectiveT])
	references := make(ReferenceMap[DirectiveT])
	scopedNodeEntities := make(ScopedNodeEntities)
	expressions := make(map[expression_parser.AST]TemplateEntity)
	symbols := make(map[TemplateEntity]ScopedNode)
	nestingLevel := make(map[ScopedNode]int)
	usedPipes := make(map[string]bool)
	eagerPipes := make(map[string]bool)
	deferBlocks := make(DeferBlockScopes, 0)
	conflictingHostDirectiveBindings := make(map[DirectiveOwner][]ConflictingHostDirectiveBinding[DirectiveT])
	deferredNodes := make(map[any]bool)

	if target.Template != nil {
		scope := ApplyScope(target.Template)
		ExtractScopedNodeEntities(scope, scopedNodeEntities)

		ApplyDirectiveBinder(
			target.Template,
			b.DirectiveMatcher,
			b.ForeignComponentMatcher,
			directives,
			foreignComponents,
			&eagerDirectives,
			missingDirectives,
			bindings,
			references,
			conflictingHostDirectiveBindings,
		)

		ApplyTemplateBinderWithScope(
			target.Template,
			scope,
			expressions,
			symbols,
			nestingLevel,
			usedPipes,
			eagerPipes,
			&deferBlocks,
			deferredNodes,
		)
	}

	if target.Host != nil {
		directives[target.Host.Node] = target.Host.Directives
		ApplyTemplateBinderWithScope(
			[]Node{target.Host.Node},
			ApplyScope([]Node{target.Host.Node}),
			expressions,
			symbols,
			nestingLevel,
			usedPipes,
			eagerPipes,
			&deferBlocks,
			deferredNodes,
		)
	}

	return NewR3BoundTarget(
		target,
		directives,
		foreignComponents,
		eagerDirectives,
		missingDirectives,
		bindings,
		references,
		expressions,
		symbols,
		nestingLevel,
		scopedNodeEntities,
		usedPipes,
		eagerPipes,
		deferBlocks,
		conflictingHostDirectiveBindings,
		deferredNodes,
	)
}

type Scope struct {
	ParentScope        *Scope
	RootNode           ScopedNode
	NamedEntities      map[string]TemplateEntity
	ElementLikeInScope map[any]bool
	ChildScopes        map[ScopedNode]*Scope
	IsDeferred         bool
}

func NewRootScope() *Scope {
	return &Scope{
		NamedEntities:      make(map[string]TemplateEntity),
		ElementLikeInScope: make(map[any]bool),
		ChildScopes:        make(map[ScopedNode]*Scope),
	}
}

func NewScope(parentScope *Scope, rootNode ScopedNode) *Scope {
	s := &Scope{
		ParentScope:        parentScope,
		RootNode:           rootNode,
		NamedEntities:      make(map[string]TemplateEntity),
		ElementLikeInScope: make(map[any]bool),
		ChildScopes:        make(map[ScopedNode]*Scope),
	}
	if parentScope != nil && parentScope.IsDeferred {
		s.IsDeferred = true
	} else {
		_, isDeferredBlock := rootNode.(*DeferredBlock)
		s.IsDeferred = isDeferredBlock
	}
	return s
}

func ApplyScope(template any) *Scope {
	scope := NewRootScope()
	scope.Ingest(template)
	return scope
}

func (s *Scope) Ingest(nodeOrNodes any) {
	switch n := nodeOrNodes.(type) {
	case *Template:
		for _, v := range n.Variables {
			s.VisitVariable(v)
		}
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *IfBlockBranch:
		if n.ExpressionAlias != nil {
			s.VisitVariable(n.ExpressionAlias)
		}
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *ForLoopBlock:
		s.VisitVariable(n.Item)
		for _, v := range n.ContextVariables {
			s.VisitVariable(v)
		}
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *SwitchBlockCaseGroup:
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *ForLoopBlockEmpty:
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *DeferredBlock:
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *DeferredBlockError:
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *DeferredBlockPlaceholder:
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *DeferredBlockLoading:
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *Content:
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *HostElement:
	case []Node:
		for _, node := range n {
			node.Visit(s)
		}
	}
}

func (s *Scope) VisitElement(element *Element) any { s.visitElementLike(element); return nil }
func (s *Scope) VisitTemplate(template *Template) any {
	for _, d := range template.Directives {
		d.Visit(s)
	}
	for _, r := range template.References {
		s.VisitReference(r)
	}
	s.ingestScopedNode(template)
	return nil
}
func (s *Scope) VisitVariable(variable *Variable) any    { s.maybeDeclare(variable); return nil }
func (s *Scope) VisitReference(reference *Reference) any { s.maybeDeclare(reference); return nil }
func (s *Scope) VisitDeferredBlock(deferred *DeferredBlock) any {
	s.ingestScopedNode(deferred)
	if deferred.Placeholder != nil {
		deferred.Placeholder.Visit(s)
	}
	if deferred.Loading != nil {
		deferred.Loading.Visit(s)
	}
	if deferred.Error != nil {
		deferred.Error.Visit(s)
	}
	return nil
}
func (s *Scope) VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) any {
	s.ingestScopedNode(block)
	return nil
}
func (s *Scope) VisitDeferredBlockError(block *DeferredBlockError) any {
	s.ingestScopedNode(block)
	return nil
}
func (s *Scope) VisitDeferredBlockLoading(block *DeferredBlockLoading) any {
	s.ingestScopedNode(block)
	return nil
}
func (s *Scope) VisitSwitchBlock(block *SwitchBlock) any {
	for _, g := range block.Groups {
		g.Visit(s)
	}
	return nil
}
func (s *Scope) VisitSwitchBlockCase(block *SwitchBlockCase) any { return nil }
func (s *Scope) VisitSwitchBlockCaseGroup(block *SwitchBlockCaseGroup) any {
	s.ingestScopedNode(block)
	return nil
}
func (s *Scope) VisitSwitchExhaustiveCheck(block *SwitchExhaustiveCheck) any { return nil }
func (s *Scope) VisitForLoopBlock(block *ForLoopBlock) any {
	s.ingestScopedNode(block)
	if block.Empty != nil {
		block.Empty.Visit(s)
	}
	return nil
}
func (s *Scope) VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) any {
	s.ingestScopedNode(block)
	return nil
}
func (s *Scope) VisitIfBlock(block *IfBlock) any {
	for _, b := range block.Branches {
		b.Visit(s)
	}
	return nil
}
func (s *Scope) VisitIfBlockBranch(block *IfBlockBranch) any  { s.ingestScopedNode(block); return nil }
func (s *Scope) VisitContent(content *Content) any            { s.ingestScopedNode(content); return nil }
func (s *Scope) VisitLetDeclaration(decl *LetDeclaration) any { s.maybeDeclare(decl); return nil }
func (s *Scope) VisitComponent(component *Component) any      { s.visitElementLike(component); return nil }
func (s *Scope) VisitDirective(directive *Directive) any {
	for _, r := range directive.References {
		s.VisitReference(r)
	}
	return nil
}
func (s *Scope) VisitBoundAttribute(attr *BoundAttribute) any     { return nil }
func (s *Scope) VisitBoundEvent(event *BoundEvent) any            { return nil }
func (s *Scope) VisitBoundText(text *BoundText) any               { return nil }
func (s *Scope) VisitText(text *Text) any                         { return nil }
func (s *Scope) VisitTextAttribute(attr *TextAttribute) any       { return nil }
func (s *Scope) VisitIcu(icu *Icu) any                            { return nil }
func (s *Scope) VisitDeferredTrigger(trigger DeferredTrigger) any { return nil }
func (s *Scope) VisitUnknownBlock(block *UnknownBlock) any        { return nil }

func (s *Scope) visitElementLike(node any) {
	switch n := node.(type) {
	case *Element:
		for _, d := range n.Directives {
			d.Visit(s)
		}
		for _, r := range n.References {
			s.VisitReference(r)
		}
		for _, c := range n.Children {
			c.Visit(s)
		}
	case *Component:
		for _, d := range n.Directives {
			d.Visit(s)
		}
		for _, r := range n.References {
			s.VisitReference(r)
		}
		for _, c := range n.Children {
			c.Visit(s)
		}
	}
	s.ElementLikeInScope[node] = true
}

func (s *Scope) maybeDeclare(thing TemplateEntity) {
	if _, ok := s.NamedEntities[thing.GetName()]; !ok {
		s.NamedEntities[thing.GetName()] = thing
	}
}

func (s *Scope) Lookup(name string) TemplateEntity {
	if val, ok := s.NamedEntities[name]; ok {
		return val
	} else if s.ParentScope != nil {
		return s.ParentScope.Lookup(name)
	}
	return nil
}

func (s *Scope) GetChildScope(node ScopedNode) *Scope {
	if res, ok := s.ChildScopes[node]; ok {
		return res
	}
	panic("Assertion error: child scope not found")
}

func (s *Scope) ingestScopedNode(node ScopedNode) {
	scope := NewScope(s, node)
	scope.Ingest(node)
	s.ChildScopes[node] = scope
}

type directiveBinder[DirectiveT DirectiveMeta] struct {
	directiveMatcher    any
	foreignMatcher      *SelectorlessMatcher[ForeignComponentMeta]
	directives          MatchedDirectives[DirectiveT]
	foreignComponents  map[*Element]*ForeignComponentMeta
	eagerDirectives     *[]DirectiveT
	missingDirectives   map[string]bool
	bindings            BindingsMap[DirectiveT]
	references          ReferenceMap[DirectiveT]
	conflictingBindings map[DirectiveOwner][]ConflictingHostDirectiveBinding[DirectiveT]
	isInDeferBlock      bool
}

func ApplyDirectiveBinder[DirectiveT DirectiveMeta](
	template []Node,
	directiveMatcher any,
	foreignMatcher any,
	directives MatchedDirectives[DirectiveT],
	foreignComponents map[*Element]*ForeignComponentMeta,
	eagerDirectives *[]DirectiveT,
	missingDirectives map[string]bool,
	bindings BindingsMap[DirectiveT],
	references ReferenceMap[DirectiveT],
	conflictingHostDirectiveBindings map[DirectiveOwner][]ConflictingHostDirectiveBinding[DirectiveT],
) {
	var fMatcher *SelectorlessMatcher[ForeignComponentMeta]
	if foreignMatcher != nil {
		if fm, ok := foreignMatcher.(*SelectorlessMatcher[ForeignComponentMeta]); ok {
			fMatcher = fm
		}
	}

	db := &directiveBinder[DirectiveT]{
		directiveMatcher:    directiveMatcher,
		foreignMatcher:      fMatcher,
		directives:          directives,
		foreignComponents:  foreignComponents,
		eagerDirectives:     eagerDirectives,
		missingDirectives:   missingDirectives,
		bindings:            bindings,
		references:          references,
		conflictingBindings: conflictingHostDirectiveBindings,
	}
	db.walk(template)
}

func (db *directiveBinder[DirectiveT]) walk(nodes []Node) {
	for _, node := range nodes {
		db.visitNode(node)
	}
}

func (db *directiveBinder[DirectiveT]) visitNode(node Node) {
	switch n := node.(type) {
	case *Element:
		db.visitElementOrTemplate(n, n.Name, n.Attributes, n.Inputs, n.Outputs, n.References, n.Children, n.Directives)
	case *Template:
		tagName := "ng-template"
		var attrs []*TextAttribute
		var inputs []*BoundAttribute
		var outputs []*BoundEvent

		isImplicit := n.TagName != nil && *n.TagName != "" && *n.TagName != "ng-template"

		if isImplicit {
			for _, attrNode := range n.TemplateAttrs {
				switch a := attrNode.(type) {
				case *TextAttribute:
					attrs = append(attrs, a)
				case *BoundAttribute:
					inputs = append(inputs, a)
				}
			}
		} else {
			attrs = make([]*TextAttribute, len(n.Attributes))
			copy(attrs, n.Attributes)
			inputs = make([]*BoundAttribute, len(n.Inputs))
			copy(inputs, n.Inputs)
			outputs = make([]*BoundEvent, len(n.Outputs))
			copy(outputs, n.Outputs)

			for _, attrNode := range n.TemplateAttrs {
				switch a := attrNode.(type) {
				case *TextAttribute:
					attrs = append(attrs, a)
				case *BoundAttribute:
					inputs = append(inputs, a)
				}
			}
		}

		db.visitElementOrTemplate(n, tagName, attrs, inputs, outputs, n.References, n.Children, n.Directives)
	case *Component:
		var matched []DirectiveT
		if db.directiveMatcher != nil {
			if m, ok := db.directiveMatcher.(*SelectorMatcher[[]DirectiveT]); ok {
				cssSel := db.createCssSelector(n, n.ComponentName, nil, nil, nil)
				m.Match(cssSel, func(c *CssSelector, res []DirectiveT) {
					matched = append(matched, res...)
				})
			} else if sm, ok := db.directiveMatcher.(*SelectorlessMatcher[DirectiveT]); ok {
				matched = sm.MatchByName(n.ComponentName)
			}
		}
		matched = db.dedupeAndMergeDirectives(n, matched)

		if len(matched) > 0 {
			db.directives[n] = matched
			if !db.isInDeferBlock {
				*db.eagerDirectives = append(*db.eagerDirectives, matched...)
			}
		} else if db.foreignMatcher == nil {
			db.missingDirectives[n.ComponentName] = true
		}

		for _, ref := range n.References {
			if ref.Value == "" {
				db.references[ref] = n
			} else {
				var target any = n
				for _, dir := range matched {
					if db.hasExportAs(dir, ref.Value) {
						target = dir
						break
					}
				}
				db.references[ref] = target
			}
		}

		for _, input := range n.Inputs {
			var consumer any = n
			if len(matched) > 0 {
				if (input.Type == 0 || input.Type == 5) && db.hasInput(matched[0], input.Name) {
					consumer = matched[0]
				} else {
					consumer = nil
					if input.Type == 0 || input.Type == 5 {
						for _, dir := range matched[1:] {
							if db.hasInput(dir, input.Name) {
								consumer = dir
								break
							}
						}
					}
				}
			}
			db.bindings[input] = consumer
		}
		for _, attr := range n.Attributes {
			var consumer any = n
			if len(matched) > 0 {
				if db.hasInput(matched[0], attr.Name) {
					consumer = matched[0]
				} else {
					consumer = nil
					for _, dir := range matched[1:] {
						if db.hasInput(dir, attr.Name) {
							consumer = dir
							break
						}
					}
				}
			}
			db.bindings[attr] = consumer
		}
		for _, output := range n.Outputs {
			var consumer any = n
			if len(matched) > 0 {
				if db.hasOutput(matched[0], output.Name) {
					consumer = matched[0]
				} else {
					consumer = nil
					for _, dir := range matched[1:] {
						if db.hasOutput(dir, output.Name) {
							consumer = dir
							break
						}
					}
				}
			}
			db.bindings[output] = consumer
		}

		for _, d := range n.Directives {
			db.visitNode(d)
		}
		db.walk(n.Children)
	case *Directive:
		var matched []DirectiveT
		if db.directiveMatcher != nil {
			if sm, ok := db.directiveMatcher.(*SelectorlessMatcher[DirectiveT]); ok {
				matched = sm.MatchByName(n.Name)
			}
		}
		matched = db.dedupeAndMergeDirectives(n, matched)
		if len(matched) > 0 {
			db.directives[n] = matched
			if !db.isInDeferBlock {
				*db.eagerDirectives = append(*db.eagerDirectives, matched...)
			}
		} else {
			db.missingDirectives[n.Name] = true
		}

		for _, ref := range n.References {
			db.references[ref] = n
		}
		for _, input := range n.Inputs {
			var consumer any = nil
			if len(matched) > 0 && db.hasInput(matched[0], input.Name) {
				consumer = matched[0]
			}
			db.bindings[input] = consumer
		}
		for _, attr := range n.Attributes {
			var consumer any = nil
			if len(matched) > 0 && db.hasInput(matched[0], attr.Name) {
				consumer = matched[0]
			}
			db.bindings[attr] = consumer
		}
		for _, output := range n.Outputs {
			var consumer any = nil
			if len(matched) > 0 && db.hasOutput(matched[0], output.Name) {
				consumer = matched[0]
			}
			db.bindings[output] = consumer
		}
	case *DeferredBlock:
		wasInDefer := db.isInDeferBlock
		db.isInDeferBlock = true
		db.walk(n.Children)
		db.isInDeferBlock = wasInDefer

		if n.Placeholder != nil {
			db.visitNode(n.Placeholder)
		}
		if n.Loading != nil {
			db.visitNode(n.Loading)
		}
		if n.Error != nil {
			db.visitNode(n.Error)
		}
	case *DeferredBlockPlaceholder:
		db.walk(n.Children)
	case *DeferredBlockLoading:
		db.walk(n.Children)
	case *DeferredBlockError:
		db.walk(n.Children)
	case *IfBlock:
		for _, branch := range n.Branches {
			db.visitNode(branch)
		}
	case *IfBlockBranch:
		db.walk(n.Children)
	case *ForLoopBlock:
		db.walk(n.Children)
		if n.Empty != nil {
			db.visitNode(n.Empty)
		}
	case *ForLoopBlockEmpty:
		db.walk(n.Children)
	case *SwitchBlock:
		for _, group := range n.Groups {
			db.visitNode(group)
		}
	case *SwitchBlockCaseGroup:
		db.walk(n.Children)
	case *Content:
		db.walk(n.Children)
	}
}

func (db *directiveBinder[DirectiveT]) visitElementOrTemplate(
	node DirectiveOwner,
	tagName string,
	attributes []*TextAttribute,
	inputs []*BoundAttribute,
	outputs []*BoundEvent,
	references []*Reference,
	children []Node,
	directives []*Directive,
) {
	var matched []DirectiveT
	if db.directiveMatcher != nil {
		m, ok := db.directiveMatcher.(*SelectorMatcher[[]DirectiveT])
		if ok {
			cssSel := db.createCssSelector(node, tagName, attributes, inputs, outputs)
			m.Match(cssSel, func(c *CssSelector, res []DirectiveT) {
				matched = append(matched, res...)
			})
		} else if sm, ok := db.directiveMatcher.(*SelectorlessMatcher[DirectiveT]); ok {
			if _, isEl := node.(*Element); !isEl {
				// Find matches by selectorless annotations inside node.directives
				for _, d := range directives {
					matches := sm.MatchByName(d.Name)
					matched = append(matched, matches...)
				}
			}
		}
	}
	matched = db.dedupeAndMergeDirectives(node, matched)

	if db.foreignMatcher != nil {
		if el, ok := node.(*Element); ok {
			foreignMatches := db.foreignMatcher.MatchByName(el.Name)
			if len(foreignMatches) > 0 {
				if len(matched) > 0 {
					panic(fmt.Sprintf("Conflict: Element '%s' matches both an Angular directive and a foreign component.", el.Name))
				}
				db.foreignComponents[el] = &foreignMatches[0]
			}
		}
	}

	if len(matched) > 0 {
		db.directives[node] = matched
		if !db.isInDeferBlock {
			*db.eagerDirectives = append(*db.eagerDirectives, matched...)
		}
	}

	for _, input := range inputs {
		var consumer any = node
		if input.Type == 0 || input.Type == 5 {
			for _, dir := range matched {
				if db.hasInput(dir, input.Name) {
					consumer = dir
					break
				}
			}
		}
		db.bindings[input] = consumer
	}
	for _, attr := range attributes {
		var consumer any = node
		for _, dir := range matched {
			if db.hasInput(dir, attr.Name) {
				consumer = dir
				break
			}
		}
		db.bindings[attr] = consumer
	}
	for _, output := range outputs {
		var consumer any = node
		for _, dir := range matched {
			if db.hasOutput(dir, output.Name) {
				consumer = dir
				break
			}
		}
		db.bindings[output] = consumer
	}

	for _, ref := range references {
		if strings.TrimSpace(ref.Value) == "" {
			var target any = node
			for _, dir := range matched {
				if dir.IsComponent() {
					target = dir
					break
				}
			}
			db.references[ref] = target
		} else {
			var target any = node
			for _, dir := range matched {
				if db.hasExportAs(dir, ref.Value) {
					target = dir
					break
				}
			}
			db.references[ref] = target
		}
	}

	for _, d := range directives {
		db.visitNode(d)
	}

	db.walk(children)
}

func (db *directiveBinder[DirectiveT]) hasInput(dir DirectiveMeta, name string) bool {
	inputs := dir.GetInputs()
	if pm, ok := inputs.(interface{ HasBindingPropertyName(string) bool }); ok {
		return pm.HasBindingPropertyName(name)
	}
	return false
}

func (db *directiveBinder[DirectiveT]) hasOutput(dir DirectiveMeta, name string) bool {
	outputs := dir.GetOutputs()
	if pm, ok := outputs.(interface{ HasBindingPropertyName(string) bool }); ok {
		return pm.HasBindingPropertyName(name)
	}
	return false
}

func (db *directiveBinder[DirectiveT]) hasExportAs(dir DirectiveMeta, name string) bool {
	for _, ea := range dir.GetExportAs() {
		if ea == name {
			return true
		}
	}
	return false
}

func (db *directiveBinder[DirectiveT]) createCssSelector(
	node DirectiveOwner,
	tagName string,
	attributes []*TextAttribute,
	inputs []*BoundAttribute,
	outputs []*BoundEvent,
) *CssSelector {
	cssSelector := NewCssSelector()
	elementNameNoNs := splitNsName(tagName)[1]
	cssSelector.SetElement(elementNameNoNs)

	for _, attr := range attributes {
		if attr.Name == "i18n" || strings.HasPrefix(attr.Name, "i18n-") {
			continue
		}
		nameNoNs := splitNsName(attr.Name)[1]
		cssSelector.AddAttribute(nameNoNs, attr.Value)
		if strings.ToLower(attr.Name) == "class" {
			classes := strings.Fields(attr.Value)
			for _, className := range classes {
				cssSelector.AddClassName(className)
			}
		}
	}
	for _, input := range inputs {
		nameNoNs := splitNsName(input.Name)[1]
		cssSelector.AddAttribute(nameNoNs, "")
	}
	for _, output := range outputs {
		nameNoNs := splitNsName(output.Name)[1]
		cssSelector.AddAttribute(nameNoNs, "")
	}
	return cssSelector
}

func splitNsName(name string) [2]string {
	if len(name) > 0 && name[0] == ':' {
		parts := strings.SplitN(name[1:], ":", 2)
		if len(parts) == 2 {
			return [2]string{parts[0], parts[1]}
		}
		return [2]string{"", name[1:]}
	}
	parts := strings.SplitN(name, ":", 2)
	if len(parts) == 2 {
		return [2]string{parts[0], parts[1]}
	}
	return [2]string{"", name}
}


type templateBinder struct {
	scope         *Scope
	expressions   map[expression_parser.AST]TemplateEntity
	symbols       map[TemplateEntity]ScopedNode
	nestingLevel  map[ScopedNode]int
	usedPipes     map[string]bool
	eagerPipes    map[string]bool
	deferBlocks   *DeferBlockScopes
	deferredNodes map[any]bool
}

func ApplyTemplateBinderWithScope(
	nodeOrNodes any,
	scope *Scope,
	expressions map[expression_parser.AST]TemplateEntity,
	symbols map[TemplateEntity]ScopedNode,
	nestingLevel map[ScopedNode]int,
	usedPipes map[string]bool,
	eagerPipes map[string]bool,
	deferBlocks *DeferBlockScopes,
	deferredNodes map[any]bool,
) {
	tb := &templateBinder{
		scope:         scope,
		expressions:   expressions,
		symbols:       symbols,
		nestingLevel:  nestingLevel,
		usedPipes:     usedPipes,
		eagerPipes:    eagerPipes,
		deferBlocks:   deferBlocks,
		deferredNodes: deferredNodes,
	}
	tb.walk(nodeOrNodes)
}

func (tb *templateBinder) walk(nodeOrNodes any) {
	switch val := nodeOrNodes.(type) {
	case []Node:
		for _, node := range val {
			tb.visitNode(node)
		}
	case Node:
		tb.visitNode(val)
	}
}

func (tb *templateBinder) visitNode(node Node) {
	if tb.scope.IsDeferred {
		tb.deferredNodes[node] = true
	}

	switch n := node.(type) {
	case *Element:
		tb.walk(n.Children)
		for _, input := range n.Inputs {
			tb.resolveExpression(input.Value)
		}
		for _, output := range n.Outputs {
			tb.resolveExpression(output.Handler)
		}
	case *Template:
		level := 1
		curr := tb.scope
		for curr != nil {
			if _, ok := curr.RootNode.(*Template); ok {
				level++
			}
			curr = curr.ParentScope
		}
		tb.nestingLevel[n] = level

		for _, v := range n.Variables {
			tb.symbols[v] = n
		}
		for _, r := range n.References {
			tb.symbols[r] = n
		}

		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)

		for _, input := range n.Inputs {
			tb.resolveExpression(input.Value)
		}
		for _, attrNode := range n.TemplateAttrs {
			if ba, ok := attrNode.(*BoundAttribute); ok {
				tb.resolveExpression(ba.Value)
			}
		}
		for _, output := range n.Outputs {
			tb.resolveExpression(output.Handler)
		}
	case *LetDeclaration:
		tb.symbols[n] = tb.scope.RootNode
		tb.resolveExpression(n.Value)
	case *DeferredBlock:
		blockScope := DeferBlockScope{Block: n, Scope: tb.scope}
		*tb.deferBlocks = append(*tb.deferBlocks, blockScope)

		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)

		if n.Placeholder != nil {
			tb.visitNode(n.Placeholder)
		}
		if n.Loading != nil {
			tb.visitNode(n.Loading)
		}
		if n.Error != nil {
			tb.visitNode(n.Error)
		}
	case *DeferredBlockPlaceholder:
		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
	case *DeferredBlockLoading:
		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
	case *DeferredBlockError:
		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
	case *IfBlock:
		for _, branch := range n.Branches {
			tb.visitNode(branch)
		}
	case *IfBlockBranch:
		if n.ExpressionAlias != nil {
			tb.symbols[n.ExpressionAlias] = n
		}

		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
	case *ForLoopBlock:
		tb.symbols[n.Item] = n
		for _, v := range n.ContextVariables {
			tb.symbols[v] = n
		}

		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
		if n.Empty != nil {
			tb.visitNode(n.Empty)
		}
	case *ForLoopBlockEmpty:
		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
	case *SwitchBlock:
		for _, group := range n.Groups {
			tb.visitNode(group)
		}
	case *SwitchBlockCaseGroup:
		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
	case *Content:
		childScope := tb.scope.GetChildScope(n)
		childBinder := &templateBinder{
			scope:         childScope,
			expressions:   tb.expressions,
			symbols:       tb.symbols,
			nestingLevel:  tb.nestingLevel,
			usedPipes:     tb.usedPipes,
			eagerPipes:    tb.eagerPipes,
			deferBlocks:   tb.deferBlocks,
			deferredNodes: tb.deferredNodes,
		}
		childBinder.walk(n.Children)
	case *BoundText:
		tb.resolveExpression(n.Value)
	case *Icu:
		for _, expr := range n.Vars {
			tb.resolveExpression(expr.Value)
		}
		for _, expr := range n.Placeholders {
			tb.visitNode(expr)
		}
	}
}

func (tb *templateBinder) resolveExpression(ast expression_parser.AST) {
	if ast == nil {
		return
	}
	if aws, ok := ast.(*expression_parser.ASTWithSource); ok {
		tb.resolveAST(aws.Ast)
		return
	}
	tb.resolveAST(ast)
}

func (tb *templateBinder) resolveAST(ast expression_parser.AST) {
	if ast == nil {
		return
	}
	switch expr := ast.(type) {
	case *expression_parser.ASTWithSource:
		tb.resolveAST(expr.Ast)
	case *expression_parser.Interpolation:
		for _, exp := range expr.Expressions {
			tb.resolveAST(exp)
		}
	case *expression_parser.BindingPipe:
		tb.usedPipes[expr.Name] = true
		if !tb.scope.IsDeferred {
			tb.eagerPipes[expr.Name] = true
		}
		tb.resolveAST(expr.Exp)
		for _, arg := range expr.Args {
			tb.resolveAST(arg)
		}
	case *expression_parser.PropertyRead:
		if _, ok := expr.Receiver.(*expression_parser.ImplicitReceiver); ok {
			if entity := tb.scope.Lookup(expr.Name); entity != nil {
				tb.expressions[expr] = entity
			}
		} else {
			tb.resolveAST(expr.Receiver)
		}
	case *expression_parser.PropertyWrite:
		if _, ok := expr.Receiver.(*expression_parser.ImplicitReceiver); ok {
			if entity := tb.scope.Lookup(expr.Name); entity != nil {
				tb.expressions[expr] = entity
			}
		} else {
			tb.resolveAST(expr.Receiver)
		}
		tb.resolveAST(expr.Value)
	case *expression_parser.SafePropertyRead:
		if _, ok := expr.Receiver.(*expression_parser.ImplicitReceiver); ok {
			if entity := tb.scope.Lookup(expr.Name); entity != nil {
				tb.expressions[expr] = entity
			}
		} else {
			tb.resolveAST(expr.Receiver)
		}
	case *expression_parser.MethodCall:
		if _, ok := expr.Receiver.(*expression_parser.ImplicitReceiver); ok {
			if entity := tb.scope.Lookup(expr.Name); entity != nil {
				tb.expressions[expr] = entity
			}
		} else {
			tb.resolveAST(expr.Receiver)
		}
		for _, arg := range expr.Args {
			tb.resolveAST(arg)
		}
	case *expression_parser.SafeMethodCall:
		if _, ok := expr.Receiver.(*expression_parser.ImplicitReceiver); ok {
			if entity := tb.scope.Lookup(expr.Name); entity != nil {
				tb.expressions[expr] = entity
			}
		} else {
			tb.resolveAST(expr.Receiver)
		}
		for _, arg := range expr.Args {
			tb.resolveAST(arg)
		}
	case *expression_parser.Binary:
		tb.resolveAST(expr.Left)
		tb.resolveAST(expr.Right)
	case *expression_parser.Conditional:
		tb.resolveAST(expr.Condition)
		tb.resolveAST(expr.TrueExp)
		tb.resolveAST(expr.FalseExp)
	case *expression_parser.PrefixNot:
		tb.resolveAST(expr.Expression)
	case *expression_parser.Chain:
		for _, exp := range expr.Expressions {
			tb.resolveAST(exp)
		}
	case *expression_parser.Call:
		tb.resolveAST(expr.Receiver)
		for _, arg := range expr.Args {
			tb.resolveAST(arg)
		}
	case *expression_parser.LiteralArray:
		for _, exp := range expr.Expressions {
			tb.resolveAST(exp)
		}
	case *expression_parser.LiteralMap:
		for _, exp := range expr.Values {
			tb.resolveAST(exp)
		}
	}
}

func NewR3BoundTarget[DirectiveT DirectiveMeta](
	target Target[DirectiveT],
	directives MatchedDirectives[DirectiveT],
	foreignComponents map[*Element]*ForeignComponentMeta,
	eagerDirectives []DirectiveT,
	missingDirectives map[string]bool,
	bindings BindingsMap[DirectiveT],
	references ReferenceMap[DirectiveT],
	exprTargets map[expression_parser.AST]TemplateEntity,
	symbols map[TemplateEntity]ScopedNode,
	nestingLevel map[ScopedNode]int,
	scopedNodeEntities ScopedNodeEntities,
	usedPipes map[string]bool,
	eagerPipes map[string]bool,
	rawDeferred DeferBlockScopes,
	conflictingHostDirectiveBindings map[DirectiveOwner][]ConflictingHostDirectiveBinding[DirectiveT],
	deferredNodes map[any]bool,
) *R3BoundTargetImpl[DirectiveT] {
	return &R3BoundTargetImpl[DirectiveT]{
		target:                           target,
		directives:                       directives,
		foreignComponents:               foreignComponents,
		eagerDirectives:                  eagerDirectives,
		missingDirectives:                missingDirectives,
		bindings:                         bindings,
		references:                       references,
		exprTargets:                      exprTargets,
		symbols:                          symbols,
		nestingLevel:                     nestingLevel,
		scopedNodeEntities:               scopedNodeEntities,
		usedPipes:                        usedPipes,
		eagerPipes:                       eagerPipes,
		deferBlocks:                      rawDeferred,
		conflictingHostDirectiveBindings: conflictingHostDirectiveBindings,
		deferredNodes:                    deferredNodes,
	}
}

type R3BoundTargetImpl[DirectiveT DirectiveMeta] struct {
	target                           Target[DirectiveT]
	directives                       MatchedDirectives[DirectiveT]
	foreignComponents               map[*Element]*ForeignComponentMeta
	eagerDirectives                  []DirectiveT
	missingDirectives                map[string]bool
	bindings                         BindingsMap[DirectiveT]
	references                       ReferenceMap[DirectiveT]
	exprTargets                      map[expression_parser.AST]TemplateEntity
	symbols                          map[TemplateEntity]ScopedNode
	nestingLevel                     map[ScopedNode]int
	scopedNodeEntities               ScopedNodeEntities
	usedPipes                        map[string]bool
	eagerPipes                       map[string]bool
	deferBlocks                      DeferBlockScopes
	conflictingHostDirectiveBindings map[DirectiveOwner][]ConflictingHostDirectiveBinding[DirectiveT]
	deferredNodes                    map[any]bool
}

func (r *R3BoundTargetImpl[DirectiveT]) GetTarget() Target[DirectiveT] { return r.target }
func (r *R3BoundTargetImpl[DirectiveT]) GetDirectivesOfNode(node DirectiveOwner) []DirectiveT {
	if dirs, ok := r.directives[node]; ok {
		return dirs
	}
	return nil
}
func (r *R3BoundTargetImpl[DirectiveT]) GetForeignComponent(element *Element) *ForeignComponentMeta {
	return r.foreignComponents[element]
}
func (r *R3BoundTargetImpl[DirectiveT]) GetReferenceTarget(ref *Reference) *ReferenceTarget[DirectiveT] {
	targetVal, ok := r.references[ref]
	if !ok {
		return nil
	}
	res := &ReferenceTarget[DirectiveT]{}
	if el, ok := targetVal.(*Element); ok {
		res.Element = el
		res.Node = el
	} else if tmpl, ok := targetVal.(*Template); ok {
		res.Template = tmpl
		res.Node = tmpl
	} else if cmp, ok := targetVal.(*Component); ok {
		virtualEl := &Element{Name: cmp.ComponentName}
		res.Element = virtualEl
		res.Node = cmp
	} else if dir, ok := targetVal.(*Directive); ok {
		res.Node = dir
		for _, dirs := range r.directives {
			for _, d := range dirs {
				if d.GetName() == dir.Name {
					res.Directive = d
					break
				}
			}
		}
	} else if d, ok := targetVal.(DirectiveT); ok {
		res.Directive = d
		for node, dirs := range r.directives {
			for _, dir := range dirs {
				if dir.GetRefKey() == d.GetRefKey() {
					res.Node = node
					if el, ok := node.(*Element); ok {
						res.Element = el
					} else if tmpl, ok := node.(*Template); ok {
						res.Template = tmpl
					} else if cmp, ok := node.(*Component); ok {
						res.Element = &Element{Name: cmp.ComponentName}
					}
					break
				}
			}
		}
	}
	return res
}
func (r *R3BoundTargetImpl[DirectiveT]) GetConsumerOfBinding(binding any) any {
	if consumer, ok := r.bindings[binding]; ok {
		return consumer
	}
	var bindingName string
	switch b := binding.(type) {
	case *BoundAttribute:
		bindingName = b.Name
	case *TextAttribute:
		bindingName = b.Name
	case *BoundEvent:
		bindingName = b.Name
	}
	if bindingName != "" {
		for k, consumer := range r.bindings {
			switch b := k.(type) {
			case *BoundAttribute:
				if b.Name == bindingName {
					return consumer
				}
			case *TextAttribute:
				if b.Name == bindingName {
					return consumer
				}
			case *BoundEvent:
				if b.Name == bindingName {
					return consumer
				}
			}
		}
	}
	return nil
}
func (r *R3BoundTargetImpl[DirectiveT]) GetExpressionTarget(expr expression_parser.AST) TemplateEntity {
	if aws, ok := expr.(*expression_parser.ASTWithSource); ok {
		expr = aws.Ast
	}
	if target, ok := r.exprTargets[expr]; ok {
		return target
	}
	if pr, ok := expr.(*expression_parser.PropertyRead); ok {
		for k, v := range r.exprTargets {
			if kPr, ok := k.(*expression_parser.PropertyRead); ok && kPr.Name == pr.Name {
				return v
			}
		}
	}
	return nil
}
func (r *R3BoundTargetImpl[DirectiveT]) GetDefinitionNodeOfSymbol(symbol TemplateEntity) ScopedNode {
	if node, ok := r.symbols[symbol]; ok {
		return node
	}
	for k, v := range r.symbols {
		if k.GetName() == symbol.GetName() {
			return v
		}
	}
	return nil
}
func (r *R3BoundTargetImpl[DirectiveT]) GetNestingLevel(node ScopedNode) int {
	return r.nestingLevel[node]
}
func (r *R3BoundTargetImpl[DirectiveT]) GetEntitiesInScope(node ScopedNode) []TemplateEntity {
	var res []TemplateEntity
	if entities, ok := r.scopedNodeEntities[node]; ok {
		for e := range entities {
			res = append(res, e)
		}
	} else if node == nil {
		for k, entities := range r.scopedNodeEntities {
			if k == nil {
				for e := range entities {
					res = append(res, e)
				}
			}
		}
	}
	return res
}
func (r *R3BoundTargetImpl[DirectiveT]) GetUsedDirectives() []DirectiveT {
	var res []DirectiveT
	seen := make(map[string]bool)
	for _, dirs := range r.directives {
		for _, d := range dirs {
			if !seen[d.GetName()] {
				seen[d.GetName()] = true
				res = append(res, d)
			}
		}
	}
	return res
}
func (r *R3BoundTargetImpl[DirectiveT]) GetEagerlyUsedDirectives() []DirectiveT {
	var res []DirectiveT
	seen := make(map[string]bool)
	for _, d := range r.eagerDirectives {
		if !seen[d.GetName()] {
			seen[d.GetName()] = true
			res = append(res, d)
		}
	}
	return res
}
func (r *R3BoundTargetImpl[DirectiveT]) GetUsedPipes() []string {
	var res []string
	for p := range r.usedPipes {
		res = append(res, p)
	}
	return res
}
func (r *R3BoundTargetImpl[DirectiveT]) GetEagerlyUsedPipes() []string {
	var res []string
	for p := range r.eagerPipes {
		res = append(res, p)
	}
	return res
}
func (r *R3BoundTargetImpl[DirectiveT]) GetDeferBlocks() []*DeferredBlock {
	var res []*DeferredBlock
	for _, b := range r.deferBlocks {
		res = append(res, b.Block)
	}
	return res
}
func (r *R3BoundTargetImpl[DirectiveT]) GetDeferredTriggerTarget(block *DeferredBlock, trigger DeferredTrigger) *Element {
	var triggerName string
	switch t := trigger.(type) {
	case *ViewportDeferredTrigger:
		if t.Reference != nil {
			triggerName = *t.Reference
		}
	case *HoverDeferredTrigger:
		if t.Reference != nil {
			triggerName = *t.Reference
		}
	case *InteractionDeferredTrigger:
		if t.Reference != nil {
			triggerName = *t.Reference
		}
	}
	
	if triggerName == "" {
		if block.Placeholder != nil {
			var rootEl *Element
			count := 0
			for _, child := range block.Placeholder.Children {
				if _, isComment := child.(*Comment); isComment {
					continue
				}
				if _, isText := child.(*Text); isText {
					continue
				}
				if el, ok := child.(*Element); ok {
					rootEl = el
					count++
				} else if cmp, ok := child.(*Component); ok {
					rootEl = &Element{Name: cmp.ComponentName}
					count++
				} else {
					count++
				}
			}
			if count == 1 && rootEl != nil {
				return rootEl
			}
		}
		return nil
	}
	
	var blockScope *Scope
	for _, bs := range r.deferBlocks {
		if bs.Block == block {
			blockScope = bs.Scope
			break
		}
	}
	if blockScope == nil {
		return nil
	}
	
	entity := blockScope.Lookup(triggerName)
	if entity == nil {
		if block.Placeholder != nil {
			if ps, ok := blockScope.ChildScopes[block.Placeholder]; ok {
				entity = ps.Lookup(triggerName)
			}
		}
		if entity == nil && block.Loading != nil {
			if ls, ok := blockScope.ChildScopes[block.Loading]; ok {
				entity = ls.Lookup(triggerName)
			}
		}
		if entity == nil && block.Error != nil {
			if es, ok := blockScope.ChildScopes[block.Error]; ok {
				entity = es.Lookup(triggerName)
			}
		}
	}
	if ref, ok := entity.(*Reference); ok {
		targetVal := r.references[ref]
		if el, ok := targetVal.(*Element); ok {
			return el
		}
		if cmp, ok := targetVal.(*Component); ok {
			return &Element{Name: cmp.ComponentName}
		}
		if d, ok := targetVal.(DirectiveT); ok {
			for node, dirs := range r.directives {
				for _, dir := range dirs {
					if dir.GetRefKey() == d.GetRefKey() {
						if el, ok := node.(*Element); ok {
							return el
						} else if cmp, ok := node.(*Component); ok {
							return &Element{Name: cmp.ComponentName}
						}
					}
				}
			}
		}
	}
	return nil
}
func (r *R3BoundTargetImpl[DirectiveT]) IsDeferred(node *Element) bool {
	return r.deferredNodes[node]
}
func (r *R3BoundTargetImpl[DirectiveT]) ReferencedDirectiveExists(name string) bool {
	for _, dirs := range r.directives {
		for _, d := range dirs {
			if d.GetName() == name {
				return true
			}
		}
	}
	return false
}
func (r *R3BoundTargetImpl[DirectiveT]) GetConflictingHostDirectiveBindings(node DirectiveOwner) []ConflictingHostDirectiveBinding[DirectiveT] {
	return r.conflictingHostDirectiveBindings[node]
}

func ExtractScopedNodeEntities(rootScope *Scope, templateEntities ScopedNodeEntities) {
	var traverse func(s *Scope)
	traverse = func(s *Scope) {
		entities := make(map[TemplateEntity]bool)
		for _, entity := range s.NamedEntities {
			entities[entity] = true
		}
		if s.ParentScope != nil {
			if parentEntities, ok := templateEntities[s.ParentScope.RootNode]; ok {
				for pe := range parentEntities {
					entities[pe] = true
				}
			}
		}
		templateEntities[s.RootNode] = entities

		for _, child := range s.ChildScopes {
			traverse(child)
		}
	}
	traverse(rootScope)
}

func mergeDirectiveMeta[DirectiveT DirectiveMeta](dir DirectiveT, inputs any, outputs any) DirectiveT {
	if withIO, ok := any(dir).(interface {
		WithInputsAndOutputs(inputs any, outputs any) DirectiveT
	}); ok {
		return withIO.WithInputsAndOutputs(inputs, outputs)
	}

	val := reflect.ValueOf(dir)
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct {
		elem := val.Elem()
		newPtr := reflect.New(elem.Type())
		newElem := newPtr.Elem()
		newElem.Set(elem)

		for _, fieldName := range []string{"inputs", "Inputs"} {
			f := newElem.FieldByName(fieldName)
			if f.IsValid() && f.CanSet() {
				f.Set(reflect.ValueOf(inputs))
				break
			}
		}
		for _, fieldName := range []string{"outputs", "Outputs"} {
			f := newElem.FieldByName(fieldName)
			if f.IsValid() && f.CanSet() {
				f.Set(reflect.ValueOf(outputs))
				break
			}
		}

		return newPtr.Interface().(DirectiveT)
	}

	return dir
}

func (db *directiveBinder[DirectiveT]) mergeMapping(
	node DirectiveOwner,
	directive DirectiveT,
	kind string,
	accumulator map[string]*InputOrOutput,
	bindings any,
) {
	if bindings == nil {
		return
	}
	pm, ok := bindings.(*ClassPropertyMappingGeneric)
	if !ok {
		return
	}

	pm.ForEach(func(binding *InputOrOutput) {
		existing, exists := accumulator[binding.ClassPropertyName]

		// Untracked binding, track it.
		if !exists {
			accumulator[binding.ClassPropertyName] = binding
			return
		}

		// If the binding is already tracked, but is equivalent to the existing binding, we can keep it.
		if existing.BindingPropertyName == binding.BindingPropertyName &&
			existing.ClassPropertyName == binding.ClassPropertyName &&
			existing.IsSignal == binding.IsSignal {
			return
		}

		// Otherwise track the binding as conflicting so it can be reported later.
		conflictsForNode := db.conflictingBindings[node]

		var conflict *ConflictingHostDirectiveBinding[DirectiveT]
		for i := range conflictsForNode {
			if conflictsForNode[i].Directive.GetRefKey() == directive.GetRefKey() &&
				conflictsForNode[i].Kind == kind &&
				conflictsForNode[i].ClassPropertyName == binding.ClassPropertyName {
				conflict = &conflictsForNode[i]
				break
			}
		}

		if conflict == nil {
			newConflict := ConflictingHostDirectiveBinding[DirectiveT]{
				Directive:          directive,
				Kind:               kind,
				ClassPropertyName:  existing.ClassPropertyName,
				ConflictingAliases: map[string]bool{existing.BindingPropertyName: true},
			}
			conflictsForNode = append(conflictsForNode, newConflict)
			db.conflictingBindings[node] = conflictsForNode
			conflict = &db.conflictingBindings[node][len(conflictsForNode)-1]
		}

		conflict.ConflictingAliases[binding.BindingPropertyName] = true
	})
}

func (db *directiveBinder[DirectiveT]) dedupeAndMergeDirectives(node DirectiveOwner, matches []DirectiveT) []DirectiveT {
	if len(matches) == 0 {
		return matches
	}

	allSelector := true
	for _, d := range matches {
		if d.GetMatchSource() != MatchSourceSelector {
			allSelector = false
			break
		}
	}
	if allSelector {
		return matches
	}

	selectorMatches := make(map[string]bool)
	hostDirectives := make(map[string][]DirectiveT)

	for _, dir := range matches {
		key := dir.GetRefKey()
		if dir.GetMatchSource() == MatchSourceSelector {
			selectorMatches[key] = true
		} else {
			hostDirectives[key] = append(hostDirectives[key], dir)
		}
	}

	mergedHostDirectives := make(map[string]DirectiveT)
	var keys []string
	for _, dir := range matches {
		if dir.GetMatchSource() != MatchSourceSelector {
			key := dir.GetRefKey()
			if _, found := mergedHostDirectives[key]; !found && !selectorMatches[key] {
				keys = append(keys, key)
				mergedHostDirectives[key] = dir
			}
		}
	}

	for _, key := range keys {
		directives := hostDirectives[key]
		if len(directives) == 1 {
			mergedHostDirectives[key] = directives[0]
			continue
		}

		inputs := make(map[string]*InputOrOutput)
		outputs := make(map[string]*InputOrOutput)

		for _, dir := range directives {
			db.mergeMapping(node, dir, "input", inputs, dir.GetInputs())
			db.mergeMapping(node, dir, "output", outputs, dir.GetOutputs())
		}

		inputsObj := make(map[string]interface{})
		for k, v := range inputs {
			inputsObj[k] = v
		}
		outputsObj := make(map[string]interface{})
		for k, v := range outputs {
			outputsObj[k] = v
		}

		mergedInputs := ClassPropertyMappingFromMappedObject(inputsObj)
		mergedOutputs := ClassPropertyMappingFromMappedObject(outputsObj)

		mergedHostDirectives[key] = mergeDirectiveMeta[DirectiveT](directives[0], mergedInputs, mergedOutputs)
	}

	var result []DirectiveT
	for _, dir := range matches {
		if dir.GetMatchSource() == MatchSourceSelector {
			result = append(result, dir)
		} else {
			key := dir.GetRefKey()
			if mergedDir, exists := mergedHostDirectives[key]; exists {
				result = append(result, mergedDir)
				delete(mergedHostDirectives, key)
			}
		}
	}

	return result
}

func (s *Scope) Visit(node Node) any               { node.Visit(s); return nil }
func (s *Scope) VisitVariableExpr(v *Variable) any { return nil }
