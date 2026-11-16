package types

// Unifying two aliases with identical expansions (#2329).
//
// Two separately declared aliases with the same structure — a dense DAG
// `type D5 = (D0, D1, D2, D3, D4)`, ... and its twin over E, or chains that
// pass a parameter through (`type P1[a] = (P0[a], P0[a])` and the same for Q)
// — stay alias applications on both sides of `f(x: D119) -> E119` or
// `f(u) -> Q22[int] = g(u)`, and unification expands them one level at a time.
// D119 ~ E119 needs D114 ~ E114 ... D118 ~ E118, each of which needs the pairs
// below it again: every pair is met once per path through the DAG, so the walk
// is exponential although only n distinct pairs exist.
//
// When the two applications have the same arguments and the two aliases are
// EQUIVALENT — their bodies are the same structure, parameter for parameter,
// with corresponding alias applications in turn equivalent — both sides expand
// to the same type, which unifies with itself under any substitution and binds
// nothing. Unify then succeeds at once. Equivalence is decided once per pair of
// aliases (memoised), so the check costs the number of distinct pairs.
// Anything the comparison does not understand (a function or row type, an
// alias applied with different arguments, an alias meeting a non-alias) is
// "not known equivalent", and unification proceeds as before: the shortcut can
// only skip work whose outcome it has proved, never decide a different one.

// sameAliasApplication reports whether t1 and t2 are applications of
// equivalent aliases to the same arguments.
func (u *Unifier) sameAliasApplication(t1, t2 Type) bool {
	k1, args1, ok1 := u.aliasApplication(t1)
	if !ok1 {
		return false
	}
	k2, args2, ok2 := u.aliasApplication(t2)
	if !ok2 || len(args1) != len(args2) {
		return false
	}
	for i := range args1 {
		if !sameArgument(args1[i], args2[i]) {
			return false
		}
	}
	return k1 == k2 || u.aliasesEquivalent(k1, k2)
}

// aliasApplication reads an alias head and its arguments (none for a nullary
// alias). A newtype is nominal and never expands, so it is not an alias here.
func (u *Unifier) aliasApplication(t Type) (string, []Type, bool) {
	if len(u.aliasEnv) == 0 {
		return "", nil, false
	}
	var head Type = t
	var args []Type
	if app, ok := t.(*TApp); ok {
		if con, direct := app.Constructor.(*TCon); direct {
			head, args = con, app.Args // the common case: no allocation
		} else {
			head, args = decomposeApp(app)
		}
	}
	con, ok := head.(*TCon)
	if !ok {
		return "", nil, false
	}
	key := con.KeyOf()
	if _, alias := u.aliasEnv[key]; !alias || u.newtypes[key] || len(u.aliasParams[key]) != len(args) {
		return "", nil, false
	}
	return key, args, true
}

// sameArgument is a cheap identity test: the same node, the same type
// constructor, or the same variable.
func sameArgument(a, b Type) bool {
	if a == b {
		return true
	}
	switch x := a.(type) {
	case *TCon:
		y, ok := b.(*TCon)
		return ok && x.KeyOf() == y.KeyOf()
	case *TVar:
		y, ok := b.(*TVar)
		return ok && x.Name == y.Name
	case *TVar2:
		y, ok := b.(*TVar2)
		return ok && x.Name == y.Name
	}
	return false
}

// aliasesEquivalent decides, once per pair, whether two aliases of the same
// arity expand to the same type for every argument list. A pair met again
// while its own comparison is in progress (a cycle, which the alias cycle
// check rejects anyway) is not equivalent.
func (u *Unifier) aliasesEquivalent(k1, k2 string) bool {
	pair := [2]string{k1, k2}
	if eq, ok := u.aliasEquiv[pair]; ok {
		return eq
	}
	if u.aliasEquiv == nil {
		u.aliasEquiv = map[[2]string]bool{}
	}
	u.aliasEquiv[pair] = false
	p1, p2 := u.aliasParams[k1], u.aliasParams[k2]
	eq := len(p1) == len(p2)
	if eq {
		params := make(map[[2]string]bool, len(p1))
		for i := range p1 {
			params[[2]string{p1[i], p2[i]}] = true
		}
		eq = u.sameShape(u.aliasEnv[k1], u.aliasEnv[k2], params)
	}
	u.aliasEquiv[pair] = eq
	return eq
}

// sameShape compares two alias bodies; params pairs the parameter names that
// correspond. Recursion is bounded by the bodies' written size: an alias
// application is compared by aliasesEquivalent, never expanded.
func (u *Unifier) sameShape(a, b Type, params map[[2]string]bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	if k1, args1, ok := u.aliasApplication(a); ok {
		k2, args2, ok2 := u.aliasApplication(b)
		if !ok2 || len(args1) != len(args2) {
			return false
		}
		for i := range args1 {
			if !u.sameShape(args1[i], args2[i], params) {
				return false
			}
		}
		return k1 == k2 || u.aliasesEquivalent(k1, k2)
	}
	if _, _, ok := u.aliasApplication(b); ok {
		return false
	}
	all := func(xs, ys []Type) bool {
		if len(xs) != len(ys) {
			return false
		}
		for i := range xs {
			if !u.sameShape(xs[i], ys[i], params) {
				return false
			}
		}
		return true
	}
	switch x := a.(type) {
	case *TVar:
		y, ok := b.(*TVar)
		return ok && (params[[2]string{x.Name, y.Name}] || (x.Name == y.Name && !isParamName(x.Name, params)))
	case *TVar2:
		y, ok := b.(*TVar2)
		return ok && (params[[2]string{x.Name, y.Name}] || (x.Name == y.Name && !isParamName(x.Name, params)))
	case *TCon:
		y, ok := b.(*TCon)
		return ok && x.KeyOf() == y.KeyOf()
	case *TList:
		y, ok := b.(*TList)
		return ok && u.sameShape(x.Element, y.Element, params)
	case *TArray:
		y, ok := b.(*TArray)
		return ok && u.sameShape(x.Element, y.Element, params)
	case *TMap:
		y, ok := b.(*TMap)
		return ok && u.sameShape(x.Key, y.Key, params) && u.sameShape(x.Value, y.Value, params)
	case *TTuple:
		y, ok := b.(*TTuple)
		return ok && all(x.Elements, y.Elements)
	case *TApp:
		y, ok := b.(*TApp)
		return ok && u.sameShape(x.Constructor, y.Constructor, params) && all(x.Args, y.Args)
	case *TRecord:
		y, ok := b.(*TRecord)
		if !ok || x.Row != nil || y.Row != nil || x.TypeName != y.TypeName || len(x.Fields) != len(y.Fields) {
			return false
		}
		for name, f := range x.Fields {
			g, has := y.Fields[name]
			if !has || !u.sameShape(f, g, params) {
				return false
			}
		}
		return true
	}
	return false // functions, rows, open records, labels: not compared
}

// isParamName reports whether name is one side's parameter: a parameter may
// only match its counterpart, never a same-named free variable.
func isParamName(name string, params map[[2]string]bool) bool {
	for p := range params {
		if p[0] == name || p[1] == name {
			return true
		}
	}
	return false
}
