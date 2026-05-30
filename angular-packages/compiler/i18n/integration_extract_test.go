package i18n

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
)

const integrationHTML = `
<div>
    <h1 i18n>i18n attribute on tags</h1>

    <div id="i18n-1"><p i18n>nested</p></div>

    <div id="i18n-2"><p i18n="different meaning|">nested</p></div>

    <div id="i18n-3"><p i18n><i>with placeholders</i></p></div>
    <div id="i18n-3b"><p i18n><i class="preserved-on-placeholders">with placeholders</i></p></div>
    <div id="i18n-3c"><div i18n><div>with <div>nested</div> placeholders</div></div></div>

    <div>
        <p id="i18n-4" i18n-title title="on not translatable node" i18n-data-html data-html="<b>bold</b>"></p>
        <p id="i18n-5" i18n i18n-title title="on translatable node"></p>
        <p id="i18n-6" i18n-title title></p>
    </div>

    <!-- no ph below because the ICU node is the only child of the div, i.e. no text nodes -->
    <div i18n id="i18n-7">{count, plural, =0 {zero} =1 {one} =2 {two} other {<b>many</b>}}</div>

    <div i18n id="i18n-8">
        {sex, select, male {m} female {f} other {other}}
    </div>
    <div i18n id="i18n-8b">
        {sexB, select, male {m} female {f}}
    </div>

    <div i18n id="i18n-9">{{ "count = " + count }}</div>
    <div i18n id="i18n-10">sex = {{ sex }}</div>
    <div i18n id="i18n-11">{{ "custom name" //i18n(ph="CUSTOM_NAME") }}</div>
</div>

<!-- i18n -->
    <h1 id="i18n-12" >Markers in html comments</h1>
    <div id="i18n-13" i18n-title title="in a translatable section"></div>
    <div id="i18n-14">{count, plural, =0 {zero} =1 {one} =2 {two} other {<b>many</b>}}</div>
<!-- /i18n -->

<div id="i18n-15"><ng-container i18n>it <b>should</b> work</ng-container></div>

<div id="i18n-16" i18n="@@i18n16">with an explicit ID</div>
<div id="i18n-17" i18n="@@i18n17">{count, plural, =0 {zero} =1 {one} =2 {two} other {<b>many</b>}}</div>

<!-- make sure that ICU messages are not treated as text nodes -->
<div id="i18n-17-5" i18n="desc">{
    response.getItemsList().length,
    plural,
    =0 {Found no results}
    =1 {Found one result}
    other {Found {{response.getItemsList().length}} results}
}</div>

<div i18n id="i18n-18">foo<a i18n-title title="in a translatable section">bar</a></div>

<div id="i18n-19" i18n>{{ 'test' //i18n(ph="map name") }}</div>
`

func serializeTranslationsForTest(html string, serializer Serializer) string {
	catalog := NewMessageBundle(ml_parser.NewHtmlParser(), []string{}, map[string][]string{}, nil)
	catalog.UpdateFromTemplate(html, "file.ts", nil)
	return catalog.Write(serializer, nil)
}

func assertContainsAll(t *testing.T, got string, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(got, fragment) {
			t.Fatalf("missing fragment:\n%s\n\nGOT:\n%s", fragment, got)
		}
	}
}

func TestXliffIntegrationExtract(t *testing.T) {
	fragments := []string{
		`<trans-unit id="3cb04208df1c2f62553ed48e75939cf7107f9dad" datatype="html">
        <source>i18n attribute on tags</source>`,
		`<trans-unit id="34fec9cc62e28e8aa6ffb306fa8569ef0a8087fe" datatype="html">
        <source><x id="START_ITALIC_TEXT" ctype="x-i" equiv-text="&lt;i&gt;"/>with placeholders<x id="CLOSE_ITALIC_TEXT" ctype="x-i" equiv-text="&lt;/i&gt;"/></source>`,
		`<trans-unit id="dc5536bb9e0e07291c185a0d306601a2ecd4813f" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {zero} =1 {one} =2 {two} other {<x id="START_BOLD_TEXT" ctype="x-b" equiv-text="&lt;b&gt;"/>many<x id="CLOSE_BOLD_TEXT" ctype="x-b" equiv-text="&lt;/b&gt;"/>} }</source>`,
		`<trans-unit id="7f6272480ea8e7ffab548da885ab8105ee2caa93" datatype="html">
        <source>
    <x id="START_HEADING_LEVEL1" ctype="x-h1" equiv-text="&lt;h1&gt;"/>Markers in html comments`,
		`<trans-unit id="i18n16" datatype="html">
        <source>with an explicit ID</source>`,
		`<trans-unit id="2e013b311caa0916478941a985887e091d8288b6" datatype="html">
        <source><x id="MAP NAME" equiv-text="{{ &apos;test&apos; //i18n(ph=&quot;map name&quot;) }}"/></source>`,
	}

	t.Run("LF", func(t *testing.T) {
		got := serializeTranslationsForTest(integrationHTML, NewXliff())
		assertContainsAll(t, got, fragments)
		assertContainsAll(t, got, []string{`<trans-unit id="2370d995bdcc1e7496baa32df20654aff65c2d10" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {Found no results} =1 {Found one result} other {Found <x id="INTERPOLATION" equiv-text="{{response.getItemsList().length}}"/> results} }</source>`})
	})

	t.Run("CRLF", func(t *testing.T) {
		got := serializeTranslationsForTest(strings.ReplaceAll(integrationHTML, "\n", "\r\n"), NewXliff())
		assertContainsAll(t, got, fragments)
		assertContainsAll(t, got, []string{`<trans-unit id="73a09babbde7a003ece74b02acfd22057507717b" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {Found no results} =1 {Found one result} other {Found <x id="INTERPOLATION" equiv-text="{{response.getItemsList().length}}"/> results} }</source>`})
	})
}

