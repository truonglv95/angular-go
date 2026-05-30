package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

const MINIMUM_PARTIAL_LINKER_VERSION = "12.0.0"

func CompileDeclareFactoryFunction(meta R3DeclareFactoryMetadata) render3.R3CompiledExpression {
	definitionMap := render3.NewDefinitionMap()
	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))
	definitionMap.Set("type", meta.Type)
	definitionMap.Set("deps", CompileDependencies(meta.Deps))

	var targetName string
	switch meta.Target {
	case render3.FactoryTargetDirective:
		targetName = "Directive"
	case render3.FactoryTargetComponent:
		targetName = "Component"
	case render3.FactoryTargetPipe:
		targetName = "Pipe"
	case render3.FactoryTargetInjectable:
		targetName = "Injectable"
	case render3.FactoryTargetNgModule:
		targetName = "NgModule"
	}
	definitionMap.Set("target", render3.ImportExpr(*render3.Identifiers.FactoryTarget).Prop(targetName, nil))

	return render3.R3CompiledExpression{
		Expression: render3.ImportExpr(*render3.Identifiers.DeclareFactory).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil),
		Statements: []output.Statement{},
		Type:       output.INFERRED_TYPE,
	}
}
