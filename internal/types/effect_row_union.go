package types

import "fmt"

// CheckedUnionEffectRows preserves a shared tail and reports an irreducible
// two-tail requirement. Inference must solve determining equalities first.
func CheckedUnionEffectRows(a, b *Row) (*Row, error) {
	if a != nil && b != nil && a.Tail != nil && b.Tail != nil && !a.Tail.Equals(b.Tail) {
		left, right := a.Tail.Name, b.Tail.Name
		if left > right {
			left, right = right, left
		}
		return nil, fmt.Errorf("distinct unresolved effect tails %q and %q; use a shared effect row", left, right)
	}
	return UnionEffectRows(a, b), nil
}
