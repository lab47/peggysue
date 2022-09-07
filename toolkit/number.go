package toolkit

import (
	"errors"
	"math/big"

	p "github.com/lab47/peggysue"
)

type NumberValue struct {
	Base     int
	Str      string
	Negative bool

	PostDecimal string
	Power       *NumberValue
}

// Dup creates a shallow copy of NumberValue and returns it.
func (n *NumberValue) Dup() *NumberValue {
	nw := *n
	return &nw
}

func lower(c byte) byte {
	return c | ('x' - 'X')
}

// AsBig returns either a big.Int or a big.Rat, depending on
// the type of kind of number that was parsed.
func (n *NumberValue) AsBig() (interface{}, error) {
	if n.PostDecimal == "" && n.Power == nil {
		return n.AsBigInt()
	}

	return n.AsBigRat()
}

// AsBigRat returns a big.Rat representation of the number
// value. big.Rat retains a high precision view of floating
// point numbers by maintaining a seperate numerator and
// denominator.
func (n *NumberValue) AsBigRat() (*big.Rat, error) {
	numb, err := n.AsBigInt()
	if err != nil {
		return nil, err
	}

	rhs, err := asBigInt(n.PostDecimal, int64(n.Base))
	if err != nil {
		return nil, err
	}

	base := big.NewInt(int64(n.Base))

	acc := new(big.Int).Div(rhs, base)

	width := 1
	for acc.Int64() != 0 {
		acc.Div(acc, base)
		width++
	}

	offset := base.Exp(base, big.NewInt(int64(width)), nil)

	numb.Mul(numb, offset)
	numb.Add(numb, rhs)

	if n.Negative {
		numb.Neg(numb)
	}

	denom := offset

	if n.Power != nil {
		pi, err := asBigInt(n.Power.Str, 10)
		if err != nil {
			return nil, err
		}

		piBase := big.NewInt(int64(n.Power.Base))
		piFact := piBase.Exp(piBase, pi, nil)

		if n.Power.Negative {
			denom.Mul(denom, piFact)
		} else {
			numb.Mul(numb, piFact)
		}
	}

	return new(big.Rat).SetFrac(numb, denom), nil
}

var ErrRangeError = errors.New("range error")

// AsBigInt returns a big.Int representation of the number.
// big.Int can integers of infinite bit length.
func (n *NumberValue) AsBigInt() (*big.Int, error) {
	return asBigInt(n.Str, int64(n.Base))
}

func digToByte(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'A' <= c && c <= 'Z':
		return c - 'A' + 10
	case 'a' <= c && c <= 'z':
		return lower(c) - 'a' + 10
	default:
		panic("bad digit")
	}
}

func asBigInt(str string, base int64) (*big.Int, error) {
	var x int64

	for _, c := range []byte(str) {
		var d byte

		switch {
		case c == '_':
			continue
		case '0' <= c && c <= '9':
			d = c - '0'
		case 'a' <= lower(c) && lower(c) <= 'z':
			d = lower(c) - 'a' + 10
		}

		if d > byte(base) {
			return nil, ErrRangeError
		}

		x *= base
		x += int64(d)
	}

	return big.NewInt(x), nil
}

// AsInt returns the number value as a Go int.
func (n *NumberValue) AsInt() (int, error) {
	bi, err := n.AsBigInt()
	if err != nil {
		return 0, err
	}

	if n.Negative {
		bi.Neg(bi)
	}

	return int(bi.Int64()), nil
}

// AsInt returns the number value as a Go float64.
func (n *NumberValue) AsFloat64() (float64, error) {
	r, err := n.AsBigRat()
	if err != nil {
		return 0, err
	}

	v, _ := r.Float64()
	return v, nil
}

func xset(r Rule) Rule {
	return p.Seq(r, p.Star(p.Or(p.S("_"), r)))
}

