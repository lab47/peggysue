package peggysue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexpToRule(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
		value   any
	}{
		// Literals
		{
			name:    "simple literal",
			pattern: "hello",
			input:   "hello",
			want:    true,
			value:   "hello",
		},
		{
			name:    "literal no match",
			pattern: "hello",
			input:   "world",
			want:    false,
		},
		
		// Dot wildcard
		{
			name:    "dot matches any",
			pattern: "a.b",
			input:   "axb",
			want:    true,
			value:   "axb",
		},
		{
			name:    "dot matches newline",
			pattern: "a.b",
			input:   "a\nb",
			want:    true,
			value:   "a\nb",
		},
		
		// Character classes
		{
			name:    "character class",
			pattern: "[abc]",
			input:   "b",
			want:    true,
			value:   "b",
		},
		{
			name:    "character class no match",
			pattern: "[abc]",
			input:   "d",
			want:    false,
		},
		{
			name:    "character range",
			pattern: "[a-z]",
			input:   "m",
			want:    true,
			value:   "m",
		},
		{
			name:    "negated character class",
			pattern: "[^abc]",
			input:   "d",
			want:    true,
			value:   "d",
		},
		{
			name:    "negated character class no match",
			pattern: "[^abc]",
			input:   "a",
			want:    false,
		},
		
		// Quantifiers
		{
			name:    "star quantifier empty",
			pattern: "a*",
			input:   "",
			want:    true,
			value:   "",
		},
		{
			name:    "star quantifier multiple",
			pattern: "a*",
			input:   "aaa",
			want:    true,
			value:   "aaa",
		},
		{
			name:    "plus quantifier",
			pattern: "a+",
			input:   "aaa",
			want:    true,
			value:   "aaa",
		},
		{
			name:    "plus quantifier no match",
			pattern: "a+",
			input:   "",
			want:    false,
		},
		{
			name:    "question quantifier present",
			pattern: "a?",
			input:   "a",
			want:    true,
			value:   "a",
		},
		{
			name:    "question quantifier absent",
			pattern: "a?",
			input:   "",
			want:    true,
			value:   "",
		},
		
		// Alternation
		{
			name:    "alternation first",
			pattern: "cat|dog",
			input:   "cat",
			want:    true,
			value:   "cat",
		},
		{
			name:    "alternation second",
			pattern: "cat|dog",
			input:   "dog",
			want:    true,
			value:   "dog",
		},
		{
			name:    "alternation no match",
			pattern: "cat|dog",
			input:   "bird",
			want:    false,
		},
		
		// Groups
		{
			name:    "group",
			pattern: "(abc)",
			input:   "abc",
			want:    true,
			value:   "abc",
		},
		{
			name:    "group with quantifier",
			pattern: "(ab)+",
			input:   "ababab",
			want:    true,
			value:   "ababab",
		},
		
		// Escape sequences
		{
			name:    "digit escape",
			pattern: "\\d",
			input:   "5",
			want:    true,
			value:   "5",
		},
		{
			name:    "digit escape no match",
			pattern: "\\d",
			input:   "a",
			want:    false,
		},
		{
			name:    "non-digit escape",
			pattern: "\\D",
			input:   "a",
			want:    true,
			value:   "a",
		},
		{
			name:    "space escape",
			pattern: "\\s",
			input:   " ",
			want:    true,
			value:   " ",
		},
		{
			name:    "word escape",
			pattern: "\\w",
			input:   "a",
			want:    true,
			value:   "a",
		},
		{
			name:    "word escape digit",
			pattern: "\\w",
			input:   "5",
			want:    true,
			value:   "5",
		},
		{
			name:    "word escape underscore",
			pattern: "\\w",
			input:   "_",
			want:    true,
			value:   "_",
		},
		
		// Complex patterns
		{
			name:    "email-like pattern",
			pattern: "\\w+@\\w+\\.\\w+",
			input:   "test@example.com",
			want:    true,
			value:   "test@example.com",
		},
		{
			name:    "number pattern",
			pattern: "\\d{3}-\\d{3}-\\d{4}",
			input:   "123-456-7890",
			want:    true,
			value:   "123-456-7890",
		},
		{
			name:    "optional protocol",
			pattern: "(https?://)?example\\.com",
			input:   "https://example.com",
			want:    true,
			value:   "https://example.com",
		},
		{
			name:    "optional protocol without",
			pattern: "(https?://)?example\\.com",
			input:   "example.com",
			want:    true,
			value:   "example.com",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := Regexp(tt.pattern)
			parser := New()
			
			val, matched, err := parser.Parse(rule, tt.input)
			require.NoError(t, err)
			
			if tt.want {
				assert.True(t, matched, "expected pattern to match")
				if tt.value != nil {
					assert.Equal(t, tt.value, val)
				}
			} else {
				assert.False(t, matched, "expected pattern not to match")
			}
		})
	}
}

