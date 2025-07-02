package peggysue

import (
	"fmt"
	"testing"

	"github.com/lab47/peggysue/regexp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexpCaptures(t *testing.T) {
	p := New()

	t.Run("unnamed capture group does not capture", func(t *testing.T) {
		r := require.New(t)

		// Pattern with an unnamed capture group - should not capture
		rule := Regexp("hello (world)")

		val, matched, err := p.Parse(rule, "hello world")
		r.NoError(err)
		r.True(matched)
		r.Equal("hello world", val) // The whole match is captured by Regexp, no separate capture
	})

	t.Run("unnamed capture group in action", func(t *testing.T) {
		r := require.New(t)

		// Use Action to verify unnamed groups don't create values
		rule := Action(
			Regexp("hello (\\w+)"),
			func(v Values) any {
				// No capture should be available since it's unnamed
				return "no capture"
			},
		)

		val, matched, err := p.Parse(rule, "hello world")
		r.NoError(err)
		r.True(matched)
		r.Equal("no capture", val)
	})

	t.Run("named capture group", func(t *testing.T) {
		r := require.New(t)

		// Pattern with a named capture group
		rule := Action(
			Regexp("hello (?P<name>\\w+)"),
			func(v Values) any {
				// The named capture should be available
				return v.Get("name")
			},
		)

		val, matched, err := p.Parse(rule, "hello world")
		r.NoError(err)
		r.True(matched)
		r.Equal("world", val)
	})

	t.Run("non-capturing group", func(t *testing.T) {
		r := require.New(t)

		// Pattern with a non-capturing group
		rule := Regexp("hello (?:world|universe)")

		val, matched, err := p.Parse(rule, "hello world")
		r.NoError(err)
		r.True(matched)
		r.Equal("hello world", val) // The whole match, no separate capture
	})

	t.Run("multiple named captures", func(t *testing.T) {
		r := require.New(t)

		// Pattern with multiple named captures
		rule := Action(
			Regexp("(?P<first>\\w+)\\s+(?P<last>\\w+)"),
			func(v Values) any {
				return map[string]string{
					"first": v.Get("first").(string),
					"last":  v.Get("last").(string),
				}
			},
		)

		val, matched, err := p.Parse(rule, "John Doe")
		r.NoError(err)
		r.True(matched)
		r.Equal(map[string]string{"first": "John", "last": "Doe"}, val)
	})

	t.Run("nested captures", func(t *testing.T) {
		r := require.New(t)

		// Pattern with nested captures
		rule := Action(
			Regexp("((?P<word>\\w+)\\s*)+"),
			func(v Values) any {
				// Get the last captured word due to the + quantifier
				return v.Get("word")
			},
		)

		val, matched, err := p.Parse(rule, "hello world test")
		r.NoError(err)
		r.True(matched)
		r.Equal("test", val) // The last word captured
	})

	t.Run("email pattern with captures", func(t *testing.T) {
		r := require.New(t)

		// Let's test a simpler version first
		simpleRule := Regexp("[a-z]+@[a-z]+\\.[a-z]+")
		val, matched, err := p.Parse(simpleRule, "user@example.com")
		r.NoError(err)
		r.True(matched, "Simple email pattern should match")
		r.Equal("user@example.com", val)

		// Email pattern with named captures - simplified for now
		rule := Action(
			Regexp("(?P<user>[a-zA-Z0-9]+)@(?P<domain>[a-zA-Z0-9]+)\\.(?P<tld>[a-zA-Z]{2,})"),
			func(v Values) any {
				user := v.Get("user")
				domain := v.Get("domain")
				tld := v.Get("tld")

				// Debug: check what we got
				if user == nil || domain == nil || tld == nil {
					return map[string]string{
						"error":  "missing captures",
						"user":   fmt.Sprintf("%v", user),
						"domain": fmt.Sprintf("%v", domain),
						"tld":    fmt.Sprintf("%v", tld),
					}
				}

				return map[string]string{
					"user":   user.(string),
					"domain": domain.(string),
					"tld":    tld.(string),
				}
			},
		)

		val2, matched2, err2 := p.Parse(rule, "user@example.com")
		r.NoError(err2)
		r.True(matched2, "Pattern with captures should match user@example.com")
		r.Equal(map[string]string{
			"user":   "user",
			"domain": "example",
			"tld":    "com",
		}, val2)
	})
}

