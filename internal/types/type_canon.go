package types

import (
	"fmt"
	"strings"
)

// typeCanon numbers types by structure, memoised per node (#2363 review, round
// 2): two structurally equal types get one number however many nodes they are
// built from, in time linear in the nodes numbered. Only the forms whose
// structure is fully their children are numbered structurally (constructors,
// variables, lists, arrays, maps, tuples, applications); anything else —
// functions, records, rows, labels — is numbered by its own identity, so two of
// them are equal only when they are the same node. That is conservative: an
// unequal number only costs a missed sharing, never a wrong one.
type typeCanon struct {
	ids    map[Type]int
	intern map[string]int
}

// isStructuralArg reports the composite forms typeCanon numbers by structure.
func isStructuralArg(t Type) bool {
	switch t.(type) {
	case *TList, *TArray, *TMap, *TTuple, *TApp:
		return true
	}
	return false
}

func newTypeCanon() *typeCanon {
	return &typeCanon{ids: map[Type]int{}, intern: map[string]int{}}
}

func (c *typeCanon) id(t Type) int {
	if n, ok := c.ids[t]; ok {
		return n
	}
	var s strings.Builder
	list := func(open string, ts []Type) {
		s.WriteString(open)
		for _, e := range ts {
			fmt.Fprintf(&s, "%d,", c.id(e))
		}
	}
	switch x := t.(type) {
	case nil:
		s.WriteString("nil")
	case *TCon:
		s.WriteString("c:" + x.KeyOf())
	case *TVar:
		s.WriteString("v:" + x.Name)
	case *TVar2:
		s.WriteString("w:" + x.Name)
	case *TList:
		list("[", []Type{x.Element})
	case *TArray:
		list("A[", []Type{x.Element})
	case *TMap:
		list("M[", []Type{x.Key, x.Value})
	case *TTuple:
		list("(", x.Elements)
	case *TApp:
		list("@", append([]Type{x.Constructor}, x.Args...))
	default:
		fmt.Fprintf(&s, "#%p", t) // only itself
	}
	n, ok := c.intern[s.String()]
	if !ok {
		n = len(c.intern) + 1
		c.intern[s.String()] = n
	}
	c.ids[t] = n
	return n
}
