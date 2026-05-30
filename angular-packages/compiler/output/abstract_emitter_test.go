package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbstractEmitter_EscapeIdentifier(t *testing.T) {
	t.Run("should escape single quotes", func(t *testing.T) {
		assert.Equal(t, `'\''`, EscapeIdentifier(`'`, false))
	})

	t.Run("should escape backslash", func(t *testing.T) {
		assert.Equal(t, `'\\'`, EscapeIdentifier(`\`, false))
	})

	t.Run("should escape newlines", func(t *testing.T) {
		assert.Equal(t, `'\n'`, EscapeIdentifier("\n", false))
	})

	t.Run("should escape carriage returns", func(t *testing.T) {
		assert.Equal(t, `'\r'`, EscapeIdentifier("\r", false))
	})

	t.Run("should add quotes for non-identifiers", func(t *testing.T) {
		assert.Equal(t, `'=='`, EscapeIdentifier("==", false))
	})

	t.Run("does not escape class (but it probably should)", func(t *testing.T) {
		assert.Equal(t, "class", EscapeIdentifier("class", false))
	})
}
