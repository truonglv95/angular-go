package ml_parser

import "github.com/microsoft/typescript-go/angular-packages/compiler/tags"

type XmlTagDefinition struct{}

func NewXmlTagDefinition() *XmlTagDefinition {
	return &XmlTagDefinition{}
}

func (t *XmlTagDefinition) ClosedByParent() bool {
	return false
}

func (t *XmlTagDefinition) ImplicitNamespacePrefix() *string {
	return nil
}

func (t *XmlTagDefinition) IsVoid() bool {
	return false
}

func (t *XmlTagDefinition) IgnoreFirstLf() bool {
	return false
}

func (t *XmlTagDefinition) CanSelfClose() bool {
	return true
}

func (t *XmlTagDefinition) PreventNamespaceInheritance() bool {
	return false
}

func (t *XmlTagDefinition) RequireExtraParent(currentParent string) bool {
	return false
}

func (t *XmlTagDefinition) IsClosedByChild(name string) bool {
	return false
}

func (t *XmlTagDefinition) GetContentType(prefix *string) tags.TagContentType {
	return tags.TagContentTypeParsableData
}

var _TAG_DEFINITION = NewXmlTagDefinition()

func GetXmlTagDefinition(tagName string) *XmlTagDefinition {
	return _TAG_DEFINITION
}
