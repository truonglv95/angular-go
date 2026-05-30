package i18n_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/microsoft/typescript-go/angular-packages/compiler/ml_parser"
	"github.com/stretchr/testify/assert"
)

type testSerializer struct{}

func (s *testSerializer) Write(messages []i18n.Message, locale *string) string {
	var parts []string
	for _, msg := range messages {
		parts = append(parts, fmt.Sprintf("%s (%s|%s)", strings.Join(i18n.SerializeNodes(msg.Nodes), ""), msg.Meaning, msg.Description))
	}
	return strings.Join(parts, "//")
}

func (s *testSerializer) Load(content string, url string) i18n.LoadResult {
	return i18n.LoadResult{Locale: nil, I18nNodesByMsgId: nil}
}

func (s *testSerializer) Digest(message *i18n.Message) string {
	if message.Id != "" {
		return message.Id
	}
	return "default"
}

func (s *testSerializer) CreateNameMapper(message *i18n.Message) i18n.PlaceholderMapper {
	return nil
}

func humanizeMessages(catalog *i18n.MessageBundle) []string {
	return strings.Split(catalog.Write(&testSerializer{}, nil), "//")
}

func TestMessageBundle(t *testing.T) {
	t.Run("Messages", func(t *testing.T) {
		t.Run("should extract the message to the catalog", func(t *testing.T) {
			messages := i18n.NewMessageBundle(ml_parser.NewHtmlParser(), []string{}, make(map[string][]string), nil)
			errors := messages.UpdateFromTemplate("<p i18n=\"m|d\">Translate Me</p>", "url", nil)
			assert.Equal(t, 0, len(errors))
			assert.Equal(t, []string{"Translate Me (m|d)"}, humanizeMessages(messages))
		})

		t.Run("should extract and dedup messages", func(t *testing.T) {
			messages := i18n.NewMessageBundle(ml_parser.NewHtmlParser(), []string{}, make(map[string][]string), nil)
			errors := messages.UpdateFromTemplate(
				"<p i18n=\"m|d@@1\">Translate Me</p><p i18n=\"@@2\">Translate Me</p><p i18n=\"@@2\">Translate Me</p>",
				"url",
				nil,
			)
			assert.Equal(t, 0, len(errors))
			assert.Equal(t, []string{"Translate Me (m|d)", "Translate Me (|)"}, humanizeMessages(messages))
		})
	})
}
