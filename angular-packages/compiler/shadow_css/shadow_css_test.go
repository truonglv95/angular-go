package shadow_css

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestShadowCss(t *testing.T) {
	t.Run("should handle empty string", func(t *testing.T) {
		assertEqualCss(t, shimCss("", "contenta", ""), "")
	})

	t.Run("should add an attribute to every rule", func(t *testing.T) {
		css := "one {color: red;}two {color: red;}"
		expected := "one[contenta] {color:red;}two[contenta] {color:red;}"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)
	})

	t.Run("should handle invalid css", func(t *testing.T) {
		css := "one {color: red;}garbage"
		expected := "one[contenta] {color:red;}garbage"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)
	})

	t.Run("should add an attribute to every selector", func(t *testing.T) {
		css := "one, two {color: red;}"
		expected := "one[contenta], two[contenta] {color:red;}"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)
	})

	t.Run("should support newlines in the selector and content", func(t *testing.T) {
		css := `
      one,
      two {
        color: red;
      }
    `
		expected := `
      one[contenta],
      two[contenta] {
        color: red;
      }
    `
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)
	})

	t.Run("should support newlines in the same selector and content", func(t *testing.T) {
		selector := `.foo:not(
      .bar) {
        background-color:
          green;
    }`
		assertEqualCss(t, shimCss(selector, "contenta", "a-host"), ".foo[contenta]:not( .bar) { background-color:green;}")
	})

	t.Run("should handle complicated selectors", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{"one::before {}", "one[contenta]::before {}"},
			{"one two {}", "one[contenta] two[contenta] {}"},
			{"one > two {}", "one[contenta] > two[contenta] {}"},
			{"one + two {}", "one[contenta] + two[contenta] {}"},
			{"one ~ two {}", "one[contenta] ~ two[contenta] {}"},
			{".one.two > three {}", ".one.two[contenta] > three[contenta] {}"},
			{`one[attr="value"] {}`, `one[attr="value"][contenta] {}`},
			{"one[attr=value] {}", "one[attr=value][contenta] {}"},
			{`one[attr^="value"] {}`, `one[attr^="value"][contenta] {}`},
			{`one[attr$="value"] {}`, `one[attr$="value"][contenta] {}`},
			{`one[attr*="value"] {}`, `one[attr*="value"][contenta] {}`},
			{`one[attr|="value"] {}`, `one[attr|="value"][contenta] {}`},
			{`one[attr~="value"] {}`, `one[attr~="value"][contenta] {}`},
			{`one[attr="va lue"] {}`, `one[attr="va lue"][contenta] {}`},
			{"one[attr] {}", "one[attr][contenta] {}"},
			{`[is="one"] {}`, `[is="one"][contenta] {}`},
			{"[attr] {}", "[attr][contenta] {}"},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", ""), tt.want)
		}
	})

	t.Run("should transform :host with attributes", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{":host [attr] {}", "[hosta] [attr][contenta] {}"},
			{":host(create-first-project) {}", "create-first-project[hosta] {}"},
			{":host[attr] {}", "[attr][hosta] {}"},
			{":host[attr]:where(:not(.cm-button)) {}", "[attr][hosta]:where(:not(.cm-button)) {}"},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", "hosta"), tt.want)
		}
	})

	t.Run("should leave calc unchanged", func(t *testing.T) {
		styleStr := "div {height:calc(100% - 55px);}"
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), "div[contenta] {height:calc(100% - 55px);}")
	})

	t.Run("should shim rules with quoted content", func(t *testing.T) {
		styleStr := `div {background-image: url("a.jpg"); color: red;}`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), `div[contenta] {background-image:url("a.jpg"); color:red;}`)
	})

	t.Run("should handle when quoted content contains a closing parenthesis", func(t *testing.T) {
		assertEqualCss(t, shimCss(`p { background-image: url(")") } p { color: red }`, "contenta", ""), `p[contenta] { background-image: url(")") } p[contenta] { color: red }`)
	})

	t.Run("should shim rules with an escaped quote inside quoted content", func(t *testing.T) {
		styleStr := `div::after { content: "\"" }`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), `div[contenta]::after { content:"\""}`)
	})

	t.Run("should shim rules with curly braces inside quoted content", func(t *testing.T) {
		styleStr := `div::after { content: "{}" }`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), `div[contenta]::after { content:"{}"}`)
	})

	t.Run("should retain multiline selectors", func(t *testing.T) {
		styleStr := ".foo,\n.bar { color: red;}"
		if got := shimCss(styleStr, "contenta", ""); got != ".foo[contenta], \n.bar[contenta] { color: red;}" {
			t.Fatalf("multiline selector shim = %q", got)
		}
	})
}

