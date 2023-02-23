package peggysue

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForms(t *testing.T) {
	type op struct {
		lhs interface{}
		op  string
		rhs interface{}
	}

	t.Run("precendence", func(t *testing.T) {
		// This demos a classic way to model operator precendence using left
		// recursion.
		n := Capture(Re("[0-9]+"))
		s := Branches("s", func(bb BranchesBuilder, r Rule) {
			bb.Add("a", Action(
				Seq(
					Named("lhs", r),
					Named("op", Capture(Set('*', '/'))),
					Named("rhs", n),
				),
				func(v Values) interface{} {
					return op{
						lhs: v.Get("lhs"),
						op:  v.Get("op").(string),
						rhs: v.Get("rhs"),
					}
				},
			))
			bb.Add("n", n)
		})

		e := Branches("s", func(bb BranchesBuilder, r Rule) {
			bb.Add("a", Action(
				Seq(
					Named("lhs", r),
					Named("op", Capture(Set('+', '-'))),
					Named("rhs", s),
				),
				func(v Values) interface{} {
					return op{
						lhs: v.Get("lhs"),
						op:  v.Get("op").(string),
						rhs: v.Get("rhs"),
					}
				},
			))
			bb.Add("s", s)
		})

		p := New(WithDebug(false))

		v, ok, err := p.Parse(e, "1+2*3")
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, op{
			lhs: "1",
			op:  "+",
			rhs: op{
				lhs: "2",
				op:  "*",
				rhs: "3",
			},
		}, v)

		v, ok, err = p.Parse(e, "1+2+3")
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, op{
			lhs: op{
				lhs: "1",
				op:  "+",
				rhs: "2",
			},
			op:  "+",
			rhs: "3",
		}, v)

		v, ok, err = p.Parse(e, "1*2*3")
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, op{
			lhs: op{
				lhs: "1",
				op:  "*",
				rhs: "2",
			},
			op:  "*",
			rhs: "3",
		}, v)
	})
}
