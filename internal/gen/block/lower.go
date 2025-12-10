package block

import "github.com/sunholo/ailang/internal/core"

// Lower transforms a Core expression into a Block by extracting top-level
// Let bindings into a flat sequence of statements.
//
// This ONLY extracts consecutive top-level Let bindings. Nested Let expressions
// inside Value positions are preserved - they will become IIFEs when generated,
// but at most ONE level deep (not O(n) nesting).
//
// Evaluation order is preserved: bindings appear in the order they were declared
// in the original Core AST.
//
// Examples:
//
//	Lower(Let{x, 1, Let{y, 2, x+y}})
//	→ Block{Stmts: [{x,1}, {y,2}], FinalExpr: x+y}
//
//	Lower(Let{x, Let{y, 1, y}, x})  // Let inside Value
//	→ Block{Stmts: [{x, Let{y,1,y}}], FinalExpr: x}  // Inner Let preserved!
//
//	Lower(App{f, x})  // Non-let expression
//	→ Block{Stmts: [], FinalExpr: App{f, x}}
func Lower(expr core.CoreExpr) *Block {
	var stmts []Stmt
	current := expr

	// Extract consecutive Let bindings from top level
	for {
		let, ok := current.(*core.Let)
		if !ok {
			break
		}

		// Add this binding to the flat list
		stmts = append(stmts, Stmt{
			Name:  let.Name,
			Value: let.Value, // Value is NOT recursively lowered - preserves inner Lets
		})

		// Move to the body for next iteration
		current = let.Body
	}

	return &Block{
		Stmts:     stmts,
		FinalExpr: current,
	}
}

// LowerLetRec transforms a LetRec expression into a Block.
// LetRec bindings are all added as statements, preserving mutual recursion capability.
//
// Note: LetRec bindings may reference each other, so order matters for codegen
// (forward declaration pattern in Go).
func LowerLetRec(letrec *core.LetRec) *Block {
	stmts := make([]Stmt, len(letrec.Bindings))
	for i, bind := range letrec.Bindings {
		stmts[i] = Stmt{
			Name:  bind.Name,
			Value: bind.Value,
		}
	}

	return &Block{
		Stmts:     stmts,
		FinalExpr: letrec.Body,
	}
}
