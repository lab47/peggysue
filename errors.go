package peggysue

import (
	"fmt"
	"strings"
)

// ParseError represents a detailed parse failure with position and expectation information
type ParseError struct {
	Input       string
	MaxPos      int
	Failures    []RuleFailure
	filename    string
	linePos     []int
}

// RuleFailure represents a single rule that failed to match at a specific position
type RuleFailure struct {
	Pos      int
	Rule     Rule
	Expected string // Human-readable expectation
}

// Error implements the error interface
func (e *ParseError) Error() string {
	if len(e.Failures) == 0 {
		return fmt.Sprintf("parse failed at position %d", e.MaxPos)
	}

	// Find the furthest failures for the most relevant error
	furthest := e.getFurthestFailures()
	
	// Use the actual failure position, not necessarily MaxPos
	errorPos := e.MaxPos
	if len(furthest) > 0 {
		errorPos = furthest[0].Pos
	}
	
	line, col := e.lineCol(errorPos)
	preview := e.getPreview(errorPos)
	
	var expectations []string
	seen := make(map[string]bool)
	
	for _, f := range furthest {
		exp := f.Expected
		if exp == "" {
			exp = Print(f.Rule)
		}
		if !seen[exp] {
			seen[exp] = true
			expectations = append(expectations, exp)
		}
	}
	
	var msg strings.Builder
	if e.filename != "" {
		fmt.Fprintf(&msg, "%s:%d:%d: ", e.filename, line, col)
	} else {
		fmt.Fprintf(&msg, "line %d, column %d: ", line, col)
	}
	
	fmt.Fprintf(&msg, "expected %s", formatExpectations(expectations))
	
	if preview != "" {
		fmt.Fprintf(&msg, "\n%s", preview)
	}
	
	return msg.String()
}

// Position returns the furthest position reached during parsing
func (e *ParseError) Position() (line, col int) {
	return e.lineCol(e.MaxPos)
}

// getFurthestFailures returns failures at the maximum position
func (e *ParseError) getFurthestFailures() []RuleFailure {
	var furthest []RuleFailure
	for _, f := range e.Failures {
		if f.Pos == e.MaxPos {
			furthest = append(furthest, f)
		}
	}
	return furthest
}

// lineCol converts a byte position to line and column numbers
func (e *ParseError) lineCol(pos int) (line, col int) {
	line = 1
	lastNewline := -1
	
	for _, nlPos := range e.linePos {
		if nlPos >= pos {
			break
		}
		line++
		lastNewline = nlPos
	}
	
	col = pos - lastNewline
	return line, col
}

// getPreview returns a preview of the input around the error position
func (e *ParseError) getPreview(pos int) string {
	if pos >= len(e.Input) {
		pos = len(e.Input) - 1
	}
	if pos < 0 {
		return ""
	}
	
	// Find start of line
	start := pos
	for start > 0 && e.Input[start-1] != '\n' {
		start--
	}
	
	// Find end of line
	end := pos
	for end < len(e.Input) && e.Input[end] != '\n' {
		end++
	}
	
	if start >= end {
		return ""
	}
	
	line := e.Input[start:end]
	pointer := strings.Repeat(" ", pos-start) + "^"
	
	return fmt.Sprintf("  %s\n  %s", line, pointer)
}

// formatExpectations formats a list of expectations into a readable string
func formatExpectations(expectations []string) string {
	switch len(expectations) {
	case 0:
		return "valid input"
	case 1:
		return expectations[0]
	case 2:
		return expectations[0] + " or " + expectations[1]
	default:
		return strings.Join(expectations[:len(expectations)-1], ", ") + ", or " + expectations[len(expectations)-1]
	}
}

// ParseOption configures parsing behavior
type ParseOption func(*parseConfig)

type parseConfig struct {
	collectErrors bool
	partialOk     bool
	debug         bool
	filename      string
}

// WithErrors enables detailed error collection during parsing
func WithErrors() ParseOption {
	return func(c *parseConfig) {
		c.collectErrors = true
	}
}

// WithPartialMatch allows partial matching (doesn't require full input consumption)
func WithPartialMatch() ParseOption {
	return func(c *parseConfig) {
		c.partialOk = true
	}
}

// WithFilename sets the filename for error reporting
func WithFilename(filename string) ParseOption {
	return func(c *parseConfig) {
		c.filename = filename
	}
}

// Parse attempts to match the given rule against the input string with options.
// By default, no error tracking is performed for backward compatibility.
// Use WithErrors() to enable detailed error reporting.
func Parse(r Rule, input string, opts ...ParseOption) (val any, matched bool, err error) {
	config := parseConfig{
		collectErrors: false, // default to no error collection for backward compatibility
		partialOk:     false, // default to full match required
	}
	
	for _, opt := range opts {
		opt(&config)
	}
	
	s := &state{
		input:         input,
		inputSize:     len(input),
		pos:           0,
		maxPos:        0,
		filename:      config.filename,
		collectErrors: config.collectErrors,
		linePos:       computeLines(input),
	}
	
	if config.collectErrors {
		s.failures = make([]RuleFailure, 0)
	}
	
	s.setupBadGood(config.debug)
	
	res := s.match(r)
	
	if !res.matched {
		if config.collectErrors && len(s.failures) > 0 {
			return nil, false, &ParseError{
				Input:    input,
				MaxPos:   s.maxPos,
				Failures: s.failures,
				filename: config.filename,
				linePos:  s.linePos,
			}
		}
		return nil, false, nil
	}
	
	if !config.partialOk && s.pos != len(input) {
		if config.collectErrors {
			return res.value, false, &ParseError{
				Input:    input,
				MaxPos:   s.pos,
				Failures: []RuleFailure{{Pos: s.pos, Expected: "end of input"}},
				filename: config.filename,
				linePos:  s.linePos,
			}
		}
		
		// Maintain backward compatibility with existing error
		return res.value, false, &ErrInputNotConsumed{
			MaxPos:  s.maxPos,
			MaxRule: s.maxRule,
		}
	}
	
	return res.value, true, nil
}