package peggysue

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/go-hclog"
)

// SetPositioner is an optional interface. When values implement it, peggysue
// will call it with the information about the position of the value in the
// inputs tream.
type SetPositioner interface {
	SetPosition(start, end int)
}

type ruleSet map[Rule]struct{}

// Rule is the currency of peggysue. The parser provides the ability
// to match a Rule against an input stream. Rules are created with
// the functions on peggysue, like peggysue.S() to match a literal
// string.
type Rule interface {
	match(s *state) result
	detectLeftRec(r Rule, rs ruleSet) bool
	print() string
}

type result struct {
	matched bool
	value   interface{}
}

func (rs ruleSet) Add(r Rule) bool {
	if _, ok := rs[r]; ok {
		return false
	}

	rs[r] = struct{}{}

	return true
}

// These are the rules!

type matchAny struct {
}

func (m *matchAny) match(s *state) result {
	if s.pos >= len(s.input) {
		return result{}
	}

	b := s.input[s.pos]

	var sz int

	if b < utf8.RuneSelf {
		sz = 1
	} else {
		_, sz = utf8.DecodeRuneInString(s.cur())
	}

	s.advance(m, sz)

	return result{matched: true}
}

func (m *matchAny) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchAny) print() string {
	return "."
}

// Any returns a rule that will match one rune from the input stream
// of any value. In other words, it only fails if there is no more input.
//
// The value of the match is nil.
func Any() Rule {
	return &matchAny{}
}

type matchString struct {
	str string
}

func (m *matchString) match(s *state) result {
	if len(m.str) > len(s.cur()) {
		s.bad(m)
		return result{}
	}

	if strings.HasPrefix(s.cur(), m.str) {
		s.advance(m, len(m.str))
		return result{matched: true}
	}

	s.bad(m)
	return result{}
}

func (m *matchString) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchString) print() string {
	return strconv.Quote(m.str)
}

// S returns a Rule that will match a literal string exactly.
//
// The value of the match is nil.
func S(str string) Rule {
	return &matchString{str: str}
}

type matchRegexp struct {
	re  *regexp.Regexp
	str string
}

func (m *matchRegexp) match(s *state) result {
	loc := m.re.FindStringIndex(s.cur())
	if loc == nil {
		s.bad(m)
		return result{}
	}

	s.advance(m, loc[1])
	return result{matched: true}
}

func (m *matchRegexp) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchRegexp) print() string {
	return "/" + m.str + "/"
}

// Re returns a Rule that will match a regexp at the current input
// position. This regexp can only match at the beginning of the input,
// it does not search the input for a match.
//
// The value of the match is nil.
func Re(re string) Rule {
	return &matchRegexp{
		str: re,
		re:  regexp.MustCompile(`\A` + re),
	}
}

type matchCharRange struct {
	start, end rune
}

func (m *matchCharRange) match(s *state) result {
	if s.pos >= len(s.input) {
		s.bad(m)
		return result{}
	}

	b := s.input[s.pos]

	var (
		rn rune
		sz int
	)

	if b < utf8.RuneSelf {
		rn = rune(b)
		sz = 1
	} else {
		rn, sz = utf8.DecodeRuneInString(s.cur())
	}

	if rn >= m.start && rn <= m.end {
		s.advance(m, sz)
		return result{matched: true}
	}

	s.bad(m)
	return result{}
}

func (m *matchCharRange) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchCharRange) print() string {
	return fmt.Sprintf("[%c-%c]", m.start, m.end)
}

// Range returns a rule that will match the next rune in the input
// stream as being at least 'start', and at most 'end'. This corresponds
// with the regexp pattern `[A-Z]` but is much faster as it does not require
// any regexp tracking.
//
// The value of the match is nil.
func Range(start, end rune) Rule {
	return &matchCharRange{
		start: start,
		end:   end,
	}
}

type matchOr struct {
	rules []Rule
}

