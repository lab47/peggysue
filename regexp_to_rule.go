package peggysue

import (
	"fmt"

	"github.com/lab47/peggysue/regexp"
)

// RegexpToRule converts a regexp AST to a peggysue Rule
func RegexpToRule(node regexp.Node) Rule {
	switch n := node.(type) {
	case *regexp.Pattern:
		return RegexpToRule(n.Node)

	case *regexp.Literal:
		// Single character literal
		return S(string(n.Char))

	case *regexp.Dot:
		// Match any character
		return Any()

	case *regexp.Sequence:
		// Sequence of rules
		if len(n.Elements) == 0 {
			// Empty sequence - match empty string
			return S("")
		}
		if len(n.Elements) == 1 {
			return RegexpToRule(n.Elements[0])
		}

		// Special handling for sequences with anchors
		// If we have a $ anchor at the end, we need to handle it specially
		rules := make([]Rule, 0, len(n.Elements))
		hasEndAnchor := false

		for i, elem := range n.Elements {
			if anchor, ok := elem.(*regexp.Anchor); ok && anchor.Type == regexp.AnchorEnd && i == len(n.Elements)-1 {
				hasEndAnchor = true
				continue
			}
			rules = append(rules, RegexpToRule(elem))
		}

		var seqRule Rule
		if len(rules) == 1 {
			seqRule = rules[0]
		} else {
			seqRule = Seq(rules...)
		}

		if hasEndAnchor {
			// Wrap in a rule that checks we're at the end after matching
			return &endOfInputRule{inner: seqRule}
		}

		return seqRule

	case *regexp.Alternation:
		// Choice between options
		if len(n.Options) == 0 {
			return S("")
		}
		if len(n.Options) == 1 {
			return RegexpToRule(n.Options[0])
		}
		rules := make([]Rule, len(n.Options))
		for i, opt := range n.Options {
			rules[i] = RegexpToRule(opt)
		}
		return Or(rules...)

	case *regexp.CharClass:
		// Character class
		if len(n.Elements) == 0 {
			// Empty character class matches nothing
			return Not(Any())
		}

		// Optimize character classes using PrefixTable for better performance
		var singleChars []rune
		var rules []Rule
		var prefixEntries []any

		for _, elem := range n.Elements {
			switch e := elem.(type) {
			case regexp.CharRange:
				if e.Start == e.End {
					// Single character - candidate for PrefixTable
					singleChars = append(singleChars, e.Start)
				} else {
					// Range - keep as separate rule
					rules = append(rules, Range(e.Start, e.End))
				}
			case *regexp.EscapeSequence:
				// For now, keep escape sequences as separate rules
				// TODO: Could optimize \d, \w, etc. with custom predicates
				rules = append(rules, escapeToRule(e))
			}
		}

		// Build the final rule
		var classRule Rule

		// Use PrefixTable optimization if we have single characters
		// This is more efficient than Set() for character sets with ASCII chars
		if len(singleChars) >= 3 {
			// Build PrefixTable entries for single characters
			for _, ch := range singleChars {
				if ch <= 127 { // ASCII only for PrefixTable
					prefixEntries = append(prefixEntries, byte(ch), Byte(byte(ch)))
				} else {
					// Non-ASCII single chars fall back to rules
					rules = append(rules, S(string(ch)))
				}
			}

			if len(prefixEntries) > 0 {
				if len(rules) == 0 {
					// Pure PrefixTable case
					classRule = PrefixTable(prefixEntries...)
				} else {
					// Mixed case: PrefixTable + other rules
					prefixRule := PrefixTable(prefixEntries...)
					rules = append([]Rule{prefixRule}, rules...)
					classRule = Or(rules...)
				}
			} else {
				// No ASCII chars, use Set fallback
				if len(singleChars) > 0 {
					rules = append([]Rule{Set(singleChars...)}, rules...)
				}
				if len(rules) == 1 {
					classRule = rules[0]
				} else {
					classRule = Or(rules...)
				}
			}
		} else {
			// Use original Set() optimization for smaller sets or many ranges
			if len(singleChars) > 0 {
				rules = append([]Rule{Set(singleChars...)}, rules...)
			}

			if len(rules) == 1 {
				classRule = rules[0]
			} else {
				classRule = Or(rules...)
			}
		}

		if n.Negated {
			// Negated character class
			return Seq(Not(classRule), Any())
		}
		return classRule

	case *regexp.Quantifier:
		// Repetition
		baseRule := RegexpToRule(n.Expression)

		if n.Min == 0 && n.Max == -1 {
			// * quantifier
			return Star(baseRule)
		} else if n.Min == 1 && n.Max == -1 {
			// + quantifier
			return Plus(baseRule)
		} else if n.Min == 0 && n.Max == 1 {
			// ? quantifier
			return Maybe(baseRule)
		} else {
			// General {min,max} quantifier
			return Many(baseRule, n.Min, n.Max, nil)
		}

	case *regexp.Group:
		// Grouped expression - just return the inner rule
		// (capturing behavior would need to be handled separately)
		return RegexpToRule(n.Expression)

	case *regexp.Anchor:
		// Anchors need special handling
		switch n.Type {
		case regexp.AnchorStart:
			// ^ - match at start of input
			// This is tricky in PEG - we'll use a special rule
			return &anchorStartRule{}
		case regexp.AnchorEnd:
			// $ - match at end of input
			return &anchorEndRule{}
		default:
			// Other anchors not supported yet
			return Not(Any()) // Never matches
		}

	case *regexp.EscapeSequence:
		return escapeToRule(n)

	default:
		panic(fmt.Sprintf("unknown regexp node type: %T", node))
	}
}

