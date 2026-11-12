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
// (`type Score = int`; Num[Score] was rejected even before #2247) and for Eq.
//
// An expansion is used only when it SOLVES the constraint outright: it is
// ground and an instance resolves it (which records the dictionary for the
// operator node), or it reduces to nothing. Otherwise the constraint keeps
// its written type and is judged, and reported, as written. That rule is what
// keeps every consumer that picks an implementation from a node's type safe:
//   - `type Id[a] = a` with `x + x` on `x: Id[b]` would expand to Num[b],
//     which is deferred, records no dictionary, and leaves the `+` node typed
//     Id[b]; lowering then picks the Int implementation and `dbl(1.5)`
//     crashes. Kept as written, it is "No instance for Num[Id[b]]", as on dev
//     (LIMITATION: a polymorphic operator on an alias of a type variable is
//     rejected; write the variable instead).
//   - an alias body naming a variable it does not bind (`type Count =
//     integer`) would turn the ground Num[Count] into a deferred, then
//     dropped, constraint on a fresh variable.
//   - a self-referential alias (`type L[a] = [L[a]]`) unfolds to something
//     unsolvable, so the diagnostic names L[int], not a type unfolded
//     aliasExpandDepthCap levels deep.
//
// Two kinds of type are never expanded: a newtype ADT (`type W =
// W({x: int})`) and a record alias (`type P = {x: int}`). Both are nominal:
// their `deriving (Eq)` instance is keyed by their defining module, while the
// record path of Eq synthesis is keyed by bare name, so expanding them would
// let one borrow a same-named type's `deriving (Eq)` from another module, or
// give a record that never derived Eq an ==.

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

// expandConstraintAliases rewrites a constraint's type with its aliases
// expanded when that solves it outright (see the file comment); otherwise
// the constraint keeps its written type.
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
			if tc.constraintSolvedNow(e) {
				c = e
			}
		}
		out[i] = c
	}
	return out
}

// constraintSolvedNow reports, without recording anything, whether c is
// solved at this boundary: a ground constraint an instance resolves, or one
// that context reduction discharges with nothing left to defer.
func (tc *CoreTypeChecker) constraintSolvedNow(c ClassConstraint) bool {
	if constraintIsGround(c) {
		_, err := tc.instanceEnv.Lookup(c.Class, c.Type)
		return err == nil
	}
	residual, err := tc.instanceEnv.ReduceConstraints([]ClassConstraint{c})
	return err == nil && len(residual) == 0
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
	// withNewtypes=false: a newtype ADT stays nominal, and so does a record
	// alias (see the file comment).
	// Repeated because expandAliasMode treats a second application of the
	// same alias as a cycle: Id[Id[int]] stops at Id[int] in one call.
	for i := 0; i < aliasExpandDepthCap; i++ {
		e := u.expandAliasMode(t, false)
		if e == t {
			break
		}
		if _, isRecord := e.(*TRecord); isRecord {
			break
		}
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