func (m *matchOr) match(s *state) result {
	save := s.mark()

	for _, r := range m.rules {
		res := r.match(s)
		if res.matched {
			s.good(m)
			return res
		}

		s.restore(save)
	}

	s.bad(m)
	return result{}
}

func (m *matchOr) detectLeftRec(r Rule, rs ruleSet) bool {
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

func (m *matchOr) print() string {
	var subs []string

	for _, r := range m.rules {
		subs = append(subs, r.print())
	}
	return strings.Join(subs, " | ")
}

// Or returns a Rule that will try each of the given rules, completing when
// the first one successfully matches. This corresponds with a PEG's "ordered
// choice" operation.
//
// The value of the match is the value of the sub-rule that matched correctly.
func Or(rules ...Rule) Rule {
	return &matchOr{rules: rules}
}

type matchSeq struct {
	rules []Rule
}

func (m *matchSeq) match(s *state) result {
	var ret result

	for _, r := range m.rules {
		res := r.match(s)
		if !res.matched {
			s.bad(m)
			return result{}
		}

		if res.value != nil {
			ret.value = res.value
		}
	}

	ret.matched = true
	s.good(m)
	return ret
}

func (m *matchSeq) detectLeftRec(r Rule, rs ruleSet) bool {
	sub := m.rules[0]

	if !rs.Add(sub) {
		return false
	}

	if sub == r {
		return true
	}

	return sub.detectLeftRec(r, rs)
}

func (m *matchSeq) print() string {
	var subs []string

	for _, r := range m.rules {
		subs = append(subs, r.print())
	}
	return strings.Join(subs, " ")
}

// Seq returns a rule that will attempt to match each of the given rules
// in order. It only matches succesfully if each of it's rules match.
//
// The value of the match is the value of the right most sub-rule that
// produced a non-nil value.
func Seq(rules ...Rule) Rule {
	return &matchSeq{rules: rules}
}

type matchZeroOrMore struct {
	rule Rule
}

func (m *matchZeroOrMore) match(s *state) result {
	for {
		res := m.rule.match(s)
		if res.matched {
			continue
		}

		s.good(m)
		return result{value: res.value, matched: true}
	}
}

func (m *matchZeroOrMore) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchZeroOrMore) print() string {
	return addParens(m.rule) + "*"
}

// Star returns a rule that will attempt to match it's given rule
// as many times as possible. This rule always matches because it
// allows for zero matches. It corresponds to the star rule ("e*") in most
// PEGs.
//
// The value of the match is the value of the last iteration of applying
// the sub rule.
func Star(rule Rule) Rule {
	return &matchZeroOrMore{rule: rule}
}

type matchOneOrMore struct {
	rule Rule
}

func (m *matchOneOrMore) match(s *state) result {
	res := m.rule.match(s)
	if !res.matched {
		return result{}
	}

	for {
		res := m.rule.match(s)
		if res.matched {
			continue
		}

		return s.check(m, result{value: res.value, matched: true})
	}
}

func (m *matchOneOrMore) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func addParens(r Rule) string {
	switch r.(type) {
	case *matchOr, *matchSeq:
		return "(" + r.print() + ")"
	default:
		return r.print()
	}
}

func (m *matchOneOrMore) print() string {
	return addParens(m.rule) + "+"
}

// Plus returns a rule that attempts to match it's given rule
// as many times as possible. The rule requires that the given
// rule match at least once. It corresponds to the plus rule ("e+") in
// most PEGs.
//
// The value of the match is the value of the last successful match of
// the sub-rule
func Plus(rule Rule) Rule {
	return &matchOneOrMore{rule: rule}
}

type matchOptional struct {
	rule Rule
}

func (m *matchOptional) match(s *state) result {
	res := m.rule.match(s)
	res.matched = true

	return s.check(m, res)
}

func (m *matchOptional) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchOptional) print() string {
	return m.rule.print() + "?"
}

