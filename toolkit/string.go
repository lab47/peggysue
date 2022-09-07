package toolkit

import (
	"fmt"
	"strings"

	p "github.com/lab47/peggysue"
)

type StringValue struct {
	Value string
}

var (
	em = func(in, out string) Rule {
		r := out[0]
		return p.Transform(p.S(in), func(s string) interface{} {
			return r
		})
	}

	octalSet = p.Range('0', '7')

	escaped = p.Seq(p.S(`\`), p.Or(p.PrefixTable(
		`a`, em(`a`, "\a"),
		`b`, em(`b`, "\b"),
		`\`, em(`\`, "\\"),
		`n`, em(`n`, "\n"),
		`t`, em(`t`, "\t"),
		`f`, em(`f`, "\f"),
		`v`, em(`v`, "\v"),
		`r`, em(`r`, "\r"),
		`x`, p.Seq(p.S(`x`), p.Transform(p.Seq(hexSet, hexSet), func(s string) interface{} {
			d := (digToByte(s[0]) << 4) | digToByte(s[1])
			return string([]byte{d})
		})),
		`u`, p.Seq(p.S(`u`), p.Transform(p.Many(hexSet, 4, 4, nil), func(s string) interface{} {
			d :=
				(rune(digToByte(s[0])) << 12) |
					(rune(digToByte(s[1])) << 8) |
					(rune(digToByte(s[2])) << 4) |
					rune(digToByte(s[3]))

			return rune(d)
		})),
		`U`, p.Seq(p.S(`U`), p.Transform(p.Many(hexSet, 8, 8, nil), func(s string) interface{} {
			d :=
				(rune(digToByte(s[0])) << 28) |
					(rune(digToByte(s[1])) << 24) |
					(rune(digToByte(s[2])) << 20) |
					(rune(digToByte(s[3])) << 16) |
					(rune(digToByte(s[4])) << 12) |
					(rune(digToByte(s[5])) << 8) |
					(rune(digToByte(s[6])) << 4) |
					rune(digToByte(s[7]))

			return rune(d)
		}))),
		p.Seq(p.Transform(p.Seq(octalSet, octalSet, octalSet), func(s string) interface{} {
			d := (digToByte(s[0]) << 6) | (digToByte(s[1]) << 3) | digToByte(s[2])
			return rune(d)
		})),
	))

	TripleDoubleQuotedString = makeString(`"""`, escaped)

	DoubleQuotedString = makeString(`"`, escaped)

	singleEscape = p.Or(em(`\'`, "'"), p.Capture(p.Seq(p.S(`\`), p.Any())))

	SingleQuotedString = makeString(`'`, singleEscape)

	TripleSingleQuotedString = makeString(`'''`, singleEscape)

	String = p.Or(TripleSingleQuotedString, SingleQuotedString, TripleDoubleQuotedString, DoubleQuotedString)
)

func makeString(quote string, escaped Rule) Rule {
	normal := p.Capture(p.Scan(func(str string) int {
		for i, b := range []byte(str) {
			if b == '\\' {
				if i == 0 {
					return -1
				}
				return i
			}

			if b == quote[0] && strings.HasPrefix(str[i:], quote) {
				if i == 0 {
					return -1
				}
				return i
			}
		}

		return len(str)
	}))

	segment := p.Or(escaped, normal)

	strBody := p.Many(p.Seq(p.Not(p.S(quote)), segment), 0, -1, func(vals []interface{}) interface{} {
		var sb strings.Builder

		for _, v := range vals {
			switch sv := v.(type) {
			case string:
				sb.WriteString(sv)
			case rune:
				sb.WriteRune(sv)
			case byte:
				sb.WriteByte(sv)
			default:
				panic(fmt.Sprintf("unexpected value: %T", v))
			}
		}

		return &StringValue{Value: sb.String()}
	})

	return p.Seq(p.S(quote), strBody, p.S(quote))
}
