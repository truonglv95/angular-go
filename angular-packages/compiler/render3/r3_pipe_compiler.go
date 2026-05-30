package render3

// Port of angular/packages/compiler/src/render3/r3_pipe_any

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
)

// R3PipeMetadata holds metadata for pipe pipeline.
type R3PipeMetadata struct {
	// Name of the pipe type.
	Name string

	// An expression representing a reference to the pipe itself.
	Type R3Reference

	// Number of generic type parameters of the type itself.
	TypeArgumentCount int

	// Name of the pipe (from the @Pipe decorator).
	PipeName *string // null if not provided

	// Dependencies of the pipe's constructor.
	Deps []R3DependencyMetadata // nil if no constructor

	// Whether the pipe is marked as pure.
	Pure bool

	// Whether the pipe is standalone.
	IsStandalone bool
}

// CompilePipeFromMetadata compiles a pipe from its metadata into an R3CompiledExpression.
func CompilePipeFromMetadata(metadata R3PipeMetadata) R3CompiledExpression {
	// Build the definition map values
	type mapEntry struct {
		Key    string
		Quoted bool
		Value  output.Expression
	}
	var definitionMapValues []mapEntry

	// e.g. `name: 'myPipe'`
	pipeName := metadata.Name
	if metadata.PipeName != nil {
		pipeName = *metadata.PipeName
	}
	definitionMapValues = append(definitionMapValues, mapEntry{
		Key:    "name",
		Value:  LiteralExpr(pipeName),
		Quoted: false,
	})

	// e.g. `type: MyPipe`
	definitionMapValues = append(definitionMapValues, mapEntry{
		Key:    "type",
		Value:  metadata.Type.Value,
		Quoted: false,
	})

	// e.g. `pure: true`
	definitionMapValues = append(definitionMapValues, mapEntry{
		Key:    "pure",
		Value:  LiteralExpr(metadata.Pure),
		Quoted: false,
	})

	if !metadata.IsStandalone {
		definitionMapValues = append(definitionMapValues, mapEntry{
			Key:    "standalone",
			Value:  LiteralExpr(false),
			Quoted: false,
		})
	}

	entries := make([]output.LiteralMapEntry, len(definitionMapValues))
	for i, e := range definitionMapValues {
		entries[i] = output.NewLiteralMapPropertyAssignment(e.Key, e.Value, e.Quoted)
	}
	literalMap := output.NewLiteralMapExpr(entries, nil, nil, nil)

	expression := ImportExpr(*Identifiers.DefinePipe).
		CallFn([]output.Expression{literalMap}, nil, true, nil)
	type_ := CreatePipeType(metadata)

	return R3CompiledExpression{Expression: expression, Type: type_, Statements: []output.Statement{}}
}

// CreatePipeType creates the type expression for a pipe.
func CreatePipeType(metadata R3PipeMetadata) output.Type {
	var pipeNameVal interface{} = nil
	if metadata.PipeName != nil {
		pipeNameVal = *metadata.PipeName
	}
	return output.NewExpressionType(
		output.NewExternalExpr(
			*Identifiers.PipeDeclaration,
			nil,
			[]output.Type{
				TypeWithParameters(metadata.Type.Type, metadata.TypeArgumentCount),
				output.NewExpressionType(output.NewLiteralExpr(pipeNameVal, nil, nil, nil)),
				output.NewExpressionType(output.NewLiteralExpr(metadata.IsStandalone, nil, nil, nil)),
			},
			nil,
			nil,
		),
	)
}
