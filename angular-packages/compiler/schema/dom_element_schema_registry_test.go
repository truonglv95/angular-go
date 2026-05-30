package schema

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
)

func TestDomElementSchemaRegistry(t *testing.T) {
	registry := NewDomElementSchemaRegistry()

	t.Run("should detect elements", func(t *testing.T) {
		for _, tag := range []string{"div", "b", "ng-container", "ng-content"} {
			if !registry.HasElement(tag, nil) {
				t.Fatalf("HasElement(%q) = false, want true", tag)
			}
		}
		for _, tag := range []string{"my-cmp", "abc"} {
			if registry.HasElement(tag, nil) {
				t.Fatalf("HasElement(%q) = true, want false", tag)
			}
		}
	})

	t.Run("should detect elements missing from chrome", func(t *testing.T) {
		for _, tag := range []string{"data", "menuitem", "summary", "time"} {
			if !registry.HasElement(tag, nil) {
				t.Fatalf("HasElement(%q) = false, want true", tag)
			}
		}
	})

	t.Run("should detect properties on regular elements", func(t *testing.T) {
		for _, tc := range []struct {
			tag  string
			prop string
		}{
			{"div", "id"},
			{"div", "title"},
			{"div", "inert"},
			{"h1", "align"},
			{"h2", "align"},
			{"h3", "align"},
			{"h4", "align"},
			{"h5", "align"},
			{"h6", "align"},
			{"textarea", "disabled"},
			{"input", "disabled"},
		} {
			if !registry.HasProperty(tc.tag, tc.prop, nil) {
				t.Fatalf("HasProperty(%q, %q) = false, want true", tc.tag, tc.prop)
			}
		}
		for _, tc := range []struct {
			tag  string
			prop string
		}{
			{"h7", "align"},
			{"div", "unknown"},
		} {
			if registry.HasProperty(tc.tag, tc.prop, nil) {
				t.Fatalf("HasProperty(%q, %q) = true, want false", tc.tag, tc.prop)
			}
		}
	})

	t.Run("should detect properties on elements missing from Chrome", func(t *testing.T) {
		for _, tc := range []struct {
			tag  string
			prop string
		}{
			{"data", "value"},
			{"menuitem", "type"},
			{"menuitem", "default"},
			{"time", "dateTime"},
		} {
			if !registry.HasProperty(tc.tag, tc.prop, nil) {
				t.Fatalf("HasProperty(%q, %q) = false, want true", tc.tag, tc.prop)
			}
		}
	})

	t.Run("should detect different kinds of types", func(t *testing.T) {
		for _, prop := range []string{"className", "id", "scrollLeft", "height", "autoplay", "classList"} {
			if !registry.HasProperty("video", prop, nil) {
				t.Fatalf("HasProperty(video, %q) = false, want true", prop)
			}
		}
		if registry.HasProperty("video", "click", nil) {
			t.Fatal("HasProperty(video, click) = true, want false")
		}
	})

	t.Run("should treat custom elements as an unknown element by default", func(t *testing.T) {
		if registry.HasProperty("custom-like", "unknown", nil) {
			t.Fatal("HasProperty(custom-like, unknown) = true, want false")
		}
		for _, prop := range []string{"className", "style", "id"} {
			if !registry.HasProperty("custom-like", prop, nil) {
				t.Fatalf("HasProperty(custom-like, %q) = false, want true", prop)
			}
		}
	})

	t.Run("should return true for custom-like elements if CUSTOM_ELEMENTS_SCHEMA was used", func(t *testing.T) {
		schemas := []core.SchemaMetadata{core.CustomElementsSchema}
		if !registry.HasProperty("custom-like", "unknown", schemas) {
			t.Fatal("HasProperty custom schema = false, want true")
		}
		if !registry.HasElement("custom-like", schemas) {
			t.Fatal("HasElement custom schema = false, want true")
		}
	})

	t.Run("should return true for all elements if NO_ERRORS_SCHEMA was used", func(t *testing.T) {
		schemas := []core.SchemaMetadata{core.NoErrorsSchema}
		for _, tc := range []struct {
			tag  string
			prop string
		}{
			{"custom-like", "unknown"},
			{"a", "unknown"},
		} {
			if !registry.HasProperty(tc.tag, tc.prop, schemas) {
				t.Fatalf("HasProperty(%q, %q, NO_ERRORS_SCHEMA) = false, want true", tc.tag, tc.prop)
			}
		}
		for _, tag := range []string{"custom-like", "unknown"} {
			if !registry.HasElement(tag, schemas) {
				t.Fatalf("HasElement(%q, NO_ERRORS_SCHEMA) = false, want true", tag)
			}
		}
	})

	t.Run("should remap property names that are specified in DOM facade", func(t *testing.T) {
		if got := registry.GetMappedPropName("readonly"); got != "readOnly" {
			t.Fatalf("GetMappedPropName(readonly) = %q, want readOnly", got)
		}
	})

	t.Run("should not remap property names that are not specified in DOM facade", func(t *testing.T) {
		for _, prop := range []string{"title", "exotic-unknown"} {
			if got := registry.GetMappedPropName(prop); got != prop {
				t.Fatalf("GetMappedPropName(%q) = %q, want %q", prop, got, prop)
			}
		}
	})

	t.Run("should return an error message when asserting event properties", func(t *testing.T) {
		report := registry.ValidateProperty("onClick")
		if !report.Error || report.Msg == nil {
			t.Fatal("ValidateProperty(onClick) did not report error")
		}
		want := "Binding to event property 'onClick' is disallowed for security reasons, please use (Click)=...\nIf 'onClick' is a directive input, make sure the directive is imported by the current module."
		if *report.Msg != want {
			t.Fatalf("ValidateProperty(onClick) msg = %q, want %q", *report.Msg, want)
		}
		report = registry.ValidateProperty("onAnything")
		if !report.Error || report.Msg == nil {
			t.Fatal("ValidateProperty(onAnything) did not report error")
		}
		want = "Binding to event property 'onAnything' is disallowed for security reasons, please use (Anything)=...\nIf 'onAnything' is a directive input, make sure the directive is imported by the current module."
		if *report.Msg != want {
			t.Fatalf("ValidateProperty(onAnything) msg = %q, want %q", *report.Msg, want)
		}
	})

	t.Run("should return an error message when asserting event attributes", func(t *testing.T) {
		report := registry.ValidateAttribute("onClick")
		if !report.Error || report.Msg == nil {
			t.Fatal("ValidateAttribute(onClick) did not report error")
		}
		want := "Binding to event attribute 'onClick' is disallowed for security reasons, please use (Click)=..."
		if *report.Msg != want {
			t.Fatalf("ValidateAttribute(onClick) msg = %q, want %q", *report.Msg, want)
		}
		report = registry.ValidateAttribute("onAnything")
		if !report.Error || report.Msg == nil {
			t.Fatal("ValidateAttribute(onAnything) did not report error")
		}
		want = "Binding to event attribute 'onAnything' is disallowed for security reasons, please use (Anything)=..."
		if *report.Msg != want {
			t.Fatalf("ValidateAttribute(onAnything) msg = %q, want %q", *report.Msg, want)
		}
	})

	t.Run("should not return an error message when asserting non-event properties or attributes", func(t *testing.T) {
		for _, prop := range []string{"title", "exotic-unknown"} {
			report := registry.ValidateProperty(prop)
			if report.Error || report.Msg != nil {
				t.Fatalf("ValidateProperty(%q) = %#v, want no error", prop, report)
			}
		}
	})

	t.Run("should return security contexts for elements", func(t *testing.T) {
		tests := []struct {
			tag       string
			prop      string
			attribute bool
			want      core.SecurityContext
		}{
			{"iframe", "srcdoc", false, core.SecurityContextHTML},
			{"p", "innerHTML", false, core.SecurityContextHTML},
			{"a", "href", false, core.SecurityContextURL},
			{"a", "style", false, core.SecurityContextStyle},
			{"base", "href", false, core.SecurityContextResourceURL},
			{":svg:animate", "to", false, core.SecurityContextAttributeNoBinding},
			{":svg:animate", "from", false, core.SecurityContextAttributeNoBinding},
			{":svg:animate", "values", false, core.SecurityContextAttributeNoBinding},
			{":svg:set", "to", false, core.SecurityContextAttributeNoBinding},
			{":svg:a", "href", false, core.SecurityContextURL},
			{":svg:a", "xlink:href", false, core.SecurityContextURL},
			{":svg:a", "href", true, core.SecurityContextURL},
			{":svg:a", "xlink:href", true, core.SecurityContextURL},
		}
		for _, tt := range tests {
			if got := registry.SecurityContext(tt.tag, tt.prop, tt.attribute); got != tt.want {
				t.Fatalf("SecurityContext(%q, %q, %v) = %v, want %v", tt.tag, tt.prop, tt.attribute, got, tt.want)
			}
		}
	})

	t.Run("should detect properties on namespaced elements", func(t *testing.T) {
		if !registry.HasProperty(":svg:style", "type", nil) {
			t.Fatal("HasProperty(:svg:style, type) = false, want true")
		}
	})

	t.Run("should check security contexts case insensitive", func(t *testing.T) {
		tests := []struct {
			prop string
			want core.SecurityContext
		}{
			{"iNnErHtMl", core.SecurityContextHTML},
			{"formaction", core.SecurityContextURL},
			{"formAction", core.SecurityContextURL},
		}
		for _, tt := range tests {
			if got := registry.SecurityContext("p", tt.prop, false); got != tt.want {
				t.Fatalf("SecurityContext(p, %q, false) = %v, want %v", tt.prop, got, tt.want)
			}
		}
	})

	t.Run("should check security contexts for attributes", func(t *testing.T) {
		tests := []struct {
			prop string
			want core.SecurityContext
		}{
			{"innerHtml", core.SecurityContextHTML},
			{"formaction", core.SecurityContextURL},
		}
		for _, tt := range tests {
			if got := registry.SecurityContext("p", tt.prop, true); got != tt.want {
				t.Fatalf("SecurityContext(p, %q, true) = %v, want %v", tt.prop, got, tt.want)
			}
		}
	})

	t.Run("Angular custom elements", func(t *testing.T) {
		if registry.HasProperty("ng-container", "id", nil) {
			t.Fatal("HasProperty(ng-container, id) = true, want false")
		}
		if registry.HasProperty("ng-content", "id", nil) {
			t.Fatal("HasProperty(ng-content, id) = true, want false")
		}
		if registry.HasProperty("ng-content", "select", nil) {
			t.Fatal("HasProperty(ng-content, select) = true, want false")
		}
	})

	t.Run("Custom XML/XHTML namespaces", func(t *testing.T) {
		for _, tag := range []string{":xhtml:a", ":foo:div"} {
			if !registry.HasElement(tag, nil) {
				t.Fatalf("HasElement(%q) = false, want true", tag)
			}
		}
		for _, tc := range []struct {
			tag  string
			prop string
		}{
			{":xhtml:a", "href"},
			{":foo:div", "id"},
		} {
			if !registry.HasProperty(tc.tag, tc.prop, nil) {
				t.Fatalf("HasProperty(%q, %q) = false, want true", tc.tag, tc.prop)
			}
		}
		tests := []struct {
			tag  string
			prop string
			want core.SecurityContext
		}{
			{":xhtml:a", "href", core.SecurityContextURL},
			{":foo:div", "innerHTML", core.SecurityContextHTML},
		}
		for _, tt := range tests {
			if got := registry.SecurityContext(tt.tag, tt.prop, false); got != tt.want {
				t.Fatalf("SecurityContext(%q, %q, false) = %v, want %v", tt.tag, tt.prop, got, tt.want)
			}
		}
	})

	t.Run("normalizeAnimationStyleProperty", func(t *testing.T) {
		tests := map[string]string{
			"border-radius":     "borderRadius",
			"zIndex":            "zIndex",
			"-webkit-animation": "WebkitAnimation",
		}
		for input, want := range tests {
			if got := registry.NormalizeAnimationStyleProperty(input); got != want {
				t.Fatalf("NormalizeAnimationStyleProperty(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("normalizeAnimationStyleValue", func(t *testing.T) {
		if got := registry.NormalizeAnimationStyleValue("borderRadius", "border-radius", 10).Value; got != "10px" {
			t.Fatalf("borderRadius 10 = %q, want 10px", got)
		}
		if got := registry.NormalizeAnimationStyleValue("opacity", "opacity", 0).Value; got != "0" {
			t.Fatalf("opacity 0 = %q, want 0", got)
		}
		if got := registry.NormalizeAnimationStyleValue("width", "width", 0).Value; got != "0" {
			t.Fatalf("width 0 = %q, want 0", got)
		}
		if got := registry.NormalizeAnimationStyleValue("borderRadius", "border-radius", "10em").Value; got != "10em" {
			t.Fatalf("borderRadius 10em = %q, want 10em", got)
		}
		if got := registry.NormalizeAnimationStyleValue("color", "color", "   red ").Value; got != "red" {
			t.Fatalf("color red = %q, want red", got)
		}
		if got := registry.NormalizeAnimationStyleValue("zIndex", "zIndex", 10).Value; got != "10" {
			t.Fatalf("zIndex 10 = %q, want 10", got)
		}
		if got := registry.NormalizeAnimationStyleValue("opacity", "opacity", 0.5).Value; got != "0.5" {
			t.Fatalf("opacity 0.5 = %q, want 0.5", got)
		}
	})

	t.Run("should support aria property if attribute is also supported", func(t *testing.T) {
		rootSchema := SCHEMA[0]
		for _, prop := range ATTR_TO_PROP {
			if len(prop) >= len("aria") && prop[:len("aria")] == "aria" && !contains(rootSchema, prop) {
				t.Fatalf("root schema does not contain %q", prop)
			}
		}
	})
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return len(needle) == 0
}