func TestShadowCssComments(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want string
	}{
		{"should remove inline comments without adding extra lines", "/* b {} */ b {}", " b[contenta] {}"},
		{"should preserve internal newlines from multiline comments", "/* b {}\n */ b {}", "\n b[contenta] {}"},
		{"should remove multiple inline comments without adding extra lines", "/* b {} */ b {} /* a {} */ a {}", " b[contenta] {}  a[contenta] {}"},
		{"should keep sourceMappingURL comments", "b {} /*# sourceMappingURL=data:x */", "b[contenta] {} /*# sourceMappingURL=data:x */"},
		{"should keep spaced sourceMappingURL comments", "b {}/* #sourceMappingURL=data:x */", "b[contenta] {}/* #sourceMappingURL=data:x */"},
		{"should handle adjacent comments", "/* comment 1 */ /* comment 2 */ b {}", "  b[contenta] {}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shimCss(tt.css, "contenta", ""); got != tt.want {
				t.Fatalf("shimCss comments = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShadowCssEscapedSequencesAndPseudoFunctions(t *testing.T) {
	t.Run("should handle escaped sequences in selectors", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{`one\/two {}`, `one\/two[contenta] {}`},
			{`one\:two {}`, `one\:two[contenta] {}`},
			{`one\\:two {}`, `one\\[contenta]:two {}`},
			{`.one\:two {}`, `.one\:two[contenta] {}`},
			{`.one\:\fc ber {}`, `.one\:\fc ber[contenta] {}`},
			{`.one\:two .three\:four {}`, `.one\:two[contenta] .three\:four[contenta] {}`},
			{`div:where(.one) {}`, `div[contenta]:where(.one) {}`},
			{`div:where() {}`, `div[contenta]:where() {}`},
			{`:where(a):where(b) {}`, `:where(a[contenta]):where(b[contenta]) {}`},
			{`*:where(.one) {}`, `*[contenta]:where(.one) {}`},
			{`*:where(.one) ::ng-deep .foo {}`, `*[contenta]:where(.one) .foo {}`},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", "hosta"), tt.want)
		}
	})

	t.Run("should handle pseudo functions correctly", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{`:where(.one) {}`, `:where(.one[contenta]) {}`},
			{`:where(div.one span.two) {}`, `:where(div.one[contenta] span.two[contenta]) {}`},
			{`:where(.one) .two {}`, `:where(.one[contenta]) .two[contenta] {}`},
			{`:where(:host) {}`, `:where([hosta]) {}`},
			{`:where(:host) .one {}`, `:where([hosta]) .one[contenta] {}`},
			{`:where(.one) :where(:host) {}`, `:where(.one) :where([hosta]) {}`},
			{`:where(.one :host) {}`, `:where(.one [hosta]) {}`},
			{`div :where(.one) {}`, `div[contenta] :where(.one[contenta]) {}`},
			{`:host :where(.one .two) {}`, `[hosta] :where(.one[contenta] .two[contenta]) {}`},
			{`:where(.one, .two) {}`, `:where(.one[contenta], .two[contenta]) {}`},
			{`:where(.one > .two) {}`, `:where(.one[contenta] > .two[contenta]) {}`},
			{`:where(> .one) {}`, `:where( > .one[contenta]) {}`},
			{`:where(:not(.one) ~ .two) {}`, `:where([contenta]:not(.one) ~ .two[contenta]) {}`},
			{`:where([foo]) {}`, `:where([foo][contenta]) {}`},
			{`div:is(.foo) {}`, `div[contenta]:is(.foo) {}`},
			{`:is(.dark :host) {}`, `:is(.dark [hosta]) {}`},
			{`:is(.dark) :is(:host) {}`, `:is(.dark) :is([hosta]) {}`},
			{`:host:is(.foo) {}`, `[hosta]:is(.foo) {}`},
			{`:is(.foo) {}`, `:is(.foo[contenta]) {}`},
			{`:is(.foo, .bar, .baz) {}`, `:is(.foo[contenta], .bar[contenta], .baz[contenta]) {}`},
			{`:is(.foo, .bar) :host {}`, `:is(.foo, .bar) [hosta] {}`},
			{`:is(.foo, .bar) :is(.baz) :where(.one, .two) :host :where(.three:first-child) {}`, `:is(.foo, .bar) :is(.baz) :where(.one, .two) [hosta] :where(.three[contenta]:first-child) {}`},
			{`:where(:is(a)) {}`, `:where(:is(a[contenta])) {}`},
			{`:where(:is(a, b)) {}`, `:where(:is(a[contenta], b[contenta])) {}`},
			{`:where(:host:is(.one, .two)) {}`, `:where([hosta]:is(.one, .two)) {}`},
			{`:where(:host :is(.one, .two)) {}`, `:where([hosta] :is(.one[contenta], .two[contenta])) {}`},
			{`:where(:is(a, b) :is(.one, .two)) {}`, `:where(:is(a[contenta], b[contenta]) :is(.one[contenta], .two[contenta])) {}`},
			{`:where(:where(a:has(.foo), b) :is(.one, .two:where(.foo > .bar))) {}`, `:where(:where(a[contenta]:has(.foo), b[contenta]) :is(.one[contenta], .two[contenta]:where(.foo > .bar))) {}`},
			{`:where(.two):first-child {}`, `[contenta]:where(.two):first-child {}`},
			{`:first-child:where(.two) {}`, `[contenta]:first-child:where(.two) {}`},
			{`:where(.two):nth-child(3) {}`, `[contenta]:where(.two):nth-child(3) {}`},
			{`table :where(td, th):hover { color: lime; }`, `table[contenta] [contenta]:where(td, th):hover { color:lime;}`},
			{`:nth-child(3n of :not(p, a), :is(.foo)) {}`, `[contenta]:nth-child(3n of :not(p, a), :is(.foo)) {}`},
			{`li:nth-last-child(-n + 3) {}`, `li[contenta]:nth-last-child(-n + 3) {}`},
			{`dd:nth-last-of-type(3n) {}`, `dd[contenta]:nth-last-of-type(3n) {}`},
			{`dd:nth-of-type(even) {}`, `dd[contenta]:nth-of-type(even) {}`},
			{`:host:is([foo],[foo-2])>div.example-2 {}`, `[hosta]:is([foo],[foo-2]) > div.example-2[contenta] {}`},
			{`:host:is([foo], [foo-2]) > div.example-2 {}`, `[hosta]:is([foo], [foo-2]) > div.example-2[contenta] {}`},
			{`:host:has([foo],[foo-2])>div.example-2 {}`, `[hosta]:has([foo],[foo-2]) > div.example-2[contenta] {}`},
			{`div:has(a) {}`, `div[contenta]:has(a) {}`},
			{`div:has(a) :host {}`, `div:has(a) [hosta] {}`},
			{`:has(a) :host :has(b) {}`, `:has(a) [hosta] [contenta]:has(b) {}`},
			{`div:has(~ .one) {}`, `div[contenta]:has(~ .one) {}`},
			{`:has(a) :has(b) {}`, `[contenta]:has(a) [contenta]:has(b) {}`},
			{`:has(a, b) {}`, `[contenta]:has(a, b) {}`},
			{`:has(a, b:where(.foo), :is(.bar)) {}`, `[contenta]:has(a, b:where(.foo), :is(.bar)) {}`},
			{`:has(a, b:where(.foo), :is(.bar):first-child):first-letter {}`, `[contenta]:has(a, b:where(.foo), :is(.bar):first-child):first-letter {}`},
			{`:where(a, b:where(.foo), :has(.bar):first-child) {}`, `:where(a[contenta], b[contenta]:where(.foo), [contenta]:has(.bar):first-child) {}`},
			{`:has(.one :host, .two) {}`, `[contenta]:has(.one [hosta], .two) {}`},
			{`:has(.one, :host) {}`, `[contenta]:has(.one, [hosta]) {}`},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", "hosta"), tt.want)
		}
	})
}

func TestShadowCssHostInclusionsInsidePseudoSelectors(t *testing.T) {
	tests := []struct {
		css  string
		want string
	}{
		{`.header:not(.admin) {}`, `.header[contenta]:not(.admin) {}`},
		{`.header:is(:host > .toolbar, :host ~ .panel) {}`, `.header[contenta]:is([hosta] > .toolbar, [hosta] ~ .panel) {}`},
		{`.header:where(:host > .toolbar, :host ~ .panel) {}`, `.header[contenta]:where([hosta] > .toolbar, [hosta] ~ .panel) {}`},
		{`.header:not(.admin, :host.super .header) {}`, `.header[contenta]:not(.admin, .super[hosta] .header) {}`},
		{`.header:not(.admin, :host.super .header, :host.mega .header) {}`, `.header[contenta]:not(.admin, .super[hosta] .header, .mega[hosta] .header) {}`},
		{`.one :where(.two, :host) {}`, `.one :where(.two[contenta], [hosta]) {}`},
		{`.one :where(:host, .two) {}`, `.one :where([hosta], .two[contenta]) {}`},
		{`:is(.foo):is(:host):is(.two) {}`, `:is(.foo):is([hosta]):is(.two[contenta]) {}`},
		{`:where(.one, :host .two):first-letter {}`, `[contenta]:where(.one, [hosta] .two):first-letter {}`},
		{`:first-child:where(.one, :host .two) {}`, `[contenta]:first-child:where(.one, [hosta] .two) {}`},
		{`:where(.one, :host .two):nth-child(3):is(.foo, a:where(.bar)) {}`, `[contenta]:where(.one, [hosta] .two):nth-child(3):is(.foo, a:where(.bar)) {}`},
	}
	for _, tt := range tests {
		assertEqualCss(t, shimCss(tt.css, "contenta", "hosta"), tt.want)
	}
}

