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
	// Simple AILANG program that uses type classes
	input := `
let x = 2 + 3 in
let y = 5 > 3 in  
let z = 4 == 4 in
x
`

	fmt.Println("=== Testing Class Instance Resolution ===\n")
	fmt.Println("Input program:")
	fmt.Println(input)
	fmt.Println()

	// Step 1: Lex and Parse
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

	fmt.Println("✓ Parsing successful")

	// Step 2: Elaborate to Core
	elaborator := elaborate.NewElaborator()
	
	// Elaborate the whole program
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		log.Fatalf("Elaboration failed: %v", err)
	}
	fmt.Println("✓ Elaboration to Core successful")
	
	// Get the first declaration for testing
	if len(coreProg.Decls) == 0 {
		log.Fatal("No declarations in Core program")
	}
	coreExpr := coreProg.Decls[0]

	// Step 3: Type check WITH instances
	fmt.Println("\n--- Type Checking with Empty Instance Environment ---")
	
	// First, try without instances (should fail)
	tcEmpty := types.NewCoreTypeChecker()
	_, _, err1 := tcEmpty.CheckCoreExpr(coreExpr, types.NewTypeEnvWithBuiltins())
	if err1 != nil {
		fmt.Printf("✗ Expected failure: %v\n", err1)
	} else {
		fmt.Println("✓ Unexpected success without instances")
	}

	fmt.Println("\n--- Type Checking with Preloaded Instances ---")
	
	// Now with instances loaded
	instances := types.LoadBuiltinInstances()
	tcWithInstances := types.NewCoreTypeCheckerWithInstances(instances)
	
	_, _, err2 := tcWithInstances.CheckCoreExpr(coreExpr, types.NewTypeEnvWithBuiltins())
	if err2 != nil {
		fmt.Printf("✗ Type checking failed: %v\n", err2)
		return
	}

	fmt.Println("✓ Type checking successful!")
	
	// Show what instances were needed
	fmt.Println("\n--- Required Instances ---")
	testInstances(instances)
}

func testInstances(env *types.InstanceEnv) {
	// Test what instances are available and used
	testCases := []struct {
		class string
		typ   types.Type
		op    string
	}{
		{"Num", types.TInt, "for + operator"},
		{"Ord", types.TInt, "for > operator"},
		{"Eq", types.TInt, "for == operator"},
	}

	for _, tc := range testCases {
		inst, err := env.Lookup(tc.class, tc.typ)
		if err != nil {
			fmt.Printf("✗ %s[%s] not found %s\n", tc.class, tc.typ, tc.op)
		} else {
			fmt.Printf("✓ %s[%s] found %s\n", tc.class, tc.typ, tc.op)
			if tc.class == "Eq" && inst.Dict["eq"] == "derived_eq_from_ord_int" {
				fmt.Printf("  (derived from Ord[Int])\n")
			}
		}
	}
	
	// Test missing instance
	_, err := env.Lookup("Num", types.TString)
	if err != nil {
		fmt.Printf("\n✓ Correctly rejects Num[String]: %v\n", err.(*types.MissingInstanceError).Error())
	}
}