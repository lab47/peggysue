package peggysue

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("parses a string", func(t *testing.T) {
		p := New()

		rule := S("foo")

		_, ok, err := p.Parse(rule, "foo")
		r := require.New(t)

		r.NoError(err)

		r.True(ok)

		_, ok, err = p.Parse(rule, "blah")

		r.NoError(err)

		r.False(ok)
	})

	t.Run("parses a regexp", func(t *testing.T) {
		p := New()

		rule := Re(`\d+`)

		_, ok, err := p.Parse(rule, "123")
		r := require.New(t)

		r.NoError(err)
		r.True(ok)

		_, ok, err = p.Parse(rule, "blah")
		r.NoError(err)
		r.False(ok)

		_, ok, err = p.Parse(rule, "blah123")
		r.NoError(err)
		r.False(ok)
	})

	t.Run("parses an ordered choise", func(t *testing.T) {
		p := New()

		r1 := S("foo")
		r2 := S("blah")
		r3 := Or(r1, r2)

		_, ok, err := p.Parse(r3, "foo")
		r := require.New(t)

		r.NoError(err)

		r.True(ok)

		_, ok, err = p.Parse(r3, "blah")

		r.NoError(err)

		r.True(ok)
	})

	t.Run("parses a sequence", func(t *testing.T) {
		p := New()

		r1 := S("foo")
		r2 := S(" blah")
		r3 := Seq(r1, r2)

		_, ok, err := p.Parse(r3, "foo")
		r := require.New(t)

		r.NoError(err)

		r.False(ok)

		_, ok, err = p.Parse(r3, "foo blah")

		r.NoError(err)

		r.True(ok)
	})

	t.Run("parses an or of sequences", func(t *testing.T) {
		p := New()

		r1 := S("foo")
		r2 := S(" blah")
		r3 := Seq(r1, r2)

		r4 := S("qux")
		r5 := S(" wooo")
		r6 := Seq(r4, r5)

		r7 := Or(r3, r6)

		_, ok, err := p.Parse(r7, "foo")
		r := require.New(t)

		r.NoError(err)

		r.False(ok)

		_, ok, err = p.Parse(r7, "foo blah")

		r.NoError(err)

		r.True(ok)

		_, ok, err = p.Parse(r7, "qux wooo")

		r.NoError(err)

		r.True(ok)
	})

	t.Run("parses zero or more", func(t *testing.T) {
		r := require.New(t)
		p := New()

		r1 := S("foo")
		r2 := Star(r1)

		_, ok, err := p.Parse(r2, "foo")
		r.NoError(err)
		r.True(ok)

		_, ok, err = p.Parse(r2, "blah")
		r.Error(err)
		r.False(ok)

		_, ok, err = p.Parse(r2, "foofoofoo")
		r.NoError(err)
		r.True(ok)
	})

	t.Run("parses one or more", func(t *testing.T) {
		p := New()

		r := require.New(t)
		r1 := S("foo")
		r2 := Plus(r1)

		_, ok, err := p.Parse(r2, "foo")
		r.NoError(err)
		r.True(ok)

		_, ok, err = p.Parse(r2, "blah")
		r.NoError(err)
		r.False(ok)

		_, ok, err = p.Parse(r2, "foofoofoo")
		r.NoError(err)
		r.True(ok)
	})

	t.Run("parses an optional", func(t *testing.T) {
		p := New()

		r := require.New(t)
		r1 := S("foo")
		r2 := Maybe(r1)

		_, ok, err := p.Parse(r2, "foo")
		r.NoError(err)
		r.True(ok)

		_, ok, err = p.Parse(r2, "blah")
		r.Error(err)
		r.False(ok)
	})

	t.Run("parses a check", func(t *testing.T) {
		p := New()

		r := require.New(t)
		r1 := S("foo")
		r2 := Or(r1, S("blah"))
		r3 := S("f")
		r4 := Seq(Check(r3), r2)

		_, ok, err := p.Parse(r4, "foo")
		r.NoError(err)
		r.True(ok)

		_, ok, err = p.Parse(r4, "blah")
		r.NoError(err)
		r.False(ok)
	})

	t.Run("parses a not", func(t *testing.T) {
		p := New()

		r := require.New(t)
		r1 := S("foo")
		r3 := S("b")
		r4 := Seq(Not(r3), r1)

		_, ok, err := p.Parse(r4, "foo")
		r.NoError(err)
		r.True(ok)

		_, ok, err = p.Parse(r4, "blah")
		r.NoError(err)
		r.False(ok)
	})

	t.Run("can use a reference", func(t *testing.T) {
		p := New()

		r := require.New(t)

		f1 := R("f1")

		r1 := Seq(f1, S(" done"))

		f1.Set(S("ok"))

		_, ok, err := p.Parse(r1, "ok done")
		r.NoError(err)
		r.True(ok)
	})

	t.Run("can use a reference in left rec", func(t *testing.T) {
		p := New()

		r := require.New(t)

		num := S("1")

		f1 := R("f1")

		f1.Set(
			Or(
				Seq(f1, S("+"), num),
				num,
			),
		)

		r.True(f1.LeftRecursive())

		_, ok, err := p.Parse(f1, "1+1")
		r.NoError(err)
		r.True(ok)
	})

	t.Run("can use a reference without left rec", func(t *testing.T) {
		p := New()

		r := require.New(t)

		f1 := R("f1")

		f1.Set(S("1"))

		r1 := Or(
			Seq(f1, S("+")),
			Seq(f1, S("-")),
		)

		r.False(f1.LeftRecursive())

		st, res := p.parse(r1, "1-", "")
		r.True(res.matched)

		r.NotNil(st.memos[0][f1])
	})

	t.Run("references can be automatically created using a label factory", func(t *testing.T) {
		p := New()

		r := require.New(t)

		l := Refs()

		r1 := Or(
			Seq(l.Ref("one"), S("+")),
			Seq(l.Ref("one"), S("-")),
		)

		f1 := l.Set("one", S("1"))

		r.False(f1.LeftRecursive())

		st, res := p.parse(r1, "1-", "")
		r.True(res.matched)

		r.NotNil(st.memos[0][f1])
	})

	t.Run("allows for actions to produce results", func(t *testing.T) {
		p := New()

		r := require.New(t)

		r1 := Action(S("1"), func(Values) interface{} { return 1 })

		result, ok, err := p.Parse(r1, "1")
		r.NoError(err)
		r.True(ok)

		r.Equal(1, result)
	})

	t.Run("allows for naming results and passing them to actions", func(t *testing.T) {
		p := New()

		r := require.New(t)

		num := Transform(Re(`\d+`), func(str string) interface{} {
			i, _ := strconv.Atoi(str)
			return i
		})

		r1 := Action(
			Seq(
				Named("i", num),
				S("+"),
				Named("j", num),
			),
			func(vals Values) interface{} {
				i := vals.Get("i").(int)
				j := vals.Get("j").(int)

				return i + j
			},
		)

		result, ok, err := p.Parse(r1, "3+4")
		r.NoError(err)
		r.True(ok)

		r.Equal(7, result)
	})

	t.Run("allows check actions", func(t *testing.T) {
		p := New()

		r := require.New(t)
		r1 := S("foo")
		r2 := Or(r1, S("blah"))
		r4 := Seq(
			Named("prefix", Capture(r2)),
			CheckAction(func(vals Values) bool {
				return vals.Get("prefix").(string)[0] == 'f'
			}),
		)

		_, ok, err := p.Parse(r4, "foo")
		r.NoError(err)
		r.True(ok)

		_, ok, err = p.Parse(r4, "blah")
		r.NoError(err)
		r.False(ok)
	})

	t.Run("can populate positions on results", func(t *testing.T) {
		p := New()

		r := require.New(t)

		num := Transform(Re(`\d+`), func(str string) interface{} {
			i, _ := strconv.Atoi(str)

			return &testIntNode{Val: i}
		})

		calc := Action(
			Seq(Named("i", num), S("+"), Named("j", num)),
			func(v Values) interface{} {
				return &testPlusNode{
					i: v.Get("i").(*testIntNode),
					j: v.Get("j").(*testIntNode),
				}
			})

		res, ok, err := p.Parse(calc, "3+4")
		r.NoError(err)
		r.True(ok)

		n := res.(*testPlusNode)

		r.Equal(0, n.i.posStart)
		r.Equal(1, n.i.posEnd)
		r.Equal(1, n.i.line)

		r.Equal(2, n.j.posStart)
		r.Equal(3, n.j.posEnd)
		r.Equal(1, n.i.line)

		r.Equal(0, n.posStart)
		r.Equal(3, n.posEnd)
		r.Equal(1, n.i.line)
	})

	t.Run("properly memoizes results", func(t *testing.T) {
		p := New()

		r := require.New(t)

		num := Transform(Plus(Range('0', '9')), func(str string) interface{} {
			i, _ := strconv.Atoi(str)

			return &testIntNode{Val: i}
		})

		i := Memo(Named("i", num))
		j := Memo(Named("j", num))

		calc := Action(
			Or(
				Seq(i, S("-"), j),
				Seq(i, S("+"), j),
			),
			func(v Values) interface{} {
				return &testPlusNode{
					i: v.Get("i").(*testIntNode),
					j: v.Get("j").(*testIntNode),
				}
			})

		st, res := p.parse(calc, "3+4", "")
		r.True(res.matched)

		r.Equal(1, st.memos[0][i].used)
	})

	t.Run("tracks the furthest it got", func(t *testing.T) {
		p := New()

		r := require.New(t)

		num := Plus(Range('0', '9'))

		i := num
		j := num

		calc := Or(
			Seq(i, S("-"), j),
			Seq(i, S("+"), j),
			Seq(i, S("*"), S("2")),
		)

		st, res := p.parse(calc, "3*4", "")
		r.False(res.matched)

		r.Equal(2, st.maxPos)
	})

	t.Run("a simple calculator", func(t *testing.T) {
		r := require.New(t)
		num := Transform(Plus(Range('0', '9')), func(s string) interface{} {
			i, _ := strconv.Atoi(s)
			return i
		})
		x := R("x")
		x.Set(
			Or(
				Action(Seq(Named("i", x), S("+"), Named("j", num)), func(v Values) interface{} {
					i := v.Get("i").(int)
					j := v.Get("j").(int)

					return i + j
				}),
				num,
			),
		)

		r.True(x.LeftRecursive())

		ps := New()

		val, ok, _ := ps.Parse(Seq(x, EOS()), "3")
		r.True(ok)
		r.Equal(3, val)

		y := Seq(x, EOS())

		val, ok, _ = ps.Parse(y, "3+4")
		r.True(ok)
		r.Equal(7, val)
	})

	t.Run("can apply a match into a struct", func(t *testing.T) {
		p := New()

		r := require.New(t)

		numLit := Transform(Plus(Range('0', '9')), func(str string) interface{} {
			i, _ := strconv.Atoi(str)
			return i
		})

		num := Apply(Named("val", numLit), testIntNode{})

		i := Memo(Named("i", num))
		j := Memo(Named("j", num))

		calc := Action(
			Or(
				Seq(i, S("-"), j),
				Seq(i, S("+"), j),
			),
			func(v Values) interface{} {
				return &testPlusNode{
					i: v.Get("i").(*testIntNode),
					j: v.Get("j").(*testIntNode),
				}
			})

		val, ok, err := p.Parse(calc, "3+4")
		r.NoError(err)
		r.True(ok)

		n := val.(*testPlusNode)

		r.Equal(3, n.i.Val)
		r.Equal(4, n.j.Val)
	})

}

type testIntNode struct {
	Val int `ast:"val"`

	posStart int
	posEnd   int
	line     int
}

func (t *testIntNode) SetPosition(start, end, line int, filename string) {
	t.posStart = start
	t.posEnd = end
	t.line = line
}

type testPlusNode struct {
	i, j *testIntNode

	posStart int
	posEnd   int
	line     int
}

func (t *testPlusNode) SetPosition(start, end, line int, filename string) {
	t.posStart = start
	t.posEnd = end
	t.line = line
}

func BenchmarkParse(b *testing.B) {
	p := New()

	num := Transform(Plus(Range('0', '9')), func(str string) interface{} {
		return nil
	})

	i := Named("i", num)
	j := Named("j", num)

	calc := Action(
		Or(
			Seq(i, S("-"), j),
			Seq(i, S("+"), j),
		),
		func(v Values) interface{} {
			return nil
		},
	)

	for i := 0; i < b.N; i++ {
		p.Parse(calc, "3+4")
	}
}
