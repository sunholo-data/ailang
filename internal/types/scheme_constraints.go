package types

// schemeConstraintsFor keeps the class constraints a scheme must carry: those
// mentioning at least one generalized type variable, each (class, type) once.
//
// Callers pass every unsolved constraint of the whole inference context, not
// just this binding's. Copying them all into the scheme made inferVar re-emit
// the entire context at every use, so each nested let multiplied the constraint
// list: sunholo/crew_core's session.ail reached millions of constraints, ~4 GB
// live and 10-14 GB peak on a cold check, which OOM-killed the 512 MiB registry
// validator (#1811). A constraint on non-generalized variables is not lost — it
// stays in ctx.constraints (append-only) and is enforced at the decl boundary.
func schemeConstraintsFor(constraints []ClassConstraint, generalized []string) []Constraint {
	if len(constraints) == 0 || len(generalized) == 0 {
		return []Constraint{}
	}
	gen := make(map[string]bool, len(generalized))
	for _, v := range generalized {
		gen[v] = true
	}
	out := []Constraint{}
	seen := make(map[string]bool)
	for _, c := range constraints {
		vars := make(map[string]bool)
		collectFreeVars(c.Type, vars)
		relevant := false
		for v := range vars {
			if gen[v] {
				relevant = true
				break
			}
		}
		if !relevant {
			continue
		}
		key := c.Class + "\x00" + c.Type.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Constraint{Class: c.Class, Type: c.Type})
	}
	return out
}
