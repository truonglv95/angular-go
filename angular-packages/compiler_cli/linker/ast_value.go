package linker

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/output"
	"github.com/microsoft/typescript-go/internal/ast"
)

type AstObject struct {
	expression *ast.Node
	obj        map[string]*ast.Node
	host       AstHost
}

func ParseAstObject(expression *ast.Node, host AstHost) (*AstObject, error) {
	obj, err := host.ParseObjectLiteral(expression)
	if err != nil {
		return nil, err
	}
	return &AstObject{expression: expression, obj: obj, host: host}, nil
}

func (o *AstObject) Has(propertyName string) bool {
	_, ok := o.obj[propertyName]
	return ok
}

func (o *AstObject) GetNode(propertyName string) (*ast.Node, error) {
	node, ok := o.obj[propertyName]
	if !ok {
		return nil, linkerError(o.expression, "Expected property %q to be present.", propertyName)
	}
	return node, nil
}

func (o *AstObject) GetString(propertyName string) (string, error) {
	node, err := o.GetNode(propertyName)
	if err != nil {
		return "", err
	}
	return o.host.ParseStringLiteral(node)
}

func (o *AstObject) GetBoolean(propertyName string) (bool, error) {
	node, err := o.GetNode(propertyName)
	if err != nil {
		return false, err
	}
	return o.host.ParseBooleanLiteral(node)
}

func (o *AstObject) GetObject(propertyName string) (*AstObject, error) {
	node, err := o.GetNode(propertyName)
	if err != nil {
		return nil, err
	}
	return ParseAstObject(node, o.host)
}

func (o *AstObject) GetArray(propertyName string) ([]*AstValue, error) {
	node, err := o.GetNode(propertyName)
	if err != nil {
		return nil, err
	}
	nodes, err := o.host.ParseArrayLiteral(node)
	if err != nil {
		return nil, err
	}
	values := make([]*AstValue, 0, len(nodes))
	for _, node := range nodes {
		values = append(values, &AstValue{expression: node, host: o.host})
	}
	return values, nil
}

func (o *AstObject) GetValue(propertyName string) (*AstValue, error) {
	node, err := o.GetNode(propertyName)
	if err != nil {
		return nil, err
	}
	return &AstValue{expression: node, host: o.host}, nil
}

func (o *AstObject) GetOpaque(propertyName string) (output.Expression, error) {
	node, err := o.GetNode(propertyName)
	if err != nil {
		return nil, err
	}
	return output.NewWrappedNodeExpr(node, nil, nil, nil), nil
}

func (o *AstObject) ToLiteral(mapper func(value *AstValue, key string) (any, error)) (map[string]any, error) {
	result := map[string]any{}
	for key, expression := range o.obj {
		value, err := mapper(&AstValue{expression: expression, host: o.host}, key)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

type AstValue struct {
	expression *ast.Node
	host       AstHost
}

func (v *AstValue) Node() *ast.Node {
	return v.expression
}

func (v *AstValue) GetSymbolName() string {
	return v.host.GetSymbolName(v.expression)
}

func (v *AstValue) IsString() bool {
	return v.host.IsStringLiteral(v.expression)
}

func (v *AstValue) GetString() (string, error) {
	return v.host.ParseStringLiteral(v.expression)
}

func (v *AstValue) IsBoolean() bool {
	return v.host.IsBooleanLiteral(v.expression)
}

func (v *AstValue) GetBoolean() (bool, error) {
	return v.host.ParseBooleanLiteral(v.expression)
}

func (v *AstValue) IsNumber() bool {
	return v.host.IsNumericLiteral(v.expression)
}

func (v *AstValue) GetNumber() (float64, error) {
	return v.host.ParseNumericLiteral(v.expression)
}

func (v *AstValue) IsObject() bool {
	return v.host.IsObjectLiteral(v.expression)
}

func (v *AstValue) GetObject() (*AstObject, error) {
	return ParseAstObject(v.expression, v.host)
}

func (v *AstValue) IsArray() bool {
	return v.host.IsArrayLiteral(v.expression)
}

func (v *AstValue) GetArray() ([]*AstValue, error) {
	nodes, err := v.host.ParseArrayLiteral(v.expression)
	if err != nil {
		return nil, err
	}
	values := make([]*AstValue, 0, len(nodes))
	for _, node := range nodes {
		values = append(values, &AstValue{expression: node, host: v.host})
	}
	return values, nil
}

func (v *AstValue) IsNull() bool {
	return v.host.IsNull(v.expression)
}

func (v *AstValue) GetOpaque() output.Expression {
	return output.NewWrappedNodeExpr(v.expression, nil, nil, nil)
}

func (v *AstValue) GetRange() Range {
	return v.host.GetRange(v.expression)
}
