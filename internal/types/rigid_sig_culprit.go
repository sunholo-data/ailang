package types

// rigidCulprit replays the context's equalities in order and returns the path
// of the first one after which v is no longer a variable of its own: the place
// in the body that broke the promise. Runs only on failure.
//
// A deferred field access (#2239/#2284) is replayed as the solver solves it:
// while its receiver is undecided it waits, and it is folded (receiver ~ the
// record it needs) as soon as the replayed equalities decide the receiver, so a
// type the access passes on (`let get = \r. r.v; get(p) + 1`) reaches the later
// equalities, and an access that itself breaks the promise (a rigid row read
// through a let-bound copy) is named "(field access)". Accesses still waiting
// at the end are settled as record accesses, as solveDeferred settles them.
func (ctx *InferenceContext) rigidCulprit(sig *RigidSig, v *rigidViolation, groupFirst bool) []string {
	r := &culpritReplay{ctx: ctx, v: v, sub: make(Substitution)}
	order := ctx.constraints
	if groupFirst {
		order = make([]TypeConstraint, 0, len(ctx.constraints))
		var rest []TypeConstraint
		for _, c := range ctx.constraints {
			if eq, ok := c.(TypeEq); ok && len(eq.Path) == 1 && ctx.rigidGroup[eq.Path[0]] {
				order = append(order, c)
			} else {
				rest = append(rest, c)
			}
		}
		order = append(order, rest...)
	}
	for _, c := range order {
		if groupFirst {
			if eq, ok := c.(TypeEq); ok && len(eq.Path) == 1 && ctx.rigidGroup[eq.Path[0]] {
				var err error
				if r.sub, err = ctx.unifier.Unify(r.at(eq.Left), r.at(eq.Right), r.sub); err != nil {
					return nil
				}
				continue // tying the group is not the culprit
			}
		}
		var err error
		var path []string
		equality := true
		switch eq := c.(type) {
		case TypeEq:
			path = eq.Path
			r.sub, err = ctx.unifier.Unify(r.at(eq.Left), r.at(eq.Right), r.sub)
		case RowEq:
			path = eq.Path
			r.sub, err = ctx.unifier.rowUnifier.UnifyRows(eq.Left, eq.Right, r.sub)
		case *fieldAccess:
			r.pending, equality = append(r.pending, eq), false
		default:
			continue
		}
		if err != nil {
			return nil
		}
		if equality && r.broken() {
			return path
		}
		if at, stop := r.foldAccesses(false); stop {
			return at
		}
	}
	at, _ := r.foldAccesses(true)
	return at
}

// culpritReplay is rigidCulprit's state: the substitution built so far and
// the field accesses whose receiver it has not decided yet.
type culpritReplay struct {
	ctx     *InferenceContext
	v       *rigidViolation
	sub     Substitution
	pending []*fieldAccess
}

func (r *culpritReplay) at(t Type) Type { return ApplySubstitution(r.sub, t) }

// broken reports whether v no longer resolves to a variable of its own.
func (r *culpritReplay) broken() bool {
	name, ok := rigidVarResolution(r.v.v, r.at)
	if !ok {
		return true
	}
	if r.v.kind != "equal" {
		return false
	}
	o, ok := rigidVarResolution(r.v.other, r.at)
	return ok && o == name
}

// foldAccesses folds every pending access whose receiver is now decided (all
// of them when settle is set), until nothing changes. It stops with the
// access's path when the access breaks v, or when its receiver is the
// signature variable itself (the access is what makes it a record). stop with
// a nil path means the replay cannot continue (an equality failed).
func (r *culpritReplay) foldAccesses(settle bool) ([]string, bool) {
	for progress := true; progress && len(r.pending) > 0; {
		progress = false
		var open []*fieldAccess
		for _, fa := range r.pending {
			recv := r.at(fa.recv)
			if name, ok := rigidVarResolution(r.v.v, r.at); ok {
				if tv, isVar := recv.(*TVar2); isVar && tv.Name == name {
					return fa.path, true
				}
			}
			if isTypeVariable(recv) && !settle {
				open = append(open, fa)
				continue
			}
			eq := TypeEq{Left: fa.recv, Right: fa.expected(), Path: fa.path}
			if !settle {
				if decided, ok := r.ctx.fieldAccessEquation(fa, r.sub); ok {
					eq = decided
				}
			}
			var err error
			if r.sub, err = r.ctx.unifier.Unify(r.at(eq.Left), r.at(eq.Right), r.sub); err != nil {
				return nil, true
			}
			if r.broken() {
				return fa.path, true
			}
			progress = true
		}
		r.pending = open
	}
	return nil, false
}

// isTypeVariable reports whether t is an undecided type variable.
func isTypeVariable(t Type) bool {
	switch t.(type) {
	case *TVar2, *TVar:
		return true
	}
	return false
}
