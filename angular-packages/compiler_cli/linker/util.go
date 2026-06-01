package linker

import (
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/angular-packages/compiler/render3"
)

func linkedDefinitionFromCompiled(compiled render3.R3CompiledExpression) LinkedDefinition {
	return LinkedDefinition{
		Expression: compiled.Expression,
		Statements: compiled.Statements,
	}
}

func getDefaultStandaloneValue(version string) bool {
	major := parseMajorVersion(version)
	if major == 0 {
		return true
	}
	return major >= 19
}

func parseMajorVersion(version string) int {
	if version == "" || strings.Contains(version, "PLACEHOLDER") {
		return 0
	}
	part := strings.Split(version, ".")[0]
	major, _ := strconv.Atoi(part)
	return major
}

func toStringArray(metaObj *AstObject, property string) ([]string, error) {
	if !metaObj.Has(property) {
		return nil, nil
	}
	values, err := metaObj.GetArray(property)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		s, err := value.GetString()
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func toStringMap(metaObj *AstObject, property string) (map[string]string, error) {
	if !metaObj.Has(property) {
		return map[string]string{}, nil
	}
	return toStringMapFromObject(metaObj, property)
}

func toStringMapFromObject(metaObj *AstObject, property string) (map[string]string, error) {
	obj, err := metaObj.GetObject(property)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for key, node := range obj.obj {
		value, err := obj.host.ParseStringLiteral(node)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

func wrappedExpr(value *AstValue) output.Expression {
	return output.NewWrappedNodeExpr(value.Node(), nil, nil, nil)
}
