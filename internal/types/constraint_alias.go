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
//   - a self-referential alias (`type L[a] = [L[a]]`, or the branching
//     `type T2[a] = (T2[a], T2[a])`) never reaches a finite ground type: the
//     expansion stops where the alias recurs, and the diagnostic names L[int].
//
// Two kinds of type are never expanded: a newtype ADT (`type W =
// W({x: int})`) and a record alias (`type P = {x: int}`). Both are nominal:
// their `deriving (Eq)` instance is keyed by their defining module, while the
// record path of Eq synthesis is keyed by bare name, so expanding them would
// let one borrow a same-named type's `deriving (Eq)` from another module, or
// give a record that never derived Eq an ==.

// aliasExpandNodeBudget bounds the work of expanding one constraint type.
// The written type is finite, but expansion can multiply it: a nest of
// nullary pair aliases (`type P2 = (P1, P1)`, ... `P20`) is legitimately
// 2^20 leaves. Running out of budget means "not solved outright", so the
// constraint is judged as written, exactly as it would be on dev.
const aliasExpandNodeBudget = 4096

// expandConstraintAliases rewrites a constraint's type with its aliases
// expanded when that solves it outright (see the file comment); otherwise
// the constraint keeps its written type.
func (tc *CoreTypeChecker) expandConstraintAliases(constraints []ClassConstraint) []ClassConstraint {
	if len(constraints) == 0 || (len(tc.aliasEnv) == 0 && len(tc.aliasCaptures) == 0) {
		return constraints
	}
	u := tc.aliasUnifier()
	out := make([]ClassConstraint, len(constraints))
	for i, c := range constraints {
		if expanded, ok := expandAliasesForConstraint(u, c.Type); ok && expanded != c.Type {
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

// expandAliasesForConstraint expands the aliases in t: at its head and inside
// the structures context reduction looks through (type-application arguments,
// list elements, tuple elements). ok is false when expansion cannot finish:
// an alias recurs inside its own expansion (`type T2[a] = (T2[a], T2[a])`,
// which can never reach a finite ground type), or the node budget runs out.
// The caller then judges the constraint as written.
func expandAliasesForConstraint(u *Unifier, t Type) (Type, bool) {
	x := &aliasExpander{u: u, onPath: map[string]bool{}, budget: aliasExpandNodeBudget}
	return x.expand(t)
}

// aliasExpander carries the expansion path and the remaining node budget.
// Termination: every call spends budget, so the walk visits at most
// aliasExpandNodeBudget nodes; an alias is never expanded inside its own
// expansion (onPath), so a self-referential alias stops at its first
// recurrence instead of being unfolded.
type aliasExpander struct {
	u      *Unifier
	onPath map[string]bool
	budget int
}

func (x *aliasExpander) expand(t Type) (Type, bool) {
	if t == nil {
		return t, true
	}
	if x.budget--; x.budget < 0 {
		return t, false
	}
	// Arguments first, so an alias in an argument (Id[Id[float]], Pair[Pair[int]])
	// is expanded before its enclosing alias and is never mistaken for that
	// alias recurring in its own body.
	switch tt := t.(type) {
	case *TApp:
		args, changed, ok := x.expandEach(tt.Args)
		if !ok {
			return t, false
		}
		if changed {
			t = &TApp{Constructor: tt.Constructor, Args: args}
		}
	case *TList:
		el, ok := x.expand(tt.Element)
		if !ok {
			return t, false
		}
		if el != tt.Element {
			t = &TList{Element: el}
		}
	case *TTuple:
		elems, changed, ok := x.expandEach(tt.Elements)
		if !ok {
			return t, false
		}
		if changed {
			t = &TTuple{Elements: elems}
		}
	}
	key, isAlias := x.aliasHead(t)
	if !isAlias {
		return t, true
	}
	if x.onPath[key] {
		return t, false // the alias recurs inside its own expansion
	}
	// expandAlias never expands a newtype ADT (#2239), so it stays nominal; a
	// record alias stays nominal too (see the file comment).
	e := x.u.expandAlias(t)
	if e == t {
		return t, true
	}
	if _, isRecord := e.(*TRecord); isRecord {
		return t, true
	}
	x.onPath[key] = true
	defer delete(x.onPath, key)
	return x.expand(e)
}

func (x *aliasExpander) expandEach(ts []Type) ([]Type, bool, bool) {
	out := make([]Type, len(ts))
	changed := false
	for i, a := range ts {
		e, ok := x.expand(a)
		if !ok {
			return nil, false, false
		}
		out[i] = e
		changed = changed || e != a
	}
	return out, changed, true
}

// aliasHead returns the alias key of t's head (a nullary alias, or the
// constructor of an applied one) when that head is an alias.
func (x *aliasExpander) aliasHead(t Type) (string, bool) {
	head := t
	if app, ok := t.(*TApp); ok {
		head, _ = decomposeApp(app)
	}
	con, ok := head.(*TCon)
	if !ok {
		return "", false
	}
	key := con.KeyOf()
	_, isAlias := x.u.aliasEnv[key]
	return key, isAlias
}
