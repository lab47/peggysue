package peggysue_test

import (
	"fmt"
	"github.com/lab47/peggysue"
)

func ExampleRegexp() {
	// Create a parser
	parser := peggysue.New()
	
	// Simple pattern matching
	emailPattern := peggysue.Regexp(`\w+@\w+\.\w+`)
	val, matched, _ := parser.Parse(emailPattern, "john@example.com")
	fmt.Printf("Email match: %v, value: %v\n", matched, val)
	
	// Pattern with quantifiers
	phonePattern := peggysue.Regexp(`\d{3}-\d{3}-\d{4}`)
	val2, matched2, _ := parser.Parse(phonePattern, "555-123-4567")
	fmt.Printf("Phone match: %v, value: %v\n", matched2, val2)
	
	// Character classes
	hexPattern := peggysue.Regexp(`[0-9a-fA-F]+`)
	val3, matched3, _ := parser.Parse(hexPattern, "deadbeef")
	fmt.Printf("Hex match: %v, value: %v\n", matched3, val3)
	
	// Output:
	// Email match: true, value: john@example.com
	// Phone match: true, value: 555-123-4567
	// Hex match: true, value: deadbeef
}

func ExampleRegexp_anchors() {
	parser := peggysue.New()
	
	// Start anchor
	startPattern := peggysue.Regexp(`^Hello`)
	_, matched, _ := parser.Parse(startPattern, "Hello World")
	fmt.Printf("Start anchor (full): %v\n", matched)
	
	// With partial parsing
	partialParser := peggysue.New(peggysue.WithPartial(true))
	val2, matched2, _ := partialParser.Parse(startPattern, "Hello World")
	fmt.Printf("Start anchor (partial): %v, value: %v\n", matched2, val2)
	
	// End anchor
	endPattern := peggysue.Regexp(`world$`)
	val3, matched3, _ := parser.Parse(endPattern, "world")
	fmt.Printf("End anchor: %v, value: %v\n", matched3, val3)
	
	// Output:
	// Start anchor (full): false
	// Start anchor (partial): true, value: Hello
	// End anchor: true, value: world
}