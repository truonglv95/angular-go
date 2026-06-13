package render3

// Port of angular/packages/compiler/src/render3/r3_factory.ts

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// FactoryTarget represents the target type for factory generation.
// Port of FactoryTarget from compiler_facade_interface.ts.
type FactoryTarget int

const (
	FactoryTargetDirective  FactoryTarget = 0
	FactoryTargetComponent  FactoryTarget = 1
	FactoryTargetPipe       FactoryTarget = 2
	FactoryTargetInjectable FactoryTarget = 3
	FactoryTargetNgModule   FactoryTarget = 4
)

// R3FactoryDelegateType represents whether the factory delegates to a class or function.
type R3FactoryDelegateType int

const (
	R3FactoryDelegateTypeClass    R3FactoryDelegateType = 0
	R3FactoryDelegateTypeFunction R3FactoryDelegateType = 1
)

// R3DependencyMetadata describes a dependency to be injected.
type R3DependencyMetadata struct {
	// An expression representing the token or value to be injected.
	// Nil if the dependency could not be resolved.
	Token output.Expression // nil if invalid

	// If an @Attribute decorator is present, this is the literal type of the attribute name.
	AttributeNameType output.Expression // nil if not an @Attribute

	// Whether the dependency has an @Host qualifier.
	Host bool

	// Whether the dependency has an @Optional qualifier.
	Optional bool

	// Whether the dependency has an @Self qualifier.
	Self bool

	// Whether the dependency has an @SkipSelf qualifier.
	SkipSelf bool
}

// R3ConstructorFactoryMetadata is the base metadata for factory generation.
type R3ConstructorFactoryMetadata struct {
	// String name of the type being generated.
	Name string

	// An expression representing the interface type being constructed.
	Type R3Reference

	// Number of arguments for the type.
	TypeArgumentCount int

	// Dependencies for injection. Use nil for "no constructor" (use base class factory),
	// or InvalidDeps for invalid dependencies.
	Deps interface{} // []R3DependencyMetadata | "invalid" | nil

	// Type of the target being created.
	Target FactoryTarget
}

// R3DelegatedFnOrClassMetadata extends R3ConstructorFactoryMetadata with delegation info.
type R3DelegatedFnOrClassMetadata struct {
	R3ConstructorFactoryMetadata
	Delegate     output.Expression
	DelegateType R3FactoryDelegateType
	DelegateDeps []R3DependencyMetadata
}

// R3ExpressionFactoryMetadata extends R3ConstructorFactoryMetadata with an expression factory.
type R3ExpressionFactoryMetadata struct {
	R3ConstructorFactoryMetadata
	Expression output.Expression
}

// R3FactoryMetadata is the union type for factory metadata.
type R3FactoryMetadata interface {
	getBase() R3ConstructorFactoryMetadata
}

func (m R3ConstructorFactoryMetadata) getBase() R3ConstructorFactoryMetadata { return m }
func (m *R3DelegatedFnOrClassMetadata) getBase() R3ConstructorFactoryMetadata {
	return m.R3ConstructorFactoryMetadata
}
func (m *R3ExpressionFactoryMetadata) getBase() R3ConstructorFactoryMetadata {
	return m.R3ConstructorFactoryMetadata
}

// IsDelegatedFactoryMetadata returns true if meta is R3DelegatedFnOrClassMetadata.
func IsDelegatedFactoryMetadata(meta R3FactoryMetadata) (*R3DelegatedFnOrClassMetadata, bool) {
	d, ok := meta.(*R3DelegatedFnOrClassMetadata)
	return d, ok
}

// IsExpressionFactoryMetadata returns true if meta is R3ExpressionFactoryMetadata.
func IsExpressionFactoryMetadata(meta R3FactoryMetadata) (*R3ExpressionFactoryMetadata, bool) {
	e, ok := meta.(*R3ExpressionFactoryMetadata)
	return e, ok
}

