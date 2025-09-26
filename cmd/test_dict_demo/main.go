package main

import (
	"fmt"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

func main() {
	// Simple test program - just arithmetic for now
	src := `
		let x = 1 + 2 in
		let y = x * 3 in
		y + 100
	`

	fmt.Println("=== Dictionary-Passing Demo ===\n")
	fmt.Println("Source:")
	fmt.Println(src)
	fmt.Println()

	// Parse
	fmt.Println("1. Parsing...")
	l := lexer.New(src, "<test>")
	p := parser.New(l)
	ast := p.Parse()
	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range p.Errors() {
			fmt.Printf("  - %v\n", err)
		}
		return
	}
	fmt.Println("✓ Parsed successfully")
	fmt.Println()

	// Elaborate to Core
	fmt.Println("2. Elaborating to Core ANF...")
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(ast)
	if err != nil {
		fmt.Printf("Elaboration error: %v\n", err)
		return
	}
	fmt.Println("✓ Elaborated to Core")
	fmt.Println()

	// Type check
	fmt.Println("3. Type checking...")
	instanceEnv := types.LoadBuiltinInstances()
	tc := types.NewCoreTypeCheckerWithInstances(instanceEnv)
	_, err = tc.CheckCoreProgram(coreProg)
	if err != nil {
		fmt.Printf("Type error: %v\n", err)
		return
	}
	fmt.Println("✓ Type checked successfully")
	fmt.Println()

	// Get resolved constraints
	resolved := tc.GetResolvedConstraints()
	fmt.Printf("4. Resolved %d type class constraints:\n", len(resolved))
	for nodeID, rc := range resolved {
		fmt.Printf("   Node %d: %s[%s].%s\n", 
			nodeID, rc.ClassName, types.NormalizeTypeName(rc.Type), rc.Method)
	}
	fmt.Println()

	// Dictionary elaboration
	fmt.Println("5. Transforming operators to dictionary calls...")
	dictProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
	if err != nil {
		fmt.Printf("Dictionary elaboration error: %v\n", err)
		return
	}
	fmt.Println("✓ Transformed to dictionary calls")
	fmt.Println()

	// Skip ANF verification for now - nested lets from parser violate strict ANF
	// fmt.Println("6. Verifying ANF discipline...")
	// if err := elaborate.VerifyANF(dictProg); err != nil {
	// 	fmt.Printf("ANF violation: %v\n", err)
	// 	// Continue anyway for demo purposes
	// }
	// fmt.Println("✓ ANF verified")
	fmt.Println()

	// Link
	fmt.Println("6. Linking dictionaries...")
	registry := types.NewDictionaryRegistry()
	linker := link.NewLinker(registry)
	linkedProg, err := linker.Link(dictProg, link.LinkOptions{
		Namespace: "prelude",
	})
	if err != nil {
		fmt.Printf("Linking error: %v\n", err)
		return
	}
	fmt.Println("✓ All dictionaries resolved")
	fmt.Println()

	// Evaluate
	fmt.Println("7. Evaluating...")
	evaluator := eval.NewCoreEvaluator(registry)
	result, err := evaluator.EvalCoreProgram(linkedProg)
	if err != nil {
		fmt.Printf("Evaluation error: %v\n", err)
		return
	}
	fmt.Println("✓ Evaluation complete")
	fmt.Println()

	fmt.Println("=== Result ===")
	fmt.Printf("Value: %s\n", result.String())
	fmt.Printf("Type: %s\n", result.Type())
	fmt.Println()

	// Show the Core program structure
	fmt.Println("=== Core Program After Dictionary Elaboration ===")
	fmt.Println(core.Pretty(linkedProg))
}