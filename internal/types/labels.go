package types

import (
	"fmt"
	"sort"
	"strings"
)

// Label represents an information-flow label in the lattice:
//
//	L ::= ⊥ | ℓ | α | L₁ ⊔ L₂
//
// The zero value of the interface is not valid; always use the constructors.
type Label interface {
	labelNode()
	String() string
}

// --- Concrete label types ---

// labelBottom is the bottom element ⊥ (untainted / no label).
type labelBottom struct{}

func (labelBottom) labelNode()     {}
func (labelBottom) String() string { return "⊥" }

// labelConst is a constant label like "email" or "pii".
// Rendered as <name> in AILANG syntax.
type labelConst struct{ name string }

func (l labelConst) labelNode()     {}
func (l labelConst) String() string { return fmt.Sprintf("<%s>", l.name) }

// labelVar is a label variable (used for polymorphic label inference), e.g. α or L.
type labelVar struct{ name string }

func (l labelVar) labelNode()     {}
func (l labelVar) String() string { return l.name }

// labelJoin is the least upper bound L1 ⊔ L2.
// Internally stored in a sorted, de-duplicated flat set of constituent labels
// so that Equal and String are canonical.
type labelJoin struct {
	// parts holds the flattened, sorted, deduplicated constituents.
	// Invariant: len(parts) >= 2, no part is ⊥, no part is another Join.
	parts []Label
}

func (l labelJoin) labelNode() {}
func (l labelJoin) String() string {
	pieces := make([]string, len(l.parts))
	for i, p := range l.parts {
		pieces[i] = p.String()
	}
	return strings.Join(pieces, " ⊔ ")
}

// --- Constructors ---

// LabelBottom returns the bottom element ⊥.
func LabelBottom() Label { return labelBottom{} }

// LabelConst returns a constant label with the given name.
func LabelConst(name string) Label { return labelConst{name: name} }

// LabelVar returns a label variable with the given name.
func LabelVar(name string) Label { return labelVar{name: name} }

// LabelJoin returns the least upper bound L1 ⊔ L2 with simplification:
//   - ⊥ is the identity element
//   - idempotence: L ⊔ L = L
//   - the result is normalised to a sorted, flattened, deduplicated join
func LabelJoin(a, b Label) Label {
	parts := flattenJoin(a, b)
	switch len(parts) {
	case 0:
		return labelBottom{}
	case 1:
		return parts[0]
	default:
		return labelJoin{parts: parts}
	}
}

// flattenJoin collects and normalises the constituent labels from a ⊔ b.
// It flattens nested joins, drops ⊥ elements, deduplicates, and sorts.
func flattenJoin(a, b Label) []Label {
	set := map[string]Label{}
	collectParts(a, set)
	collectParts(b, set)

	// Sort for canonical representation
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]Label, len(keys))
	for i, k := range keys {
		result[i] = set[k]
	}
	return result
}

// collectParts recursively flattens a label into canonical parts.
// ⊥ is dropped; joins are flattened; everything else is keyed by String().
func collectParts(l Label, set map[string]Label) {
	switch v := l.(type) {
	case labelBottom:
		// identity element — drop
	case labelJoin:
		for _, p := range v.parts {
			collectParts(p, set)
		}
	default:
		set[l.String()] = l
	}
}

// --- Lattice operations ---

// LabelEqual reports whether two labels are structurally equal after normalisation.
func LabelEqual(a, b Label) bool {
	// Normalise both sides by round-tripping through flattenJoin logic
	pa := normalisedParts(a)
	pb := normalisedParts(b)
	if len(pa) != len(pb) {
		return false
	}
	for i := range pa {
		if pa[i].String() != pb[i].String() {
			return false
		}
	}
	return true
}

// normalisedParts returns the sorted canonical constituent list for a label.
func normalisedParts(l Label) []Label {
	set := map[string]Label{}
	collectParts(l, set)
	if len(set) == 0 {
		// l was ⊥
		return []Label{labelBottom{}}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]Label, len(keys))
	for i, k := range keys {
		result[i] = set[k]
	}
	return result
}

// LabelSubsumes reports whether label ℓ appears (by name) anywhere in L's join.
// Semantics: "L is tainted with at least ℓ".
func LabelSubsumes(L, ell Label) bool {
	ellStr := ell.String()
	return containsLabel(L, ellStr)
}

// containsLabel walks L recursively checking if any constituent matches key.
func containsLabel(L Label, key string) bool {
	switch v := L.(type) {
	case labelBottom:
		return false
	case labelJoin:
		for _, p := range v.parts {
			if containsLabel(p, key) {
				return true
			}
		}
		return false
	default:
		return L.String() == key
	}
}

// EvalNot evaluates the refinement predicate {not ℓ} against label L.
// Returns true iff ℓ is NOT subsumed by L (i.e., the argument is safe for a {not ℓ} sink).
func EvalNot(L Label, ell Label) bool {
	return !LabelSubsumes(L, ell)
}
