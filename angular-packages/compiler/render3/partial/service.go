package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

type R3ServiceMetadata struct {
	Type              *render3.R3Reference
	TypeArgumentCount int
	AutoProvided      *bool
	Factory           output.Expression
}

const MINIMUM_PARTIAL_LINKER_VERSION_SERVICE = "22.0.0"

func CompileDeclareServiceFromMetadata(meta R3ServiceMetadata) render3.R3CompiledExpression {
	definitionMap := createServiceDefinitionMap(meta)

	// Stub CreateInjectableType since we removed compiler import
	typeExpr := output.INFERRED_TYPE // compiler.CreateInjectableType(meta.Type.Type, meta.TypeArgumentCount)

	return render3.R3CompiledExpression{
		Expression: render3.ImportExpr(*render3.Identifiers.DeclareService).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil),
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func createServiceDefinitionMap(meta R3ServiceMetadata) *render3.DefinitionMap {
	definitionMap := render3.NewDefinitionMap()

	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_SERVICE))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))
	definitionMap.Set("type", meta.Type.Value)

	if meta.AutoProvided != nil && !*meta.AutoProvided {
		definitionMap.Set("autoProvided", render3.LiteralExpr(false))
	}

	if meta.Factory != nil {
		definitionMap.Set("factory", meta.Factory)
	}

	return definitionMap
}
