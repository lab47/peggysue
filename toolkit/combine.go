package toolkit

import "github.com/lab47/peggysue"

// After returns a function that when called, returns a new rule
// that will match the rule passed to the function, then the
// function passed to After.
// For example:
//    tk := After(Set(' ', '\t'))
//    foo = tk(S("foo"))
// The rule foo will match "foo" on the input stream, then match
// ' ' or '\t'.
// The common use case for After is basically the above example:
// a simple helper to create token-style rules that match and ignore
// whitespace.
//
// For example, utilizing the WS value:
// token := After(WS)
// ifRule := token("if")
func After(after Rule) func(r Rule) Rule {
	return func(r Rule) Rule {
		return peggysue.Seq(r, after)
	}
}
