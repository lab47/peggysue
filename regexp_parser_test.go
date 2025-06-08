package peggysue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexpParser(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
		wantErr  bool
	}{
		// Literals
		{
			name:     "simple literal",
			pattern:  "hello",
			expected: "hello",
		},
		{
			name:     "literal with spaces",
			pattern:  "hello world",
			expected: "hello world",
		},
		
		// Dot wildcard
		{
			name:     "dot wildcard",
			pattern:  "a.b",
			expected: "a.b",
		},
		
		// Character classes
		{
			name:     "simple character class",
			pattern:  "[abc]",
			expected: "[abc]",
		},
		{
			name:     "character class with range",
			pattern:  "[a-z]",
			expected: "[a-z]",
		},
		{
			name:     "negated character class",
			pattern:  "[^abc]",
			expected: "[^abc]",
		},
		{
			name:     "character class with multiple ranges",
			pattern:  "[a-zA-Z0-9]",
			expected: "[a-zA-Z0-9]",
		},
		
		// Quantifiers
		{
			name:     "star quantifier",
			pattern:  "a*",
			expected: "a*",
		},
		{
			name:     "plus quantifier",
			pattern:  "a+",
			expected: "a+",
		},
		{
			name:     "question quantifier",
			pattern:  "a?",
			expected: "a?",
		},
		{
			name:     "exact quantifier",
			pattern:  "a{3}",
			expected: "a{3}",
		},
		{
			name:     "range quantifier",
			pattern:  "a{2,5}",
			expected: "a{2,5}",
		},
		{
			name:     "open range quantifier",
			pattern:  "a{2,}",
			expected: "a{2,}",
		},
		{
			name:     "lazy star",
			pattern:  "a*?",
			expected: "a*?",
		},
		{
			name:     "lazy plus",
			pattern:  "a+?",
			expected: "a+?",
		},
		{
			name:     "lazy question",
			pattern:  "a??",
			expected: "a??",
		},
		
		// Alternation
		{
			name:     "simple alternation",
			pattern:  "cat|dog",
			expected: "cat|dog",
		},
		{
			name:     "multiple alternation",
			pattern:  "cat|dog|bird",
			expected: "cat|dog|bird",
		},
		
		// Groups
		{
			name:     "capturing group",
			pattern:  "(abc)",
			expected: "(abc)",
		},
		{
			name:     "non-capturing group",
			pattern:  "(?:abc)",
			expected: "(?:abc)",
		},
		{
			name:     "named group",
			pattern:  "(?P<name>abc)",
			expected: "(?P<name>abc)",
		},
		
		// Anchors
		{
			name:     "start anchor",
			pattern:  "^hello",
			expected: "^hello",
		},
		{
			name:     "end anchor",
			pattern:  "world$",
			expected: "world$",
		},
		{
			name:     "both anchors",
			pattern:  "^hello$",
			expected: "^hello$",
		},
		
		// Escape sequences
		{
			name:     "newline escape",
			pattern:  "hello\\nworld",
			expected: "hello\\nworld",
		},
		{
			name:     "tab escape",
			pattern:  "hello\\tworld",
			expected: "hello\\tworld",
		},
		{
			name:     "escaped dot",
			pattern:  "\\.",
			expected: "\\.",
		},
		{
			name:     "escaped backslash",
			pattern:  "\\\\",
			expected: "\\\\",
		},
		
		// Character class escapes
		{
			name:     "digit class",
			pattern:  "\\d",
			expected: "\\d",
		},
		{
			name:     "non-digit class",
			pattern:  "\\D",
			expected: "\\D",
		},
		{
			name:     "space class",
			pattern:  "\\s",
			expected: "\\s",
		},
		{
			name:     "word class",
			pattern:  "\\w+",
			expected: "\\w+",
		},
		
		// Complex patterns
		{
			name:     "email pattern",
			pattern:  "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
			expected: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
		},
		{
			name:     "URL pattern",
			pattern:  "https?://[^\\s]+",
			expected: "https?://[^\\s]+",
		},
		{
			name:     "phone pattern",
			pattern:  "\\d{3}-\\d{3}-\\d{4}",
			expected: "\\d{3}-\\d{3}-\\d{4}",
		},
		{
			name:     "grouped alternation",
			pattern:  "(cat|dog)s?",
			expected: "(cat|dog)s?",
		},
		{
			name:     "nested groups",
			pattern:  "((a|b)+c)*",
			expected: "((a|b)+c)*",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := ParseRegexp(tt.pattern)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, ast)
			
			// Convert AST back to string to verify structure
			got := ast.String()
			assert.Equal(t, tt.expected, got, "AST string representation should match")
		})
	}
}

