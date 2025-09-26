package main

import (
	"fmt"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
	"io/ioutil"
)

// TypeCheckFile demonstrates type inference on an actual AILANG file
func TypeCheckFile(filename string) {
	// Read file
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Parse the file
	l := lexer.New(string(content), filename)
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		fmt.Println("Parser errors:")
		for _, err := range p.Errors() {
			fmt.Printf("  • %s\n", err)
		}
		return
	}

	fmt.Printf("Successfully parsed %s\n", filename)
	fmt.Println("----------------------------------------")

	// Type check the program
	tc := types.NewTypeChecker()
	typedProgram, err := tc.CheckProgram(program)
	if err != nil {
		fmt.Printf("Type checking errors: %v\n", err)
	} else {
		fmt.Println("Type checking successful!")
		if typedProgram != nil {
			fmt.Printf("Program contains %d top-level declarations\n", len(typedProgram.Statements))
		}
	}
}

// DemoManualTypeInference shows type inference on manually constructed AST
func DemoManualTypeInference() {
	fmt.Println("\nManual Type Inference Demo")
	fmt.Println("==========================\n")

	ctx := types.NewInferenceContext()
	ctx.SetEnv(types.NewTypeEnvWithBuiltins())

	// Create AST for: let x = 5 in let y = x + 3 in y
	expr := &ast.Let{
		Name: "x",
		Value: &ast.Literal{Kind: ast.IntLit, Value: 5},
		Body: &ast.Let{
			Name: "y",
			Value: &ast.BinaryOp{
				Left: &ast.Identifier{Name: "x"},
				Op: "+",
				Right: &ast.Literal{Kind: ast.IntLit, Value: 3},
			},
			Body: &ast.Identifier{Name: "y"},
		},
	}

	fmt.Println("Expression: let x = 5 in let y = x + 3 in y")
	
	// Infer type
	typ, eff, err := ctx.Infer(expr)
	if err != nil {
		fmt.Printf("Type inference error: %v\n", err)
		return
	}

	// Solve constraints
	sub, unsolved, err := ctx.SolveConstraints()
	if err != nil {
		fmt.Printf("Constraint solving error: %v\n", err)
		return
	}

	// Apply substitution
	finalType := types.ApplySubstitution(sub, typ)
	finalEffects := types.ApplySubstitution(sub, eff)

	fmt.Printf("Inferred type: %s\n", finalType)
	fmt.Printf("Effects: %s\n", finalEffects)
	
	if len(unsolved) > 0 {
		fmt.Println("Unsolved constraints:")
		for _, c := range unsolved {
			fmt.Printf("  • %s\n", c)
		}
	}
}

func main() {
	// First run manual demos
	DemoManualTypeInference()

	// Then try to type check actual files
	fmt.Println("\n\nType Checking Files")
	fmt.Println("===================\n")

	// Try the minimal demo
	fmt.Println("Checking type_demo_minimal.ail:")
	TypeCheckFile("examples/type_demo_minimal.ail")
}