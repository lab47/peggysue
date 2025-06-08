package regexp

import (
	"fmt"
	"strings"
)

// Node is the base interface for all regexp AST nodes
type Node interface {
	String() string
}

// Pattern is the root node of a regexp AST
type Pattern struct {
	Node Node
}

func (r *Pattern) String() string {
	return r.Node.String()
}

// Alternation represents the | operator
type Alternation struct {
	Options []Node
}

func (a *Alternation) String() string {
	parts := make([]string, len(a.Options))
	for i, opt := range a.Options {
		parts[i] = opt.String()
	}
	return strings.Join(parts, "|")
}

// Sequence represents a sequence of regexp elements
type Sequence struct {
	Elements []Node
}

func (s *Sequence) String() string {
	var sb strings.Builder
	for _, elem := range s.Elements {
		sb.WriteString(elem.String())
	}
	return sb.String()
}

// Literal represents a literal character
type Literal struct {
	Char rune
}

func (l *Literal) String() string {
	// Escape special characters
	switch l.Char {
	case '.', '^', '$', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|', '\\':
		return "\\" + string(l.Char)
	default:
		return string(l.Char)
	}
}

// Dot represents the . wildcard
type Dot struct{}

func (d *Dot) String() string {
	return "."
}

// CharClass represents a character class [...]
type CharClass struct {
	Negated bool
	Elements []any // Can be CharRange or EscapeSequence
}

func (c *CharClass) String() string {
	var sb strings.Builder
	sb.WriteRune('[')
	if c.Negated {
		sb.WriteRune('^')
	}
	for _, elem := range c.Elements {
		switch e := elem.(type) {
		case CharRange:
			sb.WriteString(e.String())
		case *EscapeSequence:
			sb.WriteString(e.String())
		}
	}
	sb.WriteRune(']')
	return sb.String()
}

// CharRange represents a character or character range in a character class
type CharRange struct {
	Start rune
	End   rune // If End == Start, it's a single character
}

func (c *CharRange) String() string {
	if c.Start == c.End {
		return string(c.Start)
	}
	return fmt.Sprintf("%c-%c", c.Start, c.End)
}

// Quantifier represents repetition operators
type Quantifier struct {
	Min        int
	Max        int  // -1 means unbounded
	Greedy     bool // false for non-greedy (lazy) quantifiers
	Expression Node
}

func (q *Quantifier) String() string {
	expr := q.Expression.String()
	
	// Wrap in parentheses if needed
	needsParens := false
	switch q.Expression.(type) {
	case *Alternation, *Sequence:
		if seq, ok := q.Expression.(*Sequence); ok && len(seq.Elements) > 1 {
			needsParens = true
		} else if _, ok := q.Expression.(*Alternation); ok {
			needsParens = true
		}
	}
	
	if needsParens {
		expr = "(?:" + expr + ")"
	}
	
	var quant string
	if q.Min == 0 && q.Max == -1 {
		quant = "*"
	} else if q.Min == 1 && q.Max == -1 {
		quant = "+"
	} else if q.Min == 0 && q.Max == 1 {
		quant = "?"
	} else if q.Min == q.Max {
		quant = fmt.Sprintf("{%d}", q.Min)
	} else if q.Max == -1 {
		quant = fmt.Sprintf("{%d,}", q.Min)
	} else {
		quant = fmt.Sprintf("{%d,%d}", q.Min, q.Max)
	}
	
	if !q.Greedy {
		quant += "?"
	}
	
	return expr + quant
}

// Group represents a grouped expression
type Group struct {
	Expression  Node
	Capturing   bool
	Name        string // For named groups
	NonCapturing bool
}

func (g *Group) String() string {
	var sb strings.Builder
	sb.WriteRune('(')
	
	if g.NonCapturing {
		sb.WriteString("?:")
	} else if g.Name != "" {
		sb.WriteString("?P<")
		sb.WriteString(g.Name)
		sb.WriteRune('>')
	}
	
	sb.WriteString(g.Expression.String())
	sb.WriteRune(')')
	return sb.String()
}

// Anchor represents position assertions
type Anchor struct {
	Type AnchorType
}

type AnchorType int

const (
	AnchorStart      AnchorType = iota // ^
	AnchorEnd                          // $
	AnchorStartText                    // \A
	AnchorEndText                      // \z
	AnchorWordBoundary                 // \b
	AnchorNonWordBoundary              // \B
)

func (a *Anchor) String() string {
	switch a.Type {
	case AnchorStart:
		return "^"
	case AnchorEnd:
		return "$"
	case AnchorStartText:
		return "\\A"
	case AnchorEndText:
		return "\\z"
	case AnchorWordBoundary:
		return "\\b"
	case AnchorNonWordBoundary:
		return "\\B"
	default:
		return ""
	}
}

// EscapeSequence represents escape sequences
type EscapeSequence struct {
	Type  EscapeType
	Value string // For hex/unicode escapes
}

type EscapeType int

const (
	EscapeChar        EscapeType = iota // Single character escapes like \n, \t
	EscapeHex                           // \x00
	EscapeUnicode                       // \u0000
	EscapeUnicodeLong                   // \U00000000
	EscapeDigit                         // \d
	EscapeNonDigit                      // \D
	EscapeSpace                         // \s
	EscapeNonSpace                      // \S
	EscapeWord                          // \w
	EscapeNonWord                       // \W
	EscapeProperty                      // \p{...}
	EscapeNonProperty                   // \P{...}
)

func (e *EscapeSequence) String() string {
	switch e.Type {
	case EscapeChar:
		// Convert back to escape sequence representation
		switch e.Value {
		case "\n":
			return "\\n"
		case "\t":
			return "\\t"
		case "\r":
			return "\\r"
		case "\f":
			return "\\f"
		case "\v":
			return "\\v"
		case "\a":
			return "\\a"
		case "\b":
			return "\\b"
		default:
			return "\\" + e.Value
		}
	case EscapeHex:
		return "\\x" + e.Value
	case EscapeUnicode:
		return "\\u" + e.Value
	case EscapeUnicodeLong:
		return "\\U" + e.Value
	case EscapeDigit:
		return "\\d"
	case EscapeNonDigit:
		return "\\D"
	case EscapeSpace:
		return "\\s"
	case EscapeNonSpace:
		return "\\S"
	case EscapeWord:
		return "\\w"
	case EscapeNonWord:
		return "\\W"
	case EscapeProperty:
		return "\\p{" + e.Value + "}"
	case EscapeNonProperty:
		return "\\P{" + e.Value + "}"
	default:
		return ""
	}
}

// Flags represents regexp flags
type Flags struct {
	CaseInsensitive bool // i
	Multiline       bool // m
	DotAll          bool // s
	Ungreedy        bool // U
}