// CompileFactoryFunction compiles a factory function expression for the given R3FactoryMetadata.
func CompileFactoryFunction(meta R3FactoryMetadata) R3CompiledExpression {
	base := meta.getBase()
	tName := "__ngFactoryType__"
	t := VariableExpr(tName)
	var baseFactoryVar *output.ReadVarExpr

	// The type to instantiate via constructor invocation.
	var typeForCtor output.Expression
	if _, isDelegate := IsDelegatedFactoryMetadata(meta); !isDelegate {
		// typeForCtor = t || meta.type.value
		typeForCtor = output.NewBinaryOperatorExpr(output.BinaryOperatorOr, t, base.Type.Value, nil, nil, nil)
	} else {
		typeForCtor = t
	}

	var ctorExpr output.Expression

	// Build factory comments if deps has elements
	var factoryComments []output.LeadingComment
	if deps, ok := base.Deps.([]R3DependencyMetadata); ok && len(deps) > 0 {
		factoryComments = []output.LeadingComment{TsIgnoreComment()}
	}

	if base.Deps != nil {
		// There is a constructor (either explicitly or implicitly defined).
		if deps, ok := base.Deps.([]R3DependencyMetadata); ok {
			// deps is valid (not "invalid")
			ctorExpr = output.NewInstantiateExpr(typeForCtor, injectDependencies(deps, base.Target), nil, nil, nil)
		}
		// If deps == "invalid", ctorExpr stays nil
	} else {
		// There is no constructor, use the base class' factory.
		baseFactoryVarName := "ɵ" + base.Name + "_BaseFactory"
		baseFactoryVar = VariableExpr(baseFactoryVarName)
		ctorExpr = baseFactoryVar.CallFn([]output.Expression{typeForCtor}, nil, false, nil)
	}

	var body []output.Statement
	var retExpr output.Expression

	makeConditionalFactory := func(nonCtorExpr output.Expression) *output.ReadVarExpr {
		r := VariableExpr("__ngConditionalFactory__")
		body = append(body, output.NewDeclareVarStmt("__ngConditionalFactory__", output.NULL_EXPR, output.DYNAMIC_TYPE, output.StmtModifierNone, nil, nil))
		var ctorStmt output.Statement
		if ctorExpr != nil {
			ctorStmt = output.NewBinaryOperatorExpr(output.BinaryOperatorAssign, r, ctorExpr, nil, nil, nil).ToStmt(factoryComments)
		} else {
			ctorStmt = ImportExpr(*Identifiers.InvalidFactory).CallFn(nil, nil, false, nil).ToStmt(nil)
		}
		// Always add a `ts-ignore` on the alternate factory.
		altStmt := output.NewBinaryOperatorExpr(output.BinaryOperatorAssign, r, nonCtorExpr, nil, nil, nil).
			ToStmt([]output.LeadingComment{TsIgnoreComment()})
		body = append(body, output.NewIfStmt(t, []output.Statement{ctorStmt}, []output.Statement{altStmt}, nil, nil))
		return r
	}

	if delegated, ok := IsDelegatedFactoryMetadata(meta); ok {
		// This type is created with a delegated factory.
		delegateArgs := injectDependencies(delegated.DelegateDeps, delegated.Target)
		var factoryExpr output.Expression
		if delegated.DelegateType == R3FactoryDelegateTypeClass {
			factoryExpr = output.NewInstantiateExpr(delegated.Delegate, delegateArgs, nil, nil, nil)
		} else {
			factoryExpr = output.NewInvokeFunctionExpr(delegated.Delegate, delegateArgs, nil, nil, false, nil, false)
		}
		retExpr = makeConditionalFactory(factoryExpr)
	} else if expr, ok := IsExpressionFactoryMetadata(meta); ok {
		retExpr = makeConditionalFactory(expr.Expression)
	} else {
		retExpr = ctorExpr
	}

	if retExpr == nil {
		// The expression cannot be formed so render an invalidFactory() call.
		body = append(body, ImportExpr(*Identifiers.InvalidFactory).CallFn(nil, nil, false, nil).ToStmt(nil))
	} else if baseFactoryVar != nil {
		// This factory uses a base factory, so call getInheritedFactory() to compute it.
		getInheritedFactoryCall := ImportExpr(*Identifiers.GetInheritedFactory).
			CallFn([]output.Expression{base.Type.Value}, nil, false, nil)
		// Memoize: baseFactory || (baseFactory = ɵɵgetInheritedFactory(...))
		baseFactory := output.NewBinaryOperatorExpr(
			output.BinaryOperatorOr,
			baseFactoryVar,
			output.NewBinaryOperatorExpr(output.BinaryOperatorAssign, baseFactoryVar, getInheritedFactoryCall, nil, nil, nil),
			nil,
			nil,
			nil,
		)
		body = append(body, output.NewReturnStatement(baseFactory.CallFn([]output.Expression{typeForCtor}, nil, false, nil), nil, nil))
	} else {
		// Straightforward factory, just return it.
		body = append(body, output.NewReturnStatement(retExpr, nil, factoryComments))
	}

	factoryFnName := base.Name + "_Factory"
	var factoryFn output.Expression = output.NewFunctionExpr(
		[]*output.FnParam{output.NewFnParam(tName, output.DYNAMIC_TYPE)},
		body,
		output.INFERRED_TYPE,
		nil,
		&factoryFnName,
		nil,
	)

	if baseFactoryVar != nil {
		// Wrap in an IIFE that declares baseFactoryVar.
		iife := output.NewArrowFunctionExpr(
			nil,
			[]output.Statement{
				output.NewDeclareVarStmt(baseFactoryVar.Name, nil, output.DYNAMIC_TYPE, output.StmtModifierNone, nil, nil),
				output.NewReturnStatement(factoryFn, nil, nil),
			},
			nil,
			nil,
			nil,
		)
		factoryFn = iife.CallFn(nil, nil, true, nil)
	}

	return R3CompiledExpression{
		Expression: factoryFn,
		Statements: []output.Statement{},
		Type:       CreateFactoryType(meta),
	}
}

