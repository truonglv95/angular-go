package compiler

import "testing"

func TestIsStyleUrlResolvable(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "should resolve relative urls", url: "someUrl.css", want: true},
		{name: "should resolve package urls", url: "package:someUrl.css", want: true},
		{name: "should resolve asset urls", url: "asset:someUrl.css", want: true},
		{name: "should not resolve empty urls", url: "", want: false},
		{name: "should not resolve urls with other schema", url: "http://otherurl", want: false},
		{name: "should not resolve urls with absolute paths", url: "/otherurl", want: false},
		{name: "should not resolve protocol-relative absolute paths", url: "//otherurl", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStyleUrlResolvable(tt.url); got != tt.want {
				t.Fatalf("IsStyleUrlResolvable(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