func TestRegexpParserDetailed(t *testing.T) {
	t.Run("literal parsing", func(t *testing.T) {
		ast, err := ParseRegexp("abc")
		require.NoError(t, err)
		
		seq, ok := ast.Node.(*Sequence)
		require.True(t, ok)
		require.Len(t, seq.Elements, 3)
		
		for i, ch := range []rune{'a', 'b', 'c'} {
			lit, ok := seq.Elements[i].(*Literal)
			require.True(t, ok)
			assert.Equal(t, ch, lit.Char)
		}
	})
	
	t.Run("quantifier parsing", func(t *testing.T) {
		ast, err := ParseRegexp("a*b+c?")
		require.NoError(t, err)
		
		seq, ok := ast.Node.(*Sequence)
		require.True(t, ok)
		require.Len(t, seq.Elements, 3)
		
		// Check a*
		q1, ok := seq.Elements[0].(*Quantifier)
		require.True(t, ok)
		assert.Equal(t, 0, q1.Min)
		assert.Equal(t, -1, q1.Max)
		assert.True(t, q1.Greedy)
		
		// Check b+
		q2, ok := seq.Elements[1].(*Quantifier)
		require.True(t, ok)
		assert.Equal(t, 1, q2.Min)
		assert.Equal(t, -1, q2.Max)
		
		// Check c?
		q3, ok := seq.Elements[2].(*Quantifier)
		require.True(t, ok)
		assert.Equal(t, 0, q3.Min)
		assert.Equal(t, 1, q3.Max)
	})
	
	t.Run("alternation parsing", func(t *testing.T) {
		ast, err := ParseRegexp("cat|dog|bird")
		require.NoError(t, err)
		
		alt, ok := ast.Node.(*Alternation)
		require.True(t, ok)
		require.Len(t, alt.Options, 3)
	})
	
	t.Run("character class parsing", func(t *testing.T) {
		ast, err := ParseRegexp("[a-zA-Z0-9]")
		require.NoError(t, err)
		
		cc, ok := ast.Node.(*CharClass)
		require.True(t, ok)
		assert.False(t, cc.Negated)
		require.Len(t, cc.Elements, 3)
		
		// Check elements
		r0, ok := cc.Elements[0].(CharRange)
		require.True(t, ok)
		assert.Equal(t, 'a', r0.Start)
		assert.Equal(t, 'z', r0.End)
		
		r1, ok := cc.Elements[1].(CharRange)
		require.True(t, ok)
		assert.Equal(t, 'A', r1.Start)
		assert.Equal(t, 'Z', r1.End)
		
		r2, ok := cc.Elements[2].(CharRange)
		require.True(t, ok)
		assert.Equal(t, '0', r2.Start)
		assert.Equal(t, '9', r2.End)
	})
	
	t.Run("group parsing", func(t *testing.T) {
		// Capturing group
		ast, err := ParseRegexp("(abc)")
		require.NoError(t, err)
		
		grp, ok := ast.Node.(*Group)
		require.True(t, ok)
		assert.True(t, grp.Capturing)
		assert.False(t, grp.NonCapturing)
		assert.Empty(t, grp.Name)
		
		// Non-capturing group
		ast, err = ParseRegexp("(?:abc)")
		require.NoError(t, err)
		
		grp, ok = ast.Node.(*Group)
		require.True(t, ok)
		assert.False(t, grp.Capturing)
		assert.True(t, grp.NonCapturing)
		
		// Named group
		ast, err = ParseRegexp("(?P<word>\\w+)")
		require.NoError(t, err)
		
		grp, ok = ast.Node.(*Group)
		require.True(t, ok)
		assert.True(t, grp.Capturing)
		assert.Equal(t, "word", grp.Name)
	})
	
	t.Run("escape sequences", func(t *testing.T) {
		ast, err := ParseRegexp("\\n\\t\\d\\w")
		require.NoError(t, err)
		
		seq, ok := ast.Node.(*Sequence)
		require.True(t, ok)
		require.Len(t, seq.Elements, 4)
		
		// Check \n
		esc1, ok := seq.Elements[0].(*EscapeSequence)
		require.True(t, ok)
		assert.Equal(t, EscapeChar, esc1.Type)
		assert.Equal(t, "\n", esc1.Value)
		
		// Check \t
		esc2, ok := seq.Elements[1].(*EscapeSequence)
		require.True(t, ok)
		assert.Equal(t, EscapeChar, esc2.Type)
		assert.Equal(t, "\t", esc2.Value)
		
		// Check \d
		esc3, ok := seq.Elements[2].(*EscapeSequence)
		require.True(t, ok)
		assert.Equal(t, EscapeDigit, esc3.Type)
		
		// Check \w
		esc4, ok := seq.Elements[3].(*EscapeSequence)
		require.True(t, ok)
		assert.Equal(t, EscapeWord, esc4.Type)
	})
}

func TestRegexpParserEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "empty pattern",
			pattern: "",
			wantErr: true,
		},
		{
			name:    "unmatched parenthesis",
			pattern: "(abc",
			wantErr: true,
		},
		{
			name:    "unmatched bracket",
			pattern: "[abc",
			wantErr: true,
		},
		{
			name:    "invalid quantifier",
			pattern: "a{,5}",
			wantErr: true,
		},
		{
			name:    "quantifier without atom",
			pattern: "*",
			wantErr: true,
		},
		{
			name:    "empty alternation",
			pattern: "a|",
			wantErr: false, // This is valid in Go regexp
		},
		{
			name:    "complex nested pattern",
			pattern: "((a*|b+)c?)*d",
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRegexp(tt.pattern)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func BenchmarkRegexpParser(b *testing.B) {
	patterns := []struct {
		name    string
		pattern string
	}{
		{"simple", "hello"},
		{"email", "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"},
		{"complex", "((a*|b+)c?)*d"},
		{"alternation", "cat|dog|bird|fish|hamster"},
		{"quantifiers", "a*b+c?d{2,5}e{3}"},
	}
	
	for _, p := range patterns {
		b.Run(p.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := ParseRegexp(p.pattern)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}