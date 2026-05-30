package i18n

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/core"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/stretchr/testify/assert"
)

func TestI18NHtmlParser(t *testing.T) {
	t.Run("should parse the translations only once", func(t *testing.T) {
		htmlParser := ml_parser.NewHtmlParser()
		translations := "translations"
		format := "xlf"
		
		i18nHtmlParser := NewI18NHtmlParser(
			htmlParser,
			&translations,
			&format,
			core.MissingTranslationStrategyWarning,
			nil,
		)
		
		if i18nHtmlParser.translationBundle == nil {
			t.Errorf("expected translationBundle to be set")
		}
		
		i18nHtmlParser.Parse("source", "url", nil)
		i18nHtmlParser.Parse("source", "url", nil)
	})

	t.Run("should parse and merge translations based on validateHtml integration cases", func(t *testing.T) {
		htmlParser := ml_parser.NewHtmlParser()
		translations := xliffTranslations
		format := "xlf"

		i18nHtmlParser := NewI18NHtmlParser(
			htmlParser,
			&translations,
			&format,
			core.MissingTranslationStrategyWarning,
			nil,
		)

		result := i18nHtmlParser.Parse(htmlInput, "file.html", &ml_parser.TokenizeOptions{TokenizeExpansionForms: true})
		assert.Empty(t, result.Errors)

		serialized := strings.Join(serializeHtmlNodes(result.RootNodes), "")

		// Assertions mapping directly to integration_common.ts validateHtml cases:
		// 1. h1: attributs i18n sur les balises
		assert.Contains(t, serialized, "<h1>attributs i18n sur les balises</h1>")

		// 2. #i18n-1: <div id="i18n-1"><p>imbriqué</p></div>
		assert.Contains(t, serialized, `<div id="i18n-1"><p>imbriqué</p></div>`)

		// 3. #i18n-2: <div id="i18n-2"><p>imbriqué</p></div>
		assert.Contains(t, serialized, `<div id="i18n-2"><p>imbriqué</p></div>`)

		// 4. #i18n-3: <div id="i18n-3"><p><i>avec des espaces réservés</i></p></div>
		assert.Contains(t, serialized, `<div id="i18n-3"><p><i>avec des espaces réservés</i></p></div>`)

		// 5. #i18n-3b: <div id="i18n-3b"><p><i class="preserved-on-placeholders">avec des espaces réservés</i></p></div>
		assert.Contains(t, serialized, `<div id="i18n-3b"><p><i class="preserved-on-placeholders">avec des espaces réservés</i></p></div>`)

		// 6. #i18n-4: <p id="i18n-4" title="sur des balises non traductibles" data-html="<b>gras</b>"></p>
		assert.Contains(t, serialized, `<p id="i18n-4" title="sur des balises non traductibles" data-html="<b>gras</b>"></p>`)

		// 7. #i18n-5: <p id="i18n-5" title="sur des balises traductibles"></p>
		assert.Contains(t, serialized, `<p id="i18n-5" title="sur des balises traductibles"></p>`)

		// 8. #i18n-6: <p id="i18n-6" title=""></p>
		assert.Contains(t, serialized, `<p id="i18n-6" title=""></p>`)

		// 9. #i18n-7 plural: zero, un, deux, beaucoup
		assert.Contains(t, serialized, `{count, plural, =0 {zero} =1 {un} =2 {deux} other {<b>beaucoup</b>}}`)

		// 10. #i18n-8 select: homme, femme, autre
		assert.Contains(t, serialized, `{sex, select, other {autre} male {homme} female {femme}}`)

		// 11. #i18n-8b select: femme, homme
		assert.Contains(t, serialized, `{sexB, select, male {homme} female {femme}}`)

		// 12. #i18n-9 interpolation: count = 123
		assert.Contains(t, serialized, `<div id="i18n-9">{{ "count = " + count }}</div>`)

		// 13. #i18n-10 interpolation: sexe = {{ sex }}
		assert.Contains(t, serialized, `<div id="i18n-10">sexe = {{ sex }}</div>`)

		// 14. #i18n-11 interpolation: custom name
		assert.Contains(t, serialized, `<div id="i18n-11">{{ "custom name" //i18n(ph="CUSTOM_NAME") }}</div>`)

		// 15. #i18n-12 comment markers: Balises dans les commentaires html
		assert.Contains(t, serialized, `<h1 id="i18n-12">Balises dans les commentaires html</h1>`)

		// 16. #i18n-13: title="dans une section traductible"
		assert.Contains(t, serialized, `<div id="i18n-13" title="dans une section traductible"></div>`)

		// 17. #i18n-15: ca devrait marcher
		assert.Contains(t, serialized, `<div id="i18n-15"><ng-container>ca <b>devrait</b> marcher</ng-container></div>`)

		// 18. #i18n-16 explicit ID: avec un ID explicite
		assert.Contains(t, serialized, `<div id="i18n-16">avec un ID explicite</div>`)

		// 19. #i18n-17 plural: zero, un, deux, beaucoup
		assert.Contains(t, serialized, `<div id="i18n-17">{count, plural, =0 {zero} =1 {un} =2 {deux} other {<b>beaucoup</b>}}</div>`)

		// 20. #i18n-17-5 plural response: Pas de réponse, Une réponse, réponses
		assert.Contains(t, serialized, `response.getItemsList().length, plural, =0 {Pas de réponse} =1 {Une réponse} other {{{response.getItemsList().length}} réponses}`)

		// 21. #i18n-18: foo<a title="dans une section traductible">bar</a>
		assert.Contains(t, serialized, `<div id="i18n-18">FOO<a title="dans une section traductible">BAR</a></div>`)
	})
}

