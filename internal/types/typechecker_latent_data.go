package types

import "sort"

// M-EFFECT-LATENT-NESTED-CALLBACKS (#1718, #1826): ONE answer to "what does
// calling the function values inside this value cost?". The callee mask, the
// per-argument callback rows, the implicit-ownership set and the App constraint
// row all read a function value's latent row through these helpers, so a tail
// erased inside a callee body is charged at every call site.

// maxDataArrowDepth bounds the walk. Recursive aliases (`type T = [T]`) and
// infinite expansions terminate here; real data nesting is far shallower.
const maxDataArrowDepth = 16

// dataArrowsWith returns the arrows reachable from t through DATA positions
// only: list/array elements, map key and value, tuple elements, type-application
// arguments, record fields (TRecord, TRecordOpen, TRecord2 labels). It never
// descends into an arrow's parameters or return — calling a stored function is
// what costs its row; its parameters are the callee's business.
//
// expand, when non-nil, resolves an alias head at each level (effectAliasHead).
// Cycle-safety: depth-capped at maxDataArrowDepth, so recursive aliases and
// self-referential expansions terminate. Record fields are visited in sorted
// order so the result is deterministic.
func dataArrowsWith(t Type, expand func(Type) Type) []*TFunc2 {
	var out []*TFunc2
	var walk func(Type, int)
	walk = func(t Type, depth int) {
		if t == nil || depth > maxDataArrowDepth {
			return
		}
		if expand != nil {
			t = expand(t)
		}
		switch x := t.(type) {
		case *TFunc2:
			out = append(out, x)
		case *TList:
			walk(x.Element, depth+1)
		case *TArray:
			walk(x.Element, depth+1)
		case *TMap:
			walk(x.Key, depth+1)
			walk(x.Value, depth+1)
		case *TTuple:
			for _, e := range x.Elements {
				walk(e, depth+1)
			}
		case *TApp:
			for _, a := range x.Args {
				walk(a, depth+1)
			}
		case *TRecord:
			for _, name := range sortedTypeKeys(x.Fields) {
				walk(x.Fields[name], depth+1)
			}
		case *TRecordOpen:
			for _, name := range sortedTypeKeys(x.Fields) {
				walk(x.Fields[name], depth+1)
			}
		case *TRecord2:
			if x.Row != nil {
				for _, name := range sortedTypeKeys(x.Row.Labels) {
					walk(x.Row.Labels[name], depth+1)
				}
			}
		}
	}
	walk(t, 0)
	return out
}

func sortedTypeKeys(m map[string]Type) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// dataArrows is dataArrowsWith expanding alias heads at every level.
func (tc *CoreTypeChecker) dataArrows(t Type) []*TFunc2 {
	return dataArrowsWith(t, tc.effectAliasHead)
}

// isLatentCallback reports an arrow whose effects are the CALLER's to pay at a
// hand-off: an open row that is not a concrete (callee-owned) contract.
func isLatentCallback(fn *TFunc2) bool {
	return fn.EffectRow != nil && fn.EffectRow.Tail != nil && !fn.ConcreteEffectContract
}

// latentRow is the cost of invoking fn: its row, with a ConcreteEffectContract's
// L1-open compatibility tail closed — the rule applicationCallRow already applies
// to a direct callee. A row-less arrow costs nothing.
func latentRow(fn *TFunc2) *Row {
	if fn.EffectRow == nil {
		return EmptyEffectRow()
	}
	row := cloneApplicationRow(fn.EffectRow)
	if fn.ConcreteEffectContract {
		row.Tail = nil
	}
	return row
}

// nestedLatentRow unions latentRow over the data-position arrows of t. Each
// arrow's row is first passed through zonk (nil = identity); a tail the caller
// owns implicitly (`implicitOwned`) is the caller's caller's to pay and is
// dropped per arrow, as the validator does for a top-level ImplicitCallbacks
// row. Two remaining distinct tails are an irreducible #616 conflict: the result
// is nil (never first-wins), so a masked argument trips the validator's
// fail-loud missing-callback-row invariant. No arrows also yields nil.
func nestedLatentRow(arrows []*TFunc2, zonk func(*Row) *Row, implicitOwned map[string]bool) *Row {
	var acc *Row
	for _, fn := range arrows {
		row := latentRow(fn)
		if zonk != nil {
			row = zonk(row)
		}
		if row.Tail != nil && implicitOwned[row.Tail.Name] {
			row.Tail = nil
		}
		union, err := CheckedUnionEffectRows(acc, row)
		if err != nil {
			return nil
		}
		if union == nil { // UnionEffectRows spells "pure" as nil
			union = EmptyEffectRow()
		}
		acc = union
	}
	return acc
}

// nestedLatentMask reports whether a non-arrow parameter type nests, in a data
// position, an arrow whose effects the caller must pay at the hand-off.
func (tc *CoreTypeChecker) nestedLatentMask(param Type) bool {
	if _, top := param.(*TFunc2); top {
		return false
	}
	for _, fn := range tc.dataArrows(param) {
		if isLatentCallback(fn) {
			return true
		}
	}
	return false
}
