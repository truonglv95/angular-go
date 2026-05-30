package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

const MINIMUM_PARTIAL_LINKER_VERSION_PIPE = "14.0.0"

func CompileDeclarePipeFromMetadata(meta render3.R3PipeMetadata) render3.R3CompiledExpression {
	definitionMap := createPipeDefinitionMap(meta)

	expression := render3.ImportExpr(*render3.Identifiers.DeclarePipe).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
	typeExpr := output.INFERRED_TYPE

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func createPipeDefinitionMap(meta render3.R3PipeMetadata) *render3.DefinitionMap {
	definitionMap := render3.NewDefinitionMap()

	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_PIPE))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))

	definitionMap.Set("type", meta.Type.Value)

	if meta.IsStandalone {
		definitionMap.Set("isStandalone", render3.LiteralExpr(meta.IsStandalone))
	}

	name := meta.PipeName
	if meta.Name == "" {

	}
	definitionMap.Set("name", render3.LiteralExpr(name))

	if !meta.Pure {
		definitionMap.Set("pure", render3.LiteralExpr(false))
	}

	return definitionMap
}
