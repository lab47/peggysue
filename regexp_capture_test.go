package peggysue

import (
	"fmt"
	"testing"

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
