package peggysue

import (
	"testing"
)

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
