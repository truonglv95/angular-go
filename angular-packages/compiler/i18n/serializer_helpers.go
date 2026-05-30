package i18n

import (
	"strings"

	xmlser "github.com/microsoft/typescript-go/angular-packages/compiler/i18n/serializers"
)

func xmbToPublicName(internalName string) string {
	var sb strings.Builder
	for _, c := range strings.ToUpper(internalName) {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			sb.WriteRune(c)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

func getXliffCtypeForTag(tag string) string {
	switch strings.ToLower(tag) {
	case "br":
		return "lb"
	case "img":
		return "image"
	default:
		return "x-" + tag
	}
}

func getXliff2TypeForTag(tag string) string {
	switch strings.ToLower(tag) {
	case "br", "b", "i", "u":
		return "fmt"
	case "img":
		return "image"
	case "a":
		return "link"
	default:
		return "other"
	}
}

func serializeI18nXML(nodes []xmlser.Node) string {
	return xmlser.Serialize(nodes)
}
