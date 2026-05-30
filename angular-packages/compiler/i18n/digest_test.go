package i18n_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler/i18n"
	"github.com/stretchr/testify/assert"
)

func TestDigest(t *testing.T) {
	t.Run("digest", func(t *testing.T) {
		t.Run("must return the ID if it's explicit", func(t *testing.T) {
			msg := &i18n.Message{
				Id: "i",
			}
			assert.Equal(t, "i", i18n.Digest(msg))
		})
	})

	t.Run("sha1", func(t *testing.T) {
		t.Run("should work on empty strings", func(t *testing.T) {
			assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", i18n.Sha1(""))
		})

		t.Run("should returns the sha1 of \"hello world\"", func(t *testing.T) {
			assert.Equal(t, "a9993e364706816aba3e25717850c26c9cd0d89d", i18n.Sha1("abc"))
		})

		t.Run("should returns the sha1 of unicode strings", func(t *testing.T) {
			assert.Equal(t, "3becb03b015ed48050611c8d7afe4b88f70d5a20", i18n.Sha1("你好，世界"))
		})

		t.Run("should support arbitrary string size", func(t *testing.T) {
			prefix := []rune("你好，世界")
			result := []rune(i18n.Sha1(string(prefix)))
			for size := len(prefix); size < 5000; size += 101 {
				result = append(prefix, []rune(i18n.Sha1(string(result)))...)
				for len(result) < size {
					result = append(result, result...)
				}
				result = result[len(result)-size:]
			}
			assert.Equal(t, "24c2dae5c1ac6f604dbe670a60290d7ce6320b45", i18n.Sha1(string(result)))
		})
	})

	t.Run("decimal fingerprint", func(t *testing.T) {
		t.Run("should work on well known inputs w/o meaning", func(t *testing.T) {
			fixtures := map[string]string{
				"  Spaced  Out  ": "3976450302996657536",
				"Last Name":        "4407559560004943843",
				"First Name":       "6028371114637047813",
				"View":             "2509141182388535183",
				"START_BOLDNUMEND_BOLD of START_BOLDmillionsEND_BOLD": "29997634073898638",
				"The customer's credit card was authorized for AMOUNT and passed all risk checks.": "6836487644149622036",
				"Hello world!":                      "3022994926184248873",
				"Jalape\u00f1o":                     "8054366208386598941",
				"The set of SET_NAME is {XXX, ...}.": "135956960462609535",
				"NAME took a trip to DESTINATION.":   "768490705511913603",
				"by AUTHOR (YEAR)":                  "7036633296476174078",
				"":                                  "4416290763660062288",
			}

			for msg, want := range fixtures {
				assert.Equal(t, want, i18n.ComputeMsgId(msg, ""), "message: %q", msg)
			}
		})

		t.Run("should work on well known inputs with meaning", func(t *testing.T) {
			fixtures := map[string][2]string{
				"7790835225175622807": {"Last Name", "Gmail UI"},
				"1809086297585054940": {"First Name", "Gmail UI"},
				"3993998469942805487": {"View", "Gmail UI"},
			}

			for id, val := range fixtures {
				assert.Equal(t, id, i18n.ComputeMsgId(val[0], val[1]), "msgId for: %v", val)
			}
		})

		t.Run("should support arbitrary string size", func(t *testing.T) {
			prefix := []rune("你好，世界")
			result := []rune(i18n.ComputeMsgId(string(prefix), ""))
			for size := len(prefix); size < 5000; size += 101 {
				result = append(prefix, []rune(i18n.ComputeMsgId(string(result), ""))...)
				for len(result) < size {
					result = append(result, result...)
				}
				result = result[len(result)-size:]
			}
			assert.Equal(t, "2122606631351252558", i18n.ComputeMsgId(string(result), ""))
		})
	})
}
