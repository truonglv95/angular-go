package injectable

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/partial_evaluator"
	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/reflection"
	"github.com/microsoft/typescript-go/internal/ast"
)

// -- Dummy structs for structural parity --

type R3InjectableMetadata struct {
	Name              string
	Type              *WrappedNodeExpr
	TypeArgumentCount int
	ProvidedIn        *MaybeForwardRefExpression
	UseValue          *MaybeForwardRefExpression
	UseExisting       *MaybeForwardRefExpression
	UseClass          *MaybeForwardRefExpression
	UseFactory        *WrappedNodeExpr
	Deps              []R3DependencyMetadata
}

type R3DependencyMetadata struct {
	Token             *WrappedNodeExpr
	AttributeNameType interface{}
	Host              bool
	Optional          bool
	Self              bool
	SkipSelf          bool
}

type R3ClassMetadata struct{}

type R3CompiledExpression struct {
	Expression interface{}
	Statements []interface{}
	Type       interface{}
}

type WrappedNodeExpr struct {
	Node any
}

type LiteralExpr struct {
	Value interface{}
}

type ForwardRefHandling int

const (
	ForwardRefHandlingNone ForwardRefHandling = iota
	ForwardRefHandlingUnwrapped
)

type MaybeForwardRefExpression struct {
	Expression interface{}
	Handling   ForwardRefHandling
}

type FactoryTarget int

const (
	FactoryTargetInjectable FactoryTarget = iota
)

type CompileResult struct {
	Name              string
	Initializer       interface{}
	Statements        []interface{}
	Type              interface{}
	DeferrableImports interface{}
}

type AnalysisOutput struct {
	Analysis *InjectableHandlerData
}

type DetectResult struct {
	Trigger   any
	Decorator *reflection.Decorator
	Metadata  *reflection.Decorator
}

type ResolveResult struct {
	Diagnostics []interface{}
}

type InjectableClassRegistry interface {
	RegisterInjectable(node any, data interface{})
}

type PerfRecorder interface {
	EventCount(event string)
}

type CompilationMode string

const (
	CompilationModeLocal CompilationMode = "LOCAL"
	CompilationModeFull  CompilationMode = "FULL"
)

type CompileFactoryFn func(meta interface{}) *CompileResult
type CompileClassMetadataFn func(meta *R3ClassMetadata) *CompileResult

type ErrorCode int

const (
	ErrorCodeDecoratorNotCalled      ErrorCode = 1
	ErrorCodeDecoratorArgNotLiteral  ErrorCode = 2
	ErrorCodeValueNotLiteral         ErrorCode = 3
	ErrorCodeDecoratorArityWrong     ErrorCode = 4
	ErrorCodeInjectableDuplicateProv ErrorCode = 5
)

type FatalDiagnosticError struct {
	Code    ErrorCode
	Node    any
	Message string
}

func (e *FatalDiagnosticError) Error() string { return e.Message }

// -- End dummy structs --

type InjectableHandlerData struct {
	Meta          *R3InjectableMetadata
	ClassMetadata *R3ClassMetadata
	CtorDeps      interface{} // []R3DependencyMetadata, "invalid", or nil
	NeedsFactory  bool
}

type InjectableDecoratorHandler struct {
	reflector            reflection.ReflectionHost
	evaluator            *partial_evaluator.PartialEvaluator
	isCore               bool
	strictCtorDeps       bool
	injectableRegistry   InjectableClassRegistry
	perf                 PerfRecorder
	includeClassMetadata bool
	compilationMode      CompilationMode
	errorOnDuplicateProv bool

	Precedence int
	Name       string
}

