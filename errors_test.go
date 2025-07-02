package peggysue

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_BasicErrors(t *testing.T) {
	tests := []struct {
		name          string
		rule          Rule
		input         string
		opts          []ParseOption
		wantErr       bool
		errContains   string
		expectContains []string
	}{
		{
			name:          "simple string mismatch",
			rule:          S("hello"),
			input:         "world",
			wantErr:       true,
			errContains:   "expected \"hello\"",
			expectContains: []string{"line 1, column 1"},
		},
		{
			name:          "EOF expected", 
			rule:          S("hello"),
			input:         "hello world",
			opts:          []ParseOption{},
			wantErr:       true,
			errContains:   "expected end of input",
			expectContains: []string{"line 1, column 6"},
		},
		{
			name:          "multiple alternatives",
			rule:          Or(S("foo"), S("bar"), S("baz")),
			input:         "qux",
			wantErr:       true,
			errContains:   "expected \"foo\"",
			expectContains: []string{"line 1, column 1"},
		},
		{
			name:          "range mismatch",
			rule:          Range('a', 'z'),
			input:         "123",
			wantErr:       true,
			errContains:   "expected character in range 'a'-'z'",
			expectContains: []string{"line 1, column 1"},
		},
		{
			name:          "successful parse",
			rule:          S("hello"),
			input:         "hello",
			wantErr:       false,
		},
		{
			name:          "named rule failure",
			rule:          N("identifier", Plus(Range('a', 'z'))),
			input:         "123",
			wantErr:       true,
			errContains:   "expected character in range 'a'-'z'",
			expectContains: []string{"line 1, column 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := append([]ParseOption{WithErrors()}, tt.opts...)
			val, matched, err := Parse(tt.rule, tt.input, opts...)
			
			if tt.wantErr {
				require.Error(t, err)
				require.False(t, matched)
				require.Nil(t, val)
				
				parseErr, ok := err.(*ParseError)
				require.True(t, ok, "error should be *ParseError")
				
				errMsg := parseErr.Error()
				if tt.errContains != "" {
					assert.Contains(t, errMsg, tt.errContains)
				}
				
				for _, contains := range tt.expectContains {
					assert.Contains(t, errMsg, contains)
				}
			} else {
				require.NoError(t, err)
				require.True(t, matched)
			}
		})
	}
}

func TestParse_MultilineErrors(t *testing.T) {
	input := `line one
line two
line three`
	
	// Try to match "foo" at the beginning of line three
	rule := Seq(S("line one\nline two\n"), S("foo"))
	
	val, matched, err := Parse(rule, input, WithErrors())
	
	require.Error(t, err)
	require.False(t, matched)
	require.Nil(t, val)
	
	parseErr, ok := err.(*ParseError)
	require.True(t, ok)
	
	line, col := parseErr.Position()
	assert.Equal(t, 3, line)
	assert.Equal(t, 1, col)
	
	errMsg := parseErr.Error()
	assert.Contains(t, errMsg, "line 3, column 1")
	assert.Contains(t, errMsg, "expected \"foo\"")
	assert.Contains(t, errMsg, "line three")
	assert.Contains(t, errMsg, "^")
}

func TestParse_ComplexGrammar(t *testing.T) {
	// Simple arithmetic expression grammar
	var expr Rule
	
	number := N("number", Plus(Range('0', '9')))
	operator := N("operator", Or(S("+"), S("-"), S("*"), S("/")))
	
	expr = Seq(number, Star(Seq(operator, number)))
	
	tests := []struct {
		input       string
		shouldMatch bool
		errContains string
	}{
		{"123+456", true, ""},
		{"123+", false, "end of input"},
		{"abc", false, "character in range '0'-'9'"},
		{"123&456", false, "end of input"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			val, matched, err := Parse(expr, tt.input, WithErrors())
			
			if tt.shouldMatch {
				require.NoError(t, err)
				require.True(t, matched)
			} else {
				require.Error(t, err)
				require.False(t, matched)
				require.Nil(t, val)
				
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestParse_WithFilename(t *testing.T) {
	rule := S("hello")
	input := "world"
	
	_, _, err := Parse(rule, input, WithErrors(), WithFilename("test.txt"))
	
	require.Error(t, err)
	parseErr := err.(*ParseError)
	
	errMsg := parseErr.Error()
	assert.Contains(t, errMsg, "test.txt:1:1:")
}

func TestParse_PartialMatching(t *testing.T) {
	// Test partial matching option
	rule := S("hello")
	input := "hello world"
	
	val, matched, err := Parse(rule, input, WithPartialMatch())
	
	require.NoError(t, err)
	require.True(t, matched)
	require.Nil(t, val)
}

func TestParse_WithErrorsOption(t *testing.T) {
	// Test that Parse with WithErrors() enables error tracking
	rule := S("hello")
	input := "world"
	
	val, matched, err := Parse(rule, input, WithErrors())
	
	require.Error(t, err)
	require.False(t, matched)
	require.Nil(t, val)
	
	parseErr, ok := err.(*ParseError)
	require.True(t, ok, "error should be *ParseError")
	
	errMsg := parseErr.Error()
	assert.Contains(t, errMsg, "expected \"hello\"")
	assert.Contains(t, errMsg, "line 1, column 1")
}

func TestParse_NoErrorsByDefault(t *testing.T) {
	// Test that Parse has no error tracking by default (legacy behavior)
	rule := S("hello")
	input := "world"
	
	val, matched, err := Parse(rule, input)
	
	require.NoError(t, err)
	require.False(t, matched)
	require.Nil(t, val)
}

func TestParseError_Preview(t *testing.T) {
	input := "this is a long line with an error somewhere in the middle"
	rule := Seq(S("this is a long line with an "), S("success"))
	
	_, _, err := Parse(rule, input, WithErrors())
	
	require.Error(t, err)
	errMsg := err.Error()
	
	// Check that the preview shows the line and a pointer
	assert.Contains(t, errMsg, input)
	assert.Contains(t, errMsg, strings.Repeat(" ", 29) + "^")
}