func TestShadowCssEscapedSelectorWithSpace(t *testing.T) {
	tests := []struct {
		css  string
		want string
	}{
		{`.\\fc ber {}`, `.\\fc ber[contenta] {}`},
		{`.\\fc ker {}`, `.\\fc[contenta]   ker[contenta] {}`},
		{`.pr\\fc fung {}`, `.pr\\fc fung[contenta] {}`},
	}
	for _, tt := range tests {
		if got := shimCss(tt.css, "contenta", ""); got != tt.want {
			t.Fatalf("shimCss(%q) = %q, want %q", tt.css, got, tt.want)
		}
	}
}

func TestShadowCssNgDeep(t *testing.T) {
	t.Run("should handle /deep/", func(t *testing.T) {
		assertEqualCss(t, shimCss("x /deep/ y {}", "contenta", ""), "x[contenta] y {}")
	})

	t.Run("should handle >>>", func(t *testing.T) {
		assertEqualCss(t, shimCss("x >>> y {}", "contenta", ""), "x[contenta] y {}")
	})

	t.Run("should handle ::ng-deep", func(t *testing.T) {
		tests := []struct {
			css      string
			hostAttr string
			want     string
		}{
			{"::ng-deep y {}", "", "y {}"},
			{"x ::ng-deep y {}", "", "x[contenta] y {}"},
			{":host > ::ng-deep .x {}", "h", "[h] > .x {}"},
			{":host ::ng-deep > .x {}", "h", "[h] > .x {}"},
			{":host > ::ng-deep > .x {}", "h", "[h] > > .x {}"},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", tt.hostAttr), tt.want)
		}
	})
}

func TestShadowCssRepeatGroups(t *testing.T) {
	t.Run("should do nothing if multiples is 0", func(t *testing.T) {
		groups := [][]string{{"a1", "b1", "c1"}, {"a2", "b2", "c2"}}
		got := repeatGroups(groups, 0)
		assertStringGroups(t, got, [][]string{{"a1", "b1", "c1"}, {"a2", "b2", "c2"}})
	})

	t.Run("should do nothing if multiples is 1", func(t *testing.T) {
		groups := [][]string{{"a1", "b1", "c1"}, {"a2", "b2", "c2"}}
		got := repeatGroups(groups, 1)
		assertStringGroups(t, got, [][]string{{"a1", "b1", "c1"}, {"a2", "b2", "c2"}})
	})

	t.Run("should add clones of the original groups if multiples is greater than 1", func(t *testing.T) {
		group1 := []string{"a1", "b1", "c1"}
		group2 := []string{"a2", "b2", "c2"}
		groups := [][]string{group1, group2}
		got := repeatGroups(groups, 3)
		assertStringGroups(t, got, [][]string{group1, group2, group1, group2, group1, group2})
		if &got[2][0] == &group1[0] || &got[3][0] == &group2[0] || &got[4][0] == &group1[0] || &got[5][0] == &group2[0] {
			t.Fatal("repeatGroups should clone repeated groups")
		}
	})
}

func TestShadowCssProcessRules(t *testing.T) {
	t.Run("parse rules", func(t *testing.T) {
		assertRules(t, "", []CssRule{})
		assertRules(t, "a;", []CssRule{{Selector: "a", Content: ""}})
		assertRules(t, "a {b}", []CssRule{{Selector: "a", Content: "b"}})
		assertRules(t, "a {b {c}} d {e}", []CssRule{
			{Selector: "a", Content: "b {c}"},
			{Selector: "d", Content: "e"},
		})
		assertRules(t, "@import a ; b {c}", []CssRule{
			{Selector: "@import a", Content: ""},
			{Selector: "b", Content: "c"},
		})
	})

	t.Run("modify rules", func(t *testing.T) {
		got := processRules("@import a; b {c {d}} e {f}", func(rule CssRule) CssRule {
			return CssRule{Selector: rule.Selector + "2", Content: rule.Content}
		})
		if got != "@import a2; b2 {c {d}} e2 {f}" {
			t.Fatalf("processRules selector rewrite = %q", got)
		}

		got = processRules("a {b}", func(rule CssRule) CssRule {
			return CssRule{Selector: rule.Selector, Content: rule.Content + "2"}
		})
		if got != "a {b2}" {
			t.Fatalf("processRules content rewrite = %q", got)
		}
	})
}

func TestShadowCssAtRules(t *testing.T) {
	t.Run("media", func(t *testing.T) {
		css := "@media screen and (max-width: 800px) {div {font-size: 50px;}} div {}"
		expected := "@media screen and (max-width:800px) {div[contenta] {font-size:50px;}} div[contenta] {}"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)

		css = "@media screen and (max-width:800px, max-height:100%) {div {font-size:50px;}}"
		expected = "@media screen and (max-width:800px, max-height:100%) {div[contenta] {font-size:50px;}}"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)
	})

	t.Run("page", func(t *testing.T) {
		css := `
        @page {
          margin-right: 4in;

          @top-left {
            content: "Hamlet";
          }

          @top-right {
            content: "Page " counter(page);
          }
        }

        @page main {
          margin-left: 4in;
        }

        @page :left {
          margin-left: 3cm;
          margin-right: 4cm;
        }

        @page :right {
          margin-left: 4cm;
          margin-right: 3cm;
        }
      `
		result := shimCss(css, "contenta", "")
		assertEqualCss(t, result, css)
		if regexp.MustCompile(`contenta`).MatchString(result) {
			t.Fatalf("@page output should not contain content attr: %q", result)
		}

		assertEqualCss(t, shimCss("@page { margin-right: 4in; }", "contenta", "h"), "@page { margin-right:4in;}")
		assertEqualCss(t, shimCss(`@page { ::ng-deep @top-left { content: "Hamlet";}}`, "contenta", "h"), `@page { @top-left { content:"Hamlet";}}`)
		assertEqualCss(t, shimCss(`@page { :host ::ng-deep @top-left { content:"Hamlet";}}`, "contenta", "h"), `@page { @top-left { content:"Hamlet";}}`)
	})

	t.Run("supports", func(t *testing.T) {
		css := "@supports (display: flex) {section {display: flex;}}"
		expected := "@supports (display:flex) {section[contenta] {display:flex;}}"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)

		css = "@supports (display: flex) { @font-face { :host ::ng-deep font-family{} } }"
		expected = "@supports (display:flex) { @font-face { font-family{}}}"
		assertEqualCss(t, shimCss(css, "contenta", "h"), expected)
	})

	t.Run("font-face", func(t *testing.T) {
		assertEqualCss(t, shimCss("@font-face { font-family {} }", "contenta", "h"), "@font-face { font-family {}}")
		assertEqualCss(t, shimCss("@font-face { ::ng-deep font-family{} }", "contenta", "h"), "@font-face { font-family{}}")
		assertEqualCss(t, shimCss("@font-face { :host ::ng-deep font-family{} }", "contenta", "h"), "@font-face { font-family{}}")
	})

	t.Run("import", func(t *testing.T) {
		styleStr := `@import url("https://fonts.googleapis.com/css?family=Roboto");`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), styleStr)

		styleStr = `@import url("a"); div {}`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), `@import url("a"); div[contenta] {}`)

		styleStr = `@import url("a"); div {background-image: url("a.jpg"); color: red;}`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), `@import url("a"); div[contenta] {background-image:url("a.jpg"); color:red;}`)

		styleStr = `@import url("https://fonts.googleapis.com/css2?family=Roboto:wght@400;500&display=swap");`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), styleStr)

		styleStr = `@import url("https://fonts.googleapis.com/css2?family=Roboto:wght@400;500&display=swap"); div {}`
		assertEqualCss(t, shimCss(styleStr, "contenta", ""), `@import url("https://fonts.googleapis.com/css2?family=Roboto:wght@400;500&display=swap"); div[contenta] {}`)
	})

	t.Run("container", func(t *testing.T) {
		css := `@container max(max-width: 500px) {
               .item {
                 color: red;
               }
             }`
		assertEqualCss(t, shimCss(css, "host-a", ""), `
        @container max(max-width: 500px) {
           .item[host-a] {
             color: red;
           }
         }`)

		css = `
          @container container max(max-width: 500px) {
               .item {
                 color: red;
               }
          }`
		assertEqualCss(t, shimCss(css, "host-a", ""), `
        @container container max(max-width: 500px) {
          .item[host-a] {
            color: red;
          }
        }`)
	})

	t.Run("scope", func(t *testing.T) {
		css := `
          @scope (.media-object) to (.content > *) {
              img { border-radius: 50%; }
              .content { padding: 1em; }
          }`
		assertEqualCss(t, shimCss(css, "host-a", ""), `
        @scope (.media-object) to (.content > *) {
          img[host-a] { border-radius: 50%; }
          .content[host-a] { padding: 1em; }
        }`)

		css = `
          @scope (.light-scheme) {
              a { color: darkmagenta; }
          }`
		assertEqualCss(t, shimCss(css, "host-a", ""), `
        @scope (.light-scheme) {
          a[host-a] { color: darkmagenta; }
        }`)
	})

	t.Run("document", func(t *testing.T) {
		css := "@document url(http://www.w3.org/) {div {font-size:50px;}}"
		expected := "@document url(http://www.w3.org/) {div[contenta] {font-size:50px;}}"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)
	})

	t.Run("layer", func(t *testing.T) {
		css := "@layer utilities {section {display: flex;}}"
		expected := "@layer utilities {section[contenta] {display:flex;}}"
		assertEqualCss(t, shimCss(css, "contenta", ""), expected)
	})

	t.Run("starting-style", func(t *testing.T) {
		css := `
          @starting-style {
              img { border-radius: 50%; }
              .content { padding: 1em; }
          }`
		assertEqualCss(t, shimCss(css, "host-a", ""), `
        @starting-style {
          img[host-a] { border-radius: 50%; }
          .content[host-a] { padding: 1em; }
        }`)
	})
}

