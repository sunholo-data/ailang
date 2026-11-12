package types

import "fmt"

// Context reduction at a top-level boundary (#2247).
//
// A class constraint may leave a declaration in exactly two ways: an instance
// solves it, or it is a constraint on a bare type variable that the caller
// will instantiate (deferred, as in `func lt[a](x: a, y: a) = x < y`).
// Anything else has a constructor at its head, and that head alone decides
// whether an instance can ever exist, so it is decided NOW.
//
// Before #2247 only Eq was reduced (M-EQ-DERIVE-CONTAINERS R-D7); every other
// class "passed through unchanged" and was then dropped, because nothing
// downstream reads a top-level declaration's residual constraints. So
// `g(7)` with `g[a](o: Box[a])` raised Num[Box[α]], which was neither ground
// (so never looked up) nor a variable (so never generalised into anything a
// caller would check): `check` passed and the match saw an int at run time.
//
// Eq keeps its structural decomposition (reduceEq). Every other class has
// only instances with a nullary head (Num/Fractional/Ord/Show at int, float,
// string, bool), so a non-variable type is solved only by an instance at
// exactly that type; a type with a free variable under a constructor can
// never match one, and the lookup reports the missing instance.

// ReduceConstraints reduces every non-ground constraint that is not on a bare
// type variable: Eq decomposes structurally, every other class must be solved
// by an instance. What survives is a constraint on a type variable, which
// stays polymorphic (generalised, or phantom as in `None == None`).
func (env *InstanceEnv) ReduceConstraints(constraints []ClassConstraint) ([]ClassConstraint, error) {
	out := make([]ClassConstraint, 0, len(constraints))
	for _, c := range constraints {
		residual, err := env.reduceConstraint(c.Class, c.Type)
		if err != nil {
			if len(c.Path) > 0 {
				return nil, fmt.Errorf("at %s: %w", c.Path[0], err)
			}
			return nil, err
		}
		for _, r := range residual {
			rc := c
			rc.Type = r
			out = append(out, rc)
		}
	}
	return out, nil
}

// reduceConstraint returns the residual types class still constrains, or the
// reason no instance can solve class[typ].
func (env *InstanceEnv) reduceConstraint(class string, typ Type) ([]Type, error) {
	if class == "Eq" {
		return env.reduceEq(typ, 0)
	}
	if hasVariableHead(typ) {
		return []Type{typ}, nil
	}
	if _, err := env.Lookup(class, typ); err != nil {
		return nil, err
	}
	return nil, nil
}

// hasVariableHead reports whether typ is a type variable or an application
// whose head is one (`f[a]`): only then can a later substitution still pick
// the instance, so only then may the constraint be deferred.
func hasVariableHead(typ Type) bool {
	head := typ
	if app, ok := typ.(*TApp); ok {
		head, _ = decomposeApp(app)
	}
	switch head.(type) {
	case *TVar, *TVar2:
		return true
	}
	return false
}
