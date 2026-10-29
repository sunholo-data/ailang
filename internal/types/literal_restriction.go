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
// `let`). Module declarations are checked one at a time and exported, so
// withholding there would need cross-declaration plumbing; they keep full
// generalization, and an integer literal inside a generic top-level function
// used at Float is still a runtime error (now a loud, specific one — see
// eval.checkMixedNumericDictArgs).

// literalWithhold returns the type variables the let (or letrec) `binder`
// must not generalize: every free variable of an integer literal's Num
// constraint, unless binder is the module declaration being checked.
func (tc *CoreTypeChecker) literalWithhold(ctx *InferenceContext, binder core.CoreExpr, constraints []ClassConstraint) map[string]bool {
	if ctx == nil {
		return nil
	}
	if tc.moduleDeclMode && binder != nil && ctx.declRoot == binder {
		return nil
	}
	return literalConstrainedVars(constraints)
}

// literalConstrainedVars returns the free type variables of every integer
// literal's Num constraint in constraints (nil when there are none).
func literalConstrainedVars(constraints []ClassConstraint) map[string]bool {
	var out map[string]bool
	for _, c := range constraints {
		if !c.IntLiteral {
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
