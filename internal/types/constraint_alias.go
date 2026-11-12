package types

// Alias expansion of class-constraint types (#2247 review).
//
// Unification expands an alias only when it meets a different head, so a
// parameter declared `x: Id[a]` (with `type Id[a] = int`) keeps the type
// Id[a] through `x + 1`, and the constraint reads Num[Id[a]]. Context
// reduction decides a constraint by its head, and Id is not a class
// instance's head: before #2247 the constraint was dropped (and the program
// ran, by luck of the runtime picking the dictionary from the values), and
// with #2247 it was "No instance for Num[Id[a]]" for a correct program. The
// alias has to be seen through before the head is judged: Num[Id[a]] is
// Num[int]. The same holds for a nullary alias of a primitive
// (`type Score = int`; Num[Score] was rejected even before #2247), for a
// record alias (Eq on `type P = {x: int} deriving (Eq)` resolves through the
// record's declared name), and for Eq.
//
// Two expansions are refused:
//   - one that introduces a type variable the unexpanded type did not have.
//     An alias whose body names a variable it does not bind
//     (`type Count = integer`: a lowercase name is a type variable) would turn
//     the ground Num[Count] into a constraint on a fresh variable, which is
//     deferred and then dropped, so `inc("x")` passed check;
//   - a newtype ADT (`type W = W({x: int})`), which is a nominal type, not an
//     alias: its Eq instance is keyed by its defining module, and expanding it
//     would let it borrow a same-named type's `deriving (Eq)` from another
//     module through the bare-name record path.
//
// And a constraint keeps its written type whenever the expanded one cannot be
// solved either, so a diagnostic names what the user wrote: `Eq[P]`, not the
// record P stands for, and `Eq[L[int]]` for a self-referential alias
// (`type L[a] = [L[a]]`), not a type unfolded aliasExpandDepthCap levels deep.

// aliasExpandDepthCap bounds the structural walk (a self-referential alias
// such as `type L[a] = [L[a]]` would otherwise unfold forever).
const aliasExpandDepthCap = 32

// newAliasUnifier builds the unifier that sees through this checker's
// aliases (local and imported, with applied-alias parameters, name-shadow
// captures and newtypes).
func (tc *CoreTypeChecker) newAliasUnifier() *Unifier {
	if len(tc.aliasEnv) == 0 && len(tc.aliasCaptures) == 0 {
		return NewUnifier()
	}
	// M-XMOD-ALIAS-POLY: param env so applied aliases (`Box[int]`) instantiate.
	u := NewUnifierWithAliasesAndParams(tc.aliasEnv, tc.aliasParams)
	u.SetAliasCaptures(tc.aliasCaptures)
	u.SetNewtypes(tc.newtypes)
	return u
}

// expandConstraintAliases rewrites every constraint's type with its aliases
// expanded, so instance resolution and context reduction judge the type the
// alias stands for. A constraint keeps its written type when the expanded one
// cannot be solved either: the diagnostic then names the type the user wrote
// (`Eq[P]`, not the record P stands for), exactly as before the expansion.
func (tc *CoreTypeChecker) expandConstraintAliases(constraints []ClassConstraint) []ClassConstraint {
	if len(constraints) == 0 || (len(tc.aliasEnv) == 0 && len(tc.aliasCaptures) == 0) {
		return constraints
	}
	u := tc.newAliasUnifier()
	out := make([]ClassConstraint, len(constraints))
	for i, c := range constraints {
		if expanded := expandAliasesForConstraint(u, c.Type, 0); expanded != c.Type {
			e := c
			e.Type = expanded
			if tc.constraintSolvable(e) {
				c = e
			}
		}
		out[i] = c
	}
	return out
}

// constraintSolvable reports, without recording anything, whether c would
// pass instance resolution (ground) or context reduction (non-ground).
func (tc *CoreTypeChecker) constraintSolvable(c ClassConstraint) bool {
	if constraintIsGround(c) {
		_, err := tc.instanceEnv.Lookup(c.Class, c.Type)
		return err == nil
	}
	_, err := tc.instanceEnv.ReduceConstraints([]ClassConstraint{c})
	return err == nil
}

// expandAliasesForConstraint expands alias heads at t and inside the
// structures context reduction looks through (type-application arguments,
// list elements, tuple elements).
//
// Cycle-safety: alias chains are cut by expandAliasMode's own seen-set, and
// the structural recursion stops at aliasExpandDepthCap (a self-referential
// alias); what it returns there is unsolvable, so expandConstraintAliases
// keeps the written type.
func expandAliasesForConstraint(u *Unifier, t Type, depth int) Type {
	if t == nil || depth > aliasExpandDepthCap {
		return t
	}
	// withNewtypes=false: a newtype ADT stays nominal (see the file comment).
	if e := u.expandAliasMode(t, false); e != t && !introducesTypeVars(t, e) {
		t = e
	}
	switch tt := t.(type) {
	case *TApp:
		if args, changed := expandAliasesInEach(u, tt.Args, depth); changed {
			return &TApp{Constructor: tt.Constructor, Args: args}
		}
	case *TList:
		if el := expandAliasesForConstraint(u, tt.Element, depth+1); el != tt.Element {
			return &TList{Element: el}
		}
	case *TTuple:
		if elems, changed := expandAliasesInEach(u, tt.Elements, depth); changed {
			return &TTuple{Elements: elems}
		}
	}
	return t
}

func expandAliasesInEach(u *Unifier, ts []Type, depth int) ([]Type, bool) {
	out := make([]Type, len(ts))
	changed := false
	for i, a := range ts {
		out[i] = expandAliasesForConstraint(u, a, depth+1)
		changed = changed || out[i] != a
	}
	return out, changed
}

// introducesTypeVars reports whether after has a free type variable that
// before does not: the mark of an alias body naming a variable it does not
// bind. Cycle-safety: collectFreeVars tracks visited types.
func introducesTypeVars(before, after Type) bool {
	had := make(map[string]bool)
	collectFreeVars(before, had)
	now := make(map[string]bool)
	collectFreeVars(after, now)
	for v := range now {
		if !had[v] {
			return true
		}
	}
	return false
}