func TestRegexpToRuleAnchors(t *testing.T) {
	// Note: PEG parsers match from the current position, they don't search.
	// So patterns like "world$" won't find "world" at the end of "hello world"
	// unless we implement a search mechanism (which is beyond the scope here).
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
		partial bool // whether to allow partial matches
	}{
		{
			name:    "start anchor match",
			pattern: "^hello",
			input:   "hello world",
			want:    true,
			partial: true,
		},
		{
			name:    "start anchor no match",
			pattern: "^world",
			input:   "hello world",
			want:    false,
			partial: true,
		},
		{
			name:    "end anchor match",
			pattern: "world$",
			input:   "world",
			want:    true,
			partial: false,
		},
		{
			name:    "end anchor no match",
			pattern: "hello$",
			input:   "hello world",
			want:    false,
			partial: true,
		},
		{
			name:    "both anchors match",
			pattern: "^hello$",
			input:   "hello",
			want:    true,
			partial: false,
		},
		{
			name:    "both anchors no match",
			pattern: "^hello$",
			input:   "hello world",
			want:    false,
			partial: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := Regexp(tt.pattern)
			var parser *Parser
			if tt.partial {
				parser = New(WithPartial(true))
			} else {
				parser = New()
			}
			
			_, matched, err := parser.Parse(rule, tt.input)
			require.NoError(t, err)
			
			if tt.want {
				assert.True(t, matched, "expected pattern to match")
			} else {
				assert.False(t, matched, "expected pattern not to match")
			}
		})
	}
}

func TestRegexpToRuleCharacterClasses(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{
			name:    "character class with escape",
			pattern: "[^\\s]+",
			input:   "hello",
			want:    true,
		},
		{
			name:    "character class with escape no match",
			pattern: "[^\\s]+",
			input:   " ",
			want:    false,
		},
		{
			name:    "mixed character class",
			pattern: "[a-z\\d]+",
			input:   "abc123",
			want:    true,
		},
		{
			name:    "negated with multiple ranges",
			pattern: "[^a-zA-Z]+",
			input:   "123",
			want:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := Regexp(tt.pattern)
			parser := New()
			
			_, matched, err := parser.Parse(rule, tt.input)
			require.NoError(t, err)
			
			if tt.want {
				assert.True(t, matched, "expected pattern to match")
			} else {
				assert.False(t, matched, "expected pattern not to match")
			}
		})
	}
}

func TestRegexpToRuleEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{
			name:    "empty alternation left",
			pattern: "|a",
			input:   "",
			want:    true,
		},
		{
			name:    "empty alternation right",
			pattern: "a|",
			input:   "",
			want:    true,
		},
		{
			name:    "nested groups",
			pattern: "((a|b)+c)*",
			input:   "abcbcac",
			want:    true,
		},
		{
			name:    "escaped special chars",
			pattern: "\\[\\]\\.",
			input:   "[].",
			want:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := Regexp(tt.pattern)
			parser := New()
			
			_, matched, err := parser.Parse(rule, tt.input)
			require.NoError(t, err)
			
			if tt.want {
				assert.True(t, matched, "expected pattern to match")
			} else {
				assert.False(t, matched, "expected pattern not to match")
			}
		})
	}
}

func BenchmarkRegexpToRule(b *testing.B) {
	patterns := []struct {
		name    string
		pattern string
		input   string
	}{
		{"simple", "hello", "hello"},
		{"email", "\\w+@\\w+\\.\\w+", "test@example.com"},
		{"number", "\\d{3}-\\d{3}-\\d{4}", "123-456-7890"},
		{"complex", "((a|b)+c)*d", "abcbcacd"},
	}
	
	for _, p := range patterns {
		b.Run(p.name, func(b *testing.B) {
			rule := Regexp(p.pattern)
			parser := New()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, matched, err := parser.Parse(rule, p.input)
				if err != nil {
					b.Fatal(err)
				}
				if !matched {
					b.Fatalf("pattern %q did not match %q", p.pattern, p.input)
				}
			}
		})
	}
}

func BenchmarkRegexpVsRe(b *testing.B) {
	patterns := []struct {
		name    string
		pattern string
		input   string
	}{
		{"simple", "hello", "hello"},
		{"word", "\\w+", "hello"},
		{"email", "\\w+@\\w+\\.\\w+", "test@example.com"},
		{"number", "\\d{3}-\\d{3}-\\d{4}", "123-456-7890"},
		{"alternation", "(cat|dog)", "cat"},
		{"quantifier", "a+b*c?", "aaabbc"},
	}
	
	for _, p := range patterns {
		b.Run(p.name+"/Regexp", func(b *testing.B) {
			rule := Regexp(p.pattern)
			parser := New()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, matched, err := parser.Parse(rule, p.input)
				if err != nil {
					b.Fatal(err)
				}
				if !matched {
					b.Fatalf("pattern %q did not match %q", p.pattern, p.input)
				}
			}
		})
		
		b.Run(p.name+"/Re", func(b *testing.B) {
			rule := Re(p.pattern)
			parser := New()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, matched, err := parser.Parse(rule, p.input)
				if err != nil {
					b.Fatal(err)
				}
				if !matched {
					b.Fatalf("pattern %q did not match %q", p.pattern, p.input)
				}
			}
		})
	}
}