// CreateFactoryType creates the type expression for a factory.
func CreateFactoryType(meta R3FactoryMetadata) output.Type {
	base := meta.getBase()
	var ctorDepsType output.Type
	if deps, ok := base.Deps.([]R3DependencyMetadata); ok {
		ctorDepsType = createCtorDepsType(deps)
	} else {
		ctorDepsType = output.NONE_TYPE
	}
	return output.NewExpressionType(
		output.NewExternalExpr(
			*Identifiers.FactoryDeclaration,
			nil,
			[]output.Type{
				TypeWithParameters(base.Type.Type, base.TypeArgumentCount),
				ctorDepsType,
			},
			nil,
			nil,
		),
	)
}

func injectDependencies(deps []R3DependencyMetadata, target FactoryTarget) []output.Expression {
	result := make([]output.Expression, len(deps))
	for i, dep := range deps {
		result[i] = compileInjectDependency(dep, target, i)
	}
	return result
}

func compileInjectDependency(dep R3DependencyMetadata, target FactoryTarget, index int) output.Expression {
	if dep.AttributeNameType != nil {
		// @Attribute() dependencies are resolved from the host element's static
		// attributes, not through the DI token inferred from the parameter type.
		return ImportExpr(*Identifiers.InjectAttribute).
			CallFn([]output.Expression{dep.AttributeNameType}, nil, false, nil)
	} else if dep.Token == nil {
		return ImportExpr(*Identifiers.InvalidFactoryDep).
			CallFn([]output.Expression{LiteralExpr(index)}, nil, false, nil)
	} else {
		// Build up the injection flags according to the metadata.
		flags := int(core.InjectFlagsDefault)
		if dep.Self {
			flags |= int(core.InjectFlagsSelf)
		}
		if dep.SkipSelf {
			flags |= int(core.InjectFlagsSkipSelf)
		}
		if dep.Host {
			flags |= int(core.InjectFlagsHost)
		}
		if dep.Optional {
			flags |= int(core.InjectFlagsOptional)
		}
		if target == FactoryTargetPipe {
			flags |= int(core.InjectFlagsForPipe)
		}

		// If flags are non-default or optional, pass flags param.
		var flagsParam output.Expression
		if flags != int(core.InjectFlagsDefault) || dep.Optional {
			flagsParam = LiteralExpr(flags)
		}

		injectArgs := []output.Expression{dep.Token}
		if flagsParam != nil {
			injectArgs = append(injectArgs, flagsParam)
		}
		injectFn := getInjectFn(target)
		return ImportExpr(*injectFn).CallFn(injectArgs, nil, false, nil)
	}
}

func createCtorDepsType(deps []R3DependencyMetadata) output.Type {
	hasTypes := false
	attributeTypes := make([]output.Expression, len(deps))
	for i, dep := range deps {
		t := createCtorDepType(dep)
		if t != nil {
			hasTypes = true
			attributeTypes[i] = t
		} else {
			attributeTypes[i] = LiteralExpr(nil)
		}
	}

	if hasTypes {
		return output.NewExpressionType(LiteralArr(attributeTypes))
	}
	return output.NONE_TYPE
}

func createCtorDepType(dep R3DependencyMetadata) *output.LiteralMapExpr {
	var entries []output.LiteralMapEntry

	if dep.AttributeNameType != nil {
		entries = append(entries, output.NewLiteralMapPropertyAssignment("attribute", dep.AttributeNameType, false))
	}
	if dep.Optional {
		entries = append(entries, output.NewLiteralMapPropertyAssignment("optional", LiteralExpr(true), false))
	}
	if dep.Host {
		entries = append(entries, output.NewLiteralMapPropertyAssignment("host", LiteralExpr(true), false))
	}
	if dep.Self {
		entries = append(entries, output.NewLiteralMapPropertyAssignment("self", LiteralExpr(true), false))
	}
	if dep.SkipSelf {
		entries = append(entries, output.NewLiteralMapPropertyAssignment("skipSelf", LiteralExpr(true), false))
	}

	if len(entries) > 0 {
		return output.NewLiteralMapExpr(entries, nil, nil, nil)
	}
	return nil
}

func getInjectFn(target FactoryTarget) *output.ExternalReference {
	switch target {
	case FactoryTargetComponent, FactoryTargetDirective, FactoryTargetPipe:
		return Identifiers.DirectiveInject
	default:
		return Identifiers.Inject
	}
}