func TestXliff2IntegrationExtract(t *testing.T) {
	fragments := []string{
		`<unit id="615790887472569365">
      <notes>
        <note category="location">file.ts:3</note>`,
		`<unit id="3780349238193953556">
      <notes>
        <note category="location">file.ts:9</note>
        <note category="location">file.ts:10</note>`,
		`<unit id="4593805537723189714">
      <notes>
        <note category="location">file.ts:20</note>
        <note category="location">file.ts:37</note>`,
		`<unit id="2329001734457059408">
      <notes>
        <note category="location">file.ts:34,38</note>`,
		`<unit id="i18n16">
      <notes>
        <note category="location">file.ts:42</note>`,
		`<unit id="5339604010413301604">
      <notes>
        <note category="location">file.ts:56</note>
      </notes>
      <segment>
        <source><ph id="0" equiv="MAP NAME" disp="{{ &apos;test&apos; //i18n(ph=&quot;map name&quot;) }}"/></source>`,
	}

	t.Run("LF", func(t *testing.T) {
		got := serializeTranslationsForTest(integrationHTML, NewXliff2())
		assertContainsAll(t, got, fragments)
		assertContainsAll(t, got, []string{`<unit id="4085484936881858615">
      <notes>
        <note category="description">desc</note>
        <note category="location">file.ts:46,52</note>`})
	})

	t.Run("CRLF", func(t *testing.T) {
		got := serializeTranslationsForTest(strings.ReplaceAll(integrationHTML, "\n", "\r\n"), NewXliff2())
		assertContainsAll(t, got, fragments)
		assertContainsAll(t, got, []string{`<unit id="4085484936881858615">
      <notes>
        <note category="description">desc</note>
        <note category="location">file.ts:46,52</note>`})
	})
}

func TestXmbIntegrationExtract(t *testing.T) {
	fragments := []string{
		`<msg id="615790887472569365"><source>file.ts:3</source>i18n attribute on tags</msg>`,
		`<msg id="3780349238193953556"><source>file.ts:9</source><source>file.ts:10</source><ph name="START_ITALIC_TEXT"><ex>&lt;i&gt;</ex>&lt;i&gt;</ph>with placeholders<ph name="CLOSE_ITALIC_TEXT"><ex>&lt;/i&gt;</ex>&lt;/i&gt;</ph></msg>`,
		`<msg id="4593805537723189714"><source>file.ts:20</source><source>file.ts:37</source>{VAR_PLURAL, plural, =0 {zero} =1 {one} =2 {two} other {<ph name="START_BOLD_TEXT"><ex>&lt;b&gt;</ex>&lt;b&gt;</ph>many<ph name="CLOSE_BOLD_TEXT"><ex>&lt;/b&gt;</ex>&lt;/b&gt;</ph>} }</msg>`,
		`<msg id="2329001734457059408"><source>file.ts:34,38</source>
    <ph name="START_HEADING_LEVEL1"><ex>&lt;h1&gt;</ex>&lt;h1&gt;</ph>Markers in html comments`,
		`<msg id="i18n16"><source>file.ts:42</source>with an explicit ID</msg>`,
		`<msg id="5339604010413301604"><source>file.ts:56</source><ph name="MAP_NAME"><ex>{{ &apos;test&apos; //i18n(ph=&quot;map name&quot;) }}</ex>{{ &apos;test&apos; //i18n(ph=&quot;map name&quot;) }}</ph></msg>`,
	}

	t.Run("LF", func(t *testing.T) {
		got := serializeTranslationsForTest(integrationHTML, NewXmb())
		assertContainsAll(t, got, fragments)
		assertContainsAll(t, got, []string{`<msg id="4085484936881858615" desc="desc"><source>file.ts:46,52</source>{VAR_PLURAL, plural, =0 {Found no results} =1 {Found one result} other {Found <ph name="INTERPOLATION"><ex>{{response.getItemsList().length}}</ex>{{response.getItemsList().length}}</ph> results} }</msg>`})
	})

	t.Run("CRLF", func(t *testing.T) {
		got := serializeTranslationsForTest(strings.ReplaceAll(integrationHTML, "\n", "\r\n"), NewXmb())
		assertContainsAll(t, got, fragments)
		assertContainsAll(t, got, []string{`<msg id="4085484936881858615" desc="desc"><source>file.ts:46,52</source>{VAR_PLURAL, plural, =0 {Found no results} =1 {Found one result} other {Found <ph name="INTERPOLATION"><ex>{{response.getItemsList().length}}</ex>{{response.getItemsList().length}}</ph> results} }</msg>`})
	})
}
