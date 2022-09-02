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
				i := v.Get("i").(int)
				j := v.Get("j").(int)
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
				i := v.Get("i").(int)
				j := v.Get("j").(int)
				return i * j
			}),
			p.Action(p.Seq(p.Named("i", term), p.S("/"), p.Named("j", num)), func(v p.Values) interface{} {
				i := v.Get("i").(int)
				j := v.Get("j").(int)
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