// escapeToRule converts an escape sequence to a rule
func escapeToRule(esc *regexp.EscapeSequence) Rule {
	switch esc.Type {
	case regexp.EscapeChar:
		// Character escape
		if len(esc.Value) > 0 {
			return S(esc.Value)
		}
		return S("\\")

	case regexp.EscapeDigit:
		// \d - digit
		return Range('0', '9')

	case regexp.EscapeNonDigit:
		// \D - non-digit
		return Seq(Not(Range('0', '9')), Any())

	case regexp.EscapeSpace:
		// \s - whitespace
		return Set(' ', '\t', '\n', '\r', '\f')

	case regexp.EscapeNonSpace:
		// \S - non-whitespace
		return Seq(Not(Set(' ', '\t', '\n', '\r', '\f')), Any())

	case regexp.EscapeWord:
		// \w - word character [a-zA-Z0-9_]
		return Or(
			Range('a', 'z'),
			Range('A', 'Z'),
			Range('0', '9'),
			S("_"),
		)

	case regexp.EscapeNonWord:
		// \W - non-word character
		wordChar := Or(
			Range('a', 'z'),
			Range('A', 'Z'),
			Range('0', '9'),
			S("_"),
		)
		return Seq(Not(wordChar), Any())

	default:
		// Other escape types not supported yet
		return Not(Any()) // Never matches
	}
}

// Special rule types for anchors
type anchorStartRule struct {
	basicRule
}

func (r *anchorStartRule) match(s *state) result {
	if s.pos == 0 {
		return result{matched: true, value: nil}
	}
	return result{matched: false}
}

func (r *anchorStartRule) detectLeftRec(self Rule, rs ruleSet) bool {
	return false
}

func (r *anchorStartRule) print() string {
	return "^"
}

type anchorEndRule struct {
	basicRule
}

func (r *anchorEndRule) match(s *state) result {
	if s.pos == s.inputSize {
		return result{matched: true, value: nil}
	}
	return result{matched: false}
}

func (r *anchorEndRule) detectLeftRec(self Rule, rs ruleSet) bool {
	return false
}

func (r *anchorEndRule) print() string {
	return "$"
}

// endOfInputRule matches a rule and then checks we're at end of input
type endOfInputRule struct {
	basicRule
	inner Rule
}

func (r *endOfInputRule) match(s *state) result {
	res := r.inner.match(s)
	if !res.matched {
		return res
	}

	// Check if we're at the end of input
	if s.pos == s.inputSize {
		return res
	}

	return result{matched: false}
}

func (r *endOfInputRule) detectLeftRec(self Rule, rs ruleSet) bool {
	return r.inner.detectLeftRec(self, rs)
}

func (r *endOfInputRule) print() string {
	return r.inner.print() + "$"
}

// Regexp creates a rule from a regexp pattern string using our own regexp parser.
// This is an alternative to the Re() function that uses Go's regexp package.
//
// The regexp syntax supported includes:
//   - Literals: abc
//   - Character classes: [abc], [a-z], [^abc]
//   - Escape sequences: \n, \t, \d, \s, \w, etc.
//   - Quantifiers: *, +, ?, {n}, {n,m}, {n,}
//   - Lazy quantifiers: *?, +?, ??
//   - Alternation: cat|dog
//   - Groups: (abc), (?:abc), (?P<name>abc)
//   - Anchors: ^, $
//   - Dot: . (matches any character)
//
// Note: Unlike Go's regexp, this implementation:
//   - Does not search for matches; it only matches at the current position
//   - Anchors (^ and $) work within the PEG parsing model
//   - May have different performance characteristics
//   - Quantifiers are greedy and don't backtrack (PEG behavior)
//     For example, [.]+\. won't match "...." because the + consumes all dots
//
// The matched text is captured and returned as the rule's value.
func Regexp(pattern string) Rule {
	ast, err := ParseRegexp(pattern)
	if err != nil {
		panic(fmt.Sprintf("invalid regexp pattern: %v", err))
	}
	// Wrap the rule in Capture to return the matched text
	return Capture(RegexpToRule(ast))
}
