package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

const MINIMUM_PARTIAL_LINKER_VERSION_INJECTOR = "12.0.0"

func CompileDeclareInjectorFromMetadata(meta render3.R3InjectorMetadata) render3.R3CompiledExpression {
	definitionMap := createInjectorDefinitionMap(meta)

	expression := render3.ImportExpr(*render3.Identifiers.DeclareInjector).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
	typeExpr := render3.CreateInjectorType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func createInjectorDefinitionMap(meta render3.R3InjectorMetadata) *render3.DefinitionMap {
	definitionMap := render3.NewDefinitionMap()

	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_INJECTOR))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))

	definitionMap.Set("type", meta.Type.Value)
	definitionMap.Set("providers", meta.Providers)

	if len(meta.Imports) > 0 {
		definitionMap.Set("imports", render3.LiteralArr(meta.Imports))
	}

	return definitionMap
}
