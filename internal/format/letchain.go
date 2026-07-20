package format

import "github.com/sunholo-data/ailang/internal/ast"

// letchain.go implements the shared maximal-chain flattening used by BOTH comment
// attachment (attach.go) and conditional multi-line emission (expr.go) for
// M-AILANG-FMT-INLINE-INTERIOR. Keeping one flattening definition guarantees the
// attacher and the printer agree on the logical child sequence and its boundary
// indexes — the load-bearing requirement for idempotence (the second pass must
// reconstruct the exact same chain and boundaries).

// letChain is the flattened logical view of a maximal nested `let … in` chain rooted
// at Root. Bindings holds each binding `*ast.Let` in source order (Root first); Tail
// is the terminal non-let expression. The logical child sequence the attacher keys
// on is [Bindings[0], Bindings[1], …, Tail], so there are len(Bindings)+1 children
// and len(Bindings)+2 boundaries (0 … len(Bindings)+1).
//
// The `in` keyword has no AST position (V13: *ast.Let is {Name,Type,Value,Body,Pos}),
// so each binding child's byte range ends at subtreeEnd(binding.Value) and the `in`
// is emitted as part of the separator after the value.
type letChain struct {
	Root     *ast.Let
	Bindings []*ast.Let
	Tail     ast.Expr
}

// flattenLetChain walks Body links while they remain *ast.Let, collecting each
// binding, and returns the flattened chain. root must be non-nil with a non-nil Body.
// A single-binding chain (`let x = v in tail`, Body is not a *ast.Let) yields one
// binding and the tail — M0 proved 2 of the 28 targets are exactly this shape.
func flattenLetChain(root *ast.Let) letChain {
	lc := letChain{Root: root}
	cur := root
	for {
		lc.Bindings = append(lc.Bindings, cur)
		next, ok := cur.Body.(*ast.Let)
		if !ok {
			lc.Tail = cur.Body
			break
		}
		cur = next
	}
	return lc
}

// isSingleLetChainBlock reports whether b is a braced block whose SOLE expression is
// a let-chain root (a *ast.Let with a non-nil Body). Such a block is subsumed by the
// chain list registered when the attacher recurses into its expression, so the block
// itself must not register a competing one-child list.
func isSingleLetChainBlock(b *ast.Block) bool {
	if b == nil || len(b.Exprs) != 1 {
		return false
	}
	let, ok := b.Exprs[0].(*ast.Let)
	return ok && let.Body != nil
}

// chainChildNodes returns the logical child sequence [binding0, binding1, …, tail]
// as ast.Node values, for boundary indexing.
func (lc letChain) chainChildNodes() []ast.Node {
	nodes := make([]ast.Node, 0, len(lc.Bindings)+1)
	for _, b := range lc.Bindings {
		nodes = append(nodes, b)
	}
	if lc.Tail != nil {
		nodes = append(nodes, lc.Tail)
	}
	return nodes
}