// Maybe returns a rule that will allow it's rule to match, but
// will always return that it's succesfully matched, regardless
// of what it's rule does. This corresponds with the question mark
// rule ("e?") in most PEGs.
//
// The value of the match is the value of the sub-rule.
func Maybe(rule Rule) Rule {
	return &matchOptional{rule: rule}
}

type matchCheck struct {
	rule Rule
}

func (m *matchCheck) match(s *state) result {
	defer s.restore(s.mark())

	return s.check(m, m.rule.match(s))
}

func (m *matchCheck) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchCheck) print() string {
	return "&" + m.rule.print()
}

// Check returns a rule that will attempt to match it's given rule
// and returns it's given rules match result, but it does not consume
// an input on the stream. This corresponds with the and-predicate ("&e") in
// most PEGs.
//
// The value of the match is the value of the sub-rule.
func Check(rule Rule) Rule {
	return &matchCheck{rule: rule}
}

type matchNot struct {
	rule Rule
}

func (m *matchNot) match(s *state) result {
	defer s.restore(s.mark())

	res := m.rule.match(s)
	res.matched = !res.matched

	return s.check(m, res)
}

func (m *matchNot) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchNot) print() string {
	return "!" + m.rule.print()
}

// Not returns a rule that will attempt to match it's given rule.
// If the rule matches, it returns that it did not match (inverting
// the match result). Regardless of matching, it does not consume any
// input on the stream. This corresponds with the not-predicate ("!e") in
// most PEGs.
//
// The value of the match is the value of the sub-rule.
func Not(rule Rule) Rule {
	return &matchNot{rule: rule}
}

// Ref is a type that provides a reference to a rule. This allows for creating
// recursive rule sets. Ref rules are memoized, meaning the
// value of the ref's rule and it's position in the stream are saved and returned
// to prevent constantly re-running the same rule on the same input. This
// is a key feature of Packrat parsing as it tames the time complexity of infinite
// backtracking to linear time.
type Ref interface {
	Rule

	// Set assigns the given rule to the ref. When the ref is matched as a rule,
	// it will delegate the matching to this rule. The rule will not be invoked
	// the multiple times at the same input position as the Ref caches the result
	// of previous attempts. Thusly it's critical that the rule not depend on state
	// when calculate it's value.
	Set(r Rule)

	// Indicates if this reference has left recursive properties.
	LeftRecursive() bool
}

type matchRef struct {
	name    string
	rule    Rule
	leftRec bool
}

func (r *matchRef) Set(rule Rule) {
	// When invoking a ref, introduce a new scope since this matches the
	// semantics of all parsers, where within a single named rule, there
	// is a unique scope of produced values from it's parts.
	if _, ok := rule.(*matchScope); ok {
		rule = &matchScope{rule: rule}
	}

	r.rule = rule

	rs := make(ruleSet)

	if rule == r {
		r.leftRec = true
	} else {
		rs.Add(rule)
		if rule.detectLeftRec(r, rs) {
			r.leftRec = true
		}
	}
}

func (r *matchRef) LeftRecursive() bool {
	return r.leftRec
}

func (m *matchRef) match(s *state) result {
	// The memoization code was ported from
	// https://github.com/we-like-parsers/pegen_experiments/blob/master/story7/memo.py

	pos := s.mark()
	memo := s.memos[pos]
	if memo == nil {
		memo = make(map[Rule]*memoResult)
		s.memos[pos] = memo
	}

	if res, ok := memo[m]; ok {
		res.used++
		s.restore(res.endPos)
		return s.check(m, res.result)
	} else if m.leftRec {
		var (
			lastRes = result{}
			lastPos = pos
		)

		mr := &memoResult{endPos: pos}
		memo[m] = mr

		for {
			s.restore(pos)

			res := m.rule.match(s)
			endPos := s.mark()

			if endPos <= lastPos {
				break
			}

			lastRes = res
			lastPos = endPos

			mr.result = res
			mr.endPos = endPos
		}

		s.restore(lastPos)
		return s.check(m, lastRes)
	} else {
		if res, ok := memo[m]; ok {
			s.restore(res.endPos)
			return res.result
		}
		res := m.rule.match(s)
		endPos := s.mark()

		if res.matched {
			if endPos <= pos {
				panic("bad memo")
			}
		} else if endPos != pos {
			panic("bad memo")
		}

		memo[m] = &memoResult{result: res, endPos: endPos}

		return s.check(m, res)
	}
}

