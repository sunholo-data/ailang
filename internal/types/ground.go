package types

import "sync/atomic"

// Ground-type memo for substitution (#1787).
//
// #1706 publishes nullary aliases as fully expanded structural records. Those
// bodies hold no type variables, so substituting into them changes nothing,
// yet safeSubstitute walked every field on every call: CoreTypeInfo and the
// solver apply substitutions thousands of times per module, and motoko's
// session.ail went from 34 s to 77 s cold.
//
// A record that is ground stays ground: record nodes are never edited in place
// after construction. The only by-value copies (the alias closer) change
// TypeName, which does not affect groundness. So the answer is cached on the
// node. Row and TFunc2 are NOT cached: they are cloned and then have fields
// replaced (cloneApplicationRow, typechecker_latent_mask), which would carry a
// stale memo onto a different structure.
const (
	groundUnknown uint32 = iota
	groundYes
	groundNo
)

// isGroundType reports whether t contains no type, row, or effect variables,
// so that any substitution leaves it unchanged. Unknown type kinds are
// conservatively not ground.
//
// Cycle-safe: visiting holds the nodes on the current path; reaching one again
// is assumed ground (a cycle adds no variables of its own). A ground result that
// leaned on that assumption is not memoised, because the node on the path might
// still turn out non-ground through a later child.
func isGroundType(t Type) bool {
	ground, _ := groundWalk(t, nil)
	return ground
}

func groundWalk(t Type, visiting map[Type]bool) (ground, provisional bool) {
	switch typ := t.(type) {
	case nil:
		return true, false
	case *TCon:
		return true, false
	case *TVar, *TVar2, *RowVar:
		return false, false
	case *TRecord:
		switch atomic.LoadUint32(&typ.groundMemo) {
		case groundYes:
			return true, false
		case groundNo:
			return false, false
		}
		if visiting[t] {
			return true, true
		}
		if visiting == nil {
			visiting = map[Type]bool{}
		}
		visiting[t] = true
		ground, provisional = true, false
		for _, f := range typ.Fields {
			g, p := groundWalk(f, visiting)
			provisional = provisional || p
			if !g {
				ground = false
				break
			}
		}
		if ground {
			g, p := groundWalk(typ.Row, visiting)
			ground, provisional = g, provisional || p
		}
		delete(visiting, t)
		switch {
		case !ground:
			atomic.StoreUint32(&typ.groundMemo, groundNo)
		case !provisional:
			atomic.StoreUint32(&typ.groundMemo, groundYes)
		}
		return ground, provisional
	case *TList:
		return groundWalk(typ.Element, visiting)
	case *TArray:
		return groundWalk(typ.Element, visiting)
	case *TMap:
		return groundAll(visiting, typ.Key, typ.Value)
	case *TTuple:
		return groundAll(visiting, typ.Elements...)
	case *TApp:
		g, p := groundWalk(typ.Constructor, visiting)
		if !g {
			return false, p
		}
		g2, p2 := groundAll(visiting, typ.Args...)
		return g2, p || p2
	case *TFunc2:
		g, p := groundAll(visiting, typ.Params...)
		if !g {
			return false, p
		}
		g2, p2 := groundWalk(typ.Return, visiting)
		if !g2 {
			return false, p || p2
		}
		if typ.EffectRow == nil {
			return true, p || p2
		}
		g3, p3 := groundWalk(typ.EffectRow, visiting)
		return g3, p || p2 || p3
	case *Row:
		if typ == nil {
			return true, false
		}
		if typ.Tail != nil {
			return false, false
		}
		provisional := false
		for _, l := range typ.Labels {
			g, p := groundWalk(l, visiting)
			provisional = provisional || p
			if !g {
				return false, provisional
			}
		}
		return true, provisional
	default:
		return false, false
	}
}

func groundAll(visiting map[Type]bool, ts ...Type) (ground, provisional bool) {
	for _, t := range ts {
		g, p := groundWalk(t, visiting)
		provisional = provisional || p
		if !g {
			return false, provisional
		}
	}
	return true, provisional
}
