package tags

import (
	"fmt"
	"strings"
)

type TagContentType int

const (
	TagContentTypeRawText TagContentType = iota
	TagContentTypeEscapableRawText
	TagContentTypeParsableData
)

type TagDefinition interface {
	ClosedByParent() bool
	ImplicitNamespacePrefix() *string
	IsVoid() bool
	IgnoreFirstLf() bool
	CanSelfClose() bool
	PreventNamespaceInheritance() bool
	IsClosedByChild(name string) bool
	GetContentType(prefix *string) TagContentType
}

func SplitNsName(elementName string, fatal bool) (*string, string, error) {
	if len(elementName) == 0 || elementName[0] != ':' {
		return nil, elementName, nil
	}

	colonIndex := strings.Index(elementName[1:], ":")
	if colonIndex == -1 {
		if fatal {
			return nil, "", fmt.Errorf("Unsupported format \"%s\" expecting \":namespace:name\"", elementName)
		}
		return nil, elementName, nil
	}
	colonIndex++ // adjust for sliced string

	prefix := elementName[1:colonIndex]
	name := elementName[colonIndex+1:]
	return &prefix, name, nil
}

func IsNgContainer(tagName string) bool {
	_, name, _ := SplitNsName(tagName, false)
	return name == "ng-container"
}

func IsNgContent(tagName string) bool {
	_, name, _ := SplitNsName(tagName, false)
	return name == "ng-content"
}

func IsNgTemplate(tagName string) bool {
	_, name, _ := SplitNsName(tagName, false)
	return name == "ng-template"
}

func GetNsPrefix(fullName *string) *string {
	if fullName == nil {
		return nil
	}
	prefix, _, _ := SplitNsName(*fullName, false)
	return prefix
}

func MergeNsAndName(prefix string, localName string) string {
	if prefix != "" {
		return fmt.Sprintf(":%s:%s", prefix, localName)
	}
	return localName
}
