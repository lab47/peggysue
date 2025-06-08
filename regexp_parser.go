package peggysue

import (
	"fmt"
	"strconv"
)

// RegexpParser creates a parser for Go regexp syntax
func RegexpParser() Rule {
	// Create references for recursive rules
	regexp := R("regexp")
	alternation := R("alternation")
	sequence := R("sequence")
	atom := R("atom")
	quantifier := R("quantifier")
	group := R("group")
	charClass := R("charClass")
	charClassContent := R("charClassContent")
	escape := R("escape")
	
	// Helper rules
	ws := Star(Set(' ', '\t', '\n', '\r'))
	digit := Range('0', '9')
	hexDigit := Or(digit, Range('a', 'f'), Range('A', 'F'))
	
	// Number parser for quantifiers
	number := Transform(Plus(digit), func(s string) any {
		n, _ := strconv.Atoi(s)
		return n
	})
	
	// Escape sequences
	simpleEscape := Transform(Seq(
		S("\\"),
		Set('n', 't', 'r', 'f', 'v', 'a', 'b', '\\', '.', '^', '$', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|'),
	), func(s string) any {
		if len(s) < 2 {
			return &EscapeSequence{Type: EscapeChar, Value: "\\"}
		}
		ch := s[1]
		var value string
		switch ch {
		case 'n':
			value = "\n"
		case 't':
			value = "\t"
		case 'r':
			value = "\r"
		case 'f':
			value = "\f"
		case 'v':
			value = "\v"
		case 'a':
			value = "\a"
		case 'b':
			value = "\b"
		default:
			value = string(ch)
		}
		return &EscapeSequence{Type: EscapeChar, Value: value}
	})
	
	hexEscape := Action(Seq(
		S("\\x"),
		Named("hex", Transform(Count(hexDigit, 2), func(s string) any { return s })),
	), func(v Values) any {
		return &EscapeSequence{Type: EscapeHex, Value: v.Get("hex").(string)}
	})
	
	unicodeEscape := Action(Seq(
		S("\\u"),
		Named("hex", Transform(Count(hexDigit, 4), func(s string) any { return s })),
	), func(v Values) any {
		return &EscapeSequence{Type: EscapeUnicode, Value: v.Get("hex").(string)}
	})
	
	unicodeLongEscape := Action(Seq(
		S("\\U"),
		Named("hex", Transform(Count(hexDigit, 8), func(s string) any { return s })),
	), func(v Values) any {
		return &EscapeSequence{Type: EscapeUnicodeLong, Value: v.Get("hex").(string)}
	})
	
	// Character class escapes
	charClassEscape := Transform(Seq(
		S("\\"),
		Set('d', 'D', 's', 'S', 'w', 'W'),
	), func(s string) any {
		if len(s) < 2 {
			return &EscapeSequence{Type: EscapeChar, Value: "\\"}
		}
		t := s[1]
		var escType EscapeType
		switch t {
		case 'd':
			escType = EscapeDigit
		case 'D':
			escType = EscapeNonDigit
		case 's':
			escType = EscapeSpace
		case 'S':
			escType = EscapeNonSpace
		case 'w':
			escType = EscapeWord
		case 'W':
			escType = EscapeNonWord
		}
		return &EscapeSequence{Type: escType}
	})
	
	// Unicode property escapes
	propertyName := Transform(Plus(Or(
		Range('a', 'z'),
		Range('A', 'Z'),
		S("_"),
	)), func(s string) any { return s })
	
	propertyEscape := Action(Seq(
		S("\\"),
		Named("type", Set('p', 'P')),
		S("{"),
		Named("prop", propertyName),
		S("}"),
	), func(v Values) any {
		t := v.Get("type").(string)[0]
		prop := v.Get("prop").(string)
		if t == 'P' {
			return &EscapeSequence{Type: EscapeNonProperty, Value: prop}
		}
		return &EscapeSequence{Type: EscapeProperty, Value: prop}
	})
	
	// All escape types
	escape.Set(Or(
		simpleEscape,
		hexEscape,
		unicodeLongEscape,
		unicodeEscape,
		charClassEscape,
		propertyEscape,
	))
	
	// Anchors
	anchor := Or(
		Action(S("^"), func(v Values) any { return &Anchor{Type: AnchorStart} }),
		Action(S("$"), func(v Values) any { return &Anchor{Type: AnchorEnd} }),
		Action(S("\\A"), func(v Values) any { return &Anchor{Type: AnchorStartText} }),
		Action(S("\\z"), func(v Values) any { return &Anchor{Type: AnchorEndText} }),
		Action(S("\\b"), func(v Values) any { return &Anchor{Type: AnchorWordBoundary} }),
		Action(S("\\B"), func(v Values) any { return &Anchor{Type: AnchorNonWordBoundary} }),
	)
	
	// Character class content - can be either a character range or an escape sequence
	charClassElement := Action(Seq(
		Named("start", Or(
			escape,
			Transform(Seq(Not(Set(']', '\\')), Any()), func(s string) any {
				if len(s) > 0 {
					return &Literal{Char: rune(s[0])}
				}
				return &Literal{Char: ' '}
			}),
		)),
		Maybe(Seq(
			S("-"),
			Named("end", Or(
				escape,
				Transform(Seq(Not(Set(']', '\\')), Any()), func(s string) any {
					if len(s) > 0 {
						return &Literal{Char: rune(s[0])}
					}
					return &Literal{Char: ' '}
				}),
			)),
		)),
	), func(v Values) any {
		start := v.Get("start")
		end := v.Get("end")
		
		// If it's a non-char escape sequence without a range, return it as-is
		if esc, ok := start.(*EscapeSequence); ok && esc.Type != EscapeChar && end == nil {
			return esc
		}
		
		var startRune, endRune rune
		
		// Extract rune from start
		switch s := start.(type) {
		case *Literal:
			startRune = s.Char
		case *EscapeSequence:
			if s.Type == EscapeChar && len(s.Value) > 0 {
				startRune = rune(s.Value[0])
			} else {
				// For non-literal escapes in a range, this is invalid
				// but we'll handle it as a single character
				return s
			}
		}
		
		if end == nil {
			return CharRange{Start: startRune, End: startRune}
		}
		
		// Extract rune from end
		switch e := end.(type) {
		case *Literal:
			endRune = e.Char
		case *EscapeSequence:
			if e.Type == EscapeChar && len(e.Value) > 0 {
				endRune = rune(e.Value[0])
			}
		}
		
		return CharRange{Start: startRune, End: endRune}
	})
	
	charClassContent.Set(Many(charClassElement, 0, -1, func(vals []any) any {
		elements := make([]any, 0, len(vals))
		for _, v := range vals {
			elements = append(elements, v)
		}
		return elements
	}))
	
	// Character class
	charClass.Set(Action(Seq(
		S("["),
		Named("neg", Capture(Maybe(S("^")))),
		Named("content", charClassContent),
		S("]"),
	), func(v Values) any {
		negated := v.Get("neg").(string) == "^"
		elements := v.Get("content").([]any)
		return &CharClass{Negated: negated, Elements: elements}
	}))
	
	// Grouping
	namedGroup := Action(Seq(
		S("(?P<"),
		Named("name", Transform(Plus(Or(
			Range('a', 'z'),
			Range('A', 'Z'),
			Range('0', '9'),
			S("_"),
		)), func(s string) any { return s })),
		S(">"),
		Named("expr", regexp),
		S(")"),
	), func(v Values) any {
		return &Group{
			Expression: v.Get("expr").(RegexpNode),
			Capturing:  true,
			Name:       v.Get("name").(string),
		}
	})
	
	nonCapturingGroup := Action(Seq(
		S("(?:"),
		Named("expr", regexp),
		S(")"),
	), func(v Values) any {
		return &Group{
			Expression:   v.Get("expr").(RegexpNode),
			NonCapturing: true,
		}
	})
	
	capturingGroup := Action(Seq(
		S("("),
		Not(S("?")),
		Named("expr", regexp),
		S(")"),
	), func(v Values) any {
		return &Group{
			Expression: v.Get("expr").(RegexpNode),
			Capturing:  true,
		}
	})
	
	group.Set(Or(namedGroup, nonCapturingGroup, capturingGroup))
	
	// Literals - match any character that's not a special regexp character
	literal := Transform(
		Seq(
			Not(Set('.', '^', '$', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|', '\\')),
			Any(), // Actually consume the character
		),
		func(s string) any {
			if len(s) > 0 {
				return &Literal{Char: rune(s[0])}
			}
			return &Literal{Char: ' '}
		},
	)
	
	// Dot wildcard
	dot := Action(S("."), func(v Values) any { return &Dot{} })
	
	// Atom (basic regexp element)
	atom.Set(Or(
		group,
		charClass,
		escape,
		anchor,
		dot,
		literal,
	))
	
	// Quantifiers
	starQuantifier := Action(Seq(
		Named("expr", atom),
		S("*"),
		Named("lazy", Capture(Maybe(S("?")))),
	), func(v Values) any {
		return &Quantifier{
			Min:        0,
			Max:        -1,
			Greedy:     v.Get("lazy").(string) == "",
			Expression: v.Get("expr").(RegexpNode),
		}
	})
	
	plusQuantifier := Action(Seq(
		Named("expr", atom),
		S("+"),
		Named("lazy", Capture(Maybe(S("?")))),
	), func(v Values) any {
		return &Quantifier{
			Min:        1,
			Max:        -1,
			Greedy:     v.Get("lazy").(string) == "",
			Expression: v.Get("expr").(RegexpNode),
		}
	})
	
	questionQuantifier := Action(Seq(
		Named("expr", atom),
		S("?"),
		Named("lazy", Capture(Maybe(S("?")))),
	), func(v Values) any {
		return &Quantifier{
			Min:        0,
			Max:        1,
			Greedy:     v.Get("lazy").(string) == "",
			Expression: v.Get("expr").(RegexpNode),
		}
	})
	
	bracedQuantifier := Action(Seq(
		Named("expr", atom),
		S("{"),
		ws,
		Named("min", number),
		ws,
		Named("hasComma", Capture(Maybe(Seq(
			S(","),
			ws,
			Named("max", Maybe(number)),
		)))),
		ws,
		S("}"),
		Named("lazy", Capture(Maybe(S("?")))),
	), func(v Values) any {
		min := v.Get("min").(int)
		hasComma := v.Get("hasComma")
		lazy := v.Get("lazy")
		
		var max int
		if hasCommaStr, ok := hasComma.(string); ok && hasCommaStr != "" {
			// {n,} or {n,m}
			if maxVal := v.Get("max"); maxVal != nil {
				max = maxVal.(int)
			} else {
				max = -1 // {n,} means n or more
			}
		} else {
			// {n} means exactly n
			max = min
		}
		
		return &Quantifier{
			Min:        min,
			Max:        max,
			Greedy:     lazy.(string) == "",
			Expression: v.Get("expr").(RegexpNode),
		}
	})
	
	quantifier.Set(Or(
		starQuantifier,
		plusQuantifier,
		questionQuantifier,
		bracedQuantifier,
		atom, // No quantifier
	))
	
	// Sequence (can be empty for alternation)
	sequence.Set(Action(
		Named("elements", Many(quantifier, 0, -1, func(vals []any) any {
			elements := make([]RegexpNode, len(vals))
			for i, v := range vals {
				elements[i] = v.(RegexpNode)
			}
			return elements
		})),
		func(v Values) any {
			elements := v.Get("elements").([]RegexpNode)
			if len(elements) == 0 {
				// Empty sequence (for empty alternation)
				return &Sequence{Elements: []RegexpNode{}}
			}
			if len(elements) == 1 {
				return elements[0]
			}
			return &Sequence{Elements: elements}
		},
	))
	
	// Alternation
	alternation.Set(Action(Seq(
		Named("first", sequence),
		Named("rest", Many(
			Action(Seq(
				S("|"),
				Named("seq", sequence),
			), func(v Values) any {
				return v.Get("seq")
			}),
			0, -1, func(vals []any) any {
				return vals
			})),
	), func(v Values) any {
		first := v.Get("first").(RegexpNode)
		rest := v.Get("rest").([]any)
		
		if len(rest) == 0 {
			return first
		}
		
		alternatives := []RegexpNode{first}
		for _, r := range rest {
			alternatives = append(alternatives, r.(RegexpNode))
		}
		
		return &Alternation{Options: alternatives}
	}))
	
	// Main regexp rule
	regexp.Set(Action(Named("node", alternation), func(v Values) any {
		return &RegexpPattern{Node: v.Get("node").(RegexpNode)}
	}))
	
	return regexp
}

// ParseRegexp parses a Go regexp string and returns the AST
func ParseRegexp(pattern string) (*RegexpPattern, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty regexp pattern")
	}
	
	parser := New()
	grammar := RegexpParser()
	
	val, matched, err := parser.Parse(grammar, pattern)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, fmt.Errorf("invalid regexp pattern: %q", pattern)
	}
	
	return val.(*RegexpPattern), nil
}