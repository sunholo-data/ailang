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
// A record alias stays nominal: a record's == comes from its own
// `deriving (Eq)` declaration, which is keyed by the alias name.

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
// alias stands for.
func (tc *CoreTypeChecker) expandConstraintAliases(constraints []ClassConstraint) []ClassConstraint {
	if len(constraints) == 0 || (len(tc.aliasEnv) == 0 && len(tc.aliasCaptures) == 0) {
		return constraints
	}
	u := tc.newAliasUnifier()
	out := make([]ClassConstraint, len(constraints))
	for i, c := range constraints {
		c.Type = expandAliasesForConstraint(u, c.Type, 0)
		out[i] = c
	}
	return out
}

// expandAliasesForConstraint expands alias heads at t and inside the
// structures context reduction looks through (type-application arguments,
// list elements, tuple elements). A newtype ADT and a record alias stay
// nominal.
//
// Cycle-safety: alias chains are cut by expandAliasMode's own seen-set, and
// the structural recursion is capped at aliasExpandDepthCap.
func expandAliasesForConstraint(u *Unifier, t Type, depth int) Type {
	if t == nil || depth > aliasExpandDepthCap {
		return t
	}
	if e := u.expandAliasMode(t, false); e != t {
		if _, isRecord := e.(*TRecord); !isRecord {
			t = e
		}
	}
	switch tt := t.(type) {
	case *TApp:
		args := make([]Type, len(tt.Args))
		changed := false
		for i, a := range tt.Args {
			args[i] = expandAliasesForConstraint(u, a, depth+1)
			changed = changed || args[i] != a
		}
		if changed {
			return &TApp{Constructor: tt.Constructor, Args: args}
		}
	case *TList:
		if el := expandAliasesForConstraint(u, tt.Element, depth+1); el != tt.Element {
			return &TList{Element: el}
		}
	case *TTuple:
		elems := make([]Type, len(tt.Elements))
		changed := false
		for i, e := range tt.Elements {
			elems[i] = expandAliasesForConstraint(u, e, depth+1)
			changed = changed || elems[i] != e
		}
		if changed {
			return &TTuple{Elements: elems}
		}
	}
	return t
}