func NewInjectableDecoratorHandler(
	reflector reflection.ReflectionHost,
	evaluator *partial_evaluator.PartialEvaluator,
	isCore bool,
	strictCtorDeps bool,
	injectableRegistry InjectableClassRegistry,
	perf PerfRecorder,
	includeClassMetadata bool,
	compilationMode CompilationMode,
	errorOnDuplicateProv bool,
) *InjectableDecoratorHandler {
	return &InjectableDecoratorHandler{
		reflector:            reflector,
		evaluator:            evaluator,
		isCore:               isCore,
		strictCtorDeps:       strictCtorDeps,
		injectableRegistry:   injectableRegistry,
		perf:                 perf,
		includeClassMetadata: includeClassMetadata,
		compilationMode:      compilationMode,
		errorOnDuplicateProv: errorOnDuplicateProv,
		Precedence:           1, // HandlerPrecedence.SHARED
		Name:                 "InjectableDecoratorHandler",
	}
}

func (h *InjectableDecoratorHandler) Detect(node any, decorators []reflection.Decorator) *DetectResult {
	if decorators == nil {
		return nil
	}
	decorator := findAngularDecorator(decorators, "Injectable", h.isCore)
	if decorator != nil {
		return &DetectResult{
			Trigger:   decorator.Node,
			Decorator: decorator,
			Metadata:  decorator,
		}
	}
	return nil
}

func (h *InjectableDecoratorHandler) Analyze(node *ast.Node, decorator *reflection.Decorator) (*AnalysisOutput, error) {
	h.perf.EventCount("AnalyzeInjectable")

	meta, err := extractInjectableMetadata(node, decorator, h.reflector)
	if err != nil {
		return nil, err
	}

	decorators := h.reflector.GetDecoratorsOfDeclaration(node)

	ctorDeps, err := extractInjectableCtorDeps(node, meta, decorator, h.reflector, h.isCore, h.strictCtorDeps)
	if err != nil {
		return nil, err
	}

	var classMetadata *R3ClassMetadata
	if h.includeClassMetadata {
		classMetadata = extractClassMetadata(node, h.reflector, h.isCore)
	}

	needsFactory := true
	if decorators != nil {
		allNotCoreOrInjectable := true
		for _, current := range decorators {
			if isAngularCore(&current) && current.Name != "Injectable" {
				allNotCoreOrInjectable = false
				break
			}
		}
		needsFactory = allNotCoreOrInjectable
	}

	return &AnalysisOutput{
		Analysis: &InjectableHandlerData{
			Meta:          meta,
			CtorDeps:      ctorDeps,
			ClassMetadata: classMetadata,
			NeedsFactory:  needsFactory,
		},
	}, nil
}

func (h *InjectableDecoratorHandler) Symbol() interface{} {
	return nil
}

func (h *InjectableDecoratorHandler) Register(node *ast.Node, analysis *InjectableHandlerData) {
	if h.compilationMode == CompilationModeLocal {
		return
	}
	h.injectableRegistry.RegisterInjectable(node, map[string]interface{}{
		"ctorDeps": analysis.CtorDeps,
	})
}

func (h *InjectableDecoratorHandler) Resolve(node *ast.Node, analysis *InjectableHandlerData) *ResolveResult {
	if h.compilationMode == CompilationModeLocal {
		return &ResolveResult{}
	}

	if requiresValidCtor(analysis.Meta) {
		diagnostic := checkInheritanceOfInjectable(node, h.injectableRegistry, h.reflector, h.evaluator, h.strictCtorDeps, "Injectable")
		if diagnostic != nil {
			return &ResolveResult{
				Diagnostics: []interface{}{diagnostic},
			}
		}
	}

	return &ResolveResult{}
}

func (h *InjectableDecoratorHandler) CompileFull(node *ast.Node, analysis *InjectableHandlerData) ([]CompileResult, error) {
	return h.Compile(
		compileNgFactoryDefField,
		func(meta *R3InjectableMetadata) *R3CompiledExpression { return compileInjectable(meta, false) },
		compileClassMetadata,
		node,
		analysis,
	)
}