func (m *matchRef) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || (m.rule != nil && m.rule.detectLeftRec(r, rs))
}

func (m *matchRef) print() string {
	return m.name
}

// R returns a Ref type. These are used to create recursive rule sets where a rule
// is used before it's definition is created. Ref rules are memoized, meaning the
// value of the ref's rule and it's position in the stream are saved and returned
// to prevent constantly re-running the same rule on the same input. This
// is a key feature of Packrat parsing as it tames the time complexity of infinite
// backtracking to linear time.
//
// The value of the match is the value of the sub-rule.
func R(name string) Ref {
	return &matchRef{name: name}
}

// Memo creates a rule that perform memoization as part of matching. Memoization is used
// to speed up matching.
//
// The value of the match is the value of the sub-rule.
func Memo(rule Rule) Rule {
	r := R("")
	r.Set(rule)
	return r
}

// Values provides the same of rule values gathered. The names correspond
// to Named rules that were observed in the current scope.
type Values interface {
	Get(name string) interface{}
}

type valMap map[string]interface{}

func (m valMap) Get(name string) interface{} {
	return m[name]
}

type matchAction struct {
	rule Rule
	fn   func(Values) interface{}
}

func (m *matchAction) match(s *state) result {
	pos := s.mark()

	res := m.rule.match(s)
	if res.matched {
		res.value = m.fn(s.values)

		if sp, ok := res.value.(SetPositioner); ok {
			sp.SetPosition(pos, s.mark())
		}
	}

	return s.check(m, res)
}

func (m *matchAction) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchAction) print() string {
	return m.rule.print()
}

// Action returns a rule that when it's given rule is matched, the given
// function is called. The return value of the function becomes the rule's
// value. The Values argument contains all rule values observed in the curent
// rule scope (toplevel or from invoking a Ref).
//
// The value of the match is the return value of the given function.
func Action(r Rule, fn func(Values) interface{}) Rule {
	return &matchScope{rule: &matchAction{rule: r, fn: fn}}
}

type matchApply struct {
	rule Rule
	typ  reflect.Type
}

func (m *matchApply) match(s *state) result {
	res := m.rule.match(s)
	if res.matched {
		res.value = m.expand(s)
	}

	return res
}

func (m *matchApply) expand(s *state) interface{} {
	ret := reflect.New(m.typ)

	rv := ret.Elem()

	for i := 0; i < rv.NumField(); i++ {
		ft := m.typ.Field(i)

		if !ft.IsExported() {
			continue
		}

		name := ft.Tag.Get("ast")
		if name == "" {
			name = ft.Name
		}

		if val := s.values.Get(name); val != nil {
			rv.Field(i).Set(reflect.ValueOf(val))
		}
	}

	return ret.Interface()
}

func (m *matchApply) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchApply) print() string {
	return m.rule.print()
}

// Apply returns a rule that when it's given rule matches, it will
// make a new struct and attempt to assign rule values in scope
// to the members of the struct. The struct type is computed from
// the argument v and then saved to be used at match time.
// The value names should exactly match the struct fields OR if
// the struct fields use the `ast:` tag, it will be used to look
// for values.
//
// For example, given:
// type Node struct {
//    Age int `ast:"age"`
// }
//
// Apply(Named("age", numberRule), Node{})
//
// When the above Apply matches, the value from numberRule will be assigned
// to a new value of Node.
//
// The value of the match is the newly created and populated struct.
func Apply(rule Rule, v interface{}) Rule {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		panic("apply must be passed a struct to infer the type of")
	}

	return &matchApply{
		rule: rule,
		typ:  rv.Type(),
	}
}

