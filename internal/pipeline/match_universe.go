package pipeline

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// The constructor universe of the match-exhaustiveness check
// (M-V1-MATCH-EXHAUSTIVENESS §8): for each ADT, keyed by its nominal identity
// types.NominalKey(defining module, name), EVERY constructor with its arity —
// from the module's own declarations and from every loaded interface, never
// from "the constructors in scope". Lists are sorted by name: interfaces keep
// constructors in a map, so name order is the only order every path agrees on.

// addIfaceToMatchUniverse adds every constructor an interface declares.
func addIfaceToMatchUniverse(u types.MatchUniverse, ifc *iface.Iface) {
	for name, cs := range ifc.Constructors {
		if cs == nil {
			continue
		}
		ctor := cs.CtorName
		if ctor == "" {
			ctor = name
		}
		addMatchCtor(u, types.NominalKey(ifc.Module, cs.TypeName), ctor, cs.Arity)
	}
}

// addMatchCtor adds one constructor, keeping the list sorted and unique.
func addMatchCtor(u types.MatchUniverse, key, ctor string, arity int) {
	sigs := u[key]
	i := sort.Search(len(sigs), func(i int) bool { return sigs[i].Name >= ctor })
	if i < len(sigs) && sigs[i].Name == ctor {
		return
	}
	sigs = append(sigs, exhaust.CtorSig{})
	copy(sigs[i+1:], sigs[i:])
	sigs[i] = exhaust.CtorSig{Name: ctor, Arity: arity}
	u[key] = sigs
}

// moduleMatchUniverse is the universe a module is checked against: the
// loaded interfaces' ADTs plus the module's own.
func moduleMatchUniverse(loaded types.MatchUniverse, modID string, local map[string]*ConstructorInfo) types.MatchUniverse {
	u := make(types.MatchUniverse, len(loaded)+len(local))
	for k, sigs := range loaded {
		u[k] = sigs
	}
	owned := make(map[string]bool)
	for name, info := range local {
		if info == nil {
			continue
		}
		ctor := info.CtorName
		if ctor == "" {
			ctor = name
		}
		key := types.NominalKey(modID, info.TypeName)
		if !owned[key] {
			// The module's own declaration is authoritative for its types.
			u[key] = nil
			owned[key] = true
		}
		addMatchCtor(u, key, ctor, info.Arity)
	}
	return u
}

// elaboratorMatchUniverse is the universe of the single-file / embedded path:
// the ADTs the elaborator saw declared (each with its own module).
func elaboratorMatchUniverse(ctors map[string]*elaborate.ConstructorInfo) types.MatchUniverse {
	u := make(types.MatchUniverse)
	for name, info := range ctors {
		if info == nil || info.IsImported {
			continue
		}
		ctor := info.CtorName
		if ctor == "" {
			ctor = name
		}
		addMatchCtor(u, types.NominalKey(info.Module, info.TypeName), ctor, info.Arity)
	}
	return u
}