func (h *InjectableDecoratorHandler) CompilePartial(node *ast.Node, analysis *InjectableHandlerData) ([]CompileResult, error) {
	return h.Compile(
		compileDeclareFactory,
		compileDeclareInjectableFromMetadata,
		compileDeclareClassMetadata,
		node,
		analysis,
	)
}

func (h *InjectableDecoratorHandler) CompileLocal(node *ast.Node, analysis *InjectableHandlerData) ([]CompileResult, error) {
	return h.Compile(
		compileNgFactoryDefField,
		func(meta *R3InjectableMetadata) *R3CompiledExpression { return compileInjectable(meta, false) },
		compileClassMetadata,
		node,
		analysis,
	)
}

func (h *InjectableDecoratorHandler) Compile(
	compileFactoryFn CompileFactoryFn,
	compileInjectableFn func(meta *R3InjectableMetadata) *R3CompiledExpression,
	compileClassMetadataFn CompileClassMetadataFn,
	node *ast.Node,
	analysis *InjectableHandlerData,
) ([]CompileResult, error) {
	var results []CompileResult

	if analysis.NeedsFactory {
		meta := analysis.Meta
		factoryRes := compileFactoryFn(toFactoryMetadata(meta, analysis.CtorDeps, FactoryTargetInjectable))
		if factoryRes != nil {
			if analysis.ClassMetadata != nil {
				if len(compileClassMetadataFn(analysis.ClassMetadata).Statements) > 0 {
					stmt := compileClassMetadataFn(analysis.ClassMetadata).Statements[0]
					factoryRes.Statements = append(factoryRes.Statements, stmt)
				}
			}
			results = append(results, *factoryRes)
		}
	}

	var prov *reflection.ClassMember
	members := h.reflector.GetMembersOfClass(node)
	for i := range members {
		if members[i].Name == "ɵprov" {
			prov = &members[i]
			break
		}
	}

	if prov != nil && h.errorOnDuplicateProv {
		errNode := prov.NameNode
		if errNode == nil {
			errNode = prov.Node
		}
		if errNode == nil {
			errNode = node
		}
		return nil, &FatalDiagnosticError{
			Code:    ErrorCodeInjectableDuplicateProv,
			Node:    errNode,
			Message: "Injectables cannot contain a static ɵprov property, because the compiler is going to generate one.",
		}
	}

	if prov == nil {
		res := compileInjectableFn(analysis.Meta)
		results = append(results, CompileResult{
			Name:              "ɵprov",
			Initializer:       res.Expression,
			Statements:        res.Statements,
			Type:              res.Type,
			DeferrableImports: nil,
		})
	}

	return results, nil
}

