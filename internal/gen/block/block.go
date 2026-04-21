// Package block provides a language-neutral Block IR for flattening nested
// let bindings. This intermediate representation sits between Core AST and
// target language emission (e.g., Go), enabling flat code generation without
// deeply nested closures.
//
// M-CODEGEN-V2: Eliminate nested IIFEs by lowering let chains to flat blocks.
package block

import "github.com/sunholo-data/ailang/internal/core"

// Block represents a sequence of variable bindings followed by a final expression.
// This is the result of flattening nested let expressions from Core AST.
//
// Example Core:
//
//	Let{x, 1, Let{y, 2, Let{z, 3, x + y + z}}}
//
// Becomes Block:
//
//	Block{
//	  Stmts: [{x, 1}, {y, 2}, {z, 3}],
//	  FinalExpr: x + y + z
//	}
type Block struct {
	// Stmts contains variable bindings in evaluation order.
	// Each Stmt declares a variable and its value expression.
	Stmts []Stmt

	// FinalExpr is the body expression after all bindings are established.
	// This is what the block evaluates to.
	FinalExpr core.CoreExpr
}

// Stmt represents a single variable binding: var name = value.
type Stmt struct {
	// Name is the variable name being bound.
	Name string

	// Value is the expression whose result is bound to Name.
	Value core.CoreExpr
}

// IsEmpty returns true if the block has no statements (just a final expression).
func (b *Block) IsEmpty() bool {
	return len(b.Stmts) == 0
}

// Len returns the number of statements in the block.
func (b *Block) Len() int {
	return len(b.Stmts)
}
