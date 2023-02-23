package peggysue

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/exp/slices"
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

	Name() string
	SetName(name string)
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

type basicRule struct {
	name string
}

func (b *basicRule) Name() string {
	return b.name
}

func (b *basicRule) SetName(name string) {
	b.name = name
}

// N sets the name of the given rule and returns it.
func N(name string, r Rule) Rule {
	r.SetName(name)
	return r
}

type matchAny struct {
	basicRule
}

func (m *matchAny) match(s *state) result {
	pos := s.pos
	if pos >= s.inputSize {
		return result{}
	}

	b := s.input[pos]

	var sz int

	if b < utf8.RuneSelf {
		sz = 1
	} else {
		_, sz = utf8.DecodeRuneInString(s.cur())
	}

	s.goodRange(m, sz)
	s.advance(sz, m)

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
	basicRule
	str string
}

func (m *matchString) match(s *state) result {
	sz := len(m.str)
	if sz > len(s.cur()) {
		s.bad(m)
		return result{}
	}

	if strings.HasPrefix(s.cur(), m.str) {
		s.goodRange(m, sz)
		s.advance(sz, m)
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
	switch len(str) {
	case 1:
		return &matchString1{b: str[0]}
	case 2:
		return &matchString2{a: str[0], b: str[1]}
	default:
		return &matchString{str: str}
	}
}

type matchRegexp struct {
	basicRule
	re  *regexp.Regexp
	str string
}

func (m *matchRegexp) match(s *state) result {
	loc := m.re.FindStringIndex(s.cur())
	if loc == nil {
		s.bad(m)
		return result{}
	}

	s.goodRange(m, loc[1])
	s.advance(loc[1], m)
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
	basicRule
	start, end rune
}

func (m *matchCharRange) match(s *state) result {
	pos := s.pos
	if pos >= s.inputSize {
		s.bad(m)
		return result{}
	}

	b := s.input[pos]

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

	if rn < m.start || rn > m.end {
		s.bad(m)
		return result{}
	}

	s.goodRange(m, sz)
	s.advance(sz, m)
	return result{matched: true}
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

type matchCharSet struct {
	basicRule
	set []rune
}

func (m *matchCharSet) match(s *state) result {
	pos := s.pos
	if pos >= s.inputSize {
		s.bad(m)
		return result{}
	}

	b := s.input[pos]

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

	for _, mr := range m.set {
		if rn == mr {
			s.good(m)
			s.advance(sz, m)
			return result{matched: true}
		}
	}

	s.bad(m)
	return result{}
}

func (m *matchCharSet) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchCharSet) print() string {
	var strs []string

	for _, r := range m.set {
		strs = append(strs, fmt.Sprintf("%q", r))

	}
	return "{" + strings.Join(strs, ",") + "}"
}

// Set returns a rule that will match the next rune in the input
// stream as one of the given runes. This corresponds
// with the regexp pattern `[abc]` but is much faster as it does not require
// any regexp tracking.
//
// The value of the match is nil.
func Set(runes ...rune) Rule {
	return &matchCharSet{
		set: runes,
	}
}

type matchRunePredicate struct {
	basicRule
	fn func(r rune) bool
}

func (m *matchRunePredicate) match(s *state) result {
	pos := s.pos
	if pos >= s.inputSize {
		s.bad(m)
		return result{}
	}

	b := s.input[pos]

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

	if m.fn(rn) {
		s.good(m)
		s.advance(sz, m)
		return result{matched: true}
	}

	return result{}
}

func (m *matchRunePredicate) detectLeftRec(r Rule, rs ruleSet) bool {
	return false
}

func (m *matchRunePredicate) print() string {
	return ". &{ /* check rune */ }"
}

// Rune returns a rule that attempts to match the next rune in the
// input stream againnt a Go function. These are useful for be able
// utilizing existing logic to match rules (such as logic in the
// unicode package)
func Rune(fn func(rune) bool) Rule {
	return &matchRunePredicate{
		fn: fn,
	}
}

type matchOr struct {
	basicRule
	rules []Rule
}

func (m *matchOr) match(s *state) result {
	save := s.mark()

	for _, r := range m.rules {
		res := s.match(r)
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
		subs = append(subs, Print(r))
	}
	return strings.Join(subs, " | ")
}

// Or returns a Rule that will try each of the given rules, completing when
// the first one successfully matches. This corresponds with a PEG's "ordered
// choice" operation.
//
// The value of the match is the value of the sub-rule that matched correctly.
func Or(rules ...Rule) Rule {
	switch len(rules) {
	case 1:
		return rules[0]
	case 2:
		return &matchEither{a: rules[0], b: rules[1]}
	}
	return &matchOr{rules: rules}
}

type branch struct {
	name string
	r    Rule
}

type matchBranch struct {
	basicRule
	rules []branch
	ref   Ref
}

func (m *matchBranch) match(s *state) result {
	save := s.mark()

	rs := s.refStack

	defer func() {
		s.refStack = rs
	}()

	for _, r := range m.rules {
		s.refStack = append(rs, r.name)

		res := s.match(r.r)
		if res.matched {
			s.good(m)
			return res
		}

		s.restore(save)
	}

	s.bad(m)
	return result{}
}

func (m *matchBranch) detectLeftRec(r Rule, rs ruleSet) bool {
	for _, sub := range m.rules {
		if !rs.Add(sub.r) {
			return false
		}

		if r == sub.r {
			return true
		}

		if sub.r.detectLeftRec(r, rs) {
			return true
		}
	}

	return false
}

func (m *matchBranch) print() string {
	var subs []string

	for _, r := range m.rules {
		subs = append(subs, r.name)
	}
	return strings.Join(subs, " | ")
}

func (m *matchBranch) Add(name string, r Rule) Rule {
	m.rules = append(m.rules, branch{"+" + name, r})

	return r
}

func (m *matchBranch) Ref() Ref {
	return m.ref
}

type BranchesBuilder interface {
	// Add another branch
	Add(name string, branch Rule) Rule
}

// Or returns a Rule that will try each of the given rules, completing when
// the first one successfully matches. This corresponds with a PEG's "ordered
// choice" operation.
//
// The value of the match is the value of the sub-rule that matched correctly.
func Branches(name string, f func(b BranchesBuilder, r Rule)) Rule {
	mb := &matchBranch{
		ref: R(name),
	}

	f(mb, mb.ref)

	mb.ref.Set(mb)

	return mb.ref
}

type matchSeq struct {
	basicRule
	rules []Rule
}

func (m *matchSeq) match(s *state) result {
	var ret result

	mark := s.mark()

	for _, r := range m.rules {
		res := s.match(r)
		if !res.matched {
			s.restore(mark)
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
		subs = append(subs, Print(r))
	}
	return strings.Join(subs, " ")
}

// Seq returns a rule that will attempt to match each of the given rules
// in order. It only matches successfully if each of it's rules match.
//
// The value of the match is the value of the right most sub-rule that
// produced a non-nil value.
func Seq(rules ...Rule) Rule {
	switch len(rules) {
	case 1:
		return rules[0]
	case 2:
		return &matchBoth{a: rules[0], b: rules[1]}
	case 3:
		return &matchThree{a: rules[0], b: rules[1], c: rules[2]}
	default:
		return &matchSeq{rules: rules}
	}
}

type matchZeroOrMore struct {
	basicRule
	rule Rule
}

func (m *matchZeroOrMore) match(s *state) result {

	for {
		mark := s.mark()

		res := s.match(m.rule)
		if res.matched {
			continue
		}

		s.restore(mark)

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
	basicRule
	rule Rule
}

func (m *matchOneOrMore) match(s *state) result {
	mark := s.mark()

	res := s.match(m.rule)
	if !res.matched {
		s.restore(mark)
		return result{}
	}

	val := res.value

	for {
		mark := s.mark()

		res := s.match(m.rule)
		if res.matched {
			val = res.value
			continue
		}

		s.restore(mark)

		return s.check(m, result{value: val, matched: true})
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
		return "(" + Print(r) + ")"
	default:
		return Print(r)
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

type matchMany struct {
	basicRule
	min, max int
	rule     Rule
	fn       func([]interface{}) interface{}
}

var manyResultsPool = sync.Pool{
	New: func() interface{} {
		val := make([]interface{}, 0, 10)
		return &val
	},
}

func (m *matchMany) match(s *state) result {
	pv := manyResultsPool.Get().(*[]interface{})
	defer manyResultsPool.Put(pv)

	results := (*pv)[:0]

	top := s.mark()

	for {
		mark := s.mark()

		res := s.match(m.rule)
		if !res.matched {
			s.restore(mark)
			break

		}

		results = append(results, res.value)

		if m.max != -1 && len(results) == m.max {
			break
		}
	}

	if len(results) < m.min {
		s.restore(top)
		s.bad(m)
		return result{}
	}

	var val interface{}

	if m.fn != nil {
		val = m.fn(results)
	}

	s.good(m)
	return result{value: val, matched: true}
}

func (m *matchMany) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchMany) print() string {
	if m.min == 0 && m.max == -1 {
		return addParens(m.rule) + "*"
	}
	if m.min == 1 && m.max == -1 {
		return addParens(m.rule) + "+"
	}

	return fmt.Sprintf("%s[%d,%d]", addParens(m.rule), m.min, m.max)
}

// Many returns a rule that will attempt to match it's given rule
// at least `min` times, and at most `max` times. If max is -1, there is no
// maximum. Upon matching results, if `fn` is not nil, it wil be called and
// passed the results of each sub-rule value. If `fn` is nil, the result
// is the nil.
//
// Important note: the results slice is reused to reduce allocations. The results
// should be copied out of the results slice if needed.
//
// This is a general purpose form of Star and Plus that adds the ability
// to process the resulting values as a slice.
//
// The value of the return value of `fn` or, if `fn` is nil, the slice
// the sub-rule values.
func Many(rule Rule, min, max int, fn func(values []interface{}) interface{}) Rule {
	return &matchMany{rule: rule, min: min, max: max, fn: fn}
}

func copyGroup(values []interface{}) interface{} {
	return slices.Clone(values)
}

func PlusCapture(rule Rule) Rule {
	return Many(rule, 1, -1, copyGroup)
}

type matchOptional struct {
	basicRule
	rule Rule
}

func (m *matchOptional) match(s *state) result {
	mark := s.mark()

	res := s.match(m.rule)

	if !res.matched {
		s.restore(mark)
	}

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
	return Print(m.rule) + "?"
}

// Maybe returns a rule that will allow it's rule to match, but
// will always return that it's successfully matched, regardless
// of what it's rule does. This corresponds with the question mark
// rule ("e?") in most PEGs.
//
// The value of the match is the value of the sub-rule.
func Maybe(rule Rule) Rule {
	return &matchOptional{rule: rule}
}

type matchCheck struct {
	basicRule
	rule Rule
}

func (m *matchCheck) match(s *state) result {
	defer s.restore(s.mark())

	return s.check(m, s.match(m.rule))
}

func (m *matchCheck) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchCheck) print() string {
	return "&" + Print(m.rule)
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
	basicRule
	rule Rule
}

func (m *matchNot) match(s *state) result {
	defer s.restore(s.mark())

	if s.pos >= len(s.input) {
		return result{}
	}

	res := s.match(m.rule)
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
	return "!" + Print(m.rule)
}

// Not returns a rule that will attempt to match it's given rule.
// If the rule matches, it returns that it did not match (inverting
// the match result). Regardless of matching, it does not consume any
// input on the stream. This corresponds with the not-predicate ("!e") in
// most PEGs.
//
// The value of the match is the value of the sub-rule.
func Not(rule Rule) Rule {
	if ms, ok := rule.(*matchString1); ok {
		return &matchNotByte{b: ms.b}
	}
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
	basicRule
	name    string
	rule    Rule
	leftRec bool
}

func (r *matchRef) Set(rule Rule) {
	// Fail because this is almost always a programmer mistake
	// where they used the same name twice on accident.
	if r.rule != nil {
		panic(fmt.Sprintf("rule already set: %s", r.name))
	}

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
	if m.rule == nil {
		panic(fmt.Sprintf("unset ref detected: %s", m.name))
	}

	cur := s.curRef
	defer func() {
		s.curRef = cur
	}()

	s.curRef = m

	// The memoization code was ported from
	// https://github.com/we-like-parsers/pegen_experiments/blob/master/story7/memo.py

	if s.memos == nil {
		s.memos = make(map[int]map[Rule]*memoResult)
	}

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

			res := s.match(m.rule)
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

		res := s.match(m.rule)
		endPos := s.mark()

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
	return &matchRef{
		basicRule: basicRule{
			name: name,
		},
	}
}

func SetRef(name string, rule Rule) Ref {
	r := R(name)
	r.Set(rule)
	return r
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

	set(name string, val interface{}) bool
}

type cvEntry struct {
	name string
	val  interface{}
}

type compactedValues struct {
	used    int
	entries [5]cvEntry
}

func (v *compactedValues) set(name string, val interface{}) bool {
	idx := v.used
	if idx >= len(v.entries) {
		return false
	}

	v.used = idx + 1

	v.entries[idx].name = name
	v.entries[idx].val = val

	return true
}

func (v *compactedValues) Get(name string) interface{} {
	for i := 0; i < v.used; i++ {
		if v.entries[i].name == name {
			return v.entries[i].val
		}
	}

	return nil
}

var cvPool = sync.Pool{
	New: func() interface{} {
		return &compactedValues{}
	},
}

func returnValues(v Values) {
	if cv, ok := v.(*compactedValues); ok {
		cv.used = 0
		cvPool.Put(cv)
	}
}

type valMap map[string]interface{}

func (m valMap) Get(name string) interface{} {
	return m[name]
}

func (m valMap) set(name string, val interface{}) bool {
	m[name] = val
	return true
}

type matchAction struct {
	basicRule
	rule Rule
	fn   func(Values) interface{}
}

func (m *matchAction) match(s *state) result {
	pos := s.mark()

	res := s.match(m.rule)
	if res.matched {
		res.value = m.fn(s.values)

		if sp, ok := res.value.(SetPositioner); ok {
			sp.SetPosition(pos, s.mark())
		}
	} else {
		s.restore(pos)
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
	return Print(m.rule)
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
	basicRule
	rule Rule
	typ  reflect.Type
}

func (m *matchApply) match(s *state) result {
	pos := s.mark()

	res := s.match(m.rule)
	if res.matched {
		res.value = m.expand(s)
	} else {
		s.restore(pos)
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
	return Print(m.rule)
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
//
//	type Node struct {
//	   Age int `ast:"age"`
//	}
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

	return &matchScope{
		rule: &matchApply{
			rule: rule,
			typ:  rv.Type(),
		},
	}
}

type matchScope struct {
	basicRule
	rule Rule
}

func (m *matchScope) match(s *state) result {
	curValues := s.values
	defer func() {
		returnValues(s.values)
		s.values = curValues
	}()

	v := cvPool.Get().(*compactedValues)
	s.values = v

	return s.check(m, s.match(m.rule))
}

func (m *matchScope) detectLeftRec(r Rule, rs ruleSet) bool {
	if !rs.Add(m.rule) {
		return false
	}

	return m.rule == r || m.rule.detectLeftRec(r, rs)
}

func (m *matchScope) print() string {
	return Print(m.rule)
}

// Scope introduces a new rule scope. This is generally not needed as
// most users will use the automatically created scope from an Action rule.
//
// The value of the match is the value of the sub-rule.
func Scope(rule Rule) Rule {
	return &matchScope{rule: rule}
}

type matchNamed struct {
	basicRule
	name string
	rule Rule
}

func (m *matchNamed) match(s *state) result {
	res := s.match(m.rule)
	if res.matched {
		if s.p.debug {
			fmt.Printf("N (%p) %s => %#v\n", s.values, m.name, res.value)
		}
		if !s.values.set(m.name, res.value) {
			vm := make(valMap)
			for _, ent := range s.values.(*compactedValues).entries {
				vm.set(ent.name, ent.val)
			}

			vm.set(m.name, res.value)

			s.values = vm
		}
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
	return fmt.Sprintf("%s:%s", Print(m.rule), m.name)
}

// Named will assign the value of the given rule to the current rule scope
// if it matches.
//
// The value of the match is the value of the sub-rule.
func Named(name string, rule Rule) Rule {
	return &matchNamed{name: name, rule: rule}
}

type matchTransform struct {
	basicRule
	rule Rule
	fn   func(str string) interface{}
}

func (m *matchTransform) match(s *state) result {
	pos := s.mark()

	res := s.match(m.rule)
	if res.matched {
		res.value = m.fn(s.input[pos:s.mark()])

		if sp, ok := res.value.(SetPositioner); ok {
			sp.SetPosition(pos, s.mark())
		}
	} else {
		s.restore(pos)
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
	return Print(m.rule)
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
	basicRule
	rule Rule
}

func (m *matchCapture) match(s *state) result {
	pos := s.mark()

	res := s.match(m.rule)
	if res.matched {
		res.value = s.input[pos:s.mark()]
	} else {
		s.restore(pos)
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
	return fmt.Sprintf("< %s >", Print(m.rule))
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
	basicRule
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
	basicRule
}

func (m *matchEOS) match(s *state) result {
	return s.check(m, result{matched: s.pos >= s.inputSize})
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

type ErrInputNotConsumed struct {
	MaxPos  int
	MaxRule Rule
}

func (*ErrInputNotConsumed) Error() string {
	return "full input not consume"
}

type memoResult struct {
	result
	endPos int
	used   int
}

type state struct {
	p         *Parser
	input     string
	inputSize int
	pos       int
	memos     map[int]map[Rule]*memoResult
	values    Values

	curRef  Ref
	maxPos  int
	maxRule Rule

	debug     bool
	refStack  []string
	check     func(r Rule, res result) result
	good      func(r Rule)
	goodRange func(r Rule, sz int)
	bad       func(r Rule)

	match func(r Rule) result
}

func (s *state) matchFast(r Rule) result {
	return r.match(s)
}

func (s *state) matchDebug(r Rule) result {
	n := r.Name()
	if n == "" {
		return r.match(s)
	}

	rs := s.refStack
	s.refStack = append(s.refStack, n)

	fmt.Printf("R -- %s\n", strings.Join(s.refStack, ", "))

	res := r.match(s)

	s.refStack = rs

	if res.matched {
		fmt.Printf("  + G -- %s\n", strings.Join(s.refStack, ", "))
	} else {
		fmt.Printf("  - B -- %s\n", strings.Join(s.refStack, ", "))
	}

	return res
}

func (s *state) cur() string {
	return s.input[s.pos:]
}

func (s *state) curRune() string {
	if s.pos >= s.inputSize {
		return "EOF"
	} else {
		return s.input[s.pos : s.pos+1]
	}
}

func (s *state) advance(l int, r Rule) {
	s.pos += l

	if s.pos > s.maxPos {
		s.maxPos = s.pos
		s.maxRule = s.curRef
	}
}

func (s *state) mark() int {
	return s.pos
}

func (s *state) restore(p int) {
	s.pos = p
}

func (s *state) goodRangeDebug(r Rule, sz int) {
	fmt.Printf("G @ %d-%d (%q) => %s\n", s.pos, s.pos+sz, s.input[s.pos:s.pos+sz], Print(r))
}

func goodRangeId(r Rule, sz int) {}

func (s *state) goodDebug(r Rule) {
	fmt.Printf("G @ %d (%q) => %s\n", s.mark(), s.curRune(), Print(r))
}

func goodId(r Rule) {}

func (s *state) badDebug(r Rule) {
	fmt.Printf("B @ %d (%q) => %s\n", s.mark(), s.curRune(), Print(r))
}

func badId(r Rule) {}

func (s *state) checkDebug(r Rule, res result) result {
	if res.matched {
		s.good(r)
	} else {
		s.bad(r)
	}

	return res
}

func checkId(r Rule, res result) result {
	return res
}

// Parser is the interface for running a rule against some input
type Parser struct {
	log     hclog.Logger
	partial bool
	debug   bool
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

func WithPartial(on bool) Option {
	return func(p *Parser) {
		p.partial = on
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
		p:         p,
		input:     input,
		inputSize: len(input),
		values:    cvPool.Get().(Values),
		debug:     p.debug,
	}

	if p.debug {
		s.check = s.checkDebug
		s.good = s.goodDebug
		s.goodRange = s.goodRangeDebug
		s.bad = s.badDebug
		s.match = s.matchDebug
	} else {
		s.check = checkId
		s.good = goodId
		s.goodRange = goodRangeId
		s.bad = badId
		s.match = s.matchFast
	}

	defer returnValues(s.values)

	return s, s.match(r)
}

// Parse attempts to match the given rule against the input string. If
// the rule matches, the value of the rule is returned. If the rule matches
// a portion of input, the ErrInputNotConsumed error is returned.
func (p *Parser) Parse(r Rule, input string) (val interface{}, matched bool, err error) {
	s, res := p.parse(r, input)
	if !res.matched {
		return nil, false, nil
	}

	if !p.partial {
		if s.pos != s.inputSize {
			return res.value, false, &ErrInputNotConsumed{
				MaxPos:  s.maxPos,
				MaxRule: s.maxRule,
			}
		}
	}

	return res.value, true, nil
}

// Print outputs either the rules name (if it has one) or a description
// of it's operations.
func Print(n Rule) string {
	if n.Name() != "" {
		return n.Name()
	}

	return n.print()
}

// Repr outputs a description of the rules operations.
func Repr(n Rule) string {
	return n.print()
}
