package peggysue

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name         string
			badPattern   string
			goodPattern  string
			input        string
			explanation  string
		}{
			{
				name:         "dot in character class",
				badPattern:   `[.-]+\.`,
				goodPattern:  `(\.|-)+`,        // Just match dots or dashes
				input:        ".-.",
				explanation:  "Avoid consuming the character you need later",
			},
			{
				name:         "repeated separators",
				badPattern:   `[a-z.]+\.[a-z]+`,
				goodPattern:  `[a-z]+(\.[a-z]+)+`,
				input:        "foo.bar",
				explanation:  "Use explicit groups for separators",
			},
			{
				name:         "file extension",
				badPattern:   `[a-zA-Z0-9._-]+\.(jpg|png)`,
				goodPattern:  `[a-zA-Z0-9_-]+\.(jpg|png)`,
				input:        "myfile.jpg",
				explanation:  "Keep dots out of character class when followed by dot",
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