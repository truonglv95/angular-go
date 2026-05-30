package schema

import (
	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
)

type ValidationResult struct {
	Error bool
	Msg   *string
}

type AnimationStyleNormalizationResult struct {
	Error string
	Value string
}

type ElementSchemaRegistry interface {
	HasProperty(tagName string, propName string, schemaMetas []core.SchemaMetadata) bool
	HasElement(tagName string, schemaMetas []core.SchemaMetadata) bool
	SecurityContext(elementName string, propName string, isAttribute bool) core.SecurityContext
	AllKnownElementNames() []string
	GetMappedPropName(propName string) string
	GetDefaultComponentElementName() string
	ValidateProperty(name string) ValidationResult
	ValidateAttribute(name string) ValidationResult
	NormalizeAnimationStyleProperty(propName string) string
	NormalizeAnimationStyleValue(camelCaseProp string, userProvidedProp string, val any) AnimationStyleNormalizationResult
}