func TestShadowCssHost(t *testing.T) {
	t.Run("should handle no context", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host {}", "contenta", "a-host"), "[a-host] {}")
	})

	t.Run("should handle tag selector", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host(ul) {}", "contenta", "a-host"), "ul[a-host] {}")
	})

	t.Run("should handle class selector", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host(.x) {}", "contenta", "a-host"), ".x[a-host] {}")
	})

	t.Run("should handle attribute selector", func(t *testing.T) {
		assertEqualCss(t, shimCss(`:host([a="b"]) {}`, "contenta", "a-host"), `[a="b"][a-host] {}`)
		assertEqualCss(t, shimCss(":host([a=b]) {}", "contenta", "a-host"), "[a=b][a-host] {}")
	})

	t.Run("should handle attribute and next operator without spaces", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host[foo]>div {}", "contenta", "a-host"), "[foo][a-host] > div[contenta] {}")
	})

	t.Run("should handle compound class selectors", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host(.a.b) {}", "contenta", "a-host"), ".a.b[a-host] {}")
	})

	t.Run("should ignore :host with a selector list containing top-level commas", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host(.a, .b) {}", "contenta", "a-host"), "[contenta]:host(.a, .b) {}")
		assertEqualCss(t, shimCss(".outer :host(.a, .b) .inner {}", "contenta", "a-host"), ".outer[contenta] [contenta]:host(.a, .b) .inner[contenta] {}")
	})

	t.Run("should handle pseudo selectors", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{":host(:before) {}", "[a-host]:before {}"},
			{":host:before {}", "[a-host]:before {}"},
			{":host:nth-child(8n+1) {}", "[a-host]:nth-child(8n+1) {}"},
			{":host(:nth-child(3n of :not(p, a))) {}", "[a-host]:nth-child(3n of :not(p, a)) {}"},
			{":host:nth-of-type(8n+1) {}", "[a-host]:nth-of-type(8n+1) {}"},
			{":host(.class):before {}", ".class[a-host]:before {}"},
			{":host.class:before {}", ".class[a-host]:before {}"},
			{":host(:not(p)):before {}", "[a-host]:not(p):before {}"},
			{":host(:not(:has(p))) {}", "[a-host]:not(:has(p)) {}"},
			{":host:not(:host.foo) {}", "[a-host]:not([a-host].foo) {}"},
			{":host:not(.foo:host) {}", "[a-host]:not(.foo[a-host]) {}"},
			{":host:not(:host.foo, :host.bar) {}", "[a-host]:not([a-host].foo, .bar[a-host]) {}"},
			{":host:not(:host.foo, .bar :host) {}", "[a-host]:not([a-host].foo, .bar [a-host]) {}"},
			{":host:not(.foo, .bar) {}", "[a-host]:not(.foo, .bar) {}"},
			{":host:not(:has(p, a)) {}", "[a-host]:not(:has(p, a)) {}"},
			{":host(:not(.foo, .bar)) {}", "[a-host]:not(.foo, .bar) {}"},
			{":host:has(> child-element:not(.foo)) {}", "[a-host]:has(> child-element:not(.foo)) {}"},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", "a-host"), tt.want)
		}

		// xit("should handle host with escaped class selector")
		// We advise to a more simple class name that doesn't require escaping.
		// assertEqualCss(t, shimCss(":host.pr\\fc fung {}", "contenta", "a-host"), ".pr\\fc fung[a-host] {}")
	})

	t.Run("should handle unexpected selectors in the most reasonable way", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{"cmp:host {}", "cmp[a-host] {}"},
			{"cmp:host >>> {}", "cmp[a-host] {}"},
			{"cmp:host child {}", "cmp[a-host] child[contenta] {}"},
			{"cmp:host >>> child {}", "cmp[a-host] child {}"},
			{"cmp :host {}", "cmp [a-host] {}"},
			{"cmp :host >>> {}", "cmp [a-host] {}"},
			{"cmp :host child {}", "cmp [a-host] child[contenta] {}"},
			{"cmp :host >>> child {}", "cmp [a-host] child {}"},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", "a-host"), tt.want)
		}
	})

	t.Run("should support newlines in the same selector and content", func(t *testing.T) {
		selector := `.foo:not(
        :host) {
          background-color:
            green;
      }`
		assertEqualCss(t, shimCss(selector, "contenta", "a-host"), ".foo[contenta]:not( [a-host]) { background-color:green;}")
	})
}

