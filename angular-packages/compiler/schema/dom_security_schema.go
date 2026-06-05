package schema

import (
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
)

var (
	securitySchema map[string]core.SecurityContext
	securitySchemaOnce sync.Once
)

func SECURITY_SCHEMA() map[string]core.SecurityContext {
	securitySchemaOnce.Do(func() {
		securitySchema = make(map[string]core.SecurityContext)

		registerSecurityContext(core.SecurityContextHTML, "", []securityContextSpec{
			{Tag: "iframe", Attrs: []string{"srcdoc"}},
			{Tag: "*", Attrs: []string{"innerHTML", "outerHTML"}},
		})
		registerSecurityContext(core.SecurityContextStyle, "", []securityContextSpec{
			{Tag: "*", Attrs: []string{"style"}},
		})
		registerSecurityContext(core.SecurityContextURL, "", []securityContextSpec{
			{Tag: "*", Attrs: []string{"formAction"}},
			{Tag: "area", Attrs: []string{"href"}},
			{Tag: "a", Attrs: []string{"href", "xlink:href"}},
			{Tag: "form", Attrs: []string{"action"}},
			{Tag: "img", Attrs: []string{"src"}},
			{Tag: "video", Attrs: []string{"src"}},
		})

		registerSecurityContext(core.SecurityContextURL, core.MATH_ML_NAMESPACE, []securityContextSpec{
			{Tag: "annotation", Attrs: []string{"href", "xlink:href"}},
			{Tag: "annotation-xml", Attrs: []string{"href", "xlink:href"}},
			{Tag: "maction", Attrs: []string{"href", "xlink:href"}},
			{Tag: "malignmark", Attrs: []string{"href", "xlink:href"}},
			{Tag: "math", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mroot", Attrs: []string{"href", "xlink:href"}},
			{Tag: "msqrt", Attrs: []string{"href", "xlink:href"}},
			{Tag: "merror", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mfrac", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mglyph", Attrs: []string{"href", "xlink:href"}},
			{Tag: "msub", Attrs: []string{"href", "xlink:href"}},
			{Tag: "msup", Attrs: []string{"href", "xlink:href"}},
			{Tag: "msubsup", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mmultiscripts", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mprescripts", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mi", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mn", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mo", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mpadded", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mphantom", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mrow", Attrs: []string{"href", "xlink:href"}},
			{Tag: "ms", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mspace", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mstyle", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mtable", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mtd", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mtr", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mtext", Attrs: []string{"href", "xlink:href"}},
			{Tag: "mover", Attrs: []string{"href", "xlink:href"}},
			{Tag: "munder", Attrs: []string{"href", "xlink:href"}},
			{Tag: "munderover", Attrs: []string{"href", "xlink:href"}},
			{Tag: "semantics", Attrs: []string{"href", "xlink:href"}},
			{Tag: "none", Attrs: []string{"href", "xlink:href"}},
		})

		registerSecurityContext(core.SecurityContextResourceURL, "", []securityContextSpec{
			{Tag: "base", Attrs: []string{"href"}},
			{Tag: "embed", Attrs: []string{"src"}},
			{Tag: "frame", Attrs: []string{"src"}},
			{Tag: "iframe", Attrs: []string{"src"}},
			{Tag: "link", Attrs: []string{"href"}},
			{Tag: "object", Attrs: []string{"codebase", "data"}},
		})
		registerSecurityContext(core.SecurityContextURL, core.SVG_NAMESPACE, []securityContextSpec{
			{Tag: "a", Attrs: []string{"href", "xlink:href"}},
		})
		registerSecurityContext(core.SecurityContextAttributeNoBinding, core.SVG_NAMESPACE, []securityContextSpec{
			{Tag: "animate", Attrs: []string{"attributeName", "values", "to", "from"}},
			{Tag: "set", Attrs: []string{"to", "attributeName"}},
			{Tag: "animateMotion", Attrs: []string{"attributeName"}},
			{Tag: "animateTransform", Attrs: []string{"attributeName"}},
		})
		registerSecurityContext(core.SecurityContextAttributeNoBinding, "", []securityContextSpec{
			{Tag: "unknown", Attrs: []string{"attributeName", "values", "to", "from", "sandbox", "allow", "allowFullscreen", "referrerPolicy", "csp", "fetchPriority"}},
			{Tag: "iframe", Attrs: []string{"sandbox", "allow", "allowFullscreen", "referrerPolicy", "csp", "fetchPriority"}},
		})
	})
	return securitySchema
}

type securityContextSpec struct {
	Tag   string
	Attrs []string
}

func registerSecurityContext(ctx core.SecurityContext, namespace string, specs []securityContextSpec) {
	for _, spec := range specs {
		tagName := spec.Tag
		if namespace != "" && tagName != "*" && tagName != "unknown" {
			tagName = ":" + namespace + ":" + tagName
		}
		tagName = strings.ToLower(tagName)
		for _, attr := range spec.Attrs {
			securitySchema[tagName+"|"+strings.ToLower(attr)] = ctx
		}
	}
}
