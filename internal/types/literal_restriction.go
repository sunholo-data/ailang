package types

import "github.com/sunholo-data/ailang/internal/core"

// M-SOUND-NUMERIC-LITERALS: the integer-literal monomorphism restriction.
//
// An integer literal has type `Num a => a`. Where `a` resolves to a ground
// type, dictionary elaboration rewrites the literal to the value of that type
// (internal/elaborate/dictionaries.go: `0` at Float becomes `0.0`). AILANG's
// runtime has no dictionary passing for let-polymorphism, though: a
// let-generalized binding is ONE runtime value shared by every instantiation,
// and an operator inside it picks its dictionary from the runtime values it
// receives (evalDictApp's M-POLY-ARITH correction). That works for
// literal-free code (`let add = \x. \y. x + y` at Int and at Float), but a
// literal inside the binding has no instantiation to take its type from, so it
// stays an Int everywhere:
//
//	let w = 70; w / (h * h)          -- type-checked, then "Fractional::Int::add"
//	let inc = \x. x + 1; inc(1.5)    -- type-checked, then "expected float arguments"
//
// The type checker promised a polymorphism the runtime cannot deliver. The
// restriction makes the checker stop promising it, as Haskell's monomorphism
// restriction does for constrained bindings: a type variable that carries an
// integer literal's Num constraint is not generalized at a let, so the
// binding's use sites unify it with their own type and the literal is
// elaborated at that type. Using such a binding at two different numeric
// types is a type error at `ailang check`, never a runtime crash or a
// silently-Int float.
//
// Exemption: a module declaration's own binding (a top-level `func` or
// `let`). It keeps full generalization, because it is compiled once per
// numeric instance of its Num variables (numeric_variants.go, #2378): `func
// inc(x) = x + 1` runs at int and at float, and each use picks its variant.
// When a signature declares the variable (`incG[a](x: a) -> a`) the literal is
// TC_RIGID_001 at check time (#2273, rigid_sig.go).
//
// A let-bound function (specialisableLet: a non-expansive value ending in a
// lambda, or a letrec of one) is generic in its parameter positions like a
// top-level one, and copied per use over its literal-carrying variables
// (numeric_variants.go): `let h = \u. u + 1` and `let g = \y. inc(y)` give
// h(2) == 3 and h(2.5) == 3.5. Only a literal-carrying variable in no
// parameter position of its type (a constant's: `let w = 70`) stays
// monomorphic, its uses deciding its type. Any other value (a tuple of
// values) is restricted as before; an expansive one is never generalised.

// literalWithhold returns the type variables the let (or letrec) `binder`,
// whose value has type valueType, must not generalize: every free variable of
// an integer literal's Num constraint (or a PolyNumeric one) — for a let-bound
// function only those in no parameter position — unless binder is the module
// declaration being checked.
func (tc *CoreTypeChecker) literalWithhold(ctx *InferenceContext, binder core.CoreExpr, constraints []ClassConstraint, valueType Type, sub Substitution) map[string]bool {
	if ctx == nil {
		return nil
	}
	if tc.moduleDeclMode && binder != nil && ctx.declRoot == binder {
		return nil
	}
	vars := literalConstrainedVars(constraints)
	if tc.copiesPerUse(ctx, binder) && len(vars) > 0 {
		// Copied per use over its parameter-position variables (#2378).
		params := map[string]bool{}
		paramPositionVars(valueType, params, 0)
		ctx.throughFieldAccesses(sub, params)
		for v := range params {
			delete(vars, v)
		}
	}
	tc.noteSingleValue(ctx, binder, vars)
	// M-V1-NOMINAL-TYPE-IDENTITY: a variable an open constructor choice still
	// depends on stays monomorphic too, so a later use can decide the choice.
	// Recorded, so a conflict between two uses names the constructor as its
	// cause (ambiguous_ctor_cause.go).
	ctx.noteMonomorphicLet(binder)
	for v := range ctx.pendingCtorVars {
		if vars == nil {
			vars = make(map[string]bool)
		}
		vars[v] = true
	}
	return vars
}

// literalConstrainedVars returns the free type variables of every integer
// literal's Num constraint, and every PolyNumeric one, in constraints (nil
// when there are none).
func literalConstrainedVars(constraints []ClassConstraint) map[string]bool {
	var out map[string]bool
	for _, c := range constraints {
		if !c.IntLiteral && !c.PolyNumeric {
			continue
		}
		vars := make(map[string]bool)
		collectFreeVars(c.Type, vars)
		for v := range vars {
			if out == nil {
				out = make(map[string]bool)
			}
			out[v] = true
		}
	}
	return out
}

// throughFieldAccesses adds to vars the field types of the undecided field
// accesses (#2239) on a value whose type mentions vars, until nothing changes:
// `\p. {x: p.x + 1}` has p's type a variable at generalisation, its field's
// type reachable only through the access.
func (ctx *InferenceContext) throughFieldAccesses(sub Substitution, vars map[string]bool) {
	for changed := true; changed; {
		changed = false
		for _, c := range ctx.constraints {
			fa, ok := c.(*fieldAccess)
			if !ok || !overlaps(freeTypeVars(ApplySubstitution(sub, fa.recv)), vars) {
				continue
			}
			for v := range freeTypeVars(ApplySubstitution(sub, fa.fieldType)) {
				if !vars[v] {
					vars[v] = true
					changed = true
				}
			}
		}
	}
}
