package iface

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// FuncContracts returns fd's requires/ensures clauses as normalised strings,
// in source order: "requires <expr>" / "ensures <expr>". The expression is
// rendered with the AST printer, which is fully parenthesised and carries no
// source positions, so reformatting a contract does not change its string and
// changing what it says does. Property-test and invariant clauses are not
// contracts on the function's interface and are skipped.
//
// M-PKG-PARALLEL-AGENTS D7: these lines enter the v2 signature set so a
// contract change is classified C, while interface_hash_v2 stays unchanged.
func FuncContracts(fd *ast.FuncDecl) []string {
	if fd == nil {
		return nil
	}
	var out []string
	for _, p := range fd.Properties {
		if p == nil || p.Expr == nil {
			continue
		}
		if p.Kind != ast.RequiresKind && p.Kind != ast.EnsuresKind {
			continue
		}
		out = append(out, fmt.Sprintf("%s %s", p.Kind, p.Expr.String()))
	}
	return out
}
