package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

const MINIMUM_PARTIAL_LINKER_VERSION_NG_MODULE = "14.0.0"

func CompileDeclareNgModuleFromMetadata(meta render3.R3NgModuleMetadata) render3.R3CompiledExpression {
	definitionMap := createNgModuleDefinitionMap(meta)

	expression := render3.ImportExpr(*render3.Identifiers.DeclareNgModule).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
	typeExpr := render3.CreateNgModuleType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func createNgModuleDefinitionMap(meta render3.R3NgModuleMetadata) *render3.DefinitionMap {
	globalMeta, ok := meta.(*render3.R3NgModuleMetadataGlobal)
	if !ok {
		panic("Partial compilation only supports Global metadata")
	}
	definitionMap := render3.NewDefinitionMap()

	if globalMeta.Kind == render3.R3NgModuleMetadataKindLocal {
		panic("Invalid path! Local compilation mode should not get into the partial compilation path")
	}

	if globalMeta.Kind == render3.R3NgModuleMetadataKindIsolated {
		panic("Invalid path! Isolated compilation mode should not get into the partial compilation path")
	}

	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_NG_MODULE))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))
	definitionMap.Set("type", globalMeta.Type.Value)

	if len(globalMeta.Bootstrap) > 0 {
		definitionMap.Set("bootstrap", render3.RefsToArray(globalMeta.Bootstrap, false))
	}

	if len(globalMeta.Declarations) > 0 {
		definitionMap.Set("declarations", render3.RefsToArray(globalMeta.Declarations, false))
	}

	if len(globalMeta.Imports) > 0 {
		definitionMap.Set("imports", render3.RefsToArray(globalMeta.Imports, false))
	}

	if len(globalMeta.Exports) > 0 {
		definitionMap.Set("exports", render3.RefsToArray(globalMeta.Exports, false))
	}

	if len(globalMeta.Schemas) > 0 {
		var schemaExprs []output.Expression
		for _, ref := range globalMeta.Schemas {
			schemaExprs = append(schemaExprs, ref.Value)
		}
		definitionMap.Set("schemas", render3.LiteralExpr(nil)) // TODO: schemasExprs))
	}

	if globalMeta.Id != nil {
		definitionMap.Set("id", globalMeta.Id)
	}

	return definitionMap
}
