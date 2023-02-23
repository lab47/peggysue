package peggysue

import (
	"fmt"
	"strconv"
	"strings"
)

type matchScan struct {
	basicRule
	fn func(str string) int
}

func (m *matchScan) match(s *state) result {
	if s.pos >= s.inputSize {
		return result{}
	}

	loc := m.fn(s.cur())
	if loc == -1 {
		s.bad(m)
		return result{}
	}

	s.goodRange(m, loc)
	s.advance(loc, m)
	return result{matched: true}
}

func (m *matchScan) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchScan) print() string {
	return ":<scan>"
}

// Scan is a manual optimization rule. It returns a Rule that calls
// the given function, passing the current input. The return value
// is how much of the input sequence to consume. If -1 is returned,
// the rule fails, otherwise the requested about of input is consumed
// and the rule passes.
//
// The value of the match is nil.
func Scan(fn func(str string) int) Rule {
	return &matchScan{
		fn: fn,
	}
}

// matchString1 is a specialized form of matchString used to match
// just a single byte.
type matchString1 struct {
	basicRule
	b byte
}

func (m *matchString1) match(s *state) result {
	if s.pos >= s.inputSize {
		return result{}
	}

	if s.input[s.pos] == m.b {
		s.good(m)
		s.advance(1, m)
		return result{matched: true}
	}

	s.bad(m)
	return result{}
}

func (m *matchString1) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchString1) print() string {
	return strconv.Quote(string([]byte{m.b}))
}

// matchString2 is a specialized form of matchString used to match
// the next 2 bytes.
type matchString2 struct {
	basicRule
	a, b byte
}

func (m *matchString2) match(s *state) result {
	if s.pos+1 >= s.inputSize {
		return result{}
	}

	if s.input[s.pos] != m.a {
		s.bad(m)
		return result{}
	}

	if s.input[s.pos+1] != m.b {
		s.bad(m)
		return result{}
	}

	s.good(m)
	s.advance(2, m)
	return result{matched: true}
}

func (m *matchString2) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchString2) print() string {
	return strconv.Quote(string([]byte{m.a, m.b}))
}

type matchPrefixTable struct {
	basicRule
	rules map[byte]Rule
}

func (m *matchPrefixTable) match(s *state) result {
	if s.pos >= s.inputSize {
		return result{}
	}

	b := s.input[s.pos]

	r, ok := m.rules[b]
	if !ok {
		return result{}
	}

	return s.match(r)
}

func (m *matchPrefixTable) detectLeftRec(r Rule, rs ruleSet) bool {
	for _, sub := range m.rules {
		if !rs.Add(sub) {
			return false
		}

		if r == sub {
			return true
		}

		if sub.detectLeftRec(r, rs) {
			return true
		}
	}

	return false
}

func (m *matchPrefixTable) print() string {
	var subs []string

	for _, r := range m.rules {
		subs = append(subs, r.print())
	}
	return strings.Join(subs, " | ")
}

// PrefixTable is a manual optimization rule. It is used to create a non-ordered
// choice to a set of a rules based on the next byte in the input sequence.
// The detection of the byte does not consume any input, which makes PrefixTable
// the equivalent of using Or(Seq(Check(rune), ...), ...).
// The arguments are alternating pairs of (byte|string, rule). If a string is passed,
// the first byte of the string is used only.
// This rule is used to optimize large a large Or() where each rule has a fixed
// byte prefix that determines if it should rune. An example is in toolkit/toolkit.go,
// in the escaped rule.
//
// The value of the match is the value of the sub-rule that matched correctly.
func PrefixTable(entries ...interface{}) Rule {
	matches := make(map[byte]Rule)

	for i := 0; i < len(entries); i += 2 {
		var b byte
		switch sv := entries[i].(type) {
		case byte:
			b = sv
		case string:
			if len(sv) != 1 {
				panic("only accepts a one character string")
			}

			b = sv[0]
		default:
			panic("key must be byte or 1-char string")
		}

		r := entries[i+1].(Rule)

		matches[b] = r
	}

	return &matchPrefixTable{rules: matches}
}