const htmlInput = `
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

const xliffTranslations = `<?xml version="1.0" encoding="UTF-8" ?>
<xliff version="1.2" xmlns="urn:oasis:names:tc:xliff:document:1.2">
  <file source-language="en" datatype="plaintext" original="ng2.template">
    <body>
      <trans-unit id="3cb04208df1c2f62553ed48e75939cf7107f9dad" datatype="html">
        <source>i18n attribute on tags</source>
        <target>attributs i18n sur les balises</target>
      </trans-unit>
      <trans-unit id="52895b1221effb3f3585b689f049d2784d714952" datatype="html">
        <source>nested</source>
        <target>imbriqué</target>
      </trans-unit>
      <trans-unit id="88d5f22050a9df477ee5646153558b3a4862d47e" datatype="html">
        <source>nested</source>
        <target>imbriqué</target>
        <note priority="1" from="meaning">different meaning</note>
      </trans-unit>
      <trans-unit id="34fec9cc62e28e8aa6ffb306fa8569ef0a8087fe" datatype="html">
        <source><x id="START_ITALIC_TEXT" ctype="x-i" equiv-text="&lt;i&gt;"/>with placeholders<x id="CLOSE_ITALIC_TEXT" ctype="x-i" equiv-text="&lt;/i&gt;"/></source>
        <target><x id="START_ITALIC_TEXT" ctype="x-i"/>avec des espaces réservés<x id="CLOSE_ITALIC_TEXT" ctype="x-i"/></target>
      </trans-unit>
      <trans-unit id="651d7249d3a225037eb66f3433d98ad4a86f0a22" datatype="html">
        <source><x id="START_TAG_DIV" ctype="x-div"/>with <x id="START_TAG_DIV" ctype="x-div"/>nested<x id="CLOSE_TAG_DIV" ctype="x-div"/> placeholders<x id="CLOSE_TAG_DIV" ctype="x-div"/></source>
        <target><x id="START_TAG_DIV" ctype="x-div"/>with <x id="START_TAG_DIV" ctype="x-div"/>nested<x id="CLOSE_TAG_DIV" ctype="x-div"/> placeholders<x id="CLOSE_TAG_DIV" ctype="x-div"/></target>
        <context-group purpose="location">
          <context context-type="sourcefile">file.ts</context>
          <context context-type="linenumber">11</context>
        </context-group>
      </trans-unit>
      <trans-unit id="1fe4616cce80a57c7707bac1c97054aa8e244a67" datatype="html">
        <source>on not translatable node</source>
        <target>sur des balises non traductibles</target>
      </trans-unit>
      <trans-unit id="480aaeeea1570bc1dde6b8404e380dee11ed0759" datatype="html">
        <source>&lt;b&gt;bold&lt;/b&gt;</source>
        <target>&lt;b&gt;gras&lt;/b&gt;</target>
      </trans-unit>
      <trans-unit id="67162b5af5f15fd0eb6480c88688dafdf952b93a" datatype="html">
        <source>on translatable node</source>
        <target>sur des balises traductibles</target>
      </trans-unit>
      <trans-unit id="dc5536bb9e0e07291c185a0d306601a2ecd4813f" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {zero} =1 {one} =2 {two} other {&lt;b&gt;many&lt;/b&gt;} }</source>
        <target>{VAR_PLURAL, plural, =0 {zero} =1 {un} =2 {deux} other {&lt;b&gt;beaucoup&lt;/b&gt;} }</target>
      </trans-unit>
      <trans-unit id="49feb201083cbd2c8bfc48a4ae11f105fb984876" datatype="html">
        <source>
        <x id="ICU" equiv-text="{sex, select, male {...} female {...}}"/>
    </source>
        <target><x id="ICU"/></target>
      </trans-unit>
      <trans-unit id="f3be30eb9a18f6e336cc3ca4dd66bbc3a35c5f97" datatype="html">
        <source>{VAR_SELECT, select, other {other} male {m} female {f} }</source>
        <target>{VAR_SELECT, select, other {autre} male {homme} female {femme}}</target>
      </trans-unit>
      <trans-unit id="cc16e9745fa0b95b2ebc2f18b47ed8e64fe5f0f9" datatype="html">
        <source>
        <x id="ICU" equiv-text="{sexB, select, m {...} f {...}}"/>
    </source>
        <target><x id="ICU"/></target>
      </trans-unit>
      <trans-unit id="4573f2edb0329d69afc2ab8c73c71e2f8b08f807" datatype="html">
        <source>{VAR_SELECT, select, male {m} female {f} }</source>
        <target>{VAR_SELECT, select, male {homme} female {femme} }</target>
      </trans-unit>
      <trans-unit id="d9879678f727b244bc7c7e20f22b63d98cb14890" datatype="html">
        <source><x id="INTERPOLATION" equiv-text="{{ &quot;count = &quot; + count }}"/></source>
        <target><x id="INTERPOLATION"/></target>
      </trans-unit>
      <trans-unit id="50dac33dc6fc0578884baac79d875785ed77c928" datatype="html">
        <source>sex = <x id="INTERPOLATION" equiv-text="{{ sex }}"/></source>
        <target>sexe = <x id="INTERPOLATION"/></target>
      </trans-unit>
      <trans-unit id="a46f833b1fe6ca49e8b97c18f4b7ea0b930c9383" datatype="html">
        <source><x id="CUSTOM_NAME" equiv-text="{{ &quot;custom name&quot; //i18n(ph=&quot;CUSTOM_NAME&quot;) }}"/></source>
        <target><x id="CUSTOM_NAME"/></target>
      </trans-unit>
      <trans-unit id="2ec983b4893bcd5b24af33bebe3ecba63868453c" datatype="html">
        <source>in a translatable section</source>
        <target>dans une section traductible</target>
      </trans-unit>
      <trans-unit id="7f6272480ea8e7ffab548da885ab8105ee2caa93" datatype="html">
        <source>
    <x id="START_HEADING_LEVEL1" ctype="x-h1" equiv-text="&lt;h1&gt;"/>Markers in html comments<x id="CLOSE_HEADING_LEVEL1" ctype="x-h1" equiv-text="&lt;/h1&gt;"/>
    <x id="START_TAG_DIV" ctype="x-div" equiv-text="&lt;div&gt;"/><x id="CLOSE_TAG_DIV" ctype="x-div" equiv-text="&lt;/div&gt;"/>
    <x id="START_TAG_DIV_1" ctype="x-div" equiv-text="&lt;div&gt;"/><x id="ICU" equiv-text="{count, plural, =0 {...} =1 {...} =2 {...} other {...}}"/><x id="CLOSE_TAG_DIV" ctype="x-div" equiv-text="&lt;/div&gt;"/>
