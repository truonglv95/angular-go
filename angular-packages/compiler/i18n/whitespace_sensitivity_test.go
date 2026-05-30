package i18n

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
)

type debugSerializerVisitor struct{}

func (v *debugSerializerVisitor) VisitText(text *Text, context any) any {
	return text.Value
}

func (v *debugSerializerVisitor) VisitContainer(container *Container, context any) any {
	var children []string
	for _, child := range container.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("[%s]", strings.Join(children, ", "))
}

func (v *debugSerializerVisitor) VisitIcu(icu *Icu, context any) any {
	var strCases []string
	for _, k := range icu.CaseOrders {
		c := icu.Cases[k]
		strCases = append(strCases, fmt.Sprintf("%s {%s}", k, c.Visit(v, nil).(string)))
	}
	return fmt.Sprintf("{%s, %s, %s}", icu.Expression, icu.Type, strings.Join(strCases, ", "))
}

func (v *debugSerializerVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context any) any {
	if ph.IsVoid {
		return fmt.Sprintf("<ph tag name=\"%s\"/>", ph.StartName)
	}
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("<ph tag name=\"%s\">%s</ph name=\"%s\">", ph.StartName, strings.Join(children, ", "), ph.CloseName)
}

func (v *debugSerializerVisitor) VisitPlaceholder(ph *Placeholder, context any) any {
	return fmt.Sprintf("<ph name=\"%s\"/>", ph.Name)
}

func (v *debugSerializerVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context any) any {
	return fmt.Sprintf("<ph icu name=\"%s\"/>", ph.Name)
}

func (v *debugSerializerVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context any) any {
	var children []string
	for _, child := range ph.Children {
		children = append(children, child.Visit(v, nil).(string))
	}
	return fmt.Sprintf("<ph block name=\"%s\">%s</ph name=\"%s\">", ph.StartName, strings.Join(children, ", "), ph.CloseName)
}

type assertableMessage struct {
	id   string
	text string
}

func extractMessagesForWhitespace(source string, preserveWhitespace bool) []assertableMessage {
	bundle := NewMessageBundle(
		ml_parser.NewHtmlParser(),
		[]string{} /* implicitTags */,
		map[string][]string{} /* implicitAttrs */,
		nil /* locale */,
	)
	bundle.preserveWhitespace = preserveWhitespace

	errors := bundle.UpdateFromTemplate(source, "url", nil)
	if len(errors) > 0 {
		var errMsgs []string
		for _, err := range errors {
			errMsgs = append(errMsgs, err.Error())
		}
		panic(fmt.Sprintf("Failed to parse template:\n%s", strings.Join(errMsgs, "\n\n")))
	}

	messages := bundle.GetMessages()
	var result []assertableMessage
	visitor := &debugSerializerVisitor{}
	for _, msg := range messages {
		var nodeTexts []string
		for _, node := range msg.Nodes {
			nodeTexts = append(nodeTexts, node.Visit(visitor, nil).(string))
		}
		result = append(result, assertableMessage{
			id:   DecimalDigest(msg),
			text: strings.Join(nodeTexts, ""),
		})
	}
	return result
}

