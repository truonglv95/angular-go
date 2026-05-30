package i18n_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
)

func TestXliffSerializer(t *testing.T) {
	const html = `
<p i18n-title title="translatable attribute">not translatable</p>
<p i18n>translatable element <b>with placeholders</b> {{ interpolation}}</p>
<!-- i18n -->{ count, plural, =0 {<p>test</p>}}<!-- /i18n -->
<p i18n="m|d">foo</p>
<p i18n="m|d">foo</p>
<p i18n="m|d@@i">foo</p>
<p i18n="@@bar">foo</p>
<p i18n="ph names"><br><img><div></div></p>
<p i18n="@@baz">{ count, plural, =0 { { sex, select, other {<p>deeply nested</p>}} }}</p>
<p i18n>Test: { count, plural, =0 { { sex, select, other {<p>deeply nested</p>}} } =other {a lot}}</p>
<p i18n>multi
lines</p>
<p i18n>translatable element @if (foo) {with} @else if (bar) {blocks}</p>
`

	const writeXliff = `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit id="983775b9a51ce14b036be72d4cfd65d68d64e231" datatype="html">
        <source>translatable attribute</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">2</context>
        </context-group>
      </trans-unit>
      <trans-unit id="ec1d033f2436133c14ab038286c4f5df4697484a" datatype="html">
        <source>translatable element <x id="START_BOLD_TEXT" ctype="x-b" equiv-text="&lt;b&gt;"/>with placeholders<x id="CLOSE_BOLD_TEXT" ctype="x-b" equiv-text="&lt;/b&gt;"/> <x id="INTERPOLATION" equiv-text="{{ interpolation}}"/></source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">3</context>
        </context-group>
      </trans-unit>
      <trans-unit id="e2ccf3d131b15f54aa1fcf1314b1ca77c14bfcc2" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {<x id="START_PARAGRAPH" ctype="x-p" equiv-text="&lt;p&gt;"/>test<x id="CLOSE_PARAGRAPH" ctype="x-p" equiv-text="&lt;/p&gt;"/>} }</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">4</context>
        </context-group>
      </trans-unit>
      <trans-unit id="db3e0a6a5a96481f60aec61d98c3eecddef5ac23" datatype="html">
        <source>foo</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">5</context>
        </context-group>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">6</context>
        </context-group>
        <note priority="1" from="description">d</note>
        <note priority="1" from="meaning">m</note>
      </trans-unit>
      <trans-unit id="i" datatype="html">
        <source>foo</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">7</context>
        </context-group>
        <note priority="1" from="description">d</note>
        <note priority="1" from="meaning">m</note>
      </trans-unit>
      <trans-unit id="bar" datatype="html">
        <source>foo</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">8</context>
        </context-group>
      </trans-unit>
      <trans-unit id="d7fa2d59aaedcaa5309f13028c59af8c85b8c49d" datatype="html">
        <source><x id="LINE_BREAK" ctype="lb" equiv-text="&lt;br/&gt;"/><x id="TAG_IMG" ctype="image" equiv-text="&lt;img/&gt;"/><x id="START_TAG_DIV" ctype="x-div" equiv-text="&lt;div&gt;"/><x id="CLOSE_TAG_DIV" ctype="x-div" equiv-text="&lt;/div&gt;"/></source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">9</context>
        </context-group>
        <note priority="1" from="description">ph names</note>
      </trans-unit>
      <trans-unit id="baz" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {{VAR_SELECT, select, other {<x id="START_PARAGRAPH" ctype="x-p" equiv-text="&lt;p&gt;"/>deeply nested<x id="CLOSE_PARAGRAPH" ctype="x-p" equiv-text="&lt;/p&gt;"/>} } } }</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">10</context>
        </context-group>
      </trans-unit>
      <trans-unit id="52ffa620dcd76247a56d5331f34e73f340a43cdb" datatype="html">
        <source>Test: <x id="ICU" equiv-text="{ count, plural, =0 {...} =other {...}}"/></source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">11</context>
        </context-group>
      </trans-unit>
      <trans-unit id="1503afd0ccc20ff01d5e2266a9157b7b342ba494" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {{VAR_SELECT, select, other {<x id="START_PARAGRAPH" ctype="x-p" equiv-text="&lt;p&gt;"/>deeply nested<x id="CLOSE_PARAGRAPH" ctype="x-p" equiv-text="&lt;/p&gt;"/>} } } =other {a lot} }</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">11</context>
        </context-group>
      </trans-unit>
      <trans-unit id="fcfa109b0e152d4c217dbc02530be0bcb8123ad1" datatype="html">
        <source>multi
lines</source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">12</context>
        </context-group>
      </trans-unit>
      <trans-unit id="3e17847a6823c7777ca57c7338167badca0f4d19" datatype="html">
        <source>translatable element <x id="START_BLOCK_IF" ctype="x-if" equiv-text="@if"/>with<x id="CLOSE_BLOCK_IF" ctype="x-if" equiv-text="}"/> <x id="START_BLOCK_ELSE_IF" ctype="x-else-if" equiv-text="@else if"/>blocks<x id="CLOSE_BLOCK_ELSE_IF" ctype="x-else-if" equiv-text="}"/></source>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">14</context>
        </context-group>
      </trans-unit>
    </body>
  </file>
</xliff>
`

	const loadXliff = `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" target-language="fr" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit id="983775b9a51ce14b036be72d4cfd65d68d64e231" datatype="html">
        <source>translatable attribute</source>
        <target>etubirtta elbatalsnart</target>
      </trans-unit>
      <trans-unit id="ec1d033f2436133c14ab038286c4f5df4697484a" datatype="html">
        <source>translatable element <x id="START_BOLD_TEXT" ctype="b"/>with placeholders<x id="CLOSE_BOLD_TEXT" ctype="b"/> <x id="INTERPOLATION"/></source>
        <target><x id="INTERPOLATION"/> footnemele elbatalsnart <x id="START_BOLD_TEXT" ctype="x-b"/>sredlohecalp htiw<x id="CLOSE_BOLD_TEXT" ctype="x-b"/></target>
      </trans-unit>
      <trans-unit id="e2ccf3d131b15f54aa1fcf1314b1ca77c14bfcc2" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {<x id="START_PARAGRAPH" ctype="x-p"/>test<x id="CLOSE_PARAGRAPH" ctype="x-p"/>} }</source>
        <target>{VAR_PLURAL, plural, =0 {<x id="START_PARAGRAPH" ctype="x-p"/>TEST<x id="CLOSE_PARAGRAPH" ctype="x-p"/>} }</target>
      </trans-unit>
      <trans-unit id="db3e0a6a5a96481f60aec61d98c3eecddef5ac23" datatype="html">
        <source>foo</source>
        <target>oof</target>
      </trans-unit>
      <trans-unit id="i" datatype="html">
        <source>foo</source>
        <target>toto</target>
      </trans-unit>
      <trans-unit id="bar" datatype="html">
        <source>foo</source>
        <target>tata</target>
      </trans-unit>
      <trans-unit id="d7fa2d59aaedcaa5309f13028c59af8c85b8c49d" datatype="html">
        <source><x id="LINE_BREAK" ctype="lb"/><x id="TAG_IMG" ctype="image"/><x id="START_TAG_DIV" ctype="x-div"/><x id="CLOSE_TAG_DIV" ctype="x-div"/></source>
        <target><x id="START_TAG_DIV" ctype="x-div"/><x id="CLOSE_TAG_DIV" ctype="x-div"/><x id="TAG_IMG" ctype="image"/><x id="LINE_BREAK" ctype="lb"/></target>
      </trans-unit>
      <trans-unit id="empty target" datatype="html">
        <source><x id="LINE_BREAK" ctype="lb"/><x id="TAG_IMG" ctype="image"/><x id="START_TAG_DIV" ctype="x-div"/><x id="CLOSE_TAG_DIV" ctype="x-div"/></source>
        <target/>
      </trans-unit>
      <trans-unit id="baz" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {{VAR_SELECT, select, other {<x id="START_PARAGRAPH" ctype="x-p"/>deeply nested<x id="CLOSE_PARAGRAPH" ctype="x-p"/>} } } }</source>
        <target>{VAR_PLURAL, plural, =0 {{VAR_SELECT, select, other {<x id="START_PARAGRAPH" ctype="x-p"/>profondément imbriqué<x id="CLOSE_PARAGRAPH" ctype="x-p"/>} } } }</target>
      </trans-unit>
      <trans-unit id="52ffa620dcd76247a56d5331f34e73f340a43cdb" datatype="html">
        <source>Test: <x id="ICU" equiv-text="{ count, plural, =0 {...} =other {...}}"/></source>
        <target>Test: <x id="ICU" equiv-text="{ count, plural, =0 {...} =other {...}}"/></target>
      </trans-unit>
      <trans-unit id="1503afd0ccc20ff01d5e2266a9157b7b342ba494" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {{VAR_SELECT, select, other {<x id="START_PARAGRAPH" ctype="x-p"/>deeply nested<x id="CLOSE_PARAGRAPH" ctype="x-p"/>} } } =other {a lot} }</source>
        <target>{VAR_PLURAL, plural, =0 {{VAR_SELECT, select, other {<x id="START_PARAGRAPH" ctype="x-p"/>profondément imbriqué<x id="CLOSE_PARAGRAPH" ctype="x-p"/>} } } =other {beaucoup} }</target>
      </trans-unit>
      <trans-unit id="fcfa109b0e152d4c217dbc02530be0bcb8123ad1" datatype="html">
        <source>multi
lines</source>
        <target>multi
lignes</target>
      </trans-unit>
      <trans-unit id="3e17847a6823c7777ca57c7338167badca0f4d19" datatype="html">
        <source>translatable element <x id="START_BLOCK_IF" ctype="x-if" equiv-text="@if"/>with<x id="CLOSE_BLOCK_IF" ctype="x-if" equiv-text="}"/> <x id="START_BLOCK_ELSE_IF" ctype="x-else-if" equiv-text="@else if"/>blocks<x id="CLOSE_BLOCK_ELSE_IF" ctype="x-else-if" equiv-text="}"/></source>
        <target>élément traduisible <x id="START_BLOCK_IF" ctype="x-if" equiv-text="@if"/>avec<x id="CLOSE_BLOCK_IF" ctype="x-if" equiv-text="}"/> <x id="START_BLOCK_ELSE_IF" ctype="x-else-if" equiv-text="@else if"/>des blocs<x id="CLOSE_BLOCK_ELSE_IF" ctype="x-else-if" equiv-text="}"/></target>
      </trans-unit>
      <trans-unit id="mrk-test">
        <source>First sentence.</source>
        <seg-source>
          <invalid-tag>Should not be parsed</invalid-tag>
        </seg-source>
        <target>Translated <mrk mtype="seg" mid="1">first sentence</mrk>.</target>
      </trans-unit>
      <trans-unit id="mrk-test2">
        <source>First sentence. Second sentence.</source>
        <seg-source>
          <invalid-tag>Should not be parsed</invalid-tag>
        </seg-source>
        <target>Translated <mrk mtype="seg" mid="1"><mrk mtype="seg" mid="2">first</mrk> sentence</mrk>.</target>
      </trans-unit>
    </body>
  </file>
</xliff>
`

	serializer := i18n.NewXliff()

	toXliff := func(html string, locale *string) string {
		catalog := i18n.NewMessageBundle(ml_parser.NewHtmlParser(), []string{}, map[string][]string{}, locale)
		catalog.UpdateFromTemplate(html, "file.ts", nil)
		return catalog.Write(serializer, nil)
	}

	loadAsMap := func(xliff string) map[string]string {
		res := serializer.Load(xliff, "url")
		msgMap := map[string]string{}
		for _, id := range res.I18nNodesByMsgId.Keys() {
			nodes, ok := res.I18nNodesByMsgId.Get(id)
			if !ok {
				t.Fatalf("missing id %q", id)
			}
			msgMap[id] = strings.Join(i18n.SerializeNodes(nodes), "")
		}
		return msgMap
	}

	t.Run("should write a valid xliff file", func(t *testing.T) {
		if got := toXliff(html, nil); got != writeXliff {
			t.Fatalf("xliff mismatch\nGOT:\n%s\nWANT:\n%s", got, writeXliff)
		}
	})

	t.Run("should write a valid xliff file with a source language", func(t *testing.T) {
		fr := "fr"
		if got := toXliff(html, &fr); !strings.Contains(got, `file source-language="fr"`) {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("should load XLIFF files", func(t *testing.T) {
		got := loadAsMap(loadXliff)
		want := map[string]string{
			"983775b9a51ce14b036be72d4cfd65d68d64e231": "etubirtta elbatalsnart",
			"ec1d033f2436133c14ab038286c4f5df4697484a": `<ph name="INTERPOLATION"/> footnemele elbatalsnart <ph name="START_BOLD_TEXT"/>sredlohecalp htiw<ph name="CLOSE_BOLD_TEXT"/>`,
			"e2ccf3d131b15f54aa1fcf1314b1ca77c14bfcc2": `{VAR_PLURAL, plural, =0 {[<ph name="START_PARAGRAPH"/>, TEST, <ph name="CLOSE_PARAGRAPH"/>]}}`,
			"db3e0a6a5a96481f60aec61d98c3eecddef5ac23": "oof",
			"i":   "toto",
			"bar": "tata",
			"d7fa2d59aaedcaa5309f13028c59af8c85b8c49d": `<ph name="START_TAG_DIV"/><ph name="CLOSE_TAG_DIV"/><ph name="TAG_IMG"/><ph name="LINE_BREAK"/>`,
			"empty target": ``,
			"baz":          `{VAR_PLURAL, plural, =0 {[{VAR_SELECT, select, other {[<ph name="START_PARAGRAPH"/>, profondément imbriqué, <ph name="CLOSE_PARAGRAPH"/>]}},  ]}}`,
			"52ffa620dcd76247a56d5331f34e73f340a43cdb": `Test: <ph name="ICU"/>`,
			"1503afd0ccc20ff01d5e2266a9157b7b342ba494": `{VAR_PLURAL, plural, =0 {[{VAR_SELECT, select, other {[<ph name="START_PARAGRAPH"/>, profondément imbriqué, <ph name="CLOSE_PARAGRAPH"/>]}},  ]}, =other {[beaucoup]}}`,
			"fcfa109b0e152d4c217dbc02530be0bcb8123ad1": "multi\nlignes",
			"3e17847a6823c7777ca57c7338167badca0f4d19": `élément traduisible <ph name="START_BLOCK_IF"/>avec<ph name="CLOSE_BLOCK_IF"/> <ph name="START_BLOCK_ELSE_IF"/>des blocs<ph name="CLOSE_BLOCK_ELSE_IF"/>`,
			"mrk-test":  `Translated first sentence.`,
			"mrk-test2": `Translated first sentence.`,
		}
		if len(got) != len(want) {
			t.Fatalf("got %v", got)
		}
		for id, expected := range want {
			if got[id] != expected {
				t.Fatalf("%s got %q want %q", id, got[id], expected)
			}
		}
	})

	t.Run("should return the target locale", func(t *testing.T) {
		res := serializer.Load(loadXliff, "url")
		if res.Locale == nil || *res.Locale != "fr" {
			t.Fatalf("locale = %v", res.Locale)
		}
	})

	t.Run("should ignore alt-trans targets", func(t *testing.T) {
		xliff := `
          <xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
            <file source-language="en" target-language="fr" datatype="plaintext" original="ng2.template">
              <body>
                <trans-unit datatype="html" approved="no" id="registration.submit">
                  <source>Continue</source>
                  <target state="translated" xml:lang="de">Weiter</target>
                  <context-group purpose="location">
                    <context context-type="sourcefile">src/app/auth/registration-form/registration-form.component.html</context>
                    <context context-type="linenumber">69</context>
                  </context-group>
                  <?sid 1110954287-0?>
                  <alt-trans origin="autoFuzzy" tool="Swordfish" match-quality="71" ts="63">
                    <source xml:lang="en">Content</source>
                    <target state="translated" xml:lang="de">Content</target>
                  </alt-trans>
              </trans-unit>
              </body>
            </file>
          </xliff>`
		got := loadAsMap(xliff)
		if got["registration.submit"] != "Weiter" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("should throw when a trans-unit has no translation", func(t *testing.T) {
		xliff := `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit id="missingtarget">
        <source/>
      </trans-unit>
    </body>
  </file>
</xliff>`
		defer expectPanicMatch(t, `Message missingtarget misses a translation`)()
		loadAsMap(xliff)
	})

	t.Run("should throw when a trans-unit has no id attribute", func(t *testing.T) {
		xliff := `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit datatype="html">
        <source/>
        <target/>
      </trans-unit>
    </body>
  </file>
</xliff>`
		defer expectPanicMatch(t, `<trans-unit> misses the "id" attribute`)()
		loadAsMap(xliff)
	})

	t.Run("should throw on duplicate trans-unit id", func(t *testing.T) {
		xliff := `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit id="deadbeef">
        <source/>
        <target/>
      </trans-unit>
      <trans-unit id="deadbeef">
        <source/>
        <target/>
      </trans-unit>
    </body>
  </file>
</xliff>`
		defer expectPanicMatch(t, `Duplicated translations for msg deadbeef`)()
		loadAsMap(xliff)
	})

	t.Run("should throw on unknown message tags", func(t *testing.T) {
		xliff := `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit id="deadbeef" datatype="html">
        <source/>
        <target><b>msg should contain only ph tags</b></target>
      </trans-unit>
    </body>
  </file>
</xliff>`
		defer expectPanicRegex(t, regexp.MustCompile(regexp.QuoteMeta(`[ERROR ->]<b>msg should contain only ph tags</b>`)))()
		loadAsMap(xliff)
	})

	t.Run("should throw when a placeholder misses an id attribute", func(t *testing.T) {
		xliff := `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit id="deadbeef" datatype="html">
        <source/>
        <target><x/></target>
      </trans-unit>
    </body>
  </file>
</xliff>`
		defer expectPanicMatch(t, `<x> misses the "id" attribute`)()
		loadAsMap(xliff)
	})
}

func expectPanicMatch(t *testing.T, substr string) func() {
	return func() {
		t.Helper()
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		} else if !strings.Contains(r.(string), substr) {
			t.Fatalf("panic = %v", r)
		}
	}
}

func expectPanicRegex(t *testing.T, re *regexp.Regexp) func() {
	return func() {
		t.Helper()
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		} else if !re.MatchString(r.(string)) {
			t.Fatalf("panic = %v", r)
		}
	}
}