type matchScope struct {
	rule Rule
}

func (m *matchScope) match(s *state) result {
	curValues := s.values
	defer func() {
		s.values = curValues
	}()

	v := make(valMap)
	s.values = v

	return s.check(m, m.rule.match(s))
}

func (m *matchScope) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchScope) print() string {
	return m.rule.print()
}

// Scope introduces a new rule scope. This is generally not needed as
// most users will use the automatically created scope from an Action rule.
//
// The value of the match is the value of the sub-rule.
func Scope(rule Rule) Rule {
	return &matchScope{rule: rule}
}

type matchNamed struct {
	name string
	rule Rule
}

func (m *matchNamed) match(s *state) result {
	res := m.rule.match(s)
	if res.matched {
		if s.p.debug {
			fmt.Printf("N (%p) %s => %v\n", s.values, m.name, res.value)
		}
		s.values[m.name] = res.value
	}

	return s.check(m, res)
}

func (m *matchNamed) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchNamed) print() string {
	return fmt.Sprintf("%s:%s", m.rule.print(), m.name)
}

// Named will assign the value of the given rule to the current rule scope
// if it matches.
//
// The value of the match is the value of the sub-rule.
func Named(name string, rule Rule) Rule {
	return &matchNamed{name: name, rule: rule}
}

type matchTransform struct {
	rule Rule
	fn   func(str string) interface{}
}

func (m *matchTransform) match(s *state) result {
	pos := s.mark()

	res := m.rule.match(s)
	if res.matched {
		res.value = m.fn(s.input[pos:s.mark()])

		if sp, ok := res.value.(SetPositioner); ok {
			sp.SetPosition(pos, s.mark())
		}
	}

	return s.check(m, res)
}

func (m *matchTransform) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchTransform) print() string {
	return m.rule.print()
}

// Transform returns a Rule that invokes it's given rule and if it matches
// calls the given function, passing the section of the input stream that
// was matched. The return value becomes the value of the rule.
//
// The value of the match is the return value of the given function.
func Transform(r Rule, fn func(string) interface{}) Rule {
	return &matchTransform{rule: r, fn: fn}
}

type matchCapture struct {
	rule Rule
}

func (m *matchCapture) match(s *state) result {
	pos := s.mark()

	res := m.rule.match(s)
	if res.matched {
		res.value = s.input[pos:s.mark()]
	}

	return s.check(m, res)
}

func (m *matchCapture) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchCapture) print() string {
	return fmt.Sprintf("< %s >", m.rule.print())
}

// Capture returns a Rule that attempts to match it's given rule. If it
// matches, the value of the rule is set to the section of the input stream
// that was matched. Said another way, Capture pulls the matched text up
// as a value.
//
// The value of the match is portion of the input stream that matched
// the sub-rule.
func Capture(r Rule) Rule {
	return &matchCapture{rule: r}
}

type matchCheckAction struct {
	rule Rule
	fn   func(vals Values) bool
}

func (m *matchCheckAction) match(s *state) result {
	defer s.restore(s.mark())

	if m.fn(s.values) {
		s.good(m)
		return result{matched: true}
	}

	s.bad(m)
	return result{}
}

func (m *matchCheckAction) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

// CheckAction returns a rule that when a match is attempted, calls the given
// function. If the function returns true, the match succeeds. This corresponds with
// check functions ("&{ a > 2}") style constructions in other PEGs. The Values
// argument provides access to the current scope.
//
// The value of the match is nil.
func CheckAction(fn func(Values) bool) Rule {
	return &matchCheckAction{fn: fn}
}

func (m *matchCheckAction) print() string {
	return "&<go-func>"
}

type matchEOS struct {
}

func (m *matchEOS) match(s *state) result {
	return s.check(m, result{matched: s.pos >= len(s.input)})
}

func (m *matchEOS) detectLeftRec(Rule, ruleSet) bool {
	return false
}