var (
	Numbers = p.Refs()

	hexSet = p.Or(p.Range('0', '9'), p.Range('a', 'f'), p.Range('A', 'F'))

	// HexInt parses a hexidemical integer, such as 0x1d.
	HexInt = Numbers.Set("hex-int", p.Seq(p.S("0x"),
		p.Transform(xset(hexSet),
			func(s string) interface{} {
				return &NumberValue{Base: 16, Str: s}
			})))

	// BinINt parses a base 2 (ie binary) number, such as 0b1101.
	BinaryInt = Numbers.Set("binary-int", p.Seq(p.S("0b"), p.Transform(
		xset(p.Range('0', '1')), func(s string) interface{} {
			return &NumberValue{Base: 2, Str: s}
		})))

	// OctalInt parsers a base 8 (ie octal) number, such as 0644 and 0o644
	OctalInt = Numbers.Set("octal-int", p.Or(p.Seq(p.S("0o"), p.Transform(
		xset(p.Range('0', '7')), func(s string) interface{} {
			return &NumberValue{Base: 8, Str: s}
		})),
		p.Seq(p.S("0"), p.Transform(
			xset(p.Range('0', '7')), func(s string) interface{} {
				return &NumberValue{Base: 8, Str: s}
			}))))

	// DecimalInt parses a base 10 (ie decimal) number, such as 42.
	DecimalInt = Numbers.Set("decimal-int", p.Transform(xset(p.Range('0', '9')), func(s string) interface{} {
		return &NumberValue{Base: 10, Str: s}
	}))

	// UnsignedInt parses unsigned base 2, base 8, base 10, and base 16 numbers.
	UnsignedInt = Numbers.Set("unsigned-int", p.Or(
		HexInt,
		BinaryInt,
		OctalInt,
		DecimalInt,
	))

	setSign = func(v p.Values) interface{} {
		num := v.Get("num").(*NumberValue)

		if s, ok := v.Get("sign").(bool); ok {
			num = num.Dup()
			num.Negative = s
		}

		return num
	}

	// Int parses signed and unsigned base 2, base 8, base 10, and base 16 numbers.
	Int = Numbers.Set("int",
		p.Action(p.Seq(
			p.Maybe(p.Named("sign", sign)),
			p.Named("num", UnsignedInt)),
			setSign),
	)

	// UnsignedFloat parses an unsigned floating point number, such as 1.42.
	UnsignedFloat = Numbers.Set("unsigned-float", p.Action(
		// TODO should we support hexidecimal floats?
		p.Seq(
			p.Named("lhs", DecimalInt),
			p.S("."),
			p.Named("rhs", DecimalInt),
		),
		func(v p.Values) interface{} {
			lhs := v.Get("lhs").(*NumberValue).Dup()
			rhs := v.Get("rhs").(*NumberValue)

			lhs.PostDecimal = rhs.Str

			return lhs
		}))

	// UnsignedHexFloat parses an unsigned hex floating point number,
	// such as 0x123.fffp5
	UnsignedHexFloat = Numbers.Set("unsigned-hex-float", p.Action(
		// TODO should we support hexidecimal floats?
		p.Seq(
			p.Named("lhs", HexInt),
			p.S("."),
			p.Named("rhs", p.Capture(p.Star(hexSet))),
			p.Set('p', 'P'),
			p.Named("sign", p.Maybe(sign)),
			p.Named("power", DecimalInt),
		),
		func(v p.Values) interface{} {
			lhs := v.Get("lhs").(*NumberValue).Dup()
			rhs := v.Get("rhs").(string)

			power := v.Get("power").(*NumberValue).Dup()

			if sv, ok := v.Get("sign").(bool); ok {
				power.Negative = sv
			}

			power.Base = 2
			lhs.PostDecimal = rhs
			lhs.Power = power

			return lhs
		}))

	sign = p.Transform(p.Set('-', '+'), func(s string) interface{} {
		return s == "-"
	})

	// Float parses floating point number, such as 1.42 and -3.14.
	Float = Numbers.Set("float",
		p.Action(p.Seq(
			p.Named("sign", p.Maybe(sign)),
			p.Named("num", p.Or(UnsignedFloat, UnsignedHexFloat))),
			setSign),
	)

	// SciNum parsers a number in scientific notation, such as 1e16 and -3.14e8
	SciNum = Numbers.Set("sci", p.Action(
		p.Seq(
			p.Named("num", p.Or(Float, UnsignedInt)),
			p.Set('e', 'E'),
			p.Maybe(p.Named("sign", sign)),
			p.Named("power", DecimalInt),
		),
		func(v p.Values) interface{} {
			num := v.Get("num").(*NumberValue)
			power := v.Get("power").(*NumberValue)

			if sign, ok := v.Get("sign").(bool); ok {
				power = power.Dup()
				power.Negative = sign
			}

			ret := num.Dup()
			ret.Power = power

			return ret
		}))

	Number = Numbers.Set("number", p.Or(
		SciNum,
		Float,
		Int,
	))
)
