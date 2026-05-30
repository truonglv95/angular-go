package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

const MINIMUM_PARTIAL_LINKER_VERSION_CLASS = "12.0.0"
const MINIMUM_PARTIAL_LINKER_DEFER_SUPPORT_VERSION = "18.0.0"

func CompileDeclareClassMetadata(metadata render3.R3ClassMetadata) output.Expression {
	definitionMap := render3.NewDefinitionMap()
	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_CLASS))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))
	definitionMap.Set("type", metadata.Type)
	definitionMap.Set("decorators", metadata.Decorators)

	if metadata.CtorParameters != nil {
		definitionMap.Set("ctorParameters", metadata.CtorParameters)
	}
	if metadata.PropDecorators != nil {
		definitionMap.Set("propDecorators", metadata.PropDecorators)
	}

	return render3.ImportExpr(*render3.Identifiers.DeclareClassMetadata).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
}

func CompileComponentDeclareClassMetadata(
	metadata render3.R3ClassMetadata,
	dependencies []render3.R3DeferPerComponentDependency,
) output.Expression {
	if len(dependencies) == 0 {
		return CompileDeclareClassMetadata(metadata)
	}

	definitionMap := render3.NewDefinitionMap()
	callbackReturnDefinitionMap := render3.NewDefinitionMap()
	callbackReturnDefinitionMap.Set("decorators", metadata.Decorators)

	if metadata.CtorParameters != nil {
		callbackReturnDefinitionMap.Set("ctorParameters", metadata.CtorParameters)
	} else {
		callbackReturnDefinitionMap.Set("ctorParameters", render3.LiteralExpr(nil))
	}

	if metadata.PropDecorators != nil {
		callbackReturnDefinitionMap.Set("propDecorators", metadata.PropDecorators)
	} else {
		callbackReturnDefinitionMap.Set("propDecorators", render3.LiteralExpr(nil))
	}

	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_DEFER_SUPPORT_VERSION))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))
	definitionMap.Set("type", metadata.Type)
	definitionMap.Set("resolveDeferredDeps", render3.CompileComponentMetadataAsyncResolver(dependencies))

	var fnParams []*output.FnParam
	for _, dep := range dependencies {
		fnParams = append(fnParams, &output.FnParam{Name: dep.SymbolName, Type: output.DYNAMIC_TYPE})
	}
	definitionMap.Set("resolveMetadata", output.NewArrowFunctionExpr(fnParams, callbackReturnDefinitionMap.ToLiteralMap(), nil, nil, nil))

	return render3.ImportExpr(*render3.Identifiers.DeclareClassMetadataAsync).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
}
