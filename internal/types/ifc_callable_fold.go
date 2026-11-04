package types

// ifc_callable_fold.go — finite callables for recursive lambdas
// (M-IFC-RECURSIVE-LAMBDA-FOLD).
//
// A lambda instance carries its result callable and deferred applications
// inline (fnKey content-addresses them). When a lambda's body reaches itself
// through recursion — `f(xs) = flatMap(\b. … f(…) …, xs)` — the module fixed
// point builds λ whose result holds the previous round's λ, whose result holds
// the round before's, and so on: every round is a new, deeper value, so
// summariseModule never converges and runs until the work budget, retaining
// every key (1.4 GB on a 239-line module).
//
// foldSelf cuts that chain: inside a fresh instance of lambda X, any nested
// instance of the same literal X is merged into the outer one (its deferred
// applications and result alternatives are joined in) and replaced by fnSelf,
// "lambda X itself". Merging alternatives of one literal is a join, which is
// checked against every demand of either side, so no demand is dropped.
// resolveSelf turns fnSelf back into the instance it denotes whenever a
// lambda's result or deferred applications are read, so fnSelf never leaves
// the instance it belongs to.

// foldSelf normalises a freshly built lambda instance in place: afterwards no
// instance of n's own literal occurs inside n.result or n.deferred.
func foldSelf(n *ifcFn) {
	var nested []*ifcFn
	cut := func(cl *ifcCallable) *ifcCallable { return cutSelf(cl, n.owner, &nested) }
	n.result = cut(n.result)
	n.deferred = cutDeferred(n.deferred, cut)
	seen := map[string]bool{}
	for _, d := range n.deferred {
		seen[deferredKey(d)] = true
	}
	for len(nested) > 0 {
		inner := nested[0]
		nested = nested[1:]
		n.result = joinCallable(n.result, cut(inner.result))
		for _, d := range cutDeferred(inner.deferred, cut) {
			if k := deferredKey(d); !seen[k] {
				seen[k] = true
				n.deferred = append(n.deferred, d)
			}
		}
	}
	n.key = ""
}

// cutSelf replaces every instance of lambda owner inside cl (at any depth) by
// fnSelf and appends the replaced instances to nested. cl is not modified.
func cutSelf(cl *ifcCallable, owner string, nested *[]*ifcFn) *ifcCallable {
	if cl == nil {
		return nil
	}
	return mapAlts(cl, func(a *ifcFn) *ifcFn {
		if a.kind != fnLambda {
			return a
		}
		if a.owner == owner {
			*nested = append(*nested, a)
			return &ifcFn{kind: fnSelf, name: a.name, owner: owner}
		}
		return rebuildLambda(a, func(x *ifcCallable) *ifcCallable { return cutSelf(x, owner, nested) })
	})
}

// resolveSelf replaces fnSelf(inst.owner) inside cl, at any depth, by inst.
func resolveSelf(cl *ifcCallable, inst *ifcFn) *ifcCallable {
	if cl == nil {
		return nil
	}
	return mapAlts(cl, func(a *ifcFn) *ifcFn {
		switch {
		case a.kind == fnSelf && a.owner == inst.owner:
			return inst
		case a.kind == fnLambda:
			return rebuildLambda(a, func(x *ifcCallable) *ifcCallable { return resolveSelf(x, inst) })
		}
		return a
	})
}

// resolveDeferred resolves fnSelf in deferred applications read from inst.
func resolveDeferred(ds []ifcDeferred, inst *ifcFn) []ifcDeferred {
	if inst == nil {
		return ds
	}
	return cutDeferred(ds, func(x *ifcCallable) *ifcCallable { return resolveSelf(x, inst) })
}

// rebuildLambda applies f to a lambda's result and deferred callables, and
// returns a itself when nothing changed (keeps memoised keys and sharing).
func rebuildLambda(a *ifcFn, f func(*ifcCallable) *ifcCallable) *ifcFn {
	res := f(a.result)
	ds := cutDeferred(a.deferred, f)
	if res == a.result && sameDeferred(ds, a.deferred) {
		return a
	}
	return &ifcFn{kind: fnLambda, name: a.name, params: a.params, owner: a.owner, result: res, deferred: ds}
}

// mapAlts rebuilds cl with f applied to each alternative, deduplicating by
// identity; it returns cl itself when f changed nothing.
func mapAlts(cl *ifcCallable, f func(*ifcFn) *ifcFn) *ifcCallable {
	changed := false
	alts := make([]*ifcFn, len(cl.alts))
	for i, a := range cl.alts {
		alts[i] = f(a)
		changed = changed || alts[i] != a
	}
	if !changed {
		return cl
	}
	var out *ifcCallable
	for _, a := range alts {
		out = joinCallable(out, single(a))
	}
	return out
}

// cutDeferred applies f to every callable of ds; it returns ds itself when f
// changed nothing.
func cutDeferred(ds []ifcDeferred, f func(*ifcCallable) *ifcCallable) []ifcDeferred {
	var out []ifcDeferred
	for i, d := range ds {
		nd := ifcDeferred{target: f(d.target), via: d.via, args: d.args}
		argsChanged := false
		for j, x := range d.args {
			if fx := f(x.fn); fx != x.fn {
				if !argsChanged {
					nd.args = append([]ifcVal(nil), d.args...)
					argsChanged = true
				}
				nd.args[j] = ifcVal{label: x.label, fn: fx}
			}
		}
		if out == nil && (nd.target != d.target || argsChanged) {
			out = append(make([]ifcDeferred, 0, len(ds)), ds[:i]...)
		}
		if out != nil {
			out = append(out, nd)
		}
	}
	if out == nil {
		return ds
	}
	return out
}

func sameDeferred(a, b []ifcDeferred) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}
