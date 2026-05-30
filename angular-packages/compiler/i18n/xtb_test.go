package i18n_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
)

func TestXtbSerializer(t *testing.T) {
	serializer := i18n.NewXtb()

	loadAsMap := func(xtb string) map[string]string {
		res := serializer.Load(xtb, "url")
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

	t.Run("should load XTB files without placeholders", func(t *testing.T) {
		xtb := `<?xml version="1.0" encoding="UTF-8"?>
<translationbundle>
  <translation id="8841459487341224498">rab</translation>
</translationbundle>`
		got := loadAsMap(xtb)
		if got["8841459487341224498"] != "rab" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("should load XTB files with a doctype", func(t *testing.T) {
		xtb := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE translationbundle [<!ELEMENT translationbundle (translation)*>
<!ATTLIST translationbundle lang CDATA #REQUIRED>

<!ELEMENT translation (#PCDATA|ph)*>
<!ATTLIST translation id CDATA #REQUIRED>

<!ELEMENT ph EMPTY>
<!ATTLIST ph name CDATA #REQUIRED>
]>
<translationbundle>
  <translation id="8841459487341224498">rab</translation>
</translationbundle>`
		got := loadAsMap(xtb)
		if got["8841459487341224498"] != "rab" {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("should return the target locale", func(t *testing.T) {
		xtb := `<?xml version="1.0" encoding="UTF-8"?>
<translationbundle lang='fr'>
  <translation id="8841459487341224498">rab</translation>
</translationbundle>`
		res := serializer.Load(xtb, "url")
		if res.Locale == nil || *res.Locale != "fr" {
			t.Fatalf("locale = %v", res.Locale)
		}
	})

	t.Run("should load XTB files with placeholders", func(t *testing.T) {
		xtb := `<?xml version="1.0" encoding="UTF-8"?>
<translationbundle>
  <translation id="8877975308926375834"><ph name="START_PARAGRAPH"/>rab<ph name="CLOSE_PARAGRAPH"/></translation>
</translationbundle>`
		got := loadAsMap(xtb)
		want := `<ph name="START_PARAGRAPH"/>rab<ph name="CLOSE_PARAGRAPH"/>`
		if got["8877975308926375834"] != want {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("should replace ICU placeholders with their translations", func(t *testing.T) {
		xtb := `<?xml version="1.0" encoding="UTF-8" ?>
<translationbundle>
  <translation id="7717087045075616176">*<ph name="ICU"/>*</translation>
  <translation id="5115002811911870583">{VAR_PLURAL, plural, =1 {<ph name="START_PARAGRAPH"/>rab<ph name="CLOSE_PARAGRAPH"/>}}</translation>
</translationbundle>`
		got := loadAsMap(xtb)
		if got["7717087045075616176"] != `*<ph name="ICU"/>*` {
			t.Fatalf("got %v", got)
		}
		if got["5115002811911870583"] != `{VAR_PLURAL, plural, =1 {[<ph name="START_PARAGRAPH"/>, rab, <ph name="CLOSE_PARAGRAPH"/>]}}` {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("should load complex XTB files", func(t *testing.T) {
		xtb := `<?xml version="1.0" encoding="UTF-8" ?>
<translationbundle>
  <translation id="8281795707202401639"><ph name="INTERPOLATION"/><ph name="START_BOLD_TEXT"/>rab<ph name="CLOSE_BOLD_TEXT"/> oof</translation>
  <translation id="5115002811911870583">{VAR_PLURAL, plural, =1 {<ph name="START_PARAGRAPH"/>rab<ph name="CLOSE_PARAGRAPH"/>}}</translation>
  <translation id="130772889486467622">oof</translation>
  <translation id="4739316421648347533">{VAR_PLURAL, plural, =1 {{VAR_GENDER, gender, male {<ph name="START_PARAGRAPH"/>rab<ph name="CLOSE_PARAGRAPH"/>}} }}</translation>
</translationbundle>`
		got := loadAsMap(xtb)
		if got["8281795707202401639"] != `<ph name="INTERPOLATION"/><ph name="START_BOLD_TEXT"/>rab<ph name="CLOSE_BOLD_TEXT"/> oof` {
			t.Fatalf("got %v", got)
		}
		if got["5115002811911870583"] != `{VAR_PLURAL, plural, =1 {[<ph name="START_PARAGRAPH"/>, rab, <ph name="CLOSE_PARAGRAPH"/>]}}` {
			t.Fatalf("got %v", got)
		}
		if got["130772889486467622"] != `oof` {
			t.Fatalf("got %v", got)
		}
		if got["4739316421648347533"] != `{VAR_PLURAL, plural, =1 {[{VAR_GENDER, gender, male {[<ph name="START_PARAGRAPH"/>, rab, <ph name="CLOSE_PARAGRAPH"/>]}},  ]}}` {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("should be able to parse non-angular xtb files without error until access", func(t *testing.T) {
		xtb := `<?xml version="1.0" encoding="UTF-8" ?>
<translationbundle>
  <translation id="angular">is great</translation>
  <translation id="non angular">is <invalid>less</invalid> {count, plural, =0 {{GREAT}}}</translation>
</translationbundle>`

		var store i18n.TranslationStore
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("unexpected panic on load: %v", r)
				}
			}()
			store = serializer.Load(xtb, "url").I18nNodesByMsgId
		}()

		if len(store.Keys()) != 2 {
			t.Fatalf("keys = %v", store.Keys())
		}

		nodes, ok := store.Get("angular")
		if !ok || strings.Join(i18n.SerializeNodes(nodes), "") != "is great" {
			t.Fatalf("angular translation = %v, ok=%v", nodes, ok)
		}

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on lazy access")
			}
		}()
		store.Get("non angular")
	})

	t.Run("should throw on nested <translationbundle>", func(t *testing.T) {
		xtb := `<translationbundle><translationbundle></translationbundle></translationbundle>`
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(string), `<translationbundle> elements can not be nested`) {
				t.Fatalf("panic = %v", r)
			}
		}()
		loadAsMap(xtb)
	})

	t.Run("should throw when a <translation> has no id attribute", func(t *testing.T) {
		xtb := `<translationbundle>
  <translation></translation>
</translationbundle>`
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(string), `<translation> misses the "id" attribute`) {
				t.Fatalf("panic = %v", r)
			}
		}()
		loadAsMap(xtb)
	})

	t.Run("should throw when a placeholder has no name attribute", func(t *testing.T) {
		xtb := `<translationbundle>
  <translation id="1186013544048295927"><ph /></translation>
</translationbundle>`
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(string), `<ph> misses the "name" attribute`) {
				t.Fatalf("panic = %v", r)
			}
		}()
		loadAsMap(xtb)
	})

	t.Run("should throw on unknown xtb tags", func(t *testing.T) {
		xtb := `<what></what>`
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			} else if !regexp.MustCompile(`Unexpected tag`).MatchString(r.(string)) {
				t.Fatalf("panic = %v", r)
			}
		}()
		loadAsMap(xtb)
	})

	t.Run("should throw on unknown message tags", func(t *testing.T) {
		xtb := `<translationbundle>
  <translation id="1186013544048295927"><b>msg should contain only ph tags</b></translation>
</translationbundle>`
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			} else if !strings.Contains(r.(string), `[ERROR ->]<b>msg should contain only ph tags</b>`) {
				t.Fatalf("panic = %v", r)
			}
		}()
		loadAsMap(xtb)
	})

	t.Run("should throw on duplicate message id", func(t *testing.T) {
		xtb := `<translationbundle>
  <translation id="1186013544048295927">msg1</translation>
  <translation id="1186013544048295927">msg2</translation>
</translationbundle>`
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(string), `Duplicated translations for msg 1186013544048295927`) {
				t.Fatalf("panic = %v", r)
			}
		}()
		loadAsMap(xtb)
	})

	t.Run("should throw when trying to save an xtb file", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(string), "Unsupported") {
				t.Fatalf("panic = %v", r)
			}
		}()
		serializer.Write(nil, nil)
	})
}