func TestShadowCssHostContext(t *testing.T) {
	t.Run("should transform :host-context with pseudo selectors", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{
				":host-context(backdrop:not(.borderless)) .backdrop {}",
				"backdrop:not(.borderless)[hosta] .backdrop[contenta], backdrop:not(.borderless) [hosta] .backdrop[contenta] {}",
			},
			{
				":where(:host-context(backdrop)) {}",
				":where(backdrop[hosta]), :where(backdrop [hosta]) {}",
			},
			{
				":where(:host-context(outer1)) :host(bar) {}",
				":where(outer1) bar[hosta] {}",
			},
			{
				":where(:host-context(backdrop)) .foo ~ .bar {}",
				":where(backdrop[hosta]) .foo[contenta] ~ .bar[contenta], :where(backdrop [hosta]) .foo[contenta] ~ .bar[contenta] {}",
			},
			{
				":where(:host-context(backdrop)) :host {}",
				":where(backdrop) [hosta] {}",
			},
			{
				"div:where(:host-context(backdrop)) :host {}",
				"div:where(backdrop) [hosta] {}",
			},
			{
				":where(:host-context(.one)) :where(:host-context(.two)) {}",
				":where(.one.two[hosta]), :where(.one.two [hosta]), :where(.one .two[hosta]), :where(.one .two [hosta]), :where(.two .one[hosta]), :where(.two .one [hosta]) {}",
			},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", "hosta"), tt.want)
		}
	})

	t.Run("should transform :host-context with nested pseudo selectors", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{
				":host-context(:where(.foo:not(.bar))) {}",
				":where(.foo:not(.bar))[hosta], :where(.foo:not(.bar)) [hosta] {}",
			},
			{
				":host-context(:is(.foo:not(.bar))) {}",
				":is(.foo:not(.bar))[hosta], :is(.foo:not(.bar)) [hosta] {}",
			},
			{
				":host-context(:where(.foo:not(.bar, .baz))) .inner {}",
				":where(.foo:not(.bar, .baz))[hosta] .inner[contenta], :where(.foo:not(.bar, .baz)) [hosta] .inner[contenta] {}",
			},
		}
		for _, tt := range tests {
			assertEqualCss(t, shimCss(tt.css, "contenta", "hosta"), tt.want)
		}
	})

	t.Run("should handle tag selector", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context(div) {}", "contenta", "a-host"), "div[a-host], div [a-host] {}")
		assertEqualCss(t, shimCss(":host-context(ul) > .y {}", "contenta", "a-host"), "ul[a-host] > .y[contenta], ul [a-host] > .y[contenta] {}")
	})

	t.Run("should handle class selector", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context(.x) {}", "contenta", "a-host"), ".x[a-host], .x [a-host] {}")
		assertEqualCss(t, shimCss(":host-context(.x) > .y {}", "contenta", "a-host"), ".x[a-host] > .y[contenta], .x [a-host] > .y[contenta] {}")
	})

	t.Run("should handle attribute selector", func(t *testing.T) {
		assertEqualCss(t, shimCss(`:host-context([a="b"]) {}`, "contenta", "a-host"), `[a="b"][a-host], [a="b"] [a-host] {}`)
		assertEqualCss(t, shimCss(":host-context([a=b]) {}", "contenta", "a-host"), "[a=b][a-host], [a=b] [a-host] {}")
	})

	t.Run("should handle multiple :host-context() selectors", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context(.one):host-context(.two) {}", "contenta", "a-host"), ".one.two[a-host], .one.two [a-host], .one .two[a-host], .one .two [a-host], .two .one[a-host], .two .one [a-host] {}")
		assertEqualCss(t, shimCss(":host-context(.X):host-context(.Y):host-context(.Z) {}", "contenta", "a-host"), ".X.Y.Z[a-host], .X.Y.Z [a-host], .X.Y .Z[a-host], .X.Y .Z [a-host], .X.Z .Y[a-host], .X.Z .Y [a-host], .X .Y.Z[a-host], .X .Y.Z [a-host], .X .Y .Z[a-host], .X .Y .Z [a-host], .X .Z .Y[a-host], .X .Z .Y [a-host], .Y.Z .X[a-host], .Y.Z .X [a-host], .Y .Z .X[a-host], .Y .Z .X [a-host], .Z .Y .X[a-host], .Z .Y .X [a-host] {}")
	})

	t.Run("should handle :host-context with no ancestor selectors", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context .inner {}", "contenta", "a-host"), "[contenta]:host-context .inner[contenta] {}")
		assertEqualCss(t, shimCss(":host-context() .inner {}", "contenta", "a-host"), "[contenta]:host-context() .inner[contenta] {}")
		assertEqualCss(t, shimCss(":host-context :host-context(.a) {}", "contenta", "host-a"), ":host-context .a[host-a], .a [host-a] {}")
	})

	t.Run("should handle selectors", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context(.one,.two) .inner {}", "contenta", "a-host"), ".one[a-host] .inner[contenta], .one [a-host] .inner[contenta], .two[a-host] .inner[contenta], .two [a-host] .inner[contenta] {}")
	})

	t.Run("should handle :host-context with comma-separated child selector", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context(.foo) a:not(.a, .b) {}", "contenta", "a-host"), ".foo[a-host] a[contenta]:not(.a, .b), .foo [a-host] a[contenta]:not(.a, .b) {}")
		assertEqualCss(t, shimCss(":host-context(.foo) a:not([a], .b), .bar, :host-context(.baz) a:not([c], .d) {}", "contenta", "a-host"), ".foo[a-host] a[contenta]:not([a], .b), .foo [a-host] a[contenta]:not([a], .b), .bar[contenta], .baz[a-host] a[contenta]:not([c], .d), .baz [a-host] a[contenta]:not([c], .d) {}")
	})
}

func TestShadowCssHostContextAndHostCombination(t *testing.T) {
	t.Run("should handle selectors on the same element", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context(div):host(.x) > .y {}", "contenta", "a-host"), "div.x[a-host] > .y[contenta] {}")
	})

	t.Run("should handle no selector :host", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host:host-context(.one) {}", "contenta", "a-host"), ".one[a-host][a-host], .one [a-host] {}")
		assertEqualCss(t, shimCss(":host-context(.one) :host {}", "contenta", "a-host"), ".one [a-host] {}")
	})

	t.Run("should handle selectors on different elements", func(t *testing.T) {
		assertEqualCss(t, shimCss(":host-context(div) :host(.x) > .y {}", "contenta", "a-host"), "div .x[a-host] > .y[contenta] {}")
		assertEqualCss(t, shimCss(":host-context(div) > :host(.x) > .y {}", "contenta", "a-host"), "div > .x[a-host] > .y[contenta] {}")
	})

	t.Run("should parse multiple rules containing :host-context and :host", func(t *testing.T) {
		input := `
            :host-context(outer1) :host(bar) {}
            :host-context(outer2) :host(foo) {}
        `
		assertEqualCss(t, shimCss(input, "contenta", "a-host"), "outer1 bar[a-host] {} outer2 foo[a-host] {}")
	})
}