</source>
        <target>
    <x id="START_HEADING_LEVEL1" ctype="x-h1"/>Balises dans les commentaires html<x id="CLOSE_HEADING_LEVEL1" ctype="x-h1"/>
    <x id="START_TAG_DIV" ctype="x-div"/><x id="CLOSE_TAG_DIV" ctype="x-div"/>
    <x id="START_TAG_DIV_1" ctype="x-div"/><x id="ICU"/><x id="CLOSE_TAG_DIV" ctype="x-div"/>
</target>
      </trans-unit>
      <trans-unit id="93a30c67d4e6c9b37aecfe2ac0f2b5d366d7b520" datatype="html">
        <source>it <x id="START_BOLD_TEXT" ctype="x-b" equiv-text="&lt;b&gt;"/>should<x id="CLOSE_BOLD_TEXT" ctype="x-b" equiv-text="&lt;/b&gt;"/> work</source>
        <target>ca <x id="START_BOLD_TEXT" ctype="x-b"/>devrait<x id="CLOSE_BOLD_TEXT" ctype="x-b"/> marcher</target>
      </trans-unit>
      <trans-unit id="i18n16" datatype="html">
        <source>with an explicit ID</source>
        <target>avec un ID explicite</target>
      </trans-unit>
      <trans-unit id="i18n17" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {zero} =1 {one} =2 {two} other {&lt;b&gt;many&lt;/b&gt;} }</source>
        <target>{VAR_PLURAL, plural, =0 {zero} =1 {un} =2 {deux} other {&lt;b&gt;beaucoup&lt;/b&gt;} }</target>
      </trans-unit>
      <trans-unit id="296ab5eab8d370822488c152586db3a5875ee1a2" datatype="html">
        <source>foo<x id="START_LINK" ctype="x-a" equiv-text="&lt;a&gt;"/>bar<x id="CLOSE_LINK" ctype="x-a" equiv-text="&lt;/a&gt;"/></source>
        <target>FOO<x id="START_LINK" ctype="x-a"/>BAR<x id="CLOSE_LINK" ctype=" x-a"/></target>
      </trans-unit>
      <trans-unit id="2e013b311caa0916478941a985887e091d8288b6" datatype="html">
        <source><x id="MAP NAME" equiv-text="{{ &apos;test&apos; //i18n(ph=&quot;map name&quot;) }}"/></source>
        <target><x id="MAP NAME"/></target>
      </trans-unit>
      <trans-unit id="2370d995bdcc1e7496baa32df20654aff65c2d10" datatype="html">
        <source>{VAR_PLURAL, plural, =0 {Found no results} =1 {Found one result} other {Found <x id="INTERPOLATION" equiv-text="{{response.getItemsList().length}}"/> results} }</source>
        <target>{VAR_PLURAL, plural, =0 {Pas de réponse} =1 {Une réponse} other {<x id="INTERPOLATION"/> réponses} }</target>
        <note priority="1" from="description">desc</note>
      </trans-unit>
    </body>
  </file>
</xliff>`
