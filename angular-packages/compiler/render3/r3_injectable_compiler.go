package render3

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

type R3InjectableMetadata struct {
	Name             string
	Type             R3Reference
	TypeArgumentCount int
	ProvidedIn       output.Expression
	UseClass         output.Expression
	UseFactory       output.Expression
	UseExisting      output.Expression
	UseValue         output.Expression
}

func CompileInjectableFromMetadata(metadata R3InjectableMetadata) R3CompiledExpression {
	type mapEntry struct {
		Key    string
		Quoted bool
		Value  output.Expression
	}
	var definitionMapValues []mapEntry

	definitionMapValues = append(definitionMapValues, mapEntry{
		Key:    "token",
		Value:  metadata.Type.Value,
		Quoted: false,
	})
	
	var facRef output.Expression
	if metadata.UseClass != nil {
		facRef = output.NewReadPropExpr(metadata.UseClass, "ɵfac", nil, nil, nil, false)
	} else if metadata.UseFactory != nil {
		facRef = metadata.UseFactory
	} else if metadata.UseValue != nil {
		facRef = output.NewFunctionExpr([]*output.FnParam{}, []output.Statement{output.NewReturnStatement(metadata.UseValue, nil, nil)}, nil, nil, nil, nil)
	} else if metadata.UseExisting != nil {
		facRef = output.NewFunctionExpr([]*output.FnParam{}, []output.Statement{output.NewReturnStatement(ImportExpr(*Identifiers.Inject).CallFn([]output.Expression{metadata.UseExisting}, nil, false, nil), nil, nil)}, nil, nil, nil, nil)
	} else {
		facRef = output.NewReadPropExpr(metadata.Type.Value, "ɵfac", nil, nil, nil, false)
	}
	
	definitionMapValues = append(definitionMapValues, mapEntry{
		Key:    "factory",
		Value:  facRef,
		Quoted: false,
	})

	if metadata.ProvidedIn != nil {
		definitionMapValues = append(definitionMapValues, mapEntry{
			Key:    "providedIn",
			Value:  metadata.ProvidedIn,
			Quoted: false,
		})
	}

	entries := make([]output.LiteralMapEntry, len(definitionMapValues))
	for i, e := range definitionMapValues {
		entries[i] = output.NewLiteralMapPropertyAssignment(e.Key, e.Value, e.Quoted)
	}
	literalMap := output.NewLiteralMapExpr(entries, nil, nil, nil)

	expression := ImportExpr(*Identifiers.ƟɵdefineInjectable).
		CallFn([]output.Expression{literalMap}, nil, true, nil)
	
	type_ := output.NewExpressionType(
		output.NewExternalExpr(*Identifiers.InjectableDeclaration, nil, []output.Type{
			TypeWithParameters(metadata.Type.Type, metadata.TypeArgumentCount),
		}, nil, nil),
	)

	return R3CompiledExpression{Expression: expression, Type: type_, Statements: []output.Statement{}}
}
