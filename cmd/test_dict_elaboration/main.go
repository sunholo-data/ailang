package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

func main() {
	fmt.Println("=== Testing Dictionary Elaboration ===")
	
	testCases := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "Simple addition",
			input: `2 + 3`,
			desc:  "Should transform to DictApp(dict_Num_Int, \"add\", [2, 3])",
		},
		{
			name:  "Mixed operators",
			input: `let x = 5 > 3 in if x then 1 + 2 else 4 == 4`,
			desc:  "Should transform comparisons and equality to Ord/Eq dict calls",
		},
		{
			name:  "Nested arithmetic",
			input: `(1 + 2) * (3 - 4)`,
			desc:  "Should transform all operators to ANF-bound dict calls",
		},
	}
	
	for _, tc := range testCases {
		fmt.Printf("Test: %s\n", tc.name)
		fmt.Printf("Input: %s\n", tc.input)
		fmt.Printf("Expected: %s\n", tc.desc)
		fmt.Println()
		
		runElaborationTest(tc.input)
		fmt.Println()
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println()
	}
}

func runElaborationTest(input string) {
	// Step 1: Parse
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
	
	// Step 2: Elaborate to Core
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		log.Printf("Elaboration failed: %v", err)
		return
	}
	
	fmt.Println("Core (before dictionary transformation):")
	printCore(coreProg)
	
	// Step 3: Type check
	instances := types.LoadBuiltinInstances()
	tc := types.NewCoreTypeCheckerWithInstances(instances)
	
	// Type check the first declaration
	if len(coreProg.Decls) == 0 {
		fmt.Println("No declarations to type check")
		return
	}
	
	typed, _, err := tc.CheckCoreExpr(coreProg.Decls[0], types.NewTypeEnvWithBuiltins())
	if err != nil {
		fmt.Printf("Type checking failed: %v\n", err)
		return
	}
	
	fmt.Printf("\nType: %v\n", typed.GetType())
	
	// Get resolved constraints
	resolved := tc.GetResolvedConstraints()
	fmt.Printf("\nResolved constraints: %d\n", len(resolved))
	for nodeID, rc := range resolved {
		fmt.Printf("  NodeID %d: %s[%s].%s\n", 
			nodeID, rc.ClassName, types.NormalizeTypeName(rc.Type), rc.Method)
	}
	
	// Step 4: Dictionary elaboration
	dictProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
	if err != nil {
		log.Printf("Dictionary elaboration failed: %v", err)
		return
	}
	
	fmt.Println("\nCore (after dictionary transformation):")
	printCore(dictProg)
}

func printCore(prog *core.Program) {
	for i, decl := range prog.Decls {
		if i > 0 {
			fmt.Println()
		}
		printCoreExpr(decl, 0)
	}
}

func printCoreExpr(expr core.CoreExpr, indent int) {
	prefix := strings.Repeat("  ", indent)
	
	switch e := expr.(type) {
	case *core.Let:
		fmt.Printf("%slet %s =\n", prefix, e.Name)
		printCoreExpr(e.Value, indent+1)
		fmt.Printf("%sin\n", prefix)
		printCoreExpr(e.Body, indent)
		
	case *core.DictApp:
		fmt.Printf("%sDictApp(\n", prefix)
		fmt.Printf("%s  dict: ", prefix)
		printCoreExpr(e.Dict, 0)
		fmt.Printf("%s  method: %s\n", prefix, e.Method)
		fmt.Printf("%s  args: [", prefix)
		for i, arg := range e.Args {
			if i > 0 {
				fmt.Printf(", ")
			}
			printCoreExpr(arg, 0)
		}
		fmt.Printf("]\n%s)", prefix)
		
	case *core.DictRef:
		fmt.Printf("dict_%s_%s", e.ClassName, e.TypeName)
		
	case *core.Var:
		fmt.Printf("%s", e.Name)
		
	case *core.Lit:
		fmt.Printf("%v", e.Value)
		
	case *core.BinOp:
		fmt.Printf("%s(", prefix)
		printCoreExpr(e.Left, 0)
		fmt.Printf(" %s ", e.Op)
		printCoreExpr(e.Right, 0)
		fmt.Printf(")")
		
	case *core.If:
		fmt.Printf("%sif ", prefix)
		printCoreExpr(e.Cond, 0)
		fmt.Printf(" then\n")
		printCoreExpr(e.Then, indent+1)
		fmt.Printf("\n%selse\n", prefix)
		printCoreExpr(e.Else, indent+1)
		
	default:
		fmt.Printf("%s%s", prefix, expr.String())
	}
}