package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_v2 <file.ail>")
		os.Exit(1)
	}

	// Read file
	content, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	input := string(content)
	fmt.Printf("=== Running AILANG v2.0 Pipeline ===\n")
	fmt.Printf("Input file: %s\n\n", os.Args[1])

	// Phase 1: Parse
	fmt.Println("1. Parsing...")
	l := lexer.New(input, os.Args[1])
	p := parser.New(l)
	prog := p.Parse()
	if len(p.Errors()) > 0 {
		fmt.Println("❌ Parse errors:")
		for _, err := range p.Errors() {
			fmt.Printf("   %v\n", err)
		}
		os.Exit(1)
	}
	fmt.Println("✅ Parsed successfully")

	// Phase 2: Elaborate to Core
	fmt.Println("\n2. Elaborating to Core AST...")
	elab := elaborate.NewElaborator()
	coreProg, err := elab.Elaborate(prog)
	if err != nil {
		fmt.Printf("❌ Elaboration error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Elaborated to Core AST")

	// Phase 3: Type Check
	fmt.Println("\n3. Type checking...")
	tc := types.NewCoreTypeChecker()
	typedProg, err := tc.CheckCoreProgram(coreProg)
	if err != nil {
		// Check if it's just unsolved class constraints
		errStr := err.Error()
		if contains(errStr, "unsolved type class constraint") && contains(errStr, "Num") {
			fmt.Println("⚠️  Type checking passed with warnings:")
			fmt.Println("   - Num class constraints collected but not resolved")
			fmt.Println("   - This is expected in Phase 1 (class instances not implemented yet)")
			
			// Show inferred type if available
			if typedProg != nil && len(typedProg.Decls) > 0 {
				resultType := typedProg.Decls[0].GetType()
				if resultType != nil {
					fmt.Printf("\n📊 Inferred type: %v\n", resultType)
				}
			}
		} else {
			fmt.Printf("❌ Type checking error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("✅ Type checked successfully")
		
		// Show inferred type
		if typedProg != nil && len(typedProg.Decls) > 0 {
			resultType := typedProg.Decls[0].GetType()
			if resultType != nil {
				fmt.Printf("\n📊 Inferred type: %v\n", resultType)
			}
		}
	}

	fmt.Println("\n=== Pipeline Complete ===")
	fmt.Println("Note: Evaluation not yet connected in Phase 1")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}