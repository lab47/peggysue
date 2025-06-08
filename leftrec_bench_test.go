package peggysue

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkLeftRecursive tests the performance of left recursive rules
func BenchmarkLeftRecursive(b *testing.B) {
	// Test with different expression lengths
	testCases := []struct {
		name     string
		numTerms int
	}{
		{"3terms", 3},
		{"5terms", 5},
		{"10terms", 10},
		{"20terms", 20},
		{"50terms", 50},
		{"100terms", 100},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Build expression like "1 + 2 + 3 + ..."
			terms := make([]string, tc.numTerms)
			for i := 0; i < tc.numTerms; i++ {
				terms[i] = strconv.Itoa(i + 1)
			}
			input := strings.Join(terms, " + ")

			// Create the grammar once
			grammar := createLeftRecursiveArithmeticGrammar()
			parser := New()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				val, matched, err := parser.Parse(grammar, input)
				if !matched || err != nil {
					b.Fatalf("Failed to parse: matched=%v, err=%v", matched, err)
				}
				if val == nil {
					b.Fatal("Expected non-nil value")
				}
			}
		})
	}
}

// BenchmarkLeftRecursiveVsRightRecursive compares left-recursive and right-recursive implementations
func BenchmarkLeftRecursiveVsRightRecursive(b *testing.B) {
	numTerms := 20
	terms := make([]string, numTerms)
	for i := 0; i < numTerms; i++ {
		terms[i] = strconv.Itoa(i + 1)
	}
	input := strings.Join(terms, " + ")

	b.Run("LeftRecursive", func(b *testing.B) {
		grammar := createLeftRecursiveArithmeticGrammar()
		parser := New()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, matched, err := parser.Parse(grammar, input)
			if !matched || err != nil {
				b.Fatalf("Failed to parse: matched=%v, err=%v", matched, err)
			}
		}
	})

	b.Run("RightRecursive", func(b *testing.B) {
		grammar := createRightRecursiveArithmeticGrammar()
		parser := New()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, matched, err := parser.Parse(grammar, input)
			if !matched || err != nil {
				b.Fatalf("Failed to parse: matched=%v, err=%v", matched, err)
			}
		}
	})
}

// BenchmarkDeeplyNestedLeftRecursion tests performance with deeply nested expressions
func BenchmarkDeeplyNestedLeftRecursion(b *testing.B) {
	testCases := []struct {
		name  string
		depth int
	}{
		{"depth5", 5},
		{"depth10", 10},
		{"depth20", 20},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Build nested expression like "((1 + 2) * 3) + 4..."
			input := buildNestedExpression(tc.depth)
			grammar := createLeftRecursiveComplexGrammar()
			parser := New()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, matched, err := parser.Parse(grammar, input)
				if !matched || err != nil {
					b.Fatalf("Failed to parse: matched=%v, err=%v", matched, err)
				}
			}
		})
	}
}

// Helper function to create a left-recursive arithmetic grammar
// expr := expr '+' term | term
// term := number
func createLeftRecursiveArithmeticGrammar() Rule {
	expr := R("expr")
	term := R("term")
	number := R("number")

	// Define number as a sequence of digits
	number.Set(Transform(Plus(Range('0', '9')), func(s string) any {
		n, _ := strconv.Atoi(s)
		return n
	}))

	// Define term as just a number (for simplicity)
	term.Set(number)

	// Define expr with left recursion: expr := expr '+' term | term
	expr.Set(Or(
		Action(Seq(
			Named("left", expr),
			Maybe(Star(S(" "))),
			S("+"),
			Maybe(Star(S(" "))),
			Named("right", term),
		), func(v Values) any {
			left := v.Get("left").(int)
			right := v.Get("right").(int)
			return left + right
		}),
		term,
	))

	if !expr.LeftRecursive() {
		panic("Expected left recursion in arithmetic grammar")
	}

	return expr
}