func extractInjectableMetadata(clazz *ast.Node, decorator *reflection.Decorator, reflector reflection.ReflectionHost) (*R3InjectableMetadata, error) {
	name := ""
	if ast.IsClassDeclaration(clazz) {
		classDecl := clazz.AsClassDeclaration()
		if classDecl.Name() != nil {
			name = classDecl.Name().AsIdentifier().Text
		}
	}
	typeRef := wrapTypeReference(clazz)
	typeArgumentCount := reflector.GetGenericArityOfClass(clazz)
	if typeArgumentCount < 0 {
		typeArgumentCount = 0
	}

	if decorator.Args == nil {
		return nil, &FatalDiagnosticError{
			Code:    ErrorCodeDecoratorNotCalled,
			Node:    decorator.Node,
			Message: "@Injectable must be called",
		}
	}

	if len(decorator.Args) == 0 {
		return &R3InjectableMetadata{
			Name:              name,
			Type:              typeRef,
			TypeArgumentCount: typeArgumentCount,
			ProvidedIn:        createMayBeForwardRefExpression(&LiteralExpr{Value: nil}, ForwardRefHandlingNone),
		}, nil
	} else if len(decorator.Args) == 1 {
		metaNode := decorator.Args[0]
		if !ast.IsObjectLiteralExpression(metaNode) {
			return nil, &FatalDiagnosticError{
				Code:    ErrorCodeDecoratorArgNotLiteral,
				Node:    metaNode,
				Message: "@Injectable argument must be an object literal",
			}
		}

		meta := reflectObjectLiteral(metaNode.AsObjectLiteralExpression())

		var providedIn *MaybeForwardRefExpression
		if expr, has := meta["providedIn"]; has {
			providedIn = getProviderExpression(expr, reflector)
		} else {
			providedIn = createMayBeForwardRefExpression(&LiteralExpr{Value: nil}, ForwardRefHandlingNone)
		}

		var deps []R3DependencyMetadata
		hasUseClass := false
		if _, ok := meta["useClass"]; ok {
			hasUseClass = true
		}
		hasUseFactory := false
		if _, ok := meta["useFactory"]; ok {
			hasUseFactory = true
		}

		if (hasUseClass || hasUseFactory) && meta["deps"] != nil {
			depsExpr := meta["deps"]
			if !ast.IsArrayLiteralExpression(depsExpr) {
				return nil, &FatalDiagnosticError{
					Code:    ErrorCodeValueNotLiteral,
					Node:    depsExpr,
					Message: "@Injectable deps metadata must be an inline array",
				}
			}
			for _, dep := range []any{} {
				deps = append(deps, getDep(dep, reflector))
			}
		}

		result := &R3InjectableMetadata{
			Name:              name,
			Type:              typeRef,
			TypeArgumentCount: typeArgumentCount,
			ProvidedIn:        providedIn,
		}

		if expr, ok := meta["useValue"]; ok {
			result.UseValue = getProviderExpression(expr, reflector)
		} else if expr, ok := meta["useExisting"]; ok {
			result.UseExisting = getProviderExpression(expr, reflector)
		} else if expr, ok := meta["useClass"]; ok {
			result.UseClass = getProviderExpression(expr, reflector)
			result.Deps = deps
		} else if expr, ok := meta["useFactory"]; ok {
			result.UseFactory = &WrappedNodeExpr{Node: expr}
			result.Deps = deps
		}

		return result, nil
	} else {
		var errNode *ast.Node
		if len(decorator.Args) > 2 {
			errNode = decorator.Args[2]
		}
		return nil, &FatalDiagnosticError{
			Code:    ErrorCodeDecoratorArityWrong,
			Node:    errNode,
			Message: "Too many arguments to @Injectable",
		}
	}
}

func getProviderExpression(expression *ast.Node, reflector reflection.ReflectionHost) *MaybeForwardRefExpression {
	forwardRefValue := tryUnwrapForwardRef(expression, reflector)
	exprNode := forwardRefValue
	if exprNode == nil {
		exprNode = expression
	}
	handling := ForwardRefHandlingNone
	if forwardRefValue != nil {
		handling = ForwardRefHandlingUnwrapped
	}
	return createMayBeForwardRefExpression(&WrappedNodeExpr{Node: exprNode}, handling)
}

func extractInjectableCtorDeps(
	clazz *ast.Node,
	meta *R3InjectableMetadata,
	decorator *reflection.Decorator,
	reflector reflection.ReflectionHost,
	isCore bool,
	strictCtorDeps bool,
) (interface{}, error) {
	if decorator.Args == nil {
		return nil, &FatalDiagnosticError{
			Code:    ErrorCodeDecoratorNotCalled,
			Node:    decorator.Node,
			Message: "@Injectable must be called",
		}
	}

	var ctorDeps interface{}

	if len(decorator.Args) == 0 {
		if strictCtorDeps && !isAbstractClassDeclaration(clazz) {
			ctorDeps = getValidConstructorDependencies(clazz, reflector, isCore)
		} else {
			ctorDeps = unwrapConstructorDependencies(
				getConstructorDependencies(clazz, reflector, isCore),
			)
		}
		return ctorDeps, nil
	} else if len(decorator.Args) == 1 {
		rawCtorDeps := getConstructorDependencies(clazz, reflector, isCore)

		if strictCtorDeps && !isAbstractClassDeclaration(clazz) && requiresValidCtor(meta) {
			ctorDeps = validateConstructorDependencies(clazz, rawCtorDeps)
		} else {
			ctorDeps = unwrapConstructorDependencies(rawCtorDeps)
		}
	}

	return ctorDeps, nil
}

