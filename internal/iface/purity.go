package iface

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/traverse"
)

// derivePurity reads checked schemes after AST row restoration, never IsPure.
// A function's own outer row determines purity. An undeclared tail shared with
// a callback is latent: the caller supplies and pays for the callback's effects.
// Without a declaration, an open outer row fails closed.
//
// Non-function values use a deep policy: no effect labels or effect row variables
// anywhere in their type, including nested functions. Record row variables alone
// are harmless. All walks are cycle-safe through types/traverse.
func derivePurity(scheme *types.Scheme, decl *ast.FuncDecl) bool {
	if scheme == nil || scheme.Type == nil {
		return false
	}
	if fn, ok := scheme.Type.(*types.TFunc2); ok {
		row := fn.EffectRow
		if row == nil {
			return true
		}
		if len(row.Labels) != 0 {
			return false
		}
		if row.Tail == nil {
			return true
		}
		if decl == nil || hasRowVarEffects(decl.Effects) {
			return false
		}
		return callbackSharesTail(fn.Params, row.Tail.Name)
	}
	pure := true
	traverse.Walk(scheme.Type, func(t types.Type) {
		switch typ := t.(type) {
		case *types.Row:
			if typ.Kind.Equals(types.EffectRow) && (len(typ.Labels) > 0 || typ.Tail != nil) {
				pure = false
			}
		case *types.RowVar:
			if typ.Kind.Equals(types.EffectRow) {
				pure = false
			}
		}
	})
	return pure
}

// callbackSharesTail only searches parameter types, including callbacks nested
// in containers or other function types; a return-only occurrence is not latent.
func callbackSharesTail(params []types.Type, tail string) bool {
	shared := false
	for _, param := range params {
		traverse.Walk(param, func(t types.Type) {
			if fn, ok := t.(*types.TFunc2); ok && fn.EffectRow != nil && fn.EffectRow.Tail != nil && fn.EffectRow.Tail.Name == tail {
				shared = true
			}
		})
	}
	return shared
}
