package types

// The nominal head of a type (#2244).
//
// A type alias names its target; it is not a type of its own. `type Hue =
// Color` makes Hue and Color one type, and `type IntOpt = Option[int]` makes
// IntOpt one instance of Option. So every question about WHICH ADT a value
// belongs to — the owner of a constructor pattern, that constructor's arity,
// the universe a match must cover, the choice between two imported
// constructors of one name — is asked of the type's head AFTER alias
// expansion, and nominalHead is the one place that computes it.
//
// A newtype ADT (`type W = W({ x: int })`, registered as an alias of its
// record so `w.x` works) is NOT expanded: it is a nominal type of its own,
// and its head is W.

// nominalHead returns the type constructor heading t once aliases (alias
// chains and applied parameterised aliases included) are expanded, or nil
// while t is not headed by a named type: a type variable, a record, a
// function, a tuple. A nil unifier expands nothing.
//
// Cycle-safety: expandAliasMode stops on a repeated alias key, and the head
// is read one level deep.
func (u *Unifier) nominalHead(t Type) *TCon {
	if u != nil {
		t = u.expandAliasMode(t, false)
	}
	if app, ok := t.(*TApp); ok {
		t, _ = decomposeApp(app)
	}
	h, _ := t.(*TCon)
	return h
}

// scrutineeADTName returns the name of the ADT a constructor pattern's
// scrutinee concretely has, aliases expanded — "" when its type is not yet a
// named ADT (a type variable, a primitive, a record...), so callers defer to
// unification exactly as before.
func (tc *CoreTypeChecker) scrutineeADTName(t Type) string {
	h := tc.aliasUnifier().nominalHead(t)
	if h == nil {
		return ""
	}
	return extractADTName(h)
}

// writtenAlias returns the name the scrutinee's type was written with when
// it is an alias of adt (so a diagnostic naming adt can say so), else "".
func writtenAlias(scrutType Type, adt string) string {
	if written := extractADTName(scrutType); written != adt {
		return written
	}
	return ""
}