func requiresValidCtor(meta *R3InjectableMetadata) bool {
	return meta.UseValue == nil &&
		meta.UseExisting == nil &&
		meta.UseClass == nil &&
		meta.UseFactory == nil
}

func getDep(dep any, reflector reflection.ReflectionHost) R3DependencyMetadata {
	return R3DependencyMetadata{}
}

// -- Placeholder functions --

func findAngularDecorator(decorators []reflection.Decorator, name string, isCore bool) *reflection.Decorator {
	for i := range decorators {
		dec := &decorators[i]
		if dec.Name != name {
			continue
		}
		// When isCore is true, the class is inside @angular/core itself, so no import check.
		if isCore {
			return dec
		}
		// Otherwise, the decorator must be imported from @angular/core.
		if dec.Import != nil && isAngularCoreImport(dec.Import.From) {
			return dec
		}
	}
	return nil
}

// isAngularCoreImport returns true if the module path is @angular/core or an alias for it.
func isAngularCoreImport(from string) bool {
	return from == "@angular/core"
}

func isAngularCore(decorator *reflection.Decorator) bool {
	if decorator == nil {
		return false
	}
	return decorator.Import != nil && isAngularCoreImport(decorator.Import.From)
}

func extractClassMetadata(node *ast.Node, reflector reflection.ReflectionHost, isCore bool) *R3ClassMetadata {
	return nil
}

func checkInheritanceOfInjectable(node *ast.Node, registry InjectableClassRegistry, reflector reflection.ReflectionHost, evaluator *partial_evaluator.PartialEvaluator, strictCtorDeps bool, decoratorName string) interface{} {
	return nil
}

func compileNgFactoryDefField(meta interface{}) *CompileResult {
	return nil
}

func compileDeclareFactory(meta interface{}) *CompileResult {
	return nil
}

func compileInjectable(meta *R3InjectableMetadata, b bool) *R3CompiledExpression {
	return nil
}

func compileDeclareInjectableFromMetadata(meta *R3InjectableMetadata) *R3CompiledExpression {
	return nil
}

func compileClassMetadata(meta *R3ClassMetadata) *CompileResult {
	return nil
}

func compileDeclareClassMetadata(meta *R3ClassMetadata) *CompileResult {
	return nil
}

func toFactoryMetadata(meta *R3InjectableMetadata, ctorDeps interface{}, target FactoryTarget) interface{} {
	return nil
}

func wrapTypeReference(clazz *ast.Node) *WrappedNodeExpr {
	return &WrappedNodeExpr{Node: clazz}
}

func createMayBeForwardRefExpression(expr interface{}, handling ForwardRefHandling) *MaybeForwardRefExpression {
	return &MaybeForwardRefExpression{Expression: expr, Handling: handling}
}

func reflectObjectLiteral(node *ast.ObjectLiteralExpression) map[string]*ast.Node {
	return map[string]*ast.Node{}
}

func tryUnwrapForwardRef(expr *ast.Node, reflector reflection.ReflectionHost) *ast.Node {
	return nil
}

func isAbstractClassDeclaration(clazz *ast.Node) bool {
	return false
}

func getValidConstructorDependencies(clazz *ast.Node, reflector reflection.ReflectionHost, isCore bool) interface{} {
	return nil
}

func getConstructorDependencies(clazz *ast.Node, reflector reflection.ReflectionHost, isCore bool) interface{} {
	return nil
}

func unwrapConstructorDependencies(deps interface{}) interface{} {
	return nil
}

func validateConstructorDependencies(clazz *ast.Node, deps interface{}) interface{} {
	return nil
}
