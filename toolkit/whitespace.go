package toolkit

import (
	"unicode"

	p "github.com/lab47/peggysue"
)

// WS is the start whitespace rule, matching any amount of spaces, tabs,
// or newlines.
var (
	IsWhiteSpace = p.Rune(unicode.IsSpace)
	WS           = p.Star(IsWhiteSpace)
)
