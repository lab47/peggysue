package toolkit

import (
	"testing"

	p "github.com/lab47/peggysue"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Run("parses a double quoted string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `"hello"`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("hello", sv.Value)
	})

	t.Run("parses a escape in a double quote string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `"h\tello"`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("h\tello", sv.Value)
	})

	t.Run("parses a hex escape in a double quote string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `"h\x12ello"`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("h\x12ello", sv.Value)
	})

	t.Run("parses an octal escape in a double quote string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `"h\112ello"`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("h\112ello", sv.Value)
	})

	t.Run("parses a unicode escape in a double quote string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `"h\u0012ello"`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("h\u0012ello", sv.Value)
	})

	t.Run("parses an extended unicode escape in a double quote string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `"h\u0012ello"`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("h\U00000012ello", sv.Value)
	})

	t.Run("parses a single quoted string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `'hello\n'`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("hello\\n", sv.Value)
	})

	t.Run("parses a single quoted string with escaping", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, `'he\'llo\n'`)
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("he'llo\\n", sv.Value)
	})

	t.Run("parses a triple quoted string", func(t *testing.T) {
		r := require.New(t)

		pr := p.New()

		val, ok, err := pr.Parse(String, "'''\nhello\n'''")
		r.NoError(err)
		r.True(ok)

		sv := val.(*StringValue)

		r.Equal("\nhello\n", sv.Value)
	})
}

func BenchmarkString(b *testing.B) {
	pr := p.New()

	str := `"hello\n this has normal stuff and \t escape codes"`

	for i := 0; i < b.N; i++ {
		pr.Parse(String, str)
	}
}
