package main

import (
	"fmt"
	"log"

	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

func main() {
	fmt.Println("=== Testing Numeric Literal Defaulting ===\n")
	
	// Test cases showing numeric literals that should default
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Ambiguous numeric literal",
			input:    `42`,
			expected: "Int (defaulted from Num)",
		},
		{
			name:     "Fractional literal",
			input:    `3.14`,
			expected: "Float (defaulted from Fractional)",
		},
		{
			name:     "Arithmetic with literals",
			input:    `let x = 10 + 20 in x`,
			expected: "Int (defaulted from Num)",
		},
		{
			name:     "Mixed but compatible",
			input:    `let x = 2.5 * 4.0 in x`,
			expected: "Float (both Fractional)",
		},
	}
	
	for _, tc := range testCases {
		fmt.Printf("Test: %s\n", tc.name)
		fmt.Printf("Input: %s\n", tc.input)
		runDefaultingTest(tc.input)
		fmt.Println()
	}
	
	// Test with tracing enabled
	fmt.Println("=== With Debug Tracing ===")
	runWithTracing(`
let x = 42 in
let y = 3.14 in
let z = x + 1 in
z
`)
}

func runDefaultingTest(input string) {
	// Parse
	l := lexer.New(input, "test")
	p := parser.New(l)
	program := p.Parse()
	
	if errs := p.Errors(); len(errs) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range errs {
			fmt.Printf("  • %s\n", err)
		}
		return
	}
	
	// Elaborate to Core
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		log.Printf("Elaboration failed: %v", err)
		return
	}
	
	if len(coreProg.Decls) == 0 {
		fmt.Println("No declarations to type check")
		return
	}
	
	// Type check with instances and defaulting
	instances := types.LoadBuiltinInstances()
	tc := types.NewCoreTypeCheckerWithInstances(instances)
	
	// Use standard defaulting config
	defaultingConfig := types.NewDefaultingConfig()
	tc.SetDefaultingConfig(defaultingConfig)
	
	typed, _, err := tc.CheckCoreExpr(coreProg.Decls[0], types.NewTypeEnvWithBuiltins())
	if err != nil {
		fmt.Printf("✗ Type checking failed: %v\n", err)
		return
	}
	
	// Show the inferred type
	typ := typed.GetType()
	var typeStr string
	if stringer, ok := typ.(fmt.Stringer); ok {
		typeStr = stringer.String()
	} else {
		typeStr = fmt.Sprintf("%v", typ)
	}
	fmt.Printf("✓ Type: %s\n", typeStr)
	
	// Show if defaulting occurred
	if len(defaultingConfig.Traces) > 0 {
		fmt.Println("  Defaulting applied:")
		for _, trace := range defaultingConfig.Traces {
			fmt.Printf("    • %s[%s] → %s\n", 
				trace.ClassName, trace.TypeVar, trace.Default.String())
		}
	}
}

func runWithTracing(input string) {
	fmt.Println("Input:")
	fmt.Println(input)
	
	// Parse
	l := lexer.New(input, "test")
	p := parser.New(l)
	program := p.Parse()
	
	if errs := p.Errors(); len(errs) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range errs {
			fmt.Printf("  • %s\n", err)
		}
		return
	}
	
	// Elaborate
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		log.Printf("Elaboration failed: %v", err)
		return
	}
	
	if len(coreProg.Decls) == 0 {
		fmt.Println("No declarations to type check")
		return
	}
	
	// Type check with debug mode
	instances := types.LoadBuiltinInstances()
	tc := types.NewCoreTypeCheckerWithInstances(instances)
	tc.SetDebugMode(true) // Enable debug tracing
	
	fmt.Println("\n--- Type Checking with Trace ---")
	typed, _, err := tc.CheckCoreExpr(coreProg.Decls[0], types.NewTypeEnvWithBuiltins())
	if err != nil {
		fmt.Printf("✗ Type checking failed: %v\n", err)
		return
	}
	
	typ := typed.GetType()
	var typeStr string
	if stringer, ok := typ.(fmt.Stringer); ok {
		typeStr = stringer.String()
	} else {
		typeStr = fmt.Sprintf("%v", typ)
	}
	fmt.Printf("\n✓ Final type: %s\n", typeStr)
}