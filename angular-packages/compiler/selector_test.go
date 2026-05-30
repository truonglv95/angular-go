package compiler

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type selectorInput struct {
	tag     string
	attrs   [][2]string
	classes string
}

func getSelectorFor(input selectorInput) *CssSelector {
	selector := NewCssSelector()
	selector.SetElement(input.tag)
	for _, attr := range input.attrs {
		selector.AddAttribute(attr[0], attr[1])
	}
	for _, className := range strings.Fields(input.classes) {
		selector.AddClassName(className)
	}
	return selector
}

func expectMatched(t *testing.T, got []any, want ...any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matched = %#v, want %#v", got, want)
	}
}

func expectPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil {
			t.Fatalf("expected panic %q", want)
		}
		if fmt.Sprint(got) != want {
			t.Fatalf("panic = %q, want %q", fmt.Sprint(got), want)
		}
	}()
	fn()
}

func TestSelectorMatcher(t *testing.T) {
	var matcher *SelectorMatcher[int]
	var matched []any
	reset := func() { matched = nil }
	collect := func(selector *CssSelector, context int) {
		matched = append(matched, selector, context)
	}
	setup := func() {
		reset()
		matcher = NewSelectorMatcher[int]()
	}

	t.Run("should select by element name case sensitive", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("someTag")
		matcher.AddSelectables(s1, 1)

		if matcher.Match(getSelectorFor(selectorInput{tag: "SOMEOTHERTAG"}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)

		if matcher.Match(getSelectorFor(selectorInput{tag: "SOMETAG"}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)

		if !matcher.Match(getSelectorFor(selectorInput{tag: "someTag"}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)
	})

	t.Run("should select by class name case insensitive", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse(".someClass")
		s2 := CssSelectorParse(".someClass.class2")
		matcher.AddSelectables(s1, 1)
		matcher.AddSelectables(s2, 2)

		if matcher.Match(getSelectorFor(selectorInput{classes: "SOMEOTHERCLASS"}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)

		if !matcher.Match(getSelectorFor(selectorInput{classes: "SOMECLASS"}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)

		reset()
		if !matcher.Match(getSelectorFor(selectorInput{classes: "someClass class2"}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1, s2[0], 2)
	})

	t.Run("should not throw for class name constructor", func(t *testing.T) {
		setup()
		if matcher.Match(getSelectorFor(selectorInput{classes: "constructor"}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)
	})

	t.Run("should select by attr name case sensitive independent of the value", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("[someAttr]")
		s2 := CssSelectorParse("[someAttr][someAttr2]")
		matcher.AddSelectables(s1, 1)
		matcher.AddSelectables(s2, 2)

		cases := []selectorInput{
			{attrs: [][2]string{{"SOMEOTHERATTR", ""}}},
			{attrs: [][2]string{{"SOMEATTR", ""}}},
			{attrs: [][2]string{{"SOMEATTR", "someValue"}}},
		}
		for _, tc := range cases {
			if matcher.Match(getSelectorFor(tc), collect) {
				t.Fatalf("unexpected match for %#v", tc)
			}
			expectMatched(t, matched)
		}

		for _, tc := range []selectorInput{
			{attrs: [][2]string{{"someAttr", ""}, {"someAttr2", ""}}},
			{attrs: [][2]string{{"someAttr", "someValue"}, {"someAttr2", ""}}},
			{attrs: [][2]string{{"someAttr2", ""}, {"someAttr", "someValue"}}},
			{attrs: [][2]string{{"someAttr2", "someValue"}, {"someAttr", ""}}},
		} {
			reset()
			if !matcher.Match(getSelectorFor(tc), collect) {
				t.Fatalf("expected match for %#v", tc)
			}
			expectMatched(t, matched, s1[0], 1, s2[0], 2)
		}
	})

	t.Run("should support dot in attribute names", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("[foo.bar]")
		matcher.AddSelectables(s1, 1)

		if matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"barfoo", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)

		reset()
		if !matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"foo.bar", ""}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)
	})

	t.Run("should support dollar in attribute names", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("[someAttr\\$]")
		matcher.AddSelectables(s1, 1)
		if matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"someAttr", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)

		reset()
		if !matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"someAttr$", ""}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)

		reset()
		s1 = CssSelectorParse("[some\\$attr]")
		matcher.AddSelectables(s1, 1)
		if matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"someattr", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)
		if !matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"some$attr", ""}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)

		reset()
		s1 = CssSelectorParse("[\\$someAttr]")
		matcher.AddSelectables(s1, 1)
		if matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"someAttr", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)
		if !matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"$someAttr", ""}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)

		reset()
		s1 = CssSelectorParse("[some-\\$Attr]")
		s2 := CssSelectorParse("[some-\\$Attr][some-\\$-attr]")
		matcher.AddSelectables(s1, 1)
		matcher.AddSelectables(s2, 2)
		if matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"some\\$Attr", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)
		if !matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"some-$-attr", "someValue"}, {"some-$Attr", ""}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1, s2[0], 2)

		reset()
		if matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"someattr$", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)
		if matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"some-simple-attr", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)
	})

	t.Run("should select by attr name only once if the value is from the DOM", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("[some-decor]")
		matcher.AddSelectables(s1, 1)
		selector := NewCssSelector()
		selector.AddAttribute("some-decor", "")
		matcher.Match(selector, collect)
		expectMatched(t, matched, s1[0], 1)
	})

	t.Run("should select by attr name case sensitive and value case insensitive", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("[someAttr=someValue]")
		matcher.AddSelectables(s1, 1)

		for _, tc := range []selectorInput{
			{attrs: [][2]string{{"SOMEATTR", "SOMEOTHERATTR"}}},
			{attrs: [][2]string{{"SOMEATTR", "SOMEVALUE"}}},
		} {
			if matcher.Match(getSelectorFor(tc), collect) {
				t.Fatalf("unexpected match for %#v", tc)
			}
			expectMatched(t, matched)
		}

		if !matcher.Match(getSelectorFor(selectorInput{attrs: [][2]string{{"someAttr", "SOMEVALUE"}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)
	})

	t.Run("should select by element name class name and attribute name with value", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("someTag.someClass[someAttr=someValue]")
		matcher.AddSelectables(s1, 1)

		for _, tc := range []selectorInput{
			{tag: "someOtherTag", classes: "someOtherClass", attrs: [][2]string{{"someOtherAttr", ""}}},
			{tag: "someTag", classes: "someOtherClass", attrs: [][2]string{{"someOtherAttr", ""}}},
			{tag: "someTag", classes: "someClass", attrs: [][2]string{{"someOtherAttr", ""}}},
			{tag: "someTag", classes: "someClass", attrs: [][2]string{{"someAttr", ""}}},
		} {
			if matcher.Match(getSelectorFor(tc), collect) {
				t.Fatalf("unexpected match for %#v", tc)
			}
			expectMatched(t, matched)
		}

		if !matcher.Match(getSelectorFor(selectorInput{tag: "someTag", classes: "someClass", attrs: [][2]string{{"someAttr", "someValue"}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)
	})

	t.Run("should select by many attributes and independent of the value", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("input[type=text][control]")
		matcher.AddSelectables(s1, 1)
		selector := NewCssSelector()
		selector.SetElement("input")
		selector.AddAttribute("type", "text")
		selector.AddAttribute("control", "one")

		if !matcher.Match(selector, collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)
	})

	t.Run("should select independent of the order in the css selector", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("[someAttr].someClass")
		s2 := CssSelectorParse(".someClass[someAttr]")
		s3 := CssSelectorParse(".class1.class2")
		s4 := CssSelectorParse(".class2.class1")
		matcher.AddSelectables(s1, 1)
		matcher.AddSelectables(s2, 2)
		matcher.AddSelectables(s3, 3)
		matcher.AddSelectables(s4, 4)

		if !matcher.Match(CssSelectorParse("[someAttr].someClass")[0], collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1, s2[0], 2)

		reset()
		if !matcher.Match(CssSelectorParse(".someClass[someAttr]")[0], collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1, s2[0], 2)

		reset()
		if !matcher.Match(CssSelectorParse(".class1.class2")[0], collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s3[0], 3, s4[0], 4)

		reset()
		if !matcher.Match(CssSelectorParse(".class2.class1")[0], collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s4[0], 4, s3[0], 3)
	})

	t.Run("should not select with a matching not selector", func(t *testing.T) {
		setup()
		matcher.AddSelectables(CssSelectorParse("p:not(.someClass)"), 1)
		matcher.AddSelectables(CssSelectorParse("p:not([someAttr])"), 2)
		matcher.AddSelectables(CssSelectorParse(":not(.someClass)"), 3)
		matcher.AddSelectables(CssSelectorParse(":not(p)"), 4)
		matcher.AddSelectables(CssSelectorParse(":not(p[someAttr])"), 5)

		if matcher.Match(getSelectorFor(selectorInput{tag: "p", classes: "someClass", attrs: [][2]string{{"someAttr", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		expectMatched(t, matched)
	})

	t.Run("should select with a non matching not selector", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("p:not(.someClass)")
		s2 := CssSelectorParse("p:not(.someOtherClass[someAttr])")
		s3 := CssSelectorParse(":not(.someClass)")
		s4 := CssSelectorParse(":not(.someOtherClass[someAttr])")
		matcher.AddSelectables(s1, 1)
		matcher.AddSelectables(s2, 2)
		matcher.AddSelectables(s3, 3)
		matcher.AddSelectables(s4, 4)

		if !matcher.Match(getSelectorFor(selectorInput{tag: "p", classes: "someOtherClass", attrs: [][2]string{{"someOtherAttr", ""}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1, s2[0], 2, s3[0], 3, s4[0], 4)
	})

	t.Run("should match star with not selector", func(t *testing.T) {
		setup()
		matcher.AddSelectables(CssSelectorParse(":not([a])"), 1)
		if !matcher.Match(getSelectorFor(selectorInput{tag: "div"}), func(*CssSelector, int) {}) {
			t.Fatal("expected match")
		}
	})

	t.Run("should match with multiple not selectors", func(t *testing.T) {
		setup()
		matcher.AddSelectables(CssSelectorParse("div:not([a]):not([b])"), 1)
		if matcher.Match(getSelectorFor(selectorInput{tag: "div", attrs: [][2]string{{"a", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		if matcher.Match(getSelectorFor(selectorInput{tag: "div", attrs: [][2]string{{"b", ""}}}), collect) {
			t.Fatal("unexpected match")
		}
		if !matcher.Match(getSelectorFor(selectorInput{tag: "div", attrs: [][2]string{{"c", ""}}}), collect) {
			t.Fatal("expected match")
		}
	})

	t.Run("should select with one match in a list", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("input[type=text], textbox")
		matcher.AddSelectables(s1, 1)

		if !matcher.Match(getSelectorFor(selectorInput{tag: "textbox"}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[1], 1)

		reset()
		if !matcher.Match(getSelectorFor(selectorInput{tag: "input", attrs: [][2]string{{"type", "text"}}}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)
	})

	t.Run("should not select twice with two matches in a list", func(t *testing.T) {
		setup()
		s1 := CssSelectorParse("input, .someClass")
		matcher.AddSelectables(s1, 1)

		if !matcher.Match(getSelectorFor(selectorInput{tag: "input", classes: "someclass"}), collect) {
			t.Fatal("expected match")
		}
		expectMatched(t, matched, s1[0], 1)
	})
}

func TestCssSelectorParse(t *testing.T) {
	t.Run("should detect element names", func(t *testing.T) {
		selector := CssSelectorParse("sometag")[0]
		if selector.Element == nil || *selector.Element != "sometag" {
			t.Fatalf("element = %#v, want sometag", selector.Element)
		}
		if got := selector.String(); got != "sometag" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should detect attr names with escaped dollar", func(t *testing.T) {
		for _, tt := range []struct {
			input string
			attrs []string
			text  string
		}{
			{input: "[attrname\\$]", attrs: []string{"attrname$", ""}, text: "[attrname\\$]"},
			{input: "[\\$attrname]", attrs: []string{"$attrname", ""}, text: "[\\$attrname]"},
			{input: "[foo\\$bar]", attrs: []string{"foo$bar", ""}, text: "[foo\\$bar]"},
		} {
			selector := CssSelectorParse(tt.input)[0]
			if !reflect.DeepEqual(selector.Attrs, tt.attrs) {
				t.Fatalf("%s attrs = %#v, want %#v", tt.input, selector.Attrs, tt.attrs)
			}
			if got := selector.String(); got != tt.text {
				t.Fatalf("%s String() = %q, want %q", tt.input, got, tt.text)
			}
		}
	})

	t.Run("should error on attr names with unescaped dollar", func(t *testing.T) {
		want := `Error in attribute selector "%s". Unescaped "$" is not supported. Please escape with "\$".`
		for _, input := range []string{"attrname$", "$attrname", "foo$bar", "foo\\$bar$"} {
			expectPanic(t, fmt.Sprintf(want, input), func() {
				CssSelectorParse("[" + input + "]")
			})
		}
	})

	t.Run("should detect class names", func(t *testing.T) {
		selector := CssSelectorParse(".someClass")[0]
		if !reflect.DeepEqual(selector.ClassNames, []string{"someclass"}) {
			t.Fatalf("classNames = %#v", selector.ClassNames)
		}
		if got := selector.String(); got != ".someclass" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should detect attr names", func(t *testing.T) {
		selector := CssSelectorParse("[attrname]")[0]
		if !reflect.DeepEqual(selector.Attrs, []string{"attrname", ""}) {
			t.Fatalf("attrs = %#v", selector.Attrs)
		}
		if got := selector.String(); got != "[attrname]" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should detect attr values", func(t *testing.T) {
		for _, input := range []string{"[attrname=attrvalue]", `[attrname="attrvalue"]`, "[attrname='attrvalue']"} {
			selector := CssSelectorParse(input)[0]
			if !reflect.DeepEqual(selector.Attrs, []string{"attrname", "attrvalue"}) {
				t.Fatalf("%s attrs = %#v", input, selector.Attrs)
			}
			if got := selector.String(); got != "[attrname=attrvalue]" {
				t.Fatalf("%s String() = %q", input, got)
			}
		}
	})

	t.Run("should detect id syntax and treat as attribute", func(t *testing.T) {
		selector := CssSelectorParse("#some-value")[0]
		if !reflect.DeepEqual(selector.Attrs, []string{"id", "some-value"}) {
			t.Fatalf("attrs = %#v", selector.Attrs)
		}
		if got := selector.String(); got != "[id=some-value]" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should detect multiple parts", func(t *testing.T) {
		selector := CssSelectorParse("sometag[attrname=attrvalue].someclass")[0]
		if selector.Element == nil || *selector.Element != "sometag" {
			t.Fatalf("element = %#v", selector.Element)
		}
		if !reflect.DeepEqual(selector.Attrs, []string{"attrname", "attrvalue"}) {
			t.Fatalf("attrs = %#v", selector.Attrs)
		}
		if !reflect.DeepEqual(selector.ClassNames, []string{"someclass"}) {
			t.Fatalf("classNames = %#v", selector.ClassNames)
		}
		if got := selector.String(); got != "sometag.someclass[attrname=attrvalue]" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should detect multiple attributes", func(t *testing.T) {
		selector := CssSelectorParse("input[type=text][control]")[0]
		if selector.Element == nil || *selector.Element != "input" {
			t.Fatalf("element = %#v", selector.Element)
		}
		if !reflect.DeepEqual(selector.Attrs, []string{"type", "text", "control", ""}) {
			t.Fatalf("attrs = %#v", selector.Attrs)
		}
		if got := selector.String(); got != "input[type=text][control]" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should detect not", func(t *testing.T) {
		selector := CssSelectorParse("sometag:not([attrname=attrvalue].someclass)")[0]
		if selector.Element == nil || *selector.Element != "sometag" {
			t.Fatalf("element = %#v", selector.Element)
		}
		if len(selector.Attrs) != 0 || len(selector.ClassNames) != 0 {
			t.Fatalf("positive attrs/classes = %#v/%#v", selector.Attrs, selector.ClassNames)
		}
		notSelector := selector.NotSelectors[0]
		if notSelector.Element != nil {
			t.Fatalf("not element = %#v, want nil", notSelector.Element)
		}
		if !reflect.DeepEqual(notSelector.Attrs, []string{"attrname", "attrvalue"}) {
			t.Fatalf("not attrs = %#v", notSelector.Attrs)
		}
		if !reflect.DeepEqual(notSelector.ClassNames, []string{"someclass"}) {
			t.Fatalf("not classNames = %#v", notSelector.ClassNames)
		}
		if got := selector.String(); got != "sometag:not(.someclass[attrname=attrvalue])" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should detect not without truthy", func(t *testing.T) {
		selector := CssSelectorParse(":not([attrname=attrvalue].someclass)")[0]
		if selector.Element == nil || *selector.Element != "*" {
			t.Fatalf("element = %#v", selector.Element)
		}
		notSelector := selector.NotSelectors[0]
		if !reflect.DeepEqual(notSelector.Attrs, []string{"attrname", "attrvalue"}) {
			t.Fatalf("not attrs = %#v", notSelector.Attrs)
		}
		if !reflect.DeepEqual(notSelector.ClassNames, []string{"someclass"}) {
			t.Fatalf("not classNames = %#v", notSelector.ClassNames)
		}
		if got := selector.String(); got != "*:not(.someclass[attrname=attrvalue])" {
			t.Fatalf("String() = %q", got)
		}
	})

	t.Run("should throw when nested not", func(t *testing.T) {
		expectPanic(t, "Nesting :not in a selector is not allowed", func() {
			CssSelectorParse("sometag:not(:not([attrname=attrvalue].someclass))")
		})
	})

	t.Run("should throw when multiple selectors in not", func(t *testing.T) {
		expectPanic(t, "Multiple selectors in :not are not supported", func() {
			CssSelectorParse("sometag:not(a,b)")
		})
	})

	t.Run("should detect lists of selectors", func(t *testing.T) {
		selectors := CssSelectorParse(".someclass,[attrname=attrvalue], sometag")
		if len(selectors) != 3 {
			t.Fatalf("len = %d", len(selectors))
		}
		if !reflect.DeepEqual(selectors[0].ClassNames, []string{"someclass"}) {
			t.Fatalf("first classNames = %#v", selectors[0].ClassNames)
		}
		if !reflect.DeepEqual(selectors[1].Attrs, []string{"attrname", "attrvalue"}) {
			t.Fatalf("second attrs = %#v", selectors[1].Attrs)
		}
		if selectors[2].Element == nil || *selectors[2].Element != "sometag" {
			t.Fatalf("third element = %#v", selectors[2].Element)
		}
	})

	t.Run("should detect lists of selectors with not", func(t *testing.T) {
		selectors := CssSelectorParse("input[type=text], :not(textarea), textbox:not(.special)")
		if len(selectors) != 3 {
			t.Fatalf("len = %d", len(selectors))
		}
		if selectors[0].Element == nil || *selectors[0].Element != "input" {
			t.Fatalf("first element = %#v", selectors[0].Element)
		}
		if !reflect.DeepEqual(selectors[0].Attrs, []string{"type", "text"}) {
			t.Fatalf("first attrs = %#v", selectors[0].Attrs)
		}
		if selectors[1].Element == nil || *selectors[1].Element != "*" {
			t.Fatalf("second element = %#v", selectors[1].Element)
		}
		if selectors[1].NotSelectors[0].Element == nil || *selectors[1].NotSelectors[0].Element != "textarea" {
			t.Fatalf("second not element = %#v", selectors[1].NotSelectors[0].Element)
		}
		if selectors[2].Element == nil || *selectors[2].Element != "textbox" {
			t.Fatalf("third element = %#v", selectors[2].Element)
		}
		if !reflect.DeepEqual(selectors[2].NotSelectors[0].ClassNames, []string{"special"}) {
			t.Fatalf("third not classNames = %#v", selectors[2].NotSelectors[0].ClassNames)
		}
	})
}
