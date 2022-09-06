package toolkit

import (
	"testing"

	"github.com/lab47/peggysue"
	"github.com/stretchr/testify/require"
)

func TestWhitespace(t *testing.T) {
	t.Run("matches unicode's definition of whitespace", func(t *testing.T) {
		p := peggysue.New()

		_, ok, err := p.Parse(peggysue.Plus(IsWhiteSpace), " \t\n\v")
		r := require.New(t)
		r.NoError(err)
		r.True(ok)
	})
}