func TestWhitespaceSensitivity(t *testing.T) {
	t.Run("ignores whitespace changes when disabled", func(t *testing.T) {
		t.Run("from converting one-line messages to block messages", func(t *testing.T) {
			initialSource := strings.TrimSpace(`
<div i18n>Hello, World!</div>
<div i18n>Hello {{ abc }}</div>
<div i18n>Start {{ abc }} End</div>
<div i18n>{{ first }} middle {{ end }}</div>
<div i18n><a href="/foo">First Second</a></div>
<div i18n>Before <a href="/foo">First Second</a> After</div>
<div i18n><input type="text" /></div>
<div i18n>Before <input type="text" /> After</div>
<div i18n>{apples, plural, =1 {One apple.} =other {Many apples.}}</div>
<div i18n>{apples, plural, =other {{bananas, plural, =other{Many apples and bananas.}}}}</div>

i18nPreserveWhitespaceForLegacyExtraction does not support changing ICU case text.
Test case is disabled by omitting the i18n attribute.
<div>{apples, plural, =1 {One apple.} =other {Many apples.}}</div>
`)
			initial := extractMessagesForWhitespace(initialSource, false)

			multiLineSource := strings.TrimSpace(`
<div i18n>
  Hello, World!
</div>
<div i18n>
  Hello {{ abc }}
</div>
<div i18n>
  Start {{ abc }} End
</div>
<div i18n>
  {{ first }} middle {{ end }}
</div>
<div i18n>
  <a href="/foo">
    First
    Second
  </a>
</div>
<div i18n>
  Before
  <a href="/foo">
    First
    Second
  </a>
  After
</div>
<div i18n>
  <input type="text" />
</div>
<div i18n>
  Before
  <input type="text" />
  After
</div>
<div i18n>{
  apples, plural,
  =1 {One apple.}
  =other {Many apples.}
}</div>
<div i18n>{
  apples, plural,
  =other {{bananas, plural,
    =other{Many apples and bananas.}
  }}
}</div>

i18nPreserveWhitespaceForLegacyExtraction does not support changing ICU case text.
Test case is disabled by omitting the i18n attribute.
<div>{
  apples, plural,
  =1 {
    One
    apple.
  }
  =other {
    Many
    apples.
  }
}</div>
`)
			multiLine := extractMessagesForWhitespace(multiLineSource, false)

			if len(multiLine) != 10 {
				t.Errorf("expected 10 messages, got %d", len(multiLine))
			}

			var initialIds, multiLineIds []string
			for _, m := range initial {
				initialIds = append(initialIds, m.id)
			}
			for _, m := range multiLine {
				multiLineIds = append(multiLineIds, m.id)
			}

			if !reflect.DeepEqual(multiLineIds, initialIds) {
				t.Errorf("expected ids to match, got %v want %v", multiLineIds, initialIds)
			}
		})

		t.Run("from indenting a message", func(t *testing.T) {
			initialSource := strings.TrimSpace(`
<div i18n>
  Hello, World!
</div>
<div i18n>
  Hello {{ abc }}
</div>
<div i18n>
  Start {{ abc }} End
</div>
<div i18n>
  {{ first }} middle {{ end }}
</div>
<div i18n>
  <a href="/foo">
    Foo
  </a>
</div>
<div i18n>
  Before
  <a href="/foo">Link</a>
  After
</div>
<div i18n>
  <input type="text" />
</div>
<div i18n>
  Before <input type="text" /> After
</div>
<div i18n>{
  apples, plural,
  =1 {One apple.}
  =other {Many apples.}
}</div>
<div i18n>{
  apples, plural,
  =other {{bananas, plural,
    =other{Many apples and bananas.}
  }}
}</div>

i18nPreserveWhitespaceForLegacyExtraction does not support indenting ICU case text.
Test case is disabled by omitting the i18n attribute.
<div>{
  apples, plural,
  =1 {
    One
    apple.
  }
  =other {
    Many
    apples.
  }
}</div>
`)
			initial := extractMessagesForWhitespace(initialSource, false)

			indentedSource := strings.TrimSpace(`
<div id="container">
  <div i18n>
    Hello, World!
  </div>
  <div i18n>
    Hello {{ abc }}
  </div>
  <div i18n>
    Start {{ abc }} End
  </div>
  <div i18n>
    {{ first }} middle {{ end }}
  </div>
  <div i18n>
    <a href="/foo">
      Foo
    </a>
  </div>
  <div i18n>
    Before
    <a href="/foo">Link</a>
    After
  </div>
  <div i18n>
    <input type="text" />
  </div>
  <div i18n>
    Before <input type="text" /> After
  </div>
  <div i18n>{
    apples, plural,
    =1 {One apple.}
    =other {Many apples.}
  }</div>
  <div i18n>{
    apples, plural,
    =other {{bananas, plural,
      =other{Many apples and bananas.}
    }}
  }</div>

  i18nPreserveWhitespaceForLegacyExtraction does not support indenting ICU case text.
  Test case is disabled by omitting the i18n attribute.
  <div>{
    apples, plural,
    =1 {
      One
      apple.
    }
    =other {
      Many
      apples.
    }
  }</div>
</div>
`)
			indented := extractMessagesForWhitespace(indentedSource, false)

			if len(indented) != 10 {
				t.Errorf("expected 10 messages, got %d", len(indented))
			}

			var initialIds, indentedIds []string
			for _, m := range initial {
				initialIds = append(initialIds, m.id)
			}
			for _, m := range indented {
				indentedIds = append(indentedIds, m.id)
			}

			if !reflect.DeepEqual(indentedIds, initialIds) {
				t.Errorf("expected ids to match, got %v want %v", indentedIds, initialIds)
			}
		})

		t.Run("from adjusting line wrapping", func(t *testing.T) {
			initialSource := strings.TrimSpace(`
<div i18n>
  This is a long message which maybe
  exceeds line length.
</div>
<div i18n>
  Hello {{ veryLongExpressionWhichMaybeExceedsLineLength | async }}
</div>
<div i18n>
  This is a long {{ abc }} which maybe
  exceeds line length.
</div>
<div i18n>
  {{ first }} long message {{ end }}
</div>
<div i18n>
  <a href="/foo" veryLongAttributeWhichMaybeExceedsLineLength>
    This is a long message which maybe
    exceeds line length.
  </a>
</div>
<div i18n>
  This is a
  long <a href="/foo" veryLongAttributeWhichMaybeExceedsLineLength>message</a> which
  maybe exceeds line length.
</div>
<div i18n>
  <input type="text" veryLongAttributeWhichMaybeExceedsLineLength />
</div>
<div i18n>
  Before <input type="text" veryLongAttributeWhichMaybeExceedsLineLength /> After
</div>

i18nPreserveWhitespaceForLegacyExtraction does not support line wrapping ICU case text.
Test case is disabled by omitting the i18n attribute.
<div>{
  apples, plural, =other {Very long text which maybe exceeds line length.}
}</div>
`)
			initial := extractMessagesForWhitespace(initialSource, false)

			adjustedSource := strings.TrimSpace(`
<div i18n>
  This is a long message which
  maybe exceeds line length.
</div>
<div i18n>
  Hello {{
    veryLongExpressionWhichMaybeExceedsLineLength
    | async
  }}
</div>
<div i18n>
  This is a long {{ abc }} which
  maybe exceeds line length.
</div>
<div i18n>
  {{ first }}
  long message
  {{ end }}
</div>
<div i18n>
  <a
      href="/foo"
      veryLongAttributeWhichMaybeExceedsLineLength>
    This is a long message which
    maybe exceeds line length.
  </a>
</div>
<div i18n>
  This is a long
  <a
      href="/foo"
      veryLongAttributeWhichMaybeExceedsLineLength>
    message
  </a>
  which maybe exceeds line length.
</div>
<div i18n>
  <input
    type="text"
    veryLongAttributeWhichMaybeExceedsLineLength
  />
</div>
<div i18n>
  Before
  <input
    type="text"
    veryLongAttributeWhichMaybeExceedsLineLength
  />
  After
 </div>

i18nPreserveWhitespaceForLegacyExtraction does not support line wrapping ICU case text.
Test case is disabled by omitting the i18n attribute.
<div>{
  apples, plural, =other {
    Very long text which
    maybe exceeds line length.
  }
}</div>
`)
			adjusted := extractMessagesForWhitespace(adjustedSource, false)

			if len(adjusted) != 8 {
				t.Errorf("expected 8 messages, got %d", len(adjusted))
			}

			var initialIds, adjustedIds []string
			for _, m := range initial {
				initialIds = append(initialIds, m.id)
			}
			for _, m := range adjusted {
				adjustedIds = append(adjustedIds, m.id)
			}

			if !reflect.DeepEqual(adjustedIds, initialIds) {
				t.Errorf("expected ids to match, got %v want %v", adjustedIds, initialIds)
			}
		})

		t.Run("from trimming significant whitespace", func(t *testing.T) {
			initialSource := strings.TrimSpace(`
<div i18n> Hello, World! </div>
<div i18n> Hello {{ abc }} </div>
<div i18n> Start {{ abc }} End </div>
<div i18n> {{ first }} middle {{ end }} </div>
<div i18n> <a href="/foo">Foo</a> </div>
<div i18n><a href="/foo"> Foo </a></div>
<div i18n> Before <a href="/foo">Link</a> After </div>
<div i18n> <input type="text" /> </div>
<div i18n> Before <input type="text" /> After </div>

i18nPreserveWhitespaceForLegacyExtraction does not support trimming ICU case text.
Test case is disabled by omitting the i18n attribute.
<div>Hello {
  apples, plural,
  =1 { One apple. }
  =other { Many apples. }
}</div>
`)
			initial := extractMessagesForWhitespace(initialSource, false)

			trimmedSource := strings.TrimSpace(`
<div i18n>Hello, World!</div>
<div i18n>Hello {{ abc }}</div>
<div i18n>Start {{ abc }} End</div>
<div i18n>{{ first }} middle {{ end }}</div>
<div i18n><a href="/foo">Foo</a></div>
<div i18n><a href="/foo">Foo</a></div>
<div i18n>Before <a href="/foo">Link</a> After</div>
<div i18n><input type="text" /></div>
<div i18n>Before <input type="text" /> After</div>

i18nPreserveWhitespaceForLegacyExtraction does not support trimming ICU case text.
Test case is disabled by omitting the i18n attribute.
<div>Hello {
  apples, plural,
  =1 {One apple.}
  =other {Many apples.}
}</div>
`)
			trimmed := extractMessagesForWhitespace(trimmedSource, false)

			if len(trimmed) != 9 {
				t.Errorf("expected 9 messages, got %d", len(trimmed))
			}

			if !reflect.DeepEqual(trimmed, initial) {
				t.Errorf("expected trimmed to match initial, got %v want %v", trimmed, initial)
			}
		})
	})
}
