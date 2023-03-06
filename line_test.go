package peggysue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLines(t *testing.T) {
	t.Run("can calculate line from byte position", func(t *testing.T) {
		var s state

		s.linePos = computeLines("foo\nbar\n\nbaz")

		r := assert.New(t)

		r.Equal(1, s.line(0))
		r.Equal(1, s.line(3))
		r.Equal(2, s.line(4))
		r.Equal(3, s.line(8))
		r.Equal(4, s.line(9))
		r.Equal(4, s.line(10))

	})
}
