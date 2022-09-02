# Peggysue

# Peggysue  [![GoDoc](https://godoc.org/github.com/lab47/peggysue?status.svg)](http://godoc.org/github.com/lab47/peggysue) [![Report card](https://goreportcard.com/badge/github.com/lab47/peggysue)](https://goreportcard.com/report/github.com/lab47/peggysue) [![Sourcegraph](https://sourcegraph.com/github.com/lab47/peggysue/-/badge.svg)](https://sourcegraph.com/github.com/lab47/peggysue?badge)

A helpful, neighborhood PEG framework.

## Usage

Peggysue implements a PEG framework as a package rather than being an external
grammar. This makes it super easy to embed in existing applications to provide
easy parsing of whatever you need.

## Differences from other PEGs

Peggysue provides memoization of values (a critical function of Packrat parsing)
but only memoizes Ref rules. This gives the programmer an easy way to trade off
memory for time. Most larger grammars will contain the major rules of the the grammar
with another set of helper rules. In most grammars, all rules regardless of use
are memoized. This can create huge memory footprints for rules that would be better
off just run again than saved.

## Values

Each rule when executed against the input stream will produce a value when the rule
matches successfully. The documentaiton for each rule creationing function documentments
the value of the rule. Action, Transform, and Apply are rules that exist explicitly
to return a value. The other rules have a heiristic about what value they return, with
many returning nil.

# Example

Here is the classic calculator example:

```golang
package main

import (
	"fmt"
	"strconv"

	p "github.com/lab47/peggysue"
)

func main() {
	var (
		expr = p.R("expr")
		term = p.R("term")
		num  = p.R("num")
	)

	expr.Set(
		p.Or(
			p.Action(p.Seq(p.Named("i", expr), p.S("+"), p.Named("j", term)), func(v p.Values) interface{} {
				i := v.Get("i").(int)
				j := v.Get("j").(int)
				return i + j
			}),
			p.Action(p.Seq(p.Named("i", expr), p.S("-"), p.Named("j", term)), func(v p.Values) interface{} {
				i := v.Get("x").(int)
				j := v.Get("y").(int)
				return i - j
			}),
			term,
		),
	)

	num.Set(
		p.Transform(p.Plus(p.Range('0', '9')), func(s string) interface{} {
			i, _ := strconv.Atoi(s)
			return i
		}),
	)

	term.Set(
		p.Or(
			p.Action(p.Seq(p.Named("i", term), p.S("*"), p.Named("j", num)), func(v p.Values) interface{} {
				i := v.Get("ii").(int)
				j := v.Get("jj").(int)
				return i * j
			}),
			p.Action(p.Seq(p.Named("i", term), p.S("/"), p.Named("j", num)), func(v p.Values) interface{} {
				i := v.Get("a").(int)
				j := v.Get("b").(int)
				return i / j
			}),
			num,
		),
	)

	val, ok, _ := p.New().Parse(expr, "3*1+2*2")
	if !ok {
		panic("failed")
	}

	fmt.Printf("=> %d\n", val)
}

```

## License

BSD-2-Clause
