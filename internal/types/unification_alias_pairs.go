package types

// Sharing the unification of two nullary aliases (#2329).
//
// Two separately declared aliases with the same structure — a dense DAG
// `type D5 = (D0, D1, D2, D3, D4)`, ..., and its twin over E — stay single
// alias names on both sides of `f(x: D119) -> E119 = x`, and unification
// expands them one level at a time: D119 ~ E119 needs D114 ~ E114 ... D118 ~
// E118, each of which needs the pairs below it again. Every pair is met once
// per path through the DAG, so the walk is exponential although only n
// distinct pairs exist.
//
// A pair of GROUND nullary aliases (no type, row or effect variable anywhere in
// their expansion) unifies the same way under any substitution and binds
// nothing, so once it has succeeded it is skipped. Only successes are
// remembered; a pair with a variable in its expansion is never skipped,
// because unifying it binds the variable in a substitution a later call may
// not carry.

// groundAliasPair reports the alias keys of t1 and t2 when both are ground
// nullary aliases (not newtypes, which never expand).
func (u *Unifier) groundAliasPair(t1, t2 Type) ([2]string, bool) {
	k1, ok1 := u.groundNullaryAlias(t1)
	if !ok1 {
		return [2]string{}, false
	}
	k2, ok2 := u.groundNullaryAlias(t2)
	if !ok2 {
		return [2]string{}, false
	}
	if u.aliasPairs == nil {
		u.aliasPairs = map[[2]string]bool{}
	}
	return [2]string{k1, k2}, true
}

func (u *Unifier) groundNullaryAlias(t Type) (string, bool) {
	con, ok := t.(*TCon)
	if !ok || u.aliasEnv == nil {
		return "", false
	}
	key := con.KeyOf()
	if _, alias := u.aliasEnv[key]; !alias || len(u.aliasParams[key]) > 0 || u.newtypes[key] {
		return "", false
	}
	return key, u.aliasIsGround(key)
}

// aliasIsGround reports whether a nullary alias's full expansion holds no
// variable. Memoised per alias, and each body is walked once per node, so a
// dense DAG of aliases costs its size, not its expansion. An alias reached
// again while its own check is in progress (a cycle, rejected at the
// declaration by TC_TYPE_ALIAS_CYCLE_001) counts as not ground.
func (u *Unifier) aliasIsGround(key string) bool {
	if g, ok := u.aliasGround[key]; ok {
		return g
	}
	if u.aliasGround == nil {
		u.aliasGround = map[string]bool{}
	}
	u.aliasGround[key] = false
	g := u.groundBody(u.aliasEnv[key], map[Type]bool{})
	u.aliasGround[key] = g
	return g
}

// groundBody reports whether t holds no variable, expanding nullary aliases.
// An applied alias or an unknown form counts as not ground (conservative: the
// pair is then simply not shared). Cycle-safe: a node in progress is not ground.
func (u *Unifier) groundBody(t Type, memo map[Type]bool) bool {
	if t == nil {
		return true
	}
	if g, ok := memo[t]; ok {
		return g
	}
	memo[t] = false
	g := u.groundNode(t, memo)
	memo[t] = g
	return g
}

func (u *Unifier) groundNode(t Type, memo map[Type]bool) bool {
	all := func(ts ...Type) bool {
		for _, c := range ts {
			if !u.groundBody(c, memo) {
				return false
			}
		}
		return true
	}
	switch x := t.(type) {
	case *TCon:
		key := x.KeyOf()
		if _, alias := u.aliasEnv[key]; !alias || u.newtypes[key] {
			return true
		}
		return len(u.aliasParams[key]) == 0 && u.aliasIsGround(key)
	case *TList:
		return all(x.Element)
	case *TArray:
		return all(x.Element)
	case *TMap:
		return all(x.Key, x.Value)
	case *TTuple:
		return all(x.Elements...)
	case *TRecord:
		for _, f := range x.Fields {
			if !all(f) {
				return false
			}
		}
		return all(x.Row)
	case *Row:
		if x.Tail != nil {
			return false
		}
		for _, l := range x.Labels {
			if !all(l) {
				return false
			}
		}
		return true
	case *TFunc2:
		if x.EffectRow != nil && !all(x.EffectRow) {
			return false
		}
		return all(x.Params...) && all(x.Return)
	case *TApp:
		if con, ok := x.Constructor.(*TCon); ok {
			if _, alias := u.aliasEnv[con.KeyOf()]; alias {
				return false
			}
		}
		return all(x.Constructor) && all(x.Args...)
	}
	return false // variables, open records, labelled and unknown forms
}