func TestShadowCssKeyframesAndAnimations(t *testing.T) {
	t.Run("should scope keyframes rules", func(t *testing.T) {
		css := "@keyframes foo {0% {transform:translate(-50%) scaleX(0);}}"
		expected := "@keyframes host-a_foo {0% {transform:translate(-50%) scaleX(0);}}"
		if got := shimCss(css, "host-a", ""); got != expected {
			t.Fatalf("shimCss keyframes = %q, want %q", got, expected)
		}
	})

	t.Run("should scope -webkit-keyframes rules", func(t *testing.T) {
		css := "@-webkit-keyframes foo {0% {-webkit-transform:translate(-50%) scaleX(0);}} "
		expected := "@-webkit-keyframes host-a_foo {0% {-webkit-transform:translate(-50%) scaleX(0);}}"
		if got := shimCss(css, "host-a", ""); got != expected {
			t.Fatalf("shimCss webkit keyframes = %q, want %q", got, expected)
		}
	})

	t.Run("should scope animations using local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation: foo 10s ease;
        }
        @keyframes foo {
            0% {
            transform: translate(-50%) scaleX(0);
            }
        }
        `
		assertContains(t, shimCss(css, "host-a", ""), "animation: host-a_foo 10s ease;")
	})

	t.Run("should not scope animations using non-local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation: foo 10s ease;
        }
        `
		assertContains(t, shimCss(css, "host-a", ""), "animation: foo 10s ease;")
	})

	t.Run("should scope animation-names using local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation-name: foo;
        }
        @keyframes foo {
            0% {
            transform: translate(-50%) scaleX(0);
            }
        }
        `
		assertContains(t, shimCss(css, "host-a", ""), "animation-name: host-a_foo;")
	})

	t.Run("should not scope animation-names using non-local keyframes identifiers", func(t *testing.T) {
		css := `
        button {
            animation-name: foo;
        }
        `
		assertContains(t, shimCss(css, "host-a", ""), "animation-name: foo;")
	})

	t.Run("should handle multiple animation-names", func(t *testing.T) {
		css := `
        button {
            animation-name: foo, bar,baz, qux , quux ,corge ,grault ,garply, waldo;
        }
        @keyframes foo {}
        @keyframes baz {}
        @keyframes quux {}
        @keyframes grault {}
        @keyframes waldo {}`
		result := shimCss(css, "host-a", "")
		expected := "animation-name: host-a_foo, bar,host-a_baz, qux , host-a_quux ,corge ,host-a_grault ,garply, host-a_waldo;"
		assertContains(t, result, expected)
	})

	t.Run("should handle multiple animation-names defined over multiple lines", func(t *testing.T) {
		css := `
        button {
            animation-name: foo,
                            bar,baz,
                            qux ,
                            quux ,
                            grault,
                            garply, waldo;
        }
        @keyframes foo {}
        @keyframes baz {}
        @keyframes quux {}
        @keyframes grault {}`
		result := shimCss(css, "host-a", "")
		for _, scoped := range []string{"foo", "baz", "quux", "grault"} {
			assertContains(t, result, "host-a_"+scoped)
		}
		for _, nonScoped := range []string{"bar", "qux", "garply", "waldo"} {
			assertContains(t, result, nonScoped)
			assertNotContains(t, result, "host-a_"+nonScoped)
		}
	})

	t.Run("should handle animation definitions with names without preceding space", func(t *testing.T) {
		contentAttr := "_ngcontent-%COMP%"
		css := `.test {
      animation:my-anim 1s,my-anim2 2s, my-anim3 3s,my-anim4 4s;
    }

    @keyframes my-anim {
      0% {color: red}
      100% {color: blue}
    }

    @keyframes my-anim2 {
      0% {font-size: 1em}
      100% {font-size: 1.2em}
    }
    `
		result := shimCss(css, contentAttr, "_nghost-%COMP%")
		animationLine := regexp.MustCompile(`animation:[^;]+;`).FindString(result)
		for _, scoped := range []string{"my-anim", "my-anim2"} {
			assertContains(t, animationLine, contentAttr+"_"+scoped)
		}
		for _, nonScoped := range []string{"my-anim3", "my-anim4"} {
			assertContains(t, animationLine, nonScoped)
			assertNotContains(t, animationLine, contentAttr+"_"+nonScoped)
		}
	})

	t.Run("should maintain spacing for keyframes and animations", func(t *testing.T) {
		css := `
        div {
            animation-name : foo;
            animation:  5s bar   1s backwards;
            animation : 3s baz ;
            animation-name:foobar ;
            animation:1s "foo" ,   2s "bar",3s "quux";
        }

        @-webkit-keyframes  bar {}
        @keyframes foobar  {}
        @keyframes quux {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "animation-name : foo;")
		assertContains(t, result, "animation:  5s host-a_bar   1s backwards;")
		assertContains(t, result, "animation : 3s baz ;")
		assertContains(t, result, "animation-name:host-a_foobar ;")
		assertContains(t, result, "@-webkit-keyframes  host-a_bar {}")
		assertContains(t, result, "@keyframes host-a_foobar  {}")
		assertContains(t, result, `animation:1s "foo" ,   2s "host-a_bar",3s "host-a_quux"`)
	})

	t.Run("should correctly process animations defined without any prefixed space", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{".test{display: flex;animation:foo 1s forwards;} @keyframes foo {}", ".test[host-a]{display: flex;animation:host-a_foo 1s forwards;} @keyframes host-a_foo {}"},
			{".test{animation:foo 2s forwards;} @keyframes foo {}", ".test[host-a]{animation:host-a_foo 2s forwards;} @keyframes host-a_foo {}"},
			{"button {display: block;animation-name: foobar;} @keyframes foobar {}", "button[host-a] {display: block;animation-name: host-a_foobar;} @keyframes host-a_foobar {}"},
		}
		for _, tt := range tests {
			if got := shimCss(tt.css, "host-a", ""); got != tt.want {
				t.Fatalf("shimCss(%q) = %q, want %q", tt.css, got, tt.want)
			}
		}
	})

	t.Run("should correctly process keyframes defined without any prefixed space", func(t *testing.T) {
		tests := []struct {
			css  string
			want string
		}{
			{".test{display: flex;animation:bar 1s forwards;}@keyframes bar {}", ".test[host-a]{display: flex;animation:host-a_bar 1s forwards;}@keyframes host-a_bar {}"},
			{".test{animation:bar 2s forwards;}@-webkit-keyframes bar {}", ".test[host-a]{animation:host-a_bar 2s forwards;}@-webkit-keyframes host-a_bar {}"},
		}
		for _, tt := range tests {
			if got := shimCss(tt.css, "host-a", ""); got != tt.want {
				t.Fatalf("shimCss(%q) = %q, want %q", tt.css, got, tt.want)
			}
		}
	})

	t.Run("should not modify css variables ending with animation names", func(t *testing.T) {
		css := `
        button {
            --variable-animation: foo;
            --variable-animation-name: foo;
        }
        @keyframes foo {}`
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "--variable-animation: foo;")
		assertContains(t, result, "--variable-animation-name: foo;")
	})

	t.Run("should handle multiple animation definitions in a single declaration", func(t *testing.T) {
		css := `
        div {
            animation: 1s ease foo, 2s bar infinite, forwards baz 3s;
        }

        p {
            animation: 1s "foo", 2s "bar";
        }

        span {
            animation: .5s ease 'quux',
                        1s foo infinite, forwards "baz'" 1.5s,
                        2s bar;
        }

        button {
            animation: .5s bar,
                        1s foo 0.3s, 2s quux;
        }

        @keyframes bar {}
        @keyframes quux {}
        @keyframes "baz'" {}`
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "animation: 1s ease foo, 2s host-a_bar infinite, forwards baz 3s;")
		assertContains(t, result, `animation: 1s "foo", 2s "host-a_bar";`)
		assertContains(t, result, "animation: .5s host-a_bar,\n                        1s foo 0.3s, 2s host-a_quux;")
		assertContains(t, result, "animation: .5s ease 'host-a_quux',\n                        1s foo infinite, forwards \"host-a_baz'\" 1.5s,\n                        2s host-a_bar;")
	})

	t.Run("should ignore keyword values when scoping local animations", func(t *testing.T) {
		css := `
        div {
            animation: inherit;
            animation: unset;
            animation: 3s ease reverse foo;
            animation: 5s foo 1s backwards;
            animation: none 1s foo;
            animation: .5s foo paused;
            animation: 1s running foo;
            animation: 3s linear 1s infinite running foo;
            animation: 5s foo ease;
            animation: 3s .5s infinite steps(3,end) foo;
            animation: 5s steps(9, jump-start) jump .5s;
            animation: 1s step-end steps;
        }

        @keyframes foo {}
        @keyframes inherit {}
        @keyframes unset {}
        @keyframes ease {}
        @keyframes reverse {}
        @keyframes backwards {}
        @keyframes none {}
        @keyframes paused {}
        @keyframes linear {}
        @keyframes running {}
        @keyframes end {}
        @keyframes jump {}
        @keyframes start {}
        @keyframes steps {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "animation: inherit;")
		assertContains(t, result, "animation: unset;")
		assertContains(t, result, "animation: 3s ease reverse host-a_foo;")
		assertContains(t, result, "animation: 5s host-a_foo 1s backwards;")
		assertContains(t, result, "animation: none 1s host-a_foo;")
		assertContains(t, result, "animation: .5s host-a_foo paused;")
		assertContains(t, result, "animation: 1s running host-a_foo;")
		assertContains(t, result, "animation: 3s linear 1s infinite running host-a_foo;")
		assertContains(t, result, "animation: 5s host-a_foo ease;")
		assertContains(t, result, "animation: 3s .5s infinite steps(3,end) host-a_foo;")
		assertContains(t, result, "animation: 5s steps(9, jump-start) host-a_jump .5s;")
		assertContains(t, result, "animation: 1s step-end host-a_steps;")
	})

	t.Run("should handle quoted animation names", func(t *testing.T) {
		css := `
        div {
            animation: 1.5s foo;
        }

        p {
            animation: 1s 'foz bar';
        }

        @keyframes 'foo' {}
        @keyframes "foz bar" {}
        @keyframes bar {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "@keyframes 'host-a_foo' {}")
		assertContains(t, result, `@keyframes "host-a_foz bar" {}`)
		assertContains(t, result, "animation: 1.5s host-a_foo;")
		assertContains(t, result, "animation: 1s 'host-a_foz bar';")
	})

	t.Run("should handle quotes containing escaped quotes", func(t *testing.T) {
		css := `
        div {
            animation: 1.5s "foo\"bar";
        }

        p {
            animation: 1s 'bar\' \'baz';
        }

        button {
            animation-name: 'foz " baz';
        }

        @keyframes "foo\"bar" {}
        @keyframes "bar' 'baz" {}
        @keyframes "foz \" baz" {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, `@keyframes "host-a_foo\"bar" {}`)
		assertContains(t, result, `@keyframes "host-a_bar' 'baz" {}`)
		assertContains(t, result, `@keyframes "host-a_foz \" baz" {}`)
		assertContains(t, result, `animation: 1.5s "host-a_foo\"bar";`)
		assertContains(t, result, `animation: 1s 'host-a_bar\' \'baz';`)
		assertContains(t, result, `animation-name: 'host-a_foz " baz';`)
	})

	t.Run("should handle commas in multiple animation declarations", func(t *testing.T) {
		css := `
         button {
           animation: 1s "foo bar, baz", 2s 'qux quux';
         }

         div {
           animation: 500ms foo, 1s 'bar, baz', 1500ms bar;
         }

         p {
           animation: 3s "bar, baz", 3s 'foo, bar' 1s, 3s "qux quux";
         }

         @keyframes "qux quux" {}
         @keyframes "bar, baz" {}
       `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, `@keyframes "host-a_qux quux" {}`)
		assertContains(t, result, `@keyframes "host-a_bar, baz" {}`)
		assertContains(t, result, `animation: 1s "foo bar, baz", 2s 'host-a_qux quux';`)
		assertContains(t, result, `animation: 500ms foo, 1s 'host-a_bar, baz', 1500ms bar;`)
		assertContains(t, result, `animation: 3s "host-a_bar, baz", 3s 'foo, bar' 1s, 3s "host-a_qux quux";`)
	})

	t.Run("should handle double quote escaping in multiple animation declarations", func(t *testing.T) {
		css := `
        div {
            animation: 1s "foo", 1.5s "bar";
            animation: 2s "fo\"o", 2.5s "bar";
            animation: 3s "foo\"", 3.5s "bar", 3.7s "ba\"r";
            animation: 4s "foo\\", 4.5s "bar", 4.7s "baz\"";
            animation: 5s "fo\\\"o", 5.5s "bar", 5.7s "baz\"";
        }

        @keyframes "foo" {}
        @keyframes "fo\"o" {}
        @keyframes 'foo"' {}
        @keyframes 'foo\\' {}
        @keyframes bar {}
        @keyframes "ba\"r" {}
        @keyframes "fo\\\"o" {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, `@keyframes "host-a_foo" {}`)
		assertContains(t, result, `@keyframes "host-a_fo\"o" {}`)
		assertContains(t, result, `@keyframes 'host-a_foo"' {}`)
		assertContains(t, result, `@keyframes 'host-a_foo\\' {}`)
		assertContains(t, result, `@keyframes host-a_bar {}`)
		assertContains(t, result, `@keyframes "host-a_ba\"r" {}`)
		assertContains(t, result, `@keyframes "host-a_fo\\\"o"`)
		assertContains(t, result, `animation: 1s "host-a_foo", 1.5s "host-a_bar";`)
		assertContains(t, result, `animation: 2s "host-a_fo\"o", 2.5s "host-a_bar";`)
		assertContains(t, result, `animation: 3s "host-a_foo\"", 3.5s "host-a_bar", 3.7s "host-a_ba\"r";`)
		assertContains(t, result, `animation: 4s "host-a_foo\\", 4.5s "host-a_bar", 4.7s "baz\"";`)
		assertContains(t, result, `animation: 5s "host-a_fo\\\"o", 5.5s "host-a_bar", 5.7s "baz\"";`)
	})

	t.Run("should handle single quote escaping in multiple animation declarations", func(t *testing.T) {
		css := `
        div {
            animation: 1s 'foo', 1.5s 'bar';
            animation: 2s 'fo\'o', 2.5s 'bar';
            animation: 3s 'foo\'', 3.5s 'bar', 3.7s 'ba\'r';
            animation: 4s 'foo\\', 4.5s 'bar', 4.7s 'baz\'';
            animation: 5s 'fo\\\'o', 5.5s 'bar', 5.7s 'baz\'';
        }

        @keyframes foo {}
        @keyframes 'fo\'o' {}
        @keyframes 'foo\'' {}
        @keyframes 'foo\\' {}
        @keyframes "bar" {}
        @keyframes 'ba\'r' {}
        @keyframes "fo\\\'o" {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, `@keyframes host-a_foo {}`)
		assertContains(t, result, `@keyframes 'host-a_fo\'o' {}`)
		assertContains(t, result, `@keyframes 'host-a_foo\'' {}`)
		assertContains(t, result, `@keyframes 'host-a_foo\\' {}`)
		assertContains(t, result, `@keyframes "host-a_bar" {}`)
		assertContains(t, result, `@keyframes 'host-a_ba\'r' {}`)
		assertContains(t, result, `@keyframes "host-a_fo\\\'o" {}`)
		assertContains(t, result, `animation: 1s 'host-a_foo', 1.5s 'host-a_bar';`)
		assertContains(t, result, `animation: 2s 'host-a_fo\'o', 2.5s 'host-a_bar';`)
		assertContains(t, result, `animation: 3s 'host-a_foo\'', 3.5s 'host-a_bar', 3.7s 'host-a_ba\'r';`)
		assertContains(t, result, `animation: 4s 'host-a_foo\\', 4.5s 'host-a_bar', 4.7s 'baz\'';`)
		assertContains(t, result, `animation: 5s 'host-a_fo\\\'o', 5.5s 'host-a_bar', 5.7s 'baz\''`)
	})

	t.Run("should handle mixed quote escaping in multiple animation declarations", func(t *testing.T) {
		css := `
        div {
            animation: 1s 'f\"oo', 1.5s "ba\'r";
            animation: 2s "fo\"\"o", 2.5s 'b\\"ar';
            animation: 3s 'foo\\', 3.5s "b\\\"ar", 3.7s 'ba\'\"\'r';
            animation: 4s 'fo\'o', 4.5s 'b\"ar\"', 4.7s "baz\'";
        }

        @keyframes 'f"oo' {}
        @keyframes 'fo""o' {}
        @keyframes 'foo\\' {}
        @keyframes 'fo\'o' {}
        @keyframes 'ba\'r' {}
        @keyframes 'b\\"ar' {}
        @keyframes 'b\\\"ar' {}
        @keyframes 'b"ar"' {}
        @keyframes 'ba\'\"\'r' {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, `@keyframes 'host-a_f"oo' {}`)
		assertContains(t, result, `@keyframes 'host-a_fo""o' {}`)
		assertContains(t, result, `@keyframes 'host-a_foo\\' {}`)
		assertContains(t, result, `@keyframes 'host-a_fo\'o' {}`)
		assertContains(t, result, `@keyframes 'host-a_ba\'r' {}`)
		assertContains(t, result, `@keyframes 'host-a_b\\"ar' {}`)
		assertContains(t, result, `@keyframes 'host-a_b\\\"ar' {}`)
		assertContains(t, result, `@keyframes 'host-a_b"ar"' {}`)
		assertContains(t, result, `@keyframes 'host-a_ba\'\"\'r' {}`)
		assertContains(t, result, `animation: 1s 'host-a_f\"oo', 1.5s "host-a_ba\'r";`)
		assertContains(t, result, `animation: 2s "host-a_fo\"\"o", 2.5s 'host-a_b\\"ar';`)
		assertContains(t, result, `animation: 3s 'host-a_foo\\', 3.5s "host-a_b\\\"ar", 3.7s 'host-a_ba\'\"\'r';`)
		assertContains(t, result, `animation: 4s 'host-a_fo\'o', 4.5s 'host-a_b\"ar\"', 4.7s "baz\'";`)
	})

	t.Run("should handle commas inside quotes", func(t *testing.T) {
		css := `
        div {
            animation: 3s 'bar,, baz';
        }

        p {
            animation-name: "bar,, baz", foo,'ease, linear , inherit', bar;
        }

        @keyframes 'foo' {}
        @keyframes 'bar,, baz' {}
        @keyframes 'ease, linear , inherit' {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "@keyframes 'host-a_bar,, baz' {}")
		assertContains(t, result, "animation: 3s 'host-a_bar,, baz';")
		assertContains(t, result, `animation-name: "host-a_bar,, baz", host-a_foo,'host-a_ease, linear , inherit', bar;`)
	})

	t.Run("should not ignore animation keywords when they are inside quotes", func(t *testing.T) {
		css := `
        div {
            animation: 3s 'unset';
        }

        button {
            animation: 5s "forwards" 1s forwards;
        }

        @keyframes unset {}
        @keyframes forwards {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "@keyframes host-a_unset {}")
		assertContains(t, result, "@keyframes host-a_forwards {}")
		assertContains(t, result, "animation: 3s 'host-a_unset';")
		assertContains(t, result, `animation: 5s "host-a_forwards" 1s forwards;`)
	})

	t.Run("should handle css functions correctly", func(t *testing.T) {
		css := `
        div {
            animation: foo 0.5s alternate infinite cubic-bezier(.17, .67, .83, .67);
        }

        button {
            animation: calc(2s / 2) calc;
        }

        @keyframes foo {}
        @keyframes cubic-bezier {}
        @keyframes calc {}
        `
		result := shimCss(css, "host-a", "")
		assertContains(t, result, "@keyframes host-a_cubic-bezier {}")
		assertContains(t, result, "@keyframes host-a_calc {}")
		assertContains(t, result, "animation: host-a_foo 0.5s alternate infinite cubic-bezier(.17, .67, .83, .67);")
		assertContains(t, result, "animation: calc(2s / 2) host-a_calc;")
	})
}

