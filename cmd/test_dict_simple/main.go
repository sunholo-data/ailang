package main

import (
	"fmt"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/types"
)

func main() {
	fmt.Println("=== Testing Dictionary Elaboration (Simple) ===")
	
	// Create a simple Core expression: 2 + 3
	binop := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 1, CoreSpan: ast.Pos{}, OrigSpan: ast.Pos{}},
		Op:       "+",
		Left:     &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: 2},
		Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 3}, Kind: core.IntLit, Value: 3},
	}
	
	prog := &core.Program{
		Decls: []core.CoreExpr{binop},
	}
	
	fmt.Println("Core (before dictionary transformation):")
	fmt.Println(binop.String())
	fmt.Println()
	
	// Create resolved constraints manually
	resolved := map[uint64]*types.ResolvedConstraint{
		1: {
			NodeID:    1,
			ClassName: "Num",
			Type:      types.TInt,
			Method:    "add",
		},
	}
	
	fmt.Println("Resolved constraints:")
	for nodeID, rc := range resolved {
		fmt.Printf("  NodeID %d: %s[%s].%s\n", 
			nodeID, rc.ClassName, types.NormalizeTypeName(rc.Type), rc.Method)
	}
	fmt.Println()
	
	// Apply dictionary elaboration
	dictProg, err := elaborate.ElaborateWithDictionaries(prog, resolved)
	if err != nil {
		fmt.Printf("Dictionary elaboration failed: %v\n", err)
		return
	}
	
	fmt.Println("Core (after dictionary transformation):")
	printCore(dictProg.Decls[0], 0)
	fmt.Println()
	
	// Test another operator: comparison
	fmt.Println()
	fmt.Println("=== Testing Comparison Operator ===")
	
	compOp := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 4},
		Op:       "<",
		Left:     &core.Lit{CoreNode: core.CoreNode{NodeID: 5}, Kind: core.IntLit, Value: 5},
		Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 6}, Kind: core.IntLit, Value: 10},
	}
	
	prog2 := &core.Program{
		Decls: []core.CoreExpr{compOp},
	}
	
	fmt.Println("Core (before):")
	fmt.Println(compOp.String())
	
	resolved2 := map[uint64]*types.ResolvedConstraint{
		4: {
			NodeID:    4,
			ClassName: "Ord",
			Type:      types.TInt,
			Method:    "lt",
		},
	}
	
	dictProg2, _ := elaborate.ElaborateWithDictionaries(prog2, resolved2)
	
	fmt.Println("\nCore (after):")
	printCore(dictProg2.Decls[0], 0)
}

func printCore(expr core.CoreExpr, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	
	switch e := expr.(type) {
	case *core.Let:
		fmt.Printf("%slet %s =\n", prefix, e.Name)
		printCore(e.Value, indent+1)
		fmt.Printf("%sin\n", prefix)
		printCore(e.Body, indent)
		
	case *core.DictApp:
		fmt.Printf("%sDictApp(dict: ", prefix)
		printCore(e.Dict, 0)
		fmt.Printf(", method: \"%s\", args: [", e.Method)
		for i, arg := range e.Args {
			if i > 0 {
				fmt.Printf(", ")
			}
			printCore(arg, 0)
		}
		fmt.Printf("])")
		
	case *core.DictRef:
		fmt.Printf("dict_%s_%s", e.ClassName, e.TypeName)
		
	case *core.Var:
		fmt.Printf("%s", e.Name)
		
	case *core.Lit:
		fmt.Printf("%v", e.Value)
		
	case *core.BinOp:
		fmt.Printf("(")
		printCore(e.Left, 0)
		fmt.Printf(" %s ", e.Op)
		printCore(e.Right, 0)
		fmt.Printf(")")
		
	default:
		fmt.Printf("%s", expr.String())
	}
}