// Helper function to create a right-recursive arithmetic grammar for comparison
// expr := term '+' expr | term
// term := number
func createRightRecursiveArithmeticGrammar() Rule {
	expr := R("expr")
	term := R("term")
	number := R("number")

	// Define number as a sequence of digits
	number.Set(Transform(Plus(Range('0', '9')), func(s string) any {
		n, _ := strconv.Atoi(s)
		return n
	}))

	// Define term as just a number
	term.Set(number)

	// Define expr with right recursion: expr := term '+' expr | term
	expr.Set(Or(
		Action(Seq(
			Named("left", term),
			Maybe(Star(S(" "))),
			S("+"),
			Maybe(Star(S(" "))),
			Named("right", expr),
		), func(v Values) any {
			left := v.Get("left").(int)
			right := v.Get("right").(int)
			return left + right
		}),
		term,
	))

	if expr.LeftRecursive() {
		panic("Expected right recursion in arithmetic grammar")
	}

	return expr
}

// Helper function to create a more complex left-recursive grammar with multiple operators
// expr := expr '+' term | expr '-' term | term
// term := term '*' factor | term '/' factor | factor
// factor := '(' expr ')' | number
func createLeftRecursiveComplexGrammar() Rule {
	expr := R("expr")
	term := R("term")
	factor := R("factor")
	number := R("number")

	// Define number
	number.Set(Transform(Plus(Range('0', '9')), func(s string) any {
		n, _ := strconv.Atoi(s)
		return n
	}))

	// Define factor: '(' expr ')' | number
	factor.Set(Or(
		Action(Seq(
			S("("),
			Maybe(Star(S(" "))),
			Named("expr", expr),
			Maybe(Star(S(" "))),
			S(")"),
		), func(v Values) any {
			return v.Get("expr")
		}),
		number,
	))

	// Define term with left recursion: term '*' factor | term '/' factor | factor
	term.Set(Or(
		Action(Seq(
			Named("left", term),
			Maybe(Star(S(" "))),
			S("*"),
			Maybe(Star(S(" "))),
			Named("right", factor),
		), func(v Values) any {
			left := v.Get("left").(int)
			right := v.Get("right").(int)
			return left * right
		}),
		Action(Seq(
			Named("left", term),
			Maybe(Star(S(" "))),
			S("/"),
			Maybe(Star(S(" "))),
			Named("right", factor),
		), func(v Values) any {
			left := v.Get("left").(int)
			right := v.Get("right").(int)
			if right == 0 {
				return 0 // Avoid division by zero in benchmarks
			}
			return left / right
		}),
		factor,
	))

	// Define expr with left recursion: expr '+' term | expr '-' term | term
	expr.Set(Or(
		Action(Seq(
			Named("left", expr),
			Maybe(Star(S(" "))),
			S("+"),
			Maybe(Star(S(" "))),
			Named("right", term),
		), func(v Values) any {
			left := v.Get("left").(int)
			right := v.Get("right").(int)
			return left + right
		}),
		Action(Seq(
			Named("left", expr),
			Maybe(Star(S(" "))),
			S("-"),
			Maybe(Star(S(" "))),
			Named("right", term),
		), func(v Values) any {
			left := v.Get("left").(int)
			right := v.Get("right").(int)
			return left - right
		}),
		term,
	))

	return expr
}

// Helper function to build nested expressions
func buildNestedExpression(depth int) string {
	if depth <= 1 {
		return "1"
	}

	expr := "1"
	for i := 2; i <= depth; i++ {
		if i%2 == 0 {
			expr = "(" + expr + " + " + strconv.Itoa(i) + ")"
		} else {
			expr = "(" + expr + " * " + strconv.Itoa(i) + ")"
		}
	}
	return expr
}

// BenchmarkMemoization tests the effectiveness of memoization in left-recursive rules
func BenchmarkMemoization(b *testing.B) {
	// Create an expression with repeated sub-expressions
	// This tests how well memoization works
	input := "1 + 2 + 3 + 1 + 2 + 3 + 1 + 2 + 3"

	grammar := createLeftRecursiveArithmeticGrammar()
	parser := New()

	b.Run("WithMemoization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, matched, err := parser.Parse(grammar, input)
			if !matched || err != nil {
				b.Fatalf("Failed to parse: matched=%v, err=%v", matched, err)
			}
		}
	})
}

