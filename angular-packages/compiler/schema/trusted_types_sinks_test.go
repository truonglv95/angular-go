package schema

import "testing"

func TestIsTrustedTypesSink(t *testing.T) {
	t.Run("should classify Trusted Types sinks", func(t *testing.T) {
		tests := []struct {
			tag  string
			prop string
			want bool
		}{
			{tag: "iframe", prop: "srcdoc", want: true},
			{tag: "p", prop: "innerHTML", want: true},
			{tag: "embed", prop: "src", want: true},
			{tag: "iframe", prop: "src", want: true},
			{tag: "a", prop: "href", want: false},
			{tag: "base", prop: "href", want: false},
			{tag: "div", prop: "style", want: false},
		}
		for _, tt := range tests {
			if got := IsTrustedTypesSink(tt.tag, tt.prop); got != tt.want {
				t.Fatalf("IsTrustedTypesSink(%q, %q) = %v, want %v", tt.tag, tt.prop, got, tt.want)
			}
		}
	})

	t.Run("should classify Trusted Types sinks case insensitive", func(t *testing.T) {
		tests := []struct {
			tag  string
			prop string
			want bool
		}{
			{tag: "p", prop: "iNnErHtMl", want: true},
			{tag: "p", prop: "formaction", want: false},
			{tag: "p", prop: "formAction", want: false},
		}
		for _, tt := range tests {
			if got := IsTrustedTypesSink(tt.tag, tt.prop); got != tt.want {
				t.Fatalf("IsTrustedTypesSink(%q, %q) = %v, want %v", tt.tag, tt.prop, got, tt.want)
			}
		}
	})

	t.Run("should classify attributes as Trusted Types sinks", func(t *testing.T) {
		tests := []struct {
			tag  string
			prop string
			want bool
		}{
			{tag: "p", prop: "innerHtml", want: true},
			{tag: "p", prop: "formaction", want: false},
		}
		for _, tt := range tests {
			if got := IsTrustedTypesSink(tt.tag, tt.prop); got != tt.want {
				t.Fatalf("IsTrustedTypesSink(%q, %q) = %v, want %v", tt.tag, tt.prop, got, tt.want)
			}
		}
	})
}