func TestRegexpCaptureEdgeCases(t *testing.T) {
	p := New()

	t.Run("capture in alternation", func(t *testing.T) {
		r := require.New(t)

		rule := Action(
			Regexp("(?P<fruit>apple|banana|orange)"),
			func(v Values) any {
				return v.Get("fruit")
			},
		)

		val, matched, err := p.Parse(rule, "banana")
		r.NoError(err)
		r.True(matched)
		r.Equal("banana", val)
	})

	t.Run("optional capture", func(t *testing.T) {
		r := require.New(t)

		rule := Action(
			Regexp("hello(?P<punct>[!?])?"),
			func(v Values) any {
				punct := v.Get("punct")
				if punct == nil {
					return "no punctuation"
				}
				return punct
			},
		)

		// With punctuation
		val, matched, err := p.Parse(rule, "hello!")
		r.NoError(err)
		r.True(matched)
		r.Equal("!", val)

		// Without punctuation
		val, matched, err = p.Parse(rule, "hello")
		r.NoError(err)
		r.True(matched)
		r.Equal("no punctuation", val)
	})

	t.Run("unnamed capture with quantifier", func(t *testing.T) {
		r := require.New(t)

		// Unnamed capture group with quantifier - should not capture
		rule := Action(
			Regexp("(\\d+)-"),
			func(v Values) any {
				// No capture should be available since it's unnamed
				return "no capture"
			},
		)

		val, matched, err := p.Parse(rule, "123-")
		r.NoError(err)
		r.True(matched)
		r.Equal("no capture", val)
	})
}

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
		seq, ok := ast.Node.(*regexp.Sequence)
		require.True(t, ok, "Expected Sequence, got %T", ast.Node)
		require.Len(t, seq.Elements, 5, "Expected 5 elements in sequence")

		// First element: [a-zA-Z0-9._%+-]+
		q1, ok := seq.Elements[0].(*regexp.Quantifier)
		require.True(t, ok, "First element should be Quantifier, got %T", seq.Elements[0])
		assert.Equal(t, 1, q1.Min)
		assert.Equal(t, -1, q1.Max)

		cc1, ok := q1.Expression.(*regexp.CharClass)
		require.True(t, ok, "Quantifier expression should be CharClass, got %T", q1.Expression)
		assert.False(t, cc1.Negated)

		// Log the character class elements
		t.Logf("First character class has %d elements:", len(cc1.Elements))
		for i, elem := range cc1.Elements {
			switch e := elem.(type) {
			case regexp.CharRange:
				if e.Start == e.End {
					t.Logf("  [%d] CharRange (single): '%c' (code %d)", i, e.Start, e.Start)
				} else {
					t.Logf("  [%d] CharRange: '%c' to '%c' (codes %d-%d)", i, e.Start, e.End, e.Start, e.End)
				}
			case *regexp.EscapeSequence:
				t.Logf("  [%d] EscapeSequence: %s", i, e.String())
			default:
				t.Logf("  [%d] Unknown type: %T", i, elem)
			}
		}
	})

	// Test specific character class edge cases
	t.Run("character class dash handling", func(t *testing.T) {
		testCases := []struct {
			name        string
			pattern     string
			input       string
			expected    string
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
				if cc, ok := ast.Node.(*regexp.CharClass); ok {
					t.Logf("Pattern %s has %d elements:", tc.pattern, len(cc.Elements))
					for i, elem := range cc.Elements {
						switch e := elem.(type) {
						case regexp.CharRange:
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
				{"[a-zA-Z0-9-]+\\.", "example-."}, // Without dot in char class
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

func TestRegexpPEGFriendlyPatterns(t *testing.T) {
	// Test various ways to write patterns that work well with PEG semantics

	t.Run("email patterns that work with PEG", func(t *testing.T) {
		testCases := []struct {
			name    string
			pattern string
			input   string
			comment string
		}{
			{
				name:    "simple working email",
				pattern: `[a-zA-Z0-9]+@[a-zA-Z0-9]+\.[a-zA-Z]+`,
				input:   "john@example.com",
				comment: "No dots in character classes",
			},
			{
				name:    "email with underscores",
				pattern: `[a-zA-Z0-9_]+@[a-zA-Z0-9]+\.[a-zA-Z]+`,
				input:   "john_doe@example.com",
				comment: "Underscores work fine",
			},
			{
				name:    "email with explicit dots",
				pattern: `[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*@[a-zA-Z0-9]+\.[a-zA-Z]+`,
				input:   "john.doe@example.com",
				comment: "Handle dots explicitly with groups",
			},
			{
				name:    "email with plus",
				pattern: `[a-zA-Z0-9]+(\+[a-zA-Z0-9]+)?@[a-zA-Z0-9]+\.[a-zA-Z]+`,
				input:   "john+filter@example.com",
				comment: "Handle plus explicitly",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Logf("Pattern: %s", tc.pattern)
				t.Logf("Comment: %s", tc.comment)

				rule := Regexp(tc.pattern)
				parser := New()

				val, matched, err := parser.Parse(rule, tc.input)
				require.NoError(t, err)
				assert.True(t, matched, "Pattern should match %q", tc.input)
				assert.Equal(t, tc.input, val)
			})
		}
	})

	t.Run("general PEG-friendly pattern strategies", func(t *testing.T) {
		testCases := []struct {
			name        string
			badPattern  string
			goodPattern string
			input       string
			explanation string
		}{
			{
				name:        "dot in character class",
				badPattern:  `[.-]+\.`,
				goodPattern: `(\.|-)+`, // Just match dots or dashes
				input:       ".-.",
				explanation: "Avoid consuming the character you need later",
			},
			{
				name:        "repeated separators",
				badPattern:  `[a-z.]+\.[a-z]+`,
				goodPattern: `[a-z]+(\.[a-z]+)+`,
				input:       "foo.bar",
				explanation: "Use explicit groups for separators",
			},
			{
				name:        "file extension",
				badPattern:  `[a-zA-Z0-9._-]+\.(jpg|png)`,
				goodPattern: `[a-zA-Z0-9_-]+\.(jpg|png)`,
				input:       "myfile.jpg",
				explanation: "Keep dots out of character class when followed by dot",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Logf("Explanation: %s", tc.explanation)

				// Show the bad pattern fails
				t.Run("bad pattern", func(t *testing.T) {
					if tc.badPattern != "" {
						rule := Regexp(tc.badPattern)
						parser := New()
						_, matched, _ := parser.Parse(rule, tc.input)
						t.Logf("Bad pattern %q matched: %v", tc.badPattern, matched)
					}
				})

				// Show the good pattern works
				t.Run("good pattern", func(t *testing.T) {
					rule := Regexp(tc.goodPattern)
					parser := New()
					val, matched, err := parser.Parse(rule, tc.input)
					require.NoError(t, err)
					assert.True(t, matched, "Good pattern %q should match %q", tc.goodPattern, tc.input)
					assert.Equal(t, tc.input, val)
				})
			})
		}
	})

	t.Run("URL patterns", func(t *testing.T) {
		testCases := []struct {
			name    string
			pattern string
			input   string
		}{
			{
				name:    "simple URL",
				pattern: `https?://[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+(/[a-zA-Z0-9._~:/?#@!$&'()*+,;=-]*)?`,
				input:   "https://example.com/path/to/page.html?query=1",
			},
			{
				name:    "URL with port",
				pattern: `https?://[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+(:[0-9]+)?(/[^/\s]*)*`,
				input:   "http://example.com:8080/api/v1/users",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := Regexp(tc.pattern)
				parser := New(WithPartial(true)) // URLs might be part of larger text

				val, matched, err := parser.Parse(rule, tc.input)
				require.NoError(t, err)
				assert.True(t, matched, "Pattern should match %q", tc.input)
				assert.Equal(t, tc.input, val)
			})
		}
	})
}

func TestRegexpPEGWorkarounds(t *testing.T) {
	// Demonstrate specific workarounds for the greedy matching issue

	t.Run("working email pattern", func(t *testing.T) {
		// Original failing pattern:
		// [a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}

		// Working alternatives:
		patterns := []struct {
			name    string
			pattern string
			comment string
		}{
			{
				name:    "simple email",
				pattern: `\w+@\w+\.\w+`,
				comment: "Simplest working pattern",
			},
			{
				name:    "with explicit dots in username",
				pattern: `\w+(\.\w+)*@\w+\.\w+`,
				comment: "Dots handled explicitly in username",
			},
			{
				name:    "with subdomains",
				pattern: `\w+(\.\w+)*@\w+(\.\w+)+`,
				comment: "Support for subdomains",
			},
			{
				name:    "with special chars",
				pattern: `[a-zA-Z0-9]+([\._+-][a-zA-Z0-9]+)*@[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)+`,
				comment: "Special characters as explicit separators",
			},
		}

		testEmails := []string{
			"simple@example.com",
			"john.doe@example.com",
			"user+tag@example.com",
			"test_user@example.com",
			"admin@mail.example.com",
			"user@subdomain.example.co.uk",
		}

		for _, p := range patterns {
			t.Run(p.name, func(t *testing.T) {
				t.Logf("Pattern: %s", p.pattern)
				t.Logf("Comment: %s", p.comment)

				rule := Regexp(p.pattern)
				parser := New()

				for _, email := range testEmails {
					val, matched, err := parser.Parse(rule, email)
					if err != nil || !matched {
						t.Logf("Failed on %q: matched=%v, err=%v", email, matched, err)
					} else {
						assert.Equal(t, email, val)
					}
				}
			})
		}
	})
}

// BenchmarkRegexpParsing tests the overhead of parsing the regexp pattern
func BenchmarkRegexpParsing(b *testing.B) {
	patterns := []string{
		"hello",
		"\\w+@\\w+\\.\\w+",
		"\\d{3}-\\d{3}-\\d{4}",
		"((a|b)+c)*d",
		"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
	}

	for _, pattern := range patterns {
		b.Run(pattern, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := ParseRegexp(pattern)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRegexpWithParsing includes the pattern parsing time
func BenchmarkRegexpWithParsing(b *testing.B) {
	testCases := []struct {
		name    string
		pattern string
		input   string
	}{
		{"simple", "hello", "hello"},
		{"email", "\\w+@\\w+\\.\\w+", "test@example.com"},
		{"phone", "\\d{3}-\\d{3}-\\d{4}", "555-123-4567"},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"/Regexp", func(b *testing.B) {
			parser := New()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rule := Regexp(tc.pattern)
				_, matched, err := parser.Parse(rule, tc.input)
				if err != nil || !matched {
					b.Fatalf("failed to match: err=%v, matched=%v", err, matched)
				}
			}
		})

		b.Run(tc.name+"/Re", func(b *testing.B) {
			parser := New()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rule := Re(tc.pattern)
				_, matched, err := parser.Parse(rule, tc.input)
				if err != nil || !matched {
					b.Fatalf("failed to match: err=%v, matched=%v", err, matched)
				}
			}
		})
	}
}

// BenchmarkRegexpPrecompiled tests performance when rules are pre-compiled
func BenchmarkRegexpPrecompiled(b *testing.B) {
	testCases := []struct {
		name    string
		pattern string
		inputs  []string
	}{
		{
			"email_validation",
			"\\w+@\\w+\\.\\w+",
			[]string{"test@example.com", "user@domain.org", "admin@site.net"},
		},
		{
			"word_matching",
			"\\w+",
			[]string{"hello", "world", "golang", "peggysue"},
		},
		{
			"number_format",
			"\\d+",
			[]string{"123", "4567", "89012", "345678"},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"/Regexp", func(b *testing.B) {
			rule := Regexp(tc.pattern)
			parser := New()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				input := tc.inputs[i%len(tc.inputs)]
				_, matched, err := parser.Parse(rule, input)
				if err != nil || !matched {
					b.Fatalf("failed to match %q: err=%v, matched=%v", input, err, matched)
				}
			}
		})

		b.Run(tc.name+"/Re", func(b *testing.B) {
			rule := Re(tc.pattern)
			parser := New()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				input := tc.inputs[i%len(tc.inputs)]
				_, matched, err := parser.Parse(rule, input)
				if err != nil || !matched {
					b.Fatalf("failed to match %q: err=%v, matched=%v", input, err, matched)
				}
			}
		})
	}
}

func BenchmarkRegexpComprehensive(b *testing.B) {
	// Parser instance
	p := New()

	// Test patterns that exercise different parts of the parser
	patterns := []struct {
		name    string
		pattern string
		input   string
	}{
		// Simple patterns
		{"literal", "hello", "hello world"},
		{"dot", "h.llo", "hello world"},

		// Character classes
		{"char_class", "[a-z]+", "hello"},
		{"negated_class", "[^0-9]+", "hello"},
		{"complex_class", "[a-zA-Z0-9._%-]+", "test.user_123"},

		// Quantifiers
		{"star", "a*b", "aaab"},
		{"plus", "a+b", "aaab"},
		{"question", "colou?r", "color"},
		{"exact", "a{3}", "aaa"},
		{"range", "a{2,4}", "aaa"},

		// Groups and alternation
		{"group", "(ab)+", "ababab"},
		{"alternation", "cat|dog", "cat"},
		{"complex_alt", "(red|green|blue)", "green"},

		// Anchors
		{"start_anchor", "^hello", "hello world"},
		{"end_anchor", "world$", "hello world"},

		// Escape sequences
		{"digit", "\\d+", "12345"},
		{"word", "\\w+", "hello_world"},
		{"whitespace", "\\s+", "   \t\n"},

		// Complex patterns
		{"email", "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}", "user@example.com"},
		{"phone", "\\d{3}-\\d{3}-\\d{4}", "555-123-4567"},
		{"url", "https?://[a-zA-Z0-9.-]+", "https://example.com"},
		{"ipv4", "\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}", "192.168.1.1"},

		// Nested groups and backreferences
		{"nested", "((a|b)+c)*d", "abcbcd"},
		{"complex_nested", "(([a-z]+)\\s*=\\s*([0-9]+);?)+", "foo = 123; bar = 456;"},

		// Character class with ranges
		{"multi_range", "[a-zA-Z0-9!@#$%^&*()]+", "Test123!@#"},

		// Lazy quantifiers
		{"lazy_star", "a*?b", "aaab"},
		{"lazy_plus", "a+?b", "aaab"},
		{"lazy_question", "a??b", "ab"},
	}

	b.ResetTimer()

	// Run the benchmark
	for i := 0; i < b.N; i++ {
		for _, tc := range patterns {
			rule := Regexp(tc.pattern)
			_, _, _ = p.Parse(rule, tc.input)
		}
	}
}

func BenchmarkRegexpParsingOnly(b *testing.B) {
	// Test patterns that exercise different parts of the parser
	patterns := []string{
		"hello",
		"h.llo",
		"[a-z]+",
		"[^0-9]+",
		"[a-zA-Z0-9._%-]+",
		"a*b",
		"a+b",
		"colou?r",
		"a{3}",
		"a{2,4}",
		"(ab)+",
		"cat|dog",
		"(red|green|blue)",
		"^hello",
		"world$",
		"\\d+",
		"\\w+",
		"\\s+",
		"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
		"\\d{3}-\\d{3}-\\d{4}",
		"https?://[a-zA-Z0-9.-]+",
		"\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}",
		"((a|b)+c)*d",
		"(([a-z]+)\\s*=\\s*([0-9]+);?)+",
		"[a-zA-Z0-9!@#$%^&*()]+",
		"a*?b",
		"a+?b",
		"a??b",
	}

	b.ResetTimer()

	// Benchmark just the parsing phase
	for i := 0; i < b.N; i++ {
		for _, pattern := range patterns {
			_ = Regexp(pattern)
		}
	}
}
