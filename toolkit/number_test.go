package toolkit

import (
	"math"
	"testing"

	"github.com/lab47/peggysue"
	"github.com/stretchr/testify/require"
)

func TestNumbers(t *testing.T) {
	t.Run("matches a base 10 string", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		val, ok, err := p.Parse(Number, "10")
		r.NoError(err)
		r.True(ok)

		nv, ok := val.(*NumberValue)
		r.True(ok)

		i, err := nv.AsInt()
		r.NoError(err)

		r.Equal(10, i)
	})

	t.Run("matches a base 16 string", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val int
		}{
			{"0x10", 0x10},
			{"0xaf", 0xaf},
			{"0xAD", 0xAD},
			{"0x1aD", 0x1aD},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			i, err := nv.AsInt()
			r.NoError(err)

			r.Equal(rt.val, i)
		}
	})

	t.Run("matches a base 8 string", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val int
		}{
			{"0644", 0644},
			{"0100", 0100},
			{"0o644", 0644},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			i, err := nv.AsInt()
			r.NoError(err)

			r.Equal(rt.val, i)
		}
	})

	t.Run("matches a base 2 string", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val int
		}{
			{"0b1", 1},
			{"0b0", 0},
			{"0b11", 3},
			{"0b0011", 3},
			{"0b101", 5},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			i, err := nv.AsInt()
			r.NoError(err)

			r.Equal(rt.val, i)
		}
	})

	t.Run("matches a minus before a number", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val int
		}{
			{"-1", -1},
			{"-0b1", -1},
			{"-0x1", -1},
			{"-01", -1},
			{"-0o1", -1},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			i, err := nv.AsInt()
			r.NoError(err)

			r.Equal(rt.val, i)
		}
	})

	t.Run("allows underscores in numbers", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val int
		}{
			{"1_0", 10},
			{"0xabc_def", 0xabcdef},
			{"1_000_000", 1000000},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			i, err := nv.AsInt()
			r.NoError(err)

			r.Equal(rt.val, i)
		}
	})

	t.Run("requires a number before an underscores", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val int
		}{
			{"_1_0", 10},
			{"0x_abc_def", 0xabcdef},
			{"_1_000_000", 1000000},
		}

		for _, rt := range tests {
			_, ok, _ := p.Parse(Number, rt.in)
			r.False(ok, "parsing << %s >>", rt.in)
		}
	})

	t.Run("parses a rational number using a decimal", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val float64
		}{
			{"1.0", 1.0},
			{"3.14", 3.14},
			{"100.1", 100.1},
			{"-100.1", -100.1},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err, "parsing << %s >>", rt.in)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			f, err := nv.AsFloat64()
			r.NoError(err, "parsing << %s >>", rt.in)

			r.Equal(rt.val, f)
		}
	})

	t.Run("parses a scientific notation float", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val float64
		}{
			{"1e1", 1e1},
			{"3.14e19", 3.14e19},
			{"1e-9", 1e-9},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err, "parsing << %s >>", rt.in)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			f, err := nv.AsFloat64()
			r.NoError(err, "parsing << %s >>", rt.in)

			r.Equal(rt.val, f, "parsing << %s >>", rt.in)
		}
	})

	t.Run("parses hexadecimal floating points", func(t *testing.T) {
		r := require.New(t)

		p := peggysue.New()

		tests := []struct {
			in  string
			val float64
		}{
			{"0x1.921fb54442d18p1", math.Pi},
			// {"0x123.fffp5", 0x123.fffp5},
			{"0x12.p7", 0x12.p7},
		}

		for _, rt := range tests {
			val, ok, err := p.Parse(Number, rt.in)
			r.NoError(err, "parsing << %s >>", rt.in)
			r.True(ok)

			nv, ok := val.(*NumberValue)
			r.True(ok)

			f, err := nv.AsFloat64()
			r.NoError(err, "parsing << %s >>", rt.in)

			r.Equal(rt.val, f, "parsing << %s >>", rt.in)
		}
	})

}