func shimCss(css string, contentAttr string, hostAttr string) string {
	return NewShadowCss().ShimCssText(css, contentAttr, hostAttr)
}

func assertEqualCss(t *testing.T, actual string, expected string) {
	t.Helper()
	actualCss := extractCssContent(actual)
	expectedCss := extractCssContent(expected)
	if actualCss != expectedCss {
		t.Fatalf("CSS mismatch\ngot:  %q\nwant: %q\nraw got: %q", actualCss, expectedCss, actual)
	}
}

func assertStringGroups(t *testing.T, got [][]string, want [][]string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("groups = %#v, want %#v", got, want)
	}
}

func assertRules(t *testing.T, input string, want []CssRule) {
	t.Helper()
	var got []CssRule
	processRules(input, func(rule CssRule) CssRule {
		got = append(got, rule)
		return rule
	})
	if got == nil {
		got = []CssRule{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("captureRules(%q) = %#v, want %#v", input, got, want)
	}
}

func assertContains(t *testing.T, actual string, expectedSubstring string) {
	t.Helper()
	if !strings.Contains(actual, expectedSubstring) {
		t.Fatalf("expected output to contain %q\nactual: %q", expectedSubstring, actual)
	}
}

func assertNotContains(t *testing.T, actual string, unexpectedSubstring string) {
	t.Helper()
	if strings.Contains(actual, unexpectedSubstring) {
		t.Fatalf("expected output not to contain %q\nactual: %q", unexpectedSubstring, actual)
	}
}

var (
	leadingCssWhitespaceRe  = regexp.MustCompile(`^\n\s+`)
	trailingCssWhitespaceRe = regexp.MustCompile(`\n\s+$`)
	cssWhitespaceRe         = regexp.MustCompile(`\s+`)
	cssColonSpaceRe         = regexp.MustCompile(`:\s`)
	cssSpaceBeforeBlockRe   = regexp.MustCompile(` }`)
)

func extractCssContent(css string) string {
	css = leadingCssWhitespaceRe.ReplaceAllString(css, "")
	css = trailingCssWhitespaceRe.ReplaceAllString(css, "")
	css = cssWhitespaceRe.ReplaceAllString(css, " ")
	css = cssColonSpaceRe.ReplaceAllString(css, ":")
	css = cssSpaceBeforeBlockRe.ReplaceAllString(css, "}")
	return css
}
