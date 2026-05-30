package partial

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler"
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

const MINIMUM_PARTIAL_LINKER_VERSION_INJECTABLE = "12.0.0"

func CompileDeclareInjectableFromMetadata(meta *compiler.R3InjectableMetadata) render3.R3CompiledExpression {
	definitionMap := createInjectableDefinitionMap(meta)

	expression := render3.ImportExpr(*render3.Identifiers.DeclareInjectable).CallFn([]output.Expression{definitionMap.ToLiteralMap()}, nil, false, nil)
	typeExpr := compiler.CreateInjectableType(meta.Type.Type, meta.TypeArgumentCount)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typeExpr,
		Statements: []output.Statement{},
	}
}

func createInjectableDefinitionMap(meta *compiler.R3InjectableMetadata) *render3.DefinitionMap {
	definitionMap := render3.NewDefinitionMap()

	definitionMap.Set("minVersion", render3.LiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_INJECTABLE))
	definitionMap.Set("version", render3.LiteralExpr("0.0.0-PLACEHOLDER"))
	definitionMap.Set("ngImport", render3.ImportExpr(*render3.Identifiers.Core))
	definitionMap.Set("type", meta.Type.Value)

	if meta.ProvidedIn != nil {
		providedIn := render3.ConvertFromMaybeForwardRefExpression(*meta.ProvidedIn)
		if lit, ok := providedIn.(*output.LiteralExpr); !(ok && lit.Value == nil) {
			definitionMap.Set("providedIn", providedIn)
		}
	}

	if meta.UseClass != nil {
		definitionMap.Set("useClass", render3.ConvertFromMaybeForwardRefExpression(*meta.UseClass))
	}
	if meta.UseExisting != nil {
		definitionMap.Set("useExisting", render3.ConvertFromMaybeForwardRefExpression(*meta.UseExisting))
	}
	if meta.UseValue != nil {
		definitionMap.Set("useValue", render3.ConvertFromMaybeForwardRefExpression(*meta.UseValue))
	}
	if meta.UseFactory != nil {
		definitionMap.Set("useFactory", meta.UseFactory)
	}

	if len(meta.Deps) > 0 {
		var deps []output.Expression
		for _, dep := range meta.Deps {
			deps = append(deps, CompileDependency(*dep))
		}
		definitionMap.Set("deps", render3.LiteralArr(deps))
	} else if meta.Deps != nil {
		// Empty array
		definitionMap.Set("deps", render3.LiteralArr([]output.Expression{}))
	}

	return definitionMap
}