// matchEither is an automatic optimization rule. It is used when Or() is passed 2 rules.
type matchEither struct {
	basicRule
	a Rule
	b Rule
}

func (m *matchEither) match(s *state) result {
	save := s.mark()

	res := m.a.match(s)
	if res.matched {
		s.good(m)
		return res
	}

	s.restore(save)

	res = m.b.match(s)
	if res.matched {
		s.good(m)
		return res
	}

	s.restore(save)
	s.bad(m)
	return result{}
}

func (m *matchEither) detectLeftRec(r Rule, rs ruleSet) bool {
	for _, sub := range []Rule{m.a, m.b} {
		if !rs.Add(sub) {
			return false
		}

		if r == sub {
			return true
		}

		if sub.detectLeftRec(r, rs) {
			return true
		}
	}

	return false
}

func (m *matchEither) print() string {
	var subs []string

	for _, r := range []Rule{m.a, m.b} {
		subs = append(subs, r.print())
	}
	return strings.Join(subs, " | ")
}

// matchNotByte is an automatic optimization rule. It's used when detected Not(S("x")), where
// x is anything.
type matchNotByte struct {
	basicRule
	b byte
}

func (m *matchNotByte) match(s *state) result {
	if s.pos >= len(s.input) {
		return result{}
	}

	return result{matched: m.b != s.input[s.pos]}
}

func (m *matchNotByte) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchNotByte) print() string {
	return fmt.Sprintf(`!"%s"`, string([]byte{m.b}))
}

// matchBoth is an automatic optimization rule. It's used when Seq() is passed 2 rules.
type matchBoth struct {
	basicRule
	a Rule
	b Rule
}

func (m *matchBoth) match(s *state) result {
	mark := s.mark()

	res := m.a.match(s)
	if !res.matched {
		s.restore(mark)
		s.bad(m)
		return result{}
	}

	res2 := m.b.match(s)
	if !res2.matched {
		s.restore(mark)
		s.bad(m)
		return result{}
	}

	if res2.value != nil {
		res.value = res2.value
	}

	s.good(m)
	return res
}

func (m *matchBoth) detectLeftRec(r Rule, rs ruleSet) bool {
	for _, o := range []Rule{m.a, m.b} {
		if !rs.Add(o) {
			return false
		}

		if o == r || o.detectLeftRec(r, rs) {
			return true
		}
	}

	return false
}

func (m *matchBoth) print() string {
	var subs []string

	for _, r := range []Rule{m.a, m.b} {
		subs = append(subs, r.print())
	}
	return strings.Join(subs, " ")
}

// matchBoth is an automatic optimization rule. It's used when Seq() is passed 3 rules.
type matchThree struct {
	basicRule
	a Rule
	b Rule
	c Rule
}

func (m *matchThree) match(s *state) result {
	pos := s.mark()

	res := m.a.match(s)
	if !res.matched {
		s.restore(pos)
		s.bad(m)
		return result{}
	}

	res2 := m.b.match(s)
	if !res2.matched {
		s.restore(pos)
		s.bad(m)
		return result{}
	}

	if res2.value != nil {
		res.value = res2.value
	}

	res3 := m.c.match(s)
	if !res3.matched {
		s.restore(pos)
		s.bad(m)
		return result{}
	}

	if res3.value != nil {
		res.value = res3.value
	}

	s.good(m)
	return res
}

func (m *matchThree) detectLeftRec(r Rule, rs ruleSet) bool {
	for _, o := range []Rule{m.a, m.b, m.c} {
		if !rs.Add(o) {
			return false
		}

		if o == r || o.detectLeftRec(r, rs) {
			return true
		}
	}

	return false
}

func (m *matchThree) print() string {
	var subs []string

	for _, r := range []Rule{m.a, m.b, m.c} {
		subs = append(subs, r.print())
	}
	return strings.Join(subs, " ")
}
