package peggysue

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexpComplexEmail(t *testing.T) {
	// This test investigates a limitation with PEG's greedy matching
	// When a quantified character class containing a dot is followed by a dot,
	// the quantifier greedily consumes all dots, leaving none for the following dot.
	// For example: [.]+\. fails to match "...." because Plus(Or(S("."))) consumes all dots.
	// This is correct PEG behavior but differs from regexp backtracking.
	pattern := "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
	input := "john.doe+filter@example.com"
	
	t.Run("parse pattern", func(t *testing.T) {
		ast, err := ParseRegexp(pattern)
		require.NoError(t, err)
		require.NotNil(t, ast)
		
		// Verify the pattern parses correctly
		reconstructed := ast.String()
		t.Logf("Original pattern: %s", pattern)
		t.Logf("Reconstructed:    %s", reconstructed)
		assert.Equal(t, pattern, reconstructed)
	})
	
	t.Run("match with Regexp", func(t *testing.T) {
		rule := Regexp(pattern)
		parser := New()
		
		val, matched, err := parser.Parse(rule, input)
		t.Logf("Regexp result: val=%v, matched=%v, err=%v", val, matched, err)
		
		// Try with partial parsing
		partialParser := New(WithPartial(true))
		val2, matched2, err2 := partialParser.Parse(rule, input)
		t.Logf("Regexp partial result: val=%v, matched=%v, err=%v", val2, matched2, err2)
	})
	
	t.Run("match with Re for comparison", func(t *testing.T) {
		rule := Re(pattern)
		parser := New()
		
		val, matched, err := parser.Parse(rule, input)
		t.Logf("Re result: val=%v, matched=%v, err=%v", val, matched, err)
		
		// Try with partial parsing
		partialParser := New(WithPartial(true))
		val2, matched2, err2 := partialParser.Parse(rule, input)
		t.Logf("Re partial result: val=%v, matched=%v, err=%v", val2, matched2, err2)
	})
	
	// Test individual components of the pattern
	t.Run("test pattern components", func(t *testing.T) {
		components := []struct {
			name    string
			pattern string
			input   string
			expect  string
		}{
			{"username chars", "[a-zA-Z0-9._%+-]+", "john.doe+filter", "john.doe+filter"},
			{"at symbol", "@", "@", "@"},
			{"domain chars", "[a-zA-Z0-9.-]+", "example", "example"},
			{"dot", "\\.", ".", "."},
			{"tld", "[a-zA-Z]{2,}", "com", "com"},
			{"simple email part", "[a-z]+@[a-z]+", "john@example", "john@example"},
			{"with dots", "[a-z.]+@[a-z.]+", "john.doe@example.com", "john.doe@example.com"},
			{"with special chars", "[a-z.+]+@[a-z.]+", "john.doe+filter@example.com", "john.doe+filter@example.com"},
		}
		
		for _, tc := range components {
			t.Run(tc.name, func(t *testing.T) {
				rule := Regexp(tc.pattern)
				parser := New()
				
				val, matched, err := parser.Parse(rule, tc.input)
				if err != nil {
					// Check if it's just not consuming full input
					if errNotConsumed, ok := err.(*ErrInputNotConsumed); ok {
						t.Logf("Pattern %q matched %q but didn't consume full input (stopped at pos %d)", 
							tc.pattern, tc.input, errNotConsumed.MaxPos)
						// Try with partial parsing
						partialParser := New(WithPartial(true))
						val2, matched2, err2 := partialParser.Parse(rule, tc.input)
						require.NoError(t, err2)
						assert.True(t, matched2)
						assert.Equal(t, tc.expect, val2)
						return
					}
				}
				
				require.NoError(t, err)
				assert.True(t, matched, "Pattern %q should match %q", tc.pattern, tc.input)
				assert.Equal(t, tc.expect, val)
			})
		}
	})
	
	// Test the AST structure directly
	t.Run("inspect AST structure", func(t *testing.T) {
		ast, err := ParseRegexp(pattern)
		require.NoError(t, err)
		
		// The pattern should be a sequence
		seq, ok := ast.Node.(*Sequence)
		require.True(t, ok, "Expected Sequence, got %T", ast.Node)
		require.Len(t, seq.Elements, 5, "Expected 5 elements in sequence")
		
		// First element: [a-zA-Z0-9._%+-]+
		q1, ok := seq.Elements[0].(*Quantifier)
		require.True(t, ok, "First element should be Quantifier, got %T", seq.Elements[0])
		assert.Equal(t, 1, q1.Min)
		assert.Equal(t, -1, q1.Max)
		
		cc1, ok := q1.Expression.(*CharClass)
		require.True(t, ok, "Quantifier expression should be CharClass, got %T", q1.Expression)
		assert.False(t, cc1.Negated)
		
		// Log the character class elements
		t.Logf("First character class has %d elements:", len(cc1.Elements))
		for i, elem := range cc1.Elements {
			switch e := elem.(type) {
			case CharRange:
				if e.Start == e.End {
					t.Logf("  [%d] CharRange (single): '%c' (code %d)", i, e.Start, e.Start)
				} else {
					t.Logf("  [%d] CharRange: '%c' to '%c' (codes %d-%d)", i, e.Start, e.End, e.Start, e.End)
				}
			case *EscapeSequence:
				t.Logf("  [%d] EscapeSequence: %s", i, e.String())
			default:
				t.Logf("  [%d] Unknown type: %T", i, elem)
			}
		}
	})
	
	// Test specific character class edge cases
	t.Run("character class dash handling", func(t *testing.T) {
		testCases := []struct {
			name     string
			pattern  string
			input    string
			expected string
			shouldMatch bool
		}{
			{"dash at end", "[a-z-]", "-", "-", true},
			{"dash at end", "[a-z-]", "a", "a", true},
			{"dash at start", "[-a-z]", "-", "-", true},
			{"dash at start", "[-a-z]", "a", "a", true},
			{"dash in middle as range", "[a-z]", "m", "m", true},
			{"dash in middle as range", "[a-z]", "-", "", false},
			{"complex with dash at end", "[a-z+-]", "+", "+", true},
			{"complex with dash at end", "[a-z+-]", "-", "-", true},
			{"complex with dash at end", "[a-z+-]", "a", "a", true},
			{"the problematic pattern", "[._%+-]", "+", "+", true},
			{"the problematic pattern", "[._%+-]", "-", "-", true},
			{"the problematic pattern", "[._%+-]", "%", "%", true},
			{"exact email char class", "[a-zA-Z0-9._%+-]", "a", "a", true},
			{"exact email char class", "[a-zA-Z0-9._%+-]", ".", ".", true},
			{"exact email char class", "[a-zA-Z0-9._%+-]", "+", "+", true},
			{"exact email char class", "[a-zA-Z0-9._%+-]", "-", "-", true},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name+" with "+tc.input, func(t *testing.T) {
				// Parse and inspect AST
				ast, err := ParseRegexp(tc.pattern)
				require.NoError(t, err)
				
				// Log the character class structure
				if cc, ok := ast.Node.(*CharClass); ok {
					t.Logf("Pattern %s has %d elements:", tc.pattern, len(cc.Elements))
					for i, elem := range cc.Elements {
						switch e := elem.(type) {
						case CharRange:
							if e.Start == e.End {
								t.Logf("  [%d] Single char: '%c'", i, e.Start)
							} else {
								t.Logf("  [%d] Range: '%c' to '%c'", i, e.Start, e.End)
							}
						}
					}
				}
				
				// Test matching
				rule := Regexp(tc.pattern)
				parser := New()
				
				val, matched, err := parser.Parse(rule, tc.input)
				require.NoError(t, err)
				
				if tc.shouldMatch {
					assert.True(t, matched, "Pattern %q should match %q", tc.pattern, tc.input)
					assert.Equal(t, tc.expected, val)
				} else {
					assert.False(t, matched, "Pattern %q should not match %q", tc.pattern, tc.input)
				}
			})
		}
	})
	
	// Debug the full email pattern matching
	t.Run("debug full email pattern", func(t *testing.T) {
		pattern := "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
		
		// Test progressively longer portions of the input
		testInputs := []string{
			"j",
			"john",
			"john.",
			"john.doe",
			"john.doe+",
			"john.doe+filter",
			"john.doe+filter@",
			"john.doe+filter@example",
			"john.doe+filter@example.",
			"john.doe+filter@example.com",
		}
		
		for _, testInput := range testInputs {
			t.Run(testInput, func(t *testing.T) {
				rule := Regexp(pattern)
				parser := New(WithPartial(true))
				
				val, matched, err := parser.Parse(rule, testInput)
				t.Logf("Input: %q, Matched: %v, Value: %q, Error: %v", testInput, matched, val, err)
			})
		}
		
		// Also try matching individual parts
		t.Run("parts", func(t *testing.T) {
			parts := []struct {
				pattern string
				input   string
			}{
				{"[a-zA-Z0-9._%+-]+", "john.doe+filter"},
				{"@", "@"},
				{"[a-zA-Z0-9.-]+", "example"},
				{"\\.", "."},
				{"[a-zA-Z]{2,}", "com"},
			}
			
			for _, part := range parts {
				rule := Regexp(part.pattern)
				parser := New()
				val, matched, err := parser.Parse(rule, part.input)
				t.Logf("Pattern: %q, Input: %q, Matched: %v, Value: %q, Error: %v", 
					part.pattern, part.input, matched, val, err)
			}
		})
		
		// Test simpler versions of the pattern
		t.Run("simplified patterns", func(t *testing.T) {
			patterns := []struct {
				pattern string
				input   string
			}{
				// Remove the {2,} quantifier
				{"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]+", "john.doe+filter@example.com"},
				// Use exact quantifier
				{"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{3}", "john.doe+filter@example.com"},
				// Simplify character classes
				{"[a-z]+@[a-z]+\\.[a-z]+", "john@example.com"},
				// Even simpler
				{"a+@a+\\.a+", "aaa@aaa.aaa"},
				// Test the {2,} quantifier separately
				{"a{2,}", "aaa"},
				{"[a-z]{2,}", "abc"},
				// Test character classes with special chars
				{"[a-z.]+", "abc.def"},
				{"[a-z_]+", "abc_def"},
				{"[a-z%]+", "abc%def"},
				{"[a-z._%]+", "abc._%def"},
				{"[._%]+", "._%"},
				{"[a-z0-9._%+-]+", "abc._%+-123"},
				// Test exact character classes from email pattern
				{"[a-zA-Z0-9._%+-]+", "john.doe+filter"},
				{"[a-zA-Z0-9.-]+", "example.com"},
				// Test combinations
				{"[a-zA-Z0-9._%+-]+@", "john.doe+filter@"},
				{"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+", "john.doe+filter@example"},
				// Test with escaped dot
				{"\\.", "."},
				{"\\.[a-z]+", ".com"},  
				{"[a-z]+\\.[a-z]+", "example.com"},
				{"[a-zA-Z0-9.-]+\\.", "example."},
				{"[a-zA-Z0-9.-]+\\.[a-z]+", "example.com"},
				{"[a-zA-Z0-9.-]+\\.[a-zA-Z]+", "example.com"},
				// Build up to the full pattern
				{"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.", "john.doe+filter@example."},
				// Test dot in character class issue
				{"[a-z.-]+", "example.com"},
				{"[a-z-]+", "example-com"}, 
				{"[a-z-]+\\.", "example-."},
				{"[.-]+", "..--"},
				{"[.-]+\\.", "..--."},
				{"[a-zA-Z0-9-]+\\.", "example-."},  // Without dot in char class
			}
			
			for _, p := range patterns {
				rule := Regexp(p.pattern)
				parser := New(WithPartial(true))
				val, matched, err := parser.Parse(rule, p.input)
				t.Logf("Pattern: %q, Input: %q, Matched: %v, Value: %q, Error: %v", 
					p.pattern, p.input, matched, val, err)
			}
		})
	})
}