func (m *matchEOS) print() string {
	return "EOF"
}

// EOS produces a rule that only matches when the input stream
// has been exhausted.
//
// The value of the match is nil.
func EOS() Rule {
	return &matchEOS{}
}

// That's all the rules!

// Labels provides a simple database for named refs. This is used to cleanup rule
// set creation.
type Labels interface {
	// Ref creates or returns a Ref of the given name.
	Ref(name string) Rule

	// Set assigns the given rule to a Ref of the given name.
	Set(name string, rule Rule) Ref
}

// Refs returns a Labels value.
func Refs() Labels {
	return &labels{
		refs: make(map[string]Ref),
	}
}

type labels struct {
	refs map[string]Ref
}

func (l *labels) Ref(name string) Rule {
	if ref, ok := l.refs[name]; ok {
		return ref
	}

	ref := R(name)

	l.refs[name] = ref

	return ref
}

func (l *labels) Set(name string, rule Rule) Ref {
	if ref, ok := l.refs[name]; ok {
		ref.Set(rule)
		return ref
	}

	ref := R(name)

	l.refs[name] = ref

	ref.Set(rule)
	return ref
}

var ErrInputNotConsumed = errors.New("full input not consume")

type memoResult struct {
	result
	endPos int
	used   int
}

type state struct {
	p      *Parser
	input  string
	pos    int
	memos  map[int]map[Rule]*memoResult
	values valMap
	maxPos int
}

func (s *state) cur() string {
	return s.input[s.pos:]
}

func (s *state) curRune() string {
	if s.pos >= len(s.input) {
		return "EOF"
	} else {
		return s.input[s.pos : s.pos+1]
	}
}

func (s *state) advance(r Rule, l int) {
	start := s.pos

	s.pos += l

	if s.pos > s.maxPos {
		s.maxPos = s.pos
	}

	if s.p.debug {
		fmt.Printf("G @ %d-%d (%q) => %s\n", start, s.pos, s.input[start:s.pos], r.print())
	}
}

func (s *state) mark() int {
	return s.pos
}

func (s *state) restore(p int) {
	s.pos = p
}

func (s *state) good(r Rule) {
	if s.p.debug {
		fmt.Printf("G @ %d (%s) => %s\n", s.mark(), s.curRune(), r.print())
	}
}

func (s *state) bad(r Rule) {
	if s.p.debug {
		fmt.Printf("B @ %d (%s) => %s\n", s.mark(), s.curRune(), r.print())
	}
}

func (s *state) check(r Rule, res result) result {
	if res.matched {
		s.good(r)
	} else {
		s.bad(r)
	}

	return res
}

// Parser is the interface for running a rule against some input
type Parser struct {
	log   hclog.Logger
	debug bool
}

type Option func(p *Parser)

func WithDebug(on bool) Option {
	return func(p *Parser) {
		p.debug = on
	}
}

func WithLogger(log hclog.Logger) Option {
	return func(p *Parser) {
		p.log = log
	}
}

// New creates a new Parser value
func New(opts ...Option) *Parser {
	p := &Parser{
		log: hclog.L(),
	}

	for _, o := range opts {
		o(p)
	}

	return p
}

func (p *Parser) parse(r Rule, input string) (*state, result) {
	s := &state{
		p:      p,
		input:  input,
		memos:  make(map[int]map[Rule]*memoResult),
		values: make(valMap),
	}

	return s, r.match(s)
}

// Parse attempts to match the given rule against the input string. If
// the rule matches, the value of the rule is returned. If the rule matches
// a portion of input, the ErrInputNotConsumed error is returned.
func (p *Parser) Parse(r Rule, input string) (val interface{}, matched bool, err error) {
	s, res := p.parse(r, input)
	if !res.matched {
		return nil, false, nil
	}

	if s.pos != len(s.input) {
		return res.value, true, ErrInputNotConsumed
	}

	return res.value, true, nil
}

func Print(n Rule) string {
	return n.print()
}
